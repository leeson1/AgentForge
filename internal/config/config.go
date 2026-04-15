package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/leeson1/agent-forge/internal/notify"
)

// Config 全局配置
type Config struct {
	mu sync.RWMutex

	// 服务器配置
	Server ServerConfig `json:"server"`
	// 通知配置
	Notification NotificationConfig `json:"notification"`
	// CLI 配置
	CLI CLIConfig `json:"cli"`
	// 成本监控
	Cost CostConfig `json:"cost"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	WebhookURL    string                    `json:"webhook_url"`
	EnabledEvents map[notify.EventType]bool `json:"enabled_events"`
	Headers       map[string]string         `json:"headers,omitempty"`
}

// CLIConfig CLI 相关配置
type CLIConfig struct {
	Provider       string `json:"provider"`
	ClaudePath     string `json:"claude_path"`
	CodexPath      string `json:"codex_path"`
	Model          string `json:"model,omitempty"`
	MaxRetries     int    `json:"max_retries"`
	DefaultTimeout string `json:"default_timeout"`
}

// CostConfig 成本监控配置
type CostConfig struct {
	AlertThreshold   float64 `json:"alert_threshold"`     // 告警阈值（美元）
	HardLimit        float64 `json:"hard_limit"`          // 硬限制（美元），超过自动暂停
	InputCostPerMil  float64 `json:"input_cost_per_mil"`  // 每百万 input token 成本
	OutputCostPerMil float64 `json:"output_cost_per_mil"` // 每百万 output token 成本
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Notification: NotificationConfig{
			EnabledEvents: map[notify.EventType]bool{
				notify.EventTaskComplete:  true,
				notify.EventTaskFailed:    true,
				notify.EventMergeConflict: true,
				notify.EventCostAlert:     true,
			},
		},
		CLI: CLIConfig{
			Provider:       "claude",
			ClaudePath:     "claude",
			CodexPath:      "codex",
			MaxRetries:     3,
			DefaultTimeout: "30m",
		},
		Cost: CostConfig{
			AlertThreshold:   10.0,
			HardLimit:        50.0,
			InputCostPerMil:  3.0,  // Claude Sonnet input
			OutputCostPerMil: 15.0, // Claude Sonnet output
		},
	}
}

// ConfigPath 返回配置文件路径
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agent-forge", "config.json")
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	if path == "" {
		path = ConfigPath()
	}

	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // 不存在则返回默认
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if path == "" {
		path = ConfigPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetWebhookConfig 返回 Webhook 通知配置
func (c *Config) GetWebhookConfig() notify.WebhookConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return notify.WebhookConfig{
		URL:           c.Notification.WebhookURL,
		EnabledEvents: c.Notification.EnabledEvents,
		MaxRetries:    3,
		Headers:       c.Notification.Headers,
	}
}

// EstimateCost 估算 token 成本
func (c *Config) EstimateCost(inputTokens, outputTokens int) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	inputCost := float64(inputTokens) * c.Cost.InputCostPerMil / 1_000_000
	outputCost := float64(outputTokens) * c.Cost.OutputCostPerMil / 1_000_000
	return inputCost + outputCost
}
