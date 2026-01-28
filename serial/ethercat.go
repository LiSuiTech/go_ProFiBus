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

// EtherCAT EtherCAT协议实现
// EtherCAT (Ethernet for Control Automation Technology) 是高性能实时以太网协议
type EtherCAT struct {
	config     *EtherCATConfig
	conn       net.PacketConn
	mu         sync.RWMutex
	monitoring bool
	dataChan   chan *interfaces.DataUpdate
	stopChan   chan struct{}
	stats      *interfaces.ProtocolStatus
	slaves     map[uint16]*EtherCATSlave // 从站列表 (地址 -> 从站)
	topology   *EtherCATTopology          // 网络拓扑
}

// EtherCATConfig EtherCAT配置
type EtherCATConfig struct {
	// 网络配置
	InterfaceName string // 网络接口名 (如 eth0)
	MACAddress    string // 本地MAC地址

	// EtherCAT配置
	MasterMode    bool          // 主站模式
	CycleTime     time.Duration // 循环周期 (通常50μs-10ms)
	SyncMode      EtherCATSyncMode // 同步模式
	DCEnabled     bool          // 启用分布式时钟(DC)
	DCCycleTime   time.Duration // DC循环时间

	// 从站配置
	ExpectedSlaves int    // 期望的从站数量
	SlaveAddresses []uint16 // 从站地址列表

	// 通信配置
	Timeout        time.Duration // 超时时间
	RetryCount     int           // 重试次数
	MaxFrameSize   int           // 最大帧大小 (通常1500字节)

	// 轮询配置
	PollingInterval time.Duration
	PollingEnabled  bool
}

// EtherCATSyncMode EtherCAT同步模式
type EtherCATSyncMode string

const (
	EtherCATSyncFreeRun    EtherCATSyncMode = "freerun"    // 自由运行
	EtherCATSyncSM         EtherCATSyncMode = "sm"         // 同步管理器
	EtherCATSyncDC         EtherCATSyncMode = "dc"         // 分布式时钟
)

// EtherCATSlave EtherCAT从站
type EtherCAT Slave struct {
	Address         uint16 // 从站地址 (0-65535)
	Position        int    // 拓扑位置
	AliasAddress    uint16 // 别名地址
	VendorID        uint32 // 供应商ID
	ProductCode     uint32 // 产品代码
	RevisionNumber  uint32 // 版本号
	SerialNumber    uint32 // 序列号
	Name            string // 从站名称

	// 状态机
	State           EtherCATState // 当前状态
	RequestedState  EtherCATState // 请求状态
	ErrorCode       uint16        // 错误代码

	// 邮箱通信
	MBoxOffset      uint16 // 邮箱偏移
	MBoxSize        uint16 // 邮箱大小

	// 过程数据
	InputOffset     uint32 // 输入数据偏移
	InputSize       uint32 // 输入数据大小
	OutputOffset    uint32 // 输出数据偏移
	OutputSize      uint32 // 输出数据大小
	InputData       []byte // 输入数据
	OutputData      []byte // 输出数据

	// FMMU配置 (Fieldbus Memory Management Unit)
	FMMUs           []*EtherCATFMMU

	// 同步管理器
	SyncManagers    []*EtherCATSyncManager

	// 分布式时钟
	DCSupport       bool      // 支持DC
	DCReceiveTime   uint64    // DC接收时间
	DCSystemTime    uint64    // DC系统时间

	Active          bool      // 是否在线
	LastUpdateTime  time.Time // 最后更新时间
}

// EtherCATState EtherCAT状态
type EtherCATState byte

const (
	EtherCATStateInit       EtherCATState = 0x01 // 初始化
	EtherCATStatePreOp      EtherCATState = 0x02 // 预操作
	EtherCATStateSafeOp     EtherCATState = 0x04 // 安全操作
	EtherCATStateOp         EtherCATState = 0x08 // 操作
)

// EtherCATFMMU FMMU配置
type EtherCATFMMU struct {
	LogicalStartAddr uint32 // 逻辑起始地址
	Length           uint16 // 长度
	LogicalStartBit  byte   // 逻辑起始位
	LogicalEndBit    byte   // 逻辑结束位
	PhysicalStartAddr uint16 // 物理起始地址
	PhysicalStartBit byte   // 物理起始位
	Type             byte   // 类型 (读/写)
	Activate         bool   // 激活
}

// EtherCATSyncManager 同步管理器
type EtherCATSyncManager struct {
	PhysicalStartAddr uint16 // 物理起始地址
	Length            uint16 // 长度
	ControlRegister   byte   // 控制寄存器
	Enable            bool   // 启用
	Type              EtherCATSMType // 类型
}

// EtherCATSMType 同步管理器类型
type EtherCATSMType byte

const (
	EtherCATSMTypeUnused   EtherCATSMType = 0x00 // 未使用
	EtherCATSMTypeMBoxOut  EtherCATSMType = 0x01 // 邮箱输出
	EtherCATSMTypeMBoxIn   EtherCATSMType = 0x02 // 邮箱输入
	EtherCATSMTypeOutput   EtherCATSMType = 0x03 // 过程数据输出
	EtherCATSMTypeInput    EtherCATSMType = 0x04 // 过程数据输入
)

// EtherCATTopology EtherCAT网络拓扑
type EtherCATTopology struct {
	SlaveCount int                    // 从站数量
	Slaves     []*EtherCATSlave       // 从站列表 (按拓扑顺序)
	TotalDelay time.Duration          // 总延迟
}

// EtherCAT命令类型
const (
	EtherCATCmdNOP  = 0x00 // 无操作
	EtherCATCmdAPRD = 0x01 // 自动增量物理读
	EtherCATCmdAPWR = 0x02 // 自动增量物理写
	EtherCATCmdAPRW = 0x03 // 自动增量物理读写
	EtherCATCmdFPRD = 0x04 // 配置物理读
	EtherCATCmdFPWR = 0x05 // 配置物理写
	EtherCATCmdFPRW = 0x06 // 配置物理读写
	EtherCATCmdBRD  = 0x07 // 广播读
	EtherCATCmdBWR  = 0x08 // 广播写
	EtherCATCmdBRW  = 0x09 // 广播读写
	EtherCATCmdLRD  = 0x0A // 逻辑读
	EtherCATCmdLWR  = 0x0B // 逻辑写
	EtherCATCmdLRW  = 0x0C // 逻辑读写
)

// EtherCAT以太网类型
const (
	EtherCATEtherType = 0x88A4 // EtherCAT以太网类型
)

// Validate 验证配置
func (c *EtherCATConfig) Validate() error {
	if c.InterfaceName == "" {
		return fmt.Errorf("interface_name is required")
	}

	if c.CycleTime == 0 {
		c.CycleTime = 1 * time.Millisecond // 默认1ms
	}

	if c.SyncMode == "" {
		c.SyncMode = EtherCATSyncFreeRun
	}

	if c.DCCycleTime == 0 {
		c.DCCycleTime = c.CycleTime
	}

	if c.Timeout == 0 {
		c.Timeout = 100 * time.Millisecond
	}

	if c.RetryCount == 0 {
		c.RetryCount = 3
	}

	if c.MaxFrameSize == 0 {
		c.MaxFrameSize = 1500
	}

	if c.PollingInterval == 0 {
		c.PollingInterval = 10 * time.Millisecond
	}

	return nil
}

// GetType 获取配置类型
func (c *EtherCATConfig) GetType() interfaces.ProtocolType {
	return interfaces.ProtocolEtherCAT
}

// NewEtherCAT 创建EtherCAT实例
func NewEtherCAT(config *EtherCATConfig) (*EtherCAT, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &EtherCAT{
		config:   config,
		dataChan: make(chan *interfaces.DataUpdate, 1000),
		stopChan: make(chan struct{}),
		slaves:   make(map[uint16]*EtherCATSlave),
		topology: &EtherCATTopology{},
		stats: &interfaces.ProtocolStatus{
			Connected: false,
			Running:   false,
		},
	}, nil
}

// Open 打开EtherCAT连接
func (e *EtherCAT) Open(devicePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 打开原始套接字 (需要root权限)
	// 实际实现需要使用: import "golang.org/x/net/bpf"
	// 这里提供简化的接口示例

	conn, err := net.ListenPacket("ip4:ethercat", "0.0.0.0")
	if err != nil {
		return fmt.Errorf("failed to open raw socket: %w", err)
	}

	e.conn = conn
	e.stats.Connected = true

	// 扫描从站
	if err := e.scanSlaves(); err != nil {
		return fmt.Errorf("failed to scan slaves: %w", err)
	}

	// 配置从站
	if err := e.configureSlaves(); err != nil {
		return fmt.Errorf("failed to configure slaves: %w", err)
	}

	return nil
}

// Close 关闭EtherCAT连接
func (e *EtherCAT) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.monitoring {
		close(e.stopChan)
		e.monitoring = false
	}

	// 将所有从站切换到Init状态
	for _, slave := range e.slaves {
		e.setSlaveState(slave.Address, EtherCATStateInit)
	}

	if e.conn != nil {
		err := e.conn.Close()
		e.conn = nil
		e.stats.Connected = false
		return err
	}

	return nil
}

// scanSlaves 扫描从站
func (e *EtherCAT) scanSlaves() error {
	// 使用BRD (广播读) 命令扫描从站

	slaveCount := 0

	// 读取所有从站的类型寄存器 (0x0000)
	for position := 0; position < 256; position++ {
		data, err := e.readAPRD(uint16(position), 0x0000, 2)
		if err != nil {
			break // 没有更多从站
		}

		if len(data) >= 2 {
			slaveType := binary.LittleEndian.Uint16(data)
			if slaveType == 0 {
				break
			}

			// 创建从站
			slave := &EtherCATSlave{
				Address:  uint16(position),
				Position: position,
				State:    EtherCATStateInit,
			}

			// 读取从站信息
			e.readSlaveInfo(slave)

			e.slaves[uint16(position)] = slave
			e.topology.Slaves = append(e.topology.Slaves, slave)
			slaveCount++
		}
	}

	e.topology.SlaveCount = slaveCount

	return nil
}

// readSlaveInfo 读取从站信息
func (e *EtherCAT) readSlaveInfo(slave *EtherCATSlave) error {
	// 读取供应商ID (0x0008-0x000B)
	data, err := e.readAPRD(slave.Address, 0x0008, 4)
	if err == nil && len(data) >= 4 {
		slave.VendorID = binary.LittleEndian.Uint32(data)
	}

	// 读取产品代码 (0x000C-0x000F)
	data, err = e.readAPRD(slave.Address, 0x000C, 4)
	if err == nil && len(data) >= 4 {
		slave.ProductCode = binary.LittleEndian.Uint32(data)
	}

	// 读取版本号 (0x0010-0x0013)
	data, err = e.readAPRD(slave.Address, 0x0010, 4)
	if err == nil && len(data) >= 4 {
		slave.RevisionNumber = binary.LittleEndian.Uint32(data)
	}

	// 读取序列号 (0x0014-0x0017)
	data, err = e.readAPRD(slave.Address, 0x0014, 4)
	if err == nil && len(data) >= 4 {
		slave.SerialNumber = binary.LittleEndian.Uint32(data)
	}

	slave.Active = true
	return nil
}

// configureSlaves 配置从站
func (e *EtherCAT) configureSlaves() error {
	// 状态转换序列: Init -> PreOp -> SafeOp -> Op

	for _, slave := range e.slaves {
		// Init -> PreOp
		if err := e.setSlaveState(slave.Address, EtherCATStatePreOp); err != nil {
			return fmt.Errorf("failed to set slave %d to PreOp: %w", slave.Address, err)
		}

		// 配置FMMU和同步管理器
		// 简化实现

		// PreOp -> SafeOp
		if err := e.setSlaveState(slave.Address, EtherCATStateSafeOp); err != nil {
			return fmt.Errorf("failed to set slave %d to SafeOp: %w", slave.Address, err)
		}

		// SafeOp -> Op
		if err := e.setSlaveState(slave.Address, EtherCATStateOp); err != nil {
			return fmt.Errorf("failed to set slave %d to Op: %w", slave.Address, err)
		}
	}

	return nil
}

// setSlaveState 设置从站状态
func (e *EtherCAT) setSlaveState(address uint16, state EtherCATState) error {
	// 写入AL Control寄存器 (0x0120)
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, uint16(state))

	err := e.writeAPWR(address, 0x0120, data)
	if err != nil {
		return err
	}

	// 等待状态转换完成
	timeout := time.After(1 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for state change")
		case <-ticker.C:
			// 读取AL Status寄存器 (0x0130)
			statusData, err := e.readAPRD(address, 0x0130, 2)
			if err != nil {
				continue
			}

			if len(statusData) >= 2 {
				currentState := EtherCATState(statusData[0] & 0x0F)
				if currentState == state {
					if slave, exists := e.slaves[address]; exists {
						slave.State = state
					}
					return nil
				}
			}
		}
	}
}

// readAPRD 自动增量物理读
func (e *EtherCAT) readAPRD(address uint16, offset uint16, length uint16) ([]byte, error) {
	return e.sendCommand(EtherCATCmdAPRD, address, offset, length, nil)
}

// writeAPWR 自动增量物理写
func (e *EtherCAT) writeAPWR(address uint16, offset uint16, data []byte) error {
	_, err := e.sendCommand(EtherCATCmdAPWR, address, offset, uint16(len(data)), data)
	return err
}

// readLRD 逻辑读
func (e *EtherCAT) readLRD(logicalAddr uint32, length uint16) ([]byte, error) {
	return e.sendCommand(EtherCATCmdLRD, 0, uint16(logicalAddr), length, nil)
}

// writeLWR 逻辑写
func (e *EtherCAT) writeLWR(logicalAddr uint32, data []byte) error {
	_, err := e.sendCommand(EtherCATCmdLWR, 0, uint16(logicalAddr), uint16(len(data)), data)
	return err
}

// sendCommand 发送EtherCAT命令
func (e *EtherCAT) sendCommand(cmd byte, address uint16, offset uint16, length uint16, data []byte) ([]byte, error) {
	startTime := time.Now()

	// 构建EtherCAT帧
	frame := e.buildFrame(cmd, address, offset, length, data)

	// 发送帧
	_, err := e.conn.WriteTo(frame, nil)
	if err != nil {
		e.stats.ErrorCount++
		return nil, err
	}

	e.stats.BytesSent += uint64(len(frame))
	e.stats.RequestCount++

	// 接收响应
	buffer := make([]byte, e.config.MaxFrameSize)
	n, _, err := e.conn.ReadFrom(buffer)
	if err != nil {
		return nil, err
	}

	e.stats.BytesReceived += uint64(n)
	e.stats.CurrentLatency = time.Since(startTime)

	// 解析响应
	if n >= 44 {
		// 提取数据部分
		dataStart := 44
		dataEnd := dataStart + int(length)
		if dataEnd <= n {
			return buffer[dataStart:dataEnd], nil
		}
	}

	return nil, fmt.Errorf("invalid response")
}

// buildFrame 构建EtherCAT帧
func (e *EtherCAT) buildFrame(cmd byte, address uint16, offset uint16, length uint16, data []byte) []byte {
	// EtherCAT帧格式:
	// [Ethernet Header (14)] [EtherCAT Header (2)] [Datagram(s)] [Working Counter (2)]

	frameSize := 14 + 2 + 10 + int(length) + 2
	frame := make([]byte, frameSize)

	// Ethernet Header
	// Destination MAC (广播)
	for i := 0; i < 6; i++ {
		frame[i] = 0xFF
	}
	// Source MAC (简化)
	for i := 6; i < 12; i++ {
		frame[i] = 0x00
	}
	// EtherType (EtherCAT)
	binary.BigEndian.PutUint16(frame[12:14], EtherCATEtherType)

	// EtherCAT Header
	// Length and Type
	binary.LittleEndian.PutUint16(frame[14:16], length&0x07FF)

	// Datagram Header
	pos := 16
	frame[pos] = cmd                                        // Command
	frame[pos+1] = 0                                        // Index
	binary.LittleEndian.PutUint16(frame[pos+2:pos+4], address) // Address (ADP)
	binary.LittleEndian.PutUint16(frame[pos+4:pos+6], offset)  // Address (ADO)
	binary.LittleEndian.PutUint16(frame[pos+6:pos+8], length)  // Length
	binary.LittleEndian.PutUint16(frame[pos+8:pos+10], 0)      // Interrupt

	// Data
	if len(data) > 0 {
		copy(frame[pos+10:], data)
	}

	// Working Counter
	binary.LittleEndian.PutUint16(frame[frameSize-2:], 0)

	return frame
}

// StartCyclicIO 启动循环I/O
func (e *EtherCAT) StartCyclicIO(ctx context.Context) error {
	e.mu.Lock()
	if e.monitoring {
		e.mu.Unlock()
		return fmt.Errorf("cyclic IO already started")
	}

	e.monitoring = true
	e.stats.Running = true
	e.mu.Unlock()

	go e.cyclicLoop(ctx)

	return nil
}

// StopCyclicIO 停止循环I/O
func (e *EtherCAT) StopCyclicIO() error {
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

// cyclicLoop 循环I/O
func (e *EtherCAT) cyclicLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.CycleTime)
	defer ticker.Stop()

	logicalAddr := uint32(0x1000) // 逻辑地址起点

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			// 使用LRW (逻辑读写) 一次性交换所有数据
			totalSize := uint16(0)
			for _, slave := range e.topology.Slaves {
				totalSize += uint16(slave.InputSize + slave.OutputSize)
			}

			if totalSize > 0 {
				// 读取输入数据
				data, err := e.readLRD(logicalAddr, totalSize)
				if err != nil {
					continue
				}

				// 更新从站数据
				offset := 0
				for _, slave := range e.topology.Slaves {
					if !slave.Active {
						continue
					}

					inputSize := int(slave.InputSize)
					if offset+inputSize <= len(data) {
						newData := data[offset : offset+inputSize]

						// 检测数据变化
						if !bytesEqual(slave.InputData, newData) {
							update := &interfaces.DataUpdate{
								TargetID:   fmt.Sprintf("EtherCAT_Slave_%d", slave.Address),
								Value:      newData,
								OldValue:   slave.InputData,
								Quality:    1.0,
								Timestamp:  time.Now(),
								ChangeType: "value",
								Metadata: map[string]interface{}{
									"protocol":     "EtherCAT",
									"slave_addr":   slave.Address,
									"position":     slave.Position,
									"vendor_id":    slave.VendorID,
									"product_code": slave.ProductCode,
								},
							}

							select {
							case e.dataChan <- update:
							default:
							}

							slave.InputData = newData
						}

						offset += inputSize
					}
				}
			}
		}
	}
}

// GetDataChannel 获取数据通道
func (e *EtherCAT) GetDataChannel() <-chan *interfaces.DataUpdate {
	return e.dataChan
}

// GetStatus 获取协议状态
func (e *EtherCAT) GetStatus() *interfaces.ProtocolStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := *e.stats
	return &status
}

// GetAllSlaves 获取所有从站
func (e *EtherCAT) GetAllSlaves() map[uint16]*EtherCATSlave {
	e.mu.RLock()
	defer e.mu.RUnlock()

	slaves := make(map[uint16]*EtherCATSlave)
	for addr, slave := range e.slaves {
		slaveCopy := *slave
		slaves[addr] = &slaveCopy
	}

	return slaves
}

// Read 实现io.Reader接口
func (e *EtherCAT) Read(buffer []byte) (int, error) {
	if e.conn == nil {
		return 0, fmt.Errorf("not connected")
	}
	n, _, err := e.conn.ReadFrom(buffer)
	return n, err
}

// Write 实现io.Writer接口
func (e *EtherCAT) Write(data []byte) (int, error) {
	if e.conn == nil {
		return 0, fmt.Errorf("not connected")
	}
	return e.conn.WriteTo(data, nil)
}
