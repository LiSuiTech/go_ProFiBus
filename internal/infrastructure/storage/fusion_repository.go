package storage

import (
	"context"
	"encoding/json"
	"fmt"
	fusionDomain "go_ProFiBus/internal/domain/fusion"
	"go_ProFiBus/pkg/interfaces"
	"time"

	"github.com/jackc/pgx/v5"
)

// FusionRepositoryImpl 通用融合仓储实现
type FusionRepositoryImpl struct {
	store *PostgresStore
}

// NewFusionRepository 创建融合仓储
func NewFusionRepository(store *PostgresStore) interfaces.FusionRepository {
	return &FusionRepositoryImpl{store: store}
}

// CreateDataSource 创建数据源
func (r *FusionRepositoryImpl) CreateDataSource(ctx context.Context, source *fusionDomain.DataSource) error {
	configJSON, err := json.Marshal(source.SourceConfig)
	if err != nil {
		return fmt.Errorf("序列化数据源配置失败: %w", err)
	}

	metadataJSON, err := json.Marshal(source.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO fusion_data_sources (id, source_name, source_type, device_id, channel_id,
		                                field_name, source_config, fusion_weight, enabled, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.store.Exec(query,
		source.ID,
		source.SourceName,
		string(source.SourceType),
		source.DeviceID,
		source.ChannelID,
		source.FieldName,
		configJSON,
		source.FusionWeight,
		source.Enabled,
		metadataJSON,
		source.CreatedAt,
		source.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建数据源失败: %w", err)
	}

	return nil
}

// GetDataSourceByID 根据ID获取数据源
func (r *FusionRepositoryImpl) GetDataSourceByID(ctx context.Context, id string) (*fusionDomain.DataSource, error) {
	query := `
		SELECT id, source_name, source_type, device_id, channel_id, field_name,
		       source_config, fusion_weight, enabled, metadata, created_at, updated_at
		FROM fusion_data_sources
		WHERE id = $1
	`

	var source fusionDomain.DataSource
	var sourceType string
	var configJSON, metadataJSON []byte

	err := r.store.QueryRow(query, id).Scan(
		&source.ID,
		&source.SourceName,
		&sourceType,
		&source.DeviceID,
		&source.ChannelID,
		&source.FieldName,
		&configJSON,
		&source.FusionWeight,
		&source.Enabled,
		&metadataJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("数据源不存在: %s", id)
		}
		return nil, fmt.Errorf("查询数据源失败: %w", err)
	}

	source.SourceType = fusionDomain.SourceType(sourceType)

	if err := json.Unmarshal(configJSON, &source.SourceConfig); err != nil {
		r.store.log.Warn("反序列化数据源配置失败: %v", err)
		source.SourceConfig = make(map[string]interface{})
	}

	if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
		r.store.log.Warn("反序列化元数据失败: %v", err)
		source.Metadata = make(map[string]interface{})
	}

	return &source, nil
}

// GetDataSourceByName 根据名称获取数据源
func (r *FusionRepositoryImpl) GetDataSourceByName(ctx context.Context, name string) (*fusionDomain.DataSource, error) {
	query := `
		SELECT id, source_name, source_type, device_id, channel_id, field_name,
		       source_config, fusion_weight, enabled, metadata, created_at, updated_at
		FROM fusion_data_sources
		WHERE source_name = $1
	`

	var source fusionDomain.DataSource
	var sourceType string
	var configJSON, metadataJSON []byte

	err := r.store.QueryRow(query, name).Scan(
		&source.ID,
		&source.SourceName,
		&sourceType,
		&source.DeviceID,
		&source.ChannelID,
		&source.FieldName,
		&configJSON,
		&source.FusionWeight,
		&source.Enabled,
		&metadataJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("数据源不存在: %s", name)
		}
		return nil, fmt.Errorf("查询数据源失败: %w", err)
	}

	source.SourceType = fusionDomain.SourceType(sourceType)

	if err := json.Unmarshal(configJSON, &source.SourceConfig); err != nil {
		r.store.log.Warn("反序列化数据源配置失败: %v", err)
		source.SourceConfig = make(map[string]interface{})
	}

	if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
		r.store.log.Warn("反序列化元数据失败: %v", err)
		source.Metadata = make(map[string]interface{})
	}

	return &source, nil
}

// ListDataSources 列出数据源
func (r *FusionRepositoryImpl) ListDataSources(ctx context.Context, filters interfaces.DataSourceFilters) ([]*fusionDomain.DataSource, error) {
	query := `
		SELECT id, source_name, source_type, device_id, channel_id, field_name,
		       source_config, fusion_weight, enabled, metadata, created_at, updated_at
		FROM fusion_data_sources
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.SourceType != nil {
		query += fmt.Sprintf(" AND source_type = $%d", argIndex)
		args = append(args, string(*filters.SourceType))
		argIndex++
	}

	if filters.DeviceID != nil {
		query += fmt.Sprintf(" AND device_id = $%d", argIndex)
		args = append(args, *filters.DeviceID)
		argIndex++
	}

	if filters.ChannelID != nil {
		query += fmt.Sprintf(" AND channel_id = $%d", argIndex)
		args = append(args, *filters.ChannelID)
		argIndex++
	}

	if filters.Enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, *filters.Enabled)
		argIndex++
	}

	query += " ORDER BY source_name"

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
		return nil, fmt.Errorf("查询数据源列表失败: %w", err)
	}
	defer rows.Close()

	sources := make([]*fusionDomain.DataSource, 0)
	for rows.Next() {
		var source fusionDomain.DataSource
		var sourceType string
		var configJSON, metadataJSON []byte

		err := rows.Scan(
			&source.ID,
			&source.SourceName,
			&sourceType,
			&source.DeviceID,
			&source.ChannelID,
			&source.FieldName,
			&configJSON,
			&source.FusionWeight,
			&source.Enabled,
			&metadataJSON,
			&source.CreatedAt,
			&source.UpdatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描数据源失败: %v", err)
			continue
		}

		source.SourceType = fusionDomain.SourceType(sourceType)

		if err := json.Unmarshal(configJSON, &source.SourceConfig); err != nil {
			r.store.log.Warn("反序列化数据源配置失败: %v", err)
			source.SourceConfig = make(map[string]interface{})
		}

		if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
			r.store.log.Warn("反序列化元数据失败: %v", err)
			source.Metadata = make(map[string]interface{})
		}

		sources = append(sources, &source)
	}

	return sources, nil
}

// UpdateDataSource 更新数据源
func (r *FusionRepositoryImpl) UpdateDataSource(ctx context.Context, source *fusionDomain.DataSource) error {
	configJSON, err := json.Marshal(source.SourceConfig)
	if err != nil {
		return fmt.Errorf("序列化数据源配置失败: %w", err)
	}

	metadataJSON, err := json.Marshal(source.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE fusion_data_sources
		SET source_name = $1, source_type = $2, device_id = $3, channel_id = $4,
		    field_name = $5, source_config = $6, fusion_weight = $7, enabled = $8,
		    metadata = $9, updated_at = $10
		WHERE id = $11
	`

	tag, err := r.store.Exec(query,
		source.SourceName,
		string(source.SourceType),
		source.DeviceID,
		source.ChannelID,
		source.FieldName,
		configJSON,
		source.FusionWeight,
		source.Enabled,
		metadataJSON,
		time.Now(),
		source.ID,
	)

	if err != nil {
		return fmt.Errorf("更新数据源失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("数据源不存在: %s", source.ID)
	}

	return nil
}

// DeleteDataSource 删除数据源
func (r *FusionRepositoryImpl) DeleteDataSource(ctx context.Context, id string) error {
	query := `DELETE FROM fusion_data_sources WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除数据源失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("数据源不存在: %s", id)
	}

	return nil
}

// CreateFusionConfig 创建融合配置
func (r *FusionRepositoryImpl) CreateFusionConfig(ctx context.Context, config *fusionDomain.FusionConfig) error {
	sourceWeightsJSON, err := json.Marshal(config.SourceWeights)
	if err != nil {
		return fmt.Errorf("序列化数据源权重失败: %w", err)
	}

	fieldWeightsJSON, err := json.Marshal(config.FieldWeights)
	if err != nil {
		return fmt.Errorf("序列化字段权重失败: %w", err)
	}

	outputFieldsJSON, err := json.Marshal(config.OutputFields)
	if err != nil {
		return fmt.Errorf("序列化输出字段失败: %w", err)
	}

	metadataJSON, err := json.Marshal(config.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO fusion_configs (id, name, description, fusion_strategy, time_window_ms,
		                          min_sources, source_weights, field_weights, output_fields,
		                          enabled, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.store.Exec(query,
		config.ID,
		config.Name,
		config.Description,
		config.FusionStrategy,
		config.TimeWindowMs,
		config.MinSources,
		sourceWeightsJSON,
		fieldWeightsJSON,
		outputFieldsJSON,
		config.Enabled,
		metadataJSON,
		config.CreatedAt,
		config.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建融合配置失败: %w", err)
	}

	return nil
}

// GetFusionConfigByID 根据ID获取融合配置
func (r *FusionRepositoryImpl) GetFusionConfigByID(ctx context.Context, id string) (*fusionDomain.FusionConfig, error) {
	query := `
		SELECT id, name, description, fusion_strategy, time_window_ms, min_sources,
		       source_weights, field_weights, output_fields, enabled, metadata, created_at, updated_at
		FROM fusion_configs
		WHERE id = $1
	`

	var config fusionDomain.FusionConfig
	var sourceWeightsJSON, fieldWeightsJSON, outputFieldsJSON, metadataJSON []byte

	err := r.store.QueryRow(query, id).Scan(
		&config.ID,
		&config.Name,
		&config.Description,
		&config.FusionStrategy,
		&config.TimeWindowMs,
		&config.MinSources,
		&sourceWeightsJSON,
		&fieldWeightsJSON,
		&outputFieldsJSON,
		&metadataJSON,
		&config.Enabled,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("融合配置不存在: %s", id)
		}
		return nil, fmt.Errorf("查询融合配置失败: %w", err)
	}

	if err := json.Unmarshal(sourceWeightsJSON, &config.SourceWeights); err != nil {
		r.store.log.Warn("反序列化数据源权重失败: %v", err)
		config.SourceWeights = make(map[string]float64)
	}

	if err := json.Unmarshal(fieldWeightsJSON, &config.FieldWeights); err != nil {
		r.store.log.Warn("反序列化字段权重失败: %v", err)
		config.FieldWeights = make(map[string]float64)
	}

	if err := json.Unmarshal(outputFieldsJSON, &config.OutputFields); err != nil {
		r.store.log.Warn("反序列化输出字段失败: %v", err)
		config.OutputFields = make([]string, 0)
	}

	if err := json.Unmarshal(metadataJSON, &config.Metadata); err != nil {
		r.store.log.Warn("反序列化元数据失败: %v", err)
		config.Metadata = make(map[string]interface{})
	}

	return &config, nil
}

// GetFusionConfigByName 根据名称获取融合配置
func (r *FusionRepositoryImpl) GetFusionConfigByName(ctx context.Context, name string) (*fusionDomain.FusionConfig, error) {
	query := `
		SELECT id, name, description, fusion_strategy, time_window_ms, min_sources,
		       source_weights, field_weights, output_fields, enabled, metadata, created_at, updated_at
		FROM fusion_configs
		WHERE name = $1
	`

	var config fusionDomain.FusionConfig
	var sourceWeightsJSON, fieldWeightsJSON, outputFieldsJSON, metadataJSON []byte

	err := r.store.QueryRow(query, name).Scan(
		&config.ID,
		&config.Name,
		&config.Description,
		&config.FusionStrategy,
		&config.TimeWindowMs,
		&config.MinSources,
		&sourceWeightsJSON,
		&fieldWeightsJSON,
		&outputFieldsJSON,
		&metadataJSON,
		&config.Enabled,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("融合配置不存在: %s", name)
		}
		return nil, fmt.Errorf("查询融合配置失败: %w", err)
	}

	if err := json.Unmarshal(sourceWeightsJSON, &config.SourceWeights); err != nil {
		r.store.log.Warn("反序列化数据源权重失败: %v", err)
		config.SourceWeights = make(map[string]float64)
	}

	if err := json.Unmarshal(fieldWeightsJSON, &config.FieldWeights); err != nil {
		r.store.log.Warn("反序列化字段权重失败: %v", err)
		config.FieldWeights = make(map[string]float64)
	}

	if err := json.Unmarshal(outputFieldsJSON, &config.OutputFields); err != nil {
		r.store.log.Warn("反序列化输出字段失败: %v", err)
		config.OutputFields = make([]string, 0)
	}

	if err := json.Unmarshal(metadataJSON, &config.Metadata); err != nil {
		r.store.log.Warn("反序列化元数据失败: %v", err)
		config.Metadata = make(map[string]interface{})
	}

	return &config, nil
}

// ListFusionConfigs 列出融合配置
func (r *FusionRepositoryImpl) ListFusionConfigs(ctx context.Context, filters interfaces.FusionConfigFilters) ([]*fusionDomain.FusionConfig, error) {
	query := `
		SELECT id, name, description, fusion_strategy, time_window_ms, min_sources,
		       source_weights, field_weights, output_fields, enabled, metadata, created_at, updated_at
		FROM fusion_configs
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.Enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, *filters.Enabled)
		argIndex++
	}

	query += " ORDER BY name"

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
		return nil, fmt.Errorf("查询融合配置列表失败: %w", err)
	}
	defer rows.Close()

	configs := make([]*fusionDomain.FusionConfig, 0)
	for rows.Next() {
		var config fusionDomain.FusionConfig
		var sourceWeightsJSON, fieldWeightsJSON, outputFieldsJSON, metadataJSON []byte

		err := rows.Scan(
			&config.ID,
			&config.Name,
			&config.Description,
			&config.FusionStrategy,
			&config.TimeWindowMs,
			&config.MinSources,
			&sourceWeightsJSON,
			&fieldWeightsJSON,
			&outputFieldsJSON,
			&metadataJSON,
			&config.Enabled,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描融合配置失败: %v", err)
			continue
		}

		if err := json.Unmarshal(sourceWeightsJSON, &config.SourceWeights); err != nil {
			r.store.log.Warn("反序列化数据源权重失败: %v", err)
			config.SourceWeights = make(map[string]float64)
		}

		if err := json.Unmarshal(fieldWeightsJSON, &config.FieldWeights); err != nil {
			r.store.log.Warn("反序列化字段权重失败: %v", err)
			config.FieldWeights = make(map[string]float64)
		}

		if err := json.Unmarshal(outputFieldsJSON, &config.OutputFields); err != nil {
			r.store.log.Warn("反序列化输出字段失败: %v", err)
			config.OutputFields = make([]string, 0)
		}

		if err := json.Unmarshal(metadataJSON, &config.Metadata); err != nil {
			r.store.log.Warn("反序列化元数据失败: %v", err)
			config.Metadata = make(map[string]interface{})
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// UpdateFusionConfig 更新融合配置
func (r *FusionRepositoryImpl) UpdateFusionConfig(ctx context.Context, config *fusionDomain.FusionConfig) error {
	sourceWeightsJSON, err := json.Marshal(config.SourceWeights)
	if err != nil {
		return fmt.Errorf("序列化数据源权重失败: %w", err)
	}

	fieldWeightsJSON, err := json.Marshal(config.FieldWeights)
	if err != nil {
		return fmt.Errorf("序列化字段权重失败: %w", err)
	}

	outputFieldsJSON, err := json.Marshal(config.OutputFields)
	if err != nil {
		return fmt.Errorf("序列化输出字段失败: %w", err)
	}

	metadataJSON, err := json.Marshal(config.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE fusion_configs
		SET name = $1, description = $2, fusion_strategy = $3, time_window_ms = $4,
		    min_sources = $5, source_weights = $6, field_weights = $7, output_fields = $8,
		    enabled = $9, metadata = $10, updated_at = $11
		WHERE id = $12
	`

	tag, err := r.store.Exec(query,
		config.Name,
		config.Description,
		config.FusionStrategy,
		config.TimeWindowMs,
		config.MinSources,
		sourceWeightsJSON,
		fieldWeightsJSON,
		outputFieldsJSON,
		config.Enabled,
		metadataJSON,
		time.Now(),
		config.ID,
	)

	if err != nil {
		return fmt.Errorf("更新融合配置失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("融合配置不存在: %s", config.ID)
	}

	return nil
}

// DeleteFusionConfig 删除融合配置
func (r *FusionRepositoryImpl) DeleteFusionConfig(ctx context.Context, id string) error {
	query := `DELETE FROM fusion_configs WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除融合配置失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("融合配置不存在: %s", id)
	}

	return nil
}

// AddSourceToConfig 添加数据源到融合配置
func (r *FusionRepositoryImpl) AddSourceToConfig(ctx context.Context, configID, sourceID string, weight float64) error {
	query := `
		INSERT INTO fusion_config_sources (fusion_config_id, source_id, weight, enabled, created_at)
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (fusion_config_id, source_id) DO UPDATE SET weight = $3, enabled = true
	`

	_, err := r.store.Exec(query, configID, sourceID, weight, time.Now())
	if err != nil {
		return fmt.Errorf("添加数据源到配置失败: %w", err)
	}

	return nil
}

// RemoveSourceFromConfig 从融合配置移除数据源
func (r *FusionRepositoryImpl) RemoveSourceFromConfig(ctx context.Context, configID, sourceID string) error {
	query := `DELETE FROM fusion_config_sources WHERE fusion_config_id = $1 AND source_id = $2`

	_, err := r.store.Exec(query, configID, sourceID)
	if err != nil {
		return fmt.Errorf("从配置移除数据源失败: %w", err)
	}

	return nil
}

// GetConfigSources 获取配置的数据源
func (r *FusionRepositoryImpl) GetConfigSources(ctx context.Context, configID string) ([]*fusionDomain.ConfigSourceRelation, error) {
	query := `
		SELECT fusion_config_id, source_id, weight, enabled, created_at
		FROM fusion_config_sources
		WHERE fusion_config_id = $1
		ORDER BY created_at
	`

	rows, err := r.store.Query(query, configID)
	if err != nil {
		return nil, fmt.Errorf("查询配置数据源失败: %w", err)
	}
	defer rows.Close()

	relations := make([]*fusionDomain.ConfigSourceRelation, 0)
	for rows.Next() {
		var rel fusionDomain.ConfigSourceRelation
		err := rows.Scan(
			&rel.FusionConfigID,
			&rel.SourceID,
			&rel.Weight,
			&rel.Enabled,
			&rel.CreatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描配置数据源关系失败: %v", err)
			continue
		}
		relations = append(relations, &rel)
	}

	return relations, nil
}

// UpdateConfigSourceWeight 更新配置中的数据源权重
func (r *FusionRepositoryImpl) UpdateConfigSourceWeight(ctx context.Context, configID, sourceID string, weight float64) error {
	query := `
		UPDATE fusion_config_sources
		SET weight = $1
		WHERE fusion_config_id = $2 AND source_id = $3
	`

	tag, err := r.store.Exec(query, weight, configID, sourceID)
	if err != nil {
		return fmt.Errorf("更新数据源权重失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("配置数据源关系不存在")
	}

	return nil
}

// SaveFusionResult 保存融合结果
func (r *FusionRepositoryImpl) SaveFusionResult(ctx context.Context, result *fusionDomain.FusionResult) error {
	fusedDataJSON, err := json.Marshal(result.FusedData)
	if err != nil {
		return fmt.Errorf("序列化融合数据失败: %w", err)
	}

	metadataJSON, err := json.Marshal(result.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO fusion_results (id, fusion_config_id, fusion_config_name, timestamp,
		                          fused_data, source_count, source_ids, fusion_strategy,
		                          quality_score, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.store.Exec(query,
		result.ID,
		result.FusionConfigID,
		result.FusionConfigName,
		result.Timestamp,
		fusedDataJSON,
		result.SourceCount,
		result.SourceIDs,
		result.FusionStrategy,
		result.QualityScore,
		metadataJSON,
		result.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("保存融合结果失败: %w", err)
	}

	return nil
}

// GetFusionResults 获取融合结果
func (r *FusionRepositoryImpl) GetFusionResults(ctx context.Context, filters interfaces.FusionResultFilters) ([]*fusionDomain.FusionResult, error) {
	query := `
		SELECT id, fusion_config_id, fusion_config_name, timestamp, fused_data,
		       source_count, source_ids, fusion_strategy, quality_score, metadata, created_at
		FROM fusion_results
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.FusionConfigID != nil {
		query += fmt.Sprintf(" AND fusion_config_id = $%d", argIndex)
		args = append(args, *filters.FusionConfigID)
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	query += " ORDER BY timestamp DESC"

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
		return nil, fmt.Errorf("查询融合结果失败: %w", err)
	}
	defer rows.Close()

	results := make([]*fusionDomain.FusionResult, 0)
	for rows.Next() {
		var result fusionDomain.FusionResult
		var fusedDataJSON, metadataJSON []byte

		err := rows.Scan(
			&result.ID,
			&result.FusionConfigID,
			&result.FusionConfigName,
			&result.Timestamp,
			&fusedDataJSON,
			&result.SourceCount,
			&result.SourceIDs,
			&result.FusionStrategy,
			&result.QualityScore,
			&metadataJSON,
			&result.CreatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描融合结果失败: %v", err)
			continue
		}

		if err := json.Unmarshal(fusedDataJSON, &result.FusedData); err != nil {
			r.store.log.Warn("反序列化融合数据失败: %v", err)
			result.FusedData = make(map[string]interface{})
		}

		if err := json.Unmarshal(metadataJSON, &result.Metadata); err != nil {
			r.store.log.Warn("反序列化元数据失败: %v", err)
			result.Metadata = make(map[string]interface{})
		}

		results = append(results, &result)
	}

	return results, nil
}

// GetLatestFusionResult 获取最新融合结果
func (r *FusionRepositoryImpl) GetLatestFusionResult(ctx context.Context, configID string) (*fusionDomain.FusionResult, error) {
	query := `
		SELECT id, fusion_config_id, fusion_config_name, timestamp, fused_data,
		       source_count, source_ids, fusion_strategy, quality_score, metadata, created_at
		FROM fusion_results
		WHERE fusion_config_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var result fusionDomain.FusionResult
	var fusedDataJSON, metadataJSON []byte

	err := r.store.QueryRow(query, configID).Scan(
		&result.ID,
		&result.FusionConfigID,
		&result.FusionConfigName,
		&result.Timestamp,
		&fusedDataJSON,
		&result.SourceCount,
		&result.SourceIDs,
		&result.FusionStrategy,
		&result.QualityScore,
		&metadataJSON,
		&result.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("融合结果不存在: %s", configID)
		}
		return nil, fmt.Errorf("查询最新融合结果失败: %w", err)
	}

	if err := json.Unmarshal(fusedDataJSON, &result.FusedData); err != nil {
		r.store.log.Warn("反序列化融合数据失败: %v", err)
		result.FusedData = make(map[string]interface{})
	}

	if err := json.Unmarshal(metadataJSON, &result.Metadata); err != nil {
		r.store.log.Warn("反序列化元数据失败: %v", err)
		result.Metadata = make(map[string]interface{})
	}

	return &result, nil
}

// SaveSourceDataCache 保存数据源数据缓存
func (r *FusionRepositoryImpl) SaveSourceDataCache(ctx context.Context, cache *fusionDomain.SourceDataCache) error {
	dataJSON, err := json.Marshal(cache.Data)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	metadataJSON, err := json.Marshal(cache.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO fusion_source_data_cache (id, source_id, timestamp, data, quality, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.store.Exec(query,
		cache.ID,
		cache.SourceID,
		cache.Timestamp,
		dataJSON,
		cache.Quality,
		metadataJSON,
		cache.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("保存数据缓存失败: %w", err)
	}

	return nil
}

// GetSourceDataCache 获取数据源数据缓存
func (r *FusionRepositoryImpl) GetSourceDataCache(ctx context.Context, sourceID string, timeWindow time.Duration) ([]*fusionDomain.SourceDataCache, error) {
	cutoffTime := time.Now().Add(-timeWindow)

	query := `
		SELECT id, source_id, timestamp, data, quality, metadata, created_at
		FROM fusion_source_data_cache
		WHERE source_id = $1 AND timestamp >= $2
		ORDER BY timestamp DESC
	`

	rows, err := r.store.Query(query, sourceID, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("查询数据缓存失败: %w", err)
	}
	defer rows.Close()

	caches := make([]*fusionDomain.SourceDataCache, 0)
	for rows.Next() {
		var cache fusionDomain.SourceDataCache
		var dataJSON, metadataJSON []byte

		err := rows.Scan(
			&cache.ID,
			&cache.SourceID,
			&cache.Timestamp,
			&dataJSON,
			&cache.Quality,
			&metadataJSON,
			&cache.CreatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描数据缓存失败: %v", err)
			continue
		}

		if err := json.Unmarshal(dataJSON, &cache.Data); err != nil {
			r.store.log.Warn("反序列化数据失败: %v", err)
			cache.Data = make(map[string]interface{})
		}

		if err := json.Unmarshal(metadataJSON, &cache.Metadata); err != nil {
			r.store.log.Warn("反序列化元数据失败: %v", err)
			cache.Metadata = make(map[string]interface{})
		}

		caches = append(caches, &cache)
	}

	return caches, nil
}

// CleanExpiredCache 清理过期缓存
func (r *FusionRepositoryImpl) CleanExpiredCache(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM fusion_source_data_cache WHERE timestamp < $1`

	_, err := r.store.Exec(query, olderThan)
	if err != nil {
		return fmt.Errorf("清理过期缓存失败: %w", err)
	}

	return nil
}
