package main

import (
	"os"
	"reflect"
	"testing"
)

func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	// 用管道模拟标准输入，便于测试交互式确认
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	old := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = old
		r.Close()
	}()

	fn()
}

func TestConfirm(t *testing.T) {
	// y 应该通过确认
	withStdin(t, "y\n", func() {
		if !confirm() {
			t.Fatalf("expected confirm to accept y")
		}
	})

	// yes 应该通过确认
	withStdin(t, "yes\n", func() {
		if !confirm() {
			t.Fatalf("expected confirm to accept yes")
		}
	})

	// n 应该拒绝确认
	withStdin(t, "n\n", func() {
		if confirm() {
			t.Fatalf("expected confirm to reject n")
		}
	})
}

func TestFindCommandAfterGlobalConfigFlag(t *testing.T) {
	index, command := findCommand([]string{"--config", "custom.env", "config", "list"})
	if index != 2 || command != "config" {
		t.Fatalf("expected command config at index 2, got %q at %d", command, index)
	}
}

func TestPrepareConfigCommandArgsForwardsGlobalConfig(t *testing.T) {
	got := prepareConfigCommandArgs([]string{"--config", "custom.env"}, []string{"list"})
	want := []string{"--config", "custom.env", "list"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestPrepareConfigCommandArgsKeepsSubcommandConfig(t *testing.T) {
	got := prepareConfigCommandArgs([]string{"--config", "global.env"}, []string{"list", "--config", "local.env"})
	want := []string{"list", "--config", "local.env"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
