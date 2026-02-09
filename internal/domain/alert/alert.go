package alert

import (
	"github.com/google/uuid"
	"time"
)

// Alert 告警实体
type Alert struct {
	ID              string
	RuleID          string
	DeviceID        string
	ChannelID       string
	EventID         string
	Level           AlertLevel
	Status          AlertStatus
	Message         string
	Details         map[string]interface{}
	FirstOccurredAt time.Time
	LastOccurredAt  time.Time
	AcknowledgedAt  *time.Time
	AcknowledgedBy  string
	ResolvedAt      *time.Time
	ResolvedBy      string
	Count           int // 告警发生次数（用于聚合）
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusActive      AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved    AlertStatus = "resolved"
	AlertStatusSuppressed  AlertStatus = "suppressed"
)

// NewAlert 创建新告警
func NewAlert(ruleID, deviceID, channelID string, level AlertLevel, message string) *Alert {
	now := time.Now()
	return &Alert{
		ID:              generateID(),
		RuleID:          ruleID,
		DeviceID:        deviceID,
		ChannelID:       channelID,
		Level:           level,
		Status:          AlertStatusActive,
		Message:         message,
		Details:         make(map[string]interface{}),
		FirstOccurredAt: now,
		LastOccurredAt:  now,
		Count:           1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// Acknowledge 确认告警
func (a *Alert) Acknowledge(acknowledgedBy string) {
	now := time.Now()
	a.Status = AlertStatusAcknowledged
	a.AcknowledgedAt = &now
	a.AcknowledgedBy = acknowledgedBy
	a.UpdatedAt = now
}

// Resolve 解决告警
func (a *Alert) Resolve(resolvedBy string) {
	now := time.Now()
	a.Status = AlertStatusResolved
	a.ResolvedAt = &now
	a.ResolvedBy = resolvedBy
	a.UpdatedAt = now
}

// IncrementCount 增加告警次数
func (a *Alert) IncrementCount() {
	a.Count++
	a.LastOccurredAt = time.Now()
	a.UpdatedAt = time.Now()
}

// IsActive 检查告警是否处于活动状态
func (a *Alert) IsActive() bool {
	return a.Status == AlertStatusActive
}

// IsCritical 检查是否为严重告警
func (a *Alert) IsCritical() bool {
	return a.Level == AlertLevelCritical || a.Level == AlertLevelError
}

// generateID 生成告警ID
func generateID() string {
	// 使用UUID生成ID
	return "alert_" + uuid.New().String()
}

// AlertRule 告警规则实体
type AlertRule struct {
	ID            string
	Name          string
	Description   string
	Condition     map[string]interface{} // 告警触发条件
	Level         AlertLevel
	Enabled       bool
	CooldownSeconds int // 冷却时间（秒）
	MaxExecutions   int // 最大执行次数（0=无限制）
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewAlertRule 创建新告警规则
func NewAlertRule(id, name string, condition map[string]interface{}, level AlertLevel) *AlertRule {
	return &AlertRule{
		ID:              id,
		Name:            name,
		Condition:       condition,
		Level:           level,
		Enabled:         true,
		CooldownSeconds: 300, // 默认5分钟
		MaxExecutions:   0,   // 无限制
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// Evaluate 评估告警规则（简化实现，实际应该根据条件进行复杂评估）
func (r *AlertRule) Evaluate(data map[string]interface{}) bool {
	if !r.Enabled {
		return false
	}

	// TODO: 实现条件评估逻辑
	// 这里应该根据 condition 中的字段、操作符、阈值等进行评估
	// 目前返回 false，需要根据实际需求实现

	return false
}
