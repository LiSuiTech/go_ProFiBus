package orchestrator

import (
	"context"
	"go_ProFiBus/pkg/interfaces"
)

// PipelineSink 管道数据接收器
// 用于将管道输出连接到另一个管道的数据源
type PipelineSink struct {
	name   string
	source *PipelineDataSource
}

// NewPipelineSink 创建管道数据接收器
func NewPipelineSink(name string, source *PipelineDataSource) *PipelineSink {
	return &PipelineSink{
		name:   name,
		source: source,
	}
}

// Write 实现 DataSink 接口
func (ps *PipelineSink) Write(ctx context.Context, data interface{}) error {
	sample, ok := data.(interfaces.DataSample)
	if !ok {
		return nil
	}
	return ps.source.Write(sample)
}

// WriteBatch 实现 DataSink 接口
func (ps *PipelineSink) WriteBatch(ctx context.Context, data []interface{}) error {
	for _, d := range data {
		if err := ps.Write(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// Close 实现 DataSink 接口
func (ps *PipelineSink) Close() error {
	// PipelineSink 不需要关闭，由 PipelineDataSource 管理生命周期
	return nil
}

// Flush 实现 DataSink 接口（无缓冲，直接返回）
func (ps *PipelineSink) Flush(ctx context.Context) error {
	return nil
}

// GetName 实现 DataSink 接口
func (ps *PipelineSink) GetName() string {
	return ps.name
}

// 确保实现了 DataSink 接口
var _ interfaces.DataSink = (*PipelineSink)(nil)
