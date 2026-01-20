// Package fusion provides legacy data fusion functionality.
//
// Deprecated: This package is part of the legacy architecture.
// New code should use internal/application/orchestrator for pipeline-based processing.
package fusion

import (
	"go_ProFiBus/collector"
	"go_ProFiBus/errors"
	"go_ProFiBus/logger"
	"math"
	"sort"
	"sync"
	"time"
)

// FusionStrategy 融合策略
type FusionStrategy int

const (
	// 基础融合策略
	StrategyAverage FusionStrategy = iota // 平均融合
	StrategyWeighted                      // 加权融合
	StrategyKalman                        // 卡尔曼滤波
	StrategyBayesian                      // 贝叶斯融合
	StrategyDempsterShafer                // D-S证据理论

	// 时序融合策略
	StrategyTimeSync       // 时间同步
	StrategyInterpolation  // 插值
	StrategyExtrapolation  // 外推
	StrategyMovingAverage  // 移动平均
	StrategyExponentialSMA // 指数移动平均
)

// FusedData 融合后的数据
type FusedData struct {
	Timestamp    time.Time              // 时间戳
	SourceIDs    []string               // 参与融合的数据源ID
	Strategy     FusionStrategy         // 使用的融合策略
	Data         map[string]interface{} // 融合后的数据
	Confidence   float64                // 置信度 (0-1)
	Quality      float64                // 数据质量 (0-1)
	SourceCount  int                    // 数据源数量
	FusionWeight map[string]float64     // 各数据源的权重
}

// DataFusion 数据融合器
type DataFusion struct {
	strategy     FusionStrategy
	sources      map[string]*collector.DataSample
	weights      map[string]float64
	mu           sync.RWMutex
	timeWindow   time.Duration
	log          *logger.Logger
	kalmanFilter *KalmanFilter
}

// NewDataFusion 创建数据融合器
func NewDataFusion(strategy FusionStrategy, timeWindow time.Duration) *DataFusion {
	df := &DataFusion{
		strategy:   strategy,
		sources:    make(map[string]*collector.DataSample),
		weights:    make(map[string]float64),
		timeWindow: timeWindow,
		log:        logger.GetLogger(),
	}

	// 如果使用卡尔曼滤波，初始化滤波器
	if strategy == StrategyKalman {
		df.kalmanFilter = NewKalmanFilter(1, 1)
	}

	return df
}

// AddDataSource 添加数据源
func (df *DataFusion) AddDataSource(sample *collector.DataSample, weight float64) error {
	df.mu.Lock()
	defer df.mu.Unlock()

	if weight < 0 || weight > 1 {
		return errors.Newf(errors.ErrInvalidParam, "权重必须在 [0, 1] 范围内")
	}

	df.sources[sample.SourceID] = sample
	df.weights[sample.SourceID] = weight

	df.log.Debug("添加数据源: %s, 权重: %.2f", sample.SourceID, weight)
	return nil
}

// RemoveDataSource 移除数据源
func (df *DataFusion) RemoveDataSource(sourceID string) {
	df.mu.Lock()
	defer df.mu.Unlock()

	delete(df.sources, sourceID)
	delete(df.weights, sourceID)

	df.log.Debug("移除数据源: %s", sourceID)
}

// SetWeight 设置数据源权重
func (df *DataFusion) SetWeight(sourceID string, weight float64) error {
	df.mu.Lock()
	defer df.mu.Unlock()

	if weight < 0 || weight > 1 {
		return errors.Newf(errors.ErrInvalidParam, "权重必须在 [0, 1] 范围内")
	}

	if _, exists := df.sources[sourceID]; !exists {
		return errors.Newf(errors.ErrInvalidDataSource, "数据源不存在: %s", sourceID)
	}

	df.weights[sourceID] = weight
	return nil
}

// Fuse 执行数据融合
func (df *DataFusion) Fuse() (*FusedData, error) {
	df.mu.RLock()
	defer df.mu.RUnlock()

	if len(df.sources) == 0 {
		return nil, errors.New(errors.ErrInvalidDataSource, nil)
	}

	// 根据策略执行融合
	switch df.strategy {
	case StrategyAverage:
		return df.averageFusion()
	case StrategyWeighted:
		return df.weightedFusion()
	case StrategyKalman:
		return df.kalmanFusion()
	case StrategyMovingAverage:
		return df.movingAverageFusion()
	case StrategyExponentialSMA:
		return df.exponentialSMAFusion()
	default:
		return df.averageFusion()
	}
}

// averageFusion 平均融合
func (df *DataFusion) averageFusion() (*FusedData, error) {
	fusedData := &FusedData{
		Timestamp:    time.Now(),
		SourceIDs:    make([]string, 0, len(df.sources)),
		Strategy:     StrategyAverage,
		Data:         make(map[string]interface{}),
		SourceCount:  len(df.sources),
		FusionWeight: make(map[string]float64),
	}

	// 收集所有数据字段
	fieldValues := make(map[string][]float64)
	totalQuality := 0.0

	for sourceID, sample := range df.sources {
		fusedData.SourceIDs = append(fusedData.SourceIDs, sourceID)
		fusedData.FusionWeight[sourceID] = 1.0 / float64(len(df.sources))
		totalQuality += sample.Quality

		// 提取数值型数据
		for key, value := range sample.ParsedData {
			if floatVal, ok := value.(float64); ok {
				fieldValues[key] = append(fieldValues[key], floatVal)
			}
		}
	}

	// 计算平均值
	for field, values := range fieldValues {
		sum := 0.0
		for _, val := range values {
			sum += val
		}
		fusedData.Data[field] = sum / float64(len(values))
	}

	fusedData.Quality = totalQuality / float64(len(df.sources))
	fusedData.Confidence = calculateConfidence(fieldValues)

	return fusedData, nil
}

// weightedFusion 加权融合
func (df *DataFusion) weightedFusion() (*FusedData, error) {
	fusedData := &FusedData{
		Timestamp:    time.Now(),
		SourceIDs:    make([]string, 0, len(df.sources)),
		Strategy:     StrategyWeighted,
		Data:         make(map[string]interface{}),
		SourceCount:  len(df.sources),
		FusionWeight: make(map[string]float64),
	}

	// 归一化权重
	totalWeight := 0.0
	for sourceID := range df.sources {
		weight := df.weights[sourceID]
		totalWeight += weight
	}

	if totalWeight == 0 {
		return df.averageFusion() // 回退到平均融合
	}

	// 收集加权数据
	fieldValues := make(map[string]float64)
	fieldWeights := make(map[string]float64)
	totalQuality := 0.0

	for sourceID, sample := range df.sources {
		weight := df.weights[sourceID] / totalWeight
		fusedData.SourceIDs = append(fusedData.SourceIDs, sourceID)
		fusedData.FusionWeight[sourceID] = weight
		totalQuality += sample.Quality * weight

		// 加权累加
		for key, value := range sample.ParsedData {
			if floatVal, ok := value.(float64); ok {
				fieldValues[key] += floatVal * weight
				fieldWeights[key] += weight
			}
		}
	}

	// 归一化结果
	for field, value := range fieldValues {
		if fieldWeights[field] > 0 {
			fusedData.Data[field] = value / fieldWeights[field]
		}
	}

	fusedData.Quality = totalQuality
	fusedData.Confidence = 0.8 // 加权融合通常有较高置信度

	return fusedData, nil
}

// kalmanFusion 卡尔曼滤波融合
func (df *DataFusion) kalmanFusion() (*FusedData, error) {
	if df.kalmanFilter == nil {
		df.kalmanFilter = NewKalmanFilter(1, 1)
	}

	fusedData := &FusedData{
		Timestamp:    time.Now(),
		SourceIDs:    make([]string, 0, len(df.sources)),
		Strategy:     StrategyKalman,
		Data:         make(map[string]interface{}),
		SourceCount:  len(df.sources),
		FusionWeight: make(map[string]float64),
	}

	// 收集测量值
	measurements := make(map[string][]float64)

	for sourceID, sample := range df.sources {
		fusedData.SourceIDs = append(fusedData.SourceIDs, sourceID)
		for key, value := range sample.ParsedData {
			if floatVal, ok := value.(float64); ok {
				measurements[key] = append(measurements[key], floatVal)
			}
		}
	}

	// 对每个字段执行卡尔曼滤波
	for field, values := range measurements {
		if len(values) > 0 {
			// 使用平均值作为测量值
			avgMeasurement := 0.0
			for _, v := range values {
				avgMeasurement += v
			}
			avgMeasurement /= float64(len(values))

			// 卡尔曼滤波预测和更新
			df.kalmanFilter.Predict(0.0) // 假设无外部控制
			estimate := df.kalmanFilter.Update(avgMeasurement)

			fusedData.Data[field] = estimate
		}
	}

	fusedData.Quality = 0.9
	fusedData.Confidence = 0.95 // 卡尔曼滤波提供高置信度

	return fusedData, nil
}

// movingAverageFusion 移动平均融合
func (df *DataFusion) movingAverageFusion() (*FusedData, error) {
	// 简化实现，实际应维护历史窗口
	return df.averageFusion()
}

// exponentialSMAFusion 指数移动平均融合
func (df *DataFusion) exponentialSMAFusion() (*FusedData, error) {
	// 简化实现，实际应维护历史状态
	return df.weightedFusion()
}

// calculateConfidence 计算置信度
func calculateConfidence(fieldValues map[string][]float64) float64 {
	if len(fieldValues) == 0 {
		return 0.0
	}

	totalVariance := 0.0
	fieldCount := 0

	for _, values := range fieldValues {
		if len(values) < 2 {
			continue
		}

		// 计算方差
		mean := 0.0
		for _, v := range values {
			mean += v
		}
		mean /= float64(len(values))

		variance := 0.0
		for _, v := range values {
			diff := v - mean
			variance += diff * diff
		}
		variance /= float64(len(values))

		totalVariance += variance
		fieldCount++
	}

	if fieldCount == 0 {
		return 0.5
	}

	avgVariance := totalVariance / float64(fieldCount)

	// 方差越小，置信度越高
	confidence := 1.0 / (1.0 + avgVariance)
	return confidence
}

// KalmanFilter 卡尔曼滤波器
type KalmanFilter struct {
	stateSize       int       // 状态维度
	measurementSize int       // 测量维度
	state           []float64 // 状态估计
	covariance      []float64 // 协方差矩阵
	processNoise    []float64 // 过程噪声
	measurementNoise []float64 // 测量噪声
}

// NewKalmanFilter 创建卡尔曼滤波器
func NewKalmanFilter(stateSize, measurementSize int) *KalmanFilter {
	kf := &KalmanFilter{
		stateSize:        stateSize,
		measurementSize:  measurementSize,
		state:            make([]float64, stateSize),
		covariance:       make([]float64, stateSize*stateSize),
		processNoise:     make([]float64, stateSize*stateSize),
		measurementNoise: make([]float64, measurementSize*measurementSize),
	}

	// 初始化协方差矩阵为单位矩阵
	for i := 0; i < stateSize; i++ {
		kf.covariance[i*stateSize+i] = 1.0
		kf.processNoise[i*stateSize+i] = 0.01
	}

	for i := 0; i < measurementSize; i++ {
		kf.measurementNoise[i*measurementSize+i] = 0.1
	}

	return kf
}

// Predict 预测步骤
func (kf *KalmanFilter) Predict(control float64) {
	// 简化的一维实现
	// state = state (假设状态转移矩阵为单位矩阵)
	// covariance = covariance + processNoise
	for i := 0; i < kf.stateSize; i++ {
		kf.covariance[i] += kf.processNoise[i]
	}
}

// Update 更新步骤
func (kf *KalmanFilter) Update(measurement float64) float64 {
	// 简化的一维卡尔曼滤波
	// 计算卡尔曼增益
	innovation := measurement - kf.state[0]
	innovationCovariance := kf.covariance[0] + kf.measurementNoise[0]
	kalmanGain := kf.covariance[0] / innovationCovariance

	// 更新状态估计
	kf.state[0] += kalmanGain * innovation

	// 更新协方差
	kf.covariance[0] *= (1 - kalmanGain)

	return kf.state[0]
}

// GetState 获取当前状态估计
func (kf *KalmanFilter) GetState() []float64 {
	return kf.state
}

// TimeSynchronizer 时间同步器
type TimeSynchronizer struct {
	samples      []*collector.DataSample
	mu           sync.RWMutex
	maxDelay     time.Duration
	interpolator Interpolator
	log          *logger.Logger
}

// Interpolator 插值器接口
type Interpolator interface {
	Interpolate(t time.Time, samples []*collector.DataSample) (*collector.DataSample, error)
}

// NewTimeSynchronizer 创建时间同步器
func NewTimeSynchronizer(maxDelay time.Duration) *TimeSynchronizer {
	return &TimeSynchronizer{
		samples:      make([]*collector.DataSample, 0),
		maxDelay:     maxDelay,
		interpolator: &LinearInterpolator{},
		log:          logger.GetLogger(),
	}
}

// AddSample 添加样本
func (ts *TimeSynchronizer) AddSample(sample *collector.DataSample) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.samples = append(ts.samples, sample)

	// 按时间戳排序
	sort.Slice(ts.samples, func(i, j int) bool {
		return ts.samples[i].Timestamp.Before(ts.samples[j].Timestamp)
	})

	// 清理过期样本
	cutoff := time.Now().Add(-ts.maxDelay)
	for i, s := range ts.samples {
		if s.Timestamp.After(cutoff) {
			ts.samples = ts.samples[i:]
			break
		}
	}
}

// Synchronize 同步到指定时间
func (ts *TimeSynchronizer) Synchronize(targetTime time.Time) ([]*collector.DataSample, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if len(ts.samples) < 2 {
		return nil, errors.New(errors.ErrSyncFailed, nil)
	}

	// 按数据源分组
	sourceGroups := make(map[string][]*collector.DataSample)
	for _, sample := range ts.samples {
		sourceGroups[sample.SourceID] = append(sourceGroups[sample.SourceID], sample)
	}

	// 对每个数据源进行插值
	synchronized := make([]*collector.DataSample, 0, len(sourceGroups))
	for sourceID, samples := range sourceGroups {
		interpSample, err := ts.interpolator.Interpolate(targetTime, samples)
		if err != nil {
			ts.log.Warn("插值失败 [%s]: %v", sourceID, err)
			continue
		}
		synchronized = append(synchronized, interpSample)
	}

	return synchronized, nil
}

// LinearInterpolator 线性插值器
type LinearInterpolator struct{}

// Interpolate 线性插值
func (li *LinearInterpolator) Interpolate(t time.Time, samples []*collector.DataSample) (*collector.DataSample, error) {
	if len(samples) < 2 {
		if len(samples) == 1 {
			return samples[0], nil
		}
		return nil, errors.Newf(errors.ErrInvalidParam, "样本数量不足")
	}

	// 找到目标时间前后的样本
	var before, after *collector.DataSample
	for i, sample := range samples {
		if sample.Timestamp.Before(t) || sample.Timestamp.Equal(t) {
			before = sample
		}
		if sample.Timestamp.After(t) || sample.Timestamp.Equal(t) {
			after = sample
			if i > 0 && before == nil {
				before = samples[i-1]
			}
			break
		}
	}

	if before == nil {
		return samples[0], nil
	}
	if after == nil {
		return samples[len(samples)-1], nil
	}
	if before.Timestamp.Equal(t) {
		return before, nil
	}
	if after.Timestamp.Equal(t) {
		return after, nil
	}

	// 线性插值
	totalDuration := after.Timestamp.Sub(before.Timestamp).Seconds()
	targetDuration := t.Sub(before.Timestamp).Seconds()
	ratio := targetDuration / totalDuration

	interpolated := &collector.DataSample{
		Timestamp:  t,
		SourceID:   before.SourceID,
		Protocol:   before.Protocol,
		ParsedData: make(map[string]interface{}),
		Quality:    before.Quality*(1-ratio) + after.Quality*ratio,
	}

	// 对数值字段进行插值
	for key := range before.ParsedData {
		beforeVal, beforeOk := before.ParsedData[key].(float64)
		afterVal, afterOk := after.ParsedData[key].(float64)

		if beforeOk && afterOk {
			interpolated.ParsedData[key] = beforeVal*(1-ratio) + afterVal*ratio
		} else {
			// 非数值字段使用最近的值
			if ratio < 0.5 {
				interpolated.ParsedData[key] = before.ParsedData[key]
			} else {
				interpolated.ParsedData[key] = after.ParsedData[key]
			}
		}
	}

	return interpolated, nil
}

// MovingAverageFilter 移动平均滤波器
type MovingAverageFilter struct {
	windowSize int
	buffer     []float64
	mu         sync.Mutex
}

// NewMovingAverageFilter 创建移动平均滤波器
func NewMovingAverageFilter(windowSize int) *MovingAverageFilter {
	return &MovingAverageFilter{
		windowSize: windowSize,
		buffer:     make([]float64, 0, windowSize),
	}
}

// Add 添加新值
func (maf *MovingAverageFilter) Add(value float64) float64 {
	maf.mu.Lock()
	defer maf.mu.Unlock()

	maf.buffer = append(maf.buffer, value)
	if len(maf.buffer) > maf.windowSize {
		maf.buffer = maf.buffer[1:]
	}

	sum := 0.0
	for _, v := range maf.buffer {
		sum += v
	}

	return sum / float64(len(maf.buffer))
}

// Reset 重置滤波器
func (maf *MovingAverageFilter) Reset() {
	maf.mu.Lock()
	defer maf.mu.Unlock()
	maf.buffer = maf.buffer[:0]
}

// ExponentialMovingAverage 指数移动平均
type ExponentialMovingAverage struct {
	alpha   float64
	value   float64
	initialized bool
	mu      sync.Mutex
}

// NewExponentialMovingAverage 创建指数移动平均
func NewExponentialMovingAverage(alpha float64) *ExponentialMovingAverage {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.2 // 默认值
	}

	return &ExponentialMovingAverage{
		alpha: alpha,
	}
}

// Update 更新EMA
func (ema *ExponentialMovingAverage) Update(newValue float64) float64 {
	ema.mu.Lock()
	defer ema.mu.Unlock()

	if !ema.initialized {
		ema.value = newValue
		ema.initialized = true
	} else {
		ema.value = ema.alpha*newValue + (1-ema.alpha)*ema.value
	}

	return ema.value
}

// Get 获取当前EMA值
func (ema *ExponentialMovingAverage) Get() float64 {
	ema.mu.Lock()
	defer ema.mu.Unlock()
	return ema.value
}

// Reset 重置EMA
func (ema *ExponentialMovingAverage) Reset() {
	ema.mu.Lock()
	defer ema.mu.Unlock()
	ema.initialized = false
	ema.value = 0
}

// OutlierDetector 异常值检测器
type OutlierDetector struct {
	threshold float64
	buffer    []float64
	mu        sync.Mutex
}

// NewOutlierDetector 创建异常值检测器
func NewOutlierDetector(threshold float64) *OutlierDetector {
	return &OutlierDetector{
		threshold: threshold,
		buffer:    make([]float64, 0),
	}
}

// IsOutlier 检测是否为异常值
func (od *OutlierDetector) IsOutlier(value float64) bool {
	od.mu.Lock()
	defer od.mu.Unlock()

	if len(od.buffer) < 2 {
		od.buffer = append(od.buffer, value)
		return false
	}

	// 计算均值和标准差
	mean := 0.0
	for _, v := range od.buffer {
		mean += v
	}
	mean /= float64(len(od.buffer))

	variance := 0.0
	for _, v := range od.buffer {
		diff := v - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(len(od.buffer)))

	// 使用Z-score检测异常值
	zscore := math.Abs(value-mean) / (stddev + 1e-10)

	od.buffer = append(od.buffer, value)
	if len(od.buffer) > 100 {
		od.buffer = od.buffer[1:]
	}

	return zscore > od.threshold
}
