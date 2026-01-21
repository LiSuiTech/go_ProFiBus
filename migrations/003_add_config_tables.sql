-- Migration: Add configuration tables for Phase 3 Algorithm Configuration System
-- Description: Creates tables for rule, analyzer, and processor configurations,
--              along with templates and history tracking

-- Rule configurations table
CREATE TABLE IF NOT EXISTS rule_configs (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- 'threshold', 'statistical', 'ml'
    version VARCHAR(50) NOT NULL,
    parameters JSONB NOT NULL,
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255)
);

-- Analyzer configurations table
CREATE TABLE IF NOT EXISTS analyzer_configs (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- 'statistical', 'ml', 'ensemble'
    version VARCHAR(50) NOT NULL,
    parameters JSONB NOT NULL,
    rule_ids TEXT[], -- Array of rule IDs
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255)
);

-- Processor configurations table
CREATE TABLE IF NOT EXISTS processor_configs (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- 'filter', 'transform', 'aggregate'
    version VARCHAR(50) NOT NULL,
    parameters JSONB NOT NULL,
    "order" INTEGER DEFAULT 0, -- Processing order
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255)
);

-- Configuration templates table
CREATE TABLE IF NOT EXISTS config_templates (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- 'rule', 'analyzer', 'processor'
    category VARCHAR(100),
    schema JSONB NOT NULL, -- JSON schema definition
    defaults JSONB, -- Default parameter values
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Configuration history table
CREATE TABLE IF NOT EXISTS config_history (
    id SERIAL PRIMARY KEY,
    config_id VARCHAR(255) NOT NULL,
    config_type VARCHAR(50) NOT NULL, -- 'rule', 'analyzer', 'processor'
    action VARCHAR(50) NOT NULL, -- 'created', 'updated', 'deleted'
    changed_by VARCHAR(255),
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    old_value JSONB,
    new_value JSONB,
    reason TEXT
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_rule_configs_type ON rule_configs(type);
CREATE INDEX IF NOT EXISTS idx_rule_configs_enabled ON rule_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_rule_configs_created_at ON rule_configs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rule_configs_priority ON rule_configs(priority DESC);

CREATE INDEX IF NOT EXISTS idx_analyzer_configs_type ON analyzer_configs(type);
CREATE INDEX IF NOT EXISTS idx_analyzer_configs_enabled ON analyzer_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_analyzer_configs_created_at ON analyzer_configs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_processor_configs_type ON processor_configs(type);
CREATE INDEX IF NOT EXISTS idx_processor_configs_enabled ON processor_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_processor_configs_order ON processor_configs("order");
CREATE INDEX IF NOT EXISTS idx_processor_configs_created_at ON processor_configs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_config_templates_type ON config_templates(type);
CREATE INDEX IF NOT EXISTS idx_config_templates_category ON config_templates(category);

CREATE INDEX IF NOT EXISTS idx_config_history_config_id ON config_history(config_id);
CREATE INDEX IF NOT EXISTS idx_config_history_config_type ON config_history(config_type);
CREATE INDEX IF NOT EXISTS idx_config_history_changed_at ON config_history(changed_at DESC);

-- Create trigger function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for all config tables
CREATE TRIGGER update_rule_configs_updated_at
    BEFORE UPDATE ON rule_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_analyzer_configs_updated_at
    BEFORE UPDATE ON analyzer_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_processor_configs_updated_at
    BEFORE UPDATE ON processor_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_config_templates_updated_at
    BEFORE UPDATE ON config_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default templates

-- Threshold rule template
INSERT INTO config_templates (id, name, description, type, category, schema, defaults) VALUES
(
    'template-rule-threshold',
    'Threshold Rule Template',
    'Basic threshold-based anomaly detection rule',
    'rule',
    'anomaly_detection',
    '{
        "type": "object",
        "properties": {
            "metric": {
                "type": "string",
                "description": "Metric field to monitor"
            },
            "threshold": {
                "type": "number",
                "description": "Threshold value"
            },
            "operator": {
                "type": "string",
                "enum": ["gt", "lt", "eq", "gte", "lte"],
                "description": "Comparison operator (gt=greater than, lt=less than)"
            },
            "window_size": {
                "type": "integer",
                "description": "Time window size in seconds",
                "minimum": 1
            }
        },
        "required": ["metric", "threshold", "operator"]
    }',
    '{
        "operator": "gt",
        "window_size": 60
    }'
),
(
    'template-rule-statistical',
    'Statistical Rule Template',
    'Statistical anomaly detection using Z-score',
    'rule',
    'anomaly_detection',
    '{
        "type": "object",
        "properties": {
            "metric": {
                "type": "string",
                "description": "Metric field to monitor"
            },
            "zscore_threshold": {
                "type": "number",
                "description": "Z-score threshold (typically 2 or 3)",
                "minimum": 0,
                "default": 3
            },
            "window_size": {
                "type": "integer",
                "description": "Rolling window size for calculating statistics",
                "minimum": 10,
                "default": 100
            },
            "min_samples": {
                "type": "integer",
                "description": "Minimum samples before detection starts",
                "minimum": 10,
                "default": 30
            }
        },
        "required": ["metric", "zscore_threshold"]
    }',
    '{
        "zscore_threshold": 3,
        "window_size": 100,
        "min_samples": 30
    }'
),
(
    'template-analyzer-ensemble',
    'Ensemble Analyzer Template',
    'Combines multiple rules using voting or averaging',
    'analyzer',
    'ensemble',
    '{
        "type": "object",
        "properties": {
            "strategy": {
                "type": "string",
                "enum": ["majority_vote", "unanimous", "weighted_average"],
                "description": "Aggregation strategy"
            },
            "weights": {
                "type": "object",
                "description": "Rule weights for weighted_average strategy"
            },
            "threshold": {
                "type": "number",
                "description": "Confidence threshold for classification",
                "minimum": 0,
                "maximum": 1,
                "default": 0.5
            }
        },
        "required": ["strategy"]
    }',
    '{
        "strategy": "majority_vote",
        "threshold": 0.5
    }'
),
(
    'template-processor-filter',
    'Filter Processor Template',
    'Filters data based on conditions',
    'processor',
    'data_processing',
    '{
        "type": "object",
        "properties": {
            "field": {
                "type": "string",
                "description": "Field to filter on"
            },
            "operator": {
                "type": "string",
                "enum": ["eq", "ne", "gt", "lt", "gte", "lte", "contains", "regex"],
                "description": "Filter operator"
            },
            "value": {
                "description": "Value to compare against"
            },
            "action": {
                "type": "string",
                "enum": ["keep", "drop"],
                "description": "Action when condition matches",
                "default": "keep"
            }
        },
        "required": ["field", "operator", "value"]
    }',
    '{
        "action": "keep"
    }'
),
(
    'template-processor-transform',
    'Transform Processor Template',
    'Transforms data fields using expressions',
    'processor',
    'data_processing',
    '{
        "type": "object",
        "properties": {
            "transformations": {
                "type": "array",
                "description": "List of field transformations",
                "items": {
                    "type": "object",
                    "properties": {
                        "source_field": {
                            "type": "string",
                            "description": "Source field name"
                        },
                        "target_field": {
                            "type": "string",
                            "description": "Target field name"
                        },
                        "operation": {
                            "type": "string",
                            "enum": ["copy", "scale", "normalize", "log", "exp"],
                            "description": "Transformation operation"
                        },
                        "parameters": {
                            "type": "object",
                            "description": "Operation-specific parameters"
                        }
                    },
                    "required": ["source_field", "target_field", "operation"]
                }
            }
        },
        "required": ["transformations"]
    }',
    '{}'
);

-- Add comments to tables
COMMENT ON TABLE rule_configs IS 'Stores anomaly detection rule configurations';
COMMENT ON TABLE analyzer_configs IS 'Stores analyzer configurations that apply rules';
COMMENT ON TABLE processor_configs IS 'Stores data processor configurations';
COMMENT ON TABLE config_templates IS 'Stores reusable configuration templates';
COMMENT ON TABLE config_history IS 'Tracks all configuration changes for audit purposes';

COMMENT ON COLUMN rule_configs.parameters IS 'Rule-specific parameters stored as JSON';
COMMENT ON COLUMN rule_configs.priority IS 'Execution priority (higher values execute first)';
COMMENT ON COLUMN analyzer_configs.rule_ids IS 'Array of rule IDs applied by this analyzer';
COMMENT ON COLUMN processor_configs."order" IS 'Processing order in pipeline (lower values execute first)';
COMMENT ON COLUMN config_templates.schema IS 'JSON schema for validating configuration parameters';
COMMENT ON COLUMN config_history.old_value IS 'Configuration state before change';
COMMENT ON COLUMN config_history.new_value IS 'Configuration state after change';
