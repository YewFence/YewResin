# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

YewResin 是一个 Docker Compose 服务的自动化备份脚本，使用 Kopia + rclone 实现本地快照与云端同步。脚本会依次停止所有 Docker 服务，创建一致性快照，然后按优先级恢复服务。

## 构建命令

```bash
# 合并 src/ 模块生成 yewresin.sh（必须在修改代码后执行）
make build

# 清理生成的脚本
make clean
```

## 架构

脚本采用模块化设计，源代码在 `src/` 目录下，按数字前缀顺序拼接：

| 模块 | 职责 |
|------|------|
| `00-header.sh` | shebang、set -eo pipefail、记录开始时间 |
| `01-logging.sh` | 日志输出（tee 到文件和终端）、`log()` 函数 |
| `02-args.sh` | 命令行参数解析（`--dry-run`、`-y`、`--help`） |
| `03-config.sh` | 配置加载（从 `.env` 读取）、默认值、`print_config()` |
| `04-utils.sh` | 通用工具函数 |
| `05-notification.sh` | Apprise 通知发送 |
| `06-gist.sh` | GitHub Gist 日志上传和清理 |
| `07-dependencies.sh` | 依赖检查（rclone、kopia） |
| `08-services.sh` | Docker 服务管理：停止、启动、状态检查、cleanup |
| `09-main.sh` | 主流程：停止服务 → Kopia 快照 → 启动服务 |

**核心函数在 `08-services.sh`：**
- `stop_all_services()` / `start_all_services()` - 批量服务管理
- `is_service_running()` - 检测服务运行状态
- `cleanup()` - 异常退出时自动恢复服务（trap EXIT）

**服务启停优先级逻辑：**
- 优先服务（`PRIORITY_SERVICES`）：最后停止，最先启动（如网关 caddy/nginx）
- 普通服务：先停止，后启动
- 只恢复原本在运行的服务（通过 `RUNNING_SERVICES` 关联数组追踪）

## 开发流程

1. 修改 `src/` 下的模块文件
2. 执行 `make build` 重新生成 `yewresin.sh`
3. 提交 `src/`、`Makefile` 和 `yewresin.sh`

## 运行与测试

```bash
# 本地模拟运行（不执行实际操作）
./yewresin.sh --dry-run

# 执行备份（需确认）
./yewresin.sh

# 跳过确认（用于 cron）
./yewresin.sh -y
```

## 配置

必须在 `.env` 中设置：
- `BASE_DIR` - Docker Compose 项目总目录
- `EXPECTED_REMOTE` - Kopia 远程路径（如 `gdrive:backup`）

可参考 `.env.example` 查看完整配置项。

## CI/CD

- `dev-release.yml` - main 分支推送后自动构建并发布到 `latest` tag
- `prod-release.yml` - 手动触发正式版本发布
