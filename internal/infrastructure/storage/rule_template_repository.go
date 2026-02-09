package storage

import (
	"context"
	"encoding/json"
	"fmt"
	templateDomain "go_ProFiBus/internal/domain/rule_template"

	"github.com/jackc/pgx/v5"
)

// RuleTemplateRepositoryImpl 规则模板仓储实现
type RuleTemplateRepositoryImpl struct {
	store *PostgresStore
}

// NewRuleTemplateRepository 创建规则模板仓储
func NewRuleTemplateRepository(store *PostgresStore) *RuleTemplateRepositoryImpl {
	return &RuleTemplateRepositoryImpl{
		store: store,
	}
}

// CreateTemplate 创建模板
func (r *RuleTemplateRepositoryImpl) CreateTemplate(ctx context.Context, template *templateDomain.RuleTemplate) error {
	conditionTemplateJSON, _ := json.Marshal(template.ConditionTemplate)
	variablesConfigJSON, _ := json.Marshal(template.VariablesConfig)
	outputConfigJSON, _ := json.Marshal(template.OutputConfig)
	metadataJSON, _ := json.Marshal(template.Metadata)

	query := `
		INSERT INTO rule_templates (id, name, description, category, rule_type, tags, icon,
		                           condition_template, variables_config, output_config,
		                           usage_count, rating, enabled, created_at, updated_at, created_by, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.store.pool.Exec(ctx, query,
		template.ID, template.Name, template.Description, template.Category, template.RuleType,
		template.Tags, template.Icon, conditionTemplateJSON, variablesConfigJSON, outputConfigJSON,
		template.UsageCount, template.Rating, template.Enabled,
		template.CreatedAt, template.UpdatedAt, template.CreatedBy, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("创建模板失败: %w", err)
	}

	return nil
}

// GetTemplateByID 根据ID获取模板
func (r *RuleTemplateRepositoryImpl) GetTemplateByID(ctx context.Context, id string) (*templateDomain.RuleTemplate, error) {
	query := `
		SELECT id, name, description, category, rule_type, tags, icon,
		       condition_template, variables_config, output_config,
		       usage_count, rating, enabled, created_at, updated_at, created_by, metadata
		FROM rule_templates
		WHERE id = $1
	`

	var template templateDomain.RuleTemplate
	var conditionTemplateJSON, variablesConfigJSON, outputConfigJSON, metadataJSON []byte

	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&template.ID, &template.Name, &template.Description, &template.Category, &template.RuleType,
		&template.Tags, &template.Icon, &conditionTemplateJSON, &variablesConfigJSON,
		&outputConfigJSON, &template.UsageCount, &template.Rating, &template.Enabled,
		&template.CreatedAt, &template.UpdatedAt, &template.CreatedBy, &metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("模板不存在: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	json.Unmarshal(conditionTemplateJSON, &template.ConditionTemplate)
	json.Unmarshal(variablesConfigJSON, &template.VariablesConfig)
	json.Unmarshal(outputConfigJSON, &template.OutputConfig)
	json.Unmarshal(metadataJSON, &template.Metadata)

	return &template, nil
}

// ListTemplates 列出模板
func (r *RuleTemplateRepositoryImpl) ListTemplates(ctx context.Context, filters map[string]interface{}) ([]*templateDomain.RuleTemplate, error) {
	query := `
		SELECT id, name, description, category, rule_type, tags, icon,
		       condition_template, variables_config, output_config,
		       usage_count, rating, enabled, created_at, updated_at, created_by, metadata
		FROM rule_templates
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if category, ok := filters["category"].(string); ok && category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}

	if ruleType, ok := filters["rule_type"].(string); ok && ruleType != "" {
		query += fmt.Sprintf(" AND rule_type = $%d", argIdx)
		args = append(args, ruleType)
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

	rows, err := r.store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询模板列表失败: %w", err)
	}
	defer rows.Close()

	var templates []*templateDomain.RuleTemplate
	for rows.Next() {
		var template templateDomain.RuleTemplate
		var conditionTemplateJSON, variablesConfigJSON, outputConfigJSON, metadataJSON []byte

		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &template.Category, &template.RuleType,
			&template.Tags, &template.Icon, &conditionTemplateJSON, &variablesConfigJSON,
			&outputConfigJSON, &template.UsageCount, &template.Rating, &template.Enabled,
			&template.CreatedAt, &template.UpdatedAt, &template.CreatedBy, &metadataJSON,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(conditionTemplateJSON, &template.ConditionTemplate)
		json.Unmarshal(variablesConfigJSON, &template.VariablesConfig)
		json.Unmarshal(outputConfigJSON, &template.OutputConfig)
		json.Unmarshal(metadataJSON, &template.Metadata)

		templates = append(templates, &template)
	}

	return templates, nil
}

// UpdateTemplate 更新模板
func (r *RuleTemplateRepositoryImpl) UpdateTemplate(ctx context.Context, template *templateDomain.RuleTemplate) error {
	conditionTemplateJSON, _ := json.Marshal(template.ConditionTemplate)
	variablesConfigJSON, _ := json.Marshal(template.VariablesConfig)
	outputConfigJSON, _ := json.Marshal(template.OutputConfig)
	metadataJSON, _ := json.Marshal(template.Metadata)

	query := `
		UPDATE rule_templates
		SET name = $2, description = $3, category = $4, rule_type = $5, tags = $6, icon = $7,
		    condition_template = $8, variables_config = $9, output_config = $10,
		    usage_count = $11, rating = $12, enabled = $13, updated_at = $14, metadata = $15
		WHERE id = $1
	`

	_, err := r.store.pool.Exec(ctx, query,
		template.ID, template.Name, template.Description, template.Category, template.RuleType,
		template.Tags, template.Icon, conditionTemplateJSON, variablesConfigJSON, outputConfigJSON,
		template.UsageCount, template.Rating, template.Enabled, template.UpdatedAt, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("更新模板失败: %w", err)
	}

	return nil
}

// DeleteTemplate 删除模板
func (r *RuleTemplateRepositoryImpl) DeleteTemplate(ctx context.Context, id string) error {
	query := `DELETE FROM rule_templates WHERE id = $1`
	_, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}
	return nil
}

// IncrementUsage 增加使用次数
func (r *RuleTemplateRepositoryImpl) IncrementUsage(ctx context.Context, id string) error {
	query := `UPDATE rule_templates SET usage_count = usage_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("更新使用次数失败: %w", err)
	}
	return nil
}

// SaveTestResult 保存测试结果
func (r *RuleTemplateRepositoryImpl) SaveTestResult(ctx context.Context, result *templateDomain.RuleTestResult) error {
	testDataJSON, _ := json.Marshal(result.TestData)
	ruleConfigJSON, _ := json.Marshal(result.RuleConfig)
	testResultJSON, _ := json.Marshal(result.TestResult)

	query := `
		INSERT INTO rule_test_results (id, rule_id, template_id, test_data, rule_config, test_result,
		                              triggered, execution_time_ms, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.store.pool.Exec(ctx, query,
		result.ID, result.RuleID, result.TemplateID, testDataJSON, ruleConfigJSON, testResultJSON,
		result.Triggered, result.ExecutionTimeMs, result.CreatedAt, result.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("保存测试结果失败: %w", err)
	}

	return nil
}

// ListTestResults 列出测试结果
func (r *RuleTemplateRepositoryImpl) ListTestResults(ctx context.Context, filters map[string]interface{}) ([]*templateDomain.RuleTestResult, error) {
	query := `
		SELECT id, rule_id, template_id, test_data, rule_config, test_result,
		       triggered, execution_time_ms, created_at, created_by
		FROM rule_test_results
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if ruleID, ok := filters["rule_id"].(string); ok && ruleID != "" {
		query += fmt.Sprintf(" AND rule_id = $%d", argIdx)
		args = append(args, ruleID)
		argIdx++
	}

	if templateID, ok := filters["template_id"].(string); ok && templateID != "" {
		query += fmt.Sprintf(" AND template_id = $%d", argIdx)
		args = append(args, templateID)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
		if offset, ok := filters["offset"].(int); ok && offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIdx)
			args = append(args, offset)
		}
	}

	rows, err := r.store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询测试结果列表失败: %w", err)
	}
	defer rows.Close()

	var results []*templateDomain.RuleTestResult
	for rows.Next() {
		var result templateDomain.RuleTestResult
		var testDataJSON, ruleConfigJSON, testResultJSON []byte

		err := rows.Scan(
			&result.ID, &result.RuleID, &result.TemplateID, &testDataJSON, &ruleConfigJSON,
			&testResultJSON, &result.Triggered, &result.ExecutionTimeMs,
			&result.CreatedAt, &result.CreatedBy,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(testDataJSON, &result.TestData)
		json.Unmarshal(ruleConfigJSON, &result.RuleConfig)
		json.Unmarshal(testResultJSON, &result.TestResult)

		results = append(results, &result)
	}

	return results, nil
}
