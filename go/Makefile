# YewResin Go 版本构建脚本
# 支持多平台交叉编译

BINARY := yewresin
DIST_DIR := dist

# 版本号：优先使用 VERSION 环境变量，否则尝试 git tag，最后使用时间戳
VERSION ?= $(shell git describe --tags --always 2>/dev/null || date -u +"%Y%m%d.%H%M%S")

# 构建标志
LDFLAGS := -s -w -X main.version=$(VERSION)
GO_BUILD := go build -ldflags "$(LDFLAGS)"

# 目标平台
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build all linux darwin windows clean test help

# 默认目标：构建当前平台
build:
	@echo "Building $(BINARY) $(VERSION) for current platform..."
	$(GO_BUILD) -o $(BINARY) .
	@echo "Done: $(BINARY)"

# 构建所有平台
all: clean
	@echo "Building $(BINARY) $(VERSION) for all platforms..."
	@mkdir -p $(DIST_DIR)
	@$(MAKE) --no-print-directory linux
	@$(MAKE) --no-print-directory darwin
	@$(MAKE) --no-print-directory windows
	@echo ""
	@echo "All builds completed:"
	@ls -lh $(DIST_DIR)/

# Linux 平台
linux: $(DIST_DIR)/$(BINARY)-linux-amd64 $(DIST_DIR)/$(BINARY)-linux-arm64

$(DIST_DIR)/$(BINARY)-linux-amd64:
	@mkdir -p $(DIST_DIR)
	@echo "  Building linux/amd64..."
	@GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $@ .

$(DIST_DIR)/$(BINARY)-linux-arm64:
	@mkdir -p $(DIST_DIR)
	@echo "  Building linux/arm64..."
	@GOOS=linux GOARCH=arm64 $(GO_BUILD) -o $@ .

# macOS 平台
darwin: $(DIST_DIR)/$(BINARY)-darwin-amd64 $(DIST_DIR)/$(BINARY)-darwin-arm64

$(DIST_DIR)/$(BINARY)-darwin-amd64:
	@mkdir -p $(DIST_DIR)
	@echo "  Building darwin/amd64..."
	@GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $@ .

$(DIST_DIR)/$(BINARY)-darwin-arm64:
	@mkdir -p $(DIST_DIR)
	@echo "  Building darwin/arm64..."
	@GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o $@ .

# Windows 平台
windows: $(DIST_DIR)/$(BINARY)-windows-amd64.exe

$(DIST_DIR)/$(BINARY)-windows-amd64.exe:
	@mkdir -p $(DIST_DIR)
	@echo "  Building windows/amd64..."
	@GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $@ .

# 清理构建产物
clean:
	@echo "Cleaning..."
	@rm -rf $(DIST_DIR)
	@rm -f $(BINARY) $(BINARY).exe
	@echo "Done"

# 运行测试
test:
	@echo "Running tests..."
	@go test -v ./...

help:
	@echo "YewResin Go 版本构建脚本"
	@echo ""
	@echo "用法:"
	@echo "  make          - 构建当前平台"
	@echo "  make all      - 构建所有平台"
	@echo "  make linux    - 构建 Linux (amd64, arm64)"
	@echo "  make darwin   - 构建 macOS (amd64, arm64)"
	@echo "  make windows  - 构建 Windows (amd64)"
	@echo "  make test     - 运行测试"
	@echo "  make clean    - 清理构建产物"
	@echo "  make help     - 显示帮助信息"
	@echo ""
	@echo "环境变量:"
	@echo "  VERSION       - 指定版本号 (默认: git tag 或时间戳)"
	@echo ""
	@echo "示例:"
	@echo "  make all                    # 构建所有平台"
	@echo "  VERSION=v2.0.0 make all     # 指定版本构建"
