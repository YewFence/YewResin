package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// 版本信息，构建时注入
var version = "dev"

func isVersionFlag(arg string) bool {
	return arg == "--version" || arg == "-version"
}

func isHelpFlag(arg string) bool {
	return arg == "--help" || arg == "-help" || arg == "-h"
}

func extractConfigValue(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--config" || args[i] == "-config":
			if i+1 >= len(args) {
				return "", false
			}
			return args[i+1], true
		case strings.HasPrefix(args[i], "--config="):
			return strings.TrimPrefix(args[i], "--config="), true
		case strings.HasPrefix(args[i], "-config="):
			return strings.TrimPrefix(args[i], "-config="), true
		}
	}
	return "", false
}

func hasConfigFlag(args []string) bool {
	_, ok := extractConfigValue(args)
	return ok
}

func prepareConfigCommandArgs(globalArgs, subcommandArgs []string) []string {
	if hasConfigFlag(subcommandArgs) {
		return subcommandArgs
	}
	configValue, ok := extractConfigValue(globalArgs)
	if !ok {
		return subcommandArgs
	}
	return append([]string{"--config", configValue}, subcommandArgs...)
}

func findCommand(args []string) (int, string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				return i + 1, args[i+1]
			}
			return -1, ""
		case arg == "--config" || arg == "-config":
			i++
		case strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-config="):
		case arg == "--dry-run" || arg == "-n" || arg == "--yes" || arg == "-y":
		case isVersionFlag(arg) || isHelpFlag(arg):
		case strings.HasPrefix(arg, "-"):
			return -1, ""
		default:
			return i, arg
		}
	}
	return -1, ""
}

func main() {
	// --version 全局处理（兼容有无子命令的情况）
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if isVersionFlag(arg) {
				fmt.Printf("YewResin %s\n", version)
				os.Exit(0)
			}
		}

		if commandIndex, command := findCommand(os.Args[1:]); commandIndex >= 0 {
			switch command {
			case "config":
				runConfigCmd(prepareConfigCommandArgs(os.Args[1:1+commandIndex], os.Args[2+commandIndex:]))
				return
			default:
				fmt.Fprintf(os.Stderr, "未知命令: %s\n运行 '%s --help' 查看用法\n", command, os.Args[0])
				os.Exit(1)
			}
		}
	}

	runBackup()
}

// runBackup 执行备份主流程（默认行为）
func runBackup() {
	// CLI 参数定义
	dryRun := flag.Bool("dry-run", false, "模拟运行，不执行实际操作")
	flag.BoolVar(dryRun, "n", false, "模拟运行（-dry-run 的简写）")

	autoConfirm := flag.Bool("yes", false, "跳过交互式确认")
	flag.BoolVar(autoConfirm, "y", false, "跳过确认（-yes 的简写）")

	configFile := flag.String("config", "", "配置文件路径（默认为程序同目录的 .env）")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "YewResin - Docker 服务备份工具 (Go 版本)\n\n")
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      %s <命令> [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "命令:\n")
		fmt.Fprintf(os.Stderr, "  config export   导出配置文件（加密归档）\n")
		fmt.Fprintf(os.Stderr, "  config import   导入配置文件（解密还原）\n")
		fmt.Fprintf(os.Stderr, "  config list     列出可导出的配置文件\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s --dry-run     # 模拟运行\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -y            # 跳过确认直接执行\n", os.Args[0])
	}

	flag.Parse()

	// 加载配置（先加载配置以获取日志文件路径）
	cfg, err := LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志（支持文件输出）
	logFile, err := InitLogger(cfg.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	// 打印配置信息
	cfg.Print()

	// 创建备份编排器
	orch := NewOrchestrator(cfg, *dryRun)

	// 设置信号处理（Ctrl+C 等）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		slog.Warn("收到终止信号，正在清理...", "signal", sig)
		orch.Cleanup()
		os.Exit(1)
	}()

	// 检查依赖
	if err := orch.CheckDependencies(); err != nil {
		slog.Error("依赖检查失败", "error", err)
		os.Exit(1)
	}

	// 交互式确认
	if !*dryRun && !*autoConfirm {
		if !confirm() {
			fmt.Println("已取消操作")
			os.Exit(0)
		}
	}

	// 执行备份
	startTime := time.Now().UTC()
	if err := orch.Run(); err != nil {
		slog.Error("备份失败", "error", err)
		os.Exit(1)
	}

	// 打印耗时
	elapsed := time.Since(startTime)
	slog.Info("备份完成", "耗时", elapsed.Round(time.Second))
}

// confirm 交互式确认
func confirm() bool {
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("⚠️  警告：即将执行备份操作")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("此操作将会：")
	fmt.Println("  1. 停止所有 Docker 服务")
	fmt.Println("  2. 创建 Kopia 快照备份")
	fmt.Println("  3. 重新启动所有服务")
	fmt.Println()
	fmt.Print("确认执行备份？[y/N] ")

	var response string
	fmt.Scanln(&response)
	return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes")
}
