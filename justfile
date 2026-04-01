# YewResin 构建脚本
# 支持多平台交叉编译

binary := "yewresin"
dist_dir := "dist"

# 版本号：优先使用环境变量，否则 git tag，最后时间戳
version := env_var_or_default("VERSION", `git describe --tags --always 2>/dev/null || date -u +"%Y%m%d.%H%M%S"`)

ldflags := "-s -w -X main.version=" + version
go_build := "CGO_ENABLED=0 go build -ldflags \"" + ldflags + "\""

# 默认目标：构建当前平台
default: build

# 构建当前平台
build:
    @echo "Building {{binary}} {{version}} for current platform..."
    {{go_build}} -o {{binary}} .
    @echo "Done: {{binary}}"

# 构建所有平台
all: clean
    @echo "Building {{binary}} {{version}} for all platforms..."
    @mkdir -p {{dist_dir}}
    @just linux
    @just darwin
    @just windows
    @echo ""
    @echo "All builds completed:"
    @ls -lh {{dist_dir}}/

# Linux 平台 (amd64, arm64)
linux: (build_target "linux" "amd64") (build_target "linux" "arm64")

# macOS 平台 (amd64, arm64)
darwin: (build_target "darwin" "amd64") (build_target "darwin" "arm64")

# Windows 平台 (amd64)
windows: (build_target "windows" "amd64" true)

# 构建指定平台目标
build_target os arch windows_ext=false:
    @mkdir -p {{dist_dir}}
    #! /usr/bin/env bash
    ext=""
    if [ "{{windows_ext}}" = "true" ]; then
        ext=".exe"
    fi
    echo "  Building {{os}}/{{arch}}..."
    GOOS={{os}} GOARCH={{arch}} {{go_build}} -o {{dist_dir}}/{{binary}}-{{os}}-{{arch}}${ext} .

# 清理构建产物
clean:
    @echo "Cleaning..."
    @rm -rf {{dist_dir}}
    @rm -f {{binary}} {{binary}}.exe
    @echo "Done"

# 运行测试
test:
    @echo "Running tests..."
    go test -v ./...

# 显示帮助信息
help:
    @echo "YewResin 构建脚本"
    @echo ""
    @echo "用法:"
    @echo "  just          - 构建当前平台"
    @echo "  just all      - 构建所有平台"
    @echo "  just linux    - 构建 Linux (amd64, arm64)"
    @echo "  just darwin   - 构建 macOS (amd64, arm64)"
    @echo "  just windows  - 构建 Windows (amd64)"
    @echo "  just test     - 运行测试"
    @echo "  just clean    - 清理构建产物"
    @echo "  just help     - 显示帮助信息"
    @echo ""
    @echo "环境变量:"
    @echo "  VERSION       - 指定版本号 (默认: git tag 或时间戳)"
    @echo ""
    @echo "示例:"
    @echo "  just all                      # 构建所有平台"
    @echo "  VERSION=v2.0.0 just all       # 指定版本构建"
