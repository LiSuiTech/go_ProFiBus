package control

import (
	"context"
	"fmt"
	"sync"

	"go_ProFiBus/pkg/interfaces"
)

// DeviceControllerImpl 设备控制器实现
type DeviceControllerImpl struct {
	actuators map[string]interfaces.Actuator
	policies  map[string]interfaces.ControlPolicy
	mu        sync.RWMutex
}

// NewDeviceController 创建设备控制器
func NewDeviceController() *DeviceControllerImpl {
	return &DeviceControllerImpl{
		actuators: make(map[string]interfaces.Actuator),
		policies:  make(map[string]interfaces.ControlPolicy),
	}
}

// RegisterActuator 注册执行器 - 实现 DeviceController 接口
func (dc *DeviceControllerImpl) RegisterActuator(actuator interfaces.Actuator) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if actuator == nil {
		return fmt.Errorf("actuator cannot be nil")
	}

	actuatorID := actuator.GetID()
	if _, exists := dc.actuators[actuatorID]; exists {
		return fmt.Errorf("actuator %s already registered", actuatorID)
	}

	dc.actuators[actuatorID] = actuator
	return nil
}

// UnregisterActuator 注销执行器 - 实现 DeviceController 接口
func (dc *DeviceControllerImpl) UnregisterActuator(actuatorID string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if _, exists := dc.actuators[actuatorID]; !exists {
		return fmt.Errorf("actuator %s not found", actuatorID)
	}

	delete(dc.actuators, actuatorID)
	return nil
}

// GetActuator 获取执行器 - 实现 DeviceController 接口
func (dc *DeviceControllerImpl) GetActuator(actuatorID string) (interfaces.Actuator, error) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	actuator, exists := dc.actuators[actuatorID]
	if !exists {
		return nil, fmt.Errorf("actuator %s not found", actuatorID)
	}

	return actuator, nil
}

// ListActuators 列出所有执行器 - 实现 DeviceController 接口
func (dc *DeviceControllerImpl) ListActuators() []interfaces.Actuator {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	list := make([]interfaces.Actuator, 0, len(dc.actuators))
	for _, actuator := range dc.actuators {
		list = append(list, actuator)
	}

	return list
}

// ExecuteAction 执行控制动作 - 实现 DeviceController 接口
func (dc *DeviceControllerImpl) ExecuteAction(ctx context.Context, action *interfaces.ControlAction) (*interfaces.ControlResult, error) {
	// 查找合适的执行器
	actuator, err := dc.findActuatorForAction(action)
	if err != nil {
		return nil, fmt.Errorf("no suitable actuator found: %w", err)
	}

	// 执行动作
	return actuator.Execute(ctx, action)
}

// ExecuteActions 批量执行控制动作（并行）- 实现 DeviceController 接口
func (dc *DeviceControllerImpl) ExecuteActions(ctx context.Context, actions []*interfaces.ControlAction) ([]*interfaces.ControlResult, error) {
	results := make([]*interfaces.ControlResult, len(actions))
	errors := make([]error, len(actions))

	var wg sync.WaitGroup
	for i, action := range actions {
		wg.Add(1)
		go func(index int, act *interfaces.ControlAction) {
			defer wg.Done()

			result, err := dc.ExecuteAction(ctx, act)
			results[index] = result
			errors[index] = err
		}(i, action)
	}

	wg.Wait()

	// 检查是否有错误
	hasError := false
	for _, err := range errors {
		if err != nil {
			hasError = true
			break
		}
	}

	if hasError {
		return results, fmt.Errorf("some actions failed")
	}

	return results, nil
}

// RegisterPolicy 注册控制策略
func (dc *DeviceControllerImpl) RegisterPolicy(policy interfaces.ControlPolicy) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	policyID := policy.GetID()
	if _, exists := dc.policies[policyID]; exists {
		return fmt.Errorf("policy %s already registered", policyID)
	}

	dc.policies[policyID] = policy
	return nil
}

// UnregisterPolicy 注销控制策略
func (dc *DeviceControllerImpl) UnregisterPolicy(policyID string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if _, exists := dc.policies[policyID]; !exists {
		return fmt.Errorf("policy %s not found", policyID)
	}

	delete(dc.policies, policyID)
	return nil
}

// EvaluateAndExecute 评估分析结果并执行必要的控制动作
func (dc *DeviceControllerImpl) EvaluateAndExecute(ctx context.Context, results []interfaces.AnalysisResult) ([]*interfaces.ControlResult, error) {
	// 收集所有策略生成的动作
	allActions := make([]*interfaces.ControlAction, 0)

	dc.mu.RLock()
	policies := make([]interfaces.ControlPolicy, 0, len(dc.policies))
	for _, policy := range dc.policies {
		if policy.IsEnabled() {
			policies = append(policies, policy)
		}
	}
	dc.mu.RUnlock()

	// 评估所有启用的策略
	for _, policy := range policies {
		actions, err := policy.Evaluate(results)
		if err != nil {
			continue // 忽略评估失败的策略
		}
		allActions = append(allActions, actions...)
	}

	// 如果没有需要执行的动作，直接返回
	if len(allActions) == 0 {
		return nil, nil
	}

	// 执行所有动作（并行）
	return dc.ExecuteActions(ctx, allActions)
}

// findActuatorForAction 查找适合执行指定动作的执行器
func (dc *DeviceControllerImpl) findActuatorForAction(action *interfaces.ControlAction) (interfaces.Actuator, error) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// 简单策略：选择第一个启用且已连接的执行器
	// 实际应用中可以根据设备类型、优先级等选择
	for _, actuator := range dc.actuators {
		status := actuator.GetStatus()
		if status.Enabled && status.Connected {
			return actuator, nil
		}
	}

	return nil, fmt.Errorf("no enabled and connected actuator available")
}

// 确保实现了DeviceController接口
var _ interfaces.DeviceController = (*DeviceControllerImpl)(nil)
