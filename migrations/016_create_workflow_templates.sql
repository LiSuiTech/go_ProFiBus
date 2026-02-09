-- ============================================
-- Workflow Templates Tables
-- ============================================

-- 创建工作流模板表
CREATE TABLE IF NOT EXISTS workflow_templates (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100), -- 模板分类：monitoring, control, analysis, etc.
    tags TEXT[], -- 标签数组
    icon VARCHAR(100), -- 图标名称
    thumbnail_url VARCHAR(500), -- 缩略图URL
    workflow_data JSONB NOT NULL, -- 工作流定义（nodes, edges, variables）
    variables_config JSONB NOT NULL DEFAULT '{}', -- 可配置变量说明
    usage_count INTEGER DEFAULT 0, -- 使用次数
    rating DOUBLE PRECISION DEFAULT 0.0, -- 评分 0-5
    enabled BOOLEAN NOT NULL DEFAULT true, -- 是否启用
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255), -- 创建者
    metadata JSONB NOT NULL DEFAULT '{}' -- 模板元数据
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_workflow_templates_category ON workflow_templates(category);
CREATE INDEX IF NOT EXISTS idx_workflow_templates_enabled ON workflow_templates(enabled);
CREATE INDEX IF NOT EXISTS idx_workflow_templates_created_at ON workflow_templates(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_templates_tags ON workflow_templates USING GIN(tags);

-- 添加注释
COMMENT ON TABLE workflow_templates IS '工作流模板表';
COMMENT ON COLUMN workflow_templates.workflow_data IS '工作流定义JSON：包含nodes、edges、variables';
COMMENT ON COLUMN workflow_templates.variables_config IS '可配置变量说明JSON：描述哪些变量可以在应用模板时配置';
COMMENT ON COLUMN workflow_templates.category IS '模板分类：monitoring(监控), control(控制), analysis(分析), data_processing(数据处理), etc.';

-- 插入一些示例模板
INSERT INTO workflow_templates (id, name, description, category, tags, icon, workflow_data, variables_config, created_by) VALUES
(
    'template-device-monitoring',
    '设备监控工作流',
    '从设备采集数据，进行规则检测，触发告警',
    'monitoring',
    ARRAY['设备', '监控', '告警'],
    'Monitor',
    '{
        "nodes": [
            {
                "id": "device_source_1",
                "type": "device_source",
                "name": "设备数据源",
                "config": {"device_id": "${device_id}"},
                "position": {"x": 100, "y": 100},
                "inputs": [],
                "outputs": [{"id": "output_1", "label": "设备数据", "type": "data", "param_name": "device_data", "data_type": "object"}]
            },
            {
                "id": "rule_detection_1",
                "type": "rule_detection",
                "name": "规则检测",
                "config": {"rule_id": "${rule_id}"},
                "position": {"x": 300, "y": 100},
                "inputs": [{"id": "input_1", "label": "输入数据", "type": "data", "param_name": "input_data", "data_type": "object", "required": true}],
                "outputs": [{"id": "output_1", "label": "检测结果", "type": "data", "param_name": "detection_result", "data_type": "object"}]
            },
            {
                "id": "alert_output_1",
                "type": "alert_output",
                "name": "告警输出",
                "config": {"level": "${alert_level}", "channel_id": "${channel_id}"},
                "position": {"x": 500, "y": 100},
                "inputs": [{"id": "input_1", "label": "告警数据", "type": "data", "param_name": "alert_data", "data_type": "object", "required": true}],
                "outputs": []
            }
        ],
        "edges": [
            {
                "id": "edge_1",
                "source": "device_source_1",
                "target": "rule_detection_1",
                "source_port": "output_1",
                "target_port": "input_1",
                "param_mapping": {"input_data": "device_data"}
            },
            {
                "id": "edge_2",
                "source": "rule_detection_1",
                "target": "alert_output_1",
                "source_port": "output_1",
                "target_port": "input_1",
                "param_mapping": {"alert_data": "detection_result"}
            }
        ],
        "variables": []
    }',
    '{
        "device_id": {"type": "string", "description": "设备ID", "required": true},
        "rule_id": {"type": "string", "description": "规则ID", "required": true},
        "alert_level": {"type": "string", "description": "告警级别", "default": "warning"},
        "channel_id": {"type": "string", "description": "告警通道ID", "required": false}
    }',
    'system'
),
(
    'template-device-control',
    '设备控制工作流',
    '采集设备数据，进行ML分析，根据结果执行设备控制',
    'control',
    ARRAY['设备', '控制', 'ML'],
    'SwitchButton',
    '{
        "nodes": [
            {
                "id": "device_source_1",
                "type": "device_source",
                "name": "设备数据源",
                "config": {"device_id": "${device_id}"},
                "position": {"x": 100, "y": 100},
                "inputs": [],
                "outputs": [{"id": "output_1", "label": "设备数据", "type": "data", "param_name": "device_data", "data_type": "object"}]
            },
            {
                "id": "ml_analysis_1",
                "type": "ml_analysis",
                "name": "ML分析",
                "config": {"model_id": "${model_id}"},
                "position": {"x": 300, "y": 100},
                "inputs": [{"id": "input_1", "label": "输入数据", "type": "data", "param_name": "input_data", "data_type": "object", "required": true}],
                "outputs": [{"id": "output_1", "label": "分析结果", "type": "data", "param_name": "analysis_result", "data_type": "object"}]
            },
            {
                "id": "condition_1",
                "type": "condition",
                "name": "条件判断",
                "config": {"condition": "${control_condition}"},
                "position": {"x": 500, "y": 100},
                "inputs": [{"id": "input_1", "label": "输入数据", "type": "data", "param_name": "input_data", "data_type": "object", "required": true}],
                "outputs": [
                    {"id": "output_true", "label": "真", "type": "data", "param_name": "true_output", "data_type": "object"},
                    {"id": "output_false", "label": "假", "type": "data", "param_name": "false_output", "data_type": "object"}
                ]
            },
            {
                "id": "device_control_1",
                "type": "device_control",
                "name": "设备控制",
                "config": {"device_id": "${device_id}", "action_type": "${action_type}"},
                "position": {"x": 700, "y": 50},
                "inputs": [{"id": "input_1", "label": "控制参数", "type": "data", "param_name": "control_params", "data_type": "object", "required": true}],
                "outputs": []
            }
        ],
        "edges": [
            {
                "id": "edge_1",
                "source": "device_source_1",
                "target": "ml_analysis_1",
                "source_port": "output_1",
                "target_port": "input_1",
                "param_mapping": {"input_data": "device_data"}
            },
            {
                "id": "edge_2",
                "source": "ml_analysis_1",
                "target": "condition_1",
                "source_port": "output_1",
                "target_port": "input_1",
                "param_mapping": {"input_data": "analysis_result"}
            },
            {
                "id": "edge_3",
                "source": "condition_1",
                "target": "device_control_1",
                "source_port": "output_true",
                "target_port": "input_1",
                "param_mapping": {"control_params": "true_output"}
            }
        ],
        "variables": []
    }',
    '{
        "device_id": {"type": "string", "description": "设备ID", "required": true},
        "model_id": {"type": "string", "description": "ML模型ID", "required": true},
        "control_condition": {"type": "string", "description": "控制条件表达式", "default": "result.score > 0.8"},
        "action_type": {"type": "string", "description": "控制动作类型", "default": "set_parameter"}
    }',
    'system'
),
(
    'template-data-processing',
    '数据处理工作流',
    '数据采集 -> 数据清洗 -> 数据融合 -> 存储',
    'data_processing',
    ARRAY['数据处理', '清洗', '融合'],
    'DataAnalysis',
    '{
        "nodes": [
            {
                "id": "data_source_1",
                "type": "data_source",
                "name": "数据源",
                "config": {"source_id": "${source_id}"},
                "position": {"x": 100, "y": 100},
                "inputs": [],
                "outputs": [{"id": "output_1", "label": "原始数据", "type": "data", "param_name": "raw_data", "data_type": "object"}]
            },
            {
                "id": "filter_1",
                "type": "filter",
                "name": "数据过滤",
                "config": {"filter_rule": "${filter_rule}"},
                "position": {"x": 300, "y": 100},
                "inputs": [{"id": "input_1", "label": "输入数据", "type": "data", "param_name": "input_data", "data_type": "object", "required": true}],
                "outputs": [{"id": "output_1", "label": "过滤后数据", "type": "data", "param_name": "filtered_data", "data_type": "object"}]
            },
            {
                "id": "transform_1",
                "type": "transform",
                "name": "数据转换",
                "config": {"transform_config": "${transform_config}"},
                "position": {"x": 500, "y": 100},
                "inputs": [{"id": "input_1", "label": "输入数据", "type": "data", "param_name": "input_data", "data_type": "object", "required": true}],
                "outputs": [{"id": "output_1", "label": "转换后数据", "type": "data", "param_name": "transformed_data", "data_type": "object"}]
            },
            {
                "id": "output_1",
                "type": "output",
                "name": "数据输出",
                "config": {"output_type": "${output_type}"},
                "position": {"x": 700, "y": 100},
                "inputs": [{"id": "input_1", "label": "输出数据", "type": "data", "param_name": "output_data", "data_type": "object", "required": true}],
                "outputs": []
            }
        ],
        "edges": [
            {
                "id": "edge_1",
                "source": "data_source_1",
                "target": "filter_1",
                "source_port": "output_1",
                "target_port": "input_1",
                "param_mapping": {"input_data": "raw_data"}
            },
            {
                "id": "edge_2",
                "source": "filter_1",
                "target": "transform_1",
                "source_port": "output_1",
                "target_port": "input_1",
                "param_mapping": {"input_data": "filtered_data"}
            },
            {
                "id": "edge_3",
                "source": "transform_1",
                "target": "output_1",
                "source_port": "output_1",
                "target_port": "input_1",
                "param_mapping": {"output_data": "transformed_data"}
            }
        ],
        "variables": []
    }',
    '{
        "source_id": {"type": "string", "description": "数据源ID", "required": true},
        "filter_rule": {"type": "string", "description": "过滤规则", "default": ""},
        "transform_config": {"type": "object", "description": "转换配置", "default": {}},
        "output_type": {"type": "string", "description": "输出类型", "default": "database"}
    }',
    'system'
)
ON CONFLICT (id) DO NOTHING;
