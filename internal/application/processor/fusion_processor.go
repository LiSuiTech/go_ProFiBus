package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// FusionProcessor 多数据源融合处理器
// 注意：融合功能已简化，仅支持简单的加权平均融合
type FusionProcessor struct {
	name           string
	strategy       interfaces.FusionStrategy
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
func NewFusionProcessor(name string, strategy interfaces.FusionStrategy) *FusionProcessor {
	return &FusionProcessor{
		name:          name,
		strategy:      strategy,
		sourceWeights: make(map[string]float64),
		timeWindow:    1 * time.Second,
		bufferSize:    100,
		buffer:        make(map[string]interfaces.DataSample),
	}
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

	// 3. 执行简单的加权平均融合
	fusedSample, err := fp.simpleFusion()
	if err != nil {
		return input, fmt.Errorf("fusion failed: %w", err)
	}

	// 4. 清理过期数据
	fp.cleanExpiredData()

	return fusedSample, nil
}

// simpleFusion 简单的加权平均融合
func (fp *FusionProcessor) simpleFusion() (interfaces.DataSample, error) {
	if len(fp.buffer) == 0 {
		return nil, fmt.Errorf("buffer is empty")
	}

	// 收集所有样本
	samples := make([]interfaces.DataSample, 0, len(fp.buffer))
	for _, sample := range fp.buffer {
		samples = append(samples, sample)
	}

	// 计算权重总和
	totalWeight := 0.0
	for _, sample := range samples {
		sourceID := sample.GetSourceID()
		weight := 1.0 / float64(len(samples)) // 默认平均权重
		if w, ok := fp.sourceWeights[sourceID]; ok {
			weight = w
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		totalWeight = float64(len(samples))
	}

	// 融合数据
	fusedData := make(map[string]interface{})
	for _, sample := range samples {
		sourceID := sample.GetSourceID()
		weight := 1.0 / float64(len(samples))
		if w, ok := fp.sourceWeights[sourceID]; ok {
			weight = w / totalWeight
		} else {
			weight = weight / totalWeight
		}

		for key, value := range sample.GetData() {
			if numVal, ok := value.(float64); ok {
				if existing, exists := fusedData[key]; exists {
					if existingNum, ok := existing.(float64); ok {
						fusedData[key] = existingNum + numVal*weight
					}
				} else {
					fusedData[key] = numVal * weight
				}
			} else {
				// 非数值类型，使用第一个样本的值
				if _, exists := fusedData[key]; !exists {
					fusedData[key] = value
				}
			}
		}
	}

	// 计算平均质量
	totalQuality := 0.0
	for _, sample := range samples {
		totalQuality += sample.GetQuality()
	}
	avgQuality := totalQuality / float64(len(samples))

	// 创建融合后的样本
	// 使用第一个样本的时间戳和源ID
	firstSample := samples[0]
	metadata := make(map[string]interface{})
	if firstSample.GetMetadata() != nil {
		for k, v := range firstSample.GetMetadata() {
			metadata[k] = v
		}
	}
	metadata["fusion_strategy"] = string(fp.strategy)
	metadata["source_count"] = len(samples)

	// 创建新的 DataSample（需要使用 domain 包）
	// 这里简化处理，返回第一个样本的副本但更新数据
	return &simpleDataSample{
		timestamp: firstSample.GetTimestamp(),
		sourceID:  "fused",
		data:      fusedData,
		quality:   avgQuality,
		metadata:  metadata,
	}, nil
}

// simpleDataSample 简单的 DataSample 实现
type simpleDataSample struct {
	timestamp time.Time
	sourceID  string
	data      map[string]interface{}
	quality   float64
	metadata  map[string]interface{}
}

func (s *simpleDataSample) GetTimestamp() time.Time { return s.timestamp }
func (s *simpleDataSample) GetSourceID() string      { return s.sourceID }
func (s *simpleDataSample) GetData() map[string]interface{} { return s.data }
func (s *simpleDataSample) GetQuality() float64     { return s.quality }
func (s *simpleDataSample) GetMetadata() map[string]interface{} { return s.metadata }

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

// SetSourceWeight 设置数据源权重
func (fp *FusionProcessor) SetSourceWeight(sourceID string, weight float64) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.sourceWeights[sourceID] = weight
}

// parseStrategy 解析融合策略字符串
func parseStrategy(strategyStr string) (interfaces.FusionStrategy, error) {
	switch strategyStr {
	case "average":
		return interfaces.FusionStrategyAverage, nil
	case "weighted":
		return interfaces.FusionStrategyWeighted, nil
	case "kalman":
		return interfaces.FusionStrategyKalman, nil
	case "bayesian":
		return interfaces.FusionStrategyBayesian, nil
	case "dempster_shafer":
		return interfaces.FusionStrategyDempsterShafer, nil
	case "time_sync":
		return interfaces.FusionStrategyTimeSync, nil
	case "interpolation":
		return interfaces.FusionStrategyInterpolation, nil
	case "extrapolation":
		return interfaces.FusionStrategyExtrapolation, nil
	case "moving_average":
		return interfaces.FusionStrategyMovingAverage, nil
	case "exponential_sma":
		return interfaces.FusionStrategyExponentialSMA, nil
	default:
		return interfaces.FusionStrategyWeighted, fmt.Errorf("unknown strategy: %s", strategyStr)
	}
}
