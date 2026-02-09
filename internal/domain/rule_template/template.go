package rule_template

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RuleTemplate 规则模板实体
type RuleTemplate struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	RuleType        string                 `json:"rule_type"`
	Tags            []string               `json:"tags"`
	Icon            string                 `json:"icon"`
	ConditionTemplate map[string]interface{} `json:"condition_template"` // 条件模板（支持变量占位符）
	VariablesConfig map[string]interface{} `json:"variables_config"`      // 可配置变量说明
	OutputConfig    map[string]interface{} `json:"output_config"`        // 输出配置
	UsageCount      int                    `json:"usage_count"`
	Rating          float64                `json:"rating"`
	Enabled         bool                   `json:"enabled"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CreatedBy       string                 `json:"created_by,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// NewRuleTemplate 创建新的规则模板
func NewRuleTemplate(name, category, ruleType string, conditionTemplate map[string]interface{}) *RuleTemplate {
	now := time.Now()
	return &RuleTemplate{
		ID:               uuid.New().String(),
		Name:             name,
		Category:         category,
		RuleType:         ruleType,
		Tags:             []string{},
		ConditionTemplate: conditionTemplate,
		VariablesConfig:  make(map[string]interface{}),
		OutputConfig:     make(map[string]interface{}),
		UsageCount:       0,
		Rating:           0.0,
		Enabled:          true,
		CreatedAt:        now,
		UpdatedAt:        now,
		Metadata:         make(map[string]interface{}),
	}
}

// IncrementUsage 增加使用次数
func (t *RuleTemplate) IncrementUsage() {
	t.UsageCount++
	t.UpdatedAt = time.Now()
}

// SetRating 设置评分
func (t *RuleTemplate) SetRating(rating float64) {
	if rating < 0 {
		rating = 0
	}
	if rating > 5 {
		rating = 5
	}
	t.Rating = rating
	t.UpdatedAt = time.Now()
}

// ToJSON 转换为JSON
func (t *RuleTemplate) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// FromJSON 从JSON解析
func (t *RuleTemplate) FromJSON(data []byte) error {
	return json.Unmarshal(data, t)
}

// RuleTestResult 规则测试结果实体
type RuleTestResult struct {
	ID            string                 `json:"id"`
	RuleID        string                 `json:"rule_id,omitempty"`
	TemplateID    string                 `json:"template_id,omitempty"`
	TestData      map[string]interface{} `json:"test_data"`
	RuleConfig    map[string]interface{} `json:"rule_config"`
	TestResult    map[string]interface{} `json:"test_result"`
	Triggered     bool                   `json:"triggered"`
	ExecutionTimeMs int                  `json:"execution_time_ms"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by,omitempty"`
}

// NewRuleTestResult 创建新的规则测试结果
func NewRuleTestResult(testData, ruleConfig map[string]interface{}) *RuleTestResult {
	return &RuleTestResult{
		ID:         uuid.New().String(),
		TestData:   testData,
		RuleConfig: ruleConfig,
		TestResult: make(map[string]interface{}),
		Triggered:  false,
		CreatedAt:  time.Now(),
	}
}
