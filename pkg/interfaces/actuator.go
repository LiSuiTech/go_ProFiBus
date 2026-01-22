package interfaces

import (
	"context"
	"time"
)

// Actuator 执行器接口 - 设备控制的核心接口
// 用于执行各种设备控制操作（急停、关闭、启动等）
type Actuator interface {
	// GetID 获取执行器ID
	GetID() string

	// GetName 获取执行器名称
	GetName() string

	// GetType 获取执行器类型
	GetType() ActuatorType

	// Execute 执行控制命令
	Execute(ctx context.Context, action *ControlAction) (*ControlResult, error)

	// GetStatus 获取执行器状态
	GetStatus() ActuatorStatus

	// Initialize 初始化执行器
	Initialize(config ActuatorConfig) error

	// Close 关闭执行器
	Close() error
}

// ActuatorType 执行器类型
type ActuatorType string

const (
	// ActuatorTypeMQTT MQTT执行器（通过MQTT发送控制命令）
	ActuatorTypeMQTT ActuatorType = "mqtt"

	// ActuatorTypeOPCUA OPC-UA执行器（调用OPC-UA方法或写入节点）
	ActuatorTypeOPCUA ActuatorType = "opcua"

	// ActuatorTypeModbus Modbus执行器（写入寄存器）
	ActuatorTypeModbus ActuatorType = "modbus"

	// ActuatorTypeHTTP HTTP执行器（发送HTTP请求）
	ActuatorTypeHTTP ActuatorType = "http"

	// ActuatorTypeScript 脚本执行器（执行shell脚本）
	ActuatorTypeScript ActuatorType = "script"

	// ActuatorTypeCustom 自定义执行器
	ActuatorTypeCustom ActuatorType = "custom"
)

// ControlAction 控制动作
type ControlAction struct {
	// ActionType 动作类型
	ActionType ActionType

	// TargetDevice 目标设备ID
	TargetDevice string

	// Parameters 动作参数
	Parameters map[string]interface{}

	// Reason 执行原因
	Reason string

	// Severity 严重程度（用于判断是否需要确认）
	Severity int

	// RequireConfirmation 是否需要确认
	RequireConfirmation bool

	// Timeout 超时时间
	Timeout time.Duration

	// Metadata 元数据
	Metadata map[string]interface{}
}

// ActionType 动作类型
type ActionType string

const (
	// ActionEmergencyStop 紧急停止
	ActionEmergencyStop ActionType = "emergency_stop"

	// ActionShutdown 关闭设备
	ActionShutdown ActionType = "shutdown"

	// ActionStart 启动设备
	ActionStart ActionType = "start"

	// ActionPause 暂停设备
	ActionPause ActionType = "pause"

	// ActionResume 恢复设备
	ActionResume ActionType = "resume"

	// ActionSetValue 设置值
	ActionSetValue ActionType = "set_value"

	// ActionCallMethod 调用方法
	ActionCallMethod ActionType = "call_method"

	// ActionSendCommand 发送命令
	ActionSendCommand ActionType = "send_command"

	// ActionCustom 自定义动作
	ActionCustom ActionType = "custom"
)

// ControlResult 控制结果
type ControlResult struct {
	// Success 是否成功
	Success bool

	// Message 结果消息
	Message string

	// Error 错误信息
	Error error

	// Response 响应数据
	Response map[string]interface{}

	// ExecutedAt 执行时间
	ExecutedAt time.Time

	// Duration 执行耗时
	Duration time.Duration
}

// ActuatorStatus 执行器状态
type ActuatorStatus struct {
	// Enabled 是否启用
	Enabled bool

	// Connected 是否已连接
	Connected bool

	// Error 错误信息
	Error error

	// LastAction 最后执行的动作
	LastAction *ControlAction

	// LastResult 最后执行的结果
	LastResult *ControlResult

	// ActionCount 执行次数统计
	ActionCount uint64

	// SuccessCount 成功次数
	SuccessCount uint64

	// FailureCount 失败次数
	FailureCount uint64
}

// ActuatorConfig 执行器配置接口
type ActuatorConfig interface {
	// Validate 验证配置
	Validate() error

	// GetType 获取执行器类型
	GetType() ActuatorType
}

// DeviceController 设备控制器接口
// 管理多个执行器，协调设备控制
type DeviceController interface {
	// RegisterActuator 注册执行器
	RegisterActuator(actuator Actuator) error

	// UnregisterActuator 注销执行器
	UnregisterActuator(actuatorID string) error

	// GetActuator 获取执行器
	GetActuator(actuatorID string) (Actuator, error)

	// ListActuators 列出所有执行器
	ListActuators() []Actuator

	// ExecuteAction 执行控制动作
	ExecuteAction(ctx context.Context, action *ControlAction) (*ControlResult, error)

	// ExecuteActions 批量执行控制动作（并行）
	ExecuteActions(ctx context.Context, actions []*ControlAction) ([]*ControlResult, error)

	// RegisterPolicy 注册控制策略
	RegisterPolicy(policy ControlPolicy) error

	// UnregisterPolicy 注销控制策略
	UnregisterPolicy(policyID string) error

	// EvaluateAndExecute 评估分析结果并执行必要的控制动作
	EvaluateAndExecute(ctx context.Context, results []AnalysisResult) ([]*ControlResult, error)
}

// ControlPolicy 控制策略接口
// 定义何时以及如何执行设备控制
type ControlPolicy interface {
	// Evaluate 评估是否需要执行控制
	// 输入：分析结果，输出：需要执行的控制动作列表
	Evaluate(results []AnalysisResult) ([]*ControlAction, error)

	// GetID 获取策略ID
	GetID() string

	// GetName 获取策略名称
	GetName() string

	// IsEnabled 是否启用
	IsEnabled() bool

	// SetEnabled 设置启用状态
	SetEnabled(enabled bool)
}

// ControlRule 控制规则
// 用于定义触发条件和控制动作
type ControlRule struct {
	// ID 规则ID
	ID string

	// Name 规则名称
	Name string

	// Enabled 是否启用
	Enabled bool

	// Priority 优先级（数字越大优先级越高）
	Priority int

	// Condition 触发条件
	Condition RuleCondition

	// Actions 要执行的动作列表
	Actions []*ControlAction

	// Cooldown 冷却时间（避免频繁触发）
	Cooldown time.Duration

	// MaxExecutions 最大执行次数（0=无限制）
	MaxExecutions int

	// Description 规则描述
	Description string
}

// RuleCondition 规则条件
type RuleCondition struct {
	// AnalysisType 分析类型（如 "ml_anomaly_detection"）
	AnalysisType string

	// MinScore 最小分数阈值
	MinScore float64

	// MinSeverity 最小严重程度
	MinSeverity int

	// RequiredDetails 必须包含的详细信息
	RequiredDetails map[string]interface{}

	// CustomEvaluator 自定义评估函数
	CustomEvaluator func(AnalysisResult) bool
}

// Permission 权限定义
type Permission struct {
	// Action 允许的动作类型
	Action ActionType

	// TargetDevices 允许控制的设备列表（空=所有设备）
	TargetDevices []string

	// MaxSeverity 允许的最大严重程度
	MaxSeverity int

	// RequireConfirmation 是否需要人工确认
	RequireConfirmation bool

	// AllowedTimeRanges 允许执行的时间范围
	AllowedTimeRanges []TimeRange
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// AuthorizationManager 授权管理器接口
type AuthorizationManager interface {
	// CheckPermission 检查是否有权限执行动作
	CheckPermission(userID string, action *ControlAction) (bool, error)

	// GrantPermission 授予权限
	GrantPermission(userID string, permission *Permission) error

	// RevokePermission 撤销权限
	RevokePermission(userID string, action ActionType) error

	// GetPermissions 获取用户权限
	GetPermissions(userID string) ([]*Permission, error)
}
