package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_ProFiBus/fusion"
	"go_ProFiBus/inference"
	"go_ProFiBus/internal/application/control"
	"go_ProFiBus/internal/application/processor"
	"go_ProFiBus/internal/infrastructure/actuator"
	"go_ProFiBus/internal/infrastructure/analyzer"
	"go_ProFiBus/pkg/interfaces"
)

// 本示例展示：AI异常检测 + 自动设备控制完整工作流
//
// 业务场景：工厂电机监控与保护
//
// 数据流：
// [温度传感器] [振动传感器] [电流传感器]
//       ↓           ↓           ↓
//     [数据融合] ← 多传感器数据融合
//       ↓
//   [特征提取] ← 提取ML特征
//       ↓
//   [AI分析器] ← 异常检测
//       ↓
//   [控制策略] ← 评估是否需要控制
//       ↓
//   [执行器] ← 执行设备控制（紧急停止/关闭/降速）
//
// 控制规则：
// 1. 温度>90°C → 触发优雅关闭
// 2. 振动>5.0 → 触发暂停
// 3. AI异常分数>0.9 → 触发紧急停止

func main() {
	fmt.Println("=== AI异常检测 + 自动设备控制示例 ===")
	fmt.Println("业务场景：工厂电机监控与保护\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ========== 第1步：创建设备控制器 ==========
	fmt.Println("--- 第1步：设置设备控制系统 ---")

	deviceController := control.NewDeviceController()

	// 创建MQTT执行器（用于向设备发送控制命令）
	mqttActuatorConfig := &actuator.MQTTActuatorConfig{
		ActuatorID:   "mqtt_controller",
		Name:         "MQTT设备控制器",
		Broker:       "tcp://localhost:1883",
		QoS:          1,
		ControlTopic: "factory/control",
		DeviceTopics: map[string]string{
			"motor-001": "factory/control/motor-001",
			"motor-002": "factory/control/motor-002",
		},
	}

	mqttActuator, err := actuator.NewMQTTActuator(mqttActuatorConfig)
	if err != nil {
		log.Printf("警告: 创建MQTT执行器失败: %v (将跳过MQTT控制)", err)
	} else {
		// 初始化执行器（连接MQTT）
		err = mqttActuator.Initialize(mqttActuatorConfig)
		if err != nil {
			log.Printf("警告: MQTT执行器初始化失败: %v (将跳过MQTT控制)", err)
			mqttActuator = nil
		} else {
			deviceController.RegisterActuator(mqttActuator)
			fmt.Println("  ✓ MQTT执行器已注册")
		}
	}

	// 创建OPC-UA执行器（用于调用PLC方法）
	opcuaActuatorConfig := &actuator.OPCUAActuatorConfig{
		ActuatorID:     "opcua_controller",
		Name:           "OPC-UA设备控制器",
		Endpoint:       "opc.tcp://localhost:4840",
		SecurityPolicy: "None",
		SecurityMode:   "None",
		DeviceNodes: map[string]*actuator.DeviceNodeConfig{
			"motor-001": {
				ObjectID:            "ns=2;s=Motor001",
				EmergencyStopMethod: "ns=2;s=Motor001.EmergencyStop",
				ShutdownMethod:      "ns=2;s=Motor001.Shutdown",
				StartMethod:         "ns=2;s=Motor001.Start",
				ControlNodes: map[string]string{
					"speed": "ns=2;s=Motor001.Speed",
				},
			},
		},
	}

	opcuaActuator, err := actuator.NewOPCUAActuator(opcuaActuatorConfig)
	if err != nil {
		log.Printf("警告: 创建OPC-UA执行器失败: %v (将跳过OPC-UA控制)", err)
	} else {
		err = opcuaActuator.Initialize(opcuaActuatorConfig)
		if err != nil {
			log.Printf("警告: OPC-UA执行器初始化失败: %v (将跳过OPC-UA控制)", err)
			opcuaActuator = nil
		} else {
			deviceController.RegisterActuator(opcuaActuator)
			fmt.Println("  ✓ OPC-UA执行器已注册")
		}
	}

	if mqttActuator == nil && opcuaActuator == nil {
		fmt.Println("\n  ⚠️  没有可用的执行器，将只进行模拟演示\n")
	}

	// ========== 第2步：配置控制策略和规则 ==========
	fmt.Println("\n--- 第2步：配置控制策略和规则 ---")

	policy := control.NewRuleBasedPolicy("motor_protection", "电机保护策略")

	// 规则1：紧急停止（AI检测到严重异常）
	emergencyStopRule := control.CreateEmergencyStopRule("motor-001", 0.9)
	policy.AddRule(emergencyStopRule)
	fmt.Println("  ✓ 已添加规则: 紧急停止 (AI分数>0.9)")

	// 规则2：高温关闭
	highTempRule := control.CreateHighTempShutdownRule("motor-001", 90.0)
	policy.AddRule(highTempRule)
	fmt.Println("  ✓ 已添加规则: 高温关闭 (温度>90°C)")

	// 规则3：过度振动保护
	vibrationRule := control.CreateExcessiveVibrationRule("motor-001", 5.0)
	policy.AddRule(vibrationRule)
	fmt.Println("  ✓ 已添加规则: 过度振动保护 (振动>5.0)")

	// 注册策略到设备控制器
	deviceController.RegisterPolicy(policy)

	// ========== 第3步：创建AI分析Pipeline ==========
	fmt.Println("\n--- 第3步：创建AI分析Pipeline ---")

	// 融合处理器
	fusionProcessor := processor.NewFusionProcessor("sensor-fusion", fusion.StrategyWeighted)
	fusionProcessor.SetSourceWeight("temp-sensor", 0.4)
	fusionProcessor.SetSourceWeight("vib-sensor", 0.3)
	fusionProcessor.SetSourceWeight("current-sensor", 0.3)
	fmt.Println("  ✓ 数据融合处理器")

	// 特征提取器
	featureExtractor := processor.NewFeatureExtractor("ml-features")
	featureExtractor.AddFeature("numeric_mean", nil)
	featureExtractor.AddFeature("numeric_stddev", nil)
	featureExtractor.AddFeature("quality", nil)
	fmt.Println("  ✓ 特征提取器")

	// ML分析器
	mlAnalyzer := analyzer.NewMLAnalyzer("motor-anomaly-detector", "motor-nn-model")
	mlAnalyzer.Configure(map[string]interface{}{
		"model_type":        "neural_network",
		"anomaly_threshold": 0.7,
		"confidence_min":    0.6,
	})

	// 创建并训练模型
	model := inference.NewNeuralNetworkModel([]int{10, 20, 10, 2})
	mlAnalyzer.UpdateModel(model)
	fmt.Println("  ✓ ML异常检测器")

	// 包装为带控制功能的分析器
	controlAnalyzer := analyzer.NewControlAnalyzer("motor-control-analyzer", mlAnalyzer, deviceController)
	fmt.Println("  ✓ 控制分析器（集成自动控制）")

	// ========== 第4步：开始模拟数据和分析 ==========
	fmt.Println("\n--- 第4步：开始监控和保护 ---\n")

	// 模拟电机运行
	motorStatus := &MotorStatus{
		Temperature: 75.0, // 初始温度
		Vibration:   2.0,  // 初始振动
		Current:     45.0, // 初始电流
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	iterationCount := 0

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("🔍 监控中... (按 Ctrl+C 退出)\n")

	for {
		select {
		case <-ctx.Done():
			goto cleanup
		case <-sigChan:
			goto cleanup
		case <-ticker.C:
			iterationCount++

			// 模拟电机状态变化
			motorStatus.SimulateOperation(iterationCount)

			// 创建数据样本
			sample := createMotorSample("motor-001", motorStatus)

			fmt.Printf("\n[时刻 %d] %s\n", iterationCount, time.Now().Format("15:04:05"))
			fmt.Printf("  📊 电机状态: 温度=%.1f°C, 振动=%.2f, 电流=%.1fA\n",
				motorStatus.Temperature,
				motorStatus.Vibration,
				motorStatus.Current)

			// 通过pipeline处理
			processedSample, _ := fusionProcessor.Process(ctx, sample)
			processedSample, _ = featureExtractor.Process(ctx, processedSample)

			// 执行分析和控制（并行）
			results, err := controlAnalyzer.Analyze(ctx, processedSample)
			if err != nil {
				log.Printf("  ❌ 分析失败: %v", err)
				continue
			}

			// 显示分析结果
			for _, result := range results {
				fmt.Printf("  🤖 AI检测: 分数=%.3f, 严重度=%d, %s\n",
					result.Score,
					result.Severity,
					result.Message)

				if isAnomaly, ok := result.Details["is_anomaly"].(bool); ok && isAnomaly {
					fmt.Printf("     ⚠️  检测到异常! 控制系统将自动响应...\n")
				}
			}
		}
	}

cleanup:
	fmt.Println("\n\n--- 正在关闭 ---")
	cancel()

	// 清理资源
	if mqttActuator != nil {
		mqttActuator.Close()
	}
	if opcuaActuator != nil {
		opcuaActuator.Close()
	}

	// 显示统计
	fmt.Println("\n--- 最终统计 ---")
	stats := mlAnalyzer.GetStats()
	fmt.Printf("总样本数: %d\n", stats.TotalSamples)
	fmt.Printf("检测异常: %d\n", stats.AnomaliesDetected)
	fmt.Printf("异常率: %.2f%%\n", mlAnalyzer.GetAnomalyRate()*100)

	fmt.Println("\n✓ 示例完成!")
}

// MotorStatus 电机状态
type MotorStatus struct {
	Temperature float64
	Vibration   float64
	Current     float64
}

// SimulateOperation 模拟电机运行
func (m *MotorStatus) SimulateOperation(iteration int) {
	// 正常运行时的小幅波动
	m.Temperature += (rand.Float64() - 0.5) * 2
	m.Vibration += (rand.Float64() - 0.5) * 0.3
	m.Current += (rand.Float64() - 0.5) * 3

	// 模拟异常场景
	if iteration == 10 {
		// 第10次：温度突然升高
		m.Temperature = 92.0
		fmt.Println("\n  🔥 模拟场景: 温度异常升高!")
	} else if iteration == 15 {
		// 第15次：振动突然增大
		m.Vibration = 5.5
		fmt.Println("\n  📳 模拟场景: 振动异常增大!")
	} else if iteration == 20 {
		// 第20次：多重异常（应触发紧急停止）
		m.Temperature = 95.0
		m.Vibration = 6.0
		m.Current = 80.0
		fmt.Println("\n  🚨 模拟场景: 多重严重异常!")
	}

	// 限制范围
	if m.Temperature < 60 {
		m.Temperature = 60
	}
	if m.Vibration < 1.0 {
		m.Vibration = 1.0
	}
	if m.Current < 30 {
		m.Current = 30
	}
}

// createMotorSample 创建电机数据样本
func createMotorSample(motorID string, status *MotorStatus) interfaces.DataSample {
	data := map[string]interface{}{
		"temperature": status.Temperature,
		"vibration":   status.Vibration,
		"current":     status.Current,
	}

	return &motorDataSample{
		timestamp: time.Now(),
		sourceID:  motorID,
		data:      data,
		quality:   0.95,
		metadata:  make(map[string]interface{}),
	}
}

// motorDataSample 电机数据样本实现
type motorDataSample struct {
	timestamp time.Time
	sourceID  string
	data      map[string]interface{}
	quality   float64
	metadata  map[string]interface{}
}

func (s *motorDataSample) GetTimestamp() time.Time                { return s.timestamp }
func (s *motorDataSample) GetSourceID() string                    { return s.sourceID }
func (s *motorDataSample) GetData() map[string]interface{}        { return s.data }
func (s *motorDataSample) GetQuality() float64                    { return s.quality }
func (s *motorDataSample) GetMetadata() map[string]interface{}    { return s.metadata }
