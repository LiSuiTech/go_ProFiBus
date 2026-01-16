package tracing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go_ProFiBus/pkg/interfaces"
	"time"
)

// GenerateSampleID 生成数据样本ID
// 基于 sourceID + timestamp 生成唯一ID
func GenerateSampleID(sample interfaces.DataSample) string {
	h := sha256.New()
	h.Write([]byte(sample.GetSourceID()))
	h.Write([]byte(sample.GetTimestamp().Format(time.RFC3339Nano)))

	return hex.EncodeToString(h.Sum(nil))[:16]
}

// CreateTraceEvent 创建追踪事件的辅助函数
func CreateTraceEvent(
	pipelineID, sampleID string,
	componentType, componentID, componentName string,
	action string,
) *interfaces.TraceEvent {
	return &interfaces.TraceEvent{
		PipelineID:    pipelineID,
		SampleID:      sampleID,
		ComponentType: componentType,
		ComponentID:   componentID,
		ComponentName: componentName,
		Action:        action,
		Timestamp:     time.Now(),
		Status:        interfaces.StatusSuccess,
	}
}

// CreateErrorTraceEvent 创建错误追踪事件
func CreateErrorTraceEvent(
	pipelineID, sampleID string,
	componentType, componentID, componentName string,
	err error,
) *interfaces.TraceEvent {
	event := CreateTraceEvent(
		pipelineID, sampleID,
		componentType, componentID, componentName,
		interfaces.ActionError,
	)
	event.Status = interfaces.StatusError
	event.Error = err.Error()

	return event
}

// MeasureDuration 测量执行时间的辅助函数
func MeasureDuration(event *interfaces.TraceEvent, fn func() error) error {
	start := time.Now()
	err := fn()
	event.Duration = time.Since(start)

	if err != nil {
		event.Status = interfaces.StatusError
		event.Error = err.Error()
	}

	return err
}

// FormatDuration 格式化持续时间为人类可读格式
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// CalculateErrorRate 计算错误率
func CalculateErrorRate(total, errors int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(errors) / float64(total) * 100
}
