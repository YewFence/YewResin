# 开发指南

## 环境要求

- **Go 1.23+**
- **[just](https://github.com/casey/just)** - 命令运行器
- **Docker** - 用于跨平台构建（GoReleaser 通过 Docker 运行）
- **[pinact](https://github.com/suzuki-shunsuke/pinact)** - 用于 pin GitHub Actions 版本
- **[ghtkn](https://github.com/suzuki-shunsuke/ghtkn)** - （可选的）用于使 `pinact` 绕过 GitHub API 速率限制
- **[lefthook](https://github.com/evilmartians/lefthook)** - 管理 pinact 检测 hook

## 项目结构

```text
YewResin/
├── main.go                     # 程序入口，CLI 参数解析
├── main_test.go
├── internal/yewresin/          # 核心逻辑
│   ├── orchestrator.go         # 备份流程编排：锁机制、信号处理、cleanup
│   ├── docker.go               # 服务发现、启停、并行操作
│   ├── backup.go               # Kopia 快照创建、依赖检查
│   ├── config.go               # 环境变量加载、配置验证
│   ├── logger.go               # slog 日志系统、文件输出
│   ├── gist.go                 # Gist 上传和旧日志清理
│   └── notify.go               # Apprise 异步通知
├── justfile                    # 构建脚本
├── compose.goreleaser.yaml     # GoReleaser Docker Compose 配置
├── .goreleaser.yaml            # GoReleaser 发布配置
└── .github/workflows/
    ├── build-artifact.yml      # PR 构建验证
    ├── release.yml             # 正式发布（push tag）
    └── docs.yml                # 文档站部署
```

## 常用命令

```bash
# 快速构建当前平台（不需要 Docker）
just

# 运行测试
just test

# 清理构建产物
just clean
```

### Github Action 更新

```bash
# 安装检测 hook
lefthook install

# 更新 github action 版本
ghtkn get | pinact token set -stdin
pinact run -u
```

## 跨平台构建（GoReleaser）

跨平台构建通过 Docker 运行 GoReleaser，**不需要本地安装 GoReleaser**，只需要 Docker。

```bash
# 构建全平台可执行文件（linux/darwin/windows × amd64/arm64），不发布
just release-snapshot

# 模拟完整发布流程（含 changelog 生成），不推送
just release-dry
```

构建产物输出到 `dist/` 目录。

## 正式发布

正式发布需要：

1. 创建 `.env.goreleaser` 文件（参考下方配置）
2. 打 `v*` 格式的 git tag
3. 运行 `just release`

`.env.goreleaser` 示例：

```bash
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
TAP_GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx  # Homebrew Tap 仓库的 PAT
HOMEBREW_TAP_OWNER=YewFence
HOMEBREW_TAP_NAME=homebrew-tap
```

也可以直接推送 tag 触发 GitHub Actions 自动发布：

```bash
git tag v2.1.0
git push origin v2.1.0
```

## 服务启停优先级逻辑

- **优先服务**（`PRIORITY_SERVICES_LIST`，默认 `caddy nginx gateway`）：最后停止，最先启动
- **普通服务**：先停止，后启动
- 所有服务并行启停，只恢复原本在运行的服务

## 开发流程

1. 修改 `internal/yewresin/` 下的源文件
2. `just test` 确保测试通过
3. `just build` 构建本地二进制进行测试
4. 提交代码
