
# ================= 日志捕获 =================
# 日志文件路径，默认为脚本同目录下的 yewresin.log
# 可通过 LOG_FILE 环境变量自定义
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="${LOG_FILE:-$SCRIPT_DIR/yewresin.log}"

# 每次运行清空日志文件（避免日志无限增长）
: > "$LOG_FILE"

# 使用 tee 同时输出到终端和日志文件
exec > >(tee -a "$LOG_FILE")
exec 2>&1

log() {
    echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] $1"
}
