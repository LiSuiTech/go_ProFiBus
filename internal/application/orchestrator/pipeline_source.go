package orchestrator

import (
	"context"
	"go_ProFiBus/pkg/interfaces"
	"sync"
)

// PipelineDataSource 管道数据源
// 用于将一个管道的输出连接到另一个管道的输入
type PipelineDataSource struct {
	id        string
	name      string
	dataChan  chan interfaces.DataSample
	mu        sync.RWMutex
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewPipelineDataSource 创建管道数据源
func NewPipelineDataSource(id, name string) *PipelineDataSource {
	ctx, cancel := context.WithCancel(context.Background())
	return &PipelineDataSource{
		id:       id,
		name:     name,
		dataChan: make(chan interfaces.DataSample, 1000),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动数据源
func (pds *PipelineDataSource) Start(ctx context.Context) error {
	pds.mu.Lock()
	defer pds.mu.Unlock()

	if pds.running {
		return nil
	}

	pds.running = true
	return nil
}

// Stop 停止数据源
func (pds *PipelineDataSource) Stop() error {
	pds.mu.Lock()
	defer pds.mu.Unlock()

	if !pds.running {
		return nil
	}

	pds.running = false
	pds.cancel()
	close(pds.dataChan)
	return nil
}

// GetData 获取数据通道
func (pds *PipelineDataSource) GetData() <-chan interfaces.DataSample {
	return pds.dataChan
}

// Write 写入数据（供上游管道调用）
func (pds *PipelineDataSource) Write(sample interfaces.DataSample) error {
	pds.mu.RLock()
	defer pds.mu.RUnlock()

	if !pds.running {
		return nil // 如果未运行，忽略数据
	}

	select {
	case pds.dataChan <- sample:
		return nil
	case <-pds.ctx.Done():
		return pds.ctx.Err()
	default:
		// 通道已满，丢弃数据
		return nil
	}
}

// GetStatus 获取状态
func (pds *PipelineDataSource) GetStatus() interfaces.SourceStatus {
	pds.mu.RLock()
	defer pds.mu.RUnlock()

	return interfaces.SourceStatus{
		Running:          pds.running,
		Connected:        pds.running,
		SamplesCollected: 0, // PipelineDataSource 不统计样本数
	}
}

// GetID 获取ID
func (pds *PipelineDataSource) GetID() string {
	return pds.id
}

// GetName 获取名称
func (pds *PipelineDataSource) GetName() string {
	return pds.name
}

// 确保实现了 DataSource 接口
var _ interfaces.DataSource = (*PipelineDataSource)(nil)
