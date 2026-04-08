# 定时任务配置

## 快速开始

推荐直接用 `schedule` 子命令来管理定时任务。默认后端是当前用户的 `cron`，不需要 `sudo`：

```bash
yewresin schedule install
yewresin schedule status

# 不需要时可以卸载
yewresin schedule uninstall
```

默认表达式是服务器本地时区的 `0 3 * * *`。如果服务器跑的是 UTC，那就按 UTC 去换算你想要的时间。

可以自定义部分参数：

```bash
# 每 6 小时执行一次
yewresin schedule install --expr "0 */6 * * *"

# Linux 下切到 systemd user timer
yewresin schedule install --backend systemd-user --on-calendar "*-*-* 03:00:00"
```

- `cron` 后端适用于 Linux / macOS
- `systemd-user` 只支持 Linux
- `schedule` 会写入 `yewresin` 的绝对路径，并尽量带上自动发现到的 `--config` 路径

## 手动配置 Cron

### Cron 表达式格式

```text
┌───────────── 分钟 (0-59)
│ ┌─────────── 小时 (0-23)
│ │ ┌───────── 日期 (1-31)
│ │ │ ┌─────── 月份 (1-12)
│ │ │ │ ┌───── 星期 (0-7，0 和 7 都表示周日)
│ │ │ │ │
* * * * *
```

### 示例

```bash
crontab -e

# 每天凌晨 3 点执行
0 3 * * * /path/to/yewresin --config /path/to/config.toml -y

# 每 6 小时执行一次
0 */6 * * * /path/to/yewresin --config /path/to/config.toml -y
```

## 手动配置 systemd user timer

如果你想用 `systemd`，更推荐用户级 timer，而不是系统级 `/etc/systemd/system`。

创建 `~/.config/systemd/user/yewresin-backup.service`：

```ini
[Unit]
Description=YewResin Docker Backup
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
ExecStart=/path/to/yewresin --config /path/to/config.toml -y
StandardOutput=journal
StandardError=journal
```

创建 `~/.config/systemd/user/yewresin-backup.timer`：

```ini
[Unit]
Description=Run YewResin backup

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
```

启用：

```bash
systemctl --user daemon-reload
systemctl --user enable --now yewresin-backup.timer

# 查看状态
systemctl --user status yewresin-backup.timer
journalctl --user -u yewresin-backup.service -f
```

如果你希望退出登录后也继续触发 timer，还需要管理员额外执行一次：

```bash
sudo loginctl enable-linger <username>
```

## 需要 root 调度时

使用 root 的 `crontab` 或系统级 systemd timer，注意 `root` 看不到普通用户的 Kopia / rclone 配置，这时候通常要显式指定：

```bash
KOPIA_CONFIG_FILE="/home/youruser/.config/kopia/repository.config"
RCLONE_CONFIG="/home/youruser/.config/rclone/rclone.conf"
```

## 注意事项

- **使用绝对路径**：手动写 `cron` 或 `systemd` 时，不要依赖交互式 shell 的 `PATH`
