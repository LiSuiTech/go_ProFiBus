package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有来源（生产环境应该配置严格的CheckOrigin）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler WebSocket处理器
type Handler struct {
	hub *Hub
}

// NewHandler 创建新的WebSocket处理器
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

// ServeWS 处理WebSocket连接请求
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// 创建新客户端
	client := NewClient(h.hub, conn)
	h.hub.register <- client

	// 启动读写pump
	go client.WritePump()
	go client.ReadPump()

	log.Printf("New WebSocket client connected, total clients: %d", h.hub.GetClientCount())
}
