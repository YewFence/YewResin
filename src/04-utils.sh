# ================= 工具函数 =================
# dry_run_exec simulates or executes a command depending on the DRY_RUN flag: if DRY_RUN is true it echoes "[DRY-RUN] 将执行: <command...>" and returns 0, otherwise it invokes the command with the provided arguments.
dry_run_exec() {
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY-RUN] 将执行: $*"
        return 0
    else
        "$@"
    fi
}