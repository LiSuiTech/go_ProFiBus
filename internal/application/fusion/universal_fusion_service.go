package fusion

import (
	"context"
	"fmt"
	"sync"
	"time"

	fusionDomain "go_ProFiBus/internal/domain/fusion"
	"go_ProFiBus/pkg/interfaces"
)

// UniversalFusionService 通用融合服务
type UniversalFusionService struct {
	repo     interfaces.FusionRepository
	configs  map[string]*FusionConfigCache // 配置缓存
	configsMu sync.RWMutex
}

// FusionConfigCache 融合配置缓存
type FusionConfigCache struct {
	Config      *fusionDomain.FusionConfig
	Sources     []*fusionDomain.ConfigSourceRelation
	TimeWindow  time.Duration
	LastUpdated time.Time
}

// NewUniversalFusionService 创建通用融合服务
func NewUniversalFusionService(repo interfaces.FusionRepository) *UniversalFusionService {
	return &UniversalFusionService{
		repo:    repo,
		configs: make(map[string]*FusionConfigCache),
	}
}

// LoadFusionConfig 加载融合配置
func (s *UniversalFusionService) LoadFusionConfig(ctx context.Context, configID string) error {
	config, err := s.repo.GetFusionConfigByID(ctx, configID)
	if err != nil {
		return fmt.Errorf("获取融合配置失败: %w", err)
	}

	if !config.Enabled {
		return fmt.Errorf("融合配置未启用: %s", configID)
	}

	// 获取关联的数据源
	sources, err := s.repo.GetConfigSources(ctx, configID)
	if err != nil {
		return fmt.Errorf("获取配置数据源失败: %w", err)
	}

	// 缓存配置
	s.configsMu.Lock()
	s.configs[configID] = &FusionConfigCache{
		Config:      config,
		Sources:     sources,
		TimeWindow:  time.Duration(config.TimeWindowMs) * time.Millisecond,
		LastUpdated: time.Now(),
	}
	s.configsMu.Unlock()

	return nil
}

// SubmitData 提交数据到数据源
func (s *UniversalFusionService) SubmitData(ctx context.Context, sourceID string, data map[string]interface{}, quality float64) error {
	// 验证数据源
	source, err := s.repo.GetDataSourceByID(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("数据源不存在: %w", err)
	}

	if !source.Enabled {
		return fmt.Errorf("数据源未启用: %s", sourceID)
	}

	// 保存到缓存
	cache := fusionDomain.NewSourceDataCache(
		fmt.Sprintf("cache_%s_%d", sourceID, time.Now().UnixNano()),
		sourceID,
		data,
		quality,
	)

	if err := s.repo.SaveSourceDataCache(ctx, cache); err != nil {
		return fmt.Errorf("保存数据缓存失败: %w", err)
	}

	// 检查所有使用此数据源的融合配置，尝试触发融合
	return s.triggerFusionForSource(ctx, sourceID)
}

// triggerFusionForSource 为数据源触发融合
func (s *UniversalFusionService) triggerFusionForSource(ctx context.Context, sourceID string) error {
	s.configsMu.RLock()
	defer s.configsMu.RUnlock()

	// 遍历所有配置，检查是否包含此数据源
	for configID, cache := range s.configs {
		// 检查配置是否包含此数据源
		hasSource := false
		for _, rel := range cache.Sources {
			if rel.SourceID == sourceID && rel.Enabled {
				hasSource = true
				break
			}
		}

		if hasSource {
			// 尝试执行融合
			if err := s.performFusion(ctx, configID); err != nil {
				// 记录错误但不中断其他配置的处理
				fmt.Printf("融合配置 %s 执行失败: %v\n", configID, err)
			}
		}
	}

	return nil
}

// performFusion 执行融合
func (s *UniversalFusionService) performFusion(ctx context.Context, configID string) error {
	cache, ok := s.configs[configID]
	if !ok {
		// 配置未加载，尝试加载
		if err := s.LoadFusionConfig(ctx, configID); err != nil {
			return err
		}
		cache = s.configs[configID]
	}

	config := cache.Config
	if !config.Enabled {
		return nil // 配置未启用，跳过
	}

	// 收集时间窗口内的所有数据源数据
	sourceDataMap := make(map[string]*fusionDomain.SourceDataCache)
	now := time.Now()

	for _, rel := range cache.Sources {
		if !rel.Enabled {
			continue
		}

		// 获取数据源的最新数据（在时间窗口内）
		caches, err := s.repo.GetSourceDataCache(ctx, rel.SourceID, cache.TimeWindow)
		if err != nil {
			continue // 跳过无法获取的数据源
		}

		if len(caches) == 0 {
			continue // 没有数据
		}

		// 使用最新的数据
		latest := caches[0]
		for _, c := range caches {
			if c.Timestamp.After(latest.Timestamp) {
				latest = c
			}
		}

		// 检查数据是否在时间窗口内
		if now.Sub(latest.Timestamp) <= cache.TimeWindow {
			sourceDataMap[rel.SourceID] = latest
		}
	}

	// 检查是否有足够的数据源
	if len(sourceDataMap) < config.MinSources {
		return nil // 数据源不足，不执行融合
	}

	// 根据策略执行融合
	var fusedData map[string]interface{}
	var err error

	switch config.FusionStrategy {
	case "weighted":
		fusedData, err = s.weightedFusion(config, cache.Sources, sourceDataMap)
	case "average":
		fusedData, err = s.averageFusion(sourceDataMap)
	default:
		fusedData, err = s.weightedFusion(config, cache.Sources, sourceDataMap)
	}

	if err != nil {
		return fmt.Errorf("融合失败: %w", err)
	}

	// 过滤输出字段（如果配置了）
	if len(config.OutputFields) > 0 {
		filteredData := make(map[string]interface{})
		for _, field := range config.OutputFields {
			if value, exists := fusedData[field]; exists {
				filteredData[field] = value
			}
		}
		fusedData = filteredData
	}

	// 计算质量评分
	qualityScore := s.calculateQualityScore(sourceDataMap)

	// 保存融合结果
	sourceIDs := make([]string, 0, len(sourceDataMap))
	for id := range sourceDataMap {
		sourceIDs = append(sourceIDs, id)
	}

	result := fusionDomain.NewFusionResult(
		fmt.Sprintf("result_%s_%d", configID, now.UnixNano()),
		configID,
		config.Name,
		fusedData,
	)
	result.SourceCount = len(sourceDataMap)
	result.SourceIDs = sourceIDs
	result.FusionStrategy = config.FusionStrategy
	result.SetQualityScore(qualityScore)
	result.Metadata["source_weights"] = config.SourceWeights
	result.Metadata["field_weights"] = config.FieldWeights

	if err := s.repo.SaveFusionResult(ctx, result); err != nil {
		return fmt.Errorf("保存融合结果失败: %w", err)
	}

	return nil
}

// weightedFusion 加权融合
func (s *UniversalFusionService) weightedFusion(
	config *fusionDomain.FusionConfig,
	relations []*fusionDomain.ConfigSourceRelation,
	sourceDataMap map[string]*fusionDomain.SourceDataCache,
) (map[string]interface{}, error) {
	if len(sourceDataMap) == 0 {
		return nil, fmt.Errorf("没有数据可融合")
	}

	// 收集所有字段
	allFields := make(map[string]bool)
	for _, cache := range sourceDataMap {
		for field := range cache.Data {
			allFields[field] = true
		}
	}

	fusedData := make(map[string]interface{})

	// 对每个字段进行加权融合
	for field := range allFields {
		totalWeight := 0.0
		weightedSum := 0.0
		fieldCount := 0

		for _, rel := range relations {
			cache, exists := sourceDataMap[rel.SourceID]
			if !exists || !rel.Enabled {
				continue
			}

			if value, exists := cache.Data[field]; exists {
				// 获取数据源权重（优先使用配置中的权重，否则使用数据源默认权重）
				sourceWeight := rel.Weight
				if sourceWeight <= 0 {
					if w, ok := config.SourceWeights[rel.SourceID]; ok {
						sourceWeight = w
					} else {
						source, _ := s.repo.GetDataSourceByID(context.Background(), rel.SourceID)
						if source != nil {
							sourceWeight = source.FusionWeight
						} else {
							sourceWeight = 1.0
						}
					}
				}

				// 获取字段权重
				fieldWeight := 1.0
				if w, ok := config.FieldWeights[field]; ok {
					fieldWeight = w
				}

				// 综合权重 = 数据源权重 * 字段权重 * 质量
				weight := sourceWeight * fieldWeight * cache.Quality
				totalWeight += weight

				// 转换为数值
				if numVal, ok := value.(float64); ok {
					weightedSum += numVal * weight
					fieldCount++
				} else if intVal, ok := value.(int); ok {
					weightedSum += float64(intVal) * weight
					fieldCount++
				} else if intVal, ok := value.(int64); ok {
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
func (s *UniversalFusionService) averageFusion(sourceDataMap map[string]*fusionDomain.SourceDataCache) (map[string]interface{}, error) {
	if len(sourceDataMap) == 0 {
		return nil, fmt.Errorf("没有数据可融合")
	}

	// 收集所有字段
	allFields := make(map[string]bool)
	for _, cache := range sourceDataMap {
		for field := range cache.Data {
			allFields[field] = true
		}
	}

	fusedData := make(map[string]interface{})

	// 对每个字段进行平均
	for field := range allFields {
		sum := 0.0
		count := 0

		for _, cache := range sourceDataMap {
			if value, exists := cache.Data[field]; exists {
				if numVal, ok := value.(float64); ok {
					sum += numVal
					count++
				} else if intVal, ok := value.(int); ok {
					sum += float64(intVal)
					count++
				} else if intVal, ok := value.(int64); ok {
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
func (s *UniversalFusionService) calculateQualityScore(sourceDataMap map[string]*fusionDomain.SourceDataCache) float64 {
	if len(sourceDataMap) == 0 {
		return 0.0
	}

	totalQuality := 0.0
	for _, cache := range sourceDataMap {
		totalQuality += cache.Quality
	}

	return totalQuality / float64(len(sourceDataMap))
}

// GetFusionResults 获取融合结果
func (s *UniversalFusionService) GetFusionResults(ctx context.Context, configID string, start, end time.Time, limit int) ([]*fusionDomain.FusionResult, error) {
	filters := interfaces.FusionResultFilters{
		FusionConfigID: &configID,
		StartTime:      &start,
		EndTime:        &end,
		Limit:          limit,
	}
	return s.repo.GetFusionResults(ctx, filters)
}

// GetLatestFusionResult 获取最新融合结果
func (s *UniversalFusionService) GetLatestFusionResult(ctx context.Context, configID string) (*fusionDomain.FusionResult, error) {
	return s.repo.GetLatestFusionResult(ctx, configID)
}

// CleanExpiredCache 清理过期缓存
func (s *UniversalFusionService) CleanExpiredCache(ctx context.Context, olderThan time.Time) error {
	return s.repo.CleanExpiredCache(ctx, olderThan)
}
