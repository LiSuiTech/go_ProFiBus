package multimodal

import (
	"context"
	"fmt"
	"go_ProFiBus/collector"
	"go_ProFiBus/errors"
	"go_ProFiBus/fusion"
	"go_ProFiBus/inference"
	"go_ProFiBus/logger"
	"sync"
	"time"
)

// ModalityType 模态类型
type ModalityType int

const (
	ModalityTimeSeries ModalityType = iota // 时序数据
	ModalityImage                           // 图像数据
	ModalityAudio                           // 音频数据
	ModalityText                            // 文本数据
	ModalityVideo                           // 视频数据
	ModalitySensor                          // 传感器数据
	ModalityEvent                           // 事件数据
)

var modalityNames = map[ModalityType]string{
	ModalityTimeSeries: "时序数据",
	ModalityImage:      "图像数据",
	ModalityAudio:      "音频数据",
	ModalityText:       "文本数据",
	ModalityVideo:      "视频数据",
	ModalitySensor:     "传感器数据",
	ModalityEvent:      "事件数据",
}

// ModalityData 模态数据
type ModalityData struct {
	Type       ModalityType           // 模态类型
	Timestamp  time.Time              // 时间戳
	SourceID   string                 // 数据源ID
	RawData    interface{}            // 原始数据
	Features   map[string]interface{} // 提取的特征
	Embedding  []float64              // 嵌入向量
	Confidence float64                // 置信度
	Metadata   map[string]interface{} // 元数据
}

// MultiModalAnalyzer 多模态分析器
type MultiModalAnalyzer struct {
	modalities        map[ModalityType][]*ModalityData
	fusionEngine      *fusion.DataFusion
	inferenceEngine   *inference.InferenceEngine
	featureExtractors map[ModalityType]FeatureExtractor
	alignmentStrategy AlignmentStrategy
	mu                sync.RWMutex
	running           bool
	ctx               context.Context
	cancel            context.CancelFunc
	log               *logger.Logger
}

// FeatureExtractor 特征提取器接口
type FeatureExtractor interface {
	Extract(data interface{}) (map[string]interface{}, error)
	GetFeatureDimension() int
}

// AlignmentStrategy 对齐策略
type AlignmentStrategy int

const (
	AlignmentNearestNeighbor AlignmentStrategy = iota // 最近邻对齐
	AlignmentLinearInterp                              // 线性插值对齐
	AlignmentDTW                                       // 动态时间规整
	AlignmentCrossCorrelation                          // 互相关对齐
)

// NewMultiModalAnalyzer 创建多模态分析器
func NewMultiModalAnalyzer(alignment AlignmentStrategy) *MultiModalAnalyzer {
	ctx, cancel := context.WithCancel(context.Background())

	return &MultiModalAnalyzer{
		modalities:        make(map[ModalityType][]*ModalityData),
		fusionEngine:      fusion.NewDataFusion(fusion.StrategyWeighted, 1*time.Second),
		inferenceEngine:   inference.NewInferenceEngine(),
		featureExtractors: make(map[ModalityType]FeatureExtractor),
		alignmentStrategy: alignment,
		ctx:               ctx,
		cancel:            cancel,
		log:               logger.GetLogger(),
	}
}

// RegisterFeatureExtractor 注册特征提取器
func (mma *MultiModalAnalyzer) RegisterFeatureExtractor(modality ModalityType, extractor FeatureExtractor) {
	mma.mu.Lock()
	defer mma.mu.Unlock()

	mma.featureExtractors[modality] = extractor
	mma.log.Info("注册特征提取器: %s", modalityNames[modality])
}

// AddModalityData 添加模态数据
func (mma *MultiModalAnalyzer) AddModalityData(data *ModalityData) error {
	mma.mu.Lock()
	defer mma.mu.Unlock()

	// 提取特征
	if extractor, exists := mma.featureExtractors[data.Type]; exists {
		features, err := extractor.Extract(data.RawData)
		if err != nil {
			mma.log.Error("特征提取失败 [%s]: %v", modalityNames[data.Type], err)
			return err
		}
		data.Features = features
	}

	// 添加到对应模态
	mma.modalities[data.Type] = append(mma.modalities[data.Type], data)

	// 限制缓存大小
	maxCacheSize := 1000
	if len(mma.modalities[data.Type]) > maxCacheSize {
		mma.modalities[data.Type] = mma.modalities[data.Type][1:]
	}

	return nil
}

// AlignModalities 对齐多模态数据
func (mma *MultiModalAnalyzer) AlignModalities(targetTime time.Time) (map[ModalityType]*ModalityData, error) {
	mma.mu.RLock()
	defer mma.mu.RUnlock()

	aligned := make(map[ModalityType]*ModalityData)

	for modality, dataList := range mma.modalities {
		if len(dataList) == 0 {
			continue
		}

		switch mma.alignmentStrategy {
		case AlignmentNearestNeighbor:
			nearest := findNearestData(targetTime, dataList)
			aligned[modality] = nearest

		case AlignmentLinearInterp:
			interpolated, err := interpolateData(targetTime, dataList)
			if err != nil {
				mma.log.Warn("插值失败 [%s]: %v", modalityNames[modality], err)
				continue
			}
			aligned[modality] = interpolated

		default:
			nearest := findNearestData(targetTime, dataList)
			aligned[modality] = nearest
		}
	}

	return aligned, nil
}

// FuseModalities 融合多模态数据
func (mma *MultiModalAnalyzer) FuseModalities(alignedData map[ModalityType]*ModalityData) (*MultiModalFusionResult, error) {
	if len(alignedData) == 0 {
		return nil, errors.Newf(errors.ErrInvalidParam, "无可融合的模态数据")
	}

	result := &MultiModalFusionResult{
		Timestamp:       time.Now(),
		Modalities:      make(map[ModalityType]*ModalityData),
		FusedFeatures:   make(map[string]interface{}),
		FusedEmbedding:  make([]float64, 0),
		ModalityWeights: make(map[ModalityType]float64),
	}

	totalConfidence := 0.0
	embeddingSize := 0

	// 收集所有模态的特征和嵌入
	for modality, data := range alignedData {
		result.Modalities[modality] = data
		totalConfidence += data.Confidence

		// 合并特征
		for key, value := range data.Features {
			featureKey := fmt.Sprintf("%s_%s", modalityNames[modality], key)
			result.FusedFeatures[featureKey] = value
		}

		// 累加嵌入向量长度
		embeddingSize += len(data.Embedding)
	}

	// 归一化权重（基于置信度）
	for modality, data := range alignedData {
		if totalConfidence > 0 {
			result.ModalityWeights[modality] = data.Confidence / totalConfidence
		} else {
			result.ModalityWeights[modality] = 1.0 / float64(len(alignedData))
		}
	}

	// 拼接嵌入向量（可以使用更复杂的融合策略）
	result.FusedEmbedding = make([]float64, embeddingSize)
	offset := 0
	for modality, data := range alignedData {
		weight := result.ModalityWeights[modality]
		for i, val := range data.Embedding {
			result.FusedEmbedding[offset+i] = val * weight
		}
		offset += len(data.Embedding)
	}

	// 计算总体置信度
	result.Confidence = totalConfidence / float64(len(alignedData))

	mma.log.Debug("融合了 %d 个模态，总置信度: %.2f", len(alignedData), result.Confidence)

	return result, nil
}

// Analyze 执行多模态分析
func (mma *MultiModalAnalyzer) Analyze(targetTime time.Time, modelName string) (*AnalysisResult, error) {
	// 1. 对齐模态数据
	alignedData, err := mma.AlignModalities(targetTime)
	if err != nil {
		return nil, err
	}

	if len(alignedData) == 0 {
		return nil, errors.Newf(errors.ErrDataMismatch, "没有可分析的模态数据")
	}

	// 2. 融合模态数据
	fusionResult, err := mma.FuseModalities(alignedData)
	if err != nil {
		return nil, err
	}

	// 3. 执行推理（如果指定了模型）
	var prediction *inference.Tensor
	if modelName != "" {
		inputTensor, err := inference.NewTensor(
			[]int{1, len(fusionResult.FusedEmbedding)},
			fusionResult.FusedEmbedding,
		)
		if err != nil {
			return nil, err
		}

		prediction, err = mma.inferenceEngine.Predict(modelName, inputTensor)
		if err != nil {
			mma.log.Warn("推理失败: %v", err)
			// 推理失败不影响整体分析结果
		}
	}

	// 4. 构建分析结果
	analysisResult := &AnalysisResult{
		Timestamp:     targetTime,
		FusionResult:  fusionResult,
		Prediction:    prediction,
		ModalityCount: len(alignedData),
		Confidence:    fusionResult.Confidence,
		Insights:      make(map[string]interface{}),
	}

	// 5. 生成洞察
	analysisResult.Insights["modality_diversity"] = calculateModalityDiversity(alignedData)
	analysisResult.Insights["temporal_coherence"] = calculateTemporalCoherence(alignedData, targetTime)
	analysisResult.Insights["data_quality"] = fusionResult.Confidence

	return analysisResult, nil
}

// AnalyzeStream 流式多模态分析
func (mma *MultiModalAnalyzer) AnalyzeStream(interval time.Duration, modelName string) (<-chan *AnalysisResult, error) {
	mma.mu.Lock()
	if mma.running {
		mma.mu.Unlock()
		return nil, errors.Newf(errors.ErrInvalidParam, "分析器已在运行")
	}
	mma.running = true
	mma.mu.Unlock()

	resultChan := make(chan *AnalysisResult, 100)

	go func() {
		defer close(resultChan)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-mma.ctx.Done():
				return
			case <-ticker.C:
				result, err := mma.Analyze(time.Now(), modelName)
				if err != nil {
					mma.log.Error("流式分析失败: %v", err)
					continue
				}

				select {
				case resultChan <- result:
				case <-mma.ctx.Done():
					return
				default:
					mma.log.Warn("结果通道已满，丢弃分析结果")
				}
			}
		}
	}()

	return resultChan, nil
}

// Stop 停止流式分析
func (mma *MultiModalAnalyzer) Stop() {
	mma.mu.Lock()
	defer mma.mu.Unlock()

	if !mma.running {
		return
	}

	mma.cancel()
	mma.running = false
	mma.log.Info("多模态分析器已停止")
}

// MultiModalFusionResult 多模态融合结果
type MultiModalFusionResult struct {
	Timestamp       time.Time                      // 时间戳
	Modalities      map[ModalityType]*ModalityData // 参与融合的模态数据
	FusedFeatures   map[string]interface{}         // 融合后的特征
	FusedEmbedding  []float64                      // 融合后的嵌入向量
	ModalityWeights map[ModalityType]float64       // 各模态的权重
	Confidence      float64                        // 融合置信度
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Timestamp     time.Time                   // 时间戳
	FusionResult  *MultiModalFusionResult     // 融合结果
	Prediction    *inference.Tensor           // 预测结果
	ModalityCount int                         // 模态数量
	Confidence    float64                     // 总体置信度
	Insights      map[string]interface{}      // 分析洞察
}

// findNearestData 查找最接近目标时间的数据
func findNearestData(targetTime time.Time, dataList []*ModalityData) *ModalityData {
	if len(dataList) == 0 {
		return nil
	}

	nearest := dataList[0]
	minDiff := abs(targetTime.Sub(dataList[0].Timestamp))

	for _, data := range dataList[1:] {
		diff := abs(targetTime.Sub(data.Timestamp))
		if diff < minDiff {
			minDiff = diff
			nearest = data
		}
	}

	return nearest
}

// interpolateData 线性插值数据
func interpolateData(targetTime time.Time, dataList []*ModalityData) (*ModalityData, error) {
	if len(dataList) < 2 {
		if len(dataList) == 1 {
			return dataList[0], nil
		}
		return nil, errors.Newf(errors.ErrInvalidParam, "数据不足")
	}

	// 找到目标时间前后的数据
	var before, after *ModalityData
	for i, data := range dataList {
		if data.Timestamp.Before(targetTime) || data.Timestamp.Equal(targetTime) {
			before = data
		}
		if data.Timestamp.After(targetTime) || data.Timestamp.Equal(targetTime) {
			after = data
			if i > 0 && before == nil {
				before = dataList[i-1]
			}
			break
		}
	}

	if before == nil {
		return dataList[0], nil
	}
	if after == nil {
		return dataList[len(dataList)-1], nil
	}
	if before.Timestamp.Equal(targetTime) {
		return before, nil
	}
	if after.Timestamp.Equal(targetTime) {
		return after, nil
	}

	// 线性插值
	totalDuration := after.Timestamp.Sub(before.Timestamp).Seconds()
	targetDuration := targetTime.Sub(before.Timestamp).Seconds()
	ratio := targetDuration / totalDuration

	interpolated := &ModalityData{
		Type:       before.Type,
		Timestamp:  targetTime,
		SourceID:   before.SourceID,
		Features:   make(map[string]interface{}),
		Embedding:  make([]float64, len(before.Embedding)),
		Confidence: before.Confidence*(1-ratio) + after.Confidence*ratio,
		Metadata:   before.Metadata,
	}

	// 插值嵌入向量
	for i := range before.Embedding {
		if i < len(after.Embedding) {
			interpolated.Embedding[i] = before.Embedding[i]*(1-ratio) + after.Embedding[i]*ratio
		} else {
			interpolated.Embedding[i] = before.Embedding[i]
		}
	}

	// 插值特征
	for key := range before.Features {
		beforeVal, beforeOk := before.Features[key].(float64)
		afterVal, afterOk := after.Features[key].(float64)

		if beforeOk && afterOk {
			interpolated.Features[key] = beforeVal*(1-ratio) + afterVal*ratio
		} else {
			if ratio < 0.5 {
				interpolated.Features[key] = before.Features[key]
			} else {
				interpolated.Features[key] = after.Features[key]
			}
		}
	}

	return interpolated, nil
}

// abs 绝对值
func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// calculateModalityDiversity 计算模态多样性
func calculateModalityDiversity(alignedData map[ModalityType]*ModalityData) float64 {
	// 简单实现：归一化的模态数量
	maxModalities := 7.0 // 总共定义了7种模态类型
	return float64(len(alignedData)) / maxModalities
}

// calculateTemporalCoherence 计算时间一致性
func calculateTemporalCoherence(alignedData map[ModalityType]*ModalityData, targetTime time.Time) float64 {
	if len(alignedData) == 0 {
		return 0.0
	}

	totalDeviation := 0.0
	for _, data := range alignedData {
		deviation := abs(targetTime.Sub(data.Timestamp)).Seconds()
		totalDeviation += deviation
	}

	avgDeviation := totalDeviation / float64(len(alignedData))

	// 转换为一致性分数 (0-1)，假设1秒内的偏差为完全一致
	coherence := 1.0 / (1.0 + avgDeviation)
	return coherence
}

// SensorFeatureExtractor 传感器特征提取器
type SensorFeatureExtractor struct {
	dimension int
}

// NewSensorFeatureExtractor 创建传感器特征提取器
func NewSensorFeatureExtractor() *SensorFeatureExtractor {
	return &SensorFeatureExtractor{
		dimension: 10,
	}
}

// Extract 提取特征
func (sfe *SensorFeatureExtractor) Extract(data interface{}) (map[string]interface{}, error) {
	// 假设输入是DataSample
	sample, ok := data.(*collector.DataSample)
	if !ok {
		return nil, errors.Newf(errors.ErrInvalidParam, "无效的数据类型")
	}

	features := make(map[string]interface{})

	// 提取基本统计特征
	if len(sample.RawData) > 0 {
		features["data_length"] = len(sample.RawData)
		features["quality"] = sample.Quality

		// 计算简单的统计量
		sum := 0.0
		for _, b := range sample.RawData {
			sum += float64(b)
		}
		features["mean"] = sum / float64(len(sample.RawData))
	}

	// 从解析数据中提取特征
	for key, value := range sample.ParsedData {
		features[key] = value
	}

	return features, nil
}

// GetFeatureDimension 获取特征维度
func (sfe *SensorFeatureExtractor) GetFeatureDimension() int {
	return sfe.dimension
}

// TimeSeriesFeatureExtractor 时序数据特征提取器
type TimeSeriesFeatureExtractor struct {
	windowSize int
	dimension  int
}

// NewTimeSeriesFeatureExtractor 创建时序特征提取器
func NewTimeSeriesFeatureExtractor(windowSize int) *TimeSeriesFeatureExtractor {
	return &TimeSeriesFeatureExtractor{
		windowSize: windowSize,
		dimension:  20,
	}
}

// Extract 提取时序特征
func (tsfe *TimeSeriesFeatureExtractor) Extract(data interface{}) (map[string]interface{}, error) {
	values, ok := data.([]float64)
	if !ok {
		return nil, errors.Newf(errors.ErrInvalidParam, "期望 []float64 类型")
	}

	if len(values) == 0 {
		return nil, errors.Newf(errors.ErrInvalidParam, "数据为空")
	}

	features := make(map[string]interface{})

	// 计算统计特征
	features["mean"] = calculateMean(values)
	features["std"] = calculateStd(values)
	features["min"] = calculateMin(values)
	features["max"] = calculateMax(values)
	features["range"] = features["max"].(float64) - features["min"].(float64)

	// 计算趋势
	features["trend"] = calculateTrend(values)

	// 计算波动性
	features["volatility"] = calculateVolatility(values)

	return features, nil
}

// GetFeatureDimension 获取特征维度
func (tsfe *TimeSeriesFeatureExtractor) GetFeatureDimension() int {
	return tsfe.dimension
}

// 统计计算辅助函数
func calculateMean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStd(values []float64) float64 {
	mean := calculateMean(values)
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return variance / float64(len(values))
}

func calculateMin(values []float64) float64 {
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func calculateMax(values []float64) float64 {
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func calculateTrend(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}
	// 简单的线性趋势：最后值 - 第一值
	return values[len(values)-1] - values[0]
}

func calculateVolatility(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}

	changes := make([]float64, len(values)-1)
	for i := 1; i < len(values); i++ {
		changes[i-1] = values[i] - values[i-1]
	}

	return calculateStd(changes)
}
