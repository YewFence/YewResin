
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
stop_all_services

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
        send_notification "❌ 备份失败" "Kopia 快照创建失败"
    else
        log ">>> 备份成功！"
    fi
fi

# 5. 启动容器
start_all_services

log ">>> 所有任务完成。"

# ================= 显示耗时统计 =================
SCRIPT_END_TIME=$(date -u +%s)
SCRIPT_END_DATETIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
TOTAL_SECS=$((SCRIPT_END_TIME - SCRIPT_START_TIME))

# 转换为时分秒格式
HOURS=$((TOTAL_SECS / 3600))
MINUTES=$(((TOTAL_SECS % 3600) / 60))
SECS=$((TOTAL_SECS % 60))
log "$(printf "  %-20s %s" "开始时间:" "$SCRIPT_START_DATETIME")"
log "$(printf "  %-20s %s" "结束时间:" "$SCRIPT_END_DATETIME")"
if [ $HOURS -gt 0 ]; then
    log "$(printf "  %-20s %d 小时 %d 分 %d 秒" "总耗时:" "$HOURS" "$MINUTES" "$SECS")"
elif [ $MINUTES -gt 0 ]; then
    log "$(printf "  %-20s %d 分 %d 秒" "总耗时:" "$MINUTES" "$SECS")"
else
    log "$(printf "  %-20s %d 秒" "总耗时:" "$SECS")"
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
