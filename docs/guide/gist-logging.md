# GitHub Gist 日志推送

脚本支持将每日备份日志自动推送到 GitHub Gist，实现日志持久化和远程查看。

## 为什么使用 Gist？

- 持久化存储，不会被清理
- 每次备份独立文件（如 `yewresin-backup-2025-12-20_03-00-15.log`），精确到秒
- 有版本历史，可以查看每次备份的变化
- 免费、稳定，支持 API 操作
- 可以通过链接方便地分享和查看

## 配置步骤

### 1. 创建 GitHub Personal Access Token

访问 [GitHub Token 设置](https://github.com/settings/tokens/new)，创建一个新的 token：

- **Note**: YewResin Backup Logger
- **Expiration**: 自定义（建议选择较长期限）
- **Select scopes**: 只勾选 `gist` 权限

创建后复制 token（只会显示一次）。

### 2. 创建一个空的 Gist

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

### 3. 配置环境变量

在 `.env` 文件中添加：

```bash
GIST_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
GIST_ID=abc123def456789
GIST_LOG_PREFIX=my-server-backup  # 可选，自定义日志文件名前缀
GIST_MAX_LOGS=30                  # 可选，最大保留日志数量，默认 30
GIST_KEEP_FIRST_FILE=false        # 可选，清理时保留第一个文件
```

### 4. 依赖检查

脚本需要 `jq` 工具来处理 JSON：

```bash
# Debian/Ubuntu
sudo apt install jq

# macOS
brew install jq
```

## 使用效果

每次备份完成后，脚本会自动创建新的日志文件到 Gist，文件名格式为 `<prefix>-YYYY-MM-DD_HH-MM-SS.log`（精确到秒），包含：

- 备份状态（成功/失败）
- 执行时间和耗时
- 配置信息
- 完整的日志输出

默认前缀为 `yewresin-backup`，可以通过 `GIST_LOG_PREFIX` 环境变量自定义。

## 自动清理旧日志

上传成功后，脚本会自动检查并清理超出数量限制的旧日志文件：

- `GIST_MAX_LOGS`：最大保留日志数量（默认 30，设为 0 禁用清理）
- `GIST_KEEP_FIRST_FILE`：设为 `true` 时，清理会跳过按文件名排序最小的文件

**使用场景**：如果你想在 Gist 中保留一个自定义的标题/描述文件（如 `00-README.md`），可以：

1. 在 Gist 中创建一个文件名较小的文件（如 `00-README.md`）作为标题
2. 设置 `GIST_KEEP_FIRST_FILE=true`

这样清理时会自动跳过这个标题文件，只清理日志文件。

你可以通过 `https://gist.github.com/your_username/GIST_ID` 访问查看所有日志。Gist 会按文件名自动排序，由于日期是递增的，最新的备份日志在最下面。
