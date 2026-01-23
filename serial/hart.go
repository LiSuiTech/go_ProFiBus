package serial

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// HART HART协议实现 (Highway Addressable Remote Transducer)
// HART是过程自动化行业的混合协议，在4-20mA模拟信号上叠加数字通信
type HART struct {
	config     *HARTConfig
	port       io.ReadWriteCloser
	mu         sync.RWMutex
	monitoring bool
	dataChan   chan *interfaces.DataUpdate
	stopChan   chan struct{}
	stats      *interfaces.ProtocolStatus
	devices    map[byte]*HARTDevice // 设备列表 (地址 -> 设备)
}

// HARTConfig HART配置
type HARTConfig struct {
	// 串口配置
	SerialPort string // 串口设备路径
	BaudRate   int    // 波特率 (通常1200)
	DataBits   int    // 数据位 (通常8)
	StopBits   int    // 停止位 (通常1)
	Parity     ParityMode

	// HART配置
	Mode           HARTMode     // HART模式
	MasterAddress  byte         // 主站地址 (0-1)
	SlaveAddresses []byte       // 从站地址列表
	PollAddress    byte         // 轮询地址 (点对点模式=0, 多点模式=1-15)
	RetryCount     int          // 重试次数
	Timeout        time.Duration // 超时时间
	PreambleCount  int          // 前导字节数 (5-20)

	// 轮询配置
	PollingInterval time.Duration // 轮询间隔
	PollingEnabled  bool          // 是否启用轮询
}

// HARTMode HART模式
type HARTMode string

const (
	HARTModePointToPoint HARTMode = "point_to_point" // 点对点模式 (4-20mA + 数字)
	HARTModeMultidrop    HARTMode = "multidrop"      // 多点模式 (仅数字, 4mA固定)
)

// HARTDevice HART设备信息
type HARTDevice struct {
	Address          byte   // 设备地址
	Manufacturer     uint16 // 制造商ID
	DeviceType       byte   // 设备类型
	DeviceID         [3]byte // 设备ID
	UniversalRev     byte   // 通用命令版本
	TransmitterRev   byte   // 变送器固件版本
	SoftwareRev      byte   // 软件版本
	Tag              string // 设备标签
	Descriptor       string // 描述符
	Message          string // 消息
	PrimaryVariable  float32 // 主变量值
	PrimaryVariableUnit byte  // 主变量单位
	Active           bool    // 是否在线
}

// HART命令代码
const (
	HARTCmdReadUniqueID            = 0  // 读取唯一ID
	HARTCmdReadPrimaryVariable     = 1  // 读取主变量
	HARTCmdReadCurrentAndPercent   = 2  // 读取电流和百分比
	HARTCmdReadDynamicVariables    = 3  // 读取动态变量
	HARTCmdReadTag                 = 13 // 读取标签、描述符、日期
	HARTCmdReadMessage             = 12 // 读取消息
	HARTCmdWriteMessage            = 17 // 写入消息
	HARTCmdWriteTag                = 18 // 写入标签
	HARTCmdReadDeviceInfo          = 48 // 读取设备信息
)

// Validate 验证配置
func (c *HARTConfig) Validate() error {
	if c.SerialPort == "" {
		return fmt.Errorf("serial_port is required")
	}

	if c.BaudRate == 0 {
		c.BaudRate = 1200 // HART标准波特率
	}
	if c.BaudRate != 1200 {
		return fmt.Errorf("HART requires 1200 baud rate")
	}

	if c.DataBits == 0 {
		c.DataBits = 8
	}
	if c.StopBits == 0 {
		c.StopBits = 1
	}
	if c.Parity == "" {
		c.Parity = ParityOdd // HART使用奇校验
	}

	if c.Mode == "" {
		c.Mode = HARTModePointToPoint
	}

	if c.PreambleCount == 0 {
		c.PreambleCount = 5
	}
	if c.PreambleCount < 5 || c.PreambleCount > 20 {
		return fmt.Errorf("preamble count must be 5-20")
	}

	if c.RetryCount == 0 {
		c.RetryCount = 3
	}
	if c.Timeout == 0 {
		c.Timeout = 500 * time.Millisecond
	}
	if c.PollingInterval == 0 {
		c.PollingInterval = 2 * time.Second
	}

	return nil
}

// GetType 获取配置类型
func (c *HARTConfig) GetType() interfaces.ProtocolType {
	return interfaces.ProtocolHART
}

// NewHART 创建HART协议实例
func NewHART(config *HARTConfig) (*HART, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &HART{
		config:   config,
		dataChan: make(chan *interfaces.DataUpdate, 1000),
		stopChan: make(chan struct{}),
		devices:  make(map[byte]*HARTDevice),
		stats: &interfaces.ProtocolStatus{
			Connected: false,
			Running:   false,
		},
	}, nil
}

// Open 打开HART连接
func (h *HART) Open(devicePath string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 打开串口
	port := &RealSerialPort{}
	err := port.Open(h.config.SerialPort)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}

	h.port = port
	h.stats.Connected = true

	// 扫描设备
	if len(h.config.SlaveAddresses) > 0 {
		for _, addr := range h.config.SlaveAddresses {
			device, err := h.identifyDevice(addr)
			if err == nil {
				h.devices[addr] = device
				device.Active = true
			}
		}
	}

	return nil
}

// Close 关闭HART连接
func (h *HART) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.monitoring {
		close(h.stopChan)
		h.monitoring = false
	}

	if h.port != nil {
		err := h.port.Close()
		h.port = nil
		h.stats.Connected = false
		return err
	}

	return nil
}

// identifyDevice 识别设备
func (h *HART) identifyDevice(address byte) (*HARTDevice, error) {
	// 发送命令0 - 读取唯一ID
	response, err := h.sendCommand(address, HARTCmdReadUniqueID, nil)
	if err != nil {
		return nil, err
	}

	if len(response) < 14 {
		return nil, fmt.Errorf("invalid response length")
	}

	device := &HARTDevice{
		Address:        address,
		Manufacturer:   binary.BigEndian.Uint16(response[0:2]),
		DeviceType:     response[2],
		UniversalRev:   response[6],
		TransmitterRev: response[7],
		SoftwareRev:    response[8],
	}

	copy(device.DeviceID[:], response[3:6])

	return device, nil
}

// ReadPrimaryVariable 读取主变量 (温度、压力等)
func (h *HART) ReadPrimaryVariable(address byte) (float32, byte, error) {
	h.mu.RLock()
	if h.port == nil {
		h.mu.RUnlock()
		return 0, 0, fmt.Errorf("not connected")
	}
	h.mu.RUnlock()

	// 发送命令1 - 读取主变量
	response, err := h.sendCommand(address, HARTCmdReadPrimaryVariable, nil)
	if err != nil {
		return 0, 0, err
	}

	if len(response) < 5 {
		return 0, 0, fmt.Errorf("invalid response length")
	}

	// 解析主变量 (4字节IEEE-754浮点数)
	value := binary.BigEndian.Uint32(response[0:4])
	primaryVariable := float32(value)

	// 主变量单位代码
	unit := response[4]

	// 更新设备状态
	if device, exists := h.devices[address]; exists {
		device.PrimaryVariable = primaryVariable
		device.PrimaryVariableUnit = unit
	}

	return primaryVariable, unit, nil
}

// ReadTag 读取设备标签和描述符
func (h *HART) ReadTag(address byte) (string, string, error) {
	response, err := h.sendCommand(address, HARTCmdReadTag, nil)
	if err != nil {
		return "", "", err
	}

	if len(response) < 21 {
		return "", "", fmt.Errorf("invalid response length")
	}

	tag := string(response[0:8])
	descriptor := string(response[8:20])

	// 更新设备信息
	if device, exists := h.devices[address]; exists {
		device.Tag = tag
		device.Descriptor = descriptor
	}

	return tag, descriptor, nil
}

// WriteMessage 写入设备消息
func (h *HART) WriteMessage(address byte, message string) error {
	if len(message) > 24 {
		message = message[:24]
	}

	// 填充到24字节
	data := make([]byte, 24)
	copy(data, []byte(message))

	_, err := h.sendCommand(address, HARTCmdWriteMessage, data)
	return err
}

// sendCommand 发送HART命令
func (h *HART) sendCommand(address byte, command byte, data []byte) ([]byte, error) {
	startTime := time.Now()

	// 构建HART帧
	frame := h.buildFrame(address, command, data)

	var response []byte
	var err error

	// 重试机制
	for attempt := 0; attempt <= h.config.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(50 * time.Millisecond)
		}

		// 发送帧
		_, err = h.port.Write(frame)
		if err != nil {
			h.stats.ErrorCount++
			continue
		}

		h.stats.BytesSent += uint64(len(frame))
		h.stats.RequestCount++

		// 接收响应
		response, err = h.receiveFrame()
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("command failed after %d retries: %w", h.config.RetryCount, err)
	}

	h.stats.CurrentLatency = time.Since(startTime)
	h.stats.BytesReceived += uint64(len(response))

	// 验证响应
	if len(response) < 7 {
		return nil, fmt.Errorf("response too short")
	}

	// 检查响应码 (字节5)
	responseCode := response[5]
	if responseCode != 0 {
		return nil, fmt.Errorf("device error: response code %d", responseCode)
	}

	// 返回数据部分 (跳过帧头和CRC)
	dataStart := 6
	dataEnd := len(response) - 1 // 减去CRC
	if dataEnd <= dataStart {
		return []byte{}, nil
	}

	return response[dataStart:dataEnd], nil
}

// buildFrame 构建HART帧
func (h *HART) buildFrame(address byte, command byte, data []byte) []byte {
	frame := make([]byte, 0, 256)

	// 1. 前导字节 (0xFF)
	for i := 0; i < h.config.PreambleCount; i++ {
		frame = append(frame, 0xFF)
	}

	// 2. 起始字节
	// 格式: [1位起始标志(1)] [1位地址类型] [1位主站类型] [2位物理层] [3位帧类型]
	// 简化: 0x02 (短帧格式, 主站请求)
	startByte := byte(0x02)
	if h.config.Mode == HARTModeMultidrop {
		startByte |= 0x80 // 设置地址类型位为长地址
	}
	frame = append(frame, startByte)

	// 3. 地址字节
	frame = append(frame, address)

	// 4. 命令字节
	frame = append(frame, command)

	// 5. 字节计数 (数据长度)
	frame = append(frame, byte(len(data)))

	// 6. 数据
	if len(data) > 0 {
		frame = append(frame, data...)
	}

	// 7. 校验和 (纵向奇偶校验)
	checksum := h.calculateChecksum(frame[h.config.PreambleCount:])
	frame = append(frame, checksum)

	return frame
}

// receiveFrame 接收HART响应帧
func (h *HART) receiveFrame() ([]byte, error) {
	buffer := make([]byte, 256)
	totalRead := 0

	// 设置超时
	deadline := time.Now().Add(h.config.Timeout)

	// 读取响应 (简化实现)
	for time.Now().Before(deadline) {
		n, err := h.port.Read(buffer[totalRead:])
		if err != nil {
			return nil, err
		}

		totalRead += n

		// 检查是否收到完整帧 (至少7字节: 起始+地址+命令+字节计数+响应码+状态+校验)
		if totalRead >= 7 {
			// 解析字节计数
			if totalRead >= 4 {
				byteCount := int(buffer[3])
				expectedLength := 6 + byteCount + 1 // 起始+地址+命令+计数+响应码+状态+数据+校验
				if totalRead >= expectedLength {
					// 验证校验和
					checksum := h.calculateChecksum(buffer[:expectedLength-1])
					if checksum == buffer[expectedLength-1] {
						return buffer[:expectedLength], nil
					}
					return nil, fmt.Errorf("checksum error")
				}
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for response")
}

// calculateChecksum 计算纵向奇偶校验
func (h *HART) calculateChecksum(data []byte) byte {
	var checksum byte = 0
	for _, b := range data {
		checksum ^= b
	}
	return checksum
}

// StartCyclicPolling 启动循环轮询
func (h *HART) StartCyclicPolling(ctx context.Context) error {
	h.mu.Lock()
	if h.monitoring {
		h.mu.Unlock()
		return fmt.Errorf("polling already started")
	}

	h.monitoring = true
	h.stats.Running = true
	h.mu.Unlock()

	go h.pollingLoop(ctx)

	return nil
}

// StopPolling 停止轮询
func (h *HART) StopPolling() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.monitoring {
		return nil
	}

	h.stopChan <- struct{}{}
	h.monitoring = false
	h.stats.Running = false

	return nil
}

// pollingLoop 轮询循环
func (h *HART) pollingLoop(ctx context.Context) {
	ticker := time.NewTicker(h.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopChan:
			return
		case <-ticker.C:
			// 轮询所有设备
			for addr, device := range h.devices {
				if !device.Active {
					continue
				}

				// 读取主变量
				value, unit, err := h.ReadPrimaryVariable(addr)
				if err != nil {
					continue
				}

				// 发送数据更新
				update := &interfaces.DataUpdate{
					TargetID:  fmt.Sprintf("HART_Device_%d", addr),
					Value:     value,
					OldValue:  device.PrimaryVariable,
					Quality:   1.0,
					Timestamp: time.Now(),
					ChangeType: "value",
					Metadata: map[string]interface{}{
						"protocol":   "HART",
						"address":    addr,
						"unit":       unit,
						"tag":        device.Tag,
						"descriptor": device.Descriptor,
					},
				}

				select {
				case h.dataChan <- update:
				default:
					// 缓冲区满，丢弃
				}
			}
		}
	}
}

// GetDataChannel 获取数据通道
func (h *HART) GetDataChannel() <-chan *interfaces.DataUpdate {
	return h.dataChan
}

// GetStatus 获取协议状态
func (h *HART) GetStatus() *interfaces.ProtocolStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status := *h.stats
	return &status
}

// GetAllDevices 获取所有设备
func (h *HART) GetAllDevices() map[byte]*HARTDevice {
	h.mu.RLock()
	defer h.mu.RUnlock()

	devices := make(map[byte]*HARTDevice)
	for addr, device := range h.devices {
		deviceCopy := *device
		devices[addr] = &deviceCopy
	}

	return devices
}

// Read 实现io.Reader接口
func (h *HART) Read(buffer []byte) (int, error) {
	if h.port == nil {
		return 0, fmt.Errorf("not connected")
	}
	return h.port.Read(buffer)
}

// Write 实现io.Writer接口
func (h *HART) Write(data []byte) (int, error) {
	if h.port == nil {
		return 0, fmt.Errorf("not connected")
	}
	return h.port.Write(data)
}
