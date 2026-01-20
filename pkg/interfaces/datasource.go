package interfaces

import (
	"context"
	"time"
)

// DataSource 数据源接口
// 提供统一的数据采集抽象，支持多种协议和数据源
type DataSource interface {
	// Start 启动数据源
	Start(ctx context.Context) error

	// Stop 停止数据源
	Stop() error

	// GetData 获取数据通道
	GetData() <-chan DataSample

	// GetStatus 获取数据源状态
	GetStatus() SourceStatus

	// GetID 获取数据源ID
	GetID() string

	// GetName 获取数据源名称
	GetName() string
}

// DataSample 数据样本接口
// 表示从数据源采集的单个数据样本
type DataSample interface {
	// GetTimestamp 获取时间戳
	GetTimestamp() time.Time

	// GetSourceID 获取数据源ID
	GetSourceID() string

	// GetData 获取原始数据
	GetData() map[string]interface{}

	// GetQuality 获取数据质量（0.0-1.0）
	GetQuality() float64

	// GetMetadata 获取元数据
	GetMetadata() map[string]interface{}
}

// SourceStatus 数据源状态
type SourceStatus struct {
	// Running 是否正在运行
	Running bool

	// Connected 是否已连接
	Connected bool

	// Error 错误信息
	Error error

	// SamplesCollected 已采集样本数
	SamplesCollected uint64

	// LastSampleTime 最后采样时间
	LastSampleTime time.Time
}

// DataSourceConfig 数据源配置接口
type DataSourceConfig interface {
	// Validate 验证配置
	Validate() error

	// GetType 获取数据源类型
	GetType() string
}
