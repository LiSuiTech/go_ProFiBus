package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"go_ProFiBus/fusion"
	"go_ProFiBus/inference"
	"go_ProFiBus/internal/application/processor"
	"go_ProFiBus/internal/infrastructure/analyzer"
	"go_ProFiBus/pkg/interfaces"
)

// 本示例展示：完整的数据流转过程
// 从原始数据 → 融合 → 特征提取 → AI检测 → 结果输出

func main() {
	fmt.Println("========================================")
	fmt.Println("   数据流转完整示例")
	fmt.Println("========================================\n")

	// ============================================================
	// 步骤1: 准备原始样例数据（来自3个传感器）
	// ============================================================
	fmt.Println("【步骤1】原始样例数据（3个传感器）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 传感器1数据（温度传感器 - 高精度）
	sensor1Data := &DataSampleImpl{
		timestamp: time.Now(),
		sourceID:  "temp-sensor-001",
		data: map[string]interface{}{
			"temperature": 85.3,
			"pressure":    1.013,
			"humidity":    45.2,
			"location":    "区域A",
		},
		quality:  0.95,
		metadata: map[string]interface{}{},
	}

	// 传感器2数据（温度传感器 - 中等精度）
	sensor2Data := &DataSampleImpl{
		timestamp: time.Now(),
		sourceID:  "temp-sensor-002",
		data: map[string]interface{}{
			"temperature": 87.1,
			"pressure":    1.015,
			"humidity":    46.8,
			"location":    "区域A",
		},
		quality:  0.85,
		metadata: map[string]interface{}{},
	}

	// 传感器3数据（温度传感器 - 低精度）
	sensor3Data := &DataSampleImpl{
		timestamp: time.Now(),
		sourceID:  "temp-sensor-003",
		data: map[string]interface{}{
			"temperature": 89.5,
			"pressure":    1.020,
			"humidity":    48.5,
			"location":    "区域A",
		},
		quality:  0.70,
		metadata: map[string]interface{}{},
	}

	printSample("传感器1 (高精度)", sensor1Data)
	printSample("传感器2 (中精度)", sensor2Data)
	printSample("传感器3 (低精度)", sensor3Data)

	// ============================================================
	// 步骤2: 数据融合（加权融合策略）
	// ============================================================
	fmt.Println("\n【步骤2】数据融合处理")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	fusionProcessor := processor.NewFusionProcessor("multi-sensor-fusion", fusion.StrategyWeighted)

	// 设置权重（高精度传感器权重更高）
	fusionProcessor.SetSourceWeight("temp-sensor-001", 0.5)  // 50%
	fusionProcessor.SetSourceWeight("temp-sensor-002", 0.3)  // 30%
	fusionProcessor.SetSourceWeight("temp-sensor-003", 0.2)  // 20%

	fmt.Println("融合策略: 加权融合 (Weighted)")
	fmt.Println("权重配置:")
	fmt.Println("  - 传感器1: 50% (高精度)")
	fmt.Println("  - 传感器2: 30% (中精度)")
	fmt.Println("  - 传感器3: 20% (低精度)\n")

	ctx := context.Background()

	// 依次处理三个传感器的数据
	fusedSample, _ := fusionProcessor.Process(ctx, sensor1Data)
	fusedSample, _ = fusionProcessor.Process(ctx, sensor2Data)
	fusedSample, _ = fusionProcessor.Process(ctx, sensor3Data)

	fmt.Println("融合计算过程:")
	fmt.Println("  temperature = 85.3×0.5 + 87.1×0.3 + 89.5×0.2")
	fmt.Printf("              = %.2f + %.2f + %.2f\n", 85.3*0.5, 87.1*0.3, 89.5*0.2)
	fmt.Printf("              = %.2f °C\n", 85.3*0.5+87.1*0.3+89.5*0.2)

	printSample("融合后数据", fusedSample)

	// ============================================================
	// 步骤3: 特征提取
	// ============================================================
	fmt.Println("\n【步骤3】特征提取")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	featureExtractor := processor.NewFeatureExtractor("ml-features")

	fmt.Println("提取的特征:")
	fmt.Println("  1. numeric_mean    - 数值平均值")
	fmt.Println("  2. numeric_stddev  - 标准差")
	fmt.Println("  3. quality         - 数据质量")
	fmt.Println("  4. confidence      - 融合置信度\n")

	// 特征提取
	enrichedSample, _ := featureExtractor.Process(ctx, fusedSample)

	// 提取特征向量
	metadata := enrichedSample.GetMetadata()
	if features, ok := metadata["features"].(*inference.Tensor); ok {
		fmt.Println("提取的特征向量:")
		fmt.Printf("  Shape: %v\n", features.Shape)
		fmt.Printf("  Values: [")
		for i, val := range features.Data {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%.3f", val)
		}
		fmt.Println("]\n")

		// 特征解释
		featureNames := featureExtractor.GetFeatureNames()
		fmt.Println("特征详解:")
		for i, name := range featureNames {
			if i < len(features.Data) {
				fmt.Printf("  %d. %-18s = %.3f\n", i+1, name, features.Data[i])
			}
		}
	}

	// ============================================================
	// 步骤4: AI模型检测（神经网络异常检测）
	// ============================================================
	fmt.Println("\n【步骤4】AI模型检测")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	mlAnalyzer := analyzer.NewMLAnalyzer("anomaly-detector", "temp-anomaly-model")

	// 配置ML分析器
	mlAnalyzer.Configure(map[string]interface{}{
		"model_name":          "temp-anomaly-model",
		"model_type":          "neural_network",
		"anomaly_threshold":   0.7,
		"confidence_min":      0.5,
		"use_feature_extractor": false,
	})

	// 创建并加载模型
	model := createSimpleAnomalyModel()
	mlAnalyzer.UpdateModel(model)

	fmt.Println("模型信息:")
	fmt.Println("  类型: 神经网络 (NeuralNetwork)")
	fmt.Println("  结构: 4层 [4 → 8 → 4 → 2]")
	fmt.Println("  输入: 4个特征")
	fmt.Println("  输出: [异常分数, 置信度]")
	fmt.Println("  异常阈值: 0.7\n")

	// 执行AI检测
	results, _ := mlAnalyzer.Analyze(ctx, enrichedSample)

	fmt.Println("AI检测结果:")
	for _, result := range results {
		fmt.Printf("  类型: %s\n", result.Type)
		fmt.Printf("  严重程度: %d/5 ", result.Severity)

		severityText := ""
		switch result.Severity {
		case 1:
			severityText = "(Info - 信息)"
		case 2:
			severityText = "(Low - 低)"
		case 3:
			severityText = "(Medium - 中)"
		case 4:
			severityText = "(High - 高)"
		case 5:
			severityText = "(Critical - 严重)"
		}
		fmt.Println(severityText)

		fmt.Printf("  异常分数: %.3f\n", result.Score)
		fmt.Printf("  消息: %s\n", result.Message)

		if details := result.Details; details != nil {
			fmt.Println("\n  详细信息:")
			if modelName, ok := details["model_name"].(string); ok {
				fmt.Printf("    模型名称: %s\n", modelName)
			}
			if modelType, ok := details["model_type"].(string); ok {
				fmt.Printf("    模型类型: %s\n", modelType)
			}
			if anomalyScore, ok := details["anomaly_score"].(float64); ok {
				fmt.Printf("    异常分数: %.3f\n", anomalyScore)
			}
			if confidence, ok := details["confidence"].(float64); ok {
				fmt.Printf("    检测置信度: %.3f\n", confidence)
			}
			if isAnomaly, ok := details["is_anomaly"].(bool); ok {
				fmt.Printf("    是否异常: %v\n", isAnomaly)
			}
			if threshold, ok := details["threshold"].(float64); ok {
				fmt.Printf("    阈值: %.3f\n", threshold)
			}
			if predVector, ok := details["prediction_vector"].([]float64); ok {
				fmt.Printf("    预测向量: [%.3f, %.3f]\n", predVector[0], predVector[1])
			}
		}
	}

	// ============================================================
	// 步骤5: 完整数据流总结
	// ============================================================
	fmt.Println("\n\n【步骤5】完整数据流总结")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	fmt.Println("数据流转路径:")
	fmt.Println("  ┌─────────────────┐")
	fmt.Println("  │  原始数据       │  3个传感器数据")
	fmt.Println("  │  Sensor 1,2,3   │  温度: 85.3, 87.1, 89.5 °C")
	fmt.Println("  └────────┬────────┘")
	fmt.Println("           │")
	fmt.Println("           ▼")
	fmt.Println("  ┌─────────────────┐")
	fmt.Println("  │  数据融合       │  加权融合策略")
	fmt.Println("  │  (Weighted)     │  权重: 50%, 30%, 20%")
	fmt.Printf("  │  结果: %.2f°C   │\n", 85.3*0.5+87.1*0.3+89.5*0.2)
	fmt.Println("  └────────┬────────┘")
	fmt.Println("           │")
	fmt.Println("           ▼")
	fmt.Println("  ┌─────────────────┐")
	fmt.Println("  │  特征提取       │  提取4个特征")
	fmt.Println("  │  (Features)     │  均值、标准差、质量、置信度")
	fmt.Println("  └────────┬────────┘")
	fmt.Println("           │")
	fmt.Println("           ▼")
	fmt.Println("  ┌─────────────────┐")
	fmt.Println("  │  AI模型检测     │  神经网络异常检测")
	fmt.Println("  │  (ML Analyzer)  │  输出: 异常分数 + 置信度")
	fmt.Println("  └────────┬────────┘")
	fmt.Println("           │")
	fmt.Println("           ▼")
	fmt.Println("  ┌─────────────────┐")
	if len(results) > 0 && results[0].Score >= 0.7 {
		fmt.Println("  │  检测结果       │  ⚠️  异常警告")
	} else {
		fmt.Println("  │  检测结果       │  ✓  正常状态")
	}
	fmt.Println("  └─────────────────┘")

	fmt.Println("\n数据转换详情:")
	fmt.Printf("  输入数据量: 3个传感器样本\n")
	fmt.Printf("  输出数据量: 1个分析结果\n")
	fmt.Printf("  数据压缩率: 3:1\n")
	fmt.Printf("  处理精度提升: 从 70-95%% → %.0f%%\n", fusedSample.GetQuality()*100)

	if len(results) > 0 {
		if confidence, ok := results[0].Details["confidence"].(float64); ok {
			fmt.Printf("  检测置信度: %.0f%%\n", confidence*100)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("   示例完成!")
	fmt.Println("========================================\n")
}

// ============================================================
// 辅助函数
// ============================================================

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

// printSample 打印数据样本
func printSample(title string, sample interfaces.DataSample) {
	fmt.Printf("【%s】\n", title)
	fmt.Printf("  时间戳: %s\n", sample.GetTimestamp().Format("2006-01-02 15:04:05.000"))
	fmt.Printf("  数据源ID: %s\n", sample.GetSourceID())
	fmt.Printf("  数据质量: %.2f\n", sample.GetQuality())

	fmt.Println("  数据内容:")
	data := sample.GetData()
	for key, value := range data {
		switch v := value.(type) {
		case float64:
			fmt.Printf("    - %s: %.2f\n", key, v)
		case string:
			fmt.Printf("    - %s: %s\n", key, v)
		default:
			fmt.Printf("    - %s: %v\n", key, v)
		}
	}

	metadata := sample.GetMetadata()
	if len(metadata) > 0 {
		fmt.Println("  元数据:")
		for key, value := range metadata {
			if key == "features" {
				// 特征向量特殊处理
				if tensor, ok := value.(*inference.Tensor); ok {
					fmt.Printf("    - %s: Tensor%v\n", key, tensor.Shape)
				}
			} else if key == "fusion_weights" {
				// 融合权重特殊处理
				fmt.Printf("    - %s: ", key)
				weightsJSON, _ := json.Marshal(value)
				fmt.Println(string(weightsJSON))
			} else {
				fmt.Printf("    - %s: %v\n", key, value)
			}
		}
	}
	fmt.Println()
}

// createSimpleAnomalyModel 创建简单的异常检测模型
func createSimpleAnomalyModel() inference.Model {
	// 创建一个4层神经网络: 4 → 8 → 4 → 2
	// 输入: 4个特征 (mean, stddev, quality, confidence)
	// 输出: 2个值 (anomaly_score, confidence)
	model := inference.NewNeuralNetworkModel([]int{4, 8, 4, 2})

	// 注：实际应用中需要用真实数据训练模型
	// 这里使用未训练的模型仅用于演示数据流

	return model
}
