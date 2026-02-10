-- Migration: 添加数据流追踪功能
-- Created: 2026-01-16
-- Description: 添加 trace_events 表用于存储数据流追踪事件，支持实时可视化

-- 追踪事件表
CREATE TABLE IF NOT EXISTS trace_events (
    -- 注意：TimescaleDB hypertable 的唯一约束/主键必须包含分区列（本表为 timestamp）
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    pipeline_id VARCHAR(255) NOT NULL,
    sample_id VARCHAR(255) NOT NULL,
    component_type VARCHAR(50) NOT NULL,
    component_id VARCHAR(255) NOT NULL,
    component_name VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    duration_ms INTEGER,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    metadata JSONB,
    data_snapshot JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
);

-- 索引优化查询性能
CREATE INDEX idx_trace_events_pipeline ON trace_events(pipeline_id);
CREATE INDEX idx_trace_events_sample ON trace_events(sample_id);
CREATE INDEX idx_trace_events_timestamp ON trace_events(timestamp DESC);
CREATE INDEX idx_trace_events_component ON trace_events(component_type, component_id);
CREATE INDEX idx_trace_events_status ON trace_events(status);
CREATE INDEX idx_trace_events_action ON trace_events(action);

-- 组合索引用于常见查询
CREATE INDEX idx_trace_events_pipeline_time ON trace_events(pipeline_id, timestamp DESC);
CREATE INDEX idx_trace_events_sample_time ON trace_events(sample_id, timestamp ASC);

-- 使用 TimescaleDB 将表转换为 hypertable（优化时序数据存储）
-- 注意：需要先创建 TimescaleDB 扩展
-- CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
SELECT create_hypertable('trace_events', 'timestamp',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- 设置数据保留策略（保留 30 天）
SELECT add_retention_policy('trace_events', INTERVAL '30 days', if_not_exists => TRUE);

-- 启用压缩（节省存储空间）
ALTER TABLE trace_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'pipeline_id,component_type',
    timescaledb.compress_orderby = 'timestamp DESC'
);

-- 自动压缩 7 天前的数据
SELECT add_compression_policy('trace_events', INTERVAL '7 days', if_not_exists => TRUE);

-- 拓扑配置表（存储 Pipeline 配置用于前端渲染）
CREATE TABLE IF NOT EXISTS pipeline_topology (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_pipeline_topology_status ON pipeline_topology(status);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_pipeline_topology_updated_at
BEFORE UPDATE ON pipeline_topology
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- 性能指标聚合物化视图（1分钟粒度）
CREATE MATERIALIZED VIEW IF NOT EXISTS pipeline_metrics_1min AS
SELECT
    pipeline_id,
    component_type,
    component_id,
    component_name,
    time_bucket('1 minute', timestamp) as time_bucket,
    COUNT(*) as event_count,
    AVG(duration_ms) as avg_duration_ms,
    MAX(duration_ms) as max_duration_ms,
    MIN(duration_ms) as min_duration_ms,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms) as median_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration_ms,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99_duration_ms,
    COUNT(CASE WHEN status = 'success' THEN 1 END) as success_count,
    COUNT(CASE WHEN status = 'error' THEN 1 END) as error_count,
    COUNT(CASE WHEN status = 'skip' THEN 1 END) as skip_count,
    ROUND(
        COUNT(CASE WHEN status = 'error' THEN 1 END)::NUMERIC /
        NULLIF(COUNT(*)::NUMERIC, 0) * 100,
        2
    ) as error_rate
FROM trace_events
WHERE duration_ms IS NOT NULL
GROUP BY pipeline_id, component_type, component_id, component_name, time_bucket;

CREATE UNIQUE INDEX ON pipeline_metrics_1min (pipeline_id, component_type, component_id, time_bucket DESC);

-- 性能指标聚合物化视图（1小时粒度）
CREATE MATERIALIZED VIEW IF NOT EXISTS pipeline_metrics_1hour AS
SELECT
    pipeline_id,
    component_type,
    component_id,
    component_name,
    time_bucket('1 hour', timestamp) as time_bucket,
    COUNT(*) as event_count,
    AVG(duration_ms) as avg_duration_ms,
    MAX(duration_ms) as max_duration_ms,
    MIN(duration_ms) as min_duration_ms,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms) as median_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration_ms,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99_duration_ms,
    COUNT(CASE WHEN status = 'success' THEN 1 END) as success_count,
    COUNT(CASE WHEN status = 'error' THEN 1 END) as error_count,
    COUNT(CASE WHEN status = 'skip' THEN 1 END) as skip_count,
    ROUND(
        COUNT(CASE WHEN status = 'error' THEN 1 END)::NUMERIC /
        NULLIF(COUNT(*)::NUMERIC, 0) * 100,
        2
    ) as error_rate
FROM trace_events
WHERE duration_ms IS NOT NULL
GROUP BY pipeline_id, component_type, component_id, component_name, time_bucket;

CREATE UNIQUE INDEX ON pipeline_metrics_1hour (pipeline_id, component_type, component_id, time_bucket DESC);

-- 定期刷新物化视图（每分钟）
-- 注意：这需要 pg_cron 扩展或应用层定时任务
-- CREATE EXTENSION IF NOT EXISTS pg_cron;
-- SELECT cron.schedule('refresh-pipeline-metrics-1min', '* * * * *', 'REFRESH MATERIALIZED VIEW CONCURRENTLY pipeline_metrics_1min');
-- SELECT cron.schedule('refresh-pipeline-metrics-1hour', '0 * * * *', 'REFRESH MATERIALIZED VIEW CONCURRENTLY pipeline_metrics_1hour');

-- 样本追踪链路视图（获取单个样本的完整追踪链路）
CREATE OR REPLACE VIEW sample_trace_chain AS
SELECT
    sample_id,
    pipeline_id,
    ARRAY_AGG(
        jsonb_build_object(
            'component_type', component_type,
            'component_id', component_id,
            'component_name', component_name,
            'action', action,
            'timestamp', timestamp,
            'duration_ms', duration_ms,
            'status', status,
            'error_message', error_message
        ) ORDER BY timestamp ASC
    ) as trace_chain,
    MIN(timestamp) as start_time,
    MAX(timestamp) as end_time,
    EXTRACT(EPOCH FROM (MAX(timestamp) - MIN(timestamp))) * 1000 as total_duration_ms,
    COUNT(*) as event_count,
    COUNT(CASE WHEN status = 'error' THEN 1 END) > 0 as has_error
FROM trace_events
GROUP BY sample_id, pipeline_id;

-- 实时活跃管道视图
CREATE OR REPLACE VIEW active_pipelines AS
SELECT DISTINCT
    pipeline_id,
    MAX(timestamp) as last_activity,
    COUNT(*) as recent_events
FROM trace_events
WHERE timestamp > NOW() - INTERVAL '5 minutes'
GROUP BY pipeline_id;

-- 组件性能排行榜视图
CREATE OR REPLACE VIEW component_performance_ranking AS
SELECT
    component_type,
    component_id,
    component_name,
    COUNT(*) as total_events,
    AVG(duration_ms) as avg_duration_ms,
    MAX(duration_ms) as max_duration_ms,
    COUNT(CASE WHEN status = 'error' THEN 1 END) as error_count,
    ROUND(
        COUNT(CASE WHEN status = 'error' THEN 1 END)::NUMERIC /
        NULLIF(COUNT(*)::NUMERIC, 0) * 100,
        2
    ) as error_rate
FROM trace_events
WHERE timestamp > NOW() - INTERVAL '1 hour'
  AND duration_ms IS NOT NULL
GROUP BY component_type, component_id, component_name
ORDER BY avg_duration_ms DESC;

-- 添加注释
COMMENT ON TABLE trace_events IS '数据流追踪事件表，记录数据在 Pipeline 中的流动过程';
COMMENT ON TABLE pipeline_topology IS 'Pipeline 拓扑配置表，存储管道结构用于前端渲染';
COMMENT ON MATERIALIZED VIEW pipeline_metrics_1min IS 'Pipeline 性能指标聚合（1分钟粒度）';
COMMENT ON MATERIALIZED VIEW pipeline_metrics_1hour IS 'Pipeline 性能指标聚合（1小时粒度）';
COMMENT ON VIEW sample_trace_chain IS '样本追踪链路视图，展示单个样本的完整处理链路';
COMMENT ON VIEW active_pipelines IS '实时活跃管道视图';
COMMENT ON VIEW component_performance_ranking IS '组件性能排行榜视图';
