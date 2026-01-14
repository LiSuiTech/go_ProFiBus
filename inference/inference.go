package inference

import (
	"fmt"
	"go_ProFiBus/errors"
	"go_ProFiBus/logger"
	"math"
	"sync"
)

// ModelType 模型类型
type ModelType int

const (
	ModelLinearRegression ModelType = iota
	ModelLogisticRegression
	ModelDecisionTree
	ModelNeuralNetwork
	ModelSVM
	ModelKNN
	ModelCustom
)

// Tensor 张量数据结构
type Tensor struct {
	Shape []int     // 形状
	Data  []float64 // 数据
}

// NewTensor 创建新的张量
func NewTensor(shape []int, data []float64) (*Tensor, error) {
	size := 1
	for _, dim := range shape {
		size *= dim
	}

	if len(data) != size {
		return nil, errors.Newf(errors.ErrInvalidInputShape,
			"数据大小 %d 与形状 %v 不匹配", len(data), shape)
	}

	return &Tensor{
		Shape: shape,
		Data:  data,
	}, nil
}

// Reshape 重塑张量
func (t *Tensor) Reshape(newShape []int) error {
	newSize := 1
	for _, dim := range newShape {
		newSize *= dim
	}

	if newSize != len(t.Data) {
		return errors.Newf(errors.ErrInvalidInputShape,
			"无法将大小 %d 重塑为 %v", len(t.Data), newShape)
	}

	t.Shape = newShape
	return nil
}

// Model 模型接口
type Model interface {
	Load(modelPath string) error
	Predict(input *Tensor) (*Tensor, error)
	GetType() ModelType
	GetInputShape() []int
	GetOutputShape() []int
}

// InferenceEngine 推理引擎
type InferenceEngine struct {
	models map[string]Model
	mu     sync.RWMutex
	log    *logger.Logger
}

// NewInferenceEngine 创建推理引擎
func NewInferenceEngine() *InferenceEngine {
	return &InferenceEngine{
		models: make(map[string]Model),
		log:    logger.GetLogger(),
	}
}

// RegisterModel 注册模型
func (ie *InferenceEngine) RegisterModel(name string, model Model) error {
	ie.mu.Lock()
	defer ie.mu.Unlock()

	if _, exists := ie.models[name]; exists {
		return errors.Newf(errors.ErrInvalidParam, "模型已存在: %s", name)
	}

	ie.models[name] = model
	ie.log.Info("已注册模型: %s", name)
	return nil
}

// UnregisterModel 注销模型
func (ie *InferenceEngine) UnregisterModel(name string) error {
	ie.mu.Lock()
	defer ie.mu.Unlock()

	if _, exists := ie.models[name]; !exists {
		return errors.Newf(errors.ErrModelNotLoaded, "模型不存在: %s", name)
	}

	delete(ie.models, name)
	ie.log.Info("已注销模型: %s", name)
	return nil
}

// Predict 执行推理
func (ie *InferenceEngine) Predict(modelName string, input *Tensor) (*Tensor, error) {
	ie.mu.RLock()
	model, exists := ie.models[modelName]
	ie.mu.RUnlock()

	if !exists {
		return nil, errors.Newf(errors.ErrModelNotLoaded, "模型未加载: %s", modelName)
	}

	output, err := model.Predict(input)
	if err != nil {
		ie.log.Error("推理失败 [%s]: %v", modelName, err)
		return nil, errors.Wrap(errors.ErrInferenceFailed, err)
	}

	return output, nil
}

// BatchPredict 批量推理
func (ie *InferenceEngine) BatchPredict(modelName string, inputs []*Tensor) ([]*Tensor, error) {
	ie.mu.RLock()
	model, exists := ie.models[modelName]
	ie.mu.RUnlock()

	if !exists {
		return nil, errors.Newf(errors.ErrModelNotLoaded, "模型未加载: %s", modelName)
	}

	outputs := make([]*Tensor, len(inputs))
	for i, input := range inputs {
		output, err := model.Predict(input)
		if err != nil {
			ie.log.Error("批量推理失败 [%s] 索引 %d: %v", modelName, i, err)
			return nil, errors.Wrap(errors.ErrInferenceFailed, err)
		}
		outputs[i] = output
	}

	return outputs, nil
}

// GetModel 获取模型
func (ie *InferenceEngine) GetModel(name string) (Model, error) {
	ie.mu.RLock()
	defer ie.mu.RUnlock()

	model, exists := ie.models[name]
	if !exists {
		return nil, errors.Newf(errors.ErrModelNotLoaded, "模型未加载: %s", name)
	}

	return model, nil
}

// ListModels 列出所有模型
func (ie *InferenceEngine) ListModels() []string {
	ie.mu.RLock()
	defer ie.mu.RUnlock()

	names := make([]string, 0, len(ie.models))
	for name := range ie.models {
		names = append(names, name)
	}
	return names
}

// LinearRegressionModel 线性回归模型
type LinearRegressionModel struct {
	weights []float64
	bias    float64
	loaded  bool
}

// NewLinearRegressionModel 创建线性回归模型
func NewLinearRegressionModel() *LinearRegressionModel {
	return &LinearRegressionModel{}
}

// Load 加载模型
func (lrm *LinearRegressionModel) Load(modelPath string) error {
	// 实际应用中应从文件加载模型参数
	// 这里提供一个示例实现
	lrm.loaded = true
	return nil
}

// Predict 预测
func (lrm *LinearRegressionModel) Predict(input *Tensor) (*Tensor, error) {
	if !lrm.loaded {
		return nil, errors.New(errors.ErrModelNotLoaded, nil)
	}

	if len(input.Shape) != 2 {
		return nil, errors.Newf(errors.ErrInvalidInputShape,
			"期望2维输入，得到 %d 维", len(input.Shape))
	}

	batchSize := input.Shape[0]
	features := input.Shape[1]

	if len(lrm.weights) == 0 {
		// 初始化权重
		lrm.weights = make([]float64, features)
		for i := range lrm.weights {
			lrm.weights[i] = 1.0
		}
	}

	output := make([]float64, batchSize)
	for i := 0; i < batchSize; i++ {
		sum := lrm.bias
		for j := 0; j < features; j++ {
			sum += input.Data[i*features+j] * lrm.weights[j]
		}
		output[i] = sum
	}

	return NewTensor([]int{batchSize, 1}, output)
}

// GetType 获取模型类型
func (lrm *LinearRegressionModel) GetType() ModelType {
	return ModelLinearRegression
}

// GetInputShape 获取输入形状
func (lrm *LinearRegressionModel) GetInputShape() []int {
	return []int{-1, len(lrm.weights)} // -1 表示批大小可变
}

// GetOutputShape 获取输出形状
func (lrm *LinearRegressionModel) GetOutputShape() []int {
	return []int{-1, 1}
}

// SetWeights 设置权重
func (lrm *LinearRegressionModel) SetWeights(weights []float64, bias float64) {
	lrm.weights = weights
	lrm.bias = bias
	lrm.loaded = true
}

// NeuralNetworkModel 简单神经网络模型
type NeuralNetworkModel struct {
	layers [][]float64 // 每层的权重
	biases [][]float64 // 每层的偏置
	loaded bool
}

// NewNeuralNetworkModel 创建神经网络模型
func NewNeuralNetworkModel(layerSizes []int) *NeuralNetworkModel {
	if len(layerSizes) < 2 {
		return nil
	}

	nn := &NeuralNetworkModel{
		layers: make([][]float64, len(layerSizes)-1),
		biases: make([][]float64, len(layerSizes)-1),
	}

	// 初始化权重和偏置
	for i := 0; i < len(layerSizes)-1; i++ {
		inputSize := layerSizes[i]
		outputSize := layerSizes[i+1]

		// 使用简单的随机初始化
		nn.layers[i] = make([]float64, inputSize*outputSize)
		nn.biases[i] = make([]float64, outputSize)

		// 小随机值初始化
		for j := range nn.layers[i] {
			nn.layers[i][j] = 0.01
		}
	}

	return nn
}

// Load 加载模型
func (nn *NeuralNetworkModel) Load(modelPath string) error {
	// 实际应用中应从文件加载模型参数
	nn.loaded = true
	return nil
}

// sigmoid 激活函数
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// relu 激活函数
func relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

// Predict 前向传播预测
func (nn *NeuralNetworkModel) Predict(input *Tensor) (*Tensor, error) {
	if !nn.loaded {
		return nil, errors.New(errors.ErrModelNotLoaded, nil)
	}

	if len(input.Shape) != 2 {
		return nil, errors.Newf(errors.ErrInvalidInputShape,
			"期望2维输入，得到 %d 维", len(input.Shape))
	}

	current := input.Data
	batchSize := input.Shape[0]

	// 逐层前向传播
	for layerIdx := 0; layerIdx < len(nn.layers); layerIdx++ {
		layer := nn.layers[layerIdx]
		bias := nn.biases[layerIdx]

		inputSize := len(current) / batchSize
		outputSize := len(bias)

		next := make([]float64, batchSize*outputSize)

		for b := 0; b < batchSize; b++ {
			for o := 0; o < outputSize; o++ {
				sum := bias[o]
				for i := 0; i < inputSize; i++ {
					sum += current[b*inputSize+i] * layer[i*outputSize+o]
				}

				// 使用 ReLU 激活函数（最后一层除外）
				if layerIdx == len(nn.layers)-1 {
					next[b*outputSize+o] = sum
				} else {
					next[b*outputSize+o] = relu(sum)
				}
			}
		}

		current = next
	}

	outputSize := len(nn.biases[len(nn.biases)-1])
	return NewTensor([]int{batchSize, outputSize}, current)
}

// GetType 获取模型类型
func (nn *NeuralNetworkModel) GetType() ModelType {
	return ModelNeuralNetwork
}

// GetInputShape 获取输入形状
func (nn *NeuralNetworkModel) GetInputShape() []int {
	if len(nn.layers) == 0 {
		return []int{-1, 0}
	}
	inputSize := len(nn.layers[0]) / len(nn.biases[0])
	return []int{-1, inputSize}
}

// GetOutputShape 获取输出形状
func (nn *NeuralNetworkModel) GetOutputShape() []int {
	if len(nn.biases) == 0 {
		return []int{-1, 0}
	}
	return []int{-1, len(nn.biases[len(nn.biases)-1])}
}

// PreprocessingPipeline 数据预处理管道
type PreprocessingPipeline struct {
	steps []PreprocessingStep
}

// PreprocessingStep 预处理步骤
type PreprocessingStep interface {
	Transform(input *Tensor) (*Tensor, error)
	Fit(data []*Tensor) error
}

// NewPreprocessingPipeline 创建预处理管道
func NewPreprocessingPipeline() *PreprocessingPipeline {
	return &PreprocessingPipeline{
		steps: make([]PreprocessingStep, 0),
	}
}

// AddStep 添加预处理步骤
func (pp *PreprocessingPipeline) AddStep(step PreprocessingStep) {
	pp.steps = append(pp.steps, step)
}

// Transform 执行转换
func (pp *PreprocessingPipeline) Transform(input *Tensor) (*Tensor, error) {
	current := input
	var err error

	for i, step := range pp.steps {
		current, err = step.Transform(current)
		if err != nil {
			return nil, fmt.Errorf("步骤 %d 失败: %v", i, err)
		}
	}

	return current, nil
}

// Fit 拟合预处理参数
func (pp *PreprocessingPipeline) Fit(data []*Tensor) error {
	for i, step := range pp.steps {
		if err := step.Fit(data); err != nil {
			return fmt.Errorf("拟合步骤 %d 失败: %v", i, err)
		}
	}
	return nil
}

// Normalizer 归一化器
type Normalizer struct {
	mean   []float64
	stddev []float64
	fitted bool
}

// NewNormalizer 创建归一化器
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// Fit 计算均值和标准差
func (n *Normalizer) Fit(data []*Tensor) error {
	if len(data) == 0 {
		return errors.Newf(errors.ErrInvalidParam, "数据为空")
	}

	// 假设所有张量形状相同
	features := len(data[0].Data)
	n.mean = make([]float64, features)
	n.stddev = make([]float64, features)

	// 计算均值
	for _, tensor := range data {
		for i, val := range tensor.Data {
			n.mean[i] += val
		}
	}
	for i := range n.mean {
		n.mean[i] /= float64(len(data))
	}

	// 计算标准差
	for _, tensor := range data {
		for i, val := range tensor.Data {
			diff := val - n.mean[i]
			n.stddev[i] += diff * diff
		}
	}
	for i := range n.stddev {
		n.stddev[i] = math.Sqrt(n.stddev[i] / float64(len(data)))
		if n.stddev[i] == 0 {
			n.stddev[i] = 1.0 // 避免除零
		}
	}

	n.fitted = true
	return nil
}

// Transform 标准化数据
func (n *Normalizer) Transform(input *Tensor) (*Tensor, error) {
	if !n.fitted {
		return nil, errors.Newf(errors.ErrInvalidParam, "归一化器未拟合")
	}

	normalized := make([]float64, len(input.Data))
	for i, val := range input.Data {
		idx := i % len(n.mean)
		normalized[i] = (val - n.mean[idx]) / n.stddev[idx]
	}

	return NewTensor(input.Shape, normalized)
}
