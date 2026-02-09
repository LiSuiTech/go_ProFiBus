package interfaces

import (
	"context"
	predictionDomain "go_ProFiBus/internal/domain/prediction"
	"time"
)

// PredictionRepository 预测仓储接口
type PredictionRepository interface {
	// CreatePrediction 创建预测结果
	CreatePrediction(ctx context.Context, prediction *predictionDomain.Prediction) error

	// GetPredictionByID 根据ID获取预测结果
	GetPredictionByID(ctx context.Context, id string) (*predictionDomain.Prediction, error)

	// ListPredictions 列出预测结果
	ListPredictions(ctx context.Context, filters PredictionFilters) ([]*predictionDomain.Prediction, error)

	// GetPredictionsByDevice 获取设备的预测结果
	GetPredictionsByDevice(ctx context.Context, deviceID string, predictionType predictionDomain.PredictionType, limit int) ([]*predictionDomain.Prediction, error)

	// CreateModel 创建预测模型
	CreateModel(ctx context.Context, model *predictionDomain.PredictionModel) error

	// GetModelByID 根据ID获取预测模型
	GetModelByID(ctx context.Context, id string) (*predictionDomain.PredictionModel, error)

	// ListModels 列出预测模型
	ListModels(ctx context.Context, filters ModelFilters) ([]*predictionDomain.PredictionModel, error)

	// UpdateModel 更新预测模型
	UpdateModel(ctx context.Context, model *predictionDomain.PredictionModel) error

	// DeleteModel 删除预测模型
	DeleteModel(ctx context.Context, id string) error

	// GetDeployedModels 获取已部署的模型
	GetDeployedModels(ctx context.Context, modelType *predictionDomain.ModelType) ([]*predictionDomain.PredictionModel, error)
}

// PredictionFilters 预测过滤器
type PredictionFilters struct {
	ModelID        *string
	DeviceID       *string
	ChannelID      *string
	PredictionType *predictionDomain.PredictionType
	StartTime      *time.Time
	EndTime        *time.Time
	Limit          int
	Offset         int
}

// ModelFilters 模型过滤器
type ModelFilters struct {
	Type   *predictionDomain.ModelType
	Status *predictionDomain.ModelStatus
	Limit  int
	Offset int
}
