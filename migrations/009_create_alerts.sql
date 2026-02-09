-- ============================================
-- Alert Management Tables
-- ============================================

-- 创建告警规则表
CREATE TABLE IF NOT EXISTS alert_rules (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    condition JSONB NOT NULL, -- 告警触发条件（JSON格式）
    level VARCHAR(20) NOT NULL DEFAULT 'warning', -- info, warning, error, critical
    enabled BOOLEAN NOT NULL DEFAULT true,
    cooldown_seconds INTEGER DEFAULT 300, -- 冷却时间（秒）
    max_executions INTEGER DEFAULT 0, -- 最大执行次数（0=无限制）
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建告警表
CREATE TABLE IF NOT EXISTS alerts (
    id VARCHAR(64) PRIMARY KEY,
    rule_id VARCHAR(64) REFERENCES alert_rules(id) ON DELETE SET NULL,
    device_id VARCHAR(64) REFERENCES devices(id) ON DELETE SET NULL,
    channel_id VARCHAR(64) REFERENCES channels(id) ON DELETE SET NULL,
    event_id VARCHAR(64), -- 关联的事件ID（如果有）
    level VARCHAR(20) NOT NULL, -- info, warning, error, critical
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, acknowledged, resolved, suppressed
    message TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}', -- 告警详细信息
    first_occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP,
    acknowledged_by VARCHAR(255),
    resolved_at TIMESTAMP,
    resolved_by VARCHAR(255),
    count INTEGER DEFAULT 1, -- 告警发生次数（用于聚合）
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_alert_rules_level ON alert_rules(level);
CREATE INDEX IF NOT EXISTS idx_alerts_rule_id ON alerts(rule_id);
CREATE INDEX IF NOT EXISTS idx_alerts_device_id ON alerts(device_id);
CREATE INDEX IF NOT EXISTS idx_alerts_channel_id ON alerts(channel_id);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_level ON alerts(level);
CREATE INDEX IF NOT EXISTS idx_alerts_first_occurred_at ON alerts(first_occurred_at);
CREATE INDEX IF NOT EXISTS idx_alerts_last_occurred_at ON alerts(last_occurred_at);

-- 添加注释
COMMENT ON TABLE alert_rules IS '告警规则表';
COMMENT ON TABLE alerts IS '告警表';
COMMENT ON COLUMN alert_rules.condition IS '告警触发条件JSON，包含字段、操作符、阈值等';
COMMENT ON COLUMN alert_rules.level IS '告警级别: info, warning, error, critical';
COMMENT ON COLUMN alerts.level IS '告警级别: info, warning, error, critical';
COMMENT ON COLUMN alerts.status IS '告警状态: active, acknowledged, resolved, suppressed';
COMMENT ON COLUMN alerts.count IS '告警发生次数，用于告警聚合';
