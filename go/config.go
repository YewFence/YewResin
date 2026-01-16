package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config 应用配置
type Config struct {
	// 必填配置
	BaseDir        string   // Docker Compose 项目总目录
	ExpectedRemote string   // Kopia 远程路径

	// 服务管理
	PriorityServices []string // 优先服务列表（最后停，最先启）

	// 文件路径
	LockFile string // 锁文件路径
	LogFile  string // 日志文件路径

	// 通知配置
	DeviceName      string // 设备名称（用于通知标题）
	AppriseURL      string // Apprise 服务地址
	AppriseNotifyURL string // 通知目标 URL

	// Gist 配置
	GistToken         string // GitHub Token
	GistID            string // Gist ID
	GistLogPrefix     string // 日志文件名前缀
	GistMaxLogs       int    // 最大保留日志数
	GistKeepFirstFile bool   // 清理时保留第一个文件

	// Kopia
	KopiaPassword string // Kopia 仓库密码
}

// LoadConfig 从 .env 文件和环境变量加载配置
func LoadConfig(configPath string) (*Config, error) {
	originalPath := configPath
	// 确定配置文件路径
	if configPath == "" {
		// 默认使用程序所在目录的 .env
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("获取程序路径失败: %w", err)
		}
		configPath = filepath.Join(filepath.Dir(exe), ".env")
	}

	// 加载 .env 文件（如果存在）
	if _, err := os.Stat(configPath); err != nil {
		if originalPath == "" && os.IsNotExist(err) {
			// 默认路径不存在时允许继续
		} else if originalPath == "" {
			return nil, fmt.Errorf("检查配置文件失败: %w", err)
		} else {
			return nil, fmt.Errorf("配置文件不存在或不可访问: %w", err)
		}
	} else {
		if err := godotenv.Load(configPath); err != nil {
			return nil, fmt.Errorf("加载配置文件失败: %w", err)
		}
	}

	cfg := &Config{
		// 必填项
		BaseDir:        os.Getenv("BASE_DIR"),
		ExpectedRemote: os.Getenv("EXPECTED_REMOTE"),

		// 默认值
		LockFile:      getEnvDefault("LOCK_FILE", "/tmp/backup_maintenance.lock"),
		LogFile:       getEnvDefault("LOG_FILE", ""),

		// 通知
		DeviceName:       os.Getenv("DEVICE_NAME"),
		AppriseURL:       os.Getenv("APPRISE_URL"),
		AppriseNotifyURL: os.Getenv("APPRISE_NOTIFY_URL"),

		// Gist
		GistToken:         os.Getenv("GIST_TOKEN"),
		GistID:            os.Getenv("GIST_ID"),
		GistLogPrefix:     getEnvDefault("GIST_LOG_PREFIX", "yewresin-backup"),
		GistMaxLogs:       getEnvInt("GIST_MAX_LOGS", 30),
		GistKeepFirstFile: getEnvBool("GIST_KEEP_FIRST_FILE", false),

		// Kopia
		KopiaPassword: os.Getenv("KOPIA_PASSWORD"),
	}

	// 解析优先服务列表
	priorityList := getEnvDefault("PRIORITY_SERVICES_LIST", "caddy nginx gateway")
	cfg.PriorityServices = strings.Fields(priorityList)

	// 验证必填项
	if cfg.BaseDir == "" {
		return nil, fmt.Errorf("BASE_DIR 未设置")
	}
	if cfg.ExpectedRemote == "" {
		return nil, fmt.Errorf("EXPECTED_REMOTE 未设置")
	}

	// 检查 BASE_DIR 是否存在
	if info, err := os.Stat(cfg.BaseDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("BASE_DIR 目录不存在: %s", cfg.BaseDir)
	}

	return cfg, nil
}

// Print 打印配置信息（敏感信息脱敏）
func (c *Config) Print() {
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("当前配置信息")
	fmt.Println("==========================================")

	printField("BASE_DIR(工作目录)", c.BaseDir)
	printField("EXPECTED_REMOTE(Kopia远程仓库)", c.ExpectedRemote)
	printField("PRIORITY_SERVICES(优先服务)", strings.Join(c.PriorityServices, ", "))
	printField("LOCK_FILE(锁文件路径)", c.LockFile)

	if c.DeviceName != "" {
		printField("DEVICE_NAME(设备名称)", c.DeviceName)
	}

	if c.GistToken != "" && c.GistID != "" {
		printField("GIST_ID", c.GistID)
		printField("GIST_TOKEN", "******(已配置)")
	}

	if c.KopiaPassword != "" {
		printField("KOPIA_PASSWORD", "******(已配置)")
	}

	if c.AppriseURL != "" {
		printField("APPRISE_URL", maskString(c.AppriseURL))
	}

	fmt.Println("==========================================")
	fmt.Println()
}

// 辅助函数

func getEnvDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		val = strings.ToLower(val)
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

func printField(name, value string) {
	fmt.Printf("  %-40s %s\n", name+":", value)
}

func maskString(s string) string {
	if len(s) <= 15 {
		return "****(已配置)"
	}
	return s[:8] + "..." + s[len(s)-4:]
}
