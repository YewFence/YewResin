# 配置参考

## 命令行参数

| 参数 | 说明 |
|------|------|
| `--dry-run`, `-n` | 模拟运行，只检查依赖和显示操作，不实际执行 |
| `-y`, `--yes` | 跳过交互式确认 |
| `--help`, `-h` | 显示帮助信息 |
| `--config <path>` | 指定配置文件路径 |
| `--version` | 显示版本信息 |

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
| `LOG_FILE` | 程序同目录下 `yewresin.log` | 日志文件路径 | 否 |
| `DOCKER_COMMAND_TIMEOUT_SECONDS` | `120` | Docker 命令超时时间（秒） | 否 |
| `DEVICE_NAME` | - | 设备名称，用于区分不同服务器的通知 | 否 |
| `APPRISE_URL` | - | Apprise 服务地址 | 否 |
| `APPRISE_NOTIFY_URL` | - | 通知目标 URL | 否 |
| `GIST_TOKEN` | - | GitHub Personal Access Token（需要 gist 权限） | 否 |
| `GIST_ID` | - | GitHub Gist ID（日志上传目标） | 否 |
| `GIST_LOG_PREFIX` | `yewresin-backup` | Gist 日志文件名前缀 | 否 |
| `GIST_MAX_LOGS` | `30` | Gist 最大保留日志数量（设为 0 禁用清理） | 否 |
| `GIST_KEEP_FIRST_FILE` | `true` | 清理时保留第一个文件（用于自定义 Gist 标题） | 否 |
| `CONFIG_FILE` | 程序同目录的 `.env` | 配置文件路径 | 否 |
