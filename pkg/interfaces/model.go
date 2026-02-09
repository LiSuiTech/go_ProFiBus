package interfaces

// MLModel ML模型接口
// 替代废弃的 inference.Model
type MLModel interface {
	// Load 加载模型
	Load(modelPath string) error

	// Predict 预测
	Predict(input *Tensor) (*Tensor, error)

	// GetType 获取模型类型
	GetType() string

	// GetInputShape 获取输入形状
	GetInputShape() []int

	// GetOutputShape 获取输出形状
	GetOutputShape() []int
}

// Tensor 张量数据结构
type Tensor struct {
	Shape []int     // 形状
	Data  []float64 // 数据
}

// InferenceEngine 推理引擎接口
type InferenceEngine interface {
	// RegisterModel 注册模型
	RegisterModel(name string, model MLModel) error

	// UnregisterModel 注销模型
	UnregisterModel(name string) error

	// Predict 执行推理
	Predict(modelName string, input *Tensor) (*Tensor, error)

	// GetModel 获取模型
	GetModel(name string) (MLModel, error)

	// ListModels 列出所有模型
	ListModels() []string
}
