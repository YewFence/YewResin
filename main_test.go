package main

import (
	"os"
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
