package event

import (
	"time"

	"go_ProFiBus/pkg/interfaces"
	"github.com/google/uuid"
)

// Event 事件实体
type Event struct {
	id          string
	eventType   string
	status      string
	timestamp   time.Time
	severity    int
	score       float64
	description string
	metadata    map[string]interface{}
}

// EventStatus 事件状态
const (
	StatusNew        = "new"
	StatusProcessing = "processing"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
)

// EventType 事件类型
const (
	TypeAnomaly     = "anomaly"
	TypeThreshold   = "threshold"
	TypePattern     = "pattern"
	TypeCorrelation = "correlation"
	TypeSystem      = "system"
)

// NewEvent 创建新事件
func NewEvent(eventType string, severity int, description string) *Event {
	return &Event{
		id:          uuid.New().String(),
		eventType:   eventType,
		status:      StatusNew,
		timestamp:   time.Now(),
		severity:    severity,
		score:       0.0,
		description: description,
		metadata:    make(map[string]interface{}),
	}
}

// NewEventFromAnalysisResult 从分析结果创建事件
func NewEventFromAnalysisResult(result interfaces.AnalysisResult) *Event {
	return &Event{
		id:          uuid.New().String(),
		eventType:   result.Type,
		status:      StatusNew,
		timestamp:   result.Timestamp,
		severity:    result.Severity,
		score:       result.Score,
		description: result.Message,
		metadata:    result.Details,
	}
}

// GetID 实现 Event 接口
func (e *Event) GetID() string {
	return e.id
}

// GetType 实现 Event 接口
func (e *Event) GetType() string {
	return e.eventType
}

// GetStatus 实现 Event 接口
func (e *Event) GetStatus() string {
	return e.status
}

// GetTimestamp 实现 Event 接口
func (e *Event) GetTimestamp() time.Time {
	return e.timestamp
}

// GetSeverity 实现 Event 接口
func (e *Event) GetSeverity() int {
	return e.severity
}

// GetScore 实现 Event 接口
func (e *Event) GetScore() float64 {
	return e.score
}

// GetDescription 实现 Event 接口
func (e *Event) GetDescription() string {
	return e.description
}

// GetMetadata 实现 Event 接口
func (e *Event) GetMetadata() map[string]interface{} {
	return e.metadata
}

// SetStatus 设置事件状态
func (e *Event) SetStatus(status string) {
	e.status = status
}

// SetMetadata 设置元数据
func (e *Event) SetMetadata(key string, value interface{}) {
	if e.metadata == nil {
		e.metadata = make(map[string]interface{})
	}
	e.metadata[key] = value
}

// UpdateScore 更新分数
func (e *Event) UpdateScore(score float64) {
	e.score = score
}

// IsResolved 是否已解决
func (e *Event) IsResolved() bool {
	return e.status == StatusResolved || e.status == StatusClosed
}

// IsCritical 是否为严重事件
func (e *Event) IsCritical() bool {
	return e.severity >= 4
}

// 确保 Event 实现了 interfaces.Event 接口
var _ interfaces.Event = (*Event)(nil)
