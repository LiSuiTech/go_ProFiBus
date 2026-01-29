package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_ProFiBus/internal/infrastructure/collector"
	"go_ProFiBus/pkg/interfaces"
)

// 本示例展示：MQTT协议数据采集
//
// 功能：
// 1. 连接到MQTT代理
// 2. 订阅多个主题
// 3. 接收并处理MQTT消息
// 4. 支持QoS、TLS、认证等配置
//
// 使用场景：
// - IoT设备数据采集
// - 传感器网络
// - 实时消息订阅
// - 事件驱动数据采集

func main() {
	fmt.Println("=== MQTT数据采集示例 ===\n")

	// 1. 创建MQTT数据源配置
	config := &collector.MQTTSourceConfig{
		SourceID:     "mqtt-sensor-network",
		Name:         "工厂传感器MQTT网络",
		Broker:       "tcp://localhost:1883", // MQTT代理地址
		ClientID:     "",                      // 留空则自动生成
		Username:     "",                      // 如果需要认证，填写用户名
		Password:     "",                      // 如果需要认证，填写密码
		Topics: []string{
			"factory/sensors/temperature/#", // 温度传感器（通配符）
			"factory/sensors/pressure/#",    // 压力传感器（通配符）
			"factory/sensors/vibration/#",   // 振动传感器（通配符）
			"factory/alerts/#",              // 告警主题（通配符）
		},
		QoS:          1,     // QoS等级1：至少一次
		CleanSession: true,  // 清除会话
		KeepAlive:    60,    // 保活60秒
		UseTLS:       false, // 不使用TLS（本地测试）
		BufferSize:   1000,  // 缓冲区1000条消息
	}

	fmt.Println("📋 MQTT配置:")
	fmt.Printf("  代理地址: %s\n", config.Broker)
	fmt.Printf("  订阅主题: %v\n", config.Topics)
	fmt.Printf("  QoS等级: %d\n", config.QoS)
	fmt.Printf("  缓冲区大小: %d\n\n", config.BufferSize)

	// 2. 创建MQTT数据源
	mqttSource, err := collector.NewMQTTSource(config)
	if err != nil {
		log.Fatalf("创建MQTT数据源失败: %v", err)
	}

	fmt.Println("✓ MQTT数据源已创建")

	// 3. 启动数据源
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = mqttSource.Start(ctx)
	if err != nil {
		log.Fatalf("启动MQTT数据源失败: %v", err)
	}

	fmt.Println("✓ MQTT数据源已启动，等待消息...\n")

	// 4. 启动数据处理协程
	go processData(mqttSource)

	// 5. 启动状态监控协程
	go monitorStatus(mqttSource)

	// 6. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("--- 开始接收MQTT消息 ---")
	fmt.Println("提示: 使用以下命令发布测试消息:")
	fmt.Println("  mosquitto_pub -t factory/sensors/temperature/zone1 -m '{\"value\": 85.5, \"unit\": \"C\"}'")
	fmt.Println("  mosquitto_pub -t factory/sensors/pressure/tank1 -m '{\"value\": 2.5, \"unit\": \"bar\"}'")
	fmt.Println("  mosquitto_pub -t factory/alerts/high_temp -m '{\"zone\": \"zone1\", \"value\": 95.0}'")
	fmt.Println("\n按 Ctrl+C 退出...\n")

	<-sigChan

	// 7. 清理资源
	fmt.Println("\n\n--- 正在关闭 ---")
	cancel()
	mqttSource.Stop()

	// 显示最终统计
	status := mqttSource.GetStatus()
	fmt.Printf("\n最终统计:\n")
	fmt.Printf("  总接收消息: %d\n", status.SamplesCollected)
	fmt.Printf("  最后消息时间: %v\n", status.LastSampleTime.Format("2006-01-02 15:04:05"))

	fmt.Println("\n✓ 示例完成!")
}

// processData 处理数据
func processData(source interfaces.DataSource) {
	dataChan := source.GetData()
	messageCount := 0

	for sample := range dataChan {
		messageCount++

		fmt.Printf("\n[消息 #%d] 时间: %s\n",
			messageCount,
			sample.GetTimestamp().Format("15:04:05.000"))

		// 获取主题信息
		metadata := sample.GetMetadata()
		if topic, ok := metadata["mqtt_topic"].(string); ok {
			fmt.Printf("  主题: %s\n", topic)
		}

		// 显示数据
		data := sample.GetData()
		fmt.Printf("  数据: %v\n", data)

		// 显示质量和元数据
		fmt.Printf("  质量: %.2f\n", sample.GetQuality())

		if qos, ok := metadata["mqtt_qos"].(byte); ok {
			fmt.Printf("  QoS: %d\n", qos)
		}
		if retained, ok := metadata["mqtt_retained"].(bool); ok && retained {
			fmt.Printf("  保留消息: 是\n")
		}

		// 特殊处理：解析具体传感器数据
		if value, ok := data["value"].(float64); ok {
			if unit, ok := data["unit"].(string); ok {
				fmt.Printf("  📊 传感器读数: %.2f %s\n", value, unit)
			}
		}

		// 特殊处理：告警消息
		if topic, ok := metadata["mqtt_topic"].(string); ok {
			if len(topic) > 14 && topic[:14] == "factory/alerts" {
				fmt.Printf("  ⚠️  告警消息!\n")
			}
		}

		fmt.Println("  " + string('─'))
	}
}

// monitorStatus 监控数据源状态
func monitorStatus(source interfaces.DataSource) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		status := source.GetStatus()

		fmt.Printf("\n[状态报告]\n")
		fmt.Printf("  运行中: %v\n", status.Running)
		fmt.Printf("  已连接: %v\n", status.Connected)
		fmt.Printf("  累计消息: %d\n", status.SamplesCollected)

		if !status.LastSampleTime.IsZero() {
			fmt.Printf("  最后消息: %v\n", time.Since(status.LastSampleTime).Round(time.Second))
		}

		if status.Error != nil {
			fmt.Printf("  错误: %v\n", status.Error)
		}
		fmt.Println()
	}
}
