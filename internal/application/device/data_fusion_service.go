package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	deviceDomain "go_ProFiBus/internal/domain/device"
	"go_ProFiBus/pkg/interfaces"
)

// DataFusionService 设备数据融合服务
type DataFusionService struct {
	dataRepo      interfaces.DeviceDataRepository
	fusionConfig  *deviceDomain.FusionConfig
	buffer        map[string][]*DataSampleBuffer // 数据源 -> 数据样本缓冲区
	bufferMu      sync.RWMutex
	timeWindow    time.Duration
	streamBridge  *DataStreamBridge // 数据流桥接（可选）
}

// DataSampleBuffer 数据样本缓冲区
type DataSampleBuffer struct {
	Timestamp time.Time
	Data      map[string]interface{}
	Quality   float64
	SourceID  string
}

// NewDataFusionService 创建数据融合服务
func NewDataFusionService(dataRepo interfaces.DeviceDataRepository) *DataFusionService {
	return &DataFusionService{
		dataRepo:   dataRepo,
		buffer:     make(map[string][]*DataSampleBuffer),
		timeWindow: 1 * time.Second,
	}
}

// SetStreamBridge 设置数据流桥接
func (s *DataFusionService) SetStreamBridge(bridge *DataStreamBridge) {
	s.streamBridge = bridge
}

// LoadFusionConfig 加载融合配置
func (s *DataFusionService) LoadFusionConfig(ctx context.Context, deviceID string) error {
	config, err := s.dataRepo.GetFusionConfigByDevice(ctx, deviceID)
	if err != nil {
		// 如果没有配置，创建默认配置
		config = deviceDomain.NewFusionConfig(fmt.Sprintf("fusion_%s", deviceID), deviceID)
		if err := s.dataRepo.CreateFusionConfig(ctx, config); err != nil {
			return fmt.Errorf("创建默认融合配置失败: %w", err)
		}
	}

	s.fusionConfig = config
	s.timeWindow = time.Duration(config.TimeWindowMs) * time.Millisecond
	return nil
}

// AddDataSample 添加数据样本
func (s *DataFusionService) AddDataSample(ctx context.Context, deviceID string, sourceID string, data map[string]interface{}, quality float64) error {
	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()

	// 加载融合配置（如果未加载）
	if s.fusionConfig == nil || s.fusionConfig.DeviceID != deviceID {
		if err := s.LoadFusionConfig(ctx, deviceID); err != nil {
			return err
		}
	}

	// 添加到缓冲区
	if s.buffer[sourceID] == nil {
		s.buffer[sourceID] = make([]*DataSampleBuffer, 0)
	}

	sample := &DataSampleBuffer{
		Timestamp: time.Now(),
		Data:      data,
		Quality:   quality,
		SourceID:  sourceID,
	}

	s.buffer[sourceID] = append(s.buffer[sourceID], sample)

	// 清理过期数据
	s.cleanExpiredData()

	// 检查是否可以执行融合
	if s.canFuse() {
		return s.performFusion(ctx, deviceID)
	}

	return nil
}

// canFuse 检查是否可以执行融合
func (s *DataFusionService) canFuse() bool {
	if s.fusionConfig == nil || !s.fusionConfig.Enabled {
		return false
	}

	// 检查是否有足够的数据源
	validSources := 0
	now := time.Now()
	for _, samples := range s.buffer {
		if len(samples) > 0 {
			// 检查最新样本是否在时间窗口内
			latest := samples[len(samples)-1]
			if now.Sub(latest.Timestamp) <= s.timeWindow {
				validSources++
			}
		}
	}

	return validSources >= s.fusionConfig.MinSources
}

// performFusion 执行融合
func (s *DataFusionService) performFusion(ctx context.Context, deviceID string) error {
	// 收集时间窗口内的所有样本
	samples := s.collectSamplesInWindow()

	if len(samples) < s.fusionConfig.MinSources {
		return nil
	}

	// 根据策略执行融合
	var fusedData map[string]interface{}
	var err error

	switch s.fusionConfig.FusionStrategy {
	case "weighted":
		fusedData, err = s.weightedFusion(samples)
	case "average":
		fusedData, err = s.averageFusion(samples)
	default:
		fusedData, err = s.weightedFusion(samples)
	}

	if err != nil {
		return fmt.Errorf("融合失败: %w", err)
	}

	// 计算质量评分
	qualityScore := s.calculateQualityScore(samples)

	// 保存融合结果
	fused := deviceDomain.NewFusedData(
		fmt.Sprintf("fused_%s_%d", deviceID, time.Now().UnixNano()),
		deviceID,
		fusedData,
	)
	fused.SourceCount = len(samples)
	fused.FusionStrategy = s.fusionConfig.FusionStrategy
	fused.SetQualityScore(qualityScore)
	fused.Metadata["field_weights"] = s.fusionConfig.FieldWeights
	fused.Metadata["source_weights"] = s.fusionConfig.SourceWeights

	if err := s.dataRepo.SaveFusedData(ctx, fused); err != nil {
		return fmt.Errorf("保存融合数据失败: %w", err)
	}

	// 推送到WebSocket（如果配置了桥接）
	if s.streamBridge != nil {
		s.streamBridge.BroadcastFusedData(deviceID, fusedData, "", fused.GetQualityScore())
	}

	return nil
}

// collectSamplesInWindow 收集时间窗口内的样本
func (s *DataFusionService) collectSamplesInWindow() []*DataSampleBuffer {
	now := time.Now()
	samples := make([]*DataSampleBuffer, 0)

	for _, bufferSamples := range s.buffer {
		if len(bufferSamples) > 0 {
			// 获取最新的样本
			latest := bufferSamples[len(bufferSamples)-1]
			if now.Sub(latest.Timestamp) <= s.timeWindow {
				samples = append(samples, latest)
			}
		}
	}

	return samples
}

// weightedFusion 加权融合
func (s *DataFusionService) weightedFusion(samples []*DataSampleBuffer) (map[string]interface{}, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("没有样本可融合")
	}

	// 收集所有字段
	allFields := make(map[string]bool)
	for _, sample := range samples {
		for field := range sample.Data {
			allFields[field] = true
		}
	}

	fusedData := make(map[string]interface{})

	// 对每个字段进行加权融合
	for field := range allFields {
		totalWeight := 0.0
		weightedSum := 0.0
		fieldCount := 0

		for _, sample := range samples {
			if value, exists := sample.Data[field]; exists {
				// 获取字段权重
				fieldWeight := 1.0
				if w, ok := s.fusionConfig.FieldWeights[field]; ok {
					fieldWeight = w
				}

				// 获取数据源权重
				sourceWeight := 1.0
				if w, ok := s.fusionConfig.SourceWeights[sample.SourceID]; ok {
					sourceWeight = w
				}

				// 综合权重 = 字段权重 * 数据源权重 * 质量
				weight := fieldWeight * sourceWeight * sample.Quality
				totalWeight += weight

				// 转换为数值
				if numVal, ok := value.(float64); ok {
					weightedSum += numVal * weight
					fieldCount++
				} else if intVal, ok := value.(int); ok {
					weightedSum += float64(intVal) * weight
					fieldCount++
				}
			}
		}

		if fieldCount > 0 && totalWeight > 0 {
			fusedData[field] = weightedSum / totalWeight
		}
	}

	return fusedData, nil
}

// averageFusion 平均融合
func (s *DataFusionService) averageFusion(samples []*DataSampleBuffer) (map[string]interface{}, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("没有样本可融合")
	}

	// 收集所有字段
	allFields := make(map[string]bool)
	for _, sample := range samples {
		for field := range sample.Data {
			allFields[field] = true
		}
	}

	fusedData := make(map[string]interface{})

	// 对每个字段进行平均
	for field := range allFields {
		sum := 0.0
		count := 0

		for _, sample := range samples {
			if value, exists := sample.Data[field]; exists {
				if numVal, ok := value.(float64); ok {
					sum += numVal
					count++
				} else if intVal, ok := value.(int); ok {
					sum += float64(intVal)
					count++
				}
			}
		}

		if count > 0 {
			fusedData[field] = sum / float64(count)
		}
	}

	return fusedData, nil
}

// calculateQualityScore 计算质量评分
func (s *DataFusionService) calculateQualityScore(samples []*DataSampleBuffer) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	totalQuality := 0.0
	for _, sample := range samples {
		totalQuality += sample.Quality
	}

	return totalQuality / float64(len(samples))
}

// cleanExpiredData 清理过期数据
func (s *DataFusionService) cleanExpiredData() {
	now := time.Now()
	for sourceID, samples := range s.buffer {
		validSamples := make([]*DataSampleBuffer, 0)
		for _, sample := range samples {
			if now.Sub(sample.Timestamp) <= s.timeWindow*2 { // 保留2倍时间窗口的数据
				validSamples = append(validSamples, sample)
			}
		}
		s.buffer[sourceID] = validSamples
	}
}

// GetFusedData 获取融合数据
func (s *DataFusionService) GetFusedData(ctx context.Context, deviceID string, start, end time.Time, limit int) ([]*deviceDomain.FusedData, error) {
	return s.dataRepo.GetFusedDataByDevice(ctx, deviceID, start, end, limit)
}

// GetLatestFusedData 获取最新融合数据
func (s *DataFusionService) GetLatestFusedData(ctx context.Context, deviceID string) (*deviceDomain.FusedData, error) {
	return s.dataRepo.GetLatestFusedData(ctx, deviceID)
}
