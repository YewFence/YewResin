package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedConfigTemplateStaysInSyncWithRootExample(t *testing.T) {
	rootTemplatePath := filepath.Join("..", "..", "config.toml.example")
	rootTemplate, err := os.ReadFile(rootTemplatePath)
	if err != nil {
		t.Fatalf("read root config template: %v", err)
	}

	embeddedTemplate, err := os.ReadFile("config.toml.example")
	if err != nil {
		t.Fatalf("read embedded config template: %v", err)
	}

	if string(rootTemplate) != string(embeddedTemplate) {
		t.Fatalf("config template drift detected between %s and cmd/yewresin/config.toml.example", rootTemplatePath)
	}
}
