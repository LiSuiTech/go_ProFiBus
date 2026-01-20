package application

import (
	"fmt"
	"go_ProFiBus/serial"
)

// ProtocolType 定义支持的协议类型
type ProtocolType int

const (
	UART ProtocolType = iota
	CAN
	USB
	OneWire
	Modbus
	RS232
	RS485
	I2C
	SPI
)

// NewProtocolBus 创建一个新的协议总线实例
func NewProtocolBus(protocol ProtocolType, portName string) (serial.SerialPort, error) {
	var port serial.SerialPort

	switch protocol {
	case UART:
		port = &serial.UART{}
	case CAN:
		port = &serial.CAN{}
	case USB:
		port = &serial.USB{}
	case OneWire:
		port = &serial.OneWire{}
	case Modbus:
		port = &serial.Modbus{}
	case RS232:
		port = &serial.RS232{}
	case RS485:
		port = serial.NewRS485()
	case I2C:
		port = serial.NewI2C()
	case SPI:
		port = serial.NewSpiDevice()
	default:
		return nil, fmt.Errorf("不支持的协议类型")
	}

	// 打开串口
	if err := port.Open(portName); err != nil {
		return nil, fmt.Errorf("打开 %s 设备失败: %v", portName, err)
	}

	return port, nil
}

// ExampleUsage 示例用法
func ExampleUsage() {
	portName := "/dev/ttyUSB0" // 根据实际情况修改

	// 创建 UART 总线
	uartBus, err := NewProtocolBus(UART, portName)
	if err != nil {
		fmt.Println("错误:", err)
		return
	}
	defer uartBus.Close()

	// 写入数据
	dataToSend := []byte("Hello UART")
	if _, err := uartBus.Write(dataToSend); err != nil {
		fmt.Println("写入错误:", err)
		return
	}

	// 读取数据
	buffer := make([]byte, 100)
	n, err := uartBus.Read(buffer)
	if err != nil {
		fmt.Println("读取错误:", err)
		return
	}
	fmt.Printf("接收到数据: %s\n", buffer[:n])
}
