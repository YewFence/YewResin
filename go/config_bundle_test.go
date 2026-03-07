package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
