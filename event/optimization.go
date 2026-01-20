package event

import (
	"fmt"
	"go_ProFiBus/inference"
	"go_ProFiBus/logger"
	"math"
	"sync"
	"time"
)

// ModelOptimizer 模型优化器
type ModelOptimizer struct {
	eventStore      *EventStore
	inferenceEngine *inference.InferenceEngine
	config          *OptimizerConfig
	mu              sync.RWMutex
	log             *logger.Logger
	stats           *OptimizerStats
}

// OptimizerConfig 优化器配置
type OptimizerConfig struct {
	MinTrainingSamples int           // 最小训练样本数
	RetrainingInterval time.Duration // 重训练间隔
	ValidationRatio    float64       // 验证集比例
	LearningRate       float64       // 学习率
	MaxEpochs          int           // 最大训练轮数
	EarlyStoppingRounds int          // 早停轮数
	AutoUpdate         bool          // 是否自动更新模型
}

// OptimizerStats 优化器统计
type OptimizerStats struct {
	TotalRetrainings uint64
	LastRetraining   time.Time
	LastAccuracy     float64
	BestAccuracy     float64
	TrainingSamples  int
	mu               sync.RWMutex
}

// NewModelOptimizer 创建模型优化器
func NewModelOptimizer(
	eventStore *EventStore,
	inferenceEngine *inference.InferenceEngine,
	config *OptimizerConfig,
) *ModelOptimizer {
	if config == nil {
		config = GetDefaultOptimizerConfig()
	}

	return &ModelOptimizer{
		eventStore:      eventStore,
		inferenceEngine: inferenceEngine,
		config:          config,
		log:             logger.GetLogger(),
		stats:           &OptimizerStats{},
	}
}

// GetDefaultOptimizerConfig 获取默认优化器配置
func GetDefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		MinTrainingSamples:  100,
		RetrainingInterval:  24 * time.Hour,
		ValidationRatio:     0.2,
		LearningRate:        0.01,
		MaxEpochs:           100,
		EarlyStoppingRounds: 10,
		AutoUpdate:          true,
	}
}

// PrepareTrainingData 准备训练数据
func (mo *ModelOptimizer) PrepareTrainingData() (*TrainingDataset, error) {
	mo.mu.RLock()
	defer mo.mu.RUnlock()

	// 获取所有已确认的事件
	allEvents := mo.eventStore.GetAllEvents()

	if len(allEvents) < mo.config.MinTrainingSamples {
		return nil, fmt.Errorf("训练样本不足: 需要 %d, 当前 %d",
			mo.config.MinTrainingSamples, len(allEvents))
	}

	mo.log.Info("准备训练数据: %d 个样本", len(allEvents))

	dataset := &TrainingDataset{
		Features: make([][]float64, 0, len(allEvents)),
		Labels:   make([]int, 0, len(allEvents)),
		Metadata: make([]map[string]interface{}, 0, len(allEvents)),
	}

	for _, event := range allEvents {
		// 提取特征
		if event.ContextData == nil || len(event.ContextData.Features) == 0 {
			continue
		}

		features := event.ContextData.Features
		dataset.Features = append(dataset.Features, features)

		// 标签：确认为1，拒绝为0
		label := 0
		if event.Status == EventStatusConfirmed {
			label = 1
		}
		dataset.Labels = append(dataset.Labels, label)

		// 元数据
		metadata := map[string]interface{}{
			"event_id":  event.ID,
			"type":      event.Type,
			"severity":  event.Severity,
			"timestamp": event.Timestamp,
		}
		dataset.Metadata = append(dataset.Metadata, metadata)
	}

	mo.log.Info("准备完成: %d 个有效样本", len(dataset.Features))
	return dataset, nil
}

// TrainingDataset 训练数据集
type TrainingDataset struct {
	Features [][]float64
	Labels   []int
	Metadata []map[string]interface{}
}

// Split 分割训练集和验证集
func (td *TrainingDataset) Split(validationRatio float64) (*TrainingDataset, *TrainingDataset) {
	validationSize := int(float64(len(td.Features)) * validationRatio)
	trainSize := len(td.Features) - validationSize

	train := &TrainingDataset{
		Features: td.Features[:trainSize],
		Labels:   td.Labels[:trainSize],
		Metadata: td.Metadata[:trainSize],
	}

	validation := &TrainingDataset{
		Features: td.Features[trainSize:],
		Labels:   td.Labels[trainSize:],
		Metadata: td.Metadata[trainSize:],
	}

	return train, validation
}

// TrainModel 训练模型
func (mo *ModelOptimizer) TrainModel(modelName string, dataset *TrainingDataset) (*TrainingResult, error) {
	mo.log.Info("开始训练模型: %s", modelName)

	// 分割数据集
	trainData, valData := dataset.Split(mo.config.ValidationRatio)

	mo.log.Info("训练集: %d 样本, 验证集: %d 样本",
		len(trainData.Features), len(valData.Features))

	// 创建新模型（这里使用简化的逻辑回归模型）
	model := mo.createLogisticRegressionModel(trainData)

	// 训练模型
	result := &TrainingResult{
		ModelName:    modelName,
		TrainSamples: len(trainData.Features),
		ValSamples:   len(valData.Features),
		StartTime:    time.Now(),
		Epochs:       make([]EpochMetrics, 0),
	}

	bestAccuracy := 0.0
	noImprovementCount := 0

	for epoch := 0; epoch < mo.config.MaxEpochs; epoch++ {
		// 训练一个epoch
		trainLoss := mo.trainEpoch(model, trainData)

		// 在验证集上评估
		valLoss, valAccuracy := mo.evaluate(model, valData)

		metrics := EpochMetrics{
			Epoch:         epoch + 1,
			TrainLoss:     trainLoss,
			ValLoss:       valLoss,
			ValAccuracy:   valAccuracy,
		}
		result.Epochs = append(result.Epochs, metrics)

		mo.log.Debug("Epoch %d: 训练损失=%.4f, 验证损失=%.4f, 准确率=%.4f",
			epoch+1, trainLoss, valLoss, valAccuracy)

		// 早停检查
		if valAccuracy > bestAccuracy {
			bestAccuracy = valAccuracy
			noImprovementCount = 0
		} else {
			noImprovementCount++
			if noImprovementCount >= mo.config.EarlyStoppingRounds {
				mo.log.Info("早停: %d 轮无改善", noImprovementCount)
				break
			}
		}
	}

	result.EndTime = time.Now()
	result.FinalAccuracy = bestAccuracy
	result.Duration = result.EndTime.Sub(result.StartTime)

	mo.log.Info("训练完成: 准确率=%.4f, 用时=%v", bestAccuracy, result.Duration)

	// 更新统计
	mo.stats.mu.Lock()
	mo.stats.TotalRetrainings++
	mo.stats.LastRetraining = time.Now()
	mo.stats.LastAccuracy = bestAccuracy
	if bestAccuracy > mo.stats.BestAccuracy {
		mo.stats.BestAccuracy = bestAccuracy
	}
	mo.stats.TrainingSamples = len(dataset.Features)
	mo.stats.mu.Unlock()

	// 如果启用自动更新，注册模型
	if mo.config.AutoUpdate {
		err := mo.inferenceEngine.RegisterModel(modelName, model)
		if err != nil {
			mo.log.Error("注册模型失败: %v", err)
			return result, err
		}
		mo.log.Info("模型已自动更新: %s", modelName)
	}

	return result, nil
}

// createLogisticRegressionModel 创建逻辑回归模型
func (mo *ModelOptimizer) createLogisticRegressionModel(dataset *TrainingDataset) *SimpleClassifier {
	if len(dataset.Features) == 0 {
		return nil
	}

	featureDim := len(dataset.Features[0])
	return &SimpleClassifier{
		Weights: make([]float64, featureDim),
		Bias:    0.0,
	}
}

// trainEpoch 训练一个epoch
func (mo *ModelOptimizer) trainEpoch(model *SimpleClassifier, dataset *TrainingDataset) float64 {
	totalLoss := 0.0

	for i, features := range dataset.Features {
		label := float64(dataset.Labels[i])

		// 前向传播
		pred := model.Predict(features)

		// 计算损失（交叉熵）
		loss := -label*math.Log(pred+1e-10) - (1-label)*math.Log(1-pred+1e-10)
		totalLoss += loss

		// 反向传播和更新权重
		error := pred - label
		for j := range model.Weights {
			model.Weights[j] -= mo.config.LearningRate * error * features[j]
		}
		model.Bias -= mo.config.LearningRate * error
	}

	return totalLoss / float64(len(dataset.Features))
}

// evaluate 评估模型
func (mo *ModelOptimizer) evaluate(model *SimpleClassifier, dataset *TrainingDataset) (float64, float64) {
	totalLoss := 0.0
	correct := 0

	for i, features := range dataset.Features {
		label := float64(dataset.Labels[i])
		pred := model.Predict(features)

		// 计算损失
		loss := -label*math.Log(pred+1e-10) - (1-label)*math.Log(1-pred+1e-10)
		totalLoss += loss

		// 计算准确率
		predictedLabel := 0
		if pred >= 0.5 {
			predictedLabel = 1
		}
		if predictedLabel == dataset.Labels[i] {
			correct++
		}
	}

	avgLoss := totalLoss / float64(len(dataset.Features))
	accuracy := float64(correct) / float64(len(dataset.Features))

	return avgLoss, accuracy
}

// SimpleClassifier 简单分类器
type SimpleClassifier struct {
	Weights []float64
	Bias    float64
}

// Predict 预测
func (sc *SimpleClassifier) Predict(features []float64) float64 {
	z := sc.Bias
	for i, w := range sc.Weights {
		if i < len(features) {
			z += w * features[i]
		}
	}
	// Sigmoid 激活
	return 1.0 / (1.0 + math.Exp(-z))
}

// Load 加载模型（inference.Model接口）
func (sc *SimpleClassifier) Load(modelPath string) error {
	return nil
}

// Predict 推理（inference.Model接口）
func (sc *SimpleClassifier) PredictTensor(input *inference.Tensor) (*inference.Tensor, error) {
	if len(input.Shape) != 2 {
		return nil, fmt.Errorf("期望2维输入")
	}

	batchSize := input.Shape[0]
	output := make([]float64, batchSize)

	features := input.Shape[1]
	for i := 0; i < batchSize; i++ {
		z := sc.Bias
		for j := 0; j < features; j++ {
			z += sc.Weights[j] * input.Data[i*features+j]
		}
		output[i] = 1.0 / (1.0 + math.Exp(-z))
	}

	return inference.NewTensor([]int{batchSize, 1}, output)
}

// GetType 获取模型类型
func (sc *SimpleClassifier) GetType() inference.ModelType {
	return inference.ModelLogisticRegression
}

// GetInputShape 获取输入形状
func (sc *SimpleClassifier) GetInputShape() []int {
	return []int{-1, len(sc.Weights)}
}

// GetOutputShape 获取输出形状
func (sc *SimpleClassifier) GetOutputShape() []int {
	return []int{-1, 1}
}

// TrainingResult 训练结果
type TrainingResult struct {
	ModelName      string
	TrainSamples   int
	ValSamples     int
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	FinalAccuracy  float64
	Epochs         []EpochMetrics
}

// EpochMetrics 每轮指标
type EpochMetrics struct {
	Epoch       int
	TrainLoss   float64
	ValLoss     float64
	ValAccuracy float64
}

// ShouldRetrain 判断是否需要重训练
func (mo *ModelOptimizer) ShouldRetrain() bool {
	mo.stats.mu.RLock()
	defer mo.stats.mu.RUnlock()

	// 检查是否到了重训练时间
	if time.Since(mo.stats.LastRetraining) < mo.config.RetrainingInterval {
		return false
	}

	// 检查是否有足够的新样本
	allEvents := mo.eventStore.GetAllEvents()
	if len(allEvents) < mo.config.MinTrainingSamples {
		return false
	}

	// 如果从未训练过，应该训练
	if mo.stats.LastRetraining.IsZero() {
		return true
	}

	// 如果新样本增加了一定比例，应该重训练
	newSamplesRatio := float64(len(allEvents)) / float64(mo.stats.TrainingSamples)
	return newSamplesRatio >= 1.2 // 增加了20%
}

// AutoRetrain 自动重训练
func (mo *ModelOptimizer) AutoRetrain(modelName string) error {
	if !mo.ShouldRetrain() {
		return nil
	}

	mo.log.Info("触发自动重训练: %s", modelName)

	dataset, err := mo.PrepareTrainingData()
	if err != nil {
		return err
	}

	result, err := mo.TrainModel(modelName, dataset)
	if err != nil {
		return err
	}

	mo.log.Info("自动重训练完成: 准确率=%.4f", result.FinalAccuracy)
	return nil
}

// GetStats 获取统计信息
func (mo *ModelOptimizer) GetStats() OptimizerStats {
	mo.stats.mu.RLock()
	defer mo.stats.mu.RUnlock()
	return *mo.stats
}

// ContinuousLearningLoop 持续学习循环
type ContinuousLearningLoop struct {
	optimizer   *ModelOptimizer
	modelName   string
	interval    time.Duration
	running     bool
	stopChan    chan struct{}
	mu          sync.Mutex
	log         *logger.Logger
}

// NewContinuousLearningLoop 创建持续学习循环
func NewContinuousLearningLoop(
	optimizer *ModelOptimizer,
	modelName string,
	checkInterval time.Duration,
) *ContinuousLearningLoop {
	return &ContinuousLearningLoop{
		optimizer: optimizer,
		modelName: modelName,
		interval:  checkInterval,
		stopChan:  make(chan struct{}),
		log:       logger.GetLogger(),
	}
}

// Start 启动持续学习
func (cll *ContinuousLearningLoop) Start() {
	cll.mu.Lock()
	if cll.running {
		cll.mu.Unlock()
		return
	}
	cll.running = true
	cll.mu.Unlock()

	go cll.run()
	cll.log.Info("持续学习循环已启动: %s", cll.modelName)
}

// Stop 停止持续学习
func (cll *ContinuousLearningLoop) Stop() {
	cll.mu.Lock()
	defer cll.mu.Unlock()

	if !cll.running {
		return
	}

	close(cll.stopChan)
	cll.running = false
	cll.log.Info("持续学习循环已停止: %s", cll.modelName)
}

// run 运行循环
func (cll *ContinuousLearningLoop) run() {
	ticker := time.NewTicker(cll.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cll.optimizer.AutoRetrain(cll.modelName); err != nil {
				cll.log.Error("自动重训练失败: %v", err)
			}
		case <-cll.stopChan:
			return
		}
	}
}

// IsRunning 检查是否正在运行
func (cll *ContinuousLearningLoop) IsRunning() bool {
	cll.mu.Lock()
	defer cll.mu.Unlock()
	return cll.running
}
