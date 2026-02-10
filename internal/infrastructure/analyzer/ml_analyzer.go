package analyzer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/internal/application/processor"
	"go_ProFiBus/pkg/interfaces"
)

// MLAnalyzer 机器学习分析器
// 使用ML模型插件系统进行推理
type MLAnalyzer struct {
	name            string
	analyzerType    interfaces.AnalyzerType
	modelName       string

	// ML模型引擎（可选）
	engine          interfaces.InferenceEngine

	// 特征提取器（可选）
	featureExtractor *processor.FeatureExtractor

	// 阈值配置
	anomalyThreshold float64 // 异常分数阈值
	confidenceMin    float64 // 最低置信度要求

	// 性能统计
	stats      *AnalyzerStats
	statsMu    sync.RWMutex

	// 配置
	config     map[string]interface{}
	mu         sync.RWMutex
}

// AnalyzerStats 分析器统计信息
type AnalyzerStats struct {
	TotalSamples      int64
	AnomaliesDetected int64
	AvgInferenceTime  time.Duration
	LastInferenceTime time.Duration
	ModelAccuracy     float64
}

// MLAnalyzerConfig ML分析器配置
type MLAnalyzerConfig struct {
	ModelName         string             `json:"model_name"`          // 模型名称
	ModelType         string             `json:"model_type"`          // 模型类型: neural_network, svm, etc
	ModelPath         string             `json:"model_path"`          // 模型文件路径
	AnomalyThreshold  float64            `json:"anomaly_threshold"`   // 异常阈值 (0-1)
	ConfidenceMin     float64            `json:"confidence_min"`      // 最低置信度
	UseFeatureExtractor bool             `json:"use_feature_extractor"` // 是否使用特征提取
	Features          []string           `json:"features"`            // 要提取的特征列表
	ModelConfig       map[string]interface{} `json:"model_config"`    // 模型特定配置
}

// NewMLAnalyzer 创建ML分析器
func NewMLAnalyzer(name string, modelName string) *MLAnalyzer {
	return &MLAnalyzer{
		name:             name,
		analyzerType:     interfaces.AnalyzerTypeML,
		modelName:        modelName,
		anomalyThreshold: 0.7,  // 默认阈值
		confidenceMin:    0.5,  // 默认最低置信度
		stats: &AnalyzerStats{
			ModelAccuracy: 0.0,
		},
		config: make(map[string]interface{}),
	}
}

// SetEngine 设置推理引擎
func (mla *MLAnalyzer) SetEngine(engine interfaces.InferenceEngine) {
	mla.mu.Lock()
	defer mla.mu.Unlock()
	mla.engine = engine
}

// Analyze 分析数据样本 - 实现 Analyzer 接口
func (mla *MLAnalyzer) Analyze(ctx context.Context, data interfaces.DataSample) ([]interfaces.AnalysisResult, error) {
	mla.mu.RLock()
	engine := mla.engine
	modelName := mla.modelName
	mla.mu.RUnlock()

	// 如果没有设置引擎或模型名称，返回空结果
	if engine == nil || modelName == "" {
		return []interfaces.AnalysisResult{}, nil
	}

	// 准备输入数据（GetData() 已返回 map[string]interface{}）
	inputData := make([]float64, 0)
	dataMap := data.GetData()
	if dataMap != nil {
		// 提取数值特征
		for _, v := range dataMap {
			switch val := v.(type) {
			case float64:
				inputData = append(inputData, val)
			case int:
				inputData = append(inputData, float64(val))
			case int64:
				inputData = append(inputData, float64(val))
			}
		}
	}

	if len(inputData) == 0 {
		return []interfaces.AnalysisResult{}, nil
	}

	// 创建输入张量
	inputTensor := &interfaces.Tensor{
		Shape: []int{len(inputData)},
		Data:  inputData,
	}

	// 执行预测
	startTime := time.Now()
	output, err := engine.Predict(modelName, inputTensor)
	inferenceTime := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("ML模型预测失败: %w", err)
	}

	// 更新统计信息
	mla.statsMu.Lock()
	mla.stats.TotalSamples++
	mla.stats.LastInferenceTime = inferenceTime
	if mla.stats.TotalSamples > 0 {
		avgTime := mla.stats.AvgInferenceTime
		avgTime = (avgTime*time.Duration(mla.stats.TotalSamples-1) + inferenceTime) / time.Duration(mla.stats.TotalSamples)
		mla.stats.AvgInferenceTime = avgTime
	}
	mla.statsMu.Unlock()

	// 构建分析结果
	results := make([]interfaces.AnalysisResult, 0)
	if len(output.Data) > 0 {
		// 检查异常分数
		anomalyScore := output.Data[0]
		if anomalyScore > mla.anomalyThreshold {
			results = append(results, interfaces.AnalysisResult{
				Type:      "anomaly",
				Severity:  3,
				Score:     anomalyScore,
				Message:   fmt.Sprintf("检测到异常，分数: %.2f", anomalyScore),
				Details:   map[string]interface{}{"anomaly_score": anomalyScore},
				Timestamp: time.Now(),
			})
		}
	}

	return results, nil
}

// GetType 获取分析器类型 - 实现 Analyzer 接口
func (mla *MLAnalyzer) GetType() interfaces.AnalyzerType {
	return mla.analyzerType
}

// GetName 获取分析器名称 - 实现 Analyzer 接口
func (mla *MLAnalyzer) GetName() string {
	return mla.name
}

// Configure 配置分析器 - 实现 Analyzer 接口
func (mla *MLAnalyzer) Configure(config map[string]interface{}) error {
	mla.mu.Lock()
	defer mla.mu.Unlock()

	mla.config = config

	// 解析配置
	if modelName, ok := config["model_name"].(string); ok {
		mla.modelName = modelName
	}

	if threshold, ok := config["anomaly_threshold"].(float64); ok {
		mla.anomalyThreshold = threshold
	}

	if confidenceMin, ok := config["confidence_min"].(float64); ok {
		mla.confidenceMin = confidenceMin
	}

	// 配置特征提取器
	if useExtractor, ok := config["use_feature_extractor"].(bool); ok && useExtractor {
		mla.featureExtractor = processor.NewFeatureExtractor("ml_feature_extractor")

		if features, ok := config["features"].([]interface{}); ok {
			for _, f := range features {
				if featureName, ok := f.(string); ok {
					// 特征提取器会自动添加预定义特征
					_ = featureName
				}
			}
		}
	}

	return nil
}

// GetStats 获取统计信息
func (mla *MLAnalyzer) GetStats() *AnalyzerStats {
	mla.statsMu.RLock()
	defer mla.statsMu.RUnlock()

	stats := *mla.stats
	return &stats
}

// GetThreshold 获取异常阈值
func (mla *MLAnalyzer) GetThreshold() float64 {
	mla.mu.RLock()
	defer mla.mu.RUnlock()

	return mla.anomalyThreshold
}

// SetThreshold 设置异常阈值
func (mla *MLAnalyzer) SetThreshold(threshold float64) {
	mla.mu.Lock()
	defer mla.mu.Unlock()

	mla.anomalyThreshold = threshold
}

// GetAnomalyRate 获取当前异常率（已检测异常数/总样本数）
func (mla *MLAnalyzer) GetAnomalyRate() float64 {
	mla.statsMu.RLock()
	defer mla.statsMu.RUnlock()

	if mla.stats.TotalSamples <= 0 {
		return 0
	}
	return float64(mla.stats.AnomaliesDetected) / float64(mla.stats.TotalSamples)
}

// Close 关闭分析器 - 实现 Analyzer 接口
func (mla *MLAnalyzer) Close() error {
	return nil
}
