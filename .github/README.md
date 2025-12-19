# GitHub Actions 备份配置说明

## 功能说明

这个 GitHub Action 会通过 SSH 连接到你的服务器并执行备份脚本。

## 配置步骤

### 1. 在 GitHub 仓库设置 Secrets

前往仓库的 `Settings` → `Secrets and variables` → `Actions` → `New repository secret`，添加以下密钥:

- **SSH_PRIVATE_KEY**: 你的 SSH 私钥内容
- **SSH_HOST**: 服务器地址 (例如: `192.168.1.100` 或 `example.com`)
- **SSH_USER**: SSH 用户名 (例如: `root` 或 `ubuntu`)
- **SSH_PORT**: SSH 端口号 (例如: `22`，如果是默认端口可以省略)

### 2. 生成 SSH 密钥对 (如果还没有)

```bash
ssh-keygen -t rsa -b 4096 -C "github-actions-backup"
```

生成后:
- 将公钥 (`~/.ssh/id_rsa.pub`) 添加到服务器的 `~/.ssh/authorized_keys`
- 将私钥 (`~/.ssh/id_rsa`) 的内容复制到 GitHub Secrets 的 `SSH_PRIVATE_KEY`

### 3. 配置定时任务

默认配置是每天北京时间凌晨 3 点执行。如需修改，编辑 [backup.yml](.github/workflows/backup.yml) 中的 cron 表达式:

```yaml
schedule:
  - cron: '0 19 * * *'  # UTC 19:00 = 北京时间 03:00
```

### 4. 手动触发

在 GitHub 仓库的 `Actions` 标签页 → 选择 `Docker Backup` → 点击 `Run workflow` 即可手动触发备份。

## 查看日志

备份执行后，可以在 `Actions` 标签页查看详细的执行日志。

## 环境变量说明

- **IGNORE_BACKUP_ERROR**: 即使备份失败也继续执行后续启动步骤 (默认: `true`)
  - 可以在手动触发时选择 `true` 或 `false`
  - 定时任务默认使用 `true`

## 注意事项

1. 确保服务器上的 `backup.sh` 脚本有执行权限
2. 确保 SSH 密钥权限正确配置
3. 建议先手动触发测试一次，确认配置正确
4. 如果使用了 `.env` 文件，确保服务器上有对应的配置文件
