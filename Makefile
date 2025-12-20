# YewResin Backup Script Builder
# 将 src/ 目录下的模块文件合并为 backup.sh

SOURCES := $(sort $(wildcard src/*.sh))
TARGET := backup.sh

.PHONY: build clean help

# 默认目标
build: $(TARGET)

$(TARGET): $(SOURCES)
	@echo "Building $(TARGET) from modules..."
	@cat $(SOURCES) > $(TARGET)
	@chmod +x $(TARGET)
	@echo "Done: $(TARGET) ($(shell wc -l < $(TARGET)) lines)"

clean:
	@echo "Removing $(TARGET)..."
	@rm -f $(TARGET)
	@echo "Done"

help:
	@echo "Usage:"
	@echo "  make build  - Build backup.sh from src/ modules"
	@echo "  make clean  - Remove generated backup.sh"
	@echo "  make help   - Show this help message"
	@echo ""
	@echo "Modules:"
	@for f in $(SOURCES); do echo "  $$f"; done
