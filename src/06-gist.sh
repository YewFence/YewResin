
# ================= GitHub Gist 上传 =================
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
        SECONDS=0
    fi

    log "上传日志到 GitHub Gist..."

    local timestamp
    timestamp=$(date '+%Y-%m-%d_%H-%M-%S')

    # 使用自定义前缀，如果为空则使用默认值
    local prefix="${GIST_LOG_PREFIX:-yewresin-backup}"
    local filename="${prefix}-${timestamp}.log"

    # 读取日志文件内容
    local raw_log
    if [ -f "$LOG_OUTPUT_FILE" ]; then
        raw_log=$(cat "$LOG_OUTPUT_FILE")
    else
        raw_log="日志文件不存在"
    fi

    # 构建日志内容（包含完整执行信息）
    local log_content
    log_content=$(cat <<EOF
========================================
YewResin Docker 备份日志
========================================
日期: $SCRIPT_START_DATETIME
状态: $([ "$backup_success" = true ] && echo "✅ 成功" || echo "⚠️ 有警告")
耗时: $([ $HOURS -gt 0 ] && echo "$HOURS 小时 ")$([ $MINUTES -gt 0 ] && echo "$MINUTES 分 ")$SECONDS 秒
========================================

基础配置信息:
  BASE_DIR: $BASE_DIR
  EXPECTED_REMOTE: $EXPECTED_REMOTE
  PRIORITY_SERVICES: ${PRIORITY_SERVICES[*]}

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
    payload=$(cat <<EOF
{
  "files": {
    "$filename": {
      "content": $log_content
    }
  }
}
EOF
)

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

    if echo "$response" | grep -q '"id"'; then
        log "✓ 日志已上传到 Gist: https://gist.github.com/$GIST_ID"
    else
        log "⚠ Gist 上传失败: $response"
    fi
}
