-- ============================================
-- Channels and Points Tables
-- ============================================

-- 创建 channels 表
CREATE TABLE IF NOT EXISTS channels (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    protocol VARCHAR(50) NOT NULL,
    device_name VARCHAR(255) NOT NULL,
    device_port VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'stopped',
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建 channel_points 表
CREATE TABLE IF NOT EXISTS channel_points (
    id VARCHAR(64) PRIMARY KEY,
    channel_id VARCHAR(64) NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    address VARCHAR(255) NOT NULL,
    data_type VARCHAR(20) NOT NULL,
    unit VARCHAR(50),
    scale DOUBLE PRECISION DEFAULT 1.0,
    "offset" DOUBLE PRECISION DEFAULT 0.0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_channels_status ON channels(status);
CREATE INDEX IF NOT EXISTS idx_channels_protocol ON channels(protocol);
CREATE INDEX IF NOT EXISTS idx_channels_enabled ON channels(enabled);
CREATE INDEX IF NOT EXISTS idx_channel_points_channel_id ON channel_points(channel_id);
CREATE INDEX IF NOT EXISTS idx_channel_points_enabled ON channel_points(enabled);

-- 添加注释
COMMENT ON TABLE channels IS '采集通道配置表';
COMMENT ON TABLE channel_points IS '采集点位配置表';
COMMENT ON COLUMN channels.protocol IS '协议类型: UART, CAN, USB, Modbus, RS-485, RS-232, I2C, SPI, 1-Wire';
COMMENT ON COLUMN channels.status IS '通道状态: stopped, running, error';
COMMENT ON COLUMN channels.config IS '协议特定配置参数(JSON格式)';
COMMENT ON COLUMN channel_points.data_type IS '数据类型: int, float, bool, string, bytes';
COMMENT ON COLUMN channel_points.address IS '点位地址（协议相关，如Modbus寄存器地址）';
COMMENT ON COLUMN channel_points.scale IS '缩放系数: value = raw_value * scale + offset';
