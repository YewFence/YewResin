
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
