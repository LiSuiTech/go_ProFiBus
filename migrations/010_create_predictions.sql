-- ============================================
-- Prediction Analysis Tables
-- ============================================

-- 创建预测结果表
CREATE TABLE IF NOT EXISTS predictions (
    id VARCHAR(64) PRIMARY KEY,
    model_id VARCHAR(64) NOT NULL, -- 模型ID
    device_id VARCHAR(64) REFERENCES devices(id) ON DELETE SET NULL,
    channel_id VARCHAR(64) REFERENCES channels(id) ON DELETE SET NULL,
    prediction_type VARCHAR(50) NOT NULL, -- forecast, anomaly, trend, performance
    field_name VARCHAR(255), -- 预测的字段名
    predicted_value DOUBLE PRECISION NOT NULL, -- 预测值
    confidence DOUBLE PRECISION DEFAULT 0.0, -- 置信度 0-1
    actual_value DOUBLE PRECISION, -- 实际值（用于对比）
    error_rate DOUBLE PRECISION, -- 误差率
    time_range_start TIMESTAMP NOT NULL, -- 预测时间范围开始
    time_range_end TIMESTAMP NOT NULL, -- 预测时间范围结束
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB NOT NULL DEFAULT '{}' -- 预测元数据
);

-- 创建预测模型表
CREATE TABLE IF NOT EXISTS prediction_models (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- linear_regression, neural_network, svm, etc
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    file_path VARCHAR(500), -- 模型文件路径
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, training, deployed, archived
    accuracy DOUBLE PRECISION, -- 模型准确度
    training_samples INTEGER DEFAULT 0, -- 训练样本数
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deployed_at TIMESTAMP,
    metadata JSONB NOT NULL DEFAULT '{}' -- 模型元数据
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_predictions_model_id ON predictions(model_id);
CREATE INDEX IF NOT EXISTS idx_predictions_device_id ON predictions(device_id);
CREATE INDEX IF NOT EXISTS idx_predictions_channel_id ON predictions(channel_id);
CREATE INDEX IF NOT EXISTS idx_predictions_type ON predictions(prediction_type);
CREATE INDEX IF NOT EXISTS idx_predictions_time_range ON predictions(time_range_start, time_range_end);
CREATE INDEX IF NOT EXISTS idx_prediction_models_type ON prediction_models(type);
CREATE INDEX IF NOT EXISTS idx_prediction_models_status ON prediction_models(status);

-- 添加注释
COMMENT ON TABLE predictions IS '预测结果表';
COMMENT ON TABLE prediction_models IS '预测模型表';
COMMENT ON COLUMN predictions.prediction_type IS '预测类型: forecast(趋势预测), anomaly(异常预测), trend(趋势分析), performance(性能预测)';
COMMENT ON COLUMN predictions.confidence IS '预测置信度 0-1';
COMMENT ON COLUMN prediction_models.status IS '模型状态: draft, training, deployed, archived';
