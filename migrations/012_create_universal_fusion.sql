-- ============================================
-- Universal Data Fusion Tables
-- ============================================

-- 创建通用数据源配置表（支持设备字段、设备、通道等）
CREATE TABLE IF NOT EXISTS fusion_data_sources (
    id VARCHAR(64) PRIMARY KEY,
    source_name VARCHAR(255) NOT NULL UNIQUE, -- 数据源名称
    source_type VARCHAR(50) NOT NULL, -- 类型: device_field, device, channel, external, calculated
    device_id VARCHAR(64) REFERENCES devices(id) ON DELETE CASCADE, -- 关联设备（可选）
    channel_id VARCHAR(64) REFERENCES channels(id) ON DELETE SET NULL, -- 关联通道（可选）
    field_name VARCHAR(255), -- 字段名（当source_type为device_field时使用）
    source_config JSONB NOT NULL DEFAULT '{}', -- 数据源配置（字段映射、计算公式等）
    fusion_weight DOUBLE PRECISION DEFAULT 1.0, -- 融合权重
    enabled BOOLEAN NOT NULL DEFAULT true, -- 是否启用
    metadata JSONB NOT NULL DEFAULT '{}', -- 元数据
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建通用融合配置表（不绑定到单一设备）
CREATE TABLE IF NOT EXISTS fusion_configs (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE, -- 融合配置名称
    description TEXT, -- 描述
    fusion_strategy VARCHAR(50) NOT NULL DEFAULT 'weighted', -- 融合策略: weighted, average, kalman等
    time_window_ms INTEGER DEFAULT 1000, -- 时间窗口(毫秒)
    min_sources INTEGER DEFAULT 1, -- 最小数据源数量
    source_weights JSONB NOT NULL DEFAULT '{}', -- 数据源权重: {"source_id": weight}
    field_weights JSONB NOT NULL DEFAULT '{}', -- 字段权重（可选）: {"field_name": weight}
    output_fields JSONB NOT NULL DEFAULT '[]', -- 输出字段列表
    enabled BOOLEAN NOT NULL DEFAULT true, -- 是否启用融合
    metadata JSONB NOT NULL DEFAULT '{}', -- 元数据
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建融合配置与数据源的关联表（多对多关系）
CREATE TABLE IF NOT EXISTS fusion_config_sources (
    fusion_config_id VARCHAR(64) NOT NULL REFERENCES fusion_configs(id) ON DELETE CASCADE,
    source_id VARCHAR(64) NOT NULL REFERENCES fusion_data_sources(id) ON DELETE CASCADE,
    weight DOUBLE PRECISION DEFAULT 1.0, -- 在此融合配置中的权重（覆盖数据源默认权重）
    enabled BOOLEAN NOT NULL DEFAULT true, -- 是否在此配置中启用
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (fusion_config_id, source_id)
);

-- 创建通用融合结果表
CREATE TABLE IF NOT EXISTS fusion_results (
    id VARCHAR(64) PRIMARY KEY,
    fusion_config_id VARCHAR(64) NOT NULL REFERENCES fusion_configs(id) ON DELETE CASCADE,
    fusion_config_name VARCHAR(255), -- 冗余字段，便于查询
    timestamp TIMESTAMP NOT NULL,
    fused_data JSONB NOT NULL, -- 融合后的数据
    source_count INTEGER DEFAULT 0, -- 参与融合的数据源数量
    source_ids TEXT[], -- 参与融合的数据源ID列表
    fusion_strategy VARCHAR(50), -- 使用的融合策略
    quality_score DOUBLE PRECISION DEFAULT 1.0, -- 数据质量评分
    metadata JSONB NOT NULL DEFAULT '{}', -- 元数据
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建数据源数据缓存表（用于时间窗口内的数据收集）
CREATE TABLE IF NOT EXISTS fusion_source_data_cache (
    id VARCHAR(64) PRIMARY KEY,
    source_id VARCHAR(64) NOT NULL REFERENCES fusion_data_sources(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    data JSONB NOT NULL, -- 数据内容
    quality DOUBLE PRECISION DEFAULT 1.0, -- 数据质量
    metadata JSONB NOT NULL DEFAULT '{}', -- 元数据
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_fusion_cache_source_time (source_id, timestamp DESC)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_fusion_data_sources_type ON fusion_data_sources(source_type);
CREATE INDEX IF NOT EXISTS idx_fusion_data_sources_device ON fusion_data_sources(device_id);
CREATE INDEX IF NOT EXISTS idx_fusion_data_sources_enabled ON fusion_data_sources(enabled);
CREATE INDEX IF NOT EXISTS idx_fusion_configs_enabled ON fusion_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_fusion_config_sources_config ON fusion_config_sources(fusion_config_id);
CREATE INDEX IF NOT EXISTS idx_fusion_config_sources_source ON fusion_config_sources(source_id);
CREATE INDEX IF NOT EXISTS idx_fusion_results_config_id ON fusion_results(fusion_config_id);
CREATE INDEX IF NOT EXISTS idx_fusion_results_config_time ON fusion_results(fusion_config_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_fusion_results_timestamp ON fusion_results(timestamp DESC);

-- 添加注释
COMMENT ON TABLE fusion_data_sources IS '通用数据源配置表（支持设备字段、设备、通道等）';
COMMENT ON TABLE fusion_configs IS '通用融合配置表（不绑定到单一设备）';
COMMENT ON TABLE fusion_config_sources IS '融合配置与数据源关联表（多对多）';
COMMENT ON TABLE fusion_results IS '通用融合结果表';
COMMENT ON TABLE fusion_source_data_cache IS '数据源数据缓存表（用于时间窗口数据收集）';
COMMENT ON COLUMN fusion_data_sources.source_type IS '数据源类型: device_field(设备字段), device(设备), channel(通道), external(外部), calculated(计算)';
COMMENT ON COLUMN fusion_configs.fusion_strategy IS '融合策略: weighted(加权), average(平均), kalman(卡尔曼滤波)等';
