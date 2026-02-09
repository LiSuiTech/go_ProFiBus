package ml

import (
	"fmt"
	"os"
	"path/filepath"

	"go_ProFiBus/pkg/interfaces"
)

// ModelLoader 模型加载器
type ModelLoader struct {
	registry       *ModelRegistry
	modelDirectory string
}

// NewModelLoader 创建模型加载器
func NewModelLoader(registry *ModelRegistry, modelDirectory string) *ModelLoader {
	// 确保模型目录存在
	if err := os.MkdirAll(modelDirectory, 0755); err != nil {
		// 如果创建失败，使用默认目录
		modelDirectory = "models"
		_ = os.MkdirAll(modelDirectory, 0755)
	}

	return &ModelLoader{
		registry:       registry,
		modelDirectory: modelDirectory,
	}
}

// LoadModel 加载模型
func (l *ModelLoader) LoadModel(name, modelType, modelPath string) (interfaces.MLModel, error) {
	// 检查文件是否存在
	fullPath := filepath.Join(l.modelDirectory, modelPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("模型文件不存在: %s", fullPath)
	}

	// 根据模型类型创建对应的模型实例
	var model interfaces.MLModel
	switch modelType {
	case "linear_regression":
		model = NewLinearRegressionModel(name)
	case "neural_network":
		model = NewNeuralNetworkModel(name)
	case "svm":
		model = NewSVMModel(name)
	case "decision_tree":
		model = NewDecisionTreeModel(name)
	case "lstm":
		model = NewLSTMModel(name)
	case "custom":
		model = NewCustomModel(name)
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", modelType)
	}

	// 加载模型文件
	if err := model.Load(fullPath); err != nil {
		return nil, fmt.Errorf("加载模型文件失败: %w", err)
	}

	// 注册模型
	if err := l.registry.Register(name, model); err != nil {
		return nil, fmt.Errorf("注册模型失败: %w", err)
	}

	return model, nil
}

// UnloadModel 卸载模型
func (l *ModelLoader) UnloadModel(name string) error {
	return l.registry.Unregister(name)
}

// GetModelDirectory 获取模型目录
func (l *ModelLoader) GetModelDirectory() string {
	return l.modelDirectory
}
