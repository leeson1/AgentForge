package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leeson1/agent-forge/internal/config"
	"github.com/leeson1/agent-forge/internal/session"
)

func TestConfigHandlers_GetAndUpdate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/config", nil)
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/config status: got %d, want %d", w.Code, http.StatusOK)
	}

	var cfg config.Config
	if err := json.NewDecoder(w.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	cfg.CLI.Provider = session.ProviderCodex
	cfg.CLI.CodexPath = "/usr/local/bin/codex"
	cfg.CLI.Model = "gpt-5.4"
	cfg.CLI.DefaultTimeout = "45m"

	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	req = httptest.NewRequest("PUT", "/api/config", bytes.NewReader(body))
	w = httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /api/config status: got %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	execConfig := s.executor.Config()
	if execConfig.Provider != session.ProviderCodex {
		t.Fatalf("executor provider: got %s, want %s", execConfig.Provider, session.ProviderCodex)
	}
	if execConfig.CodexPath != "/usr/local/bin/codex" {
		t.Fatalf("executor codex path: got %q", execConfig.CodexPath)
	}
	if execConfig.Model != "gpt-5.4" {
		t.Fatalf("executor model: got %q", execConfig.Model)
	}
}

func TestUpdateConfig_RejectsInvalidProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s := setupTestServer(t)

	cfg := config.DefaultConfig()
	cfg.CLI.Provider = "unknown"
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	req := httptest.NewRequest("PUT", "/api/config", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PUT /api/config status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}
