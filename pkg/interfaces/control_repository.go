package interfaces

import (
	"context"
	"time"
	controlDomain "go_ProFiBus/internal/domain/control"
)

// ControlRepository 控制仓储接口
type ControlRepository interface {
	// ControlPolicy相关
	CreateControlPolicy(ctx context.Context, policy *controlDomain.ControlPolicy) error
	GetControlPolicyByID(ctx context.Context, id string) (*controlDomain.ControlPolicy, error)
	ListControlPolicies(ctx context.Context, filters ControlPolicyFilters) ([]*controlDomain.ControlPolicy, error)
	UpdateControlPolicy(ctx context.Context, policy *controlDomain.ControlPolicy) error
	DeleteControlPolicy(ctx context.Context, id string) error

	// ControlAction相关
	CreateControlAction(ctx context.Context, action *controlDomain.ControlAction) error
	GetControlActionByID(ctx context.Context, id string) (*controlDomain.ControlAction, error)
	ListControlActions(ctx context.Context, filters ControlActionFilters) ([]*controlDomain.ControlAction, error)
	UpdateControlAction(ctx context.Context, action *controlDomain.ControlAction) error

	// AuditLog相关
	CreateAuditLog(ctx context.Context, log *controlDomain.AuditLog) error
	GetAuditLogs(ctx context.Context, filters AuditLogFilters) ([]*controlDomain.AuditLog, error)

	// ControlPermission相关
	CreateControlPermission(ctx context.Context, permission *controlDomain.ControlPermission) error
	GetControlPermission(ctx context.Context, userID string, actionType controlDomain.ActionType) (*controlDomain.ControlPermission, error)
	ListControlPermissions(ctx context.Context, filters ControlPermissionFilters) ([]*controlDomain.ControlPermission, error)
	UpdateControlPermission(ctx context.Context, permission *controlDomain.ControlPermission) error
	DeleteControlPermission(ctx context.Context, id string) error
}

// ControlPolicyFilters 控制策略过滤器
type ControlPolicyFilters struct {
	Enabled *bool
	Limit   int
	Offset  int
}

// ControlActionFilters 控制动作过滤器
type ControlActionFilters struct {
	PolicyID   *string
	DeviceID   *string
	ActionType *controlDomain.ActionType
	Status     *controlDomain.ActionStatus
	ExecutedBy *string
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
}

// AuditLogFilters 审计日志过滤器
type AuditLogFilters struct {
	ActionID  *string
	UserID    *string
	EventType *controlDomain.AuditEventType
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// ControlPermissionFilters 控制权限过滤器
type ControlPermissionFilters struct {
	UserID     *string
	ActionType *controlDomain.ActionType
	Enabled    *bool
	Limit      int
	Offset     int
}
