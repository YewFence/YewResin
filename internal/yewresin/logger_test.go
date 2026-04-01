package yewresin

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogCapture(t *testing.T) {
	var buf bytes.Buffer
	lc := NewLogCapture(&buf)

	if lc == nil {
		t.Fatalf("NewLogCapture should not return nil")
	}
	if len(lc.writers) != 1 {
		t.Fatalf("expected 1 writer, got %d", len(lc.writers))
	}
}

func TestLogCaptureWriteGetContent(t *testing.T) {
	lc := NewLogCapture()

	n, err := lc.Write([]byte("hello "))
	if err != nil || n != 6 {
		t.Fatalf("Write: n=%d err=%v", n, err)
	}
	lc.Write([]byte("world"))

	content := lc.GetContent()
	if content != "hello world" {
		t.Fatalf("GetContent: got %q, want %q", content, "hello world")
	}
}

func TestLogCaptureMultiWriter(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	lc := NewLogCapture(&buf1, &buf2)

	lc.Write([]byte("multi"))

	if buf1.String() != "multi" || buf2.String() != "multi" {
		t.Fatalf("writers should receive written data: buf1=%q buf2=%q", buf1.String(), buf2.String())
	}
}

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
