package storage

import (
	"context"
	"encoding/json"
	"fmt"
	deviceDomain "go_ProFiBus/internal/domain/device"
	"go_ProFiBus/pkg/interfaces"
	"time"

	"github.com/jackc/pgx/v5"
)

// DeviceRepositoryImpl 设备仓储实现
type DeviceRepositoryImpl struct {
	store *PostgresStore
}

// NewDeviceRepository 创建设备仓储
func NewDeviceRepository(store *PostgresStore) interfaces.DeviceRepository {
	return &DeviceRepositoryImpl{store: store}
}

// Create 创建设备
func (r *DeviceRepositoryImpl) Create(ctx context.Context, device *deviceDomain.Device) error {
	metadataJSON, err := json.Marshal(device.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO devices (id, name, description, type, status, health_score, location_x, location_y, location_z, area, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.store.Exec(query,
		device.ID,
		device.Name,
		device.Description,
		string(device.Type),
		string(device.Status),
		device.HealthScore,
		device.Location.X,
		device.Location.Y,
		device.Location.Z,
		device.Area,
		metadataJSON,
		device.CreatedAt,
		device.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建设备失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取设备
func (r *DeviceRepositoryImpl) GetByID(ctx context.Context, id string) (*deviceDomain.Device, error) {
	query := `
		SELECT id, name, description, type, status, health_score, location_x, location_y, location_z, area, metadata, created_at, updated_at
		FROM devices
		WHERE id = $1
	`

	var device deviceDomain.Device
	var metadataJSON []byte
	var deviceType, status string

	err := r.store.QueryRow(query, id).Scan(
		&device.ID,
		&device.Name,
		&device.Description,
		&deviceType,
		&status,
		&device.HealthScore,
		&device.Location.X,
		&device.Location.Y,
		&device.Location.Z,
		&device.Area,
		&metadataJSON,
		&device.CreatedAt,
		&device.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("设备不存在: %s", id)
		}
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	device.Type = deviceDomain.DeviceType(deviceType)
	device.Status = deviceDomain.DeviceStatus(status)

	if err := json.Unmarshal(metadataJSON, &device.Metadata); err != nil {
		r.store.log.Warn("反序列化元数据失败: %v", err)
		device.Metadata = make(map[string]interface{})
	}

	return &device, nil
}

// List 列出设备
func (r *DeviceRepositoryImpl) List(ctx context.Context, filters interfaces.DeviceFilters) ([]*deviceDomain.Device, error) {
	query := `
		SELECT id, name, description, type, status, health_score, location_x, location_y, location_z, area, metadata, created_at, updated_at
		FROM devices
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, string(*filters.Type))
		argIndex++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
		argIndex++
	}

	if filters.Area != nil {
		query += fmt.Sprintf(" AND area = $%d", argIndex)
		args = append(args, *filters.Area)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询设备列表失败: %w", err)
	}
	defer rows.Close()

	devices := make([]*deviceDomain.Device, 0)

	for rows.Next() {
		var device deviceDomain.Device
		var metadataJSON []byte
		var deviceType, status string

		err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.Description,
			&deviceType,
			&status,
			&device.HealthScore,
			&device.Location.X,
			&device.Location.Y,
			&device.Location.Z,
			&device.Area,
			&metadataJSON,
			&device.CreatedAt,
			&device.UpdatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描设备失败: %v", err)
			continue
		}

		device.Type = deviceDomain.DeviceType(deviceType)
		device.Status = deviceDomain.DeviceStatus(status)

		if err := json.Unmarshal(metadataJSON, &device.Metadata); err != nil {
			r.store.log.Warn("反序列化元数据失败: %v", err)
			device.Metadata = make(map[string]interface{})
		}

		devices = append(devices, &device)
	}

	return devices, nil
}

// Update 更新设备
func (r *DeviceRepositoryImpl) Update(ctx context.Context, device *deviceDomain.Device) error {
	metadataJSON, err := json.Marshal(device.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE devices
		SET name = $1, description = $2, type = $3, status = $4, health_score = $5,
		    location_x = $6, location_y = $7, location_z = $8, area = $9, metadata = $10, updated_at = $11
		WHERE id = $12
	`

	tag, err := r.store.Exec(query,
		device.Name,
		device.Description,
		string(device.Type),
		string(device.Status),
		device.HealthScore,
		device.Location.X,
		device.Location.Y,
		device.Location.Z,
		device.Area,
		metadataJSON,
		time.Now(),
		device.ID,
	)

	if err != nil {
		return fmt.Errorf("更新设备失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("设备不存在: %s", device.ID)
	}

	return nil
}

// Delete 删除设备
func (r *DeviceRepositoryImpl) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM devices WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除设备失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("设备不存在: %s", id)
	}

	return nil
}

// UpdateStatus 更新设备状态
func (r *DeviceRepositoryImpl) UpdateStatus(ctx context.Context, id string, status deviceDomain.DeviceStatus) error {
	query := `UPDATE devices SET status = $1, updated_at = $2 WHERE id = $3`

	tag, err := r.store.Exec(query, string(status), time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新设备状态失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("设备不存在: %s", id)
	}

	return nil
}

// UpdateHealthScore 更新健康度评分
func (r *DeviceRepositoryImpl) UpdateHealthScore(ctx context.Context, id string, score float64) error {
	query := `UPDATE devices SET health_score = $1, updated_at = $2 WHERE id = $3`

	tag, err := r.store.Exec(query, score, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新健康度评分失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("设备不存在: %s", id)
	}

	return nil
}

// GetByChannelID 根据通道ID获取关联的设备列表
func (r *DeviceRepositoryImpl) GetByChannelID(ctx context.Context, channelID string) ([]*deviceDomain.Device, error) {
	query := `
		SELECT d.id, d.name, d.description, d.type, d.status, d.health_score, 
		       d.location_x, d.location_y, d.location_z, d.area, d.metadata, d.created_at, d.updated_at
		FROM devices d
		INNER JOIN device_channels dc ON d.id = dc.device_id
		WHERE dc.channel_id = $1 AND dc.enabled = true
	`

	rows, err := r.store.Query(query, channelID)
	if err != nil {
		return nil, fmt.Errorf("查询设备列表失败: %w", err)
	}
	defer rows.Close()

	devices := make([]*deviceDomain.Device, 0)

	for rows.Next() {
		var device deviceDomain.Device
		var metadataJSON []byte
		var deviceType, status string

		err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.Description,
			&deviceType,
			&status,
			&device.HealthScore,
			&device.Location.X,
			&device.Location.Y,
			&device.Location.Z,
			&device.Area,
			&metadataJSON,
			&device.CreatedAt,
			&device.UpdatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描设备失败: %v", err)
			continue
		}

		device.Type = deviceDomain.DeviceType(deviceType)
		device.Status = deviceDomain.DeviceStatus(status)

		if err := json.Unmarshal(metadataJSON, &device.Metadata); err != nil {
			r.store.log.Warn("反序列化元数据失败: %v", err)
			device.Metadata = make(map[string]interface{})
		}

		devices = append(devices, &device)
	}

	return devices, nil
}

// AddChannel 添加设备与通道的关联
func (r *DeviceRepositoryImpl) AddChannel(ctx context.Context, deviceID, channelID string) error {
	query := `
		INSERT INTO device_channels (id, device_id, channel_id, enabled, created_at)
		VALUES (gen_random_uuid()::text, $1, $2, true, CURRENT_TIMESTAMP)
		ON CONFLICT (device_id, channel_id) DO UPDATE SET enabled = true
	`

	_, err := r.store.Exec(query, deviceID, channelID)
	if err != nil {
		return fmt.Errorf("添加设备通道关联失败: %w", err)
	}

	return nil
}

// RemoveChannel 移除设备与通道的关联
func (r *DeviceRepositoryImpl) RemoveChannel(ctx context.Context, deviceID, channelID string) error {
	query := `UPDATE device_channels SET enabled = false WHERE device_id = $1 AND channel_id = $2`

	_, err := r.store.Exec(query, deviceID, channelID)
	if err != nil {
		return fmt.Errorf("移除设备通道关联失败: %w", err)
	}

	return nil
}

// GetChannels 获取设备关联的通道ID列表
func (r *DeviceRepositoryImpl) GetChannels(ctx context.Context, deviceID string) ([]string, error) {
	query := `SELECT channel_id FROM device_channels WHERE device_id = $1 AND enabled = true`

	rows, err := r.store.Query(query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("查询通道列表失败: %w", err)
	}
	defer rows.Close()

	channelIDs := make([]string, 0)
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			r.store.log.Warn("扫描通道ID失败: %v", err)
			continue
		}
		channelIDs = append(channelIDs, channelID)
	}

	return channelIDs, nil
}
