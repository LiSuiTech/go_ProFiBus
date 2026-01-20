// Package collector provides legacy data collection functionality.
//
// Deprecated: This package is part of the legacy architecture.
// New code should use:
//   - pkg/interfaces/datasource.go for data source interfaces
//   - internal/domain/datasample for data sample entities
//   - internal/infrastructure/collector for adapter implementations
package collector

import (
	"context"
	"go_ProFiBus/errors"
	"go_ProFiBus/logger"
	"go_ProFiBus/serial"
	"sync"
	"time"
)

// DataSample 数据样本
type DataSample struct {
	Timestamp  time.Time              // 时间戳
	SourceID   string                 // 数据源ID
	Protocol   string                 // 协议类型
	RawData    []byte                 // 原始数据
	ParsedData map[string]interface{} // 解析后的数据
	Quality    float64                // 数据质量分数 (0-1)
}

// DataHandler 数据处理函数
type DataHandler func(*DataSample) error

// CollectorConfig 数据采集器配置
type CollectorConfig struct {
	SourceID     string              // 数据源ID
	Port         serial.SerialPort   // 串口设备
	SampleRate   time.Duration       // 采样率
	BufferSize   int                 // 缓冲区大小
	EnableCache  bool                // 是否启用缓存
	CacheSize    int                 // 缓存大小
	RetryCount   int                 // 重试次数
	RetryDelay   time.Duration       // 重试延迟
	Handler      DataHandler         // 数据处理函数
	ErrorHandler func(error)         // 错误处理函数
}

// Collector 数据采集器
type Collector struct {
	config   *CollectorConfig
	running  bool
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	dataChan chan *DataSample
	cache    []*DataSample
	cacheMu  sync.RWMutex
	stats    *CollectorStats
	log      *logger.Logger
}

// CollectorStats 采集统计
type CollectorStats struct {
	TotalSamples   uint64    // 总采样数
	SuccessSamples uint64    // 成功采样数
	FailedSamples  uint64    // 失败采样数
	LastSampleTime time.Time // 最后采样时间
	StartTime      time.Time // 启动时间
	mu             sync.RWMutex
}

// NewCollector 创建新的数据采集器
func NewCollector(config *CollectorConfig) *Collector {
	if config.SampleRate == 0 {
		config.SampleRate = 100 * time.Millisecond // 默认10Hz
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	if config.CacheSize == 0 {
		config.CacheSize = 100
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 100 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Collector{
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
		dataChan: make(chan *DataSample, config.BufferSize),
		cache:    make([]*DataSample, 0, config.CacheSize),
		stats:    &CollectorStats{},
		log:      logger.GetLogger(),
	}
}

// Start 启动数据采集
func (c *Collector) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return errors.New(errors.ErrCollectorAlreadyRunning, nil)
	}

	c.running = true
	c.stats.StartTime = time.Now()

	// 启动采集协程
	go c.collectLoop()

	// 启动处理协程
	if c.config.Handler != nil {
		go c.processLoop()
	}

	c.log.Info("数据采集器已启动: %s, 采样率: %v", c.config.SourceID, c.config.SampleRate)
	return nil
}

// Stop 停止数据采集
func (c *Collector) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return errors.New(errors.ErrCollectorNotRunning, nil)
	}

	c.running = false
	c.cancel()
	close(c.dataChan)

	c.log.Info("数据采集器已停止: %s", c.config.SourceID)
	return nil
}

// IsRunning 检查是否正在运行
func (c *Collector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// collectLoop 采集循环
func (c *Collector) collectLoop() {
	ticker := time.NewTicker(c.config.SampleRate)
	defer ticker.Stop()

	buffer := make([]byte, 4096)

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			sample := c.collectSample(buffer)
			if sample != nil {
				select {
				case c.dataChan <- sample:
					c.updateStats(true)
				default:
					c.log.Warn("数据缓冲区已满，丢弃样本")
					c.updateStats(false)
				}
			}
		}
	}
}

// collectSample 采集单个样本
func (c *Collector) collectSample(buffer []byte) *DataSample {
	var lastErr error

	for i := 0; i < c.config.RetryCount; i++ {
		n, err := c.config.Port.Read(buffer)
		if err != nil {
			lastErr = err
			c.log.Debug("读取失败 (尝试 %d/%d): %v", i+1, c.config.RetryCount, err)
			time.Sleep(c.config.RetryDelay)
			continue
		}

		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])

			sample := &DataSample{
				Timestamp:  time.Now(),
				SourceID:   c.config.SourceID,
				Protocol:   "unknown", // 可以根据实际情况设置
				RawData:    data,
				ParsedData: make(map[string]interface{}),
				Quality:    1.0,
			}

			// 添加到缓存
			if c.config.EnableCache {
				c.addToCache(sample)
			}

			return sample
		}
	}

	if lastErr != nil && c.config.ErrorHandler != nil {
		c.config.ErrorHandler(lastErr)
	}

	c.updateStats(false)
	return nil
}

// processLoop 处理循环
func (c *Collector) processLoop() {
	for sample := range c.dataChan {
		if err := c.config.Handler(sample); err != nil {
			c.log.Error("处理数据样本失败: %v", err)
			if c.config.ErrorHandler != nil {
				c.config.ErrorHandler(err)
			}
		}
	}
}

// addToCache 添加到缓存
func (c *Collector) addToCache(sample *DataSample) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	if len(c.cache) >= c.config.CacheSize {
		// 移除最旧的样本
		c.cache = c.cache[1:]
	}
	c.cache = append(c.cache, sample)
}

// GetCache 获取缓存的样本
func (c *Collector) GetCache() []*DataSample {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	result := make([]*DataSample, len(c.cache))
	copy(result, c.cache)
	return result
}

// ClearCache 清空缓存
func (c *Collector) ClearCache() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache = c.cache[:0]
}

// updateStats 更新统计信息
func (c *Collector) updateStats(success bool) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	c.stats.TotalSamples++
	if success {
		c.stats.SuccessSamples++
	} else {
		c.stats.FailedSamples++
	}
	c.stats.LastSampleTime = time.Now()
}

// GetStats 获取统计信息
func (c *Collector) GetStats() CollectorStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return *c.stats
}

// GetDataChannel 获取数据通道（用于外部消费）
func (c *Collector) GetDataChannel() <-chan *DataSample {
	return c.dataChan
}
