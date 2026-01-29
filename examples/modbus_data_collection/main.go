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

// 本示例展示：Modbus协议的完整功能
//
// 功能：
// 1. 数据读取 - 读取保持寄存器、输入寄存器、线圈、离散输入
// 2. 数据监听 - 持续轮询数据变化
// 3. 设备管控 - 写入线圈、寄存器，控制设备

func main() {
	fmt.Println("=== Modbus数据采集与控制示例 ===\n")

	// 选择Modbus模式
	fmt.Println("选择Modbus模式:")
	fmt.Println("1. Modbus RTU (串口)")
	fmt.Println("2. Modbus TCP (网络)")
	fmt.Print("\n请选择 (1/2, 默认2): ")

	var choice int
	fmt.Scanf("%d", &choice)
	if choice == 0 {
		choice = 2 // 默认TCP
	}

	var modbus *serial.Modbus
	var err error

	switch choice {
	case 1:
		// Modbus RTU模式
		fmt.Println("\n=== Modbus RTU模式 ===")
		config := &serial.ModbusConfig{
			Mode:       serial.ModbusRTU,
			Role:       serial.ModbusMaster,
			SerialPort: "/dev/ttyUSB0",
			BaudRate:   9600,
			DataBits:   8,
			StopBits:   1,
			Parity:     "N",
			Timeout:    1 * time.Second,
		}

		modbus, err = serial.NewModbusMaster(config)
		if err != nil {
			log.Fatalf("创建Modbus主站失败: %v", err)
		}

		err = modbus.Open("")
		if err != nil {
			log.Fatalf("打开串口失败: %v\n提示: 请确保串口设备存在", err)
		}

	case 2:
		// Modbus TCP模式
		fmt.Println("\n=== Modbus TCP模式 ===")
		config := &serial.ModbusConfig{
			Mode:       serial.ModbusTCP,
			Role:       serial.ModbusMaster,
			TCPAddress: "127.0.0.1:502",
			Timeout:    2 * time.Second,
		}

		modbus, err = serial.NewModbusMaster(config)
		if err != nil {
			log.Fatalf("创建Modbus主站失败: %v", err)
		}

		err = modbus.Open("")
		if err != nil {
			log.Printf("连接Modbus TCP服务器失败: %v\n", err)
			log.Println("提示: 请确保Modbus TCP服务器运行在 127.0.0.1:502")
			log.Println("      可以使用modbus-tcp-server或其他Modbus模拟器")
			log.Println("\n将使用模拟模式演示...")
			useSimulationMode()
			return
		}
	}

	defer modbus.Close()

	fmt.Println("✓ Modbus连接成功\n")

	// 启动三个功能演示
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 数据读取演示
	fmt.Println("=== 功能1: 数据读取 ===")
	demonstrateDataReading(modbus)

	// 2. 数据监听演示
	fmt.Println("\n=== 功能2: 数据监听 ===")
	go demonstrateDataMonitoring(ctx, modbus)

	// 3. 设备管控演示
	fmt.Println("\n=== 功能3: 设备管控 ===")
	go demonstrateDeviceControl(ctx, modbus)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n--- Modbus主站运行中 ---")
	fmt.Println("按 Ctrl+C 退出...\n")

	<-sigChan

	fmt.Println("\n\n正在关闭...")
	cancel()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("✓ 示例完成!")
}

// demonstrateDataReading 演示数据读取
func demonstrateDataReading(modbus *serial.Modbus) {
	slaveID := byte(1)

	fmt.Println("读取从站地址: 1")

	// 读取保持寄存器
	fmt.Print("  → 读取保持寄存器 (地址40001-40010)...")
	registers, err := modbus.ReadHoldingRegisters(slaveID, 0, 10)
	if err != nil {
		fmt.Printf(" 失败: %v\n", err)
	} else {
		fmt.Printf(" 成功\n")
		fmt.Printf("     数据: %v\n", registers)
		fmt.Printf("     十进制值: ")
		for i, reg := range registers {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%d", reg)
		}
		fmt.Println()
	}

	// 读取输入寄存器
	fmt.Print("  → 读取输入寄存器 (地址30001-30005)...")
	inputs, err := modbus.ReadInputRegisters(slaveID, 0, 5)
	if err != nil {
		fmt.Printf(" 失败: %v\n", err)
	} else {
		fmt.Printf(" 成功\n")
		fmt.Printf("     数据: %v\n", inputs)
	}

	// 读取线圈
	fmt.Print("  → 读取线圈状态 (地址00001-00016)...")
	coils, err := modbus.ReadCoils(slaveID, 0, 16)
	if err != nil {
		fmt.Printf(" 失败: %v\n", err)
	} else {
		fmt.Printf(" 成功\n")
		fmt.Printf("     状态: ")
		for i, coil := range coils {
			if coil {
				fmt.Printf("%d:ON ", i+1)
			} else {
				fmt.Printf("%d:OFF ", i+1)
			}
		}
		fmt.Println()
	}

	// 读取离散输入
	fmt.Print("  → 读取离散输入 (地址10001-10008)...")
	discretes, err := modbus.ReadDiscreteInputs(slaveID, 0, 8)
	if err != nil {
		fmt.Printf(" 失败: %v\n", err)
	} else {
		fmt.Printf(" 成功\n")
		fmt.Printf("     状态: %v\n", discretes)
	}
}

// demonstrateDataMonitoring 演示数据监听
func demonstrateDataMonitoring(ctx context.Context, modbus *serial.Modbus) {
	slaveID := byte(1)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fmt.Println("启动数据监听 (每2秒轮询一次)")
	fmt.Println("监控寄存器 40001-40004 的值变化...")

	var lastValues []uint16

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			registers, err := modbus.ReadHoldingRegisters(slaveID, 0, 4)
			if err != nil {
				continue
			}

			// 检测变化
			if lastValues != nil {
				changed := false
				for i, val := range registers {
					if val != lastValues[i] {
						changed = true
						fmt.Printf("[监听] 寄存器 %d 变化: %d → %d\n", 40001+i, lastValues[i], val)
					}
				}
				if !changed {
					// 不输出，避免刷屏
				}
			}

			lastValues = registers
		}
	}
}

// demonstrateDeviceControl 演示设备管控
func demonstrateDeviceControl(ctx context.Context, modbus *serial.Modbus) {
	slaveID := byte(1)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Println("启动设备管控 (每5秒执行一次控制操作)")

	controlCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			controlCount++

			switch controlCount % 4 {
			case 1:
				// 写入单个寄存器
				value := uint16(1000 + controlCount)
				fmt.Printf("[管控] 写入寄存器 40001 = %d\n", value)
				err := modbus.WriteSingleRegister(slaveID, 0, value)
				if err != nil {
					fmt.Printf("       失败: %v\n", err)
				} else {
					fmt.Println("       ✓ 成功")
				}

			case 2:
				// 写入单个线圈
				value := controlCount%2 == 0
				fmt.Printf("[管控] 写入线圈 00001 = %v\n", value)
				err := modbus.WriteSingleCoil(slaveID, 0, value)
				if err != nil {
					fmt.Printf("       失败: %v\n", err)
				} else {
					fmt.Println("       ✓ 成功")
				}

			case 3:
				// 写入多个寄存器
				values := []uint16{100, 200, 300, 400}
				fmt.Printf("[管控] 写入寄存器 40001-40004 = %v\n", values)
				err := modbus.WriteMultipleRegisters(slaveID, 0, values)
				if err != nil {
					fmt.Printf("       失败: %v\n", err)
				} else {
					fmt.Println("       ✓ 成功")
				}

			case 0:
				// 写入多个线圈
				values := []bool{true, false, true, false, true, false}
				fmt.Printf("[管控] 写入线圈 00001-00006 = %v\n", values)
				err := modbus.WriteMultipleCoils(slaveID, 0, values)
				if err != nil {
					fmt.Printf("       失败: %v\n", err)
				} else {
					fmt.Println("       ✓ 成功")
				}
			}
		}
	}
}

// useSimulationMode 使用模拟模式演示
func useSimulationMode() {
	fmt.Println("\n=== 模拟模式演示 ===\n")

	// 模拟数据读取
	fmt.Println("1. 数据读取:")
	fmt.Println("   保持寄存器 40001-40010: [100, 200, 300, 400, 500, 600, 700, 800, 900, 1000]")
	fmt.Println("   输入寄存器 30001-30005: [25, 30, 35, 40, 45]")
	fmt.Println("   线圈状态 00001-00016:  [ON, OFF, ON, OFF, ON, OFF, ON, OFF, ...]")

	// 模拟数据监听
	fmt.Println("\n2. 数据监听:")
	fmt.Println("   持续轮询寄存器 40001-40004")
	fmt.Println("   检测数值变化并实时报告")

	// 模拟设备管控
	fmt.Println("\n3. 设备管控:")
	fmt.Println("   写入寄存器: 40001 = 1000")
	fmt.Println("   写入线圈: 00001 = ON")
	fmt.Println("   批量写入: 40001-40004 = [100, 200, 300, 400]")

	fmt.Println("\n提示: 要运行真实的Modbus通信，请:")
	fmt.Println("  1. 安装Modbus模拟器（如 pymodbus, modpoll等）")
	fmt.Println("  2. 运行Modbus TCP服务器在端口502")
	fmt.Println("  3. 或连接真实的Modbus RTU设备到串口")
}
