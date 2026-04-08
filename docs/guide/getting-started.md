# 快速开始

## 依赖

- [Kopia](https://kopia.io/docs/installation/) - 快照备份工具
- Docker & Docker Compose
- 可选：[rclone](https://rclone.org/downloads/) - 云存储同步工具

## 1. 安装依赖

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

## 2. 安装 YewResin

### Homebrew（macOS/Linux）

```bash
brew install yewfence/tap/yewresin
```

### 下载可执行文件

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

## 3. 配置

### Homebrew 安装 {#homebrew}

默认会优先读取用户配置目录中的配置文件：

- Linux: `~/.config/yewresin/config.toml`
- macOS: `~/Library/Application Support/yewresin/config.toml`
- Windows: `%AppData%\yewresin\config.toml`

Homebrew 安装场景下，推荐直接把 `config.toml` 放到这个位置：

```bash
# 交互式初始化默认配置
yewresin config init

# 用 EDITOR 打开配置
yewresin config edit
```

或者直接使用环境变量（适合 cron 场景）：

```bash
BASE_DIR=/opt/docker_file EXPECTED_REMOTE=gdrive:backup yewresin -y
```

### 直接下载安装

推荐同样放到用户配置目录；如果你就是想把配置跟二进制放一起，也还能继续用程序同目录的 `config.toml` / `.env`：

```bash
cd ~/yewresin
wget https://github.com/YewFence/YewResin/releases/latest/download/config.toml.example -O config.toml
```

如果你更习惯环境变量风格，也可以继续使用 `.env`：

```bash
cd ~/yewresin
wget https://github.com/YewFence/YewResin/releases/latest/download/.env.example
cp .env.example .env
```

### 必要配置项

```bash
# Docker Compose 项目总目录
BASE_DIR=/opt/docker_file
# Kopia 远程路径
EXPECTED_REMOTE=gdrive:backup
```

`config.toml` 示例：

```toml
base_dir = "/opt/docker_file"
expected_remote = "gdrive:backup"
priority_services = ["caddy", "nginx", "gateway"]

[kopia]
password = "your_kopia_password"
```

配置加载优先级：

- 当前进程中的环境变量
- `--config` 指定的 `.toml` / `.env`
- 用户配置目录中的 `config.toml`
- 用户配置目录中的 `.env`
- 程序同目录的 `config.toml`
- 程序同目录的 `.env`
- 程序内置默认值
- `BASE_DIR` 和 `EXPECTED_REMOTE` 等必填项如果最终为空，程序会直接退出

如果你把 `KOPIA_PASSWORD` 这类敏感信息放进 `config.toml` 也能生效，但更推荐交给环境变量。

也可以直接用子命令管理默认配置文件：

```bash
yewresin config init
yewresin config init --force
yewresin config edit
```

`config edit` 会优先使用 `EDITOR`，如果没设置，会按平台尝试常见编辑器作为兜底。

## 4. 运行

```bash
# 模拟运行（推荐先测试）
yewresin --dry-run

# 执行备份（需确认）
yewresin

# 跳过确认直接执行（适用于 cron）
yewresin -y
```

## 5. 配置定时执行

推荐直接用 `schedule` 子命令，默认会写入当前用户的 `cron`，不需要 `sudo`：

```bash
yewresin schedule install
yewresin schedule status
```

默认表达式是服务器本地时区的 `0 3 * * *`。也可以使用参数自定义表达式或切换后端：

```bash
# 每 6 小时执行一次
yewresin schedule install --expr "0 */6 * * *"

# Linux 下切到 systemd user timer
yewresin schedule install --backend systemd-user --on-calendar "*-*-* 03:00:00"
```

> `cron` 后端适用于 Linux / macOS，`systemd-user` 只支持 Linux、
> 
> 你也可以参考 [定时任务](/guide/scheduling) 手写 `crontab` 或 `systemd` 单元

## 6. 备份连接凭证

为了保证可以快速异地恢复，建议备份连接 Kopia 仓库与 rclone (如有)的连接凭证

### Rclone 连接凭证
```bash
rclone config file  # 确认 rclone 配置文件路径
# 复制配置文件内容并安全保存
# 此处以默认路径为例
cat ~/.config/rclone/rclone.conf
```
### Kopia 连接凭证

```bash
# 打印 kopia 仓库连接状态，输出中包含完整的重连命令
kopia repository status -t -s
# 部分示例输出：
# To reconnect to the repository use:
# $ kopia repository connect from-config --token eyJ2ZXJz...
# 复制并保存该命令内容
```

### 注意事项
- 连接凭证可以**直接用于访问仓库**，请妥善保管
- 建议将凭证存储在安全的密码管理器中，如 [Bitwarden](https://bitwarden.com/) ，或使用 [age](https://github.com/FiloSottile/age) 等工具加密保存

更多异地恢复信息请参考 [恢复指南](/guide/recovery)，

更多运行参数请参考 [配置参考](/reference/configuration)。
