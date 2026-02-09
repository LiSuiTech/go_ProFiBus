package ml

import (
	"encoding/json"
	"fmt"
	"os"
)

// ModelMetadata 模型元数据
type ModelMetadata struct {
	Version     string                 `json:"version"`
	ModelType   string                 `json:"model_type"`
	InputShape  []int                  `json:"input_shape"`
	OutputShape []int                  `json:"output_shape"`
	CreatedAt   string                 `json:"created_at"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// LinearRegressionModelData 线性回归模型数据
type LinearRegressionModelData struct {
	Metadata ModelMetadata `json:"metadata"`
	Weights  []float64     `json:"weights"`
	Bias     float64       `json:"bias"`
}

// NeuralNetworkModelData 神经网络模型数据
type NeuralNetworkModelData struct {
	Metadata ModelMetadata `json:"metadata"`
	Layers   []LayerConfig `json:"layers"`
}

// LayerConfig 神经网络层配置
type LayerConfig struct {
	Type      string      `json:"type"`      // dense, relu, sigmoid, tanh, etc.
	Weights   [][]float64 `json:"weights,omitempty"`
	Biases    []float64   `json:"biases,omitempty"`
	Activation string     `json:"activation,omitempty"`
}

// SVMModelData SVM模型数据
type SVMModelData struct {
	Metadata    ModelMetadata `json:"metadata"`
	SupportVectors [][]float64 `json:"support_vectors"`
	Weights     []float64     `json:"weights"`
	Bias        float64       `json:"bias"`
	Kernel      string        `json:"kernel"` // linear, rbf, polynomial
	Gamma       float64       `json:"gamma,omitempty"`
	Degree      int           `json:"degree,omitempty"`
}

// DecisionTreeModelData 决策树模型数据
type DecisionTreeModelData struct {
	Metadata ModelMetadata `json:"metadata"`
	Tree     TreeNode      `json:"tree"`
}

// TreeNode 决策树节点
type TreeNode struct {
	IsLeaf     bool        `json:"is_leaf"`
	Feature    int         `json:"feature,omitempty"`
	Threshold  float64     `json:"threshold,omitempty"`
	Value      interface{} `json:"value,omitempty"`
	Left       *TreeNode   `json:"left,omitempty"`
	Right      *TreeNode   `json:"right,omitempty"`
}

// LSTMModelData LSTM模型数据
type LSTMModelData struct {
	Metadata     ModelMetadata `json:"metadata"`
	HiddenSize   int           `json:"hidden_size"`
	NumLayers    int           `json:"num_layers"`
	Weights      map[string][][]float64 `json:"weights"`
	Biases       map[string][]float64  `json:"biases"`
}

// LoadModelFile 加载模型文件（JSON格式）
func LoadModelFile(modelPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("读取模型文件失败: %w", err)
	}

	var modelData map[string]interface{}
	if err := json.Unmarshal(data, &modelData); err != nil {
		return nil, fmt.Errorf("解析模型文件失败: %w", err)
	}

	return modelData, nil
}

// LoadLinearRegressionModel 加载线性回归模型
func LoadLinearRegressionModel(modelPath string) (*LinearRegressionModelData, error) {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("读取模型文件失败: %w", err)
	}

	var modelData LinearRegressionModelData
	if err := json.Unmarshal(data, &modelData); err != nil {
		return nil, fmt.Errorf("解析模型文件失败: %w", err)
	}

	return &modelData, nil
}

// LoadNeuralNetworkModel 加载神经网络模型
func LoadNeuralNetworkModel(modelPath string) (*NeuralNetworkModelData, error) {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("读取模型文件失败: %w", err)
	}

	var modelData NeuralNetworkModelData
	if err := json.Unmarshal(data, &modelData); err != nil {
		return nil, fmt.Errorf("解析模型文件失败: %w", err)
	}

	return &modelData, nil
}

// LoadSVMModel 加载SVM模型
func LoadSVMModel(modelPath string) (*SVMModelData, error) {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("读取模型文件失败: %w", err)
	}

	var modelData SVMModelData
	if err := json.Unmarshal(data, &modelData); err != nil {
		return nil, fmt.Errorf("解析模型文件失败: %w", err)
	}

	return &modelData, nil
}

// LoadDecisionTreeModel 加载决策树模型
func LoadDecisionTreeModel(modelPath string) (*DecisionTreeModelData, error) {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("读取模型文件失败: %w", err)
	}

	var modelData DecisionTreeModelData
	if err := json.Unmarshal(data, &modelData); err != nil {
		return nil, fmt.Errorf("解析模型文件失败: %w", err)
	}

	return &modelData, nil
}

// LoadLSTMModel 加载LSTM模型
func LoadLSTMModel(modelPath string) (*LSTMModelData, error) {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("读取模型文件失败: %w", err)
	}

	var modelData LSTMModelData
	if err := json.Unmarshal(data, &modelData); err != nil {
		return nil, fmt.Errorf("解析模型文件失败: %w", err)
	}

	return &modelData, nil
}
