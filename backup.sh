#!/bin/bash

set -eo pipefail

# 加载环境变量配置文件（可选）
# 支持通过 CONFIG_FILE 环境变量指定配置文件路径
CONFIG_FILE="${CONFIG_FILE:-$(dirname "$0")/.env}"
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
# ==========================================

# ================= 依赖检查 =================
# 依赖检查专用的通知函数（在主 send_notification 定义之前使用）
send_dep_notification() {
    local title="$1"
    local body="$2"

    if [ -z "$APPRISE_URL" ] || [ -z "$APPRISE_NOTIFY_URL" ]; then
        return 0
    fi

    curl -X POST "$APPRISE_URL" \
        -H "Content-Type: application/json" \
        -d "{
            \"urls\": \"$APPRISE_NOTIFY_URL\",
            \"body\": \"$body\",
            \"title\": \"$title\"
        }" \
        --max-time 10 \
        --silent \
        --show-error || echo "警告：通知发送失败"
}

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
        send_dep_notification "❌ 备份失败" "依赖检查未通过: ${error_msg}请手动配置后重试"
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
        if ! kopia repository connect rclone --remote-path="$EXPECTED_REMOTE"; then
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

log() {
    echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] $1"
}

# 发送通知函数（需要配置 APPRISE_URL 和 APPRISE_NOTIFY_URL）
send_notification() {
    local title="$1"
    local body="$2"

    # 如果没配置 Apprise，跳过通知
    if [ -z "$APPRISE_URL" ] || [ -z "$APPRISE_NOTIFY_URL" ]; then
        return 0
    fi

    curl -X POST "$APPRISE_URL" \
        -H "Content-Type: application/json" \
        -d "{
            \"urls\": \"$APPRISE_NOTIFY_URL\",
            \"body\": \"$body\",
            \"title\": \"$title\"
        }" \
        --max-time 10 \
        --silent \
        --show-error || echo "警告：通知发送失败"
}

# 执行依赖检查
check_dependencies

# 记录原本运行中的服务
declare -A RUNNING_SERVICES

# 检查服务是否正在运行
is_service_running() {
    local svc_path="$1"
    local svc_name
    svc_name=$(basename "$svc_path")

    # 方法1: 如果有 compose-status.sh 脚本，优先使用它
    if [ -x "$svc_path/compose-status.sh" ]; then
        if (cd "$svc_path" && ./compose-status.sh) >/dev/null 2>&1; then
            return 0
        fi
        return 1
    fi

    # 方法1.5: 如果有 compose-log.sh，也尝试用它检查（能查看日志说明服务在运行）
    if [ -x "$svc_path/compose-log.sh" ]; then
        if (cd "$svc_path" && timeout 2 ./compose-log.sh --tail=1) >/dev/null 2>&1; then
            return 0
        fi
    fi

    # 方法2: 查找目录下所有 compose 相关的 yaml 文件
    local yaml_files=()
    while IFS= read -r -d '' file; do
        yaml_files+=("$file")
    done < <(find "$svc_path" -maxdepth 1 \( -name "docker-compose*.yml" -o -name "docker-compose*.yaml" -o -name "compose*.yml" -o -name "compose*.yaml" \) -print0 2>/dev/null)

    # 如果找到了 yaml 文件，尝试用第一个检查状态
    if [ ${#yaml_files[@]} -gt 0 ]; then
        local running_containers
        running_containers=$(cd "$svc_path" && docker compose ps -q 2>/dev/null | wc -l)
        if [ "$running_containers" -gt 0 ]; then
            return 0
        fi
    fi

    # 方法3: 如果有 compose-up.sh 但没有 yaml 文件，用项目名检查
    if [ -x "$svc_path/compose-up.sh" ]; then
        local running_containers
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
        log "跳过 $svc_name (服务未运行)"
        return 0
    fi

    # 记录该服务原本是运行中的
    RUNNING_SERVICES["$svc_name"]=1

    if [ -x "$svc_path/compose-down.sh" ]; then
        log "Stopping $svc_name (使用 compose-down.sh)..."
        (cd "$svc_path" && ./compose-down.sh) || log "警告：停止 $svc_name 失败"
    elif [ -f "$svc_path/docker-compose.yml" ]; then
        log "Stopping $svc_name ..."
        (cd "$svc_path" && docker compose down) || log "警告：停止 $svc_name 失败"
    fi
}

# 启动单个服务的函数
start_service() {
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
        (cd "$svc_path" && ./compose-up.sh) || log "警告：启动 $svc_name 失败"
    elif [ -f "$svc_path/docker-compose.yml" ]; then
        log "Starting $svc_name ..."
        (cd "$svc_path" && docker compose up -d) || log "警告：启动 $svc_name 失败"
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

    log "正在恢复网关服务 (Priority)..."
    for svc in "${PRIORITY_SERVICES[@]}"; do
        if [ -d "$BASE_DIR/$svc" ]; then
            if ! start_service_with_status "$BASE_DIR/$svc"; then
                failed_services+=("$svc")
            fi
        fi
    done

    log "正在恢复普通服务..."
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
}

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
log "正在停止普通服务..."
for svc in "${NORMAL_SERVICES[@]}"; do
    if [ -d "$BASE_DIR/$svc" ]; then
        stop_service "$BASE_DIR/$svc"
    fi
done

# 3.2 最后停止网关服务
log "正在停止网关服务 (Priority)..."
for svc in "${PRIORITY_SERVICES[@]}"; do
    if [ -d "$BASE_DIR/$svc" ]; then
        stop_service "$BASE_DIR/$svc"
    fi
done

# 4. 执行 Kopia 备份
log ">>> 服务已全部停止，准备执行 Kopia 快照..."

# 4.1 执行快照
log "开始创建快照..."
backup_success=true
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

# 5. 启动容器
start_all_services

# 6. (可选) 清理旧快照
log ">>> 执行策略清理..."
kopia maintenance run --auto || log "警告：策略清理失败"

log ">>> 所有任务完成。"

# 发送最终通知
if [ "$backup_success" = true ]; then
    send_notification "✅ 备份成功" "所有服务已恢复运行"
else
    send_notification "⚠️ 备份完成（有警告）" "快照创建失败，但服务已恢复运行"
fi
