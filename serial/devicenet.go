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

// DeviceNet DeviceNet协议实现
// DeviceNet是基于CAN的现场总线协议，广泛应用于离散制造业
type DeviceNet struct {
	config     *DeviceNetConfig
	port       io.ReadWriteCloser
	mu         sync.RWMutex
	monitoring bool
	dataChan   chan *interfaces.DataUpdate
	stopChan   chan struct{}
	stats      *interfaces.ProtocolStatus
	nodes      map[byte]*DeviceNetNode // 节点列表
}

// DeviceNetConfig DeviceNet配置
type DeviceNetConfig struct {
	// CAN接口配置
	CANInterface string // CAN接口 (如 can0, vcan0)
	Bitrate      int    // 波特率 (125000, 250000, 500000)

	// DeviceNet配置
	MasterMAC  byte   // 主站MAC地址 (0-63)
	SlaveMACs  []byte // 从站MAC地址列表
	ScanTime   time.Duration // 扫描周期
	RetryCount int
	Timeout    time.Duration

	// 轮询配置
	PollingInterval time.Duration
	PollingEnabled  bool
}

// DeviceNetNode DeviceNet节点
type DeviceNetNode struct {
	MAC            byte   // MAC地址 (0-63)
	VendorID       uint16 // 供应商ID
	DeviceType     uint16 // 设备类型
	ProductCode    uint16 // 产品代码
	MajorRevision  byte   // 主版本号
	MinorRevision  byte   // 次版本号
	SerialNumber   uint32 // 序列号
	ProductName    string // 产品名称
	State          DeviceNetState
	InputData      []byte // 输入数据
	OutputData     []byte // 输出数据
	InputSize      byte   // 输入数据大小
	OutputSize     byte   // 输出数据大小
	ConfigSize     byte   // 配置数据大小
	Active         bool
}

// DeviceNetState DeviceNet节点状态
type DeviceNetState byte

const (
	DeviceNetStateNonexistent       DeviceNetState = 0x00 // 不存在
	DeviceNetStateDeviceStandby     DeviceNetState = 0x01 // 设备待机
	DeviceNetStateNoCommsEstablished DeviceNetState = 0x02 // 未建立通信
	DeviceNetStateConnectionEstablished DeviceNetState = 0x03 // 已建立连接
	DeviceNetStateConnectionTimedOut DeviceNetState = 0x04 // 连接超时
	DeviceNetStateDuplicateMAC      DeviceNetState = 0x05 // MAC地址冲突
)

// DeviceNet消息组ID (Group ID)
const (
	DeviceNetGroupExplicit    = 0 // 显式消息
	DeviceNetGroupIOPolled    = 1 // 轮询I/O
	DeviceNetGroupIOBitStrobe = 2 // 位选通I/O
	DeviceNetGroupIOCOS       = 3 // 状态变化I/O
	DeviceNetGroupIOCyclic    = 4 // 循环I/O
)

// DeviceNet服务代码
const (
	DeviceNetServiceGetAttributeSingle  = 0x0E // 获取单个属性
	DeviceNetServiceSetAttributeSingle  = 0x10 // 设置单个属性
	DeviceNetServiceAllocateMasterSlaveConnection = 0x4B // 分配主从连接
	DeviceNetServiceReleaseMasterSlaveConnection = 0x4C // 释放主从连接
)

// Validate 验证配置
func (c *DeviceNetConfig) Validate() error {
	if c.CANInterface == "" {
		return fmt.Errorf("can_interface is required")
	}

	// 验证波特率
	validBitrates := map[int]bool{
		125000: true,
		250000: true,
		500000: true,
	}
	if c.Bitrate == 0 {
		c.Bitrate = 250000 // 默认250kbps
	}
	if !validBitrates[c.Bitrate] {
		return fmt.Errorf("bitrate must be 125000, 250000, or 500000")
	}

	if c.MasterMAC > 63 {
		return fmt.Errorf("master MAC must be 0-63")
	}

	if c.ScanTime == 0 {
		c.ScanTime = 10 * time.Millisecond
	}
	if c.RetryCount == 0 {
		c.RetryCount = 3
	}
	if c.Timeout == 0 {
		c.Timeout = 100 * time.Millisecond
	}
	if c.PollingInterval == 0 {
		c.PollingInterval = 100 * time.Millisecond
	}

	return nil
}

// GetType 获取配置类型
func (c *DeviceNetConfig) GetType() interfaces.ProtocolType {
	return interfaces.ProtocolDeviceNet
}

// NewDeviceNet 创建DeviceNet实例
func NewDeviceNet(config *DeviceNetConfig) (*DeviceNet, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &DeviceNet{
		config:   config,
		dataChan: make(chan *interfaces.DataUpdate, 1000),
		stopChan: make(chan struct{}),
		nodes:    make(map[byte]*DeviceNetNode),
		stats: &interfaces.ProtocolStatus{
			Connected: false,
			Running:   false,
		},
	}, nil
}

// Open 打开DeviceNet连接
func (d *DeviceNet) Open(devicePath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 注意：实际实现需要使用SocketCAN或其他CAN库
	// 这里提供简化的接口示例
	// 实际使用时需要: import "github.com/brutella/can"

	// 简化实现：使用虚拟CAN接口
	port := &RealSerialPort{}
	err := port.Open(d.config.CANInterface)
	if err != nil {
		return fmt.Errorf("failed to open CAN interface: %w", err)
	}

	d.port = port
	d.stats.Connected = true

	// 扫描网络上的节点
	if len(d.config.SlaveMACs) > 0 {
		for _, mac := range d.config.SlaveMACs {
			node, err := d.identifyNode(mac)
			if err == nil {
				d.nodes[mac] = node
				node.Active = true
			}
		}
	}

	return nil
}

// Close 关闭DeviceNet连接
func (d *DeviceNet) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.monitoring {
		close(d.stopChan)
		d.monitoring = false
	}

	if d.port != nil {
		err := d.port.Close()
		d.port = nil
		d.stats.Connected = false
		return err
	}

	return nil
}

// identifyNode 识别节点
func (d *DeviceNet) identifyNode(mac byte) (*DeviceNetNode, error) {
	// 读取Identity Object (Class ID 0x01, Instance 1)
	response, err := d.getAttributeSingle(mac, 0x01, 1, 1) // Vendor ID
	if err != nil {
		return nil, err
	}

	node := &DeviceNetNode{
		MAC:    mac,
		State:  DeviceNetStateConnectionEstablished,
	}

	if len(response) >= 2 {
		node.VendorID = binary.LittleEndian.Uint16(response[0:2])
	}

	// 读取设备类型
	response, err = d.getAttributeSingle(mac, 0x01, 1, 2)
	if err == nil && len(response) >= 2 {
		node.DeviceType = binary.LittleEndian.Uint16(response[0:2])
	}

	// 读取产品代码
	response, err = d.getAttributeSingle(mac, 0x01, 1, 3)
	if err == nil && len(response) >= 2 {
		node.ProductCode = binary.LittleEndian.Uint16(response[0:2])
	}

	// 读取序列号
	response, err = d.getAttributeSingle(mac, 0x01, 1, 6)
	if err == nil && len(response) >= 4 {
		node.SerialNumber = binary.LittleEndian.Uint32(response[0:4])
	}

	return node, nil
}

// getAttributeSingle 获取单个属性 (显式消息)
func (d *DeviceNet) getAttributeSingle(mac byte, classID uint16, instance byte, attribute byte) ([]byte, error) {
	// 构建显式消息请求
	request := make([]byte, 8)
	request[0] = DeviceNetServiceGetAttributeSingle
	request[1] = byte(classID & 0xFF)
	request[2] = byte((classID >> 8) & 0xFF)
	request[3] = instance
	request[4] = attribute

	// 发送显式消息
	response, err := d.sendExplicitMessage(mac, request[:5])
	if err != nil {
		return nil, err
	}

	// 检查响应服务代码 (应该是请求服务代码 + 0x80)
	if len(response) > 0 && response[0] != (DeviceNetServiceGetAttributeSingle+0x80) {
		return nil, fmt.Errorf("invalid response service code: 0x%02X", response[0])
	}

	// 返回数据部分 (跳过服务代码)
	if len(response) > 1 {
		return response[1:], nil
	}

	return []byte{}, nil
}

// sendExplicitMessage 发送显式消息
func (d *DeviceNet) sendExplicitMessage(mac byte, data []byte) ([]byte, error) {
	startTime := time.Now()

	// 构建CAN消息
	// DeviceNet CAN ID格式: [Group ID (3位)] [MAC ID (6位)] [Message ID (2位)]
	// 显式消息使用 Group 0
	canID := uint32(DeviceNetGroupExplicit<<9) | uint32(mac<<3)

	frame := d.buildCANFrame(canID, data)

	var response []byte
	var err error

	// 重试机制
	for attempt := 0; attempt <= d.config.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(20 * time.Millisecond)
		}

		// 发送CAN帧
		_, err = d.port.Write(frame)
		if err != nil {
			d.stats.ErrorCount++
			continue
		}

		d.stats.BytesSent += uint64(len(frame))
		d.stats.RequestCount++

		// 接收响应
		response, err = d.receiveCANFrame(mac)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("explicit message failed: %w", err)
	}

	d.stats.CurrentLatency = time.Since(startTime)
	d.stats.BytesReceived += uint64(len(response))

	return response, nil
}

// ReadInputData 读取节点输入数据
func (d *DeviceNet) ReadInputData(mac byte) ([]byte, error) {
	d.mu.RLock()
	if d.port == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	d.mu.RUnlock()

	// 发送轮询I/O请求
	canID := uint32(DeviceNetGroupIOPolled<<9) | uint32(mac<<3)
	frame := d.buildCANFrame(canID, nil) // 空数据 = 请求

	_, err := d.port.Write(frame)
	if err != nil {
		return nil, err
	}

	// 接收响应
	response, err := d.receiveCANFrame(mac)
	if err != nil {
		return nil, err
	}

	// 更新节点数据
	if node, exists := d.nodes[mac]; exists {
		node.InputData = response
	}

	return response, nil
}

// WriteOutputData 写入节点输出数据
func (d *DeviceNet) WriteOutputData(mac byte, data []byte) error {
	d.mu.RLock()
	if d.port == nil {
		d.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	d.mu.RUnlock()

	// 发送轮询I/O命令
	canID := uint32(DeviceNetGroupIOPolled<<9) | uint32(mac<<3)
	frame := d.buildCANFrame(canID, data)

	_, err := d.port.Write(frame)
	if err != nil {
		return err
	}

	// 更新节点数据
	if node, exists := d.nodes[mac]; exists {
		node.OutputData = data
	}

	return nil
}

// buildCANFrame 构建CAN帧
func (d *DeviceNet) buildCANFrame(canID uint32, data []byte) []byte {
	// 简化的CAN帧格式 (实际应使用SocketCAN格式)
	// [CAN ID (4字节)] [DLC (1字节)] [Data (0-8字节)]
	frame := make([]byte, 5+len(data))

	binary.LittleEndian.PutUint32(frame[0:4], canID)
	frame[4] = byte(len(data)) // DLC

	if len(data) > 0 {
		copy(frame[5:], data)
	}

	return frame
}

// receiveCANFrame 接收CAN帧
func (d *DeviceNet) receiveCANFrame(expectedMAC byte) ([]byte, error) {
	buffer := make([]byte, 128)
	deadline := time.Now().Add(d.config.Timeout)

	for time.Now().Before(deadline) {
		n, err := d.port.Read(buffer)
		if err != nil {
			return nil, err
		}

		if n >= 5 {
			// 解析CAN ID
			canID := binary.LittleEndian.Uint32(buffer[0:4])
			mac := byte((canID >> 3) & 0x3F)

			// 检查是否是期望的MAC地址
			if mac == expectedMAC {
				dlc := buffer[4]
				if n >= int(5+dlc) {
					return buffer[5 : 5+dlc], nil
				}
			}
		}

		time.Sleep(5 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for response from MAC %d", expectedMAC)
}

// StartIOPolling 启动I/O轮询
func (d *DeviceNet) StartIOPolling(ctx context.Context) error {
	d.mu.Lock()
	if d.monitoring {
		d.mu.Unlock()
		return fmt.Errorf("polling already started")
	}

	d.monitoring = true
	d.stats.Running = true
	d.mu.Unlock()

	go d.pollingLoop(ctx)

	return nil
}

// StopPolling 停止轮询
func (d *DeviceNet) StopPolling() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.monitoring {
		return nil
	}

	d.stopChan <- struct{}{}
	d.monitoring = false
	d.stats.Running = false

	return nil
}

// pollingLoop 轮询循环
func (d *DeviceNet) pollingLoop(ctx context.Context) {
	ticker := time.NewTicker(d.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopChan:
			return
		case <-ticker.C:
			// 轮询所有节点
			for mac, node := range d.nodes {
				if !node.Active {
					continue
				}

				// 读取输入数据
				data, err := d.ReadInputData(mac)
				if err != nil {
					continue
				}

				// 检测数据变化
				if !bytesEqual(node.InputData, data) {
					update := &interfaces.DataUpdate{
						TargetID:   fmt.Sprintf("DeviceNet_Node_%d", mac),
						Value:      data,
						OldValue:   node.InputData,
						Quality:    1.0,
						Timestamp:  time.Now(),
						ChangeType: "value",
						Metadata: map[string]interface{}{
							"protocol":     "DeviceNet",
							"mac":          mac,
							"vendor_id":    node.VendorID,
							"device_type":  node.DeviceType,
							"product_name": node.ProductName,
						},
					}

					select {
					case d.dataChan <- update:
					default:
						// 缓冲区满，丢弃
					}
				}
			}
		}
	}
}

// bytesEqual 比较字节数组
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GetDataChannel 获取数据通道
func (d *DeviceNet) GetDataChannel() <-chan *interfaces.DataUpdate {
	return d.dataChan
}

// GetStatus 获取协议状态
func (d *DeviceNet) GetStatus() *interfaces.ProtocolStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status := *d.stats
	return &status
}

// GetAllNodes 获取所有节点
func (d *DeviceNet) GetAllNodes() map[byte]*DeviceNetNode {
	d.mu.RLock()
	defer d.mu.RUnlock()

	nodes := make(map[byte]*DeviceNetNode)
	for mac, node := range d.nodes {
		nodeCopy := *node
		nodes[mac] = &nodeCopy
	}

	return nodes
}

// Read 实现io.Reader接口
func (d *DeviceNet) Read(buffer []byte) (int, error) {
	if d.port == nil {
		return 0, fmt.Errorf("not connected")
	}
	return d.port.Read(buffer)
}

// Write 实现io.Writer接口
func (d *DeviceNet) Write(data []byte) (int, error) {
	if d.port == nil {
		return 0, fmt.Errorf("not connected")
	}
	return d.port.Write(data)
}
