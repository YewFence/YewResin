# 定时任务配置

## Cron 表达式格式

```text
┌───────────── 分钟 (0-59)
│ ┌─────────── 小时 (0-23)
│ │ ┌───────── 日期 (1-31)
│ │ │ ┌─────── 月份 (1-12)
│ │ │ │ ┌───── 星期 (0-7，0 和 7 都表示周日)
│ │ │ │ │
* * * * *
```

## 常用配置示例

> **注意**：
> - 以下示例假设服务器使用 UTC 时区，时间已转换为北京时间对应的 UTC 时间
> - 请先确认服务器时区（`timedatectl` 或 `date`），如服务器使用本地时区则无需转换
> - 脚本会自动将日志输出到 `LOG_FILE`，无需在 cron 中配置重定向

```bash
# 编辑 crontab
crontab -e

# 每天北京时间凌晨 3 点执行备份（UTC 19:00）
0 19 * * * /path/to/yewresin -y

# 每周日北京时间凌晨 2 点执行备份（UTC 周六 18:00）
0 18 * * 6 /path/to/yewresin -y

# 每 6 小时执行一次（UTC 0点、6点、12点、18点）
0 */6 * * * /path/to/yewresin -y

# 每天北京时间凌晨 3 点和 15 点执行（UTC 19:00 和 07:00）
0 7,19 * * * /path/to/yewresin -y

# 每月 2 日和 16 日北京时间凌晨 4 点执行（对应 UTC 时间 1 日和 15 日的 20:00）
0 20 1,15 * * /path/to/yewresin -y
```

## 使用 Systemd Timer

相比 cron，systemd timer 提供更好的日志管理和错误处理。

创建服务文件 `/etc/systemd/system/yewresin-backup.service`：

```ini
[Unit]
Description=YewResin Docker Backup
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
ExecStart=/path/to/yewresin -y
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

## 注意事项

- **使用绝对路径**：cron 环境的 PATH 与交互式 shell 不同，务必使用脚本的绝对路径
- **日志轮转**：建议配合 logrotate 管理日志文件大小
- **错误通知**：脚本已集成 Apprise 通知，配置后可自动发送备份结果
- **避免重叠**：脚本内置锁机制，防止多个备份任务同时运行

## 使用 sudo cron 运行

Docker 操作通常需要 root 权限，但 Kopia 和 rclone 的配置文件默认存储在**当前用户**的 home 目录下。如果你以普通用户配置了 Kopia 和 rclone，然后在 `sudo crontab` 中运行脚本，root 用户会找不到配置文件。

通过 `KOPIA_CONFIG_FILE` 和 `RCLONE_CONFIG` 环境变量，你可以将配置文件路径指向原来的非 root 用户目录，避免手动复制配置：

```bash
# 假设你以 yewfence 用户配置了 kopia 和 rclone
# 在 .env 中添加以下配置：

# Kopia 配置文件（默认位于 ~/.config/kopia/repository.config）
KOPIA_CONFIG_FILE="/home/yewfence/.config/kopia/repository.config"

# Rclone 配置文件（默认位于 ~/.config/rclone/rclone.conf）
RCLONE_CONFIG="/home/yewfence/.config/rclone/rclone.conf"
```

然后在 root 的 crontab 中配置定时任务：

```bash
sudo crontab -e

# 每天北京时间凌晨 3 点执行（UTC 19:00）
0 19 * * * /home/yewfence/yewresin/yewresin -y
```

> **提示**：
> - 用 `echo ~$USER` 确认当前用户的 home 目录路径
> - 如果你的普通用户在 `docker` 用户组中可以免 sudo 运行 Docker，也可以直接使用普通用户的 `crontab -e` 配置，这样无需额外指定配置文件路径
