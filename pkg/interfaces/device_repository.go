package interfaces

import (
	"context"
	deviceDomain "go_ProFiBus/internal/domain/device"
)

// DeviceRepository 设备仓储接口
type DeviceRepository interface {
	// Create 创建设备
	Create(ctx context.Context, device *deviceDomain.Device) error

	// GetByID 根据ID获取设备
	GetByID(ctx context.Context, id string) (*deviceDomain.Device, error)

	// List 列出设备
	List(ctx context.Context, filters DeviceFilters) ([]*deviceDomain.Device, error)

	// Update 更新设备
	Update(ctx context.Context, device *deviceDomain.Device) error

	// Delete 删除设备
	Delete(ctx context.Context, id string) error

	// UpdateStatus 更新设备状态
	UpdateStatus(ctx context.Context, id string, status deviceDomain.DeviceStatus) error

	// UpdateHealthScore 更新健康度评分
	UpdateHealthScore(ctx context.Context, id string, score float64) error

	// GetByChannelID 根据通道ID获取关联的设备列表
	GetByChannelID(ctx context.Context, channelID string) ([]*deviceDomain.Device, error)

	// AddChannel 添加设备与通道的关联
	AddChannel(ctx context.Context, deviceID, channelID string) error

	// RemoveChannel 移除设备与通道的关联
	RemoveChannel(ctx context.Context, deviceID, channelID string) error

	// GetChannels 获取设备关联的通道ID列表
	GetChannels(ctx context.Context, deviceID string) ([]string, error)
}

// DeviceFilters 设备过滤器
type DeviceFilters struct {
	Type   *deviceDomain.DeviceType
	Status *deviceDomain.DeviceStatus
	Area   *string
	Limit  int
	Offset int
}
