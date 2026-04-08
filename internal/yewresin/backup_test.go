package yewresin

import (
	"strings"
	"testing"
)

func TestNewKopiaBackup(t *testing.T) {
	kb := NewKopiaBackup("gdrive:backup", "secret", "/etc/kopia/config", "/etc/rclone.conf", true)

	if kb.expectedRemote != "gdrive:backup" {
		t.Fatalf("expectedRemote: got %q, want %q", kb.expectedRemote, "gdrive:backup")
	}
	if kb.password != "secret" {
		t.Fatalf("password: got %q, want %q", kb.password, "secret")
	}
	if kb.configFile != "/etc/kopia/config" {
		t.Fatalf("configFile: got %q, want %q", kb.configFile, "/etc/kopia/config")
	}
	if kb.rcloneConfig != "/etc/rclone.conf" {
		t.Fatalf("rcloneConfig: got %q, want %q", kb.rcloneConfig, "/etc/rclone.conf")
	}
	if !kb.dryRun {
		t.Fatalf("dryRun: got false, want true")
	}
}

func TestNewKopiaBackupEmptyRclone(t *testing.T) {
	kb := NewKopiaBackup("gdrive:backup", "", "", "", false)

	// rcloneConfig 为空时，cachedEnv 不应包含 RCLONE_CONFIG
	for _, env := range kb.cachedEnv {
		if strings.HasPrefix(env, "RCLONE_CONFIG=") {
			t.Fatalf("cachedEnv should not contain RCLONE_CONFIG when rcloneConfig is empty, got %q", env)
		}
	}
}

func TestBuildKopiaArgsWithoutConfig(t *testing.T) {
	kb := NewKopiaBackup("gdrive:backup", "", "", "", false)
	args := kb.buildKopiaArgs("snapshot", "create", "/data")

	if len(args) != 3 || args[0] != "snapshot" {
		t.Fatalf("expected raw args, got %v", args)
	}
}

func TestBuildKopiaArgsWithConfig(t *testing.T) {
	kb := NewKopiaBackup("gdrive:backup", "", "/etc/kopia/config", "", false)
	args := kb.buildKopiaArgs("snapshot", "create", "/data")

	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if !strings.HasPrefix(args[0], "--config-file=") {
		t.Fatalf("first arg should be --config-file, got %q", args[0])
	}
	if args[1] != "snapshot" || args[2] != "create" || args[3] != "/data" {
		t.Fatalf("unexpected trailing args: %v", args[1:])
	}
}

func TestBuildEnvWithRclone(t *testing.T) {
	kb := NewKopiaBackup("gdrive:backup", "", "", "/etc/rclone.conf", false)
	env := kb.buildEnv()

	found := false
	for _, e := range env {
		if e == "RCLONE_CONFIG=/etc/rclone.conf" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("buildEnv should contain RCLONE_CONFIG=/etc/rclone.conf")
	}
}

func TestBuildEnvWithKopiaPassword(t *testing.T) {
	kb := NewKopiaBackup("gdrive:backup", "secret", "", "", false)
	env := kb.buildEnv()

	found := false
	for _, e := range env {
		if e == "KOPIA_PASSWORD=secret" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("buildEnv should contain KOPIA_PASSWORD=secret")
	}
}

func TestCreateSnapshotDryRun(t *testing.T) {
	kb := NewKopiaBackup("gdrive:backup", "", "", "", true)
	if err := kb.CreateSnapshot("/data"); err != nil {
		t.Fatalf("dry-run CreateSnapshot should not error, got: %v", err)
	}
}
