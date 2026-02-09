package device

import "time"

// DataField 设备数据字段
type DataField struct {
	ID          string
	DeviceID    string
	FieldName   string
	FieldType   FieldType
	Unit        string
	MinValue    *float64
	MaxValue    *float64
	DefaultValue *float64
	Description string
	Enabled     bool
	FusionWeight float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// FieldType 字段类型
type FieldType string

const (
	FieldTypeFloat  FieldType = "float"
	FieldTypeInt    FieldType = "int"
	FieldTypeString FieldType = "string"
	FieldTypeBool   FieldType = "bool"
)

// NewDataField 创建新数据字段
func NewDataField(id, deviceID, fieldName string, fieldType FieldType) *DataField {
	return &DataField{
		ID:          id,
		DeviceID:    deviceID,
		FieldName:   fieldName,
		FieldType:   fieldType,
		Enabled:     true,
		FusionWeight: 1.0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SetUnit 设置单位
func (df *DataField) SetUnit(unit string) {
	df.Unit = unit
	df.UpdatedAt = time.Now()
}

// SetRange 设置取值范围
func (df *DataField) SetRange(min, max *float64) {
	df.MinValue = min
	df.MaxValue = max
	df.UpdatedAt = time.Now()
}

// SetFusionWeight 设置融合权重
func (df *DataField) SetFusionWeight(weight float64) {
	if weight < 0 {
		weight = 0
	}
	if weight > 1 {
		weight = 1
	}
	df.FusionWeight = weight
	df.UpdatedAt = time.Now()
}

// DataSource 设备数据源
type DataSource struct {
	ID            string
	DeviceID      string
	SourceName    string
	SourceType    SourceType
	ChannelID     string
	FieldMapping  map[string]string // 字段映射: {"temperature": "temp", "vibration": "vib"}
	FusionEnabled bool
	FusionWeight  float64
	SampleRate    int // 采样率(ms)
	Enabled       bool
	Metadata      map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SourceType 数据源类型
type SourceType string

const (
	SourceTypeSensor     SourceType = "sensor"
	SourceTypeChannel    SourceType = "channel"
	SourceTypeCalculated SourceType = "calculated"
	SourceTypeExternal   SourceType = "external"
)

// NewDataSource 创建新数据源
func NewDataSource(id, deviceID, sourceName string, sourceType SourceType) *DataSource {
	return &DataSource{
		ID:            id,
		DeviceID:      deviceID,
		SourceName:    sourceName,
		SourceType:    sourceType,
		FieldMapping:  make(map[string]string),
		FusionEnabled: true,
		FusionWeight:  1.0,
		SampleRate:    1000,
		Enabled:       true,
		Metadata:      make(map[string]interface{}),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// SetFieldMapping 设置字段映射
func (ds *DataSource) SetFieldMapping(mapping map[string]string) {
	ds.FieldMapping = mapping
	ds.UpdatedAt = time.Now()
}

// SetFusionWeight 设置融合权重
func (ds *DataSource) SetFusionWeight(weight float64) {
	if weight < 0 {
		weight = 0
	}
	ds.FusionWeight = weight
	ds.UpdatedAt = time.Now()
}

// FusionConfig 设备融合配置
type FusionConfig struct {
	ID            string
	DeviceID      string
	FusionStrategy string // 融合策略: weighted, average, kalman等
	TimeWindowMs  int     // 时间窗口(毫秒)
	MinSources    int     // 最小数据源数量
	FieldWeights  map[string]float64 // 字段权重
	SourceWeights map[string]float64 // 数据源权重
	Enabled       bool
	Metadata      map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewFusionConfig 创建融合配置
func NewFusionConfig(id, deviceID string) *FusionConfig {
	return &FusionConfig{
		ID:            id,
		DeviceID:      deviceID,
		FusionStrategy: "weighted",
		TimeWindowMs:  1000,
		MinSources:    1,
		FieldWeights:  make(map[string]float64),
		SourceWeights: make(map[string]float64),
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

// SetFieldWeight 设置字段权重
func (fc *FusionConfig) SetFieldWeight(fieldName string, weight float64) {
	if fc.FieldWeights == nil {
		fc.FieldWeights = make(map[string]float64)
	}
	fc.FieldWeights[fieldName] = weight
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

// FusedData 融合数据结果
type FusedData struct {
	ID            string
	DeviceID      string
	Timestamp     time.Time
	FusedData     map[string]interface{} // 融合后的数据
	SourceCount   int
	FusionStrategy string
	QualityScore  float64
	Metadata      map[string]interface{}
	CreatedAt     time.Time
}

// NewFusedData 创建融合数据
func NewFusedData(id, deviceID string, fusedData map[string]interface{}) *FusedData {
	return &FusedData{
		ID:          id,
		DeviceID:    deviceID,
		Timestamp:   time.Now(),
		FusedData:   fusedData,
		SourceCount: 0,
		QualityScore: 1.0,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
	}
}

// SetQualityScore 设置质量评分
func (fd *FusedData) SetQualityScore(score float64) {
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	fd.QualityScore = score
}
