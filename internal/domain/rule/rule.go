package rule

import (
	"fmt"
	"time"

	"github.com/LiSuiTech/go_ProFiBus/pkg/interfaces"
)

// Rule 规则实体
type Rule struct {
	id          string
	name        string
	ruleType    string
	enabled     bool
	config      map[string]interface{}
	evaluator   RuleEvaluator
	description string
	createdAt   time.Time
	updatedAt   time.Time
}

// RuleEvaluator 规则评估函数类型
type RuleEvaluator func(data interfaces.DataSample, config map[string]interface{}) (*interfaces.RuleResult, error)

// NewRule 创建新规则
func NewRule(id, name, ruleType string, config map[string]interface{}, evaluator RuleEvaluator) *Rule {
	now := time.Now()
	return &Rule{
		id:        id,
		name:      name,
		ruleType:  ruleType,
		enabled:   true,
		config:    config,
		evaluator: evaluator,
		createdAt: now,
		updatedAt: now,
	}
}

// GetID 实现 Rule 接口
func (r *Rule) GetID() string {
	return r.id
}

// GetName 实现 Rule 接口
func (r *Rule) GetName() string {
	return r.name
}

// GetType 实现 Rule 接口
func (r *Rule) GetType() string {
	return r.ruleType
}

// IsEnabled 实现 Rule 接口
func (r *Rule) IsEnabled() bool {
	return r.enabled
}

// Evaluate 实现 Rule 接口
func (r *Rule) Evaluate(data interfaces.DataSample) (*interfaces.RuleResult, error) {
	if !r.enabled {
		return nil, nil
	}

	if r.evaluator == nil {
		return nil, fmt.Errorf("rule %s has no evaluator", r.id)
	}

	return r.evaluator(data, r.config)
}

// GetConfig 实现 Rule 接口
func (r *Rule) GetConfig() map[string]interface{} {
	return r.config
}

// SetEnabled 设置启用状态
func (r *Rule) SetEnabled(enabled bool) {
	r.enabled = enabled
	r.updatedAt = time.Now()
}

// UpdateConfig 更新配置
func (r *Rule) UpdateConfig(config map[string]interface{}) {
	r.config = config
	r.updatedAt = time.Now()
}

// SetEvaluator 设置评估函数
func (r *Rule) SetEvaluator(evaluator RuleEvaluator) {
	r.evaluator = evaluator
	r.updatedAt = time.Now()
}

// GetDescription 获取描述
func (r *Rule) GetDescription() string {
	return r.description
}

// SetDescription 设置描述
func (r *Rule) SetDescription(description string) {
	r.description = description
	r.updatedAt = time.Now()
}

// GetCreatedAt 获取创建时间
func (r *Rule) GetCreatedAt() time.Time {
	return r.createdAt
}

// GetUpdatedAt 获取更新时间
func (r *Rule) GetUpdatedAt() time.Time {
	return r.updatedAt
}

// 确保 Rule 实现了 interfaces.Rule 接口
var _ interfaces.Rule = (*Rule)(nil)
