package device

import (
	"context"
	"time"

	websocket "go_ProFiBus/internal/interfaces/websocket"
)

// DataStreamBridge 数据流桥接服务，将融合数据推送到WebSocket
type DataStreamBridge struct {
	dataHub      *websocket.DataHub
	fusionService *DataFusionService
	pollInterval  time.Duration
	devices      map[string]time.Time // 设备ID -> 最后检查时间
}

// NewDataStreamBridge 创建数据流桥接服务
func NewDataStreamBridge(dataHub *websocket.DataHub, fusionService *DataFusionService) *DataStreamBridge {
	return &DataStreamBridge{
		dataHub:      dataHub,
		fusionService: fusionService,
		pollInterval:  1 * time.Second, // 每秒轮询一次
		devices:      make(map[string]time.Time),
	}
}

// Start 启动数据流桥接服务
func (b *DataStreamBridge) Start(ctx context.Context) {
	ticker := time.NewTicker(b.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.checkAndBroadcast(ctx)
		}
	}
}

// checkAndBroadcast 检查并广播最新数据
func (b *DataStreamBridge) checkAndBroadcast(ctx context.Context) {
	// 获取所有设备的最新融合数据
	// 这里简化实现，实际应该从设备仓库获取所有设备列表
	// 为了演示，我们假设有设备ID列表
	// 实际实现中应该从设备仓库获取

	// 注意：这是一个简化的实现
	// 更好的方法是让融合服务在保存数据时直接调用推送方法
}

// BroadcastFusedData 广播融合数据（由融合服务调用）
func (b *DataStreamBridge) BroadcastFusedData(deviceID string, data map[string]interface{}, sourceID string, quality float64) {
	if b.dataHub == nil {
		return
	}

	event := websocket.DeviceDataEvent{
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Data:      data,
		SourceID:  sourceID,
		Quality:   quality,
	}

	b.dataHub.BroadcastDataEvent(event)
}
