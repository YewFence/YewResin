package yewresin

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultPriorityServices            = "caddy nginx gateway"
	defaultLockFile                    = "/tmp/backup_maintenance.lock"
	defaultDockerCommandTimeoutSeconds = 120
	defaultGistLogPrefix               = "yewresin-backup"
	defaultGistMaxLogs                 = 30
	defaultConfigDirName               = "yewresin"
)

var (
	getExecutablePath = os.Executable
	getUserConfigDir  = os.UserConfigDir
)

// Config 应用配置
type Config struct {
	// 必填配置
	BaseDir        string // Docker Compose 项目总目录
	ExpectedRemote string // Kopia 远程路径

	// 服务管理
	PriorityServices            []string // 优先服务列表（最后停，最先启）
	DockerCommandTimeoutSeconds int      // Docker 命令超时时间（秒）

	// 文件路径
	LockFile string // 锁文件路径
	LogFile  string // 日志文件路径

	// 通知配置
	DeviceName       string // 设备名称（用于通知标题）
	AppriseURL       string // Apprise 服务地址
	AppriseNotifyURL string // 通知目标 URL

	// Gist 配置
	GistToken         string // GitHub Token
	GistID            string // Gist ID
	GistLogPrefix     string // 日志文件名前缀
	GistMaxLogs       int    // 最大保留日志数
	GistKeepFirstFile bool   // 清理时保留第一个文件

	// Kopia / Rclone
	KopiaPassword   string // Kopia 仓库密码（仅用于子进程环境）
	KopiaConfigFile string // Kopia 配置文件路径
	RcloneConfig    string // Rclone 配置文件路径
}

type fileConfig struct {
	BaseDir        string
	ExpectedRemote string

	PriorityServices    []string
	HasPriorityServices bool

	LockFile string
	LogFile  string

	DockerCommandTimeoutSeconds *int

	DeviceName       string
	AppriseURL       string
	AppriseNotifyURL string

	GistToken         string
	GistID            string
	GistLogPrefix     string
	GistMaxLogs       *int
	GistKeepFirstFile *bool

	KopiaPassword   string
	KopiaConfigFile string
	RcloneConfig    string
}

// LoadConfig 从环境变量和配置文件加载配置。
// 优先级：环境变量 > 显式配置文件 > 用户配置目录 > 程序同目录 > 内置默认值。
func LoadConfig(configPath string) (*Config, error) {
	resolvedConfigPath, err := resolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}

	fileCfg := &fileConfig{}
	if resolvedConfigPath != "" {
		fileCfg, err = loadConfigFile(resolvedConfigPath)
		if err != nil {
			return nil, err
		}
	}

	cfg := &Config{
		BaseDir:        pickString("BASE_DIR", fileCfg.BaseDir, ""),
		ExpectedRemote: pickString("EXPECTED_REMOTE", fileCfg.ExpectedRemote, ""),

		LockFile:                    pickString("LOCK_FILE", fileCfg.LockFile, defaultLockFile),
		LogFile:                     pickString("LOG_FILE", fileCfg.LogFile, ""),
		DockerCommandTimeoutSeconds: pickInt("DOCKER_COMMAND_TIMEOUT_SECONDS", fileCfg.DockerCommandTimeoutSeconds, defaultDockerCommandTimeoutSeconds),

		DeviceName:       pickString("DEVICE_NAME", fileCfg.DeviceName, ""),
		AppriseURL:       pickString("APPRISE_URL", fileCfg.AppriseURL, ""),
		AppriseNotifyURL: pickString("APPRISE_NOTIFY_URL", fileCfg.AppriseNotifyURL, ""),

		GistToken:         pickString("GIST_TOKEN", fileCfg.GistToken, ""),
		GistID:            pickString("GIST_ID", fileCfg.GistID, ""),
		GistLogPrefix:     pickString("GIST_LOG_PREFIX", fileCfg.GistLogPrefix, defaultGistLogPrefix),
		GistMaxLogs:       pickInt("GIST_MAX_LOGS", fileCfg.GistMaxLogs, defaultGistMaxLogs),
		GistKeepFirstFile: pickBool("GIST_KEEP_FIRST_FILE", fileCfg.GistKeepFirstFile, true),

		KopiaPassword:   pickString("KOPIA_PASSWORD", fileCfg.KopiaPassword, ""),
		KopiaConfigFile: pickString("KOPIA_CONFIG_FILE", fileCfg.KopiaConfigFile, ""),
		RcloneConfig:    pickString("RCLONE_CONFIG", fileCfg.RcloneConfig, ""),
	}

	cfg.PriorityServices = pickPriorityServices(fileCfg)

	if cfg.BaseDir == "" {
		return nil, fmt.Errorf("BASE_DIR 未设置")
	}
	if cfg.ExpectedRemote == "" {
		return nil, fmt.Errorf("EXPECTED_REMOTE 未设置")
	}

	if info, err := os.Stat(cfg.BaseDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("BASE_DIR 目录不存在: %s", cfg.BaseDir)
	}

	return cfg, nil
}

func resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		info, err := os.Stat(configPath)
		if err != nil {
			return "", fmt.Errorf("配置文件不存在或不可访问: %w", err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("配置文件路径不能是目录: %s", configPath)
		}
		return configPath, nil
	}

	candidates, err := defaultConfigCandidates()
	if err != nil {
		return "", err
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("默认配置路径不能是目录: %s", candidate)
			}
			return candidate, nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("检查配置文件失败: %w", err)
		}
	}

	return "", nil
}

func defaultConfigCandidates() ([]string, error) {
	var candidates []string

	if userConfigDir, err := getUserConfigDir(); err == nil && userConfigDir != "" {
		configDir := filepath.Join(userConfigDir, defaultConfigDirName)
		candidates = append(candidates,
			filepath.Join(configDir, "config.toml"),
			filepath.Join(configDir, ".env"),
		)
	}

	exe, err := getExecutablePath()
	if err != nil {
		if len(candidates) > 0 {
			return candidates, nil
		}
		return nil, fmt.Errorf("获取程序路径失败: %w", err)
	}

	exeDir := filepath.Dir(exe)
	candidates = append(candidates,
		filepath.Join(exeDir, "config.toml"),
		filepath.Join(exeDir, ".env"),
	)

	return candidates, nil
}

func loadConfigFile(configPath string) (*fileConfig, error) {
	ext := strings.ToLower(filepath.Ext(configPath))

	switch {
	case ext == ".toml":
		return loadTOMLConfig(configPath)
	case ext == ".env" || strings.EqualFold(filepath.Base(configPath), ".env"):
		return loadEnvConfig(configPath)
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s（仅支持 .env 和 .toml）", configPath)
	}
}

func loadEnvConfig(configPath string) (*fileConfig, error) {
	values, err := godotenv.Read(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}

	cfg := &fileConfig{
		BaseDir:        values["BASE_DIR"],
		ExpectedRemote: values["EXPECTED_REMOTE"],
		LockFile:       values["LOCK_FILE"],
		LogFile:        values["LOG_FILE"],

		DeviceName:       values["DEVICE_NAME"],
		AppriseURL:       values["APPRISE_URL"],
		AppriseNotifyURL: values["APPRISE_NOTIFY_URL"],

		GistToken:     values["GIST_TOKEN"],
		GistID:        values["GIST_ID"],
		GistLogPrefix: values["GIST_LOG_PREFIX"],

		KopiaPassword:   values["KOPIA_PASSWORD"],
		KopiaConfigFile: values["KOPIA_CONFIG_FILE"],
		RcloneConfig:    values["RCLONE_CONFIG"],
	}

	if priorityList := values["PRIORITY_SERVICES_LIST"]; priorityList != "" {
		cfg.PriorityServices = strings.Fields(priorityList)
		cfg.HasPriorityServices = true
	}
	if timeout := parseOptionalInt(values["DOCKER_COMMAND_TIMEOUT_SECONDS"]); timeout != nil {
		cfg.DockerCommandTimeoutSeconds = timeout
	}
	if maxLogs := parseOptionalInt(values["GIST_MAX_LOGS"]); maxLogs != nil {
		cfg.GistMaxLogs = maxLogs
	}
	if keepFirstFile := parseOptionalBool(values["GIST_KEEP_FIRST_FILE"]); keepFirstFile != nil {
		cfg.GistKeepFirstFile = keepFirstFile
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
	printField("DOCKER_COMMAND_TIMEOUT_SECONDS(Docker超时秒)", fmt.Sprintf("%d", c.DockerCommandTimeoutSeconds))
	printField("LOCK_FILE(锁文件路径)", c.LockFile)

	if c.LogFile != "" {
		printField("LOG_FILE(日志文件)", c.LogFile)
	}
	if c.DeviceName != "" {
		printField("DEVICE_NAME(设备名称)", c.DeviceName)
	}
	if c.KopiaPassword != "" {
		printField("KOPIA_PASSWORD", "******(已配置)")
	}
	if c.GistToken != "" && c.GistID != "" {
		printField("GIST_ID", c.GistID)
		printField("GIST_TOKEN", "******(已配置)")
	}
	if c.KopiaConfigFile != "" {
		printField("KOPIA_CONFIG_FILE(Kopia配置文件)", c.KopiaConfigFile)
	}
	if c.RcloneConfig != "" {
		printField("RCLONE_CONFIG(Rclone配置文件)", c.RcloneConfig)
	}
	if c.AppriseURL != "" {
		printField("APPRISE_URL", maskString(c.AppriseURL))
	}

	fmt.Println("==========================================")
	fmt.Println()
}

func pickString(envKey, fileValue, defaultValue string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	if fileValue != "" {
		return fileValue
	}
	return defaultValue
}

func pickInt(envKey string, fileValue *int, defaultValue int) int {
	if value := parseOptionalInt(os.Getenv(envKey)); value != nil {
		return *value
	}
	if fileValue != nil {
		return *fileValue
	}
	return defaultValue
}

func pickBool(envKey string, fileValue *bool, defaultValue bool) bool {
	if value := parseOptionalBool(os.Getenv(envKey)); value != nil {
		return *value
	}
	if fileValue != nil {
		return *fileValue
	}
	return defaultValue
}

func pickPriorityServices(fileCfg *fileConfig) []string {
	if priorityList := os.Getenv("PRIORITY_SERVICES_LIST"); priorityList != "" {
		return strings.Fields(priorityList)
	}
	if fileCfg.HasPriorityServices {
		return append([]string(nil), fileCfg.PriorityServices...)
	}
	return strings.Fields(defaultPriorityServices)
}

func parseOptionalInt(value string) *int {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}

	return &parsed
}

func parseOptionalBool(value string) *bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	switch strings.ToLower(value) {
	case "true", "1", "yes":
		result := true
		return &result
	case "false", "0", "no":
		result := false
		return &result
	default:
		return nil
	}
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

func loadTOMLConfig(configPath string) (*fileConfig, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("打开 TOML 配置文件失败: %w", err)
	}
	defer file.Close()

	cfg := &fileConfig{}
	scanner := bufio.NewScanner(file)
	section := ""
	lineNo := 0

	for scanner.Scan() {
		lineNo++

		line := strings.TrimSpace(stripTOMLComment(scanner.Text()))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, fmt.Errorf("TOML 第 %d 行格式错误：section 未闭合", lineNo)
			}

			section = strings.TrimSpace(line[1 : len(line)-1])
			if !isSupportedTOMLSection(section) {
				return nil, fmt.Errorf("TOML 第 %d 行包含不支持的 section: [%s]", lineNo, section)
			}
			continue
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("TOML 第 %d 行格式错误：缺少 '='", lineNo)
		}

		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)
		if key == "" {
			return nil, fmt.Errorf("TOML 第 %d 行格式错误：缺少 key", lineNo)
		}

		if err := assignTOMLValue(cfg, section, key, rawValue); err != nil {
			return nil, fmt.Errorf("TOML 第 %d 行配置无效: %w", lineNo, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 TOML 配置文件失败: %w", err)
	}

	return cfg, nil
}

func isSupportedTOMLSection(section string) bool {
	switch section {
	case "", "logging", "notifications", "gist", "kopia", "rclone":
		return true
	default:
		return false
	}
}

func assignTOMLValue(cfg *fileConfig, section, key, rawValue string) error {
	switch section {
	case "":
		return assignRootTOMLValue(cfg, key, rawValue)
	case "logging":
		return assignLoggingTOMLValue(cfg, key, rawValue)
	case "notifications":
		return assignNotificationsTOMLValue(cfg, key, rawValue)
	case "gist":
		return assignGistTOMLValue(cfg, key, rawValue)
	case "kopia":
		return assignKopiaTOMLValue(cfg, key, rawValue)
	case "rclone":
		return assignRcloneTOMLValue(cfg, key, rawValue)
	default:
		return fmt.Errorf("不支持的 section [%s]", section)
	}
}

func assignRootTOMLValue(cfg *fileConfig, key, rawValue string) error {
	switch key {
	case "base_dir":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.BaseDir = value
	case "expected_remote":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.ExpectedRemote = value
	case "priority_services":
		value, err := parseTOMLStringArray(rawValue)
		if err != nil {
			return err
		}
		cfg.PriorityServices = value
		cfg.HasPriorityServices = true
	case "lock_file":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.LockFile = value
	case "log_file":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.LogFile = value
	case "docker_command_timeout_seconds":
		value, err := parseTOMLInt(rawValue)
		if err != nil {
			return err
		}
		cfg.DockerCommandTimeoutSeconds = &value
	case "device_name":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.DeviceName = value
	case "apprise_url":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.AppriseURL = value
	case "apprise_notify_url":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.AppriseNotifyURL = value
	case "gist_token":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.GistToken = value
	case "gist_id":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.GistID = value
	case "gist_log_prefix":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.GistLogPrefix = value
	case "gist_max_logs":
		value, err := parseTOMLInt(rawValue)
		if err != nil {
			return err
		}
		cfg.GistMaxLogs = &value
	case "gist_keep_first_file":
		value, err := parseTOMLBool(rawValue)
		if err != nil {
			return err
		}
		cfg.GistKeepFirstFile = &value
	case "kopia_password":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.KopiaPassword = value
	case "kopia_config_file":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.KopiaConfigFile = value
	case "rclone_config":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.RcloneConfig = value
	default:
		return fmt.Errorf("未知根级配置项 %q", key)
	}

	return nil
}

func assignLoggingTOMLValue(cfg *fileConfig, key, rawValue string) error {
	switch key {
	case "file":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.LogFile = value
	case "docker_command_timeout_seconds":
		value, err := parseTOMLInt(rawValue)
		if err != nil {
			return err
		}
		cfg.DockerCommandTimeoutSeconds = &value
	default:
		return fmt.Errorf("未知 [logging] 配置项 %q", key)
	}

	return nil
}

func assignNotificationsTOMLValue(cfg *fileConfig, key, rawValue string) error {
	switch key {
	case "device_name":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.DeviceName = value
	case "apprise_url":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.AppriseURL = value
	case "apprise_notify_url":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.AppriseNotifyURL = value
	default:
		return fmt.Errorf("未知 [notifications] 配置项 %q", key)
	}

	return nil
}

func assignGistTOMLValue(cfg *fileConfig, key, rawValue string) error {
	switch key {
	case "token":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.GistToken = value
	case "id":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.GistID = value
	case "log_prefix":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.GistLogPrefix = value
	case "max_logs":
		value, err := parseTOMLInt(rawValue)
		if err != nil {
			return err
		}
		cfg.GistMaxLogs = &value
	case "keep_first_file":
		value, err := parseTOMLBool(rawValue)
		if err != nil {
			return err
		}
		cfg.GistKeepFirstFile = &value
	default:
		return fmt.Errorf("未知 [gist] 配置项 %q", key)
	}

	return nil
}

func assignKopiaTOMLValue(cfg *fileConfig, key, rawValue string) error {
	switch key {
	case "password":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.KopiaPassword = value
	case "config_file":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.KopiaConfigFile = value
	default:
		return fmt.Errorf("未知 [kopia] 配置项 %q", key)
	}

	return nil
}

func assignRcloneTOMLValue(cfg *fileConfig, key, rawValue string) error {
	switch key {
	case "config_file":
		value, err := parseTOMLString(rawValue)
		if err != nil {
			return err
		}
		cfg.RcloneConfig = value
	default:
		return fmt.Errorf("未知 [rclone] 配置项 %q", key)
	}

	return nil
}

func parseTOMLString(rawValue string) (string, error) {
	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" {
		return "", nil
	}

	if len(rawValue) >= 2 {
		switch rawValue[0] {
		case '"':
			if rawValue[len(rawValue)-1] != '"' {
				return "", fmt.Errorf("字符串缺少结束双引号")
			}
			value, err := strconv.Unquote(rawValue)
			if err != nil {
				return "", fmt.Errorf("解析字符串失败: %w", err)
			}
			return value, nil
		case '\'':
			if rawValue[len(rawValue)-1] != '\'' {
				return "", fmt.Errorf("字符串缺少结束单引号")
			}
			return rawValue[1 : len(rawValue)-1], nil
		}
	}

	return rawValue, nil
}

func parseTOMLInt(rawValue string) (int, error) {
	value, err := parseTOMLString(rawValue)
	if err != nil {
		return 0, err
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("整数值无效: %q", rawValue)
	}
	return parsed, nil
}

func parseTOMLBool(rawValue string) (bool, error) {
	value, err := parseTOMLString(rawValue)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("布尔值无效: %q", rawValue)
	}
}

func parseTOMLStringArray(rawValue string) ([]string, error) {
	rawValue = strings.TrimSpace(rawValue)
	if !strings.HasPrefix(rawValue, "[") || !strings.HasSuffix(rawValue, "]") {
		return nil, fmt.Errorf("字符串数组必须使用 [...]")
	}

	content := strings.TrimSpace(rawValue[1 : len(rawValue)-1])
	if content == "" {
		return []string{}, nil
	}

	parts, err := splitTOMLArray(content)
	if err != nil {
		return nil, err
	}

	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value, err := parseTOMLString(part)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, nil
}

func splitTOMLArray(content string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false

	for _, r := range content {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
			continue
		case r == '\\' && inDoubleQuote:
			current.WriteRune(r)
			escaped = true
			continue
		case r == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case r == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case r == ',' && !inDoubleQuote && !inSingleQuote:
			part := strings.TrimSpace(current.String())
			if part == "" {
				return nil, fmt.Errorf("数组中存在空元素")
			}
			parts = append(parts, part)
			current.Reset()
			continue
		}

		current.WriteRune(r)
	}

	if escaped || inDoubleQuote || inSingleQuote {
		return nil, fmt.Errorf("数组字符串未正确闭合")
	}

	part := strings.TrimSpace(current.String())
	if part == "" {
		return nil, fmt.Errorf("数组中存在空元素")
	}

	parts = append(parts, part)
	return parts, nil
}

func stripTOMLComment(line string) string {
	var result strings.Builder
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false

	for _, r := range line {
		switch {
		case escaped:
			result.WriteRune(r)
			escaped = false
			continue
		case r == '\\' && inDoubleQuote:
			result.WriteRune(r)
			escaped = true
			continue
		case r == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case r == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case r == '#' && !inDoubleQuote && !inSingleQuote:
			return result.String()
		}

		result.WriteRune(r)
	}

	return result.String()
}
