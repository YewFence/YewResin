#!/bin/bash

set -eo pipefail

# ================= 记录开始时间 =================
SCRIPT_START_TIME=$(date +%s)
SCRIPT_START_DATETIME=$(date '+%Y-%m-%d %H:%M:%S')

# ================= 日志捕获 =================
# 创建临时文件保存日志输出
LOG_OUTPUT_FILE=$(mktemp)
# 使用 tee 同时输出到终端和文件
exec > >(tee -a "$LOG_OUTPUT_FILE")
exec 2>&1

log() {
    echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] $1"
}
# shellcheck shell=bash
# ================= 命令行参数解析 =================
DRY_RUN=false
SHOW_HELP=false
AUTO_CONFIRM=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run|-n)
            DRY_RUN=true
            shift
            ;;
        --help|-h)
            SHOW_HELP=true
            shift
            ;;
        -y|--yes)
            AUTO_CONFIRM=true
            shift
            ;;
        *)
            echo "未知参数: $1"
            echo "使用 --help 查看帮助"
            exit 1
            ;;
    esac
done

if [ "$SHOW_HELP" = true ]; then
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --dry-run, -n    模拟运行，只检查依赖和显示要执行的操作，不实际执行"
    echo "  -y, --yes        跳过交互式确认，自动确认执行"
    echo "  --help, -h       显示此帮助信息"
    echo ""
    echo "环境变量:"
    echo "  BASE_DIR              Docker Compose 项目目录 (默认: /opt/docker_file)"
    echo "  IGNORE_BACKUP_ERROR   备份失败时是否继续 (默认: true)"
    echo "  EXPECTED_REMOTE       Kopia 远程路径 (默认: gdrive:backup)"
    echo "  KOPIA_PASSWORD        Kopia 仓库密码 (必须通过环境变量传入)"
    echo "  PRIORITY_SERVICES_LIST 网关服务列表，空格分隔 (默认: caddy nginx gateway)"
    exit 0
fi

# ================= 配置加载 =================
# 加载环境变量配置文件（可选）
# 支持通过 CONFIG_FILE 环境变量指定配置文件路径
CONFIG_FILE="${CONFIG_FILE:-$(dirname "${BASH_SOURCE[0]}")/.env}"
if [ -f "$CONFIG_FILE" ]; then
    # shellcheck source=/dev/null
    source "$CONFIG_FILE"
fi

# ================= 配置区 =================
# 所有配置均可通过环境变量或 .env 文件覆盖

# 你的 Docker Compose 项目总目录
BASE_DIR="${BASE_DIR:-/opt/docker_file}"
# 即使 Kopia 命令失败也继续执行后续启动步骤吗？(true/false)
IGNORE_BACKUP_ERROR="${IGNORE_BACKUP_ERROR:-true}"
# 定义你的网关服务文件夹名称 (最后关，最先开)
# 通过 PRIORITY_SERVICES_LIST 环境变量设置，用空格分隔
if [ -n "$PRIORITY_SERVICES_LIST" ]; then
    IFS=' ' read -r -a PRIORITY_SERVICES <<< "$PRIORITY_SERVICES_LIST"
else
    PRIORITY_SERVICES=("caddy" "nginx" "gateway")
fi
# 锁文件路径
LOCK_FILE="${LOCK_FILE:-/tmp/backup_maintenance.lock}"
# Kopia 远程路径预期值
EXPECTED_REMOTE="${EXPECTED_REMOTE:-gdrive:backup}"
# GitHub Gist 配置（可选）
GIST_TOKEN="${GIST_TOKEN:-}"
GIST_ID="${GIST_ID:-}"
GIST_LOG_PREFIX="${GIST_LOG_PREFIX:-yewresin-backup}"
# Gist 日志清理配置
GIST_MAX_LOGS="${GIST_MAX_LOGS:-30}"
GIST_KEEP_FIRST_FILE="${GIST_KEEP_FIRST_FILE:-false}"
# ==========================================

# ================= 打印配置信息 =================
print_config() {
    echo ""
    echo "=========================================="
    echo "当前配置信息"
    echo "=========================================="
    # 使用 printf 对齐输出，%-38s 表示左对齐占 38 字符宽度
    local fmt="  %-38s %s\n"
    printf "$fmt" "BASE_DIR(工作目录):" "$BASE_DIR"
    printf "$fmt" "IGNORE_BACKUP_ERROR(忽略备份错误?):" "$IGNORE_BACKUP_ERROR"
    printf "$fmt" "EXPECTED_REMOTE(预期远程仓库):" "$EXPECTED_REMOTE"
    printf "$fmt" "PRIORITY_SERVICES(优先服务):" "${PRIORITY_SERVICES[*]}"
    printf "$fmt" "LOCK_FILE(锁文件路径):" "$LOCK_FILE"
    printf "$fmt" "DRY_RUN(模拟运行?):" "$DRY_RUN"
    printf "$fmt" "AUTO_CONFIRM(自动确认):" "$AUTO_CONFIRM"
    # Gist 配置
    if [ -n "$GIST_TOKEN" ] && [ -n "$GIST_ID" ]; then
        printf "$fmt" "GIST_ID(Gist ID):" "$GIST_ID"
        printf "$fmt" "GIST_LOG_PREFIX(Gist 日志前缀):" "$GIST_LOG_PREFIX"
        printf "$fmt" "GIST_MAX_LOGS(Gist 最大日志数):" "$GIST_MAX_LOGS"
        printf "$fmt" "GIST_KEEP_FIRST_FILE(Gist 保留首文件?):" "$GIST_KEEP_FIRST_FILE"
        printf "$fmt" "GIST_TOKEN(Gist Token):" "******(已配置)"
    else
        printf "$fmt" "GIST 日志上传:" "(未配置)"
    fi
    # 脱敏处理 KOPIA_PASSWORD
    if [ -n "$KOPIA_PASSWORD" ]; then
        printf "$fmt" "KOPIA_PASSWORD(仓库密码):" "******(已配置)"
    else
        printf "$fmt" "KOPIA_PASSWORD(仓库密码):" "(未配置)"
    fi

    # 脱敏处理通知 URL
    if [ -n "$APPRISE_URL" ]; then
        if [ ${#APPRISE_URL} -gt 35 ]; then
            local masked_url="${APPRISE_URL:0:20}...${APPRISE_URL: -10}"
        else
            local masked_url="****(已配置)"
        fi
        printf "$fmt" "APPRISE_URL(通知服务URL):" "$masked_url"
    else
        printf "$fmt" "APPRISE_URL(通知服务URL):" "(未配置)"
    fi

    if [ -n "$APPRISE_NOTIFY_URL" ]; then
        if [ ${#APPRISE_NOTIFY_URL} -gt 23 ]; then
            local masked_notify="${APPRISE_NOTIFY_URL:0:15}...${APPRISE_NOTIFY_URL: -8}"
        else
            local masked_notify="****(已配置)"
        fi
        printf "$fmt" "APPRISE_NOTIFY_URL(通知目标URL):" "$masked_notify"
    else
        printf "$fmt" "APPRISE_NOTIFY_URL(通知目标URL):" "(未配置)"
    fi
    echo "=========================================="
    echo ""
}

# ================= 工具函数 =================
# dry-run 模式下的模拟执行函数
dry_run_exec() {
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY-RUN] 将执行: $*"
        return 0
    else
        "$@"
    fi
}

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

    if ! echo "$gist_info" | grep -q '"id"'; then
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

    if echo "$delete_response" | grep -q '"id"'; then
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
耗时: $([ $HOURS -gt 0 ] && echo "$HOURS 小时 ")$([ $MINUTES -gt 0 ] && echo "$MINUTES 分 ")$SECS 秒
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

    if echo "$response" | grep -q '"id"'; then
        log "✓ 日志已上传到 Gist: https://gist.github.com/$GIST_ID"
        # 上传成功后清理旧日志
        cleanup_old_gist_logs
    else
        log "⚠ Gist 上传失败: $response"
    fi
}

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
#!/bin/bash
# shellcheck source-path=SCRIPTDIR
# This module is sourced by backup.sh and provides service management functions.
# Required external variables: DRY_RUN, BASE_DIR, LOCK_FILE, PRIORITY_SERVICES, NORMAL_SERVICES
# Required external functions: log(), send_notification()
# ================= 服务管理 =================
# 记录原本运行中的服务
declare -A RUNNING_SERVICES

# 检查服务是否正在运行
is_service_running() {
    local svc_path="$1"
    local svc_name
    svc_name=$(basename "$svc_path")

    # 检查是否有 compose 相关文件（yaml 或脚本）
    local has_compose=false
    if [ -x "$svc_path/compose-status.sh" ] || [ -x "$svc_path/compose-up.sh" ] || [ -x "$svc_path/compose-log.sh" ]; then
        has_compose=true
    elif find "$svc_path" -maxdepth 1 \( -name "docker-compose*.yml" -o -name "docker-compose*.yaml" -o -name "compose*.yml" -o -name "compose*.yaml" \) -print -quit 2>/dev/null | grep -q .; then
        has_compose=true
    fi

    if [ "$has_compose" = true ]; then
        local running_containers
        # 优先在目录下执行（自动识别 yaml），否则用项目名
        running_containers=$(cd "$svc_path" && docker compose ps -q 2>/dev/null | wc -l)
        if [ "$running_containers" -gt 0 ]; then
            return 0
        fi
        # 备用：用项目名检查
        running_containers=$(docker compose -p "$svc_name" ps -q 2>/dev/null | wc -l)
        if [ "$running_containers" -gt 0 ]; then
            return 0
        fi
    fi

    return 1
}

# 停止单个服务的函数
stop_service() {
    local svc_path="$1"
    local svc_name
    svc_name=$(basename "$svc_path")

    # 先检查服务是否在运行
    if ! is_service_running "$svc_path"; then
        log "跳过 $svc_name (无服务/服务未运行)"
        return 0
    fi

    # 记录该服务原本是运行中的
    RUNNING_SERVICES["$svc_name"]=1

    if [ "$DRY_RUN" = true ]; then
        if [ -x "$svc_path/compose-down.sh" ]; then
            log "[DRY-RUN] 将停止 $svc_name (使用 compose-down.sh)"
        elif [ -f "$svc_path/docker-compose.yml" ]; then
            log "[DRY-RUN] 将停止 $svc_name (使用 docker compose down)"
        fi
        return 0
    fi

    if [ -x "$svc_path/compose-down.sh" ]; then
        log "Stopping $svc_name (使用 compose-down.sh)..."
        (cd "$svc_path" && ./compose-down.sh) || log "警告：停止 $svc_name 失败"
    elif [ -f "$svc_path/docker-compose.yml" ]; then
        log "Stopping $svc_name ..."
        (cd "$svc_path" && docker compose down) || log "警告：停止 $svc_name 失败"
    fi
}

# 启动单个服务并返回状态的函数
start_service_with_status() {
    local svc_path="$1"
    local svc_name
    svc_name=$(basename "$svc_path")

    # 检查该服务是否原本在运行
    if [ -z "${RUNNING_SERVICES[$svc_name]}" ]; then
        log "跳过启动 $svc_name (原本未运行)"
        return 0
    fi

    if [ -x "$svc_path/compose-up.sh" ]; then
        log "Starting $svc_name (使用 compose-up.sh)..."
        if ! (cd "$svc_path" && ./compose-up.sh); then
            log "警告：启动 $svc_name 失败"
            return 1
        fi
    elif [ -f "$svc_path/docker-compose.yml" ]; then
        log "Starting $svc_name ..."
        if ! (cd "$svc_path" && docker compose up -d); then
            log "警告：启动 $svc_name 失败"
            return 1
        fi
    fi
    return 0
}

# 启动所有服务的函数
start_all_services() {
    local failed_services=()

    log "恢复网关服务 (Priority)..."
    for svc in "${PRIORITY_SERVICES[@]}"; do
        if [ -d "$BASE_DIR/$svc" ]; then
            if ! start_service_with_status "$BASE_DIR/$svc"; then
                failed_services+=("$svc")
            fi
        fi
    done

    log "恢复普通服务..."
    for svc in "${NORMAL_SERVICES[@]}"; do
        if [ -d "$BASE_DIR/$svc" ]; then
            if ! start_service_with_status "$BASE_DIR/$svc"; then
                failed_services+=("$svc")
            fi
        fi
    done

    # 如果有服务启动失败，发送通知
    if [ ${#failed_services[@]} -gt 0 ]; then
        log "!!! 以下服务启动失败: ${failed_services[*]}"
        send_notification "⚠️ 服务恢复异常" "以下服务启动失败: ${failed_services[*]}"
    fi
}

# 清理函数：确保异常退出时也能恢复服务
cleanup() {
    local exit_code=$?
    if [ "$exit_code" -ne 0 ]; then
        log "!!! 脚本异常退出，尝试恢复所有服务..."
        send_notification "❌ 备份异常" "脚本异常退出 (exit code: $exit_code)，正在尝试恢复服务..."
        start_all_services
    fi
    rm -rf "$LOCK_FILE"
    # 清理临时日志文件
    if [ -f "$LOG_OUTPUT_FILE" ]; then
        rm -f "$LOG_OUTPUT_FILE"
    fi
}

# ================= 主流程 =================
# 打印配置
print_config

# 执行依赖检查
check_dependencies

# ================= 交互式确认 =================
if [ "$DRY_RUN" = false ] && [ "$AUTO_CONFIRM" = false ]; then
    echo ""
    echo "=========================================="
    echo "⚠️  警告：即将执行备份操作"
    echo "=========================================="
    echo ""
    echo "此操作将会："
    echo "  1. 停止所有 Docker 服务"
    echo "  2. 创建 Kopia 快照备份"
    echo "  3. 重新启动所有服务"
    echo ""
    echo "💡 提示：建议先使用 --dry-run 参数测试："
    echo "   $0 --dry-run"
    echo ""
    read -r -p "确认执行备份？[y/N] " response
    case "$response" in
        [yY][eE][sS]|[yY])
            echo "开始执行备份..."
            ;;
        *)
            echo "已取消操作"
            exit 0
            ;;
    esac
fi

# 检查锁文件，防止重复执行（使用 mkdir 原子操作）
if ! mkdir "$LOCK_FILE" 2>/dev/null; then
    log "!!! 另一个备份进程正在运行 (锁文件: $LOCK_FILE)，退出"
    exit 1
fi

# 注册 trap，捕获退出信号
trap cleanup EXIT INT TERM

# 1. 获取所有子目录列表
NORMAL_SERVICES=()

# 2. 区分普通服务和网关服务
while IFS= read -r -d '' dir; do
    dirname=$(basename "$dir")
    is_priority=false

    # 检查是否在优先列表中
    for p in "${PRIORITY_SERVICES[@]}"; do
        if [[ "$dirname" == "$p" ]]; then
            is_priority=true
            break
        fi
    done

    if [ "$is_priority" = "false" ]; then
        NORMAL_SERVICES+=("$dirname")
    fi
done < <(find "$BASE_DIR" -mindepth 1 -maxdepth 1 -type d -print0)

log ">>> 开始执行深夜维护..."
send_notification "🔄 备份开始" "开始执行服务器备份任务"

# 3. 停止容器
# 3.1 先停止普通服务
log "停止普通服务..."
for svc in "${NORMAL_SERVICES[@]}"; do
    if [ -d "$BASE_DIR/$svc" ]; then
        stop_service "$BASE_DIR/$svc"
    fi
done

# 3.2 最后停止网关服务
log "停止网关服务 (Priority)..."
for svc in "${PRIORITY_SERVICES[@]}"; do
    if [ -d "$BASE_DIR/$svc" ]; then
        stop_service "$BASE_DIR/$svc"
    fi
done

# 4. 执行 Kopia 备份
log ">>> 服务已全部停止，准备执行 Kopia 快照..."

# 4.1 执行快照
backup_success=true
if [ "$DRY_RUN" = true ]; then
    log "[DRY-RUN] 将执行: kopia snapshot create $BASE_DIR"
else
    log "开始创建快照..."
    if ! kopia snapshot create "$BASE_DIR"; then
        log "!!! 警告：备份过程中出现错误 !!!"
        backup_success=false
        if [ "$IGNORE_BACKUP_ERROR" = false ]; then
            log "备份失败且 IGNORE_BACKUP_ERROR=false，恢复服务后退出..."
            send_notification "❌ 备份失败" "Kopia 快照创建失败，服务已恢复"
            start_all_services
            exit 1
        else
            log "IGNORE_BACKUP_ERROR=true，继续恢复服务..."
        fi
    else
        log ">>> 备份成功！"
    fi
fi

# 5. 启动容器
if [ "$DRY_RUN" = true ]; then
    log "[DRY-RUN] 将依序恢复以下服务:"
    for svc in "${PRIORITY_SERVICES[@]}"; do
        if [ -n "${RUNNING_SERVICES[$svc]}" ]; then
            log "[DRY-RUN]   - $svc (网关服务)"
        fi
    done
    for svc in "${NORMAL_SERVICES[@]}"; do
        if [ -n "${RUNNING_SERVICES[$svc]}" ]; then
            log "[DRY-RUN]   - $svc"
        fi
    done
else
    start_all_services
fi

log ">>> 所有任务完成。"

# ================= 显示耗时统计 =================
SCRIPT_END_TIME=$(date +%s)
SCRIPT_END_DATETIME=$(date '+%Y-%m-%d %H:%M:%S')
TOTAL_SECONDS=$((SCRIPT_END_TIME - SCRIPT_START_TIME))

# 转换为时分秒格式
HOURS=$((TOTAL_SECONDS / 3600))
MINUTES=$(((TOTAL_SECONDS % 3600) / 60))
SECONDS=$((TOTAL_SECONDS % 60))

echo ""
echo "=========================================="
echo "耗时统计:"
echo "=========================================="
printf "  %-20s %s\n" "开始时间:" "$SCRIPT_START_DATETIME"
printf "  %-20s %s\n" "结束时间:" "$SCRIPT_END_DATETIME"
if [ $HOURS -gt 0 ]; then
    printf "  %-20s %d 小时 %d 分 %d 秒\n" "总耗时:" "$HOURS" "$MINUTES" "$SECONDS"
elif [ $MINUTES -gt 0 ]; then
    printf "  %-20s %d 分 %d 秒\n" "总耗时:" "$MINUTES" "$SECONDS"
else
    printf "  %-20s %d 秒\n" "总耗时:" "$SECONDS"
fi
echo "=========================================="

# 发送最终通知
if [ "$DRY_RUN" = true ]; then
    log "[DRY-RUN] 模拟运行完成，未执行任何实际操作"
    send_notification "🧪 DRY-RUN 完成" "模拟运行完成，未执行任何实际操作"
elif [ "$backup_success" = true ]; then
    send_notification "✅ 备份成功" "所有服务已恢复运行"
else
    send_notification "⚠️ 备份完成（有警告）" "快照创建失败，但服务已恢复运行"
fi

# 上传日志到 Gist
upload_to_gist
