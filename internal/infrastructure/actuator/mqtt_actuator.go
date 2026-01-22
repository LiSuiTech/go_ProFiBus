package actuator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/serial"
)

// MQTTActuator MQTT执行器
// 通过MQTT发送控制命令到设备
type MQTTActuator struct {
	id       string
	name     string
	mqtt     *serial.MQTT
	config   *MQTTActuatorConfig
	status   *interfaces.ActuatorStatus
	mu       sync.RWMutex
}

// MQTTActuatorConfig MQTT执行器配置
type MQTTActuatorConfig struct {
	ActuatorID   string            `json:"actuator_id"`    // 执行器ID
	Name         string            `json:"name"`           // 执行器名称
	Broker       string            `json:"broker"`         // MQTT代理地址
	Username     string            `json:"username"`       // 用户名
	Password     string            `json:"password"`       // 密码
	UseTLS       bool              `json:"use_tls"`        // 是否使用TLS
	QoS          byte              `json:"qos"`            // QoS等级
	ControlTopic string            `json:"control_topic"`  // 控制主题
	DeviceTopics map[string]string `json:"device_topics"`  // 设备主题映射 device_id -> topic
}

// Validate 验证配置
func (c *MQTTActuatorConfig) Validate() error {
	if c.ActuatorID == "" {
		return fmt.Errorf("actuator_id is required")
	}
	if c.Broker == "" {
		return fmt.Errorf("broker is required")
	}
	if c.ControlTopic == "" && len(c.DeviceTopics) == 0 {
		return fmt.Errorf("control_topic or device_topics is required")
	}
	return nil
}

// GetType 获取类型
func (c *MQTTActuatorConfig) GetType() interfaces.ActuatorType {
	return interfaces.ActuatorTypeMQTT
}

// NewMQTTActuator 创建MQTT执行器
func NewMQTTActuator(config *MQTTActuatorConfig) (*MQTTActuator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建MQTT客户端
	mqttConfig := &serial.MQTTConfig{
		Broker:   config.Broker,
		Username: config.Username,
		Password: config.Password,
		Topics:   []string{}, // 执行器不需要订阅
		QoS:      config.QoS,
		UseTLS:   config.UseTLS,
	}

	mqtt := serial.NewMQTT(mqttConfig)

	actuator := &MQTTActuator{
		id:     config.ActuatorID,
		name:   config.Name,
		mqtt:   mqtt,
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
func (a *MQTTActuator) GetID() string {
	return a.id
}

// GetName 获取执行器名称 - 实现 Actuator 接口
func (a *MQTTActuator) GetName() string {
	return a.name
}

// GetType 获取执行器类型 - 实现 Actuator 接口
func (a *MQTTActuator) GetType() interfaces.ActuatorType {
	return interfaces.ActuatorTypeMQTT
}

// Initialize 初始化执行器 - 实现 Actuator 接口
func (a *MQTTActuator) Initialize(config interfaces.ActuatorConfig) error {
	mqttConfig, ok := config.(*MQTTActuatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	a.config = mqttConfig

	// 连接MQTT
	err := a.mqtt.Open(mqttConfig.Broker)
	if err != nil {
		a.mu.Lock()
		a.status.Connected = false
		a.status.Error = err
		a.mu.Unlock()
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	a.mu.Lock()
	a.status.Connected = true
	a.status.Error = nil
	a.mu.Unlock()

	return nil
}

// Execute 执行控制动作 - 实现 Actuator 接口
func (a *MQTTActuator) Execute(ctx context.Context, action *interfaces.ControlAction) (*interfaces.ControlResult, error) {
	startTime := time.Now()

	a.mu.Lock()
	a.status.ActionCount++
	a.status.LastAction = action
	a.mu.Unlock()

	// 构建MQTT消息
	message, err := a.buildMessage(action)
	if err != nil {
		a.recordFailure(action, err, startTime)
		return &interfaces.ControlResult{
			Success:    false,
			Message:    "Failed to build message",
			Error:      err,
			ExecutedAt: startTime,
			Duration:   time.Since(startTime),
		}, err
	}

	// 确定目标主题
	topic := a.getTargetTopic(action.TargetDevice)
	if topic == "" {
		err := fmt.Errorf("no topic configured for device: %s", action.TargetDevice)
		a.recordFailure(action, err, startTime)
		return &interfaces.ControlResult{
			Success:    false,
			Message:    "No topic configured",
			Error:      err,
			ExecutedAt: startTime,
			Duration:   time.Since(startTime),
		}, err
	}

	// 发布MQTT消息
	err = a.mqtt.Publish(topic, message)
	if err != nil {
		a.recordFailure(action, err, startTime)
		return &interfaces.ControlResult{
			Success:    false,
			Message:    "Failed to publish message",
			Error:      err,
			ExecutedAt: startTime,
			Duration:   time.Since(startTime),
		}, err
	}

	// 成功
	result := &interfaces.ControlResult{
		Success: true,
		Message: fmt.Sprintf("Command sent to %s via MQTT", action.TargetDevice),
		Response: map[string]interface{}{
			"topic":   topic,
			"message": message,
		},
		ExecutedAt: startTime,
		Duration:   time.Since(startTime),
	}

	a.recordSuccess(action, result)

	return result, nil
}

// GetStatus 获取执行器状态 - 实现 Actuator 接口
func (a *MQTTActuator) GetStatus() interfaces.ActuatorStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := *a.status
	status.Connected = a.mqtt.IsConnected()

	return status
}

// Close 关闭执行器 - 实现 Actuator 接口
func (a *MQTTActuator) Close() error {
	return a.mqtt.Close()
}

// buildMessage 构建MQTT消息
func (a *MQTTActuator) buildMessage(action *interfaces.ControlAction) (map[string]interface{}, error) {
	message := map[string]interface{}{
		"command":    string(action.ActionType),
		"device":     action.TargetDevice,
		"timestamp":  time.Now().Unix(),
		"reason":     action.Reason,
		"severity":   action.Severity,
		"parameters": action.Parameters,
	}

	// 根据动作类型添加特定字段
	switch action.ActionType {
	case interfaces.ActionEmergencyStop:
		message["emergency"] = true
		message["priority"] = "critical"

	case interfaces.ActionShutdown:
		message["graceful"] = action.Parameters["graceful"] // 是否优雅关闭

	case interfaces.ActionSetValue:
		message["value"] = action.Parameters["value"]
		message["field"] = action.Parameters["field"]

	case interfaces.ActionSendCommand:
		message["command_data"] = action.Parameters["command_data"]
	}

	return message, nil
}

// getTargetTopic 获取目标主题
func (a *MQTTActuator) getTargetTopic(deviceID string) string {
	// 优先使用设备特定主题
	if topic, exists := a.config.DeviceTopics[deviceID]; exists {
		return topic
	}

	// 使用默认控制主题
	if a.config.ControlTopic != "" {
		return fmt.Sprintf("%s/%s", a.config.ControlTopic, deviceID)
	}

	return ""
}

// recordSuccess 记录成功
func (a *MQTTActuator) recordSuccess(action *interfaces.ControlAction, result *interfaces.ControlResult) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.status.SuccessCount++
	a.status.LastResult = result
}

// recordFailure 记录失败
func (a *MQTTActuator) recordFailure(action *interfaces.ControlAction, err error, startTime time.Time) {
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
var _ interfaces.Actuator = (*MQTTActuator)(nil)
var _ interfaces.ActuatorConfig = (*MQTTActuatorConfig)(nil)
