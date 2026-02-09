package channel

import (
	"context"
	"fmt"
	channelDomain "go_ProFiBus/internal/domain/channel"
	"go_ProFiBus/pkg/interfaces"
	"sync"
)

// Manager 通道管理器
// 负责管理通道的生命周期和数据源
type Manager struct {
	repo       interfaces.ChannelRepository
	sources    map[string]interfaces.DataSource
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewManager 创建通道管理器
func NewManager(repo interfaces.ChannelRepository) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		repo:    repo,
		sources: make(map[string]interfaces.DataSource),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// StartChannel 启动通道
func (cm *Manager) StartChannel(ctx context.Context, channelID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查是否已经启动
	if _, exists := cm.sources[channelID]; exists {
		return fmt.Errorf("通道 %s 已经启动", channelID)
	}

	// 从仓库获取通道配置
	ch, err := cm.repo.GetChannel(ctx, channelID)
	if err != nil {
		return fmt.Errorf("获取通道配置失败: %w", err)
	}

	// 创建数据源（这里需要根据协议类型创建对应的数据源）
	// 目前先创建一个占位实现，后续可以根据协议类型创建具体的数据源
	source, err := cm.createDataSource(ch)
	if err != nil {
		return fmt.Errorf("创建数据源失败: %w", err)
	}

	// 启动数据源
	if err := source.Start(ctx); err != nil {
		return fmt.Errorf("启动数据源失败: %w", err)
	}

	// 保存数据源
	cm.sources[channelID] = source

	// 更新通道状态
	if err := cm.repo.UpdateChannelStatus(ctx, channelID, channelDomain.StatusRunning); err != nil {
		// 如果更新状态失败，停止数据源
		source.Stop()
		delete(cm.sources, channelID)
		return fmt.Errorf("更新通道状态失败: %w", err)
	}

	return nil
}

// StopChannel 停止通道
func (cm *Manager) StopChannel(ctx context.Context, channelID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 获取数据源
	source, exists := cm.sources[channelID]
	if !exists {
		return fmt.Errorf("通道 %s 未启动", channelID)
	}

	// 停止数据源
	if err := source.Stop(); err != nil {
		return fmt.Errorf("停止数据源失败: %w", err)
	}

	// 删除数据源
	delete(cm.sources, channelID)

	// 更新通道状态
	if err := cm.repo.UpdateChannelStatus(ctx, channelID, channelDomain.StatusStopped); err != nil {
		return fmt.Errorf("更新通道状态失败: %w", err)
	}

	return nil
}

// GetChannelStatus 获取通道状态
func (cm *Manager) GetChannelStatus(channelID string) (*interfaces.SourceStatus, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	source, exists := cm.sources[channelID]
	if !exists {
		return nil, fmt.Errorf("通道 %s 未启动", channelID)
	}

	status := source.GetStatus()
	return &status, nil
}

// IsChannelRunning 检查通道是否正在运行
func (cm *Manager) IsChannelRunning(channelID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, exists := cm.sources[channelID]
	return exists
}

// StopAll 停止所有通道
func (cm *Manager) StopAll() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var lastErr error
	for channelID, source := range cm.sources {
		if err := source.Stop(); err != nil {
			lastErr = fmt.Errorf("停止通道 %s 失败: %w", channelID, err)
		}
	}

	cm.sources = make(map[string]interfaces.DataSource)
	cm.cancel()

	return lastErr
}

// createDataSource 根据通道配置创建数据源
// TODO: 根据不同的协议类型创建对应的数据源实现
func (cm *Manager) createDataSource(ch *channelDomain.Channel) (interfaces.DataSource, error) {
	// 这里需要根据协议类型创建对应的数据源
	// 目前返回一个占位实现
	// 后续可以集成 serial 包中的协议实现

	// 示例：根据协议类型创建数据源
	switch ch.Protocol {
	case channelDomain.ProtocolModbus:
		// TODO: 创建 Modbus 数据源
		return nil, fmt.Errorf("Modbus 数据源创建尚未实现")
	case channelDomain.ProtocolUART:
		// TODO: 创建 UART 数据源
		return nil, fmt.Errorf("UART 数据源创建尚未实现")
	case channelDomain.ProtocolRS485:
		// TODO: 创建 RS485 数据源
		return nil, fmt.Errorf("RS485 数据源创建尚未实现")
	default:
		return nil, fmt.Errorf("不支持的协议类型: %s", ch.Protocol)
	}
}
