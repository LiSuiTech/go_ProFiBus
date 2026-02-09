package ml

import (
	"fmt"
	"sync"

	"go_ProFiBus/pkg/interfaces"
)

// ModelRegistry ML模型注册表
type ModelRegistry struct {
	models map[string]interfaces.MLModel
	mu     sync.RWMutex
}

// NewModelRegistry 创建模型注册表
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: make(map[string]interfaces.MLModel),
	}
}

// Register 注册模型
func (r *ModelRegistry) Register(name string, model interfaces.MLModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[name]; exists {
		return fmt.Errorf("模型 %s 已存在", name)
	}

	r.models[name] = model
	return nil
}

// Unregister 注销模型
func (r *ModelRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[name]; !exists {
		return fmt.Errorf("模型 %s 不存在", name)
	}

	delete(r.models, name)
	return nil
}

// Get 获取模型
func (r *ModelRegistry) Get(name string) (interfaces.MLModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, exists := r.models[name]
	if !exists {
		return nil, fmt.Errorf("模型 %s 不存在", name)
	}

	return model, nil
}

// List 列出所有模型名称
func (r *ModelRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.models))
	for name := range r.models {
		names = append(names, name)
	}

	return names
}

// Exists 检查模型是否存在
func (r *ModelRegistry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.models[name]
	return exists
}

// Count 获取模型数量
func (r *ModelRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.models)
}
