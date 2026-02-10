// 生产环境入口：组装依赖并启动 API 服务
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go_ProFiBus/api"
	"go_ProFiBus/internal/application/orchestrator"
	"go_ProFiBus/internal/application/tracing"
	authDomain "go_ProFiBus/internal/domain/auth"
	"go_ProFiBus/internal/infrastructure/storage"
	"go_ProFiBus/logger"
	rootStorage "go_ProFiBus/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	log := logger.GetLogger()
	log.Info("go_ProFiBus 服务启动中...")

	// 数据库配置（环境变量优先）
	dbConfig := rootStorage.GetDefaultPostgresConfig()
	if v := os.Getenv("DB_HOST"); v != "" {
		dbConfig.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			dbConfig.Port = p
		}
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		dbConfig.Database = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		dbConfig.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		dbConfig.Password = v
	}
	if v := os.Getenv("DB_MAX_CONNECTIONS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			dbConfig.MaxConnections = int32(n)
		}
	}

	store, err := rootStorage.NewPostgresStore(dbConfig)
	if err != nil {
		log.Error("创建数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Ping(); err != nil {
		log.Error("数据库健康检查失败: %v", err)
		os.Exit(1)
	}
	log.Info("数据库连接成功")

	// 基础设施：仓储与编排器
	traceRepo := storage.NewTraceRepository(store)
	configRepo := storage.NewConfigRepository(store)
	userRepo := storage.NewUserRepository(store)
	tracer := tracing.NewTracer(traceRepo, 100)
	authService := authDomain.NewAuthService(userRepo)
	authzService := authDomain.NewAuthorizationService(userRepo)
	orch := orchestrator.NewOrchestrator()

	// API 服务器配置
	apiPort := 8080
	if v := os.Getenv("API_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			apiPort = p
		}
	}
	apiConfig := &api.ServerConfig{
		Host:         getEnv("API_HOST", "0.0.0.0"),
		Port:         apiPort,
		Mode:         getEnv("API_MODE", gin.ReleaseMode),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		EnableCORS:   getEnv("API_CORS_ENABLED", "true") == "true",
		AllowOrigins: []string{getEnv("API_CORS_ORIGINS", "*")},
	}

	server, err := api.NewServer(
		apiConfig,
		store,
		tracer,
		traceRepo,
		configRepo,
		userRepo,
		authService,
		authzService,
		orch,
	)
	if err != nil {
		log.Error("创建 API 服务器失败: %v", err)
		os.Exit(1)
	}

	if err := server.Start(); err != nil {
		log.Error("启动 API 服务器失败: %v", err)
		os.Exit(1)
	}
	log.Info("API 服务已启动: http://%s:%d", apiConfig.Host, apiConfig.Port)

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Info("收到退出信号，正在关闭...")

	if err := server.Stop(); err != nil {
		log.Error("停止服务失败: %v", err)
	}
	log.Info("服务已停止")
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// 供 -ldflags 注入的版本信息（构建时设置）
var (
	Version   = "dev"
	BuildTime string
	GitCommit string
)

func init() {
	if BuildTime != "" || GitCommit != "" {
		logger.GetLogger().Info("版本: %s 构建: %s 提交: %s", Version, BuildTime, GitCommit)
	}
	_ = fmt.Sprint() // 避免未使用
}
