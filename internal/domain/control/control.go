package control

import (
	"time"
)

// ControlPolicy 控制策略
type ControlPolicy struct {
	ID              string
	Name            string
	Description     string
	Enabled         bool
	Priority        int
	ConditionConfig map[string]interface{} // 触发条件配置
	ActionConfig    map[string]interface{} // 控制动作配置
	CooldownSeconds int
	MaxExecutions   int
	ExecutionCount  int
	LastExecutedAt  *time.Time
	Metadata        map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewControlPolicy 创建控制策略
func NewControlPolicy(id, name string) *ControlPolicy {
	return &ControlPolicy{
		ID:              id,
		Name:            name,
		Enabled:         true,
		Priority:        0,
		ConditionConfig: make(map[string]interface{}),
		ActionConfig:    make(map[string]interface{}),
		CooldownSeconds: 300,
		MaxExecutions:   0,
		ExecutionCount:  0,
		Metadata:        make(map[string]interface{}),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// CanExecute 检查是否可以执行
func (p *ControlPolicy) CanExecute() bool {
	if !p.Enabled {
		return false
	}

	// 检查最大执行次数
	if p.MaxExecutions > 0 && p.ExecutionCount >= p.MaxExecutions {
		return false
	}

	// 检查冷却时间
	if p.LastExecutedAt != nil {
		cooldown := time.Duration(p.CooldownSeconds) * time.Second
		if time.Since(*p.LastExecutedAt) < cooldown {
			return false
		}
	}

	return true
}

// RecordExecution 记录执行
func (p *ControlPolicy) RecordExecution() {
	now := time.Now()
	p.ExecutionCount++
	p.LastExecutedAt = &now
	p.UpdatedAt = now
}

// ControlAction 控制动作
type ControlAction struct {
	ID                 string
	PolicyID           string
	DeviceID           string
	ActionType         ActionType
	Parameters         map[string]interface{}
	Reason             string
	Severity           int
	Status             ActionStatus
	Result             map[string]interface{}
	ErrorMessage       string
	ExecutedBy         string
	ExecutedAt         *time.Time
	CompletedAt        *time.Time
	DurationMs         int
	RequireConfirmation bool
	ConfirmedBy        string
	ConfirmedAt        *time.Time
	Metadata           map[string]interface{}
	CreatedAt          time.Time
}

// ActionType 动作类型
type ActionType string

const (
	ActionTypeEmergencyStop ActionType = "emergency_stop"
	ActionTypeShutdown      ActionType = "shutdown"
	ActionTypeStart         ActionType = "start"
	ActionTypePause         ActionType = "pause"
	ActionTypeResume        ActionType = "resume"
	ActionTypeSetValue      ActionType = "set_value"
	ActionTypeCallMethod    ActionType = "call_method"
	ActionTypeSendCommand   ActionType = "send_command"
	ActionTypeCustom        ActionType = "custom"
)

// ActionStatus 动作状态
type ActionStatus string

const (
	ActionStatusPending   ActionStatus = "pending"
	ActionStatusExecuting ActionStatus = "executing"
	ActionStatusCompleted ActionStatus = "completed"
	ActionStatusFailed    ActionStatus = "failed"
	ActionStatusCancelled ActionStatus = "cancelled"
)

// NewControlAction 创建控制动作
func NewControlAction(id, deviceID string, actionType ActionType) *ControlAction {
	return &ControlAction{
		ID:          id,
		DeviceID:    deviceID,
		ActionType:  actionType,
		Status:      ActionStatusPending,
		Severity:    1,
		Parameters:  make(map[string]interface{}),
		Result:      make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
	}
}

// Confirm 确认动作
func (a *ControlAction) Confirm(confirmedBy string) {
	now := time.Now()
	a.ConfirmedBy = confirmedBy
	a.ConfirmedAt = &now
	a.Metadata["confirmed"] = true
}

// StartExecution 开始执行
func (a *ControlAction) StartExecution(executedBy string) {
	now := time.Now()
	a.Status = ActionStatusExecuting
	a.ExecutedBy = executedBy
	a.ExecutedAt = &now
}

// Complete 完成执行
func (a *ControlAction) Complete(result map[string]interface{}, durationMs int) {
	now := time.Now()
	a.Status = ActionStatusCompleted
	a.Result = result
	a.CompletedAt = &now
	a.DurationMs = durationMs
}

// Fail 执行失败
func (a *ControlAction) Fail(errorMessage string) {
	now := time.Now()
	a.Status = ActionStatusFailed
	a.ErrorMessage = errorMessage
	a.CompletedAt = &now
}

// Cancel 取消执行
func (a *ControlAction) Cancel() {
	now := time.Now()
	a.Status = ActionStatusCancelled
	a.CompletedAt = &now
}

// IsPending 检查是否待执行
func (a *ControlAction) IsPending() bool {
	return a.Status == ActionStatusPending
}

// IsCompleted 检查是否已完成
func (a *ControlAction) IsCompleted() bool {
	return a.Status == ActionStatusCompleted
}

// AuditLog 审计日志
type AuditLog struct {
	ID        string
	ActionID  string
	EventType AuditEventType
	UserID    string
	UserName  string
	Details   map[string]interface{}
	IPAddress string
	UserAgent string
	CreatedAt time.Time
}

// AuditEventType 审计事件类型
type AuditEventType string

const (
	AuditEventCreated   AuditEventType = "created"
	AuditEventConfirmed AuditEventType = "confirmed"
	AuditEventExecuted  AuditEventType = "executed"
	AuditEventCompleted AuditEventType = "completed"
	AuditEventFailed    AuditEventType = "failed"
	AuditEventCancelled AuditEventType = "cancelled"
)

// NewAuditLog 创建审计日志
func NewAuditLog(id, actionID string, eventType AuditEventType) *AuditLog {
	return &AuditLog{
		ID:        id,
		ActionID:  actionID,
		EventType: eventType,
		Details:   make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
}

// ControlPermission 控制权限
type ControlPermission struct {
	ID                 string
	UserID             string
	ActionType         ActionType
	TargetDevices      []string
	MaxSeverity        int
	RequireConfirmation bool
	AllowedTimeRanges  []TimeRange
	Enabled            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewControlPermission 创建控制权限
func NewControlPermission(id, userID string, actionType ActionType) *ControlPermission {
	return &ControlPermission{
		ID:                 id,
		UserID:             userID,
		ActionType:         actionType,
		TargetDevices:      make([]string, 0),
		MaxSeverity:        3,
		RequireConfirmation: false,
		AllowedTimeRanges:  make([]TimeRange, 0),
		Enabled:            true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

// CanControlDevice 检查是否可以控制设备
func (p *ControlPermission) CanControlDevice(deviceID string) bool {
	if !p.Enabled {
		return false
	}

	// 如果目标设备列表为空，表示可以控制所有设备
	if len(p.TargetDevices) == 0 {
		return true
	}

	// 检查设备是否在允许列表中
	for _, id := range p.TargetDevices {
		if id == deviceID {
			return true
		}
	}

	return false
}

// CanControlAtTime 检查在指定时间是否可以控制
func (p *ControlPermission) CanControlAtTime(t time.Time) bool {
	if len(p.AllowedTimeRanges) == 0 {
		return true // 没有时间限制
	}

	for _, tr := range p.AllowedTimeRanges {
		if t.After(tr.Start) && t.Before(tr.End) {
			return true
		}
	}

	return false
}
