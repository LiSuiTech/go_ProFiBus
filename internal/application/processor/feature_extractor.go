package processor

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// FeatureExtractor 特征提取处理器
// 从数据样本中提取机器学习模型需要的特征向量
type FeatureExtractor struct {
	name       string
	features   []FeatureExtractorFunc
	featureMap map[string]int // 特征名称到索引的映射
	config     interfaces.ProcessorConfig
	mu         sync.RWMutex

	// 历史数据窗口 - 用于提取时序特征
	historyWindow time.Duration
	historyBuffer []interfaces.DataSample
	bufferMu      sync.RWMutex
}

// FeatureExtractorFunc 特征提取函数类型
type FeatureExtractorFunc func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error)

// FeatureConfig 特征提取配置
type FeatureConfig struct {
	Features      []string `json:"features"`       // 要提取的特征列表
	HistoryWindow string   `json:"history_window"` // 历史窗口大小
	HistorySize   int      `json:"history_size"`   // 历史缓冲区大小
}

// 预定义特征提取器
var (
	// 基础特征
	extractNumericMean   FeatureExtractorFunc
	extractNumericStdDev FeatureExtractorFunc
	extractNumericMin    FeatureExtractorFunc
	extractNumericMax    FeatureExtractorFunc

	// 时序特征
	extractRateOfChange  FeatureExtractorFunc
	extractTrend         FeatureExtractorFunc
	extractVariance      FeatureExtractorFunc

	// 质量特征
	extractQuality       FeatureExtractorFunc
	extractConfidence    FeatureExtractorFunc
	extractSourceCount   FeatureExtractorFunc
)

func init() {
	// 基础统计特征
	extractNumericMean = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		values := extractNumericValues(sample)
		if len(values) == 0 {
			return 0, nil
		}
		return calculateMean(values), nil
	}

	extractNumericStdDev = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		values := extractNumericValues(sample)
		if len(values) == 0 {
			return 0, nil
		}
		return calculateStdDev(values), nil
	}

	extractNumericMin = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		values := extractNumericValues(sample)
		if len(values) == 0 {
			return 0, nil
		}
		return calculateMin(values), nil
	}

	extractNumericMax = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		values := extractNumericValues(sample)
		if len(values) == 0 {
			return 0, nil
		}
		return calculateMax(values), nil
	}

	// 时序特征
	extractRateOfChange = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		if len(history) < 2 {
			return 0, nil
		}

		current := extractNumericValues(sample)
		previous := extractNumericValues(history[len(history)-1])

		if len(current) == 0 || len(previous) == 0 {
			return 0, nil
		}

		currentMean := calculateMean(current)
		previousMean := calculateMean(previous)

		if previousMean == 0 {
			return 0, nil
		}

		return (currentMean - previousMean) / previousMean, nil
	}

	extractTrend = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		if len(history) < 3 {
			return 0, nil
		}

		// 计算最近N个样本的趋势（简单线性回归斜率）
		values := make([]float64, 0, len(history))
		for _, h := range history {
			nums := extractNumericValues(h)
			if len(nums) > 0 {
				values = append(values, calculateMean(nums))
			}
		}

		if len(values) < 3 {
			return 0, nil
		}

		return calculateTrend(values), nil
	}

	extractVariance = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		if len(history) < 2 {
			return 0, nil
		}

		values := make([]float64, 0, len(history))
		for _, h := range history {
			nums := extractNumericValues(h)
			if len(nums) > 0 {
				values = append(values, calculateMean(nums))
			}
		}

		if len(values) < 2 {
			return 0, nil
		}

		mean := calculateMean(values)
		variance := 0.0
		for _, v := range values {
			diff := v - mean
			variance += diff * diff
		}
		return variance / float64(len(values)), nil
	}

	// 质量特征
	extractQuality = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		return sample.GetQuality(), nil
	}

	extractConfidence = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		metadata := sample.GetMetadata()
		if confidence, ok := metadata["confidence"].(float64); ok {
			return confidence, nil
		}
		return sample.GetQuality(), nil // 如果没有置信度，使用质量分数
	}

	extractSourceCount = func(sample interfaces.DataSample, history []interfaces.DataSample) (float64, error) {
		metadata := sample.GetMetadata()
		if count, ok := metadata["source_count"].(int); ok {
			return float64(count), nil
		}
		return 1.0, nil // 默认1个数据源
	}
}

// NewFeatureExtractor 创建特征提取器
func NewFeatureExtractor(name string) *FeatureExtractor {
	fe := &FeatureExtractor{
		name:          name,
		features:      make([]FeatureExtractorFunc, 0),
		featureMap:    make(map[string]int),
		historyWindow: 10 * time.Second,
		historyBuffer: make([]interfaces.DataSample, 0, 100),
	}

	// 添加默认特征
	fe.AddFeature("numeric_mean", extractNumericMean)
	fe.AddFeature("numeric_stddev", extractNumericStdDev)
	fe.AddFeature("quality", extractQuality)
	fe.AddFeature("confidence", extractConfidence)

	return fe
}

// AddFeature 添加特征提取函数
func (fe *FeatureExtractor) AddFeature(name string, extractor FeatureExtractorFunc) {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	fe.featureMap[name] = len(fe.features)
	fe.features = append(fe.features, extractor)
}

// Process 处理数据样本 - 实现 Processor 接口
func (fe *FeatureExtractor) Process(ctx context.Context, input interfaces.DataSample) (interfaces.DataSample, error) {
	// 1. 添加到历史缓冲区
	fe.addToHistory(input)

	// 2. 提取特征向量
	featureVector, err := fe.extractFeatures(input)
	if err != nil {
		return input, fmt.Errorf("feature extraction failed: %w", err)
	}

	// 3. 将特征向量添加到样本元数据中
	metadata := input.GetMetadata()
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	metadata["features"] = featureVector
	metadata["feature_count"] = len(featureVector.Data)

	// 4. 创建增强的样本（使用与 fusion_processor 一致的简单实现）
	enrichedSample := &featureSampleImpl{
		timestamp: input.GetTimestamp(),
		sourceID:  input.GetSourceID(),
		data:      input.GetData(),
		quality:   input.GetQuality(),
		metadata:  metadata,
	}

	return enrichedSample, nil
}

// extractFeatures 提取特征向量
func (fe *FeatureExtractor) extractFeatures(sample interfaces.DataSample) (*interfaces.Tensor, error) {
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	features := make([]float64, 0, len(fe.features))

	// 获取历史数据
	fe.bufferMu.RLock()
	history := make([]interfaces.DataSample, len(fe.historyBuffer))
	copy(history, fe.historyBuffer)
	fe.bufferMu.RUnlock()

	// 依次提取每个特征
	for _, extractor := range fe.features {
		value, err := extractor(sample, history)
		if err != nil {
			return nil, fmt.Errorf("feature extraction error: %w", err)
		}
		features = append(features, value)
	}

	// 创建张量
	tensor := &interfaces.Tensor{
		Shape: []int{len(features)},
		Data:  features,
	}

	return tensor, nil
}

// addToHistory 添加样本到历史缓冲区
func (fe *FeatureExtractor) addToHistory(sample interfaces.DataSample) {
	fe.bufferMu.Lock()
	defer fe.bufferMu.Unlock()

	// 添加到缓冲区
	fe.historyBuffer = append(fe.historyBuffer, sample)

	// 保持缓冲区大小限制
	if len(fe.historyBuffer) > 100 {
		fe.historyBuffer = fe.historyBuffer[1:]
	}

	// 清理过期数据
	now := time.Now()
	i := 0
	for i < len(fe.historyBuffer) {
		if now.Sub(fe.historyBuffer[i].GetTimestamp()) > fe.historyWindow {
			i++
		} else {
			break
		}
	}
	if i > 0 {
		fe.historyBuffer = fe.historyBuffer[i:]
	}
}

// featureSampleImpl 用于 Process 返回的 DataSample 实现（避免与 fusion_processor.simpleDataSample 重名）
type featureSampleImpl struct {
	timestamp time.Time
	sourceID  string
	data      map[string]interface{}
	quality   float64
	metadata  map[string]interface{}
}

func (s *featureSampleImpl) GetTimestamp() time.Time              { return s.timestamp }
func (s *featureSampleImpl) GetSourceID() string                   { return s.sourceID }
func (s *featureSampleImpl) GetData() map[string]interface{}       { return s.data }
func (s *featureSampleImpl) GetQuality() float64                   { return s.quality }
func (s *featureSampleImpl) GetMetadata() map[string]interface{}   { return s.metadata }

// GetName 获取处理器名称 - 实现 Processor 接口
func (fe *FeatureExtractor) GetName() string {
	return fe.name
}

// GetConfig 获取配置 - 实现 Processor 接口
func (fe *FeatureExtractor) GetConfig() interfaces.ProcessorConfig {
	fe.mu.RLock()
	defer fe.mu.RUnlock()
	return fe.config
}

// Initialize 初始化处理器 - 实现 Processor 接口
func (fe *FeatureExtractor) Initialize(config interfaces.ProcessorConfig) error {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	fe.config = config

	// 解析配置
	if features, ok := config.Parameters["features"].([]interface{}); ok {
		// 清空现有特征
		fe.features = make([]FeatureExtractorFunc, 0)
		fe.featureMap = make(map[string]int)

		// 添加指定的特征
		for _, f := range features {
			if featureName, ok := f.(string); ok {
				extractor := getFeatureExtractor(featureName)
				if extractor != nil {
					fe.AddFeature(featureName, extractor)
				}
			}
		}
	}

	if historyWindow, ok := config.Parameters["history_window"].(string); ok {
		duration, err := time.ParseDuration(historyWindow)
		if err != nil {
			return fmt.Errorf("invalid history_window: %w", err)
		}
		fe.historyWindow = duration
	}

	return nil
}

// Close 关闭处理器 - 实现 Processor 接口
func (fe *FeatureExtractor) Close() error {
	fe.bufferMu.Lock()
	defer fe.bufferMu.Unlock()

	// 清空历史缓冲区
	fe.historyBuffer = make([]interfaces.DataSample, 0)

	return nil
}

// getFeatureExtractor 根据名称获取特征提取器
func getFeatureExtractor(name string) FeatureExtractorFunc {
	extractors := map[string]FeatureExtractorFunc{
		"numeric_mean":    extractNumericMean,
		"numeric_stddev":  extractNumericStdDev,
		"numeric_min":     extractNumericMin,
		"numeric_max":     extractNumericMax,
		"rate_of_change":  extractRateOfChange,
		"trend":           extractTrend,
		"variance":        extractVariance,
		"quality":         extractQuality,
		"confidence":      extractConfidence,
		"source_count":    extractSourceCount,
	}

	return extractors[name]
}

// 辅助函数

// extractNumericValues 从数据样本中提取所有数值
func extractNumericValues(sample interfaces.DataSample) []float64 {
	values := make([]float64, 0)
	data := sample.GetData()

	for _, v := range data {
		switch val := v.(type) {
		case float64:
			values = append(values, val)
		case float32:
			values = append(values, float64(val))
		case int:
			values = append(values, float64(val))
		case int64:
			values = append(values, float64(val))
		case int32:
			values = append(values, float64(val))
		}
	}

	return values
}

// calculateMean 计算平均值
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateStdDev 计算标准差
func calculateStdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := calculateMean(values)
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

// calculateMin 计算最小值
func calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// calculateMax 计算最大值
func calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// calculateTrend 计算趋势（简单线性回归斜率）
func calculateTrend(values []float64) float64 {
	n := len(values)
	if n < 2 {
		return 0
	}

	// 计算 x 的平均值（时间索引）
	xMean := float64(n-1) / 2.0

	// 计算 y 的平均值
	yMean := calculateMean(values)

	// 计算斜率
	numerator := 0.0
	denominator := 0.0
	for i, y := range values {
		x := float64(i)
		numerator += (x - xMean) * (y - yMean)
		denominator += (x - xMean) * (x - xMean)
	}

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// ExtractFeaturesFromSample 从样本中提取特征的便捷函数
func ExtractFeaturesFromSample(sample interfaces.DataSample) (*interfaces.Tensor, error) {
	extractor := NewFeatureExtractor("default")

	// 添加所有预定义特征
	extractor.AddFeature("numeric_mean", extractNumericMean)
	extractor.AddFeature("numeric_stddev", extractNumericStdDev)
	extractor.AddFeature("numeric_min", extractNumericMin)
	extractor.AddFeature("numeric_max", extractNumericMax)
	extractor.AddFeature("rate_of_change", extractRateOfChange)
	extractor.AddFeature("trend", extractTrend)
	extractor.AddFeature("variance", extractVariance)
	extractor.AddFeature("quality", extractQuality)
	extractor.AddFeature("confidence", extractConfidence)
	extractor.AddFeature("source_count", extractSourceCount)

	return extractor.extractFeatures(sample)
}

// NormalizeFeatures 归一化特征向量（Min-Max归一化到0-1）
func NormalizeFeatures(features *interfaces.Tensor) *interfaces.Tensor {
	if len(features.Data) == 0 {
		return features
	}

	values := make([]float64, len(features.Data))
	copy(values, features.Data)

	min := calculateMin(values)
	max := calculateMax(values)

	if max-min == 0 {
		return features
	}

	normalized := make([]float64, len(values))
	for i, v := range values {
		normalized[i] = (v - min) / (max - min)
	}

	return &interfaces.Tensor{
		Shape: features.Shape,
		Data:  normalized,
	}
}

// StandardizeFeatures 标准化特征向量（Z-score标准化）
func StandardizeFeatures(features *interfaces.Tensor) *interfaces.Tensor {
	if len(features.Data) == 0 {
		return features
	}

	values := make([]float64, len(features.Data))
	copy(values, features.Data)

	mean := calculateMean(values)
	stdDev := calculateStdDev(values)

	if stdDev == 0 {
		return features
	}

	standardized := make([]float64, len(values))
	for i, v := range values {
		standardized[i] = (v - mean) / stdDev
	}

	return &interfaces.Tensor{
		Shape: features.Shape,
		Data:  standardized,
	}
}

// GetFeatureNames 获取所有特征名称
func (fe *FeatureExtractor) GetFeatureNames() []string {
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	names := make([]string, len(fe.featureMap))
	for name, idx := range fe.featureMap {
		names[idx] = name
	}

	return names
}

// GetFeatureImportance 获取特征重要性（基于方差）
func (fe *FeatureExtractor) GetFeatureImportance() map[string]float64 {
	fe.bufferMu.RLock()
	defer fe.bufferMu.RUnlock()

	if len(fe.historyBuffer) < 2 {
		return make(map[string]float64)
	}

	// 提取所有历史样本的特征
	featureVectors := make([][]float64, 0, len(fe.historyBuffer))
	for _, sample := range fe.historyBuffer {
		tensor, err := fe.extractFeatures(sample)
		if err == nil {
			featureVectors = append(featureVectors, tensor.Data)
		}
	}

	if len(featureVectors) < 2 {
		return make(map[string]float64)
	}

	// 计算每个特征的方差
	featureCount := len(featureVectors[0])
	variances := make([]float64, featureCount)

	for i := 0; i < featureCount; i++ {
		values := make([]float64, len(featureVectors))
		for j, fv := range featureVectors {
			if i < len(fv) {
				values[j] = fv[i]
			}
		}
		mean := calculateMean(values)
		variance := 0.0
		for _, v := range values {
			diff := v - mean
			variance += diff * diff
		}
		variances[i] = variance / float64(len(values))
	}

	// 归一化方差为重要性分数
	totalVariance := 0.0
	for _, v := range variances {
		totalVariance += v
	}

	importance := make(map[string]float64)
	names := fe.GetFeatureNames()
	for i, name := range names {
		if totalVariance > 0 {
			importance[name] = variances[i] / totalVariance
		} else {
			importance[name] = 1.0 / float64(len(names))
		}
	}

	return importance
}
