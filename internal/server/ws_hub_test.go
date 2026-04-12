package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leeson1/agent-forge/internal/stream"
)

func setupTestHub(t *testing.T) (*WSHub, *stream.EventBus) {
	t.Helper()
	eb := stream.NewEventBus(64)
	hub := NewWSHub(eb)
	go hub.Run()
	return hub, eb
}

func connectWS(t *testing.T, server *httptest.Server, taskID string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	if taskID != "" {
		url += "?task_id=" + taskID
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	return conn
}

func TestWSHub_ClientCount(t *testing.T) {
	hub, _ := setupTestHub(t)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount: got %d, want 0", hub.ClientCount())
	}
}

func TestWSHub_ConnectAndReceiveEvent(t *testing.T) {
	hub, eb := setupTestHub(t)

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	conn := connectWS(t, server, "")
	defer conn.Close()

	// 等待连接注册
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("ClientCount: got %d, want 1", hub.ClientCount())
	}

	// 发布事件
	ev := stream.NewEvent(stream.EventTaskStatus, "task-1", map[string]string{"status": "running"})
	eb.Publish(ev)

	// 读取 WebSocket 消息
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var received stream.Event
	if err := json.Unmarshal(message, &received); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if received.Type != stream.EventTaskStatus {
		t.Errorf("Event type: got %s, want %s", received.Type, stream.EventTaskStatus)
	}
	if received.TaskID != "task-1" {
		t.Errorf("TaskID: got %s, want task-1", received.TaskID)
	}
}

func TestWSHub_FilterByTaskID(t *testing.T) {
	hub, eb := setupTestHub(t)

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	// 连接并订阅 task-1
	conn := connectWS(t, server, "task-1")
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// 发布 task-2 事件（不应收到）
	eb.Publish(stream.NewEvent(stream.EventTaskStatus, "task-2", nil))

	// 发布 task-1 事件（应该收到）
	eb.Publish(stream.NewEvent(stream.EventTaskStatus, "task-1", nil))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var received stream.Event
	json.Unmarshal(message, &received)
	if received.TaskID != "task-1" {
		t.Errorf("Should receive task-1 event, got task_id=%s", received.TaskID)
	}
}

func TestWSHub_MultipleClients(t *testing.T) {
	hub, eb := setupTestHub(t)

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	conn1 := connectWS(t, server, "")
	defer conn1.Close()
	conn2 := connectWS(t, server, "")
	defer conn2.Close()

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 2 {
		t.Errorf("ClientCount: got %d, want 2", hub.ClientCount())
	}

	// 发布事件
	eb.Publish(stream.NewEvent(stream.EventAlert, "task-1", "hello"))

	// 两个客户端都应收到
	for _, conn := range []*websocket.Conn{conn1, conn2} {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("Client should receive message: %v", err)
		}
	}
}

func TestWSHub_ReceiveMultipleEventsAsSeparateFrames(t *testing.T) {
	hub, eb := setupTestHub(t)

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	conn := connectWS(t, server, "")
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	eb.Publish(stream.NewEvent(stream.EventTaskStatus, "task-1", map[string]string{"status": "running"}))
	eb.Publish(stream.NewEvent(stream.EventLog, "task-1", map[string]string{"content": "hello"}))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 2; i++ {
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage %d failed: %v", i, err)
		}

		var received stream.Event
		if err := json.Unmarshal(message, &received); err != nil {
			t.Fatalf("message %d should be standalone JSON, got %q: %v", i, string(message), err)
		}
	}
}

func TestWSHub_Disconnect(t *testing.T) {
	hub, _ := setupTestHub(t)

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	conn := connectWS(t, server, "")

	time.Sleep(100 * time.Millisecond)
	if hub.ClientCount() != 1 {
		t.Errorf("ClientCount before close: got %d, want 1", hub.ClientCount())
	}

	conn.Close()
	time.Sleep(200 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount after close: got %d, want 0", hub.ClientCount())
	}
}
