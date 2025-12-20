# YewResin Backup Script Builder
# 将 src/ 目录下的模块文件合并为 yewresin.sh

SOURCES := $(sort $(wildcard src/*.sh))
TARGET := yewresin.sh

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
	@echo "  make build  - Build yewresin.sh from src/ modules"
	@echo "  make clean  - Remove generated yewresin.sh"
	@echo "  make help   - Show this help message"
	@echo ""
	@echo "Modules:"
	@for f in $(SOURCES); do echo "  $$f"; done
