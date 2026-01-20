package main

import (
	"fmt"
	"go_ProFiBus/anomaly"
	"go_ProFiBus/collector"
	"go_ProFiBus/concurrent"
	"go_ProFiBus/logger"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("=== 并发数据采集与处理示例 ===")
	fmt.Println()

	// 设置日志级别
	logger.SetLevel(logger.INFO)

	// 1. 创建分区有序采集器
	fmt.Println("1. 创建分区有序采集器...")
	collector := concurrent.NewPartitionedOrderedCollector(concurrent.PartitionedCollectorConfig{
		WindowSize:     5 * time.Second,
		MaxBufferSize:  1000,
		WorkerCount:    4,
		OutputChanSize: 1000,
	})

	if err := collector.Start(); err != nil {
		fmt.Printf("启动采集器失败: %v\n", err)
		return
	}
	defer collector.Stop()
	fmt.Println("   ✓ 分区有序采集器已启动")
	fmt.Println()

	// 2. 创建批量写入器
	fmt.Println("2. 创建批量写入器...")
	writeCount := 0
	batchWriter := concurrent.NewBatchWriter(concurrent.BatchWriterConfig{
		Name:          "SensorData",
		BatchSize:     10,
		FlushInterval: 2 * time.Second,
		WorkerCount:   2,
		WriteFn: func(items []interface{}) error {
			writeCount++
			fmt.Printf("   📝 批量写入 #%d: %d 个样本\n", writeCount, len(items))
			// 这里模拟数据库写入
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	})

	if err := batchWriter.Start(); err != nil {
		fmt.Printf("启动批量写入器失败: %v\n", err)
		return
	}
	defer batchWriter.Stop()
	fmt.Println("   ✓ 批量写入器已启动 (批量大小: 10, 刷新间隔: 2秒)")
	fmt.Println()

	// 3. 创建规则引擎（支持并发评估）
	fmt.Println("3. 创建规则引擎...")
	ruleEngine := anomaly.NewRuleEngine()

	// 添加多条规则
	rule1 := anomaly.NewThresholdRule(
		"temp_high", "温度过高", "temperature", ">", 80.0, anomaly.SeverityWarning)
	rule2 := anomaly.NewRangeRule(
		"pressure_range", "压力异常", "pressure", 90.0, 110.0, anomaly.SeverityError)
	rule3 := anomaly.NewStatisticalRule(
		"temp_outlier", "温度统计异常", "temperature", 20, 3.0, anomaly.SeverityWarning)

	ruleEngine.AddRule(rule1)
	ruleEngine.AddRule(rule2)
	ruleEngine.AddRule(rule3)

	fmt.Println("   ✓ 已添加 3 条异常检测规则")
	fmt.Println()

	// 4. 模拟多个数据源并发采集
	fmt.Println("4. 模拟 3 个数据源并发采集数据...")
	sources := []string{"sensor_001", "sensor_002", "sensor_003"}

	// 启动数据生成器
	for _, sourceID := range sources {
		go dataGenerator(sourceID, collector)
	}

	// 5. 启动数据处理流程
	fmt.Println("5. 启动数据处理流程...")
	fmt.Println()

	go func() {
		anomalyCount := 0
		processedCount := 0

		for sample := range collector.GetOutput() {
			processedCount++

			// 使用并发规则评估
			results := ruleEngine.EvaluateConcurrent(sample)

			if len(results) > 0 {
				anomalyCount++
				fmt.Printf("   🚨 异常检测 [%s]: %d 条规则触发\n",
					sample.SourceID, len(results))
				for _, result := range results {
					fmt.Printf("      - %s (分数: %.2f)\n",
						result.Message, result.Score)
				}
			}

			// 提交到批量写入器
			if err := batchWriter.Write(sample); err != nil {
				fmt.Printf("写入失败: %v\n", err)
			}

			// 定期显示统计
			if processedCount%50 == 0 {
				stats := collector.GetStats()
				fmt.Printf("\n📊 统计信息 (已处理: %d 样本, 异常: %d)\n",
					processedCount, anomalyCount)
				fmt.Printf("   - 分区数: %d\n", stats["partition_count"])
				fmt.Printf("   - 缓冲区状态: %v\n", stats["partition_sizes"])
				fmt.Printf("   - 写入器缓冲: %d\n", batchWriter.GetBufferSize())
				fmt.Println()
			}
		}
	}()

	// 运行30秒
	fmt.Println("运行30秒后自动停止...")
	fmt.Println("=" + string(make([]byte, 50)) + "=")
	fmt.Println()

	time.Sleep(30 * time.Second)

	// 6. 显示最终统计
	fmt.Println()
	fmt.Println("=" + string(make([]byte, 50)) + "=")
	fmt.Println("6. 最终统计信息")
	fmt.Println()

	engineStats := ruleEngine.GetStats()
	fmt.Printf("规则引擎:\n")
	fmt.Printf("   - 总评估次数: %d\n", engineStats.TotalEvaluations)
	fmt.Printf("   - 违规次数: %d\n", engineStats.TotalViolations)
	fmt.Println()

	writerStats := batchWriter.GetStats()
	fmt.Printf("批量写入器:\n")
	fmt.Printf("   - 总接收项目: %d\n", writerStats.TotalItems.Load())
	fmt.Printf("   - 总批次: %d\n", writerStats.TotalBatches.Load())
	fmt.Printf("   - 成功写入: %d\n", writerStats.TotalWrites.Load())
	fmt.Printf("   - 失败写入: %d\n", writerStats.FailedWrites.Load())
	fmt.Println()

	fmt.Println("=== 示例完成 ===")
	fmt.Println()
	fmt.Println("✨ 并发处理特性演示:")
	fmt.Println("   1. ✅ 分区有序采集 - 每个数据源独立维护顺序")
	fmt.Println("   2. ✅ 并发规则评估 - 多条规则并行评估")
	fmt.Println("   3. ✅ 批量数据写入 - 自动累积和刷新")
	fmt.Println("   4. ✅ 多源时间戳归并 - 保证全局时序")
}

// dataGenerator 数据生成器（模拟传感器数据）
func dataGenerator(sourceID string, collector *concurrent.PartitionedOrderedCollector) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// 生成模拟数据
		temperature := 70.0 + rand.Float64()*20 - 10 // 60-80
		pressure := 100.0 + rand.Float64()*20 - 10   // 90-110

		// 随机注入异常
		if rand.Float64() < 0.1 { // 10%概率异常
			if rand.Float64() > 0.5 {
				temperature = 85.0 + rand.Float64()*10 // 温度过高
			} else {
				pressure = 120.0 + rand.Float64()*10 // 压力异常
			}
		}

		sample := &collector.DataSample{
			Timestamp: time.Now(),
			SourceID:  sourceID,
			Protocol:  "RS485",
			ParsedData: map[string]interface{}{
				"temperature": temperature,
				"pressure":    pressure,
			},
			Quality: 0.95,
		}

		if err := collector.Submit(sample); err != nil {
			fmt.Printf("提交样本失败 [%s]: %v\n", sourceID, err)
		}
	}
}
