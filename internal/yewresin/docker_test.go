package yewresin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestHasComposeFile(t *testing.T) {
	baseDir := t.TempDir()
	dm := NewDockerManager(baseDir, true, time.Second)

	// 空目录时不应识别到 compose 配置文件
	if dm.hasComposeFile(baseDir) {
		t.Fatalf("expected no compose file")
	}

	path := filepath.Join(baseDir, "compose.yaml")
	if err := os.WriteFile(path, []byte("services:\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	// 写入标准 compose 文件后应能被识别
	if !dm.hasComposeFile(baseDir) {
		t.Fatalf("expected compose file detected")
	}
}

func TestHasComposeScript(t *testing.T) {
	baseDir := t.TempDir()
	dm := NewDockerManager(baseDir, true, time.Second)

	// 空目录时不应识别到 compose 脚本
	if dm.hasComposeScript(baseDir) {
		t.Fatalf("expected no compose script")
	}

	path := filepath.Join(baseDir, "compose-stop.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("write compose script: %v", err)
	}

	// 写入脚本后应能被识别
	if !dm.hasComposeScript(baseDir) {
		t.Fatalf("expected compose script detected")
	}
}

func TestIsExecutable(t *testing.T) {
	baseDir := t.TempDir()
	dm := NewDockerManager(baseDir, true, time.Second)

	// .sh 脚本在不同平台的可执行判断
	shPath := filepath.Join(baseDir, "compose-up.sh")
	if err := os.WriteFile(shPath, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	if runtime.GOOS == "windows" {
		if !dm.isExecutable(shPath) {
			t.Fatalf("expected .sh file executable on windows")
		}
	} else {
		if err := os.Chmod(shPath, 0o700); err != nil {
			t.Fatalf("chmod script: %v", err)
		}
		if !dm.isExecutable(shPath) {
			t.Fatalf("expected executable script")
		}

		nonExec := filepath.Join(baseDir, "not-exec.sh")
		if err := os.WriteFile(nonExec, []byte("#!/bin/sh\n"), 0o600); err != nil {
			t.Fatalf("write non-exec: %v", err)
		}
		if dm.isExecutable(nonExec) {
			t.Fatalf("expected non-exec file to be non-executable")
		}
	}

	txtPath := filepath.Join(baseDir, "note.txt")
	if err := os.WriteFile(txtPath, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	if dm.isExecutable(txtPath) {
		t.Fatalf("expected non-script not executable")
	}
}

func TestClassifyServices(t *testing.T) {
	// 按优先级名称（不区分大小写）分类服务
	services := []*Service{
		{Name: "db"},
		{Name: "api"},
		{Name: "cache"},
	}

	priority, normal := ClassifyServices(services, []string{"DB", "cache"})
	if len(priority) != 2 || len(normal) != 1 {
		t.Fatalf("unexpected classification: priority=%d normal=%d", len(priority), len(normal))
	}
	if priority[0].Name == normal[0].Name || priority[1].Name == normal[0].Name {
		t.Fatalf("priority and normal overlap")
	}
}

func TestStopStartDryRun(t *testing.T) {
	baseDir := t.TempDir()
	svcDir := filepath.Join(baseDir, "svc")
	if err := os.MkdirAll(svcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(svcDir, "compose.yaml"), []byte("services:\n"), 0o600); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	// dry-run 模式下应不执行外部命令但流程可跑通
	dm := NewDockerManager(baseDir, true, time.Second)
	svc := &Service{Name: "svc", Path: svcDir, Running: true}

	if err := dm.Stop(svc); err != nil {
		t.Fatalf("Stop dry-run error: %v", err)
	}
	if err := dm.Start(svc); err != nil {
		t.Fatalf("Start dry-run error: %v", err)
	}
}

func TestStopStartMissingMethod(t *testing.T) {
	baseDir := t.TempDir()
	svcDir := filepath.Join(baseDir, "svc")
	if err := os.MkdirAll(svcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// 没有 compose 文件/脚本时应报无法识别方法
	dm := NewDockerManager(baseDir, true, time.Second)
	svc := &Service{Name: "svc", Path: svcDir, Running: true}

	if err := dm.Stop(svc); err == nil {
		t.Fatalf("expected stop missing method error")
	}
	if err := dm.Start(svc); err == nil {
		t.Fatalf("expected start missing method error")
	}
}
