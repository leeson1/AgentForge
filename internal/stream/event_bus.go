package stream

import (
	"encoding/json"
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventTaskStatus     EventType = "task_status"
	EventSessionStart   EventType = "session_start"
	EventSessionEnd     EventType = "session_end"
	EventAgentMessage   EventType = "agent_message"
	EventToolCall       EventType = "tool_call"
	EventFeatureUpdate  EventType = "feature_update"
	EventMergeConflict  EventType = "merge_conflict"
	EventBatchUpdate    EventType = "batch_update"
	EventAlert          EventType = "alert"
	EventIntervention   EventType = "intervention"
	EventLog            EventType = "log"
)

// Event 事件
type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	TaskID    string          `json:"task_id"`
	SessionID string          `json:"session_id,omitempty"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewEvent 创建事件
func NewEvent(eventType EventType, taskID string, data interface{}) *Event {
	var rawData json.RawMessage
	if data != nil {
		d, err := json.Marshal(data)
		if err == nil {
			rawData = d
		}
	}
	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		TaskID:    taskID,
		Data:      rawData,
		Timestamp: time.Now(),
	}
}

// Subscriber 订阅者
type Subscriber struct {
	ID       string
	TaskID   string          // 空串表示订阅所有
	Channel  chan *Event
}

// EventBus 事件总线
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber
	bufferSize  int
}

// NewEventBus 创建 EventBus
func NewEventBus(bufferSize int) *EventBus {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	return &EventBus{
		subscribers: make(map[string]*Subscriber),
		bufferSize:  bufferSize,
	}
}

// Publish 发布事件
func (eb *EventBus) Publish(event *Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, sub := range eb.subscribers {
		// 按 task_id 过滤
		if sub.TaskID != "" && sub.TaskID != event.TaskID {
			continue
		}
		// 非阻塞发送
		select {
		case sub.Channel <- event:
		default:
			// channel 满了，丢弃事件（避免阻塞）
		}
	}
}

// Subscribe 订阅事件
// taskID 为空串时订阅所有任务
func (eb *EventBus) Subscribe(subscriberID, taskID string) *Subscriber {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	sub := &Subscriber{
		ID:      subscriberID,
		TaskID:  taskID,
		Channel: make(chan *Event, eb.bufferSize),
	}
	eb.subscribers[subscriberID] = sub
	return sub
}

// Unsubscribe 取消订阅
func (eb *EventBus) Unsubscribe(subscriberID string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if sub, ok := eb.subscribers[subscriberID]; ok {
		close(sub.Channel)
		delete(eb.subscribers, subscriberID)
	}
}

// SubscriberCount 返回订阅者数量
func (eb *EventBus) SubscriberCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subscribers)
}

// generateEventID 生成事件 ID
var eventCounter int64
var counterMu sync.Mutex

func generateEventID() string {
	counterMu.Lock()
	eventCounter++
	id := eventCounter
	counterMu.Unlock()
	return time.Now().Format("20060102150405") + "-" + itoa(id)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
