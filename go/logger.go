package main

import (
	"io"
	"log/slog"
	"os"
)

// LogWriter 保存日志写入器的引用，用于获取日志内容
var LogWriter *LogCapture

// LogCapture 捕获日志内容，同时写入多个目标
type LogCapture struct {
	buffer  []byte
	writers []io.Writer
}

// NewLogCapture 创建日志捕获器
func NewLogCapture(writers ...io.Writer) *LogCapture {
	return &LogCapture{
		writers: writers,
	}
}

// Write 实现 io.Writer 接口
func (lc *LogCapture) Write(p []byte) (n int, err error) {
	// 保存到缓冲区
	lc.buffer = append(lc.buffer, p...)

	// 写入所有目标
	for _, w := range lc.writers {
		w.Write(p)
	}
	return len(p), nil
}

// GetContent 获取捕获的日志内容
func (lc *LogCapture) GetContent() string {
	return string(lc.buffer)
}

// InitLogger 初始化日志系统
// 返回日志文件句柄（如果有）和 LogCapture 用于获取日志内容
func InitLogger(logFile string) (*os.File, error) {
	writers := []io.Writer{os.Stdout}

	var file *os.File
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
		file = f
		writers = append(writers, f)
	}

	// 创建日志捕获器
	LogWriter = NewLogCapture(writers...)

	// 配置 slog
	logger := slog.New(slog.NewTextHandler(LogWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	return file, nil
}
