
# ================= 依赖检查 =================

# 执行 kopia 命令（支持自定义配置文件路径，正确处理空格）
run_kopia() {
    if [ -n "$KOPIA_CONFIG_FILE" ]; then
        kopia --config-file="$KOPIA_CONFIG_FILE" "$@"
    else
        kopia "$@"
    fi
}

# 获取 kopia 命令显示字符串（仅用于日志输出）
get_kopia_cmd_display() {
    if [ -n "$KOPIA_CONFIG_FILE" ]; then
        echo "kopia --config-file=\"$KOPIA_CONFIG_FILE\""
    else
        echo "kopia"
    fi
}

check_dependencies() {
    local has_error=false
    local error_msg=""

    # 检查 kopia
    if ! command -v kopia &>/dev/null; then
        echo "[错误] kopia 未安装"
        echo "       请访问 https://kopia.io/docs/installation/ 下载安装"
        error_msg+="kopia 未安装; "
        has_error=true
    fi
    if [ -z "$EXPECTED_REMOTE" ]; then
        echo "[错误] Kopia 备份用远程仓库路径未配置"
        echo "       请在配置文件中设置 EXPECTED_REMOTE"
        send_notification "❌ 备份失败" "Kopia 备份用远程仓库路径未配置"
        exit 1
    fi

    # 检查自定义配置文件是否存在
    if [ -n "$KOPIA_CONFIG_FILE" ] && [ ! -f "$KOPIA_CONFIG_FILE" ]; then
        echo "[错误] 指定的 Kopia 配置文件不存在: $KOPIA_CONFIG_FILE"
        error_msg+="Kopia 配置文件不存在; "
        has_error=true
    fi
    if [ -n "$RCLONE_CONFIG" ] && [ ! -f "$RCLONE_CONFIG" ]; then
        echo "[错误] 指定的 Rclone 配置文件不存在: $RCLONE_CONFIG"
        error_msg+="Rclone 配置文件不存在; "
        has_error=true
    fi

    # 如果基础依赖检查失败，直接退出
    if [ "$has_error" = true ]; then
        echo ""
        echo "[失败] 依赖检查未通过，脚本退出"
        send_notification "❌ 备份失败" "依赖检查未通过: ${error_msg}请手动配置后重试"
        exit 1
    fi

    # 检查 Kopia 仓库连接状态
    echo "[检查] Kopia 仓库 $EXPECTED_REMOTE 连接状态..."
    local repo_status
    repo_status=$(run_kopia repository status 2>&1)

    if echo "$repo_status" | grep -q "\"remotePath\": \"$EXPECTED_REMOTE\""; then
        echo "[✓] Kopia 仓库已正确连接到 $EXPECTED_REMOTE"
    else
        echo "[错误] Kopia 仓库未连接或连接到错误的远程路径"
        echo "       请手动执行 'kopia repository connect' 连接仓库"
        echo "       文档: https://kopia.io/docs/installation/"
        echo ""
        echo "[失败] 依赖检查未通过，脚本退出"
        send_notification "❌ 备份失败" "Kopia 仓库未连接或路径不匹配，请手动连接后重试"
        exit 1
    fi

    echo "[✓] 依赖检查通过: kopia 仓库已正确连接"
}
