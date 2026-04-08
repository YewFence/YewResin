package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/YewFence/yewresin/internal/schedule"
)

func stubScheduleCommandActions(t *testing.T) {
	t.Helper()

	originalInstall := scheduleInstallAction
	originalUninstall := scheduleUninstallAction
	originalStatus := scheduleStatusAction
	originalNormalize := scheduleNormalizeInstallAction

	t.Cleanup(func() {
		scheduleInstallAction = originalInstall
		scheduleUninstallAction = originalUninstall
		scheduleStatusAction = originalStatus
		scheduleNormalizeInstallAction = originalNormalize
	})
}

func TestRunScheduleInstallPassesDefaultCronOptions(t *testing.T) {
	stubScheduleCommandActions(t)

	var got schedule.Options
	scheduleInstallAction = func(_ io.Writer, opts schedule.Options) error {
		got = opts
		return nil
	}

	code := runScheduleCommand([]string{"install"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got.Backend != schedule.DefaultBackend {
		t.Fatalf("expected backend %q, got %q", schedule.DefaultBackend, got.Backend)
	}
	if got.Expr != "0 3 * * *" {
		t.Fatalf("expected default cron expr, got %q", got.Expr)
	}
}

func TestRunScheduleInstallPassesSystemdUserOptions(t *testing.T) {
	stubScheduleCommandActions(t)

	var got schedule.Options
	scheduleInstallAction = func(_ io.Writer, opts schedule.Options) error {
		got = opts
		return nil
	}

	code := runScheduleCommand([]string{"install", "--backend", "systemd-user"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got.Backend != "systemd-user" {
		t.Fatalf("expected backend systemd-user, got %q", got.Backend)
	}
	if got.OnCalendar != "*-*-* 03:00:00" {
		t.Fatalf("expected default OnCalendar, got %q", got.OnCalendar)
	}
}

func TestRunScheduleUninstallPassesBackend(t *testing.T) {
	stubScheduleCommandActions(t)

	var got string
	scheduleUninstallAction = func(_ io.Writer, backend string) error {
		got = backend
		return nil
	}

	code := runScheduleCommand([]string{"uninstall", "--backend", "systemd-user"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got != "systemd-user" {
		t.Fatalf("expected backend systemd-user, got %q", got)
	}
}

func TestRunScheduleStatusPassesBackend(t *testing.T) {
	stubScheduleCommandActions(t)

	var got string
	scheduleStatusAction = func(_ io.Writer, backend string) error {
		got = backend
		return nil
	}

	code := runScheduleCommand([]string{"status"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got != schedule.DefaultBackend {
		t.Fatalf("expected backend %q, got %q", schedule.DefaultBackend, got)
	}
}

func TestRunScheduleCommandUnknownSubcommand(t *testing.T) {
	code := runScheduleCommand([]string{"wat"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestParseScheduleInstallFlagsRejectsMismatchedFlags(t *testing.T) {
	stubScheduleCommandActions(t)

	_, code := parseScheduleInstallFlags([]string{"--backend", "cron", "--on-calendar", "*-*-* 03:00:00"}, &bytes.Buffer{})
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}
