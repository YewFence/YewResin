package schedule

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/YewFence/yewresin/internal/yewresin"
)

const scheduleServiceName = "yewresin-backup"

var (
	resolveConfigPath = yewresin.ResolveConfigPath
	getExecutablePath = os.Executable
	getUserConfigDir  = os.UserConfigDir
	lookPath          = exec.LookPath
	runSystemctl      = defaultRunSystemctl
	mkdirAll          = os.MkdirAll
	writeFile         = os.WriteFile
	readFile          = os.ReadFile
	removeFile        = os.Remove
	stat              = os.Stat
	currentGOOS       = runtime.GOOS
)

func commandLine(configPath string) ([]string, error) {
	exePath, err := getExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return nil, fmt.Errorf("解析可执行文件绝对路径失败: %w", err)
	}

	resolvedConfigPath, err := resolveConfigPath(configPath)
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

func normalizeLineEndings(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func removeFileIfExists(path string) error {
	if !fileExists(path) {
		return nil
	}
	if err := removeFile(path); err != nil {
		return fmt.Errorf("删除文件失败 %s: %w", path, err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := stat(path)
	return err == nil
}
