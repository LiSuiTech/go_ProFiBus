-- ============================================
-- Device Multi-Data Fields Tables
-- ============================================

-- 创建设备数据字段配置表
CREATE TABLE IF NOT EXISTS device_data_fields (
    id VARCHAR(64) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    field_name VARCHAR(255) NOT NULL, -- 字段名: temperature, vibration, speed等
    field_type VARCHAR(50) NOT NULL, -- 数据类型: float, int, string, bool
    unit VARCHAR(50), -- 单位: °C, Hz, rpm等
    min_value DOUBLE PRECISION, -- 最小值
    max_value DOUBLE PRECISION, -- 最大值
    default_value DOUBLE PRECISION, -- 默认值
    description TEXT, -- 字段描述
    enabled BOOLEAN NOT NULL DEFAULT true, -- 是否启用
    fusion_weight DOUBLE PRECISION DEFAULT 1.0, -- 融合权重
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, field_name) -- 同一设备的字段名唯一
);

-- 创建设备多数据源配置表
CREATE TABLE IF NOT EXISTS device_data_sources (
    id VARCHAR(64) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    source_name VARCHAR(255) NOT NULL, -- 数据源名称: temperature_sensor, vibration_sensor等
    source_type VARCHAR(50) NOT NULL, -- 数据源类型: sensor, channel, calculated等
    channel_id VARCHAR(64) REFERENCES channels(id) ON DELETE SET NULL, -- 关联的采集通道
    field_mapping JSONB NOT NULL DEFAULT '{}', -- 字段映射: {"temperature": "temp", "vibration": "vib"}
    fusion_enabled BOOLEAN NOT NULL DEFAULT true, -- 是否参与融合
    fusion_weight DOUBLE PRECISION DEFAULT 1.0, -- 融合权重
    sample_rate INTEGER DEFAULT 1000, -- 采样率(ms)
    enabled BOOLEAN NOT NULL DEFAULT true, -- 是否启用
    metadata JSONB NOT NULL DEFAULT '{}', -- 元数据
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, source_name) -- 同一设备的数据源名称唯一
);

-- 创建设备融合配置表
CREATE TABLE IF NOT EXISTS device_fusion_configs (
    id VARCHAR(64) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    fusion_strategy VARCHAR(50) NOT NULL DEFAULT 'weighted', -- 融合策略
    time_window_ms INTEGER DEFAULT 1000, -- 时间窗口(毫秒)
    min_sources INTEGER DEFAULT 1, -- 最小数据源数量
    field_weights JSONB NOT NULL DEFAULT '{}', -- 字段权重: {"temperature": 0.4, "vibration": 0.3, "speed": 0.3}
    source_weights JSONB NOT NULL DEFAULT '{}', -- 数据源权重
    enabled BOOLEAN NOT NULL DEFAULT true, -- 是否启用融合
    metadata JSONB NOT NULL DEFAULT '{}', -- 元数据
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id) -- 每个设备只有一个融合配置
);

-- 创建设备融合结果表
CREATE TABLE IF NOT EXISTS device_fused_data (
    id VARCHAR(64) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    fused_data JSONB NOT NULL, -- 融合后的数据: {"temperature": 75.5, "vibration": 2.3, "speed": 1500}
    source_count INTEGER DEFAULT 0, -- 参与融合的数据源数量
    fusion_strategy VARCHAR(50), -- 使用的融合策略
    quality_score DOUBLE PRECISION DEFAULT 1.0, -- 数据质量评分
    metadata JSONB NOT NULL DEFAULT '{}', -- 元数据
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_device_data_fields_device_id ON device_data_fields(device_id);
CREATE INDEX IF NOT EXISTS idx_device_data_fields_enabled ON device_data_fields(enabled);
CREATE INDEX IF NOT EXISTS idx_device_data_sources_device_id ON device_data_sources(device_id);
CREATE INDEX IF NOT EXISTS idx_device_data_sources_enabled ON device_data_sources(enabled);
CREATE INDEX IF NOT EXISTS idx_device_fusion_configs_device_id ON device_fusion_configs(device_id);
CREATE INDEX IF NOT EXISTS idx_device_fused_data_device_id ON device_fused_data(device_id);
CREATE INDEX IF NOT EXISTS idx_device_fused_data_timestamp ON device_fused_data(timestamp DESC);

-- 添加注释
COMMENT ON TABLE device_data_fields IS '设备数据字段配置表';
COMMENT ON TABLE device_data_sources IS '设备多数据源配置表';
COMMENT ON TABLE device_fusion_configs IS '设备数据融合配置表';
COMMENT ON TABLE device_fused_data IS '设备融合数据结果表';
COMMENT ON COLUMN device_data_fields.field_type IS '数据类型: float, int, string, bool';
COMMENT ON COLUMN device_fusion_configs.fusion_strategy IS '融合策略: weighted, average, kalman等';
