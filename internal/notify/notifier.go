package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// EventType 通知事件类型
type EventType string

const (
	EventTaskComplete  EventType = "task_complete"
	EventTaskFailed    EventType = "task_failed"
	EventMergeConflict EventType = "merge_conflict"
	EventCostAlert     EventType = "cost_alert"
	EventSessionCrash  EventType = "session_crash"
	EventStuckFeature  EventType = "stuck_feature"
)

// Notification 通知消息
type Notification struct {
	Type      EventType         `json:"type"`
	TaskID    string            `json:"task_id"`
	TaskName  string            `json:"task_name"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]string `json:"data,omitempty"`
}

// Notifier 通知接口
type Notifier interface {
	// Send 发送通知
	Send(ctx context.Context, n Notification) error
	// ShouldNotify 检查是否应该发送此类型通知
	ShouldNotify(eventType EventType) bool
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	URL           string            `json:"url"`
	EnabledEvents map[EventType]bool `json:"enabled_events"`
	MaxRetries    int               `json:"max_retries"`
	Headers       map[string]string  `json:"headers,omitempty"`
}

// WebhookNotifier Webhook 通知实现
type WebhookNotifier struct {
	config WebhookConfig
	client *http.Client
}

// NewWebhookNotifier 创建 Webhook 通知器
func NewWebhookNotifier(config WebhookConfig) *WebhookNotifier {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	return &WebhookNotifier{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send 发送 Webhook 通知（含重试）
func (w *WebhookNotifier) Send(ctx context.Context, n Notification) error {
	if w.config.URL == "" {
		return nil
	}

	payload, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避: 1s, 2s, 4s...
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", w.config.URL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "AgentForge/1.0")
		for k, v := range w.config.Headers {
			req.Header.Set(k, v)
		}

		resp, err := w.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("webhook request failed (attempt %d): %w", attempt+1, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil // 成功
		}

		lastErr = fmt.Errorf("webhook returned status %d (attempt %d)", resp.StatusCode, attempt+1)
	}

	return lastErr
}

// ShouldNotify 检查是否应发送此类型通知
func (w *WebhookNotifier) ShouldNotify(eventType EventType) bool {
	if len(w.config.EnabledEvents) == 0 {
		return true // 默认全部启用
	}
	return w.config.EnabledEvents[eventType]
}

// NoopNotifier 空通知器（不做任何事）
type NoopNotifier struct{}

func (NoopNotifier) Send(ctx context.Context, n Notification) error { return nil }
func (NoopNotifier) ShouldNotify(eventType EventType) bool         { return false }

// MultiNotifier 多通知器聚合
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMultiNotifier 创建多通知器
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

// Send 发送通知到所有通知器
func (m *MultiNotifier) Send(ctx context.Context, n Notification) error {
	var lastErr error
	for _, notifier := range m.notifiers {
		if notifier.ShouldNotify(n.Type) {
			if err := notifier.Send(ctx, n); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// ShouldNotify 任一通知器需要则返回 true
func (m *MultiNotifier) ShouldNotify(eventType EventType) bool {
	for _, notifier := range m.notifiers {
		if notifier.ShouldNotify(eventType) {
			return true
		}
	}
	return false
}
