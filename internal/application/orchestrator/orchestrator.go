package orchestrator

import (
	"context"
	"fmt"
	"sync"
)

// Orchestrator 数据流编排器
// 管理多个 Pipeline 的生命周期
type Orchestrator struct {
	pipelines map[string]*Pipeline
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewOrchestrator 创建编排器
func NewOrchestrator() *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Orchestrator{
		pipelines: make(map[string]*Pipeline),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// AddPipeline 添加管道
func (o *Orchestrator) AddPipeline(pipeline *Pipeline) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.pipelines[pipeline.name]; exists {
		return fmt.Errorf("pipeline %s already exists", pipeline.name)
	}

	o.pipelines[pipeline.name] = pipeline
	return nil
}

// RemovePipeline 移除管道
func (o *Orchestrator) RemovePipeline(name string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	pipeline, exists := o.pipelines[name]
	if !exists {
		return fmt.Errorf("pipeline %s not found", name)
	}

	if pipeline.IsRunning() {
		if err := pipeline.Stop(); err != nil {
			return fmt.Errorf("failed to stop pipeline: %w", err)
		}
	}

	delete(o.pipelines, name)
	return nil
}

// GetPipeline 获取管道
func (o *Orchestrator) GetPipeline(name string) (*Pipeline, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	pipeline, exists := o.pipelines[name]
	if !exists {
		return nil, fmt.Errorf("pipeline %s not found", name)
	}

	return pipeline, nil
}

// StartPipeline 启动指定管道
func (o *Orchestrator) StartPipeline(name string) error {
	pipeline, err := o.GetPipeline(name)
	if err != nil {
		return err
	}

	return pipeline.Start(o.ctx)
}

// StopPipeline 停止指定管道
func (o *Orchestrator) StopPipeline(name string) error {
	pipeline, err := o.GetPipeline(name)
	if err != nil {
		return err
	}

	return pipeline.Stop()
}

// StartAll 启动所有管道
func (o *Orchestrator) StartAll() error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for name, pipeline := range o.pipelines {
		if err := pipeline.Start(o.ctx); err != nil {
			return fmt.Errorf("failed to start pipeline %s: %w", name, err)
		}
	}

	return nil
}

// StopAll 停止所有管道
func (o *Orchestrator) StopAll() error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var errs []error
	for name, pipeline := range o.pipelines {
		if err := pipeline.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop pipeline %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping pipelines: %v", errs)
	}

	return nil
}

// GetAllPipelines 获取所有管道
func (o *Orchestrator) GetAllPipelines() map[string]*Pipeline {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// 返回副本
	pipelines := make(map[string]*Pipeline)
	for name, pipeline := range o.pipelines {
		pipelines[name] = pipeline
	}

	return pipelines
}

// GetStatus 获取所有管道状态
func (o *Orchestrator) GetStatus() []PipelineStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	statuses := make([]PipelineStatus, 0, len(o.pipelines))
	for _, pipeline := range o.pipelines {
		statuses = append(statuses, pipeline.GetStatus())
	}

	return statuses
}

// Shutdown 关闭编排器
func (o *Orchestrator) Shutdown() error {
	// 停止所有管道
	if err := o.StopAll(); err != nil {
		return err
	}

	// 取消上下文
	o.cancel()

	// 等待所有 goroutine 结束
	o.wg.Wait()

	return nil
}

// MonitorErrors 监控所有管道的错误
func (o *Orchestrator) MonitorErrors(ctx context.Context, errorHandler func(pipelineName string, err error)) {
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()

		o.mu.RLock()
		pipelines := o.GetAllPipelines()
		o.mu.RUnlock()

		// 为每个管道启动错误监控
		for name, pipeline := range pipelines {
			pipelineName := name
			errorChan := pipeline.GetErrors()

			o.wg.Add(1)
			go func() {
				defer o.wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					case <-o.ctx.Done():
						return
					case err, ok := <-errorChan:
						if !ok {
							return
						}
						errorHandler(pipelineName, err)
					}
				}
			}()
		}
	}()
}
