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
    # 上传日志到 Gist
    upload_to_gist
    # 移除锁文件
    rm -rf "$LOCK_FILE"
    # 清理临时日志文件
    if [ -f "$LOG_OUTPUT_FILE" ]; then
        rm -f "$LOG_OUTPUT_FILE"
    fi
}
