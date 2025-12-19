# YewResin - Docker 服务备份工具

一个自动化的 Docker Compose 服务备份脚本，使用 Kopia + rclone 实现本地快照与云端同步。

> 建议先 Fork 本仓库，然后根据自己的需求修改配置。

## 功能特点

- 自动停止所有 Docker Compose 服务，创建一致性快照
- 支持优先级服务（如网关）的顺序控制：最后停止，最先启动
- 只重启原本运行中的服务，不会启动原本停止的服务
- 支持 [Apprise](https://github.com/caronc/apprise-api) 通知（Telegram、微信等），可部署到 Vercel
- 支持 GitHub Actions 定时运行，方便在 GitHub 查看执行日志
- 异常退出时自动恢复服务
- 支持 dry-run 模式预览操作
- 防止重复运行的锁机制

## 依赖

- [rclone](https://rclone.org/downloads/) - 云存储同步工具
- [Kopia](https://kopia.io/docs/installation/) - 快照备份工具
- Docker & Docker Compose

## 快速开始

### 1. 安装依赖

```bash
# 安装 rclone
curl https://rclone.org/install.sh | sudo bash
rclone config  # 配置远程存储（如 Google Drive）

# 安装 kopia
# Debian/Ubuntu
curl -s https://kopia.io/signing-key | sudo gpg --dearmor -o /etc/apt/keyrings/kopia-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kopia-keyring.gpg] http://packages.kopia.io/apt/ stable main" | sudo tee /etc/apt/sources.list.d/kopia.list
sudo apt update && sudo apt install kopia

# 连接 Kopia 仓库
kopia repository connect rclone --remote-path="gdrive:backup"
```

### 2. 配置

创建 `.env` 文件（与 `backup.sh` 同目录）：

```bash
# Docker Compose 项目总目录
BASE_DIR=/opt/docker_file

# Kopia 远程路径（需与 rclone 配置匹配）
EXPECTED_REMOTE=gdrive:backup

# 网关服务列表（最后停止，最先启动），空格分隔
PRIORITY_SERVICES_LIST="caddy nginx gateway"

# 备份失败时是否继续启动服务
IGNORE_BACKUP_ERROR=true

# Apprise 通知配置（可选）
# 如果需要 Vercel 部署的 Apprise，可参考：https://github.com/YewFence/apprise
APPRISE_URL=http://your-apprise-server:8000/notify
APPRISE_NOTIFY_URL=tgram://bot_token/chat_id
```

### 3. 运行

```bash
# 模拟运行（推荐先测试）
./backup.sh --dry-run

# 执行备份（需确认）
./backup.sh

# 跳过确认直接执行（适用于 cron）
./backup.sh -y
```

## 命令行参数

| 参数 | 说明 |
|------|------|
| `--dry-run`, `-n` | 模拟运行，只检查依赖和显示操作，不实际执行 |
| `-y`, `--yes` | 跳过交互式确认 |
| `--help`, `-h` | 显示帮助信息 |

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BASE_DIR` | `/opt/docker_file` | Docker Compose 项目目录 |
| `EXPECTED_REMOTE` | `gdrive:backup` | Kopia 远程路径 |
| `PRIORITY_SERVICES_LIST` | `caddy nginx gateway` | 优先服务列表（空格分隔） |
| `IGNORE_BACKUP_ERROR` | `true` | 备份失败时是否继续 |
| `LOCK_FILE` | `/tmp/backup_maintenance.lock` | 锁文件路径 |
| `APPRISE_URL` | - | Apprise 服务地址 |
| `APPRISE_NOTIFY_URL` | - | 通知目标 URL |
| `CONFIG_FILE` | `./backup.sh` 同目录的 `.env` | 配置文件路径 |

## 定时任务示例

```bash
# 每天凌晨 3 点执行备份
0 3 * * * /path/to/backup.sh -y >> /var/log/backup.log 2>&1
```

## Github Action 配置

请参考[Github Action 配置文档](.github/README.md)

## 目录结构要求

```
/opt/docker_file/           # BASE_DIR
├── caddy/                  # 网关服务
│   ├── docker-compose.yml
│   └── compose-up.sh       # 可选：自定义启动脚本
├── nginx/
│   └── docker-compose.yml
├── app1/                   # 普通服务
│   └── docker-compose.yml
└── app2/
    └── docker-compose.yml
```

脚本会自动识别包含 `docker-compose.yml` 或 `compose*.yaml` 的目录作为服务。

## 工作流程

1. 检查依赖（rclone、kopia）
2. 停止普通服务
3. 停止网关服务
4. 创建 Kopia 快照
5. 启动网关服务
6. 启动普通服务
7. 执行 Kopia 维护清理

## 注意事项

如果 `BASE_DIR` 下存在权限敏感的目录（如 `caddy`、`ssl`、`ssh` 等），Kopia 可能会因权限问题报错。虽然备份仍会完成，但建议在 Kopia 策略中忽略这些目录：

```bash
# 设置忽略规则
kopia policy set /opt/docker_file --add-ignore "caddy/**"
kopia policy set /opt/docker_file --add-ignore "ssl/**"
kopia policy set /opt/docker_file --add-ignore ".ssh/**"
```

## License

MIT
