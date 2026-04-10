package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.CLI.ClaudePath != "claude" {
		t.Errorf("Expected claude path 'claude', got %q", cfg.CLI.ClaudePath)
	}
	if cfg.Cost.AlertThreshold != 10.0 {
		t.Errorf("Expected alert threshold 10.0, got %f", cfg.Cost.AlertThreshold)
	}
}

func TestLoadConfig_NotExist(t *testing.T) {
	cfg, err := Load("/nonexistent/config.json")
	if err != nil {
		t.Fatalf("Load nonexistent should not error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Error("Should return default config")
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{
		"server": {"host": "127.0.0.1", "port": 9090},
		"cli": {"claude_path": "/usr/local/bin/claude", "max_retries": 5}
	}`
	os.WriteFile(path, []byte(data), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.CLI.ClaudePath != "/usr/local/bin/claude" {
		t.Errorf("Expected custom claude path, got %q", cfg.CLI.ClaudePath)
	}
	if cfg.CLI.MaxRetries != 5 {
		t.Errorf("Expected max_retries 5, got %d", cfg.CLI.MaxRetries)
	}
}

func TestLoadConfig_Invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("invalid json"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")

	cfg := DefaultConfig()
	cfg.Server.Port = 3000
	cfg.Cost.AlertThreshold = 25.5

	err := cfg.Save(path)
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// 重新加载
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Server.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", loaded.Server.Port)
	}
	if loaded.Cost.AlertThreshold != 25.5 {
		t.Errorf("Expected threshold 25.5, got %f", loaded.Cost.AlertThreshold)
	}
}

func TestEstimateCost(t *testing.T) {
	cfg := DefaultConfig()

	// 1M input + 1M output = $3 + $15 = $18
	cost := cfg.EstimateCost(1_000_000, 1_000_000)
	expected := 18.0
	if cost != expected {
		t.Errorf("Expected cost $%.2f, got $%.2f", expected, cost)
	}

	// 0 tokens = $0
	cost = cfg.EstimateCost(0, 0)
	if cost != 0 {
		t.Errorf("Expected cost $0, got $%.2f", cost)
	}
}

func TestGetWebhookConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Notification.WebhookURL = "http://example.com/hook"

	wc := cfg.GetWebhookConfig()
	if wc.URL != "http://example.com/hook" {
		t.Errorf("Expected webhook URL, got %q", wc.URL)
	}
}

func TestConfigJSON(t *testing.T) {
	cfg := DefaultConfig()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if len(data) == 0 {
		t.Error("JSON should not be empty")
	}
}
