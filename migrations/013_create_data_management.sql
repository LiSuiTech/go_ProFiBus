-- ============================================
-- Data Management Center Tables
-- ============================================

-- 创建数据清洗规则表
CREATE TABLE IF NOT EXISTS data_cleaning_rules (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL, -- deduplicate, outlier_filter, missing_fill, normalize等
    enabled BOOLEAN NOT NULL DEFAULT true,
    config JSONB NOT NULL DEFAULT '{}', -- 规则配置
    priority INTEGER DEFAULT 0, -- 优先级（数字越大优先级越高）
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建数据归档策略表
CREATE TABLE IF NOT EXISTS data_archive_policies (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    source_type VARCHAR(50) NOT NULL, -- device, channel, fusion等
    source_id VARCHAR(64), -- 数据源ID（可选，为空表示所有数据源）
    retention_days INTEGER NOT NULL, -- 保留天数
    archive_after_days INTEGER NOT NULL, -- 归档天数（数据保留多少天后归档）
    compression_enabled BOOLEAN DEFAULT true, -- 是否启用压缩
    archive_location VARCHAR(500), -- 归档位置（文件路径或对象存储路径）
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMP, -- 上次执行时间
    next_run_at TIMESTAMP, -- 下次执行时间
    run_interval_hours INTEGER DEFAULT 24, -- 执行间隔（小时）
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建数据归档记录表
CREATE TABLE IF NOT EXISTS data_archive_records (
    id VARCHAR(64) PRIMARY KEY,
    policy_id VARCHAR(64) NOT NULL REFERENCES data_archive_policies(id) ON DELETE CASCADE,
    source_type VARCHAR(50) NOT NULL,
    source_id VARCHAR(64),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    record_count BIGINT DEFAULT 0, -- 归档的记录数
    archive_size BIGINT DEFAULT 0, -- 归档文件大小（字节）
    archive_path VARCHAR(500), -- 归档文件路径
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- 创建数据清洗记录表
CREATE TABLE IF NOT EXISTS data_cleaning_records (
    id VARCHAR(64) PRIMARY KEY,
    rule_id VARCHAR(64) NOT NULL REFERENCES data_cleaning_rules(id) ON DELETE CASCADE,
    source_type VARCHAR(50) NOT NULL,
    source_id VARCHAR(64),
    processed_count BIGINT DEFAULT 0, -- 处理的记录数
    cleaned_count BIGINT DEFAULT 0, -- 清洗的记录数
    removed_count BIGINT DEFAULT 0, -- 移除的记录数
    filled_count BIGINT DEFAULT 0, -- 填充的记录数
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建数据生命周期配置表
CREATE TABLE IF NOT EXISTS data_lifecycle_configs (
    id VARCHAR(64) PRIMARY KEY,
    source_type VARCHAR(50) NOT NULL,
    source_id VARCHAR(64),
    hot_storage_days INTEGER DEFAULT 7, -- 热存储天数（快速查询）
    warm_storage_days INTEGER DEFAULT 30, -- 温存储天数（可查询但较慢）
    cold_storage_days INTEGER DEFAULT 365, -- 冷存储天数（归档存储）
    delete_after_days INTEGER, -- 删除天数（0=不删除）
    compression_after_days INTEGER DEFAULT 30, -- 压缩天数
    enabled BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_type, source_id)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_data_cleaning_rules_enabled ON data_cleaning_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_data_cleaning_rules_type ON data_cleaning_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_data_archive_policies_enabled ON data_archive_policies(enabled);
CREATE INDEX IF NOT EXISTS idx_data_archive_policies_next_run ON data_archive_policies(next_run_at);
CREATE INDEX IF NOT EXISTS idx_data_archive_records_policy ON data_archive_records(policy_id);
CREATE INDEX IF NOT EXISTS idx_data_archive_records_status ON data_archive_records(status);
CREATE INDEX IF NOT EXISTS idx_data_archive_records_time ON data_archive_records(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_data_cleaning_records_rule ON data_cleaning_records(rule_id);
CREATE INDEX IF NOT EXISTS idx_data_cleaning_records_status ON data_cleaning_records(status);
CREATE INDEX IF NOT EXISTS idx_data_lifecycle_configs_source ON data_lifecycle_configs(source_type, source_id);

-- 添加注释
COMMENT ON TABLE data_cleaning_rules IS '数据清洗规则表';
COMMENT ON TABLE data_archive_policies IS '数据归档策略表';
COMMENT ON TABLE data_archive_records IS '数据归档记录表';
COMMENT ON TABLE data_cleaning_records IS '数据清洗记录表';
COMMENT ON TABLE data_lifecycle_configs IS '数据生命周期配置表';
COMMENT ON COLUMN data_cleaning_rules.rule_type IS '规则类型: deduplicate(去重), outlier_filter(异常值过滤), missing_fill(缺失值填充), normalize(标准化)等';
COMMENT ON COLUMN data_archive_records.status IS '状态: pending(待执行), running(执行中), completed(已完成), failed(失败)';
