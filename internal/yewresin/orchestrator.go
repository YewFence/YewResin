package yewresin

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Orchestrator 备份流程编排器
type Orchestrator struct {
	cfg      *Config
	dryRun   bool
	docker   dockerController
	kopia    kopiaController
	notifier notifierClient
	gist     gistClient

	// 服务分类
	priorityServices []*Service
	normalServices   []*Service

	// 锁文件
	lockAcquired bool

	// 执行时间记录
	startTime time.Time
}

type dockerController interface {
	CheckDependencies() error
	DiscoverServices() ([]*Service, error)
	StopParallel(services []*Service) []error
	Stop(svc *Service) error
	StartParallel(services []*Service) []error
	Start(svc *Service) error
}

type kopiaController interface {
	CheckRepository() error
	CreateSnapshot(path string) error
}

type notifierClient interface {
	Send(title, body string)
	Wait()
}

type gistClient interface {
	Upload(logContent string, success bool, startTime time.Time, duration time.Duration) error
}

// NewOrchestrator 创建编排器
func NewOrchestrator(cfg *Config, dryRun bool) *Orchestrator {
	return &Orchestrator{
		cfg:      cfg,
		dryRun:   dryRun,
		docker:   NewDockerManager(cfg.BaseDir, dryRun, time.Duration(cfg.DockerCommandTimeoutSeconds)*time.Second),
		kopia:    NewKopiaBackup(cfg.ExpectedRemote, cfg.KopiaConfigFile, cfg.RcloneConfig, dryRun),
		notifier: NewNotifier(cfg.AppriseURL, cfg.AppriseNotifyURL, cfg.DeviceName),
		gist:     NewGistManager(cfg.GistToken, cfg.GistID, cfg.GistLogPrefix, cfg.GistMaxLogs, cfg.GistKeepFirstFile),
	}
}

// 检查依赖项
func (o *Orchestrator) CheckDependencies() error {
	// 检查 Docker 可用性
	if err := o.docker.CheckDependencies(); err != nil {
		return err
	}

	// 检查 Kopia 仓库连接状态
	if err := o.kopia.CheckRepository(); err != nil {
		return err
	}

	return nil
}

// Run 执行备份流程
func (o *Orchestrator) Run() error {
	o.startTime = time.Now().UTC()
	defer o.notifier.Wait()

	// 1. 获取锁
	if err := o.acquireLock(); err != nil {
		return err
	}
	defer o.releaseLock()

	// 2. 发现并分类服务
	services, err := o.docker.DiscoverServices()
	if err != nil {
		return fmt.Errorf("发现服务失败: %w", err)
	}

	o.priorityServices, o.normalServices = ClassifyServices(services, o.cfg.PriorityServices)

	slog.Info("发现服务",
		"total", len(services),
		"priority", len(o.priorityServices),
		"normal", len(o.normalServices))

	// 3. 发送开始通知
	o.notifier.Send("🔄 备份开始", "开始执行服务器备份任务")

	// 4. 停止服务（普通服务并行，优先服务顺序）
	slog.Info(">>> 并行停止普通服务...")
	if errs := o.docker.StopParallel(o.normalServices); len(errs) > 0 {
		errMsgs := make([]string, len(errs))
		for i, e := range errs {
			errMsgs[i] = e.Error()
		}
		o.notifier.Send("❌ 备份中止", fmt.Sprintf("服务停止失败: %s", strings.Join(errMsgs, ", ")))
		o.startAllServices()
		return fmt.Errorf("停止普通服务失败: %v", errs)
	}

	slog.Info(">>> 顺序停止优先服务（网关）...")
	for _, svc := range o.priorityServices {
		if err := o.docker.Stop(svc); err != nil {
			o.notifier.Send("❌ 备份中止", fmt.Sprintf("服务 %s 停止失败", svc.Name))
			o.startAllServices()
			return fmt.Errorf("停止服务 %s 失败: %w", svc.Name, err)
		}
	}

	// 5. 创建快照
	slog.Info(">>> 所有服务已停止，开始创建快照...")
	backupErr := o.kopia.CreateSnapshot(o.cfg.BaseDir)

	// 6. 恢复服务（无论备份是否成功）
	o.startAllServices()

	// 7. 上传日志到 Gist
	success := backupErr == nil
	duration := time.Since(o.startTime)
	if LogWriter != nil {
		if err := o.gist.Upload(LogWriter.GetContent(), success, o.startTime, duration); err != nil {
			slog.Warn("上传日志到 Gist 失败", "error", err)
		}
	}

	// 8. 发送结果通知
	if backupErr != nil {
		o.notifier.Send("❌ 备份失败", "快照创建失败，服务已恢复")
		return backupErr
	}

	if o.dryRun {
		o.notifier.Send("🧪 DRY-RUN 完成", "模拟运行完成，未执行实际操作")
	} else {
		o.notifier.Send("✅ 备份成功", "所有服务已恢复运行")
	}

	return nil
}

// startAllServices 启动所有服务（优先服务顺序启动，普通服务并行启动）
func (o *Orchestrator) startAllServices() {
	var failedServices []string

	// 优先服务顺序启动（先恢复网关）
	slog.Info(">>> 顺序恢复优先服务（网关）...")
	for _, svc := range o.priorityServices {
		if err := o.docker.Start(svc); err != nil {
			slog.Error("启动服务失败", "service", svc.Name, "error", err)
			failedServices = append(failedServices, svc.Name)
		}
	}

	// 普通服务并行启动
	slog.Info(">>> 并行恢复普通服务...")
	if errs := o.docker.StartParallel(o.normalServices); len(errs) > 0 {
		for _, err := range errs {
			slog.Error("启动服务失败", "error", err)
			// 从结构化错误中提取服务名
			var svcErr *ServiceError
			if errors.As(err, &svcErr) {
				failedServices = append(failedServices, svcErr.Service)
			} else {
				failedServices = append(failedServices, "unknown")
			}
		}
	}

	if len(failedServices) > 0 {
		o.notifier.Send("⚠️ 服务恢复异常", fmt.Sprintf("以下服务启动失败: %v", failedServices))
	}
}

// Cleanup 清理函数（异常退出时调用）
func (o *Orchestrator) Cleanup() {
	slog.Warn("执行清理，尝试恢复服务...")
	o.startAllServices()
	o.releaseLock()
	o.notifier.Wait()
}

// acquireLock 获取锁（使用目录作为锁，原子操作）
func (o *Orchestrator) acquireLock() error {
	if o.dryRun {
		slog.Info("[DRY-RUN] 将获取锁", "path", o.cfg.LockFile)
		return nil
	}

	err := os.Mkdir(o.cfg.LockFile, 0755)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("另一个备份进程正在运行 (锁文件: %s)", o.cfg.LockFile)
		}
		return fmt.Errorf("创建锁文件失败: %w", err)
	}

	o.lockAcquired = true
	slog.Info("获取锁成功", "path", o.cfg.LockFile)
	return nil
}

// releaseLock 释放锁
func (o *Orchestrator) releaseLock() {
	if !o.lockAcquired {
		return
	}

	if err := os.RemoveAll(o.cfg.LockFile); err != nil {
		slog.Warn("释放锁失败", "error", err)
	} else {
		slog.Info("释放锁成功")
	}
	o.lockAcquired = false
}
