package serial

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"go_ProFiBus/pkg/interfaces"
)

// EtherNetIP EtherNet/IP协议实现
// EtherNet/IP是Rockwell Automation (Allen-Bradley)的工业以太网协议
// 基于CIP (Common Industrial Protocol)
type EtherNetIP struct {
	config     *EtherNetIPConfig
	conn       net.Conn
	mu         sync.RWMutex
	monitoring bool
	dataChan   chan *interfaces.DataUpdate
	stopChan   chan struct{}
	stats      *interfaces.ProtocolStatus
	devices    map[string]*EtherNetIPDevice // 设备列表 (IP -> 设备)
	sessionHandle uint32                     // 会话句柄
}

// EtherNetIPConfig EtherNet/IP配置
type EtherNetIPConfig struct {
	// 网络配置
	TargetIP   string // 目标PLC IP
	TargetPort int    // 目标端口 (通常44818)

	// 设备配置
	DeviceIPs  []string // 设备IP列表
	Slot       byte     // PLC槽位号 (通常0-7)
	Backplane  byte     // 背板类型

	// CIP路由
	Route      []byte // CIP路由路径

	// 通信配置
	Timeout    time.Duration // 超时时间
	RetryCount int           // 重试次数

	// RPI配置 (Requested Packet Interval)
	RPI        time.Duration // 请求包间隔

	// 轮询配置
	PollingInterval time.Duration
	PollingEnabled  bool
}

// EtherNetIPDevice EtherNet/IP设备
type EtherNetIPDevice struct {
	IP             string // IP地址
	ProductName    string // 产品名称
	VendorID       uint16 // 供应商ID (1 = Rockwell Automation)
	DeviceType     uint16 // 设备类型
	ProductCode    uint16 // 产品代码
	MajorRevision  byte   // 主版本号
	MinorRevision  byte   // 次版本号
	SerialNumber   uint32 // 序列号
	ProductRevision string // 产品版本

	// 连接信息
	SessionHandle  uint32 // 会话句柄
	ConnectionID   uint32 // 连接ID
	SequenceNumber uint16 // 序列号

	// I/O数据
	InputData      []byte // 输入数据
	OutputData     []byte // 输出数据
	InputSize      int    // 输入数据大小
	OutputSize     int    // 输出数据大小

	// 状态
	State          EtherNetIPDeviceState // 设备状态
	Active         bool                  // 是否在线
	LastUpdateTime time.Time             // 最后更新时间
}

// EtherNetIPDeviceState 设备状态
type EtherNetIPDeviceState byte

const (
	EtherNetIPStateOffline  EtherNetIPDeviceState = 0x00 // 离线
	EtherNetIPStateConnected EtherNetIPDeviceState = 0x01 // 已连接
	EtherNetIPStateRunning   EtherNetIPDeviceState = 0x02 // 运行中
	EtherNetIPStateFault     EtherNetIPDeviceState = 0x03 // 故障
)

// EtherNet/IP命令代码
const (
	EtherNetIPCmdNOP              = 0x0000 // 无操作
	EtherNetIPCmdListServices     = 0x0004 // 列出服务
	EtherNetIPCmdListIdentity     = 0x0063 // 列出身份
	EtherNetIPCmdListInterfaces   = 0x0064 // 列出接口
	EtherNetIPCmdRegisterSession  = 0x0065 // 注册会话
	EtherNetIPCmdUnregisterSession = 0x0066 // 注销会话
	EtherNetIPCmdSendRRData       = 0x006F // 发送RR数据
	EtherNetIPCmdSendUnitData     = 0x0070 // 发送单元数据
)

// CIP服务代码
const (
	CIPServiceGetAttributesAll    = 0x01 // 获取所有属性
	CIPServiceSetAttributeSingle  = 0x02 // 设置单个属性
	CIPServiceGetAttributeSingle  = 0x0E // 获取单个属性
	CIPServiceReset               = 0x05 // 复位
	CIPServiceStart               = 0x06 // 启动
	CIPServiceStop                = 0x07 // 停止
	CIPServiceCreate              = 0x08 // 创建
	CIPServiceDelete              = 0x09 // 删除
	CIPServiceForwardOpen         = 0x54 // 前向打开
	CIPServiceForwardClose        = 0x4E // 前向关闭
	CIPServiceUnconnectedSend     = 0x52 // 无连接发送
	CIPServiceGetConnectionData   = 0x56 // 获取连接数据
)

// CIP对象类ID
const (
	CIPClassIdentity      = 0x01 // 身份对象
	CIPClassMessageRouter = 0x02 // 消息路由器
	CIPClassAssembly      = 0x04 // 装配对象
	CIPClassConnection    = 0x05 // 连接对象
	CIPClassConnectionManager = 0x06 // 连接管理器
)

// EtherNet/IP端口
const (
	EtherNetIPPortDefault = 44818 // 默认端口 (0xAF12)
)

// Validate 验证配置
func (c *EtherNetIPConfig) Validate() error {
	if c.TargetIP == "" {
		return fmt.Errorf("target_ip is required")
	}

	if c.TargetPort == 0 {
		c.TargetPort = EtherNetIPPortDefault
	}

	if c.Timeout == 0 {
		c.Timeout = 2 * time.Second
	}

	if c.RetryCount == 0 {
		c.RetryCount = 3
	}

	if c.RPI == 0 {
		c.RPI = 100 * time.Millisecond
	}

	if c.PollingInterval == 0 {
		c.PollingInterval = 100 * time.Millisecond
	}

	// 默认路由: Backplane, Slot
	if len(c.Route) == 0 {
		c.Route = []byte{0x01, c.Backplane, 0x01, c.Slot}
	}

	return nil
}

// GetType 获取配置类型
func (c *EtherNetIPConfig) GetType() interfaces.ProtocolType {
	return interfaces.ProtocolEtherNetIP
}

// NewEtherNetIP 创建EtherNet/IP实例
func NewEtherNetIP(config *EtherNetIPConfig) (*EtherNetIP, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &EtherNetIP{
		config:   config,
		dataChan: make(chan *interfaces.DataUpdate, 1000),
		stopChan: make(chan struct{}),
		devices:  make(map[string]*EtherNetIPDevice),
		stats: &interfaces.ProtocolStatus{
			Connected: false,
			Running:   false,
		},
	}, nil
}

// Open 打开EtherNet/IP连接
func (e *EtherNetIP) Open(devicePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 连接到PLC
	addr := fmt.Sprintf("%s:%d", e.config.TargetIP, e.config.TargetPort)
	conn, err := net.DialTimeout("tcp", addr, e.config.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	e.conn = conn

	// 注册会话
	sessionHandle, err := e.registerSession()
	if err != nil {
		e.conn.Close()
		return fmt.Errorf("failed to register session: %w", err)
	}

	e.sessionHandle = sessionHandle
	e.stats.Connected = true

	// 识别设备
	device, err := e.listIdentity()
	if err == nil {
		e.devices[e.config.TargetIP] = device
		device.Active = true
		device.SessionHandle = sessionHandle
	}

	return nil
}

// Close 关闭EtherNet/IP连接
func (e *EtherNetIP) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.monitoring {
		close(e.stopChan)
		e.monitoring = false
	}

	if e.conn != nil {
		// 注销会话
		e.unregisterSession()

		err := e.conn.Close()
		e.conn = nil
		e.stats.Connected = false
		return err
	}

	return nil
}

// registerSession 注册会话
func (e *EtherNetIP) registerSession() (uint32, error) {
	// 构建RegisterSession请求
	request := e.buildEncapsulationHeader(EtherNetIPCmdRegisterSession, 0, 4)

	// Session register data
	sessionData := make([]byte, 4)
	binary.LittleEndian.PutUint16(sessionData[0:2], 0x0001) // Protocol version
	binary.LittleEndian.PutUint16(sessionData[2:4], 0x0000) // Options flags

	request = append(request, sessionData...)

	// 发送请求
	_, err := e.conn.Write(request)
	if err != nil {
		return 0, err
	}

	e.stats.BytesSent += uint64(len(request))
	e.stats.RequestCount++

	// 接收响应
	buffer := make([]byte, 1024)
	n, err := e.conn.Read(buffer)
	if err != nil {
		return 0, err
	}

	e.stats.BytesReceived += uint64(n)

	// 解析响应
	if n >= 24 {
		// 检查命令码
		cmd := binary.LittleEndian.Uint16(buffer[0:2])
		if cmd != EtherNetIPCmdRegisterSession {
			return 0, fmt.Errorf("unexpected command: 0x%04X", cmd)
		}

		// 提取会话句柄
		sessionHandle := binary.LittleEndian.Uint32(buffer[4:8])
		return sessionHandle, nil
	}

	return 0, fmt.Errorf("invalid response")
}

// unregisterSession 注销会话
func (e *EtherNetIP) unregisterSession() error {
	request := e.buildEncapsulationHeader(EtherNetIPCmdUnregisterSession, e.sessionHandle, 0)

	_, err := e.conn.Write(request)
	return err
}

// listIdentity 列出设备身份
func (e *EtherNetIP) listIdentity() (*EtherNetIPDevice, error) {
	// 使用无连接的ListIdentity命令
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", e.config.TargetIP, e.config.TargetPort))
	if err != nil {
		return nil, err
	}

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()

	// 构建ListIdentity请求
	request := e.buildEncapsulationHeader(EtherNetIPCmdListIdentity, 0, 0)

	_, err = udpConn.Write(request)
	if err != nil {
		return nil, err
	}

	// 接收响应
	buffer := make([]byte, 1024)
	udpConn.SetReadDeadline(time.Now().Add(e.config.Timeout))
	n, err := udpConn.Read(buffer)
	if err != nil {
		return nil, err
	}

	// 解析响应
	device := &EtherNetIPDevice{
		IP:    e.config.TargetIP,
		State: EtherNetIPStateConnected,
	}

	if n >= 63 {
		// 解析Identity对象 (简化)
		device.VendorID = binary.LittleEndian.Uint16(buffer[48:50])
		device.DeviceType = binary.LittleEndian.Uint16(buffer[50:52])
		device.ProductCode = binary.LittleEndian.Uint16(buffer[52:54])
		device.MajorRevision = buffer[54]
		device.MinorRevision = buffer[55]
		device.SerialNumber = binary.LittleEndian.Uint32(buffer[57:61])

		// 产品名称长度
		if n >= 64 {
			nameLen := int(buffer[63])
			if n >= 64+nameLen {
				device.ProductName = string(buffer[64 : 64+nameLen])
			}
		}
	}

	return device, nil
}

// ReadTag 读取标签值 (CIP)
func (e *EtherNetIP) ReadTag(tagName string) (interface{}, error) {
	e.mu.RLock()
	if e.conn == nil {
		e.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	e.mu.RUnlock()

	// 构建CIP读取标签请求
	cipData := e.buildCIPReadTag(tagName)

	// 使用UnconnectedSend发送
	response, err := e.sendRRData(cipData)
	if err != nil {
		return nil, err
	}

	// 解析响应
	// 简化实现：返回原始数据
	return response, nil
}

// WriteTag 写入标签值 (CIP)
func (e *EtherNetIP) WriteTag(tagName string, dataType uint16, value interface{}) error {
	e.mu.RLock()
	if e.conn == nil {
		e.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	e.mu.RUnlock()

	// 构建CIP写入标签请求
	cipData := e.buildCIPWriteTag(tagName, dataType, value)

	// 使用UnconnectedSend发送
	_, err := e.sendRRData(cipData)
	return err
}

// sendRRData 发送请求/响应数据
func (e *EtherNetIP) sendRRData(cipData []byte) ([]byte, error) {
	startTime := time.Now()

	// 构建SendRRData请求
	request := e.buildEncapsulationHeader(EtherNetIPCmdSendRRData, e.sessionHandle, uint16(6+len(cipData)))

	// CPF (Common Packet Format)
	cpf := make([]byte, 6)
	binary.LittleEndian.PutUint32(cpf[0:4], 0x00000000) // Interface handle
	binary.LittleEndian.PutUint16(cpf[4:6], 0x0000)     // Timeout

	request = append(request, cpf...)
	request = append(request, cipData...)

	// 发送请求
	_, err := e.conn.Write(request)
	if err != nil {
		e.stats.ErrorCount++
		return nil, err
	}

	e.stats.BytesSent += uint64(len(request))
	e.stats.RequestCount++

	// 接收响应
	buffer := make([]byte, 1024)
	e.conn.SetReadDeadline(time.Now().Add(e.config.Timeout))
	n, err := e.conn.Read(buffer)
	if err != nil {
		return nil, err
	}

	e.stats.BytesReceived += uint64(n)
	e.stats.CurrentLatency = time.Since(startTime)

	// 解析响应 (简化)
	if n >= 24 {
		// 提取CIP响应数据
		return buffer[24:n], nil
	}

	return nil, fmt.Errorf("invalid response")
}

// buildEncapsulationHeader 构建封装头
func (e *EtherNetIP) buildEncapsulationHeader(command uint16, sessionHandle uint32, length uint16) []byte {
	header := make([]byte, 24)

	binary.LittleEndian.PutUint16(header[0:2], command)       // Command
	binary.LittleEndian.PutUint16(header[2:4], length)        // Length
	binary.LittleEndian.PutUint32(header[4:8], sessionHandle) // Session handle
	binary.LittleEndian.PutUint32(header[8:12], 0x00000000)   // Status
	// Sender context (8 bytes) - zeros
	binary.LittleEndian.PutUint32(header[20:24], 0x00000000) // Options

	return header
}

// buildCIPReadTag 构建CIP读取标签请求
func (e *EtherNetIP) buildCIPReadTag(tagName string) []byte {
	// CIP Unconnected Send
	cipData := make([]byte, 0, 100)

	// Item count
	cipData = append(cipData, 0x00, 0x02) // 2 items

	// Null address item
	cipData = append(cipData, 0x00, 0x00) // Type ID: Null
	cipData = append(cipData, 0x00, 0x00) // Length: 0

	// Unconnected data item
	cipData = append(cipData, 0xB2, 0x00) // Type ID: Unconnected message

	// CIP Request
	cipRequest := make([]byte, 0, 50)
	cipRequest = append(cipRequest, CIPServiceUnconnectedSend) // Service
	cipRequest = append(cipRequest, 0x02)                       // Request path size
	cipRequest = append(cipRequest, 0x20, 0x06)                 // Class: Connection Manager
	cipRequest = append(cipRequest, 0x24, 0x01)                 // Instance: 1

	// Read Tag Service
	readService := make([]byte, 0, 30)
	readService = append(readService, 0x4C) // Read Tag service

	// Tag name path
	tagPath := e.buildSymbolSegment(tagName)
	readService = append(readService, byte(len(tagPath)/2)) // Path size
	readService = append(readService, tagPath...)
	readService = append(readService, 0x01, 0x00) // Element count

	// Length
	dataLength := len(cipRequest) + len(readService)
	cipData = append(cipData, byte(dataLength), byte(dataLength>>8))

	cipData = append(cipData, cipRequest...)
	cipData = append(cipData, readService...)

	return cipData
}

// buildCIPWriteTag 构建CIP写入标签请求
func (e *EtherNetIP) buildCIPWriteTag(tagName string, dataType uint16, value interface{}) []byte {
	// 简化实现
	return []byte{}
}

// buildSymbolSegment 构建符号段
func (e *EtherNetIP) buildSymbolSegment(tagName string) []byte {
	segment := make([]byte, 0, len(tagName)+2)

	segment = append(segment, 0x91) // ANSI Extended Symbol Segment
	segment = append(segment, byte(len(tagName)))
	segment = append(segment, []byte(tagName)...)

	// 填充到偶数长度
	if len(tagName)%2 != 0 {
		segment = append(segment, 0x00)
	}

	return segment
}

// StartPolling 启动轮询
func (e *EtherNetIP) StartPolling(ctx context.Context, tags []string) error {
	e.mu.Lock()
	if e.monitoring {
		e.mu.Unlock()
		return fmt.Errorf("polling already started")
	}

	e.monitoring = true
	e.stats.Running = true
	e.mu.Unlock()

	go e.pollingLoop(ctx, tags)

	return nil
}

// StopPolling 停止轮询
func (e *EtherNetIP) StopPolling() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.monitoring {
		return nil
	}

	e.stopChan <- struct{}{}
	e.monitoring = false
	e.stats.Running = false

	return nil
}

// pollingLoop 轮询循环
func (e *EtherNetIP) pollingLoop(ctx context.Context, tags []string) {
	ticker := time.NewTicker(e.config.PollingInterval)
	defer ticker.Stop()

	tagValues := make(map[string]interface{})

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			// 轮询所有标签
			for _, tag := range tags {
				value, err := e.ReadTag(tag)
				if err != nil {
					continue
				}

				// 检测值变化
				oldValue := tagValues[tag]
				if oldValue != value {
					update := &interfaces.DataUpdate{
						TargetID:   fmt.Sprintf("EtherNetIP_Tag_%s", tag),
						Value:      value,
						OldValue:   oldValue,
						Quality:    1.0,
						Timestamp:  time.Now(),
						ChangeType: "value",
						Metadata: map[string]interface{}{
							"protocol": "EtherNet/IP",
							"tag_name": tag,
							"device_ip": e.config.TargetIP,
						},
					}

					select {
					case e.dataChan <- update:
					default:
					}

					tagValues[tag] = value
				}
			}
		}
	}
}

// GetDataChannel 获取数据通道
func (e *EtherNetIP) GetDataChannel() <-chan *interfaces.DataUpdate {
	return e.dataChan
}

// GetStatus 获取协议状态
func (e *EtherNetIP) GetStatus() *interfaces.ProtocolStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := *e.stats
	return &status
}

// GetAllDevices 获取所有设备
func (e *EtherNetIP) GetAllDevices() map[string]*EtherNetIPDevice {
	e.mu.RLock()
	defer e.mu.RUnlock()

	devices := make(map[string]*EtherNetIPDevice)
	for ip, device := range e.devices {
		deviceCopy := *device
		devices[ip] = &deviceCopy
	}

	return devices
}

// Read 实现io.Reader接口
func (e *EtherNetIP) Read(buffer []byte) (int, error) {
	if e.conn == nil {
		return 0, fmt.Errorf("not connected")
	}
	return e.conn.Read(buffer)
}

// Write 实现io.Writer接口
func (e *EtherNetIP) Write(data []byte) (int, error) {
	if e.conn == nil {
		return 0, fmt.Errorf("not connected")
	}
	return e.conn.Write(data)
}
