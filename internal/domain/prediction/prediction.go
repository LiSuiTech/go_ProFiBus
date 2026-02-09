package prediction

import (
	"github.com/google/uuid"
	"time"
)

// Prediction 预测结果实体
type Prediction struct {
	ID              string
	ModelID         string
	DeviceID        string
	ChannelID       string
	PredictionType  PredictionType
	FieldName       string
	PredictedValue  float64
	Confidence      float64 // 置信度 0-1
	ActualValue     *float64 // 实际值（用于对比）
	ErrorRate       *float64 // 误差率
	TimeRangeStart  time.Time
	TimeRangeEnd    time.Time
	Metadata        map[string]interface{}
	CreatedAt       time.Time
}

// PredictionType 预测类型
type PredictionType string

const (
	PredictionTypeForecast   PredictionType = "forecast"   // 趋势预测
	PredictionTypeAnomaly   PredictionType = "anomaly"    // 异常预测
	PredictionTypeTrend     PredictionType = "trend"       // 趋势分析
	PredictionTypePerformance PredictionType = "performance" // 性能预测
)

// NewPrediction 创建新预测结果
func NewPrediction(modelID, deviceID string, predictionType PredictionType, predictedValue float64) *Prediction {
	return &Prediction{
		ID:             generateID(),
		ModelID:        modelID,
		DeviceID:       deviceID,
		PredictionType: predictionType,
		PredictedValue: predictedValue,
		Confidence:     0.0,
		Metadata:       make(map[string]interface{}),
		CreatedAt:      time.Now(),
	}
}

// SetTimeRange 设置预测时间范围
func (p *Prediction) SetTimeRange(start, end time.Time) {
	p.TimeRangeStart = start
	p.TimeRangeEnd = end
}

// SetConfidence 设置置信度
func (p *Prediction) SetConfidence(confidence float64) {
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}
	p.Confidence = confidence
}

// SetActualValue 设置实际值并计算误差率
func (p *Prediction) SetActualValue(actualValue float64) {
	p.ActualValue = &actualValue
	if p.PredictedValue != 0 {
		errorRate := abs(p.PredictedValue-actualValue) / abs(p.PredictedValue)
		p.ErrorRate = &errorRate
	}
}

// IsAccurate 检查预测是否准确（误差率 < 10%）
func (p *Prediction) IsAccurate() bool {
	if p.ErrorRate == nil {
		return false
	}
	return *p.ErrorRate < 0.1
}

// generateID 生成预测ID
func generateID() string {
	return "pred_" + uuid.New().String()
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// PredictionModel 预测模型实体
type PredictionModel struct {
	ID              string
	Name            string
	Description     string
	Type            ModelType
	Version         string
	FilePath        string
	Status          ModelStatus
	Accuracy        *float64 // 模型准确度
	TrainingSamples int      // 训练样本数
	Metadata        map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeployedAt      *time.Time
}

// ModelType 模型类型
type ModelType string

const (
	ModelTypeLinearRegression ModelType = "linear_regression"
	ModelTypeNeuralNetwork    ModelType = "neural_network"
	ModelTypeSVM              ModelType = "svm"
	ModelTypeDecisionTree     ModelType = "decision_tree"
	ModelTypeLSTM             ModelType = "lstm"
	ModelTypeCustom           ModelType = "custom"
)

// ModelStatus 模型状态
type ModelStatus string

const (
	ModelStatusDraft     ModelStatus = "draft"
	ModelStatusTraining  ModelStatus = "training"
	ModelStatusDeployed  ModelStatus = "deployed"
	ModelStatusArchived  ModelStatus = "archived"
)

// NewPredictionModel 创建新预测模型
func NewPredictionModel(id, name string, modelType ModelType) *PredictionModel {
	return &PredictionModel{
		ID:              id,
		Name:            name,
		Type:            modelType,
		Version:         "1.0.0",
		Status:          ModelStatusDraft,
		TrainingSamples: 0,
		Metadata:        make(map[string]interface{}),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// Deploy 部署模型
func (m *PredictionModel) Deploy() {
	now := time.Now()
	m.Status = ModelStatusDeployed
	m.DeployedAt = &now
	m.UpdatedAt = now
}

// Archive 归档模型
func (m *PredictionModel) Archive() {
	m.Status = ModelStatusArchived
	m.UpdatedAt = time.Now()
}

// SetAccuracy 设置模型准确度
func (m *PredictionModel) SetAccuracy(accuracy float64) {
	if accuracy < 0 {
		accuracy = 0
	}
	if accuracy > 1 {
		accuracy = 1
	}
	m.Accuracy = &accuracy
	m.UpdatedAt = time.Now()
}

// IsDeployed 检查模型是否已部署
func (m *PredictionModel) IsDeployed() bool {
	return m.Status == ModelStatusDeployed
}
