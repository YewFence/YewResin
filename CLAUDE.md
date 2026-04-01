# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

YewResin 是一个 Docker Compose 服务的自动化备份工具，使用 Kopia + rclone 实现本地快照与云端同步。工具会依次停止所有 Docker 服务，创建一致性快照，然后按优先级恢复服务。

## 构建命令

```bash
# 构建当前平台
just build

# 构建所有平台
just all

# 运行测试
just test

# 清理构建产物
just clean
```

## 源码结构

```
YewResin/
├── main.go                     # 程序入口，CLI 参数解析
├── main_test.go               # main 包测试（confirm 函数）
├── internal/yewresin/         # 核心逻辑
│   ├── config.go              # 配置加载和验证
│   ├── config_test.go
│   ├── docker.go              # Docker Compose 服务管理
│   ├── docker_test.go
│   ├── backup.go              # Kopia 备份操作
│   ├── logger.go              # 日志系统
│   ├── logger_test.go
│   ├── gist.go                # GitHub Gist 日志上传
│   ├── notify.go              # Apprise 通知发送
│   ├── orchestrator.go        # 备份流程编排器
│   └── orchestrator_test.go
├── justfile                   # 交叉编译脚本
├── go.mod / go.sum
├── .env.example
└── .github/workflows/        # CI/CD
```

**核心逻辑在 `internal/yewresin/`：**

| 文件 | 职责 |
|------|------|
| `orchestrator.go` | 备份流程编排：锁机制、信号处理、cleanup |
| `docker.go` | 服务发现、启停、并行操作 |
| `backup.go` | Kopia 快照创建、依赖检查 |
| `config.go` | 环境变量加载、配置验证 |
| `logger.go` | slog 日志系统、文件输出 |
| `gist.go` | Gist 上传和旧日志清理 |
| `notify.go` | Apprise 异步通知 |

**服务启停优先级逻辑：**
- 优先服务（`PRIORITY_SERVICES`）：最后停止，最先启动（如网关 caddy/nginx）
- 普通服务：先停止，后启动
- 并行停止/启动，提升性能
- 只恢复原本在运行的服务

## 开发流程

1. 修改 `internal/yewresin/` 下的源文件
2. 运行 `just test` 确保测试通过
3. 运行 `just build` 构建当前平台进行本地测试
4. 提交代码

## 运行与测试

```bash
# 本地模拟运行（不执行实际操作）
./yewresin --dry-run

# 执行备份（需确认）
./yewresin

# 跳过确认（用于 cron）
./yewresin -y
```

## 配置

必须在 `.env` 中设置：
- `BASE_DIR` - Docker Compose 项目总目录
- `EXPECTED_REMOTE` - Kopia 远程路径（如 `gdrive:backup`）

可参考 `.env.example` 查看完整配置项。

## CI/CD

- `build-artifact.yml` - PR 到 main 分支后自动构建测试
- `prod-release.yml` - 推送 `v*` 标签后自动发布到 GitHub Release
