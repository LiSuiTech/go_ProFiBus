package interfaces

import "context"

// Processor 数据处理器接口
// 对数据进行转换、过滤、增强等处理
type Processor interface {
	// Process 处理数据
	Process(ctx context.Context, input DataSample) (DataSample, error)

	// GetName 获取处理器名称
	GetName() string

	// GetConfig 获取处理器配置
	GetConfig() ProcessorConfig

	// Initialize 初始化处理器
	Initialize(config ProcessorConfig) error

	// Close 关闭处理器
	Close() error
}

// ProcessorChain 处理器链
// 按顺序执行多个处理器
type ProcessorChain interface {
	// AddProcessor 添加处理器
	AddProcessor(p Processor) error

	// RemoveProcessor 移除处理器
	RemoveProcessor(name string) error

	// Process 执行处理器链
	Process(ctx context.Context, input DataSample) (DataSample, error)

	// GetProcessors 获取所有处理器
	GetProcessors() []Processor
}

// ProcessorConfig is defined in config.go to avoid duplication
