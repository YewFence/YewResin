package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/YewFence/yewresin/internal/yewresin"
)

const (
	defaultScheduleBackend  = "cron"
	defaultCronExpr         = "0 3 * * *"
	defaultSystemdCalendar  = "*-*-* 03:00:00"
	scheduleServiceName     = "yewresin-backup"
	scheduleCronBeginMarker = "# BEGIN YEWRESIN SCHEDULE"
	scheduleCronEndMarker   = "# END YEWRESIN SCHEDULE"
)

var (
	scheduleResolveConfigPath = yewresin.ResolveConfigPath
	scheduleGetExecutablePath = os.Executable
	scheduleGetUserConfigDir  = os.UserConfigDir
	scheduleLookPath          = exec.LookPath
	scheduleReadCrontab       = defaultReadCrontab
	scheduleInstallCrontab    = defaultInstallCrontab
	scheduleRunSystemctl      = defaultRunSystemctl
	scheduleMkdirAll          = os.MkdirAll
	scheduleWriteFile         = os.WriteFile
	scheduleReadFile          = os.ReadFile
	scheduleRemoveFile        = os.Remove
	scheduleStat              = os.Stat
	scheduleGOOS              = runtime.GOOS
)

type scheduleOptions struct {
	backend    string
	expr       string
	onCalendar string
	configPath string
}

func runScheduleCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printScheduleUsage(stderr)
		return 2
	}

	switch args[0] {
	case "install":
		opts, code := parseScheduleInstallFlags(args[1:], stderr)
		if code != 0 {
			return code
		}
		if err := runScheduleInstall(stdout, *opts); err != nil {
			fmt.Fprintf(stderr, "安装调度失败: %v\n", err)
			return 1
		}
		return 0
	case "uninstall":
		opts, code := parseScheduleCommonFlags("schedule uninstall", args[1:], stderr)
		if code != 0 {
			return code
		}
		if err := runScheduleUninstall(stdout, *opts); err != nil {
			fmt.Fprintf(stderr, "卸载调度失败: %v\n", err)
			return 1
		}
		return 0
	case "status":
		opts, code := parseScheduleCommonFlags("schedule status", args[1:], stderr)
		if code != 0 {
			return code
		}
		if err := runScheduleStatus(stdout, *opts); err != nil {
			fmt.Fprintf(stderr, "查询调度状态失败: %v\n", err)
			return 1
		}
		return 0
	case "help", "--help", "-h":
		printScheduleUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "未知的 schedule 子命令: %s\n", args[0])
		printScheduleUsage(stderr)
		return 2
	}
}

func printScheduleUsage(w io.Writer) {
	fmt.Fprintf(w, "用法: %s schedule <install|uninstall|status> [选项]\n\n", os.Args[0])
	fmt.Fprintln(w, "子命令:")
	fmt.Fprintln(w, "  install     安装当前用户的定时调度，默认后端为 cron")
	fmt.Fprintln(w, "  uninstall   卸载当前用户的定时调度")
	fmt.Fprintln(w, "  status      查看当前用户的定时调度状态")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "常用选项:")
	fmt.Fprintln(w, "  --backend <cron|systemd-user>   选择调度后端，默认 cron")
	fmt.Fprintln(w, "  --config <path>                 写入调度命令中的配置文件路径")
	fmt.Fprintln(w, "  --expr <cron>                   cron 表达式，仅 install + cron 使用")
	fmt.Fprintln(w, "  --on-calendar <expr>            systemd OnCalendar，仅 install + systemd-user 使用")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "示例:")
	fmt.Fprintf(w, "  %s schedule install\n", os.Args[0])
	fmt.Fprintf(w, "  %s schedule install --expr \"0 */6 * * *\"\n", os.Args[0])
	fmt.Fprintf(w, "  %s schedule install --backend systemd-user --on-calendar \"*-*-* 03:00:00\"\n", os.Args[0])
}

func parseScheduleInstallFlags(args []string, stderr io.Writer) (*scheduleOptions, int) {
	flags := flag.NewFlagSet("schedule install", flag.ContinueOnError)
	flags.SetOutput(stderr)

	backend := flags.String("backend", defaultScheduleBackend, "调度后端：cron 或 systemd-user")
	expr := flags.String("expr", "", "cron 表达式（cron 后端默认 0 3 * * *）")
	onCalendar := flags.String("on-calendar", "", "systemd OnCalendar（systemd-user 后端默认 *-*-* 03:00:00）")
	configPath := flags.String("config", "", "写入调度命令的配置文件路径")

	if err := flags.Parse(args); err != nil {
		return nil, 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintln(stderr, "schedule install 不接受位置参数")
		return nil, 2
	}

	opts := &scheduleOptions{
		backend:    strings.TrimSpace(*backend),
		expr:       strings.TrimSpace(*expr),
		onCalendar: strings.TrimSpace(*onCalendar),
		configPath: strings.TrimSpace(*configPath),
	}

	if err := normalizeScheduleInstallOptions(opts); err != nil {
		fmt.Fprintln(stderr, err)
		return nil, 2
	}

	return opts, 0
}

func parseScheduleCommonFlags(name string, args []string, stderr io.Writer) (*scheduleOptions, int) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)

	backend := flags.String("backend", defaultScheduleBackend, "调度后端：cron 或 systemd-user")

	if err := flags.Parse(args); err != nil {
		return nil, 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "%s 不接受位置参数\n", name)
		return nil, 2
	}

	opts := &scheduleOptions{backend: strings.TrimSpace(*backend)}
	if opts.backend == "" {
		opts.backend = defaultScheduleBackend
	}

	return opts, 0
}

func normalizeScheduleInstallOptions(opts *scheduleOptions) error {
	if opts.backend == "" {
		opts.backend = defaultScheduleBackend
	}

	switch opts.backend {
	case "cron":
		if opts.onCalendar != "" {
			return fmt.Errorf("cron 后端不接受 --on-calendar")
		}
		if opts.expr == "" {
			opts.expr = defaultCronExpr
		}
	case "systemd-user":
		if opts.expr != "" {
			return fmt.Errorf("systemd-user 后端不接受 --expr")
		}
		if opts.onCalendar == "" {
			opts.onCalendar = defaultSystemdCalendar
		}
	default:
		return fmt.Errorf("不支持的调度后端: %s（仅支持 cron、systemd-user）", opts.backend)
	}

	return nil
}

func runScheduleInstall(stdout io.Writer, opts scheduleOptions) error {
	if err := ensureScheduleBackendSupported(opts.backend); err != nil {
		return err
	}

	switch opts.backend {
	case "cron":
		return installCronSchedule(stdout, opts)
	case "systemd-user":
		return installSystemdUserSchedule(stdout, opts)
	default:
		return fmt.Errorf("不支持的调度后端: %s", opts.backend)
	}
}

func runScheduleUninstall(stdout io.Writer, opts scheduleOptions) error {
	if err := ensureScheduleBackendSupported(opts.backend); err != nil {
		return err
	}

	switch opts.backend {
	case "cron":
		return uninstallCronSchedule(stdout)
	case "systemd-user":
		return uninstallSystemdUserSchedule(stdout)
	default:
		return fmt.Errorf("不支持的调度后端: %s", opts.backend)
	}
}

func runScheduleStatus(stdout io.Writer, opts scheduleOptions) error {
	if err := ensureScheduleBackendSupported(opts.backend); err != nil {
		return err
	}

	switch opts.backend {
	case "cron":
		return statusCronSchedule(stdout)
	case "systemd-user":
		return statusSystemdUserSchedule(stdout)
	default:
		return fmt.Errorf("不支持的调度后端: %s", opts.backend)
	}
}

func ensureScheduleBackendSupported(backend string) error {
	switch backend {
	case "cron":
		if scheduleGOOS == "windows" {
			return fmt.Errorf("Windows 不支持 cron 调度")
		}
		if _, err := scheduleLookPath("crontab"); err != nil {
			return fmt.Errorf("未找到 crontab 命令，请先安装 cron")
		}
	case "systemd-user":
		if scheduleGOOS != "linux" {
			return fmt.Errorf("systemd-user 调度仅支持 Linux")
		}
		if _, err := scheduleLookPath("systemctl"); err != nil {
			return fmt.Errorf("未找到 systemctl 命令")
		}
	default:
		return fmt.Errorf("不支持的调度后端: %s（仅支持 cron、systemd-user）", backend)
	}

	return nil
}

func installCronSchedule(stdout io.Writer, opts scheduleOptions) error {
	command, err := scheduleCommandLine(opts.configPath)
	if err != nil {
		return err
	}

	current, err := scheduleReadCrontab()
	if err != nil {
		return err
	}

	cleaned, _, err := removeManagedBlock(current, scheduleCronBeginMarker, scheduleCronEndMarker)
	if err != nil {
		return err
	}

	entry := opts.expr + " " + shellJoin(command)
	managedBlock := scheduleCronBeginMarker + "\n" + entry + "\n" + scheduleCronEndMarker + "\n"

	var content strings.Builder
	if strings.TrimSpace(cleaned) != "" {
		content.WriteString(strings.TrimRight(cleaned, "\n"))
		content.WriteString("\n\n")
	}
	content.WriteString(managedBlock)

	if err := scheduleInstallCrontab(content.String()); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "已安装 schedule（backend=cron）\n")
	fmt.Fprintf(stdout, "表达式: %s\n", opts.expr)
	fmt.Fprintf(stdout, "命令: %s\n", shellJoin(command))
	return nil
}

func uninstallCronSchedule(stdout io.Writer) error {
	current, err := scheduleReadCrontab()
	if err != nil {
		return err
	}

	cleaned, found, err := removeManagedBlock(current, scheduleCronBeginMarker, scheduleCronEndMarker)
	if err != nil {
		return err
	}
	if !found {
		fmt.Fprintln(stdout, "当前没有已安装的 cron schedule")
		return nil
	}

	if err := scheduleInstallCrontab(strings.TrimRight(cleaned, "\n") + "\n"); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "已卸载 schedule（backend=cron）")
	return nil
}

func statusCronSchedule(stdout io.Writer) error {
	current, err := scheduleReadCrontab()
	if err != nil {
		return err
	}

	block, found, err := extractManagedBlock(current, scheduleCronBeginMarker, scheduleCronEndMarker)
	if err != nil {
		return err
	}
	if !found {
		fmt.Fprintln(stdout, "schedule 状态：未安装（backend=cron）")
		return nil
	}

	fmt.Fprintln(stdout, "schedule 状态：已安装（backend=cron）")
	fmt.Fprintln(stdout, strings.TrimSpace(block))
	return nil
}

func installSystemdUserSchedule(stdout io.Writer, opts scheduleOptions) error {
	command, err := scheduleCommandLine(opts.configPath)
	if err != nil {
		return err
	}

	servicePath, timerPath, err := systemdUserUnitPaths()
	if err != nil {
		return err
	}

	if err := scheduleMkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		return fmt.Errorf("创建 systemd user 目录失败: %w", err)
	}

	serviceContent := renderSystemdService(command)
	timerContent := renderSystemdTimer(opts.onCalendar)

	if err := scheduleWriteFile(servicePath, []byte(serviceContent), 0o644); err != nil {
		return fmt.Errorf("写入 service 文件失败: %w", err)
	}
	if err := scheduleWriteFile(timerPath, []byte(timerContent), 0o644); err != nil {
		return fmt.Errorf("写入 timer 文件失败: %w", err)
	}

	if _, err := scheduleRunSystemctl("daemon-reload"); err != nil {
		return err
	}
	if _, err := scheduleRunSystemctl("enable", "--now", scheduleTimerUnitName()); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "已安装 schedule（backend=systemd-user）\n")
	fmt.Fprintf(stdout, "service: %s\n", servicePath)
	fmt.Fprintf(stdout, "timer: %s\n", timerPath)
	fmt.Fprintf(stdout, "OnCalendar: %s\n", opts.onCalendar)
	fmt.Fprintln(stdout, "如果需要在退出登录后继续触发，请手动执行：sudo loginctl enable-linger <user>")
	return nil
}

func uninstallSystemdUserSchedule(stdout io.Writer) error {
	servicePath, timerPath, err := systemdUserUnitPaths()
	if err != nil {
		return err
	}

	_, disableErr := scheduleRunSystemctl("disable", "--now", scheduleTimerUnitName())
	if disableErr != nil && !isIgnorableSystemctlError(disableErr) {
		return disableErr
	}

	if err := removeFileIfExists(timerPath); err != nil {
		return err
	}
	if err := removeFileIfExists(servicePath); err != nil {
		return err
	}

	if _, err := scheduleRunSystemctl("daemon-reload"); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "已卸载 schedule（backend=systemd-user）")
	return nil
}

func statusSystemdUserSchedule(stdout io.Writer) error {
	servicePath, timerPath, err := systemdUserUnitPaths()
	if err != nil {
		return err
	}

	serviceExists := fileExists(servicePath)
	timerExists := fileExists(timerPath)
	if !serviceExists && !timerExists {
		fmt.Fprintln(stdout, "schedule 状态：未安装（backend=systemd-user）")
		return nil
	}

	enabled := systemctlStateOrUnknown("is-enabled", scheduleTimerUnitName())
	active := systemctlStateOrUnknown("is-active", scheduleTimerUnitName())

	fmt.Fprintln(stdout, "schedule 状态：已安装（backend=systemd-user）")
	fmt.Fprintf(stdout, "service: %s\n", servicePath)
	fmt.Fprintf(stdout, "timer: %s\n", timerPath)
	fmt.Fprintf(stdout, "enabled: %s\n", enabled)
	fmt.Fprintf(stdout, "active: %s\n", active)

	timerContent, err := scheduleReadFile(timerPath)
	if err == nil {
		onCalendar := parseSystemdTimerValue(string(timerContent), "OnCalendar")
		if onCalendar != "" {
			fmt.Fprintf(stdout, "OnCalendar: %s\n", onCalendar)
		}
	}

	return nil
}

func scheduleCommandLine(configPath string) ([]string, error) {
	exePath, err := scheduleGetExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return nil, fmt.Errorf("解析可执行文件绝对路径失败: %w", err)
	}

	resolvedConfigPath, err := scheduleResolveConfigPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件路径失败: %w", err)
	}
	if resolvedConfigPath != "" {
		resolvedConfigPath, err = filepath.Abs(resolvedConfigPath)
		if err != nil {
			return nil, fmt.Errorf("解析配置文件绝对路径失败: %w", err)
		}
	}

	command := []string{exePath}
	if resolvedConfigPath != "" {
		command = append(command, "--config", resolvedConfigPath)
	}
	command = append(command, "-y")
	return command, nil
}

func removeManagedBlock(content, beginMarker, endMarker string) (string, bool, error) {
	lines := strings.Split(normalizeLineEndings(content), "\n")
	filtered := make([]string, 0, len(lines))
	inBlock := false
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch trimmed {
		case beginMarker:
			if inBlock {
				return "", false, fmt.Errorf("检测到重复的调度起始标记")
			}
			inBlock = true
			found = true
			continue
		case endMarker:
			if !inBlock {
				return "", false, fmt.Errorf("检测到孤立的调度结束标记")
			}
			inBlock = false
			continue
		}

		if !inBlock {
			filtered = append(filtered, line)
		}
	}

	if inBlock {
		return "", false, fmt.Errorf("调度标记未正确闭合")
	}

	cleaned := strings.TrimSpace(strings.Join(filtered, "\n"))
	if cleaned == "" {
		return "", found, nil
	}

	return cleaned + "\n", found, nil
}

func extractManagedBlock(content, beginMarker, endMarker string) (string, bool, error) {
	lines := strings.Split(normalizeLineEndings(content), "\n")
	var block []string
	inBlock := false
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch trimmed {
		case beginMarker:
			if inBlock {
				return "", false, fmt.Errorf("检测到重复的调度起始标记")
			}
			inBlock = true
			found = true
			block = append(block, line)
			continue
		case endMarker:
			if !inBlock {
				return "", false, fmt.Errorf("检测到孤立的调度结束标记")
			}
			block = append(block, line)
			inBlock = false
			return strings.Join(block, "\n"), true, nil
		}

		if inBlock {
			block = append(block, line)
		}
	}

	if inBlock {
		return "", false, fmt.Errorf("调度标记未正确闭合")
	}

	return "", found, nil
}

func defaultReadCrontab() (string, error) {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			message := strings.ToLower(strings.TrimSpace(string(output)))
			if strings.Contains(message, "no crontab") {
				return "", nil
			}
		}
		return "", fmt.Errorf("读取当前 crontab 失败: %s", strings.TrimSpace(string(output)))
	}

	return normalizeLineEndings(string(output)), nil
}

func defaultInstallCrontab(content string) error {
	tempFile, err := os.CreateTemp("", "yewresin-crontab-*")
	if err != nil {
		return fmt.Errorf("创建临时 crontab 文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		return fmt.Errorf("写入临时 crontab 文件失败: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("关闭临时 crontab 文件失败: %w", err)
	}

	cmd := exec.Command("crontab", tempPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("写入 crontab 失败: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func renderSystemdService(command []string) string {
	return strings.Join([]string{
		"[Unit]",
		"Description=YewResin Docker Backup",
		"After=docker.service",
		"Requires=docker.service",
		"",
		"[Service]",
		"Type=oneshot",
		"ExecStart=" + systemdJoin(command),
		"StandardOutput=journal",
		"StandardError=journal",
		"",
	}, "\n")
}

func renderSystemdTimer(onCalendar string) string {
	return strings.Join([]string{
		"[Unit]",
		"Description=Run YewResin backup",
		"",
		"[Timer]",
		"OnCalendar=" + onCalendar,
		"Persistent=true",
		"RandomizedDelaySec=300",
		"",
		"[Install]",
		"WantedBy=timers.target",
		"",
	}, "\n")
}

func systemdJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, strconv.Quote(arg))
	}
	return strings.Join(quoted, " ")
}

func systemdUserUnitPaths() (string, string, error) {
	userConfigDir, err := scheduleGetUserConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("获取用户配置目录失败: %w", err)
	}
	if userConfigDir == "" {
		return "", "", fmt.Errorf("用户配置目录为空")
	}

	unitDir := filepath.Join(userConfigDir, "systemd", "user")
	return filepath.Join(unitDir, scheduleServiceUnitName()), filepath.Join(unitDir, scheduleTimerUnitName()), nil
}

func scheduleServiceUnitName() string {
	return scheduleServiceName + ".service"
}

func scheduleTimerUnitName() string {
	return scheduleServiceName + ".timer"
}

func defaultRunSystemctl(args ...string) (string, error) {
	commandArgs := append([]string{"--user"}, args...)
	cmd := exec.Command("systemctl", commandArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			trimmed = err.Error()
		}
		return "", fmt.Errorf("执行 systemctl %s 失败: %s", strings.Join(commandArgs, " "), trimmed)
	}
	return strings.TrimSpace(string(output)), nil
}

func systemctlStateOrUnknown(args ...string) string {
	output, err := scheduleRunSystemctl(args...)
	if err != nil {
		return "unknown"
	}
	if strings.TrimSpace(output) == "" {
		return "unknown"
	}
	return strings.TrimSpace(output)
}

func parseSystemdTimerValue(content, key string) string {
	lines := strings.Split(normalizeLineEndings(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		prefix := key + "="
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

func normalizeLineEndings(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func removeFileIfExists(path string) error {
	if !fileExists(path) {
		return nil
	}
	if err := scheduleRemoveFile(path); err != nil {
		return fmt.Errorf("删除文件失败 %s: %w", path, err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := scheduleStat(path)
	return err == nil
}

func isIgnorableSystemctlError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not loaded") ||
		strings.Contains(message, "not found") ||
		strings.Contains(message, "no such file")
}
