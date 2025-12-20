
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

    # 如果基础依赖检查失败，直接退出
    if [ "$has_error" = true ]; then
        echo ""
        echo "[失败] 依赖检查未通过，脚本退出"
        send_notification "❌ 备份失败" "依赖检查未通过: ${error_msg}请手动配置后重试"
        exit 1
    fi

    # 检查 Kopia 仓库连接状态并尝试连接
    echo "[检查] Kopia 仓库连接状态..."
    local repo_status
    repo_status=$(kopia repository status 2>&1)

    if echo "$repo_status" | grep -q "\"remotePath\": \"$EXPECTED_REMOTE\""; then
        echo "[✓] Kopia 仓库已正确连接到 $EXPECTED_REMOTE"
    else
        echo "[警告] Kopia 仓库未连接或连接到错误的远程路径"
        echo "[尝试] 重新连接到 $EXPECTED_REMOTE ..."
        if ! kopia repository connect rclone --remote-path="$EXPECTED_REMOTE" --password="$KOPIA_PASSWORD"; then
            echo "[错误] 无法连接到 Kopia 仓库 $EXPECTED_REMOTE"
            echo "       请检查 rclone 配置和网络连接"
            echo "       文档: https://kopia.io/docs/installation/"
            echo ""
            echo "[失败] 依赖检查未通过，脚本退出"
            send_dep_notification "❌ 备份失败" "Kopia 仓库连接失败，请检查 rclone/kopia 配置后手动重试"
            exit 1
        fi
        echo "[✓] 成功连接到 $EXPECTED_REMOTE"
    fi

    echo "[✓] 依赖检查通过: rclone 和 kopia 均已正确配置"
}
