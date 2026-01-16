package rule

import (
	"fmt"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// ThresholdRule 阈值规则
type ThresholdRule struct {
	*Rule
	field     string
	operator  string
	threshold float64
	severity  int
}

// ThresholdOperator 阈值操作符
const (
	OperatorGreaterThan    = ">"
	OperatorLessThan       = "<"
	OperatorGreaterOrEqual = ">="
	OperatorLessOrEqual    = "<="
	OperatorEqual          = "=="
	OperatorNotEqual       = "!="
)

// NewThresholdRule 创建阈值规则
func NewThresholdRule(id, name, field, operator string, threshold float64, severity int) *ThresholdRule {
	config := map[string]interface{}{
		"field":     field,
		"operator":  operator,
		"threshold": threshold,
		"severity":  severity,
	}

	baseRule := NewRule(id, name, "threshold", config, nil)

	tr := &ThresholdRule{
		Rule:      baseRule,
		field:     field,
		operator:  operator,
		threshold: threshold,
		severity:  severity,
	}

	// 设置评估函数
	baseRule.SetEvaluator(tr.evaluate)

	return tr
}

// evaluate 阈值评估函数
func (tr *ThresholdRule) evaluate(data interfaces.DataSample, config map[string]interface{}) (*interfaces.RuleResult, error) {
	// 获取字段值
	value, ok := data.GetData()[tr.field]
	if !ok {
		return nil, fmt.Errorf("field %s not found in data", tr.field)
	}

	// 转换为 float64
	var floatValue float64
	switch v := value.(type) {
	case float64:
		floatValue = v
	case int:
		floatValue = float64(v)
	case int64:
		floatValue = float64(v)
	default:
		return nil, fmt.Errorf("field %s is not numeric", tr.field)
	}

	// 评估阈值
	triggered := false
	switch tr.operator {
	case OperatorGreaterThan:
		triggered = floatValue > tr.threshold
	case OperatorLessThan:
		triggered = floatValue < tr.threshold
	case OperatorGreaterOrEqual:
		triggered = floatValue >= tr.threshold
	case OperatorLessOrEqual:
		triggered = floatValue <= tr.threshold
	case OperatorEqual:
		triggered = floatValue == tr.threshold
	case OperatorNotEqual:
		triggered = floatValue != tr.threshold
	default:
		return nil, fmt.Errorf("unknown operator: %s", tr.operator)
	}

	// 计算分数
	score := 0.0
	if triggered {
		score = calculateScore(floatValue, tr.threshold, tr.operator)
	}

	return &interfaces.RuleResult{
		RuleID:    tr.id,
		RuleName:  tr.name,
		Triggered: triggered,
		Severity:  tr.severity,
		Score:     score,
		Message:   fmt.Sprintf("%s %s %.2f (threshold: %.2f)", tr.field, tr.operator, floatValue, tr.threshold),
		Details: map[string]interface{}{
			"field":     tr.field,
			"value":     floatValue,
			"threshold": tr.threshold,
			"operator":  tr.operator,
		},
		Timestamp: time.Now(),
	}, nil
}

// calculateScore 计算分数
func calculateScore(value, threshold float64, operator string) float64 {
	var diff float64
	switch operator {
	case OperatorGreaterThan, OperatorGreaterOrEqual:
		diff = value - threshold
	case OperatorLessThan, OperatorLessOrEqual:
		diff = threshold - value
	default:
		return 0.5
	}

	// 归一化分数到 0-1
	if threshold == 0 {
		return 1.0
	}

	score := diff / threshold
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}
