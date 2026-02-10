package api

import (
	"context"
	"fmt"
	"go_ProFiBus/api/handlers"
	"go_ProFiBus/api/middleware"
	deviceApp "go_ProFiBus/internal/application/device"
	dataManagementApp "go_ProFiBus/internal/application/datamanagement"
	fusionApp "go_ProFiBus/internal/application/fusion"
	controlApp "go_ProFiBus/internal/application/control"
	trainingApp "go_ProFiBus/internal/application/training"
	ruleTemplateApp "go_ProFiBus/internal/application/rule_template"
	"go_ProFiBus/internal/application/orchestrator"
	"go_ProFiBus/internal/application/workflow"
	configDomain "go_ProFiBus/internal/domain/config"
	"go_ProFiBus/internal/infrastructure/storage"
	websocket "go_ProFiBus/internal/interfaces/websocket"
	"go_ProFiBus/logger"
	"go_ProFiBus/pkg/interfaces"
	storageOld "go_ProFiBus/storage"
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
	config           *ServerConfig
	router           *gin.Engine
	httpServer       *http.Server
	store            *storageOld.PostgresStore
	wsHub            *websocket.Hub
	dataHub          *websocket.DataHub
	tracer           interfaces.Tracer
	traceRepository  interfaces.TraceRepository
	configRepository interfaces.ConfigRepository
	configValidator  *configDomain.Validator
	userRepository   interfaces.UserRepository
	authService      interfaces.AuthService
	authzService     interfaces.AuthorizationService
	orchestrator     *orchestrator.Orchestrator
	workflowEngine   *workflow.Engine
	log              *logger.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	streamBridge     *deviceApp.DataStreamBridge // 数据流桥接服务
}

// NewServer 创建新的API服务器
func NewServer(
	config *ServerConfig,
	store *storageOld.PostgresStore,
	tracer interfaces.Tracer,
	traceRepository interfaces.TraceRepository,
	configRepository interfaces.ConfigRepository,
	userRepository interfaces.UserRepository,
	authService interfaces.AuthService,
	authzService interfaces.AuthorizationService,
	orch *orchestrator.Orchestrator,
) (*Server, error) {
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
	dataHub := websocket.NewDataHub()

	ctx, cancel := context.WithCancel(context.Background())

	// Create config validator
	configValidator := configDomain.NewValidator()

	server := &Server{
		config:           config,
		router:           router,
		store:            store,
		wsHub:            wsHub,
		dataHub:          dataHub,
		tracer:           tracer,
		traceRepository:  traceRepository,
		configRepository: configRepository,
		configValidator:  configValidator,
		userRepository:   userRepository,
		authService:      authService,
		authzService:     authzService,
		orchestrator:     orch,
		log:              logger.GetLogger(),
		ctx:              ctx,
		cancel:           cancel,
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

	// 设备数据流WebSocket端点
	if s.dataHub != nil {
		dataHandler := websocket.NewDataHandler(s.dataHub)
		s.router.GET("/ws/data", func(c *gin.Context) {
			dataHandler.ServeWS(c.Writer, c.Request)
		})
	}

	// 定义actuator factory（用于workflow和control service）
	actuatorFactory := func(deviceID string) (interfaces.Actuator, error) {
		// TODO: 根据设备ID获取对应的执行器
		// 这里返回nil表示未实现，实际应该从设备配置中获取执行器
		return nil, fmt.Errorf("actuator factory not implemented")
	}

	// 初始化工作流引擎（如果配置了）
	if s.store != nil {
		workflowRepo := storage.NewWorkflowRepository(s.store)
		s.workflowEngine = workflow.NewEngine(workflowRepo)
		
		// 获取必要的仓储
		deviceRepo := storage.NewDeviceRepository(s.store)
		alertRepo := storage.NewAlertRepository(s.store)
		fusionRepo := storage.NewFusionRepository(s.store)
		
		// 注册默认节点执行器
		s.workflowEngine.RegisterExecutor(workflow.NewDataSourceNodeExecutor(nil))
		s.workflowEngine.RegisterExecutor(workflow.NewRuleDetectionNodeExecutor(nil))
		s.workflowEngine.RegisterExecutor(workflow.NewConditionNodeExecutor())
		s.workflowEngine.RegisterExecutor(workflow.NewVariableSetNodeExecutor())
		s.workflowEngine.RegisterExecutor(workflow.NewOutputNodeExecutor())
		s.workflowEngine.RegisterExecutor(workflow.NewTransformNodeExecutor())
		s.workflowEngine.RegisterExecutor(workflow.NewFilterNodeExecutor())
		
		// 注册设备相关节点执行器
		s.workflowEngine.RegisterExecutor(workflow.NewDeviceSourceNodeExecutor(deviceRepo, fusionRepo))
		s.workflowEngine.RegisterExecutor(workflow.NewAlertOutputNodeExecutor(alertRepo))
		s.workflowEngine.RegisterExecutor(workflow.NewDeviceControlNodeExecutor(actuatorFactory, deviceRepo))
	}

	// 创建handlers
	sensorHandler := handlers.NewSensorHandler(s.store)
	eventHandler := handlers.NewEventHandler(s.store)
	
	// 创建设备管理handler
	deviceRepo := storage.NewDeviceRepository(s.store)
	deviceHandler := handlers.NewDeviceHandler(deviceRepo)
	
	// 创建设备数据管理handler
	deviceDataRepo := storage.NewDeviceDataRepository(s.store)
	fusionServiceDevice := deviceApp.NewDataFusionService(deviceDataRepo)

	// 创建数据流桥接服务（如果dataHub已初始化）
	if s.dataHub != nil {
		s.streamBridge = deviceApp.NewDataStreamBridge(s.dataHub, fusionServiceDevice)
		fusionServiceDevice.SetStreamBridge(s.streamBridge)
	}

	deviceDataHandler := handlers.NewDeviceDataHandler(deviceDataRepo, fusionServiceDevice)
	
	// 创建告警管理handler
	alertRepo := storage.NewAlertRepository(s.store)
	alertHandler := handlers.NewAlertHandler(alertRepo)
	
	// 创建规则模板handler
	ruleTemplateRepo := storage.NewRuleTemplateRepository(s.store)
	ruleTestService := ruleTemplateApp.NewRuleTestService(ruleTemplateRepo)
	ruleTemplateHandler := handlers.NewRuleTemplateHandler(ruleTemplateRepo, ruleTestService, alertRepo)
	
	// 创建预测分析handler
	predictionRepo := storage.NewPredictionRepository(s.store)
	predictionHandler := handlers.NewPredictionHandler(predictionRepo)

	// 创建訓練服務與handler
	trainingRepo := storage.NewTrainingRepository(s.store)
	trainingService := trainingApp.NewService(trainingRepo, predictionRepo)
	trainingHandler := handlers.NewTrainingHandler(trainingService)
	
	// 创建通用融合handler
	fusionRepo := storage.NewFusionRepository(s.store)
	fusionService := fusionApp.NewUniversalFusionService(fusionRepo)
	fusionHandler := handlers.NewFusionHandler(fusionRepo, fusionService)
	
	// 创建数据管理handler
	dataManagementRepo := storage.NewDataManagementRepository(s.store)
	cleaningService := dataManagementApp.NewDataCleaningService(dataManagementRepo)
	archiveService := dataManagementApp.NewDataArchiveService(dataManagementRepo, nil, "archive")
	dataManagementHandler := handlers.NewDataManagementHandler(dataManagementRepo, cleaningService, archiveService)

	// 创建设备控制handler
	controlRepo := storage.NewControlRepository(s.store)
	controlService := controlApp.NewControlService(controlRepo, actuatorFactory)
	controlHandler := handlers.NewControlHandler(controlRepo, controlService)

	// 创建新的Phase 2 handlers
	var pipelineHandler *handlers.PipelineHandler
	var traceHandler *handlers.TraceHandler
	var metricsHandler *handlers.MetricsHandler

	if s.orchestrator != nil {
		pipelineHandler = handlers.NewPipelineHandler(s.orchestrator)
	}

	if s.tracer != nil && s.traceRepository != nil {
		traceHandler = handlers.NewTraceHandler(s.tracer, s.traceRepository)
		metricsHandler = handlers.NewMetricsHandler(s.traceRepository, s.orchestrator)
	}

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

		// 设备管理路由
		devices := v1.Group("/devices")
		{
			devices.POST("", deviceHandler.CreateDevice)
			devices.GET("", deviceHandler.ListDevices)
			devices.GET("/layout", deviceHandler.GetDeviceLayout)
			devices.GET("/:id", deviceHandler.GetDevice)
			devices.PUT("/:id", deviceHandler.UpdateDevice)
			devices.DELETE("/:id", deviceHandler.DeleteDevice)
			devices.PATCH("/:id/status", deviceHandler.UpdateDeviceStatus)
			devices.POST("/:id/channels", deviceHandler.AddDeviceChannel)
			devices.DELETE("/:id/channels/:channel_id", deviceHandler.RemoveDeviceChannel)
			
			// 设备数据字段路由
			devices.POST("/:device_id/data-fields", deviceDataHandler.CreateDataField)
			devices.GET("/:device_id/data-fields", deviceDataHandler.ListDataFields)
			
			// 设备数据源路由
			devices.POST("/:device_id/data-sources", deviceDataHandler.CreateDataSource)
			devices.GET("/:device_id/data-sources", deviceDataHandler.ListDataSources)
			
			// 设备数据提交和查询路由
			devices.POST("/:device_id/data", deviceDataHandler.SubmitDeviceData)
			devices.GET("/:device_id/fused-data", deviceDataHandler.GetFusedData)
			devices.GET("/:device_id/fused-data/latest", deviceDataHandler.GetLatestFusedData)
			
			// 设备融合配置路由
			devices.GET("/:device_id/fusion-config", deviceDataHandler.GetFusionConfig)
			devices.PUT("/:device_id/fusion-config", deviceDataHandler.UpdateFusionConfig)
		}

		// 告警管理路由
		alerts := v1.Group("/alerts")
		{
			alerts.GET("", alertHandler.ListAlerts)
			alerts.GET("/stats", alertHandler.GetAlertStats)
			alerts.GET("/:id", alertHandler.GetAlert)
			alerts.POST("/:id/acknowledge", alertHandler.AcknowledgeAlert)
			alerts.POST("/:id/resolve", alertHandler.ResolveAlert)
			
			// 告警规则路由
			alerts.POST("/rules", alertHandler.CreateAlertRule)
			alerts.GET("/rules", alertHandler.ListAlertRules)
			alerts.GET("/rules/:id", alertHandler.GetAlertRule)
			alerts.PUT("/rules/:id", alertHandler.UpdateAlertRule)
			alerts.DELETE("/rules/:id", alertHandler.DeleteAlertRule)
		}

		// 规则模板路由
		ruleTemplates := v1.Group("/rule-templates")
		{
			ruleTemplates.GET("", ruleTemplateHandler.ListTemplates)
			ruleTemplates.GET("/:id", ruleTemplateHandler.GetTemplate)
			ruleTemplates.POST("/:id/create-rule", ruleTemplateHandler.CreateRuleFromTemplate)
			ruleTemplates.POST("/:id/test", ruleTemplateHandler.TestTemplate)
			ruleTemplates.POST("/test-rule", ruleTemplateHandler.TestRule)
			ruleTemplates.GET("/test-results", ruleTemplateHandler.ListTestResults)
		}

		// 预测分析路由
		predictions := v1.Group("/predictions")
		{
			predictions.POST("", predictionHandler.CreatePrediction)
			predictions.GET("", predictionHandler.ListPredictions)
			predictions.GET("/:id", predictionHandler.GetPrediction)
			predictions.POST("/forecast", predictionHandler.Forecast)
			predictions.GET("/history", predictionHandler.GetPredictionHistory)
			
			// 预测模型路由
			predictions.POST("/models", predictionHandler.CreateModel)
			predictions.GET("/models", predictionHandler.ListModels)
			predictions.GET("/models/:id", predictionHandler.GetModel)
			predictions.PUT("/models/:id", predictionHandler.UpdateModel)
			predictions.DELETE("/models/:id", predictionHandler.DeleteModel)
			predictions.POST("/models/:id/deploy", predictionHandler.DeployModel)
			predictions.POST("/models/:id/upload", predictionHandler.UploadModelFile)
		}

		// 訓練任務路由
		training := v1.Group("/training")
		{
			training.POST("/tasks", trainingHandler.CreateTask)
			training.GET("/tasks", trainingHandler.ListTasks)
			training.GET("/tasks/:id", trainingHandler.GetTask)
			training.POST("/tasks/:id/start", trainingHandler.StartTask)
			training.POST("/tasks/:id/cancel", trainingHandler.CancelTask)
			training.GET("/tasks/:id/history", trainingHandler.GetTaskHistory)
		}

		// 通用融合路由
		fusion := v1.Group("/fusion")
		{
			// 数据源路由
			fusion.POST("/data-sources", fusionHandler.CreateDataSource)
			fusion.GET("/data-sources", fusionHandler.ListDataSources)
			fusion.POST("/data-sources/:source_id/data", fusionHandler.SubmitData)
			
			// 融合配置路由
			fusion.POST("/configs", fusionHandler.CreateFusionConfig)
			fusion.GET("/configs", fusionHandler.ListFusionConfigs)
			fusion.POST("/configs/:config_id/sources/:source_id", fusionHandler.AddSourceToConfig)
			
			// 融合结果路由
			fusion.GET("/configs/:config_id/results", fusionHandler.GetFusionResults)
			fusion.GET("/configs/:config_id/results/latest", fusionHandler.GetLatestFusionResult)
		}

		// 数据管理路由
		dataMgmt := v1.Group("/data-management")
		{
			// 清洗规则路由
			dataMgmt.POST("/cleaning-rules", dataManagementHandler.CreateCleaningRule)
			dataMgmt.GET("/cleaning-rules", dataManagementHandler.ListCleaningRules)
			
			// 归档策略路由
			dataMgmt.POST("/archive-policies", dataManagementHandler.CreateArchivePolicy)
			dataMgmt.GET("/archive-policies", dataManagementHandler.ListArchivePolicies)
			dataMgmt.POST("/archive-policies/:id/execute", dataManagementHandler.ExecuteArchive)
			dataMgmt.GET("/archive-policies/:id/stats", dataManagementHandler.GetArchiveStats)
			
			// 归档记录路由
			dataMgmt.GET("/archive-records", dataManagementHandler.ListArchiveRecords)
			
			// 生命周期配置路由
			dataMgmt.POST("/lifecycle-configs", dataManagementHandler.CreateLifecycleConfig)
			dataMgmt.GET("/lifecycle-configs/:source_type/:source_id", dataManagementHandler.GetLifecycleConfig)
			
			// 数据清洗路由
			dataMgmt.POST("/clean", dataManagementHandler.CleanData)
		}

		// 设备控制路由
		control := v1.Group("/control")
		{
			// 控制策略路由
			control.POST("/policies", controlHandler.CreateControlPolicy)
			control.GET("/policies", controlHandler.ListControlPolicies)
			control.GET("/policies/:id", controlHandler.GetControlPolicy)
			control.PUT("/policies/:id", controlHandler.UpdateControlPolicy)
			control.DELETE("/policies/:id", controlHandler.DeleteControlPolicy)

			// 控制动作路由
			control.POST("/actions", controlHandler.CreateControlAction)
			control.GET("/actions", controlHandler.ListControlActions)
			control.GET("/actions/:id", controlHandler.GetControlAction)
			control.POST("/actions/:id/confirm", controlHandler.ConfirmControlAction)

			// 审计日志路由
			control.GET("/audit-logs", controlHandler.ListAuditLogs)

			// 控制权限路由
			control.POST("/permissions", controlHandler.CreateControlPermission)
			control.GET("/permissions", controlHandler.ListControlPermissions)
			control.PUT("/permissions/:id", controlHandler.UpdateControlPermission)
			control.DELETE("/permissions/:id", controlHandler.DeleteControlPermission)
		}

		// 事件管理路由
		events := v1.Group("/events")
		{
			events.GET("", eventHandler.GetEvents)
			events.GET("/:event_id", eventHandler.GetEvent)
			events.PUT("/:event_id", eventHandler.UpdateEvent)
			events.GET("/stats", eventHandler.GetEventStats)
		}

		// 管道管理路由 (Phase 2)
		if pipelineHandler != nil {
			pipelines := v1.Group("/pipelines")
			{
				pipelines.GET("", pipelineHandler.GetPipelines)
				pipelines.GET("/:id/topology", pipelineHandler.GetPipelineTopology)
				pipelines.GET("/:id/status", pipelineHandler.GetPipelineStatus)
				pipelines.POST("/:id/start", pipelineHandler.StartPipeline)
				pipelines.POST("/:id/stop", pipelineHandler.StopPipeline)
				if metricsHandler != nil {
					pipelines.GET("/:id/metrics", metricsHandler.GetPipelineMetrics)
				}
			}
		}

		// 追踪数据路由 (Phase 2)
		if traceHandler != nil {
			traces := v1.Group("/traces")
			{
				traces.GET("", traceHandler.GetTraces)
				traces.GET("/samples/:sample_id", traceHandler.GetTraceBySample)
				traces.GET("/stats", traceHandler.GetTraceStats)
			}
		}

		// 性能指标路由 (Phase 2)
		if metricsHandler != nil {
			metrics := v1.Group("/metrics")
			{
				metrics.GET("/system", metricsHandler.GetSystemMetrics)
				metrics.GET("/component", metricsHandler.GetComponentMetrics)
			}
		}

		// 工作流路由
		if s.workflowEngine != nil {
			workflowRepo := storage.NewWorkflowRepository(s.store)
			workflowHandler := handlers.NewWorkflowHandler(s.workflowEngine, workflowRepo)
			
			// 设置模板仓储
			templateRepo := storage.NewWorkflowTemplateRepository(s.store)
			workflowHandler.SetTemplateRepository(templateRepo)
			
			workflows := v1.Group("/workflows")
			{
				workflows.GET("", workflowHandler.ListWorkflows)
				workflows.POST("", workflowHandler.CreateWorkflow)
				workflows.GET("/:id", workflowHandler.GetWorkflow)
				workflows.PUT("/:id", workflowHandler.UpdateWorkflow)
				workflows.DELETE("/:id", workflowHandler.DeleteWorkflow)
				workflows.POST("/:id/execute", workflowHandler.ExecuteWorkflow)
				workflows.GET("/:id/executions", workflowHandler.ListExecutions)
				workflows.GET("/:id/executions/:execution_id", workflowHandler.GetExecution)
				workflows.POST("/:id/executions/:execution_id/cancel", workflowHandler.CancelExecution)
				
				// 工作流模板路由
				workflows.GET("/templates", workflowHandler.ListTemplates)
				workflows.GET("/templates/:id", workflowHandler.GetTemplate)
				workflows.POST("/templates/:id/create", workflowHandler.CreateWorkflowFromTemplate)
			}
		}

		// 认证和用户管理路由 (Phase 3 - RBAC)
		if s.authService != nil && s.authzService != nil && s.userRepository != nil {
			authHandler := handlers.NewAuthHandler(s.authService, s.authzService, s.userRepository)

			// 公开的认证端点 (无需认证)
			auth := v1.Group("/auth")
			{
				auth.POST("/login", authHandler.Login)
			}

			// 需要认证的端点
			authProtected := v1.Group("/auth")
			authProtected.Use(middleware.AuthMiddleware(s.authService))
			{
				authProtected.POST("/logout", authHandler.Logout)
				authProtected.POST("/refresh", authHandler.RefreshToken)
				authProtected.GET("/me", authHandler.GetMe)
				authProtected.POST("/change-password", authHandler.ChangePassword)
			}

			// 用户管理端点 (需要 admin 权限)
			users := v1.Group("/users")
			users.Use(middleware.AuthMiddleware(s.authService))
			users.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceUser, interfaces.ActionRead))
			{
				users.GET("", authHandler.ListUsers)
				users.GET("/:id", authHandler.GetUser)
				users.GET("/:id/roles", authHandler.GetUserRoles)

				// 创建、更新、删除用户需要额外权限
				usersWrite := users.Group("")
				usersWrite.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceUser, interfaces.ActionCreate))
				{
					usersWrite.POST("", authHandler.CreateUser)
				}

				usersUpdate := users.Group("")
				usersUpdate.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceUser, interfaces.ActionUpdate))
				{
					usersUpdate.PUT("/:id", authHandler.UpdateUser)
					usersUpdate.POST("/:id/roles/:role_id", authHandler.AssignRoleToUser)
					usersUpdate.DELETE("/:id/roles/:role_id", authHandler.RemoveRoleFromUser)
				}

				usersDelete := users.Group("")
				usersDelete.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceUser, interfaces.ActionDelete))
				{
					usersDelete.DELETE("/:id", authHandler.DeleteUser)
				}
			}

			// 角色管理端点 (需要 admin 权限)
			roles := v1.Group("/roles")
			roles.Use(middleware.AuthMiddleware(s.authService))
			{
				// 读取角色 - 所有认证用户都可以
				roles.GET("", authHandler.ListRoles)
				roles.GET("/:id", authHandler.GetRole)

				// 创建、更新、删除角色需要 admin 权限
				rolesAdmin := roles.Group("")
				rolesAdmin.Use(middleware.RequireRole(s.authzService, "role-admin"))
				{
					rolesAdmin.POST("", authHandler.CreateRole)
					rolesAdmin.PUT("/:id", authHandler.UpdateRole)
					rolesAdmin.DELETE("/:id", authHandler.DeleteRole)
				}
			}
		}

		// 配置管理路由 (Phase 3)
		if s.configRepository != nil && s.configValidator != nil {
			configHandler := handlers.NewConfigHandler(s.configRepository, s.configValidator)

			configGroup := v1.Group("/config")
			// 应用认证中间件（如果RBAC启用）
			if s.authService != nil {
				configGroup.Use(middleware.AuthMiddleware(s.authService))
			}
			{
				// Rule configuration routes
				rules := configGroup.Group("/rules")
				{
					// 读取规则 - 需要 rule:read 权限
					rulesRead := rules.Group("")
					if s.authzService != nil {
						rulesRead.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceRule, interfaces.ActionRead))
					}
					{
						rulesRead.GET("", configHandler.ListRuleConfigs)
						rulesRead.GET("/:id", configHandler.GetRuleConfig)
					}

					// 创建规则 - 需要 rule:create 权限
					rulesCreate := rules.Group("")
					if s.authzService != nil {
						rulesCreate.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceRule, interfaces.ActionCreate))
					}
					{
						rulesCreate.POST("", configHandler.CreateRuleConfig)
					}

					// 更新规则 - 需要 rule:update 权限
					rulesUpdate := rules.Group("")
					if s.authzService != nil {
						rulesUpdate.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceRule, interfaces.ActionUpdate))
					}
					{
						rulesUpdate.PUT("/:id", configHandler.UpdateRuleConfig)
					}

					// 删除规则 - 需要 rule:delete 权限
					rulesDelete := rules.Group("")
					if s.authzService != nil {
						rulesDelete.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceRule, interfaces.ActionDelete))
					}
					{
						rulesDelete.DELETE("/:id", configHandler.DeleteRuleConfig)
					}
				}

				// Analyzer configuration routes
				analyzers := configGroup.Group("/analyzers")
				{
					// 读取分析器配置
					analyzersRead := analyzers.Group("")
					if s.authzService != nil {
						analyzersRead.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceAnalyzer, interfaces.ActionRead))
					}
					{
						analyzersRead.GET("", configHandler.ListAnalyzerConfigs)
						analyzersRead.GET("/:id", configHandler.GetAnalyzerConfig)
					}

					// 创建分析器配置
					analyzersCreate := analyzers.Group("")
					if s.authzService != nil {
						analyzersCreate.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceAnalyzer, interfaces.ActionCreate))
					}
					{
						analyzersCreate.POST("", configHandler.CreateAnalyzerConfig)
					}

					// 更新分析器配置
					analyzersUpdate := analyzers.Group("")
					if s.authzService != nil {
						analyzersUpdate.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceAnalyzer, interfaces.ActionUpdate))
					}
					{
						analyzersUpdate.PUT("/:id", configHandler.UpdateAnalyzerConfig)
					}

					// 删除分析器配置
					analyzersDelete := analyzers.Group("")
					if s.authzService != nil {
						analyzersDelete.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceAnalyzer, interfaces.ActionDelete))
					}
					{
						analyzersDelete.DELETE("/:id", configHandler.DeleteAnalyzerConfig)
					}
				}

				// Processor configuration routes
				processors := configGroup.Group("/processors")
				{
					// 读取处理器配置
					processorsRead := processors.Group("")
					if s.authzService != nil {
						processorsRead.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceProcessor, interfaces.ActionRead))
					}
					{
						processorsRead.GET("", configHandler.ListProcessorConfigs)
						processorsRead.GET("/:id", configHandler.GetProcessorConfig)
					}

					// 创建处理器配置
					processorsCreate := processors.Group("")
					if s.authzService != nil {
						processorsCreate.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceProcessor, interfaces.ActionCreate))
					}
					{
						processorsCreate.POST("", configHandler.CreateProcessorConfig)
					}

					// 更新处理器配置
					processorsUpdate := processors.Group("")
					if s.authzService != nil {
						processorsUpdate.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceProcessor, interfaces.ActionUpdate))
					}
					{
						processorsUpdate.PUT("/:id", configHandler.UpdateProcessorConfig)
					}

					// 删除处理器配置
					processorsDelete := processors.Group("")
					if s.authzService != nil {
						processorsDelete.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceProcessor, interfaces.ActionDelete))
					}
					{
						processorsDelete.DELETE("/:id", configHandler.DeleteProcessorConfig)
					}
				}

				// Template routes
				templates := configGroup.Group("/templates")
				{
					// 读取模板 - 所有认证用户都可以
					templates.GET("", configHandler.ListTemplates)
					templates.GET("/:id", configHandler.GetTemplate)

					// 创建、更新、删除模板需要 config:update 权限
					templatesWrite := templates.Group("")
					if s.authzService != nil {
						templatesWrite.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceConfig, interfaces.ActionUpdate))
					}
					{
						templatesWrite.POST("", configHandler.CreateTemplate)
						templatesWrite.PUT("/:id", configHandler.UpdateTemplate)
						templatesWrite.DELETE("/:id", configHandler.DeleteTemplate)
					}
				}

				// History routes - 需要 config:read 权限
				historyGroup := configGroup.Group("/history")
				if s.authzService != nil {
					historyGroup.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceConfig, interfaces.ActionRead))
				}
				{
					historyGroup.GET("/:config_type/:config_id", configHandler.GetConfigHistory)
				}

				// Import/Export routes
				exportGroup := configGroup.Group("/export")
				if s.authzService != nil {
					exportGroup.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceConfig, interfaces.ActionExport))
				}
				{
					exportGroup.GET("/:config_type", configHandler.ExportConfigs)
				}

				importGroup := configGroup.Group("/import")
				if s.authzService != nil {
					importGroup.Use(middleware.RequirePermission(s.authzService, interfaces.ResourceConfig, interfaces.ActionImport))
				}
				{
					importGroup.POST("/:config_type", configHandler.ImportConfigs)
				}

				// Validation route - 所有认证用户都可以验证配置
				configGroup.POST("/validate/:config_type", configHandler.ValidateConfig)
			}
		}
	}
}

// handleHealth 健康检查处理器
func (s *Server) handleHealth(c *gin.Context) {
	// 检查数据库连接
	dbHealthy := true
	if s.store != nil {
		if err := s.store.Ping(); err != nil {
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

	// 启动数据流WebSocket Hub
	if s.dataHub != nil {
		go s.dataHub.Run()
		s.log.Info("数据流WebSocket Hub已启动")
	}

	// 连接Tracer到WebSocket Hub
	if s.tracer != nil && s.wsHub != nil {
		go s.bridgeTracerToWebSocket()
		s.log.Info("Tracer已连接到WebSocket Hub")
	}

	// 启动数据流桥接服务
	if s.streamBridge != nil {
		go s.streamBridge.Start(s.ctx)
		s.log.Info("数据流桥接服务已启动")
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
