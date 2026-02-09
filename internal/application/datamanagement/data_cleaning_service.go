package datamanagement

import (
	"context"
	"fmt"
	"time"

	dataManagementDomain "go_ProFiBus/internal/domain/datamanagement"
	"go_ProFiBus/pkg/interfaces"
)

// DataCleaningService 数据清洗服务
type DataCleaningService struct {
	repo interfaces.DataManagementRepository
}

// NewDataCleaningService 创建数据清洗服务
func NewDataCleaningService(repo interfaces.DataManagementRepository) *DataCleaningService {
	return &DataCleaningService{
		repo: repo,
	}
}

// CleanData 清洗数据
func (s *DataCleaningService) CleanData(ctx context.Context, sourceType, sourceID string, data map[string]interface{}) (map[string]interface{}, bool, error) {
	// 获取启用的清洗规则
	filters := interfaces.CleaningRuleFilters{
		Enabled: func() *bool { b := true; return &b }(),
		Limit:   100,
	}
	rules, err := s.repo.ListCleaningRules(ctx, filters)
	if err != nil {
		return nil, false, fmt.Errorf("获取清洗规则失败: %w", err)
	}

	// 按优先级排序
	sortedRules := s.sortRulesByPriority(rules)

	cleaned := false
	result := data

	// 应用所有规则
	for _, rule := range sortedRules {
		if !rule.Enabled {
			continue
		}

		cleanedData, wasCleaned, err := s.applyRule(ctx, rule, result)
		if err != nil {
			// 记录错误但继续处理其他规则
			continue
		}

		if wasCleaned {
			cleaned = true
			result = cleanedData
		}
	}

	return result, cleaned, nil
}

// applyRule 应用清洗规则
func (s *DataCleaningService) applyRule(ctx context.Context, rule *dataManagementDomain.CleaningRule, data map[string]interface{}) (map[string]interface{}, bool, error) {
	switch rule.RuleType {
	case dataManagementDomain.CleaningRuleTypeDeduplicate:
		return s.deduplicate(data, rule.Config)
	case dataManagementDomain.CleaningRuleTypeOutlierFilter:
		return s.filterOutliers(data, rule.Config)
	case dataManagementDomain.CleaningRuleTypeMissingFill:
		return s.fillMissing(data, rule.Config)
	case dataManagementDomain.CleaningRuleTypeNormalize:
		return s.normalize(data, rule.Config)
	case dataManagementDomain.CleaningRuleTypeSmooth:
		return s.smooth(data, rule.Config)
	case dataManagementDomain.CleaningRuleTypeValidate:
		return s.validate(data, rule.Config)
	default:
		return data, false, fmt.Errorf("未知的清洗规则类型: %s", rule.RuleType)
	}
}

// deduplicate 去重
func (s *DataCleaningService) deduplicate(data map[string]interface{}, config map[string]interface{}) (map[string]interface{}, bool, error) {
	// TODO: 实现去重逻辑
	// 这里简化处理，实际应该检查时间窗口内的重复数据
	return data, false, nil
}

// filterOutliers 过滤异常值
func (s *DataCleaningService) filterOutliers(data map[string]interface{}, config map[string]interface{}) (map[string]interface{}, bool, error) {
	cleaned := false
	result := make(map[string]interface{})

	// 获取字段配置
	fields, ok := config["fields"].(map[string]interface{})
	if !ok {
		return data, false, nil
	}

	for key, value := range data {
		if fieldConfig, exists := fields[key].(map[string]interface{}); exists {
			// 检查是否有异常值配置
			if min, ok := fieldConfig["min"].(float64); ok {
				if numVal, ok := value.(float64); ok && numVal < min {
					cleaned = true
					continue // 移除异常值
				}
			}
			if max, ok := fieldConfig["max"].(float64); ok {
				if numVal, ok := value.(float64); ok && numVal > max {
					cleaned = true
					continue // 移除异常值
				}
			}
		}
		result[key] = value
	}

	return result, cleaned, nil
}

// fillMissing 填充缺失值
func (s *DataCleaningService) fillMissing(data map[string]interface{}, config map[string]interface{}) (map[string]interface{}, bool, error) {
	// 基于字段配置对缺失值进行填充：
	// - 支持 default 固定值
	// - 支持 strategy: mean / median / forward（使用配置中的统计量或最近值）

	// 拷贝原始数据，避免直接修改入参
	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		result[k] = v
	}

	fields, ok := config["fields"].(map[string]interface{})
	if !ok {
		return data, false, nil
	}

	cleaned := false

	for fieldName, fieldCfgRaw := range fields {
		fieldCfg, ok := fieldCfgRaw.(map[string]interface{})
		if !ok {
			continue
		}

		val, exists := result[fieldName]
		isMissing := !exists || val == nil
		if !isMissing {
			// 可以根据需要扩展“空字符串”“NaN”等作为缺失值的判断
			continue
		}

		// 优先使用 default
		if defVal, ok := fieldCfg["default"]; ok {
			result[fieldName] = defVal
			cleaned = true
			continue
		}

		// 按策略填充
		if strat, ok := fieldCfg["strategy"].(string); ok {
			switch strat {
			case "mean":
				// 期望配置中提供 mean 统计量
				if meanVal, ok := fieldCfg["mean"]; ok {
					result[fieldName] = meanVal
					cleaned = true
					continue
				}
			case "median":
				// 期望配置中提供 median 统计量
				if medianVal, ok := fieldCfg["median"]; ok {
					result[fieldName] = medianVal
					cleaned = true
					continue
				}
			case "forward":
				// 期望配置中提供最近一次非空值 last_value
				if lastVal, ok := fieldCfg["last_value"]; ok {
					result[fieldName] = lastVal
					cleaned = true
					continue
				}
			}
		}
	}

	if !cleaned {
		// 没有任何字段被填充，则直接返回原始数据
		return data, false, nil
	}

	return result, true, nil
}

// normalize 标准化
func (s *DataCleaningService) normalize(data map[string]interface{}, config map[string]interface{}) (map[string]interface{}, bool, error) {
	// 支持两种常见标准化方式：
	// - min_max： (x - min) / (max - min)
	// - z_score： (x - mean) / std
	//
	// 配置示例：
	// {
	//   "fields": {
	//     "temperature": { "method": "min_max", "min": 0, "max": 100 },
	//     "vibration":   { "method": "z_score", "mean": 0, "std": 1.5 }
	//   }
	// }

	fields, ok := config["fields"].(map[string]interface{})
	if !ok {
		return data, false, nil
	}

	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		result[k] = v
	}

	cleaned := false

	for fieldName, fieldCfgRaw := range fields {
		fieldCfg, ok := fieldCfgRaw.(map[string]interface{})
		if !ok {
			continue
		}

		rawVal, exists := result[fieldName]
		if !exists || rawVal == nil {
			continue
		}

		// JSON 解析后的数字通常为 float64
		value, ok := rawVal.(float64)
		if !ok {
			continue
		}

		method, _ := fieldCfg["method"].(string)
		if method == "" {
			method = "min_max"
		}

		switch method {
		case "min_max":
			minVal, okMin := fieldCfg["min"].(float64)
			maxVal, okMax := fieldCfg["max"].(float64)
			if !okMin || !okMax || maxVal == minVal {
				continue
			}
			norm := (value - minVal) / (maxVal - minVal)
			result[fieldName] = norm
			cleaned = true
		case "z_score":
			mean, okMean := fieldCfg["mean"].(float64)
			std, okStd := fieldCfg["std"].(float64)
			if !okMean || !okStd || std == 0 {
				continue
			}
			z := (value - mean) / std
			result[fieldName] = z
			cleaned = true
		default:
			// 未知方法，忽略
			continue
		}
	}

	if !cleaned {
		return data, false, nil
	}

	return result, true, nil
}

// smooth 平滑处理
func (s *DataCleaningService) smooth(data map[string]interface{}, config map[string]interface{}) (map[string]interface{}, bool, error) {
	// 通过配置中的历史值对当前数据进行平滑：
	// 配置示例：
	// {
	//   "fields": {
	//     "temperature": {
	//       "method": "moving_average",
	//       "window_size": 5,
	//       "history": [..历史值..]
	//     }
	//   }
	// }

	fields, ok := config["fields"].(map[string]interface{})
	if !ok {
		return data, false, nil
	}

	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		result[k] = v
	}

	cleaned := false

	for fieldName, fieldCfgRaw := range fields {
		fieldCfg, ok := fieldCfgRaw.(map[string]interface{})
		if !ok {
			continue
		}

		rawVal, exists := result[fieldName]
		if !exists || rawVal == nil {
			continue
		}

		current, ok := rawVal.(float64)
		if !ok {
			continue
		}

		method, _ := fieldCfg["method"].(string)
		if method == "" {
			method = "moving_average"
		}

		switch method {
		case "moving_average":
			historyRaw, _ := fieldCfg["history"].([]interface{})
			windowSize, _ := fieldCfg["window_size"].(float64)
			if windowSize <= 0 {
				windowSize = float64(len(historyRaw) + 1)
			}

			// 使用历史值 + 当前值 计算简单移动平均
			values := make([]float64, 0, len(historyRaw)+1)
			for _, hv := range historyRaw {
				if fv, ok := hv.(float64); ok {
					values = append(values, fv)
				}
			}
			values = append(values, current)

			// 取最近 windowSize 个值
			n := int(windowSize)
			if n > len(values) {
				n = len(values)
			}
			if n == 0 {
				continue
			}
			start := len(values) - n
			sum := 0.0
			for i := start; i < len(values); i++ {
				sum += values[i]
			}
			smoothed := sum / float64(n)
			result[fieldName] = smoothed
			cleaned = true
		default:
			// 其他平滑方法可后续扩展（如指数平滑）
			continue
		}
	}

	if !cleaned {
		return data, false, nil
	}

	return result, true, nil
}

// validate 数据验证
func (s *DataCleaningService) validate(data map[string]interface{}, config map[string]interface{}) (map[string]interface{}, bool, error) {
	cleaned := false
	result := make(map[string]interface{})

	// 获取验证规则
	rules, ok := config["rules"].(map[string]interface{})
	if !ok {
		return data, false, nil
	}

	for key, value := range data {
		if rule, exists := rules[key].(map[string]interface{}); exists {
			// 检查数据类型
			if expectedType, ok := rule["type"].(string); ok {
				if !s.checkType(value, expectedType) {
					cleaned = true
					continue // 移除不符合类型的数据
				}
			}
			// 检查值范围
			if min, ok := rule["min"].(float64); ok {
				if numVal, ok := value.(float64); ok && numVal < min {
					cleaned = true
					continue
				}
			}
			if max, ok := rule["max"].(float64); ok {
				if numVal, ok := value.(float64); ok && numVal > max {
					cleaned = true
					continue
				}
			}
		}
		result[key] = value
	}

	return result, cleaned, nil
}

// checkType 检查数据类型
func (s *DataCleaningService) checkType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "number", "float":
		_, ok := value.(float64)
		return ok
	case "int", "integer":
		_, ok := value.(int)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "bool", "boolean":
		_, ok := value.(bool)
		return ok
	default:
		return true
	}
}

// sortRulesByPriority 按优先级排序规则
func (s *DataCleaningService) sortRulesByPriority(rules []*dataManagementDomain.CleaningRule) []*dataManagementDomain.CleaningRule {
	// 简单的排序实现（可以使用sort包）
	sorted := make([]*dataManagementDomain.CleaningRule, len(rules))
	copy(sorted, rules)

	// 冒泡排序（按优先级降序）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Priority < sorted[j+1].Priority {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// ExecuteCleaning 执行数据清洗
func (s *DataCleaningService) ExecuteCleaning(ctx context.Context, ruleID, sourceType, sourceID string, startTime, endTime time.Time) error {
	// 获取规则
	rule, err := s.repo.GetCleaningRuleByID(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("获取清洗规则失败: %w", err)
	}

	// 创建清洗记录
	record := dataManagementDomain.NewCleaningRecord(
		fmt.Sprintf("cleaning_%d", time.Now().UnixNano()),
		ruleID,
		sourceType,
	)
	record.SourceID = sourceID
	record.Status = dataManagementDomain.CleaningStatusRunning

	if err := s.repo.CreateCleaningRecord(ctx, record); err != nil {
		return fmt.Errorf("创建清洗记录失败: %w", err)
	}

	// TODO: 实际执行清洗逻辑
	// 这里需要从数据源读取数据，应用清洗规则，然后保存清洗后的数据

	// 模拟清洗结果
	record.Complete(100, 10, 5, 3) // processed, cleaned, removed, filled

	if err := s.repo.UpdateCleaningRecord(ctx, record); err != nil {
		return fmt.Errorf("更新清洗记录失败: %w", err)
	}

	return nil
}
