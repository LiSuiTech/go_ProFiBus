package ml

import (
	"context"
	"fmt"
	"sync"

	"go_ProFiBus/internal/infrastructure/ml"
	predictionDomain "go_ProFiBus/internal/domain/prediction"
	"go_ProFiBus/pkg/interfaces"
)

// ModelManager ML模型管理器
type ModelManager struct {
	registry      *ml.ModelRegistry
	loader        *ml.ModelLoader
	engine        interfaces.InferenceEngine
	predictionRepo interfaces.PredictionRepository
	mu            sync.RWMutex
}

// NewModelManager 创建模型管理器
func NewModelManager(
	registry *ml.ModelRegistry,
	loader *ml.ModelLoader,
	engine interfaces.InferenceEngine,
	predictionRepo interfaces.PredictionRepository,
) *ModelManager {
	return &ModelManager{
		registry:      registry,
		loader:        loader,
		engine:        engine,
		predictionRepo: predictionRepo,
	}
}

// LoadModelFromDB 从数据库加载模型
func (m *ModelManager) LoadModelFromDB(ctx context.Context, modelID string) error {
	// 从数据库获取模型信息
	model, err := m.predictionRepo.GetModelByID(ctx, modelID)
	if err != nil {
		return fmt.Errorf("获取模型信息失败: %w", err)
	}

	// 检查模型状态
	if model.Status != predictionDomain.ModelStatusDeployed {
		return fmt.Errorf("模型未部署，无法加载: %s", model.Status)
	}

	// 加载模型
	_, err = m.loader.LoadModel(modelID, string(model.Type), model.FilePath)
	if err != nil {
		return fmt.Errorf("加载模型失败: %w", err)
	}

	return nil
}

// UnloadModel 卸载模型
func (m *ModelManager) UnloadModel(modelID string) error {
	return m.loader.UnloadModel(modelID)
}

// Predict 执行预测
func (m *ModelManager) Predict(ctx context.Context, modelID string, input *interfaces.Tensor) (*interfaces.Tensor, error) {
	// 检查模型是否已加载
	if !m.registry.Exists(modelID) {
		// 尝试从数据库加载
		if err := m.LoadModelFromDB(ctx, modelID); err != nil {
			return nil, fmt.Errorf("模型未加载且加载失败: %w", err)
		}
	}

	// 执行预测
	output, err := m.engine.Predict(modelID, input)
	if err != nil {
		return nil, fmt.Errorf("预测失败: %w", err)
	}

	return output, nil
}

// ListLoadedModels 列出已加载的模型
func (m *ModelManager) ListLoadedModels() []string {
	return m.registry.List()
}

// IsModelLoaded 检查模型是否已加载
func (m *ModelManager) IsModelLoaded(modelID string) bool {
	return m.registry.Exists(modelID)
}
