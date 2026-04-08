# YewResin - Docker 服务备份工具

一个自动化的 Docker Compose 服务备份工具，使用 Kopia 实现本地快照与云端同步。

## 功能特点

- 自动停止所有 Docker Compose 服务，创建一致性快照
- 支持优先级服务（如网关）的顺序控制：最后停止，最先启动
- 只重启原本运行中的服务，不会启动原本停止的服务
- **快速失败**：服务停止失败时立即中止备份，避免数据损坏
- 并行停止/启动服务，性能更优
- 支持 [Apprise](https://github.com/caronc/apprise-api) 通知
- 支持 GitHub Gist 日志推送

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

### 2. 安装 YewResin

#### Homebrew（macOS/Linux）

```bash
brew install yewfence/tap/yewresin
```

#### 下载可执行文件

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

#### Homebrew 安装

默认会优先读取用户配置目录中的配置文件：

- Linux: `~/.config/yewresin/config.toml`
- macOS: `~/Library/Application Support/yewresin/config.toml`
- Windows: `%AppData%\yewresin\config.toml`

Homebrew 安装场景下，推荐直接把 `config.toml` 放到这个位置：

```bash
# 创建配置目录
mkdir -p ~/.config/yewresin
# 下载示例文件
wget https://github.com/YewFence/YewResin/releases/latest/download/config.toml.example -O ~/.config/yewresin/config.toml
# 编辑配置
vim ~/.config/yewresin/config.toml

# 运行
yewresin
```

或者直接使用环境变量（适合 cron 场景）：

```bash
BASE_DIR=/opt/docker_file EXPECTED_REMOTE=gdrive:backup yewresin -y
```

#### 直接下载安装

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

#### 必要配置项

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

如果你把 `KOPIA_PASSWORD` 这类敏感信息放进 `config.toml` 也能生效，但更推荐交给环境变量。

### 4. 运行

```bash
# 模拟运行（推荐先测试）
yewresin --dry-run

# 执行备份（需确认）
yewresin

# 跳过确认直接执行（适用于 cron）
yewresin -y
```

## 文档

完整文档请访问 [YewResin 文档站](https://yewfence.github.io/YewResin/)

- [工作原理](https://yewfence.github.io/YewResin/guide/how-it-works)
- [Gist 日志推送](https://yewfence.github.io/YewResin/guide/gist-logging)
- [定时任务](https://yewfence.github.io/YewResin/guide/scheduling)
- [异地恢复](https://yewfence.github.io/YewResin/guide/recovery)
- [配置参考](https://yewfence.github.io/YewResin/reference/configuration)

## License

MIT
