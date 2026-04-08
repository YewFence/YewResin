package schedule

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultSystemdCalendar = "*-*-* 03:00:00"

type systemdUserBackend struct{}

func (systemdUserBackend) normalizeInstallOptions(opts *Options) error {
	if opts.Expr != "" {
		return fmt.Errorf("systemd-user 后端不接受 --expr")
	}
	if opts.OnCalendar == "" {
		opts.OnCalendar = defaultSystemdCalendar
	}
	return nil
}

func (systemdUserBackend) ensureSupported() error {
	if currentGOOS != "linux" {
		return fmt.Errorf("systemd-user 调度仅支持 Linux")
	}
	if _, err := lookPath("systemctl"); err != nil {
		return fmt.Errorf("未找到 systemctl 命令")
	}
	return nil
}

func (systemdUserBackend) install(stdout io.Writer, opts Options) error {
	command, err := commandLine(opts.ConfigPath)
	if err != nil {
		return err
	}

	servicePath, timerPath, err := systemdUserUnitPaths()
	if err != nil {
		return err
	}

	if err := mkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		return fmt.Errorf("创建 systemd user 目录失败: %w", err)
	}

	if err := writeFile(servicePath, []byte(renderSystemdService(command)), 0o644); err != nil {
		return fmt.Errorf("写入 service 文件失败: %w", err)
	}
	if err := writeFile(timerPath, []byte(renderSystemdTimer(opts.OnCalendar)), 0o644); err != nil {
		return fmt.Errorf("写入 timer 文件失败: %w", err)
	}

	if _, err := runSystemctl("daemon-reload"); err != nil {
		return err
	}
	if _, err := runSystemctl("enable", "--now", scheduleTimerUnitName()); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "已安装 schedule（backend=systemd-user）\n")
	fmt.Fprintf(stdout, "service: %s\n", servicePath)
	fmt.Fprintf(stdout, "timer: %s\n", timerPath)
	fmt.Fprintf(stdout, "OnCalendar: %s\n", opts.OnCalendar)
	fmt.Fprintln(stdout, "如果需要在退出登录后继续触发，请手动执行：sudo loginctl enable-linger <user>")
	return nil
}

func (systemdUserBackend) uninstall(stdout io.Writer) error {
	servicePath, timerPath, err := systemdUserUnitPaths()
	if err != nil {
		return err
	}

	_, disableErr := runSystemctl("disable", "--now", scheduleTimerUnitName())
	if disableErr != nil && !isIgnorableSystemctlError(disableErr) {
		return disableErr
	}

	if err := removeFileIfExists(timerPath); err != nil {
		return err
	}
	if err := removeFileIfExists(servicePath); err != nil {
		return err
	}

	if _, err := runSystemctl("daemon-reload"); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "已卸载 schedule（backend=systemd-user）")
	return nil
}

func (systemdUserBackend) status(stdout io.Writer) error {
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

	fmt.Fprintln(stdout, "schedule 状态：已安装（backend=systemd-user）")
	fmt.Fprintf(stdout, "service: %s\n", servicePath)
	fmt.Fprintf(stdout, "timer: %s\n", timerPath)
	fmt.Fprintf(stdout, "enabled: %s\n", systemctlStateOrUnknown("is-enabled", scheduleTimerUnitName()))
	fmt.Fprintf(stdout, "active: %s\n", systemctlStateOrUnknown("is-active", scheduleTimerUnitName()))

	timerContent, err := readFile(timerPath)
	if err == nil {
		onCalendar := parseSystemdTimerValue(string(timerContent), "OnCalendar")
		if onCalendar != "" {
			fmt.Fprintf(stdout, "OnCalendar: %s\n", onCalendar)
		}
	}

	return nil
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
	userConfigDir, err := getUserConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("获取用户配置目录失败: %w", err)
	}
	if userConfigDir == "" {
		return "", "", fmt.Errorf("用户配置目录为空")
	}

	unitDir := filepath.Join(userConfigDir, "systemd", "user")
	return filepath.Join(unitDir, scheduleServiceUnitName()+".service"), filepath.Join(unitDir, scheduleTimerUnitName()), nil
}

func scheduleServiceUnitName() string {
	return scheduleServiceName
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
	output, err := runSystemctl(args...)
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

func isIgnorableSystemctlError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not loaded") ||
		strings.Contains(message, "not found") ||
		strings.Contains(message, "no such file")
}
