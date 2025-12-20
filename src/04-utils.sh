
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
