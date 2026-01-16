package api

import (
	"context"
	"fmt"
	"go_ProFiBus/api/handlers"
	"go_ProFiBus/api/middleware"
	websocket "go_ProFiBus/internal/interfaces/websocket"
	"go_ProFiBus/logger"
	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/storage"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ServerConfig API服务器配置
type ServerConfig struct {
	Host         string        // 监听地址
	Port         int           // 监听端口
	Mode         string        // 运行模式: debug, release, test
	ReadTimeout  time.Duration // 读超时
	WriteTimeout time.Duration // 写超时
	EnableCORS   bool          // 启用CORS
	AllowOrigins []string      // 允许的源
}

// DefaultServerConfig 默认服务器配置
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:         "0.0.0.0",
		Port:         8080,
		Mode:         gin.ReleaseMode,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		EnableCORS:   true,
		AllowOrigins: []string{"*"},
	}
}

// Server REST API服务器
type Server struct {
	config     *ServerConfig
	router     *gin.Engine
	httpServer *http.Server
	store      *storage.PostgresStore
	wsHub      *websocket.Hub
	tracer     interfaces.Tracer
	log        *logger.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewServer 创建新的API服务器
func NewServer(config *ServerConfig, store *storage.PostgresStore, tracer interfaces.Tracer) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	// 设置Gin模式
	gin.SetMode(config.Mode)

	// 创建路由器
	router := gin.New()

	// 添加恢复中间件（防止panic导致服务器崩溃）
	router.Use(gin.Recovery())

	// 添加日志中间件
	router.Use(middleware.Logger())

	// 添加CORS中间件
	if config.EnableCORS {
		if len(config.AllowOrigins) > 0 {
			router.Use(middleware.CORSWithConfig(
				config.AllowOrigins,
				[]string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
				[]string{"Origin", "Content-Type", "Accept", "Authorization"},
			))
		} else {
			router.Use(middleware.CORS())
		}
	}

	// 创建WebSocket Hub
	wsHub := websocket.NewHub()

	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config: config,
		router: router,
		store:  store,
		wsHub:  wsHub,
		tracer: tracer,
		log:    logger.GetLogger(),
		ctx:    ctx,
		cancel: cancel,
	}

	// 注册路由
	server.registerRoutes()

	return server, nil
}

// registerRoutes 注册所有API路由
func (s *Server) registerRoutes() {
	// 健康检查端点
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/ping", s.handlePing)

	// WebSocket端点
	if s.wsHub != nil {
		wsHandler := websocket.NewHandler(s.wsHub)
		s.router.GET("/ws/trace", func(c *gin.Context) {
			wsHandler.ServeWS(c.Writer, c.Request)
		})
	}

	// 创建handlers
	sensorHandler := handlers.NewSensorHandler(s.store)
	eventHandler := handlers.NewEventHandler(s.store)
	ruleHandler := handlers.NewRuleHandler(s.store)

	// API v1路由组
	v1 := s.router.Group("/api/v1")
	{
		// 传感器数据路由
		sensors := v1.Group("/sensors")
		{
			sensors.GET("/:sensor_id/readings", sensorHandler.GetSensorReadings)
			sensors.POST("/readings", sensorHandler.PostSensorReadings)
			sensors.GET("/:sensor_id/aggregation", sensorHandler.GetSensorAggregation)
		}

		// 事件管理路由
		events := v1.Group("/events")
		{
			events.GET("", eventHandler.GetEvents)
			events.GET("/:event_id", eventHandler.GetEvent)
			events.PUT("/:event_id", eventHandler.UpdateEvent)
			events.GET("/stats", eventHandler.GetEventStats)
		}

		// 规则管理路由
		rules := v1.Group("/rules")
		{
			rules.GET("", ruleHandler.GetRules)
			rules.GET("/:rule_id", ruleHandler.GetRule)
			rules.POST("", ruleHandler.CreateRule)
			rules.PUT("/:rule_id", ruleHandler.UpdateRule)
			rules.DELETE("/:rule_id", ruleHandler.DeleteRule)
		}
	}
}

// handleHealth 健康检查处理器
func (s *Server) handleHealth(c *gin.Context) {
	// 检查数据库连接
	dbHealthy := true
	if s.store != nil {
		if err := s.store.Health(); err != nil {
			dbHealthy = false
		}
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !dbHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":   status,
		"database": dbHealthy,
		"time":     time.Now().Format(time.RFC3339),
	})
}

// handlePing ping处理器
func (s *Server) handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// Start 启动API服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// 启动WebSocket Hub
	if s.wsHub != nil {
		go s.wsHub.Run()
		s.log.Info("WebSocket Hub已启动")
	}

	// 连接Tracer到WebSocket Hub
	if s.tracer != nil && s.wsHub != nil {
		go s.bridgeTracerToWebSocket()
		s.log.Info("Tracer已连接到WebSocket Hub")
	}

	s.log.Info("启动REST API服务器 地址=%s", addr)

	// 在goroutine中启动服务器
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("REST API服务器错误: %v", err)
		}
	}()

	return nil
}

// bridgeTracerToWebSocket 桥接Tracer事件到WebSocket
func (s *Server) bridgeTracerToWebSocket() {
	// 订阅tracer事件
	eventCh := s.tracer.Subscribe()

	for {
		select {
		case <-s.ctx.Done():
			s.tracer.Unsubscribe(eventCh)
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			// 广播事件到所有WebSocket客户端
			s.wsHub.BroadcastEvent(event)
		}
	}
}

// Stop 停止API服务器
func (s *Server) Stop() error {
	s.log.Info("正在停止REST API服务器...")

	// 取消上下文
	s.cancel()

	// 优雅关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.log.Error("关闭REST API服务器失败: %v", err)
		return err
	}

	s.log.Info("REST API服务器已停止")
	return nil
}

// Router 获取Gin路由器（用于测试）
func (s *Server) Router() *gin.Engine {
	return s.router
}
