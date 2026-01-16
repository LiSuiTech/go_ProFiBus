package websocket

import (
	"encoding/json"
	"go_ProFiBus/pkg/interfaces"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 写入等待时间
	writeWait = 10 * time.Second

	// Pong等待时间
	pongWait = 60 * time.Second

	// Ping周期（必须小于pongWait）
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512 * 1024 // 512KB
)

// ClientFilter 客户端过滤器
type ClientFilter struct {
	PipelineIDs   []string          `json:"pipeline_ids,omitempty"`
	ComponentIDs  []string          `json:"component_ids,omitempty"`
	ComponentType string            `json:"component_type,omitempty"`
	MinLevel      string            `json:"min_level,omitempty"`
	SampleIDs     []string          `json:"sample_ids,omitempty"`
}

// Client WebSocket客户端
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan interfaces.TraceEvent
	filter *ClientFilter
}

// NewClient 创建新的客户端
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan interfaces.TraceEvent, 256),
		filter: &ClientFilter{},
	}
}

// shouldSendEvent 检查事件是否应该发送给客户端
func (c *Client) shouldSendEvent(event *interfaces.TraceEvent) bool {
	if c.filter == nil {
		return true
	}

	// 过滤 PipelineID
	if len(c.filter.PipelineIDs) > 0 {
		found := false
		for _, id := range c.filter.PipelineIDs {
			if id == event.PipelineID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 过滤 ComponentID
	if len(c.filter.ComponentIDs) > 0 {
		found := false
		for _, id := range c.filter.ComponentIDs {
			if id == event.ComponentID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 过滤 ComponentType
	if c.filter.ComponentType != "" && c.filter.ComponentType != event.ComponentType {
		return false
	}

	// 过滤 SampleID
	if len(c.filter.SampleIDs) > 0 {
		found := false
		for _, id := range c.filter.SampleIDs {
			if id == event.SampleID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// ReadPump 从WebSocket连接读取消息
func (c *Client) ReadPump() {
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
		var filter ClientFilter
		if err := json.Unmarshal(message, &filter); err == nil {
			c.filter = &filter
			log.Printf("Client filter updated: %+v", filter)
		}
	}
}

// WritePump 向WebSocket连接写入消息
func (c *Client) WritePump() {
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
				log.Printf("Failed to marshal event: %v", err)
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
