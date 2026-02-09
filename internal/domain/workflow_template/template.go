package workflow_template

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WorkflowTemplate 工作流模板实体
type WorkflowTemplate struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Category       string                 `json:"category"`
	Tags           []string               `json:"tags"`
	Icon           string                 `json:"icon"`
	ThumbnailURL   string                 `json:"thumbnail_url,omitempty"`
	WorkflowData   map[string]interface{} `json:"workflow_data"`   // 工作流定义（nodes, edges, variables）
	VariablesConfig map[string]interface{} `json:"variables_config"` // 可配置变量说明
	UsageCount     int                    `json:"usage_count"`
	Rating         float64                `json:"rating"`
	Enabled        bool                   `json:"enabled"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	CreatedBy      string                 `json:"created_by,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewWorkflowTemplate 创建新的工作流模板
func NewWorkflowTemplate(name, category string, workflowData map[string]interface{}) *WorkflowTemplate {
	now := time.Now()
	return &WorkflowTemplate{
		ID:             uuid.New().String(),
		Name:           name,
		Category:       category,
		Tags:           []string{},
		WorkflowData:   workflowData,
		VariablesConfig: make(map[string]interface{}),
		UsageCount:     0,
		Rating:         0.0,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
		Metadata:       make(map[string]interface{}),
	}
}

// IncrementUsage 增加使用次数
func (t *WorkflowTemplate) IncrementUsage() {
	t.UsageCount++
	t.UpdatedAt = time.Now()
}

// SetRating 设置评分
func (t *WorkflowTemplate) SetRating(rating float64) {
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
func (t *WorkflowTemplate) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// FromJSON 从JSON解析
func (t *WorkflowTemplate) FromJSON(data []byte) error {
	return json.Unmarshal(data, t)
}
