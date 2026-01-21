package interfaces

import (
	"context"
	"go_ProFiBus/internal/domain/channel"
)

// ChannelRepository 采集通道仓储接口
type ChannelRepository interface {
	// Channel 操作
	CreateChannel(ctx context.Context, ch *channel.Channel) error
	GetChannel(ctx context.Context, id string) (*channel.Channel, error)
	GetChannels(ctx context.Context) ([]*channel.Channel, error)
	UpdateChannel(ctx context.Context, ch *channel.Channel) error
	DeleteChannel(ctx context.Context, id string) error
	UpdateChannelStatus(ctx context.Context, id string, status channel.ChannelStatus) error

	// Point 操作
	CreatePoint(ctx context.Context, point *channel.Point) error
	GetPoint(ctx context.Context, id string) (*channel.Point, error)
	GetPointsByChannel(ctx context.Context, channelID string) ([]*channel.Point, error)
	UpdatePoint(ctx context.Context, point *channel.Point) error
	DeletePoint(ctx context.Context, id string) error
}
