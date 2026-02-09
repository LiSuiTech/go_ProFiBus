package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"go_ProFiBus/internal/application/workflow"
	"time"

	"github.com/jackc/pgx/v5"
)

// WorkflowRepositoryImpl 工作流仓储实现
type WorkflowRepositoryImpl struct {
	store *PostgresStore
}

// NewWorkflowRepository 创建工作流仓储
func NewWorkflowRepository(store *PostgresStore) *WorkflowRepositoryImpl {
	return &WorkflowRepositoryImpl{
		store: store,
	}
}

// Save 保存工作流
func (r *WorkflowRepositoryImpl) Save(ctx context.Context, wf *workflow.Workflow) error {
	nodesJSON, err := json.Marshal(wf.Nodes)
	if err != nil {
		return fmt.Errorf("failed to marshal nodes: %w", err)
	}

	edgesJSON, err := json.Marshal(wf.Edges)
	if err != nil {
		return fmt.Errorf("failed to marshal edges: %w", err)
	}

	variablesJSON, err := json.Marshal(wf.Variables)
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	query := `
		INSERT INTO workflows (id, name, description, nodes, edges, variables, status, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			nodes = EXCLUDED.nodes,
			edges = EXCLUDED.edges,
			variables = EXCLUDED.variables,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.store.pool.Exec(ctx, query,
		wf.ID, wf.Name, wf.Description, nodesJSON, edgesJSON, variablesJSON,
		wf.Status, wf.CreatedAt, wf.UpdatedAt, wf.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}

	return nil
}

// GetByID 根据ID获取工作流
func (r *WorkflowRepositoryImpl) GetByID(ctx context.Context, id string) (*workflow.Workflow, error) {
	query := `
		SELECT id, name, description, nodes, edges, variables, status, created_at, updated_at, created_by
		FROM workflows
		WHERE id = $1
	`

	var wf workflow.Workflow
	var nodesJSON, edgesJSON, variablesJSON []byte

	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&wf.ID, &wf.Name, &wf.Description, &nodesJSON, &edgesJSON, &variablesJSON,
		&wf.Status, &wf.CreatedAt, &wf.UpdatedAt, &wf.CreatedBy,
	)

	if err == pgx.ErrNoRows {
		return nil, workflow.ErrWorkflowNotFound{WorkflowID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// 解析JSON
	if err := json.Unmarshal(nodesJSON, &wf.Nodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
	}
	if err := json.Unmarshal(edgesJSON, &wf.Edges); err != nil {
		return nil, fmt.Errorf("failed to unmarshal edges: %w", err)
	}
	if err := json.Unmarshal(variablesJSON, &wf.Variables); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
	}

	return &wf, nil
}

// List 列出工作流
func (r *WorkflowRepositoryImpl) List(ctx context.Context, filters map[string]interface{}) ([]*workflow.Workflow, error) {
	query := `SELECT id, name, description, nodes, edges, variables, status, created_at, updated_at, created_by FROM workflows WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if status, ok := filters["status"].(string); ok {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []*workflow.Workflow
	for rows.Next() {
		var wf workflow.Workflow
		var nodesJSON, edgesJSON, variablesJSON []byte

		err := rows.Scan(
			&wf.ID, &wf.Name, &wf.Description, &nodesJSON, &edgesJSON, &variablesJSON,
			&wf.Status, &wf.CreatedAt, &wf.UpdatedAt, &wf.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		if err := json.Unmarshal(nodesJSON, &wf.Nodes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
		}
		if err := json.Unmarshal(edgesJSON, &wf.Edges); err != nil {
			return nil, fmt.Errorf("failed to unmarshal edges: %w", err)
		}
		if err := json.Unmarshal(variablesJSON, &wf.Variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}

		workflows = append(workflows, &wf)
	}

	return workflows, nil
}

// Delete 删除工作流
func (r *WorkflowRepositoryImpl) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workflows WHERE id = $1`
	result, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	if result.RowsAffected() == 0 {
		return workflow.ErrWorkflowNotFound{WorkflowID: id}
	}

	return nil
}

// SaveExecution 保存执行实例
func (r *WorkflowRepositoryImpl) SaveExecution(ctx context.Context, exec *workflow.WorkflowExecution) error {
	nodeStatusesJSON, err := json.Marshal(exec.NodeStatuses)
	if err != nil {
		return fmt.Errorf("failed to marshal node statuses: %w", err)
	}

	variablesJSON, err := json.Marshal(exec.Variables)
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	query := `
		INSERT INTO workflow_executions (id, workflow_id, status, node_statuses, variables, started_at, completed_at, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			node_statuses = EXCLUDED.node_statuses,
			variables = EXCLUDED.variables,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			error = EXCLUDED.error
	`

	_, err = r.store.pool.Exec(ctx, query,
		exec.ID, exec.WorkflowID, exec.Status, nodeStatusesJSON, variablesJSON,
		exec.StartedAt, exec.CompletedAt, exec.Error,
	)

	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	return nil
}

// GetExecution 获取执行实例
func (r *WorkflowRepositoryImpl) GetExecution(ctx context.Context, id string) (*workflow.WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, status, node_statuses, variables, started_at, completed_at, error
		FROM workflow_executions
		WHERE id = $1
	`

	var exec workflow.WorkflowExecution
	var nodeStatusesJSON, variablesJSON []byte

	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&exec.ID, &exec.WorkflowID, &exec.Status, &nodeStatusesJSON, &variablesJSON,
		&exec.StartedAt, &exec.CompletedAt, &exec.Error,
	)

	if err == pgx.ErrNoRows {
		return nil, workflow.ErrExecutionNotFound{ExecutionID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	if err := json.Unmarshal(nodeStatusesJSON, &exec.NodeStatuses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node statuses: %w", err)
	}
	if err := json.Unmarshal(variablesJSON, &exec.Variables); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
	}

	return &exec, nil
}

// ListExecutions 列出执行实例
func (r *WorkflowRepositoryImpl) ListExecutions(ctx context.Context, workflowID string) ([]*workflow.WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, status, node_statuses, variables, started_at, completed_at, error
		FROM workflow_executions
		WHERE workflow_id = $1
		ORDER BY started_at DESC
	`

	rows, err := r.store.pool.Query(ctx, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*workflow.WorkflowExecution
	for rows.Next() {
		var exec workflow.WorkflowExecution
		var nodeStatusesJSON, variablesJSON []byte

		err := rows.Scan(
			&exec.ID, &exec.WorkflowID, &exec.Status, &nodeStatusesJSON, &variablesJSON,
			&exec.StartedAt, &exec.CompletedAt, &exec.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		if err := json.Unmarshal(nodeStatusesJSON, &exec.NodeStatuses); err != nil {
			return nil, fmt.Errorf("failed to unmarshal node statuses: %w", err)
		}
		if err := json.Unmarshal(variablesJSON, &exec.Variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}

		executions = append(executions, &exec)
	}

	return executions, nil
}

// 确保实现了接口
var _ workflow.WorkflowRepository = (*WorkflowRepositoryImpl)(nil)
