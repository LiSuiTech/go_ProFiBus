package interfaces

import (
	"context"
	"time"
	dataManagementDomain "go_ProFiBus/internal/domain/datamanagement"
)

// DataManagementRepository 数据管理仓储接口
type DataManagementRepository interface {
	// CleaningRule相关
	CreateCleaningRule(ctx context.Context, rule *dataManagementDomain.CleaningRule) error
	GetCleaningRuleByID(ctx context.Context, id string) (*dataManagementDomain.CleaningRule, error)
	ListCleaningRules(ctx context.Context, filters CleaningRuleFilters) ([]*dataManagementDomain.CleaningRule, error)
	UpdateCleaningRule(ctx context.Context, rule *dataManagementDomain.CleaningRule) error
	DeleteCleaningRule(ctx context.Context, id string) error

	// ArchivePolicy相关
	CreateArchivePolicy(ctx context.Context, policy *dataManagementDomain.ArchivePolicy) error
	GetArchivePolicyByID(ctx context.Context, id string) (*dataManagementDomain.ArchivePolicy, error)
	ListArchivePolicies(ctx context.Context, filters ArchivePolicyFilters) ([]*dataManagementDomain.ArchivePolicy, error)
	UpdateArchivePolicy(ctx context.Context, policy *dataManagementDomain.ArchivePolicy) error
	DeleteArchivePolicy(ctx context.Context, id string) error
	GetPoliciesToRun(ctx context.Context) ([]*dataManagementDomain.ArchivePolicy, error)

	// ArchiveRecord相关
	CreateArchiveRecord(ctx context.Context, record *dataManagementDomain.ArchiveRecord) error
	GetArchiveRecordByID(ctx context.Context, id string) (*dataManagementDomain.ArchiveRecord, error)
	ListArchiveRecords(ctx context.Context, filters ArchiveRecordFilters) ([]*dataManagementDomain.ArchiveRecord, error)
	UpdateArchiveRecord(ctx context.Context, record *dataManagementDomain.ArchiveRecord) error

	// CleaningRecord相关
	CreateCleaningRecord(ctx context.Context, record *dataManagementDomain.CleaningRecord) error
	GetCleaningRecordByID(ctx context.Context, id string) (*dataManagementDomain.CleaningRecord, error)
	ListCleaningRecords(ctx context.Context, filters CleaningRecordFilters) ([]*dataManagementDomain.CleaningRecord, error)
	UpdateCleaningRecord(ctx context.Context, record *dataManagementDomain.CleaningRecord) error

	// LifecycleConfig相关
	CreateLifecycleConfig(ctx context.Context, config *dataManagementDomain.LifecycleConfig) error
	GetLifecycleConfig(ctx context.Context, sourceType, sourceID string) (*dataManagementDomain.LifecycleConfig, error)
	ListLifecycleConfigs(ctx context.Context, filters LifecycleConfigFilters) ([]*dataManagementDomain.LifecycleConfig, error)
	UpdateLifecycleConfig(ctx context.Context, config *dataManagementDomain.LifecycleConfig) error
	DeleteLifecycleConfig(ctx context.Context, sourceType, sourceID string) error
}

// CleaningRuleFilters 清洗规则过滤器
type CleaningRuleFilters struct {
	RuleType *dataManagementDomain.CleaningRuleType
	Enabled  *bool
	Limit    int
	Offset   int
}

// ArchivePolicyFilters 归档策略过滤器
type ArchivePolicyFilters struct {
	SourceType *string
	SourceID   *string
	Enabled    *bool
	Limit      int
	Offset     int
}

// ArchiveRecordFilters 归档记录过滤器
type ArchiveRecordFilters struct {
	PolicyID   *string
	SourceType *string
	SourceID   *string
	Status     *dataManagementDomain.ArchiveStatus
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
}

// CleaningRecordFilters 清洗记录过滤器
type CleaningRecordFilters struct {
	RuleID     *string
	SourceType *string
	SourceID   *string
	Status     *dataManagementDomain.CleaningStatus
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
}

// LifecycleConfigFilters 生命周期配置过滤器
type LifecycleConfigFilters struct {
	SourceType *string
	SourceID   *string
	Enabled    *bool
	Limit      int
	Offset     int
}
