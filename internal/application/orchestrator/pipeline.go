package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// Pipeline 数据处理管道
type Pipeline struct {
	name             string
	source           interfaces.DataSource
	processors       []interfaces.Processor
	analyzers        []interfaces.Analyzer
	sinks            []interfaces.DataSink
	errorChan        chan error
	stopChan         chan struct{}
	wg               sync.WaitGroup
	running          bool
	mu               sync.RWMutex
	statsMu          sync.RWMutex
	stopOnce         sync.Once
	samplesProcessed int64
	errors           int64
	lastSampleTime   time.Time
}

// NewPipeline 创建新的数据管道
func NewPipeline(name string, source interfaces.DataSource) *Pipeline {
	return &Pipeline{
		name:            name,
		source:          source,
		processors:       make([]interfaces.Processor, 0),
		analyzers:       make([]interfaces.Analyzer, 0),
		sinks:           make([]interfaces.DataSink, 0),
		errorChan:       make(chan error, 100),
		stopChan:        make(chan struct{}),
		samplesProcessed: 0,
		errors:          0,
	}
}

// AddProcessor 添加处理器
func (p *Pipeline) AddProcessor(processor interfaces.Processor) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		p.errorChan <- fmt.Errorf("cannot add processor while pipeline is running")
		return p
	}

	p.processors = append(p.processors, processor)
	return p
}

// AddAnalyzer 添加分析器
func (p *Pipeline) AddAnalyzer(analyzer interfaces.Analyzer) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		p.errorChan <- fmt.Errorf("cannot add analyzer while pipeline is running")
		return p
	}

	p.analyzers = append(p.analyzers, analyzer)
	return p
}

// AddSink 添加数据输出
func (p *Pipeline) AddSink(sink interfaces.DataSink) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		p.errorChan <- fmt.Errorf("cannot add sink while pipeline is running")
		return p
	}

	p.sinks = append(p.sinks, sink)
	return p
}

// Start 启动管道
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pipeline %s is already running", p.name)
	}
	p.running = true
	p.mu.Unlock()

	// 启动数据源
	if err := p.source.Start(ctx); err != nil {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		return fmt.Errorf("failed to start source: %w", err)
	}

	// 启动数据处理循环
	p.wg.Add(1)
	go p.run(ctx)

	return nil
}

// Stop 停止管道（可安全多次调用）。停止后 GetErrors() 返回的 channel 会被关闭。
func (p *Pipeline) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	var stopErr error
	p.stopOnce.Do(func() {
		close(p.stopChan)
		p.wg.Wait()
		// 关闭错误通道，使 MonitorErrors 等监听方能正常退出
		close(p.errorChan)

		if err := p.source.Stop(); err != nil {
			stopErr = fmt.Errorf("failed to stop source: %w", err)
		}
		for _, sink := range p.sinks {
			if err := sink.Close(); err != nil {
				if stopErr == nil {
					stopErr = fmt.Errorf("failed to close sink %s: %w", sink.GetName(), err)
				}
			}
		}

		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	})
	return stopErr
}

// run 运行数据处理循环
func (p *Pipeline) run(ctx context.Context) {
	defer p.wg.Done()

	dataChan := p.source.GetData()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case sample, ok := <-dataChan:
			if !ok {
				return
			}

			// 处理数据
			if err := p.processSample(ctx, sample); err != nil {
				p.errorChan <- fmt.Errorf("failed to process sample: %w", err)
			}
		}
	}
}

// processSample 处理单个数据样本
func (p *Pipeline) processSample(ctx context.Context, sample interfaces.DataSample) error {
	// 更新统计：增加样本计数和最后样本时间
	p.statsMu.Lock()
	p.samplesProcessed++
	p.lastSampleTime = sample.GetTimestamp()
	p.statsMu.Unlock()

	// 1. 通过处理器链处理数据
	processedSample := sample
	for _, processor := range p.processors {
		var err error
		processedSample, err = processor.Process(ctx, processedSample)
		if err != nil {
			// 更新错误计数
			p.statsMu.Lock()
			p.errors++
			p.statsMu.Unlock()
			return fmt.Errorf("processor %s failed: %w", processor.GetName(), err)
		}
	}

	// 2. 通过分析器分析数据
	for _, analyzer := range p.analyzers {
		results, err := analyzer.Analyze(ctx, processedSample)
		if err != nil {
			// 更新错误计数
			p.statsMu.Lock()
			p.errors++
			p.statsMu.Unlock()
			p.errorChan <- fmt.Errorf("analyzer %s failed: %w", analyzer.GetName(), err)
			continue
		}

		// 处理分析结果（可以输出到 sink 或触发事件）
		for _, result := range results {
			if result.Score > 0.7 { // 高分结果
				p.errorChan <- fmt.Errorf("high score analysis result: %s (score: %.2f)",
					result.Message, result.Score)
			}
		}
	}

	// 3. 输出到所有 sink
	for _, sink := range p.sinks {
		if err := sink.Write(ctx, processedSample); err != nil {
			// 更新错误计数
			p.statsMu.Lock()
			p.errors++
			p.statsMu.Unlock()
			p.errorChan <- fmt.Errorf("sink %s failed: %w", sink.GetName(), err)
		}
	}

	return nil
}

// GetErrors 获取错误通道
func (p *Pipeline) GetErrors() <-chan error {
	return p.errorChan
}

// GetName 获取管道名称
func (p *Pipeline) GetName() string {
	return p.name
}

// IsRunning 是否正在运行
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetStatus 获取管道状态
func (p *Pipeline) GetStatus() PipelineStatus {
	p.mu.RLock()
	running := p.running
	processorCount := len(p.processors)
	analyzerCount := len(p.analyzers)
	sinkCount := len(p.sinks)
	p.mu.RUnlock()

	p.statsMu.RLock()
	samplesProcessed := p.samplesProcessed
	errors := p.errors
	lastSampleTime := p.lastSampleTime
	p.statsMu.RUnlock()

	status := "stopped"
	if running {
		status = "running"
	}

	lastSampleTimeStr := ""
	if !lastSampleTime.IsZero() {
		lastSampleTimeStr = lastSampleTime.Format(time.RFC3339)
	}

	return PipelineStatus{
		Name:             p.name,
		Running:          running,
		Status:           status,
		SourceStatus:     p.source.GetStatus(),
		ProcessorCount:   processorCount,
		AnalyzerCount:    analyzerCount,
		SinkCount:        sinkCount,
		SamplesProcessed: samplesProcessed,
		Errors:           errors,
		LastSampleTime:   lastSampleTimeStr,
	}
}

// PipelineStatus 管道状态
type PipelineStatus struct {
	Name              string
	Running           bool
	Status            string
	SourceStatus      interfaces.SourceStatus
	ProcessorCount    int
	AnalyzerCount     int
	SinkCount         int
	SamplesProcessed  int64
	Errors            int64
	LastSampleTime    string
}

// GetSource 获取数据源
func (p *Pipeline) GetSource() interfaces.DataSource {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.source
}

// GetProcessors 获取处理器列表
func (p *Pipeline) GetProcessors() []interfaces.Processor {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// 返回副本以防止外部修改
	processors := make([]interfaces.Processor, len(p.processors))
	copy(processors, p.processors)
	return processors
}

// GetAnalyzers 获取分析器列表
func (p *Pipeline) GetAnalyzers() []interfaces.Analyzer {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// 返回副本以防止外部修改
	analyzers := make([]interfaces.Analyzer, len(p.analyzers))
	copy(analyzers, p.analyzers)
	return analyzers
}

// GetSinks 获取输出列表
func (p *Pipeline) GetSinks() []interfaces.DataSink {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// 返回副本以防止外部修改
	sinks := make([]interfaces.DataSink, len(p.sinks))
	copy(sinks, p.sinks)
	return sinks
}
