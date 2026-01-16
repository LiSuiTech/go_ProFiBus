package collector

import (
	"context"
	"go_ProFiBus/collector"
	"go_ProFiBus/internal/domain/datasample"
	"go_ProFiBus/pkg/interfaces"
	"time"
)

// DataSourceAdapter 将现有的 Collector 适配为 DataSource 接口
type DataSourceAdapter struct {
	collector  *collector.Collector
	sourceID   string
	name       string
	dataChan   chan interfaces.DataSample
	stopChan   chan struct{}
	running    bool
}

// NewDataSourceAdapter 创建数据源适配器
func NewDataSourceAdapter(sourceID, name string, coll *collector.Collector) *DataSourceAdapter {
	return &DataSourceAdapter{
		collector: coll,
		sourceID:  sourceID,
		name:      name,
		dataChan:  make(chan interfaces.DataSample, 1000),
		stopChan:  make(chan struct{}),
		running:   false,
	}
}

// Start 实现 DataSource 接口
func (a *DataSourceAdapter) Start(ctx context.Context) error {
	if a.running {
		return nil
	}

	// 启动底层采集器
	if err := a.collector.Start(); err != nil {
		return err
	}

	a.running = true

	// 启动数据转换协程
	go a.convertLoop(ctx)

	return nil
}

// Stop 实现 DataSource 接口
func (a *DataSourceAdapter) Stop() error {
	if !a.running {
		return nil
	}

	a.running = false
	close(a.stopChan)

	// 停止底层采集器
	return a.collector.Stop()
}

// GetData 实现 DataSource 接口
func (a *DataSourceAdapter) GetData() <-chan interfaces.DataSample {
	return a.dataChan
}

// GetStatus 实现 DataSource 接口
func (a *DataSourceAdapter) GetStatus() interfaces.SourceStatus {
	stats := a.collector.GetStats()

	var err error
	if !a.collector.IsRunning() {
		err = nil // 可以根据需要设置错误
	}

	return interfaces.SourceStatus{
		Running:          a.running,
		Connected:        a.collector.IsRunning(),
		Error:            err,
		SamplesCollected: stats.SuccessSamples,
		LastSampleTime:   stats.LastSampleTime,
	}
}

// GetID 实现 DataSource 接口
func (a *DataSourceAdapter) GetID() string {
	return a.sourceID
}

// GetName 实现 DataSource 接口
func (a *DataSourceAdapter) GetName() string {
	return a.name
}

// convertLoop 转换循环，将旧的 DataSample 转换为新的接口实现
func (a *DataSourceAdapter) convertLoop(ctx context.Context) {
	oldDataChan := a.collector.GetDataChannel()

	for {
		select {
		case <-ctx.Done():
			close(a.dataChan)
			return
		case <-a.stopChan:
			close(a.dataChan)
			return
		case oldSample, ok := <-oldDataChan:
			if !ok {
				close(a.dataChan)
				return
			}

			// 转换旧的 DataSample 为新的实现
			newSample := convertOldSampleToNew(oldSample)

			select {
			case a.dataChan <- newSample:
			case <-ctx.Done():
				close(a.dataChan)
				return
			case <-a.stopChan:
				close(a.dataChan)
				return
			}
		}
	}
}

// convertOldSampleToNew 将旧的 collector.DataSample 转换为新的 domain.DataSample
func convertOldSampleToNew(old *collector.DataSample) interfaces.DataSample {
	// 创建新的 DataSample
	sample := datasample.NewDataSampleWithTime(
		old.SourceID,
		old.Timestamp,
		old.ParsedData,
	)

	// 设置质量
	sample.SetQuality(old.Quality)

	// 添加原始协议信息到元数据
	sample.SetMetadata("protocol", old.Protocol)
	if len(old.RawData) > 0 {
		sample.SetMetadata("raw_data_length", len(old.RawData))
	}

	return sample
}

// 确保 DataSourceAdapter 实现了 interfaces.DataSource 接口
var _ interfaces.DataSource = (*DataSourceAdapter)(nil)
