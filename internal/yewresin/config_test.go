package yewresin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var configEnvKeys = []string{
	"BASE_DIR",
	"EXPECTED_REMOTE",
	"PRIORITY_SERVICES_LIST",
	"LOCK_FILE",
	"LOG_FILE",
	"DOCKER_COMMAND_TIMEOUT_SECONDS",
	"DEVICE_NAME",
	"APPRISE_URL",
	"APPRISE_NOTIFY_URL",
	"GIST_TOKEN",
	"GIST_ID",
	"GIST_LOG_PREFIX",
	"GIST_MAX_LOGS",
	"GIST_KEEP_FIRST_FILE",
	"KOPIA_PASSWORD",
	"KOPIA_CONFIG_FILE",
	"RCLONE_CONFIG",
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range configEnvKeys {
		t.Setenv(key, "")
	}
}

func writeTempConfigFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

func TestParseOptionalInt(t *testing.T) {
	if got := parseOptionalInt(""); got != nil {
		t.Fatalf("expected nil for empty string, got %v", *got)
	}
	if got := parseOptionalInt("42"); got == nil || *got != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
	if got := parseOptionalInt("invalid"); got != nil {
		t.Fatalf("expected nil for invalid input, got %v", *got)
	}
}

func TestParseOptionalBool(t *testing.T) {
	cases := []struct {
		val  string
		want *bool
	}{
		{"true", boolPtr(true)},
		{"1", boolPtr(true)},
		{"yes", boolPtr(true)},
		{"false", boolPtr(false)},
		{"0", boolPtr(false)},
		{"no", boolPtr(false)},
		{"", nil},
		{"invalid", nil},
	}

	for _, c := range cases {
		got := parseOptionalBool(c.val)
		switch {
		case c.want == nil && got != nil:
			t.Fatalf("val=%q expected nil, got %v", c.val, *got)
		case c.want != nil && got == nil:
			t.Fatalf("val=%q expected %v, got nil", c.val, *c.want)
		case c.want != nil && got != nil && *got != *c.want:
			t.Fatalf("val=%q expected %v, got %v", c.val, *c.want, *got)
		}
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

func TestLoadConfigSuccessFromEnv(t *testing.T) {
	clearConfigEnv(t)

	baseDir := t.TempDir()
	t.Setenv("BASE_DIR", baseDir)
	t.Setenv("EXPECTED_REMOTE", "rclone:backup")
	t.Setenv("PRIORITY_SERVICES_LIST", "db api")

	cfg, err := LoadConfig(writeTempConfigFile(t, "test.env", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filepath.Clean(cfg.BaseDir) != filepath.Clean(baseDir) {
		t.Fatalf("expected BaseDir %q, got %q", baseDir, cfg.BaseDir)
	}
	if cfg.ExpectedRemote != "rclone:backup" {
		t.Fatalf("expected ExpectedRemote, got %q", cfg.ExpectedRemote)
	}
	if got := strings.Join(cfg.PriorityServices, ","); got != "db,api" {
		t.Fatalf("unexpected PriorityServices: %q", got)
	}
}

func TestLoadConfigFromEnvFile(t *testing.T) {
	clearConfigEnv(t)

	baseDir := t.TempDir()
	configPath := writeTempConfigFile(t, "test.env", fmt.Sprintf(`
BASE_DIR="%s"
EXPECTED_REMOTE="rclone:backup"
PRIORITY_SERVICES_LIST="db api"
DOCKER_COMMAND_TIMEOUT_SECONDS="45"
GIST_MAX_LOGS="7"
GIST_KEEP_FIRST_FILE="false"
KOPIA_PASSWORD="secret"
`, filepath.ToSlash(baseDir)))

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filepath.Clean(filepath.FromSlash(cfg.BaseDir)) != filepath.Clean(baseDir) {
		t.Fatalf("expected BaseDir %q, got %q", baseDir, cfg.BaseDir)
	}
	if cfg.DockerCommandTimeoutSeconds != 45 {
		t.Fatalf("expected timeout 45, got %d", cfg.DockerCommandTimeoutSeconds)
	}
	if cfg.GistMaxLogs != 7 {
		t.Fatalf("expected GistMaxLogs 7, got %d", cfg.GistMaxLogs)
	}
	if cfg.GistKeepFirstFile {
		t.Fatalf("expected GistKeepFirstFile false")
	}
	if cfg.KopiaPassword != "secret" {
		t.Fatalf("expected KopiaPassword from env file, got %q", cfg.KopiaPassword)
	}
}

func TestLoadConfigFromTOML(t *testing.T) {
	clearConfigEnv(t)

	baseDir := t.TempDir()
	configPath := writeTempConfigFile(t, "config.toml", fmt.Sprintf(`
base_dir = '%s'
expected_remote = 'rclone:backup'
priority_services = ['db', 'api']
lock_file = '/tmp/yewresin.lock'

[logging]
file = '/var/log/yewresin.log'
docker_command_timeout_seconds = 45

[notifications]
device_name = 'HomeServer'
apprise_url = 'https://apprise.example/notify'
apprise_notify_url = 'tgram://token/chat'

[gist]
token = 'ghp_test'
id = 'gist-123'
log_prefix = 'backup-log'
max_logs = 7
keep_first_file = false

[kopia]
password = 'secret'
config_file = '/etc/kopia/repository.config'

[rclone]
config_file = '/etc/rclone/rclone.conf'
`, baseDir))

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := strings.Join(cfg.PriorityServices, ","); got != "db,api" {
		t.Fatalf("unexpected PriorityServices: %q", got)
	}
	if cfg.LockFile != "/tmp/yewresin.lock" {
		t.Fatalf("expected custom lock file, got %q", cfg.LockFile)
	}
	if cfg.LogFile != "/var/log/yewresin.log" {
		t.Fatalf("expected log file from toml, got %q", cfg.LogFile)
	}
	if cfg.DockerCommandTimeoutSeconds != 45 {
		t.Fatalf("expected timeout 45, got %d", cfg.DockerCommandTimeoutSeconds)
	}
	if cfg.DeviceName != "HomeServer" {
		t.Fatalf("expected DeviceName HomeServer, got %q", cfg.DeviceName)
	}
	if cfg.GistLogPrefix != "backup-log" {
		t.Fatalf("expected GistLogPrefix backup-log, got %q", cfg.GistLogPrefix)
	}
	if cfg.GistKeepFirstFile {
		t.Fatalf("expected GistKeepFirstFile false")
	}
	if cfg.KopiaPassword != "secret" {
		t.Fatalf("expected KopiaPassword secret, got %q", cfg.KopiaPassword)
	}
	if cfg.KopiaConfigFile != "/etc/kopia/repository.config" {
		t.Fatalf("expected KopiaConfigFile from toml, got %q", cfg.KopiaConfigFile)
	}
	if cfg.RcloneConfig != "/etc/rclone/rclone.conf" {
		t.Fatalf("expected RcloneConfig from toml, got %q", cfg.RcloneConfig)
	}
}

func TestLoadConfigEnvOverridesTOML(t *testing.T) {
	clearConfigEnv(t)

	fileBaseDir := t.TempDir()
	envBaseDir := t.TempDir()
	configPath := writeTempConfigFile(t, "config.toml", fmt.Sprintf(`
base_dir = '%s'
expected_remote = 'rclone:file'
priority_services = ['db']

[kopia]
password = 'from-file'
`, fileBaseDir))

	t.Setenv("BASE_DIR", envBaseDir)
	t.Setenv("EXPECTED_REMOTE", "rclone:env")
	t.Setenv("PRIORITY_SERVICES_LIST", "api gateway")
	t.Setenv("KOPIA_PASSWORD", "from-env")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseDir != envBaseDir {
		t.Fatalf("expected env BaseDir %q, got %q", envBaseDir, cfg.BaseDir)
	}
	if cfg.ExpectedRemote != "rclone:env" {
		t.Fatalf("expected env ExpectedRemote, got %q", cfg.ExpectedRemote)
	}
	if got := strings.Join(cfg.PriorityServices, ","); got != "api,gateway" {
		t.Fatalf("expected env PriorityServices, got %q", got)
	}
	if cfg.KopiaPassword != "from-env" {
		t.Fatalf("expected env KopiaPassword, got %q", cfg.KopiaPassword)
	}
}

func TestLoadConfigMissingBaseDir(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("EXPECTED_REMOTE", "rclone:backup")

	_, err := LoadConfig(writeTempConfigFile(t, "test.env", ""))
	if err == nil || !strings.Contains(err.Error(), "BASE_DIR 未设置") {
		t.Fatalf("expected BASE_DIR error, got %v", err)
	}
}

func TestLoadConfigMissingExpectedRemote(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("BASE_DIR", t.TempDir())

	_, err := LoadConfig(writeTempConfigFile(t, "test.env", ""))
	if err == nil || !strings.Contains(err.Error(), "EXPECTED_REMOTE 未设置") {
		t.Fatalf("expected EXPECTED_REMOTE error, got %v", err)
	}
}

func TestLoadConfigBaseDirNotExist(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("BASE_DIR", filepath.Join(t.TempDir(), "missing"))
	t.Setenv("EXPECTED_REMOTE", "rclone:backup")

	_, err := LoadConfig(writeTempConfigFile(t, "test.env", ""))
	if err == nil || !strings.Contains(err.Error(), "BASE_DIR 目录不存在") {
		t.Fatalf("expected BASE_DIR not exist error, got %v", err)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
