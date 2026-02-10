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

// PROFIBUS PROFIBUS DP/PA协议实现
// 支持DP-V0, DP-V1, DP-V2和PA (Process Automation)
type PROFIBUS struct {
	config      *PROFIBUSConfig
	transport   io.ReadWriteCloser // RS-485串口
	master      bool                // 是否为主站
	slaves      map[byte]*SlaveInfo // 从站信息
	cyclic      bool                // 是否循环通信
	monitoring  bool                // 是否监听模式
	dataChan    chan *interfaces.DataUpdate
	stopChan    chan struct{}
	mu          sync.RWMutex
	stats       *interfaces.ProtocolStatus
}

// PROFIBUSConfig PROFIBUS配置
type PROFIBUSConfig struct {
	// 基本配置
	Mode         PROFIBUSMode // DP或PA模式
	Role         PROFIBUSRole // Master/Slave
	MasterAddr   byte         // 主站地址 (0-125)
	BaudRate     int          // 波特率 (9.6K-12M)

	// 串口配置
	SerialPort   string       // 串口路径

	// DP配置
	SlaveAddrs   []byte       // 从站地址列表
	Watchdog     time.Duration // 看门狗超时
	MinSlaveInt  time.Duration // 最小从站间隔

	// PA配置 (Process Automation)
	PAProfile    string       // PA配置文件

	// 循环配置
	CycleTime    time.Duration // 循环时间
	DataExchange bool          // 是否数据交换模式

	// 高级功能
	DPVI         bool         // 是否支持DP-V1非循环通信
	DPVII        bool         // 是否支持DP-V2
	IsochronousMode bool      // 是否等时模式

	Timeout      time.Duration
}

// PROFIBUSMode PROFIBUS模式
type PROFIBUSMode int

const (
	PROFIBUS_DP PROFIBUSMode = iota // DP (Decentralized Periphery)
	PROFIBUS_PA                      // PA (Process Automation)
)

// PROFIBUSRole PROFIBUS角色
type PROFIBUSRole int

const (
	PROFIBUSMaster PROFIBUSRole = iota // 主站 (Class 1/Class 2)
	PROFIBUSSlave                       // 从站
)

// PROFIBUS功能码
const (
	// 循环数据交换
	FC_SDA  = 0x0D // Send Data with Acknowledge
	FC_SDN  = 0x0C // Send Data with No Acknowledge
	FC_SRD  = 0x0E // Send and Request Data

	// 非循环服务 (DP-V1)
	FC_MSAC1_READ  = 0x5E // Read
	FC_MSAC1_WRITE = 0x5F // Write

	// 诊断
	FC_RD_DIAG = 0x3D // Read Diagnosis

	// 参数化
	FC_SET_PRM = 0x3C // Set Parameters
	FC_CHK_CFG = 0x3E // Check Configuration

	// 控制
	FC_FDL_STATUS = 0x09 // Request FDL Status
	FC_IDENT      = 0x31 // Request Identification
)

// SlaveInfo 从站信息
type SlaveInfo struct {
	Address      byte              // 从站地址
	Configured   bool              // 是否已配置
	Active       bool              // 是否活动
	InputData    []byte            // 输入数据
	OutputData   []byte            // 输出数据
	Diagnostics  *SlaveDiagnostics // 诊断信息
	LastUpdate   time.Time         // 最后更新时间
	ErrorCount   uint32            // 错误计数
}

// SlaveDiagnostics 从站诊断
type SlaveDiagnostics struct {
	StationStatus1 byte   // 站状态1
	StationStatus2 byte   // 站状态2
	StationStatus3 byte   // 站状态3
	MasterAddr     byte   // 主站地址
	IdentNumber    uint16 // 识别号
	ExtDiag        []byte // 扩展诊断
}

// NewPROFIBUS 创建PROFIBUS实例
func NewPROFIBUS(config *PROFIBUSConfig) (*PROFIBUS, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	pb := &PROFIBUS{
		config:   config,
		master:   config.Role == PROFIBUSMaster,
		slaves:   make(map[byte]*SlaveInfo),
		dataChan: make(chan *interfaces.DataUpdate, 1000),
		stopChan: make(chan struct{}),
		stats: &interfaces.ProtocolStatus{
			Connected: false,
			Running:   false,
		},
	}

	// 初始化从站信息
	if pb.master {
		for _, addr := range config.SlaveAddrs {
			pb.slaves[addr] = &SlaveInfo{
				Address:    addr,
				Configured: false,
				Active:     false,
				InputData:  make([]byte, 0),
				OutputData: make([]byte, 0),
			}
		}
	}

	return pb, nil
}

// Validate 验证配置
func (c *PROFIBUSConfig) Validate() error {
	if c.SerialPort == "" {
		return fmt.Errorf("serial_port is required")
	}
	if c.BaudRate == 0 {
		c.BaudRate = 19200 // 默认波特率
	}
	if c.Role == PROFIBUSMaster && len(c.SlaveAddrs) == 0 {
		return fmt.Errorf("slave addresses required for master")
	}
	if c.Timeout == 0 {
		c.Timeout = 100 * time.Millisecond
	}
	if c.CycleTime == 0 {
		c.CycleTime = 10 * time.Millisecond
	}
	return nil
}

// GetType 获取配置类型
func (c *PROFIBUSConfig) GetType() interfaces.ProtocolType {
	return interfaces.ProtocolPROFIBUS
}

// Open 打开PROFIBUS连接
func (pb *PROFIBUS) Open(devicePath string) error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// 打开RS-485串口
	port := devicePath
	if port == "" {
		port = pb.config.SerialPort
	}

	serialPort := &RealSerialPort{}
	err := serialPort.Open(port,
		WithBaudRate(pb.config.BaudRate),
		WithDataBits(8),
		WithStopBits(1),
		WithParity(ParityEven), // PROFIBUS 常用偶校验
	)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}

	pb.transport = serialPort
	pb.stats.Connected = true

	// 如果是主站，初始化总线
	if pb.master {
		err = pb.initializeBus()
		if err != nil {
			pb.transport.Close()
			pb.stats.Connected = false
			return fmt.Errorf("failed to initialize bus: %w", err)
		}
	}

	return nil
}

// Close 关闭PROFIBUS连接
func (pb *PROFIBUS) Close() error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if pb.cyclic {
		pb.stopChan <- struct{}{}
	}

	if pb.transport != nil {
		pb.transport.Close()
		pb.transport = nil
	}

	pb.stats.Connected = false
	pb.stats.Running = false

	return nil
}

// initializeBus 初始化总线（主站）
func (pb *PROFIBUS) initializeBus() error {
	// 1. 发送FDL_STATUS获取总线状态
	// 2. 对每个从站进行参数化
	// 3. 检查配置
	// 4. 切换到数据交换模式

	for addr, slave := range pb.slaves {
		// 参数化从站
		err := pb.parameterizeSlave(addr)
		if err != nil {
			return fmt.Errorf("failed to parameterize slave %d: %w", addr, err)
		}

		// 检查配置
		err = pb.checkConfig(addr)
		if err != nil {
			return fmt.Errorf("failed to check config for slave %d: %w", addr, err)
		}

		slave.Configured = true
		slave.Active = true
	}

	return nil
}

// parameterizeSlave 参数化从站
func (pb *PROFIBUS) parameterizeSlave(slaveAddr byte) error {
	// 构建SET_PRM报文
	telegram := pb.buildSetPrmTelegram(slaveAddr)

	_, err := pb.sendTelegram(telegram)
	return err
}

// checkConfig 检查从站配置
func (pb *PROFIBUS) checkConfig(slaveAddr byte) error {
	// 构建CHK_CFG报文
	telegram := pb.buildChkCfgTelegram(slaveAddr)

	_, err := pb.sendTelegram(telegram)
	return err
}

// ReadInputData 读取从站输入数据 - 实现读取功能
func (pb *PROFIBUS) ReadInputData(slaveAddr byte) ([]byte, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if !pb.master {
		return nil, fmt.Errorf("only master can read from slaves")
	}

	slave, exists := pb.slaves[slaveAddr]
	if !exists {
		return nil, fmt.Errorf("slave %d not found", slaveAddr)
	}

	if !slave.Active {
		return nil, fmt.Errorf("slave %d not active", slaveAddr)
	}

	// 发送SRD (Send and Request Data)
	telegram := pb.buildSRDTelegram(slaveAddr, slave.OutputData)
	response, err := pb.sendTelegram(telegram)
	if err != nil {
		slave.ErrorCount++
		return nil, err
	}

	// 解析响应，提取输入数据
	inputData := pb.parseDataResponse(response)
	slave.InputData = inputData
	slave.LastUpdate = time.Now()

	return inputData, nil
}

// WriteOutputData 写入从站输出数据 - 实现管控功能
func (pb *PROFIBUS) WriteOutputData(slaveAddr byte, data []byte) error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if !pb.master {
		return fmt.Errorf("only master can write to slaves")
	}

	slave, exists := pb.slaves[slaveAddr]
	if !exists {
		return fmt.Errorf("slave %d not found", slaveAddr)
	}

	slave.OutputData = data

	// 发送SDN或SDA报文
	telegram := pb.buildSDATelegram(slaveAddr, data)
	_, err := pb.sendTelegram(telegram)
	if err != nil {
		slave.ErrorCount++
		return err
	}

	return nil
}

// ReadDiagnostics 读取从站诊断信息
func (pb *PROFIBUS) ReadDiagnostics(slaveAddr byte) (*SlaveDiagnostics, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// 发送RD_DIAG报文
	telegram := pb.buildRdDiagTelegram(slaveAddr)
	response, err := pb.sendTelegram(telegram)
	if err != nil {
		return nil, err
	}

	// 解析诊断数据
	diag := pb.parseDiagnostics(response)

	if slave, exists := pb.slaves[slaveAddr]; exists {
		slave.Diagnostics = diag
	}

	return diag, nil
}

// StartCyclicExchange 启动循环数据交换 - 实现监听功能
func (pb *PROFIBUS) StartCyclicExchange(ctx context.Context) error {
	if !pb.master {
		return fmt.Errorf("only master can start cyclic exchange")
	}

	if pb.cyclic {
		return fmt.Errorf("cyclic exchange already running")
	}

	pb.cyclic = true
	pb.stats.Running = true

	go pb.cyclicLoop(ctx)

	return nil
}

// cyclicLoop 循环通信主循环
func (pb *PROFIBUS) cyclicLoop(ctx context.Context) {
	ticker := time.NewTicker(pb.config.CycleTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pb.cyclic = false
			return
		case <-pb.stopChan:
			pb.cyclic = false
			return
		case <-ticker.C:
			pb.performCycle()
		}
	}
}

// performCycle 执行一个循环周期
func (pb *PROFIBUS) performCycle() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	for addr, slave := range pb.slaves {
		if !slave.Active {
			continue
		}

		// 发送输出数据，接收输入数据
		telegram := pb.buildSRDTelegram(addr, slave.OutputData)
		response, err := pb.sendTelegram(telegram)

		if err != nil {
			slave.ErrorCount++
			pb.stats.ErrorCount++
			continue
		}

		// 解析输入数据
		oldData := slave.InputData
		newData := pb.parseDataResponse(response)
		slave.InputData = newData
		slave.LastUpdate = time.Now()

		// 检测变化并发送更新
		if !bytesEqualPB(oldData, newData) {
			update := &interfaces.DataUpdate{
				TargetID:  fmt.Sprintf("slave_%d", addr),
				Value:     newData,
				OldValue:  oldData,
				Quality:   1.0,
				Timestamp: time.Now(),
				ChangeType: "value",
				Metadata: map[string]interface{}{
					"slave_addr": addr,
					"protocol":   "PROFIBUS",
				},
			}

			select {
			case pb.dataChan <- update:
			default:
				// 缓冲区满，丢弃
			}
		}
	}
}

// buildSetPrmTelegram 构建SET_PRM报文
func (pb *PROFIBUS) buildSetPrmTelegram(slaveAddr byte) []byte {
	telegram := make([]byte, 0, 64)

	// SD2 (Start Delimiter 2) - 可变长度报文
	telegram = append(telegram, 0x68)

	// LE (Length) - 稍后填充
	lengthPos := len(telegram)
	telegram = append(telegram, 0x00) // LE
	telegram = append(telegram, 0x00) // LEr (重复)
	telegram = append(telegram, 0x68) // SD2重复

	// DA (Destination Address)
	telegram = append(telegram, slaveAddr)

	// SA (Source Address)
	telegram = append(telegram, pb.config.MasterAddr)

	// FC (Function Code)
	telegram = append(telegram, FC_SET_PRM)

	// 参数数据
	// 这里应该根据实际设备的GSD文件配置
	// 简化示例：
	telegram = append(telegram, 0x00) // Station_Status
	telegram = append(telegram, 0x00) // WD_Fact_1
	telegram = append(telegram, 0x00) // WD_Fact_2
	telegram = append(telegram, 0x00) // Min_Tsdr
	telegram = append(telegram, 0x00, 0x00) // Ident_Number
	telegram = append(telegram, 0x00) // Group_Ident

	// 填充长度
	dataLen := len(telegram) - 4
	telegram[lengthPos] = byte(dataLen)
	telegram[lengthPos+1] = byte(dataLen)

	// FCS (Frame Check Sequence)
	fcs := pb.calculateFCS(telegram[4:])
	telegram = append(telegram, fcs)

	// ED (End Delimiter)
	telegram = append(telegram, 0x16)

	return telegram
}

// buildChkCfgTelegram 构建CHK_CFG报文
func (pb *PROFIBUS) buildChkCfgTelegram(slaveAddr byte) []byte {
	telegram := make([]byte, 0, 64)

	telegram = append(telegram, 0x68) // SD2

	lengthPos := len(telegram)
	telegram = append(telegram, 0x00, 0x00, 0x68)

	telegram = append(telegram, slaveAddr)               // DA
	telegram = append(telegram, pb.config.MasterAddr)    // SA
	telegram = append(telegram, FC_CHK_CFG)              // FC

	// 配置数据（简化）
	telegram = append(telegram, 0x00) // Cfg_Data

	dataLen := len(telegram) - 4
	telegram[lengthPos] = byte(dataLen)
	telegram[lengthPos+1] = byte(dataLen)

	fcs := pb.calculateFCS(telegram[4:])
	telegram = append(telegram, fcs)
	telegram = append(telegram, 0x16) // ED

	return telegram
}

// buildSRDTelegram 构建SRD报文（发送并请求数据）
func (pb *PROFIBUS) buildSRDTelegram(slaveAddr byte, outputData []byte) []byte {
	telegram := make([]byte, 0, 64)

	if len(outputData) == 0 {
		// SD1 (短报文)
		telegram = append(telegram, 0x10)
		telegram = append(telegram, slaveAddr)
		telegram = append(telegram, pb.config.MasterAddr)
		telegram = append(telegram, FC_SRD)
		fcs := pb.calculateFCS(telegram[1:])
		telegram = append(telegram, fcs)
		telegram = append(telegram, 0x16)
	} else {
		// SD2 (长报文)
		telegram = append(telegram, 0x68)
		lengthPos := len(telegram)
		telegram = append(telegram, 0x00, 0x00, 0x68)
		telegram = append(telegram, slaveAddr)
		telegram = append(telegram, pb.config.MasterAddr)
		telegram = append(telegram, FC_SRD)
		telegram = append(telegram, outputData...)

		dataLen := len(telegram) - 4
		telegram[lengthPos] = byte(dataLen)
		telegram[lengthPos+1] = byte(dataLen)

		fcs := pb.calculateFCS(telegram[4:])
		telegram = append(telegram, fcs)
		telegram = append(telegram, 0x16)
	}

	return telegram
}

// buildSDATelegram 构建SDA报文（发送数据带确认）
func (pb *PROFIBUS) buildSDATelegram(slaveAddr byte, data []byte) []byte {
	telegram := make([]byte, 0, 64)

	telegram = append(telegram, 0x68)
	lengthPos := len(telegram)
	telegram = append(telegram, 0x00, 0x00, 0x68)
	telegram = append(telegram, slaveAddr)
	telegram = append(telegram, pb.config.MasterAddr)
	telegram = append(telegram, FC_SDA)
	telegram = append(telegram, data...)

	dataLen := len(telegram) - 4
	telegram[lengthPos] = byte(dataLen)
	telegram[lengthPos+1] = byte(dataLen)

	fcs := pb.calculateFCS(telegram[4:])
	telegram = append(telegram, fcs)
	telegram = append(telegram, 0x16)

	return telegram
}

// buildRdDiagTelegram 构建RD_DIAG报文
func (pb *PROFIBUS) buildRdDiagTelegram(slaveAddr byte) []byte {
	telegram := []byte{
		0x10,                      // SD1
		slaveAddr,                 // DA
		pb.config.MasterAddr,      // SA
		FC_RD_DIAG,                // FC
	}

	fcs := pb.calculateFCS(telegram[1:])
	telegram = append(telegram, fcs)
	telegram = append(telegram, 0x16) // ED

	return telegram
}

// sendTelegram 发送报文并接收响应
func (pb *PROFIBUS) sendTelegram(telegram []byte) ([]byte, error) {
	if pb.transport == nil {
		return nil, fmt.Errorf("not connected")
	}

	// 发送
	_, err := pb.transport.Write(telegram)
	if err != nil {
		pb.stats.ErrorCount++
		return nil, err
	}

	pb.stats.BytesSent += uint64(len(telegram))
	pb.stats.RequestCount++

	// 接收响应
	response := make([]byte, 256)
	n, err := pb.transport.Read(response)
	if err != nil {
		pb.stats.ErrorCount++
		return nil, err
	}

	pb.stats.BytesReceived += uint64(n)

	return response[:n], nil
}

// parseDataResponse 解析数据响应
func (pb *PROFIBUS) parseDataResponse(response []byte) []byte {
	if len(response) < 5 {
		return nil
	}

	// 根据报文类型解析
	sd := response[0]

	if sd == 0x68 { // SD2 - 可变长度
		if len(response) < 9 {
			return nil
		}
		length := int(response[1])
		if len(response) < length+6 {
			return nil
		}
		// 数据在FC之后
		return response[7 : 7+length-3]
	} else if sd == 0x10 { // SD1 - 无数据
		return []byte{}
	}

	return nil
}

// parseDiagnostics 解析诊断数据
func (pb *PROFIBUS) parseDiagnostics(response []byte) *SlaveDiagnostics {
	if len(response) < 10 {
		return nil
	}

	diag := &SlaveDiagnostics{}

	// 根据PROFIBUS标准解析诊断数据
	// 简化示例
	dataStart := 7
	if len(response) > dataStart {
		diag.StationStatus1 = response[dataStart]
		if len(response) > dataStart+1 {
			diag.StationStatus2 = response[dataStart+1]
		}
		if len(response) > dataStart+2 {
			diag.StationStatus3 = response[dataStart+2]
		}
		if len(response) > dataStart+3 {
			diag.MasterAddr = response[dataStart+3]
		}
		if len(response) > dataStart+5 {
			diag.IdentNumber = binary.BigEndian.Uint16(response[dataStart+4:dataStart+6])
		}
	}

	return diag
}

// calculateFCS 计算帧校验序列
func (pb *PROFIBUS) calculateFCS(data []byte) byte {
	var fcs byte
	for _, b := range data {
		fcs += b
	}
	return fcs
}

// GetDataChannel 获取数据更新通道 - 实现监听功能
func (pb *PROFIBUS) GetDataChannel() <-chan *interfaces.DataUpdate {
	return pb.dataChan
}

// GetStatus 获取协议状态
func (pb *PROFIBUS) GetStatus() *interfaces.ProtocolStatus {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	status := *pb.stats
	return &status
}

// GetSlaveInfo 获取从站信息
func (pb *PROFIBUS) GetSlaveInfo(slaveAddr byte) (*SlaveInfo, error) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	slave, exists := pb.slaves[slaveAddr]
	if !exists {
		return nil, fmt.Errorf("slave %d not found", slaveAddr)
	}

	return slave, nil
}

// GetAllSlaves 获取所有从站信息
func (pb *PROFIBUS) GetAllSlaves() map[byte]*SlaveInfo {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	slaves := make(map[byte]*SlaveInfo)
	for addr, slave := range pb.slaves {
		slaves[addr] = slave
	}

	return slaves
}

// bytesEqual 比较两个字节数组是否相等
func bytesEqualPB(a, b []byte) bool {
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

// Read 实现io.Reader接口
func (pb *PROFIBUS) Read(buffer []byte) (int, error) {
	if pb.transport == nil {
		return 0, fmt.Errorf("not connected")
	}
	return pb.transport.Read(buffer)
}

// Write 实现io.Writer接口
func (pb *PROFIBUS) Write(data []byte) (int, error) {
	if pb.transport == nil {
		return 0, fmt.Errorf("not connected")
	}
	return pb.transport.Write(data)
}
