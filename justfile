# YewResin 构建脚本
# 通过 Docker Compose 使用 GoReleaser 构建

# 默认目标：构建当前平台（不需要 goreleaser）
default: build

# 快速构建当前平台（不用 Docker）
build:
    @echo "Building yewresin for current platform..."
    @CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=dev" -o yewresin ./cmd/yewresin
    @echo "Done: yewresin"

# 使用 GoReleaser 构建所有平台的可执行文件（不发布）
release-snapshot:
    docker compose -f compose.goreleaser.yaml run --rm goreleaser build --snapshot --clean

# 使用 GoReleaser 模拟完整发布流程（不推送）
release-dry:
    docker compose -f compose.goreleaser.yaml run --rm goreleaser release --snapshot --clean

# 使用 GoReleaser 正式发布（需要 git tag + .env.goreleaser 配置）
release:
    docker compose -f compose.goreleaser.yaml run --rm goreleaser release --clean

# 运行测试
test:
    @echo "Running tests..."
    go test -v ./...

# 清理构建产物
clean:
    @echo "Cleaning..."
    @rm -rf dist
    @rm -f yewresin yewresin.exe
    @echo "Done"

# 显示帮助信息
help:
    @echo "YewResin 构建脚本"
    @echo ""
    @echo "用法:"
    @echo "  just                  - 快速构建当前平台"
    @echo "  just release-snapshot - 构建全平台可执行文件（不发布）"
    @echo "  just release-dry      - 模拟完整发布流程（不推送）"
    @echo "  just release          - 正式发布（需要 tag + 配置）"
    @echo "  just test             - 运行测试"
    @echo "  just clean            - 清理构建产物"
