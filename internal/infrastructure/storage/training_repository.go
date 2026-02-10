package storage

import (
	"context"
	"encoding/json"
	"fmt"
	trainingDomain "go_ProFiBus/internal/domain/training"
	"go_ProFiBus/pkg/interfaces"

	"github.com/jackc/pgx/v5"
)

// TrainingRepositoryImpl 训练任务仓储实现
type TrainingRepositoryImpl struct {
	store *PostgresStore
}

// NewTrainingRepository 创建训练任务仓储
func NewTrainingRepository(store *PostgresStore) interfaces.TrainingRepository {
	return &TrainingRepositoryImpl{store: store}
}

// CreateTask 创建训练任务
func (r *TrainingRepositoryImpl) CreateTask(ctx context.Context, task *trainingDomain.TrainingTask) error {
	hyperparamsJSON, _ := json.Marshal(task.Hyperparameters)
	configJSON, _ := json.Marshal(task.TrainingConfig)
	metricsJSON, _ := json.Marshal(task.Metrics)
	metadataJSON, _ := json.Marshal(task.Metadata)

	query := `
		INSERT INTO training_tasks (id, model_id, name, description, status, training_type,
		                          data_source_type, data_source_ids, data_fields,
		                          start_time, end_time, progress, epochs, batch_size,
		                          learning_rate, validation_split, hyperparameters,
		                          training_config, metrics, error_message,
		                          created_at, updated_at, created_by, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
	`

	_, err := r.store.Exec(query,
		task.ID, task.ModelID, task.Name, task.Description, string(task.Status),
		string(task.TrainingType), string(task.DataSourceType), task.DataSourceIDs, task.DataFields,
		task.StartTime, task.EndTime, task.Progress, task.Epochs, task.BatchSize,
		task.LearningRate, task.ValidationSplit, hyperparamsJSON, configJSON, metricsJSON,
		task.ErrorMessage, task.CreatedAt, task.UpdatedAt, task.CreatedBy, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("创建训练任务失败: %w", err)
	}

	return nil
}

// GetTaskByID 根据ID获取训练任务
func (r *TrainingRepositoryImpl) GetTaskByID(ctx context.Context, taskID string) (*trainingDomain.TrainingTask, error) {
	query := `
		SELECT id, model_id, name, description, status, training_type,
		       data_source_type, data_source_ids, data_fields,
		       start_time, end_time, progress, epochs, batch_size,
		       learning_rate, validation_split, hyperparameters,
		       training_config, metrics, error_message,
		       created_at, updated_at, created_by, metadata
		FROM training_tasks
		WHERE id = $1
	`

	var task trainingDomain.TrainingTask
	var statusStr, trainingTypeStr, dataSourceTypeStr string
	var hyperparamsJSON, configJSON, metricsJSON, metadataJSON []byte

	err := r.store.QueryRow(query, taskID).Scan(
		&task.ID, &task.ModelID, &task.Name, &task.Description, &statusStr,
		&trainingTypeStr, &dataSourceTypeStr, &task.DataSourceIDs, &task.DataFields,
		&task.StartTime, &task.EndTime, &task.Progress, &task.Epochs, &task.BatchSize,
		&task.LearningRate, &task.ValidationSplit, &hyperparamsJSON, &configJSON, &metricsJSON,
		&task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt, &task.CreatedBy, &metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("训练任务不存在: %s", taskID)
		}
		return nil, fmt.Errorf("查询训练任务失败: %w", err)
	}

	task.Status = trainingDomain.TrainingStatus(statusStr)
	task.TrainingType = trainingDomain.TrainingType(trainingTypeStr)
	task.DataSourceType = trainingDomain.DataSourceType(dataSourceTypeStr)

	json.Unmarshal(hyperparamsJSON, &task.Hyperparameters)
	json.Unmarshal(configJSON, &task.TrainingConfig)
	json.Unmarshal(metricsJSON, &task.Metrics)
	json.Unmarshal(metadataJSON, &task.Metadata)

	return &task, nil
}

// ListTasks 列出训练任务
func (r *TrainingRepositoryImpl) ListTasks(ctx context.Context, filters interfaces.TrainingTaskFilters) ([]*trainingDomain.TrainingTask, error) {
	query := `SELECT id, model_id, name, description, status, training_type,
	                data_source_type, data_source_ids, data_fields,
	                start_time, end_time, progress, epochs, batch_size,
	                learning_rate, validation_split, hyperparameters,
	                training_config, metrics, error_message,
	                created_at, updated_at, created_by, metadata
	         FROM training_tasks WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filters.ModelID != nil {
		query += fmt.Sprintf(" AND model_id = $%d", argIndex)
		args = append(args, *filters.ModelID)
		argIndex++
	}
	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
		argIndex++
	}
	if filters.TrainingType != nil {
		query += fmt.Sprintf(" AND training_type = $%d", argIndex)
		args = append(args, string(*filters.TrainingType))
		argIndex++
	}
	if filters.CreatedBy != nil {
		query += fmt.Sprintf(" AND created_by = $%d", argIndex)
		args = append(args, *filters.CreatedBy)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
		if filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filters.Offset)
		}
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询训练任务失败: %w", err)
	}
	defer rows.Close()

	var tasks []*trainingDomain.TrainingTask
	for rows.Next() {
		var task trainingDomain.TrainingTask
		var statusStr, trainingTypeStr, dataSourceTypeStr string
		var hyperparamsJSON, configJSON, metricsJSON, metadataJSON []byte

		err := rows.Scan(
			&task.ID, &task.ModelID, &task.Name, &task.Description, &statusStr,
			&trainingTypeStr, &dataSourceTypeStr, &task.DataSourceIDs, &task.DataFields,
			&task.StartTime, &task.EndTime, &task.Progress, &task.Epochs, &task.BatchSize,
			&task.LearningRate, &task.ValidationSplit, &hyperparamsJSON, &configJSON, &metricsJSON,
			&task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt, &task.CreatedBy, &metadataJSON,
		)
		if err != nil {
			continue
		}

		task.Status = trainingDomain.TrainingStatus(statusStr)
		task.TrainingType = trainingDomain.TrainingType(trainingTypeStr)
		task.DataSourceType = trainingDomain.DataSourceType(dataSourceTypeStr)

		json.Unmarshal(hyperparamsJSON, &task.Hyperparameters)
		json.Unmarshal(configJSON, &task.TrainingConfig)
		json.Unmarshal(metricsJSON, &task.Metrics)
		json.Unmarshal(metadataJSON, &task.Metadata)

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// UpdateTask 更新训练任务
func (r *TrainingRepositoryImpl) UpdateTask(ctx context.Context, task *trainingDomain.TrainingTask) error {
	hyperparamsJSON, _ := json.Marshal(task.Hyperparameters)
	configJSON, _ := json.Marshal(task.TrainingConfig)
	metricsJSON, _ := json.Marshal(task.Metrics)
	metadataJSON, _ := json.Marshal(task.Metadata)

	query := `
		UPDATE training_tasks
		SET name = $2, description = $3, status = $4, progress = $5,
		    start_time = $6, end_time = $7, metrics = $8,
		    error_message = $9, updated_at = $10, hyperparameters = $11,
		    training_config = $12, metadata = $13
		WHERE id = $1
	`

	_, err := r.store.Exec(query,
		task.ID, task.Name, task.Description, string(task.Status), task.Progress,
		task.StartTime, task.EndTime, metricsJSON, task.ErrorMessage,
		task.UpdatedAt, hyperparamsJSON, configJSON, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("更新训练任务失败: %w", err)
	}

	return nil
}

// DeleteTask 删除训练任务
func (r *TrainingRepositoryImpl) DeleteTask(ctx context.Context, taskID string) error {
	query := `DELETE FROM training_tasks WHERE id = $1`
	_, err := r.store.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("删除训练任务失败: %w", err)
	}
	return nil
}

// CreateSample 创建训练样本
func (r *TrainingRepositoryImpl) CreateSample(ctx context.Context, sample *trainingDomain.TrainingSample) error {
	inputJSON, _ := json.Marshal(sample.InputData)
	outputJSON, _ := json.Marshal(sample.OutputData)

	query := `
		INSERT INTO training_samples (id, task_id, sample_index, input_data, output_data, label, timestamp, quality, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.store.Exec(query,
		sample.ID, sample.TaskID, sample.SampleIndex, inputJSON, outputJSON,
		sample.Label, sample.Timestamp, sample.Quality, sample.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建训练样本失败: %w", err)
	}

	return nil
}

// CreateSamples 批量创建训练样本
func (r *TrainingRepositoryImpl) CreateSamples(ctx context.Context, samples []*trainingDomain.TrainingSample) error {
	for _, sample := range samples {
		if err := r.CreateSample(ctx, sample); err != nil {
			return err
		}
	}
	return nil
}

// GetSamplesByTaskID 获取任务的训练样本
func (r *TrainingRepositoryImpl) GetSamplesByTaskID(ctx context.Context, taskID string, limit, offset int) ([]*trainingDomain.TrainingSample, error) {
	query := `
		SELECT id, task_id, sample_index, input_data, output_data, label, timestamp, quality, created_at
		FROM training_samples
		WHERE task_id = $1
		ORDER BY sample_index
		LIMIT $2 OFFSET $3
	`

	rows, err := r.store.Query(query, taskID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询训练样本失败: %w", err)
	}
	defer rows.Close()

	var samples []*trainingDomain.TrainingSample
	for rows.Next() {
		var sample trainingDomain.TrainingSample
		var inputJSON, outputJSON []byte

		err := rows.Scan(
			&sample.ID, &sample.TaskID, &sample.SampleIndex, &inputJSON, &outputJSON,
			&sample.Label, &sample.Timestamp, &sample.Quality, &sample.CreatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(inputJSON, &sample.InputData)
		json.Unmarshal(outputJSON, &sample.OutputData)

		samples = append(samples, &sample)
	}

	return samples, nil
}

// GetSampleCountByTaskID 获取任务的样本数量
func (r *TrainingRepositoryImpl) GetSampleCountByTaskID(ctx context.Context, taskID string) (int, error) {
	query := `SELECT COUNT(*) FROM training_samples WHERE task_id = $1`
	var count int
	err := r.store.QueryRow(query, taskID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("查询样本数量失败: %w", err)
	}
	return count, nil
}

// DeleteSamplesByTaskID 删除任务的所有样本
func (r *TrainingRepositoryImpl) DeleteSamplesByTaskID(ctx context.Context, taskID string) error {
	query := `DELETE FROM training_samples WHERE task_id = $1`
	_, err := r.store.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("删除训练样本失败: %w", err)
	}
	return nil
}

// CreateHistory 创建训练历史记录
func (r *TrainingRepositoryImpl) CreateHistory(ctx context.Context, history *trainingDomain.TrainingHistory) error {
	metricsJSON, _ := json.Marshal(history.Metrics)

	query := `
		INSERT INTO training_history (id, task_id, epoch, step, loss, accuracy,
		                           validation_loss, validation_accuracy, learning_rate,
		                           metrics, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.store.Exec(query,
		history.ID, history.TaskID, history.Epoch, history.Step,
		history.Loss, history.Accuracy, history.ValidationLoss, history.ValidationAccuracy,
		history.LearningRate, metricsJSON, history.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("创建训练历史失败: %w", err)
	}

	return nil
}

// CreateHistories 批量创建训练历史记录
func (r *TrainingRepositoryImpl) CreateHistories(ctx context.Context, histories []*trainingDomain.TrainingHistory) error {
	for _, history := range histories {
		if err := r.CreateHistory(ctx, history); err != nil {
			return err
		}
	}
	return nil
}

// GetHistoryByTaskID 获取任务的训练历史
func (r *TrainingRepositoryImpl) GetHistoryByTaskID(ctx context.Context, taskID string, limit, offset int) ([]*trainingDomain.TrainingHistory, error) {
	query := `
		SELECT id, task_id, epoch, step, loss, accuracy,
		       validation_loss, validation_accuracy, learning_rate, metrics, timestamp
		FROM training_history
		WHERE task_id = $1
		ORDER BY epoch, step
		LIMIT $2 OFFSET $3
	`

	rows, err := r.store.Query(query, taskID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询训练历史失败: %w", err)
	}
	defer rows.Close()

	var histories []*trainingDomain.TrainingHistory
	for rows.Next() {
		var history trainingDomain.TrainingHistory
		var metricsJSON []byte

		err := rows.Scan(
			&history.ID, &history.TaskID, &history.Epoch, &history.Step,
			&history.Loss, &history.Accuracy, &history.ValidationLoss, &history.ValidationAccuracy,
			&history.LearningRate, &metricsJSON, &history.Timestamp,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(metricsJSON, &history.Metrics)
		histories = append(histories, &history)
	}

	return histories, nil
}

// GetLatestHistoryByTaskID 获取任务的最新训练历史
func (r *TrainingRepositoryImpl) GetLatestHistoryByTaskID(ctx context.Context, taskID string) (*trainingDomain.TrainingHistory, error) {
	query := `
		SELECT id, task_id, epoch, step, loss, accuracy,
		       validation_loss, validation_accuracy, learning_rate, metrics, timestamp
		FROM training_history
		WHERE task_id = $1
		ORDER BY epoch DESC, step DESC
		LIMIT 1
	`

	var history trainingDomain.TrainingHistory
	var metricsJSON []byte

	err := r.store.QueryRow(query, taskID).Scan(
		&history.ID, &history.TaskID, &history.Epoch, &history.Step,
		&history.Loss, &history.Accuracy, &history.ValidationLoss, &history.ValidationAccuracy,
		&history.LearningRate, &metricsJSON, &history.Timestamp,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询训练历史失败: %w", err)
	}

	json.Unmarshal(metricsJSON, &history.Metrics)
	return &history, nil
}

// DeleteHistoryByTaskID 删除任务的所有训练历史
func (r *TrainingRepositoryImpl) DeleteHistoryByTaskID(ctx context.Context, taskID string) error {
	query := `DELETE FROM training_history WHERE task_id = $1`
	_, err := r.store.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("删除训练历史失败: %w", err)
	}
	return nil
}
