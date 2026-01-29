package control

import (
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// RuleBasedPolicy 基于规则的控制策略
type RuleBasedPolicy struct {
	id            string
	name          string
	enabled       bool
	rules         []*interfaces.ControlRule
	lastExecution map[string]time.Time // 规则ID -> 最后执行时间
	executionCount map[string]int       // 规则ID -> 执行次数
	mu            sync.RWMutex
}

// NewRuleBasedPolicy 创建基于规则的控制策略
func NewRuleBasedPolicy(id, name string) *RuleBasedPolicy {
	return &RuleBasedPolicy{
		id:             id,
		name:           name,
		enabled:        true,
		rules:          make([]*interfaces.ControlRule, 0),
		lastExecution:  make(map[string]time.Time),
		executionCount: make(map[string]int),
	}
}

// GetID 获取策略ID - 实现 ControlPolicy 接口
func (p *RuleBasedPolicy) GetID() string {
	return p.id
}

// GetName 获取策略名称 - 实现 ControlPolicy 接口
func (p *RuleBasedPolicy) GetName() string {
	return p.name
}

// IsEnabled 是否启用 - 实现 ControlPolicy 接口
func (p *RuleBasedPolicy) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// SetEnabled 设置启用状态 - 实现 ControlPolicy 接口
func (p *RuleBasedPolicy) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// Evaluate 评估分析结果 - 实现 ControlPolicy 接口
func (p *RuleBasedPolicy) Evaluate(results []interfaces.AnalysisResult) ([]*interfaces.ControlAction, error) {
	if !p.IsEnabled() {
		return nil, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	actions := make([]*interfaces.ControlAction, 0)

	// 按优先级排序规则（优先级高的先执行）
	sortedRules := p.getSortedRules()

	// 评估每个规则
	for _, rule := range sortedRules {
		if !rule.Enabled {
			continue
		}

		// 检查冷却时间
		if !p.checkCooldown(rule) {
			continue
		}

		// 检查最大执行次数
		if !p.checkMaxExecutions(rule) {
			continue
		}

		// 评估规则条件
		matched := false
		for _, result := range results {
			if p.evaluateCondition(rule, result) {
				matched = true
				break
			}
		}

		if matched {
			// 添加规则的所有动作
			for _, action := range rule.Actions {
				// 复制动作并添加规则信息
				actionCopy := *action
				if actionCopy.Metadata == nil {
					actionCopy.Metadata = make(map[string]interface{})
				}
				actionCopy.Metadata["rule_id"] = rule.ID
				actionCopy.Metadata["rule_name"] = rule.Name
				actionCopy.Metadata["policy_id"] = p.id
				actionCopy.Metadata["policy_name"] = p.name

				actions = append(actions, &actionCopy)
			}

			// 更新执行记录
			p.lastExecution[rule.ID] = time.Now()
			p.executionCount[rule.ID]++
		}
	}

	return actions, nil
}

// AddRule 添加规则
func (p *RuleBasedPolicy) AddRule(rule *interfaces.ControlRule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查规则ID是否已存在
	for _, r := range p.rules {
		if r.ID == rule.ID {
			return fmt.Errorf("rule %s already exists", rule.ID)
		}
	}

	p.rules = append(p.rules, rule)
	return nil
}

// RemoveRule 移除规则
func (p *RuleBasedPolicy) RemoveRule(ruleID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, rule := range p.rules {
		if rule.ID == ruleID {
			p.rules = append(p.rules[:i], p.rules[i+1:]...)
			delete(p.lastExecution, ruleID)
			delete(p.executionCount, ruleID)
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", ruleID)
}

// GetRules 获取所有规则
func (p *RuleBasedPolicy) GetRules() []*interfaces.ControlRule {
	p.mu.RLock()
	defer p.mu.RUnlock()

	rules := make([]*interfaces.ControlRule, len(p.rules))
	copy(rules, p.rules)
	return rules
}

// ResetStatistics 重置统计信息
func (p *RuleBasedPolicy) ResetStatistics() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastExecution = make(map[string]time.Time)
	p.executionCount = make(map[string]int)
}

// getSortedRules 获取按优先级排序的规则
func (p *RuleBasedPolicy) getSortedRules() []*interfaces.ControlRule {
	// 简单的冒泡排序（优先级高的在前）
	sorted := make([]*interfaces.ControlRule, len(p.rules))
	copy(sorted, p.rules)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority > sorted[i].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// checkCooldown 检查冷却时间
func (p *RuleBasedPolicy) checkCooldown(rule *interfaces.ControlRule) bool {
	if rule.Cooldown == 0 {
		return true
	}

	lastExec, exists := p.lastExecution[rule.ID]
	if !exists {
		return true
	}

	return time.Since(lastExec) >= rule.Cooldown
}

// checkMaxExecutions 检查最大执行次数
func (p *RuleBasedPolicy) checkMaxExecutions(rule *interfaces.ControlRule) bool {
	if rule.MaxExecutions == 0 {
		return true
	}

	count, exists := p.executionCount[rule.ID]
	if !exists {
		return true
	}

	return count < rule.MaxExecutions
}

// evaluateCondition 评估规则条件
func (p *RuleBasedPolicy) evaluateCondition(rule *interfaces.ControlRule, result interfaces.AnalysisResult) bool {
	cond := rule.Condition

	// 检查分析类型
	if cond.AnalysisType != "" && cond.AnalysisType != result.Type {
		return false
	}

	// 检查分数阈值
	if result.Score < cond.MinScore {
		return false
	}

	// 检查严重程度
	if result.Severity < cond.MinSeverity {
		return false
	}

	// 检查必需的详细信息
	for key, expectedValue := range cond.RequiredDetails {
		actualValue, exists := result.Details[key]
		if !exists {
			return false
		}
		if actualValue != expectedValue {
			return false
		}
	}

	// 使用自定义评估函数
	if cond.CustomEvaluator != nil {
		return cond.CustomEvaluator(result)
	}

	return true
}

// 确保实现了ControlPolicy接口
var _ interfaces.ControlPolicy = (*RuleBasedPolicy)(nil)

// CreateEmergencyStopRule 创建紧急停止规则的辅助函数
func CreateEmergencyStopRule(deviceID string, threshold float64) *interfaces.ControlRule {
	return &interfaces.ControlRule{
		ID:       fmt.Sprintf("emergency_stop_%s", deviceID),
		Name:     fmt.Sprintf("紧急停止 - %s", deviceID),
		Enabled:  true,
		Priority: 100, // 最高优先级
		Condition: interfaces.RuleCondition{
			AnalysisType: "ml_anomaly_detection",
			MinScore:     threshold,
			MinSeverity:  4, // High或Critical
			RequiredDetails: map[string]interface{}{
				"is_anomaly": true,
			},
		},
		Actions: []*interfaces.ControlAction{
			{
				ActionType:          interfaces.ActionEmergencyStop,
				TargetDevice:        deviceID,
				Reason:              "AI检测到严重异常，触发紧急停止",
				Severity:            5,
				RequireConfirmation: false, // 紧急情况不需要确认
				Timeout:             10 * time.Second,
			},
		},
		Cooldown:      5 * time.Minute, // 5分钟冷却时间
		MaxExecutions: 0,                // 无限制
		Description:   "当AI模型检测到高分数异常时，自动触发设备紧急停止",
	}
}

// CreateHighTempShutdownRule 创建高温关闭规则的辅助函数
func CreateHighTempShutdownRule(deviceID string, tempThreshold float64) *interfaces.ControlRule {
	return &interfaces.ControlRule{
		ID:       fmt.Sprintf("high_temp_shutdown_%s", deviceID),
		Name:     fmt.Sprintf("高温关闭 - %s", deviceID),
		Enabled:  true,
		Priority: 90,
		Condition: interfaces.RuleCondition{
			AnalysisType: "ml_anomaly_detection",
			MinScore:     0.75,
			MinSeverity:  3,
			CustomEvaluator: func(result interfaces.AnalysisResult) bool {
				// 检查是否是温度过高
				if temp, ok := result.Details["temperature"].(float64); ok {
					return temp > tempThreshold
				}
				return false
			},
		},
		Actions: []*interfaces.ControlAction{
			{
				ActionType:   interfaces.ActionShutdown,
				TargetDevice: deviceID,
				Parameters: map[string]interface{}{
					"graceful": true, // 优雅关闭
				},
				Reason:              fmt.Sprintf("温度超过阈值 %.1f°C", tempThreshold),
				Severity:            4,
				RequireConfirmation: false,
				Timeout:             30 * time.Second,
			},
		},
		Cooldown:      3 * time.Minute,
		MaxExecutions: 0,
		Description:   "当温度超过阈值时，自动关闭设备以避免损坏",
	}
}

// CreateExcessiveVibrationRule 创建过度振动规则的辅助函数
func CreateExcessiveVibrationRule(deviceID string, vibrationThreshold float64) *interfaces.ControlRule {
	return &interfaces.ControlRule{
		ID:       fmt.Sprintf("excessive_vibration_%s", deviceID),
		Name:     fmt.Sprintf("过度振动保护 - %s", deviceID),
		Enabled:  true,
		Priority: 85,
		Condition: interfaces.RuleCondition{
			AnalysisType: "ml_anomaly_detection",
			MinScore:     0.7,
			MinSeverity:  3,
			CustomEvaluator: func(result interfaces.AnalysisResult) bool {
				// 检查是否是振动过大
				if vib, ok := result.Details["vibration"].(float64); ok {
					return vib > vibrationThreshold
				}
				return false
			},
		},
		Actions: []*interfaces.ControlAction{
			{
				ActionType:          interfaces.ActionPause,
				TargetDevice:        deviceID,
				Reason:              fmt.Sprintf("振动超过阈值 %.2f", vibrationThreshold),
				Severity:            3,
				RequireConfirmation: true, // 需要确认
				Timeout:             15 * time.Second,
			},
		},
		Cooldown:      2 * time.Minute,
		MaxExecutions: 5, // 最多执行5次
		Description:   "当设备振动超过安全阈值时，暂停设备运行",
	}
}
