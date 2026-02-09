package interfaces

import (
	"context"
	alertDomain "go_ProFiBus/internal/domain/alert"
	"time"
)

// AlertRepository 告警仓储接口
type AlertRepository interface {
	// CreateAlert 创建告警
	CreateAlert(ctx context.Context, alert *alertDomain.Alert) error

	// GetAlertByID 根据ID获取告警
	GetAlertByID(ctx context.Context, id string) (*alertDomain.Alert, error)

	// ListAlerts 列出告警
	ListAlerts(ctx context.Context, filters AlertFilters) ([]*alertDomain.Alert, error)

	// UpdateAlert 更新告警
	UpdateAlert(ctx context.Context, alert *alertDomain.Alert) error

	// AcknowledgeAlert 确认告警
	AcknowledgeAlert(ctx context.Context, id, acknowledgedBy string) error

	// ResolveAlert 解决告警
	ResolveAlert(ctx context.Context, id, resolvedBy string) error

	// GetActiveAlerts 获取活动告警
	GetActiveAlerts(ctx context.Context, deviceID string) ([]*alertDomain.Alert, error)

	// GetAlertStats 获取告警统计
	GetAlertStats(ctx context.Context, filters AlertFilters) (*AlertStats, error)

	// CreateAlertRule 创建告警规则
	CreateAlertRule(ctx context.Context, rule *alertDomain.AlertRule) error

	// GetAlertRuleByID 根据ID获取告警规则
	GetAlertRuleByID(ctx context.Context, id string) (*alertDomain.AlertRule, error)

	// ListAlertRules 列出告警规则
	ListAlertRules(ctx context.Context, enabled *bool) ([]*alertDomain.AlertRule, error)

	// UpdateAlertRule 更新告警规则
	UpdateAlertRule(ctx context.Context, rule *alertDomain.AlertRule) error

	// DeleteAlertRule 删除告警规则
	DeleteAlertRule(ctx context.Context, id string) error
}

// AlertFilters 告警过滤器
type AlertFilters struct {
	RuleID    *string
	DeviceID  *string
	ChannelID *string
	Level     *alertDomain.AlertLevel
	Status    *alertDomain.AlertStatus
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// AlertStats 告警统计
type AlertStats struct {
	TotalAlerts      int64
	ActiveAlerts     int64
	AcknowledgedAlerts int64
	ResolvedAlerts   int64
	AlertsByLevel    map[string]int64
	AlertsByStatus   map[string]int64
}
