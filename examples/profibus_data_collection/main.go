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

// 本示例展示：PROFIBUS DP协议的完整功能
//
// 功能：
// 1. 数据读取 - 读取从站输入数据
// 2. 数据监听 - 循环数据交换，实时监控
// 3. 设备管控 - 写入从站输出数据，控制执行器
// 4. 诊断功能 - 读取从站诊断信息

func main() {
	fmt.Println("=== PROFIBUS DP数据采集与控制示例 ===\n")
	fmt.Println("PROFIBUS (Process Field Bus) - 过程现场总线")
	fmt.Println("用于连接PLC、传感器、执行器等设备\n")

	// PROFIBUS主站配置
	config := &serial.PROFIBUSConfig{
		Mode:         serial.PROFIBUS_DP,
		Role:         serial.PROFIBUSMaster,
		MasterAddr:   2,                        // 主站地址
		SerialPort:   "/dev/ttyUSB0",           // RS-485串口
		BaudRate:     19200,                    // 19.2K波特率（常用）
		SlaveAddrs:   []byte{3, 4, 5},          // 从站地址：3个从站
		CycleTime:    10 * time.Millisecond,    // 10ms循环时间
		Watchdog:     1 * time.Second,          // 看门狗1秒
		DataExchange: true,                     // 数据交换模式
		Timeout:      100 * time.Millisecond,
	}

	fmt.Println("📋 PROFIBUS配置:")
	fmt.Printf("  主站地址: %d\n", config.MasterAddr)
	fmt.Printf("  从站地址: %v\n", config.SlaveAddrs)
	fmt.Printf("  波特率: %d bps\n", config.BaudRate)
	fmt.Printf("  循环时间: %v\n", config.CycleTime)
	fmt.Println()

	// 创建PROFIBUS主站
	profibus, err := serial.NewPROFIBUS(config)
	if err != nil {
		log.Fatalf("创建PROFIBUS主站失败: %v", err)
	}

	// 打开连接
	err = profibus.Open("")
	if err != nil {
		log.Printf("打开PROFIBUS连接失败: %v\n", err)
		log.Println("\n提示: 请确保:")
		log.Println("  1. RS-485转USB适配器已连接")
		log.Println("  2. PROFIBUS从站设备已上电")
		log.Println("  3. 终端电阻已正确设置")
		log.Println("  4. 串口路径正确 (/dev/ttyUSB0)")
		log.Println("\n将使用模拟模式演示...")
		useSimulationMode(config)
		return
	}

	defer profibus.Close()

	fmt.Println("✓ PROFIBUS主站已连接")
	fmt.Println("✓ 总线初始化完成\n")

	// 显示从站状态
	fmt.Println("=== 从站状态 ===")
	slaves := profibus.GetAllSlaves()
	for addr, slave := range slaves {
		status := "离线"
		if slave.Active {
			status = "在线"
		}
		fmt.Printf("  从站 %d: %s\n", addr, status)
	}
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 数据读取演示
	fmt.Println("=== 功能1: 数据读取 ===")
	demonstrateDataReading(profibus, config.SlaveAddrs)

	// 2. 设备管控演示
	fmt.Println("\n=== 功能2: 设备管控 ===")
	go demonstrateDeviceControl(ctx, profibus, config.SlaveAddrs)

	// 3. 诊断功能演示
	fmt.Println("\n=== 功能3: 诊断功能 ===")
	go demonstrateDiagnostics(ctx, profibus, config.SlaveAddrs)

	// 4. 数据监听演示（循环数据交换）
	fmt.Println("\n=== 功能4: 数据监听（循环数据交换）===")
	err = profibus.StartCyclicExchange(ctx)
	if err != nil {
		log.Printf("启动循环数据交换失败: %v", err)
	} else {
		fmt.Println("循环数据交换已启动")
		go monitorDataUpdates(profibus)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n--- PROFIBUS主站运行中 ---")
	fmt.Println("循环周期: 10ms")
	fmt.Println("按 Ctrl+C 退出...\n")

	<-sigChan

	fmt.Println("\n\n正在关闭...")
	cancel()
	time.Sleep(500 * time.Millisecond)

	// 显示最终统计
	status := profibus.GetStatus()
	fmt.Println("\n=== 运行统计 ===")
	fmt.Printf("请求次数: %d\n", status.RequestCount)
	fmt.Printf("错误次数: %d\n", status.ErrorCount)
	fmt.Printf("发送字节: %d\n", status.BytesSent)
	fmt.Printf("接收字节: %d\n", status.BytesReceived)

	fmt.Println("\n✓ 示例完成!")
}

// demonstrateDataReading 演示数据读取
func demonstrateDataReading(profibus *serial.PROFIBUS, slaveAddrs []byte) {
	for _, addr := range slaveAddrs {
		fmt.Printf("  → 从站 %d: 读取输入数据...", addr)

		data, err := profibus.ReadInputData(addr)
		if err != nil {
			fmt.Printf(" 失败: %v\n", err)
			continue
		}

		fmt.Printf(" 成功\n")
		fmt.Printf("     数据长度: %d 字节\n", len(data))
		if len(data) > 0 {
			fmt.Printf("     数据: %v\n", data)
			fmt.Printf("     十六进制: ")
			for i, b := range data {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Printf("%02X", b)
			}
			fmt.Println()
		}
	}
}

// demonstrateDeviceControl 演示设备管控
func demonstrateDeviceControl(ctx context.Context, profibus *serial.PROFIBUS, slaveAddrs []byte) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	controlCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			controlCount++

			// 循环控制每个从站
			addr := slaveAddrs[controlCount%len(slaveAddrs)]

			// 生成控制数据
			outputData := []byte{
				byte(controlCount & 0xFF),
				byte((controlCount >> 8) & 0xFF),
				0x01, // 控制位
				0x00,
			}

			fmt.Printf("[管控] 从站 %d: 写入输出数据 %v\n", addr, outputData)

			err := profibus.WriteOutputData(addr, outputData)
			if err != nil {
				fmt.Printf("       失败: %v\n", err)
			} else {
				fmt.Println("       ✓ 成功")
			}
		}
	}
}

// demonstrateDiagnostics 演示诊断功能
func demonstrateDiagnostics(ctx context.Context, profibus *serial.PROFIBUS, slaveAddrs []byte) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Println("\n[诊断] 读取从站诊断信息:")

			for _, addr := range slaveAddrs {
				diag, err := profibus.ReadDiagnostics(addr)
				if err != nil {
					fmt.Printf("  从站 %d: 失败 - %v\n", addr, err)
					continue
				}

				fmt.Printf("  从站 %d:\n", addr)
				fmt.Printf("    状态1: 0x%02X\n", diag.StationStatus1)
				fmt.Printf("    状态2: 0x%02X\n", diag.StationStatus2)
				fmt.Printf("    状态3: 0x%02X\n", diag.StationStatus3)
				fmt.Printf("    主站地址: %d\n", diag.MasterAddr)
				fmt.Printf("    识别号: 0x%04X\n", diag.IdentNumber)

				// 解析状态位
				if diag.StationStatus1&0x01 != 0 {
					fmt.Println("    ⚠️  诊断：配置错误")
				}
				if diag.StationStatus1&0x02 != 0 {
					fmt.Println("    ⚠️  诊断：参数错误")
				}
			}

			fmt.Println()
		}
	}
}

// monitorDataUpdates 监听数据更新
func monitorDataUpdates(profibus *serial.PROFIBUS) {
	dataChan := profibus.GetDataChannel()

	updateCount := 0

	for update := range dataChan {
		updateCount++

		if updateCount%10 == 1 { // 每10次更新显示一次，避免刷屏
			fmt.Printf("[监听] %s 数据变化\n", update.TargetID)
			fmt.Printf("       时间: %s\n", update.Timestamp.Format("15:04:05.000"))

			if newData, ok := update.Value.([]byte); ok {
				fmt.Printf("       新值: %v\n", newData)
			}

			if update.Metadata != nil {
				if slaveAddr, ok := update.Metadata["slave_addr"].(byte); ok {
					fmt.Printf("       从站: %d\n", slaveAddr)
				}
			}
		}
	}
}

// useSimulationMode 使用模拟模式演示
func useSimulationMode(config *serial.PROFIBUSConfig) {
	fmt.Println("\n=== 模拟模式演示 ===\n")

	fmt.Println("1. 总线初始化:")
	for _, addr := range config.SlaveAddrs {
		fmt.Printf("   从站 %d: 参数化 → 配置检查 → 激活 ✓\n", addr)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n2. 数据读取:")
	fmt.Println("   从站 3: 输入数据 = [0x12, 0x34, 0x56, 0x78]")
	fmt.Println("   从站 4: 输入数据 = [0xAA, 0xBB, 0xCC, 0xDD]")
	fmt.Println("   从站 5: 输入数据 = [0x00, 0x00, 0x00, 0x01]")

	fmt.Println("\n3. 循环数据交换:")
	fmt.Println("   周期: 10ms")
	fmt.Println("   主站 → 从站 3: 写入 [0x01, 0x02]")
	fmt.Println("   从站 3 → 主站: 返回 [0x12, 0x34, 0x56, 0x78]")
	fmt.Println("   ...")

	fmt.Println("\n4. 设备管控:")
	fmt.Println("   写入从站 3 输出数据: [0x01, 0x00, 0x01, 0x00] ✓")
	fmt.Println("   → 控制数字输出: DO1=ON, DO2=OFF, DO3=ON, DO4=OFF")

	fmt.Println("\n5. 诊断信息:")
	fmt.Println("   从站 3:")
	fmt.Println("     状态: 正常运行")
	fmt.Println("     识别号: 0x1234")
	fmt.Println("     错误计数: 0")

	fmt.Println("\nPROFIBUS DP特性:")
	fmt.Println("  • 确定性通信（实时性高）")
	fmt.Println("  • 支持多主站")
	fmt.Println("  • 循环数据交换 + 非循环服务")
	fmt.Println("  • 波特率：9.6K - 12M bps")
	fmt.Println("  • 最多127个站点")
	fmt.Println("  • 距离：最长1200m（9.6K）")

	fmt.Println("\n应用场景:")
	fmt.Println("  • 离散制造业（汽车、机械加工）")
	fmt.Println("  • 过程工业（化工、制药-PROFIBUS PA）")
	fmt.Println("  • 楼宇自动化")
	fmt.Println("  • 运动控制")
}
