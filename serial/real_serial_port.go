package serial

import (
	"fmt"
	"os"
	"syscall"
)

// SerialConfig 是串口通信的配置。
type SerialConfig struct {
	PortName  string     // 端口名称
	BaudRate  int        // 波特率
	DataBits  int        // 数据位
	Parity    ParityMode // 校验位
	StopBits  int        // 停止位
	Simulated bool       // 是否使用模拟模式
	Address   int
}

// RealSerialPort 是一个真实的串口通信实现。
type RealSerialPort struct {
	file   *os.File
	config *SerialConfig
}

// NewSerialPort 返回一个串口通信实例。
func NewSerialPort(options ...Option) RealSerialPort {
	config := &SerialConfig{
		Simulated: true, // 默认使用模拟模式
	}

	// 应用选项
	for _, option := range options {
		option(config)
	}

	if config.Simulated {
		return RealSerialPort{config: config}
	}
	// 如果使用真实模式，返回相应的实际实现
	// TODO: 实现真实串口模式
	return RealSerialPort{config: config}
}

// Open 打开串口。
func (r *RealSerialPort) Open(portName string, options ...Option) error {
	// 打开串口设备
	file, err := os.OpenFile(portName, syscall.O_RDWR|syscall.O_NOCTTY, 0666)
	if err != nil {
		return fmt.Errorf("无法打开串口设备：%v", err)
	}
	r.file = file
	return nil
}

// Write 写入数据到串口。
func (r *RealSerialPort) Write(data []byte) (int, error) {
	if r.file == nil {
		return 0, fmt.Errorf("串口设备未打开")
	}
	n, err := r.file.Write(data)
	if err != nil {
		return n, fmt.Errorf("写入数据失败：%v", err)
	}
	return n, nil
}

// Read 从串口读取数据。
func (r *RealSerialPort) Read(buffer []byte) (int, error) {
	if r.file == nil {
		return 0, fmt.Errorf("串口设备未打开")
	}
	n, err := r.file.Read(buffer)
	if err != nil {
		return n, fmt.Errorf("读取数据失败：%v", err)
	}
	return n, nil
}

// Close 关闭串口。
func (r *RealSerialPort) Close() error {
	if r.file == nil {
		return fmt.Errorf("串口设备未打开")
	}
	err := r.file.Close()
	if err != nil {
		return fmt.Errorf("关闭串口设备失败：%v", err)
	}
	r.file = nil
	return nil
}
