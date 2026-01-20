package websocket

import (
	"go_ProFiBus/pkg/interfaces"
	"sync"
)

// Hub 管理所有WebSocket连接并广播消息
type Hub struct {
	// 已注册的客户端
	clients map[*Client]bool

	// 广播通道
	broadcast chan interfaces.TraceEvent

	// 注册客户端请求
	register chan *Client

	// 注销客户端请求
	unregister chan *Client

	// 互斥锁
	mu sync.RWMutex
}

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan interfaces.TraceEvent, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 启动Hub主循环
func (h *Hub) Run() {
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

// BroadcastEvent 广播追踪事件
func (h *Hub) BroadcastEvent(event interfaces.TraceEvent) {
	select {
	case h.broadcast <- event:
	default:
		// 广播通道满，丢弃事件
	}
}

// GetClientCount 获取当前连接的客户端数量
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
