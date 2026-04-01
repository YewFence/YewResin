# 工作原理

## 工作流程

1. 检查依赖（rclone、kopia）
2. 停止普通服务（并行）
3. 停止优先服务
4. 创建 Kopia 快照
5. 启动优先服务
6. 启动普通服务（并行）
7. 执行 Kopia 维护清理

## 目录结构要求

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

## 服务启停逻辑

服务名称以文件夹名称为准

### 单个服务启停逻辑

服务启停按以下优先级执行：

1. **自定义脚本优先**：若目录下存在 `compose-stop.sh`/`compose-down.sh`/`compose-up.sh`，优先使用脚本启停
2. **自动识别配置文件**：若无自定义脚本但存在 compose 配置文件，使用 `docker compose up -d` / `docker compose stop` 启停

### 优先服务启停逻辑

最后停止，最先启动，如网关/数据库等服务

## 快速失败机制

为保护数据完整性，脚本在停止服务阶段采用快速失败策略：

- 如果任何服务停止失败，脚本会**立即中止**，不会继续执行备份
- 已停止的服务会通过 cleanup 函数自动恢复
- 通过 Apprise 发送通知告知失败原因

这确保了不会在服务仍在运行（可能正在写入数据）时进行备份，避免数据库文件损坏等问题。

## 注意事项

如果 `BASE_DIR` 下存在权限敏感的目录（如 `caddy/data/caddy`、`ssl`、`ssh` 等），Kopia 可能会因权限问题报错。虽然备份仍会完成，但建议在 Kopia 策略中忽略这些目录。
