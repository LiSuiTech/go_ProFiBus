package orchestrator

import (
	"fmt"

	"go_ProFiBus/pkg/interfaces"
)

// PipelineBuilder Pipeline 构建器
type PipelineBuilder struct {
	name       string
	source     interfaces.DataSource
	processors []interfaces.Processor
	analyzers  []interfaces.Analyzer
	sinks      []interfaces.DataSink
	errors     []error
}

// NewPipelineBuilder 创建管道构建器
func NewPipelineBuilder(name string) *PipelineBuilder {
	return &PipelineBuilder{
		name:       name,
		processors: make([]interfaces.Processor, 0),
		analyzers:  make([]interfaces.Analyzer, 0),
		sinks:      make([]interfaces.DataSink, 0),
		errors:     make([]error, 0),
	}
}

// WithSource 设置数据源
func (b *PipelineBuilder) WithSource(source interfaces.DataSource) *PipelineBuilder {
	if source == nil {
		b.errors = append(b.errors, fmt.Errorf("source cannot be nil"))
		return b
	}
	b.source = source
	return b
}

// WithProcessor 添加处理器
func (b *PipelineBuilder) WithProcessor(processor interfaces.Processor) *PipelineBuilder {
	if processor == nil {
		b.errors = append(b.errors, fmt.Errorf("processor cannot be nil"))
		return b
	}
	b.processors = append(b.processors, processor)
	return b
}

// WithProcessors 批量添加处理器
func (b *PipelineBuilder) WithProcessors(processors ...interfaces.Processor) *PipelineBuilder {
	for _, processor := range processors {
		b.WithProcessor(processor)
	}
	return b
}

// WithAnalyzer 添加分析器
func (b *PipelineBuilder) WithAnalyzer(analyzer interfaces.Analyzer) *PipelineBuilder {
	if analyzer == nil {
		b.errors = append(b.errors, fmt.Errorf("analyzer cannot be nil"))
		return b
	}
	b.analyzers = append(b.analyzers, analyzer)
	return b
}

// WithAnalyzers 批量添加分析器
func (b *PipelineBuilder) WithAnalyzers(analyzers ...interfaces.Analyzer) *PipelineBuilder {
	for _, analyzer := range analyzers {
		b.WithAnalyzer(analyzer)
	}
	return b
}

// WithSink 添加数据输出
func (b *PipelineBuilder) WithSink(sink interfaces.DataSink) *PipelineBuilder {
	if sink == nil {
		b.errors = append(b.errors, fmt.Errorf("sink cannot be nil"))
		return b
	}
	b.sinks = append(b.sinks, sink)
	return b
}

// WithSinks 批量添加数据输出
func (b *PipelineBuilder) WithSinks(sinks ...interfaces.DataSink) *PipelineBuilder {
	for _, sink := range sinks {
		b.WithSink(sink)
	}
	return b
}

// Build 构建管道
func (b *PipelineBuilder) Build() (*Pipeline, error) {
	// 检查是否有错误
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("build errors: %v", b.errors)
	}

	// 检查必填字段
	if b.name == "" {
		return nil, fmt.Errorf("pipeline name is required")
	}
	if b.source == nil {
		return nil, fmt.Errorf("data source is required")
	}

	// 创建管道
	pipeline := NewPipeline(b.name, b.source)

	// 添加处理器
	for _, processor := range b.processors {
		pipeline.AddProcessor(processor)
	}

	// 添加分析器
	for _, analyzer := range b.analyzers {
		pipeline.AddAnalyzer(analyzer)
	}

	// 添加输出
	for _, sink := range b.sinks {
		pipeline.AddSink(sink)
	}

	return pipeline, nil
}

// Validate 验证配置
func (b *PipelineBuilder) Validate() error {
	if len(b.errors) > 0 {
		return fmt.Errorf("validation errors: %v", b.errors)
	}

	if b.name == "" {
		return fmt.Errorf("pipeline name is required")
	}

	if b.source == nil {
		return fmt.Errorf("data source is required")
	}

	// 至少需要一个输出
	if len(b.sinks) == 0 {
		return fmt.Errorf("at least one sink is required")
	}

	return nil
}

// Reset 重置构建器
func (b *PipelineBuilder) Reset() *PipelineBuilder {
	b.source = nil
	b.processors = make([]interfaces.Processor, 0)
	b.analyzers = make([]interfaces.Analyzer, 0)
	b.sinks = make([]interfaces.DataSink, 0)
	b.errors = make([]error, 0)
	return b
}

// TopologyBuilder 拓扑构建器
// 用于构建复杂的数据流拓扑（多管道协同）
type TopologyBuilder struct {
	pipelines []*Pipeline
	edges     []Edge
	errors    []error
}

// Edge 管道间的连接边
type Edge struct {
	FromPipeline string
	ToPipeline   string
	DataType     string
}

// NewTopologyBuilder 创建拓扑构建器
func NewTopologyBuilder() *TopologyBuilder {
	return &TopologyBuilder{
		pipelines: make([]*Pipeline, 0),
		edges:     make([]Edge, 0),
		errors:    make([]error, 0),
	}
}

// AddPipeline 添加管道
func (tb *TopologyBuilder) AddPipeline(pipeline *Pipeline) *TopologyBuilder {
	if pipeline == nil {
		tb.errors = append(tb.errors, fmt.Errorf("pipeline cannot be nil"))
		return tb
	}
	tb.pipelines = append(tb.pipelines, pipeline)
	return tb
}

// Connect 连接两个管道
func (tb *TopologyBuilder) Connect(fromPipeline, toPipeline, dataType string) *TopologyBuilder {
	tb.edges = append(tb.edges, Edge{
		FromPipeline: fromPipeline,
		ToPipeline:   toPipeline,
		DataType:     dataType,
	})
	return tb
}

// Build 构建拓扑
func (tb *TopologyBuilder) Build() (*Orchestrator, error) {
	if len(tb.errors) > 0 {
		return nil, fmt.Errorf("topology build errors: %v", tb.errors)
	}

	orchestrator := NewOrchestrator()

	// 添加所有管道
	for _, pipeline := range tb.pipelines {
		if err := orchestrator.AddPipeline(pipeline); err != nil {
			return nil, fmt.Errorf("failed to add pipeline %s: %w", pipeline.name, err)
		}
	}

	// TODO: 实现管道间的数据流连接
	// 这需要实现一个管道输出到另一个管道输入的机制

	return orchestrator, nil
}
