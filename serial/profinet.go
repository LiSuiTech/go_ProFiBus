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

// PROFINET PROFINET IO协议实现
// PROFINET是PROFIBUS的以太网版本，使用标准以太网技术
type PROFINET struct {
	config     *PROFINETConfig
	conn       net.Conn
	mu         sync.RWMutex
	monitoring bool
	dataChan   chan *interfaces.DataUpdate
	stopChan   chan struct{}
	stats      *interfaces.ProtocolStatus
	devices    map[string]*PROFINETDevice // 设备列表 (IP -> 设备)
}

// PROFINETConfig PROFINET配置
type PROFINETConfig struct {
	// 网络配置
	LocalIP     string // 本地IP地址
	LocalPort   int    // 本地端口
	ControllerIP string // IO控制器IP
	ControllerPort int  // IO控制器端口

	// PROFINET配置
	Mode            PROFINETMode  // PROFINET模式
	DeviceIPs       []string      // IO设备IP列表
	DeviceName      string        // 设备名称
	CycleTime       time.Duration // 循环周期 (1ms-512ms)
	ReductionRatio  int           // 简化比率 (1-512)
	WatchdogTime    time.Duration // 看门狗时间

	// 通信配置
	UseRTC          bool          // 使用实时通信(RTC)
	UseIRT          bool          // 使用等时实时通信(IRT)
	Timeout         time.Duration // 超时时间
	RetryCount      int           // 重试次数

	// 轮询配置
	PollingInterval time.Duration // 轮询间隔
	PollingEnabled  bool          // 是否启用轮询
}

// PROFINETMode PROFINET模式
type PROFINETMode string

const (
	PROFINETModeController PROFINETMode = "controller" // IO控制器
	PROFINETModeDevice     PROFINETMode = "device"     // IO设备
	PROFINETModeSupervisor PROFINETMode = "supervisor" // 监视器
)

// PROFINETDevice PROFINET设备
type PROFINETDevice struct {
	IP              string // IP地址
	MAC             string // MAC地址
	DeviceName      string // 设备名称
	VendorID        uint16 // 供应商ID
	DeviceID        uint16 // 设备ID
	StationName     string // 站名称
	StationType     string // 站类型

	// 模块信息
	Modules         []*PROFINETModule // 模块列表

	// I/O数据
	InputData       []byte    // 输入数据
	OutputData      []byte    // 输出数据
	InputSize       int       // 输入数据大小
	OutputSize      int       // 输出数据大小

	// 诊断信息
	State           PROFINETDeviceState // 设备状态
	AlarmCount      int                 // 报警计数
	LastUpdateTime  time.Time           // 最后更新时间
	Active          bool                // 是否在线
}

// PROFINETModule PROFINET模块
type PROFINETModule struct {
	SlotNumber      int    // 槽位号
	SubslotNumber   int    // 子槽位号
	ModuleIdentNumber uint32 // 模块标识号
	SubmoduleIdentNumber uint32 // 子模块标识号
	InputSize       int    // 输入数据大小
	OutputSize      int    // 输出数据大小
}

// PROFINETDeviceState PROFINET设备状态
type PROFINETDeviceState byte

const (
	PROFINETStateOffline    PROFINETDeviceState = 0x00 // 离线
	PROFINETStateStartup    PROFINETDeviceState = 0x01 // 启动中
	PROFINETStateRun        PROFINETDeviceState = 0x02 // 运行
	PROFINETStateStop       PROFINETDeviceState = 0x03 // 停止
	PROFINETStateFault      PROFINETDeviceState = 0x04 // 故障
	PROFINETStateMaintenance PROFINETDeviceState = 0x05 // 维护
)

// PROFINET DCP协议类型 (Discovery and Configuration Protocol)
const (
	PROFINETDCPServiceIDGet     = 0x03 // Get请求
	PROFINETDCPServiceIDSet     = 0x04 // Set请求
	PROFINETDCPServiceIDIdentify = 0x05 // Identify请求
)

// PROFINET以太网类型
const (
	PROFINETEtherTypeDCP    = 0x8892 // DCP协议
	PROFINETEtherTypeRTC    = 0x8892 // 实时通信
	PROFINETEtherTypeAlarm  = 0x8100 // 报警帧
)

// Validate 验证配置
func (c *PROFINETConfig) Validate() error {
	if c.LocalIP == "" {
		return fmt.Errorf("local_ip is required")
	}

	if c.LocalPort == 0 {
		c.LocalPort = 34964 // PROFINET标准端口
	}

	if c.ControllerIP == "" && c.Mode == PROFINETModeDevice {
		return fmt.Errorf("controller_ip is required for device mode")
	}

	if c.ControllerPort == 0 {
		c.ControllerPort = 34964
	}

	if c.Mode == "" {
		c.Mode = PROFINETModeController
	}

	if c.CycleTime == 0 {
		c.CycleTime = 10 * time.Millisecond // 默认10ms
	}

	if c.ReductionRatio == 0 {
		c.ReductionRatio = 1
	}

	if c.WatchdogTime == 0 {
		c.WatchdogTime = 3 * c.CycleTime
	}

	if c.Timeout == 0 {
		c.Timeout = 1 * time.Second
	}

	if c.RetryCount == 0 {
		c.RetryCount = 3
	}

	if c.PollingInterval == 0 {
		c.PollingInterval = 100 * time.Millisecond
	}

	return nil
}

// GetType 获取配置类型
func (c *PROFINETConfig) GetType() interfaces.ProtocolType {
	return interfaces.ProtocolPROFINET
}

// NewPROFINET 创建PROFINET实例
func NewPROFINET(config *PROFINETConfig) (*PROFINET, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &PROFINET{
		config:   config,
		dataChan: make(chan *interfaces.DataUpdate, 1000),
		stopChan: make(chan struct{}),
		devices:  make(map[string]*PROFINETDevice),
		stats: &interfaces.ProtocolStatus{
			Connected: false,
			Running:   false,
		},
	}, nil
}

// Open 打开PROFINET连接
func (p *PROFINET) Open(devicePath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 根据模式选择连接方式
	var conn net.Conn
	var err error

	if p.config.Mode == PROFINETModeController {
		// 控制器模式：监听端口
		addr := fmt.Sprintf("%s:%d", p.config.LocalIP, p.config.LocalPort)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}

		// 保存listener用于后续接受连接
		go p.acceptConnections(listener)

	} else if p.config.Mode == PROFINETModeDevice {
		// 设备模式：连接到控制器
		addr := fmt.Sprintf("%s:%d", p.config.ControllerIP, p.config.ControllerPort)
		conn, err = net.DialTimeout("tcp", addr, p.config.Timeout)
		if err != nil {
			return fmt.Errorf("failed to connect to controller: %w", err)
		}
		p.conn = conn
	}

	p.stats.Connected = true

	// 扫描网络上的设备 (仅控制器模式)
	if p.config.Mode == PROFINETModeController && len(p.config.DeviceIPs) > 0 {
		for _, ip := range p.config.DeviceIPs {
			device, err := p.identifyDevice(ip)
			if err == nil {
				p.devices[ip] = device
				device.Active = true
			}
		}
	}

	return nil
}

// acceptConnections 接受连接 (控制器模式)
func (p *PROFINET) acceptConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		// 处理新连接
		go p.handleConnection(conn)
	}
}

// handleConnection 处理连接
func (p *PROFINET) handleConnection(conn net.Conn) {
	defer conn.Close()

	// 读取设备信息
	// 简化实现
}

// Close 关闭PROFINET连接
func (p *PROFINET) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.monitoring {
		close(p.stopChan)
		p.monitoring = false
	}

	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		p.stats.Connected = false
		return err
	}

	return nil
}

// identifyDevice 识别设备 (使用DCP协议)
func (p *PROFINET) identifyDevice(ip string) (*PROFINETDevice, error) {
	// 发送DCP Identify请求
	// 简化实现：使用TCP连接读取设备信息

	addr := fmt.Sprintf("%s:%d", ip, 34964)
	conn, err := net.DialTimeout("tcp", addr, p.config.Timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	device := &PROFINETDevice{
		IP:    ip,
		State: PROFINETStateRun,
	}

	// 读取设备标识（简化）
	buffer := make([]byte, 256)
	n, err := conn.Read(buffer)
	if err == nil && n > 0 {
		device.DeviceName = string(buffer[:n])
	}

	return device, nil
}

// ReadInputData 读取设备输入数据
func (p *PROFINET) ReadInputData(deviceIP string) ([]byte, error) {
	p.mu.RLock()
	device, exists := p.devices[deviceIP]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceIP)
	}

	if !device.Active {
		return nil, fmt.Errorf("device offline: %s", deviceIP)
	}

	// 发送读取请求
	// 简化实现
	return device.InputData, nil
}

// WriteOutputData 写入设备输出数据
func (p *PROFINET) WriteOutputData(deviceIP string, data []byte) error {
	p.mu.RLock()
	device, exists := p.devices[deviceIP]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("device not found: %s", deviceIP)
	}

	if !device.Active {
		return fmt.Errorf("device offline: %s", deviceIP)
	}

	// 发送写入请求
	// 简化实现
	device.OutputData = data
	return nil
}

// sendDCPRequest 发送DCP请求
func (p *PROFINET) sendDCPRequest(deviceIP string, serviceID byte, data []byte) ([]byte, error) {
	// 构建DCP请求帧
	frame := p.buildDCPFrame(serviceID, data)

	// 发送请求
	addr := fmt.Sprintf("%s:%d", deviceIP, 34964)
	conn, err := net.DialTimeout("tcp", addr, p.config.Timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write(frame)
	if err != nil {
		return nil, err
	}

	p.stats.BytesSent += uint64(len(frame))
	p.stats.RequestCount++

	// 接收响应
	buffer := make([]byte, 1500)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}

	p.stats.BytesReceived += uint64(n)

	return buffer[:n], nil
}

// buildDCPFrame 构建DCP帧
func (p *PROFINET) buildDCPFrame(serviceID byte, data []byte) []byte {
	// DCP帧格式 (简化)
	// [Frame ID (2)] [Service ID (1)] [Service Type (1)] [XID (4)] [Length (2)] [Data]

	frame := make([]byte, 10+len(data))

	// Frame ID (PROFINET DCP)
	binary.BigEndian.PutUint16(frame[0:2], 0xFEFE)

	// Service ID
	frame[2] = serviceID

	// Service Type
	frame[3] = 0x00 // Request

	// Transaction ID (XID)
	binary.BigEndian.PutUint32(frame[4:8], 0x12345678)

	// Length
	binary.BigEndian.PutUint16(frame[8:10], uint16(len(data)))

	// Data
	if len(data) > 0 {
		copy(frame[10:], data)
	}

	return frame
}

// StartCyclicIO 启动循环I/O
func (p *PROFINET) StartCyclicIO(ctx context.Context) error {
	p.mu.Lock()
	if p.monitoring {
		p.mu.Unlock()
		return fmt.Errorf("cyclic IO already started")
	}

	p.monitoring = true
	p.stats.Running = true
	p.mu.Unlock()

	go p.cyclicLoop(ctx)

	return nil
}

// StopCyclicIO 停止循环I/O
func (p *PROFINET) StopCyclicIO() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.monitoring {
		return nil
	}

	p.stopChan <- struct{}{}
	p.monitoring = false
	p.stats.Running = false

	return nil
}

// cyclicLoop 循环I/O
func (p *PROFINET) cyclicLoop(ctx context.Context) {
	ticker := time.NewTicker(p.config.CycleTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			// 循环读取所有设备
			for ip, device := range p.devices {
				if !device.Active {
					continue
				}

				// 读取输入数据
				data, err := p.ReadInputData(ip)
				if err != nil {
					continue
				}

				// 检测数据变化
				if !bytesEqual(device.InputData, data) {
					update := &interfaces.DataUpdate{
						TargetID:   fmt.Sprintf("PROFINET_Device_%s", ip),
						Value:      data,
						OldValue:   device.InputData,
						Quality:    1.0,
						Timestamp:  time.Now(),
						ChangeType: "value",
						Metadata: map[string]interface{}{
							"protocol":    "PROFINET",
							"device_ip":   ip,
							"device_name": device.DeviceName,
							"vendor_id":   device.VendorID,
						},
					}

					select {
					case p.dataChan <- update:
					default:
						// 缓冲区满，丢弃
					}

					device.InputData = data
				}

				device.LastUpdateTime = time.Now()
			}
		}
	}
}

// GetDataChannel 获取数据通道
func (p *PROFINET) GetDataChannel() <-chan *interfaces.DataUpdate {
	return p.dataChan
}

// GetStatus 获取协议状态
func (p *PROFINET) GetStatus() *interfaces.ProtocolStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := *p.stats
	return &status
}

// GetAllDevices 获取所有设备
func (p *PROFINET) GetAllDevices() map[string]*PROFINETDevice {
	p.mu.RLock()
	defer p.mu.RUnlock()

	devices := make(map[string]*PROFINETDevice)
	for ip, device := range p.devices {
		deviceCopy := *device
		devices[ip] = &deviceCopy
	}

	return devices
}

// Read 实现io.Reader接口
func (p *PROFINET) Read(buffer []byte) (int, error) {
	if p.conn == nil {
		return 0, fmt.Errorf("not connected")
	}
	return p.conn.Read(buffer)
}

// Write 实现io.Writer接口
func (p *PROFINET) Write(data []byte) (int, error) {
	if p.conn == nil {
		return 0, fmt.Errorf("not connected")
	}
	return p.conn.Write(data)
}
