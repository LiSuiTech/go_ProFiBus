package ml

import (
	"fmt"

	"go_ProFiBus/pkg/interfaces"
)

// InferenceEngineImpl 推理引擎实现
type InferenceEngineImpl struct {
	registry *ModelRegistry
}

// NewInferenceEngine 创建推理引擎
func NewInferenceEngine(registry *ModelRegistry) interfaces.InferenceEngine {
	return &InferenceEngineImpl{
		registry: registry,
	}
}

// RegisterModel 注册模型
func (e *InferenceEngineImpl) RegisterModel(name string, model interfaces.MLModel) error {
	return e.registry.Register(name, model)
}

// UnregisterModel 注销模型
func (e *InferenceEngineImpl) UnregisterModel(name string) error {
	return e.registry.Unregister(name)
}

// Predict 执行推理
func (e *InferenceEngineImpl) Predict(modelName string, input *interfaces.Tensor) (*interfaces.Tensor, error) {
	model, err := e.registry.Get(modelName)
	if err != nil {
		return nil, fmt.Errorf("获取模型失败: %w", err)
	}

	// 验证输入形状
	expectedShape := model.GetInputShape()
	if len(expectedShape) != len(input.Shape) {
		return nil, fmt.Errorf("输入形状不匹配: 期望 %v, 实际 %v", expectedShape, input.Shape)
	}

	for i, dim := range expectedShape {
		if dim != -1 && dim != input.Shape[i] {
			return nil, fmt.Errorf("输入维度 %d 不匹配: 期望 %d, 实际 %d", i, dim, input.Shape[i])
		}
	}

	// 执行预测
	output, err := model.Predict(input)
	if err != nil {
		return nil, fmt.Errorf("预测失败: %w", err)
	}

	return output, nil
}

// GetModel 获取模型
func (e *InferenceEngineImpl) GetModel(name string) (interfaces.MLModel, error) {
	return e.registry.Get(name)
}

// ListModels 列出所有模型
func (e *InferenceEngineImpl) ListModels() []string {
	return e.registry.List()
}
