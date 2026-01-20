package interfaces

import (
	"context"
	"time"
)

// Repository 通用仓储接口
// 提供数据持久化的抽象
type Repository interface {
	// Store 存储数据
	Store(ctx context.Context, data interface{}) error

	// Query 查询数据
	Query(ctx context.Context, query Query) ([]interface{}, error)

	// Update 更新数据
	Update(ctx context.Context, id string, data interface{}) error

	// Delete 删除数据
	Delete(ctx context.Context, id string) error

	// Health 健康检查
	Health() error

	// Close 关闭连接
	Close() error
}

// TimeSeriesRepository 时序数据仓储接口
// 专门用于时序数据的存储和查询
type TimeSeriesRepository interface {
	Repository

	// WriteSamples 批量写入数据样本
	WriteSamples(ctx context.Context, samples []DataSample) error

	// QueryByTimeRange 按时间范围查询
	QueryByTimeRange(ctx context.Context, sourceID string, start, end time.Time) ([]DataSample, error)

	// QueryLatest 查询最新数据
	QueryLatest(ctx context.Context, sourceID string, limit int) ([]DataSample, error)

	// Aggregate 数据聚合
	Aggregate(ctx context.Context, query AggregateQuery) ([]AggregateResult, error)

	// DeleteOldData 删除旧数据
	DeleteOldData(ctx context.Context, before time.Time) error
}

// EventRepository 事件仓储接口
type EventRepository interface {
	Repository

	// SaveEvent 保存事件
	SaveEvent(ctx context.Context, event Event) error

	// GetEventByID 根据ID获取事件
	GetEventByID(ctx context.Context, eventID string) (Event, error)

	// QueryEvents 查询事件
	QueryEvents(ctx context.Context, filters EventFilters) ([]Event, error)

	// UpdateEvent 更新事件
	UpdateEvent(ctx context.Context, eventID string, updates map[string]interface{}) error

	// GetEventStats 获取事件统计
	GetEventStats(ctx context.Context, start, end time.Time) (EventStats, error)
}

// RuleRepository 规则仓储接口
type RuleRepository interface {
	Repository

	// SaveRule 保存规则
	SaveRule(ctx context.Context, rule Rule) error

	// GetRuleByID 根据ID获取规则
	GetRuleByID(ctx context.Context, ruleID string) (Rule, error)

	// ListRules 列出规则
	ListRules(ctx context.Context, filters RuleFilters) ([]Rule, error)

	// UpdateRule 更新规则
	UpdateRule(ctx context.Context, ruleID string, updates map[string]interface{}) error

	// DeleteRule 删除规则
	DeleteRule(ctx context.Context, ruleID string) error
}

// Query 查询接口
type Query interface {
	// GetFilters 获取过滤条件
	GetFilters() map[string]interface{}

	// GetLimit 获取限制数量
	GetLimit() int

	// GetOffset 获取偏移量
	GetOffset() int

	// GetOrderBy 获取排序字段
	GetOrderBy() string

	// GetOrderDesc 是否降序
	GetOrderDesc() bool
}

// AggregateQuery 聚合查询
type AggregateQuery struct {
	// SourceID 数据源ID
	SourceID string

	// StartTime 开始时间
	StartTime time.Time

	// EndTime 结束时间
	EndTime time.Time

	// Interval 聚合间隔
	Interval time.Duration

	// AggFunc 聚合函数 (avg, sum, min, max, count)
	AggFunc string

	// Fields 聚合字段
	Fields []string
}

// AggregateResult 聚合结果
type AggregateResult struct {
	// Timestamp 时间戳
	Timestamp time.Time

	// Values 聚合值
	Values map[string]float64

	// Count 数据点数量
	Count int
}

// Event 事件接口
type Event interface {
	// GetID 获取事件ID
	GetID() string

	// GetType 获取事件类型
	GetType() string

	// GetStatus 获取事件状态
	GetStatus() string

	// GetTimestamp 获取时间戳
	GetTimestamp() time.Time

	// GetSeverity 获取严重程度
	GetSeverity() int

	// GetScore 获取分数
	GetScore() float64

	// GetDescription 获取描述
	GetDescription() string

	// GetMetadata 获取元数据
	GetMetadata() map[string]interface{}
}

// EventFilters 事件过滤器
type EventFilters struct {
	// Status 状态过滤
	Status *string

	// Type 类型过滤
	Type *string

	// Severity 严重程度过滤
	Severity *int

	// StartTime 开始时间
	StartTime *time.Time

	// EndTime 结束时间
	EndTime *time.Time

	// Limit 限制数量
	Limit int

	// Offset 偏移量
	Offset int
}

// EventStats 事件统计
type EventStats struct {
	// Total 总数
	Total int

	// ByStatus 按状态统计
	ByStatus map[string]int

	// ByType 按类型统计
	ByType map[string]int

	// BySeverity 按严重程度统计
	BySeverity map[int]int
}

// RuleFilters 规则过滤器
type RuleFilters struct {
	// Enabled 是否启用
	Enabled *bool

	// Type 类型过滤
	Type string

	// Limit 限制数量
	Limit int

	// Offset 偏移量
	Offset int
}
