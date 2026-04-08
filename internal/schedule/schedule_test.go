package schedule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func stubScheduleDeps(t *testing.T) {
	t.Helper()

	originalResolveConfigPath := resolveConfigPath
	originalGetExecutablePath := getExecutablePath
	originalGetUserConfigDir := getUserConfigDir
	originalLookPath := lookPath
	originalReadCrontab := readCrontab
	originalInstallCrontab := installCrontab
	originalRunSystemctl := runSystemctl
	originalMkdirAll := mkdirAll
	originalWriteFile := writeFile
	originalReadFile := readFile
	originalRemoveFile := removeFile
	originalStat := stat
	originalGOOS := currentGOOS

	resolveConfigPath = func(path string) (string, error) { return path, nil }
	getExecutablePath = func() (string, error) { return filepath.Join(t.TempDir(), "yewresin"), nil }
	getUserConfigDir = func() (string, error) { return t.TempDir(), nil }
	lookPath = func(file string) (string, error) { return file, nil }
	readCrontab = func() (string, error) { return "", nil }
	installCrontab = func(string) error { return nil }
	runSystemctl = func(args ...string) (string, error) { return "", nil }
	mkdirAll = os.MkdirAll
	writeFile = os.WriteFile
	readFile = os.ReadFile
	removeFile = os.Remove
	stat = os.Stat
	currentGOOS = "linux"

	t.Cleanup(func() {
		resolveConfigPath = originalResolveConfigPath
		getExecutablePath = originalGetExecutablePath
		getUserConfigDir = originalGetUserConfigDir
		lookPath = originalLookPath
		readCrontab = originalReadCrontab
		installCrontab = originalInstallCrontab
		runSystemctl = originalRunSystemctl
		mkdirAll = originalMkdirAll
		writeFile = originalWriteFile
		readFile = originalReadFile
		removeFile = originalRemoveFile
		stat = originalStat
		currentGOOS = originalGOOS
	})
}

func TestInstallCronWritesManagedBlock(t *testing.T) {
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

	getExecutablePath = func() (string, error) { return exePath, nil }
	resolveConfigPath = func(path string) (string, error) { return configPath, nil }

	var installed string
	readCrontab = func() (string, error) {
		return "MAILTO=\"\"\n", nil
	}
	installCrontab = func(content string) error {
		installed = content
		return nil
	}

	opts := Options{Backend: "cron"}
	if err := NormalizeInstallOptions(&opts); err != nil {
		t.Fatalf("normalize options: %v", err)
	}

	if err := Install(&bytes.Buffer{}, opts); err != nil {
		t.Fatalf("install cron: %v", err)
	}

	if !strings.Contains(installed, cronBeginMarker) {
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
}

func TestUninstallCronRemovesManagedBlock(t *testing.T) {
	stubScheduleDeps(t)

	var installed string
	readCrontab = func() (string, error) {
		return strings.Join([]string{
			"MAILTO=\"\"",
			cronBeginMarker,
			"0 3 * * * '/tmp/yewresin' -y",
			cronEndMarker,
			"",
		}, "\n"), nil
	}
	installCrontab = func(content string) error {
		installed = content
		return nil
	}

	if err := Uninstall(&bytes.Buffer{}, "cron"); err != nil {
		t.Fatalf("uninstall cron: %v", err)
	}
	if strings.Contains(installed, cronBeginMarker) {
		t.Fatalf("expected managed block removed, got:\n%s", installed)
	}
	if !strings.Contains(installed, "MAILTO") {
		t.Fatalf("expected unrelated crontab content preserved, got:\n%s", installed)
	}
}

func TestStatusCronShowsInstalledBlock(t *testing.T) {
	stubScheduleDeps(t)

	readCrontab = func() (string, error) {
		return strings.Join([]string{
			cronBeginMarker,
			"0 3 * * * '/tmp/yewresin' -y",
			cronEndMarker,
			"",
		}, "\n"), nil
	}

	var stdout bytes.Buffer
	if err := Status(&stdout, "cron"); err != nil {
		t.Fatalf("status cron: %v", err)
	}
	if !strings.Contains(stdout.String(), "已安装（backend=cron）") {
		t.Fatalf("expected installed status, got %s", stdout.String())
	}
}

func TestInstallSystemdUserWritesUnitFiles(t *testing.T) {
	stubScheduleDeps(t)

	tempDir := t.TempDir()
	exePath := filepath.Join(tempDir, "bin", "yewresin")
	if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
		t.Fatalf("mkdir exe dir: %v", err)
	}
	if err := os.WriteFile(exePath, []byte(""), 0o755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	getExecutablePath = func() (string, error) { return exePath, nil }
	getUserConfigDir = func() (string, error) { return tempDir, nil }

	var systemctlCalls []string
	runSystemctl = func(args ...string) (string, error) {
		systemctlCalls = append(systemctlCalls, strings.Join(args, " "))
		return "", nil
	}

	opts := Options{Backend: "systemd-user"}
	if err := NormalizeInstallOptions(&opts); err != nil {
		t.Fatalf("normalize options: %v", err)
	}

	if err := Install(&bytes.Buffer{}, opts); err != nil {
		t.Fatalf("install systemd-user: %v", err)
	}

	servicePath := filepath.Join(tempDir, "systemd", "user", scheduleServiceUnitName()+".service")
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
}

func TestInstallRejectsWindowsCron(t *testing.T) {
	stubScheduleDeps(t)
	currentGOOS = "windows"

	err := Install(&bytes.Buffer{}, Options{Backend: "cron", Expr: defaultCronExpr})
	if err == nil || !strings.Contains(err.Error(), "Windows 不支持") {
		t.Fatalf("expected windows cron error, got %v", err)
	}
}

func TestNormalizeInstallOptionsRejectsMismatchedFlags(t *testing.T) {
	opts := &Options{
		Backend:    "cron",
		OnCalendar: "*-*-* 03:00:00",
	}

	err := NormalizeInstallOptions(opts)
	if err == nil || !strings.Contains(err.Error(), "--on-calendar") {
		t.Fatalf("expected on-calendar mismatch error, got %v", err)
	}
}
