package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookNotifier_Send(t *testing.T) {
	// 创建测试 HTTP 服务器
	var received Notification
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(200)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
		URL:        server.URL,
		MaxRetries: 1,
	})

	err := notifier.Send(context.Background(), Notification{
		Type:      EventTaskComplete,
		TaskID:    "T1",
		TaskName:  "Test Task",
		Message:   "Task completed successfully",
		Timestamp: time.Now(),
	})

	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if received.TaskID != "T1" {
		t.Errorf("Expected task_id T1, got %q", received.TaskID)
	}
	if received.Type != EventTaskComplete {
		t.Errorf("Expected type task_complete, got %q", received.Type)
	}
}

func TestWebhookNotifier_EmptyURL(t *testing.T) {
	notifier := NewWebhookNotifier(WebhookConfig{URL: ""})
	err := notifier.Send(context.Background(), Notification{})
	if err != nil {
		t.Errorf("Empty URL should not error: %v", err)
	}
}

func TestWebhookNotifier_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(500) // 前2次失败
		} else {
			w.WriteHeader(200) // 第3次成功
		}
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
		URL:        server.URL,
		MaxRetries: 3,
	})

	err := notifier.Send(context.Background(), Notification{
		Type:    EventTaskFailed,
		Message: "Test retry",
	})

	if err != nil {
		t.Fatalf("Expected success after retries, got: %v", err)
	}
	if attempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", attempts)
	}
}

func TestWebhookNotifier_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500) // 总是失败
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
		URL:        server.URL,
		MaxRetries: 1,
	})

	err := notifier.Send(context.Background(), Notification{})
	if err == nil {
		t.Error("Expected error after max retries exceeded")
	}
}

func TestWebhookNotifier_ShouldNotify(t *testing.T) {
	// 默认全部启用
	notifier := NewWebhookNotifier(WebhookConfig{URL: "http://example.com"})
	if !notifier.ShouldNotify(EventTaskComplete) {
		t.Error("Default should notify all events")
	}

	// 指定事件
	notifier2 := NewWebhookNotifier(WebhookConfig{
		URL: "http://example.com",
		EnabledEvents: map[EventType]bool{
			EventTaskFailed: true,
		},
	})
	if !notifier2.ShouldNotify(EventTaskFailed) {
		t.Error("Should notify EventTaskFailed")
	}
	if notifier2.ShouldNotify(EventTaskComplete) {
		t.Error("Should not notify EventTaskComplete")
	}
}

func TestWebhookNotifier_CustomHeaders(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(WebhookConfig{
		URL:     server.URL,
		Headers: map[string]string{"Authorization": "Bearer test-token"},
	})

	notifier.Send(context.Background(), Notification{})
	if gotAuth != "Bearer test-token" {
		t.Errorf("Expected auth header, got %q", gotAuth)
	}
}

func TestNoopNotifier(t *testing.T) {
	n := NoopNotifier{}
	if err := n.Send(context.Background(), Notification{}); err != nil {
		t.Errorf("NoopNotifier should not error: %v", err)
	}
	if n.ShouldNotify(EventTaskComplete) {
		t.Error("NoopNotifier should not notify")
	}
}

func TestMultiNotifier(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(200)
	}))
	defer server.Close()

	w1 := NewWebhookNotifier(WebhookConfig{URL: server.URL})
	w2 := NewWebhookNotifier(WebhookConfig{URL: server.URL})
	multi := NewMultiNotifier(w1, w2)

	multi.Send(context.Background(), Notification{Type: EventTaskComplete})
	if calls != 2 {
		t.Errorf("Expected 2 calls, got %d", calls)
	}
}
