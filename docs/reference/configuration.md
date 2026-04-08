# 配置参考

## 命令行参数

| 参数 | 说明 |
|------|------|
| `--dry-run`, `-n` | 模拟运行，只检查依赖和显示操作，不实际执行 |
| `-y`, `--yes` | 跳过交互式确认 |
| `--help`, `-h` | 显示帮助信息 |
| `--config <path>` | 指定配置文件路径（支持 `config.toml` / `.env`） |
| `--version` | 显示版本信息 |

## 配置子命令

| 命令 | 说明 |
|------|------|
| `config init` | 将 `config.toml.example` 初始化到默认配置目录，并只引导填写必填项 |
| `config edit` | 使用 `EDITOR` 打开默认配置文件 |

`config edit` 依赖 `EDITOR` 环境变量，例如 `EDITOR="code --wait"`。
`config init --force` 可以覆盖已有的默认配置文件。
如果 `EDITOR` 未设置，`config edit` 会按平台尝试常见编辑器作为兜底。

## 调度子命令

| 命令 | 说明 |
|------|------|
| `schedule install` | 安装当前用户的定时调度；默认后端为 `cron`，默认表达式为 `0 3 * * *` |
| `schedule uninstall` | 卸载当前用户的定时调度 |
| `schedule status` | 查看当前用户的定时调度状态 |

常用选项：

- `schedule install --expr "0 */6 * * *"`：给 `cron` 后端指定表达式
- `schedule install --backend systemd-user --on-calendar "*-*-* 03:00:00"`：Linux 下切到 `systemd-user`
- `schedule install --config /path/to/config.toml`：显式指定要写入调度命令的配置文件路径

`schedule` 会尽量写入绝对路径的可执行文件和配置文件，减少 `cron` / `systemd` 环境和交互式 shell 环境不一致的问题。

## 配置加载顺序

配置按以下顺序生效：

1. 当前进程中已存在的环境变量
2. `--config` 指定文件中的配置（支持 `.env` / `.toml`）
3. 用户配置目录中的 `config.toml`
4. 用户配置目录中的 `.env`
5. 程序同目录的 `config.toml`
6. 程序同目录的 `.env`
7. 代码中的内置默认值

## 配置项

| 环境变量 | TOML 键 | 默认值 | 说明 | 必填 |
|------|------|--------|------|------|
| `BASE_DIR` | `base_dir` | - | Docker Compose 项目目录 | 是 |
| `EXPECTED_REMOTE` | `expected_remote` | - | Kopia 远程路径 | 是 |
| `KOPIA_PASSWORD` | `[kopia].password` | - | Kopia 远程仓库密码 | 否 |
| `KOPIA_CONFIG_FILE` | `[kopia].config_file` | - | Kopia 配置文件路径（可选，用于多用户场景） | 否 |
| `RCLONE_CONFIG` | `[rclone].config_file` | - | Rclone 配置文件路径（可选，用于多用户场景） | 否 |
| `PRIORITY_SERVICES_LIST` | `priority_services` | `caddy nginx gateway` | 优先服务列表；环境变量使用空格分隔，TOML 使用字符串数组 | 否 |
| `LOCK_FILE` | `lock_file` | `/tmp/backup_maintenance.lock` | 锁文件路径 | 否 |
| `LOG_FILE` | `[logging].file` | 无（不输出日志文件） | 日志文件路径，留空则不写入文件 | 否 |
| `DOCKER_COMMAND_TIMEOUT_SECONDS` | `[logging].docker_command_timeout_seconds` | `120` | Docker 命令超时时间（秒） | 否 |
| `DEVICE_NAME` | `[notifications].device_name` | - | 设备名称，用于区分不同服务器的通知 | 否 |
| `APPRISE_URL` | `[notifications].apprise_url` | - | Apprise 服务地址 | 否 |
| `APPRISE_NOTIFY_URL` | `[notifications].apprise_notify_url` | - | 通知目标 URL | 否 |
| `GIST_TOKEN` | `[gist].token` | - | GitHub Personal Access Token（需要 gist 权限） | 否 |
| `GIST_ID` | `[gist].id` | - | GitHub Gist ID（日志上传目标） | 否 |
| `GIST_LOG_PREFIX` | `[gist].log_prefix` | `yewresin-backup` | Gist 日志文件名前缀 | 否 |
| `GIST_MAX_LOGS` | `[gist].max_logs` | `30` | Gist 最大保留日志数量（设为 0 禁用清理） | 否 |
| `GIST_KEEP_FIRST_FILE` | `[gist].keep_first_file` | `true` | 清理时保留第一个文件（用于自定义 Gist 标题） | 否 |

## TOML 示例

```toml
base_dir = "/opt/docker_file"
expected_remote = "gdrive:backup"
priority_services = ["caddy", "nginx", "gateway"]
lock_file = "/tmp/backup_maintenance.lock"

[logging]
file = "/var/log/yewresin.log"
docker_command_timeout_seconds = 120

[notifications]
device_name = "HomeServer"
apprise_url = "https://your-apprise-instance.vercel.app/notify"
apprise_notify_url = "tgram://bottoken/ChatID"

[gist]
log_prefix = "yewresin-backup"
max_logs = 30
keep_first_file = true

[kopia]
password = "your_kopia_password"
config_file = "/home/youruser/.config/kopia/repository.config"

[rclone]
config_file = "/home/youruser/.config/rclone/rclone.conf"
```

## 配置说明

如果必填项（如 `BASE_DIR` 和 `EXPECTED_REMOTE`）最终未设置，程序会直接报错并退出。部分可选项如果未设置会回退到内置默认值。

对于敏感项（如 `KOPIA_PASSWORD`、`GIST_TOKEN`），虽然可以写进 `config.toml`，但更推荐通过环境变量注入。

## 配置文件位置

- 默认配置目录基于 `os.UserConfigDir()`：
- Linux 通常是 `~/.config/yewresin/`
- macOS 通常是 `~/Library/Application Support/yewresin/`
- Windows 通常是 `%AppData%\yewresin\`
- `--config` 参数：手动指定任意路径的 `.env` 或 `.toml` 文件
- 默认会优先读取用户配置目录中的 `config.toml`，其次 `.env`
- 为了兼容已有安装方式，如果用户配置目录没有配置文件，仍会继续尝试程序同目录的 `config.toml` 和 `.env`
- 如果默认路径下不存在配置文件，程序不会报错，但需要通过环境变量提供必填项

## 日志持久化说明

- `LOG_FILE` 未配置时，日志只输出到标准输出，不会由程序写入任何日志文件
- `LOG_FILE` 配置后，日志会同时输出到标准输出和该文件
- 程序内部会临时缓存本次运行的日志内容，但这部分缓存仅存在于进程内存中，进程结束后不会保留
- 如果你依赖 `systemd`、Docker、任务计划或其他外部工具采集标准输出，这部分持久化行为由外部运行环境负责
