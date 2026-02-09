package rule_template

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	templateDomain "go_ProFiBus/internal/domain/rule_template"
)

// RuleTestService 规则测试服务
type RuleTestService struct {
	templateRepo RuleTemplateRepository
}

// RuleTemplateRepository 规则模板仓储接口
type RuleTemplateRepository interface {
	GetTemplateByID(ctx context.Context, id string) (*templateDomain.RuleTemplate, error)
	SaveTestResult(ctx context.Context, result *templateDomain.RuleTestResult) error
}

// NewRuleTestService 创建规则测试服务
func NewRuleTestService(templateRepo RuleTemplateRepository) *RuleTestService {
	return &RuleTestService{
		templateRepo: templateRepo,
	}
}

// TestRule 测试规则（从模板或已有规则）
func (s *RuleTestService) TestRule(ctx context.Context, ruleConfig map[string]interface{}, testData map[string]interface{}) (*templateDomain.RuleTestResult, error) {
	startTime := time.Now()

	// 评估规则
	triggered, resultDetails, err := s.evaluateRule(ruleConfig, testData)
	if err != nil {
		return nil, fmt.Errorf("规则评估失败: %w", err)
	}

	executionTime := time.Since(startTime)

	testResult := templateDomain.NewRuleTestResult(testData, ruleConfig)
	testResult.Triggered = triggered
	testResult.ExecutionTimeMs = int(executionTime.Milliseconds())
	testResult.TestResult = resultDetails

	return testResult, nil
}

// TestTemplate 测试模板（应用变量后测试）
func (s *RuleTestService) TestTemplate(ctx context.Context, templateID string, variables map[string]interface{}, testData map[string]interface{}) (*templateDomain.RuleTestResult, error) {
	// 获取模板
	template, err := s.templateRepo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	// 应用变量到条件模板
	ruleConfig := s.applyVariables(template.ConditionTemplate, variables)

	// 测试规则
	testResult, err := s.TestRule(ctx, ruleConfig, testData)
	if err != nil {
		return nil, err
	}

	testResult.TemplateID = templateID
	return testResult, nil
}

// evaluateRule 评估规则
func (s *RuleTestService) evaluateRule(ruleConfig map[string]interface{}, testData map[string]interface{}) (bool, map[string]interface{}, error) {
	ruleType, ok := ruleConfig["type"].(string)
	if !ok {
		return false, nil, fmt.Errorf("规则类型缺失")
	}

	resultDetails := make(map[string]interface{})
	resultDetails["rule_type"] = ruleType

	var triggered bool
	var err error

	switch ruleType {
	case "threshold":
		triggered, err = s.evaluateThreshold(ruleConfig, testData, resultDetails)
	case "anomaly":
		triggered, err = s.evaluateAnomaly(ruleConfig, testData, resultDetails)
	case "trend":
		triggered, err = s.evaluateTrend(ruleConfig, testData, resultDetails)
	case "composite":
		triggered, err = s.evaluateComposite(ruleConfig, testData, resultDetails)
	case "rate_of_change":
		triggered, err = s.evaluateRateOfChange(ruleConfig, testData, resultDetails)
	default:
		return false, nil, fmt.Errorf("不支持的规则类型: %s", ruleType)
	}

	if err != nil {
		return false, resultDetails, err
	}

	resultDetails["triggered"] = triggered
	return triggered, resultDetails, nil
}

// evaluateThreshold 评估阈值规则
func (s *RuleTestService) evaluateThreshold(ruleConfig map[string]interface{}, testData map[string]interface{}, resultDetails map[string]interface{}) (bool, error) {
	field, ok := ruleConfig["field"].(string)
	if !ok {
		return false, fmt.Errorf("字段名称缺失")
	}

	operator, _ := ruleConfig["operator"].(string)
	if operator == "" {
		operator = "gt"
	}

	threshold, ok := ruleConfig["value"].(float64)
	if !ok {
		return false, fmt.Errorf("阈值缺失或格式错误")
	}

	// 获取测试数据中的字段值
	fieldValue, ok := testData[field]
	if !ok {
		resultDetails["error"] = fmt.Sprintf("测试数据中缺少字段: %s", field)
		return false, nil
	}

	value, err := s.toFloat64(fieldValue)
	if err != nil {
		resultDetails["error"] = fmt.Sprintf("字段值无法转换为数字: %v", fieldValue)
		return false, nil
	}

	resultDetails["field"] = field
	resultDetails["value"] = value
	resultDetails["operator"] = operator
	resultDetails["threshold"] = threshold

	// 比较
	var matched bool
	switch operator {
	case "gt":
		matched = value > threshold
	case "gte":
		matched = value >= threshold
	case "lt":
		matched = value < threshold
	case "lte":
		matched = value <= threshold
	case "eq":
		matched = value == threshold
	case "ne":
		matched = value != threshold
	default:
		return false, fmt.Errorf("不支持的操作符: %s", operator)
	}

	resultDetails["matched"] = matched
	return matched, nil
}

// evaluateAnomaly 评估异常检测规则（简化实现）
func (s *RuleTestService) evaluateAnomaly(ruleConfig map[string]interface{}, testData map[string]interface{}, resultDetails map[string]interface{}) (bool, error) {
	field, ok := ruleConfig["field"].(string)
	if !ok {
		return false, fmt.Errorf("字段名称缺失")
	}

	threshold, _ := ruleConfig["threshold"].(float64)
	if threshold == 0 {
		threshold = 3.0
	}

	fieldValue, ok := testData[field]
	if !ok {
		resultDetails["error"] = fmt.Sprintf("测试数据中缺少字段: %s", field)
		return false, nil
	}

	value, err := s.toFloat64(fieldValue)
	if err != nil {
		resultDetails["error"] = fmt.Sprintf("字段值无法转换为数字: %v", fieldValue)
		return false, nil
	}

	// 简化实现：假设测试数据中有 mean 和 stddev
	mean, _ := testData["mean"].(float64)
	stddev, _ := testData["stddev"].(float64)

	if stddev == 0 {
		resultDetails["error"] = "标准差为0，无法进行异常检测"
		return false, nil
	}

	zScore := (value - mean) / stddev
	matched := zScore > threshold || zScore < -threshold

	resultDetails["field"] = field
	resultDetails["value"] = value
	resultDetails["z_score"] = zScore
	resultDetails["threshold"] = threshold
	resultDetails["matched"] = matched

	return matched, nil
}

// evaluateTrend 评估趋势规则（简化实现）
func (s *RuleTestService) evaluateTrend(ruleConfig map[string]interface{}, testData map[string]interface{}, resultDetails map[string]interface{}) (bool, error) {
	field, ok := ruleConfig["field"].(string)
	if !ok {
		return false, fmt.Errorf("字段名称缺失")
	}

	trendType, _ := ruleConfig["trend_type"].(string)
	if trendType == "" {
		trendType = "increasing"
	}

	// 简化实现：假设测试数据中有 trend_value
	trendValue, ok := testData["trend_value"].(float64)
	if !ok {
		resultDetails["error"] = "测试数据中缺少趋势值"
		return false, nil
	}

	threshold, _ := ruleConfig["threshold"].(float64)
	if threshold == 0 {
		threshold = 0.1
	}

	var matched bool
	switch trendType {
	case "increasing":
		matched = trendValue > threshold
	case "decreasing":
		matched = trendValue < -threshold
	case "stable":
		matched = trendValue >= -threshold && trendValue <= threshold
	default:
		return false, fmt.Errorf("不支持的趋势类型: %s", trendType)
	}

	resultDetails["field"] = field
	resultDetails["trend_type"] = trendType
	resultDetails["trend_value"] = trendValue
	resultDetails["threshold"] = threshold
	resultDetails["matched"] = matched

	return matched, nil
}

// evaluateComposite 评估复合规则
func (s *RuleTestService) evaluateComposite(ruleConfig map[string]interface{}, testData map[string]interface{}, resultDetails map[string]interface{}) (bool, error) {
	logic, _ := ruleConfig["logic"].(string)
	if logic == "" {
		logic = "AND"
	}

	conditions, ok := ruleConfig["conditions"].([]interface{})
	if !ok {
		return false, fmt.Errorf("条件列表缺失")
	}

	conditionResults := make([]bool, 0)
	for i, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		matched, err := s.evaluateThreshold(condMap, testData, map[string]interface{}{})
		if err != nil {
			resultDetails[fmt.Sprintf("condition_%d_error", i)] = err.Error()
			continue
		}
		conditionResults = append(conditionResults, matched)
		resultDetails[fmt.Sprintf("condition_%d", i)] = matched
	}

	if len(conditionResults) == 0 {
		return false, fmt.Errorf("没有有效的条件")
	}

	var finalResult bool
	if logic == "AND" {
		finalResult = true
		for _, r := range conditionResults {
			if !r {
				finalResult = false
				break
			}
		}
	} else if logic == "OR" {
		finalResult = false
		for _, r := range conditionResults {
			if r {
				finalResult = true
				break
			}
		}
	} else {
		return false, fmt.Errorf("不支持的逻辑操作符: %s", logic)
	}

	resultDetails["logic"] = logic
	resultDetails["matched"] = finalResult
	return finalResult, nil
}

// evaluateRateOfChange 评估变化率规则
func (s *RuleTestService) evaluateRateOfChange(ruleConfig map[string]interface{}, testData map[string]interface{}, resultDetails map[string]interface{}) (bool, error) {
	field, ok := ruleConfig["field"].(string)
	if !ok {
		return false, fmt.Errorf("字段名称缺失")
	}

	threshold, _ := ruleConfig["threshold"].(float64)
	if threshold == 0 {
		threshold = 10.0
	}

	// 假设测试数据中有 current_value 和 previous_value
	currentValue, ok1 := testData["current_value"].(float64)
	previousValue, ok2 := testData["previous_value"].(float64)

	if !ok1 || !ok2 {
		resultDetails["error"] = "测试数据中缺少 current_value 或 previous_value"
		return false, nil
	}

	if previousValue == 0 {
		resultDetails["error"] = "previous_value 为0，无法计算变化率"
		return false, nil
	}

	rateOfChange := ((currentValue - previousValue) / previousValue) * 100
	matched := rateOfChange > threshold || rateOfChange < -threshold

	resultDetails["field"] = field
	resultDetails["current_value"] = currentValue
	resultDetails["previous_value"] = previousValue
	resultDetails["rate_of_change"] = rateOfChange
	resultDetails["threshold"] = threshold
	resultDetails["matched"] = matched

	return matched, nil
}

// applyVariables 应用变量到模板
func (s *RuleTestService) applyVariables(template map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range template {
		result[k] = s.replaceVariables(v, variables)
	}
	return result
}

// replaceVariables 替换变量占位符
func (s *RuleTestService) replaceVariables(value interface{}, variables map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// 替换 ${variable_name} 格式的变量
		if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
			varName := v[2 : len(v)-1]
			if varValue, ok := variables[varName]; ok {
				return varValue
			}
			return v // 保持原值
		}
		return v
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = s.replaceVariables(val, variables)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = s.replaceVariables(val, variables)
		}
		return result
	default:
		return v
	}
}

// toFloat64 转换为float64
func (s *RuleTestService) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("无法转换为数字: %v", value)
	}
}
