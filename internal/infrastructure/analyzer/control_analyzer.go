package analyzer

import (
	"context"
	"fmt"
	"sync"

	"go_ProFiBus/pkg/interfaces"
)

// ControlAnalyzer 带控制功能的分析器包装器
// 在分析数据后，根据结果自动执行设备控制动作
type ControlAnalyzer struct {
	name             string
	wrappedAnalyzer  interfaces.Analyzer
	deviceController interfaces.DeviceController
	enableControl    bool
	mu               sync.RWMutex
}

// NewControlAnalyzer 创建带控制功能的分析器
func NewControlAnalyzer(
	name string,
	analyzer interfaces.Analyzer,
	controller interfaces.DeviceController,
) *ControlAnalyzer {
	return &ControlAnalyzer{
		name:             name,
		wrappedAnalyzer:  analyzer,
		deviceController: controller,
		enableControl:    true,
	}
}

// Analyze 分析数据并执行控制 - 实现 Analyzer 接口
func (ca *ControlAnalyzer) Analyze(ctx context.Context, data interfaces.DataSample) ([]interfaces.AnalysisResult, error) {
	// 1. 执行原始分析
	results, err := ca.wrappedAnalyzer.Analyze(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// 2. 如果启用了控制功能，评估并执行控制动作
	ca.mu.RLock()
	enableControl := ca.enableControl
	ca.mu.RUnlock()

	if enableControl && ca.deviceController != nil {
		// 并行执行控制动作（不影响分析结果返回）
		go ca.executeControlActions(ctx, results)
	}

	return results, nil
}

// GetType 获取分析器类型 - 实现 Analyzer 接口
func (ca *ControlAnalyzer) GetType() interfaces.AnalyzerType {
	return ca.wrappedAnalyzer.GetType()
}

// GetName 获取分析器名称 - 实现 Analyzer 接口
func (ca *ControlAnalyzer) GetName() string {
	return ca.name
}

// Configure 配置分析器 - 实现 Analyzer 接口
func (ca *ControlAnalyzer) Configure(config map[string]interface{}) error {
	// 配置控制功能
	if enableControl, ok := config["enable_control"].(bool); ok {
		ca.SetControlEnabled(enableControl)
	}

	// 配置包装的分析器
	return ca.wrappedAnalyzer.Configure(config)
}

// Close 关闭分析器 - 实现 Analyzer 接口
func (ca *ControlAnalyzer) Close() error {
	return ca.wrappedAnalyzer.Close()
}

// SetControlEnabled 设置是否启用控制功能
func (ca *ControlAnalyzer) SetControlEnabled(enabled bool) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.enableControl = enabled
}

// IsControlEnabled 检查控制功能是否启用
func (ca *ControlAnalyzer) IsControlEnabled() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.enableControl
}

// GetDeviceController 获取设备控制器
func (ca *ControlAnalyzer) GetDeviceController() interfaces.DeviceController {
	return ca.deviceController
}

// executeControlActions 执行控制动作
func (ca *ControlAnalyzer) executeControlActions(ctx context.Context, results []interfaces.AnalysisResult) {
	if len(results) == 0 {
		return
	}

	// 评估并执行控制动作
	controlResults, err := ca.deviceController.EvaluateAndExecute(ctx, results)
	if err != nil {
		// 记录错误但不影响主流程
		fmt.Printf("Control execution failed: %v\n", err)
		return
	}

	// 记录控制结果
	for _, result := range controlResults {
		if result != nil {
			if result.Success {
				fmt.Printf("Control action executed successfully: %s\n", result.Message)
			} else {
				fmt.Printf("Control action failed: %s (error: %v)\n", result.Message, result.Error)
			}
		}
	}
}

// 确保实现了Analyzer接口
var _ interfaces.Analyzer = (*ControlAnalyzer)(nil)
