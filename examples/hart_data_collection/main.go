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

// 本示例展示：HART协议的完整功能
//
// HART (Highway Addressable Remote Transducer)
// - 过程自动化行业标准协议
// - 在4-20mA模拟信号上叠加数字通信
// - 广泛应用于化工、石油、天然气行业
//
// 功能：
// 1. 设备识别 - 读取设备唯一ID和信息
// 2. 数据读取 - 读取主变量 (温度、压力、流量等)
// 3. 数据监听 - 循环轮询设备数据
// 4. 设备配置 - 写入标签、消息等

func main() {
	fmt.Println("=== HART协议数据采集示例 ===\n")
	fmt.Println("HART (Highway Addressable Remote Transducer)")
	fmt.Println("过程自动化协议 - 4-20mA + 数字通信\n")

	// HART配置
	config := &serial.HARTConfig{
		Mode:           serial.HARTModePointToPoint, // 点对点模式
		SerialPort:     "/dev/ttyUSB0",              // RS-485/HART Modem
		BaudRate:       1200,                        // HART标准波特率
		DataBits:       8,
		StopBits:       1,
		Parity:         serial.ParityOdd, // 奇校验

		MasterAddress:  0,           // 主站地址
		SlaveAddresses: []byte{0, 1, 2}, // 从站地址 (点对点=0, 多点=1-15)
		PreambleCount:  5,           // 前导字节数
		RetryCount:     3,
		Timeout:        500 * time.Millisecond,

		// 轮询配置
		PollingInterval: 2 * time.Second,
		PollingEnabled:  true,
	}

	fmt.Println("📋 HART配置:")
	fmt.Printf("  模式: %s\n", config.Mode)
	fmt.Printf("  串口: %s\n", config.SerialPort)
	fmt.Printf("  波特率: %d bps (HART标准)\n", config.BaudRate)
	fmt.Printf("  从站地址: %v\n", config.SlaveAddresses)
	fmt.Printf("  轮询间隔: %v\n", config.PollingInterval)
	fmt.Println()

	// 创建HART实例
	hart, err := serial.NewHART(config)
	if err != nil {
		log.Fatalf("创建HART实例失败: %v", err)
	}

	// 打开连接
	err = hart.Open("")
	if err != nil {
		log.Printf("打开HART连接失败: %v\n", err)
		log.Println("\n提示: 请确保:")
		log.Println("  1. HART Modem或RS-485适配器已连接")
		log.Println("  2. HART设备已上电且接线正确")
		log.Println("  3. 4-20mA回路正常")
		log.Println("  4. 串口路径正确")
		log.Println("\n将使用模拟模式演示...")
		useSimulationMode(config)
		return
	}

	defer hart.Close()

	fmt.Println("✓ HART连接成功")
	fmt.Println()

	// 显示已识别的设备
	fmt.Println("=== 已识别设备 ===")
	devices := hart.GetAllDevices()
	for addr, device := range devices {
		status := "离线"
		if device.Active {
			status = "在线"
		}
		fmt.Printf("  地址 %d: %s\n", addr, status)
		fmt.Printf("    制造商ID: 0x%04X\n", device.Manufacturer)
		fmt.Printf("    设备类型: 0x%02X\n", device.DeviceType)
		fmt.Printf("    设备ID: %02X%02X%02X\n", device.DeviceID[0], device.DeviceID[1], device.DeviceID[2])
		fmt.Printf("    固件版本: %d\n", device.TransmitterRev)
	}
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 读取设备标签
	fmt.Println("=== 功能1: 读取设备标签 ===")
	demonstrateReadTag(hart, config.SlaveAddresses)

	// 2. 读取主变量
	fmt.Println("\n=== 功能2: 读取主变量 ===")
	demonstrateReadVariable(hart, config.SlaveAddresses)

	// 3. 写入消息
	fmt.Println("\n=== 功能3: 写入设备消息 ===")
	demonstrateWriteMessage(hart, config.SlaveAddresses)

	// 4. 启动循环轮询
	fmt.Println("\n=== 功能4: 循环轮询 ===")
	err = hart.StartCyclicPolling(ctx)
	if err != nil {
		log.Printf("启动轮询失败: %v", err)
	} else {
		fmt.Println("循环轮询已启动")
		go monitorDataUpdates(hart)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n--- HART协议运行中 ---")
	fmt.Println("轮询周期: 2秒")
	fmt.Println("按 Ctrl+C 退出...\n")

	<-sigChan

	fmt.Println("\n\n正在关闭...")
	cancel()
	time.Sleep(500 * time.Millisecond)

	// 显示最终统计
	status := hart.GetStatus()
	fmt.Println("\n=== 运行统计 ===")
	fmt.Printf("请求次数: %d\n", status.RequestCount)
	fmt.Printf("错误次数: %d\n", status.ErrorCount)
	fmt.Printf("发送字节: %d\n", status.BytesSent)
	fmt.Printf("接收字节: %d\n", status.BytesReceived)

	fmt.Println("\n✓ 示例完成!")
}

// demonstrateReadTag 演示读取设备标签
func demonstrateReadTag(hart *serial.HART, addresses []byte) {
	for _, addr := range addresses {
		fmt.Printf("  设备 %d: 读取标签...", addr)

		tag, descriptor, err := hart.ReadTag(addr)
		if err != nil {
			fmt.Printf(" 失败: %v\n", err)
			continue
		}

		fmt.Printf(" 成功\n")
		fmt.Printf("    标签: %s\n", tag)
		fmt.Printf("    描述符: %s\n", descriptor)
	}
}

// demonstrateReadVariable 演示读取主变量
func demonstrateReadVariable(hart *serial.HART, addresses []byte) {
	unitNames := map[byte]string{
		0:  "未定义",
		1:  "°C",
		2:  "°F",
		3:  "°R",
		4:  "K",
		5:  "mV",
		6:  "V",
		7:  "mA",
		8:  "A",
		9:  "Pa",
		10: "kPa",
		11: "bar",
		12: "psi",
		13: "L",
		14: "m³",
		15: "L/s",
		16: "m³/h",
	}

	for _, addr := range addresses {
		fmt.Printf("  设备 %d: 读取主变量...", addr)

		value, unit, err := hart.ReadPrimaryVariable(addr)
		if err != nil {
			fmt.Printf(" 失败: %v\n", err)
			continue
		}

		unitName := unitNames[unit]
		if unitName == "" {
			unitName = fmt.Sprintf("(代码 %d)", unit)
		}

		fmt.Printf(" 成功\n")
		fmt.Printf("    值: %.2f %s\n", value, unitName)
	}
}

// demonstrateWriteMessage 演示写入消息
func demonstrateWriteMessage(hart *serial.HART, addresses []byte) {
	message := "HART Test Message"

	for _, addr := range addresses {
		fmt.Printf("  设备 %d: 写入消息 \"%s\"...", addr, message)

		err := hart.WriteMessage(addr, message)
		if err != nil {
			fmt.Printf(" 失败: %v\n", err)
			continue
		}

		fmt.Printf(" 成功\n")
	}
}

// monitorDataUpdates 监听数据更新
func monitorDataUpdates(hart *serial.HART) {
	dataChan := hart.GetDataChannel()

	updateCount := 0

	for update := range dataChan {
		updateCount++

		if updateCount%5 == 1 { // 每5次显示一次，避免刷屏
			fmt.Printf("[轮询] %s 数据更新\n", update.TargetID)
			fmt.Printf("       时间: %s\n", update.Timestamp.Format("15:04:05.000"))
			fmt.Printf("       值: %.2f\n", update.Value)

			if update.Metadata != nil {
				if unit, ok := update.Metadata["unit"].(byte); ok {
					unitNames := map[byte]string{
						1: "°C", 2: "°F", 9: "Pa", 10: "kPa", 11: "bar", 12: "psi",
					}
					if name, exists := unitNames[unit]; exists {
						fmt.Printf("       单位: %s\n", name)
					}
				}
				if tag, ok := update.Metadata["tag"].(string); ok && tag != "" {
					fmt.Printf("       标签: %s\n", tag)
				}
			}
			fmt.Println()
		}
	}
}

// useSimulationMode 使用模拟模式演示
func useSimulationMode(config *serial.HARTConfig) {
	fmt.Println("\n=== 模拟模式演示 ===\n")

	fmt.Println("1. 设备识别:")
	fmt.Println("   发送命令0 (读取唯一ID)")
	fmt.Println("   设备 0:")
	fmt.Println("     制造商: Rosemount (0x001E)")
	fmt.Println("     类型: 温度变送器 (0x13)")
	fmt.Println("     设备ID: 0A1B2C")
	fmt.Println("     固件版本: 5")

	fmt.Println("\n2. 读取设备标签:")
	fmt.Println("   发送命令13 (读取标签)")
	fmt.Println("   设备 0:")
	fmt.Println("     标签: TT-101")
	fmt.Println("     描述符: Reactor Temp")

	fmt.Println("\n3. 读取主变量:")
	fmt.Println("   发送命令1 (读取主变量)")
	fmt.Println("   设备 0:")
	fmt.Println("     值: 85.50 °C")
	fmt.Println("   设备 1:")
	fmt.Println("     值: 2.45 bar")
	fmt.Println("   设备 2:")
	fmt.Println("     值: 125.30 L/min")

	fmt.Println("\n4. 写入消息:")
	fmt.Println("   发送命令17 (写入消息)")
	fmt.Println("   设备 0: \"HART Test Message\" ✓")

	fmt.Println("\n5. 循环轮询:")
	fmt.Println("   每2秒读取所有设备主变量")
	fmt.Println("   [轮询] HART_Device_0: 85.52 °C")
	fmt.Println("   [轮询] HART_Device_1: 2.46 bar")
	fmt.Println("   [轮询] HART_Device_2: 125.28 L/min")

	fmt.Println("\nHART协议特性:")
	fmt.Println("  • 波特率: 1200 bps")
	fmt.Println("  • 调制: FSK (频移键控)")
	fmt.Println("  • 频率: 1200Hz (逻辑1), 2200Hz (逻辑0)")
	fmt.Println("  • 工作模式: 点对点 / 多点")
	fmt.Println("  • 通信距离: 最长3000米")
	fmt.Println("  • 多点模式: 最多15个设备")

	fmt.Println("\nHART命令类型:")
	fmt.Println("  • 通用命令 (Universal): 所有设备支持 (0-30)")
	fmt.Println("  • 常用命令 (Common Practice): 常见设备支持 (31-126)")
	fmt.Println("  • 设备特定命令 (Device-Specific): 特定设备支持 (127-253)")

	fmt.Println("\n应用场景:")
	fmt.Println("  • 化工过程控制")
	fmt.Println("  • 石油天然气生产")
	fmt.Println("  • 制药行业")
	fmt.Println("  • 水处理")
	fmt.Println("  • 食品饮料生产")
	fmt.Println("  • 电力行业")

	fmt.Println("\nHART vs 其他协议:")
	fmt.Println("  • HART: 混合协议 (模拟+数字), 成熟稳定")
	fmt.Println("  • PROFIBUS PA: 全数字, 本质安全")
	fmt.Println("  • Foundation Fieldbus: 全数字, 功能块")
	fmt.Println("  • 4-20mA: 纯模拟, 简单可靠")
}
