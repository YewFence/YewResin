package yewresin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempEnvFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.env")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatalf("failed to write temp env: %v", err)
	}
	return path
}

func TestGetEnvDefault(t *testing.T) {
	t.Setenv("TEST_DEFAULT", "")
	if got := getEnvDefault("TEST_DEFAULT", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	t.Setenv("TEST_DEFAULT", "value")
	if got := getEnvDefault("TEST_DEFAULT", "fallback"); got != "value" {
		t.Fatalf("expected value, got %q", got)
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("TEST_INT", "")
	if got := getEnvInt("TEST_INT", 7); got != 7 {
		t.Fatalf("expected default 7, got %d", got)
	}

	t.Setenv("TEST_INT", "42")
	if got := getEnvInt("TEST_INT", 7); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
	// Test invalid input - should return default
	t.Setenv("TEST_INT", "invalid")
	if got := getEnvInt("TEST_INT", 7); got != 7 {
		t.Fatalf("expected default 7 for invalid input, got %d", got)
	}
}

func TestGetEnvBool(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
	}

	for _, c := range cases {
		t.Setenv("TEST_BOOL", c.val)
		if got := getEnvBool("TEST_BOOL", false); got != c.want {
			t.Fatalf("val=%q expected %v, got %v", c.val, c.want, got)
		}
	}

	t.Setenv("TEST_BOOL", "")
	if got := getEnvBool("TEST_BOOL", true); got != true {
		t.Fatalf("expected default true, got %v", got)
	}
}

func TestMaskString(t *testing.T) {
	if got := maskString("short"); got != "****(已配置)" {
		t.Fatalf("expected masked short, got %q", got)
	}

	longVal := "https://example.com/path"
	if got := maskString(longVal); got != "https://...path" {
		t.Fatalf("expected masked long, got %q", got)
	}
}

func TestLoadConfigSuccess(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("BASE_DIR", baseDir)
	t.Setenv("EXPECTED_REMOTE", "rclone:backup")
	t.Setenv("PRIORITY_SERVICES_LIST", "db api")

	cfg, err := LoadConfig(writeTempEnvFile(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseDir != baseDir {
		t.Fatalf("expected BaseDir %q, got %q", baseDir, cfg.BaseDir)
	}
	if cfg.ExpectedRemote != "rclone:backup" {
		t.Fatalf("expected ExpectedRemote, got %q", cfg.ExpectedRemote)
	}
	if got := strings.Join(cfg.PriorityServices, ","); got != "db,api" {
		t.Fatalf("unexpected PriorityServices: %q", got)
	}
}

func TestLoadConfigMissingBaseDir(t *testing.T) {
	t.Setenv("BASE_DIR", "")
	t.Setenv("EXPECTED_REMOTE", "rclone:backup")

	_, err := LoadConfig(writeTempEnvFile(t))
	if err == nil || !strings.Contains(err.Error(), "BASE_DIR 未设置") {
		t.Fatalf("expected BASE_DIR error, got %v", err)
	}
}

func TestLoadConfigMissingExpectedRemote(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("BASE_DIR", baseDir)
	t.Setenv("EXPECTED_REMOTE", "")

	_, err := LoadConfig(writeTempEnvFile(t))
	if err == nil || !strings.Contains(err.Error(), "EXPECTED_REMOTE 未设置") {
		t.Fatalf("expected EXPECTED_REMOTE error, got %v", err)
	}
}

func TestLoadConfigBaseDirNotExist(t *testing.T) {
	t.Setenv("BASE_DIR", filepath.Join(t.TempDir(), "missing"))
	t.Setenv("EXPECTED_REMOTE", "rclone:backup")

	_, err := LoadConfig(writeTempEnvFile(t))
	if err == nil || !strings.Contains(err.Error(), "BASE_DIR 目录不存在") {
		t.Fatalf("expected BASE_DIR not exist error, got %v", err)
	}
}
