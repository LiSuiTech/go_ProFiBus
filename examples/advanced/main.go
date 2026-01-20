package main

import (
	"fmt"
	"go_ProFiBus/collector"
	"go_ProFiBus/config"
	"go_ProFiBus/fusion"
	"go_ProFiBus/inference"
	"go_ProFiBus/logger"
	"go_ProFiBus/multimodal"
	"log"
	"time"
)

func main() {
	fmt.Println("=== go_ProFiBus 高级示例 ===")
	fmt.Println()

	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Printf("警告: 无法加载配置文件，使用默认配置: %v", err)
		cfg = config.GetDefaultConfig()
	}

	// 配置日志
	logLevel := config.GetLogLevel(cfg.Logging.Level)
	logger.SetLevel(logLevel)

	if cfg.Logging.EnableFile {
		err := logger.GetLogger().EnableFileLog(cfg.Logging.FilePath)
		if err != nil {
			log.Printf("警告: 无法启用文件日志: %v", err)
		}
	}

	fmt.Println("配置已加载")
	fmt.Printf("系统名称: %s\n", cfg.System.Name)
	fmt.Printf("版本: %s\n", cfg.System.Version)
	fmt.Printf("环境: %s\n", cfg.System.Environment)
	fmt.Println()

	// 运行各个示例
	runDataCollectionExample()
	runDataFusionExample()
	runInferenceExample()
	runMultimodalExample()
}

func runDataCollectionExample() {
	fmt.Println("1. 数据采集示例")
	fmt.Println("-------------------")

	// 创建模拟数据采集器
	// 注意：实际使用时需要提供真实的串口设备
	fmt.Println("提示: 数据采集需要实际的硬件设备")
	fmt.Println("此示例展示了如何配置采集器")
	fmt.Println()

	// 配置示例
	fmt.Println("采集器配置:")
	fmt.Println("- 采样率: 100ms")
	fmt.Println("- 缓冲区大小: 1000")
	fmt.Println("- 重试次数: 3")
	fmt.Println()
}

func runDataFusionExample() {
	fmt.Println("2. 数据融合示例")
	fmt.Println("-------------------")

	// 创建数据融合器
	fusionEngine := fusion.NewDataFusion(fusion.StrategyWeighted, 1*time.Second)

	// 创建模拟数据样本
	sample1 := &collector.DataSample{
		Timestamp: time.Now(),
		SourceID:  "sensor_1",
		Protocol:  "RS485",
		RawData:   []byte{0x01, 0x02, 0x03},
		ParsedData: map[string]interface{}{
			"temperature": 25.5,
			"pressure":    101.3,
		},
		Quality: 0.95,
	}

	sample2 := &collector.DataSample{
		Timestamp: time.Now(),
		SourceID:  "sensor_2",
		Protocol:  "I2C",
		RawData:   []byte{0x04, 0x05, 0x06},
		ParsedData: map[string]interface{}{
			"temperature": 25.8,
			"pressure":    101.1,
		},
		Quality: 0.90,
	}

	// 添加数据源
	fusionEngine.AddDataSource(sample1, 0.6)
	fusionEngine.AddDataSource(sample2, 0.4)

	// 执行融合
	fusedData, err := fusionEngine.Fuse()
	if err != nil {
		log.Printf("融合失败: %v", err)
		return
	}

	fmt.Printf("融合策略: 加权融合\n")
	fmt.Printf("数据源数量: %d\n", fusedData.SourceCount)
	fmt.Printf("融合后的数据: %+v\n", fusedData.Data)
	fmt.Printf("置信度: %.2f\n", fusedData.Confidence)
	fmt.Printf("质量: %.2f\n", fusedData.Quality)
	fmt.Println()
}

func runInferenceExample() {
	fmt.Println("3. 模型推理示例")
	fmt.Println("-------------------")

	// 创建推理引擎
	engine := inference.NewInferenceEngine()

	// 创建线性回归模型
	model := inference.NewLinearRegressionModel()
	model.SetWeights([]float64{0.5, 0.3, 0.2}, 1.0)

	// 注册模型
	err := engine.RegisterModel("temperature_predictor", model)
	if err != nil {
		log.Printf("注册模型失败: %v", err)
		return
	}

	// 准备输入数据
	inputData := []float64{25.0, 101.3, 0.6} // [温度, 压力, 湿度]
	inputTensor, err := inference.NewTensor([]int{1, 3}, inputData)
	if err != nil {
		log.Printf("创建张量失败: %v", err)
		return
	}

	// 执行推理
	output, err := engine.Predict("temperature_predictor", inputTensor)
	if err != nil {
		log.Printf("推理失败: %v", err)
		return
	}

	fmt.Printf("模型: temperature_predictor\n")
	fmt.Printf("输入: %v\n", inputData)
	fmt.Printf("预测输出: %v\n", output.Data)
	fmt.Println()
}

func runMultimodalExample() {
	fmt.Println("4. 多模态融合分析示例")
	fmt.Println("-------------------")

	// 创建多模态分析器
	analyzer := multimodal.NewMultiModalAnalyzer(multimodal.AlignmentLinearInterp)

	// 注册特征提取器
	sensorExtractor := multimodal.NewSensorFeatureExtractor()
	analyzer.RegisterFeatureExtractor(multimodal.ModalitySensor, sensorExtractor)

	timeSeriesExtractor := multimodal.NewTimeSeriesFeatureExtractor(10)
	analyzer.RegisterFeatureExtractor(multimodal.ModalityTimeSeries, timeSeriesExtractor)

	// 创建模拟的多模态数据
	now := time.Now()

	// 传感器模态数据
	sensorData := &multimodal.ModalityData{
		Type:      multimodal.ModalitySensor,
		Timestamp: now,
		SourceID:  "sensor_module",
		RawData: &collector.DataSample{
			Timestamp: now,
			SourceID:  "sensor_1",
			ParsedData: map[string]interface{}{
				"value": 25.5,
			},
			Quality: 0.95,
		},
		Embedding:  []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		Confidence: 0.95,
	}

	// 时序模态数据
	timeSeriesData := &multimodal.ModalityData{
		Type:       multimodal.ModalityTimeSeries,
		Timestamp:  now,
		SourceID:   "timeseries_module",
		RawData:    []float64{25.1, 25.2, 25.3, 25.4, 25.5},
		Embedding:  []float64{0.2, 0.3, 0.4, 0.5, 0.6},
		Confidence: 0.90,
	}

	// 添加模态数据
	analyzer.AddModalityData(sensorData)
	analyzer.AddModalityData(timeSeriesData)

	// 执行分析
	result, err := analyzer.Analyze(now, "")
	if err != nil {
		log.Printf("分析失败: %v", err)
		return
	}

	fmt.Printf("分析时间: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("模态数量: %d\n", result.ModalityCount)
	fmt.Printf("总体置信度: %.2f\n", result.Confidence)
	fmt.Printf("洞察: %+v\n", result.Insights)
	fmt.Println()

	fmt.Println("多模态分析完成!")
}
