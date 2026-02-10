package storage

import (
	"context"
	"encoding/json"
	"fmt"
	templateDomain "go_ProFiBus/internal/domain/workflow_template"

	"github.com/jackc/pgx/v5"
)

// WorkflowTemplateRepositoryImpl 工作流模板仓储实现
type WorkflowTemplateRepositoryImpl struct {
	store *PostgresStore
}

// NewWorkflowTemplateRepository 创建工作流模板仓储
func NewWorkflowTemplateRepository(store *PostgresStore) *WorkflowTemplateRepositoryImpl {
	return &WorkflowTemplateRepositoryImpl{
		store: store,
	}
}

// CreateTemplate 创建模板
func (r *WorkflowTemplateRepositoryImpl) CreateTemplate(ctx context.Context, template *templateDomain.WorkflowTemplate) error {
	workflowDataJSON, _ := json.Marshal(template.WorkflowData)
	variablesConfigJSON, _ := json.Marshal(template.VariablesConfig)
	metadataJSON, _ := json.Marshal(template.Metadata)

	query := `
		INSERT INTO workflow_templates (id, name, description, category, tags, icon, thumbnail_url,
		                              workflow_data, variables_config, usage_count, rating,
		                              enabled, created_at, updated_at, created_by, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.store.GetPool().Exec(ctx, query,
		template.ID, template.Name, template.Description, template.Category, template.Tags,
		template.Icon, template.ThumbnailURL, workflowDataJSON, variablesConfigJSON,
		template.UsageCount, template.Rating, template.Enabled,
		template.CreatedAt, template.UpdatedAt, template.CreatedBy, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("创建模板失败: %w", err)
	}

	return nil
}

// GetTemplateByID 根据ID获取模板
func (r *WorkflowTemplateRepositoryImpl) GetTemplateByID(ctx context.Context, id string) (*templateDomain.WorkflowTemplate, error) {
	query := `
		SELECT id, name, description, category, tags, icon, thumbnail_url,
		       workflow_data, variables_config, usage_count, rating,
		       enabled, created_at, updated_at, created_by, metadata
		FROM workflow_templates
		WHERE id = $1
	`

	var template templateDomain.WorkflowTemplate
	var workflowDataJSON, variablesConfigJSON, metadataJSON []byte

	err := r.store.GetPool().QueryRow(ctx, query, id).Scan(
		&template.ID, &template.Name, &template.Description, &template.Category, &template.Tags,
		&template.Icon, &template.ThumbnailURL, &workflowDataJSON, &variablesConfigJSON,
		&template.UsageCount, &template.Rating, &template.Enabled,
		&template.CreatedAt, &template.UpdatedAt, &template.CreatedBy, &metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("模板不存在: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	json.Unmarshal(workflowDataJSON, &template.WorkflowData)
	json.Unmarshal(variablesConfigJSON, &template.VariablesConfig)
	json.Unmarshal(metadataJSON, &template.Metadata)

	return &template, nil
}

// ListTemplates 列出模板
func (r *WorkflowTemplateRepositoryImpl) ListTemplates(ctx context.Context, filters map[string]interface{}) ([]*templateDomain.WorkflowTemplate, error) {
	query := `
		SELECT id, name, description, category, tags, icon, thumbnail_url,
		       workflow_data, variables_config, usage_count, rating,
		       enabled, created_at, updated_at, created_by, metadata
		FROM workflow_templates
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if category, ok := filters["category"].(string); ok && category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}

	if enabled, ok := filters["enabled"].(bool); ok {
		query += fmt.Sprintf(" AND enabled = $%d", argIdx)
		args = append(args, enabled)
		argIdx++
	} else {
		// 默认只返回启用的模板
		query += fmt.Sprintf(" AND enabled = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	if tag, ok := filters["tag"].(string); ok && tag != "" {
		query += fmt.Sprintf(" AND $%d = ANY(tags)", argIdx)
		args = append(args, tag)
		argIdx++
	}

	query += " ORDER BY usage_count DESC, created_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
		if offset, ok := filters["offset"].(int); ok && offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIdx)
			args = append(args, offset)
		}
	}

	rows, err := r.store.GetPool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询模板列表失败: %w", err)
	}
	defer rows.Close()

	var templates []*templateDomain.WorkflowTemplate
	for rows.Next() {
		var template templateDomain.WorkflowTemplate
		var workflowDataJSON, variablesConfigJSON, metadataJSON []byte

		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &template.Category, &template.Tags,
			&template.Icon, &template.ThumbnailURL, &workflowDataJSON, &variablesConfigJSON,
			&template.UsageCount, &template.Rating, &template.Enabled,
			&template.CreatedAt, &template.UpdatedAt, &template.CreatedBy, &metadataJSON,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(workflowDataJSON, &template.WorkflowData)
		json.Unmarshal(variablesConfigJSON, &template.VariablesConfig)
		json.Unmarshal(metadataJSON, &template.Metadata)

		templates = append(templates, &template)
	}

	return templates, nil
}

// UpdateTemplate 更新模板
func (r *WorkflowTemplateRepositoryImpl) UpdateTemplate(ctx context.Context, template *templateDomain.WorkflowTemplate) error {
	workflowDataJSON, _ := json.Marshal(template.WorkflowData)
	variablesConfigJSON, _ := json.Marshal(template.VariablesConfig)
	metadataJSON, _ := json.Marshal(template.Metadata)

	query := `
		UPDATE workflow_templates
		SET name = $2, description = $3, category = $4, tags = $5, icon = $6,
		    thumbnail_url = $7, workflow_data = $8, variables_config = $9,
		    usage_count = $10, rating = $11, enabled = $12, updated_at = $13, metadata = $14
		WHERE id = $1
	`

	_, err := r.store.GetPool().Exec(ctx, query,
		template.ID, template.Name, template.Description, template.Category, template.Tags,
		template.Icon, template.ThumbnailURL, workflowDataJSON, variablesConfigJSON,
		template.UsageCount, template.Rating, template.Enabled, template.UpdatedAt, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("更新模板失败: %w", err)
	}

	return nil
}

// DeleteTemplate 删除模板
func (r *WorkflowTemplateRepositoryImpl) DeleteTemplate(ctx context.Context, id string) error {
	query := `DELETE FROM workflow_templates WHERE id = $1`
	_, err := r.store.GetPool().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}
	return nil
}

// IncrementUsage 增加使用次数
func (r *WorkflowTemplateRepositoryImpl) IncrementUsage(ctx context.Context, id string) error {
	query := `UPDATE workflow_templates SET usage_count = usage_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.store.GetPool().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("更新使用次数失败: %w", err)
	}
	return nil
}
