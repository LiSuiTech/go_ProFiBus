package interfaces

import (
	"context"
	"time"

	trainingDomain "go_ProFiBus/internal/domain/training"
)

// TrainingRepository 训练任务仓储接口
type TrainingRepository interface {
	// 训练任务管理
	CreateTask(ctx context.Context, task *trainingDomain.TrainingTask) error
	GetTaskByID(ctx context.Context, taskID string) (*trainingDomain.TrainingTask, error)
	ListTasks(ctx context.Context, filters TrainingTaskFilters) ([]*trainingDomain.TrainingTask, error)
	UpdateTask(ctx context.Context, task *trainingDomain.TrainingTask) error
	DeleteTask(ctx context.Context, taskID string) error

	// 训练样本管理
	CreateSample(ctx context.Context, sample *trainingDomain.TrainingSample) error
	CreateSamples(ctx context.Context, samples []*trainingDomain.TrainingSample) error
	GetSamplesByTaskID(ctx context.Context, taskID string, limit, offset int) ([]*trainingDomain.TrainingSample, error)
	GetSampleCountByTaskID(ctx context.Context, taskID string) (int, error)
	DeleteSamplesByTaskID(ctx context.Context, taskID string) error

	// 训练历史管理
	CreateHistory(ctx context.Context, history *trainingDomain.TrainingHistory) error
	CreateHistories(ctx context.Context, histories []*trainingDomain.TrainingHistory) error
	GetHistoryByTaskID(ctx context.Context, taskID string, limit, offset int) ([]*trainingDomain.TrainingHistory, error)
	GetLatestHistoryByTaskID(ctx context.Context, taskID string) (*trainingDomain.TrainingHistory, error)
	DeleteHistoryByTaskID(ctx context.Context, taskID string) error
}

// TrainingTaskFilters 训练任务过滤器
type TrainingTaskFilters struct {
	ModelID        *string
	Status         *trainingDomain.TrainingStatus
	TrainingType   *trainingDomain.TrainingType
	DataSourceType *trainingDomain.DataSourceType
	CreatedBy      *string
	StartTime      *time.Time
	EndTime        *time.Time
	Limit          int
	Offset         int
}
