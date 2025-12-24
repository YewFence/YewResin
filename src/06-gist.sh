
# ================= GitHub Gist 上传 =================

# 清理旧的 Gist 日志文件
cleanup_old_gist_logs() {
    # 如果 GIST_MAX_LOGS 为 0 或负数，跳过清理
    if [ "$GIST_MAX_LOGS" -le 0 ] 2>/dev/null; then
        return 0
    fi

    log "检查 Gist 日志数量..."

    # 获取 Gist 信息
    local gist_info
    gist_info=$(curl -s \
        -H "Authorization: token $GIST_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/gists/$GIST_ID" \
        --max-time 30)

    if ! echo "$gist_info" | jq -e '.id' > /dev/null 2>&1; then
        log "⚠ 无法获取 Gist 信息，跳过清理"
        return 1
    fi

    # 获取所有文件名（按字母顺序排序）
    local all_files
    all_files=$(echo "$gist_info" | jq -r '.files | keys | sort | .[]')

    # 计算文件总数
    local total_files
    total_files=$(echo "$all_files" | grep -c . || echo 0)

    # 如果启用了保留第一个文件，从列表中排除
    local files_to_consider="$all_files"
    local first_file=""
    if [ "$GIST_KEEP_FIRST_FILE" = "true" ] && [ "$total_files" -gt 0 ]; then
        first_file=$(echo "$all_files" | head -n 1)
        files_to_consider=$(echo "$all_files" | tail -n +2)
        log "保留第一个文件: $first_file"
    fi

    # 计算可清理的文件数量
    local cleanable_count
    cleanable_count=$(echo "$files_to_consider" | sed '/^$/d' | wc -l)

    # 如果文件数量未超过限制，跳过清理
    if [ "$cleanable_count" -le "$GIST_MAX_LOGS" ]; then
        log "当前日志数量 ($cleanable_count) 未超过限制 ($GIST_MAX_LOGS)，无需清理"
        return 0
    fi

    # 计算需要删除的文件数量
    local delete_count=$((cleanable_count - GIST_MAX_LOGS))
    log "需要删除 $delete_count 个旧日志文件..."

    # 获取需要删除的文件列表（最旧的文件，即排序后最前面的）
    local files_to_delete
    files_to_delete=$(echo "$files_to_consider" | head -n "$delete_count")

    # 构建删除 payload
    local delete_payload
    delete_payload=$(echo "$files_to_delete" | grep -v '^$' | jq -R '{ (.): null }' | jq -s 'add // {}')

    # 执行删除
    local delete_response
    delete_response=$(curl -s -X PATCH \
        -H "Authorization: token $GIST_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"files\": $delete_payload}" \
        "https://api.github.com/gists/$GIST_ID" \
        --max-time 30)

    if echo "$delete_response" | jq -e '.id' > /dev/null 2>&1; then
        log "✓ 已清理 $delete_count 个旧日志文件"
    else
        log "⚠ 清理旧日志失败: $delete_response"
    fi
}

# 上传日志到 GitHub Gist
upload_to_gist() {
    # 如果没配置 Gist，跳过上传
    if [ -z "$GIST_TOKEN" ] || [ -z "$GIST_ID" ]; then
        return 0
    fi

    # 确保变量已经计算
    if [ -z "$HOURS" ]; then
        HOURS=0
        MINUTES=0
        SECS=0
    fi

    log "上传日志到 GitHub Gist..."

    local timestamp
    timestamp=$(date '+%Y-%m-%d_%H-%M-%S')

    # 使用自定义前缀，如果为空则使用默认值
    local prefix="${GIST_LOG_PREFIX:-yewresin-backup}"
    local filename="${prefix}-${timestamp}.log"

    # 读取日志文件内容
    local raw_log
    if [ -f "$LOG_FILE" ]; then
        raw_log=$(cat "$LOG_FILE")
    else
        raw_log="日志文件不存在"
    fi

    # 构建日志内容（包含完整执行信息）
    local log_content
    log_content=$(cat <<EOF
========================================
YewResin Docker 备份日志
========================================
执行状态: $([ "$backup_success" = true ] && echo "✅ 成功" || echo "⚠️ 有警告")
开始时间: $SCRIPT_START_DATETIME
耗时: $([ $HOURS -gt 0 ] && echo "$HOURS 小时 ")$([ $MINUTES -gt 0 ] && echo "$MINUTES 分 ")$SECS 秒
结束时间：$SCRIPT_END_DATETIME
========================================
详细日志:
========================================
$raw_log
EOF
)

    # JSON 转义（处理换行和引号）- 需要 jq
    if ! command -v jq &>/dev/null; then
        log "⚠ 未安装 jq，无法上传到 Gist"
        return 1
    fi

    log_content=$(echo "$log_content" | jq -Rs .)

    # 构建 JSON payload
    local payload
    payload=$(jq -n \
        --arg filename "$filename" \
        --argjson content "$log_content" \
        '{files: {($filename): {content: $content}}}')

    # 上传到 Gist
    local response
    response=$(curl -X PATCH \
        -H "Authorization: token $GIST_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "https://api.github.com/gists/$GIST_ID" \
        --max-time 30 \
        --silent \
        --show-error 2>&1)

    if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        log "✓ 日志已上传到 Gist: https://gist.github.com/$GIST_ID"
        # 上传成功后清理旧日志
        cleanup_old_gist_logs
    else
        log "⚠ Gist 上传失败: $response"
    fi
}
