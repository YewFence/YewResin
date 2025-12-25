#!/bin/bash
# shellcheck source-path=SCRIPTDIR
# This module is sourced by yewresin.sh and provides service management functions.
# Required external variables: DRY_RUN, BASE_DIR, LOCK_FILE, PRIORITY_SERVICES, NORMAL_SERVICES
# Required external functions: log(), send_notification()
# ================= 服务管理 =================
# 记录原本运行中的服务
declare -A RUNNING_SERVICES

# 检查目录下是否存在 compose 配置文件
has_compose_file() {
    local svc_path="$1"
    [ -f "$svc_path/compose.yaml" ] || \
    [ -f "$svc_path/compose.yml" ] || \
    [ -f "$svc_path/docker-compose.yaml" ] || \
    [ -f "$svc_path/docker-compose.yml" ]
}

# 检查服务是否正在运行
is_service_running() {
    local svc_path="$1"
    local svc_name
    svc_name=$(basename "$svc_path")

    # 检查是否有 compose 相关文件（yaml 或脚本）
    local has_compose=false
    if [ -x "$svc_path/compose-status.sh" ] || [ -x "$svc_path/compose-up.sh" ] || [ -x "$svc_path/compose-log.sh" ]; then
        has_compose=true
    elif has_compose_file "$svc_path"; then
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

    # 确定停止方法
    local stop_cmd=""
    local stop_msg=""
    if [ -x "$svc_path/compose-stop.sh" ]; then
        stop_cmd="./compose-stop.sh"
        stop_msg="使用 compose-stop.sh"
    elif [ -x "$svc_path/compose-down.sh" ]; then
        stop_cmd="./compose-down.sh"
        stop_msg="使用 compose-down.sh"
    elif has_compose_file "$svc_path"; then
        stop_cmd="docker compose stop"
        stop_msg="使用 docker compose stop"
    fi

    # 无法识别停止方法
    if [ -z "$stop_cmd" ]; then
        if [ "$DRY_RUN" = true ]; then
            log "[DRY-RUN] 警告：停止 $svc_name 失败，无法识别停止方法"
            return 0
        else
            log "错误：停止 $svc_name 失败，无法识别停止方法"
            return 1
        fi
    fi

    # DRY_RUN 模式只打印
    if [ "$DRY_RUN" = true ]; then
        log "[DRY-RUN] 将停止 $svc_name ($stop_msg)"
        return 0
    fi

    # 实际执行停止
    log "停止 $svc_name ($stop_msg)..."
    if ! (cd "$svc_path" && $stop_cmd); then
        log "错误：停止 $svc_name 失败"
        return 1
    fi
    return 0
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

    # 确定启动方法
    local start_cmd=""
    local start_msg=""
    if [ -x "$svc_path/compose-up.sh" ]; then
        start_cmd="./compose-up.sh"
        start_msg="使用 compose-up.sh"
    elif has_compose_file "$svc_path"; then
        start_cmd="docker compose up -d"
        start_msg="使用 docker compose up -d"
    fi

    # 无法识别启动方法
    if [ -z "$start_cmd" ]; then
        if [ "$DRY_RUN" = true ]; then
            log "[DRY-RUN] 警告：启动 $svc_name 失败，无法识别启动方法"
        else
            log "警告：启动 $svc_name 失败，无法识别启动方法"
        fi
        return 1
    fi

    # DRY_RUN 模式只打印
    if [ "$DRY_RUN" = true ]; then
        log "[DRY-RUN] 将启动 $svc_name ($start_msg)"
        return 0
    fi

    # 实际执行启动
    log "启动 $svc_name ($start_msg)..."
    if ! (cd "$svc_path" && $start_cmd); then
        log "警告：启动 $svc_name 失败"
        return 1
    fi
    return 0
}

# 辅助函数：启动服务，如果失败则记录到数组
# 使用 nameref 引用外部数组
_start_service_or_record() {
    local svc_path="$1"
    local -n _failed_arr=$2
    local svc_name
    svc_name=$(basename "$svc_path")

    if ! start_service_with_status "$svc_path"; then
        _failed_arr+=("$svc_name")
    fi
}

# 启动所有服务的函数
start_all_services() {
    local failed_services=()

    log "恢复网关服务 (优先执行)..."
    for svc in "${PRIORITY_SERVICES[@]}"; do
        [ -d "$BASE_DIR/$svc" ] && _start_service_or_record "$BASE_DIR/$svc" failed_services
    done

    log "恢复普通服务..."
    for svc in "${NORMAL_SERVICES[@]}"; do
        [ -d "$BASE_DIR/$svc" ] && _start_service_or_record "$BASE_DIR/$svc" failed_services
    done

    # 如果有服务启动失败，发送通知
    if [ ${#failed_services[@]} -gt 0 ]; then
        log "!!! 以下服务启动失败: ${failed_services[*]}"
        send_notification "⚠️ 服务恢复异常" "以下服务启动失败: ${failed_services[*]}"
    fi
}

# 辅助函数：停止服务，如果失败则退出并发送通知
_stop_service_or_exit() {
    local svc_path="$1"
    local svc_name
    svc_name=$(basename "$svc_path")

    if ! stop_service "$svc_path"; then
        log "!!! 服务停止失败，中止备份以保护数据安全"
        send_notification "❌ 备份中止" "服务 $svc_name 停止失败，已中止备份以避免数据损坏"
        exit 1
    fi
}

stop_all_services() {
    log "停止普通服务..."
    for svc in "${NORMAL_SERVICES[@]}"; do
        [ -d "$BASE_DIR/$svc" ] && _stop_service_or_exit "$BASE_DIR/$svc"
    done

    log "停止网关服务 (最后执行)..."
    for svc in "${PRIORITY_SERVICES[@]}"; do
        [ -d "$BASE_DIR/$svc" ] && _stop_service_or_exit "$BASE_DIR/$svc"
    done
}

# 清理函数：确保异常退出时也能恢复服务
cleanup() {
    local exit_code=$?
    if [ "$exit_code" -ne 0 ]; then
        log "!!! 脚本异常退出，尝试恢复所有服务..."
        send_notification "❌ 备份异常" "脚本异常退出 (exit code: $exit_code)，正在尝试恢复服务..."
        start_all_services
    fi
    # 上传日志到 Gist
    upload_to_gist
    # 移除锁目录
    if [ -d "$LOCK_FILE" ]; then
        rmdir "$LOCK_FILE" || log "警告：无法移除锁目录，可能包含意外文件"
    fi
    # 清理临时日志文件
    if [ -f "$LOG_OUTPUT_FILE" ]; then
        rm -f "$LOG_OUTPUT_FILE"
    fi
}
