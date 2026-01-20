package main

import (
	"fmt"
	"go_ProFiBus/anomaly"
	"go_ProFiBus/collector"
	"go_ProFiBus/event"
	"go_ProFiBus/inference"
	"go_ProFiBus/logger"
	"log"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("=== 人在环中的机器学习（Human-in-the-Loop ML）示例 ===")
	fmt.Println()

	// 设置日志
	logger.SetLevel(logger.INFO)

	// 1. 初始化系统组件
	fmt.Println("1. 初始化系统组件...")
	ruleEngine, detector, workflow, optimizer, learningLoop := initializeSystem()
	fmt.Println("   ✓ 规则引擎")
	fmt.Println("   ✓ 事件检测器")
	fmt.Println("   ✓ 标注工作流")
	fmt.Println("   ✓ 模型优化器")
	fmt.Println("   ✓ 持续学习循环")
	fmt.Println()

	// 2. 模拟数据流并检测异常
	fmt.Println("2. 模拟数据流并检测异常...")
	events := simulateDataStreamAndDetect(detector, 50)
	fmt.Printf("   检测到 %d 个异常事件\n", len(events))
	fmt.Println()

	// 3. 人工标注事件
	fmt.Println("3. 人工标注事件...")
	annotateEvents(workflow, events)
	fmt.Println()

	// 4. 查看事件库统计
	fmt.Println("4. 事件库统计...")
	showEventStoreStats(workflow.eventStore)
	fmt.Println()

	// 5. 模型训练和优化
	fmt.Println("5. 模型训练和优化...")
	trainModel(optimizer)
	fmt.Println()

	// 6. 启动持续学习循环
	fmt.Println("6. 启动持续学习循环...")
	learningLoop.Start()
	fmt.Println("   持续学习循环已启动（每1小时检查一次）")
	fmt.Println()

	// 7. 演示相似事件查找
	fmt.Println("7. 演示相似事件查找...")
	demonstrateSimilaritySearch(workflow.eventStore, events)
	fmt.Println()

	// 8. 显示系统统计信息
	fmt.Println("8. 系统统计信息")
	showSystemStats(ruleEngine, detector, workflow, optimizer)
	fmt.Println()

	// 9. 停止持续学习循环
	fmt.Println("9. 停止持续学习循环...")
	learningLoop.Stop()
	fmt.Println("   持续学习循环已停止")
	fmt.Println()

	fmt.Println("=== 示例完成 ===")
	fmt.Println()
	fmt.Println("📊 系统工作流程总结:")
	fmt.Println("   1. 数据采集 → 异常检测 → 生成待标注事件")
	fmt.Println("   2. 人工标注 → 确认/拒绝事件 → 保存到事件库")
	fmt.Println("   3. 事件库积累 → 模型训练 → 模型更新")
	fmt.Println("   4. 新模型 → 提高检测准确率 → 减少误报")
	fmt.Println("   5. 持续循环 → 系统不断优化 → 良性发展")
}

// initializeSystem 初始化系统
func initializeSystem() (
	*anomaly.RuleEngine,
	*event.EventDetector,
	*event.AnnotationWorkflow,
	*event.ModelOptimizer,
	*event.ContinuousLearningLoop,
) {
	// 创建规则引擎
	ruleEngine := anomaly.NewRuleEngine()

	// 添加一些检测规则
	rule1 := anomaly.NewThresholdRule(
		"temp_high",
		"温度过高",
		"temperature",
		">",
		80.0,
		anomaly.SeverityWarning,
	)
	ruleEngine.AddRule(rule1)

	rule2 := anomaly.NewRangeRule(
		"pressure_range",
		"压力异常",
		"pressure",
		90.0,
		110.0,
		anomaly.SeverityError,
	)
	ruleEngine.AddRule(rule2)

	rule3 := anomaly.NewStatisticalRule(
		"temp_outlier",
		"温度统计异常",
		"temperature",
		20,
		3.0,
		anomaly.SeverityWarning,
	)
	ruleEngine.AddRule(rule3)

	// 创建模式匹配器
	patternMatcher := anomaly.NewPatternMatcher(anomaly.MetricCosine, 0.8)

	// 创建事件检测器
	detector := event.NewEventDetector(
		ruleEngine,
		patternMatcher,
		0.7, // 阈值
		5*time.Second, // 上下文窗口
	)

	// 创建事件管理器和事件库
	eventManager := event.NewEventManager()
	eventStore := event.NewEventStore("data/events", true)

	// 创建标注工作流
	workflow := event.NewAnnotationWorkflow(eventManager, eventStore)

	// 注册标注员
	annotator := &event.Annotator{
		ID:          "annotator_001",
		Name:        "张工程师",
		Role:        "资深工程师",
		Permissions: []string{"annotate", "review"},
		CreatedAt:   time.Now(),
	}
	workflow.RegisterAnnotator(annotator)

	// 添加事件监听器
	detector.AddListener(&EventLogger{})
	detector.AddListener(&EventForwarder{eventManager: eventManager})

	// 创建推理引擎
	inferenceEngine := inference.NewInferenceEngine()

	// 创建模型优化器
	optimizerConfig := event.GetDefaultOptimizerConfig()
	optimizerConfig.MinTrainingSamples = 10 // 降低门槛用于演示
	optimizerConfig.RetrainingInterval = 1 * time.Hour
	optimizer := event.NewModelOptimizer(eventStore, inferenceEngine, optimizerConfig)

	// 创建持续学习循环
	learningLoop := event.NewContinuousLearningLoop(
		optimizer,
		"anomaly_classifier",
		1*time.Hour,
	)

	return ruleEngine, detector, workflow, optimizer, learningLoop
}

// simulateDataStreamAndDetect 模拟数据流并检测异常
func simulateDataStreamAndDetect(detector *event.EventDetector, count int) []*event.Event {
	allEvents := make([]*event.Event, 0)

	for i := 0; i < count; i++ {
		// 生成模拟数据
		sample := generateSimulatedSample(i)

		// 检测异常
		events, _ := detector.Detect(sample)

		if len(events) > 0 {
			allEvents = append(allEvents, events...)
			fmt.Printf("   [%d] 检测到异常: %s (分数: %.2f)\n",
				i, events[0].Description, events[0].Score)
		}

		time.Sleep(10 * time.Millisecond)
	}

	return allEvents
}

// generateSimulatedSample 生成模拟样本
func generateSimulatedSample(index int) *collector.DataSample {
	// 正常值范围：温度 65-75, 压力 95-105
	temperature := 70.0 + rand.Float64()*10 - 5
	pressure := 100.0 + rand.Float64()*10 - 5

	// 每10个样本中注入一个异常
	if index%10 == 0 {
		if rand.Float64() > 0.5 {
			temperature = 85.0 + rand.Float64()*10 // 温度异常
		} else {
			pressure = 120.0 + rand.Float64()*10 // 压力异常
		}
	}

	sample := &collector.DataSample{
		Timestamp: time.Now(),
		SourceID:  "sensor_001",
		Protocol:  "RS485",
		RawData:   []byte{0x01, 0x02, 0x03},
		ParsedData: map[string]interface{}{
			"temperature": temperature,
			"pressure":    pressure,
		},
		Quality: 0.95,
	}

	return sample
}

// annotateEvents 标注事件
func annotateEvents(workflow *event.AnnotationWorkflow, events []*event.Event) {
	annotatorID := "annotator_001"

	for i, evt := range events {
		// 创建标注任务
		task, _ := workflow.CreateAnnotationTask(evt.ID, annotatorID, 1)

		// 模拟人工标注（随机确认/拒绝）
		confirmed := rand.Float64() > 0.3 // 70%确认，30%拒绝

		labels := map[string]interface{}{
			"category": "temperature_anomaly",
			"severity": evt.Severity,
		}

		annotation := "已人工审核"
		if !confirmed {
			annotation = "误报，已拒绝"
		}

		workflow.AnnotateEvent(evt.ID, annotatorID, confirmed, annotation, labels)

		status := "✓ 已确认"
		if !confirmed {
			status = "✗ 已拒绝"
		}

		fmt.Printf("   [%d] %s - %s\n", i+1, status, evt.Description)

		_ = task
	}
}

// showEventStoreStats 显示事件库统计
func showEventStoreStats(eventStore *event.EventStore) {
	stats := eventStore.GetStats()

	fmt.Printf("   总事件数: %d\n", stats.TotalEvents)
	fmt.Println("   按类型分类:")
	for eventType, count := range stats.EventsByType {
		fmt.Printf("     - %s: %d\n", eventType, count)
	}
}

// trainModel 训练模型
func trainModel(optimizer *event.ModelOptimizer) {
	dataset, err := optimizer.PrepareTrainingData()
	if err != nil {
		fmt.Printf("   准备训练数据失败: %v\n", err)
		fmt.Println("   （需要更多标注数据）")
		return
	}

	result, err := optimizer.TrainModel("anomaly_classifier", dataset)
	if err != nil {
		fmt.Printf("   训练失败: %v\n", err)
		return
	}

	fmt.Printf("   训练样本: %d\n", result.TrainSamples)
	fmt.Printf("   验证样本: %d\n", result.ValSamples)
	fmt.Printf("   训练轮数: %d\n", len(result.Epochs))
	fmt.Printf("   最终准确率: %.2f%%\n", result.FinalAccuracy*100)
	fmt.Printf("   训练用时: %v\n", result.Duration)
}

// demonstrateSimilaritySearch 演示相似事件查找
func demonstrateSimilaritySearch(eventStore *event.EventStore, events []*event.Event) {
	if len(events) == 0 {
		fmt.Println("   没有事件可供查找")
		return
	}

	// 选择第一个事件作为查询
	queryEvent := events[0]
	similarEvents := eventStore.GetSimilarEvents(queryEvent, 0.7, 5)

	fmt.Printf("   查询事件: %s\n", queryEvent.Description)
	fmt.Printf("   找到 %d 个相似事件:\n", len(similarEvents))

	for i, evt := range similarEvents {
		fmt.Printf("     [%d] %s (时间: %s)\n",
			i+1, evt.Description, evt.Timestamp.Format("15:04:05"))
	}
}

// showSystemStats 显示系统统计
func showSystemStats(
	ruleEngine *anomaly.RuleEngine,
	detector *event.EventDetector,
	workflow *event.AnnotationWorkflow,
	optimizer *event.ModelOptimizer,
) {
	engineStats := ruleEngine.GetStats()
	detectorStats := detector.GetStats()
	workflowStats := workflow.GetStats()
	optimizerStats := optimizer.GetStats()

	fmt.Println("   规则引擎:")
	fmt.Printf("     - 总评估次数: %d\n", engineStats.TotalEvaluations)
	fmt.Printf("     - 违规次数: %d\n", engineStats.TotalViolations)

	fmt.Println("   事件检测器:")
	fmt.Printf("     - 总检测次数: %d\n", detectorStats.TotalDetections)
	fmt.Printf("     - 待处理事件: %d\n", detectorStats.PendingEvents)

	fmt.Println("   标注工作流:")
	fmt.Printf("     - 总任务数: %d\n", workflowStats.TotalTasks)
	fmt.Printf("     - 已完成任务: %d\n", workflowStats.CompletedTasks)
	fmt.Printf("     - 已确认事件: %d\n", workflowStats.ConfirmedEvents)
	fmt.Printf("     - 已拒绝事件: %d\n", workflowStats.RejectedEvents)

	fmt.Println("   模型优化器:")
	fmt.Printf("     - 重训练次数: %d\n", optimizerStats.TotalRetrainings)
	fmt.Printf("     - 最佳准确率: %.2f%%\n", optimizerStats.BestAccuracy*100)
	if !optimizerStats.LastRetraining.IsZero() {
		fmt.Printf("     - 最后训练: %s\n", optimizerStats.LastRetraining.Format("2006-01-02 15:04:05"))
	}
}

// EventLogger 事件日志监听器
type EventLogger struct{}

func (el *EventLogger) OnEventDetected(evt *event.Event) error {
	log.Printf("[事件检测] %s (分数: %.2f)", evt.Description, evt.Score)
	return nil
}

// EventForwarder 事件转发器
type EventForwarder struct {
	eventManager *event.EventManager
}

func (ef *EventForwarder) OnEventDetected(evt *event.Event) error {
	return ef.eventManager.AddEvent(evt)
}
