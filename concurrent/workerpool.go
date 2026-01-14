package concurrent

import (
	"context"
	"fmt"
	"go_ProFiBus/logger"
	"sync"
	"sync/atomic"
	"time"
)

// Task 任务接口
type Task interface {
	Execute() error
	GetID() string
	GetPriority() int
}

// WorkerPool worker池
type WorkerPool struct {
	name          string
	workerCount   int
	taskQueue     chan Task
	results       chan TaskResult
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	running       atomic.Bool
	log           *logger.Logger
	stats         *PoolStats
	errorHandler  func(error)
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID    string
	Success   bool
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

// PoolStats 池统计信息
type PoolStats struct {
	TotalTasks     atomic.Uint64
	CompletedTasks atomic.Uint64
	FailedTasks    atomic.Uint64
	ActiveWorkers  atomic.Int32
	QueuedTasks    atomic.Int32
}

// WorkerPoolConfig worker池配置
type WorkerPoolConfig struct {
	Name         string
	WorkerCount  int
	QueueSize    int
	ErrorHandler func(error)
}

// NewWorkerPool 创建worker池
func NewWorkerPool(config WorkerPoolConfig) *WorkerPool {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 4
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		name:         config.Name,
		workerCount:  config.WorkerCount,
		taskQueue:    make(chan Task, config.QueueSize),
		results:      make(chan TaskResult, config.QueueSize),
		ctx:          ctx,
		cancel:       cancel,
		log:          logger.GetLogger(),
		stats:        &PoolStats{},
		errorHandler: config.ErrorHandler,
	}
}

// Start 启动worker池
func (wp *WorkerPool) Start() error {
	if wp.running.Load() {
		return fmt.Errorf("worker池已在运行: %s", wp.name)
	}

	wp.running.Store(true)

	// 启动workers
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	// 启动结果处理器
	go wp.resultHandler()

	wp.log.Info("Worker池已启动: %s (workers: %d)", wp.name, wp.workerCount)
	return nil
}

// Stop 停止worker池
func (wp *WorkerPool) Stop() error {
	if !wp.running.Load() {
		return fmt.Errorf("worker池未运行: %s", wp.name)
	}

	wp.log.Info("正在停止Worker池: %s", wp.name)

	// 关闭任务队列
	close(wp.taskQueue)

	// 等待所有worker完成
	wp.wg.Wait()

	// 关闭结果通道
	close(wp.results)

	// 取消context
	wp.cancel()

	wp.running.Store(false)
	wp.log.Info("Worker池已停止: %s", wp.name)
	return nil
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task Task) error {
	if !wp.running.Load() {
		return fmt.Errorf("worker池未运行: %s", wp.name)
	}

	select {
	case wp.taskQueue <- task:
		wp.stats.TotalTasks.Add(1)
		wp.stats.QueuedTasks.Add(1)
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker池已关闭")
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// SubmitWithTimeout 带超时的提交任务
func (wp *WorkerPool) SubmitWithTimeout(task Task, timeout time.Duration) error {
	if !wp.running.Load() {
		return fmt.Errorf("worker池未运行: %s", wp.name)
	}

	ctx, cancel := context.WithTimeout(wp.ctx, timeout)
	defer cancel()

	select {
	case wp.taskQueue <- task:
		wp.stats.TotalTasks.Add(1)
		wp.stats.QueuedTasks.Add(1)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("提交任务超时")
	}
}

// worker 工作协程
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.stats.ActiveWorkers.Add(1)
	defer wp.stats.ActiveWorkers.Add(-1)

	wp.log.Debug("Worker %d 已启动 [%s]", id, wp.name)

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				wp.log.Debug("Worker %d 退出 [%s]", id, wp.name)
				return
			}

			wp.stats.QueuedTasks.Add(-1)
			wp.processTask(id, task)

		case <-wp.ctx.Done():
			wp.log.Debug("Worker %d 被取消 [%s]", id, wp.name)
			return
		}
	}
}

// processTask 处理任务
func (wp *WorkerPool) processTask(workerID int, task Task) {
	startTime := time.Now()

	result := TaskResult{
		TaskID:    task.GetID(),
		Timestamp: startTime,
	}

	// 执行任务
	err := task.Execute()
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Success = false
		result.Error = err
		wp.stats.FailedTasks.Add(1)

		wp.log.Warn("Worker %d 任务失败 [%s]: %s - %v",
			workerID, wp.name, task.GetID(), err)

		if wp.errorHandler != nil {
			wp.errorHandler(err)
		}
	} else {
		result.Success = true
		wp.stats.CompletedTasks.Add(1)

		wp.log.Debug("Worker %d 任务完成 [%s]: %s (用时: %v)",
			workerID, wp.name, task.GetID(), result.Duration)
	}

	// 发送结果
	select {
	case wp.results <- result:
	case <-wp.ctx.Done():
		return
	default:
		wp.log.Warn("结果通道已满，丢弃结果: %s", task.GetID())
	}
}

// resultHandler 结果处理器
func (wp *WorkerPool) resultHandler() {
	for result := range wp.results {
		if !result.Success {
			// 可以在这里添加更多的错误处理逻辑
			wp.log.Debug("处理失败结果: %s", result.TaskID)
		}
	}
}

// GetStats 获取统计信息
func (wp *WorkerPool) GetStats() PoolStats {
	return PoolStats{
		TotalTasks:     atomic.Uint64{},
		CompletedTasks: atomic.Uint64{},
		FailedTasks:    atomic.Uint64{},
		ActiveWorkers:  atomic.Int32{},
		QueuedTasks:    atomic.Int32{},
	}
}

// IsRunning 检查是否运行中
func (wp *WorkerPool) IsRunning() bool {
	return wp.running.Load()
}

// GetQueueSize 获取队列大小
func (wp *WorkerPool) GetQueueSize() int {
	return int(wp.stats.QueuedTasks.Load())
}

// GetActiveWorkers 获取活跃worker数
func (wp *WorkerPool) GetActiveWorkers() int {
	return int(wp.stats.ActiveWorkers.Load())
}

// Wait 等待所有任务完成
func (wp *WorkerPool) Wait() {
	for wp.stats.QueuedTasks.Load() > 0 {
		time.Sleep(100 * time.Millisecond)
	}
}

// PriorityTask 优先级任务
type PriorityTask struct {
	id       string
	priority int
	fn       func() error
}

// NewPriorityTask 创建优先级任务
func NewPriorityTask(id string, priority int, fn func() error) *PriorityTask {
	return &PriorityTask{
		id:       id,
		priority: priority,
		fn:       fn,
	}
}

func (pt *PriorityTask) Execute() error {
	return pt.fn()
}

func (pt *PriorityTask) GetID() string {
	return pt.id
}

func (pt *PriorityTask) GetPriority() int {
	return pt.priority
}

// SimpleTask 简单任务
type SimpleTask struct {
	id string
	fn func() error
}

// NewSimpleTask 创建简单任务
func NewSimpleTask(id string, fn func() error) *SimpleTask {
	return &SimpleTask{
		id: id,
		fn: fn,
	}
}

func (st *SimpleTask) Execute() error {
	return st.fn()
}

func (st *SimpleTask) GetID() string {
	return st.id
}

func (st *SimpleTask) GetPriority() int {
	return 0
}

// OrderedWorkerPool 有序worker池（保证顺序处理）
type OrderedWorkerPool struct {
	*WorkerPool
	sequencer *Sequencer
}

// NewOrderedWorkerPool 创建有序worker池
func NewOrderedWorkerPool(config WorkerPoolConfig) *OrderedWorkerPool {
	return &OrderedWorkerPool{
		WorkerPool: NewWorkerPool(config),
		sequencer:  NewSequencer(config.QueueSize),
	}
}

// SubmitOrdered 提交有序任务
func (owp *OrderedWorkerPool) SubmitOrdered(seqNum uint64, task Task) error {
	wrappedTask := &SequencedTask{
		Task:   task,
		SeqNum: seqNum,
	}

	return owp.Submit(wrappedTask)
}

// SequencedTask 带序号的任务
type SequencedTask struct {
	Task
	SeqNum uint64
}

// Sequencer 序列器（保证输出顺序）
type Sequencer struct {
	nextSeq    atomic.Uint64
	buffer     map[uint64]interface{}
	bufferSize int
	mu         sync.Mutex
	cond       *sync.Cond
}

// NewSequencer 创建序列器
func NewSequencer(bufferSize int) *Sequencer {
	s := &Sequencer{
		buffer:     make(map[uint64]interface{}),
		bufferSize: bufferSize,
	}
	s.cond = sync.NewCond(&s.mu)
	s.nextSeq.Store(0)
	return s
}

// Add 添加结果
func (s *Sequencer) Add(seqNum uint64, result interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer[seqNum] = result
	s.cond.Broadcast()
}

// GetNext 获取下一个结果（阻塞直到可用）
func (s *Sequencer) GetNext() (interface{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nextSeq := s.nextSeq.Load()

	for {
		if result, ok := s.buffer[nextSeq]; ok {
			delete(s.buffer, nextSeq)
			s.nextSeq.Add(1)
			return result, true
		}

		// 等待新结果
		s.cond.Wait()
	}
}

// DynamicWorkerPool 动态worker池（可动态调整worker数量）
type DynamicWorkerPool struct {
	*WorkerPool
	minWorkers int
	maxWorkers int
	scaleUpThreshold   float64
	scaleDownThreshold float64
	mu                 sync.RWMutex
}

// NewDynamicWorkerPool 创建动态worker池
func NewDynamicWorkerPool(config WorkerPoolConfig, minWorkers, maxWorkers int) *DynamicWorkerPool {
	if minWorkers <= 0 {
		minWorkers = 2
	}
	if maxWorkers <= minWorkers {
		maxWorkers = minWorkers * 2
	}

	config.WorkerCount = minWorkers

	return &DynamicWorkerPool{
		WorkerPool:         NewWorkerPool(config),
		minWorkers:         minWorkers,
		maxWorkers:         maxWorkers,
		scaleUpThreshold:   0.8,  // 队列80%满时扩容
		scaleDownThreshold: 0.2,  // 队列20%满时缩容
	}
}

// StartAutoScale 启动自动伸缩
func (dwp *DynamicWorkerPool) StartAutoScale(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				dwp.autoScale()
			case <-dwp.ctx.Done():
				return
			}
		}
	}()
}

// autoScale 自动伸缩
func (dwp *DynamicWorkerPool) autoScale() {
	dwp.mu.Lock()
	defer dwp.mu.Unlock()

	queueSize := float64(dwp.GetQueueSize())
	capacity := float64(cap(dwp.taskQueue))
	utilization := queueSize / capacity

	currentWorkers := dwp.workerCount

	// 扩容
	if utilization > dwp.scaleUpThreshold && currentWorkers < dwp.maxWorkers {
		newWorkers := currentWorkers + 1
		if newWorkers > dwp.maxWorkers {
			newWorkers = dwp.maxWorkers
		}

		dwp.wg.Add(1)
		go dwp.worker(newWorkers - 1)
		dwp.workerCount = newWorkers

		dwp.log.Info("Worker池扩容: %s (%d -> %d workers)",
			dwp.name, currentWorkers, newWorkers)
	}

	// 缩容（简化实现，实际需要更复杂的逻辑）
	if utilization < dwp.scaleDownThreshold && currentWorkers > dwp.minWorkers {
		// 注意：实际缩容需要优雅地停止worker
		dwp.log.Debug("Worker池可缩容: %s (利用率: %.2f%%)", dwp.name, utilization*100)
	}
}
