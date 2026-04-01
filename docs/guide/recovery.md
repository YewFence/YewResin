# 异地恢复引导

当服务器需要迁移或灾难恢复时，按以下步骤从备份中恢复数据。

## 1. 安装依赖

在新机器上安装 Kopia 和 rclone（如果备份使用了 rclone 远端）：

```bash
# 以 Linux 为例
# 安装 kopia
curl -s https://kopia.io/signing-key | sudo gpg --dearmor -o /etc/apt/keyrings/kopia-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kopia-keyring.gpg] http://packages.kopia.io/apt/ stable main" | sudo tee /etc/apt/sources.list.d/kopia.list
sudo apt update && sudo apt install kopia

# 安装 rclone（如果使用了）
sudo -v ; curl https://rclone.org/install.sh | sudo bash
```

> 其他平台的安装方式参见 [Kopia 安装文档](https://kopia.io/docs/installation/) 和 [rclone 安装文档](https://rclone.org/downloads/)。

## 2. 配置 rclone（如需）

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

## 3. 连接 Kopia 仓库

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

## 4. 查看可用快照

```bash
kopia snapshot list
```

输出示例：

```text
user@hostname:/opt/docker_file
  2025-12-20 03:00:15 UTC k1a2b3c4d5e6f7 102.6 MB
  2025-12-21 03:00:12 UTC k8a9b0c1d2e3f4 103.1 MB (+0.5 MB)
```

## 5. 恢复数据

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

## 6. 恢复后启动服务

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
