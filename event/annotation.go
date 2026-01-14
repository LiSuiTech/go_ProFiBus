package event

import (
	"encoding/json"
	"fmt"
	"go_ProFiBus/errors"
	"go_ProFiBus/logger"
	"os"
	"sync"
	"time"
)

// AnnotationWorkflow 标注工作流
type AnnotationWorkflow struct {
	eventManager *EventManager
	eventStore   *EventStore
	annotators   map[string]*Annotator
	reviewQueue  []*AnnotationTask
	mu           sync.RWMutex
	log          *logger.Logger
	stats        *WorkflowStats
}

// Annotator 标注员
type Annotator struct {
	ID          string
	Name        string
	Role        string
	Permissions []string
	CreatedAt   time.Time
}

// AnnotationTask 标注任务
type AnnotationTask struct {
	ID          string
	EventID     string
	AssignedTo  string
	Priority    int
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// WorkflowStats 工作流统计
type WorkflowStats struct {
	TotalTasks      uint64
	PendingTasks    uint64
	CompletedTasks  uint64
	ConfirmedEvents uint64
	RejectedEvents  uint64
	mu              sync.RWMutex
}

// NewAnnotationWorkflow 创建标注工作流
func NewAnnotationWorkflow(eventManager *EventManager, eventStore *EventStore) *AnnotationWorkflow {
	return &AnnotationWorkflow{
		eventManager: eventManager,
		eventStore:   eventStore,
		annotators:   make(map[string]*Annotator),
		reviewQueue:  make([]*AnnotationTask, 0),
		log:          logger.GetLogger(),
		stats:        &WorkflowStats{},
	}
}

// RegisterAnnotator 注册标注员
func (aw *AnnotationWorkflow) RegisterAnnotator(annotator *Annotator) error {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	if _, exists := aw.annotators[annotator.ID]; exists {
		return errors.Newf(errors.ErrInvalidParam, "标注员已存在: %s", annotator.ID)
	}

	aw.annotators[annotator.ID] = annotator
	aw.log.Info("注册标注员: %s (%s)", annotator.Name, annotator.ID)
	return nil
}

// CreateAnnotationTask 创建标注任务
func (aw *AnnotationWorkflow) CreateAnnotationTask(eventID, assignedTo string, priority int) (*AnnotationTask, error) {
	// 检查事件是否存在
	_, err := aw.eventManager.GetEvent(eventID)
	if err != nil {
		return nil, err
	}

	// 检查标注员是否存在
	aw.mu.RLock()
	if _, exists := aw.annotators[assignedTo]; !exists {
		aw.mu.RUnlock()
		return nil, errors.Newf(errors.ErrInvalidParam, "标注员不存在: %s", assignedTo)
	}
	aw.mu.RUnlock()

	task := &AnnotationTask{
		ID:         fmt.Sprintf("task_%d", time.Now().UnixNano()),
		EventID:    eventID,
		AssignedTo: assignedTo,
		Priority:   priority,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	aw.mu.Lock()
	aw.reviewQueue = append(aw.reviewQueue, task)
	aw.mu.Unlock()

	aw.stats.mu.Lock()
	aw.stats.TotalTasks++
	aw.stats.PendingTasks++
	aw.stats.mu.Unlock()

	aw.log.Info("创建标注任务: %s (事件: %s, 分配给: %s)", task.ID, eventID, assignedTo)
	return task, nil
}

// AnnotateEvent 标注事件
func (aw *AnnotationWorkflow) AnnotateEvent(
	eventID string,
	annotatorID string,
	confirmed bool,
	annotation string,
	labels map[string]interface{},
) error {
	// 检查标注员权限
	aw.mu.RLock()
	annotator, exists := aw.annotators[annotatorID]
	aw.mu.RUnlock()

	if !exists {
		return errors.Newf(errors.ErrInvalidParam, "标注员不存在: %s", annotatorID)
	}

	// 获取事件
	event, err := aw.eventManager.GetEvent(eventID)
	if err != nil {
		return err
	}

	// 更新事件状态
	var status EventStatus
	if confirmed {
		status = EventStatusConfirmed
	} else {
		status = EventStatusRejected
	}

	err = aw.eventManager.UpdateEventStatus(eventID, status, annotatorID, annotation)
	if err != nil {
		return err
	}

	// 添加标签到元数据
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["labels"] = labels
	event.Metadata["annotator_name"] = annotator.Name

	// 如果确认，保存到事件库
	if confirmed {
		err = aw.eventStore.SaveEvent(event)
		if err != nil {
			aw.log.Error("保存事件到事件库失败: %v", err)
			return err
		}

		aw.stats.mu.Lock()
		aw.stats.ConfirmedEvents++
		aw.stats.mu.Unlock()

		aw.log.Info("事件已确认并保存到事件库: %s", eventID)
	} else {
		aw.stats.mu.Lock()
		aw.stats.RejectedEvents++
		aw.stats.mu.Unlock()

		aw.log.Info("事件已拒绝: %s", eventID)
	}

	// 更新任务状态
	aw.completeTask(eventID)

	return nil
}

// completeTask 完成任务
func (aw *AnnotationWorkflow) completeTask(eventID string) {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	newQueue := make([]*AnnotationTask, 0)
	for _, task := range aw.reviewQueue {
		if task.EventID == eventID {
			task.Status = "completed"
			now := time.Now()
			task.CompletedAt = &now

			aw.stats.mu.Lock()
			aw.stats.PendingTasks--
			aw.stats.CompletedTasks++
			aw.stats.mu.Unlock()
		} else {
			newQueue = append(newQueue, task)
		}
	}
	aw.reviewQueue = newQueue
}

// GetPendingTasks 获取待处理任务
func (aw *AnnotationWorkflow) GetPendingTasks(annotatorID string) []*AnnotationTask {
	aw.mu.RLock()
	defer aw.mu.RUnlock()

	tasks := make([]*AnnotationTask, 0)
	for _, task := range aw.reviewQueue {
		if task.AssignedTo == annotatorID && task.Status == "pending" {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetAllPendingTasks 获取所有待处理任务
func (aw *AnnotationWorkflow) GetAllPendingTasks() []*AnnotationTask {
	aw.mu.RLock()
	defer aw.mu.RUnlock()

	tasks := make([]*AnnotationTask, 0)
	for _, task := range aw.reviewQueue {
		if task.Status == "pending" {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetStats 获取统计信息
func (aw *AnnotationWorkflow) GetStats() WorkflowStats {
	aw.stats.mu.RLock()
	defer aw.stats.mu.RUnlock()
	return *aw.stats
}

// EventStore 事件库
type EventStore struct {
	storePath      string
	events         map[string]*Event
	eventsByType   map[EventType][]*Event
	mu             sync.RWMutex
	log            *logger.Logger
	autoSave       bool
	stats          *StoreStats
}

// StoreStats 事件库统计
type StoreStats struct {
	TotalEvents     uint64
	EventsByType    map[EventType]uint64
	EventsBySeverity map[string]uint64
	mu              sync.RWMutex
}

// NewEventStore 创建事件库
func NewEventStore(storePath string, autoSave bool) *EventStore {
	return &EventStore{
		storePath:    storePath,
		events:       make(map[string]*Event),
		eventsByType: make(map[EventType][]*Event),
		log:          logger.GetLogger(),
		autoSave:     autoSave,
		stats: &StoreStats{
			EventsByType:     make(map[EventType]uint64),
			EventsBySeverity: make(map[string]uint64),
		},
	}
}

// SaveEvent 保存事件
func (es *EventStore) SaveEvent(event *Event) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	// 只保存已确认的事件
	if event.Status != EventStatusConfirmed {
		return errors.Newf(errors.ErrInvalidParam, "只能保存已确认的事件")
	}

	es.events[event.ID] = event

	// 按类型索引
	es.eventsByType[event.Type] = append(es.eventsByType[event.Type], event)

	// 更新统计
	es.stats.mu.Lock()
	es.stats.TotalEvents++
	es.stats.EventsByType[event.Type]++
	severityKey := fmt.Sprintf("%d", event.Severity)
	es.stats.EventsBySeverity[severityKey]++
	es.stats.mu.Unlock()

	// 自动保存到文件
	if es.autoSave {
		go es.saveToFile(event)
	}

	es.log.Info("事件已保存到事件库: %s", event.ID)
	return nil
}

// saveToFile 保存事件到文件
func (es *EventStore) saveToFile(event *Event) error {
	if es.storePath == "" {
		return nil
	}

	// 确保目录存在
	os.MkdirAll(es.storePath, 0755)

	// 按日期组织文件
	date := event.Timestamp.Format("2006-01-02")
	filename := fmt.Sprintf("%s/events_%s.jsonl", es.storePath, date)

	// 以追加模式打开文件
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 序列化事件
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// 写入一行JSON
	_, err = fmt.Fprintf(file, "%s\n", data)
	return err
}

// GetEvent 获取事件
func (es *EventStore) GetEvent(eventID string) (*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	event, exists := es.events[eventID]
	if !exists {
		return nil, errors.Newf(errors.ErrInvalidParam, "事件不存在: %s", eventID)
	}

	return event, nil
}

// GetEventsByType 按类型获取事件
func (es *EventStore) GetEventsByType(eventType EventType) []*Event {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events := es.eventsByType[eventType]
	result := make([]*Event, len(events))
	copy(result, events)
	return result
}

// GetEventsByTimeRange 按时间范围获取事件
func (es *EventStore) GetEventsByTimeRange(start, end time.Time) []*Event {
	es.mu.RLock()
	defer es.mu.RUnlock()

	result := make([]*Event, 0)
	for _, event := range es.events {
		if event.Timestamp.After(start) && event.Timestamp.Before(end) {
			result = append(result, event)
		}
	}
	return result
}

// GetSimilarEvents 获取相似事件
func (es *EventStore) GetSimilarEvents(event *Event, threshold float64, limit int) []*Event {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// 简化实现：基于特征相似度
	if event.ContextData == nil || len(event.ContextData.Features) == 0 {
		return nil
	}

	calculator := NewSimilarityCalculator(MetricCosine)
	type scoredEvent struct {
		event      *Event
		similarity float64
	}

	scored := make([]scoredEvent, 0)
	for _, storedEvent := range es.events {
		if storedEvent.ID == event.ID {
			continue
		}

		if storedEvent.ContextData == nil || len(storedEvent.ContextData.Features) == 0 {
			continue
		}

		similarity := calculator.Calculate(event.ContextData.Features, storedEvent.ContextData.Features)
		if similarity >= threshold {
			scored = append(scored, scoredEvent{
				event:      storedEvent,
				similarity: similarity,
			})
		}
	}

	// 按相似度排序
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].similarity > scored[i].similarity {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// 限制数量
	if len(scored) > limit {
		scored = scored[:limit]
	}

	result := make([]*Event, len(scored))
	for i, se := range scored {
		result[i] = se.event
	}

	return result
}

// GetAllEvents 获取所有事件
func (es *EventStore) GetAllEvents() []*Event {
	es.mu.RLock()
	defer es.mu.RUnlock()

	result := make([]*Event, 0, len(es.events))
	for _, event := range es.events {
		result = append(result, event)
	}
	return result
}

// GetStats 获取统计信息
func (es *EventStore) GetStats() StoreStats {
	es.stats.mu.RLock()
	defer es.stats.mu.RUnlock()
	return *es.stats
}

// LoadFromFile 从文件加载事件
func (es *EventStore) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	count := 0

	for decoder.More() {
		var event Event
		if err := decoder.Decode(&event); err != nil {
			es.log.Warn("解析事件失败: %v", err)
			continue
		}

		es.mu.Lock()
		es.events[event.ID] = &event
		es.eventsByType[event.Type] = append(es.eventsByType[event.Type], &event)
		es.mu.Unlock()

		count++
	}

	es.log.Info("从文件加载 %d 个事件: %s", count, filename)
	return nil
}

// NewSimilarityCalculator 创建相似度计算器（从anomaly包导入）
func NewSimilarityCalculator(metric SimilarityMetric) *SimilarityCalculator {
	return &SimilarityCalculator{
		metric: metric,
	}
}

// SimilarityMetric 相似度度量方法
type SimilarityMetric int

const (
	MetricCosine SimilarityMetric = 1
)

// SimilarityCalculator 相似度计算器
type SimilarityCalculator struct {
	metric SimilarityMetric
}

// Calculate 计算相似度
func (sc *SimilarityCalculator) Calculate(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	similarity := dotProduct / (normA * normB)
	return (similarity + 1.0) / 2.0
}
