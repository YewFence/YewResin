package main

import (
	"bufio"
	_ "embed"
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

//go:embed config.toml.example
var configTemplate string

var (
	getDefaultConfigFilePath = yewresin.DefaultConfigFilePath
	runEditorCommand         = defaultRunEditorCommand
	lookPath                 = exec.LookPath
	getEditorEnv             = func() string { return os.Getenv("EDITOR") }
)

func runConfigCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printConfigUsage(stderr)
		return 2
	}

	switch args[0] {
	case "init":
		initFlags := flag.NewFlagSet("config init", flag.ContinueOnError)
		initFlags.SetOutput(stderr)
		force := initFlags.Bool("force", false, "如果默认配置文件已存在则覆盖")

		if err := initFlags.Parse(args[1:]); err != nil {
			return 2
		}
		if initFlags.NArg() > 0 {
			fmt.Fprintf(stderr, "config init 不接受位置参数\n")
			return 2
		}
		if err := runConfigInit(stdin, stdout, *force); err != nil {
			fmt.Fprintf(stderr, "初始化配置失败: %v\n", err)
			return 1
		}
		return 0
	case "edit":
		if len(args) > 1 {
			fmt.Fprintf(stderr, "config edit 不接受额外参数\n")
			return 2
		}
		if err := runConfigEdit(); err != nil {
			fmt.Fprintf(stderr, "打开配置失败: %v\n", err)
			return 1
		}
		return 0
	case "help", "--help", "-h":
		printConfigUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "未知的 config 子命令: %s\n", args[0])
		printConfigUsage(stderr)
		return 2
	}
}

func printConfigUsage(w io.Writer) {
	fmt.Fprintf(w, "用法: %s config <init|edit>\n\n", os.Args[0])
	fmt.Fprintln(w, "子命令:")
	fmt.Fprintln(w, "  init   初始化默认配置文件，只引导填写必填项")
	fmt.Fprintln(w, "         支持 --force 覆盖已有文件")
	fmt.Fprintln(w, "  edit   使用 EDITOR 打开默认配置文件")
}

func runConfigInit(stdin io.Reader, stdout io.Writer, force bool) error {
	configPath, err := getDefaultConfigFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err == nil {
		if !force {
			return fmt.Errorf("默认配置文件已存在: %s，请直接使用 `yewresin config edit` 修改，或使用 `yewresin config init --force` 覆盖", configPath)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查默认配置文件失败: %w", err)
	}

	reader := bufio.NewReader(stdin)

	fmt.Fprintf(stdout, "准备初始化配置文件：%s\n", configPath)
	fmt.Fprintln(stdout, "先只填写必填项，其他可选配置先保持模板默认状态。")
	fmt.Fprintln(stdout)

	baseDir, err := promptRequiredValue(reader, stdout, "BASE_DIR（Docker Compose 项目总目录）")
	if err != nil {
		return err
	}

	expectedRemote, err := promptRequiredValue(reader, stdout, "EXPECTED_REMOTE（Kopia 远程路径，例如 gdrive:backup）")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("创建默认配置目录失败: %w", err)
	}

	content := renderConfigTemplate(configTemplate, map[string]string{
		"base_dir":        baseDir,
		"expected_remote": expectedRemote,
	})

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	fmt.Fprintln(stdout)
	if force {
		fmt.Fprintf(stdout, "已覆盖配置文件：%s\n", configPath)
	} else {
		fmt.Fprintf(stdout, "已创建配置文件：%s\n", configPath)
	}
	fmt.Fprintln(stdout, "已写入必填项：")
	fmt.Fprintf(stdout, "  base_dir = %s\n", strconv.Quote(baseDir))
	fmt.Fprintf(stdout, "  expected_remote = %s\n", strconv.Quote(expectedRemote))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "下一步可以执行：")
	fmt.Fprintf(stdout, "  %s config edit\n", os.Args[0])

	return nil
}

func promptRequiredValue(reader *bufio.Reader, stdout io.Writer, label string) (string, error) {
	for {
		fmt.Fprintf(stdout, "请输入 %s: ", label)
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("读取输入失败: %w", err)
		}

		value := strings.TrimSpace(line)
		if value != "" {
			return value, nil
		}

		if err == io.EOF {
			return "", fmt.Errorf("%s 不能为空", label)
		}

		fmt.Fprintln(stdout, "这个值是必填的，别留空呀。")
	}
}

func renderConfigTemplate(template string, replacements map[string]string) string {
	lines := strings.Split(template, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for key, value := range replacements {
			if strings.HasPrefix(trimmed, key+" =") {
				lines[i] = key + " = " + strconv.Quote(value)
			}
		}
	}

	return strings.Join(lines, "\n")
}

func runConfigEdit() error {
	configPath, err := getDefaultConfigFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("默认配置文件不存在: %s，请先运行 `yewresin config init`", configPath)
		}
		return fmt.Errorf("检查默认配置文件失败: %w", err)
	}

	editor, err := resolveEditorCommand()
	if err != nil {
		return err
	}

	return runEditorCommand(editor, configPath)
}

func resolveEditorCommand() (string, error) {
	if editor := strings.TrimSpace(getEditorEnv()); editor != "" {
		return editor, nil
	}

	candidates := fallbackEditors()
	tried := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		tried = append(tried, strings.Join(candidate, " "))
		if _, err := lookPath(candidate[0]); err == nil {
			return strings.Join(candidate, " "), nil
		}
	}

	return "", fmt.Errorf("未设置 EDITOR，且未找到可用的兜底编辑器（已尝试：%s）", strings.Join(tried, "、"))
}

func fallbackEditors() [][]string {
	switch runtime.GOOS {
	case "windows":
		return [][]string{
			{"code", "--wait"},
			{"notepad"},
		}
	case "darwin":
		return [][]string{
			{"code", "--wait"},
			{"vim"},
			{"nano"},
			{"open", "-W", "-t"},
		}
	default:
		return [][]string{
			{"code", "--wait"},
			{"vim"},
			{"nano"},
			{"vi"},
		}
	}
}

func defaultRunEditorCommand(editor, configPath string) error {
	args, err := splitCommandLine(editor)
	if err != nil {
		return fmt.Errorf("解析 EDITOR 失败: %w", err)
	}
	if len(args) == 0 {
		return fmt.Errorf("EDITOR 不能为空")
	}

	cmd := exec.Command(args[0], append(args[1:], configPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行编辑器失败: %w", err)
	}

	return nil
}

func splitCommandLine(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, nil
	}

	var args []string
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}

	for _, r := range command {
		switch {
		case r == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case r == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case (r == ' ' || r == '\t') && !inDoubleQuote && !inSingleQuote:
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if inDoubleQuote || inSingleQuote {
		return nil, fmt.Errorf("引号未正确闭合")
	}

	flush()
	return args, nil
}
