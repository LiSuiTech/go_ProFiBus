package training

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TrainingStatus 训练任务状态
type TrainingStatus string

const (
	TrainingStatusPending   TrainingStatus = "pending"
	TrainingStatusRunning   TrainingStatus = "running"
	TrainingStatusCompleted TrainingStatus = "completed"
	TrainingStatusFailed    TrainingStatus = "failed"
	TrainingStatusCancelled TrainingStatus = "cancelled"
)

// TrainingType 训练类型
type TrainingType string

const (
	TrainingTypeSupervised     TrainingType = "supervised"
	TrainingTypeUnsupervised   TrainingType = "unsupervised"
	TrainingTypeReinforcement  TrainingType = "reinforcement"
)

// DataSourceType 数据源类型
type DataSourceType string

const (
	DataSourceTypeDevice  DataSourceType = "device"
	DataSourceTypeChannel DataSourceType = "channel"
	DataSourceTypeFusion  DataSourceType = "fusion"
	DataSourceTypeExternal DataSourceType = "external"
)

// TrainingTask 训练任务实体
type TrainingTask struct {
	ID              string                 `json:"id"`
	ModelID         string                 `json:"model_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Status          TrainingStatus         `json:"status"`
	TrainingType    TrainingType           `json:"training_type"`
	DataSourceType  DataSourceType         `json:"data_source_type"`
	DataSourceIDs   []string               `json:"data_source_ids"`
	DataFields      []string               `json:"data_fields"`
	StartTime       *time.Time             `json:"start_time,omitempty"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Progress        float64                `json:"progress"`
	Epochs           int                   `json:"epochs"`
	BatchSize        int                   `json:"batch_size"`
	LearningRate     float64               `json:"learning_rate"`
	ValidationSplit  float64               `json:"validation_split"`
	Hyperparameters  map[string]interface{} `json:"hyperparameters"`
	TrainingConfig   map[string]interface{} `json:"training_config"`
	Metrics          map[string]interface{} `json:"metrics"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
	CreatedBy        string                 `json:"created_by,omitempty"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// NewTrainingTask 创建新的训练任务
func NewTrainingTask(modelID, name string, trainingType TrainingType, dataSourceType DataSourceType) *TrainingTask {
	now := time.Now()
	return &TrainingTask{
		ID:             uuid.New().String(),
		ModelID:        modelID,
		Name:           name,
		Status:         TrainingStatusPending,
		TrainingType:   trainingType,
		DataSourceType: dataSourceType,
		DataSourceIDs:  []string{},
		DataFields:     []string{},
		Progress:       0.0,
		Epochs:         100,
		BatchSize:      32,
		LearningRate:   0.001,
		ValidationSplit: 0.2,
		Hyperparameters: make(map[string]interface{}),
		TrainingConfig:  make(map[string]interface{}),
		Metrics:         make(map[string]interface{}),
		CreatedAt:       now,
		UpdatedAt:       now,
		Metadata:        make(map[string]interface{}),
	}
}

// SetStatus 设置任务状态
func (t *TrainingTask) SetStatus(status TrainingStatus) {
	t.Status = status
	t.UpdatedAt = time.Now()
}

// SetProgress 设置训练进度
func (t *TrainingTask) SetProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	t.Progress = progress
	t.UpdatedAt = time.Now()
}

// SetMetrics 设置训练指标
func (t *TrainingTask) SetMetrics(metrics map[string]interface{}) {
	t.Metrics = metrics
	t.UpdatedAt = time.Now()
}

// SetError 设置错误信息
func (t *TrainingTask) SetError(err error) {
	if err != nil {
		t.ErrorMessage = err.Error()
		t.Status = TrainingStatusFailed
	} else {
		t.ErrorMessage = ""
	}
	t.UpdatedAt = time.Now()
}

// Start 开始训练
func (t *TrainingTask) Start() {
	now := time.Now()
	t.StartTime = &now
	t.Status = TrainingStatusRunning
	t.UpdatedAt = now
}

// Complete 完成训练
func (t *TrainingTask) Complete() {
	now := time.Now()
	t.EndTime = &now
	t.Status = TrainingStatusCompleted
	t.Progress = 1.0
	t.UpdatedAt = now
}

// Cancel 取消训练
func (t *TrainingTask) Cancel() {
	now := time.Now()
	if t.EndTime == nil {
		t.EndTime = &now
	}
	t.Status = TrainingStatusCancelled
	t.UpdatedAt = now
}

// TrainingSample 训练数据样本
type TrainingSample struct {
	ID          string                 `json:"id"`
	TaskID      string                 `json:"task_id"`
	SampleIndex int                    `json:"sample_index"`
	InputData   map[string]interface{} `json:"input_data"`
	OutputData  map[string]interface{} `json:"output_data,omitempty"`
	Label       string                 `json:"label,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Quality     float64                `json:"quality"`
	CreatedAt   time.Time              `json:"created_at"`
}

// NewTrainingSample 创建新的训练样本
func NewTrainingSample(taskID string, sampleIndex int, inputData map[string]interface{}) *TrainingSample {
	return &TrainingSample{
		ID:          uuid.New().String(),
		TaskID:      taskID,
		SampleIndex: sampleIndex,
		InputData:   inputData,
		Quality:     1.0,
		Timestamp:   time.Now(),
		CreatedAt:   time.Now(),
	}
}

// SetOutput 设置输出数据
func (s *TrainingSample) SetOutput(outputData map[string]interface{}) {
	s.OutputData = outputData
}

// SetLabel 设置标签
func (s *TrainingSample) SetLabel(label string) {
	s.Label = label
}

// TrainingHistory 训练历史记录
type TrainingHistory struct {
	ID                string                 `json:"id"`
	TaskID            string                 `json:"task_id"`
	Epoch             int                    `json:"epoch"`
	Step              int                    `json:"step"`
	Loss              *float64               `json:"loss,omitempty"`
	Accuracy          *float64               `json:"accuracy,omitempty"`
	ValidationLoss    *float64               `json:"validation_loss,omitempty"`
	ValidationAccuracy *float64              `json:"validation_accuracy,omitempty"`
	LearningRate      *float64               `json:"learning_rate,omitempty"`
	Metrics           map[string]interface{} `json:"metrics"`
	Timestamp         time.Time              `json:"timestamp"`
}

// NewTrainingHistory 创建新的训练历史记录
func NewTrainingHistory(taskID string, epoch, step int) *TrainingHistory {
	return &TrainingHistory{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Epoch:     epoch,
		Step:      step,
		Metrics:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// SetLoss 设置损失值
func (h *TrainingHistory) SetLoss(loss float64) {
	h.Loss = &loss
}

// SetAccuracy 设置准确度
func (h *TrainingHistory) SetAccuracy(accuracy float64) {
	h.Accuracy = &accuracy
}

// SetValidationLoss 设置验证损失
func (h *TrainingHistory) SetValidationLoss(loss float64) {
	h.ValidationLoss = &loss
}

// SetValidationAccuracy 设置验证准确度
func (h *TrainingHistory) SetValidationAccuracy(accuracy float64) {
	h.ValidationAccuracy = &accuracy
}

// SetLearningRate 设置学习率
func (h *TrainingHistory) SetLearningRate(lr float64) {
	h.LearningRate = &lr
}

// AddMetric 添加指标
func (h *TrainingHistory) AddMetric(key string, value interface{}) {
	if h.Metrics == nil {
		h.Metrics = make(map[string]interface{})
	}
	h.Metrics[key] = value
}

// ToJSON 转换为JSON
func (t *TrainingTask) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// FromJSON 从JSON解析
func (t *TrainingTask) FromJSON(data []byte) error {
	return json.Unmarshal(data, t)
}
