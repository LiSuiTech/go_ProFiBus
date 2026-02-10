package interfaces

import (
	"context"
	"time"
)

// IndustrialProtocol 统一的工业协议接口
// 所有工业通信协议（现场总线、工业以太网、IT协议）都实现此接口
type IndustrialProtocol interface {
	// GetName 获取协议名称
	GetName() string

	// GetType 获取协议类型
	GetType() ProtocolType

	// Connect 连接到设备/网络
	Connect(ctx context.Context, config ProtocolConfig) error

	// Disconnect 断开连接
	Disconnect() error

	// IsConnected 检查连接状态
	IsConnected() bool

	// GetStatus 获取协议状态
	GetStatus() ProtocolStatus

	// --- 数据读取功能 ---

	// Read 读取数据
	Read(ctx context.Context, request ReadRequest) (*ReadResponse, error)

	// ReadBatch 批量读取数据
	ReadBatch(ctx context.Context, requests []ReadRequest) ([]*ReadResponse, error)

	// --- 数据监听功能 ---

	// StartMonitoring 启动数据监听
	StartMonitoring(ctx context.Context, config MonitoringConfig) (<-chan DataUpdate, error)

	// StopMonitoring 停止数据监听
	StopMonitoring() error

	// --- 设备管控功能 ---

	// Write 写入数据
	Write(ctx context.Context, request WriteRequest) error

	// WriteBatch 批量写入数据
	WriteBatch(ctx context.Context, requests []WriteRequest) error

	// ExecuteCommand 执行命令
	ExecuteCommand(ctx context.Context, command Command) (*CommandResponse, error)

	// --- 诊断和管理 ---

	// GetDiagnostics 获取诊断信息
	GetDiagnostics(ctx context.Context) (*Diagnostics, error)

	// Reset 重置协议/设备
	Reset(ctx context.Context) error
}

// ProtocolType 协议类型
type ProtocolType string

const (
	// 现场总线协议
	ProtocolPROFIBUS  ProtocolType = "PROFIBUS"   // PROFIBUS DP/PA
	ProtocolModbus    ProtocolType = "Modbus"     // Modbus 通用（由配置区分 RTU/TCP）
	ProtocolModbusRTU ProtocolType = "ModbusRTU"  // Modbus RTU
	ProtocolDeviceNet ProtocolType = "DeviceNet"  // DeviceNet (CAN-based)
	ProtocolHART      ProtocolType = "HART"       // HART (Highway Addressable Remote Transducer)
	ProtocolASI       ProtocolType = "AS-i"       // AS-Interface

	// 工业以太网协议
	ProtocolPROFINET   ProtocolType = "PROFINET"   // PROFINET IO
	ProtocolEtherCAT   ProtocolType = "EtherCAT"   // EtherCAT
	ProtocolEtherNetIP ProtocolType = "EtherNetIP" // EtherNet/IP (Rockwell/Allen-Bradley)
	ProtocolModbusTCP  ProtocolType = "ModbusTCP"  // Modbus TCP
	ProtocolPowerlink  ProtocolType = "Powerlink"  // POWERLINK

	// IT层协议
	ProtocolOPCUA    ProtocolType = "OPCUA"    // OPC UA (Client/Server, Pub/Sub)
	ProtocolMQTT     ProtocolType = "MQTT"     // MQTT
	ProtocolHTTP     ProtocolType = "HTTP"     // HTTP/REST API
	ProtocolDatabase ProtocolType = "Database" // 数据库 (ODBC/JDBC)
	ProtocolOPC      ProtocolType = "OPC"      // OPC Classic (DA/AE/HDA)

	// 其他协议
	ProtocolCAN      ProtocolType = "CAN"      // CAN bus
	ProtocolCANopen  ProtocolType = "CANopen"  // CANopen
	ProtocolBACnet   ProtocolType = "BACnet"   // BACnet (楼宇自动化)
	ProtocolKNX      ProtocolType = "KNX"      // KNX (楼宇自动化)
	ProtocolLonWorks ProtocolType = "LonWorks" // LonWorks
)

// ProtocolConfig 协议配置接口
type ProtocolConfig interface {
	Validate() error
	GetType() ProtocolType
}

// ProtocolStatus 协议状态
type ProtocolStatus struct {
	Connected        bool          // 是否已连接
	Running          bool          // 是否正在运行
	LastActivity     time.Time     // 最后活动时间
	BytesSent        uint64        // 发送字节数
	BytesReceived    uint64        // 接收字节数
	RequestCount     uint64        // 请求次数
	ErrorCount       uint64        // 错误次数
	AverageLatency   time.Duration // 平均延迟
	CurrentLatency   time.Duration // 当前延迟
	ConnectionUptime time.Duration // 连接时长
	Error            error         // 最后错误
}

// ReadRequest 读取请求
type ReadRequest struct {
	// 目标标识
	TargetID   string // 设备ID/从站地址/节点ID
	TargetType string // 目标类型 (device/slave/node)

	// 数据地址
	AddressType string      // 地址类型 (register/coil/tag/variable)
	Address     interface{} // 地址（可能是数字、字符串或结构体）
	Quantity    uint16      // 数量

	// 协议特定参数
	Parameters map[string]interface{}

	// 超时设置
	Timeout time.Duration
}

// ReadResponse 读取响应
type ReadResponse struct {
	Success    bool                   // 是否成功
	Data       interface{}            // 数据（类型根据协议不同）
	DataType   string                 // 数据类型
	Quality    float64                // 数据质量 (0.0-1.0)
	Timestamp  time.Time              // 时间戳
	Error      error                  // 错误信息
	Metadata   map[string]interface{} // 元数据
	SourceAddr string                 // 源地址
}

// WriteRequest 写入请求
type WriteRequest struct {
	// 目标标识
	TargetID   string // 设备ID/从站地址/节点ID
	TargetType string // 目标类型

	// 数据地址
	AddressType string      // 地址类型
	Address     interface{} // 地址
	Value       interface{} // 要写入的值
	ValueType   string      // 值类型

	// 协议特定参数
	Parameters map[string]interface{}

	// 超时设置
	Timeout time.Duration
}

// MonitoringConfig 监听配置
type MonitoringConfig struct {
	// 监听目标
	Targets []string // 要监听的目标列表

	// 监听模式
	Mode string // polling/subscribe/cyclic

	// 轮询配置
	PollingInterval time.Duration // 轮询间隔
	PollingBatch    bool          // 是否批量轮询

	// 订阅配置
	SubscriptionFilter string // 订阅过滤器

	// 变化检测
	DetectChanges bool    // 是否检测变化
	ChangeDeadband float64 // 变化死区

	// 缓冲配置
	BufferSize int // 缓冲区大小
}

// DataUpdate 数据更新
type DataUpdate struct {
	TargetID  string                 // 目标ID
	Address   interface{}            // 地址
	Value     interface{}            // 新值
	OldValue  interface{}            // 旧值（如果有）
	Quality   float64                // 数据质量
	Timestamp time.Time              // 时间戳
	ChangeType string                 // 变化类型 (value/quality/status)
	Metadata  map[string]interface{} // 元数据
}

// Command 命令
type Command struct {
	CommandType string                 // 命令类型
	TargetID    string                 // 目标ID
	Parameters  map[string]interface{} // 参数
	Timeout     time.Duration          // 超时
}

// CommandResponse 命令响应
type CommandResponse struct {
	Success   bool                   // 是否成功
	Result    interface{}            // 结果
	Message   string                 // 消息
	Error     error                  // 错误
	Timestamp time.Time              // 时间戳
	Metadata  map[string]interface{} // 元数据
}

// Diagnostics 诊断信息
type Diagnostics struct {
	DeviceStatus   string                 // 设备状态
	Errors         []DiagnosticError      // 错误列表
	Warnings       []DiagnosticWarning    // 警告列表
	Statistics     map[string]interface{} // 统计信息
	Configuration  map[string]interface{} // 配置信息
	LastUpdateTime time.Time              // 最后更新时间
}

// DiagnosticError 诊断错误
type DiagnosticError struct {
	Code      string    // 错误码
	Message   string    // 错误消息
	Severity  int       // 严重程度 (1-5)
	Timestamp time.Time // 发生时间
	Source    string    // 来源
}

// DiagnosticWarning 诊断警告
type DiagnosticWarning struct {
	Code      string    // 警告码
	Message   string    // 警告消息
	Timestamp time.Time // 发生时间
	Source    string    // 来源
}

// ProtocolFactory 协议工厂接口
type ProtocolFactory interface {
	// CreateProtocol 创建协议实例
	CreateProtocol(protocolType ProtocolType, config ProtocolConfig) (IndustrialProtocol, error)

	// SupportedProtocols 获取支持的协议列表
	SupportedProtocols() []ProtocolType

	// GetProtocolInfo 获取协议信息
	GetProtocolInfo(protocolType ProtocolType) *ProtocolInfo
}

// ProtocolInfo 协议信息
type ProtocolInfo struct {
	Type        ProtocolType // 协议类型
	Name        string       // 协议名称
	Description string       // 描述
	Category    string       // 类别 (fieldbus/industrial_ethernet/it_layer)
	Vendor      string       // 厂商
	Standard    string       // 标准 (IEC/IEEE等)
	Features    []string     // 特性列表
	UseCase     []string     // 使用场景
}

// ProtocolAdapter 协议适配器接口
// 用于将不同协议统一适配到DataSource接口
type ProtocolAdapter interface {
	DataSource // 继承DataSource接口

	// GetProtocol 获取底层协议实例
	GetProtocol() IndustrialProtocol

	// SetProtocol 设置底层协议实例
	SetProtocol(protocol IndustrialProtocol) error
}

// ProtocolConverter 协议转换器接口
// 用于协议之间的数据转换
type ProtocolConverter interface {
	// Convert 转换数据格式
	Convert(from ProtocolType, to ProtocolType, data interface{}) (interface{}, error)

	// ConvertAddress 转换地址格式
	ConvertAddress(from ProtocolType, to ProtocolType, address interface{}) (interface{}, error)
}

// ProtocolGateway 协议网关接口
// 用于协议之间的桥接和转换
type ProtocolGateway interface {
	// AddProtocol 添加协议
	AddProtocol(name string, protocol IndustrialProtocol) error

	// RemoveProtocol 移除协议
	RemoveProtocol(name string) error

	// Route 路由数据（从一个协议到另一个协议）
	Route(ctx context.Context, fromProtocol string, toProtocol string, data interface{}) error

	// StartBridge 启动协议桥接
	StartBridge(ctx context.Context, config BridgeConfig) error

	// StopBridge 停止协议桥接
	StopBridge() error
}

// BridgeConfig 桥接配置
type BridgeConfig struct {
	SourceProtocol      string                 // 源协议
	DestinationProtocol string                 // 目标协议
	Mapping             []AddressMapping       // 地址映射
	Bidirectional       bool                   // 是否双向
	SyncInterval        time.Duration          // 同步间隔
	Parameters          map[string]interface{} // 其他参数
}

// AddressMapping 地址映射
type AddressMapping struct {
	SourceAddress      interface{}            // 源地址
	DestinationAddress interface{}            // 目标地址
	DataType           string                 // 数据类型
	Scale              float64                // 缩放因子
	Offset             float64                // 偏移量
	Transformation     string                 // 转换函数名
	Parameters         map[string]interface{} // 转换参数
}
