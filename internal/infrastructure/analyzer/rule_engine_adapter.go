package analyzer

import (
	"context"
	"go_ProFiBus/anomaly"
	"go_ProFiBus/collector"
	"go_ProFiBus/pkg/interfaces"
)

// RuleEngineAnalyzer 规则引擎分析器适配器
type RuleEngineAnalyzer struct {
	engine *anomaly.RuleEngine
	name   string
}

// NewRuleEngineAnalyzer 创建规则引擎分析器
func NewRuleEngineAnalyzer(name string, engine *anomaly.RuleEngine) *RuleEngineAnalyzer {
	return &RuleEngineAnalyzer{
		engine: engine,
		name:   name,
	}
}

// Analyze 实现 Analyzer 接口
func (a *RuleEngineAnalyzer) Analyze(ctx context.Context, data interfaces.DataSample) ([]interfaces.AnalysisResult, error) {
	// 将新的 DataSample 转换为旧的格式
	oldSample := convertNewSampleToOld(data)

	// 使用旧的规则引擎评估
	evaluations := a.engine.Evaluate(oldSample)

	// 转换评估结果为新的 AnalysisResult
	results := make([]interfaces.AnalysisResult, 0, len(evaluations))
	for _, eval := range evaluations {
		if eval.Violated {
			result := interfaces.AnalysisResult{
				Type:      mapSeverityToType(eval.Severity),
				Severity:  mapAnomalySeverityToInt(eval.Severity),
				Score:     eval.Score,
				Message:   eval.Message,
				Details: map[string]interface{}{
					"rule_id":   eval.RuleID,
					"rule_name": eval.RuleName,
				},
				Timestamp: eval.EvaluatedAt,
				SourceID:  data.GetSourceID(),
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// GetType 实现 Analyzer 接口
func (a *RuleEngineAnalyzer) GetType() interfaces.AnalyzerType {
	return interfaces.AnalyzerTypeThreshold
}

// GetName 实现 Analyzer 接口
func (a *RuleEngineAnalyzer) GetName() string {
	return a.name
}

// Configure 实现 Analyzer 接口
func (a *RuleEngineAnalyzer) Configure(config map[string]interface{}) error {
	// 配置逻辑（如果需要）
	return nil
}

// Close 实现 Analyzer 接口
func (a *RuleEngineAnalyzer) Close() error {
	// 清理资源（如果需要）
	return nil
}

// AddRule 添加规则到引擎
func (a *RuleEngineAnalyzer) AddRule(rule anomaly.Rule) error {
	return a.engine.AddRule(rule)
}

// RemoveRule 从引擎移除规则
func (a *RuleEngineAnalyzer) RemoveRule(ruleID string) error {
	return a.engine.RemoveRule(ruleID)
}

// GetEngine 获取底层规则引擎
func (a *RuleEngineAnalyzer) GetEngine() *anomaly.RuleEngine {
	return a.engine
}

// convertNewSampleToOld 将新的 DataSample 转换为旧的格式
func convertNewSampleToOld(sample interfaces.DataSample) *collector.DataSample {
	data := sample.GetData()
	metadata := sample.GetMetadata()

	oldSample := &collector.DataSample{
		Timestamp:  sample.GetTimestamp(),
		SourceID:   sample.GetSourceID(),
		Protocol:   getStringFromMetadata(metadata, "protocol", "unknown"),
		RawData:    nil, // 原始数据不可用
		ParsedData: data,
		Quality:    sample.GetQuality(),
	}

	return oldSample
}

// getStringFromMetadata 从元数据获取字符串值
func getStringFromMetadata(metadata map[string]interface{}, key, defaultValue string) string {
	if val, ok := metadata[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// mapSeverityToType 将严重程度映射为类型
func mapSeverityToType(severity anomaly.Severity) string {
	switch severity {
	case anomaly.SeverityInfo:
		return "info"
	case anomaly.SeverityWarning:
		return "warning"
	case anomaly.SeverityError:
		return "error"
	case anomaly.SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// mapAnomalySeverityToInt 将 anomaly.Severity 转换为 int
func mapAnomalySeverityToInt(severity anomaly.Severity) int {
	return int(severity) + 1 // 1-5 范围
}

// 确保实现了接口
var _ interfaces.Analyzer = (*RuleEngineAnalyzer)(nil)
