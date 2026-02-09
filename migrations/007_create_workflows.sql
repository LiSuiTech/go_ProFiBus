-- 创建工作流表
CREATE TABLE IF NOT EXISTS workflows (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    nodes       JSONB NOT NULL DEFAULT '[]',
    edges       JSONB NOT NULL DEFAULT '[]',
    variables   JSONB NOT NULL DEFAULT '[]',
    status      TEXT NOT NULL DEFAULT 'draft',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    created_by  TEXT
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
CREATE INDEX IF NOT EXISTS idx_workflows_created_by ON workflows(created_by);
CREATE INDEX IF NOT EXISTS idx_workflows_created_at ON workflows(created_at DESC);

-- 创建工作流执行表
CREATE TABLE IF NOT EXISTS workflow_executions (
    id            TEXT PRIMARY KEY,
    workflow_id   TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status        TEXT NOT NULL,
    node_statuses JSONB NOT NULL DEFAULT '{}',
    variables     JSONB NOT NULL DEFAULT '{}',
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    error         TEXT
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_workflow_executions_workflow_id ON workflow_executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_started_at ON workflow_executions(started_at DESC);

-- 添加注释
COMMENT ON TABLE workflows IS '工作流定义表';
COMMENT ON TABLE workflow_executions IS '工作流执行实例表';
