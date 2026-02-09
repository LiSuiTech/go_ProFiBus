package fusion

import (
	"time"
)

// DataSource 通用数据源
type DataSource struct {
	ID          string
	SourceName  string
	SourceType  SourceType
	DeviceID    string // 可选：关联设备ID
	ChannelID   string // 可选：关联通道ID
	FieldName   string // 可选：字段名（当SourceType为DeviceField时）
	SourceConfig map[string]interface{} // 数据源配置（字段映射、计算公式等）
	FusionWeight float64
	Enabled     bool
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SourceType 数据源类型
type SourceType string

const (
	SourceTypeDeviceField SourceType = "device_field" // 设备字段
	SourceTypeDevice      SourceType = "device"       // 设备（所有字段）
	SourceTypeChannel     SourceType = "channel"       // 通道
	SourceTypeExternal    SourceType = "external"      // 外部数据源
	SourceTypeCalculated  SourceType = "calculated"    // 计算数据源
)

// NewDataSource 创建新数据源
func NewDataSource(id, sourceName string, sourceType SourceType) *DataSource {
	return &DataSource{
		ID:          id,
		SourceName:  sourceName,
		SourceType:  sourceType,
		SourceConfig: make(map[string]interface{}),
		FusionWeight: 1.0,
		Enabled:     true,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SetFusionWeight 设置融合权重
func (ds *DataSource) SetFusionWeight(weight float64) {
	if weight < 0 {
		weight = 0
	}
	ds.FusionWeight = weight
	ds.UpdatedAt = time.Now()
}

// FusionConfig 通用融合配置
type FusionConfig struct {
	ID            string
	Name          string
	Description   string
	FusionStrategy string
	TimeWindowMs  int
	MinSources    int
	SourceWeights map[string]float64 // 数据源权重
	FieldWeights  map[string]float64 // 字段权重（可选）
	OutputFields  []string           // 输出字段列表
	Enabled       bool
	Metadata      map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewFusionConfig 创建融合配置
func NewFusionConfig(id, name string) *FusionConfig {
	return &FusionConfig{
		ID:            id,
		Name:          name,
		FusionStrategy: "weighted",
		TimeWindowMs:  1000,
		MinSources:    1,
		SourceWeights: make(map[string]float64),
		FieldWeights:  make(map[string]float64),
		OutputFields:  make([]string, 0),
		Enabled:       true,
		Metadata:      make(map[string]interface{}),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// SetFusionStrategy 设置融合策略
func (fc *FusionConfig) SetFusionStrategy(strategy string) {
	fc.FusionStrategy = strategy
	fc.UpdatedAt = time.Now()
}

// SetSourceWeight 设置数据源权重
func (fc *FusionConfig) SetSourceWeight(sourceID string, weight float64) {
	if fc.SourceWeights == nil {
		fc.SourceWeights = make(map[string]float64)
	}
	fc.SourceWeights[sourceID] = weight
	fc.UpdatedAt = time.Now()
}

// SetFieldWeight 设置字段权重
func (fc *FusionConfig) SetFieldWeight(fieldName string, weight float64) {
	if fc.FieldWeights == nil {
		fc.FieldWeights = make(map[string]float64)
	}
	fc.FieldWeights[fieldName] = weight
	fc.UpdatedAt = time.Now()
}

// AddOutputField 添加输出字段
func (fc *FusionConfig) AddOutputField(fieldName string) {
	if fc.OutputFields == nil {
		fc.OutputFields = make([]string, 0)
	}
	for _, f := range fc.OutputFields {
		if f == fieldName {
			return // 已存在
		}
	}
	fc.OutputFields = append(fc.OutputFields, fieldName)
	fc.UpdatedAt = time.Now()
}

// ConfigSourceRelation 融合配置与数据源关联
type ConfigSourceRelation struct {
	FusionConfigID string
	SourceID       string
	Weight         float64 // 在此配置中的权重（覆盖数据源默认权重）
	Enabled        bool
	CreatedAt      time.Time
}

// FusionResult 融合结果
type FusionResult struct {
	ID              string
	FusionConfigID  string
	FusionConfigName string
	Timestamp       time.Time
	FusedData       map[string]interface{}
	SourceCount     int
	SourceIDs       []string
	FusionStrategy  string
	QualityScore    float64
	Metadata        map[string]interface{}
	CreatedAt       time.Time
}

// NewFusionResult 创建融合结果
func NewFusionResult(id, fusionConfigID, fusionConfigName string, fusedData map[string]interface{}) *FusionResult {
	return &FusionResult{
		ID:              id,
		FusionConfigID:  fusionConfigID,
		FusionConfigName: fusionConfigName,
		Timestamp:       time.Now(),
		FusedData:       fusedData,
		SourceCount:     0,
		SourceIDs:       make([]string, 0),
		QualityScore:    1.0,
		Metadata:        make(map[string]interface{}),
		CreatedAt:       time.Now(),
	}
}

// SetQualityScore 设置质量评分
func (fr *FusionResult) SetQualityScore(score float64) {
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	fr.QualityScore = score
}

// SourceDataCache 数据源数据缓存
type SourceDataCache struct {
	ID        string
	SourceID  string
	Timestamp time.Time
	Data      map[string]interface{}
	Quality   float64
	Metadata  map[string]interface{}
	CreatedAt time.Time
}

// NewSourceDataCache 创建数据缓存
func NewSourceDataCache(id, sourceID string, data map[string]interface{}, quality float64) *SourceDataCache {
	return &SourceDataCache{
		ID:        id,
		SourceID:  sourceID,
		Timestamp: time.Now(),
		Data:      data,
		Quality:   quality,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
}
