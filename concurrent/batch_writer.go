package concurrent

import (
	"context"
	"fmt"
	"go_ProFiBus/logger"
	"sync"
	"sync/atomic"
	"time"
)

// BatchWriter 批量写入器
// 批量累积数据并定期或达到阈值时批量写入
type BatchWriter struct {
	name          string
	batchSize     int
	flushInterval time.Duration
	buffer        []interface{}
	writeFn       BatchWriteFunc
	workerPool    *WorkerPool
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	running       atomic.Bool
	log           *logger.Logger
	stats         *BatchWriterStats
}

// BatchWriteFunc 批量写入函数类型
type BatchWriteFunc func(items []interface{}) error

// BatchWriterConfig 批量写入器配置
type BatchWriterConfig struct {
	Name          string
	BatchSize     int           // 批量大小（100-1000）
	FlushInterval time.Duration // 刷新间隔（默认5秒）
	WorkerCount   int           // Worker数量
	WriteFn       BatchWriteFunc
}

// BatchWriterStats 批量写入器统计
type BatchWriterStats struct {
	TotalItems    atomic.Uint64
	TotalBatches  atomic.Uint64
	TotalWrites   atomic.Uint64
	FailedWrites  atomic.Uint64
	BufferedItems atomic.Int32
	LastFlush     time.Time
	mu            sync.RWMutex
}

// NewBatchWriter 创建批量写入器
func NewBatchWriter(config BatchWriterConfig) *BatchWriter {
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 2
	}
	if config.WriteFn == nil {
		panic("WriteFn不能为空")
	}

	ctx, cancel := context.WithCancel(context.Background())

	bw := &BatchWriter{
		name:          config.Name,
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		buffer:        make([]interface{}, 0, config.BatchSize),
		writeFn:       config.WriteFn,
		ctx:           ctx,
		cancel:        cancel,
		log:           logger.GetLogger(),
		stats:         &BatchWriterStats{},
	}

	// 创建worker池用于异步写入
	bw.workerPool = NewWorkerPool(WorkerPoolConfig{
		Name:        fmt.Sprintf("BatchWriter_%s", config.Name),
		WorkerCount: config.WorkerCount,
		QueueSize:   100,
		ErrorHandler: func(err error) {
			bw.log.Error("批量写入失败: %v", err)
			bw.stats.FailedWrites.Add(1)
		},
	})

	return bw
}

// Start 启动批量写入器
func (bw *BatchWriter) Start() error {
	if bw.running.Load() {
		return fmt.Errorf("批量写入器已在运行: %s", bw.name)
	}

	// 启动worker池
	if err := bw.workerPool.Start(); err != nil {
		return err
	}

	bw.running.Store(true)

	// 启动定期刷新
	go bw.periodicFlush()

	bw.log.Info("批量写入器已启动: %s (批量大小: %d, 刷新间隔: %v)",
		bw.name, bw.batchSize, bw.flushInterval)
	return nil
}

// Stop 停止批量写入器
func (bw *BatchWriter) Stop() error {
	if !bw.running.Load() {
		return fmt.Errorf("批量写入器未运行: %s", bw.name)
	}

	bw.log.Info("正在停止批量写入器: %s", bw.name)

	// 刷新剩余数据
	if err := bw.Flush(); err != nil {
		bw.log.Warn("最终刷新失败: %v", err)
	}

	// 取消context
	bw.cancel()

	// 等待worker池完成
	if err := bw.workerPool.Stop(); err != nil {
		bw.log.Warn("停止worker池失败: %v", err)
	}

	bw.running.Store(false)
	bw.log.Info("批量写入器已停止: %s", bw.name)
	return nil
}

// Write 写入单个项目
func (bw *BatchWriter) Write(item interface{}) error {
	if !bw.running.Load() {
		return fmt.Errorf("批量写入器未运行")
	}

	bw.mu.Lock()
	defer bw.mu.Unlock()

	bw.buffer = append(bw.buffer, item)
	bw.stats.TotalItems.Add(1)
	bw.stats.BufferedItems.Store(int32(len(bw.buffer)))

	// 如果达到批量大小，立即刷新
	if len(bw.buffer) >= bw.batchSize {
		return bw.flushLocked()
	}

	return nil
}

// WriteBatch 写入批量项目
func (bw *BatchWriter) WriteBatch(items []interface{}) error {
	if !bw.running.Load() {
		return fmt.Errorf("批量写入器未运行")
	}

	bw.mu.Lock()
	defer bw.mu.Unlock()

	bw.buffer = append(bw.buffer, items...)
	bw.stats.TotalItems.Add(uint64(len(items)))
	bw.stats.BufferedItems.Store(int32(len(bw.buffer)))

	// 如果达到批量大小，立即刷新
	if len(bw.buffer) >= bw.batchSize {
		return bw.flushLocked()
	}

	return nil
}

// Flush 强制刷新缓冲区
func (bw *BatchWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.flushLocked()
}

// flushLocked 刷新缓冲区（需要持有锁）
func (bw *BatchWriter) flushLocked() error {
	if len(bw.buffer) == 0 {
		return nil
	}

	// 复制缓冲区数据
	items := make([]interface{}, len(bw.buffer))
	copy(items, bw.buffer)

	// 清空缓冲区
	bw.buffer = bw.buffer[:0]
	bw.stats.BufferedItems.Store(0)
	bw.stats.TotalBatches.Add(1)

	// 提交写入任务到worker池
	task := NewSimpleTask(
		fmt.Sprintf("batch_%d", time.Now().UnixNano()),
		func() error {
			return bw.writeBatch(items)
		},
	)

	err := bw.workerPool.Submit(task)
	if err != nil {
		bw.log.Error("提交写入任务失败: %v", err)
		return err
	}

	bw.stats.mu.Lock()
	bw.stats.LastFlush = time.Now()
	bw.stats.mu.Unlock()

	bw.log.Debug("刷新批量数据: %d 项", len(items))
	return nil
}

// writeBatch 执行批量写入
func (bw *BatchWriter) writeBatch(items []interface{}) error {
	startTime := time.Now()

	err := bw.writeFn(items)
	if err != nil {
		bw.stats.FailedWrites.Add(1)
		return fmt.Errorf("批量写入失败: %w", err)
	}

	bw.stats.TotalWrites.Add(1)
	duration := time.Since(startTime)

	bw.log.Debug("批量写入成功: %d 项 (用时: %v)", len(items), duration)
	return nil
}

// periodicFlush 定期刷新
func (bw *BatchWriter) periodicFlush() {
	ticker := time.NewTicker(bw.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := bw.Flush(); err != nil {
				bw.log.Warn("定期刷新失败: %v", err)
			}
		case <-bw.ctx.Done():
			return
		}
	}
}

// GetStats 获取统计信息
func (bw *BatchWriter) GetStats() BatchWriterStats {
	bw.stats.mu.RLock()
	defer bw.stats.mu.RUnlock()

	return BatchWriterStats{
		TotalItems:    atomic.Uint64{},
		TotalBatches:  atomic.Uint64{},
		TotalWrites:   atomic.Uint64{},
		FailedWrites:  atomic.Uint64{},
		BufferedItems: atomic.Int32{},
		LastFlush:     bw.stats.LastFlush,
	}
}

// GetBufferSize 获取当前缓冲区大小
func (bw *BatchWriter) GetBufferSize() int {
	return int(bw.stats.BufferedItems.Load())
}

// IsRunning 检查是否运行中
func (bw *BatchWriter) IsRunning() bool {
	return bw.running.Load()
}

// MultiTypeBatchWriter 多类型批量写入器
// 支持不同类型的数据分别批量写入
type MultiTypeBatchWriter struct {
	writers map[string]*BatchWriter
	mu      sync.RWMutex
	log     *logger.Logger
}

// NewMultiTypeBatchWriter 创建多类型批量写入器
func NewMultiTypeBatchWriter() *MultiTypeBatchWriter {
	return &MultiTypeBatchWriter{
		writers: make(map[string]*BatchWriter),
		log:     logger.GetLogger(),
	}
}

// RegisterWriter 注册写入器
func (mtbw *MultiTypeBatchWriter) RegisterWriter(typeName string, writer *BatchWriter) {
	mtbw.mu.Lock()
	defer mtbw.mu.Unlock()
	mtbw.writers[typeName] = writer
}

// Write 写入数据到指定类型的写入器
func (mtbw *MultiTypeBatchWriter) Write(typeName string, item interface{}) error {
	mtbw.mu.RLock()
	writer, exists := mtbw.writers[typeName]
	mtbw.mu.RUnlock()

	if !exists {
		return fmt.Errorf("未找到写入器: %s", typeName)
	}

	return writer.Write(item)
}

// StartAll 启动所有写入器
func (mtbw *MultiTypeBatchWriter) StartAll() error {
	mtbw.mu.RLock()
	defer mtbw.mu.RUnlock()

	for name, writer := range mtbw.writers {
		if err := writer.Start(); err != nil {
			mtbw.log.Error("启动写入器失败 [%s]: %v", name, err)
			return err
		}
	}

	mtbw.log.Info("所有批量写入器已启动 (共 %d 个)", len(mtbw.writers))
	return nil
}

// StopAll 停止所有写入器
func (mtbw *MultiTypeBatchWriter) StopAll() error {
	mtbw.mu.RLock()
	defer mtbw.mu.RUnlock()

	for name, writer := range mtbw.writers {
		if err := writer.Stop(); err != nil {
			mtbw.log.Warn("停止写入器失败 [%s]: %v", name, err)
		}
	}

	mtbw.log.Info("所有批量写入器已停止")
	return nil
}

// FlushAll 刷新所有写入器
func (mtbw *MultiTypeBatchWriter) FlushAll() error {
	mtbw.mu.RLock()
	defer mtbw.mu.RUnlock()

	for name, writer := range mtbw.writers {
		if err := writer.Flush(); err != nil {
			mtbw.log.Warn("刷新写入器失败 [%s]: %v", name, err)
		}
	}

	return nil
}

// GetAllStats 获取所有写入器统计
func (mtbw *MultiTypeBatchWriter) GetAllStats() map[string]BatchWriterStats {
	mtbw.mu.RLock()
	defer mtbw.mu.RUnlock()

	stats := make(map[string]BatchWriterStats)
	for name, writer := range mtbw.writers {
		stats[name] = writer.GetStats()
	}

	return stats
}
