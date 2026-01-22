package serial

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Modbus功能码
const (
	FuncCodeReadCoils              = 0x01 // 读线圈
	FuncCodeReadDiscreteInputs     = 0x02 // 读离散输入
	FuncCodeReadHoldingRegisters   = 0x03 // 读保持寄存器
	FuncCodeReadInputRegisters     = 0x04 // 读输入寄存器
	FuncCodeWriteSingleCoil        = 0x05 // 写单个线圈
	FuncCodeWriteSingleRegister    = 0x06 // 写单个寄存器
	FuncCodeWriteMultipleCoils     = 0x0F // 写多个线圈
	FuncCodeWriteMultipleRegisters = 0x10 // 写多个寄存器
)

// Modbus异常码
const (
	ExceptionIllegalFunction    = 0x01
	ExceptionIllegalDataAddress = 0x02
	ExceptionIllegalDataValue   = 0x03
	ExceptionSlaveDeviceFailure = 0x04
)

// ModbusMode Modbus模式
type ModbusMode int

const (
	ModbusRTU ModbusMode = iota // Modbus RTU（串口）
	ModbusTCP                   // Modbus TCP（网络）
	ModbusASCII                 // Modbus ASCII（串口）
)

// ModbusRole Modbus角色
type ModbusRole int

const (
	ModbusMaster ModbusRole = iota // 主站
	ModbusSlave                     // 从站
)

// ModbusConfig Modbus配置
type ModbusConfig struct {
	Mode           ModbusMode    // 模式（RTU/TCP/ASCII）
	Role           ModbusRole    // 角色（Master/Slave）
	SlaveID        byte          // 从站地址（1-247）
	Timeout        time.Duration // 超时时间
	SerialPort     string        // 串口路径（RTU/ASCII模式）
	BaudRate       int           // 波特率
	DataBits       int           // 数据位
	StopBits       int           // 停止位
	Parity         string        // 校验位（N/E/O）
	TCPAddress     string        // TCP地址（TCP模式）
	MaxRetries     int           // 最大重试次数
}

// Modbus Modbus主站/从站实现
type Modbus struct {
	config    *ModbusConfig
	transport io.ReadWriteCloser // 底层传输（串口或TCP连接）
	mu        sync.Mutex
	txCounter uint16 // 事务ID计数器（TCP模式）
}

// NewModbusMaster 创建Modbus主站
func NewModbusMaster(config *ModbusConfig) (*Modbus, error) {
	if config.Role != ModbusMaster {
		return nil, fmt.Errorf("config role must be ModbusMaster")
	}

	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 1 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	mb := &Modbus{
		config:    config,
		txCounter: 0,
	}

	return mb, nil
}

// NewModbusSlave 创建Modbus从站
func NewModbusSlave(config *ModbusConfig) (*Modbus, error) {
	if config.Role != ModbusSlave {
		return nil, fmt.Errorf("config role must be ModbusSlave")
	}

	mb := &Modbus{
		config: config,
	}

	return mb, nil
}

// Open 打开连接
func (m *Modbus) Open(devicePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error

	switch m.config.Mode {
	case ModbusRTU, ModbusASCII:
		// 打开串口
		port := devicePath
		if port == "" {
			port = m.config.SerialPort
		}
		m.transport, err = m.openSerialPort(port)
		if err != nil {
			return fmt.Errorf("failed to open serial port: %w", err)
		}

	case ModbusTCP:
		// 连接TCP
		address := devicePath
		if address == "" {
			address = m.config.TCPAddress
		}
		m.transport, err = net.DialTimeout("tcp", address, m.config.Timeout)
		if err != nil {
			return fmt.Errorf("failed to connect TCP: %w", err)
		}

	default:
		return fmt.Errorf("unsupported modbus mode: %d", m.config.Mode)
	}

	return nil
}

// Close 关闭连接
func (m *Modbus) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.transport != nil {
		return m.transport.Close()
	}
	return nil
}

// ReadCoils 读线圈（功能码0x01）
func (m *Modbus) ReadCoils(slaveID byte, address uint16, quantity uint16) ([]bool, error) {
	if quantity < 1 || quantity > 2000 {
		return nil, fmt.Errorf("invalid quantity: %d (must be 1-2000)", quantity)
	}

	request := m.buildRequest(slaveID, FuncCodeReadCoils, address, quantity)
	response, err := m.sendRequest(request)
	if err != nil {
		return nil, err
	}

	return m.parseCoilsResponse(response, quantity)
}

// ReadDiscreteInputs 读离散输入（功能码0x02）
func (m *Modbus) ReadDiscreteInputs(slaveID byte, address uint16, quantity uint16) ([]bool, error) {
	if quantity < 1 || quantity > 2000 {
		return nil, fmt.Errorf("invalid quantity: %d (must be 1-2000)", quantity)
	}

	request := m.buildRequest(slaveID, FuncCodeReadDiscreteInputs, address, quantity)
	response, err := m.sendRequest(request)
	if err != nil {
		return nil, err
	}

	return m.parseCoilsResponse(response, quantity)
}

// ReadHoldingRegisters 读保持寄存器（功能码0x03）
func (m *Modbus) ReadHoldingRegisters(slaveID byte, address uint16, quantity uint16) ([]uint16, error) {
	if quantity < 1 || quantity > 125 {
		return nil, fmt.Errorf("invalid quantity: %d (must be 1-125)", quantity)
	}

	request := m.buildRequest(slaveID, FuncCodeReadHoldingRegisters, address, quantity)
	response, err := m.sendRequest(request)
	if err != nil {
		return nil, err
	}

	return m.parseRegistersResponse(response, quantity)
}

// ReadInputRegisters 读输入寄存器（功能码0x04）
func (m *Modbus) ReadInputRegisters(slaveID byte, address uint16, quantity uint16) ([]uint16, error) {
	if quantity < 1 || quantity > 125 {
		return nil, fmt.Errorf("invalid quantity: %d (must be 1-125)", quantity)
	}

	request := m.buildRequest(slaveID, FuncCodeReadInputRegisters, address, quantity)
	response, err := m.sendRequest(request)
	if err != nil {
		return nil, err
	}

	return m.parseRegistersResponse(response, quantity)
}

// WriteSingleCoil 写单个线圈（功能码0x05）
func (m *Modbus) WriteSingleCoil(slaveID byte, address uint16, value bool) error {
	var val uint16
	if value {
		val = 0xFF00
	} else {
		val = 0x0000
	}

	request := m.buildRequest(slaveID, FuncCodeWriteSingleCoil, address, val)
	_, err := m.sendRequest(request)
	return err
}

// WriteSingleRegister 写单个寄存器（功能码0x06）
func (m *Modbus) WriteSingleRegister(slaveID byte, address uint16, value uint16) error {
	request := m.buildRequest(slaveID, FuncCodeWriteSingleRegister, address, value)
	_, err := m.sendRequest(request)
	return err
}

// WriteMultipleCoils 写多个线圈（功能码0x0F）
func (m *Modbus) WriteMultipleCoils(slaveID byte, address uint16, values []bool) error {
	if len(values) < 1 || len(values) > 1968 {
		return fmt.Errorf("invalid values length: %d (must be 1-1968)", len(values))
	}

	request := m.buildWriteMultipleCoilsRequest(slaveID, address, values)
	_, err := m.sendRequest(request)
	return err
}

// WriteMultipleRegisters 写多个寄存器（功能码0x10）
func (m *Modbus) WriteMultipleRegisters(slaveID byte, address uint16, values []uint16) error {
	if len(values) < 1 || len(values) > 123 {
		return fmt.Errorf("invalid values length: %d (must be 1-123)", len(values))
	}

	request := m.buildWriteMultipleRegistersRequest(slaveID, address, values)
	_, err := m.sendRequest(request)
	return err
}

// buildRequest 构建请求
func (m *Modbus) buildRequest(slaveID byte, funcCode byte, address uint16, data uint16) []byte {
	switch m.config.Mode {
	case ModbusRTU:
		return m.buildRTURequest(slaveID, funcCode, address, data)
	case ModbusTCP:
		return m.buildTCPRequest(slaveID, funcCode, address, data)
	case ModbusASCII:
		return m.buildASCIIRequest(slaveID, funcCode, address, data)
	default:
		return nil
	}
}

// buildRTURequest 构建RTU请求
func (m *Modbus) buildRTURequest(slaveID byte, funcCode byte, address uint16, data uint16) []byte {
	pdu := make([]byte, 6)
	pdu[0] = slaveID
	pdu[1] = funcCode
	binary.BigEndian.PutUint16(pdu[2:4], address)
	binary.BigEndian.PutUint16(pdu[4:6], data)

	// 添加CRC
	crc := calculateCRC16(pdu)
	request := make([]byte, 8)
	copy(request, pdu)
	binary.LittleEndian.PutUint16(request[6:8], crc)

	return request
}

// buildTCPRequest 构建TCP请求
func (m *Modbus) buildTCPRequest(slaveID byte, funcCode byte, address uint16, data uint16) []byte {
	m.txCounter++

	mbap := make([]byte, 7) // MBAP Header
	binary.BigEndian.PutUint16(mbap[0:2], m.txCounter) // Transaction ID
	binary.BigEndian.PutUint16(mbap[2:4], 0x0000)     // Protocol ID
	binary.BigEndian.PutUint16(mbap[4:6], 6)          // Length
	mbap[6] = slaveID                                  // Unit ID

	pdu := make([]byte, 5)
	pdu[0] = funcCode
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], data)

	request := make([]byte, 12)
	copy(request[0:7], mbap)
	copy(request[7:12], pdu)

	return request
}

// buildASCIIRequest 构建ASCII请求
func (m *Modbus) buildASCIIRequest(slaveID byte, funcCode byte, address uint16, data uint16) []byte {
	pdu := make([]byte, 6)
	pdu[0] = slaveID
	pdu[1] = funcCode
	binary.BigEndian.PutUint16(pdu[2:4], address)
	binary.BigEndian.PutUint16(pdu[4:6], data)

	// 计算LRC
	lrc := calculateLRC(pdu)

	// 转换为ASCII
	ascii := make([]byte, 0, 17) // :+12个字符+2个LRC+\r\n
	ascii = append(ascii, ':')
	for i := 0; i < 6; i++ {
		ascii = append(ascii, fmt.Sprintf("%02X", pdu[i])...)
	}
	ascii = append(ascii, fmt.Sprintf("%02X", lrc)...)
	ascii = append(ascii, '\r', '\n')

	return ascii
}

// buildWriteMultipleCoilsRequest 构建写多个线圈请求
func (m *Modbus) buildWriteMultipleCoilsRequest(slaveID byte, address uint16, values []bool) []byte {
	quantity := uint16(len(values))
	byteCount := (quantity + 7) / 8

	pdu := make([]byte, 0)
	pdu = append(pdu, slaveID, FuncCodeWriteMultipleCoils)
	pdu = append(pdu, byte(address>>8), byte(address))
	pdu = append(pdu, byte(quantity>>8), byte(quantity))
	pdu = append(pdu, byte(byteCount))

	// 打包线圈值
	bytes := make([]byte, byteCount)
	for i, val := range values {
		if val {
			bytes[i/8] |= 1 << (i % 8)
		}
	}
	pdu = append(pdu, bytes...)

	// 添加CRC/LRC根据模式
	switch m.config.Mode {
	case ModbusRTU:
		crc := calculateCRC16(pdu)
		pdu = append(pdu, byte(crc), byte(crc>>8))
	case ModbusASCII:
		lrc := calculateLRC(pdu)
		// 转换为ASCII...
		_ = lrc // TODO: 完整实现
	case ModbusTCP:
		// 添加MBAP头
		m.txCounter++
		mbap := make([]byte, 7)
		binary.BigEndian.PutUint16(mbap[0:2], m.txCounter)
		binary.BigEndian.PutUint16(mbap[2:4], 0x0000)
		binary.BigEndian.PutUint16(mbap[4:6], uint16(len(pdu)-1))
		mbap[6] = slaveID
		pdu = append(mbap, pdu[1:]...) // 去掉slaveID重复
	}

	return pdu
}

// buildWriteMultipleRegistersRequest 构建写多个寄存器请求
func (m *Modbus) buildWriteMultipleRegistersRequest(slaveID byte, address uint16, values []uint16) []byte {
	quantity := uint16(len(values))
	byteCount := quantity * 2

	pdu := make([]byte, 0)
	pdu = append(pdu, slaveID, FuncCodeWriteMultipleRegisters)
	pdu = append(pdu, byte(address>>8), byte(address))
	pdu = append(pdu, byte(quantity>>8), byte(quantity))
	pdu = append(pdu, byte(byteCount))

	// 添加寄存器值
	for _, val := range values {
		pdu = append(pdu, byte(val>>8), byte(val))
	}

	// 添加CRC/LRC根据模式
	switch m.config.Mode {
	case ModbusRTU:
		crc := calculateCRC16(pdu)
		pdu = append(pdu, byte(crc), byte(crc>>8))
	case ModbusTCP:
		// 添加MBAP头
		m.txCounter++
		mbap := make([]byte, 7)
		binary.BigEndian.PutUint16(mbap[0:2], m.txCounter)
		binary.BigEndian.PutUint16(mbap[2:4], 0x0000)
		binary.BigEndian.PutUint16(mbap[4:6], uint16(len(pdu)-1))
		mbap[6] = slaveID
		pdu = append(mbap, pdu[1:]...)
	}

	return pdu
}

// sendRequest 发送请求并接收响应
func (m *Modbus) sendRequest(request []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.transport == nil {
		return nil, fmt.Errorf("modbus not connected")
	}

	// 发送请求
	_, err := m.transport.Write(request)
	if err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	// 接收响应
	response := make([]byte, 260) // 最大响应长度

	// 设置读取超时
	if conn, ok := m.transport.(net.Conn); ok {
		conn.SetReadDeadline(time.Now().Add(m.config.Timeout))
	}

	n, err := m.transport.Read(response)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	return response[:n], nil
}

// parseCoilsResponse 解析线圈响应
func (m *Modbus) parseCoilsResponse(response []byte, quantity uint16) ([]bool, error) {
	var dataStart int

	switch m.config.Mode {
	case ModbusRTU:
		if len(response) < 5 {
			return nil, fmt.Errorf("response too short")
		}
		// 验证CRC
		if !verifyCRC16(response) {
			return nil, fmt.Errorf("CRC check failed")
		}
		dataStart = 3

	case ModbusTCP:
		if len(response) < 9 {
			return nil, fmt.Errorf("response too short")
		}
		dataStart = 9
	}

	byteCount := int(response[dataStart-1])
	data := response[dataStart : dataStart+byteCount]

	// 解析位
	coils := make([]bool, quantity)
	for i := uint16(0); i < quantity; i++ {
		byteIdx := i / 8
		bitIdx := i % 8
		coils[i] = (data[byteIdx] & (1 << bitIdx)) != 0
	}

	return coils, nil
}

// parseRegistersResponse 解析寄存器响应
func (m *Modbus) parseRegistersResponse(response []byte, quantity uint16) ([]uint16, error) {
	var dataStart int

	switch m.config.Mode {
	case ModbusRTU:
		if len(response) < 5 {
			return nil, fmt.Errorf("response too short")
		}
		if !verifyCRC16(response) {
			return nil, fmt.Errorf("CRC check failed")
		}
		dataStart = 3

	case ModbusTCP:
		if len(response) < 9 {
			return nil, fmt.Errorf("response too short")
		}
		dataStart = 9
	}

	byteCount := int(response[dataStart-1])
	data := response[dataStart : dataStart+byteCount]

	// 解析寄存器
	registers := make([]uint16, quantity)
	for i := uint16(0); i < quantity; i++ {
		registers[i] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
	}

	return registers, nil
}

// calculateCRC16 计算CRC-16（Modbus）
func calculateCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// verifyCRC16 验证CRC-16
func verifyCRC16(data []byte) bool {
	if len(data) < 3 {
		return false
	}
	dataWithoutCRC := data[:len(data)-2]
	receivedCRC := binary.LittleEndian.Uint16(data[len(data)-2:])
	calculatedCRC := calculateCRC16(dataWithoutCRC)
	return receivedCRC == calculatedCRC
}

// calculateLRC 计算LRC（纵向冗余校验）
func calculateLRC(data []byte) byte {
	var lrc byte
	for _, b := range data {
		lrc += b
	}
	return byte(-int8(lrc))
}

// openSerialPort 打开串口
func (m *Modbus) openSerialPort(portName string) (io.ReadWriteCloser, error) {
	// 使用RealSerialPort打开串口
	port := &RealSerialPort{}
	config := PortConfig{
		BaudRate: m.config.BaudRate,
		DataBits: m.config.DataBits,
		StopBits: m.config.StopBits,
		Parity:   m.config.Parity,
	}

	err := port.OpenWithConfig(portName, config)
	if err != nil {
		return nil, err
	}

	return port, nil
}

// SlaveHandler Modbus从站处理器接口
type SlaveHandler interface {
	ReadCoils(address uint16, quantity uint16) ([]bool, error)
	ReadDiscreteInputs(address uint16, quantity uint16) ([]bool, error)
	ReadHoldingRegisters(address uint16, quantity uint16) ([]uint16, error)
	ReadInputRegisters(address uint16, quantity uint16) ([]uint16, error)
	WriteSingleCoil(address uint16, value bool) error
	WriteSingleRegister(address uint16, value uint16) error
	WriteMultipleCoils(address uint16, values []bool) error
	WriteMultipleRegisters(address uint16, values []uint16) error
}

// Read 实现io.Reader接口（用于从站接收请求）
func (m *Modbus) Read(buffer []byte) (int, error) {
	if m.transport == nil {
		return 0, fmt.Errorf("modbus not connected")
	}
	return m.transport.Read(buffer)
}

// Write 实现io.Writer接口（用于从站发送响应）
func (m *Modbus) Write(data []byte) (int, error) {
	if m.transport == nil {
		return 0, fmt.Errorf("modbus not connected")
	}
	return m.transport.Write(data)
}
