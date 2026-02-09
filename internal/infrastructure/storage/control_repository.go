package storage

import (
	"context"
	"encoding/json"
	"fmt"
	controlDomain "go_ProFiBus/internal/domain/control"
	"go_ProFiBus/pkg/interfaces"
	"time"

	"github.com/jackc/pgx/v5"
)

// ControlRepositoryImpl 控制仓储实现
type ControlRepositoryImpl struct {
	store *PostgresStore
}

// NewControlRepository 创建控制仓储
func NewControlRepository(store *PostgresStore) interfaces.ControlRepository {
	return &ControlRepositoryImpl{store: store}
}

// CreateControlPolicy 创建控制策略
func (r *ControlRepositoryImpl) CreateControlPolicy(ctx context.Context, policy *controlDomain.ControlPolicy) error {
	conditionJSON, err := json.Marshal(policy.ConditionConfig)
	if err != nil {
		return fmt.Errorf("序列化条件配置失败: %w", err)
	}

	actionJSON, err := json.Marshal(policy.ActionConfig)
	if err != nil {
		return fmt.Errorf("序列化动作配置失败: %w", err)
	}

	metadataJSON, err := json.Marshal(policy.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO control_policies (id, name, description, enabled, priority, condition_config,
		                            action_config, cooldown_seconds, max_executions, execution_count,
		                            last_executed_at, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = r.store.Exec(query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.Enabled,
		policy.Priority,
		conditionJSON,
		actionJSON,
		policy.CooldownSeconds,
		policy.MaxExecutions,
		policy.ExecutionCount,
		policy.LastExecutedAt,
		metadataJSON,
		policy.CreatedAt,
		policy.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建控制策略失败: %w", err)
	}

	return nil
}

// GetControlPolicyByID 根据ID获取控制策略
func (r *ControlRepositoryImpl) GetControlPolicyByID(ctx context.Context, id string) (*controlDomain.ControlPolicy, error) {
	query := `
		SELECT id, name, description, enabled, priority, condition_config, action_config,
		       cooldown_seconds, max_executions, execution_count, last_executed_at,
		       metadata, created_at, updated_at
		FROM control_policies
		WHERE id = $1
	`

	var policy controlDomain.ControlPolicy
	var conditionJSON, actionJSON, metadataJSON []byte

	err := r.store.QueryRow(query, id).Scan(
		&policy.ID,
		&policy.Name,
		&policy.Description,
		&policy.Enabled,
		&policy.Priority,
		&conditionJSON,
		&actionJSON,
		&policy.CooldownSeconds,
		&policy.MaxExecutions,
		&policy.ExecutionCount,
		&policy.LastExecutedAt,
		&metadataJSON,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("控制策略不存在: %s", id)
		}
		return nil, fmt.Errorf("查询控制策略失败: %w", err)
	}

	if err := json.Unmarshal(conditionJSON, &policy.ConditionConfig); err != nil {
		r.store.log.Warn("反序列化条件配置失败: %v", err)
		policy.ConditionConfig = make(map[string]interface{})
	}

	if err := json.Unmarshal(actionJSON, &policy.ActionConfig); err != nil {
		r.store.log.Warn("反序列化动作配置失败: %v", err)
		policy.ActionConfig = make(map[string]interface{})
	}

	if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
		r.store.log.Warn("反序列化元数据失败: %v", err)
		policy.Metadata = make(map[string]interface{})
	}

	return &policy, nil
}

// ListControlPolicies 列出控制策略
func (r *ControlRepositoryImpl) ListControlPolicies(ctx context.Context, filters interfaces.ControlPolicyFilters) ([]*controlDomain.ControlPolicy, error) {
	query := `
		SELECT id, name, description, enabled, priority, condition_config, action_config,
		       cooldown_seconds, max_executions, execution_count, last_executed_at,
		       metadata, created_at, updated_at
		FROM control_policies
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.Enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, *filters.Enabled)
		argIndex++
	}

	query += " ORDER BY priority DESC, created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询控制策略列表失败: %w", err)
	}
	defer rows.Close()

	policies := make([]*controlDomain.ControlPolicy, 0)
	for rows.Next() {
		var policy controlDomain.ControlPolicy
		var conditionJSON, actionJSON, metadataJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.Name,
			&policy.Description,
			&policy.Enabled,
			&policy.Priority,
			&conditionJSON,
			&actionJSON,
			&policy.CooldownSeconds,
			&policy.MaxExecutions,
			&policy.ExecutionCount,
			&policy.LastExecutedAt,
			&metadataJSON,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描控制策略失败: %v", err)
			continue
		}

		if err := json.Unmarshal(conditionJSON, &policy.ConditionConfig); err != nil {
			r.store.log.Warn("反序列化条件配置失败: %v", err)
			policy.ConditionConfig = make(map[string]interface{})
		}

		if err := json.Unmarshal(actionJSON, &policy.ActionConfig); err != nil {
			r.store.log.Warn("反序列化动作配置失败: %v", err)
			policy.ActionConfig = make(map[string]interface{})
		}

		if err := json.Unmarshal(metadataJSON, &policy.Metadata); err != nil {
			r.store.log.Warn("反序列化元数据失败: %v", err)
			policy.Metadata = make(map[string]interface{})
		}

		policies = append(policies, &policy)
	}

	return policies, nil
}

// UpdateControlPolicy 更新控制策略
func (r *ControlRepositoryImpl) UpdateControlPolicy(ctx context.Context, policy *controlDomain.ControlPolicy) error {
	conditionJSON, err := json.Marshal(policy.ConditionConfig)
	if err != nil {
		return fmt.Errorf("序列化条件配置失败: %w", err)
	}

	actionJSON, err := json.Marshal(policy.ActionConfig)
	if err != nil {
		return fmt.Errorf("序列化动作配置失败: %w", err)
	}

	metadataJSON, err := json.Marshal(policy.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE control_policies
		SET name = $1, description = $2, enabled = $3, priority = $4, condition_config = $5,
		    action_config = $6, cooldown_seconds = $7, max_executions = $8, execution_count = $9,
		    last_executed_at = $10, metadata = $11, updated_at = $12
		WHERE id = $13
	`

	tag, err := r.store.Exec(query,
		policy.Name,
		policy.Description,
		policy.Enabled,
		policy.Priority,
		conditionJSON,
		actionJSON,
		policy.CooldownSeconds,
		policy.MaxExecutions,
		policy.ExecutionCount,
		policy.LastExecutedAt,
		metadataJSON,
		time.Now(),
		policy.ID,
	)

	if err != nil {
		return fmt.Errorf("更新控制策略失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("控制策略不存在: %s", policy.ID)
	}

	return nil
}

// DeleteControlPolicy 删除控制策略
func (r *ControlRepositoryImpl) DeleteControlPolicy(ctx context.Context, id string) error {
	query := `DELETE FROM control_policies WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除控制策略失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("控制策略不存在: %s", id)
	}

	return nil
}

// CreateControlAction 创建控制动作
func (r *ControlRepositoryImpl) CreateControlAction(ctx context.Context, action *controlDomain.ControlAction) error {
	parametersJSON, err := json.Marshal(action.Parameters)
	if err != nil {
		return fmt.Errorf("序列化参数失败: %w", err)
	}

	resultJSON, _ := json.Marshal(action.Result)
	metadataJSON, err := json.Marshal(action.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO control_actions (id, policy_id, device_id, action_type, parameters, reason,
		                            severity, status, result, error_message, executed_by,
		                            executed_at, completed_at, duration_ms, require_confirmation,
		                            confirmed_by, confirmed_at, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err = r.store.Exec(query,
		action.ID,
		action.PolicyID,
		action.DeviceID,
		string(action.ActionType),
		parametersJSON,
		action.Reason,
		action.Severity,
		string(action.Status),
		resultJSON,
		action.ErrorMessage,
		action.ExecutedBy,
		action.ExecutedAt,
		action.CompletedAt,
		action.DurationMs,
		action.RequireConfirmation,
		action.ConfirmedBy,
		action.ConfirmedAt,
		metadataJSON,
		action.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建控制动作失败: %w", err)
	}

	return nil
}

// GetControlActionByID 根据ID获取控制动作
func (r *ControlRepositoryImpl) GetControlActionByID(ctx context.Context, id string) (*controlDomain.ControlAction, error) {
	query := `
		SELECT id, policy_id, device_id, action_type, parameters, reason, severity, status,
		       result, error_message, executed_by, executed_at, completed_at, duration_ms,
		       require_confirmation, confirmed_by, confirmed_at, metadata, created_at
		FROM control_actions
		WHERE id = $1
	`

	var action controlDomain.ControlAction
	var actionType, status string
	var parametersJSON, resultJSON, metadataJSON []byte

	err := r.store.QueryRow(query, id).Scan(
		&action.ID,
		&action.PolicyID,
		&action.DeviceID,
		&actionType,
		&parametersJSON,
		&action.Reason,
		&action.Severity,
		&status,
		&resultJSON,
		&action.ErrorMessage,
		&action.ExecutedBy,
		&action.ExecutedAt,
		&action.CompletedAt,
		&action.DurationMs,
		&action.RequireConfirmation,
		&action.ConfirmedBy,
		&action.ConfirmedAt,
		&metadataJSON,
		&action.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("控制动作不存在: %s", id)
		}
		return nil, fmt.Errorf("查询控制动作失败: %w", err)
	}

	action.ActionType = controlDomain.ActionType(actionType)
	action.Status = controlDomain.ActionStatus(status)

	if err := json.Unmarshal(parametersJSON, &action.Parameters); err != nil {
		r.store.log.Warn("反序列化参数失败: %v", err)
		action.Parameters = make(map[string]interface{})
	}

	if len(resultJSON) > 0 {
		if err := json.Unmarshal(resultJSON, &action.Result); err != nil {
			r.store.log.Warn("反序列化结果失败: %v", err)
			action.Result = make(map[string]interface{})
		}
	}

	if err := json.Unmarshal(metadataJSON, &action.Metadata); err != nil {
		r.store.log.Warn("反序列化元数据失败: %v", err)
		action.Metadata = make(map[string]interface{})
	}

	return &action, nil
}

// ListControlActions 列出控制动作
func (r *ControlRepositoryImpl) ListControlActions(ctx context.Context, filters interfaces.ControlActionFilters) ([]*controlDomain.ControlAction, error) {
	query := `
		SELECT id, policy_id, device_id, action_type, parameters, reason, severity, status,
		       result, error_message, executed_by, executed_at, completed_at, duration_ms,
		       require_confirmation, confirmed_by, confirmed_at, metadata, created_at
		FROM control_actions
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.PolicyID != nil {
		query += fmt.Sprintf(" AND policy_id = $%d", argIndex)
		args = append(args, *filters.PolicyID)
		argIndex++
	}

	if filters.DeviceID != nil {
		query += fmt.Sprintf(" AND device_id = $%d", argIndex)
		args = append(args, *filters.DeviceID)
		argIndex++
	}

	if filters.ActionType != nil {
		query += fmt.Sprintf(" AND action_type = $%d", argIndex)
		args = append(args, string(*filters.ActionType))
		argIndex++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
		argIndex++
	}

	if filters.ExecutedBy != nil {
		query += fmt.Sprintf(" AND executed_by = $%d", argIndex)
		args = append(args, *filters.ExecutedBy)
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND created_at < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询控制动作列表失败: %w", err)
	}
	defer rows.Close()

	actions := make([]*controlDomain.ControlAction, 0)
	for rows.Next() {
		var action controlDomain.ControlAction
		var actionType, status string
		var parametersJSON, resultJSON, metadataJSON []byte

		err := rows.Scan(
			&action.ID,
			&action.PolicyID,
			&action.DeviceID,
			&actionType,
			&parametersJSON,
			&action.Reason,
			&action.Severity,
			&status,
			&resultJSON,
			&action.ErrorMessage,
			&action.ExecutedBy,
			&action.ExecutedAt,
			&action.CompletedAt,
			&action.DurationMs,
			&action.RequireConfirmation,
			&action.ConfirmedBy,
			&action.ConfirmedAt,
			&metadataJSON,
			&action.CreatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描控制动作失败: %v", err)
			continue
		}

		action.ActionType = controlDomain.ActionType(actionType)
		action.Status = controlDomain.ActionStatus(status)

		if err := json.Unmarshal(parametersJSON, &action.Parameters); err != nil {
			r.store.log.Warn("反序列化参数失败: %v", err)
			action.Parameters = make(map[string]interface{})
		}

		if len(resultJSON) > 0 {
			if err := json.Unmarshal(resultJSON, &action.Result); err != nil {
				r.store.log.Warn("反序列化结果失败: %v", err)
				action.Result = make(map[string]interface{})
			}
		}

		if err := json.Unmarshal(metadataJSON, &action.Metadata); err != nil {
			r.store.log.Warn("反序列化元数据失败: %v", err)
			action.Metadata = make(map[string]interface{})
		}

		actions = append(actions, &action)
	}

	return actions, nil
}

// UpdateControlAction 更新控制动作
func (r *ControlRepositoryImpl) UpdateControlAction(ctx context.Context, action *controlDomain.ControlAction) error {
	resultJSON, _ := json.Marshal(action.Result)
	metadataJSON, err := json.Marshal(action.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		UPDATE control_actions
		SET status = $1, result = $2, error_message = $3, executed_by = $4,
		    executed_at = $5, completed_at = $6, duration_ms = $7,
		    confirmed_by = $8, confirmed_at = $9, metadata = $10
		WHERE id = $11
	`

	tag, err := r.store.Exec(query,
		string(action.Status),
		resultJSON,
		action.ErrorMessage,
		action.ExecutedBy,
		action.ExecutedAt,
		action.CompletedAt,
		action.DurationMs,
		action.ConfirmedBy,
		action.ConfirmedAt,
		metadataJSON,
		action.ID,
	)

	if err != nil {
		return fmt.Errorf("更新控制动作失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("控制动作不存在: %s", action.ID)
	}

	return nil
}

// CreateAuditLog 创建审计日志
func (r *ControlRepositoryImpl) CreateAuditLog(ctx context.Context, log *controlDomain.AuditLog) error {
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("序列化详细信息失败: %w", err)
	}

	query := `
		INSERT INTO control_audit_logs (id, action_id, event_type, user_id, user_name,
		                               details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.store.Exec(query,
		log.ID,
		log.ActionID,
		string(log.EventType),
		log.UserID,
		log.UserName,
		detailsJSON,
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建审计日志失败: %w", err)
	}

	return nil
}

// GetAuditLogs 获取审计日志
func (r *ControlRepositoryImpl) GetAuditLogs(ctx context.Context, filters interfaces.AuditLogFilters) ([]*controlDomain.AuditLog, error) {
	query := `
		SELECT id, action_id, event_type, user_id, user_name, details,
		       ip_address, user_agent, created_at
		FROM control_audit_logs
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.ActionID != nil {
		query += fmt.Sprintf(" AND action_id = $%d", argIndex)
		args = append(args, *filters.ActionID)
		argIndex++
	}

	if filters.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *filters.UserID)
		argIndex++
	}

	if filters.EventType != nil {
		query += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, string(*filters.EventType))
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND created_at < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询审计日志失败: %w", err)
	}
	defer rows.Close()

	logs := make([]*controlDomain.AuditLog, 0)
	for rows.Next() {
		var log controlDomain.AuditLog
		var eventType string
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.ActionID,
			&eventType,
			&log.UserID,
			&log.UserName,
			&detailsJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描审计日志失败: %v", err)
			continue
		}

		log.EventType = controlDomain.AuditEventType(eventType)

		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			r.store.log.Warn("反序列化详细信息失败: %v", err)
			log.Details = make(map[string]interface{})
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

// CreateControlPermission 创建控制权限
func (r *ControlRepositoryImpl) CreateControlPermission(ctx context.Context, permission *controlDomain.ControlPermission) error {
	timeRangesJSON, err := json.Marshal(permission.AllowedTimeRanges)
	if err != nil {
		return fmt.Errorf("序列化时间范围失败: %w", err)
	}

	query := `
		INSERT INTO control_permissions (id, user_id, action_type, target_devices, max_severity,
		                               require_confirmation, allowed_time_ranges, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, action_type) DO UPDATE SET
			target_devices = EXCLUDED.target_devices,
			max_severity = EXCLUDED.max_severity,
			require_confirmation = EXCLUDED.require_confirmation,
			allowed_time_ranges = EXCLUDED.allowed_time_ranges,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.store.Exec(query,
		permission.ID,
		permission.UserID,
		string(permission.ActionType),
		permission.TargetDevices,
		permission.MaxSeverity,
		permission.RequireConfirmation,
		timeRangesJSON,
		permission.Enabled,
		permission.CreatedAt,
		permission.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建控制权限失败: %w", err)
	}

	return nil
}

// GetControlPermission 获取控制权限
func (r *ControlRepositoryImpl) GetControlPermission(ctx context.Context, userID string, actionType controlDomain.ActionType) (*controlDomain.ControlPermission, error) {
	query := `
		SELECT id, user_id, action_type, target_devices, max_severity,
		       require_confirmation, allowed_time_ranges, enabled, created_at, updated_at
		FROM control_permissions
		WHERE user_id = $1 AND action_type = $2
	`

	var permission controlDomain.ControlPermission
	var timeRangesJSON []byte

	err := r.store.QueryRow(query, userID, string(actionType)).Scan(
		&permission.ID,
		&permission.UserID,
		&permission.ActionType,
		&permission.TargetDevices,
		&permission.MaxSeverity,
		&permission.RequireConfirmation,
		&timeRangesJSON,
		&permission.Enabled,
		&permission.CreatedAt,
		&permission.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("控制权限不存在: %s/%s", userID, actionType)
		}
		return nil, fmt.Errorf("查询控制权限失败: %w", err)
	}

	if err := json.Unmarshal(timeRangesJSON, &permission.AllowedTimeRanges); err != nil {
		r.store.log.Warn("反序列化时间范围失败: %v", err)
		permission.AllowedTimeRanges = make([]controlDomain.TimeRange, 0)
	}

	return &permission, nil
}

// ListControlPermissions 列出控制权限
func (r *ControlRepositoryImpl) ListControlPermissions(ctx context.Context, filters interfaces.ControlPermissionFilters) ([]*controlDomain.ControlPermission, error) {
	query := `
		SELECT id, user_id, action_type, target_devices, max_severity,
		       require_confirmation, allowed_time_ranges, enabled, created_at, updated_at
		FROM control_permissions
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *filters.UserID)
		argIndex++
	}

	if filters.ActionType != nil {
		query += fmt.Sprintf(" AND action_type = $%d", argIndex)
		args = append(args, string(*filters.ActionType))
		argIndex++
	}

	if filters.Enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, *filters.Enabled)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询控制权限列表失败: %w", err)
	}
	defer rows.Close()

	permissions := make([]*controlDomain.ControlPermission, 0)
	for rows.Next() {
		var permission controlDomain.ControlPermission
		var timeRangesJSON []byte

		err := rows.Scan(
			&permission.ID,
			&permission.UserID,
			&permission.ActionType,
			&permission.TargetDevices,
			&permission.MaxSeverity,
			&permission.RequireConfirmation,
			&timeRangesJSON,
			&permission.Enabled,
			&permission.CreatedAt,
			&permission.UpdatedAt,
		)
		if err != nil {
			r.store.log.Warn("扫描控制权限失败: %v", err)
			continue
		}

		if err := json.Unmarshal(timeRangesJSON, &permission.AllowedTimeRanges); err != nil {
			r.store.log.Warn("反序列化时间范围失败: %v", err)
			permission.AllowedTimeRanges = make([]controlDomain.TimeRange, 0)
		}

		permissions = append(permissions, &permission)
	}

	return permissions, nil
}

// UpdateControlPermission 更新控制权限
func (r *ControlRepositoryImpl) UpdateControlPermission(ctx context.Context, permission *controlDomain.ControlPermission) error {
	timeRangesJSON, err := json.Marshal(permission.AllowedTimeRanges)
	if err != nil {
		return fmt.Errorf("序列化时间范围失败: %w", err)
	}

	query := `
		UPDATE control_permissions
		SET target_devices = $1, max_severity = $2, require_confirmation = $3,
		    allowed_time_ranges = $4, enabled = $5, updated_at = $6
		WHERE id = $7
	`

	tag, err := r.store.Exec(query,
		permission.TargetDevices,
		permission.MaxSeverity,
		permission.RequireConfirmation,
		timeRangesJSON,
		permission.Enabled,
		time.Now(),
		permission.ID,
	)

	if err != nil {
		return fmt.Errorf("更新控制权限失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("控制权限不存在: %s", permission.ID)
	}

	return nil
}

// DeleteControlPermission 删除控制权限
func (r *ControlRepositoryImpl) DeleteControlPermission(ctx context.Context, id string) error {
	query := `DELETE FROM control_permissions WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除控制权限失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("控制权限不存在: %s", id)
	}

	return nil
}
