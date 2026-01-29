package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_ProFiBus/serial"
)

// 本示例展示：DeviceNet协议的完整功能
//
// DeviceNet - 基于CAN的现场总线协议
// - 离散制造业广泛应用
// - 电源和数据在同一电缆
// - 支持多种消息类型
//
// 功能：
// 1. 节点识别 - 读取节点设备信息
// 2. I/O数据读取 - 读取节点输入数据
// 3. I/O数据写入 - 写入节点输出数据
// 4. 循环轮询 - 持续监控节点数据

func main() {
	fmt.Println("=== DeviceNet协议数据采集示例 ===\n")
	fmt.Println("DeviceNet - CAN-based Industrial Network")
	fmt.Println("广泛应用于汽车和离散制造业\n")

	// DeviceNet配置
	config := &serial.DeviceNetConfig{
		CANInterface: "can0",   // CAN接口 (Linux: can0, vcan0)
		Bitrate:      250000,   // 250kbps (也可选择125k或500k)

		MasterMAC: 0,                // 主站MAC地址
		SlaveMACs: []byte{1, 2, 3, 4}, // 从站MAC地址
		ScanTime:  10 * time.Millisecond,
		RetryCount: 3,
		Timeout:    100 * time.Millisecond,

		// 轮询配置
		PollingInterval: 100 * time.Millisecond, // 100ms轮询周期
		PollingEnabled:  true,
	}

	fmt.Println("📋 DeviceNet配置:")
	fmt.Printf("  CAN接口: %s\n", config.CANInterface)
	fmt.Printf("  波特率: %d bps\n", config.Bitrate)
	fmt.Printf("  主站MAC: %d\n", config.MasterMAC)
	fmt.Printf("  从站MAC: %v\n", config.SlaveMACs)
	fmt.Printf("  轮询间隔: %v\n", config.PollingInterval)
	fmt.Println()

	// 创建DeviceNet实例
	deviceNet, err := serial.NewDeviceNet(config)
	if err != nil {
		log.Fatalf("创建DeviceNet实例失败: %v", err)
	}

	// 打开连接
	err = deviceNet.Open("")
	if err != nil {
		log.Printf("打开DeviceNet连接失败: %v\n", err)
		log.Println("\n提示: 请确保:")
		log.Println("  1. CAN接口已配置 (ip link set can0 type can bitrate 250000)")
		log.Println("  2. CAN接口已启动 (ip link set can0 up)")
		log.Println("  3. DeviceNet节点已上电")
		log.Println("  4. 网络连接正常")
		log.Println("\n将使用模拟模式演示...")
		useSimulationMode(config)
		return
	}

	defer deviceNet.Close()

	fmt.Println("✓ DeviceNet连接成功")
	fmt.Println()

	// 显示已识别的节点
	fmt.Println("=== 已识别节点 ===")
	nodes := deviceNet.GetAllNodes()
	for mac, node := range nodes {
		status := "离线"
		if node.Active {
			status = "在线"
		}
		fmt.Printf("  MAC %d: %s\n", mac, status)
		fmt.Printf("    供应商ID: 0x%04X\n", node.VendorID)
		fmt.Printf("    设备类型: 0x%04X\n", node.DeviceType)
		fmt.Printf("    产品代码: 0x%04X\n", node.ProductCode)
		fmt.Printf("    序列号: %d\n", node.SerialNumber)
		if node.ProductName != "" {
			fmt.Printf("    产品名称: %s\n", node.ProductName)
		}
	}
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 读取输入数据
	fmt.Println("=== 功能1: 读取I/O数据 ===")
	demonstrateReadIO(deviceNet, config.SlaveMACs)

	// 2. 写入输出数据
	fmt.Println("\n=== 功能2: 写入I/O数据 ===")
	go demonstrateWriteIO(ctx, deviceNet, config.SlaveMACs)

	// 3. 启动循环轮询
	fmt.Println("\n=== 功能3: 循环I/O轮询 ===")
	err = deviceNet.StartIOPolling(ctx)
	if err != nil {
		log.Printf("启动轮询失败: %v", err)
	} else {
		fmt.Println("I/O轮询已启动")
		go monitorDataUpdates(deviceNet)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n--- DeviceNet网络运行中 ---")
	fmt.Println("轮询周期: 100ms")
	fmt.Println("按 Ctrl+C 退出...\n")

	<-sigChan

	fmt.Println("\n\n正在关闭...")
	cancel()
	time.Sleep(500 * time.Millisecond)

	// 显示最终统计
	status := deviceNet.GetStatus()
	fmt.Println("\n=== 运行统计 ===")
	fmt.Printf("请求次数: %d\n", status.RequestCount)
	fmt.Printf("错误次数: %d\n", status.ErrorCount)
	fmt.Printf("发送字节: %d\n", status.BytesSent)
	fmt.Printf("接收字节: %d\n", status.BytesReceived)

	fmt.Println("\n✓ 示例完成!")
}

// demonstrateReadIO 演示读取I/O数据
func demonstrateReadIO(deviceNet *serial.DeviceNet, macs []byte) {
	for _, mac := range macs {
		fmt.Printf("  节点 %d: 读取输入数据...", mac)

		data, err := deviceNet.ReadInputData(mac)
		if err != nil {
			fmt.Printf(" 失败: %v\n", err)
			continue
		}

		fmt.Printf(" 成功\n")
		fmt.Printf("    数据长度: %d 字节\n", len(data))
		if len(data) > 0 {
			fmt.Printf("    数据: ")
			for i, b := range data {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Printf("%02X", b)
			}
			fmt.Println()

			// 解析常见I/O模块数据
			if len(data) >= 2 {
				digitalInputs := data[0]
				fmt.Printf("    数字输入: ")
				for bit := 0; bit < 8; bit++ {
					if digitalInputs&(1<<bit) != 0 {
						fmt.Print("1")
					} else {
						fmt.Print("0")
					}
				}
				fmt.Println()
			}
		}
	}
}

// demonstrateWriteIO 演示写入I/O数据
func demonstrateWriteIO(ctx context.Context, deviceNet *serial.DeviceNet, macs []byte) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	counter := byte(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			counter++

			// 循环控制每个节点
			mac := macs[int(counter)%len(macs)]

			// 生成输出数据 (例如：数字输出模块)
			outputData := []byte{
				counter & 0xFF,           // 字节0: 数字输出
				byte((counter >> 1) & 0xFF), // 字节1: 其他输出
			}

			fmt.Printf("[控制] 节点 %d: 写入输出数据 [%02X %02X]", mac, outputData[0], outputData[1])

			err := deviceNet.WriteOutputData(mac, outputData)
			if err != nil {
				fmt.Printf(" - 失败: %v\n", err)
			} else {
				fmt.Println(" - ✓")
			}
		}
	}
}

// monitorDataUpdates 监听数据更新
func monitorDataUpdates(deviceNet *serial.DeviceNet) {
	dataChan := deviceNet.GetDataChannel()

	updateCount := 0

	for update := range dataChan {
		updateCount++

		if updateCount%10 == 1 { // 每10次更新显示一次，避免刷屏
			fmt.Printf("[轮询] %s 数据变化\n", update.TargetID)
			fmt.Printf("       时间: %s\n", update.Timestamp.Format("15:04:05.000"))

			if newData, ok := update.Value.([]byte); ok {
				fmt.Printf("       新值: ")
				for i, b := range newData {
					if i > 0 {
						fmt.Print(" ")
					}
					fmt.Printf("%02X", b)
				}
				fmt.Println()
			}

			if update.Metadata != nil {
				if mac, ok := update.Metadata["mac"].(byte); ok {
					fmt.Printf("       MAC: %d\n", mac)
				}
				if vendorID, ok := update.Metadata["vendor_id"].(uint16); ok {
					fmt.Printf("       供应商: 0x%04X\n", vendorID)
				}
			}
		}
	}
}

// useSimulationMode 使用模拟模式演示
func useSimulationMode(config *serial.DeviceNetConfig) {
	fmt.Println("\n=== 模拟模式演示 ===\n")

	fmt.Println("1. 网络初始化:")
	fmt.Println("   配置CAN接口: can0, 250kbps ✓")
	fmt.Println("   扫描网络节点...")
	for _, mac := range config.SlaveMACs {
		fmt.Printf("   节点 %d: 识别成功 ✓\n", mac)
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("\n2. 节点识别:")
	vendors := map[byte]string{
		1: "Allen-Bradley (0x0001)",
		2: "Omron (0x0019)",
		3: "Festo (0x000E)",
		4: "SMC (0x002D)",
	}
	deviceTypes := map[byte]string{
		1: "数字输入模块 (0x0007)",
		2: "数字输出模块 (0x0009)",
		3: "模拟输入模块 (0x000A)",
		4: "气动阀岛 (0x001B)",
	}

	for i, mac := range config.SlaveMACs {
		fmt.Printf("   节点 %d:\n", mac)
		fmt.Printf("     供应商: %s\n", vendors[mac])
		fmt.Printf("     设备类型: %s\n", deviceTypes[mac])
		fmt.Printf("     序列号: %d\n", 1000+i)
	}

	fmt.Println("\n3. I/O数据读取:")
	fmt.Println("   节点 1 (数字输入): [0xFF] - 所有输入为高电平")
	fmt.Println("   节点 2 (数字输出): [0x00] - 所有输出为低电平")
	fmt.Println("   节点 3 (模拟输入): [0x7F 0xFF] - 模拟值: 32767")
	fmt.Println("   节点 4 (气动阀): [0x0F] - 阀1-4打开")

	fmt.Println("\n4. I/O数据写入:")
	fmt.Println("   节点 2: 写入 [0xAA] - 交替输出高低 ✓")
	fmt.Println("   节点 4: 写入 [0x01] - 仅阀1打开 ✓")

	fmt.Println("\n5. 循环轮询 (100ms):")
	fmt.Println("   [轮询] DeviceNet_Node_1: [0xFF]")
	fmt.Println("   [轮询] DeviceNet_Node_2: [0xAA]")
	fmt.Println("   [轮询] DeviceNet_Node_3: [0x7F 0xFE] ← 数据变化")
	fmt.Println("   [轮询] DeviceNet_Node_4: [0x01]")

	fmt.Println("\nDeviceNet特性:")
	fmt.Println("  • 基于CAN 2.0A (11位标识符)")
	fmt.Println("  • 波特率: 125k / 250k / 500k bps")
	fmt.Println("  • 节点数: 最多64个 (MAC 0-63)")
	fmt.Println("  • 通信距离: 500m (125k), 250m (250k), 100m (500k)")
	fmt.Println("  • 电源和数据同缆")
	fmt.Println("  • 即插即用 (Plug and Play)")

	fmt.Println("\nDeviceNet消息类型:")
	fmt.Println("  • Explicit Messaging: 显式消息 (配置、诊断)")
	fmt.Println("  • I/O Polling: 轮询I/O (主站请求)")
	fmt.Println("  • I/O Bit-Strobe: 位选通I/O (多个从站)")
	fmt.Println("  • I/O Change-of-State: 状态变化I/O (事件触发)")
	fmt.Println("  • I/O Cyclic: 循环I/O (周期性)")

	fmt.Println("\n应用场景:")
	fmt.Println("  • 汽车制造 (焊接、喷涂机器人)")
	fmt.Println("  • 包装机械")
	fmt.Println("  • 物料搬运")
	fmt.Println("  • 机床控制")
	fmt.Println("  • 装配线自动化")

	fmt.Println("\nDeviceNet vs 其他CAN协议:")
	fmt.Println("  • DeviceNet: 工业自动化，离散制造")
	fmt.Println("  • CANopen: 机械控制，运动控制")
	fmt.Println("  • J1939: 商用车辆")
	fmt.Println("  • NMEA 2000: 船舶电子")
}
