
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
        if [ ${#APPRISE_URL} -gt 20 ]; then
            local masked_url="${APPRISE_URL:0:10}...${APPRISE_URL: -5}"
        else
            local masked_url="****(已配置)"
        fi
        printf "$fmt" "APPRISE_URL(通知服务URL):" "$masked_url"
    else
        printf "$fmt" "APPRISE_URL(通知服务URL):" "(未配置)"
    fi

    if [ -n "$APPRISE_NOTIFY_URL" ]; then
        if [ ${#APPRISE_NOTIFY_URL} -gt 20 ]; then
            local masked_notify="${APPRISE_NOTIFY_URL:0:10}...${APPRISE_NOTIFY_URL: -5}"
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
