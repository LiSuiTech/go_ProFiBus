package interfaces

import (
	"context"
	"time"
	fusionDomain "go_ProFiBus/internal/domain/fusion"
)

// FusionRepository 通用融合仓储接口
type FusionRepository interface {
	// DataSource相关
	CreateDataSource(ctx context.Context, source *fusionDomain.DataSource) error
	GetDataSourceByID(ctx context.Context, id string) (*fusionDomain.DataSource, error)
	GetDataSourceByName(ctx context.Context, name string) (*fusionDomain.DataSource, error)
	ListDataSources(ctx context.Context, filters DataSourceFilters) ([]*fusionDomain.DataSource, error)
	UpdateDataSource(ctx context.Context, source *fusionDomain.DataSource) error
	DeleteDataSource(ctx context.Context, id string) error

	// FusionConfig相关
	CreateFusionConfig(ctx context.Context, config *fusionDomain.FusionConfig) error
	GetFusionConfigByID(ctx context.Context, id string) (*fusionDomain.FusionConfig, error)
	GetFusionConfigByName(ctx context.Context, name string) (*fusionDomain.FusionConfig, error)
	ListFusionConfigs(ctx context.Context, filters FusionConfigFilters) ([]*fusionDomain.FusionConfig, error)
	UpdateFusionConfig(ctx context.Context, config *fusionDomain.FusionConfig) error
	DeleteFusionConfig(ctx context.Context, id string) error

	// ConfigSourceRelation相关
	AddSourceToConfig(ctx context.Context, configID, sourceID string, weight float64) error
	RemoveSourceFromConfig(ctx context.Context, configID, sourceID string) error
	GetConfigSources(ctx context.Context, configID string) ([]*fusionDomain.ConfigSourceRelation, error)
	UpdateConfigSourceWeight(ctx context.Context, configID, sourceID string, weight float64) error

	// FusionResult相关
	SaveFusionResult(ctx context.Context, result *fusionDomain.FusionResult) error
	GetFusionResults(ctx context.Context, filters FusionResultFilters) ([]*fusionDomain.FusionResult, error)
	GetLatestFusionResult(ctx context.Context, configID string) (*fusionDomain.FusionResult, error)

	// SourceDataCache相关
	SaveSourceDataCache(ctx context.Context, cache *fusionDomain.SourceDataCache) error
	GetSourceDataCache(ctx context.Context, sourceID string, timeWindow time.Duration) ([]*fusionDomain.SourceDataCache, error)
	CleanExpiredCache(ctx context.Context, olderThan time.Time) error
}

// DataSourceFilters 数据源过滤器
type DataSourceFilters struct {
	SourceType *fusionDomain.SourceType
	DeviceID   *string
	ChannelID  *string
	Enabled    *bool
	Limit      int
	Offset     int
}

// FusionConfigFilters 融合配置过滤器
type FusionConfigFilters struct {
	Enabled *bool
	Limit   int
	Offset  int
}

// FusionResultFilters 融合结果过滤器
type FusionResultFilters struct {
	FusionConfigID *string
	StartTime      *time.Time
	EndTime        *time.Time
	Limit          int
	Offset         int
}
