package device

import (
	"time"
)

// Device 设备实体
type Device struct {
	ID          string
	Name        string
	Description string
	Type        DeviceType
	Status      DeviceStatus
	HealthScore float64 // 健康度评分 0-100
	Location    Location
	Area        string
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DeviceType 设备类型
type DeviceType string

const (
	DeviceTypePLC         DeviceType = "PLC"
	DeviceTypeSensor      DeviceType = "Sensor"
	DeviceTypeInstrument  DeviceType = "Instrument"
	DeviceTypeSmartDevice DeviceType = "SmartDevice"
)

// DeviceStatus 设备状态
type DeviceStatus string

const (
	DeviceStatusOnline      DeviceStatus = "online"
	DeviceStatusOffline     DeviceStatus = "offline"
	DeviceStatusFault       DeviceStatus = "fault"
	DeviceStatusMaintenance DeviceStatus = "maintenance"
)

// Location 设备位置
type Location struct {
	X float64
	Y float64
	Z float64 // 可选，用于3D布局
}

// NewDevice 创建新设备
func NewDevice(id, name string, deviceType DeviceType) *Device {
	return &Device{
		ID:          id,
		Name:        name,
		Type:        deviceType,
		Status:      DeviceStatusOffline,
		HealthScore: 100.0,
		Location:    Location{},
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SetLocation 设置设备位置
func (d *Device) SetLocation(x, y, z float64) {
	d.Location = Location{X: x, Y: y, Z: z}
	d.UpdatedAt = time.Now()
}

// SetStatus 设置设备状态
func (d *Device) SetStatus(status DeviceStatus) {
	d.Status = status
	d.UpdatedAt = time.Now()
}

// SetHealthScore 设置健康度评分
func (d *Device) SetHealthScore(score float64) {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	d.HealthScore = score
	d.UpdatedAt = time.Now()
}

// SetMetadata 设置元数据
func (d *Device) SetMetadata(key string, value interface{}) {
	if d.Metadata == nil {
		d.Metadata = make(map[string]interface{})
	}
	d.Metadata[key] = value
	d.UpdatedAt = time.Now()
}

// IsOnline 检查设备是否在线
func (d *Device) IsOnline() bool {
	return d.Status == DeviceStatusOnline
}

// IsHealthy 检查设备是否健康（健康度 > 70）
func (d *Device) IsHealthy() bool {
	return d.HealthScore > 70.0
}
