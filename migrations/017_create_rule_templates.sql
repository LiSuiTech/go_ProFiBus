-- ============================================
-- Rule Templates Tables
-- ============================================

-- 创建规则模板表
CREATE TABLE IF NOT EXISTS rule_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100), -- 模板分类：threshold, anomaly, trend, composite, etc.
    rule_type VARCHAR(50) NOT NULL, -- 规则类型：alert, control, data_quality, etc.
    tags TEXT[], -- 标签数组
    icon VARCHAR(100), -- 图标名称
    condition_template JSONB NOT NULL, -- 条件模板（支持变量占位符）
    variables_config JSONB NOT NULL DEFAULT '{}', -- 可配置变量说明
    output_config JSONB NOT NULL DEFAULT '{}', -- 输出配置（告警级别、消息模板等）
    usage_count INTEGER DEFAULT 0, -- 使用次数
    rating DOUBLE PRECISION DEFAULT 0.0, -- 评分 0-5
    enabled BOOLEAN NOT NULL DEFAULT true, -- 是否启用
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255), -- 创建者
    metadata JSONB NOT NULL DEFAULT '{}' -- 模板元数据
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_rule_templates_category ON rule_templates(category);
CREATE INDEX IF NOT EXISTS idx_rule_templates_rule_type ON rule_templates(rule_type);
CREATE INDEX IF NOT EXISTS idx_rule_templates_enabled ON rule_templates(enabled);
CREATE INDEX IF NOT EXISTS idx_rule_templates_created_at ON rule_templates(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rule_templates_tags ON rule_templates USING GIN(tags);

-- 添加注释
COMMENT ON TABLE rule_templates IS '规则模板表';
COMMENT ON COLUMN rule_templates.condition_template IS '条件模板JSON：支持变量占位符如 ${threshold}、${field_name}';
COMMENT ON COLUMN rule_templates.variables_config IS '可配置变量说明JSON：描述哪些变量可以在应用模板时配置';
COMMENT ON COLUMN rule_templates.output_config IS '输出配置JSON：告警级别、消息模板、动作配置等';

-- 插入一些示例模板
INSERT INTO rule_templates (id, name, description, category, rule_type, tags, icon, condition_template, variables_config, output_config, created_by) VALUES
(
    'template-threshold-exceed',
    '阈值超限规则',
    '当指定字段的值超过阈值时触发告警',
    'threshold',
    'alert',
    ARRAY['阈值', '告警', '监控'],
    'Warning',
    '{
        "type": "threshold",
        "field": "${field_name}",
        "operator": "${operator}",
        "value": "${threshold}",
        "duration": "${duration}"
    }',
    '{
        "field_name": {"type": "string", "description": "监控字段名称", "required": true},
        "operator": {"type": "string", "description": "比较操作符", "enum": ["gt", "gte", "lt", "lte", "eq", "ne"], "default": "gt"},
        "threshold": {"type": "number", "description": "阈值", "required": true},
        "duration": {"type": "number", "description": "持续时间（秒）", "default": 0}
    }',
    '{
        "level": "${alert_level}",
        "message_template": "字段 ${field_name} 的值 ${value} ${operator_desc} 阈值 ${threshold}",
        "operator_desc": {"gt": "大于", "gte": "大于等于", "lt": "小于", "lte": "小于等于", "eq": "等于", "ne": "不等于"}
    }',
    'system'
),
(
    'template-anomaly-detection',
    '异常检测规则',
    '使用统计方法检测数据异常（基于均值和标准差）',
    'anomaly',
    'alert',
    ARRAY['异常', '统计', '检测'],
    'Connection',
    '{
        "type": "anomaly",
        "field": "${field_name}",
        "method": "${method}",
        "threshold": "${threshold}"
    }',
    '{
        "field_name": {"type": "string", "description": "监控字段名称", "required": true},
        "method": {"type": "string", "description": "检测方法", "enum": ["z_score", "iqr", "isolation_forest"], "default": "z_score"},
        "threshold": {"type": "number", "description": "异常阈值（Z分数或IQR倍数）", "default": 3.0}
    }',
    '{
        "level": "${alert_level}",
        "message_template": "检测到字段 ${field_name} 的异常值 ${value}（${method}方法，阈值：${threshold}）"
    }',
    'system'
),
(
    'template-trend-analysis',
    '趋势分析规则',
    '检测数据趋势变化（上升/下降趋势）',
    'trend',
    'alert',
    ARRAY['趋势', '分析', '变化'],
    'TrendCharts',
    '{
        "type": "trend",
        "field": "${field_name}",
        "window_size": "${window_size}",
        "trend_type": "${trend_type}",
        "threshold": "${threshold}"
    }',
    '{
        "field_name": {"type": "string", "description": "监控字段名称", "required": true},
        "window_size": {"type": "number", "description": "时间窗口大小（数据点数）", "default": 10},
        "trend_type": {"type": "string", "description": "趋势类型", "enum": ["increasing", "decreasing", "stable"], "default": "increasing"},
        "threshold": {"type": "number", "description": "趋势变化阈值", "default": 0.1}
    }',
    '{
        "level": "${alert_level}",
        "message_template": "检测到字段 ${field_name} 的${trend_type_desc}趋势（窗口大小：${window_size}，阈值：${threshold}）",
        "trend_type_desc": {"increasing": "上升", "decreasing": "下降", "stable": "稳定"}
    }',
    'system'
),
(
    'template-composite-and',
    '复合规则（AND）',
    '多个条件同时满足时触发（逻辑AND）',
    'composite',
    'alert',
    ARRAY['复合', 'AND', '多条件'],
    'Operation',
    '{
        "type": "composite",
        "logic": "AND",
        "conditions": [
            {"field": "${field1}", "operator": "${operator1}", "value": "${value1}"},
            {"field": "${field2}", "operator": "${operator2}", "value": "${value2}"}
        ]
    }',
    '{
        "field1": {"type": "string", "description": "第一个字段名称", "required": true},
        "operator1": {"type": "string", "description": "第一个操作符", "enum": ["gt", "gte", "lt", "lte", "eq", "ne"], "default": "gt"},
        "value1": {"type": "number", "description": "第一个阈值", "required": true},
        "field2": {"type": "string", "description": "第二个字段名称", "required": true},
        "operator2": {"type": "string", "description": "第二个操作符", "enum": ["gt", "gte", "lt", "lte", "eq", "ne"], "default": "gt"},
        "value2": {"type": "number", "description": "第二个阈值", "required": true}
    }',
    '{
        "level": "${alert_level}",
        "message_template": "复合条件满足：${field1} ${operator1} ${value1} AND ${field2} ${operator2} ${value2}"
    }',
    'system'
),
(
    'template-rate-of-change',
    '变化率规则',
    '检测数据变化率超过阈值',
    'rate',
    'alert',
    ARRAY['变化率', '速率', '监控'],
    'DataAnalysis',
    '{
        "type": "rate_of_change",
        "field": "${field_name}",
        "time_window": "${time_window}",
        "threshold": "${threshold}"
    }',
    '{
        "field_name": {"type": "string", "description": "监控字段名称", "required": true},
        "time_window": {"type": "number", "description": "时间窗口（秒）", "default": 60},
        "threshold": {"type": "number", "description": "变化率阈值（百分比）", "default": 10.0}
    }',
    '{
        "level": "${alert_level}",
        "message_template": "字段 ${field_name} 的变化率 ${rate}% 超过阈值 ${threshold}%（时间窗口：${time_window}秒）"
    }',
    'system'
)
ON CONFLICT (id) DO NOTHING;

-- 创建规则测试记录表（用于规则测试功能）
CREATE TABLE IF NOT EXISTS rule_test_results (
    id VARCHAR(64) PRIMARY KEY,
    rule_id VARCHAR(64), -- 规则ID（如果是从规则创建的）
    template_id VARCHAR(64) REFERENCES rule_templates(id) ON DELETE SET NULL, -- 模板ID（如果是从模板测试的）
    test_data JSONB NOT NULL, -- 测试数据
    rule_config JSONB NOT NULL, -- 规则配置（应用变量后的完整规则）
    test_result JSONB NOT NULL, -- 测试结果
    triggered BOOLEAN NOT NULL DEFAULT false, -- 是否触发
    execution_time_ms INTEGER, -- 执行时间（毫秒）
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) -- 测试者
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_rule_test_results_rule_id ON rule_test_results(rule_id);
CREATE INDEX IF NOT EXISTS idx_rule_test_results_template_id ON rule_test_results(template_id);
CREATE INDEX IF NOT EXISTS idx_rule_test_results_triggered ON rule_test_results(triggered);
CREATE INDEX IF NOT EXISTS idx_rule_test_results_created_at ON rule_test_results(created_at DESC);

-- 添加注释
COMMENT ON TABLE rule_test_results IS '规则测试结果表';
COMMENT ON COLUMN rule_test_results.test_data IS '测试数据JSON：模拟的输入数据';
COMMENT ON COLUMN rule_test_results.rule_config IS '规则配置JSON：应用变量后的完整规则定义';
COMMENT ON COLUMN rule_test_results.test_result IS '测试结果JSON：包含触发状态、匹配详情、执行信息等';
