package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/collector"
	"go_ProFiBus/fusion"
	"go_ProFiBus/pkg/interfaces"
)

// FusionProcessor 多数据源融合处理器
// 支持多种融合策略：加权、卡尔曼滤波、贝叶斯融合等
type FusionProcessor struct {
	name           string
	fusionEngine   *fusion.DataFusion
	strategy       fusion.FusionStrategy
	sourceWeights  map[string]float64 // 数据源权重
	timeWindow     time.Duration       // 时间窗口
	bufferSize     int                 // 缓冲区大小

	// 数据缓冲区 - 用于收集多个数据源的数据
	buffer         map[string]interfaces.DataSample
	bufferMu       sync.RWMutex

	// 配置
	config         interfaces.ProcessorConfig
	mu             sync.RWMutex
}

// FusionConfig 融合处理器配置
type FusionConfig struct {
	Strategy      string             `json:"strategy"`       // 融合策略: weighted, kalman, bayesian等
	SourceWeights map[string]float64 `json:"source_weights"` // 数据源权重
	TimeWindow    string             `json:"time_window"`    // 时间窗口: "1s", "500ms"等
	BufferSize    int                `json:"buffer_size"`    // 缓冲区大小
	MinSources    int                `json:"min_sources"`    // 最小数据源数量
}

// NewFusionProcessor 创建融合处理器
func NewFusionProcessor(name string, strategy fusion.FusionStrategy) *FusionProcessor {
	fp := &FusionProcessor{
		name:          name,
		strategy:      strategy,
		sourceWeights: make(map[string]float64),
		timeWindow:    1 * time.Second,
		bufferSize:    100,
		buffer:        make(map[string]interfaces.DataSample),
	}

	// 创建融合引擎
	fp.fusionEngine = fusion.NewDataFusion(strategy, fp.timeWindow)

	return fp
}

// Process 处理数据样本 - 实现 Processor 接口
func (fp *FusionProcessor) Process(ctx context.Context, input interfaces.DataSample) (interfaces.DataSample, error) {
	fp.bufferMu.Lock()
	defer fp.bufferMu.Unlock()

	// 1. 将数据加入缓冲区
	sourceID := input.GetSourceID()
	fp.buffer[sourceID] = input

	// 2. 检查是否有足够的数据源进行融合
	if len(fp.buffer) < 2 {
		// 数据源不足，直接返回
		return input, nil
	}

	// 3. 执行数据融合
	fusedData, err := fp.fuseBufferedData()
	if err != nil {
		return input, fmt.Errorf("fusion failed: %w", err)
	}

	// 4. 转换为 DataSample
	fusedSample := fp.convertToDataSample(fusedData)

	// 5. 清理过期数据
	fp.cleanExpiredData()

	return fusedSample, nil
}

// fuseBufferedData 融合缓冲区中的数据
func (fp *FusionProcessor) fuseBufferedData() (*fusion.FusedData, error) {
	// 清空融合引擎
	fp.fusionEngine = fusion.NewDataFusion(fp.strategy, fp.timeWindow)

	// 添加所有缓冲的数据源
	for sourceID, sample := range fp.buffer {
		weight := fp.getSourceWeight(sourceID)

		// 转换为collector.DataSample (fusion 包使用的格式)
		oldSample := &collector.DataSample{
			Timestamp:  sample.GetTimestamp(),
			SourceID:   sample.GetSourceID(),
			Protocol:   "",
			RawData:    nil,
			ParsedData: sample.GetData(),
			Quality:    sample.GetQuality(),
		}

		// 添加数据源到融合引擎
		err := fp.fusionEngine.AddDataSource(oldSample, weight)
		if err != nil {
			return nil, fmt.Errorf("add data source %s failed: %w", sourceID, err)
		}
	}

	// 执行融合
	fusedData, err := fp.fusionEngine.Fuse()
	if err != nil {
		return nil, fmt.Errorf("fuse execution failed: %w", err)
	}

	return fusedData, nil
}

// convertToDataSample 将融合数据转换为 DataSample
func (fp *FusionProcessor) convertToDataSample(fusedData *fusion.FusedData) interfaces.DataSample {
	// 收集所有源ID
	sourceIDs := fusedData.SourceIDs
	fusedSourceID := fmt.Sprintf("fused[%v]", sourceIDs)

	// 创建元数据
	metadata := map[string]interface{}{
		"fusion_strategy":   fp.strategy.String(),
		"source_ids":        sourceIDs,
		"source_count":      fusedData.SourceCount,
		"confidence":        fusedData.Confidence,
		"fusion_weights":    fusedData.FusionWeight,
		"original_quality":  fusedData.Quality,
	}

	// 创建新的 DataSample
	sample := &dataSampleImpl{
		timestamp: fusedData.Timestamp,
		sourceID:  fusedSourceID,
		data:      fusedData.Data,
		quality:   fusedData.Confidence, // 使用置信度作为质量分数
		metadata:  metadata,
	}

	return sample
}

// getSourceWeight 获取数据源权重
func (fp *FusionProcessor) getSourceWeight(sourceID string) float64 {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	if weight, exists := fp.sourceWeights[sourceID]; exists {
		return weight
	}

	// 默认权重
	return 1.0
}

// SetSourceWeight 设置数据源权重
func (fp *FusionProcessor) SetSourceWeight(sourceID string, weight float64) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.sourceWeights[sourceID] = weight

	// 更新融合引擎中的权重
	if fp.fusionEngine != nil {
		fp.fusionEngine.SetWeight(sourceID, weight)
	}
}

// cleanExpiredData 清理过期数据
func (fp *FusionProcessor) cleanExpiredData() {
	now := time.Now()
	for sourceID, sample := range fp.buffer {
		if now.Sub(sample.GetTimestamp()) > fp.timeWindow {
			delete(fp.buffer, sourceID)
		}
	}
}

// GetName 获取处理器名称 - 实现 Processor 接口
func (fp *FusionProcessor) GetName() string {
	return fp.name
}

// GetConfig 获取配置 - 实现 Processor 接口
func (fp *FusionProcessor) GetConfig() interfaces.ProcessorConfig {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	return fp.config
}

// Initialize 初始化处理器 - 实现 Processor 接口
func (fp *FusionProcessor) Initialize(config interfaces.ProcessorConfig) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.config = config

	// 解析配置
	if strategyStr, ok := config.Parameters["strategy"].(string); ok {
		strategy, err := parseStrategy(strategyStr)
		if err != nil {
			return fmt.Errorf("invalid strategy: %w", err)
		}
		fp.strategy = strategy
		fp.fusionEngine = fusion.NewDataFusion(strategy, fp.timeWindow)
	}

	if weights, ok := config.Parameters["source_weights"].(map[string]interface{}); ok {
		for sourceID, weight := range weights {
			if w, ok := weight.(float64); ok {
				fp.sourceWeights[sourceID] = w
			}
		}
	}

	if timeWindowStr, ok := config.Parameters["time_window"].(string); ok {
		duration, err := time.ParseDuration(timeWindowStr)
		if err != nil {
			return fmt.Errorf("invalid time_window: %w", err)
		}
		fp.timeWindow = duration
	}

	if bufferSize, ok := config.Parameters["buffer_size"].(float64); ok {
		fp.bufferSize = int(bufferSize)
	}

	return nil
}

// Close 关闭处理器 - 实现 Processor 接口
func (fp *FusionProcessor) Close() error {
	fp.bufferMu.Lock()
	defer fp.bufferMu.Unlock()

	// 清空缓冲区
	fp.buffer = make(map[string]interfaces.DataSample)

	return nil
}

// parseStrategy 解析融合策略字符串
func parseStrategy(strategyStr string) (fusion.FusionStrategy, error) {
	switch strategyStr {
	case "average":
		return fusion.StrategyAverage, nil
	case "weighted":
		return fusion.StrategyWeighted, nil
	case "kalman":
		return fusion.StrategyKalman, nil
	case "bayesian":
		return fusion.StrategyBayesian, nil
	case "dempster_shafer":
		return fusion.StrategyDempsterShafer, nil
	case "time_sync":
		return fusion.StrategyTimeSync, nil
	case "interpolation":
		return fusion.StrategyInterpolation, nil
	case "extrapolation":
		return fusion.StrategyExtrapolation, nil
	case "moving_average":
		return fusion.StrategyMovingAverage, nil
	case "exponential_sma":
		return fusion.StrategyExponentialSMA, nil
	default:
		return fusion.StrategyWeighted, fmt.Errorf("unknown strategy: %s", strategyStr)
	}
}

// dataSampleImpl DataSample接口的简单实现
type dataSampleImpl struct {
	timestamp time.Time
	sourceID  string
	data      map[string]interface{}
	quality   float64
	metadata  map[string]interface{}
}

func (ds *dataSampleImpl) GetTimestamp() time.Time                 { return ds.timestamp }
func (ds *dataSampleImpl) GetSourceID() string                     { return ds.sourceID }
func (ds *dataSampleImpl) GetData() map[string]interface{}         { return ds.data }
func (ds *dataSampleImpl) GetQuality() float64                     { return ds.quality }
func (ds *dataSampleImpl) GetMetadata() map[string]interface{}     { return ds.metadata }
