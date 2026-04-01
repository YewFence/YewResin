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

	"github.com/YewFence/yewresin/internal/yewresin"
)

// 版本信息，构建时注入
var version = "dev"

func main() {
	// CLI 参数定义
	dryRun := flag.Bool("dry-run", false, "模拟运行，不执行实际操作")
	flag.BoolVar(dryRun, "n", false, "模拟运行（-dry-run 的简写）")

	autoConfirm := flag.Bool("yes", false, "跳过交互式确认")
	flag.BoolVar(autoConfirm, "y", false, "跳过确认（-yes 的简写）")

	configFile := flag.String("config", "", "配置文件路径（默认为程序同目录的 .env）")
	showVersion := flag.Bool("version", false, "显示版本信息")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "YewResin - Docker 服务备份工具\n\n")
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s --dry-run     # 模拟运行\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -y            # 跳过确认直接执行\n", os.Args[0])
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("YewResin %s\n", version)
		os.Exit(0)
	}

	// 加载配置（先加载配置以获取日志文件路径）
	cfg, err := yewresin.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志（支持文件输出）
	logFile, err := yewresin.InitLogger(cfg.LogFile)
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
	orch := yewresin.NewOrchestrator(cfg, *dryRun)

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
