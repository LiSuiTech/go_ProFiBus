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

// 本示例展示：OPC-UA协议数据采集
//
// 功能：
// 1. 连接到OPC-UA服务器
// 2. 读取多个节点数据
// 3. 支持订阅模式和轮询模式
// 4. 支持安全策略、认证等配置
//
// 使用场景：
// - 工业自动化系统
// - SCADA数据采集
// - PLC数据读取
// - 工厂设备监控

func main() {
	fmt.Println("=== OPC-UA数据采集示例 ===\n")

	// 选择采集模式
	useSubscription := true // true=订阅模式, false=轮询模式

	// 1. 创建OPC-UA数据源配置
	config := &collector.OPCUASourceConfig{
		SourceID: "opcua-plc-system",
		Name:     "工厂PLC控制系统",
		Endpoint: "opc.tcp://localhost:4840", // OPC-UA服务器端点

		// 安全配置
		SecurityPolicy: "None",        // 安全策略: None, Basic128Rsa15, Basic256, Basic256Sha256
		SecurityMode:   "None",        // 安全模式: None, Sign, SignAndEncrypt
		Username:       "",            // 用户名（可选）
		Password:       "",            // 密码（可选）
		CertFile:       "",            // 证书文件（可选）
		KeyFile:        "",            // 密钥文件（可选）

		// 要读取的节点ID列表
		// 这些是OPC-UA标准节点，大多数服务器都支持
		NodeIDs: []string{
			"ns=0;i=2258", // ServerStatus.CurrentTime
			"ns=0;i=2259", // ServerStatus.State
			"ns=0;i=2266", // ServerStatus.BuildInfo.ProductName
			// 实际应用中，这里应该是您的设备节点
			// 例如：
			// "ns=2;s=Temperature.Zone1",    // 温度传感器1
			// "ns=2;s=Pressure.Tank1",       // 压力传感器1
			// "ns=2;s=Motor.Speed",          // 电机转速
			// "ns=2;s=Valve.Position",       // 阀门位置
		},

		// 采样配置
		SamplingInterval: 1 * time.Second, // 采样间隔
		BufferSize:       1000,            // 缓冲区大小

		// 订阅模式配置
		UseSubscription: useSubscription,         // 是否使用订阅模式
		PublishInterval: 1 * time.Second,         // 发布间隔
		QueueSize:       100,                     // 队列大小
	}

	fmt.Println("📋 OPC-UA配置:")
	fmt.Printf("  服务器端点: %s\n", config.Endpoint)
	fmt.Printf("  采集模式: %s\n", func() string {
		if config.UseSubscription {
			return "订阅模式"
		}
		return "轮询模式"
	}())
	fmt.Printf("  节点数量: %d\n", len(config.NodeIDs))
	fmt.Printf("  采样间隔: %v\n", config.SamplingInterval)
	fmt.Printf("  安全策略: %s/%s\n\n", config.SecurityPolicy, config.SecurityMode)

	// 2. 创建OPC-UA数据源
	opcuaSource, err := collector.NewOPCUASource(config)
	if err != nil {
		log.Fatalf("创建OPC-UA数据源失败: %v", err)
	}

	fmt.Println("✓ OPC-UA数据源已创建")

	// 3. 启动数据源
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = opcuaSource.Start(ctx)
	if err != nil {
		log.Fatalf("启动OPC-UA数据源失败: %v\n\n", err)
		fmt.Println("提示: 请确保OPC-UA服务器正在运行")
		fmt.Println("  可以使用以下开源OPC-UA服务器进行测试:")
		fmt.Println("  - Prosys OPC UA Simulation Server: https://www.prosysopc.com/products/opc-ua-simulation-server/")
		fmt.Println("  - open62541: https://github.com/open62541/open62541")
		os.Exit(1)
	}

	fmt.Println("✓ OPC-UA数据源已启动，正在连接...\n")

	// 等待连接建立
	time.Sleep(2 * time.Second)

	status := opcuaSource.GetStatus()
	if !status.Connected {
		log.Fatalf("无法连接到OPC-UA服务器")
	}

	fmt.Println("✓ 已连接到OPC-UA服务器\n")

	// 4. 启动数据处理协程
	go processData(opcuaSource)

	// 5. 启动状态监控协程
	go monitorStatus(opcuaSource)

	// 6. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("--- 开始读取OPC-UA节点数据 ---")
	fmt.Printf("采集模式: %s\n", func() string {
		if config.UseSubscription {
			return "订阅模式（数据变更时主动推送）"
		}
		return "轮询模式（定时读取）"
	}())
	fmt.Println("\n按 Ctrl+C 退出...\n")

	<-sigChan

	// 7. 清理资源
	fmt.Println("\n\n--- 正在关闭 ---")
	cancel()
	opcuaSource.Stop()

	// 显示最终统计
	status = opcuaSource.GetStatus()
	fmt.Printf("\n最终统计:\n")
	fmt.Printf("  总读取次数: %d\n", status.SamplesCollected)
	fmt.Printf("  最后读取时间: %v\n", status.LastSampleTime.Format("2006-01-02 15:04:05"))

	fmt.Println("\n✓ 示例完成!")
}

// processData 处理数据
func processData(source interfaces.DataSource) {
	dataChan := source.GetData()
	readCount := 0

	for sample := range dataChan {
		readCount++

		fmt.Printf("\n[读取 #%d] 时间: %s\n",
			readCount,
			sample.GetTimestamp().Format("15:04:05.000"))

		// 获取节点信息
		metadata := sample.GetMetadata()
		if nodeID, ok := metadata["opcua_node_id"].(string); ok {
			fmt.Printf("  节点ID: %s\n", nodeID)
		}

		// 显示数据
		data := sample.GetData()
		if value, ok := data["value"]; ok {
			fmt.Printf("  值: %v\n", value)
		}

		// 显示状态
		if status, ok := metadata["opcua_status"].(string); ok {
			fmt.Printf("  状态: %s\n", status)
		}

		// 显示时间戳
		if sourceTime, ok := metadata["opcua_source_timestamp"].(time.Time); ok {
			fmt.Printf("  源时间戳: %s\n", sourceTime.Format("15:04:05.000"))
		}

		// 显示质量
		quality := sample.GetQuality()
		qualityStr := "良好"
		if quality < 1.0 {
			qualityStr = "错误"
		}
		fmt.Printf("  质量: %.2f (%s)\n", quality, qualityStr)

		// 解析具体值类型
		if value, ok := data["value"]; ok {
			switch v := value.(type) {
			case float64:
				fmt.Printf("  📊 数值: %.2f\n", v)
			case int64:
				fmt.Printf("  📊 整数: %d\n", v)
			case string:
				fmt.Printf("  📊 字符串: %s\n", v)
			case time.Time:
				fmt.Printf("  📊 时间: %s\n", v.Format("2006-01-02 15:04:05"))
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
		fmt.Printf("  累计读取: %d\n", status.SamplesCollected)

		if !status.LastSampleTime.IsZero() {
			fmt.Printf("  最后读取: %v\n", time.Since(status.LastSampleTime).Round(time.Second))
		}

		if status.Error != nil {
			fmt.Printf("  错误: %v\n", status.Error)
		}
		fmt.Println()
	}
}
