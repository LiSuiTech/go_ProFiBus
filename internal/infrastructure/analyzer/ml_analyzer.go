package analyzer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/go_ProFiBus/inference"
	"github.com/yourusername/go_ProFiBus/internal/application/processor"
	"github.com/yourusername/go_ProFiBus/pkg/interfaces"
)

// MLAnalyzer 机器学习分析器
// 支持多种AI模型进行异常检测和预测
type MLAnalyzer struct {
	name            string
	analyzerType    interfaces.AnalyzerType
	inferenceEngine *inference.InferenceEngine
	modelName       string
	model           inference.Model

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
		inferenceEngine:  inference.NewInferenceEngine(),
		modelName:        modelName,
		anomalyThreshold: 0.7,  // 默认阈值
		confidenceMin:    0.5,  // 默认最低置信度
		stats: &AnalyzerStats{
			ModelAccuracy: 0.0,
		},
		config: make(map[string]interface{}),
	}
}

// Analyze 分析数据样本 - 实现 Analyzer 接口
func (mla *MLAnalyzer) Analyze(ctx context.Context, data interfaces.DataSample) ([]interfaces.AnalysisResult, error) {
	startTime := time.Now()

	// 1. 提取特征向量
	features, err := mla.extractFeatures(data)
	if err != nil {
		return nil, fmt.Errorf("feature extraction failed: %w", err)
	}

	// 2. 模型推理
	prediction, err := mla.predict(features)
	if err != nil {
		return nil, fmt.Errorf("model prediction failed: %w", err)
	}

	// 3. 解析预测结果
	results := mla.interpretPrediction(data, prediction)

	// 4. 更新统计信息
	inferenceTime := time.Since(startTime)
	mla.updateStats(inferenceTime, results)

	return results, nil
}

// extractFeatures 提取特征向量
func (mla *MLAnalyzer) extractFeatures(data interfaces.DataSample) (*inference.Tensor, error) {
	// 检查样本元数据中是否已有特征向量
	metadata := data.GetMetadata()
	if metadata != nil {
		if features, ok := metadata["features"].(*inference.Tensor); ok {
			return features, nil
		}
	}

	// 如果有特征提取器，使用它
	if mla.featureExtractor != nil {
		return mla.featureExtractor.Process(context.Background(), data)
	}

	// 否则从样本中直接提取
	return processor.ExtractFeaturesFromSample(data)
}

// predict 执行模型推理
func (mla *MLAnalyzer) predict(features *inference.Tensor) (*inference.Tensor, error) {
	mla.mu.RLock()
	defer mla.mu.RUnlock()

	if mla.model == nil {
		return nil, fmt.Errorf("model not loaded")
	}

	// 执行推理
	prediction, err := mla.model.Predict(features)
	if err != nil {
		return nil, fmt.Errorf("prediction failed: %w", err)
	}

	return prediction, nil
}

// interpretPrediction 解释预测结果
func (mla *MLAnalyzer) interpretPrediction(data interfaces.DataSample, prediction *inference.Tensor) []interfaces.AnalysisResult {
	results := make([]interfaces.AnalysisResult, 0)

	// 提取异常分数（假设预测输出的第一个值是异常概率）
	anomalyScore := 0.0
	if len(prediction.Data) > 0 {
		anomalyScore = prediction.Data[0]
	}

	// 提取置信度（如果有第二个输出）
	confidence := 1.0
	if len(prediction.Data) > 1 {
		confidence = prediction.Data[1]
	}

	// 检查是否超过阈值
	isAnomaly := anomalyScore >= mla.anomalyThreshold
	hasConfidence := confidence >= mla.confidenceMin

	if !hasConfidence {
		// 置信度不足，不报告
		return results
	}

	// 计算严重程度 (1-5)
	severity := mla.calculateSeverity(anomalyScore)

	// 创建分析结果
	result := interfaces.AnalysisResult{
		Type:      "ml_anomaly_detection",
		Severity:  severity,
		Score:     anomalyScore,
		Message:   mla.generateMessage(isAnomaly, anomalyScore, confidence),
		Details: map[string]interface{}{
			"model_name":        mla.modelName,
			"model_type":        mla.model.GetType().String(),
			"anomaly_score":     anomalyScore,
			"confidence":        confidence,
			"is_anomaly":        isAnomaly,
			"threshold":         mla.anomalyThreshold,
			"prediction_vector": prediction.Data,
			"input_shape":       mla.model.GetInputShape(),
			"output_shape":      mla.model.GetOutputShape(),
		},
		Timestamp: time.Now(),
		SourceID:  data.GetSourceID(),
	}

	results = append(results, result)

	return results
}

// calculateSeverity 计算严重程度 (1-5)
func (mla *MLAnalyzer) calculateSeverity(score float64) int {
	switch {
	case score < 0.3:
		return 1 // Info
	case score < 0.5:
		return 2 // Low
	case score < 0.7:
		return 3 // Medium
	case score < 0.9:
		return 4 // High
	default:
		return 5 // Critical
	}
}

// generateMessage 生成分析消息
func (mla *MLAnalyzer) generateMessage(isAnomaly bool, score float64, confidence float64) string {
	if isAnomaly {
		return fmt.Sprintf("ML模型检测到异常 (分数: %.3f, 置信度: %.3f)", score, confidence)
	}
	return fmt.Sprintf("正常数据 (分数: %.3f, 置信度: %.3f)", score, confidence)
}

// updateStats 更新统计信息
func (mla *MLAnalyzer) updateStats(inferenceTime time.Duration, results []interfaces.AnalysisResult) {
	mla.statsMu.Lock()
	defer mla.statsMu.Unlock()

	mla.stats.TotalSamples++
	mla.stats.LastInferenceTime = inferenceTime

	// 更新平均推理时间
	if mla.stats.TotalSamples == 1 {
		mla.stats.AvgInferenceTime = inferenceTime
	} else {
		// 移动平均
		mla.stats.AvgInferenceTime = (mla.stats.AvgInferenceTime*time.Duration(mla.stats.TotalSamples-1) + inferenceTime) / time.Duration(mla.stats.TotalSamples)
	}

	// 统计异常数量
	for _, result := range results {
		if details, ok := result.Details["is_anomaly"].(bool); ok && details {
			mla.stats.AnomaliesDetected++
		}
	}
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

	// 加载模型
	if modelPath, ok := config["model_path"].(string); ok {
		if modelType, ok := config["model_type"].(string); ok {
			if err := mla.loadModel(modelType, modelPath); err != nil {
				return fmt.Errorf("model loading failed: %w", err)
			}
		}
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

// loadModel 加载模型
func (mla *MLAnalyzer) loadModel(modelType string, modelPath string) error {
	var model inference.Model
	var err error

	switch modelType {
	case "neural_network":
		model = inference.NewNeuralNetworkModel([]int{10, 20, 10, 1}) // 示例网络结构
		err = model.Load(modelPath)

	case "linear_regression":
		// TODO: 实现线性回归模型加载
		return fmt.Errorf("linear_regression not yet implemented")

	case "svm":
		// TODO: 实现SVM模型加载
		return fmt.Errorf("svm not yet implemented")

	case "decision_tree":
		// TODO: 实现决策树模型加载
		return fmt.Errorf("decision_tree not yet implemented")

	default:
		return fmt.Errorf("unsupported model type: %s", modelType)
	}

	if err != nil {
		return fmt.Errorf("model load error: %w", err)
	}

	mla.model = model
	mla.inferenceEngine.RegisterModel(mla.modelName, model)

	return nil
}

// Close 关闭分析器 - 实现 Analyzer 接口
func (mla *MLAnalyzer) Close() error {
	mla.mu.Lock()
	defer mla.mu.Unlock()

	if mla.featureExtractor != nil {
		if err := mla.featureExtractor.Close(); err != nil {
			return fmt.Errorf("feature extractor close failed: %w", err)
		}
	}

	return nil
}

// GetStats 获取统计信息
func (mla *MLAnalyzer) GetStats() AnalyzerStats {
	mla.statsMu.RLock()
	defer mla.statsMu.RUnlock()

	return *mla.stats
}

// SetThreshold 设置异常阈值
func (mla *MLAnalyzer) SetThreshold(threshold float64) {
	mla.mu.Lock()
	defer mla.mu.Unlock()

	if threshold >= 0 && threshold <= 1.0 {
		mla.anomalyThreshold = threshold
	}
}

// GetThreshold 获取异常阈值
func (mla *MLAnalyzer) GetThreshold() float64 {
	mla.mu.RLock()
	defer mla.mu.RUnlock()

	return mla.anomalyThreshold
}

// UpdateModel 更新模型（支持在线学习）
func (mla *MLAnalyzer) UpdateModel(newModel inference.Model) error {
	mla.mu.Lock()
	defer mla.mu.Unlock()

	mla.model = newModel
	mla.inferenceEngine.RegisterModel(mla.modelName, newModel)

	return nil
}

// GetModel 获取当前模型
func (mla *MLAnalyzer) GetModel() inference.Model {
	mla.mu.RLock()
	defer mla.mu.RUnlock()

	return mla.model
}

// GetInferenceEngine 获取推理引擎
func (mla *MLAnalyzer) GetInferenceEngine() *inference.InferenceEngine {
	return mla.inferenceEngine
}

// BatchAnalyze 批量分析（提高性能）
func (mla *MLAnalyzer) BatchAnalyze(ctx context.Context, samples []interfaces.DataSample) ([][]interfaces.AnalysisResult, error) {
	results := make([][]interfaces.AnalysisResult, 0, len(samples))

	for _, sample := range samples {
		result, err := mla.Analyze(ctx, sample)
		if err != nil {
			return nil, fmt.Errorf("batch analysis failed at sample %s: %w", sample.GetSourceID(), err)
		}
		results = append(results, result)
	}

	return results, nil
}

// EvaluateModelPerformance 评估模型性能
func (mla *MLAnalyzer) EvaluateModelPerformance(testSamples []interfaces.DataSample, labels []bool) (float64, error) {
	if len(testSamples) != len(labels) {
		return 0, fmt.Errorf("sample count (%d) does not match label count (%d)", len(testSamples), len(labels))
	}

	correct := 0
	total := len(testSamples)

	for i, sample := range testSamples {
		results, err := mla.Analyze(context.Background(), sample)
		if err != nil {
			continue
		}

		if len(results) > 0 {
			predicted := false
			if isAnomaly, ok := results[0].Details["is_anomaly"].(bool); ok {
				predicted = isAnomaly
			}

			if predicted == labels[i] {
				correct++
			}
		}
	}

	accuracy := float64(correct) / float64(total)

	// 更新统计
	mla.statsMu.Lock()
	mla.stats.ModelAccuracy = accuracy
	mla.statsMu.Unlock()

	return accuracy, nil
}

// GetAnomalyRate 获取异常率
func (mla *MLAnalyzer) GetAnomalyRate() float64 {
	mla.statsMu.RLock()
	defer mla.statsMu.RUnlock()

	if mla.stats.TotalSamples == 0 {
		return 0
	}

	return float64(mla.stats.AnomaliesDetected) / float64(mla.stats.TotalSamples)
}

// ResetStats 重置统计信息
func (mla *MLAnalyzer) ResetStats() {
	mla.statsMu.Lock()
	defer mla.statsMu.Unlock()

	mla.stats = &AnalyzerStats{
		ModelAccuracy: mla.stats.ModelAccuracy, // 保留准确率
	}
}
