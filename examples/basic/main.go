package main

import (
	"fmt"
	"go_ProFiBus/application"
	"go_ProFiBus/logger"
	"log"
	"time"
)

func main() {
	// 设置日志级别
	logger.SetLevel(logger.INFO)

	fmt.Println("=== go_ProFiBus 基础示例 ===")
	fmt.Println()

	// 示例1: UART通信
	runUARTExample()

	// 示例2: RS485通信
	runRS485Example()
}

func runUARTExample() {
	fmt.Println("1. UART 通信示例")
	fmt.Println("-------------------")

	portName := "/dev/ttyUSB0" // 根据实际情况修改

	// 创建 UART 总线
	uartBus, err := application.NewProtocolBus(application.UART, portName)
	if err != nil {
		log.Printf("警告: 无法打开UART设备: %v (这在没有实际硬件时是正常的)", err)
		fmt.Println()
		return
	}
	defer uartBus.Close()

	// 写入数据
	dataToSend := []byte("Hello UART")
	n, err := uartBus.Write(dataToSend)
	if err != nil {
		log.Printf("写入错误: %v", err)
		return
	}
	fmt.Printf("成功写入 %d 字节: %s\n", n, string(dataToSend))

	// 读取数据
	buffer := make([]byte, 100)
	n, err = uartBus.Read(buffer)
	if err != nil {
		log.Printf("读取错误: %v", err)
		return
	}
	fmt.Printf("接收到 %d 字节: %s\n", n, string(buffer[:n]))
	fmt.Println()
}

func runRS485Example() {
	fmt.Println("2. RS485 通信示例")
	fmt.Println("-------------------")

	portName := "/dev/ttyUSB1" // 根据实际情况修改

	// 创建 RS485 总线
	rs485Bus, err := application.NewProtocolBus(application.RS485, portName)
	if err != nil {
		log.Printf("警告: 无法打开RS485设备: %v (这在没有实际硬件时是正常的)", err)
		fmt.Println()
		return
	}
	defer rs485Bus.Close()

	// 发送命令
	command := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A} // Modbus RTU 读取命令示例
	n, err := rs485Bus.Write(command)
	if err != nil {
		log.Printf("写入错误: %v", err)
		return
	}
	fmt.Printf("发送命令 %d 字节: % X\n", n, command)

	// 等待响应
	time.Sleep(100 * time.Millisecond)

	// 读取响应
	response := make([]byte, 256)
	n, err = rs485Bus.Read(response)
	if err != nil {
		log.Printf("读取错误: %v", err)
		return
	}
	fmt.Printf("接收响应 %d 字节: % X\n", n, response[:n])
	fmt.Println()
}
