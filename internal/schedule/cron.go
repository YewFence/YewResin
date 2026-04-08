package schedule

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	defaultCronExpr = "0 3 * * *"
	cronBeginMarker = "# BEGIN YEWRESIN SCHEDULE"
	cronEndMarker   = "# END YEWRESIN SCHEDULE"
)

var (
	readCrontab    = defaultReadCrontab
	installCrontab = defaultInstallCrontab
)

type cronBackend struct{}

func (cronBackend) normalizeInstallOptions(opts *Options) error {
	if opts.OnCalendar != "" {
		return fmt.Errorf("cron 后端不接受 --on-calendar")
	}
	if opts.Expr == "" {
		opts.Expr = defaultCronExpr
	}
	return nil
}

func (cronBackend) ensureSupported() error {
	if currentGOOS == "windows" {
		return fmt.Errorf("Windows 不支持 cron 调度")
	}
	if _, err := lookPath("crontab"); err != nil {
		return fmt.Errorf("未找到 crontab 命令，请先安装 cron")
	}
	return nil
}

func (cronBackend) install(stdout io.Writer, opts Options) error {
	command, err := commandLine(opts.ConfigPath)
	if err != nil {
		return err
	}

	current, err := readCrontab()
	if err != nil {
		return err
	}

	cleaned, _, err := removeManagedBlock(current, cronBeginMarker, cronEndMarker)
	if err != nil {
		return err
	}

	entry := opts.Expr + " " + shellJoin(command)
	managedBlock := cronBeginMarker + "\n" + entry + "\n" + cronEndMarker + "\n"

	var content strings.Builder
	if strings.TrimSpace(cleaned) != "" {
		content.WriteString(strings.TrimRight(cleaned, "\n"))
		content.WriteString("\n\n")
	}
	content.WriteString(managedBlock)

	if err := installCrontab(content.String()); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "已安装 schedule（backend=cron）\n")
	fmt.Fprintf(stdout, "表达式: %s\n", opts.Expr)
	fmt.Fprintf(stdout, "命令: %s\n", shellJoin(command))
	return nil
}

func (cronBackend) uninstall(stdout io.Writer) error {
	current, err := readCrontab()
	if err != nil {
		return err
	}

	cleaned, found, err := removeManagedBlock(current, cronBeginMarker, cronEndMarker)
	if err != nil {
		return err
	}
	if !found {
		fmt.Fprintln(stdout, "当前没有已安装的 cron schedule")
		return nil
	}

	if err := installCrontab(strings.TrimRight(cleaned, "\n") + "\n"); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "已卸载 schedule（backend=cron）")
	return nil
}

func (cronBackend) status(stdout io.Writer) error {
	current, err := readCrontab()
	if err != nil {
		return err
	}

	block, found, err := extractManagedBlock(current, cronBeginMarker, cronEndMarker)
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
