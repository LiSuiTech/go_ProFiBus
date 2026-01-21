package interfaces

import "context"

// Plugin 算法插件接口
// 用于实现可配置的算法组件
type Plugin interface {
	// GetID 获取插件ID
	GetID() string

	// GetName 获取插件名称
	GetName() string

	// GetType 获取插件类型
	GetType() PluginType

	// GetVersion 获取插件版本
	GetVersion() string

	// GetDescription 获取插件描述
	GetDescription() string

	// GetSchema 获取配置Schema（用于前端渲染表单）
	GetSchema() *PluginConfigSchema

	// Configure 配置插件
	Configure(config map[string]interface{}) error

	// Execute 执行插件
	Execute(ctx context.Context, input interface{}) (interface{}, error)

	// Validate 验证输入
	Validate(input interface{}) error

	// Close 关闭插件
	Close() error
}

// PluginType 插件类型
type PluginType string

const (
	// PluginTypeSource 数据源插件
	PluginTypeSource PluginType = "source"

	// PluginTypeProcessor 处理器插件
	PluginTypeProcessor PluginType = "processor"

	// PluginTypeAnalyzer 分析器插件
	PluginTypeAnalyzer PluginType = "analyzer"

	// PluginTypeModel 模型插件
	PluginTypeModel PluginType = "model"

	// PluginTypeSink 输出插件
	PluginTypeSink PluginType = "sink"
)

// PluginConfigSchema 插件配置Schema
// 用于自动生成配置表单
type PluginConfigSchema struct {
	// Properties 配置属性
	Properties map[string]PluginPropertySchema `json:"properties"`

	// Required 必填字段
	Required []string `json:"required"`

	// Title 标题
	Title string `json:"title,omitempty"`

	// Description 描述
	Description string `json:"description,omitempty"`
}

// PluginPropertySchema 插件属性Schema
type PluginPropertySchema struct{
	// Type 数据类型: string, number, boolean, array, object
	Type string `json:"type"`

	// Title 显示标题
	Title string `json:"title"`

	// Description 描述
	Description string `json:"description"`

	// Default 默认值
	Default interface{} `json:"default,omitempty"`

	// Enum 枚举值（用于下拉选择）
	Enum []interface{} `json:"enum,omitempty"`

	// Minimum 最小值（数值类型）
	Minimum *float64 `json:"minimum,omitempty"`

	// Maximum 最大值（数值类型）
	Maximum *float64 `json:"maximum,omitempty"`

	// MinLength 最小长度（字符串类型）
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength 最大长度（字符串类型）
	MaxLength *int `json:"maxLength,omitempty"`

	// Pattern 正则表达式（字符串类型）
	Pattern string `json:"pattern,omitempty"`

	// Items 数组项Schema（数组类型）
	Items *PluginPropertySchema `json:"items,omitempty"`

	// Properties 对象属性（对象类型）
	Properties map[string]PluginPropertySchema `json:"properties,omitempty"`

	// Format 格式 (date, time, email, uri, etc.)
	Format string `json:"format,omitempty"`

	// ReadOnly 只读
	ReadOnly bool `json:"readOnly,omitempty"`

	// Hidden 隐藏
	Hidden bool `json:"hidden,omitempty"`
}

// PluginRegistry 插件注册中心接口
type PluginRegistry interface {
	// Register 注册插件
	Register(plugin Plugin) error

	// Unregister 注销插件
	Unregister(pluginID string) error

	// Get 获取插件
	Get(pluginID string) (Plugin, error)

	// List 列出插件
	List(pluginType PluginType) []Plugin

	// ListAll 列出所有插件
	ListAll() []Plugin

	// Has 检查插件是否存在
	Has(pluginID string) bool
}

// PluginFactory 插件工厂接口
type PluginFactory interface {
	// Create 创建插件实例
	Create(pluginID string, config map[string]interface{}) (Plugin, error)

	// GetSupportedPlugins 获取支持的插件列表
	GetSupportedPlugins() []string
}

// DataSink 数据输出接口
type DataSink interface {
	// Write 写入数据
	Write(ctx context.Context, data interface{}) error

	// WriteBatch 批量写入数据
	WriteBatch(ctx context.Context, data []interface{}) error

	// Flush 刷新缓冲区
	Flush(ctx context.Context) error

	// Close 关闭输出
	Close() error

	// GetName 获取名称
	GetName() string
}
