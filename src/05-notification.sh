
# ================= 通知函数 =================
# 格式化通知响应输出
format_notification_response() {
    local response="$1"
    local timestamp
    timestamp=$(date -u '+%Y-%m-%d %H:%M:%S UTC')

    if echo "$response" | grep -q '"status"'; then
        local status msg
        status=$(echo "$response" | sed -n 's/.*"status"[[:space:]]*:[[:space:]]*\([0-9]*\).*/\1/p')
        msg=$(echo "$response" | sed -n 's/.*"message"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        if [ "$status" = "200" ]; then
            printf "[%s] 通知发送成功: 状态=%-3s 信息=%s\n" "$timestamp" "$status" "$msg"
        else
            printf "[%s] 通知发送失败: 状态=%-3s 信息=%s\n" "$timestamp" "$status" "$msg"
        fi
    elif [ -n "$response" ]; then
        echo "[$timestamp] 警告：通知发送失败 - $response"
    fi
}

# 发送通知函数（需要配置 APPRISE_URL 和 APPRISE_NOTIFY_URL）
send_notification() {
    local title="$1"
    local body="$2"

    # 如果没配置 Apprise，跳过通知
    if [ -z "$APPRISE_URL" ] || [ -z "$APPRISE_NOTIFY_URL" ]; then
        return 0
    fi

    local response
    response=$(curl -X POST "$APPRISE_URL" \
        -H "Content-Type: application/json" \
        -d "{
            \"urls\": \"$APPRISE_NOTIFY_URL\",
            \"body\": \"$body\",
            \"title\": \"$title\"
        }" \
        --max-time 10 \
        --silent \
        --show-error 2>&1)

    format_notification_response "$response"
}
