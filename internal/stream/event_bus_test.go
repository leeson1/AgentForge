package stream

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	data := map[string]string{"key": "value"}
	ev := NewEvent(EventTaskStatus, "task-1", data)

	if ev.Type != EventTaskStatus {
		t.Errorf("Type: got %s, want %s", ev.Type, EventTaskStatus)
	}
	if ev.TaskID != "task-1" {
		t.Errorf("TaskID: got %s, want task-1", ev.TaskID)
	}
	if ev.ID == "" {
		t.Error("ID should not be empty")
	}
	if ev.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	var parsed map[string]string
	if err := json.Unmarshal(ev.Data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("data key: got %s, want value", parsed["key"])
	}
}

func TestEventBus_PublishSubscribe(t *testing.T) {
	eb := NewEventBus(10)

	sub := eb.Subscribe("sub-1", "")
	if eb.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount: got %d, want 1", eb.SubscriberCount())
	}

	// 发布事件
	ev := NewEvent(EventTaskStatus, "task-1", "hello")
	eb.Publish(ev)

	// 接收事件
	select {
	case received := <-sub.Channel:
		if received.Type != EventTaskStatus {
			t.Errorf("Type: got %s, want %s", received.Type, EventTaskStatus)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBus_FilterByTaskID(t *testing.T) {
	eb := NewEventBus(10)

	sub1 := eb.Subscribe("sub-1", "task-1")
	sub2 := eb.Subscribe("sub-2", "task-2")

	// 发布 task-1 事件
	eb.Publish(NewEvent(EventTaskStatus, "task-1", nil))

	// sub1 应该收到
	select {
	case <-sub1.Channel:
		// ok
	case <-time.After(time.Second):
		t.Fatal("sub1 should receive task-1 event")
	}

	// sub2 不应该收到
	select {
	case <-sub2.Channel:
		t.Fatal("sub2 should not receive task-1 event")
	case <-time.After(100 * time.Millisecond):
		// ok
	}
}

func TestEventBus_SubscribeAll(t *testing.T) {
	eb := NewEventBus(10)

	sub := eb.Subscribe("sub-all", "") // 空 taskID → 订阅所有

	eb.Publish(NewEvent(EventTaskStatus, "task-1", nil))
	eb.Publish(NewEvent(EventTaskStatus, "task-2", nil))

	count := 0
	timeout := time.After(time.Second)
	for count < 2 {
		select {
		case <-sub.Channel:
			count++
		case <-timeout:
			t.Fatalf("expected 2 events, got %d", count)
		}
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus(10)

	eb.Subscribe("sub-1", "")
	if eb.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount: got %d, want 1", eb.SubscriberCount())
	}

	eb.Unsubscribe("sub-1")
	if eb.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount after unsubscribe: got %d, want 0", eb.SubscriberCount())
	}
}

func TestEventBus_NonBlockingPublish(t *testing.T) {
	eb := NewEventBus(1) // buffer size = 1

	eb.Subscribe("sub-1", "")

	// 发布超过 buffer 大小的事件，不应该阻塞
	for i := 0; i < 100; i++ {
		eb.Publish(NewEvent(EventLog, "task-1", nil))
	}
	// 如果到这里没 hang，就通过了
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	eb := NewEventBus(10)

	sub1 := eb.Subscribe("sub-1", "")
	sub2 := eb.Subscribe("sub-2", "")

	eb.Publish(NewEvent(EventAlert, "task-1", "alert!"))

	// 两个订阅者都应收到
	for _, sub := range []*Subscriber{sub1, sub2} {
		select {
		case ev := <-sub.Channel:
			if ev.Type != EventAlert {
				t.Errorf("Event type: got %s, want %s", ev.Type, EventAlert)
			}
		case <-time.After(time.Second):
			t.Fatal("subscriber should receive event")
		}
	}
}

func TestEventBus_EventTypes(t *testing.T) {
	types := []EventType{
		EventTaskStatus, EventSessionStart, EventSessionEnd,
		EventAgentMessage, EventToolCall, EventFeatureUpdate,
		EventMergeConflict, EventBatchUpdate, EventAlert, EventLog,
	}

	for _, et := range types {
		if et == "" {
			t.Errorf("EventType should not be empty")
		}
	}
}
