# shellcheck shell=bash
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
    echo "欢迎使用 YewResin Docker 备份脚本 By YewFence"
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --dry-run, -n    模拟运行，只检查依赖和显示要执行的操作，不实际执行"
    echo "  -y, --yes        跳过交互式确认，自动确认执行"
    echo "  --help, -h       显示此帮助信息"
    echo ""
    echo "必要环境变量:"
    echo "  BASE_DIR              Docker Compose 项目目录"
    echo "  EXPECTED_REMOTE       Kopia 远程路径"
    echo "更多说明请参考项目 README 文档 https://github.com/YewFence/YewResin/"
    exit 0
fi
