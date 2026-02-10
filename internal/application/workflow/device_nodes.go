package workflow

import (
	"context"
	"fmt"
	"time"

	alertDomain "go_ProFiBus/internal/domain/alert"
	"go_ProFiBus/pkg/interfaces"
)

// DeviceSourceNodeExecutor 设备数据采集节点执行器
type DeviceSourceNodeExecutor struct {
	deviceRepo interfaces.DeviceRepository
	fusionRepo interfaces.FusionRepository
}

// NewDeviceSourceNodeExecutor 创建设备数据采集节点执行器
func NewDeviceSourceNodeExecutor(deviceRepo interfaces.DeviceRepository, fusionRepo interfaces.FusionRepository) *DeviceSourceNodeExecutor {
	return &DeviceSourceNodeExecutor{
		deviceRepo: deviceRepo,
		fusionRepo: fusionRepo,
	}
}

func (e *DeviceSourceNodeExecutor) GetNodeType() NodeType {
	return NodeTypeDeviceSource
}

func (e *DeviceSourceNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["device_id"]; !ok {
		return fmt.Errorf("device_id is required")
	}
	return nil
}

func (e *DeviceSourceNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// 获取设备ID
	deviceID, ok := node.Config["device_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid device_id")
	}

	// 获取字段列表（可选，如果未指定则获取所有字段）
	var fieldNames []string
	if fields, ok := node.Config["fields"].([]interface{}); ok {
		fieldNames = make([]string, 0, len(fields))
		for _, f := range fields {
			if fieldName, ok := f.(string); ok {
				fieldNames = append(fieldNames, fieldName)
			}
		}
	}

	// 获取数据源ID（可选，用于从融合系统获取数据）
	sourceID, _ := node.Config["source_id"].(string)

	// 获取数据
	var data map[string]interface{}
	var quality float64 = 1.0

	if sourceID != "" && e.fusionRepo != nil {
		// 从融合系统获取最新数据
		result, err := e.fusionRepo.GetLatestFusionResult(ctx, sourceID)
		if err == nil && result != nil {
			data = result.FusedData
			quality = result.QualityScore
		}
	}

	// 如果融合系统没有数据，尝试从设备获取
	if data == nil && e.deviceRepo != nil {
		device, err := e.deviceRepo.GetByID(ctx, deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get device: %w", err)
		}

		// 构建设备数据
		data = make(map[string]interface{})
		data["device_id"] = device.ID
		data["device_name"] = device.Name
		data["status"] = string(device.Status)
		data["health_score"] = device.HealthScore

		// 从元数据中提取字段
		if device.Metadata != nil {
			for k, v := range device.Metadata {
				if len(fieldNames) == 0 || contains(fieldNames, k) {
					data[k] = v
				}
			}
		}
	}

	if data == nil {
		return nil, fmt.Errorf("no data available for device %s", deviceID)
	}

	// 过滤字段（如果指定了字段列表）
	if len(fieldNames) > 0 {
		filteredData := make(map[string]interface{})
		for _, fieldName := range fieldNames {
			if value, exists := data[fieldName]; exists {
				filteredData[fieldName] = value
			}
		}
		data = filteredData
	}

	return map[string]interface{}{
		"device_id": deviceID,
		"data":      data,
		"quality":   quality,
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}

// AlertOutputNodeExecutor 告警输出节点执行器
type AlertOutputNodeExecutor struct {
	alertRepo interfaces.AlertRepository
}

// NewAlertOutputNodeExecutor 创建告警输出节点执行器
func NewAlertOutputNodeExecutor(alertRepo interfaces.AlertRepository) *AlertOutputNodeExecutor {
	return &AlertOutputNodeExecutor{
		alertRepo: alertRepo,
	}
}

func (e *AlertOutputNodeExecutor) GetNodeType() NodeType {
	return NodeTypeAlertOutput
}

func (e *AlertOutputNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	// 告警规则ID是可选的，如果没有则使用默认规则
	return nil
}

func (e *AlertOutputNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	if e.alertRepo == nil {
		return nil, fmt.Errorf("alert repository not configured")
	}

	// 获取告警配置
	ruleID, _ := node.Config["rule_id"].(string)
	alertLevel, _ := node.Config["level"].(string)
	if alertLevel == "" {
		alertLevel = "warning" // 默认级别
	}

	// 获取设备ID（从输入或配置中）
	deviceID, _ := inputs["device_id"].(string)
	if deviceID == "" {
		deviceID, _ = node.Config["device_id"].(string)
	}
	if deviceID == "" {
		deviceID, _ = variables["device_id"].(string)
	}

	// 获取数据（从输入中）
	data, ok := inputs["data"].(map[string]interface{})
	if !ok {
		// 尝试从其他输入获取
		data = make(map[string]interface{})
		for k, v := range inputs {
			if k != "device_id" && k != "quality" && k != "timestamp" {
				data[k] = v
			}
		}
	}

	// 获取消息
	message, _ := node.Config["message"].(string)
	if message == "" {
		message, _ = inputs["message"].(string)
	}
	if message == "" {
		message = fmt.Sprintf("告警触发: %v", data)
	}

	// 获取严重程度
	severity := 2 // 默认中等严重程度
	if s, ok := node.Config["severity"].(float64); ok {
		severity = int(s)
	} else if s, ok := inputs["severity"].(float64); ok {
		severity = int(s)
	}

	// 映射告警级别
	var level alertDomain.AlertLevel
	switch alertLevel {
	case "info":
		level = alertDomain.AlertLevelInfo
	case "warning":
		level = alertDomain.AlertLevelWarning
	case "error":
		level = alertDomain.AlertLevelError
	case "critical":
		level = alertDomain.AlertLevelCritical
	default:
		level = alertDomain.AlertLevelWarning
	}

	// 创建告警
	alert := alertDomain.NewAlert(ruleID, deviceID, "", level, message)
	if data != nil {
		alert.Details = data
	}
	if severity > 0 {
		if alert.Details == nil {
			alert.Details = make(map[string]interface{})
		}
		alert.Details["severity"] = severity
	}

	// 保存告警
	if err := e.alertRepo.CreateAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	return map[string]interface{}{
		"alert_id":   alert.ID,
		"level":      string(level),
		"message":    message,
		"device_id":  deviceID,
		"created_at": alert.CreatedAt.Format(time.RFC3339),
	}, nil
}

// DeviceControlNodeExecutor 设备控制节点执行器
type DeviceControlNodeExecutor struct {
	actuatorFactory func(deviceID string) (interfaces.Actuator, error)
	deviceRepo      interfaces.DeviceRepository
}

// NewDeviceControlNodeExecutor 创建设备控制节点执行器
func NewDeviceControlNodeExecutor(actuatorFactory func(deviceID string) (interfaces.Actuator, error), deviceRepo interfaces.DeviceRepository) *DeviceControlNodeExecutor {
	return &DeviceControlNodeExecutor{
		actuatorFactory: actuatorFactory,
		deviceRepo:      deviceRepo,
	}
}

func (e *DeviceControlNodeExecutor) GetNodeType() NodeType {
	return NodeTypeDeviceControl
}

func (e *DeviceControlNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["device_id"]; !ok {
		return fmt.Errorf("device_id is required")
	}
	if _, ok := config["action"]; !ok {
		return fmt.Errorf("action is required")
	}
	return nil
}

func (e *DeviceControlNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// 获取设备ID
	deviceID, ok := node.Config["device_id"].(string)
	if !ok {
		// 尝试从输入或变量中获取
		if id, ok := inputs["device_id"].(string); ok {
			deviceID = id
		} else if id, ok := variables["device_id"].(string); ok {
			deviceID = id
		} else {
			return nil, fmt.Errorf("device_id not found")
		}
	}

	// 获取动作类型
	actionTypeStr, ok := node.Config["action"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid action type")
	}

	// 获取动作参数
	parameters := make(map[string]interface{})
	if params, ok := node.Config["parameters"].(map[string]interface{}); ok {
		parameters = params
	}
	// 也可以从输入中获取参数
	if inputParams, ok := inputs["parameters"].(map[string]interface{}); ok {
		for k, v := range inputParams {
			parameters[k] = v
		}
	}

	// 获取原因
	reason, _ := node.Config["reason"].(string)
	if reason == "" {
		reason, _ = inputs["reason"].(string)
	}
	if reason == "" {
		reason = fmt.Sprintf("Workflow control action: %s", actionTypeStr)
	}

	// 获取严重程度
	severity := 1
	if s, ok := node.Config["severity"].(float64); ok {
		severity = int(s)
	}

	// 获取是否需要确认
	requireConfirmation := false
	if rc, ok := node.Config["require_confirmation"].(bool); ok {
		requireConfirmation = rc
	}

	// 创建控制动作
	action := &interfaces.ControlAction{
		ActionType:          interfaces.ActionType(actionTypeStr),
		TargetDevice:        deviceID,
		Parameters:          parameters,
		Reason:              reason,
		Severity:            severity,
		RequireConfirmation: requireConfirmation,
		Timeout:             30 * time.Second,
		Metadata:            make(map[string]interface{}),
	}
	action.Metadata["workflow_node_id"] = node.ID
	action.Metadata["workflow_node_name"] = node.Name

	// 获取执行器
	var actuator interfaces.Actuator
	var err error
	if e.actuatorFactory != nil {
		actuator, err = e.actuatorFactory(deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get actuator: %w", err)
		}
	} else {
		return nil, fmt.Errorf("actuator factory not configured")
	}

	// 执行控制动作
	result, err := actuator.Execute(ctx, action)
	if err != nil {
		return map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"device_id": deviceID,
			"action":    actionTypeStr,
		}, nil // 返回错误信息但不中断工作流
	}

	return map[string]interface{}{
		"success":    result.Success,
		"message":    result.Message,
		"device_id":  deviceID,
		"action":     actionTypeStr,
		"executed_at": result.ExecutedAt.Format(time.RFC3339),
		"duration":   result.Duration.String(),
		"response":   result.Response,
	}, nil
}

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
