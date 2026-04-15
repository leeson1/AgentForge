package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/leeson1/agent-forge/internal/config"
	"github.com/leeson1/agent-forge/internal/notify"
	"github.com/leeson1/agent-forge/internal/session"
)

// GetConfig returns the active runtime configuration.
func (s *Server) GetConfig(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	cfg := s.cfg
	s.configMu.RUnlock()

	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfig persists configuration and applies executor settings to future sessions.
func (s *Server) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := normalizeConfig(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := cfg.Save(""); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.configMu.Lock()
	s.cfg = &cfg
	s.configMu.Unlock()

	s.executor.UpdateConfig(ExecutorConfigFromConfig(&cfg))
	writeJSON(w, http.StatusOK, &cfg)
}

// ExecutorConfigFromConfig converts persisted app config to executor config.
func ExecutorConfigFromConfig(cfg *config.Config) session.ExecutorConfig {
	execConfig := session.DefaultExecutorConfig()
	if cfg == nil {
		return execConfig
	}
	if cfg.CLI.Provider != "" {
		execConfig.Provider = cfg.CLI.Provider
	}
	if cfg.CLI.ClaudePath != "" {
		execConfig.ClaudePath = cfg.CLI.ClaudePath
	}
	if cfg.CLI.CodexPath != "" {
		execConfig.CodexPath = cfg.CLI.CodexPath
	}
	if cfg.CLI.Model != "" {
		execConfig.Model = cfg.CLI.Model
	}
	if cfg.CLI.MaxRetries > 0 {
		execConfig.MaxRetries = cfg.CLI.MaxRetries
	}
	if cfg.CLI.DefaultTimeout != "" {
		if timeout, err := time.ParseDuration(cfg.CLI.DefaultTimeout); err == nil {
			execConfig.Timeout = timeout
		}
	}
	return execConfig
}

func normalizeConfig(cfg *config.Config) error {
	defaults := config.DefaultConfig()
	if cfg.Server.Host == "" {
		cfg.Server.Host = defaults.Server.Host
	}
	if cfg.Server.Port <= 0 {
		cfg.Server.Port = defaults.Server.Port
	}

	provider := strings.ToLower(strings.TrimSpace(cfg.CLI.Provider))
	if provider == "" {
		provider = session.ProviderClaude
	}
	if provider != session.ProviderClaude && provider != session.ProviderCodex {
		return fmt.Errorf("unsupported cli provider: %s", cfg.CLI.Provider)
	}
	cfg.CLI.Provider = provider
	if cfg.CLI.ClaudePath == "" {
		cfg.CLI.ClaudePath = defaults.CLI.ClaudePath
	}
	if cfg.CLI.CodexPath == "" {
		cfg.CLI.CodexPath = defaults.CLI.CodexPath
	}
	if cfg.CLI.MaxRetries <= 0 {
		cfg.CLI.MaxRetries = defaults.CLI.MaxRetries
	}
	if cfg.CLI.DefaultTimeout == "" {
		cfg.CLI.DefaultTimeout = defaults.CLI.DefaultTimeout
	}
	if _, err := time.ParseDuration(cfg.CLI.DefaultTimeout); err != nil {
		return fmt.Errorf("invalid default_timeout: %w", err)
	}

	if cfg.Notification.EnabledEvents == nil {
		cfg.Notification.EnabledEvents = map[notify.EventType]bool{
			notify.EventTaskComplete:  true,
			notify.EventTaskFailed:    true,
			notify.EventMergeConflict: true,
			notify.EventCostAlert:     true,
		}
	}
	return nil
}
