package serial

import (
	"fmt"
	"os"
)

// I2C 是 I2C 串口通信的实现。
type I2C struct {
	config *SerialConfig
	file   *os.File
}

// WithAddress 设置 I2C 设备的地址。
func WithAddress(address int) Option {
	return func(config *SerialConfig) error {
		config.Address = address
		return nil
	}
}

// NewI2C 返回一个 I2C 串口通信实例。
func NewI2C(options ...Option) SerialPort {
	config := &SerialConfig{}

	// 应用选项
	for _, option := range options {
		option(config)
	}

	return &I2C{
		config: config,
	}
}

func (i *I2C) Open(portName string, options ...Option) error {
	// 实现打开 I2C 串口的逻辑
	file, err := os.OpenFile(portName, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("无法打开 I2C 设备：%s", err)
	}

	i.config.PortName = portName
	i.file = file
	return nil
}

func (i *I2C) Write(data []byte) (int, error) {
	// 实现向 I2C 串口写入数据的逻辑
	if i.file == nil {
		return 0, fmt.Errorf("I2C 设备未打开")
	}

	// 写入数据到设备
	n, err := i.file.Write(data)
	if err != nil {
		return n, fmt.Errorf("写入数据失败：%s", err)
	}
	return n, nil
}

func (i *I2C) Read(buffer []byte) (int, error) {
	// 实现从 I2C 串口读取数据的逻辑
	if i.file == nil {
		return 0, fmt.Errorf("I2C 设备未打开")
	}

	// 从设备读取数据
	n, err := i.file.Read(buffer)
	if err != nil {
		return n, fmt.Errorf("读取数据失败：%s", err)
	}

	return n, nil
}

func (i *I2C) Close() error {
	// 实现关闭 I2C 串口的逻辑
	if i.file == nil {
		return nil
	}
	err := i.file.Close()
	if err != nil {
		return err
	}
	return nil
}
