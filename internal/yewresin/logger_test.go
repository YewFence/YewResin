package yewresin

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitLoggerNoFile(t *testing.T) {
	file, err := InitLogger("")
	if err != nil {
		t.Fatalf("InitLogger error: %v", err)
	}
	if file != nil {
		t.Fatalf("expected nil file, got %v", file)
	}

	slog.Info("hello logger")
	if LogWriter == nil {
		t.Fatalf("LogWriter should not be nil")
	}
	if !strings.Contains(LogWriter.GetContent(), "hello logger") {
		t.Fatalf("log content missing message: %q", LogWriter.GetContent())
	}
}

func TestInitLoggerWithFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "app.log")
	file, err := InitLogger(logPath)
	if err != nil {
		t.Fatalf("InitLogger error: %v", err)
	}
	if file == nil {
		t.Fatalf("expected log file handle")
	}

	slog.Info("hello file")
	file.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file error: %v", err)
	}
	if !strings.Contains(string(data), "hello file") {
		t.Fatalf("log file missing message: %q", string(data))
	}
}
