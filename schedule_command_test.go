package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func stubScheduleDeps(t *testing.T) {
	t.Helper()

	originalResolveConfigPath := scheduleResolveConfigPath
	originalGetExecutablePath := scheduleGetExecutablePath
	originalGetUserConfigDir := scheduleGetUserConfigDir
	originalLookPath := scheduleLookPath
	originalReadCrontab := scheduleReadCrontab
	originalInstallCrontab := scheduleInstallCrontab
	originalRunSystemctl := scheduleRunSystemctl
	originalMkdirAll := scheduleMkdirAll
	originalWriteFile := scheduleWriteFile
	originalReadFile := scheduleReadFile
	originalRemoveFile := scheduleRemoveFile
	originalStat := scheduleStat
	originalGOOS := scheduleGOOS

	scheduleResolveConfigPath = func(path string) (string, error) { return path, nil }
	scheduleGetExecutablePath = func() (string, error) { return filepath.Join(t.TempDir(), "yewresin"), nil }
	scheduleGetUserConfigDir = func() (string, error) { return t.TempDir(), nil }
	scheduleLookPath = func(file string) (string, error) { return file, nil }
	scheduleReadCrontab = func() (string, error) { return "", nil }
	scheduleInstallCrontab = func(string) error { return nil }
	scheduleRunSystemctl = func(args ...string) (string, error) { return "", nil }
	scheduleMkdirAll = os.MkdirAll
	scheduleWriteFile = os.WriteFile
	scheduleReadFile = os.ReadFile
	scheduleRemoveFile = os.Remove
	scheduleStat = os.Stat
	scheduleGOOS = "linux"

	t.Cleanup(func() {
		scheduleResolveConfigPath = originalResolveConfigPath
		scheduleGetExecutablePath = originalGetExecutablePath
		scheduleGetUserConfigDir = originalGetUserConfigDir
		scheduleLookPath = originalLookPath
		scheduleReadCrontab = originalReadCrontab
		scheduleInstallCrontab = originalInstallCrontab
		scheduleRunSystemctl = originalRunSystemctl
		scheduleMkdirAll = originalMkdirAll
		scheduleWriteFile = originalWriteFile
		scheduleReadFile = originalReadFile
		scheduleRemoveFile = originalRemoveFile
		scheduleStat = originalStat
		scheduleGOOS = originalGOOS
	})
}

func TestRunScheduleInstallCronWritesManagedBlock(t *testing.T) {
	stubScheduleDeps(t)

	exePath := filepath.Join(t.TempDir(), "bin", "yewresin")
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
		t.Fatalf("mkdir exe dir: %v", err)
	}
	if err := os.WriteFile(exePath, []byte(""), 0o755); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("base_dir = '/srv/docker'\nexpected_remote = 'gdrive:backup'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	scheduleGetExecutablePath = func() (string, error) { return exePath, nil }
	scheduleResolveConfigPath = func(path string) (string, error) { return configPath, nil }

	var installed string
	scheduleReadCrontab = func() (string, error) {
		return "MAILTO=\"\"\n", nil
	}
	scheduleInstallCrontab = func(content string) error {
		installed = content
		return nil
	}

	var stdout bytes.Buffer
	code := runScheduleCommand([]string{"install"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(installed, scheduleCronBeginMarker) {
		t.Fatalf("expected managed marker, got:\n%s", installed)
	}
	if !strings.Contains(installed, "0 3 * * *") {
		t.Fatalf("expected default cron expr, got:\n%s", installed)
	}
	if !strings.Contains(installed, shellQuote(exePath)) {
		t.Fatalf("expected quoted executable path, got:\n%s", installed)
	}
	if !strings.Contains(installed, shellQuote(configPath)) {
		t.Fatalf("expected quoted config path, got:\n%s", installed)
	}
	if !strings.Contains(stdout.String(), "backend=cron") {
		t.Fatalf("expected stdout mention cron backend, got %s", stdout.String())
	}
}

func TestRunScheduleUninstallCronRemovesManagedBlock(t *testing.T) {
	stubScheduleDeps(t)

	var installed string
	scheduleReadCrontab = func() (string, error) {
		return strings.Join([]string{
			"MAILTO=\"\"",
			scheduleCronBeginMarker,
			"0 3 * * * '/tmp/yewresin' -y",
			scheduleCronEndMarker,
			"",
		}, "\n"), nil
	}
	scheduleInstallCrontab = func(content string) error {
		installed = content
		return nil
	}

	code := runScheduleCommand([]string{"uninstall"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.Contains(installed, scheduleCronBeginMarker) {
		t.Fatalf("expected managed block removed, got:\n%s", installed)
	}
	if !strings.Contains(installed, "MAILTO") {
		t.Fatalf("expected unrelated crontab content preserved, got:\n%s", installed)
	}
}

func TestRunScheduleStatusCronShowsInstalledBlock(t *testing.T) {
	stubScheduleDeps(t)

	scheduleReadCrontab = func() (string, error) {
		return strings.Join([]string{
			scheduleCronBeginMarker,
			"0 3 * * * '/tmp/yewresin' -y",
			scheduleCronEndMarker,
			"",
		}, "\n"), nil
	}

	var stdout bytes.Buffer
	code := runScheduleCommand([]string{"status"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "已安装（backend=cron）") {
		t.Fatalf("expected installed status, got %s", stdout.String())
	}
}

func TestRunScheduleInstallSystemdUserWritesUnitFiles(t *testing.T) {
	stubScheduleDeps(t)

	tempDir := t.TempDir()
	exePath := filepath.Join(tempDir, "bin", "yewresin")
	if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
		t.Fatalf("mkdir exe dir: %v", err)
	}
	if err := os.WriteFile(exePath, []byte(""), 0o755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	scheduleGetExecutablePath = func() (string, error) { return exePath, nil }
	scheduleGetUserConfigDir = func() (string, error) { return tempDir, nil }

	var systemctlCalls []string
	scheduleRunSystemctl = func(args ...string) (string, error) {
		systemctlCalls = append(systemctlCalls, strings.Join(args, " "))
		return "", nil
	}

	var stdout bytes.Buffer
	code := runScheduleCommand([]string{"install", "--backend", "systemd-user"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	servicePath := filepath.Join(tempDir, "systemd", "user", scheduleServiceUnitName())
	timerPath := filepath.Join(tempDir, "systemd", "user", scheduleTimerUnitName())

	serviceContent, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read service file: %v", err)
	}
	timerContent, err := os.ReadFile(timerPath)
	if err != nil {
		t.Fatalf("read timer file: %v", err)
	}

	if !strings.Contains(string(serviceContent), "ExecStart=") {
		t.Fatalf("expected ExecStart in service, got:\n%s", string(serviceContent))
	}
	if !strings.Contains(string(timerContent), "OnCalendar="+defaultSystemdCalendar) {
		t.Fatalf("expected default OnCalendar, got:\n%s", string(timerContent))
	}
	if len(systemctlCalls) != 2 {
		t.Fatalf("expected 2 systemctl calls, got %v", systemctlCalls)
	}
	if !strings.Contains(stdout.String(), "backend=systemd-user") {
		t.Fatalf("expected stdout mention systemd-user backend, got %s", stdout.String())
	}
}

func TestRunScheduleInstallRejectsWindowsCron(t *testing.T) {
	stubScheduleDeps(t)
	scheduleGOOS = "windows"

	code := runScheduleCommand([]string{"install"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestNormalizeScheduleInstallOptionsRejectsMismatchedFlags(t *testing.T) {
	opts := &scheduleOptions{
		backend:    "cron",
		onCalendar: "*-*-* 03:00:00",
	}

	err := normalizeScheduleInstallOptions(opts)
	if err == nil || !strings.Contains(err.Error(), "--on-calendar") {
		t.Fatalf("expected on-calendar mismatch error, got %v", err)
	}
}
