package concurrent

import (
	"context"
	"fmt"
	"go_ProFiBus/collector"
	"go_ProFiBus/logger"
	"sort"
	"sync"
	"time"
)

// PartitionedOrderedCollector 分区有序采集器
// 每个数据源独立维护顺序，支持并发处理但保证源内数据顺序
type PartitionedOrderedCollector struct {
	partitions map[string]*OrderedBuffer
	workerPool *WorkerPool
	merger     *TimestampMerger
	output     chan *collector.DataSample
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	log        *logger.Logger
	running    bool
}

// OrderedBuffer 有序缓冲区（单个数据源）
type OrderedBuffer struct {
	sourceID      string
	buffer        []*collector.DataSample
	windowSize    time.Duration
	maxBufferSize int
	mu            sync.RWMutex
	cond          *sync.Cond
}

// TimestampMerger 时间戳归并器（多源归并排序）
type TimestampMerger struct {
	sources map[string]chan *collector.DataSample
	output  chan *collector.DataSample
	mu      sync.RWMutex
}

// PartitionedCollectorConfig 分区采集器配置
type PartitionedCollectorConfig struct {
	WindowSize     time.Duration // 时间窗口大小（默认5秒）
	MaxBufferSize  int           // 每个分区最大缓冲数（默认1000）
	WorkerCount    int           // Worker数量
	OutputChanSize int           // 输出通道大小
}

// NewPartitionedOrderedCollector 创建分区有序采集器
func NewPartitionedOrderedCollector(config PartitionedCollectorConfig) *PartitionedOrderedCollector {
	if config.WindowSize == 0 {
		config.WindowSize = 5 * time.Second
	}
	if config.MaxBufferSize == 0 {
		config.MaxBufferSize = 1000
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 4
	}
	if config.OutputChanSize == 0 {
		config.OutputChanSize = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())

	poc := &PartitionedOrderedCollector{
		partitions: make(map[string]*OrderedBuffer),
		output:     make(chan *collector.DataSample, config.OutputChanSize),
		ctx:        ctx,
		cancel:     cancel,
		log:        logger.GetLogger(),
	}

	// 创建worker池（用于并发处理）
	poc.workerPool = NewWorkerPool(WorkerPoolConfig{
		Name:        "OrderedCollector",
		WorkerCount: config.WorkerCount,
		QueueSize:   config.OutputChanSize,
	})

	// 创建归并器
	poc.merger = NewTimestampMerger(config.OutputChanSize)

	return poc
}

// Start 启动采集器
func (poc *PartitionedOrderedCollector) Start() error {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	if poc.running {
		return fmt.Errorf("采集器已在运行")
	}

	// 启动worker池
	if err := poc.workerPool.Start(); err != nil {
		return err
	}

	// 启动归并器
	poc.merger.Start(poc.ctx, poc.output)

	poc.running = true
	poc.log.Info("分区有序采集器已启动")
	return nil
}

// Stop 停止采集器
func (poc *PartitionedOrderedCollector) Stop() error {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	if !poc.running {
		return fmt.Errorf("采集器未运行")
	}

	// 停止worker池
	if err := poc.workerPool.Stop(); err != nil {
		poc.log.Warn("停止worker池失败: %v", err)
	}

	// 停止归并器
	poc.merger.Stop()

	// 取消context
	poc.cancel()

	// 关闭输出通道
	close(poc.output)

	poc.running = false
	poc.log.Info("分区有序采集器已停止")
	return nil
}

// Submit 提交数据样本
func (poc *PartitionedOrderedCollector) Submit(sample *collector.DataSample) error {
	if sample == nil {
		return fmt.Errorf("样本为空")
	}

	sourceID := sample.SourceID
	if sourceID == "" {
		return fmt.Errorf("样本缺少SourceID")
	}

	// 获取或创建分区缓冲区
	buffer := poc.getOrCreateBuffer(sourceID)

	// 添加到缓冲区（保证源内有序）
	buffer.Add(sample)

	return nil
}

// GetOutput 获取输出通道
func (poc *PartitionedOrderedCollector) GetOutput() <-chan *collector.DataSample {
	return poc.output
}

// getOrCreateBuffer 获取或创建缓冲区
func (poc *PartitionedOrderedCollector) getOrCreateBuffer(sourceID string) *OrderedBuffer {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	buffer, exists := poc.partitions[sourceID]
	if !exists {
		buffer = NewOrderedBuffer(sourceID, 5*time.Second, 1000)
		poc.partitions[sourceID] = buffer

		// 为新分区创建输出通道
		partitionOutput := make(chan *collector.DataSample, 100)
		poc.merger.AddSource(sourceID, partitionOutput)

		// 启动分区的刷新goroutine
		go poc.flushBufferPeriodically(buffer, partitionOutput)

		poc.log.Debug("创建新分区: %s", sourceID)
	}

	return buffer
}

// flushBufferPeriodically 定期刷新缓冲区
func (poc *PartitionedOrderedCollector) flushBufferPeriodically(buffer *OrderedBuffer, output chan *collector.DataSample) {
	ticker := time.NewTicker(buffer.windowSize / 2) // 每半个窗口刷新一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			samples := buffer.Flush()
			for _, sample := range samples {
				select {
				case output <- sample:
				case <-poc.ctx.Done():
					return
				}
			}
		case <-poc.ctx.Done():
			return
		}
	}
}

// GetStats 获取统计信息
func (poc *PartitionedOrderedCollector) GetStats() map[string]interface{} {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["partition_count"] = len(poc.partitions)
	stats["running"] = poc.running

	partitionStats := make(map[string]int)
	for sourceID, buffer := range poc.partitions {
		partitionStats[sourceID] = buffer.Size()
	}
	stats["partition_sizes"] = partitionStats

	return stats
}

// ===== OrderedBuffer 实现 =====

// NewOrderedBuffer 创建有序缓冲区
func NewOrderedBuffer(sourceID string, windowSize time.Duration, maxSize int) *OrderedBuffer {
	ob := &OrderedBuffer{
		sourceID:      sourceID,
		buffer:        make([]*collector.DataSample, 0, maxSize),
		windowSize:    windowSize,
		maxBufferSize: maxSize,
	}
	ob.cond = sync.NewCond(&ob.mu)
	return ob
}

// Add 添加样本到缓冲区
func (ob *OrderedBuffer) Add(sample *collector.DataSample) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.buffer = append(ob.buffer, sample)

	// 如果超过最大大小，强制刷新最旧的一半
	if len(ob.buffer) >= ob.maxBufferSize {
		// 排序（按时间戳）
		sort.Slice(ob.buffer, func(i, j int) bool {
			return ob.buffer[i].Timestamp.Before(ob.buffer[j].Timestamp)
		})
		// 移除前半部分（最旧的）
		ob.buffer = ob.buffer[ob.maxBufferSize/2:]
	}

	ob.cond.Signal()
}

// Flush 刷新缓冲区（返回超出时间窗口的样本）
func (ob *OrderedBuffer) Flush() []*collector.DataSample {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if len(ob.buffer) == 0 {
		return nil
	}

	now := time.Now()
	cutoff := now.Add(-ob.windowSize)

	// 排序
	sort.Slice(ob.buffer, func(i, j int) bool {
		return ob.buffer[i].Timestamp.Before(ob.buffer[j].Timestamp)
	})

	// 找到cutoff点
	cutIndex := 0
	for i, sample := range ob.buffer {
		if sample.Timestamp.After(cutoff) {
			cutIndex = i
			break
		}
	}

	if cutIndex == 0 {
		return nil
	}

	// 提取要刷新的样本
	flushed := make([]*collector.DataSample, cutIndex)
	copy(flushed, ob.buffer[:cutIndex])

	// 保留窗口内的样本
	ob.buffer = ob.buffer[cutIndex:]

	return flushed
}

// Size 返回缓冲区大小
func (ob *OrderedBuffer) Size() int {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return len(ob.buffer)
}

// ===== TimestampMerger 实现 =====

// NewTimestampMerger 创建时间戳归并器
func NewTimestampMerger(bufferSize int) *TimestampMerger {
	return &TimestampMerger{
		sources: make(map[string]chan *collector.DataSample),
	}
}

// AddSource 添加数据源
func (tm *TimestampMerger) AddSource(sourceID string, input chan *collector.DataSample) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.sources[sourceID] = input
}

// Start 启动归并
func (tm *TimestampMerger) Start(ctx context.Context, output chan *collector.DataSample) {
	go tm.merge(ctx, output)
}

// Stop 停止归并
func (tm *TimestampMerger) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, ch := range tm.sources {
		close(ch)
	}
}

// merge 归并多个源的数据（简化版，使用时间戳排序）
func (tm *TimestampMerger) merge(ctx context.Context, output chan *collector.DataSample) {
	// 使用优先级队列进行K路归并
	// 简化实现：定期收集所有源的数据并排序输出

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	buffer := make([]*collector.DataSample, 0, 100)

	for {
		select {
		case <-ticker.C:
			// 从所有源收集数据
			tm.mu.RLock()
			for _, sourceChan := range tm.sources {
				// 非阻塞读取
				for {
					select {
					case sample, ok := <-sourceChan:
						if ok {
							buffer = append(buffer, sample)
						}
					default:
						goto nextSource
					}
				}
			nextSource:
			}
			tm.mu.RUnlock()

			// 排序
			if len(buffer) > 0 {
				sort.Slice(buffer, func(i, j int) bool {
					return buffer[i].Timestamp.Before(buffer[j].Timestamp)
				})

				// 输出
				for _, sample := range buffer {
					select {
					case output <- sample:
					case <-ctx.Done():
						return
					}
				}

				// 清空缓冲
				buffer = buffer[:0]
			}

		case <-ctx.Done():
			return
		}
	}
}
