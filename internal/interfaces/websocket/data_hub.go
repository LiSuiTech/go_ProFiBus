package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DeviceDataEvent 设备数据事件
type DeviceDataEvent struct {
	DeviceID    string                 `json:"device_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	SourceID    string                 `json:"source_id,omitempty"`
	Quality     float64                `json:"quality,omitempty"`
}

// DataHub 管理设备数据WebSocket连接并广播消息
type DataHub struct {
	// 已注册的客户端
	clients map[*DataClient]bool

	// 广播通道
	broadcast chan DeviceDataEvent

	// 注册客户端请求
	register chan *DataClient

	// 注销客户端请求
	unregister chan *DataClient

	// 互斥锁
	mu sync.RWMutex
}

// NewDataHub 创建新的DataHub
func NewDataHub() *DataHub {
	return &DataHub{
		clients:    make(map[*DataClient]bool),
		broadcast:  make(chan DeviceDataEvent, 512),
		register:   make(chan *DataClient),
		unregister: make(chan *DataClient),
	}
}

// Run 启动DataHub主循环
func (h *DataHub) Run() {
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
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// 检查客户端过滤器
				if client.shouldSendEvent(&event) {
					select {
					case client.send <- event:
					default:
						// 发送失败，关闭客户端
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastDataEvent 广播设备数据事件
func (h *DataHub) BroadcastDataEvent(event DeviceDataEvent) {
	select {
	case h.broadcast <- event:
	default:
		// 广播通道满，丢弃事件
	}
}

// GetClientCount 获取当前连接的客户端数量
func (h *DataHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// DataClientFilter 数据客户端过滤器
type DataClientFilter struct {
	DeviceIDs []string `json:"device_ids,omitempty"`
	SourceIDs []string `json:"source_ids,omitempty"`
	Fields    []string `json:"fields,omitempty"` // 只接收指定字段的数据
	MinQuality float64 `json:"min_quality,omitempty"`
}

// DataClient WebSocket数据客户端
type DataClient struct {
	hub    *DataHub
	conn   *websocket.Conn
	send   chan DeviceDataEvent
	filter *DataClientFilter
}

// NewDataClient 创建新的数据客户端
func NewDataClient(hub *DataHub, conn *websocket.Conn) *DataClient {
	return &DataClient{
		hub:    hub,
		conn:   conn,
		send:   make(chan DeviceDataEvent, 256),
		filter: &DataClientFilter{},
	}
}

// shouldSendEvent 检查事件是否应该发送给客户端
func (c *DataClient) shouldSendEvent(event *DeviceDataEvent) bool {
	if c.filter == nil {
		return true
	}

	// 过滤设备ID
	if len(c.filter.DeviceIDs) > 0 {
		found := false
		for _, id := range c.filter.DeviceIDs {
			if id == event.DeviceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 过滤数据源ID
	if len(c.filter.SourceIDs) > 0 && event.SourceID != "" {
		found := false
		for _, id := range c.filter.SourceIDs {
			if id == event.SourceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 过滤质量
	if c.filter.MinQuality > 0 && event.Quality < c.filter.MinQuality {
		return false
	}

	// 过滤字段（如果指定了字段，只发送包含这些字段的数据）
	if len(c.filter.Fields) > 0 {
		hasField := false
		for _, field := range c.filter.Fields {
			if _, ok := event.Data[field]; ok {
				hasField = true
				break
			}
		}
		if !hasField {
			return false
		}
	}

	return true
}

// ReadPump 从WebSocket连接读取消息
func (c *DataClient) ReadPump() {
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
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// 解析过滤器设置
		var filter DataClientFilter
		if err := json.Unmarshal(message, &filter); err == nil {
			c.filter = &filter
			log.Printf("Data client filter updated: %+v", filter)
		}
	}
}

// WritePump 向WebSocket连接写入消息
func (c *DataClient) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub关闭了send通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 序列化事件为JSON
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Failed to marshal data event: %v", err)
				continue
			}

			// 发送JSON消息
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
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
