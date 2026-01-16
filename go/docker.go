package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ComposeFileNames 支持的 compose 配置文件名
var ComposeFileNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

// Service 表示一个 Docker Compose 服务
type Service struct {
	Name    string // 服务目录名
	Path    string // 服务完整路径
	Running bool   // 是否正在运行（备份前的状态）
}

// DockerManager 管理 Docker Compose 服务
type DockerManager struct {
	baseDir string
	dryRun  bool
}

// NewDockerManager 创建 Docker 管理器
func NewDockerManager(baseDir string, dryRun bool) *DockerManager {
	return &DockerManager{
		baseDir: baseDir,
		dryRun:  dryRun,
	}
}

// DiscoverServices 发现所有 Docker Compose 服务
func (dm *DockerManager) DiscoverServices() ([]*Service, error) {
	entries, err := os.ReadDir(dm.baseDir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	var services []*Service
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		svcPath := filepath.Join(dm.baseDir, entry.Name())
		if dm.hasComposeFile(svcPath) || dm.hasComposeScript(svcPath) {
			svc := &Service{
				Name: entry.Name(),
				Path: svcPath,
			}
			// 检查服务是否正在运行
			svc.Running = dm.IsRunning(svc)
			services = append(services, svc)
		}
	}

	return services, nil
}

// hasComposeFile 检查目录下是否有 compose 配置文件
func (dm *DockerManager) hasComposeFile(path string) bool {
	for _, name := range ComposeFileNames {
		if _, err := os.Stat(filepath.Join(path, name)); err == nil {
			return true
		}
	}
	return false
}

// hasComposeScript 检查目录下是否有 compose 脚本
func (dm *DockerManager) hasComposeScript(path string) bool {
	scripts := []string{"compose-up.sh", "compose-stop.sh", "compose-down.sh"}
	for _, script := range scripts {
		if info, err := os.Stat(filepath.Join(path, script)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// IsRunning 检查服务是否正在运行
func (dm *DockerManager) IsRunning(svc *Service) bool {
	// 在服务目录下执行 docker compose ps -q
	cmd := exec.Command("docker", "compose", "ps", "-q")
	cmd.Dir = svc.Path

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// 如果有输出（容器 ID），说明有容器在运行
	return len(bytes.TrimSpace(output)) > 0
}

// Stop 停止服务
func (dm *DockerManager) Stop(svc *Service) error {
	if !svc.Running {
		slog.Info("跳过停止（服务未运行）", "service", svc.Name)
		return nil
	}

	// 确定停止方式
	var cmd *exec.Cmd
	var method string

	stopScript := filepath.Join(svc.Path, "compose-stop.sh")
	downScript := filepath.Join(svc.Path, "compose-down.sh")

	if dm.isExecutable(stopScript) {
		cmd = exec.Command("./compose-stop.sh")
		method = "compose-stop.sh"
	} else if dm.isExecutable(downScript) {
		cmd = exec.Command("./compose-down.sh")
		method = "compose-down.sh"
	} else if dm.hasComposeFile(svc.Path) {
		cmd = exec.Command("docker", "compose", "stop")
		method = "docker compose stop"
	} else {
		return fmt.Errorf("无法识别停止方法")
	}

	if dm.dryRun {
		slog.Info("[DRY-RUN] 将停止服务", "service", svc.Name, "method", method)
		return nil
	}

	slog.Info("停止服务", "service", svc.Name, "method", method)
	cmd.Dir = svc.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("停止失败: %w", err)
	}
	return nil
}

// Start 启动服务
func (dm *DockerManager) Start(svc *Service) error {
	if !svc.Running {
		slog.Info("跳过启动（原本未运行）", "service", svc.Name)
		return nil
	}

	// 确定启动方式
	var cmd *exec.Cmd
	var method string

	upScript := filepath.Join(svc.Path, "compose-up.sh")

	if dm.isExecutable(upScript) {
		cmd = exec.Command("./compose-up.sh")
		method = "compose-up.sh"
	} else if dm.hasComposeFile(svc.Path) {
		cmd = exec.Command("docker", "compose", "up", "-d")
		method = "docker compose up -d"
	} else {
		return fmt.Errorf("无法识别启动方法")
	}

	if dm.dryRun {
		slog.Info("[DRY-RUN] 将启动服务", "service", svc.Name, "method", method)
		return nil
	}

	slog.Info("启动服务", "service", svc.Name, "method", method)
	cmd.Dir = svc.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}
	return nil
}

// isExecutable 检查文件是否可执行
func (dm *DockerManager) isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// 在 Unix 系统上检查执行权限
	return info.Mode()&0111 != 0
}

// ClassifyServices 将服务分类为优先服务和普通服务
func ClassifyServices(services []*Service, priorityNames []string) (priority, normal []*Service) {
	prioritySet := make(map[string]bool)
	for _, name := range priorityNames {
		prioritySet[strings.ToLower(name)] = true
	}

	for _, svc := range services {
		if prioritySet[strings.ToLower(svc.Name)] {
			priority = append(priority, svc)
		} else {
			normal = append(normal, svc)
		}
	}
	return
}

// StopParallel 并行停止多个服务
func (dm *DockerManager) StopParallel(services []*Service) []error {
	if len(services) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, svc := range services {
		wg.Add(1)
		go func(s *Service) {
			defer wg.Done()
			if err := dm.Stop(s); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", s.Name, err))
				mu.Unlock()
			}
		}(svc)
	}

	wg.Wait()
	return errors
}

// StartParallel 并行启动多个服务
func (dm *DockerManager) StartParallel(services []*Service) []error {
	if len(services) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, svc := range services {
		wg.Add(1)
		go func(s *Service) {
			defer wg.Done()
			if err := dm.Start(s); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", s.Name, err))
				mu.Unlock()
			}
		}(svc)
	}

	wg.Wait()
	return errors
}
