-- ============================================
-- Devices Management Tables
-- ============================================

-- 创建设备表
CREATE TABLE IF NOT EXISTS devices (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- PLC, Sensor, Instrument, SmartDevice
    status VARCHAR(20) NOT NULL DEFAULT 'offline', -- online, offline, fault, maintenance
    health_score DOUBLE PRECISION DEFAULT 100.0, -- 健康度评分 0-100
    location_x DOUBLE PRECISION, -- X坐标
    location_y DOUBLE PRECISION, -- Y坐标
    location_z DOUBLE PRECISION, -- Z坐标（可选）
    area VARCHAR(255), -- 所属区域/车间
    metadata JSONB NOT NULL DEFAULT '{}', -- 设备元数据（厂商、型号、序列号等）
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建设备与通道关联表
CREATE TABLE IF NOT EXISTS device_channels (
    id VARCHAR(64) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, channel_id)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_devices_type ON devices(type);
CREATE INDEX IF NOT EXISTS idx_devices_area ON devices(area);
CREATE INDEX IF NOT EXISTS idx_device_channels_device_id ON device_channels(device_id);
CREATE INDEX IF NOT EXISTS idx_device_channels_channel_id ON device_channels(channel_id);
CREATE INDEX IF NOT EXISTS idx_device_channels_enabled ON device_channels(enabled);

-- 添加注释
COMMENT ON TABLE devices IS '设备管理表';
COMMENT ON TABLE device_channels IS '设备与采集通道关联表';
COMMENT ON COLUMN devices.type IS '设备类型: PLC, Sensor, Instrument, SmartDevice';
COMMENT ON COLUMN devices.status IS '设备状态: online, offline, fault, maintenance';
COMMENT ON COLUMN devices.health_score IS '设备健康度评分 0-100';
COMMENT ON COLUMN devices.metadata IS '设备元数据JSON，包含厂商、型号、序列号等信息';
