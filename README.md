# YewResin - Docker 服务备份工具

一个自动化的 Docker Compose 服务备份工具，使用 Kopia 实现本地快照与云端同步。

## 功能特点

- 自动停止所有 Docker Compose 服务，创建一致性快照
- 支持优先级服务（如网关）的顺序控制：最后停止，最先启动
- 只重启原本运行中的服务，不会启动原本停止的服务
- **快速失败**：服务停止失败时立即中止备份，避免在服务运行时备份导致数据损坏
- 支持多种 compose 配置文件格式（`compose.yaml`、`compose.yml`、`docker-compose.yaml`、`docker-compose.yml`）
- 支持 [Apprise](https://github.com/caronc/apprise-api) 通知
  > 可使用 [YewFence/apprise](https://github.com/YewFence/apprise) 快速部署到 Vercel
- 异常退出时自动恢复服务
- 支持 dry-run 模式预览操作
- 防止重复运行的锁机制
- 并行停止/启动服务，性能更优

## 依赖

- [Kopia](https://kopia.io/docs/installation/) - 快照备份工具
- Docker & Docker Compose
- 可选：[rclone](https://rclone.org/downloads/) - 云存储同步工具

## 快速开始

### 1. 安装依赖

```bash
# 安装 rclone（按需）
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

### 2. 下载 YewResin

根据系统架构下载对应的二进制文件：

```bash
mkdir ~/yewresin && cd ~/yewresin

# Linux x64
wget https://github.com/YewFence/YewResin/releases/latest/download/yewresin-linux-amd64 -O yewresin
# Linux ARM64
wget https://github.com/YewFence/YewResin/releases/latest/download/yewresin-linux-arm64 -O yewresin
# macOS Apple Silicon
wget https://github.com/YewFence/YewResin/releases/latest/download/yewresin-darwin-arm64 -O yewresin
# macOS Intel
wget https://github.com/YewFence/YewResin/releases/latest/download/yewresin-darwin-amd64 -O yewresin
# Windows
# 下载 yewresin-windows-amd64.exe

chmod +x yewresin
```

> `latest` 标签会在 main 分支推送后自动更新，也可以下载指定版本（如 `v2.0.0`）。

### 3. 配置

创建 `.env` 文件（与 yewresin 同目录）：

```bash
# 在脚本所在目录下载示例文件
wget https://github.com/YewFence/YewResin/releases/latest/download/.env.example
cp .env.example .env
```

必要环境变量配置：
```bash
# Docker Compose 项目总目录
BASE_DIR=/opt/docker_file
# Kopia 远程路径
EXPECTED_REMOTE=gdrive:backup
```

### 4. 运行

```bash
# 模拟运行（推荐先测试）
./yewresin --dry-run

# 执行备份（需确认）
./yewresin

# 跳过确认直接执行（适用于 cron）
./yewresin -y
```

### 5. 定时任务

> 按需配置，此处我们以每天北京时间凌晨三点运行为例（假设服务器使用 UTC 时区）

```bash
(crontab -l 2>/dev/null; echo '0 19 * * * /path/to/yewresin -y') | crontab -
```

> **注意**：
> - cron 使用系统时区，请先确认服务器时区（`timedatectl` 或 `date`），上述示例假设服务器为 UTC 时区
> - 脚本内部使用 exec 重定向，cron 的 `>>` 重定向会被覆盖，可通过 `LOG_FILE` 环境变量自定义日志路径（默认为程序同目录下的 `yewresin.log`）

## 命令行参数

| 参数 | 说明 |
|------|------|
| `--dry-run`, `-n` | 模拟运行，只检查依赖和显示操作，不实际执行 |
| `-y`, `--yes` | 跳过交互式确认 |
| `--help`, `-h` | 显示帮助信息 |
| `--config <path>` | 指定配置文件路径 |
| `--version` | 显示版本信息 |

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BASE_DIR` | - | Docker Compose 项目目录（必填） |
| `EXPECTED_REMOTE` | - | Kopia 远程路径（必填） |
| `KOPIA_PASSWORD` | - | Kopia 远程仓库密码 |
| `KOPIA_CONFIG_FILE` | - | Kopia 配置文件路径（可选，用于多用户场景） |
| `RCLONE_CONFIG` | - | Rclone 配置文件路径（可选，用于多用户场景） |
| `PRIORITY_SERVICES_LIST` | `caddy nginx gateway` | 优先服务列表（空格分隔） |
| `LOCK_FILE` | `/tmp/backup_maintenance.lock` | 锁文件路径 |
| `LOG_FILE` | 程序同目录下 `yewresin.log` | 日志文件路径 |
| `DOCKER_COMMAND_TIMEOUT_SECONDS` | `120` | Docker 命令超时时间（秒） |
| `DEVICE_NAME` | - | 设备名称，用于区分不同服务器的通知 |
| `APPRISE_URL` | - | Apprise 服务地址 |
| `APPRISE_NOTIFY_URL` | - | 通知目标 URL |
| `GIST_TOKEN` | - | GitHub Personal Access Token（需要 gist 权限） |
| `GIST_ID` | - | GitHub Gist ID（日志上传目标） |
| `GIST_LOG_PREFIX` | `yewresin-backup` | Gist 日志文件名前缀 |
| `GIST_MAX_LOGS` | `30` | Gist 最大保留日志数量（设为 0 禁用清理） |
| `GIST_KEEP_FIRST_FILE` | `true` | 清理时保留第一个文件（用于自定义 Gist 标题） |
| `CONFIG_FILE` | 程序同目录的 `.env` | 配置文件路径 |

## 关键要求

### 目录结构要求

```text
/opt/docker_file/           # BASE_DIR
├── caddy/                  # 网关服务
│   ├── compose.yaml        # 支持多种命名格式
│   └── compose-up.sh       # 可选：自定义启动脚本
├── nginx/
│   └── docker-compose.yml
├── app1/                   # 普通服务
│   └── compose.yml
└── app2/
    └── docker-compose.yaml
```

脚本会自动识别包含以下任一配置文件的目录作为服务：
- `compose.yaml`
- `compose.yml`
- `docker-compose.yaml`
- `docker-compose.yml`

### 启停逻辑

服务启停按以下优先级执行：

1. **自定义脚本优先**：若目录下存在 `compose-stop.sh`/`compose-down.sh`/`compose-up.sh`，优先使用脚本启停
2. **自动识别配置文件**：若无自定义脚本但存在 compose 配置文件，使用 `docker compose up -d` / `docker compose stop` 启停

### 快速失败机制

为保护数据完整性，脚本在停止服务阶段采用快速失败策略：

- 如果任何服务停止失败，脚本会**立即中止**，不会继续执行备份
- 已停止的服务会通过 cleanup 函数自动恢复
- 通过 Apprise 发送通知告知失败原因

这确保了不会在服务仍在运行（可能正在写入数据）时进行备份，避免数据库文件损坏等问题。

## 工作流程

1. 检查依赖（rclone、kopia）
2. 停止普通服务（并行）
3. 停止网关服务
4. 创建 Kopia 快照
5. 启动网关服务
6. 启动普通服务（并行）
7. 执行 Kopia 维护清理

## 注意事项

如果 `BASE_DIR` 下存在权限敏感的目录（如 `caddy/data/caddy`、`ssl`、`ssh` 等），Kopia 可能会因权限问题报错。虽然备份仍会完成，但建议在 Kopia 策略中忽略这些目录。

## GitHub Gist 日志推送

脚本支持将每日备份日志自动推送到 GitHub Gist，实现日志持久化和远程查看。

### 为什么使用 Gist？

- ✅ 持久化存储，不会被清理
- ✅ 每次备份独立文件（如 `yewresin-backup-2025-12-20_03-00-15.log`），精确到秒
- ✅ 有版本历史，可以查看每次备份的变化
- ✅ 免费、稳定，支持 API 操作
- ✅ 可以通过链接方便地分享和查看

### 配置步骤

#### 1. 创建 GitHub Personal Access Token

访问 [GitHub Token 设置](https://github.com/settings/tokens/new)，创建一个新的 token：

- **Note**: YewResin Backup Logger
- **Expiration**: 自定义（建议选择较长期限）
- **Select scopes**: 只勾选 `gist` 权限

创建后复制 token（只会显示一次）。

#### 2. 创建一个空的 Gist

访问 [gist.github.com](https://gist.github.com/)，创建一个新的 Gist：

- **Filename**: 可以随便写，比如 `backup-logs.md`
- **Content**: 可以随便写，比如 `# YewResin Backup Logs`
- 选择 **Public** 或 **Secret**（推荐 Secret）

创建后，从 URL 中获取 Gist ID：
```
https://gist.github.com/username/abc123def456789
                              └─────────┬────────┘
                                    这就是 Gist ID
```

#### 3. 配置环境变量

在 `.env` 文件中添加：

```bash
GIST_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
GIST_ID=abc123def456789
GIST_LOG_PREFIX=my-server-backup  # 可选，自定义日志文件名前缀
GIST_MAX_LOGS=30                  # 可选，最大保留日志数量，默认 30
GIST_KEEP_FIRST_FILE=false        # 可选，清理时保留第一个文件
```

#### 4. 依赖检查

脚本需要 `jq` 工具来处理 JSON：

```bash
# Debian/Ubuntu
sudo apt install jq

# macOS
brew install jq
```

### 使用效果

每次备份完成后，脚本会自动创建新的日志文件到 Gist，文件名格式为 `<prefix>-YYYY-MM-DD_HH-MM-SS.log`（精确到秒），包含：

- 备份状态（成功/失败）
- 执行时间和耗时
- 配置信息
- 完整的日志输出

默认前缀为 `yewresin-backup`，可以通过 `GIST_LOG_PREFIX` 环境变量自定义。

### 自动清理旧日志

上传成功后，脚本会自动检查并清理超出数量限制的旧日志文件：

- `GIST_MAX_LOGS`：最大保留日志数量（默认 30，设为 0 禁用清理）
- `GIST_KEEP_FIRST_FILE`：设为 `true` 时，清理会跳过按文件名排序最小的文件

**使用场景**：如果你想在 Gist 中保留一个自定义的标题/描述文件（如 `00-README.md`），可以：
1. 在 Gist 中创建一个文件名较小的文件（如 `00-README.md`）作为标题
2. 设置 `GIST_KEEP_FIRST_FILE=true`

这样清理时会自动跳过这个标题文件，只清理日志文件。

你可以通过 `https://gist.github.com/your_username/GIST_ID` 访问查看所有日志。Gist 会按文件名自动排序，由于日期是递增的，最新的备份日志在最下面。

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

> **注意**：
> - 以下示例假设服务器使用 UTC 时区，时间已转换为北京时间对应的 UTC 时间
> - 请先确认服务器时区（`timedatectl` 或 `date`），如服务器使用本地时区则无需转换
> - 脚本会自动将日志输出到 `LOG_FILE`，无需在 cron 中配置重定向

```bash
# 编辑 crontab
crontab -e

# 每天北京时间凌晨 3 点执行备份（UTC 19:00）
0 19 * * * /path/to/yewresin -y

# 每周日北京时间凌晨 2 点执行备份（UTC 周六 18:00）
0 18 * * 6 /path/to/yewresin -y

# 每 6 小时执行一次（UTC 0点、6点、12点、18点）
0 */6 * * * /path/to/yewresin -y

# 每天北京时间凌晨 3 点和 15 点执行（UTC 19:00 和 07:00）
0 7,19 * * * /path/to/yewresin -y

# 每月 2 日和 16 日北京时间凌晨 4 点执行（对应 UTC 时间 1 日和 15 日的 20:00）
0 20 1,15 * * /path/to/yewresin -y
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
ExecStart=/path/to/yewresin -y
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

### 使用 sudo cron 运行

Docker 操作通常需要 root 权限，但 Kopia 和 rclone 的配置文件默认存储在**当前用户**的 home 目录下。如果你以普通用户配置了 Kopia 和 rclone，然后在 `sudo crontab` 中运行脚本，root 用户会找不到配置文件。

通过 `KOPIA_CONFIG_FILE` 和 `RCLONE_CONFIG` 环境变量，你可以将配置文件路径指向原来的非 root 用户目录，避免手动复制配置：

```bash
# 假设你以 yewfence 用户配置了 kopia 和 rclone
# 在 .env 中添加以下配置：

# Kopia 配置文件（默认位于 ~/.config/kopia/repository.config）
KOPIA_CONFIG_FILE="/home/yewfence/.config/kopia/repository.config"

# Rclone 配置文件（默认位于 ~/.config/rclone/rclone.conf）
RCLONE_CONFIG="/home/yewfence/.config/rclone/rclone.conf"
```

然后在 root 的 crontab 中配置定时任务：

```bash
sudo crontab -e

# 每天北京时间凌晨 3 点执行（UTC 19:00）
0 19 * * * /home/yewfence/yewresin/yewresin -y
```

> **提示**：
> - 用 `echo ~yewfence` 确认用户的 home 目录路径
> - 如果你的普通用户在 `docker` 用户组中可以免 sudo 运行 Docker，也可以直接使用普通用户的 `crontab -e` 配置，这样无需额外指定配置文件路径

## 异地恢复引导

当服务器需要迁移或灾难恢复时，按以下步骤从备份中恢复数据。

### 1. 安装依赖

在新机器上安装 Kopia 和 rclone（如果备份使用了 rclone 远端）：

```bash
# 安装 kopia
curl -s https://kopia.io/signing-key | sudo gpg --dearmor -o /etc/apt/keyrings/kopia-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kopia-keyring.gpg] http://packages.kopia.io/apt/ stable main" | sudo tee /etc/apt/sources.list.d/kopia.list
sudo apt update && sudo apt install kopia

# 安装 rclone（如果备份存储在云端）
curl https://rclone.org/install.sh | sudo bash
```

### 2. 配置 rclone（如需）

如果你的 Kopia 仓库使用 rclone 作为存储后端，需要先在新机器上配好相同的远端。

**最简单的方法**：直接从旧机器复制配置文件。先在旧机器上找到文件位置：

```bash
rclone config file
# Configuration file is stored at:
# /home/username/.config/rclone/rclone.conf
```

将该文件复制到新机器的相同路径即可。也可以在新机器上重新交互式配置：

```bash
rclone config
```

### 3. 连接 Kopia 仓库

**最简单的方法**：在旧机器上用 `kopia repository status -t -s` 获取连接令牌，在新机器上用令牌一步重连，无需重新配置 rclone：

```bash
# 在旧机器上运行，输出中包含完整的重连命令
kopia repository status -t -s
# To reconnect to the repository use:
# $ kopia repository connect from-config --token eyJ2ZXJz...

# 在新机器上直接执行上面的命令即可
kopia repository connect from-config --token eyJ2ZXJz...
```

如果没有旧机器可访问，可以重新手动连接：

```bash
# 连接 rclone 远端仓库（与备份时的 EXPECTED_REMOTE 一致）
kopia repository connect rclone --remote-path="gdrive:backup"

# 或连接本地/文件系统仓库
kopia repository connect filesystem --path /path/to/kopia-repo

# 或使用 S3 仓库
kopia repository connect s3 --bucket=my-backup-bucket \
    --access-key=AKIAIOSFODNN7EXAMPLE \
    --secret-access-key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

> 手动连接时需要输入创建仓库时设置的密码。令牌方式已内含凭据，无需再输入密码。

### 4. 查看可用快照

```bash
kopia snapshot list
```

输出示例：

```text
user@hostname:/opt/docker_file
  2025-12-20 03:00:15 UTC k1a2b3c4d5e6f7 102.6 MB
  2025-12-21 03:00:12 UTC k8a9b0c1d2e3f4 103.1 MB (+0.5 MB)
```

### 5. 恢复数据

**方式一：直接恢复到目标目录（推荐）**

```bash
# 恢复整个快照到指定目录
kopia snapshot restore <snapshot-id> /opt/docker_file
```

**方式二：挂载后手动选择文件**

```bash
mkdir /tmp/kopia-mount
kopia mount <snapshot-id> /tmp/kopia-mount &

# 浏览并按需复制文件
ls /tmp/kopia-mount/
cp -r /tmp/kopia-mount/some-service /opt/docker_file/

# 完成后卸载
umount /tmp/kopia-mount
```

### 6. 恢复后启动服务

```bash
# 逐个进入服务目录启动（docker compose 会自动检测 compose 文件）
cd /opt/docker_file
for dir in */; do
    if ls "$dir"compose*.y*ml "$dir"docker-compose*.y*ml 2>/dev/null | head -1 > /dev/null; then
        echo "Starting $dir..."
        (cd "$dir" && docker compose up -d)
    fi
done
```

> 更多 Kopia 用法参考 [Kopia 官方文档](https://kopia.io/docs/)，rclone 配置参考 [rclone 官方文档](https://rclone.org/docs/)。

## Kopia Web UI

Kopia 内置了一个 Web 界面，可以直观地浏览快照、手动触发备份、查看仓库状态等：

```bash
kopia server start
```

启动后在浏览器访问 `http://localhost:51515`。

## License

MIT
