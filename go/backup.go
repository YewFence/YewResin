package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// KopiaBackup Kopia 备份管理器
type KopiaBackup struct {
	expectedRemote string
	configFile     string // Kopia 配置文件路径
	rcloneConfig   string // Rclone 配置文件路径
	dryRun         bool
	cachedEnv      []string // 缓存的环境变量
}

// NewKopiaBackup 创建 Kopia 备份管理器
func NewKopiaBackup(expectedRemote, configFile, rcloneConfig string, dryRun bool) *KopiaBackup {
	// 初始化时构建并缓存环境变量
	env := os.Environ()
	if rcloneConfig != "" {
		env = append(env, "RCLONE_CONFIG="+rcloneConfig)
	}

	return &KopiaBackup{
		expectedRemote: expectedRemote,
		configFile:     configFile,
		rcloneConfig:   rcloneConfig,
		dryRun:         dryRun,
		cachedEnv:      env,
	}
}

// CheckDependencies 检查 Kopia 是否已安装
func (k *KopiaBackup) CheckDependencies() error {
	// 检查 kopia
	if _, err := exec.LookPath("kopia"); err != nil {
		return fmt.Errorf("kopia 未安装，请先安装: https://kopia.io/docs/installation/")
	}

	// 检查自定义配置文件是否存在
	if k.configFile != "" {
		if _, err := os.Stat(k.configFile); err != nil {
			return fmt.Errorf("指定的 Kopia 配置文件不可访问: %w", err)
		}
	}
	if k.rcloneConfig != "" {
		if _, err := exec.LookPath("rclone"); err != nil {
			return fmt.Errorf("rclone 未安装，请先安装")
		}
		if _, err := os.Stat(k.rcloneConfig); err != nil {
			return fmt.Errorf("指定的 Rclone 配置文件不可访问: %w", err)
		}
	}

	slog.Info("依赖检查通过", "kopia", "✓")
	return nil
}

// buildKopiaArgs 构建 kopia 命令参数（包含可选的 --config-file）
func (k *KopiaBackup) buildKopiaArgs(args ...string) []string {
	if k.configFile != "" {
		return append([]string{"--config-file=" + k.configFile}, args...)
	}
	return args
}

// buildEnv 返回缓存的环境变量（包含 RCLONE_CONFIG）
func (k *KopiaBackup) buildEnv() []string {
	return k.cachedEnv
}

// CheckRepository 检查 Kopia 仓库连接状态
func (k *KopiaBackup) CheckRepository() error {
	// 检查仓库状态
	args := k.buildKopiaArgs("repository", "status", "--json")
	cmd := exec.Command("kopia", args...)
	cmd.Env = k.buildEnv()
	output, err := cmd.CombinedOutput()

	if err != nil {
		hint := "kopia repository connect"
		if k.configFile != "" {
			hint = "kopia --config-file=" + k.configFile + " repository connect"
		}
		if k.rcloneConfig != "" {
			hint = "RCLONE_CONFIG=" + k.rcloneConfig + " " + hint
		}
		return fmt.Errorf("Kopia 仓库未连接，请手动执行 '%s' 连接仓库", hint)
	}

	var status struct {
		Storage struct {
			Type   string `json:"type"`
			Config struct {
				RemotePath string `json:"remotePath"`
			} `json:"config"`
		} `json:"storage"`
	}

	if err := json.Unmarshal(output, &status); err != nil {
		return fmt.Errorf("解析 Kopia 仓库状态失败: %w", err)
	}

	if status.Storage.Config.RemotePath == "" {
		return fmt.Errorf("Kopia 仓库状态缺少 remotePath")
	}

	if status.Storage.Config.RemotePath != k.expectedRemote {
		return fmt.Errorf("Kopia 仓库路径不匹配，期望: %s，实际: %s", k.expectedRemote, status.Storage.Config.RemotePath)
	}

	slog.Info("Kopia 仓库已连接", "remote", k.expectedRemote)
	return nil
}

// CreateSnapshot 创建快照
func (k *KopiaBackup) CreateSnapshot(path string) error {
	if k.dryRun {
		slog.Info("[DRY-RUN] 将执行快照", "path", path)
		return nil
	}

	slog.Info("开始创建快照", "path", path)

	args := k.buildKopiaArgs("snapshot", "create", path)
	cmd := exec.Command("kopia", args...)
	cmd.Env = k.buildEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("创建快照失败: %w", err)
	}

	slog.Info("快照创建成功")
	return nil
}
