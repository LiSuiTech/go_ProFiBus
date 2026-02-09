package training

import (
	"context"
	"fmt"
	"time"

	trainingDomain "go_ProFiBus/internal/domain/training"
	"go_ProFiBus/pkg/interfaces"
)

// Service ML 模型訓練服務
type Service struct {
	trainingRepo   interfaces.TrainingRepository
	predictionRepo interfaces.PredictionRepository
}

// NewService 創建訓練服務
func NewService(
	trainingRepo interfaces.TrainingRepository,
	predictionRepo interfaces.PredictionRepository,
) *Service {
	return &Service{
		trainingRepo:   trainingRepo,
		predictionRepo: predictionRepo,
	}
}

// CreateTask 創建訓練任務
func (s *Service) CreateTask(ctx context.Context, task *trainingDomain.TrainingTask) error {
	// 檢查模型是否存在
	if _, err := s.predictionRepo.GetModelByID(ctx, task.ModelID); err != nil {
		return fmt.Errorf("模型不存在: %w", err)
	}

	// 初始化時間
	if task.CreatedAt.IsZero() {
		now := time.Now()
		task.CreatedAt = now
		task.UpdatedAt = now
	}

	if task.Hyperparameters == nil {
		task.Hyperparameters = make(map[string]interface{})
	}
	if task.TrainingConfig == nil {
		task.TrainingConfig = make(map[string]interface{})
	}
	if task.Metrics == nil {
		task.Metrics = make(map[string]interface{})
	}
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}

	return s.trainingRepo.CreateTask(ctx, task)
}

// GetTask 獲取訓練任務
func (s *Service) GetTask(ctx context.Context, taskID string) (*trainingDomain.TrainingTask, error) {
	return s.trainingRepo.GetTaskByID(ctx, taskID)
}

// ListTasks 列出訓練任務
func (s *Service) ListTasks(ctx context.Context, filters interfaces.TrainingTaskFilters) ([]*trainingDomain.TrainingTask, error) {
	return s.trainingRepo.ListTasks(ctx, filters)
}

// StartTask 啟動訓練任務（簡化版，模擬訓練流程）
func (s *Service) StartTask(ctx context.Context, taskID string) (*trainingDomain.TrainingTask, error) {
	task, err := s.trainingRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 只能從 pending/failed 狀態啟動
	if task.Status != trainingDomain.TrainingStatusPending &&
		task.Status != trainingDomain.TrainingStatusFailed {
		return nil, fmt.Errorf("當前狀態無法啟動訓練: %s", task.Status)
	}

	task.Start()

	// 模擬訓練：直接完成，並填充一些指標
	// 真實場景下，這裡應該啟動長時間運行的訓練流程
	metrics := map[string]interface{}{
		"loss":           0.123,
		"accuracy":       0.95,
		"val_loss":       0.145,
		"val_accuracy":   0.93,
		"sample_count":   1000,
		"training_time":  "5s",
		"training_device": "cpu",
	}
	task.SetMetrics(metrics)
	task.SetProgress(1.0)
	task.Complete()

	if err := s.trainingRepo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}

	// 創建一條簡單的訓練歷史記錄
	history := trainingDomain.NewTrainingHistory(task.ID, 1, 1)
	history.SetLoss(0.123)
	history.SetAccuracy(0.95)
	history.SetValidationLoss(0.145)
	history.SetValidationAccuracy(0.93)
	lr := task.LearningRate
	history.SetLearningRate(lr)
	history.AddMetric("note", "mock training run")

	if err := s.trainingRepo.CreateHistory(ctx, history); err != nil {
		return nil, fmt.Errorf("保存訓練歷史失敗: %w", err)
	}

	// 更新模型的訓練相關資訊（樣本數/準確度/狀態）
	model, err := s.predictionRepo.GetModelByID(ctx, task.ModelID)
	if err == nil {
		// 這裡假設使用 metrics 中的樣本數與準確度
		if sampleCount, ok := metrics["sample_count"].(int); ok {
			model.TrainingSamples = sampleCount
		}
		if acc, ok := metrics["accuracy"].(float64); ok {
			model.SetAccuracy(acc)
		}
		// 將模型狀態更新為 training 完成（保持原部署狀態不變）
		if err := s.predictionRepo.UpdateModel(ctx, model); err != nil {
			return nil, fmt.Errorf("更新模型資訊失敗: %w", err)
		}
	}

	return task, nil
}

// CancelTask 取消訓練任務
func (s *Service) CancelTask(ctx context.Context, taskID string) (*trainingDomain.TrainingTask, error) {
	task, err := s.trainingRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.Status != trainingDomain.TrainingStatusRunning &&
		task.Status != trainingDomain.TrainingStatusPending {
		return nil, fmt.Errorf("當前狀態無法取消訓練: %s", task.Status)
	}

	task.Cancel()

	if err := s.trainingRepo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetHistory 獲取訓練歷史
func (s *Service) GetHistory(ctx context.Context, taskID string, limit, offset int) ([]*trainingDomain.TrainingHistory, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.trainingRepo.GetHistoryByTaskID(ctx, taskID, limit, offset)
}

