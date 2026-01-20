// Package event provides legacy event management functionality.
//
// Deprecated: This package is part of the legacy architecture.
// New code should use:
//   - internal/domain/event for event entities
//   - pkg/interfaces/repository.go for event storage interfaces
package event

import (
	"encoding/json"
	"fmt"
	"go_ProFiBus/anomaly"
	"go_ProFiBus/collector"
	"go_ProFiBus/errors"
	"go_ProFiBus/logger"
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventTypeAnomaly     EventType = "anomaly"      // 异常事件
	EventTypeThreshold   EventType = "threshold"    // 阈值事件
	EventTypePattern     EventType = "pattern"      // 模式事件
	EventTypeCustom      EventType = "custom"       // 自定义事件
)

// EventStatus 事件状态
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"    // 待处理
	EventStatusConfirmed EventStatus = "confirmed"  // 已确认
	EventStatusRejected  EventStatus = "rejected"   // 已拒绝
	EventStatusIgnored   EventStatus = "ignored"    // 已忽略
)

// Event 事件
type Event struct {
	ID            string                 `json:"id"`
	Type          EventType              `json:"type"`
	Status        EventStatus            `json:"status"`
	Severity      anomaly.Severity       `json:"severity"`
	Timestamp     time.Time              `json:"timestamp"`
	SourceID      string                 `json:"source_id"`
	RuleID        string                 `json:"rule_id,omitempty"`
	Score         float64                `json:"score"`
	Similarity    float64                `json:"similarity,omitempty"`
	Description   string                 `json:"description"`
	ContextData   *EventContext          `json:"context_data"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	AnnotatedBy   string                 `json:"annotated_by,omitempty"`
	AnnotatedAt   *time.Time             `json:"annotated_at,omitempty"`
	Annotation    string                 `json:"annotation,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// EventContext 事件上下文数据
type EventContext struct {
	TriggerSample  *collector.DataSample   `json:"trigger_sample"`
	BeforeSamples  []*collector.DataSample `json:"before_samples"`
	AfterSamples   []*collector.DataSample `json:"after_samples"`
	WindowDuration time.Duration           `json:"window_duration"`
	Features       []float64               `json:"features,omitempty"`
}

// EventDetector 事件检测器
type EventDetector struct {
	ruleEngine       *anomaly.RuleEngine
	patternMatcher   *anomaly.PatternMatcher
	threshold        float64
	contextWindow    time.Duration
	dataBuffer       *CircularBuffer
	eventListeners   []EventListener
	mu               sync.RWMutex
	log              *logger.Logger
	stats            *DetectorStats
}

// DetectorStats 检测器统计
type DetectorStats struct {
	TotalDetections  uint64
	PendingEvents    uint64
	ConfirmedEvents  uint64
	RejectedEvents   uint64
	mu               sync.RWMutex
}

// EventListener 事件监听器
type EventListener interface {
	OnEventDetected(event *Event) error
}

// NewEventDetector 创建事件检测器
func NewEventDetector(
	ruleEngine *anomaly.RuleEngine,
	patternMatcher *anomaly.PatternMatcher,
	threshold float64,
	contextWindow time.Duration,
) *EventDetector {
	return &EventDetector{
		ruleEngine:     ruleEngine,
		patternMatcher: patternMatcher,
		threshold:      threshold,
		contextWindow:  contextWindow,
		dataBuffer:     NewCircularBuffer(1000),
		eventListeners: make([]EventListener, 0),
		log:            logger.GetLogger(),
		stats:          &DetectorStats{},
	}
}

// AddListener 添加事件监听器
func (ed *EventDetector) AddListener(listener EventListener) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	ed.eventListeners = append(ed.eventListeners, listener)
}

// Detect 检测事件
func (ed *EventDetector) Detect(sample *collector.DataSample) ([]*Event, error) {
	// 将样本添加到缓冲区
	ed.dataBuffer.Add(sample)

	events := make([]*Event, 0)

	// 1. 基于规则的检测
	ruleViolations := ed.ruleEngine.Evaluate(sample)
	for _, violation := range ruleViolations {
		if violation.Score >= ed.threshold {
			event := ed.createEventFromViolation(violation)
			events = append(events, event)
		}
	}

	// 2. 基于模式的检测（如果配置了模式匹配器）
	if ed.patternMatcher != nil {
		// 提取最近的时间序列数据
		recentSamples := ed.dataBuffer.GetRecent(50)
		if len(recentSamples) >= 10 {
			timeseries := extractTimeSeriesData(recentSamples, "value")
			pattern := anomaly.NormalizeTimeSeries(timeseries)

			// 这里可以与已知模式进行匹配
			// 简化示例：检测突变
			if ed.detectSuddenChange(timeseries) {
				event := ed.createPatternEvent(sample, timeseries, "sudden_change")
				events = append(events, event)
			}
		}
	}

	// 更新统计
	ed.stats.mu.Lock()
	ed.stats.TotalDetections += uint64(len(events))
	ed.stats.PendingEvents += uint64(len(events))
	ed.stats.mu.Unlock()

	// 通知监听器
	for _, event := range events {
		ed.notifyListeners(event)
	}

	return events, nil
}

// createEventFromViolation 从规则违规创建事件
func (ed *EventDetector) createEventFromViolation(violation *anomaly.EvaluationResult) *Event {
	event := &Event{
		ID:          generateEventID(),
		Type:        EventTypeAnomaly,
		Status:      EventStatusPending,
		Severity:    violation.Severity,
		Timestamp:   violation.Sample.Timestamp,
		SourceID:    violation.Sample.SourceID,
		RuleID:      violation.RuleID,
		Score:       violation.Score,
		Description: violation.Message,
		ContextData: ed.extractContext(violation.Sample),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	event.Metadata["rule_name"] = violation.RuleName

	return event
}

// createPatternEvent 创建模式事件
func (ed *EventDetector) createPatternEvent(sample *collector.DataSample, timeseries []float64, patternType string) *Event {
	event := &Event{
		ID:          generateEventID(),
		Type:        EventTypePattern,
		Status:      EventStatusPending,
		Severity:    anomaly.SeverityWarning,
		Timestamp:   sample.Timestamp,
		SourceID:    sample.SourceID,
		Score:       0.8,
		Description: fmt.Sprintf("检测到模式: %s", patternType),
		ContextData: ed.extractContext(sample),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	event.Metadata["pattern_type"] = patternType
	event.ContextData.Features = anomaly.ExtractFeatures(timeseries)

	return event
}

// extractContext 提取事件上下文
func (ed *EventDetector) extractContext(triggerSample *collector.DataSample) *EventContext {
	allSamples := ed.dataBuffer.GetAll()

	beforeSamples := make([]*collector.DataSample, 0)
	afterSamples := make([]*collector.DataSample, 0)

	for _, sample := range allSamples {
		timeDiff := sample.Timestamp.Sub(triggerSample.Timestamp)

		if timeDiff < 0 && timeDiff >= -ed.contextWindow {
			beforeSamples = append(beforeSamples, sample)
		} else if timeDiff > 0 && timeDiff <= ed.contextWindow {
			afterSamples = append(afterSamples, sample)
		}
	}

	return &EventContext{
		TriggerSample:  triggerSample,
		BeforeSamples:  beforeSamples,
		AfterSamples:   afterSamples,
		WindowDuration: ed.contextWindow,
	}
}

// detectSuddenChange 检测突变
func (ed *EventDetector) detectSuddenChange(data []float64) bool {
	if len(data) < 5 {
		return false
	}

	// 计算最近几个点的变化率
	recentChanges := make([]float64, 0)
	for i := len(data) - 4; i < len(data); i++ {
		change := data[i] - data[i-1]
		recentChanges = append(recentChanges, change)
	}

	// 计算平均变化率
	avgChange := 0.0
	for _, c := range recentChanges {
		avgChange += c
	}
	avgChange /= float64(len(recentChanges))

	// 如果最近的变化率显著大于平均，认为是突变
	lastChange := recentChanges[len(recentChanges)-1]
	return lastChange > 3*avgChange || lastChange < -3*avgChange
}

// notifyListeners 通知监听器
func (ed *EventDetector) notifyListeners(event *Event) {
	ed.mu.RLock()
	listeners := make([]EventListener, len(ed.eventListeners))
	copy(listeners, ed.eventListeners)
	ed.mu.RUnlock()

	for _, listener := range listeners {
		go func(l EventListener) {
			if err := l.OnEventDetected(event); err != nil {
				ed.log.Error("事件监听器处理失败: %v", err)
			}
		}(listener)
	}
}

// GetStats 获取统计信息
func (ed *EventDetector) GetStats() DetectorStats {
	ed.stats.mu.RLock()
	defer ed.stats.mu.RUnlock()
	return *ed.stats
}

// CircularBuffer 循环缓冲区
type CircularBuffer struct {
	data  []*collector.DataSample
	size  int
	index int
	count int
	mu    sync.RWMutex
}

// NewCircularBuffer 创建循环缓冲区
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		data: make([]*collector.DataSample, size),
		size: size,
	}
}

// Add 添加样本
func (cb *CircularBuffer) Add(sample *collector.DataSample) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.data[cb.index] = sample
	cb.index = (cb.index + 1) % cb.size

	if cb.count < cb.size {
		cb.count++
	}
}

// GetRecent 获取最近的N个样本
func (cb *CircularBuffer) GetRecent(n int) []*collector.DataSample {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if n > cb.count {
		n = cb.count
	}

	result := make([]*collector.DataSample, 0, n)
	for i := 0; i < n; i++ {
		idx := (cb.index - 1 - i + cb.size) % cb.size
		if cb.data[idx] != nil {
			result = append(result, cb.data[idx])
		}
	}

	// 反转以保持时间顺序
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// GetAll 获取所有样本
func (cb *CircularBuffer) GetAll() []*collector.DataSample {
	return cb.GetRecent(cb.count)
}

// extractTimeSeriesData 从样本中提取时间序列数据
func extractTimeSeriesData(samples []*collector.DataSample, fieldName string) []float64 {
	data := make([]float64, 0, len(samples))
	for _, sample := range samples {
		if value, ok := sample.ParsedData[fieldName].(float64); ok {
			data = append(data, value)
		}
	}
	return data
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// EventManager 事件管理器
type EventManager struct {
	events       map[string]*Event
	pendingQueue []*Event
	mu           sync.RWMutex
	log          *logger.Logger
}

// NewEventManager 创建事件管理器
func NewEventManager() *EventManager {
	return &EventManager{
		events:       make(map[string]*Event),
		pendingQueue: make([]*Event, 0),
		log:          logger.GetLogger(),
	}
}

// AddEvent 添加事件
func (em *EventManager) AddEvent(event *Event) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.events[event.ID]; exists {
		return errors.Newf(errors.ErrInvalidParam, "事件已存在: %s", event.ID)
	}

	em.events[event.ID] = event

	if event.Status == EventStatusPending {
		em.pendingQueue = append(em.pendingQueue, event)
	}

	em.log.Info("添加事件: %s (%s)", event.ID, event.Description)
	return nil
}

// GetEvent 获取事件
func (em *EventManager) GetEvent(eventID string) (*Event, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	event, exists := em.events[eventID]
	if !exists {
		return nil, errors.Newf(errors.ErrInvalidParam, "事件不存在: %s", eventID)
	}

	return event, nil
}

// GetPendingEvents 获取待处理事件
func (em *EventManager) GetPendingEvents() []*Event {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]*Event, len(em.pendingQueue))
	copy(result, em.pendingQueue)
	return result
}

// UpdateEventStatus 更新事件状态
func (em *EventManager) UpdateEventStatus(eventID string, status EventStatus, annotatedBy, annotation string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	event, exists := em.events[eventID]
	if !exists {
		return errors.Newf(errors.ErrInvalidParam, "事件不存在: %s", eventID)
	}

	event.Status = status
	event.AnnotatedBy = annotatedBy
	event.Annotation = annotation
	now := time.Now()
	event.AnnotatedAt = &now
	event.UpdatedAt = now

	// 从待处理队列中移除
	if status != EventStatusPending {
		em.removePendingEvent(eventID)
	}

	em.log.Info("更新事件状态: %s -> %s (标注人: %s)", eventID, status, annotatedBy)
	return nil
}

// removePendingEvent 从待处理队列中移除事件
func (em *EventManager) removePendingEvent(eventID string) {
	newQueue := make([]*Event, 0, len(em.pendingQueue))
	for _, event := range em.pendingQueue {
		if event.ID != eventID {
			newQueue = append(newQueue, event)
		}
	}
	em.pendingQueue = newQueue
}

// GetEventsByStatus 按状态获取事件
func (em *EventManager) GetEventsByStatus(status EventStatus) []*Event {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]*Event, 0)
	for _, event := range em.events {
		if event.Status == status {
			result = append(result, event)
		}
	}
	return result
}

// ExportEvent 导出事件（JSON格式）
func (em *EventManager) ExportEvent(eventID string) ([]byte, error) {
	event, err := em.GetEvent(eventID)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(event, "", "  ")
}

// ListEvents 列出所有事件
func (em *EventManager) ListEvents() []*Event {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]*Event, 0, len(em.events))
	for _, event := range em.events {
		result = append(result, event)
	}
	return result
}
