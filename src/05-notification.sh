# ================= 通知函数 =================
# format_notification_response formats and prints a UTC-timestamped summary of an Apprise notification response, extracting `status` and `message` when present and indicating success (status 200), failure (other statuses), or a warning for non-empty non-JSON responses.
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

# send_dep_notification sends a dependency-only notification via Apprise when APPRISE_URL and APPRISE_NOTIFY_URL are configured; otherwise it does nothing.
send_dep_notification() {
    local title="$1"
    local body="$2"

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

# send_notification sends a notification via Apprise using APPRISE_URL and APPRISE_NOTIFY_URL when configured; accepts a title and a body as parameters.
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