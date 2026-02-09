package interfaces

import (
	"context"
	"time"
	deviceDomain "go_ProFiBus/internal/domain/device"
)

// DeviceDataRepository 设备数据仓储接口
type DeviceDataRepository interface {
	// DataField相关
	CreateDataField(ctx context.Context, field *deviceDomain.DataField) error
	GetDataFieldByID(ctx context.Context, id string) (*deviceDomain.DataField, error)
	GetDataFieldsByDevice(ctx context.Context, deviceID string) ([]*deviceDomain.DataField, error)
	UpdateDataField(ctx context.Context, field *deviceDomain.DataField) error
	DeleteDataField(ctx context.Context, id string) error

	// DataSource相关
	CreateDataSource(ctx context.Context, source *deviceDomain.DataSource) error
	GetDataSourceByID(ctx context.Context, id string) (*deviceDomain.DataSource, error)
	GetDataSourcesByDevice(ctx context.Context, deviceID string) ([]*deviceDomain.DataSource, error)
	UpdateDataSource(ctx context.Context, source *deviceDomain.DataSource) error
	DeleteDataSource(ctx context.Context, id string) error

	// FusionConfig相关
	CreateFusionConfig(ctx context.Context, config *deviceDomain.FusionConfig) error
	GetFusionConfigByDevice(ctx context.Context, deviceID string) (*deviceDomain.FusionConfig, error)
	UpdateFusionConfig(ctx context.Context, config *deviceDomain.FusionConfig) error
	DeleteFusionConfig(ctx context.Context, deviceID string) error

	// FusedData相关
	SaveFusedData(ctx context.Context, data *deviceDomain.FusedData) error
	GetFusedDataByDevice(ctx context.Context, deviceID string, start, end time.Time, limit int) ([]*deviceDomain.FusedData, error)
	GetLatestFusedData(ctx context.Context, deviceID string) (*deviceDomain.FusedData, error)
}
