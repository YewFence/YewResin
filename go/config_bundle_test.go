package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDetectConfigFilesReturnsErrorForInvalidEnv(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("BROKEN=\"unterminated\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	if _, err := detectConfigFiles(envPath); err == nil || !strings.Contains(err.Error(), "读取 .env 文件失败") {
		t.Fatalf("expected invalid .env error, got %v", err)
	}
}

func TestDetectConfigFilesReturnsErrorForMissingExplicitConfig(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.env")

	if _, err := detectConfigFiles(missingPath); err == nil || !strings.Contains(err.Error(), "配置文件不存在或不可访问") {
		t.Fatalf("expected missing explicit config error, got %v", err)
	}
}

func TestResolveImportPlanUsesLocalTargetPaths(t *testing.T) {
	baseDir := t.TempDir()
	envPath := filepath.Join(baseDir, ".env")
	if err := os.WriteFile(envPath, []byte(""), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	rclonePath := filepath.Join(baseDir, "target", "rclone.conf")
	kopiaPath := filepath.Join(baseDir, "target", "repository.config")
	t.Setenv("RCLONE_CONFIG", rclonePath)
	t.Setenv("KOPIA_CONFIG_FILE", kopiaPath)

	manifest := &Manifest{
		Files: []ConfigFileEntry{
			{ArchiveName: ".env", OriginalPath: "/tmp/source/.env"},
			{ArchiveName: "rclone.conf", OriginalPath: "/tmp/source/rclone.conf"},
			{ArchiveName: "repository.config", OriginalPath: "/tmp/source/repository.config"},
			{ArchiveName: "repository.config.kopia-password", OriginalPath: "/tmp/source/repository.config.kopia-password"},
		},
	}

	plan, err := resolveImportPlan(manifest, envPath)
	if err != nil {
		t.Fatalf("resolveImportPlan: %v", err)
	}

	want := map[string]string{
		".env":                             filepath.Clean(envPath),
		"rclone.conf":                      filepath.Clean(rclonePath),
		"repository.config":                filepath.Clean(kopiaPath),
		"repository.config.kopia-password": filepath.Clean(kopiaPath + ".kopia-password"),
	}

	for _, entry := range plan {
		if got := entry.TargetPath; got != want[entry.ArchiveName] {
			t.Fatalf("archive %s expected target %q, got %q", entry.ArchiveName, want[entry.ArchiveName], got)
		}
	}
}

func TestResolveImportPlanRejectsUnknownArchiveName(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte(""), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	manifest := &Manifest{
		Files: []ConfigFileEntry{{ArchiveName: "../../evil", OriginalPath: "/tmp/evil"}},
	}

	if _, err := resolveImportPlan(manifest, envPath); err == nil || !strings.Contains(err.Error(), "不受支持") {
		t.Fatalf("expected unsupported archive error, got %v", err)
	}
}

func TestWriteFileAtomicReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.txt")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	if err := writeFileAtomic(path, []byte("new")); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read new file: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("expected file content new, got %q", string(data))
	}
}

func TestDefaultExportOutputPathIncludesTimestamp(t *testing.T) {
	got := defaultExportOutputPath(time.Date(2026, time.March, 7, 11, 22, 33, 0, time.UTC))
	want := "yewresin-config-20260307-112233.age"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestOpenExportOutputFileFailsIfExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.age")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("write existing bundle: %v", err)
	}

	file, err := openExportOutputFile(path)
	if err == nil {
		file.Close()
		t.Fatal("expected error when output file already exists")
	}
	if !os.IsExist(err) {
		t.Fatalf("expected already exists error, got %v", err)
	}
}

func TestRestoreImportPlanRollsBackOnFailure(t *testing.T) {
	baseDir := t.TempDir()
	firstPath := filepath.Join(baseDir, "config", ".env")
	if err := os.MkdirAll(filepath.Dir(firstPath), 0o755); err != nil {
		t.Fatalf("mkdir first dir: %v", err)
	}
	if err := os.WriteFile(firstPath, []byte("old"), 0o640); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	blockedDir := filepath.Join(baseDir, "blocked")
	if err := os.WriteFile(blockedDir, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}
	secondPath := filepath.Join(blockedDir, "rclone.conf")

	plan := []ImportPlanEntry{
		{ArchiveName: ".env", Description: "YewResin 配置文件", TargetPath: firstPath},
		{ArchiveName: "rclone.conf", Description: "Rclone 配置文件", TargetPath: secondPath},
	}
	contents := map[string][]byte{
		".env":        []byte("new"),
		"rclone.conf": []byte("rclone"),
	}

	restored, err := restoreImportPlan(plan, contents, nil)
	if err == nil || !strings.Contains(err.Error(), "已回滚先前写入的文件") {
		t.Fatalf("expected rollback error, got restored=%d err=%v", restored, err)
	}

	data, readErr := os.ReadFile(firstPath)
	if readErr != nil {
		t.Fatalf("read rolled back file: %v", readErr)
	}
	if string(data) != "old" {
		t.Fatalf("expected original content after rollback, got %q", string(data))
	}

	info, statErr := os.Stat(firstPath)
	if statErr != nil {
		t.Fatalf("stat rolled back file: %v", statErr)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o640 {
		t.Fatalf("expected original permissions 0640, got %o", info.Mode().Perm())
	}
}
