package collector

import (
	"context"
	"go_ProFiBus/errors"
	"go_ProFiBus/logger"
	"sync"
	"time"
)

// MultiCollector 多源数据采集器
type MultiCollector struct {
	collectors map[string]*Collector
	mu         sync.RWMutex
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
	dataChan   chan *DataSample
	log        *logger.Logger
}

// NewMultiCollector 创建多源数据采集器
func NewMultiCollector(bufferSize int) *MultiCollector {
	if bufferSize == 0 {
		bufferSize = 10000
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &MultiCollector{
		collectors: make(map[string]*Collector),
		ctx:        ctx,
		cancel:     cancel,
		dataChan:   make(chan *DataSample, bufferSize),
		log:        logger.GetLogger(),
	}
}

// AddCollector 添加数据采集器
func (mc *MultiCollector) AddCollector(sourceID string, collector *Collector) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return errors.Newf(errors.ErrInvalidParam, "无法在运行时添加采集器")
	}

	if _, exists := mc.collectors[sourceID]; exists {
		return errors.Newf(errors.ErrInvalidParam, "采集器已存在: %s", sourceID)
	}

	mc.collectors[sourceID] = collector
	mc.log.Info("已添加采集器: %s", sourceID)
	return nil
}

// RemoveCollector 移除数据采集器
func (mc *MultiCollector) RemoveCollector(sourceID string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return errors.Newf(errors.ErrInvalidParam, "无法在运行时移除采集器")
	}

	if _, exists := mc.collectors[sourceID]; !exists {
		return errors.Newf(errors.ErrInvalidParam, "采集器不存在: %s", sourceID)
	}

	delete(mc.collectors, sourceID)
	mc.log.Info("已移除采集器: %s", sourceID)
	return nil
}

// Start 启动所有采集器
func (mc *MultiCollector) Start() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return errors.New(errors.ErrCollectorAlreadyRunning, nil)
	}

	if len(mc.collectors) == 0 {
		return errors.Newf(errors.ErrInvalidParam, "没有可用的采集器")
	}

	// 启动所有采集器
	for sourceID, collector := range mc.collectors {
		if err := collector.Start(); err != nil {
			mc.log.Error("启动采集器失败 %s: %v", sourceID, err)
			return err
		}

		// 启动数据转发协程
		go mc.forwardData(collector)
	}

	mc.running = true
	mc.log.Info("多源数据采集器已启动，共 %d 个数据源", len(mc.collectors))
	return nil
}

// Stop 停止所有采集器
func (mc *MultiCollector) Stop() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return errors.New(errors.ErrCollectorNotRunning, nil)
	}

	// 停止所有采集器
	for sourceID, collector := range mc.collectors {
		if err := collector.Stop(); err != nil {
			mc.log.Error("停止采集器失败 %s: %v", sourceID, err)
		}
	}

	mc.running = false
	mc.cancel()
	close(mc.dataChan)

	mc.log.Info("多源数据采集器已停止")
	return nil
}

// forwardData 转发单个采集器的数据到统一通道
func (mc *MultiCollector) forwardData(collector *Collector) {
	dataChan := collector.GetDataChannel()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case sample, ok := <-dataChan:
			if !ok {
				return
			}
			select {
			case mc.dataChan <- sample:
			case <-mc.ctx.Done():
				return
			default:
				mc.log.Warn("多源数据缓冲区已满，丢弃样本")
			}
		}
	}
}

// GetDataChannel 获取统一的数据通道
func (mc *MultiCollector) GetDataChannel() <-chan *DataSample {
	return mc.dataChan
}

// IsRunning 检查是否正在运行
func (mc *MultiCollector) IsRunning() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.running
}

// GetCollectorStats 获取所有采集器的统计信息
func (mc *MultiCollector) GetCollectorStats() map[string]CollectorStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats := make(map[string]CollectorStats)
	for sourceID, collector := range mc.collectors {
		stats[sourceID] = collector.GetStats()
	}
	return stats
}

// GetCollector 获取指定的采集器
func (mc *MultiCollector) GetCollector(sourceID string) (*Collector, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	collector, exists := mc.collectors[sourceID]
	return collector, exists
}

// GetAllCollectors 获取所有采集器
func (mc *MultiCollector) GetAllCollectors() map[string]*Collector {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*Collector)
	for k, v := range mc.collectors {
		result[k] = v
	}
	return result
}

// AggregateStats 聚合统计信息
type AggregateStats struct {
	TotalSources   int
	TotalSamples   uint64
	SuccessSamples uint64
	FailedSamples  uint64
	SuccessRate    float64
	Uptime         time.Duration
	SourceStats    map[string]CollectorStats
}

// GetAggregateStats 获取聚合统计信息
func (mc *MultiCollector) GetAggregateStats() *AggregateStats {
	stats := mc.GetCollectorStats()

	agg := &AggregateStats{
		TotalSources: len(stats),
		SourceStats:  stats,
	}

	var oldestStartTime time.Time
	for _, stat := range stats {
		agg.TotalSamples += stat.TotalSamples
		agg.SuccessSamples += stat.SuccessSamples
		agg.FailedSamples += stat.FailedSamples

		if oldestStartTime.IsZero() || stat.StartTime.Before(oldestStartTime) {
			oldestStartTime = stat.StartTime
		}
	}

	if agg.TotalSamples > 0 {
		agg.SuccessRate = float64(agg.SuccessSamples) / float64(agg.TotalSamples)
	}

	if !oldestStartTime.IsZero() {
		agg.Uptime = time.Since(oldestStartTime)
	}

	return agg
}
