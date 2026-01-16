package datasample

import (
	"time"

	"github.com/LiSuiTech/go_ProFiBus/pkg/interfaces"
)

// DataSample 数据样本实现
type DataSample struct {
	timestamp  time.Time
	sourceID   string
	data       map[string]interface{}
	quality    float64
	metadata   map[string]interface{}
}

// NewDataSample 创建数据样本
func NewDataSample(sourceID string, data map[string]interface{}) *DataSample {
	return &DataSample{
		timestamp: time.Now(),
		sourceID:  sourceID,
		data:      data,
		quality:   1.0,
		metadata:  make(map[string]interface{}),
	}
}

// NewDataSampleWithTime 创建指定时间的数据样本
func NewDataSampleWithTime(sourceID string, timestamp time.Time, data map[string]interface{}) *DataSample {
	return &DataSample{
		timestamp: timestamp,
		sourceID:  sourceID,
		data:      data,
		quality:   1.0,
		metadata:  make(map[string]interface{}),
	}
}

// GetTimestamp 实现 DataSample 接口
func (s *DataSample) GetTimestamp() time.Time {
	return s.timestamp
}

// GetSourceID 实现 DataSample 接口
func (s *DataSample) GetSourceID() string {
	return s.sourceID
}

// GetData 实现 DataSample 接口
func (s *DataSample) GetData() map[string]interface{} {
	return s.data
}

// GetQuality 实现 DataSample 接口
func (s *DataSample) GetQuality() float64 {
	return s.quality
}

// GetMetadata 实现 DataSample 接口
func (s *DataSample) GetMetadata() map[string]interface{} {
	return s.metadata
}

// SetQuality 设置数据质量
func (s *DataSample) SetQuality(quality float64) {
	if quality < 0 {
		quality = 0
	}
	if quality > 1 {
		quality = 1
	}
	s.quality = quality
}

// SetMetadata 设置元数据
func (s *DataSample) SetMetadata(key string, value interface{}) {
	if s.metadata == nil {
		s.metadata = make(map[string]interface{})
	}
	s.metadata[key] = value
}

// GetValue 获取数据字段值
func (s *DataSample) GetValue(key string) (interface{}, bool) {
	val, ok := s.data[key]
	return val, ok
}

// SetValue 设置数据字段值
func (s *DataSample) SetValue(key string, value interface{}) {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}
	s.data[key] = value
}

// Clone 克隆数据样本
func (s *DataSample) Clone() interfaces.DataSample {
	dataCopy := make(map[string]interface{})
	for k, v := range s.data {
		dataCopy[k] = v
	}

	metadataCopy := make(map[string]interface{})
	for k, v := range s.metadata {
		metadataCopy[k] = v
	}

	return &DataSample{
		timestamp: s.timestamp,
		sourceID:  s.sourceID,
		data:      dataCopy,
		quality:   s.quality,
		metadata:  metadataCopy,
	}
}

// 确保 DataSample 实现了 interfaces.DataSample 接口
var _ interfaces.DataSample = (*DataSample)(nil)
