package interfaces

import (
	"context"
	"time"
)

// Tracer 数据流追踪器接口
// 用于追踪数据在 Pipeline 中的流动过程
type Tracer interface {
	// TraceDataFlow 追踪数据流经某个组件
	TraceDataFlow(ctx context.Context, event *TraceEvent) error

	// GetTraces 获取追踪记录
	GetTraces(ctx context.Context, filter TraceFilter) ([]TraceEvent, error)

	// Subscribe 订阅追踪事件（用于实时推送）
	Subscribe() <-chan TraceEvent

	// Unsubscribe 取消订阅
	Unsubscribe(ch <-chan TraceEvent)

	// Close 关闭追踪器
	Close() error
}

// TraceEvent 追踪事件
type TraceEvent struct {
	// ID 事件ID
	ID string `json:"id"`

	// PipelineID 管道ID
	PipelineID string `json:"pipeline_id"`

	// SampleID 数据样本ID
	SampleID string `json:"sample_id"`

	// ComponentType 组件类型: source, processor, analyzer, sink
	ComponentType string `json:"component_type"`

	// ComponentID 组件ID
	ComponentID string `json:"component_id"`

	// ComponentName 组件名称
	ComponentName string `json:"component_name"`

	// Action 动作: enter, process, exit, error
	Action string `json:"action"`

	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`

	// Duration 处理耗时
	Duration time.Duration `json:"duration"`

	// Status 状态: success, error, skip
	Status string `json:"status"`

	// Error 错误信息
	Error string `json:"error,omitempty"`

	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// DataSnapshot 数据快照（可选，用于调试）
	DataSnapshot map[string]interface{} `json:"data_snapshot,omitempty"`
}

// TraceFilter 追踪过滤器
type TraceFilter struct {
	// PipelineID 管道ID过滤
	PipelineID *string

	// ComponentType 组件类型过滤
	ComponentType *string

	// ComponentID 组件ID过滤
	ComponentID *string

	// SampleID 样本ID过滤
	SampleID *string

	// Action 动作过滤
	Action *string

	// Status 状态过滤
	Status *string

	// StartTime 开始时间
	StartTime *time.Time

	// EndTime 结束时间
	EndTime *time.Time

	// Limit 限制数量
	Limit int

	// Offset 偏移量
	Offset int

	// OrderBy 排序字段
	OrderBy string

	// OrderDesc 是否降序
	OrderDesc bool
}

// TraceAction 追踪动作常量
const (
	ActionEnter   = "enter"   // 进入组件
	ActionProcess = "process" // 处理中
	ActionExit    = "exit"    // 退出组件
	ActionError   = "error"   // 错误
	ActionSkip    = "skip"    // 跳过
)

// TraceStatus 追踪状态常量
const (
	StatusSuccess = "success" // 成功
	StatusError   = "error"   // 错误
	StatusSkip    = "skip"    // 跳过
)

// ComponentType 组件类型常量
const (
	ComponentTypeSource    = "source"    // 数据源
	ComponentTypeProcessor = "processor" // 处理器
	ComponentTypeAnalyzer  = "analyzer"  // 分析器
	ComponentTypeSink      = "sink"      // 数据输出
	ComponentTypePipeline  = "pipeline"  // 管道
)

// TraceRepository 追踪数据仓储接口
type TraceRepository interface {
	// SaveTraceEvent 保存追踪事件
	SaveTraceEvent(ctx context.Context, event *TraceEvent) error

	// SaveTraceEventsBatch 批量保存追踪事件
	SaveTraceEventsBatch(ctx context.Context, events []*TraceEvent) error

	// QueryTraceEvents 查询追踪事件
	QueryTraceEvents(ctx context.Context, filter TraceFilter) ([]TraceEvent, error)

	// GetTraceBySampleID 根据样本ID获取完整追踪链路
	GetTraceBySampleID(ctx context.Context, sampleID string) ([]TraceEvent, error)

	// DeleteOldTraces 删除旧的追踪数据
	DeleteOldTraces(ctx context.Context, before time.Time) error

	// GetPipelineMetrics 获取管道性能指标
	GetPipelineMetrics(ctx context.Context, pipelineID string, start, end time.Time) (*PipelineMetrics, error)
}

// PipelineMetrics 管道性能指标
type PipelineMetrics struct {
	// PipelineID 管道ID
	PipelineID string `json:"pipeline_id"`

	// StartTime 统计开始时间
	StartTime time.Time `json:"start_time"`

	// EndTime 统计结束时间
	EndTime time.Time `json:"end_time"`

	// TotalSamples 总样本数
	TotalSamples int64 `json:"total_samples"`

	// SuccessSamples 成功处理的样本数
	SuccessSamples int64 `json:"success_samples"`

	// ErrorSamples 错误样本数
	ErrorSamples int64 `json:"error_samples"`

	// AvgDuration 平均处理耗时
	AvgDuration time.Duration `json:"avg_duration"`

	// MaxDuration 最大处理耗时
	MaxDuration time.Duration `json:"max_duration"`

	// MinDuration 最小处理耗时
	MinDuration time.Duration `json:"min_duration"`

	// ComponentMetrics 各组件性能指标
	ComponentMetrics map[string]*ComponentMetrics `json:"component_metrics"`
}

// ComponentMetrics 组件性能指标
type ComponentMetrics struct {
	// ComponentID 组件ID
	ComponentID string `json:"component_id"`

	// ComponentType 组件类型
	ComponentType string `json:"component_type"`

	// ComponentName 组件名称
	ComponentName string `json:"component_name"`

	// EventCount 事件数
	EventCount int64 `json:"event_count"`

	// AvgDuration 平均耗时
	AvgDuration time.Duration `json:"avg_duration"`

	// MaxDuration 最大耗时
	MaxDuration time.Duration `json:"max_duration"`

	// MinDuration 最小耗时
	MinDuration time.Duration `json:"min_duration"`

	// ErrorCount 错误数
	ErrorCount int64 `json:"error_count"`

	// ErrorRate 错误率
	ErrorRate float64 `json:"error_rate"`
}
