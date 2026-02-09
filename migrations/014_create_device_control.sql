-- ============================================
-- Device Control Enhancement Tables
-- ============================================

-- 创建控制策略表
CREATE TABLE IF NOT EXISTS control_policies (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER DEFAULT 0, -- 优先级（数字越大优先级越高）
    condition_config JSONB NOT NULL DEFAULT '{}', -- 触发条件配置
    action_config JSONB NOT NULL DEFAULT '{}', -- 控制动作配置
    cooldown_seconds INTEGER DEFAULT 300, -- 冷却时间（秒）
    max_executions INTEGER DEFAULT 0, -- 最大执行次数（0=无限制）
    execution_count INTEGER DEFAULT 0, -- 已执行次数
    last_executed_at TIMESTAMP, -- 上次执行时间
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建控制动作记录表
CREATE TABLE IF NOT EXISTS control_actions (
    id VARCHAR(64) PRIMARY KEY,
    policy_id VARCHAR(64) REFERENCES control_policies(id) ON DELETE SET NULL,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    action_type VARCHAR(50) NOT NULL, -- emergency_stop, shutdown, start, pause, resume, set_value等
    parameters JSONB NOT NULL DEFAULT '{}', -- 动作参数
    reason TEXT, -- 执行原因
    severity INTEGER DEFAULT 1, -- 严重程度
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, executing, completed, failed, cancelled
    result JSONB, -- 执行结果
    error_message TEXT, -- 错误信息
    executed_by VARCHAR(255), -- 执行人（用户ID或系统）
    executed_at TIMESTAMP, -- 执行时间
    completed_at TIMESTAMP, -- 完成时间
    duration_ms INTEGER, -- 执行耗时（毫秒）
    require_confirmation BOOLEAN DEFAULT false, -- 是否需要确认
    confirmed_by VARCHAR(255), -- 确认人
    confirmed_at TIMESTAMP, -- 确认时间
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建控制审计日志表
CREATE TABLE IF NOT EXISTS control_audit_logs (
    id VARCHAR(64) PRIMARY KEY,
    action_id VARCHAR(64) REFERENCES control_actions(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- created, confirmed, executed, completed, failed, cancelled
    user_id VARCHAR(255), -- 操作用户ID
    user_name VARCHAR(255), -- 操作用户名
    details JSONB NOT NULL DEFAULT '{}', -- 详细信息
    ip_address VARCHAR(50), -- IP地址
    user_agent TEXT, -- User Agent
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建控制权限表
CREATE TABLE IF NOT EXISTS control_permissions (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    action_type VARCHAR(50) NOT NULL, -- 允许的动作类型
    target_devices TEXT[], -- 允许控制的设备ID列表（空=所有设备）
    max_severity INTEGER DEFAULT 3, -- 允许的最大严重程度
    require_confirmation BOOLEAN DEFAULT false, -- 是否需要确认
    allowed_time_ranges JSONB NOT NULL DEFAULT '[]', -- 允许执行的时间范围
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, action_type)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_control_policies_enabled ON control_policies(enabled);
CREATE INDEX IF NOT EXISTS idx_control_policies_priority ON control_policies(priority DESC);
CREATE INDEX IF NOT EXISTS idx_control_actions_device ON control_actions(device_id);
CREATE INDEX IF NOT EXISTS idx_control_actions_policy ON control_actions(policy_id);
CREATE INDEX IF NOT EXISTS idx_control_actions_status ON control_actions(status);
CREATE INDEX IF NOT EXISTS idx_control_actions_executed_at ON control_actions(executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_audit_logs_action ON control_audit_logs(action_id);
CREATE INDEX IF NOT EXISTS idx_control_audit_logs_user ON control_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_control_audit_logs_created ON control_audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_permissions_user ON control_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_control_permissions_enabled ON control_permissions(enabled);

-- 添加注释
COMMENT ON TABLE control_policies IS '控制策略表';
COMMENT ON TABLE control_actions IS '控制动作记录表';
COMMENT ON TABLE control_audit_logs IS '控制审计日志表';
COMMENT ON TABLE control_permissions IS '控制权限表';
COMMENT ON COLUMN control_actions.action_type IS '动作类型: emergency_stop, shutdown, start, pause, resume, set_value, call_method等';
COMMENT ON COLUMN control_actions.status IS '状态: pending(待执行), executing(执行中), completed(已完成), failed(失败), cancelled(已取消)';
COMMENT ON COLUMN control_audit_logs.event_type IS '事件类型: created, confirmed, executed, completed, failed, cancelled';
