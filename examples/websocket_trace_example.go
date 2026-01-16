package main

import (
	"context"
	"fmt"
	"go_ProFiBus/api"
	"go_ProFiBus/internal/application/tracing"
	"go_ProFiBus/internal/infrastructure/storage"
	"go_ProFiBus/logger"
	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/storage"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 这个示例展示了如何使用WebSocket实时推送追踪事件

func main() {
	// 1. 初始化日志
	log := logger.GetLogger()
	log.Info("启动WebSocket追踪示例...")

	// 2. 连接数据库
	dbConfig := &storage2.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "profibus",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		MaxLifetime:  5 * time.Minute,
	}

	store, err := storage2.NewPostgresStore(dbConfig)
	if err != nil {
		log.Error("Failed to connect to database: %v", err)
		// 在没有数据库的情况下，使用nil store（仅演示）
		store = nil
	} else {
		defer store.Close()
	}

	// 3. 创建TraceRepository
	var traceRepo interfaces.TraceRepository
	if store != nil {
		traceRepo = storage.NewTraceRepository(store)
	}

	// 4. 创建Tracer（100事件缓冲）
	tracer := tracing.NewTracer(traceRepo, 100)
	defer tracer.Close()

	// 5. 创建API服务器（包含WebSocket支持）
	serverConfig := api.DefaultServerConfig()
	serverConfig.Port = 8080

	server, err := api.NewServer(serverConfig, store, tracer)
	if err != nil {
		log.Error("Failed to create server: %v", err)
		return
	}

	// 6. 启动服务器
	if err := server.Start(); err != nil {
		log.Error("Failed to start server: %v", err)
		return
	}

	log.Info("服务器已启动:")
	log.Info("  - REST API: http://localhost:8080")
	log.Info("  - WebSocket: ws://localhost:8080/ws/trace")
	log.Info("  - Health Check: http://localhost:8080/health")

	// 7. 模拟生成追踪事件
	go simulateTraceEvents(tracer)

	// 8. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在关闭服务器...")
	if err := server.Stop(); err != nil {
		log.Error("Failed to stop server: %v", err)
	}

	log.Info("服务器已关闭")
}

// simulateTraceEvents 模拟生成追踪事件
func simulateTraceEvents(tracer interfaces.Tracer) {
	ctx := context.Background()
	pipelineID := "pipeline-demo-001"
	sampleID := "sample-001"

	components := []struct {
		Type string
		ID   string
		Name string
	}{
		{"source", "source-001", "ModbusSource"},
		{"processor", "proc-001", "DataValidator"},
		{"processor", "proc-002", "DataTransformer"},
		{"analyzer", "analyzer-001", "ThresholdAnalyzer"},
		{"sink", "sink-001", "DatabaseSink"},
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for i := 0; ; i++ {
		<-ticker.C

		// 为每个样本生成完整的追踪链
		newSampleID := fmt.Sprintf("sample-%03d", i)

		for _, comp := range components {
			// Enter事件
			enterEvent := &interfaces.TraceEvent{
				PipelineID:    pipelineID,
				SampleID:      newSampleID,
				ComponentType: comp.Type,
				ComponentID:   comp.ID,
				ComponentName: comp.Name,
				Action:        "enter",
				Timestamp:     time.Now(),
				Status:        "success",
				Metadata: map[string]interface{}{
					"iteration": i,
				},
			}
			tracer.TraceDataFlow(ctx, enterEvent)

			// 模拟处理时间
			time.Sleep(100 * time.Millisecond)

			// Process事件
			processEvent := &interfaces.TraceEvent{
				PipelineID:    pipelineID,
				SampleID:      newSampleID,
				ComponentType: comp.Type,
				ComponentID:   comp.ID,
				ComponentName: comp.Name,
				Action:        "process",
				Timestamp:     time.Now(),
				Duration:      100 * time.Millisecond,
				Status:        "success",
				DataSnapshot: map[string]interface{}{
					"value": 42.5 + float64(i),
					"unit":  "°C",
				},
			}
			tracer.TraceDataFlow(ctx, processEvent)

			// Exit事件
			exitEvent := &interfaces.TraceEvent{
				PipelineID:    pipelineID,
				SampleID:      newSampleID,
				ComponentType: comp.Type,
				ComponentID:   comp.ID,
				ComponentName: comp.Name,
				Action:        "exit",
				Timestamp:     time.Now(),
				Status:        "success",
			}
			tracer.TraceDataFlow(ctx, exitEvent)
		}
	}
}
