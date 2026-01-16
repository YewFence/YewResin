package main

import (
	"fmt"
	"log/slog"
	"os"
)

// Orchestrator 备份流程编排器
type Orchestrator struct {
	cfg      *Config
	dryRun   bool
	docker   *DockerManager
	kopia    *KopiaBackup
	notifier *Notifier

	// 服务分类
	priorityServices []*Service
	normalServices   []*Service

	// 锁文件
	lockAcquired bool
}

// NewOrchestrator 创建编排器
func NewOrchestrator(cfg *Config, dryRun bool) *Orchestrator {
	return &Orchestrator{
		cfg:      cfg,
		dryRun:   dryRun,
		docker:   NewDockerManager(cfg.BaseDir, dryRun),
		kopia:    NewKopiaBackup(cfg.ExpectedRemote, cfg.KopiaPassword, dryRun),
		notifier: NewNotifier(cfg.AppriseURL, cfg.AppriseNotifyURL, cfg.DeviceName),
	}
}

// Run 执行备份流程
func (o *Orchestrator) Run() error {
	// 1. 检查依赖
	if err := o.kopia.CheckDependencies(); err != nil {
		return err
	}

	// 2. 检查 Kopia 仓库连接
	if err := o.kopia.CheckRepository(); err != nil {
		return err
	}

	// 3. 获取锁
	if err := o.acquireLock(); err != nil {
		return err
	}

	// 4. 发现并分类服务
	services, err := o.docker.DiscoverServices()
	if err != nil {
		return fmt.Errorf("发现服务失败: %w", err)
	}

	o.priorityServices, o.normalServices = ClassifyServices(services, o.cfg.PriorityServices)

	slog.Info("发现服务",
		"total", len(services),
		"priority", len(o.priorityServices),
		"normal", len(o.normalServices))

	// 5. 发送开始通知
	o.notifier.Send("🔄 备份开始", "开始执行服务器备份任务")

	// 6. 停止服务（先普通后优先）
	slog.Info(">>> 停止普通服务...")
	for _, svc := range o.normalServices {
		if err := o.docker.Stop(svc); err != nil {
			o.notifier.Send("❌ 备份中止", fmt.Sprintf("服务 %s 停止失败", svc.Name))
			return fmt.Errorf("停止服务 %s 失败: %w", svc.Name, err)
		}
	}

	slog.Info(">>> 停止优先服务（网关）...")
	for _, svc := range o.priorityServices {
		if err := o.docker.Stop(svc); err != nil {
			o.notifier.Send("❌ 备份中止", fmt.Sprintf("服务 %s 停止失败", svc.Name))
			return fmt.Errorf("停止服务 %s 失败: %w", svc.Name, err)
		}
	}

	// 7. 创建快照
	slog.Info(">>> 所有服务已停止，开始创建快照...")
	backupErr := o.kopia.CreateSnapshot(o.cfg.BaseDir)

	// 8. 恢复服务（无论备份是否成功）
	o.startAllServices()

	// 9. 执行维护（可选）
	if backupErr == nil {
		o.kopia.Maintenance()
	}

	// 10. 发送结果通知
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

// startAllServices 启动所有服务（先优先后普通）
func (o *Orchestrator) startAllServices() {
	var failedServices []string

	slog.Info(">>> 恢复优先服务（网关）...")
	for _, svc := range o.priorityServices {
		if err := o.docker.Start(svc); err != nil {
			slog.Error("启动服务失败", "service", svc.Name, "error", err)
			failedServices = append(failedServices, svc.Name)
		}
	}

	slog.Info(">>> 恢复普通服务...")
	for _, svc := range o.normalServices {
		if err := o.docker.Start(svc); err != nil {
			slog.Error("启动服务失败", "service", svc.Name, "error", err)
			failedServices = append(failedServices, svc.Name)
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

	if err := os.Remove(o.cfg.LockFile); err != nil {
		slog.Warn("释放锁失败", "error", err)
	} else {
		slog.Info("释放锁成功")
	}
	o.lockAcquired = false
}
