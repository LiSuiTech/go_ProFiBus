package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go_ProFiBus/fusion"
	"go_ProFiBus/inference"
	"go_ProFiBus/internal/application/orchestrator"
	"go_ProFiBus/internal/application/processor"
	"go_ProFiBus/internal/infrastructure/analyzer"
	"go_ProFiBus/pkg/interfaces"
)

// 本示例展示：AI模型检测 + 多数据源融合的完整工作流
//
// 数据流：
// [数据源1]  [数据源2]  [数据源N]
//      ↓         ↓         ↓
//        [FusionProcessor]  ← 多数据融合 (加权/卡尔曼/贝叶斯)
//                ↓
//        [FeatureExtractor] ← 特征提取 (统计/时序/质量)
//                ↓
//          [MLAnalyzer]     ← AI模型检测 (神经网络)
//                ↓
//           [结果输出]

func main() {
	fmt.Println("=== AI模型检测 + 多数据融合示例 ===\n")

	// 1. 创建多个模拟数据源
	source1 := NewMockDataSource("sensor-1", 100.0, 10.0)
	source2 := NewMockDataSource("sensor-2", 105.0, 8.0)
	source3 := NewMockDataSource("sensor-3", 98.0, 12.0)

	fmt.Println("✓ 创建了3个模拟数据源")

	// 2. 创建融合处理器（使用加权融合策略）
	fusionProcessor := processor.NewFusionProcessor("multi-sensor-fusion", fusion.StrategyWeighted)
	fusionProcessor.SetSourceWeight("sensor-1", 0.5)  // 传感器1权重50%
	fusionProcessor.SetSourceWeight("sensor-2", 0.3)  // 传感器2权重30%
	fusionProcessor.SetSourceWeight("sensor-3", 0.2)  // 传感器3权重20%

	fmt.Println("✓ 创建融合处理器 (加权策略: 50%, 30%, 20%)")

	// 3. 创建特征提取器
	featureExtractor := processor.NewFeatureExtractor("ml-feature-extractor")
	// 添加多种特征
	featureExtractor.AddFeature("numeric_mean", nil)
	featureExtractor.AddFeature("numeric_stddev", nil)
	featureExtractor.AddFeature("quality", nil)
	featureExtractor.AddFeature("confidence", nil)

	fmt.Println("✓ 创建特征提取器 (10种特征)")

	// 4. 创建ML分析器（使用神经网络）
	mlAnalyzer := analyzer.NewMLAnalyzer("anomaly-detector", "anomaly-nn-model")

	// 配置ML分析器
	err := mlAnalyzer.Configure(map[string]interface{}{
		"model_name":          "anomaly-nn-model",
		"model_type":          "neural_network",
		"anomaly_threshold":   0.7,
		"confidence_min":      0.5,
		"use_feature_extractor": false, // 已在Pipeline中使用
	})
	if err != nil {
		log.Printf("警告: ML分析器配置失败: %v (将使用默认配置)", err)
	}

	// 创建并训练一个简单的神经网络模型
	model := createAndTrainModel()
	mlAnalyzer.UpdateModel(model)

	fmt.Println("✓ 创建ML分析器 (神经网络模型)")

	// 5. 创建Pipeline
	pipeline, err := orchestrator.NewPipelineBuilder("ml-fusion-pipeline").
		WithProcessor(fusionProcessor).
		WithProcessor(featureExtractor).
		WithAnalyzer(mlAnalyzer).
		Build()

	if err != nil {
		log.Fatalf("Pipeline创建失败: %v", err)
	}

	fmt.Println("✓ 创建Pipeline: 融合 → 特征提取 → AI检测\n")

	// 6. 模拟数据流处理
	fmt.Println("--- 开始处理数据 ---\n")

	ctx := context.Background()

	for i := 0; i < 20; i++ {
		fmt.Printf("=== 时刻 %d ===\n", i+1)

		// 从三个数据源获取数据
		sample1 := source1.GenerateSample()
		sample2 := source2.GenerateSample()
		sample3 := source3.GenerateSample()

		fmt.Printf("数据源1: %.2f (质量: %.2f)\n", extractValue(sample1), sample1.GetQuality())
		fmt.Printf("数据源2: %.2f (质量: %.2f)\n", extractValue(sample2), sample2.GetQuality())
		fmt.Printf("数据源3: %.2f (质量: %.2f)\n", extractValue(sample3), sample3.GetQuality())

		// 依次通过Pipeline处理
		processedSample := sample1

		// 融合处理
		processedSample, err = fusionProcessor.Process(ctx, processedSample)
		if err == nil {
			processedSample, _ = fusionProcessor.Process(ctx, sample2)
			processedSample, _ = fusionProcessor.Process(ctx, sample3)
		}

		if err != nil {
			log.Printf("融合处理失败: %v", err)
			continue
		}

		fmt.Printf("融合后: %.2f (置信度: %.3f)\n",
			extractValue(processedSample),
			extractConfidence(processedSample))

		// 特征提取
		processedSample, err = featureExtractor.Process(ctx, processedSample)
		if err != nil {
			log.Printf("特征提取失败: %v", err)
			continue
		}

		features := extractFeatures(processedSample)
		fmt.Printf("提取特征: %d个\n", len(features.Data))

		// ML分析
		results, err := mlAnalyzer.Analyze(ctx, processedSample)
		if err != nil {
			log.Printf("ML分析失败: %v", err)
			continue
		}

		// 显示分析结果
		for _, result := range results {
			fmt.Printf("🤖 AI检测结果:\n")
			fmt.Printf("   分数: %.3f | 严重程度: %d/5 | 消息: %s\n",
				result.Score,
				result.Severity,
				result.Message)

			if isAnomaly, ok := result.Details["is_anomaly"].(bool); ok && isAnomaly {
				fmt.Printf("   ⚠️  检测到异常!\n")
			} else {
				fmt.Printf("   ✓ 正常数据\n")
			}
		}

		fmt.Println()

		// 模拟数据采集间隔
		time.Sleep(500 * time.Millisecond)
	}

	// 7. 显示统计信息
	fmt.Println("\n--- 分析统计 ---")
	stats := mlAnalyzer.GetStats()
	fmt.Printf("总样本数: %d\n", stats.TotalSamples)
	fmt.Printf("检测异常: %d\n", stats.AnomaliesDetected)
	fmt.Printf("异常率: %.2f%%\n", mlAnalyzer.GetAnomalyRate()*100)
	fmt.Printf("平均推理时间: %v\n", stats.AvgInferenceTime)
	fmt.Printf("模型准确率: %.2f%%\n", stats.ModelAccuracy*100)

	fmt.Println("\n✓ 示例完成!")
}

// MockDataSource 模拟数据源
type MockDataSource struct {
	id       string
	baseLine float64
	variance float64
	quality  float64
}

func NewMockDataSource(id string, baseLine float64, variance float64) *MockDataSource {
	return &MockDataSource{
		id:       id,
		baseLine: baseLine,
		variance: variance,
		quality:  0.8 + rand.Float64()*0.2,
	}
}

func (m *MockDataSource) GenerateSample() interfaces.DataSample {
	// 随机生成数据（正态分布）
	value := m.baseLine + (rand.Float64()-0.5)*m.variance*2

	// 10%概率生成异常数据
	if rand.Float64() < 0.1 {
		value += m.variance * 5 // 异常偏移
	}

	return &DataSampleImpl{
		timestamp: time.Now(),
		sourceID:  m.id,
		data: map[string]interface{}{
			"temperature": value,
			"pressure":    value * 0.5,
			"vibration":   value * 0.3,
		},
		quality:  m.quality,
		metadata: make(map[string]interface{}),
	}
}

// DataSampleImpl 数据样本实现
type DataSampleImpl struct {
	timestamp time.Time
	sourceID  string
	data      map[string]interface{}
	quality   float64
	metadata  map[string]interface{}
}

func (ds *DataSampleImpl) GetTimestamp() time.Time                 { return ds.timestamp }
func (ds *DataSampleImpl) GetSourceID() string                     { return ds.sourceID }
func (ds *DataSampleImpl) GetData() map[string]interface{}         { return ds.data }
func (ds *DataSampleImpl) GetQuality() float64                     { return ds.quality }
func (ds *DataSampleImpl) GetMetadata() map[string]interface{}     { return ds.metadata }

// createAndTrainModel 创建并训练一个简单的神经网络模型
func createAndTrainModel() inference.Model {
	// 创建一个简单的神经网络：输入10个特征 → 隐层20 → 隐层10 → 输出2 (异常分数, 置信度)
	model := inference.NewNeuralNetworkModel([]int{10, 20, 10, 2})

	// 这里应该用实际数据训练模型
	// 为了演示，我们跳过训练步骤

	return model
}

// 辅助函数

func extractValue(sample interfaces.DataSample) float64 {
	data := sample.GetData()
	if temp, ok := data["temperature"].(float64); ok {
		return temp
	}
	return 0.0
}

func extractConfidence(sample interfaces.DataSample) float64 {
	metadata := sample.GetMetadata()
	if conf, ok := metadata["confidence"].(float64); ok {
		return conf
	}
	return sample.GetQuality()
}

func extractFeatures(sample interfaces.DataSample) *inference.Tensor {
	metadata := sample.GetMetadata()
	if features, ok := metadata["features"].(*inference.Tensor); ok {
		return features
	}
	return &inference.Tensor{Shape: []int{0}, Data: []float64{}}
}
