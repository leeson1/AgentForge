package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leeson1/agent-forge/internal/stream"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境允许所有来源
	},
}

// WSClient WebSocket 客户端连接
type WSClient struct {
	hub        *WSHub
	conn       *websocket.Conn
	send       chan []byte
	taskID     string // 订阅的任务 ID（空表示所有）
	subscriberID string
}

// WSHub WebSocket 连接管理中心
type WSHub struct {
	mu         sync.RWMutex
	clients    map[*WSClient]bool
	eventBus   *stream.EventBus
	register   chan *WSClient
	unregister chan *WSClient
}

// NewWSHub 创建 WSHub
func NewWSHub(eventBus *stream.EventBus) *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		eventBus:   eventBus,
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
}

// Run 运行 WSHub 主循环
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.eventBus.Unsubscribe(client.subscriberID)
			}
			h.mu.Unlock()
		}
	}
}

// ClientCount 返回连接的客户端数量
func (h *WSHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ServeWS 处理 WebSocket 升级请求
// URL: /ws?task_id=xxx（可选）
func (h *WSHub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	taskID := r.URL.Query().Get("task_id")
	subscriberID := fmt.Sprintf("ws-%d", time.Now().UnixNano())

	client := &WSClient{
		hub:          h,
		conn:         conn,
		send:         make(chan []byte, 256),
		taskID:       taskID,
		subscriberID: subscriberID,
	}

	h.register <- client

	// 订阅 EventBus
	sub := h.eventBus.Subscribe(subscriberID, taskID)

	// 启动读写协程
	go client.writePump()
	go client.readPump()
	go client.eventPump(sub)
}

// readPump 从 WebSocket 读取消息（主要是 pong 和关闭）
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump 向 WebSocket 写入消息
func (c *WSClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// eventPump 从 EventBus 读取事件并转发到 send channel
func (c *WSClient) eventPump(sub *stream.Subscriber) {
	for event := range sub.Channel {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		select {
		case c.send <- data:
		default:
			// channel 满了，关闭连接
			c.hub.unregister <- c
			return
		}
	}
}
