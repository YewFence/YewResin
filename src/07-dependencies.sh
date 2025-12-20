
# ================= 依赖检查 =================
check_dependencies() {
    local has_error=false
    local error_msg=""

    # 检查 rclone
    if ! command -v rclone &>/dev/null; then
        echo "[错误] rclone 未安装"
        echo "       请访问 https://rclone.org/downloads/ 下载安装"
        error_msg+="rclone 未安装; "
        has_error=true
    elif ! rclone listremotes 2>/dev/null | grep -q .; then
        echo "[错误] rclone 已安装但未配置任何远程存储"
        echo "       请运行 'rclone config' 配置远程存储"
        echo "       文档: https://rclone.org/downloads/"
        error_msg+="rclone 未配置远程存储; "
        has_error=true
    fi

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

    # 如果基础依赖检查失败，直接退出
    if [ "$has_error" = true ]; then
        echo ""
        echo "[失败] 依赖检查未通过，脚本退出"
        send_notification "❌ 备份失败" "依赖检查未通过: ${error_msg}请手动配置后重试"
        exit 1
    fi

    # 检查 Kopia 仓库连接状态并尝试连接
    echo "[检查] Kopia 仓库 $EXPECTED_REMOTE 连接状态..."
    local repo_status
    repo_status=$(kopia repository status 2>&1)

    if echo "$repo_status" | grep -q "\"remotePath\": \"$EXPECTED_REMOTE\""; then
        echo "[✓] Kopia 仓库已正确连接到 $EXPECTED_REMOTE"
    else
        echo "[警告] Kopia 仓库未连接或连接到错误的远程路径"
        if [ -n "$KOPIA_PASSWORD" ]; then
            echo "[尝试] 使用已配置的 KOPIA_PASSWORD 尝试重新连接仓库 ..."
            if ! kopia repository connect rclone --remote-path="$EXPECTED_REMOTE" --password="$KOPIA_PASSWORD"; then
                echo "[错误] 无法连接到 Kopia 仓库 $EXPECTED_REMOTE"
                echo "       请检查 rclone 配置和网络连接"
                echo "       文档: https://kopia.io/docs/installation/"
                echo ""
                echo "[失败] 依赖检查未通过，脚本退出"
                send_notification "❌ 备份失败" "Kopia 仓库连接失败，请检查 rclone/kopia 配置后手动重试"
                exit 1
            fi
            echo "[✓] 成功连接到 $EXPECTED_REMOTE"
        else
            echo "[提示] 未检测到 KOPIA_PASSWORD，无法自动连接仓库"
            echo "       请设置 KOPIA_PASSWORD 环境变量后手动重试"
            echo ""
            echo "[失败] 依赖检查未通过，脚本退出"
            send_notification "❌ 备份失败" "Kopia 仓库未连接且未配置 KOPIA_PASSWORD，无法自动重试"
            exit 1
        fi
    fi

    echo "[✓] 依赖检查通过: rclone 和 kopia 均已正确配置"
}
