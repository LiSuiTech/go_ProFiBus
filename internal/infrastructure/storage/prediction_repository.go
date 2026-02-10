package storage

import (
	"context"
	"encoding/json"
	"fmt"
	predictionDomain "go_ProFiBus/internal/domain/prediction"
	"go_ProFiBus/pkg/interfaces"
	"time"

	"github.com/jackc/pgx/v5"
)

// PredictionRepositoryImpl 预测仓储实现
type PredictionRepositoryImpl struct {
	store *PostgresStore
}

// NewPredictionRepository 创建预测仓储
func NewPredictionRepository(store *PostgresStore) interfaces.PredictionRepository {
	return &PredictionRepositoryImpl{store: store}
}

// CreatePrediction 创建预测结果
func (r *PredictionRepositoryImpl) CreatePrediction(ctx context.Context, prediction *predictionDomain.Prediction) error {
	metadataJSON, err := json.Marshal(prediction.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO predictions (id, model_id, device_id, channel_id, prediction_type, field_name,
		                        predicted_value, confidence, actual_value, error_rate,
		                        time_range_start, time_range_end, created_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = r.store.Exec(query,
		prediction.ID,
		prediction.ModelID,
		prediction.DeviceID,
		prediction.ChannelID,
		string(prediction.PredictionType),
		prediction.FieldName,
		prediction.PredictedValue,
		prediction.Confidence,
		prediction.ActualValue,
		prediction.ErrorRate,
		prediction.TimeRangeStart,
		prediction.TimeRangeEnd,
		prediction.CreatedAt,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("创建预测结果失败: %w", err)
	}

	return nil
}

// GetPredictionByID 根据ID获取预测结果
func (r *PredictionRepositoryImpl) GetPredictionByID(ctx context.Context, id string) (*predictionDomain.Prediction, error) {
	query := `
		SELECT id, model_id, device_id, channel_id, prediction_type, field_name,
		       predicted_value, confidence, actual_value, error_rate,
		       time_range_start, time_range_end, created_at, metadata
		FROM predictions
		WHERE id = $1
	`

	var prediction predictionDomain.Prediction
	var metadataJSON []byte
	var predictionType string

	err := r.store.QueryRow(query, id).Scan(
		&prediction.ID,
		&prediction.ModelID,
		&prediction.DeviceID,
		&prediction.ChannelID,
		&predictionType,
		&prediction.FieldName,
		&prediction.PredictedValue,
		&prediction.Confidence,
		&prediction.ActualValue,
		&prediction.ErrorRate,
		&prediction.TimeRangeStart,
		&prediction.TimeRangeEnd,
		&prediction.CreatedAt,
		&metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("预测结果不存在: %s", id)
		}
		return nil, fmt.Errorf("查询预测结果失败: %w", err)
	}

	prediction.PredictionType = predictionDomain.PredictionType(predictionType)

	if err := json.Unmarshal(metadataJSON, &prediction.Metadata); err != nil {
		r.store.Log().Warn("反序列化元数据失败: %v", err)
		prediction.Metadata = make(map[string]interface{})
	}

	return &prediction, nil
}

// ListPredictions 列出预测结果
func (r *PredictionRepositoryImpl) ListPredictions(ctx context.Context, filters interfaces.PredictionFilters) ([]*predictionDomain.Prediction, error) {
	query := `
		SELECT id, model_id, device_id, channel_id, prediction_type, field_name,
		       predicted_value, confidence, actual_value, error_rate,
		       time_range_start, time_range_end, created_at, metadata
		FROM predictions
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.ModelID != nil {
		query += fmt.Sprintf(" AND model_id = $%d", argIndex)
		args = append(args, *filters.ModelID)
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

	if filters.PredictionType != nil {
		query += fmt.Sprintf(" AND prediction_type = $%d", argIndex)
		args = append(args, string(*filters.PredictionType))
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND time_range_start >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND time_range_end < $%d", argIndex)
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
		return nil, fmt.Errorf("查询预测结果列表失败: %w", err)
	}
	defer rows.Close()

	predictions := make([]*predictionDomain.Prediction, 0)

	for rows.Next() {
		var prediction predictionDomain.Prediction
		var metadataJSON []byte
		var predictionType string

		err := rows.Scan(
			&prediction.ID,
			&prediction.ModelID,
			&prediction.DeviceID,
			&prediction.ChannelID,
			&predictionType,
			&prediction.FieldName,
			&prediction.PredictedValue,
			&prediction.Confidence,
			&prediction.ActualValue,
			&prediction.ErrorRate,
			&prediction.TimeRangeStart,
			&prediction.TimeRangeEnd,
			&prediction.CreatedAt,
			&metadataJSON,
		)
		if err != nil {
			r.store.Log().Warn("扫描预测结果失败: %v", err)
			continue
		}

		prediction.PredictionType = predictionDomain.PredictionType(predictionType)

		if err := json.Unmarshal(metadataJSON, &prediction.Metadata); err != nil {
			r.store.Log().Warn("反序列化元数据失败: %v", err)
			prediction.Metadata = make(map[string]interface{})
		}

		predictions = append(predictions, &prediction)
	}

	return predictions, nil
}

// GetPredictionsByDevice 获取设备的预测结果
func (r *PredictionRepositoryImpl) GetPredictionsByDevice(ctx context.Context, deviceID string, predictionType predictionDomain.PredictionType, limit int) ([]*predictionDomain.Prediction, error) {
	filters := interfaces.PredictionFilters{
		DeviceID:       &deviceID,
		PredictionType: &predictionType,
		Limit:          limit,
	}
	return r.ListPredictions(ctx, filters)
}

// CreateModel 创建预测模型
func (r *PredictionRepositoryImpl) CreateModel(ctx context.Context, model *predictionDomain.PredictionModel) error {
	metadataJSON, err := json.Marshal(model.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO prediction_models (id, name, description, type, version, file_path, status,
		                             accuracy, training_samples, created_at, updated_at, deployed_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.store.Exec(query,
		model.ID,
		model.Name,
		model.Description,
		string(model.Type),
		model.Version,
		model.FilePath,
		string(model.Status),
		model.Accuracy,
		model.TrainingSamples,
		model.CreatedAt,
		model.UpdatedAt,
		model.DeployedAt,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("创建预测模型失败: %w", err)
	}

	return nil
}

// GetModelByID 根据ID获取预测模型
func (r *PredictionRepositoryImpl) GetModelByID(ctx context.Context, id string) (*predictionDomain.PredictionModel, error) {
	query := `
		SELECT id, name, description, type, version, file_path, status,
		       accuracy, training_samples, created_at, updated_at, deployed_at, metadata
		FROM prediction_models
		WHERE id = $1
	`

	var model predictionDomain.PredictionModel
	var metadataJSON []byte
	var modelType, status string

	err := r.store.QueryRow(query, id).Scan(
		&model.ID,
		&model.Name,
		&model.Description,
		&modelType,
		&model.Version,
		&model.FilePath,
		&status,
		&model.Accuracy,
		&model.TrainingSamples,
		&model.CreatedAt,
		&model.UpdatedAt,
		&model.DeployedAt,
		&metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("预测模型不存在: %s", id)
		}
		return nil, fmt.Errorf("查询预测模型失败: %w", err)
	}

	model.Type = predictionDomain.ModelType(modelType)
	model.Status = predictionDomain.ModelStatus(status)

	if err := json.Unmarshal(metadataJSON, &model.Metadata); err != nil {
		r.store.Log().Warn("反序列化元数据失败: %v", err)
		model.Metadata = make(map[string]interface{})
	}

	return &model, nil
}

// ListModels 列出预测模型
func (r *PredictionRepositoryImpl) ListModels(ctx context.Context, filters interfaces.ModelFilters) ([]*predictionDomain.PredictionModel, error) {
	query := `
		SELECT id, name, description, type, version, file_path, status,
		       accuracy, training_samples, created_at, updated_at, deployed_at, metadata
		FROM prediction_models
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, string(*filters.Type))
		argIndex++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
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
		return nil, fmt.Errorf("查询预测模型列表失败: %w", err)
	}
	defer rows.Close()

	models := make([]*predictionDomain.PredictionModel, 0)

	for rows.Next() {
		var model predictionDomain.PredictionModel
		var metadataJSON []byte
		var modelType, status string

		err := rows.Scan(
			&model.ID,
			&model.Name,
			&model.Description,
			&modelType,
			&model.Version,
			&model.FilePath,
			&status,
			&model.Accuracy,
			&model.TrainingSamples,
			&model.CreatedAt,
			&model.UpdatedAt,
			&model.DeployedAt,
			&metadataJSON,
		)
		if err != nil {
			r.store.Log().Warn("扫描预测模型失败: %v", err)
			continue
		}

		model.Type = predictionDomain.ModelType(modelType)
		model.Status = predictionDomain.ModelStatus(status)

		if err := json.Unmarshal(metadataJSON, &model.Metadata); err != nil {
			r.store.Log().Warn("反序列化元数据失败: %v", err)
			model.Metadata = make(map[string]interface{})
		}

		models = append(models, &model)
	}

	return models, nil
}

// UpdateModel 更新预测模型
func (r *PredictionRepositoryImpl) UpdateModel(ctx context.Context, model *predictionDomain.PredictionModel) error {
	metadataJSON, err := json.Marshal(model.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE prediction_models
		SET name = $1, description = $2, type = $3, version = $4, file_path = $5, status = $6,
		    accuracy = $7, training_samples = $8, updated_at = $9, deployed_at = $10, metadata = $11
		WHERE id = $12
	`

	tag, err := r.store.Exec(query,
		model.Name,
		model.Description,
		string(model.Type),
		model.Version,
		model.FilePath,
		string(model.Status),
		model.Accuracy,
		model.TrainingSamples,
		time.Now(),
		model.DeployedAt,
		metadataJSON,
		model.ID,
	)

	if err != nil {
		return fmt.Errorf("更新预测模型失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("预测模型不存在: %s", model.ID)
	}

	return nil
}

// DeleteModel 删除预测模型
func (r *PredictionRepositoryImpl) DeleteModel(ctx context.Context, id string) error {
	query := `DELETE FROM prediction_models WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除预测模型失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("预测模型不存在: %s", id)
	}

	return nil
}

// GetDeployedModels 获取已部署的模型
func (r *PredictionRepositoryImpl) GetDeployedModels(ctx context.Context, modelType *predictionDomain.ModelType) ([]*predictionDomain.PredictionModel, error) {
	filters := interfaces.ModelFilters{
		Status: func() *predictionDomain.ModelStatus {
			s := predictionDomain.ModelStatusDeployed
			return &s
		}(),
		Limit: 100,
	}
	if modelType != nil {
		filters.Type = modelType
	}
	return r.ListModels(ctx, filters)
}
