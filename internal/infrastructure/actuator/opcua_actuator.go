package actuator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/serial"
)

// OPCUAActuator OPC-UA执行器
// 通过OPC-UA调用方法或写入节点来控制设备
type OPCUAActuator struct {
	id     string
	name   string
	opcua  *serial.OPCUA
	config *OPCUAActuatorConfig
	status *interfaces.ActuatorStatus
	mu     sync.RWMutex
}

// OPCUAActuatorConfig OPC-UA执行器配置
type OPCUAActuatorConfig struct {
	ActuatorID     string                       `json:"actuator_id"`      // 执行器ID
	Name           string                       `json:"name"`             // 执行器名称
	Endpoint       string                       `json:"endpoint"`         // OPC-UA服务器端点
	SecurityPolicy string                       `json:"security_policy"`  // 安全策略
	SecurityMode   string                       `json:"security_mode"`    // 安全模式
	Username       string                       `json:"username"`         // 用户名
	Password       string                       `json:"password"`         // 密码
	DeviceNodes    map[string]*DeviceNodeConfig `json:"device_nodes"`     // 设备节点配置
}

// DeviceNodeConfig 设备节点配置
type DeviceNodeConfig struct {
	ObjectID            string            `json:"object_id"`             // 对象节点ID
	EmergencyStopMethod string            `json:"emergency_stop_method"` // 紧急停止方法ID
	ShutdownMethod      string            `json:"shutdown_method"`       // 关闭方法ID
	StartMethod         string            `json:"start_method"`          // 启动方法ID
	ControlNodes        map[string]string `json:"control_nodes"`         // 控制节点映射 (action -> node_id)
}

// Validate 验证配置
func (c *OPCUAActuatorConfig) Validate() error {
	if c.ActuatorID == "" {
		return fmt.Errorf("actuator_id is required")
	}
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	if len(c.DeviceNodes) == 0 {
		return fmt.Errorf("device_nodes is required")
	}
	return nil
}

// GetType 获取类型
func (c *OPCUAActuatorConfig) GetType() interfaces.ActuatorType {
	return interfaces.ActuatorTypeOPCUA
}

// NewOPCUAActuator 创建OPC-UA执行器
func NewOPCUAActuator(config *OPCUAActuatorConfig) (*OPCUAActuator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建OPC-UA客户端
	opcuaConfig := &serial.OPCUAConfig{
		Endpoint:       config.Endpoint,
		SecurityPolicy: config.SecurityPolicy,
		SecurityMode:   config.SecurityMode,
		Username:       config.Username,
		Password:       config.Password,
	}

	opcua := serial.NewOPCUA(opcuaConfig)

	actuator := &OPCUAActuator{
		id:     config.ActuatorID,
		name:   config.Name,
		opcua:  opcua,
		config: config,
		status: &interfaces.ActuatorStatus{
			Enabled:      true,
			Connected:    false,
			ActionCount:  0,
			SuccessCount: 0,
			FailureCount: 0,
		},
	}

	return actuator, nil
}

// GetID 获取执行器ID - 实现 Actuator 接口
func (a *OPCUAActuator) GetID() string {
	return a.id
}

// GetName 获取执行器名称 - 实现 Actuator 接口
func (a *OPCUAActuator) GetName() string {
	return a.name
}

// GetType 获取执行器类型 - 实现 Actuator 接口
func (a *OPCUAActuator) GetType() interfaces.ActuatorType {
	return interfaces.ActuatorTypeOPCUA
}

// Initialize 初始化执行器 - 实现 Actuator 接口
func (a *OPCUAActuator) Initialize(config interfaces.ActuatorConfig) error {
	opcuaConfig, ok := config.(*OPCUAActuatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	a.config = opcuaConfig

	// 连接OPC-UA
	err := a.opcua.Open(opcuaConfig.Endpoint)
	if err != nil {
		a.mu.Lock()
		a.status.Connected = false
		a.status.Error = err
		a.mu.Unlock()
		return fmt.Errorf("failed to connect OPC-UA: %w", err)
	}

	a.mu.Lock()
	a.status.Connected = true
	a.status.Error = nil
	a.mu.Unlock()

	return nil
}

// Execute 执行控制动作 - 实现 Actuator 接口
func (a *OPCUAActuator) Execute(ctx context.Context, action *interfaces.ControlAction) (*interfaces.ControlResult, error) {
	startTime := time.Now()

	a.mu.Lock()
	a.status.ActionCount++
	a.status.LastAction = action
	a.mu.Unlock()

	// 获取设备节点配置
	deviceNode, exists := a.config.DeviceNodes[action.TargetDevice]
	if !exists {
		err := fmt.Errorf("no node config for device: %s", action.TargetDevice)
		a.recordFailure(action, err, startTime)
		return &interfaces.ControlResult{
			Success:    false,
			Message:    "Device not configured",
			Error:      err,
			ExecutedAt: startTime,
			Duration:   time.Since(startTime),
		}, err
	}

	// 根据动作类型执行不同的操作
	var err error
	var response map[string]interface{}

	switch action.ActionType {
	case interfaces.ActionEmergencyStop:
		err = a.executeEmergencyStop(deviceNode)
		response = map[string]interface{}{
			"method": "emergency_stop",
			"node":   deviceNode.EmergencyStopMethod,
		}

	case interfaces.ActionShutdown:
		err = a.executeShutdown(deviceNode, action.Parameters)
		response = map[string]interface{}{
			"method": "shutdown",
			"node":   deviceNode.ShutdownMethod,
		}

	case interfaces.ActionStart:
		err = a.executeStart(deviceNode)
		response = map[string]interface{}{
			"method": "start",
			"node":   deviceNode.StartMethod,
		}

	case interfaces.ActionSetValue:
		err = a.executeSetValue(deviceNode, action.Parameters)
		response = map[string]interface{}{
			"method": "set_value",
		}

	case interfaces.ActionCallMethod:
		var results []interface{}
		results, err = a.executeCallMethod(deviceNode, action.Parameters)
		response = map[string]interface{}{
			"method":  "call_method",
			"results": results,
		}

	default:
		err = fmt.Errorf("unsupported action type: %s", action.ActionType)
	}

	if err != nil {
		a.recordFailure(action, err, startTime)
		return &interfaces.ControlResult{
			Success:    false,
			Message:    fmt.Sprintf("Failed to execute %s", action.ActionType),
			Error:      err,
			ExecutedAt: startTime,
			Duration:   time.Since(startTime),
		}, err
	}

	// 成功
	result := &interfaces.ControlResult{
		Success:    true,
		Message:    fmt.Sprintf("Successfully executed %s on %s", action.ActionType, action.TargetDevice),
		Response:   response,
		ExecutedAt: startTime,
		Duration:   time.Since(startTime),
	}

	a.recordSuccess(action, result)

	return result, nil
}

// GetStatus 获取执行器状态 - 实现 Actuator 接口
func (a *OPCUAActuator) GetStatus() interfaces.ActuatorStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := *a.status
	status.Connected = a.opcua.IsConnected()

	return status
}

// Close 关闭执行器 - 实现 Actuator 接口
func (a *OPCUAActuator) Close() error {
	return a.opcua.Close()
}

// executeEmergencyStop 执行紧急停止
func (a *OPCUAActuator) executeEmergencyStop(deviceNode *DeviceNodeConfig) error {
	if deviceNode.EmergencyStopMethod == "" {
		return fmt.Errorf("emergency stop method not configured")
	}

	// 调用紧急停止方法
	_, err := a.opcua.CallMethod(deviceNode.ObjectID, deviceNode.EmergencyStopMethod)
	return err
}

// executeShutdown 执行关闭
func (a *OPCUAActuator) executeShutdown(deviceNode *DeviceNodeConfig, params map[string]interface{}) error {
	if deviceNode.ShutdownMethod == "" {
		return fmt.Errorf("shutdown method not configured")
	}

	// 是否优雅关闭
	graceful := false
	if g, ok := params["graceful"].(bool); ok {
		graceful = g
	}

	// 调用关闭方法
	_, err := a.opcua.CallMethod(deviceNode.ObjectID, deviceNode.ShutdownMethod, graceful)
	return err
}

// executeStart 执行启动
func (a *OPCUAActuator) executeStart(deviceNode *DeviceNodeConfig) error {
	if deviceNode.StartMethod == "" {
		return fmt.Errorf("start method not configured")
	}

	// 调用启动方法
	_, err := a.opcua.CallMethod(deviceNode.ObjectID, deviceNode.StartMethod)
	return err
}

// executeSetValue 执行设置值
func (a *OPCUAActuator) executeSetValue(deviceNode *DeviceNodeConfig, params map[string]interface{}) error {
	field, ok := params["field"].(string)
	if !ok {
		return fmt.Errorf("field parameter is required")
	}

	value, ok := params["value"]
	if !ok {
		return fmt.Errorf("value parameter is required")
	}

	// 从控制节点映射中查找节点ID
	nodeID, exists := deviceNode.ControlNodes[field]
	if !exists {
		return fmt.Errorf("control node for field '%s' not configured", field)
	}

	// 写入节点值
	return a.opcua.WriteNode(nodeID, value)
}

// executeCallMethod 执行调用方法
func (a *OPCUAActuator) executeCallMethod(deviceNode *DeviceNodeConfig, params map[string]interface{}) ([]interface{}, error) {
	methodID, ok := params["method_id"].(string)
	if !ok {
		return nil, fmt.Errorf("method_id parameter is required")
	}

	args, ok := params["args"].([]interface{})
	if !ok {
		args = []interface{}{}
	}

	// 调用方法
	return a.opcua.CallMethod(deviceNode.ObjectID, methodID, args...)
}

// recordSuccess 记录成功
func (a *OPCUAActuator) recordSuccess(action *interfaces.ControlAction, result *interfaces.ControlResult) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.status.SuccessCount++
	a.status.LastResult = result
}

// recordFailure 记录失败
func (a *OPCUAActuator) recordFailure(action *interfaces.ControlAction, err error, startTime time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.status.FailureCount++
	a.status.Error = err
	a.status.LastResult = &interfaces.ControlResult{
		Success:    false,
		Error:      err,
		ExecutedAt: startTime,
		Duration:   time.Since(startTime),
	}
}

// 确保实现了Actuator接口
var _ interfaces.Actuator = (*OPCUAActuator)(nil)
var _ interfaces.ActuatorConfig = (*OPCUAActuatorConfig)(nil)
