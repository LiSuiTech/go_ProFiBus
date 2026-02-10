package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go_ProFiBus/internal/domain/channel"
	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/storage"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ChannelRepositoryImpl 采集通道仓储实现
type ChannelRepositoryImpl struct {
	store *storage.PostgresStore
}

// NewChannelRepository 创建采集通道仓储
func NewChannelRepository(store *storage.PostgresStore) interfaces.ChannelRepository {
	return &ChannelRepositoryImpl{store: store}
}

// CreateChannel 创建通道
func (r *ChannelRepositoryImpl) CreateChannel(ctx context.Context, ch *channel.Channel) error {
	if ch.ID == "" {
		ch.ID = uuid.New().String()
	}
	ch.CreatedAt = time.Now()
	ch.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(ch.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO channels (
			id, name, description, protocol, device_name, device_port,
			status, config, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.store.GetPool().Exec(ctx, query,
		ch.ID, ch.Name, ch.Description, ch.Protocol, ch.DeviceName, ch.DevicePort,
		ch.Status, configJSON, ch.Enabled, ch.CreatedAt, ch.UpdatedAt,
	)

	return err
}

// GetChannel 获取通道
func (r *ChannelRepositoryImpl) GetChannel(ctx context.Context, id string) (*channel.Channel, error) {
	query := `
		SELECT id, name, description, protocol, device_name, device_port,
		       status, config, enabled, created_at, updated_at
		FROM channels WHERE id = $1
	`

	var ch channel.Channel
	var configJSON []byte

	err := r.store.GetPool().QueryRow(ctx, query, id).Scan(
		&ch.ID, &ch.Name, &ch.Description, &ch.Protocol, &ch.DeviceName, &ch.DevicePort,
		&ch.Status, &configJSON, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("channel not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(configJSON, &ch.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 获取关联的点位
	points, err := r.GetPointsByChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	ch.Points = make([]channel.Point, len(points))
	for i, p := range points {
		ch.Points[i] = *p
	}

	return &ch, nil
}

// GetChannels 获取所有通道
func (r *ChannelRepositoryImpl) GetChannels(ctx context.Context) ([]*channel.Channel, error) {
	query := `
		SELECT id, name, description, protocol, device_name, device_port,
		       status, config, enabled, created_at, updated_at
		FROM channels ORDER BY created_at DESC
	`

	rows, err := r.store.GetPool().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*channel.Channel
	for rows.Next() {
		var ch channel.Channel
		var configJSON []byte

		err := rows.Scan(
			&ch.ID, &ch.Name, &ch.Description, &ch.Protocol, &ch.DeviceName, &ch.DevicePort,
			&ch.Status, &configJSON, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(configJSON, &ch.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		channels = append(channels, &ch)
	}

	// 批量获取每个通道的点位
	for _, ch := range channels {
		points, err := r.GetPointsByChannel(ctx, ch.ID)
		if err != nil {
			return nil, err
		}
		ch.Points = make([]channel.Point, len(points))
		for i, p := range points {
			ch.Points[i] = *p
		}
	}

	return channels, nil
}

// UpdateChannel 更新通道
func (r *ChannelRepositoryImpl) UpdateChannel(ctx context.Context, ch *channel.Channel) error {
	ch.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(ch.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE channels SET
			name = $2, description = $3, protocol = $4, device_name = $5, device_port = $6,
			status = $7, config = $8, enabled = $9, updated_at = $10
		WHERE id = $1
	`

	tag, err := r.store.GetPool().Exec(ctx, query,
		ch.ID, ch.Name, ch.Description, ch.Protocol, ch.DeviceName, ch.DevicePort,
		ch.Status, configJSON, ch.Enabled, ch.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("channel not found: %s", ch.ID)
	}

	return nil
}

// DeleteChannel 删除通道
func (r *ChannelRepositoryImpl) DeleteChannel(ctx context.Context, id string) error {
	// 先删除关联的点位
	_, err := r.store.GetPool().Exec(ctx, "DELETE FROM channel_points WHERE channel_id = $1", id)
	if err != nil {
		return err
	}

	// 再删除通道
	tag, err := r.store.GetPool().Exec(ctx, "DELETE FROM channels WHERE id = $1", id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("channel not found: %s", id)
	}

	return nil
}

// UpdateChannelStatus 更新通道状态
func (r *ChannelRepositoryImpl) UpdateChannelStatus(ctx context.Context, id string, status channel.ChannelStatus) error {
	query := "UPDATE channels SET status = $2, updated_at = $3 WHERE id = $1"

	tag, err := r.store.GetPool().Exec(ctx, query, id, status, time.Now())
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("channel not found: %s", id)
	}

	return nil
}

// CreatePoint 创建点位
func (r *ChannelRepositoryImpl) CreatePoint(ctx context.Context, point *channel.Point) error {
	if point.ID == "" {
		point.ID = uuid.New().String()
	}
	point.CreatedAt = time.Now()
	point.UpdatedAt = time.Now()

	query := `
		INSERT INTO channel_points (
			id, channel_id, name, description, address, data_type,
			unit, scale, offset, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.store.GetPool().Exec(ctx, query,
		point.ID, point.ChannelID, point.Name, point.Description, point.Address,
		point.DataType, point.Unit, point.Scale, point.Offset,
		point.Enabled, point.CreatedAt, point.UpdatedAt,
	)

	return err
}

// GetPoint 获取点位
func (r *ChannelRepositoryImpl) GetPoint(ctx context.Context, id string) (*channel.Point, error) {
	query := `
		SELECT id, channel_id, name, description, address, data_type,
		       unit, scale, offset, enabled, created_at, updated_at
		FROM channel_points WHERE id = $1
	`

	var point channel.Point
	err := r.store.GetPool().QueryRow(ctx, query, id).Scan(
		&point.ID, &point.ChannelID, &point.Name, &point.Description, &point.Address,
		&point.DataType, &point.Unit, &point.Scale, &point.Offset,
		&point.Enabled, &point.CreatedAt, &point.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("point not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	return &point, nil
}

// GetPointsByChannel 获取通道的所有点位
func (r *ChannelRepositoryImpl) GetPointsByChannel(ctx context.Context, channelID string) ([]*channel.Point, error) {
	query := `
		SELECT id, channel_id, name, description, address, data_type,
		       unit, scale, offset, enabled, created_at, updated_at
		FROM channel_points WHERE channel_id = $1 ORDER BY created_at
	`

	rows, err := r.store.GetPool().Query(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []*channel.Point
	for rows.Next() {
		var point channel.Point
		err := rows.Scan(
			&point.ID, &point.ChannelID, &point.Name, &point.Description, &point.Address,
			&point.DataType, &point.Unit, &point.Scale, &point.Offset,
			&point.Enabled, &point.CreatedAt, &point.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		points = append(points, &point)
	}

	return points, nil
}

// UpdatePoint 更新点位
func (r *ChannelRepositoryImpl) UpdatePoint(ctx context.Context, point *channel.Point) error {
	point.UpdatedAt = time.Now()

	query := `
		UPDATE channel_points SET
			name = $2, description = $3, address = $4, data_type = $5,
			unit = $6, scale = $7, offset = $8, enabled = $9, updated_at = $10
		WHERE id = $1
	`

	tag, err := r.store.GetPool().Exec(ctx, query,
		point.ID, point.Name, point.Description, point.Address, point.DataType,
		point.Unit, point.Scale, point.Offset, point.Enabled, point.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("point not found: %s", point.ID)
	}

	return nil
}

// DeletePoint 删除点位
func (r *ChannelRepositoryImpl) DeletePoint(ctx context.Context, id string) error {
	tag, err := r.store.GetPool().Exec(ctx, "DELETE FROM channel_points WHERE id = $1", id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("point not found: %s", id)
	}

	return nil
}
