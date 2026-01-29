package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_ProFiBus/fusion"
	"go_ProFiBus/inference"
	"go_ProFiBus/internal/application/orchestrator"
	"go_ProFiBus/internal/application/processor"
	"go_ProFiBus/internal/infrastructure/analyzer"
	"go_ProFiBus/internal/infrastructure/collector"
	"go_ProFiBus/pkg/interfaces"
)

// 本示例展示：多协议数据融合 + AI检测
//
// 数据流：
// [MQTT传感器] [OPC-UA设备] [串口设备]
//       ↓           ↓           ↓
//   [数据源适配器] 统一为 DataSample
//       ↓           ↓           ↓
//        [融合处理器] ← 多源数据融合
//              ↓
//      [特征提取器] ← 提取ML特征
//              ↓
//        [ML分析器] ← AI异常检测
//              ↓
//        [结果输出]

func main() {
	fmt.Println("=== 多协议数据融合 + AI检测示例 ===\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 创建MQTT数据源
	mqttConfig := &collector.MQTTSourceConfig{
		SourceID: "mqtt-temp-sensor",
		Name:     "MQTT温度传感器",
		Broker:   "tcp://localhost:1883",
		Topics:   []string{"factory/sensors/temperature/zone1"},
		QoS:      1,
		BufferSize: 100,
	}

	mqttSource, err := collector.NewMQTTSource(mqttConfig)
	if err != nil {
		log.Printf("警告: 创建MQTT数据源失败: %v (将跳过MQTT)", err)
	} else {
		err = mqttSource.Start(ctx)
		if err != nil {
			log.Printf("警告: 启动MQTT数据源失败: %v (将跳过MQTT)", err)
			mqttSource = nil
		} else {
			fmt.Println("✓ MQTT数据源已启动")
		}
	}

	// 2. 创建OPC-UA数据源
	opcuaConfig := &collector.OPCUASourceConfig{
		SourceID:         "opcua-temp-sensor",
		Name:             "OPC-UA温度传感器",
		Endpoint:         "opc.tcp://localhost:4840",
		SecurityPolicy:   "None",
		SecurityMode:     "None",
		NodeIDs:          []string{"ns=2;s=Temperature.Zone1"},
		SamplingInterval: 1 * time.Second,
		UseSubscription:  true,
		BufferSize:       100,
	}

	opcuaSource, err := collector.NewOPCUASource(opcuaConfig)
	if err != nil {
		log.Printf("警告: 创建OPC-UA数据源失败: %v (将跳过OPC-UA)", err)
	} else {
		err = opcuaSource.Start(ctx)
		if err != nil {
			log.Printf("警告: 启动OPC-UA数据源失败: %v (将跳过OPC-UA)", err)
			opcuaSource = nil
		} else {
			fmt.Println("✓ OPC-UA数据源已启动")
		}
	}

	// 检查是否至少有一个数据源可用
	if mqttSource == nil && opcuaSource == nil {
		log.Println("\n提示: 没有可用的数据源。请确保:")
		log.Println("  1. MQTT服务器运行在 localhost:1883")
		log.Println("  2. OPC-UA服务器运行在 localhost:4840")
		log.Println("\n将使用模拟数据源继续演示...\n")
	}

	// 3. 创建融合处理器
	fusionProcessor := processor.NewFusionProcessor("multi-protocol-fusion", fusion.StrategyWeighted)

	// 设置数据源权重
	fusionProcessor.SetSourceWeight("mqtt-temp-sensor", 0.4)   // MQTT权重40%
	fusionProcessor.SetSourceWeight("opcua-temp-sensor", 0.4)  // OPC-UA权重40%
	fusionProcessor.SetSourceWeight("mock-temp-sensor", 0.2)   // 模拟数据权重20%

	fmt.Println("✓ 创建融合处理器 (加权策略: 40%, 40%, 20%)")

	// 4. 创建特征提取器
	featureExtractor := processor.NewFeatureExtractor("ml-feature-extractor")
	featureExtractor.AddFeature("numeric_mean", nil)
	featureExtractor.AddFeature("numeric_stddev", nil)
	featureExtractor.AddFeature("quality", nil)
	featureExtractor.AddFeature("confidence", nil)

	fmt.Println("✓ 创建特征提取器")

	// 5. 创建ML分析器
	mlAnalyzer := analyzer.NewMLAnalyzer("anomaly-detector", "multi-protocol-nn-model")

	err = mlAnalyzer.Configure(map[string]interface{}{
		"model_name":            "multi-protocol-nn-model",
		"model_type":            "neural_network",
		"anomaly_threshold":     0.7,
		"confidence_min":        0.5,
		"use_feature_extractor": false,
	})
	if err != nil {
		log.Printf("警告: ML分析器配置失败: %v", err)
	}

	// 创建并训练模型
	model := inference.NewNeuralNetworkModel([]int{10, 20, 10, 2})
	mlAnalyzer.UpdateModel(model)

	fmt.Println("✓ 创建ML分析器")

	// 6. 创建Pipeline
	pipeline, err := orchestrator.NewPipelineBuilder("multi-protocol-pipeline").
		WithProcessor(fusionProcessor).
		WithProcessor(featureExtractor).
		WithAnalyzer(mlAnalyzer).
		Build()

	if err != nil {
		log.Fatalf("Pipeline创建失败: %v", err)
	}

	fmt.Println("✓ 创建Pipeline: 融合 → 特征提取 → AI检测\n")

	// 7. 启动数据收集和处理
	fmt.Println("--- 开始多协议数据融合处理 ---\n")

	// 启动数据收集协程
	if mqttSource != nil {
		go collectFromSource(ctx, mqttSource, pipeline, fusionProcessor)
	}
	if opcuaSource != nil {
		go collectFromSource(ctx, opcuaSource, pipeline, fusionProcessor)
	}

	// 如果没有真实数据源，使用模拟数据
	if mqttSource == nil && opcuaSource == nil {
		go generateMockData(ctx, pipeline, fusionProcessor)
	}

	// 8. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("按 Ctrl+C 退出...\n")

	<-sigChan

	// 9. 清理资源
	fmt.Println("\n\n--- 正在关闭 ---")
	cancel()

	if mqttSource != nil {
		mqttSource.Stop()
	}
	if opcuaSource != nil {
		opcuaSource.Stop()
	}

	// 显示统计
	fmt.Println("\n--- 分析统计 ---")
	stats := mlAnalyzer.GetStats()
	fmt.Printf("总样本数: %d\n", stats.TotalSamples)
	fmt.Printf("检测异常: %d\n", stats.AnomaliesDetected)
	fmt.Printf("异常率: %.2f%%\n", mlAnalyzer.GetAnomalyRate()*100)
	fmt.Printf("平均推理时间: %v\n", stats.AvgInferenceTime)

	fmt.Println("\n✓ 示例完成!")
}

// collectFromSource 从数据源收集数据
func collectFromSource(
	ctx context.Context,
	source interfaces.DataSource,
	pipeline *orchestrator.Pipeline,
	fusionProcessor *processor.FusionProcessor,
) {
	dataChan := source.GetData()

	for {
		select {
		case <-ctx.Done():
			return
		case sample, ok := <-dataChan:
			if !ok {
				return
			}

			// 处理样本
			processSample(ctx, sample, pipeline, fusionProcessor)
		}
	}
}

// processSample 处理单个样本
func processSample(
	ctx context.Context,
	sample interfaces.DataSample,
	pipeline *orchestrator.Pipeline,
	fusionProcessor *processor.FusionProcessor,
) {
	fmt.Printf("[%s] 收到数据: 源=%s\n",
		sample.GetTimestamp().Format("15:04:05"),
		sample.GetSourceID())

	// 通过融合处理器
	fusedSample, err := fusionProcessor.Process(ctx, sample)
	if err != nil {
		log.Printf("融合处理失败: %v", err)
		return
	}

	// 如果只有一个数据源，可能不会触发融合
	metadata := fusedSample.GetMetadata()
	if sourceCount, ok := metadata["source_count"].(int); ok && sourceCount > 1 {
		fmt.Printf("  → 融合 %d 个数据源\n", sourceCount)
	}

	// 执行完整的Pipeline
	results, err := pipeline.Execute(ctx, fusedSample)
	if err != nil {
		log.Printf("Pipeline执行失败: %v", err)
		return
	}

	// 显示分析结果
	for _, result := range results {
		fmt.Printf("  → AI检测: 分数=%.3f, 严重度=%d, %s\n",
			result.Score,
			result.Severity,
			result.Message)

		if isAnomaly, ok := result.Details["is_anomaly"].(bool); ok && isAnomaly {
			fmt.Printf("     ⚠️  检测到异常!\n")
		}
	}

	fmt.Println()
}

// generateMockData 生成模拟数据
func generateMockData(
	ctx context.Context,
	pipeline *orchestrator.Pipeline,
	fusionProcessor *processor.FusionProcessor,
) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	mockSources := []string{"mock-temp-sensor-1", "mock-temp-sensor-2", "mock-temp-sensor-3"}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 为每个模拟源生成数据
			for _, sourceID := range mockSources {
				sample := createMockSample(sourceID)
				processSample(ctx, sample, pipeline, fusionProcessor)
			}
		}
	}
}

// createMockSample 创建模拟样本
func createMockSample(sourceID string) interfaces.DataSample {
	// 生成模拟温度数据
	baseTemp := 85.0
	variation := (float64(time.Now().UnixNano()%100) - 50) / 10.0
	temperature := baseTemp + variation

	data := map[string]interface{}{
		"temperature": temperature,
		"pressure":    temperature * 0.5,
		"vibration":   temperature * 0.3,
	}

	// 使用内部实现
	sample := &mockDataSample{
		timestamp: time.Now(),
		sourceID:  sourceID,
		data:      data,
		quality:   0.95,
		metadata:  make(map[string]interface{}),
	}

	return sample
}

// mockDataSample 模拟数据样本实现
type mockDataSample struct {
	timestamp time.Time
	sourceID  string
	data      map[string]interface{}
	quality   float64
	metadata  map[string]interface{}
}

func (s *mockDataSample) GetTimestamp() time.Time                { return s.timestamp }
func (s *mockDataSample) GetSourceID() string                    { return s.sourceID }
func (s *mockDataSample) GetData() map[string]interface{}        { return s.data }
func (s *mockDataSample) GetQuality() float64                    { return s.quality }
func (s *mockDataSample) GetMetadata() map[string]interface{}    { return s.metadata }
