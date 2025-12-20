# ================= 日志捕获 =================
# 创建临时文件保存日志输出
LOG_OUTPUT_FILE=$(mktemp)
# 使用 tee 同时输出到终端和文件
exec > >(tee -a "$LOG_OUTPUT_FILE")
exec 2>&1

# log prints MESSAGE prefixed with the current UTC timestamp in the format [YYYY-MM-DD HH:MM:SS UTC].
log() {
    echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] $1"
}