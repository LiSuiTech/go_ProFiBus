package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// DataHandler WebSocket数据处理器
type DataHandler struct {
	hub *DataHub
}

// NewDataHandler 创建新的数据WebSocket处理器
func NewDataHandler(hub *DataHub) *DataHandler {
	return &DataHandler{
		hub: hub,
	}
}

// ServeWS 处理WebSocket连接请求
func (h *DataHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// 创建新客户端
	client := NewDataClient(h.hub, conn)
	h.hub.register <- client

	// 启动读写pump
	go client.WritePump()
	go client.ReadPump()

	log.Printf("New data WebSocket client connected, total clients: %d", h.hub.GetClientCount())
}
