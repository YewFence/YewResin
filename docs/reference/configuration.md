# 配置参考

## 命令行参数

| 参数 | 说明 |
|------|------|
| `--dry-run`, `-n` | 模拟运行，只检查依赖和显示操作，不实际执行 |
| `-y`, `--yes` | 跳过交互式确认 |
| `--help`, `-h` | 显示帮助信息 |
| `--config <path>` | 指定配置文件路径（默认为程序同目录的 `.env`） |
| `--version` | 显示版本信息 |

## 配置加载顺序

配置按以下顺序生效：

1. 当前进程中已存在的环境变量
2. `--config` 指定文件中的配置；如果未指定，则尝试读取程序同目录的 `.env`
3. 代码中的内置默认值

## 环境变量

| 变量 | 默认值 | 说明 | 必填 |
|------|--------|------|------|
| `BASE_DIR` | - | Docker Compose 项目目录 | 是 |
| `EXPECTED_REMOTE` | - | Kopia 远程路径 | 是 |
| `KOPIA_PASSWORD` | - | Kopia 远程仓库密码 | 否 |
| `KOPIA_CONFIG_FILE` | - | Kopia 配置文件路径（可选，用于多用户场景） | 否 |
| `RCLONE_CONFIG` | - | Rclone 配置文件路径（可选，用于多用户场景） | 否 |
| `PRIORITY_SERVICES_LIST` | `caddy nginx gateway` | 优先服务列表（空格分隔） | 否 |
| `LOCK_FILE` | `/tmp/backup_maintenance.lock` | 锁文件路径 | 否 |
| `LOG_FILE` | 无（不输出日志文件） | 日志文件路径，留空则不写入文件 | 否 |
| `DOCKER_COMMAND_TIMEOUT_SECONDS` | `120` | Docker 命令超时时间（秒） | 否 |
| `DEVICE_NAME` | - | 设备名称，用于区分不同服务器的通知 | 否 |
| `APPRISE_URL` | - | Apprise 服务地址 | 否 |
| `APPRISE_NOTIFY_URL` | - | 通知目标 URL | 否 |
| `GIST_TOKEN` | - | GitHub Personal Access Token（需要 gist 权限） | 否 |
| `GIST_ID` | - | GitHub Gist ID（日志上传目标） | 否 |
| `GIST_LOG_PREFIX` | `yewresin-backup` | Gist 日志文件名前缀 | 否 |
| `GIST_MAX_LOGS` | `30` | Gist 最大保留日志数量（设为 0 禁用清理） | 否 |
| `GIST_KEEP_FIRST_FILE` | `true` | 清理时保留第一个文件（用于自定义 Gist 标题） | 否 |

## 配置说明

如果必填项（如 `BASE_DIR` 和 `EXPECTED_REMOTE`）最终未设置，程序会直接报错并退出。部分可选项如果未设置会回退到内置默认值。

## 配置文件位置

- 直接下载安装：程序同目录的 `.env` 文件
- `--config` 参数：手动指定任意路径的 `.env` 文件
- Homebrew 安装后，程序位于 `bin/` 目录，无法在同目录放 `.env`，推荐使用 `--config` 指定（如 `~/.config/yewresin/.env`）
- 如果默认路径下不存在 `.env` 文件，程序不会报错，但需要通过环境变量提供必填项

## 日志持久化说明

- `LOG_FILE` 未配置时，日志只输出到标准输出，不会由程序写入任何日志文件
- `LOG_FILE` 配置后，日志会同时输出到标准输出和该文件
- 程序内部会临时缓存本次运行的日志内容，但这部分缓存仅存在于进程内存中，进程结束后不会保留
- 如果你依赖 `systemd`、Docker、任务计划或其他外部工具采集标准输出，这部分持久化行为由外部运行环境负责
