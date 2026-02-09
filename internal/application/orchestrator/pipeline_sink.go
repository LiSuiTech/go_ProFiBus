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
func (ps *PipelineSink) Write(ctx context.Context, sample interfaces.DataSample) error {
	return ps.source.Write(sample)
}

// Close 实现 DataSink 接口
func (ps *PipelineSink) Close() error {
	// PipelineSink 不需要关闭，由 PipelineDataSource 管理生命周期
	return nil
}

// GetName 实现 DataSink 接口
func (ps *PipelineSink) GetName() string {
	return ps.name
}

// 确保实现了 DataSink 接口
var _ interfaces.DataSink = (*PipelineSink)(nil)
