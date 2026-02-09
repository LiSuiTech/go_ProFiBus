-- ============================================
-- ML Model Training Tables
-- ============================================

-- 创建训练任务表
CREATE TABLE IF NOT EXISTS training_tasks (
    id VARCHAR(64) PRIMARY KEY,
    model_id VARCHAR(64) NOT NULL REFERENCES prediction_models(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    training_type VARCHAR(50) NOT NULL, -- supervised, unsupervised, reinforcement
    data_source_type VARCHAR(50) NOT NULL, -- device, channel, fusion, external
    data_source_ids TEXT[] NOT NULL, -- 数据源ID列表（设备ID、通道ID等）
    data_fields TEXT[] NOT NULL, -- 训练字段列表
    start_time TIMESTAMP, -- 训练开始时间
    end_time TIMESTAMP, -- 训练结束时间
    progress DOUBLE PRECISION DEFAULT 0.0, -- 训练进度 0-1
    epochs INTEGER DEFAULT 100, -- 训练轮数
    batch_size INTEGER DEFAULT 32, -- 批次大小
    learning_rate DOUBLE PRECISION DEFAULT 0.001, -- 学习率
    validation_split DOUBLE PRECISION DEFAULT 0.2, -- 验证集比例
    hyperparameters JSONB NOT NULL DEFAULT '{}', -- 超参数配置
    training_config JSONB NOT NULL DEFAULT '{}', -- 训练配置
    metrics JSONB NOT NULL DEFAULT '{}', -- 训练指标（loss, accuracy等）
    error_message TEXT, -- 错误信息
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255), -- 创建者
    metadata JSONB NOT NULL DEFAULT '{}' -- 任务元数据
);

-- 创建训练数据样本表（用于存储训练数据）
CREATE TABLE IF NOT EXISTS training_samples (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL REFERENCES training_tasks(id) ON DELETE CASCADE,
    sample_index INTEGER NOT NULL, -- 样本索引
    input_data JSONB NOT NULL, -- 输入数据
    output_data JSONB, -- 输出数据（监督学习）
    label TEXT, -- 标签（分类任务）
    timestamp TIMESTAMP NOT NULL, -- 数据时间戳
    quality DOUBLE PRECISION DEFAULT 1.0, -- 数据质量
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建训练历史记录表（用于记录训练过程）
CREATE TABLE IF NOT EXISTS training_history (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL REFERENCES training_tasks(id) ON DELETE CASCADE,
    epoch INTEGER NOT NULL, -- 训练轮次
    step INTEGER NOT NULL, -- 训练步数
    loss DOUBLE PRECISION, -- 损失值
    accuracy DOUBLE PRECISION, -- 准确度
    validation_loss DOUBLE PRECISION, -- 验证损失
    validation_accuracy DOUBLE PRECISION, -- 验证准确度
    learning_rate DOUBLE PRECISION, -- 当前学习率
    metrics JSONB NOT NULL DEFAULT '{}', -- 其他指标
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_training_tasks_model_id ON training_tasks(model_id);
CREATE INDEX IF NOT EXISTS idx_training_tasks_status ON training_tasks(status);
CREATE INDEX IF NOT EXISTS idx_training_tasks_created_at ON training_tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_training_samples_task_id ON training_samples(task_id);
CREATE INDEX IF NOT EXISTS idx_training_samples_task_index ON training_samples(task_id, sample_index);
CREATE INDEX IF NOT EXISTS idx_training_history_task_id ON training_history(task_id);
CREATE INDEX IF NOT EXISTS idx_training_history_task_epoch ON training_history(task_id, epoch);

-- 添加注释
COMMENT ON TABLE training_tasks IS 'ML模型训练任务表';
COMMENT ON TABLE training_samples IS '训练数据样本表';
COMMENT ON TABLE training_history IS '训练历史记录表';
COMMENT ON COLUMN training_tasks.status IS '任务状态: pending, running, completed, failed, cancelled';
COMMENT ON COLUMN training_tasks.training_type IS '训练类型: supervised(监督学习), unsupervised(无监督学习), reinforcement(强化学习)';
COMMENT ON COLUMN training_tasks.data_source_type IS '数据源类型: device, channel, fusion, external';
COMMENT ON COLUMN training_tasks.progress IS '训练进度 0-1';
COMMENT ON COLUMN training_tasks.metrics IS '训练指标JSON: {loss: 0.1, accuracy: 0.95, ...}';
