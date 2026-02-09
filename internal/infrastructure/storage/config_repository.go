package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go_ProFiBus/pkg/interfaces"
)

// ConfigRepository implements the interfaces.ConfigRepository interface
type ConfigRepository struct {
	store *PostgresStore
}

// NewConfigRepository creates a new ConfigRepository
func NewConfigRepository(store *PostgresStore) *ConfigRepository {
	return &ConfigRepository{
		store: store,
	}
}

// ========== Rule Configuration Methods ==========

// CreateRuleConfig creates a new rule configuration
func (r *ConfigRepository) CreateRuleConfig(ctx context.Context, config *interfaces.RuleConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	// Check if config already exists
	var exists bool
	err := r.store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM rule_configs WHERE id = $1)", config.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check rule config existence: %w", err)
	}
	if exists {
		return interfaces.ErrConfigExists{ID: config.ID, ConfigType: "rule"}
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO rule_configs (id, name, description, type, version, parameters, enabled, priority, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.store.pool.Exec(ctx, query,
		config.ID,
		config.Name,
		config.Description,
		config.Type,
		config.Version,
		paramsJSON,
		config.Enabled,
		config.Priority,
		config.CreatedBy,
		config.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create rule config: %w", err)
	}

	// Record history
	history := &interfaces.ConfigHistory{
		ConfigID:   config.ID,
		ConfigType: "rule",
		Action:     "created",
		ChangedBy:  config.CreatedBy,
		NewValue:   config.Parameters,
	}
	_ = r.RecordHistory(ctx, history) // Don't fail creation if history fails

	return nil
}

// GetRuleConfig retrieves a rule configuration by ID
func (r *ConfigRepository) GetRuleConfig(ctx context.Context, id string) (*interfaces.RuleConfig, error) {
	query := `
		SELECT id, name, description, type, version, parameters, enabled, priority,
		       created_at, updated_at, created_by, updated_by
		FROM rule_configs
		WHERE id = $1
	`

	var config interfaces.RuleConfig
	var paramsJSON []byte

	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&config.ID,
		&config.Name,
		&config.Description,
		&config.Type,
		&config.Version,
		&paramsJSON,
		&config.Enabled,
		&config.Priority,
		&config.CreatedAt,
		&config.UpdatedAt,
		&config.CreatedBy,
		&config.UpdatedBy,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrConfigNotFound{ID: id, ConfigType: "rule"}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule config: %w", err)
	}

	// Unmarshal parameters
	if err := json.Unmarshal(paramsJSON, &config.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	return &config, nil
}

// ListRuleConfigs lists all rule configurations with optional filtering
func (r *ConfigRepository) ListRuleConfigs(ctx context.Context, enabled *bool, ruleType *string) ([]*interfaces.RuleConfig, error) {
	query := `
		SELECT id, name, description, type, version, parameters, enabled, priority,
		       created_at, updated_at, created_by, updated_by
		FROM rule_configs
	`

	var args []interface{}
	conditions := []string{}
	argIndex := 1

	if enabled != nil {
		conditions = append(conditions, fmt.Sprintf("enabled = $%d", argIndex))
		args = append(args, *enabled)
		argIndex++
	}

	if ruleType != nil && *ruleType != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *ruleType)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY priority DESC, created_at DESC"

	rows, err := r.store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list rule configs: %w", err)
	}
	defer rows.Close()

	var configs []*interfaces.RuleConfig
	for rows.Next() {
		var config interfaces.RuleConfig
		var paramsJSON []byte

		err := rows.Scan(
			&config.ID,
			&config.Name,
			&config.Description,
			&config.Type,
			&config.Version,
			&paramsJSON,
			&config.Enabled,
			&config.Priority,
			&config.CreatedAt,
			&config.UpdatedAt,
			&config.CreatedBy,
			&config.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule config: %w", err)
		}

		if err := json.Unmarshal(paramsJSON, &config.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// UpdateRuleConfig updates an existing rule configuration
func (r *ConfigRepository) UpdateRuleConfig(ctx context.Context, config *interfaces.RuleConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	// Get old config for history
	oldConfig, err := r.GetRuleConfig(ctx, config.ID)
	if err != nil {
		return err
	}

	// Marshal parameters
	paramsJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		UPDATE rule_configs
		SET name = $2, description = $3, type = $4, version = $5, parameters = $6,
		    enabled = $7, priority = $8, updated_by = $9
		WHERE id = $1
	`

	result, err := r.store.pool.Exec(ctx, query,
		config.ID,
		config.Name,
		config.Description,
		config.Type,
		config.Version,
		paramsJSON,
		config.Enabled,
		config.Priority,
		config.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update rule config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrConfigNotFound{ID: config.ID, ConfigType: "rule"}
	}

	// Record history
	history := &interfaces.ConfigHistory{
		ConfigID:   config.ID,
		ConfigType: "rule",
		Action:     "updated",
		ChangedBy:  config.UpdatedBy,
		OldValue:   oldConfig.Parameters,
		NewValue:   config.Parameters,
	}
	_ = r.RecordHistory(ctx, history)

	return nil
}

// DeleteRuleConfig deletes a rule configuration
func (r *ConfigRepository) DeleteRuleConfig(ctx context.Context, id string) error {
	// Get config for history
	oldConfig, err := r.GetRuleConfig(ctx, id)
	if err != nil {
		return err
	}

	query := "DELETE FROM rule_configs WHERE id = $1"
	result, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrConfigNotFound{ID: id, ConfigType: "rule"}
	}

	// Record history
	history := &interfaces.ConfigHistory{
		ConfigID:   id,
		ConfigType: "rule",
		Action:     "deleted",
		OldValue:   oldConfig.Parameters,
	}
	_ = r.RecordHistory(ctx, history)

	return nil
}

// ========== Analyzer Configuration Methods ==========

// CreateAnalyzerConfig creates a new analyzer configuration
func (r *ConfigRepository) CreateAnalyzerConfig(ctx context.Context, config *interfaces.AnalyzerConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	var exists bool
	err := r.store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM analyzer_configs WHERE id = $1)", config.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check analyzer config existence: %w", err)
	}
	if exists {
		return interfaces.ErrConfigExists{ID: config.ID, ConfigType: "analyzer"}
	}

	paramsJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO analyzer_configs (id, name, description, type, version, parameters, rule_ids, enabled, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.store.pool.Exec(ctx, query,
		config.ID,
		config.Name,
		config.Description,
		config.Type,
		config.Version,
		paramsJSON,
		config.RuleIDs,
		config.Enabled,
		config.CreatedBy,
		config.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create analyzer config: %w", err)
	}

	history := &interfaces.ConfigHistory{
		ConfigID:   config.ID,
		ConfigType: "analyzer",
		Action:     "created",
		ChangedBy:  config.CreatedBy,
		NewValue:   config.Parameters,
	}
	_ = r.RecordHistory(ctx, history)

	return nil
}

// GetAnalyzerConfig retrieves an analyzer configuration by ID
func (r *ConfigRepository) GetAnalyzerConfig(ctx context.Context, id string) (*interfaces.AnalyzerConfig, error) {
	query := `
		SELECT id, name, description, type, version, parameters, rule_ids, enabled,
		       created_at, updated_at, created_by, updated_by
		FROM analyzer_configs
		WHERE id = $1
	`

	var config interfaces.AnalyzerConfig
	var paramsJSON []byte

	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&config.ID,
		&config.Name,
		&config.Description,
		&config.Type,
		&config.Version,
		&paramsJSON,
		&config.RuleIDs,
		&config.Enabled,
		&config.CreatedAt,
		&config.UpdatedAt,
		&config.CreatedBy,
		&config.UpdatedBy,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrConfigNotFound{ID: id, ConfigType: "analyzer"}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get analyzer config: %w", err)
	}

	if err := json.Unmarshal(paramsJSON, &config.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	return &config, nil
}

// ListAnalyzerConfigs lists all analyzer configurations
func (r *ConfigRepository) ListAnalyzerConfigs(ctx context.Context, enabled *bool) ([]*interfaces.AnalyzerConfig, error) {
	query := `
		SELECT id, name, description, type, version, parameters, rule_ids, enabled,
		       created_at, updated_at, created_by, updated_by
		FROM analyzer_configs
	`

	var args []interface{}
	if enabled != nil {
		query += " WHERE enabled = $1"
		args = append(args, *enabled)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list analyzer configs: %w", err)
	}
	defer rows.Close()

	var configs []*interfaces.AnalyzerConfig
	for rows.Next() {
		var config interfaces.AnalyzerConfig
		var paramsJSON []byte

		err := rows.Scan(
			&config.ID,
			&config.Name,
			&config.Description,
			&config.Type,
			&config.Version,
			&paramsJSON,
			&config.RuleIDs,
			&config.Enabled,
			&config.CreatedAt,
			&config.UpdatedAt,
			&config.CreatedBy,
			&config.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analyzer config: %w", err)
		}

		if err := json.Unmarshal(paramsJSON, &config.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// UpdateAnalyzerConfig updates an existing analyzer configuration
func (r *ConfigRepository) UpdateAnalyzerConfig(ctx context.Context, config *interfaces.AnalyzerConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	oldConfig, err := r.GetAnalyzerConfig(ctx, config.ID)
	if err != nil {
		return err
	}

	paramsJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		UPDATE analyzer_configs
		SET name = $2, description = $3, type = $4, version = $5, parameters = $6,
		    rule_ids = $7, enabled = $8, updated_by = $9
		WHERE id = $1
	`

	result, err := r.store.pool.Exec(ctx, query,
		config.ID,
		config.Name,
		config.Description,
		config.Type,
		config.Version,
		paramsJSON,
		config.RuleIDs,
		config.Enabled,
		config.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update analyzer config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrConfigNotFound{ID: config.ID, ConfigType: "analyzer"}
	}

	history := &interfaces.ConfigHistory{
		ConfigID:   config.ID,
		ConfigType: "analyzer",
		Action:     "updated",
		ChangedBy:  config.UpdatedBy,
		OldValue:   oldConfig.Parameters,
		NewValue:   config.Parameters,
	}
	_ = r.RecordHistory(ctx, history)

	return nil
}

// DeleteAnalyzerConfig deletes an analyzer configuration
func (r *ConfigRepository) DeleteAnalyzerConfig(ctx context.Context, id string) error {
	oldConfig, err := r.GetAnalyzerConfig(ctx, id)
	if err != nil {
		return err
	}

	query := "DELETE FROM analyzer_configs WHERE id = $1"
	result, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete analyzer config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrConfigNotFound{ID: id, ConfigType: "analyzer"}
	}

	history := &interfaces.ConfigHistory{
		ConfigID:   id,
		ConfigType: "analyzer",
		Action:     "deleted",
		OldValue:   oldConfig.Parameters,
	}
	_ = r.RecordHistory(ctx, history)

	return nil
}

// ========== Processor Configuration Methods ==========

// CreateProcessorConfig creates a new processor configuration
func (r *ConfigRepository) CreateProcessorConfig(ctx context.Context, config *interfaces.ProcessorConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	var exists bool
	err := r.store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM processor_configs WHERE id = $1)", config.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check processor config existence: %w", err)
	}
	if exists {
		return interfaces.ErrConfigExists{ID: config.ID, ConfigType: "processor"}
	}

	paramsJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO processor_configs (id, name, description, type, version, parameters, "order", enabled, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.store.pool.Exec(ctx, query,
		config.ID,
		config.Name,
		config.Description,
		config.Type,
		config.Version,
		paramsJSON,
		config.Order,
		config.Enabled,
		config.CreatedBy,
		config.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create processor config: %w", err)
	}

	history := &interfaces.ConfigHistory{
		ConfigID:   config.ID,
		ConfigType: "processor",
		Action:     "created",
		ChangedBy:  config.CreatedBy,
		NewValue:   config.Parameters,
	}
	_ = r.RecordHistory(ctx, history)

	return nil
}

// GetProcessorConfig retrieves a processor configuration by ID
func (r *ConfigRepository) GetProcessorConfig(ctx context.Context, id string) (*interfaces.ProcessorConfig, error) {
	query := `
		SELECT id, name, description, type, version, parameters, "order", enabled,
		       created_at, updated_at, created_by, updated_by
		FROM processor_configs
		WHERE id = $1
	`

	var config interfaces.ProcessorConfig
	var paramsJSON []byte

	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&config.ID,
		&config.Name,
		&config.Description,
		&config.Type,
		&config.Version,
		&paramsJSON,
		&config.Order,
		&config.Enabled,
		&config.CreatedAt,
		&config.UpdatedAt,
		&config.CreatedBy,
		&config.UpdatedBy,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrConfigNotFound{ID: id, ConfigType: "processor"}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get processor config: %w", err)
	}

	if err := json.Unmarshal(paramsJSON, &config.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	return &config, nil
}

// ListProcessorConfigs lists all processor configurations
func (r *ConfigRepository) ListProcessorConfigs(ctx context.Context, enabled *bool) ([]*interfaces.ProcessorConfig, error) {
	query := `
		SELECT id, name, description, type, version, parameters, "order", enabled,
		       created_at, updated_at, created_by, updated_by
		FROM processor_configs
	`

	var args []interface{}
	if enabled != nil {
		query += " WHERE enabled = $1"
		args = append(args, *enabled)
	}

	query += ` ORDER BY "order", created_at DESC`

	rows, err := r.store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list processor configs: %w", err)
	}
	defer rows.Close()

	var configs []*interfaces.ProcessorConfig
	for rows.Next() {
		var config interfaces.ProcessorConfig
		var paramsJSON []byte

		err := rows.Scan(
			&config.ID,
			&config.Name,
			&config.Description,
			&config.Type,
			&config.Version,
			&paramsJSON,
			&config.Order,
			&config.Enabled,
			&config.CreatedAt,
			&config.UpdatedAt,
			&config.CreatedBy,
			&config.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan processor config: %w", err)
		}

		if err := json.Unmarshal(paramsJSON, &config.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// UpdateProcessorConfig updates an existing processor configuration
func (r *ConfigRepository) UpdateProcessorConfig(ctx context.Context, config *interfaces.ProcessorConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	oldConfig, err := r.GetProcessorConfig(ctx, config.ID)
	if err != nil {
		return err
	}

	paramsJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		UPDATE processor_configs
		SET name = $2, description = $3, type = $4, version = $5, parameters = $6,
		    "order" = $7, enabled = $8, updated_by = $9
		WHERE id = $1
	`

	result, err := r.store.pool.Exec(ctx, query,
		config.ID,
		config.Name,
		config.Description,
		config.Type,
		config.Version,
		paramsJSON,
		config.Order,
		config.Enabled,
		config.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update processor config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrConfigNotFound{ID: config.ID, ConfigType: "processor"}
	}

	history := &interfaces.ConfigHistory{
		ConfigID:   config.ID,
		ConfigType: "processor",
		Action:     "updated",
		ChangedBy:  config.UpdatedBy,
		OldValue:   oldConfig.Parameters,
		NewValue:   config.Parameters,
	}
	_ = r.RecordHistory(ctx, history)

	return nil
}

// DeleteProcessorConfig deletes a processor configuration
func (r *ConfigRepository) DeleteProcessorConfig(ctx context.Context, id string) error {
	oldConfig, err := r.GetProcessorConfig(ctx, id)
	if err != nil {
		return err
	}

	query := "DELETE FROM processor_configs WHERE id = $1"
	result, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete processor config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrConfigNotFound{ID: id, ConfigType: "processor"}
	}

	history := &interfaces.ConfigHistory{
		ConfigID:   id,
		ConfigType: "processor",
		Action:     "deleted",
		OldValue:   oldConfig.Parameters,
	}
	_ = r.RecordHistory(ctx, history)

	return nil
}

// ========== Template Methods ==========

// CreateTemplate creates a new configuration template
func (r *ConfigRepository) CreateTemplate(ctx context.Context, template *interfaces.ConfigTemplate) error {
	if template.ID == "" || template.Name == "" || template.Type == "" {
		return interfaces.ErrInvalidConfig{Field: "template", Message: "id, name, and type are required"}
	}

	var exists bool
	err := r.store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM config_templates WHERE id = $1)", template.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check template existence: %w", err)
	}
	if exists {
		return interfaces.ErrConfigExists{ID: template.ID, ConfigType: "template"}
	}

	schemaJSON, err := json.Marshal(template.Schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	defaultsJSON, err := json.Marshal(template.Defaults)
	if err != nil {
		return fmt.Errorf("failed to marshal defaults: %w", err)
	}

	query := `
		INSERT INTO config_templates (id, name, description, type, category, schema, defaults)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.store.pool.Exec(ctx, query,
		template.ID,
		template.Name,
		template.Description,
		template.Type,
		template.Category,
		schemaJSON,
		defaultsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// GetTemplate retrieves a configuration template by ID
func (r *ConfigRepository) GetTemplate(ctx context.Context, id string) (*interfaces.ConfigTemplate, error) {
	query := `
		SELECT id, name, description, type, category, schema, defaults, created_at, updated_at
		FROM config_templates
		WHERE id = $1
	`

	var template interfaces.ConfigTemplate
	var schemaJSON, defaultsJSON []byte

	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&template.ID,
		&template.Name,
		&template.Description,
		&template.Type,
		&template.Category,
		&schemaJSON,
		&defaultsJSON,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrConfigNotFound{ID: id, ConfigType: "template"}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	if err := json.Unmarshal(schemaJSON, &template.Schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	if err := json.Unmarshal(defaultsJSON, &template.Defaults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal defaults: %w", err)
	}

	return &template, nil
}

// ListTemplates lists all configuration templates with optional type filtering
func (r *ConfigRepository) ListTemplates(ctx context.Context, configType string) ([]*interfaces.ConfigTemplate, error) {
	query := `
		SELECT id, name, description, type, category, schema, defaults, created_at, updated_at
		FROM config_templates
	`

	var args []interface{}
	if configType != "" {
		query += " WHERE type = $1"
		args = append(args, configType)
	}

	query += " ORDER BY category, name"

	rows, err := r.store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []*interfaces.ConfigTemplate
	for rows.Next() {
		var template interfaces.ConfigTemplate
		var schemaJSON, defaultsJSON []byte

		err := rows.Scan(
			&template.ID,
			&template.Name,
			&template.Description,
			&template.Type,
			&template.Category,
			&schemaJSON,
			&defaultsJSON,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		if err := json.Unmarshal(schemaJSON, &template.Schema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
		}

		if err := json.Unmarshal(defaultsJSON, &template.Defaults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal defaults: %w", err)
		}

		templates = append(templates, &template)
	}

	return templates, nil
}

// UpdateTemplate updates an existing configuration template
func (r *ConfigRepository) UpdateTemplate(ctx context.Context, template *interfaces.ConfigTemplate) error {
	if template.ID == "" || template.Name == "" || template.Type == "" {
		return interfaces.ErrInvalidConfig{Field: "template", Message: "id, name, and type are required"}
	}

	schemaJSON, err := json.Marshal(template.Schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	defaultsJSON, err := json.Marshal(template.Defaults)
	if err != nil {
		return fmt.Errorf("failed to marshal defaults: %w", err)
	}

	query := `
		UPDATE config_templates
		SET name = $2, description = $3, type = $4, category = $5, schema = $6, defaults = $7
		WHERE id = $1
	`

	result, err := r.store.pool.Exec(ctx, query,
		template.ID,
		template.Name,
		template.Description,
		template.Type,
		template.Category,
		schemaJSON,
		defaultsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrConfigNotFound{ID: template.ID, ConfigType: "template"}
	}

	return nil
}

// DeleteTemplate deletes a configuration template
func (r *ConfigRepository) DeleteTemplate(ctx context.Context, id string) error {
	query := "DELETE FROM config_templates WHERE id = $1"
	result, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrConfigNotFound{ID: id, ConfigType: "template"}
	}

	return nil
}

// ========== History Methods ==========

// RecordHistory records a configuration change in history
func (r *ConfigRepository) RecordHistory(ctx context.Context, history *interfaces.ConfigHistory) error {
	oldValueJSON, _ := json.Marshal(history.OldValue)
	newValueJSON, _ := json.Marshal(history.NewValue)

	query := `
		INSERT INTO config_history (config_id, config_type, action, changed_by, old_value, new_value, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.store.pool.Exec(ctx, query,
		history.ConfigID,
		history.ConfigType,
		history.Action,
		history.ChangedBy,
		oldValueJSON,
		newValueJSON,
		history.Reason,
	)

	if err != nil {
		return fmt.Errorf("failed to record history: %w", err)
	}

	return nil
}

// GetConfigHistory retrieves the history of a configuration
func (r *ConfigRepository) GetConfigHistory(ctx context.Context, configID string, configType string) ([]*interfaces.ConfigHistory, error) {
	query := `
		SELECT id, config_id, config_type, action, changed_by, changed_at, old_value, new_value, reason
		FROM config_history
		WHERE config_id = $1 AND config_type = $2
		ORDER BY changed_at DESC
	`

	rows, err := r.store.pool.Query(ctx, query, configID, configType)
	if err != nil {
		return nil, fmt.Errorf("failed to get config history: %w", err)
	}
	defer rows.Close()

	var history []*interfaces.ConfigHistory
	for rows.Next() {
		var h interfaces.ConfigHistory
		var oldValueJSON, newValueJSON pgtype.JSONB

		err := rows.Scan(
			&h.ID,
			&h.ConfigID,
			&h.ConfigType,
			&h.Action,
			&h.ChangedBy,
			&h.ChangedAt,
			&oldValueJSON,
			&newValueJSON,
			&h.Reason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}

		// Unmarshal JSONB fields
		if oldValueJSON.Valid {
			if err := json.Unmarshal(oldValueJSON.Bytes, &h.OldValue); err != nil {
				return nil, fmt.Errorf("failed to unmarshal old value: %w", err)
			}
		}

		if newValueJSON.Valid {
			if err := json.Unmarshal(newValueJSON.Bytes, &h.NewValue); err != nil {
				return nil, fmt.Errorf("failed to unmarshal new value: %w", err)
			}
		}

		history = append(history, &h)
	}

	return history, nil
}

// ========== Import/Export Methods ==========

// ExportConfigs exports all configurations of a specific type to JSON
func (r *ConfigRepository) ExportConfigs(ctx context.Context, configType string) ([]byte, error) {
	var data interface{}
	var err error

	switch configType {
	case "rule":
		data, err = r.ListRuleConfigs(ctx, nil, nil)
	case "analyzer":
		data, err = r.ListAnalyzerConfigs(ctx, nil)
	case "processor":
		data, err = r.ListProcessorConfigs(ctx, nil)
	default:
		return nil, interfaces.ErrInvalidConfig{Field: "config_type", Message: "invalid config type"}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list configs for export: %w", err)
	}

	return json.MarshalIndent(data, "", "  ")
}

// ImportConfigs imports configurations from JSON
func (r *ConfigRepository) ImportConfigs(ctx context.Context, data []byte, configType string) error {
	switch configType {
	case "rule":
		var configs []*interfaces.RuleConfig
		if err := json.Unmarshal(data, &configs); err != nil {
			return fmt.Errorf("failed to unmarshal rule configs: %w", err)
		}
		for _, config := range configs {
			if err := r.CreateRuleConfig(ctx, config); err != nil {
				// Skip if already exists
				if _, ok := err.(interfaces.ErrConfigExists); !ok {
					return err
				}
			}
		}

	case "analyzer":
		var configs []*interfaces.AnalyzerConfig
		if err := json.Unmarshal(data, &configs); err != nil {
			return fmt.Errorf("failed to unmarshal analyzer configs: %w", err)
		}
		for _, config := range configs {
			if err := r.CreateAnalyzerConfig(ctx, config); err != nil {
				if _, ok := err.(interfaces.ErrConfigExists); !ok {
					return err
				}
			}
		}

	case "processor":
		var configs []*interfaces.ProcessorConfig
		if err := json.Unmarshal(data, &configs); err != nil {
			return fmt.Errorf("failed to unmarshal processor configs: %w", err)
		}
		for _, config := range configs {
			if err := r.CreateProcessorConfig(ctx, config); err != nil {
				if _, ok := err.(interfaces.ErrConfigExists); !ok {
					return err
				}
			}
		}

	default:
		return interfaces.ErrInvalidConfig{Field: "config_type", Message: "invalid config type"}
	}

	return nil
}
