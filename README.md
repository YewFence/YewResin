# YewResin - Docker 服务备份工具

一个自动化的 Docker Compose 服务备份脚本，使用 Kopia + rclone 实现本地快照与云端同步。

> 建议先 Fork 本仓库，然后根据自己的需求修改配置。

## 功能特点

- 自动停止所有 Docker Compose 服务，创建一致性快照
- 支持优先级服务（如网关）的顺序控制：最后停止，最先启动
- 只重启原本运行中的服务，不会启动原本停止的服务
- 支持 [Apprise](https://github.com/caronc/apprise-api) 通知（Telegram、微信等），可部署到 Vercel
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

# 下载该脚本
git clone https://github.com/YewFence/YewResin.git
cd YewResin
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

### 4. 定时任务
> 按需配置，此处我们以每天凌晨三点运行为例
```bash
(crontab -l 2>/dev/null; echo '0 3 * * * /path/to/backup.sh -y >> /var/log/docker-backup.log 2>&1') | crontab -
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

如果 `BASE_DIR` 下存在权限敏感的目录（如 `caddy/data/caddy`、`ssl`、`ssh` 等），Kopia 可能会因权限问题报错。虽然备份仍会完成，但建议在 Kopia 策略中忽略这些目录：

## 定时任务配置

### Cron 表达式格式

```
┌───────────── 分钟 (0-59)
│ ┌─────────── 小时 (0-23)
│ │ ┌───────── 日期 (1-31)
│ │ │ ┌─────── 月份 (1-12)
│ │ │ │ ┌───── 星期 (0-7，0 和 7 都表示周日)
│ │ │ │ │
* * * * *
```

### 常用配置示例

```bash
# 编辑 crontab
crontab -e

# 每天凌晨 3 点执行备份
0 3 * * * /path/to/backup.sh -y >> /var/log/backup.log 2>&1

# 每周日凌晨 2 点执行备份
0 2 * * 0 /path/to/backup.sh -y >> /var/log/backup.log 2>&1

# 每 6 小时执行一次（0点、6点、12点、18点）
0 */6 * * * /path/to/backup.sh -y >> /var/log/backup.log 2>&1

# 每天凌晨 3 点和 15 点执行（一天两次）
0 3,15 * * * /path/to/backup.sh -y >> /var/log/backup.log 2>&1

# 每月 1 日和 15 日凌晨 4 点执行
0 4 1,15 * * /path/to/backup.sh -y >> /var/log/backup.log 2>&1
```

### 使用 Systemd Timer

相比 cron，systemd timer 提供更好的日志管理和错误处理。

创建服务文件 `/etc/systemd/system/yewresin-backup.service`：

```ini
[Unit]
Description=YewResin Docker Backup
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
ExecStart=/path/to/backup.sh -y
StandardOutput=journal
StandardError=journal
```

创建定时器文件 `/etc/systemd/system/yewresin-backup.timer`：

```ini
[Unit]
Description=Run YewResin backup daily

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
```

启用定时器：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now yewresin-backup.timer

# 查看定时器状态
systemctl list-timers yewresin-backup.timer

# 查看备份日志
journalctl -u yewresin-backup.service -f
```

### 注意事项

- **使用绝对路径**：cron 环境的 PATH 与交互式 shell 不同，务必使用脚本的绝对路径
- **日志轮转**：建议配合 logrotate 管理日志文件大小
- **错误通知**：脚本已集成 Apprise 通知，配置后可自动发送备份结果
- **避免重叠**：脚本内置锁机制，防止多个备份任务同时运行


## License

MIT
