-- go_ProFiBus 数据库初始化脚本
-- 使用 TimescaleDB + PostgreSQL

-- ============================================
-- 1. 启用TimescaleDB扩展
-- ============================================
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ============================================
-- 2. 时序数据表 - sensor_readings
-- ============================================
CREATE TABLE IF NOT EXISTS sensor_readings (
    time        TIMESTAMPTZ NOT NULL,
    sensor_id   TEXT NOT NULL,
    protocol    TEXT,
    data        JSONB,
    quality     FLOAT,
    source_id   TEXT
);

-- 创建TimescaleDB超表（Hypertable）
-- 按时间分区，默认chunk间隔为7天
SELECT create_hypertable(
    'sensor_readings',
    'time',
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

-- 创建索引提高查询性能
CREATE INDEX IF NOT EXISTS idx_sensor_readings_sensor_id_time
    ON sensor_readings (sensor_id, time DESC);

CREATE INDEX IF NOT EXISTS idx_sensor_readings_source_id
    ON sensor_readings (source_id);

-- 创建JSONB数据的GIN索引（支持JSON查询）
CREATE INDEX IF NOT EXISTS idx_sensor_readings_data
    ON sensor_readings USING GIN (data);

-- 设置数据压缩策略（7天前的数据压缩）
ALTER TABLE sensor_readings SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'sensor_id'
);

SELECT add_compression_policy(
    'sensor_readings',
    INTERVAL '7 days',
    if_not_exists => TRUE
);

-- 设置数据保留策略（可选，保留90天）
-- SELECT add_retention_policy('sensor_readings', INTERVAL '90 days', if_not_exists => TRUE);

-- ============================================
-- 3. 事件表 - events
-- ============================================
CREATE TABLE IF NOT EXISTS events (
    id          TEXT PRIMARY KEY,
    type        TEXT NOT NULL,
    status      TEXT NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    severity    INTEGER,
    score       FLOAT,
    description TEXT,
    metadata    JSONB DEFAULT '{}',
    source_id   TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_events_timestamp
    ON events (timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_events_status
    ON events (status);

CREATE INDEX IF NOT EXISTS idx_events_type
    ON events (type);

CREATE INDEX IF NOT EXISTS idx_events_source_id
    ON events (source_id);

CREATE INDEX IF NOT EXISTS idx_events_severity
    ON events (severity);

-- JSONB元数据索引
CREATE INDEX IF NOT EXISTS idx_events_metadata
    ON events USING GIN (metadata);

-- 复合索引（常用查询组合）
CREATE INDEX IF NOT EXISTS idx_events_status_timestamp
    ON events (status, timestamp DESC);

-- ============================================
-- 4. 规则表 - rules
-- ============================================
CREATE TABLE IF NOT EXISTS rules (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    config      JSONB NOT NULL,
    enabled     BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_rules_enabled
    ON rules (enabled);

CREATE INDEX IF NOT EXISTS idx_rules_type
    ON rules (type);

-- ============================================
-- 5. 标注表 - annotations
-- ============================================
CREATE TABLE IF NOT EXISTS annotations (
    id           TEXT PRIMARY KEY,
    event_id     TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    annotator_id TEXT NOT NULL,
    confirmed    BOOLEAN,
    annotation   TEXT,
    labels       JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_annotations_event_id
    ON annotations (event_id);

CREATE INDEX IF NOT EXISTS idx_annotations_annotator_id
    ON annotations (annotator_id);

CREATE INDEX IF NOT EXISTS idx_annotations_confirmed
    ON annotations (confirmed);

-- ============================================
-- 6. 标注员表 - annotators
-- ============================================
CREATE TABLE IF NOT EXISTS annotators (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    role        TEXT,
    permissions TEXT[],
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 7. 触发器 - 自动更新updated_at
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rules_updated_at
    BEFORE UPDATE ON rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 8. 视图 - 常用查询视图
-- ============================================

-- 待标注事件视图
CREATE OR REPLACE VIEW pending_events AS
SELECT
    e.id,
    e.type,
    e.timestamp,
    e.severity,
    e.score,
    e.description,
    e.source_id,
    e.created_at
FROM events e
WHERE e.status = 'pending'
ORDER BY e.severity DESC, e.timestamp DESC;

-- 最近24小时事件统计视图
CREATE OR REPLACE VIEW recent_event_stats AS
SELECT
    status,
    type,
    COUNT(*) as count,
    AVG(score) as avg_score,
    MAX(severity) as max_severity
FROM events
WHERE timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY status, type;

-- 传感器数据统计视图（最近1小时）
CREATE OR REPLACE VIEW recent_sensor_stats AS
SELECT
    sensor_id,
    COUNT(*) as data_points,
    AVG(quality) as avg_quality,
    MIN(time) as first_reading,
    MAX(time) as last_reading
FROM sensor_readings
WHERE time >= NOW() - INTERVAL '1 hour'
GROUP BY sensor_id;

-- ============================================
-- 9. 函数 - 辅助查询函数
-- ============================================

-- 获取传感器数据的时间范围
CREATE OR REPLACE FUNCTION get_sensor_time_range(p_sensor_id TEXT)
RETURNS TABLE(min_time TIMESTAMPTZ, max_time TIMESTAMPTZ, count BIGINT) AS $$
BEGIN
    RETURN QUERY
    SELECT
        MIN(time) as min_time,
        MAX(time) as max_time,
        COUNT(*) as count
    FROM sensor_readings
    WHERE sensor_id = p_sensor_id;
END;
$$ LANGUAGE plpgsql;

-- 清理旧数据函数
CREATE OR REPLACE FUNCTION cleanup_old_data(days_to_keep INTEGER)
RETURNS BIGINT AS $$
DECLARE
    rows_deleted BIGINT;
BEGIN
    DELETE FROM sensor_readings
    WHERE time < NOW() - (days_to_keep || ' days')::INTERVAL;

    GET DIAGNOSTICS rows_deleted = ROW_COUNT;
    RETURN rows_deleted;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 10. 初始数据（可选）
-- ============================================

-- 插入默认标注员
INSERT INTO annotators (id, name, role, permissions)
VALUES
    ('system', 'System', 'system', ARRAY['annotate', 'review', 'admin']),
    ('default', 'Default Annotator', 'annotator', ARRAY['annotate'])
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 11. 权限设置（根据实际需求调整）
-- ============================================

-- 创建只读用户（可选）
-- CREATE USER profibus_readonly WITH PASSWORD 'readonly_password';
-- GRANT CONNECT ON DATABASE profibus TO profibus_readonly;
-- GRANT USAGE ON SCHEMA public TO profibus_readonly;
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO profibus_readonly;

-- 创建读写用户（可选）
-- CREATE USER profibus_readwrite WITH PASSWORD 'readwrite_password';
-- GRANT CONNECT ON DATABASE profibus TO profibus_readwrite;
-- GRANT USAGE ON SCHEMA public TO profibus_readwrite;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO profibus_readwrite;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO profibus_readwrite;

-- ============================================
-- 完成
-- ============================================

-- 显示创建的表
SELECT
    schemaname,
    tablename,
    tableowner
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY tablename;

-- 显示超表信息
SELECT * FROM timescaledb_information.hypertables;
