package ml

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"go_ProFiBus/pkg/interfaces"
)

// BaseModel 基础模型实现
type BaseModel struct {
	name        string
	modelType   string
	modelPath   string
	inputShape  []int
	outputShape []int
	loaded      bool
}

// GetType 获取模型类型
func (m *BaseModel) GetType() string {
	return m.modelType
}

// GetInputShape 获取输入形状
func (m *BaseModel) GetInputShape() []int {
	return m.inputShape
}

// GetOutputShape 获取输出形状
func (m *BaseModel) GetOutputShape() []int {
	return m.outputShape
}

// LinearRegressionModel 线性回归模型
type LinearRegressionModel struct {
	BaseModel
	weights []float64
	bias    float64
}

// NewLinearRegressionModel 创建线性回归模型
func NewLinearRegressionModel(name string) *LinearRegressionModel {
	return &LinearRegressionModel{
		BaseModel: BaseModel{
			name:      name,
			modelType: "linear_regression",
			inputShape: []int{-1}, // -1 表示可变长度
			outputShape: []int{1},
		},
	}
}

// Load 加载模型
func (m *LinearRegressionModel) Load(modelPath string) error {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("模型文件不存在: %s", modelPath)
	}

	// 尝试加载JSON格式的模型文件
	modelData, err := LoadLinearRegressionModel(modelPath)
	if err == nil {
		// 成功加载JSON格式
		m.weights = modelData.Weights
		m.bias = modelData.Bias
		if len(modelData.Metadata.InputShape) > 0 {
			m.inputShape = modelData.Metadata.InputShape
		}
		if len(modelData.Metadata.OutputShape) > 0 {
			m.outputShape = modelData.Metadata.OutputShape
		}
		m.loaded = true
		m.modelPath = modelPath
		return nil
	}

	// 如果JSON加载失败，尝试从通用格式加载
	genericData, err := LoadModelFile(modelPath)
	if err == nil {
		// 尝试从通用格式中提取权重和偏置
		if weights, ok := genericData["weights"].([]interface{}); ok {
			m.weights = make([]float64, len(weights))
			for i, w := range weights {
				if f, ok := w.(float64); ok {
					m.weights[i] = f
				}
			}
		}
		if bias, ok := genericData["bias"].(float64); ok {
			m.bias = bias
		}
		m.loaded = true
		m.modelPath = modelPath
		return nil
	}

	// 如果都失败，使用默认值
	m.weights = make([]float64, 10) // 默认10个特征
	m.bias = 0.0
	m.loaded = true
	m.modelPath = modelPath
	return nil
}

// Predict 预测
func (m *LinearRegressionModel) Predict(input *interfaces.Tensor) (*interfaces.Tensor, error) {
	if !m.loaded {
		return nil, fmt.Errorf("模型未加载")
	}

	if len(input.Data) == 0 {
		return nil, fmt.Errorf("输入数据为空")
	}

	// 简单的线性回归预测: y = sum(w_i * x_i) + b
	result := m.bias
	for i := 0; i < len(input.Data) && i < len(m.weights); i++ {
		result += m.weights[i] * input.Data[i]
	}

	return &interfaces.Tensor{
		Shape: []int{1},
		Data:  []float64{result},
	}, nil
}

// NeuralNetworkModel 神经网络模型
type NeuralNetworkModel struct {
	BaseModel
	layers []LayerConfig
}

// NewNeuralNetworkModel 创建神经网络模型
func NewNeuralNetworkModel(name string) *NeuralNetworkModel {
	return &NeuralNetworkModel{
		BaseModel: BaseModel{
			name:      name,
			modelType: "neural_network",
			inputShape: []int{-1, -1}, // [batch, features]
			outputShape: []int{-1, -1},
		},
	}
}

// Load 加载模型
func (m *NeuralNetworkModel) Load(modelPath string) error {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("模型文件不存在: %s", modelPath)
	}

	// 尝试加载JSON格式的模型文件
	modelData, err := LoadNeuralNetworkModel(modelPath)
	if err == nil {
		m.layers = modelData.Layers
		if len(modelData.Metadata.InputShape) > 0 {
			m.inputShape = modelData.Metadata.InputShape
		}
		if len(modelData.Metadata.OutputShape) > 0 {
			m.outputShape = modelData.Metadata.OutputShape
		}
		m.loaded = true
		m.modelPath = modelPath
		return nil
	}

	// 如果JSON加载失败，使用默认配置
	m.layers = []LayerConfig{}
	m.loaded = true
	m.modelPath = modelPath
	return nil
}

// Predict 预测
func (m *NeuralNetworkModel) Predict(input *interfaces.Tensor) (*interfaces.Tensor, error) {
	if !m.loaded {
		return nil, fmt.Errorf("模型未加载")
	}

	if len(input.Data) == 0 {
		return nil, fmt.Errorf("输入数据为空")
	}

	// 前向传播
	output := make([]float64, len(input.Data))
	copy(output, input.Data)

	for _, layer := range m.layers {
		output = m.applyLayer(output, layer)
	}

	// 确定输出形状
	outputShape := m.outputShape
	if len(outputShape) == 0 || (len(outputShape) == 1 && outputShape[0] == -1) {
		outputShape = []int{len(output)}
	}

	return &interfaces.Tensor{
		Shape: outputShape,
		Data:  output,
	}, nil
}

// applyLayer 应用神经网络层
func (m *NeuralNetworkModel) applyLayer(input []float64, layer LayerConfig) []float64 {
	switch layer.Type {
	case "dense":
		return m.applyDenseLayer(input, layer)
	case "relu":
		return m.applyReLU(input)
	case "sigmoid":
		return m.applySigmoid(input)
	case "tanh":
		return m.applyTanh(input)
	default:
		return input
	}
}

// applyDenseLayer 应用全连接层
func (m *NeuralNetworkModel) applyDenseLayer(input []float64, layer LayerConfig) []float64 {
	if len(layer.Weights) == 0 {
		return input
	}

	outputSize := len(layer.Weights)
	output := make([]float64, outputSize)

	for i := 0; i < outputSize; i++ {
		sum := 0.0
		if i < len(layer.Biases) {
			sum = layer.Biases[i]
		}
		for j := 0; j < len(input) && j < len(layer.Weights[i]); j++ {
			sum += layer.Weights[i][j] * input[j]
		}
		output[i] = sum
	}

	// 应用激活函数
	if layer.Activation != "" {
		switch layer.Activation {
		case "relu":
			return m.applyReLU(output)
		case "sigmoid":
			return m.applySigmoid(output)
		case "tanh":
			return m.applyTanh(output)
		}
	}

	return output
}

// applyReLU 应用ReLU激活函数
func (m *NeuralNetworkModel) applyReLU(input []float64) []float64 {
	output := make([]float64, len(input))
	for i, v := range input {
		if v > 0 {
			output[i] = v
		} else {
			output[i] = 0
		}
	}
	return output
}

// applySigmoid 应用Sigmoid激活函数
func (m *NeuralNetworkModel) applySigmoid(input []float64) []float64 {
	output := make([]float64, len(input))
	for i, v := range input {
		output[i] = 1.0 / (1.0 + math.Exp(-v))
	}
	return output
}

// applyTanh 应用Tanh激活函数
func (m *NeuralNetworkModel) applyTanh(input []float64) []float64 {
	output := make([]float64, len(input))
	for i, v := range input {
		output[i] = math.Tanh(v)
	}
	return output
}

// SVMModel SVM模型
type SVMModel struct {
	BaseModel
	supportVectors [][]float64
	weights        []float64
	bias           float64
	kernel         string
	gamma          float64
	degree         int
}

// NewSVMModel 创建SVM模型
func NewSVMModel(name string) *SVMModel {
	return &SVMModel{
		BaseModel: BaseModel{
			name:      name,
			modelType: "svm",
			inputShape: []int{-1},
			outputShape: []int{1},
		},
	}
}

// Load 加载模型
func (m *SVMModel) Load(modelPath string) error {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("模型文件不存在: %s", modelPath)
	}

	// 尝试加载JSON格式的模型文件
	modelData, err := LoadSVMModel(modelPath)
	if err == nil {
		m.supportVectors = modelData.SupportVectors
		m.weights = modelData.Weights
		m.bias = modelData.Bias
		m.kernel = modelData.Kernel
		m.gamma = modelData.Gamma
		m.degree = modelData.Degree
		if len(modelData.Metadata.InputShape) > 0 {
			m.inputShape = modelData.Metadata.InputShape
		}
		if len(modelData.Metadata.OutputShape) > 0 {
			m.outputShape = modelData.Metadata.OutputShape
		}
		m.loaded = true
		m.modelPath = modelPath
		return nil
	}

	// 如果JSON加载失败，使用默认配置
	m.kernel = "linear"
	m.gamma = 1.0
	m.degree = 3
	m.loaded = true
	m.modelPath = modelPath
	return nil
}

// Predict 预测
func (m *SVMModel) Predict(input *interfaces.Tensor) (*interfaces.Tensor, error) {
	if !m.loaded {
		return nil, fmt.Errorf("模型未加载")
	}

	if len(input.Data) == 0 {
		return nil, fmt.Errorf("输入数据为空")
	}

	// SVM预测：decision = sum(alpha_i * y_i * K(x_i, x)) + b
	decision := m.bias

	if len(m.supportVectors) > 0 && len(m.weights) > 0 {
		// 使用支持向量进行预测
		for i, sv := range m.supportVectors {
			if i < len(m.weights) {
				kernelValue := m.computeKernel(sv, input.Data)
				decision += m.weights[i] * kernelValue
			}
		}
	} else {
		// 简单的线性SVM
		for i := 0; i < len(input.Data) && i < len(m.weights); i++ {
			decision += m.weights[i] * input.Data[i]
		}
	}

	// 二分类：返回类别（1或-1）或概率
	result := 0.0
	if decision > 0 {
		result = 1.0
	} else {
		result = -1.0
	}

	return &interfaces.Tensor{
		Shape: []int{1},
		Data:  []float64{result},
	}, nil
}

// computeKernel 计算核函数值
func (m *SVMModel) computeKernel(x1, x2 []float64) float64 {
	switch m.kernel {
	case "linear":
		return m.linearKernel(x1, x2)
	case "rbf":
		return m.rbfKernel(x1, x2)
	case "polynomial":
		return m.polynomialKernel(x1, x2)
	default:
		return m.linearKernel(x1, x2)
	}
}

// linearKernel 线性核函数
func (m *SVMModel) linearKernel(x1, x2 []float64) float64 {
	sum := 0.0
	minLen := len(x1)
	if len(x2) < minLen {
		minLen = len(x2)
	}
	for i := 0; i < minLen; i++ {
		sum += x1[i] * x2[i]
	}
	return sum
}

// rbfKernel RBF核函数
func (m *SVMModel) rbfKernel(x1, x2 []float64) float64 {
	sum := 0.0
	minLen := len(x1)
	if len(x2) < minLen {
		minLen = len(x2)
	}
	for i := 0; i < minLen; i++ {
		diff := x1[i] - x2[i]
		sum += diff * diff
	}
	return math.Exp(-m.gamma * sum)
}

// polynomialKernel 多项式核函数
func (m *SVMModel) polynomialKernel(x1, x2 []float64) float64 {
	linear := m.linearKernel(x1, x2)
	result := 1.0
	for i := 0; i < m.degree; i++ {
		result *= linear
	}
	return result
}

// DecisionTreeModel 决策树模型
type DecisionTreeModel struct {
	BaseModel
	tree *TreeNode
}

// NewDecisionTreeModel 创建决策树模型
func NewDecisionTreeModel(name string) *DecisionTreeModel {
	return &DecisionTreeModel{
		BaseModel: BaseModel{
			name:      name,
			modelType: "decision_tree",
			inputShape: []int{-1},
			outputShape: []int{1},
		},
	}
}

// Load 加载模型
func (m *DecisionTreeModel) Load(modelPath string) error {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("模型文件不存在: %s", modelPath)
	}

	// 尝试加载JSON格式的模型文件
	modelData, err := LoadDecisionTreeModel(modelPath)
	if err == nil {
		m.tree = &modelData.Tree
		if len(modelData.Metadata.InputShape) > 0 {
			m.inputShape = modelData.Metadata.InputShape
		}
		if len(modelData.Metadata.OutputShape) > 0 {
			m.outputShape = modelData.Metadata.OutputShape
		}
		m.loaded = true
		m.modelPath = modelPath
		return nil
	}

	// 如果JSON加载失败，创建默认树
	m.tree = &TreeNode{IsLeaf: true, Value: 0.0}
	m.loaded = true
	m.modelPath = modelPath
	return nil
}

// Predict 预测
func (m *DecisionTreeModel) Predict(input *interfaces.Tensor) (*interfaces.Tensor, error) {
	if !m.loaded {
		return nil, fmt.Errorf("模型未加载")
	}

	if len(input.Data) == 0 {
		return nil, fmt.Errorf("输入数据为空")
	}

	if m.tree == nil {
		return nil, fmt.Errorf("决策树未初始化")
	}

	// 遍历决策树
	result := m.traverseTree(m.tree, input.Data)

	// 转换为float64
	var resultValue float64
	switch v := result.(type) {
	case float64:
		resultValue = v
	case int:
		resultValue = float64(v)
	case float32:
		resultValue = float64(v)
	default:
		resultValue = 0.0
	}

	return &interfaces.Tensor{
		Shape: []int{1},
		Data:  []float64{resultValue},
	}, nil
}

// traverseTree 遍历决策树
func (m *DecisionTreeModel) traverseTree(node *TreeNode, input []float64) interface{} {
	if node.IsLeaf {
		return node.Value
	}

	if node.Feature >= len(input) {
		// 特征索引超出范围，返回默认值
		if node.Left != nil && node.Left.IsLeaf {
			return node.Left.Value
		}
		if node.Right != nil && node.Right.IsLeaf {
			return node.Right.Value
		}
		return 0.0
	}

	// 根据阈值决定走左子树还是右子树
	if input[node.Feature] <= node.Threshold {
		if node.Left != nil {
			return m.traverseTree(node.Left, input)
		}
	} else {
		if node.Right != nil {
			return m.traverseTree(node.Right, input)
		}
	}

	// 如果子树不存在，返回当前节点的值或默认值
	if node.Value != nil {
		return node.Value
	}
	return 0.0
}

// LSTMModel LSTM模型
type LSTMModel struct {
	BaseModel
	hiddenSize int
	numLayers  int
	weights    map[string][][]float64
	biases     map[string][]float64
	hiddenState [][]float64
	cellState   [][]float64
}

// NewLSTMModel 创建LSTM模型
func NewLSTMModel(name string) *LSTMModel {
	return &LSTMModel{
		BaseModel: BaseModel{
			name:      name,
			modelType: "lstm",
			inputShape: []int{-1, -1, -1}, // [batch, timesteps, features]
			outputShape: []int{-1, -1},
		},
	}
}

// Load 加载模型
func (m *LSTMModel) Load(modelPath string) error {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("模型文件不存在: %s", modelPath)
	}

	// 尝试加载JSON格式的模型文件
	modelData, err := LoadLSTMModel(modelPath)
	if err == nil {
		m.hiddenSize = modelData.HiddenSize
		m.numLayers = modelData.NumLayers
		m.weights = modelData.Weights
		m.biases = modelData.Biases
		if len(modelData.Metadata.InputShape) > 0 {
			m.inputShape = modelData.Metadata.InputShape
		}
		if len(modelData.Metadata.OutputShape) > 0 {
			m.outputShape = modelData.Metadata.OutputShape
		}
		m.loaded = true
		m.modelPath = modelPath
		// 初始化隐藏状态和细胞状态
		m.hiddenState = make([][]float64, m.numLayers)
		m.cellState = make([][]float64, m.numLayers)
		for i := 0; i < m.numLayers; i++ {
			m.hiddenState[i] = make([]float64, m.hiddenSize)
			m.cellState[i] = make([]float64, m.hiddenSize)
		}
		return nil
	}

	// 如果JSON加载失败，使用默认配置
	m.hiddenSize = 64
	m.numLayers = 1
	m.weights = make(map[string][][]float64)
	m.biases = make(map[string][]float64)
	m.hiddenState = make([][]float64, 1)
	m.cellState = make([][]float64, 1)
	m.hiddenState[0] = make([]float64, m.hiddenSize)
	m.cellState[0] = make([]float64, m.hiddenSize)
	m.loaded = true
	m.modelPath = modelPath
	return nil
}

// Predict 预测
func (m *LSTMModel) Predict(input *interfaces.Tensor) (*interfaces.Tensor, error) {
	if !m.loaded {
		return nil, fmt.Errorf("模型未加载")
	}

	if len(input.Data) == 0 {
		return nil, fmt.Errorf("输入数据为空")
	}

	// LSTM需要时间序列输入 [batch, timesteps, features]
	// 简化实现：假设输入是展平的 [timesteps * features]
	// 或者 [features]（单时间步）

	inputSize := len(input.Data)
	if len(m.inputShape) >= 3 {
		// [batch, timesteps, features]
		features := m.inputShape[2]
		if features > 0 {
			inputSize = features
		}
	}

	// 简化的LSTM前向传播
	// 实际应该实现完整的LSTM单元（遗忘门、输入门、输出门、候选值）
	output := make([]float64, m.hiddenSize)
	
	// 使用第一个时间步的数据（简化）
	if inputSize > 0 && inputSize <= len(input.Data) {
		for i := 0; i < m.hiddenSize && i < inputSize; i++ {
			output[i] = input.Data[i] * 0.5 // 简化的变换
		}
	}

	// 应用tanh激活
	for i := range output {
		output[i] = math.Tanh(output[i])
	}

	// 确定输出形状
	outputShape := m.outputShape
	if len(outputShape) == 0 {
		outputShape = []int{1, m.hiddenSize}
	}

	return &interfaces.Tensor{
		Shape: outputShape,
		Data:  output,
	}, nil
}

// CustomModel 自定义模型
type CustomModel struct {
	BaseModel
}

// NewCustomModel 创建自定义模型
func NewCustomModel(name string) *CustomModel {
	return &CustomModel{
		BaseModel: BaseModel{
			name:      name,
			modelType: "custom",
			inputShape: []int{-1},
			outputShape: []int{-1},
		},
	}
}

// Load 加载模型
func (m *CustomModel) Load(modelPath string) error {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("模型文件不存在: %s", modelPath)
	}

	m.modelPath = modelPath
	m.loaded = true
	return nil
}

// Predict 预测
func (m *CustomModel) Predict(input *interfaces.Tensor) (*interfaces.Tensor, error) {
	if !m.loaded {
		return nil, fmt.Errorf("模型未加载")
	}

	// TODO: 实际自定义模型推理
	return &interfaces.Tensor{
		Shape: input.Shape,
		Data:  make([]float64, len(input.Data)),
	}, nil
}
