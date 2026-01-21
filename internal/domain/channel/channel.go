package channel

import (
	"time"
)

// ProtocolType 协议类型
type ProtocolType string

const (
	ProtocolUART   ProtocolType = "UART"
	ProtocolCAN    ProtocolType = "CAN"
	ProtocolUSB    ProtocolType = "USB"
	ProtocolOneWire ProtocolType = "1-Wire"
	ProtocolModbus ProtocolType = "Modbus"
	ProtocolRS232  ProtocolType = "RS-232"
	ProtocolRS485  ProtocolType = "RS-485"
	ProtocolI2C    ProtocolType = "I2C"
	ProtocolSPI    ProtocolType = "SPI"
)

// ChannelStatus 通道状态
type ChannelStatus string

const (
	StatusStopped ChannelStatus = "stopped"
	StatusRunning ChannelStatus = "running"
	StatusError   ChannelStatus = "error"
)

// DataType 数据类型
type DataType string

const (
	DataTypeInt     DataType = "int"
	DataTypeFloat   DataType = "float"
	DataTypeBool    DataType = "bool"
	DataTypeString  DataType = "string"
	DataTypeBytes   DataType = "bytes"
)

// Channel 采集通道
type Channel struct {
	ID          string        `json:"id" db:"id"`
	Name        string        `json:"name" db:"name"`
	Description string        `json:"description" db:"description"`
	Protocol    ProtocolType  `json:"protocol" db:"protocol"`
	DeviceName  string        `json:"device_name" db:"device_name"` // 设备名称
	DevicePort  string        `json:"device_port" db:"device_port"` // 设备端口 (如 /dev/ttyUSB0, COM1)
	Status      ChannelStatus `json:"status" db:"status"`
	Config      ProtocolConfig `json:"config"` // 协议特定配置
	Enabled     bool          `json:"enabled" db:"enabled"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
	Points      []Point       `json:"points,omitempty"` // 采集点位列表
}

// ProtocolConfig 协议配置（不同协议有不同的配置参数）
type ProtocolConfig struct {
	// 串口通用配置
	BaudRate int    `json:"baud_rate,omitempty"` // 波特率
	DataBits int    `json:"data_bits,omitempty"` // 数据位
	StopBits int    `json:"stop_bits,omitempty"` // 停止位
	Parity   string `json:"parity,omitempty"`    // 校验位 (none, odd, even)

	// Modbus 配置
	SlaveID      uint8  `json:"slave_id,omitempty"`      // 从站ID
	ModbusMode   string `json:"modbus_mode,omitempty"`   // RTU or TCP

	// I2C 配置
	I2CAddress   uint8  `json:"i2c_address,omitempty"`   // I2C 地址

	// 其他配置
	Timeout      int    `json:"timeout,omitempty"`       // 超时时间(ms)
	RetryCount   int    `json:"retry_count,omitempty"`   // 重试次数
	PollInterval int    `json:"poll_interval,omitempty"` // 轮询间隔(ms)
}

// Point 采集点位
type Point struct {
	ID          string    `json:"id" db:"id"`
	ChannelID   string    `json:"channel_id" db:"channel_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Address     string    `json:"address" db:"address"`     // 点位地址 (如 40001, 0x10)
	DataType    DataType  `json:"data_type" db:"data_type"` // 数据类型
	Unit        string    `json:"unit" db:"unit"`           // 单位
	Scale       float64   `json:"scale" db:"scale"`         // 缩放系数
	Offset      float64   `json:"offset" db:"offset"`       // 偏移量
	Enabled     bool      `json:"enabled" db:"enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ProtocolInfo 协议信息
type ProtocolInfo struct {
	Type        ProtocolType `json:"type"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	ConfigFields []ConfigField `json:"config_fields"`
}

// ConfigField 配置字段
type ConfigField struct {
	Key         string      `json:"key"`
	Label       string      `json:"label"`
	Type        string      `json:"type"` // text, number, select
	Required    bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Options     []Option    `json:"options,omitempty"` // for select type
}

// Option 选项
type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// GetSupportedProtocols 获取支持的协议列表
func GetSupportedProtocols() []ProtocolInfo {
	return []ProtocolInfo{
		{
			Type:        ProtocolUART,
			Name:        "UART",
			Description: "通用异步收发器",
			ConfigFields: []ConfigField{
				{Key: "baud_rate", Label: "波特率", Type: "select", Required: true, DefaultValue: 9600,
					Options: []Option{
						{Label: "9600", Value: "9600"},
						{Label: "19200", Value: "19200"},
						{Label: "38400", Value: "38400"},
						{Label: "57600", Value: "57600"},
						{Label: "115200", Value: "115200"},
					}},
				{Key: "data_bits", Label: "数据位", Type: "select", Required: true, DefaultValue: 8,
					Options: []Option{
						{Label: "7", Value: "7"},
						{Label: "8", Value: "8"},
					}},
				{Key: "stop_bits", Label: "停止位", Type: "select", Required: true, DefaultValue: 1,
					Options: []Option{
						{Label: "1", Value: "1"},
						{Label: "2", Value: "2"},
					}},
				{Key: "parity", Label: "校验位", Type: "select", Required: true, DefaultValue: "none",
					Options: []Option{
						{Label: "None", Value: "none"},
						{Label: "Odd", Value: "odd"},
						{Label: "Even", Value: "even"},
					}},
			},
		},
		{
			Type:        ProtocolModbus,
			Name:        "Modbus",
			Description: "工业通信协议",
			ConfigFields: []ConfigField{
				{Key: "modbus_mode", Label: "Modbus 模式", Type: "select", Required: true, DefaultValue: "RTU",
					Options: []Option{
						{Label: "RTU", Value: "RTU"},
						{Label: "TCP", Value: "TCP"},
					}},
				{Key: "slave_id", Label: "从站ID", Type: "number", Required: true, DefaultValue: 1},
				{Key: "baud_rate", Label: "波特率", Type: "select", Required: true, DefaultValue: 9600,
					Options: []Option{
						{Label: "9600", Value: "9600"},
						{Label: "19200", Value: "19200"},
						{Label: "115200", Value: "115200"},
					}},
			},
		},
		{
			Type:        ProtocolRS485,
			Name:        "RS-485",
			Description: "差分串行接口",
			ConfigFields: []ConfigField{
				{Key: "baud_rate", Label: "波特率", Type: "select", Required: true, DefaultValue: 9600,
					Options: []Option{
						{Label: "9600", Value: "9600"},
						{Label: "19200", Value: "19200"},
						{Label: "38400", Value: "38400"},
						{Label: "115200", Value: "115200"},
					}},
				{Key: "data_bits", Label: "数据位", Type: "number", Required: true, DefaultValue: 8},
				{Key: "stop_bits", Label: "停止位", Type: "number", Required: true, DefaultValue: 1},
				{Key: "parity", Label: "校验位", Type: "select", Required: true, DefaultValue: "none",
					Options: []Option{
						{Label: "None", Value: "none"},
						{Label: "Odd", Value: "odd"},
						{Label: "Even", Value: "even"},
					}},
			},
		},
		{
			Type:        ProtocolI2C,
			Name:        "I2C",
			Description: "互连集成电路",
			ConfigFields: []ConfigField{
				{Key: "i2c_address", Label: "I2C 地址", Type: "text", Required: true, DefaultValue: "0x48"},
			},
		},
		{
			Type:        ProtocolCAN,
			Name:        "CAN",
			Description: "控制器局域网络",
			ConfigFields: []ConfigField{
				{Key: "baud_rate", Label: "波特率", Type: "select", Required: true, DefaultValue: 250000,
					Options: []Option{
						{Label: "125K", Value: "125000"},
						{Label: "250K", Value: "250000"},
						{Label: "500K", Value: "500000"},
						{Label: "1M", Value: "1000000"},
					}},
			},
		},
	}
}
