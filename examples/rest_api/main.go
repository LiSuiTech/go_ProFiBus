package main

import (
	"context"
	"fmt"
	"go_ProFiBus/api"
	"go_ProFiBus/collector"
	"go_ProFiBus/logger"
	"go_ProFiBus/serial"
	"go_ProFiBus/storage"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

/*
REST API 完整示例

功能：
1. 启动PostgreSQL/TimescaleDB数据库连接
2. 启动数据采集（模拟或真实串口）
3. 启动REST API服务器
4. 提供完整的传感器数据查询、事件管理、规则管理API

使用方法：
1. 确保PostgreSQL/TimescaleDB已安装并运行
2. 创建数据库：createdb profibus
3. 运行迁移脚本：psql profibus < migrations/001_initial_schema.sql
4. 配置数据库连接（环境变量或代码中修改）
5. 运行：go run examples/rest_api/main.go

API端点：
- GET /health - 健康检查
- GET /ping - Ping测试
- GET /api/v1/sensors/:sensor_id/readings - 查询传感器历史数据
- POST /api/v1/sensors/readings - 批量写入传感器数据
- GET /api/v1/sensors/:sensor_id/aggregation - 数据聚合查询
- GET /api/v1/events - 查询事件列表
- GET /api/v1/events/:event_id - 查询事件详情
- PUT /api/v1/events/:event_id - 更新事件状态
- GET /api/v1/events/stats - 事件统计
- GET /api/v1/rules - 查询规则列表
- POST /api/v1/rules - 创建规则
- PUT /api/v1/rules/:rule_id - 更新规则
- DELETE /api/v1/rules/:rule_id - 删除规则
*/

func main() {
	// 初始化日志
	log := logger.GetLogger()
	log.Info("========================================")
	log.Info("REST API 完整示例启动")
	log.Info("========================================")

	// 从环境变量读取数据库配置
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "profibus")

	// 数据库连接字符串
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	log.Info("连接数据库: %s@%s:%s/%s", dbUser, dbHost, dbPort, dbName)

	// 创建PostgreSQL存储
	storeConfig := &storage.PostgresConfig{
		ConnString:     connString,
		MaxConnections: 20,
		MinConnections: 5,
	}

	store, err := storage.NewPostgresStore(storeConfig)
	if err != nil {
		log.Error("创建数据库连接失败: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	// 测试数据库连接
	if err := store.Health(); err != nil {
		log.Error("数据库健康检查失败: %v", err)
		log.Error("请确保：")
		log.Error("1. PostgreSQL/TimescaleDB已安装并运行")
		log.Error("2. 数据库已创建：createdb %s", dbName)
		log.Error("3. 迁移脚本已执行：psql %s < migrations/001_initial_schema.sql", dbName)
		os.Exit(1)
	}
	log.Info("数据库连接成功")

	// 创建REST API服务器配置
	apiConfig := &api.ServerConfig{
		Host:         getEnv("API_HOST", "0.0.0.0"),
		Port:         getEnvInt("API_PORT", 8080),
		Mode:         getEnv("GIN_MODE", gin.ReleaseMode),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		EnableCORS:   true,
		AllowOrigins: []string{"*"}, // 生产环境应配置具体域名
	}

	// 创建API服务器
	apiServer, err := api.NewServer(apiConfig, store)
	if err != nil {
		log.Error("创建API服务器失败: %v", err)
		os.Exit(1)
	}

	// 启动API服务器
	if err := apiServer.Start(); err != nil {
		log.Error("启动API服务器失败: %v", err)
		os.Exit(1)
	}
	log.Info("REST API服务器已启动: http://%s:%d", apiConfig.Host, apiConfig.Port)
	log.Info("API文档：")
	log.Info("  健康检查: GET http://localhost:%d/health", apiConfig.Port)
	log.Info("  Ping测试: GET http://localhost:%d/ping", apiConfig.Port)
	log.Info("  传感器数据: GET http://localhost:%d/api/v1/sensors/{id}/readings?start=...&end=...", apiConfig.Port)
	log.Info("  事件列表: GET http://localhost:%d/api/v1/events", apiConfig.Port)
	log.Info("  规则列表: GET http://localhost:%d/api/v1/rules", apiConfig.Port)

	// 可选：启动数据采集（取消注释以启用）
	// startDataCollection(store, log)

	log.Info("========================================")
	log.Info("服务已启动，按 Ctrl+C 停止")
	log.Info("========================================")

	// 等待终止信号
	waitForShutdown(apiServer, log)
}

// startDataCollection 启动数据采集（示例）
func startDataCollection(store *storage.PostgresStore, log *logger.Logger) {
	log.Info("启动数据采集...")

	// 创建模拟采集器（用于演示）
	collectorConfig := &collector.Config{
		SampleInterval: 1 * time.Second,
		BufferSize:     100,
		Timeout:        5 * time.Second,
	}

	// 创建模拟串口（实际使用时替换为真实串口）
	mockPort := &serial.MockSerialPort{
		Name:     "/dev/ttyUSB0",
		BaudRate: 9600,
	}

	dataCollector := collector.NewCollector("sensor-1", mockPort, collectorConfig)

	// 启动采集
	go func() {
		if err := dataCollector.Start(); err != nil {
			log.Error("启动采集失败: %v", err)
		}
	}()

	// 读取并存储数据
	go func() {
		samples := make([]*collector.DataSample, 0, 100)
		ticker := time.NewTicker(10 * time.Second) // 每10秒批量写入
		defer ticker.Stop()

		for {
			select {
			case sample := <-dataCollector.GetChannel():
				samples = append(samples, sample)

				// 批量写入
				if len(samples) >= 100 {
					if err := store.WriteSensorReadings(samples); err != nil {
						log.Error("写入传感器数据失败: %v", err)
					} else {
						log.Info("写入 %d 条传感器数据", len(samples))
					}
					samples = samples[:0]
				}

			case <-ticker.C:
				// 定时刷新缓冲区
				if len(samples) > 0 {
					if err := store.WriteSensorReadings(samples); err != nil {
						log.Error("写入传感器数据失败: %v", err)
					} else {
						log.Info("写入 %d 条传感器数据", len(samples))
					}
					samples = samples[:0]
				}
			}
		}
	}()

	log.Info("数据采集已启动")
}

// waitForShutdown 等待关闭信号并优雅退出
func waitForShutdown(apiServer *api.Server, log *logger.Logger) {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 等待信号
	sig := <-sigChan
	log.Info("收到信号: %v，正在关闭...", sig)

	// 创建关闭超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 停止API服务器
	if err := apiServer.Stop(); err != nil {
		log.Error("停止API服务器失败: %v", err)
	}

	// 等待所有goroutine完成或超时
	<-ctx.Done()

	log.Info("服务已停止")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整型环境变量，如果不存在或无效则返回默认值
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}
