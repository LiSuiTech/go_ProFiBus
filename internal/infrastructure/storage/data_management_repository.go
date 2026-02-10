package storage

import (
	"context"
	"encoding/json"
	"fmt"
	dataManagementDomain "go_ProFiBus/internal/domain/datamanagement"
	"go_ProFiBus/pkg/interfaces"
	"time"

	"github.com/jackc/pgx/v5"
)

// DataManagementRepositoryImpl 数据管理仓储实现
type DataManagementRepositoryImpl struct {
	store *PostgresStore
}

// NewDataManagementRepository 创建数据管理仓储
func NewDataManagementRepository(store *PostgresStore) interfaces.DataManagementRepository {
	return &DataManagementRepositoryImpl{store: store}
}

// CreateCleaningRule 创建清洗规则
func (r *DataManagementRepositoryImpl) CreateCleaningRule(ctx context.Context, rule *dataManagementDomain.CleaningRule) error {
	configJSON, err := json.Marshal(rule.Config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	query := `
		INSERT INTO data_cleaning_rules (id, name, description, rule_type, enabled, config, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.store.Exec(query,
		rule.ID,
		rule.Name,
		rule.Description,
		string(rule.RuleType),
		rule.Enabled,
		configJSON,
		rule.Priority,
		rule.CreatedAt,
		rule.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建清洗规则失败: %w", err)
	}

	return nil
}

// GetCleaningRuleByID 根据ID获取清洗规则
func (r *DataManagementRepositoryImpl) GetCleaningRuleByID(ctx context.Context, id string) (*dataManagementDomain.CleaningRule, error) {
	query := `
		SELECT id, name, description, rule_type, enabled, config, priority, created_at, updated_at
		FROM data_cleaning_rules
		WHERE id = $1
	`

	var rule dataManagementDomain.CleaningRule
	var ruleType string
	var configJSON []byte

	err := r.store.QueryRow(query, id).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&ruleType,
		&rule.Enabled,
		&configJSON,
		&rule.Priority,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("清洗规则不存在: %s", id)
		}
		return nil, fmt.Errorf("查询清洗规则失败: %w", err)
	}

	rule.RuleType = dataManagementDomain.CleaningRuleType(ruleType)

	if err := json.Unmarshal(configJSON, &rule.Config); err != nil {
		r.store.Log().Warn("反序列化配置失败: %v", err)
		rule.Config = make(map[string]interface{})
	}

	return &rule, nil
}

// ListCleaningRules 列出清洗规则
func (r *DataManagementRepositoryImpl) ListCleaningRules(ctx context.Context, filters interfaces.CleaningRuleFilters) ([]*dataManagementDomain.CleaningRule, error) {
	query := `
		SELECT id, name, description, rule_type, enabled, config, priority, created_at, updated_at
		FROM data_cleaning_rules
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.RuleType != nil {
		query += fmt.Sprintf(" AND rule_type = $%d", argIndex)
		args = append(args, string(*filters.RuleType))
		argIndex++
	}

	if filters.Enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, *filters.Enabled)
		argIndex++
	}

	query += " ORDER BY priority DESC, created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询清洗规则列表失败: %w", err)
	}
	defer rows.Close()

	rules := make([]*dataManagementDomain.CleaningRule, 0)
	for rows.Next() {
		var rule dataManagementDomain.CleaningRule
		var ruleType string
		var configJSON []byte

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&ruleType,
			&rule.Enabled,
			&configJSON,
			&rule.Priority,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描清洗规则失败: %v", err)
			continue
		}

		rule.RuleType = dataManagementDomain.CleaningRuleType(ruleType)

		if err := json.Unmarshal(configJSON, &rule.Config); err != nil {
			r.store.Log().Warn("反序列化配置失败: %v", err)
			rule.Config = make(map[string]interface{})
		}

		rules = append(rules, &rule)
	}

	return rules, nil
}

// UpdateCleaningRule 更新清洗规则
func (r *DataManagementRepositoryImpl) UpdateCleaningRule(ctx context.Context, rule *dataManagementDomain.CleaningRule) error {
	configJSON, err := json.Marshal(rule.Config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	query := `
		UPDATE data_cleaning_rules
		SET name = $1, description = $2, rule_type = $3, enabled = $4, config = $5, priority = $6, updated_at = $7
		WHERE id = $8
	`

	tag, err := r.store.Exec(query,
		rule.Name,
		rule.Description,
		string(rule.RuleType),
		rule.Enabled,
		configJSON,
		rule.Priority,
		time.Now(),
		rule.ID,
	)

	if err != nil {
		return fmt.Errorf("更新清洗规则失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("清洗规则不存在: %s", rule.ID)
	}

	return nil
}

// DeleteCleaningRule 删除清洗规则
func (r *DataManagementRepositoryImpl) DeleteCleaningRule(ctx context.Context, id string) error {
	query := `DELETE FROM data_cleaning_rules WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除清洗规则失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("清洗规则不存在: %s", id)
	}

	return nil
}

// CreateArchivePolicy 创建归档策略
func (r *DataManagementRepositoryImpl) CreateArchivePolicy(ctx context.Context, policy *dataManagementDomain.ArchivePolicy) error {
	metadataJSON, err := json.Marshal(policy.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO data_archive_policies (id, name, description, source_type, source_id,
		                                  retention_days, archive_after_days, compression_enabled,
		                                  archive_location, enabled, last_run_at, next_run_at,
		                                  run_interval_hours, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err = r.store.Exec(query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.SourceType,
		policy.SourceID,
		policy.RetentionDays,
		policy.ArchiveAfterDays,
		policy.CompressionEnabled,
		policy.ArchiveLocation,
		policy.Enabled,
		policy.LastRunAt,
		policy.NextRunAt,
		policy.RunIntervalHours,
		metadataJSON,
		policy.CreatedAt,
		policy.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建归档策略失败: %w", err)
	}

	return nil
}

// GetArchivePolicyByID 根据ID获取归档策略
func (r *DataManagementRepositoryImpl) GetArchivePolicyByID(ctx context.Context, id string) (*dataManagementDomain.ArchivePolicy, error) {
	query := `
		SELECT id, name, description, source_type, source_id, retention_days, archive_after_days,
		       compression_enabled, archive_location, enabled, last_run_at, next_run_at,
		       run_interval_hours, metadata, created_at, updated_at
		FROM data_archive_policies
		WHERE id = $1
	`

	var policy dataManagementDomain.ArchivePolicy
	var metadataJSON []byte

	err := r.store.QueryRow(query, id).Scan(
		&policy.ID,
		&policy.Name,
		&policy.Description,
		&policy.SourceType,
		&policy.SourceID,
		&policy.RetentionDays,
		&policy.ArchiveAfterDays,
		&policy.CompressionEnabled,
		&policy.ArchiveLocation,
		&policy.Enabled,
		&policy.LastRunAt,
		&policy.NextRunAt,
		&policy.RunIntervalHours,
		&metadataJSON,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("归档策略不存在: %s", id)
		}
		return nil, fmt.Errorf("查询归档策略失败: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
		r.store.Log().Warn("反序列化元数据失败: %v", err)
		policy.Metadata = make(map[string]interface{})
	}

	return &policy, nil
}

// ListArchivePolicies 列出归档策略
func (r *DataManagementRepositoryImpl) ListArchivePolicies(ctx context.Context, filters interfaces.ArchivePolicyFilters) ([]*dataManagementDomain.ArchivePolicy, error) {
	query := `
		SELECT id, name, description, source_type, source_id, retention_days, archive_after_days,
		       compression_enabled, archive_location, enabled, last_run_at, next_run_at,
		       run_interval_hours, metadata, created_at, updated_at
		FROM data_archive_policies
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.SourceType != nil {
		query += fmt.Sprintf(" AND source_type = $%d", argIndex)
		args = append(args, *filters.SourceType)
		argIndex++
	}

	if filters.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, *filters.SourceID)
		argIndex++
	}

	if filters.Enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, *filters.Enabled)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询归档策略列表失败: %w", err)
	}
	defer rows.Close()

	policies := make([]*dataManagementDomain.ArchivePolicy, 0)
	for rows.Next() {
		var policy dataManagementDomain.ArchivePolicy
		var metadataJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.Name,
			&policy.Description,
			&policy.SourceType,
			&policy.SourceID,
			&policy.RetentionDays,
			&policy.ArchiveAfterDays,
			&policy.CompressionEnabled,
			&policy.ArchiveLocation,
			&policy.Enabled,
			&policy.LastRunAt,
			&policy.NextRunAt,
			&policy.RunIntervalHours,
			&metadataJSON,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描归档策略失败: %v", err)
			continue
		}

		if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
			r.store.Log().Warn("反序列化元数据失败: %v", err)
			policy.Metadata = make(map[string]interface{})
		}

		policies = append(policies, &policy)
	}

	return policies, nil
}

// UpdateArchivePolicy 更新归档策略
func (r *DataManagementRepositoryImpl) UpdateArchivePolicy(ctx context.Context, policy *dataManagementDomain.ArchivePolicy) error {
	metadataJSON, err := json.Marshal(policy.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE data_archive_policies
		SET name = $1, description = $2, source_type = $3, source_id = $4,
		    retention_days = $5, archive_after_days = $6, compression_enabled = $7,
		    archive_location = $8, enabled = $9, last_run_at = $10, next_run_at = $11,
		    run_interval_hours = $12, metadata = $13, updated_at = $14
		WHERE id = $15
	`

	tag, err := r.store.Exec(query,
		policy.Name,
		policy.Description,
		policy.SourceType,
		policy.SourceID,
		policy.RetentionDays,
		policy.ArchiveAfterDays,
		policy.CompressionEnabled,
		policy.ArchiveLocation,
		policy.Enabled,
		policy.LastRunAt,
		policy.NextRunAt,
		policy.RunIntervalHours,
		metadataJSON,
		time.Now(),
		policy.ID,
	)

	if err != nil {
		return fmt.Errorf("更新归档策略失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("归档策略不存在: %s", policy.ID)
	}

	return nil
}

// DeleteArchivePolicy 删除归档策略
func (r *DataManagementRepositoryImpl) DeleteArchivePolicy(ctx context.Context, id string) error {
	query := `DELETE FROM data_archive_policies WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除归档策略失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("归档策略不存在: %s", id)
	}

	return nil
}

// GetPoliciesToRun 获取需要执行的归档策略
func (r *DataManagementRepositoryImpl) GetPoliciesToRun(ctx context.Context) ([]*dataManagementDomain.ArchivePolicy, error) {
	filters := interfaces.ArchivePolicyFilters{
		Enabled: func() *bool { b := true; return &b }(),
		Limit:   100,
	}

	policies, err := r.ListArchivePolicies(ctx, filters)
	if err != nil {
		return nil, err
	}

	// 过滤出需要执行的策略
	toRun := make([]*dataManagementDomain.ArchivePolicy, 0)
	for _, policy := range policies {
		if policy.ShouldRun() {
			toRun = append(toRun, policy)
		}
	}

	return toRun, nil
}

// CreateArchiveRecord 创建归档记录
func (r *DataManagementRepositoryImpl) CreateArchiveRecord(ctx context.Context, record *dataManagementDomain.ArchiveRecord) error {
	query := `
		INSERT INTO data_archive_records (id, policy_id, source_type, source_id, start_time, end_time,
		                                record_count, archive_size, archive_path, status, error_message, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.store.Exec(query,
		record.ID,
		record.PolicyID,
		record.SourceType,
		record.SourceID,
		record.StartTime,
		record.EndTime,
		record.RecordCount,
		record.ArchiveSize,
		record.ArchivePath,
		string(record.Status),
		record.ErrorMessage,
		record.CreatedAt,
		record.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("创建归档记录失败: %w", err)
	}

	return nil
}

// GetArchiveRecordByID 根据ID获取归档记录
func (r *DataManagementRepositoryImpl) GetArchiveRecordByID(ctx context.Context, id string) (*dataManagementDomain.ArchiveRecord, error) {
	query := `
		SELECT id, policy_id, source_type, source_id, start_time, end_time,
		       record_count, archive_size, archive_path, status, error_message, created_at, completed_at
		FROM data_archive_records
		WHERE id = $1
	`

	var record dataManagementDomain.ArchiveRecord
	var status string

	err := r.store.QueryRow(query, id).Scan(
		&record.ID,
		&record.PolicyID,
		&record.SourceType,
		&record.SourceID,
		&record.StartTime,
		&record.EndTime,
		&record.RecordCount,
		&record.ArchiveSize,
		&record.ArchivePath,
		&status,
		&record.ErrorMessage,
		&record.CreatedAt,
		&record.CompletedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("归档记录不存在: %s", id)
		}
		return nil, fmt.Errorf("查询归档记录失败: %w", err)
	}

	record.Status = dataManagementDomain.ArchiveStatus(status)
	return &record, nil
}

// ListArchiveRecords 列出归档记录
func (r *DataManagementRepositoryImpl) ListArchiveRecords(ctx context.Context, filters interfaces.ArchiveRecordFilters) ([]*dataManagementDomain.ArchiveRecord, error) {
	query := `
		SELECT id, policy_id, source_type, source_id, start_time, end_time,
		       record_count, archive_size, archive_path, status, error_message, created_at, completed_at
		FROM data_archive_records
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.PolicyID != nil {
		query += fmt.Sprintf(" AND policy_id = $%d", argIndex)
		args = append(args, *filters.PolicyID)
		argIndex++
	}

	if filters.SourceType != nil {
		query += fmt.Sprintf(" AND source_type = $%d", argIndex)
		args = append(args, *filters.SourceType)
		argIndex++
	}

	if filters.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, *filters.SourceID)
		argIndex++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND start_time >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND end_time < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询归档记录列表失败: %w", err)
	}
	defer rows.Close()

	records := make([]*dataManagementDomain.ArchiveRecord, 0)
	for rows.Next() {
		var record dataManagementDomain.ArchiveRecord
		var status string

		err := rows.Scan(
			&record.ID,
			&record.PolicyID,
			&record.SourceType,
			&record.SourceID,
			&record.StartTime,
			&record.EndTime,
			&record.RecordCount,
			&record.ArchiveSize,
			&record.ArchivePath,
			&status,
			&record.ErrorMessage,
			&record.CreatedAt,
			&record.CompletedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描归档记录失败: %v", err)
			continue
		}

		record.Status = dataManagementDomain.ArchiveStatus(status)
		records = append(records, &record)
	}

	return records, nil
}

// UpdateArchiveRecord 更新归档记录
func (r *DataManagementRepositoryImpl) UpdateArchiveRecord(ctx context.Context, record *dataManagementDomain.ArchiveRecord) error {
	query := `
		UPDATE data_archive_records
		SET end_time = $1, record_count = $2, archive_size = $3, archive_path = $4,
		    status = $5, error_message = $6, completed_at = $7
		WHERE id = $8
	`

	tag, err := r.store.Exec(query,
		record.EndTime,
		record.RecordCount,
		record.ArchiveSize,
		record.ArchivePath,
		string(record.Status),
		record.ErrorMessage,
		record.CompletedAt,
		record.ID,
	)

	if err != nil {
		return fmt.Errorf("更新归档记录失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("归档记录不存在: %s", record.ID)
	}

	return nil
}

// CreateCleaningRecord 创建清洗记录
func (r *DataManagementRepositoryImpl) CreateCleaningRecord(ctx context.Context, record *dataManagementDomain.CleaningRecord) error {
	query := `
		INSERT INTO data_cleaning_records (id, rule_id, source_type, source_id, processed_count,
		                                  cleaned_count, removed_count, filled_count, start_time,
		                                  end_time, status, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.store.Exec(query,
		record.ID,
		record.RuleID,
		record.SourceType,
		record.SourceID,
		record.ProcessedCount,
		record.CleanedCount,
		record.RemovedCount,
		record.FilledCount,
		record.StartTime,
		record.EndTime,
		string(record.Status),
		record.ErrorMessage,
		record.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建清洗记录失败: %w", err)
	}

	return nil
}

// GetCleaningRecordByID 根据ID获取清洗记录
func (r *DataManagementRepositoryImpl) GetCleaningRecordByID(ctx context.Context, id string) (*dataManagementDomain.CleaningRecord, error) {
	query := `
		SELECT id, rule_id, source_type, source_id, processed_count, cleaned_count,
		       removed_count, filled_count, start_time, end_time, status, error_message, created_at
		FROM data_cleaning_records
		WHERE id = $1
	`

	var record dataManagementDomain.CleaningRecord
	var status string

	err := r.store.QueryRow(query, id).Scan(
		&record.ID,
		&record.RuleID,
		&record.SourceType,
		&record.SourceID,
		&record.ProcessedCount,
		&record.CleanedCount,
		&record.RemovedCount,
		&record.FilledCount,
		&record.StartTime,
		&record.EndTime,
		&status,
		&record.ErrorMessage,
		&record.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("清洗记录不存在: %s", id)
		}
		return nil, fmt.Errorf("查询清洗记录失败: %w", err)
	}

	record.Status = dataManagementDomain.CleaningStatus(status)
	return &record, nil
}

// ListCleaningRecords 列出清洗记录
func (r *DataManagementRepositoryImpl) ListCleaningRecords(ctx context.Context, filters interfaces.CleaningRecordFilters) ([]*dataManagementDomain.CleaningRecord, error) {
	query := `
		SELECT id, rule_id, source_type, source_id, processed_count, cleaned_count,
		       removed_count, filled_count, start_time, end_time, status, error_message, created_at
		FROM data_cleaning_records
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.RuleID != nil {
		query += fmt.Sprintf(" AND rule_id = $%d", argIndex)
		args = append(args, *filters.RuleID)
		argIndex++
	}

	if filters.SourceType != nil {
		query += fmt.Sprintf(" AND source_type = $%d", argIndex)
		args = append(args, *filters.SourceType)
		argIndex++
	}

	if filters.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, *filters.SourceID)
		argIndex++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND start_time >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND start_time < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询清洗记录列表失败: %w", err)
	}
	defer rows.Close()

	records := make([]*dataManagementDomain.CleaningRecord, 0)
	for rows.Next() {
		var record dataManagementDomain.CleaningRecord
		var status string

		err := rows.Scan(
			&record.ID,
			&record.RuleID,
			&record.SourceType,
			&record.SourceID,
			&record.ProcessedCount,
			&record.CleanedCount,
			&record.RemovedCount,
			&record.FilledCount,
			&record.StartTime,
			&record.EndTime,
			&status,
			&record.ErrorMessage,
			&record.CreatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描清洗记录失败: %v", err)
			continue
		}

		record.Status = dataManagementDomain.CleaningStatus(status)
		records = append(records, &record)
	}

	return records, nil
}

// UpdateCleaningRecord 更新清洗记录
func (r *DataManagementRepositoryImpl) UpdateCleaningRecord(ctx context.Context, record *dataManagementDomain.CleaningRecord) error {
	query := `
		UPDATE data_cleaning_records
		SET processed_count = $1, cleaned_count = $2, removed_count = $3, filled_count = $4,
		    end_time = $5, status = $6, error_message = $7
		WHERE id = $8
	`

	tag, err := r.store.Exec(query,
		record.ProcessedCount,
		record.CleanedCount,
		record.RemovedCount,
		record.FilledCount,
		record.EndTime,
		string(record.Status),
		record.ErrorMessage,
		record.ID,
	)

	if err != nil {
		return fmt.Errorf("更新清洗记录失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("清洗记录不存在: %s", record.ID)
	}

	return nil
}

// CreateLifecycleConfig 创建生命周期配置
func (r *DataManagementRepositoryImpl) CreateLifecycleConfig(ctx context.Context, config *dataManagementDomain.LifecycleConfig) error {
	metadataJSON, err := json.Marshal(config.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO data_lifecycle_configs (id, source_type, source_id, hot_storage_days,
		                                  warm_storage_days, cold_storage_days, delete_after_days,
		                                  compression_after_days, enabled, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.store.Exec(query,
		config.ID,
		config.SourceType,
		config.SourceID,
		config.HotStorageDays,
		config.WarmStorageDays,
		config.ColdStorageDays,
		config.DeleteAfterDays,
		config.CompressionAfterDays,
		config.Enabled,
		metadataJSON,
		config.CreatedAt,
		config.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建生命周期配置失败: %w", err)
	}

	return nil
}

// GetLifecycleConfig 获取生命周期配置
func (r *DataManagementRepositoryImpl) GetLifecycleConfig(ctx context.Context, sourceType, sourceID string) (*dataManagementDomain.LifecycleConfig, error) {
	query := `
		SELECT id, source_type, source_id, hot_storage_days, warm_storage_days,
		       cold_storage_days, delete_after_days, compression_after_days, enabled,
		       metadata, created_at, updated_at
		FROM data_lifecycle_configs
		WHERE source_type = $1 AND source_id = $2
	`

	var config dataManagementDomain.LifecycleConfig
	var metadataJSON []byte

	err := r.store.QueryRow(query, sourceType, sourceID).Scan(
		&config.ID,
		&config.SourceType,
		&config.SourceID,
		&config.HotStorageDays,
		&config.WarmStorageDays,
		&config.ColdStorageDays,
		&config.DeleteAfterDays,
		&config.CompressionAfterDays,
		&config.Enabled,
		&metadataJSON,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("生命周期配置不存在: %s/%s", sourceType, sourceID)
		}
		return nil, fmt.Errorf("查询生命周期配置失败: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &config.Metadata); err != nil {
		r.store.Log().Warn("反序列化元数据失败: %v", err)
		config.Metadata = make(map[string]interface{})
	}

	return &config, nil
}

// ListLifecycleConfigs 列出生命周期配置
func (r *DataManagementRepositoryImpl) ListLifecycleConfigs(ctx context.Context, filters interfaces.LifecycleConfigFilters) ([]*dataManagementDomain.LifecycleConfig, error) {
	query := `
		SELECT id, source_type, source_id, hot_storage_days, warm_storage_days,
		       cold_storage_days, delete_after_days, compression_after_days, enabled,
		       metadata, created_at, updated_at
		FROM data_lifecycle_configs
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.SourceType != nil {
		query += fmt.Sprintf(" AND source_type = $%d", argIndex)
		args = append(args, *filters.SourceType)
		argIndex++
	}

	if filters.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, *filters.SourceID)
		argIndex++
	}

	if filters.Enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, *filters.Enabled)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询生命周期配置列表失败: %w", err)
	}
	defer rows.Close()

	configs := make([]*dataManagementDomain.LifecycleConfig, 0)
	for rows.Next() {
		var config dataManagementDomain.LifecycleConfig
		var metadataJSON []byte

		err := rows.Scan(
			&config.ID,
			&config.SourceType,
			&config.SourceID,
			&config.HotStorageDays,
			&config.WarmStorageDays,
			&config.ColdStorageDays,
			&config.DeleteAfterDays,
			&config.CompressionAfterDays,
			&config.Enabled,
			&metadataJSON,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描生命周期配置失败: %v", err)
			continue
		}

		if err := json.Unmarshal(metadataJSON, &config.Metadata); err != nil {
			r.store.Log().Warn("反序列化元数据失败: %v", err)
			config.Metadata = make(map[string]interface{})
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// UpdateLifecycleConfig 更新生命周期配置
func (r *DataManagementRepositoryImpl) UpdateLifecycleConfig(ctx context.Context, config *dataManagementDomain.LifecycleConfig) error {
	metadataJSON, err := json.Marshal(config.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE data_lifecycle_configs
		SET hot_storage_days = $1, warm_storage_days = $2, cold_storage_days = $3,
		    delete_after_days = $4, compression_after_days = $5, enabled = $6,
		    metadata = $7, updated_at = $8
		WHERE id = $9
	`

	tag, err := r.store.Exec(query,
		config.HotStorageDays,
		config.WarmStorageDays,
		config.ColdStorageDays,
		config.DeleteAfterDays,
		config.CompressionAfterDays,
		config.Enabled,
		metadataJSON,
		time.Now(),
		config.ID,
	)

	if err != nil {
		return fmt.Errorf("更新生命周期配置失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("生命周期配置不存在: %s", config.ID)
	}

	return nil
}

// DeleteLifecycleConfig 删除生命周期配置
func (r *DataManagementRepositoryImpl) DeleteLifecycleConfig(ctx context.Context, sourceType, sourceID string) error {
	query := `DELETE FROM data_lifecycle_configs WHERE source_type = $1 AND source_id = $2`

	tag, err := r.store.Exec(query, sourceType, sourceID)
	if err != nil {
		return fmt.Errorf("删除生命周期配置失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("生命周期配置不存在: %s/%s", sourceType, sourceID)
	}

	return nil
}
