package storage

import (
	"context"
	"encoding/json"
	"fmt"
	deviceDomain "go_ProFiBus/internal/domain/device"
	"go_ProFiBus/pkg/interfaces"
	"time"

	"github.com/jackc/pgx/v5"
)

// DeviceDataRepositoryImpl 设备数据仓储实现
type DeviceDataRepositoryImpl struct {
	store *PostgresStore
}

// NewDeviceDataRepository 创建设备数据仓储
func NewDeviceDataRepository(store *PostgresStore) interfaces.DeviceDataRepository {
	return &DeviceDataRepositoryImpl{store: store}
}

// CreateDataField 创建数据字段
func (r *DeviceDataRepositoryImpl) CreateDataField(ctx context.Context, field *deviceDomain.DataField) error {
	query := `
		INSERT INTO device_data_fields (id, device_id, field_name, field_type, unit, min_value, max_value,
		                              default_value, description, enabled, fusion_weight, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.store.Exec(query,
		field.ID,
		field.DeviceID,
		field.FieldName,
		string(field.FieldType),
		field.Unit,
		field.MinValue,
		field.MaxValue,
		field.DefaultValue,
		field.Description,
		field.Enabled,
		field.FusionWeight,
		field.CreatedAt,
		field.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建数据字段失败: %w", err)
	}

	return nil
}

// GetDataFieldByID 根据ID获取数据字段
func (r *DeviceDataRepositoryImpl) GetDataFieldByID(ctx context.Context, id string) (*deviceDomain.DataField, error) {
	query := `
		SELECT id, device_id, field_name, field_type, unit, min_value, max_value, default_value,
		       description, enabled, fusion_weight, created_at, updated_at
		FROM device_data_fields
		WHERE id = $1
	`

	var field deviceDomain.DataField
	var fieldType string

	err := r.store.QueryRow(query, id).Scan(
		&field.ID,
		&field.DeviceID,
		&field.FieldName,
		&fieldType,
		&field.Unit,
		&field.MinValue,
		&field.MaxValue,
		&field.DefaultValue,
		&field.Description,
		&field.Enabled,
		&field.FusionWeight,
		&field.CreatedAt,
		&field.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("数据字段不存在: %s", id)
		}
		return nil, fmt.Errorf("查询数据字段失败: %w", err)
	}

	field.FieldType = deviceDomain.FieldType(fieldType)
	return &field, nil
}

// GetDataFieldsByDevice 获取设备的所有数据字段
func (r *DeviceDataRepositoryImpl) GetDataFieldsByDevice(ctx context.Context, deviceID string) ([]*deviceDomain.DataField, error) {
	query := `
		SELECT id, device_id, field_name, field_type, unit, min_value, max_value, default_value,
		       description, enabled, fusion_weight, created_at, updated_at
		FROM device_data_fields
		WHERE device_id = $1
		ORDER BY field_name
	`

	rows, err := r.store.Query(query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("查询数据字段列表失败: %w", err)
	}
	defer rows.Close()

	fields := make([]*deviceDomain.DataField, 0)
	for rows.Next() {
		var field deviceDomain.DataField
		var fieldType string

		err := rows.Scan(
			&field.ID,
			&field.DeviceID,
			&field.FieldName,
			&fieldType,
			&field.Unit,
			&field.MinValue,
			&field.MaxValue,
			&field.DefaultValue,
			&field.Description,
			&field.Enabled,
			&field.FusionWeight,
			&field.CreatedAt,
			&field.UpdatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描数据字段失败: %v", err)
			continue
		}

		field.FieldType = deviceDomain.FieldType(fieldType)
		fields = append(fields, &field)
	}

	return fields, nil
}

// UpdateDataField 更新数据字段
func (r *DeviceDataRepositoryImpl) UpdateDataField(ctx context.Context, field *deviceDomain.DataField) error {
	query := `
		UPDATE device_data_fields
		SET field_name = $1, field_type = $2, unit = $3, min_value = $4, max_value = $5,
		    default_value = $6, description = $7, enabled = $8, fusion_weight = $9, updated_at = $10
		WHERE id = $11
	`

	tag, err := r.store.Exec(query,
		field.FieldName,
		string(field.FieldType),
		field.Unit,
		field.MinValue,
		field.MaxValue,
		field.DefaultValue,
		field.Description,
		field.Enabled,
		field.FusionWeight,
		time.Now(),
		field.ID,
	)

	if err != nil {
		return fmt.Errorf("更新数据字段失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("数据字段不存在: %s", field.ID)
	}

	return nil
}

// DeleteDataField 删除数据字段
func (r *DeviceDataRepositoryImpl) DeleteDataField(ctx context.Context, id string) error {
	query := `DELETE FROM device_data_fields WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除数据字段失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("数据字段不存在: %s", id)
	}

	return nil
}

// CreateDataSource 创建数据源
func (r *DeviceDataRepositoryImpl) CreateDataSource(ctx context.Context, source *deviceDomain.DataSource) error {
	mappingJSON, err := json.Marshal(source.FieldMapping)
	if err != nil {
		return fmt.Errorf("序列化字段映射失败: %w", err)
	}

	metadataJSON, err := json.Marshal(source.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO device_data_sources (id, device_id, source_name, source_type, channel_id,
		                                field_mapping, fusion_enabled, fusion_weight, sample_rate,
		                                enabled, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.store.Exec(query,
		source.ID,
		source.DeviceID,
		source.SourceName,
		string(source.SourceType),
		source.ChannelID,
		mappingJSON,
		source.FusionEnabled,
		source.FusionWeight,
		source.SampleRate,
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
func (r *DeviceDataRepositoryImpl) GetDataSourceByID(ctx context.Context, id string) (*deviceDomain.DataSource, error) {
	query := `
		SELECT id, device_id, source_name, source_type, channel_id, field_mapping,
		       fusion_enabled, fusion_weight, sample_rate, enabled, metadata, created_at, updated_at
		FROM device_data_sources
		WHERE id = $1
	`

	var source deviceDomain.DataSource
	var sourceType string
	var mappingJSON, metadataJSON []byte

	err := r.store.QueryRow(query, id).Scan(
		&source.ID,
		&source.DeviceID,
		&source.SourceName,
		&sourceType,
		&source.ChannelID,
		&mappingJSON,
		&source.FusionEnabled,
		&source.FusionWeight,
		&source.SampleRate,
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

	source.SourceType = deviceDomain.SourceType(sourceType)

	if err := json.Unmarshal(mappingJSON, &source.FieldMapping); err != nil {
		r.store.Log().Warn("反序列化字段映射失败: %v", err)
		source.FieldMapping = make(map[string]string)
	}

	if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
		r.store.Log().Warn("反序列化元数据失败: %v", err)
		source.Metadata = make(map[string]interface{})
	}

	return &source, nil
}

// GetDataSourcesByDevice 获取设备的所有数据源
func (r *DeviceDataRepositoryImpl) GetDataSourcesByDevice(ctx context.Context, deviceID string) ([]*deviceDomain.DataSource, error) {
	query := `
		SELECT id, device_id, source_name, source_type, channel_id, field_mapping,
		       fusion_enabled, fusion_weight, sample_rate, enabled, metadata, created_at, updated_at
		FROM device_data_sources
		WHERE device_id = $1
		ORDER BY source_name
	`

	rows, err := r.store.Query(query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("查询数据源列表失败: %w", err)
	}
	defer rows.Close()

	sources := make([]*deviceDomain.DataSource, 0)
	for rows.Next() {
		var source deviceDomain.DataSource
		var sourceType string
		var mappingJSON, metadataJSON []byte

		err := rows.Scan(
			&source.ID,
			&source.DeviceID,
			&source.SourceName,
			&sourceType,
			&source.ChannelID,
			&mappingJSON,
			&source.FusionEnabled,
			&source.FusionWeight,
			&source.SampleRate,
			&source.Enabled,
			&metadataJSON,
			&source.CreatedAt,
			&source.UpdatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描数据源失败: %v", err)
			continue
		}

		source.SourceType = deviceDomain.SourceType(sourceType)

		if err := json.Unmarshal(mappingJSON, &source.FieldMapping); err != nil {
			r.store.Log().Warn("反序列化字段映射失败: %v", err)
			source.FieldMapping = make(map[string]string)
		}

		if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
			r.store.Log().Warn("反序列化元数据失败: %v", err)
			source.Metadata = make(map[string]interface{})
		}

		sources = append(sources, &source)
	}

	return sources, nil
}

// UpdateDataSource 更新数据源
func (r *DeviceDataRepositoryImpl) UpdateDataSource(ctx context.Context, source *deviceDomain.DataSource) error {
	mappingJSON, err := json.Marshal(source.FieldMapping)
	if err != nil {
		return fmt.Errorf("序列化字段映射失败: %w", err)
	}

	metadataJSON, err := json.Marshal(source.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE device_data_sources
		SET source_name = $1, source_type = $2, channel_id = $3, field_mapping = $4,
		    fusion_enabled = $5, fusion_weight = $6, sample_rate = $7, enabled = $8,
		    metadata = $9, updated_at = $10
		WHERE id = $11
	`

	tag, err := r.store.Exec(query,
		source.SourceName,
		string(source.SourceType),
		source.ChannelID,
		mappingJSON,
		source.FusionEnabled,
		source.FusionWeight,
		source.SampleRate,
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
func (r *DeviceDataRepositoryImpl) DeleteDataSource(ctx context.Context, id string) error {
	query := `DELETE FROM device_data_sources WHERE id = $1`

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
func (r *DeviceDataRepositoryImpl) CreateFusionConfig(ctx context.Context, config *deviceDomain.FusionConfig) error {
	fieldWeightsJSON, err := json.Marshal(config.FieldWeights)
	if err != nil {
		return fmt.Errorf("序列化字段权重失败: %w", err)
	}

	sourceWeightsJSON, err := json.Marshal(config.SourceWeights)
	if err != nil {
		return fmt.Errorf("序列化数据源权重失败: %w", err)
	}

	metadataJSON, err := json.Marshal(config.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO device_fusion_configs (id, device_id, fusion_strategy, time_window_ms,
		                                  min_sources, field_weights, source_weights,
		                                  enabled, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.store.Exec(query,
		config.ID,
		config.DeviceID,
		config.FusionStrategy,
		config.TimeWindowMs,
		config.MinSources,
		fieldWeightsJSON,
		sourceWeightsJSON,
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

// GetFusionConfigByDevice 获取设备的融合配置
func (r *DeviceDataRepositoryImpl) GetFusionConfigByDevice(ctx context.Context, deviceID string) (*deviceDomain.FusionConfig, error) {
	query := `
		SELECT id, device_id, fusion_strategy, time_window_ms, min_sources,
		       field_weights, source_weights, enabled, metadata, created_at, updated_at
		FROM device_fusion_configs
		WHERE device_id = $1
	`

	var config deviceDomain.FusionConfig
	var fieldWeightsJSON, sourceWeightsJSON, metadataJSON []byte

	err := r.store.QueryRow(query, deviceID).Scan(
		&config.ID,
		&config.DeviceID,
		&config.FusionStrategy,
		&config.TimeWindowMs,
		&config.MinSources,
		&fieldWeightsJSON,
		&sourceWeightsJSON,
		&metadataJSON,
		&config.Enabled,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("融合配置不存在: %s", deviceID)
		}
		return nil, fmt.Errorf("查询融合配置失败: %w", err)
	}

	if err := json.Unmarshal(fieldWeightsJSON, &config.FieldWeights); err != nil {
		r.store.Log().Warn("反序列化字段权重失败: %v", err)
		config.FieldWeights = make(map[string]float64)
	}

	if err := json.Unmarshal(sourceWeightsJSON, &config.SourceWeights); err != nil {
		r.store.Log().Warn("反序列化数据源权重失败: %v", err)
		config.SourceWeights = make(map[string]float64)
	}

	if err := json.Unmarshal(metadataJSON, &config.Metadata); err != nil {
		r.store.Log().Warn("反序列化元数据失败: %v", err)
		config.Metadata = make(map[string]interface{})
	}

	return &config, nil
}

// UpdateFusionConfig 更新融合配置
func (r *DeviceDataRepositoryImpl) UpdateFusionConfig(ctx context.Context, config *deviceDomain.FusionConfig) error {
	fieldWeightsJSON, err := json.Marshal(config.FieldWeights)
	if err != nil {
		return fmt.Errorf("序列化字段权重失败: %w", err)
	}

	sourceWeightsJSON, err := json.Marshal(config.SourceWeights)
	if err != nil {
		return fmt.Errorf("序列化数据源权重失败: %w", err)
	}

	metadataJSON, err := json.Marshal(config.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE device_fusion_configs
		SET fusion_strategy = $1, time_window_ms = $2, min_sources = $3,
		    field_weights = $4, source_weights = $5, enabled = $6, metadata = $7, updated_at = $8
		WHERE id = $9
	`

	tag, err := r.store.Exec(query,
		config.FusionStrategy,
		config.TimeWindowMs,
		config.MinSources,
		fieldWeightsJSON,
		sourceWeightsJSON,
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
func (r *DeviceDataRepositoryImpl) DeleteFusionConfig(ctx context.Context, deviceID string) error {
	query := `DELETE FROM device_fusion_configs WHERE device_id = $1`

	tag, err := r.store.Exec(query, deviceID)
	if err != nil {
		return fmt.Errorf("删除融合配置失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("融合配置不存在: %s", deviceID)
	}

	return nil
}

// SaveFusedData 保存融合数据
func (r *DeviceDataRepositoryImpl) SaveFusedData(ctx context.Context, data *deviceDomain.FusedData) error {
	fusedDataJSON, err := json.Marshal(data.FusedData)
	if err != nil {
		return fmt.Errorf("序列化融合数据失败: %w", err)
	}

	metadataJSON, err := json.Marshal(data.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO device_fused_data (id, device_id, timestamp, fused_data, source_count,
		                             fusion_strategy, quality_score, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.store.Exec(query,
		data.ID,
		data.DeviceID,
		data.Timestamp,
		fusedDataJSON,
		data.SourceCount,
		data.FusionStrategy,
		data.QualityScore,
		metadataJSON,
		data.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("保存融合数据失败: %w", err)
	}

	return nil
}

// GetFusedDataByDevice 获取设备的融合数据
func (r *DeviceDataRepositoryImpl) GetFusedDataByDevice(ctx context.Context, deviceID string, start, end time.Time, limit int) ([]*deviceDomain.FusedData, error) {
	query := `
		SELECT id, device_id, timestamp, fused_data, source_count, fusion_strategy,
		       quality_score, metadata, created_at
		FROM device_fused_data
		WHERE device_id = $1 AND timestamp >= $2 AND timestamp < $3
		ORDER BY timestamp DESC
		LIMIT $4
	`

	rows, err := r.store.Query(query, deviceID, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("查询融合数据失败: %w", err)
	}
	defer rows.Close()

	results := make([]*deviceDomain.FusedData, 0)
	for rows.Next() {
		var data deviceDomain.FusedData
		var fusedDataJSON, metadataJSON []byte

		err := rows.Scan(
			&data.ID,
			&data.DeviceID,
			&data.Timestamp,
			&fusedDataJSON,
			&data.SourceCount,
			&data.FusionStrategy,
			&data.QualityScore,
			&metadataJSON,
			&data.CreatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描融合数据失败: %v", err)
			continue
		}

		if err := json.Unmarshal(fusedDataJSON, &data.FusedData); err != nil {
			r.store.Log().Warn("反序列化融合数据失败: %v", err)
			data.FusedData = make(map[string]interface{})
		}

		if err := json.Unmarshal(metadataJSON, &data.Metadata); err != nil {
			r.store.Log().Warn("反序列化元数据失败: %v", err)
			data.Metadata = make(map[string]interface{})
		}

		results = append(results, &data)
	}

	return results, nil
}

// GetLatestFusedData 获取最新融合数据
func (r *DeviceDataRepositoryImpl) GetLatestFusedData(ctx context.Context, deviceID string) (*deviceDomain.FusedData, error) {
	query := `
		SELECT id, device_id, timestamp, fused_data, source_count, fusion_strategy,
		       quality_score, metadata, created_at
		FROM device_fused_data
		WHERE device_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var data deviceDomain.FusedData
	var fusedDataJSON, metadataJSON []byte

	err := r.store.QueryRow(query, deviceID).Scan(
		&data.ID,
		&data.DeviceID,
		&data.Timestamp,
		&fusedDataJSON,
		&data.SourceCount,
		&data.FusionStrategy,
		&data.QualityScore,
		&metadataJSON,
		&data.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("融合数据不存在: %s", deviceID)
		}
		return nil, fmt.Errorf("查询最新融合数据失败: %w", err)
	}

	if err := json.Unmarshal(fusedDataJSON, &data.FusedData); err != nil {
		r.store.Log().Warn("反序列化融合数据失败: %v", err)
		data.FusedData = make(map[string]interface{})
	}

	if err := json.Unmarshal(metadataJSON, &data.Metadata); err != nil {
		r.store.Log().Warn("反序列化元数据失败: %v", err)
		data.Metadata = make(map[string]interface{})
	}

	return &data, nil
}
