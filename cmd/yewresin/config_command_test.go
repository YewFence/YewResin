package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func stubConfigCommandDeps(t *testing.T, configPath string, editorFn func(string, string) error) {
	t.Helper()

	originalConfigPath := getDefaultConfigFilePath
	originalEditorFn := runEditorCommand
	originalLookPath := lookPath
	originalGetEditorEnv := getEditorEnv

	getDefaultConfigFilePath = func() (string, error) {
		return configPath, nil
	}
	if editorFn != nil {
		runEditorCommand = editorFn
	}
	lookPath = exec.LookPath
	getEditorEnv = func() string { return os.Getenv("EDITOR") }

	t.Cleanup(func() {
		getDefaultConfigFilePath = originalConfigPath
		runEditorCommand = originalEditorFn
		lookPath = originalLookPath
		getEditorEnv = originalGetEditorEnv
	})
}

func TestRunConfigInitCreatesFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "yewresin", "config.toml")
	stubConfigCommandDeps(t, configPath, nil)

	input := strings.NewReader("/srv/docker\ngdrive:backup\n")
	var output bytes.Buffer

	if err := runConfigInit(input, &output, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	got := string(content)
	if !strings.Contains(got, `base_dir = "/srv/docker"`) {
		t.Fatalf("config should contain base_dir, got:\n%s", got)
	}
	if !strings.Contains(got, `expected_remote = "gdrive:backup"`) {
		t.Fatalf("config should contain expected_remote, got:\n%s", got)
	}
	if !strings.Contains(output.String(), "已创建配置文件") {
		t.Fatalf("output should mention created config file, got: %s", output.String())
	}
}

func TestRunConfigInitRejectsExistingFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("base_dir = \"/data\"\n"), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	stubConfigCommandDeps(t, configPath, nil)

	err := runConfigInit(strings.NewReader(""), &bytes.Buffer{}, false)
	if err == nil || !strings.Contains(err.Error(), "默认配置文件已存在") {
		t.Fatalf("expected existing file error, got %v", err)
	}
}

func TestRunConfigInitForceOverwritesExistingFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("base_dir = \"/old\"\nexpected_remote = \"old\"\n"), 0o600); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}
	stubConfigCommandDeps(t, configPath, nil)

	input := strings.NewReader("/new/docker\nnew:backup\n")
	var output bytes.Buffer

	if err := runConfigInit(input, &output, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	got := string(content)
	if !strings.Contains(got, `base_dir = "/new/docker"`) {
		t.Fatalf("config should contain overwritten base_dir, got:\n%s", got)
	}
	if !strings.Contains(output.String(), "已覆盖配置文件") {
		t.Fatalf("output should mention overwritten config file, got: %s", output.String())
	}
}

func TestRunConfigEditUsesEditor(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("base_dir = \"/data\"\n"), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	var calledEditor string
	var calledPath string

	stubConfigCommandDeps(t, configPath, func(editor, path string) error {
		calledEditor = editor
		calledPath = path
		return nil
	})

	t.Setenv("EDITOR", "code --wait")

	if err := runConfigEdit(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calledEditor != "code --wait" {
		t.Fatalf("expected editor to be passed through, got %q", calledEditor)
	}
	if calledPath != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, calledPath)
	}
}

func TestRunConfigEditFallsBackWithoutEditorEnv(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("base_dir = \"/data\"\n"), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	var calledEditor string

	stubConfigCommandDeps(t, configPath, func(editor, path string) error {
		calledEditor = editor
		return nil
	})

	getEditorEnv = func() string { return "" }
	lookPath = func(file string) (string, error) {
		if file == fallbackEditors()[0][0] {
			return file, nil
		}
		return "", os.ErrNotExist
	}

	if err := runConfigEdit(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calledEditor != strings.Join(fallbackEditors()[0], " ") {
		t.Fatalf("expected fallback editor %q, got %q", strings.Join(fallbackEditors()[0], " "), calledEditor)
	}
}

func TestResolveEditorCommandFailsWhenNoEditorAvailable(t *testing.T) {
	stubConfigCommandDeps(t, filepath.Join(t.TempDir(), "config.toml"), nil)

	getEditorEnv = func() string { return "" }
	lookPath = func(string) (string, error) {
		return "", os.ErrNotExist
	}

	_, err := resolveEditorCommand()
	if err == nil || !strings.Contains(err.Error(), "未设置 EDITOR") {
		t.Fatalf("expected missing editor error, got %v", err)
	}
}

func TestRunConfigCommandUnknownSubcommand(t *testing.T) {
	exitCode := runConfigCommand([]string{"wat"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
}

func TestRunConfigCommandInitForce(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("base_dir = \"/old\"\nexpected_remote = \"old\"\n"), 0o600); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}
	stubConfigCommandDeps(t, configPath, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runConfigCommand([]string{"init", "--force"}, strings.NewReader("/forced/docker\nforced:backup\n"), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", exitCode, stderr.String())
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if !strings.Contains(string(content), `expected_remote = "forced:backup"`) {
		t.Fatalf("config should be overwritten, got:\n%s", string(content))
	}
}

func TestSplitCommandLine(t *testing.T) {
	args, err := splitCommandLine(`"C:\Program Files\VS Code\Code.exe" --wait`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[0] != `C:\Program Files\VS Code\Code.exe` {
		t.Fatalf("unexpected command: %q", args[0])
	}
	if args[1] != "--wait" {
		t.Fatalf("unexpected arg: %q", args[1])
	}
}
