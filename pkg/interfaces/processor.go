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

// ProcessorConfig 处理器配置
type ProcessorConfig map[string]interface{}

// Get 获取配置项
func (c ProcessorConfig) Get(key string) (interface{}, bool) {
	val, ok := c[key]
	return val, ok
}

// GetString 获取字符串配置项
func (c ProcessorConfig) GetString(key string, defaultValue string) string {
	if val, ok := c[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetInt 获取整型配置项
func (c ProcessorConfig) GetInt(key string, defaultValue int) int {
	if val, ok := c[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return defaultValue
}

// GetFloat 获取浮点配置项
func (c ProcessorConfig) GetFloat(key string, defaultValue float64) float64 {
	if val, ok := c[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
		if i, ok := val.(int); ok {
			return float64(i)
		}
	}
	return defaultValue
}

// GetBool 获取布尔配置项
func (c ProcessorConfig) GetBool(key string, defaultValue bool) bool {
	if val, ok := c[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}
