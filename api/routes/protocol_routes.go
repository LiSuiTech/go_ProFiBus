package routes

import (
	"go_ProFiBus/api/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterProtocolRoutes 注册协议管理路由
func RegisterProtocolRoutes(router *gin.RouterGroup) {
	protocolHandler := handlers.NewProtocolHandler()

	protocols := router.Group("/protocols")
	{
		// 列出所有支持的协议类型
		protocols.GET("/types", protocolHandler.ListProtocols)

		// 协议实例管理
		protocols.GET("/instances", protocolHandler.ListInstances)
		protocols.POST("/instances", protocolHandler.CreateInstance)
		protocols.GET("/instances/:id", protocolHandler.GetInstance)
		protocols.DELETE("/instances/:id", protocolHandler.DeleteInstance)

		// 协议连接管理
		protocols.POST("/instances/:id/connect", protocolHandler.ConnectInstance)
		protocols.POST("/instances/:id/disconnect", protocolHandler.DisconnectInstance)

		// 数据监听管理
		protocols.POST("/instances/:id/monitor/start", protocolHandler.StartMonitoring)
		protocols.POST("/instances/:id/monitor/stop", protocolHandler.StopMonitoring)

		// 状态查询
		protocols.GET("/instances/:id/status", protocolHandler.GetStatus)
	}
}
