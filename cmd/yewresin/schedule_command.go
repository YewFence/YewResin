package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/YewFence/yewresin/internal/schedule"
)

var (
	scheduleInstallAction          = schedule.Install
	scheduleUninstallAction        = schedule.Uninstall
	scheduleStatusAction           = schedule.Status
	scheduleNormalizeInstallAction = schedule.NormalizeInstallOptions
)

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
		if err := scheduleInstallAction(stdout, *opts); err != nil {
			fmt.Fprintf(stderr, "安装调度失败: %v\n", err)
			return 1
		}
		return 0
	case "uninstall":
		opts, code := parseScheduleCommonFlags("schedule uninstall", args[1:], stderr)
		if code != 0 {
			return code
		}
		if err := scheduleUninstallAction(stdout, opts.Backend); err != nil {
			fmt.Fprintf(stderr, "卸载调度失败: %v\n", err)
			return 1
		}
		return 0
	case "status":
		opts, code := parseScheduleCommonFlags("schedule status", args[1:], stderr)
		if code != 0 {
			return code
		}
		if err := scheduleStatusAction(stdout, opts.Backend); err != nil {
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

func parseScheduleInstallFlags(args []string, stderr io.Writer) (*schedule.Options, int) {
	flags := flag.NewFlagSet("schedule install", flag.ContinueOnError)
	flags.SetOutput(stderr)

	backend := flags.String("backend", schedule.DefaultBackend, "调度后端：cron 或 systemd-user")
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

	opts := &schedule.Options{
		Backend:    normalizeScheduleBackendName(*backend),
		Expr:       strings.TrimSpace(*expr),
		OnCalendar: strings.TrimSpace(*onCalendar),
		ConfigPath: strings.TrimSpace(*configPath),
	}
	if err := scheduleNormalizeInstallAction(opts); err != nil {
		fmt.Fprintln(stderr, err)
		return nil, 2
	}

	return opts, 0
}

func parseScheduleCommonFlags(name string, args []string, stderr io.Writer) (*schedule.Options, int) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)

	backend := flags.String("backend", schedule.DefaultBackend, "调度后端：cron 或 systemd-user")

	if err := flags.Parse(args); err != nil {
		return nil, 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "%s 不接受位置参数\n", name)
		return nil, 2
	}

	return &schedule.Options{Backend: normalizeScheduleBackendName(*backend)}, 0
}

func normalizeScheduleBackendName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return schedule.DefaultBackend
	}
	return name
}
