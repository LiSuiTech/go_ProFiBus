package processor

import (
	"context"
	"fmt"
	"sync"

	"go_ProFiBus/pkg/interfaces"
)

// Chain 处理器链实现
type Chain struct {
	processors []interfaces.Processor
	mu         sync.RWMutex
}

// NewChain 创建处理器链
func NewChain() *Chain {
	return &Chain{
		processors: make([]interfaces.Processor, 0),
	}
}

// AddProcessor 实现 ProcessorChain 接口
func (c *Chain) AddProcessor(p interfaces.Processor) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p == nil {
		return fmt.Errorf("processor cannot be nil")
	}

	c.processors = append(c.processors, p)
	return nil
}

// RemoveProcessor 实现 ProcessorChain 接口
func (c *Chain) RemoveProcessor(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, processor := range c.processors {
		if processor.GetName() == name {
			// 移除处理器
			c.processors = append(c.processors[:i], c.processors[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("processor %s not found", name)
}

// Process 实现 ProcessorChain 接口
func (c *Chain) Process(ctx context.Context, input interfaces.DataSample) (interfaces.DataSample, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	current := input
	for i, processor := range c.processors {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			var err error
			current, err = processor.Process(ctx, current)
			if err != nil {
				return nil, fmt.Errorf("processor %d (%s) failed: %w", i, processor.GetName(), err)
			}
		}
	}

	return current, nil
}

// GetProcessors 实现 ProcessorChain 接口
func (c *Chain) GetProcessors() []interfaces.Processor {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本
	processors := make([]interfaces.Processor, len(c.processors))
	copy(processors, c.processors)
	return processors
}

// Size 获取处理器数量
func (c *Chain) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.processors)
}

// Clear 清空处理器链
func (c *Chain) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processors = make([]interfaces.Processor, 0)
}

// 确保 Chain 实现了 ProcessorChain 接口
var _ interfaces.ProcessorChain = (*Chain)(nil)
