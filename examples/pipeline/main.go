package main

import (
	"context"
	"fmt"
	"go_ProFiBus/anomaly"
	"go_ProFiBus/collector"
	"go_ProFiBus/internal/application/orchestrator"
	"go_ProFiBus/internal/infrastructure/analyzer"
	infracollector "go_ProFiBus/internal/infrastructure/collector"
	"go_ProFiBus/logger"
	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/serial"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ConsoleSink 简单的控制台输出 Sink
type ConsoleSink struct {
	name string
}

func NewConsoleSink(name string) *ConsoleSink {
	return &ConsoleSink{name: name}
}

func (s *ConsoleSink) Write(ctx context.Context, data interface{}) error {
	sample, ok := data.(interfaces.DataSample)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	fmt.Printf("[%s] Source: %s, Time: %s, Quality: %.2f, Data: %+v\n",
		s.name,
		sample.GetSourceID(),
		sample.GetTimestamp().Format("15:04:05"),
		sample.GetQuality(),
		sample.GetData(),
	)
	return nil
}

func (s *ConsoleSink) WriteBatch(ctx context.Context, data []interface{}) error {
	for _, d := range data {
		if err := s.Write(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsoleSink) Flush(ctx context.Context) error {
	return nil
}

func (s *ConsoleSink) Close() error {
	return nil
}

func (s *ConsoleSink) GetName() string {
	return s.name
}

// SimpleProcessor 简单的数据处理器示例
type SimpleProcessor struct {
	name   string
	factor float64
}

func NewSimpleProcessor(name string, factor float64) *SimpleProcessor {
	return &SimpleProcessor{
		name:   name,
		factor: factor,
	}
}

func (p *SimpleProcessor) Process(ctx context.Context, input interfaces.DataSample) (interfaces.DataSample, error) {
	// 获取原始数据
	data := input.GetData()

	// 如果数据中有 "value" 字段，将其乘以 factor
	if val, ok := data["value"].(float64); ok {
		data["value"] = val * p.factor
		data["processed_by"] = p.name
	}

	return input, nil
}

func (p *SimpleProcessor) GetName() string {
	return p.name
}

func (p *SimpleProcessor) GetConfig() interfaces.ProcessorConfig {
	return interfaces.ProcessorConfig{
		"name":   p.name,
		"factor": p.factor,
	}
}

func (p *SimpleProcessor) Initialize(config interfaces.ProcessorConfig) error {
	return nil
}

func (p *SimpleProcessor) Close() error {
	return nil
}

func main() {
	fmt.Println("=== Pipeline 示例程序 ===")
	fmt.Println("展示如何使用新的架构框架构建数据处理管道")
	fmt.Println()

	// 初始化日志
	logger.Init(logger.INFO)

	// 1. 创建数据源（使用现有的 Collector）
	fmt.Println("1. 创建数据源...")

	// 注意：这里使用模拟的串口，实际使用时需要真实的串口
	mockPort := &serial.MockSerialPort{} // 需要实现这个 mock

	collectorConfig := &collector.CollectorConfig{
		SourceID:     "sensor-001",
		Port:         mockPort,
		SampleRate:   1 * time.Second,
		BufferSize:   100,
		EnableCache:  true,
		CacheSize:    50,
		RetryCount:   3,
		RetryDelay:   100 * time.Millisecond,
	}

	coll := collector.NewCollector(collectorConfig)
	dataSource := infracollector.NewDataSourceAdapter("sensor-001", "温度传感器", coll)

	// 2. 创建处理器
	fmt.Println("2. 创建数据处理器...")
	processor := NewSimpleProcessor("temperature-converter", 1.8) // 摄氏度转华氏度示例

	// 3. 创建分析器（使用现有的规则引擎）
	fmt.Println("3. 创建分析器...")
	ruleEngine := anomaly.NewRuleEngine()

	// 添加温度阈值规则
	tempRule := anomaly.NewThresholdRule(
		"rule-temp-high",
		"高温警告",
		"temperature",
		">",
		80.0,
		anomaly.SeverityWarning,
	)
	ruleEngine.AddRule(tempRule)

	ruleAnalyzer := analyzer.NewRuleEngineAnalyzer("rule-engine", ruleEngine)

	// 4. 创建输出 Sink
	fmt.Println("4. 创建数据输出...")
	consoleSink := NewConsoleSink("console")

	// 5. 使用 Builder 构建 Pipeline
	fmt.Println("5. 构建数据处理管道...")
	pipeline, err := orchestrator.NewPipelineBuilder("main-pipeline").
		WithSource(dataSource).
		WithProcessor(processor).
		WithAnalyzer(ruleAnalyzer).
		WithSink(consoleSink).
		Build()

	if err != nil {
		log.Fatalf("构建管道失败: %v", err)
	}

	// 6. 创建 Orchestrator 并添加 Pipeline
	fmt.Println("6. 创建编排器...")
	orch := orchestrator.NewOrchestrator()
	if err := orch.AddPipeline(pipeline); err != nil {
		log.Fatalf("添加管道失败: %v", err)
	}

	// 7. 启动错误监控
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go orch.MonitorErrors(ctx, func(pipelineName string, err error) {
		fmt.Printf("[错误] Pipeline %s: %v\n", pipelineName, err)
	})

	// 8. 启动所有管道
	fmt.Println("7. 启动数据管道...")
	if err := orch.StartAll(); err != nil {
		log.Fatalf("启动管道失败: %v", err)
	}

	fmt.Println("管道已启动，正在处理数据...")
	fmt.Println("按 Ctrl+C 停止")
	fmt.Println()

	// 9. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 定期输出管道状态
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\n收到停止信号，正在关闭...")
			goto shutdown
		case <-ticker.C:
			statuses := orch.GetStatus()
			for _, status := range statuses {
				fmt.Printf("\n[状态] Pipeline: %s\n", status.Name)
				fmt.Printf("  运行中: %v\n", status.Running)
				fmt.Printf("  处理器数: %d\n", status.ProcessorCount)
				fmt.Printf("  分析器数: %d\n", status.AnalyzerCount)
				fmt.Printf("  输出数: %d\n", status.SinkCount)
				fmt.Printf("  数据源状态: 运行=%v, 已连接=%v, 样本数=%d\n",
					status.SourceStatus.Running,
					status.SourceStatus.Connected,
					status.SourceStatus.SamplesCollected,
				)
			}
		}
	}

shutdown:
	// 10. 优雅关闭
	fmt.Println("正在停止所有管道...")
	if err := orch.Shutdown(); err != nil {
		log.Printf("关闭管道失败: %v", err)
	}

	fmt.Println("程序已退出")
}
