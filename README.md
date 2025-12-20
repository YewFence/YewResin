# YewResin - Docker 服务备份工具

一个自动化的 Docker Compose 服务备份脚本，使用 Kopia + rclone 实现本地快照与云端同步。

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
mkdir ~/yewresin
cd ~/yewresin
wget https://github.com/YewFence/YewResin/releases/download/latest/backup.sh
```

该标签内的脚本会在main分支推送后自动更新，也可以自行下载指定版本的脚本

> 也可以下载源码后自定义逻辑
> ```bash
> git clone https://github.com/YewFence/YewResin.git
> cd YewResin
> ```
> 然后自行更改 `src/` 下的各个模块，可能需要自定义的有 `src/08-services.sh` 内路径内是否含有服务的识别逻辑，启停服务的脚本的名称/具体的命令
> 
> 然后，使用 
> ```bash 
> make
> ```
> 生成最终脚本，它会输出在项目根目录的 `backup.sh`

### 2. 配置

创建 `.env` 文件（与 `backup.sh` 同目录）：

```bash
# 在脚本所在目录下载示例文件
wget https://github.com/YewFence/YewResin/releases/download/latest/default.env.example
cp default.env.example .env
```

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

# GitHub Gist 日志上传配置（可选）
# 用于将每日备份日志推送到 GitHub Gist 持久化存储
GIST_TOKEN=your_github_personal_access_token
GIST_ID=your_gist_id
GIST_LOG_PREFIX=yewresin-backup  # 可选，日志文件名前缀，默认为 yewresin-backup
GIST_MAX_LOGS=30                 # 可选，最大保留日志数量，默认 30
GIST_KEEP_FIRST_FILE=false       # 可选，清理时保留第一个文件作为标题
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
| `KOPIA_PASSWORD` | - | Kopia 远程仓库密码 |
| `PRIORITY_SERVICES_LIST` | `caddy nginx gateway` | 优先服务列表（空格分隔） |
| `IGNORE_BACKUP_ERROR` | `true` | 备份失败时是否继续 |
| `LOCK_FILE` | `/tmp/backup_maintenance.lock` | 锁文件路径 |
| `APPRISE_URL` | - | Apprise 服务地址 |
| `APPRISE_NOTIFY_URL` | - | 通知目标 URL |
| `GIST_TOKEN` | - | GitHub Personal Access Token（需要 gist 权限）|
| `GIST_ID` | - | GitHub Gist ID（日志上传目标）|
| `GIST_LOG_PREFIX` | `yewresin-backup` | Gist 日志文件名前缀 |
| `GIST_MAX_LOGS` | `30` | Gist 最大保留日志数量（设为 0 禁用清理）|
| `GIST_KEEP_FIRST_FILE` | `true` | 清理时保留第一个文件（用于自定义 Gist 标题）|
| `CONFIG_FILE` | `./backup.sh` 同目录的 `.env` | 配置文件路径 |

## 关键要求

### 目录结构要求

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

### 启停逻辑

- 若服务目录下有 `docker-compose.yml` 脚本将尝试使用 `docker compose up -d` `docker compose down` 启停服务
- 若服务目录下无 `docker-compose.yml` 脚本将尝试使用 `compose-down.sh`/`compose-up.sh` 启停服务，请确认启停脚本已正确配置

## 开发说明

脚本采用模块化结构，源代码位于 `src/` 目录，通过 Makefile 合并生成最终的 `backup.sh`。

### 源码结构

```
YewResin/
├── backup.sh              # 生成的脚本（由 make build 生成）
├── Makefile               # 构建工具
└── src/                   # 模块源文件
    ├── 00-header.sh       # shebang 和初始化
    ├── 01-logging.sh      # 日志捕获和 log() 函数
    ├── 02-args.sh         # 命令行参数解析
    ├── 03-config.sh       # 配置加载和默认值
    ├── 04-utils.sh        # 通用工具函数
    ├── 05-notification.sh # 通知相关函数
    ├── 06-gist.sh         # GitHub Gist 上传
    ├── 07-dependencies.sh # 依赖检查
    ├── 08-services.sh     # Docker 服务管理
    └── 09-main.sh         # 主流程逻辑
```

### 构建命令

```bash
# 合并模块生成 backup.sh
make build

# 删除生成的 backup.sh
make clean

# 查看帮助
make help
```

### 开发流程

1. 修改 `src/` 目录下的模块文件
2. 运行 `make build` 重新生成 `backup.sh`
3. 提交 `src/`、`Makefile` 和 `backup.sh`

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

你可以通过 `https://gist.github.com/your_username/GIST_ID` 访问查看所有日志。Gist 会按文件名自动排序，最新的备份日志在最上面。

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
