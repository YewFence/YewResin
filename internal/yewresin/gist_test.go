package yewresin

import (
	"testing"
	"time"
)

func dummyTime() time.Time {
	return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
}

func TestNewGistManager(t *testing.T) {
	gm := NewGistManager("token123", "gist456", "yewresin", 5, true)

	if gm.token != "token123" {
		t.Fatalf("token: got %q", gm.token)
	}
	if gm.gistID != "gist456" {
		t.Fatalf("gistID: got %q", gm.gistID)
	}
	if gm.logPrefix != "yewresin" {
		t.Fatalf("logPrefix: got %q", gm.logPrefix)
	}
	if gm.maxLogs != 5 {
		t.Fatalf("maxLogs: got %d", gm.maxLogs)
	}
	if !gm.keepFirstFile {
		t.Fatalf("keepFirstFile: got false")
	}
	if gm.client == nil {
		t.Fatalf("client should not be nil")
	}
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		gistID string
		want   bool
	}{
		{"both set", "tok", "id", true},
		{"empty token", "", "id", false},
		{"empty gistID", "tok", "", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := NewGistManager(tt.token, tt.gistID, "prefix", 5, false)
			if got := gm.IsConfigured(); got != tt.want {
				t.Fatalf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadNotConfigured(t *testing.T) {
	gm := NewGistManager("", "", "prefix", 5, false)
	if err := gm.Upload("log content", true, dummyTime(), 0); err != nil {
		t.Fatalf("Upload with unconfigured GistManager should return nil, got: %v", err)
	}
}
