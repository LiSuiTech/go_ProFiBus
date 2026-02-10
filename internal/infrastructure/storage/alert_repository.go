package storage

import (
	"context"
	"encoding/json"
	"fmt"
	alertDomain "go_ProFiBus/internal/domain/alert"
	"go_ProFiBus/pkg/interfaces"
	"time"

	"github.com/jackc/pgx/v5"
)

// AlertRepositoryImpl 告警仓储实现
type AlertRepositoryImpl struct {
	store *PostgresStore
}

// NewAlertRepository 创建告警仓储
func NewAlertRepository(store *PostgresStore) interfaces.AlertRepository {
	return &AlertRepositoryImpl{store: store}
}

// CreateAlert 创建告警
func (r *AlertRepositoryImpl) CreateAlert(ctx context.Context, alert *alertDomain.Alert) error {
	detailsJSON, err := json.Marshal(alert.Details)
	if err != nil {
		return fmt.Errorf("序列化详情失败: %w", err)
	}

	query := `
		INSERT INTO alerts (id, rule_id, device_id, channel_id, event_id, level, status, message, details,
		                   first_occurred_at, last_occurred_at, count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = r.store.Exec(query,
		alert.ID,
		alert.RuleID,
		alert.DeviceID,
		alert.ChannelID,
		alert.EventID,
		string(alert.Level),
		string(alert.Status),
		alert.Message,
		detailsJSON,
		alert.FirstOccurredAt,
		alert.LastOccurredAt,
		alert.Count,
		alert.CreatedAt,
		alert.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建告警失败: %w", err)
	}

	return nil
}

// GetAlertByID 根据ID获取告警
func (r *AlertRepositoryImpl) GetAlertByID(ctx context.Context, id string) (*alertDomain.Alert, error) {
	query := `
		SELECT id, rule_id, device_id, channel_id, event_id, level, status, message, details,
		       first_occurred_at, last_occurred_at, acknowledged_at, acknowledged_by,
		       resolved_at, resolved_by, count, created_at, updated_at
		FROM alerts
		WHERE id = $1
	`

	var alert alertDomain.Alert
	var detailsJSON []byte
	var level, status string

	err := r.store.QueryRow(query, id).Scan(
		&alert.ID,
		&alert.RuleID,
		&alert.DeviceID,
		&alert.ChannelID,
		&alert.EventID,
		&level,
		&status,
		&alert.Message,
		&detailsJSON,
		&alert.FirstOccurredAt,
		&alert.LastOccurredAt,
		&alert.AcknowledgedAt,
		&alert.AcknowledgedBy,
		&alert.ResolvedAt,
		&alert.ResolvedBy,
		&alert.Count,
		&alert.CreatedAt,
		&alert.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("告警不存在: %s", id)
		}
		return nil, fmt.Errorf("查询告警失败: %w", err)
	}

	alert.Level = alertDomain.AlertLevel(level)
	alert.Status = alertDomain.AlertStatus(status)

	if err := json.Unmarshal(detailsJSON, &alert.Details); err != nil {
		r.store.Log().Warn("反序列化详情失败: %v", err)
		alert.Details = make(map[string]interface{})
	}

	return &alert, nil
}

// ListAlerts 列出告警
func (r *AlertRepositoryImpl) ListAlerts(ctx context.Context, filters interfaces.AlertFilters) ([]*alertDomain.Alert, error) {
	query := `
		SELECT id, rule_id, device_id, channel_id, event_id, level, status, message, details,
		       first_occurred_at, last_occurred_at, acknowledged_at, acknowledged_by,
		       resolved_at, resolved_by, count, created_at, updated_at
		FROM alerts
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.RuleID != nil {
		query += fmt.Sprintf(" AND rule_id = $%d", argIndex)
		args = append(args, *filters.RuleID)
		argIndex++
	}

	if filters.DeviceID != nil {
		query += fmt.Sprintf(" AND device_id = $%d", argIndex)
		args = append(args, *filters.DeviceID)
		argIndex++
	}

	if filters.ChannelID != nil {
		query += fmt.Sprintf(" AND channel_id = $%d", argIndex)
		args = append(args, *filters.ChannelID)
		argIndex++
	}

	if filters.Level != nil {
		query += fmt.Sprintf(" AND level = $%d", argIndex)
		args = append(args, string(*filters.Level))
		argIndex++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND first_occurred_at >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND first_occurred_at < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	query += " ORDER BY first_occurred_at DESC"

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
		return nil, fmt.Errorf("查询告警列表失败: %w", err)
	}
	defer rows.Close()

	alerts := make([]*alertDomain.Alert, 0)

	for rows.Next() {
		var alert alertDomain.Alert
		var detailsJSON []byte
		var level, status string

		err := rows.Scan(
			&alert.ID,
			&alert.RuleID,
			&alert.DeviceID,
			&alert.ChannelID,
			&alert.EventID,
			&level,
			&status,
			&alert.Message,
			&detailsJSON,
			&alert.FirstOccurredAt,
			&alert.LastOccurredAt,
			&alert.AcknowledgedAt,
			&alert.AcknowledgedBy,
			&alert.ResolvedAt,
			&alert.ResolvedBy,
			&alert.Count,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描告警失败: %v", err)
			continue
		}

		alert.Level = alertDomain.AlertLevel(level)
		alert.Status = alertDomain.AlertStatus(status)

		if err := json.Unmarshal(detailsJSON, &alert.Details); err != nil {
			r.store.Log().Warn("反序列化详情失败: %v", err)
			alert.Details = make(map[string]interface{})
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// UpdateAlert 更新告警
func (r *AlertRepositoryImpl) UpdateAlert(ctx context.Context, alert *alertDomain.Alert) error {
	detailsJSON, err := json.Marshal(alert.Details)
	if err != nil {
		return fmt.Errorf("序列化详情失败: %w", err)
	}

	query := `
		UPDATE alerts
		SET rule_id = $1, device_id = $2, channel_id = $3, event_id = $4, level = $5, status = $6,
		    message = $7, details = $8, first_occurred_at = $9, last_occurred_at = $10,
		    acknowledged_at = $11, acknowledged_by = $12, resolved_at = $13, resolved_by = $14,
		    count = $15, updated_at = $16
		WHERE id = $17
	`

	tag, err := r.store.Exec(query,
		alert.RuleID,
		alert.DeviceID,
		alert.ChannelID,
		alert.EventID,
		string(alert.Level),
		string(alert.Status),
		alert.Message,
		detailsJSON,
		alert.FirstOccurredAt,
		alert.LastOccurredAt,
		alert.AcknowledgedAt,
		alert.AcknowledgedBy,
		alert.ResolvedAt,
		alert.ResolvedBy,
		alert.Count,
		time.Now(),
		alert.ID,
	)

	if err != nil {
		return fmt.Errorf("更新告警失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("告警不存在: %s", alert.ID)
	}

	return nil
}

// AcknowledgeAlert 确认告警
func (r *AlertRepositoryImpl) AcknowledgeAlert(ctx context.Context, id, acknowledgedBy string) error {
	query := `
		UPDATE alerts
		SET status = 'acknowledged', acknowledged_at = $1, acknowledged_by = $2, updated_at = $3
		WHERE id = $4
	`

	tag, err := r.store.Exec(query, time.Now(), acknowledgedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("确认告警失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("告警不存在: %s", id)
	}

	return nil
}

// ResolveAlert 解决告警
func (r *AlertRepositoryImpl) ResolveAlert(ctx context.Context, id, resolvedBy string) error {
	query := `
		UPDATE alerts
		SET status = 'resolved', resolved_at = $1, resolved_by = $2, updated_at = $3
		WHERE id = $4
	`

	tag, err := r.store.Exec(query, time.Now(), resolvedBy, time.Now(), id)
	if err != nil {
		return fmt.Errorf("解决告警失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("告警不存在: %s", id)
	}

	return nil
}

// GetActiveAlerts 获取活动告警
func (r *AlertRepositoryImpl) GetActiveAlerts(ctx context.Context, deviceID string) ([]*alertDomain.Alert, error) {
	filters := interfaces.AlertFilters{
		DeviceID: &deviceID,
		Status:   func() *alertDomain.AlertStatus { s := alertDomain.AlertStatusActive; return &s }(),
		Limit:    100,
	}
	return r.ListAlerts(ctx, filters)
}

// GetAlertStats 获取告警统计
func (r *AlertRepositoryImpl) GetAlertStats(ctx context.Context, filters interfaces.AlertFilters) (*interfaces.AlertStats, error) {
	stats := &interfaces.AlertStats{
		AlertsByLevel:  make(map[string]int64),
		AlertsByStatus: make(map[string]int64),
	}

	// 构建基础查询条件
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if filters.DeviceID != nil {
		whereClause += fmt.Sprintf(" AND device_id = $%d", argIndex)
		args = append(args, *filters.DeviceID)
		argIndex++
	}

	if filters.StartTime != nil {
		whereClause += fmt.Sprintf(" AND first_occurred_at >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		whereClause += fmt.Sprintf(" AND first_occurred_at < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	// 总告警数
	query := fmt.Sprintf("SELECT COUNT(*) FROM alerts %s", whereClause)
	if err := r.store.QueryRow(query, args...).Scan(&stats.TotalAlerts); err != nil {
		return nil, fmt.Errorf("查询总告警数失败: %w", err)
	}

	// 按状态统计
	query = fmt.Sprintf(`
		SELECT status, COUNT(*)
		FROM alerts
		%s
		GROUP BY status
	`, whereClause)
	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询告警状态统计失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		stats.AlertsByStatus[status] = count
		if status == "active" {
			stats.ActiveAlerts = count
		} else if status == "acknowledged" {
			stats.AcknowledgedAlerts = count
		} else if status == "resolved" {
			stats.ResolvedAlerts = count
		}
	}

	// 按级别统计
	query = fmt.Sprintf(`
		SELECT level, COUNT(*)
		FROM alerts
		%s
		GROUP BY level
	`, whereClause)
	rows, err = r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询告警级别统计失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var level string
		var count int64
		if err := rows.Scan(&level, &count); err != nil {
			continue
		}
		stats.AlertsByLevel[level] = count
	}

	return stats, nil
}

// CreateAlertRule 创建告警规则
func (r *AlertRepositoryImpl) CreateAlertRule(ctx context.Context, rule *alertDomain.AlertRule) error {
	conditionJSON, err := json.Marshal(rule.Condition)
	if err != nil {
		return fmt.Errorf("序列化条件失败: %w", err)
	}

	query := `
		INSERT INTO alert_rules (id, name, description, condition, level, enabled, cooldown_seconds, max_executions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.store.Exec(query,
		rule.ID,
		rule.Name,
		rule.Description,
		conditionJSON,
		string(rule.Level),
		rule.Enabled,
		rule.CooldownSeconds,
		rule.MaxExecutions,
		rule.CreatedAt,
		rule.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("创建告警规则失败: %w", err)
	}

	return nil
}

// GetAlertRuleByID 根据ID获取告警规则
func (r *AlertRepositoryImpl) GetAlertRuleByID(ctx context.Context, id string) (*alertDomain.AlertRule, error) {
	query := `
		SELECT id, name, description, condition, level, enabled, cooldown_seconds, max_executions, created_at, updated_at
		FROM alert_rules
		WHERE id = $1
	`

	var rule alertDomain.AlertRule
	var conditionJSON []byte
	var level string

	err := r.store.QueryRow(query, id).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&conditionJSON,
		&level,
		&rule.Enabled,
		&rule.CooldownSeconds,
		&rule.MaxExecutions,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("告警规则不存在: %s", id)
		}
		return nil, fmt.Errorf("查询告警规则失败: %w", err)
	}

	rule.Level = alertDomain.AlertLevel(level)

	if err := json.Unmarshal(conditionJSON, &rule.Condition); err != nil {
		r.store.Log().Warn("反序列化条件失败: %v", err)
		rule.Condition = make(map[string]interface{})
	}

	return &rule, nil
}

// ListAlertRules 列出告警规则
func (r *AlertRepositoryImpl) ListAlertRules(ctx context.Context, enabled *bool) ([]*alertDomain.AlertRule, error) {
	query := `
		SELECT id, name, description, condition, level, enabled, cooldown_seconds, max_executions, created_at, updated_at
		FROM alert_rules
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	if enabled != nil {
		query += " AND enabled = $1"
		args = append(args, *enabled)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.store.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询告警规则列表失败: %w", err)
	}
	defer rows.Close()

	rules := make([]*alertDomain.AlertRule, 0)

	for rows.Next() {
		var rule alertDomain.AlertRule
		var conditionJSON []byte
		var level string

		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&conditionJSON,
			&level,
			&rule.Enabled,
			&rule.CooldownSeconds,
			&rule.MaxExecutions,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			r.store.Log().Warn("扫描告警规则失败: %v", err)
			continue
		}

		rule.Level = alertDomain.AlertLevel(level)

		if err := json.Unmarshal(conditionJSON, &rule.Condition); err != nil {
			r.store.Log().Warn("反序列化条件失败: %v", err)
			rule.Condition = make(map[string]interface{})
		}

		rules = append(rules, &rule)
	}

	return rules, nil
}

// UpdateAlertRule 更新告警规则
func (r *AlertRepositoryImpl) UpdateAlertRule(ctx context.Context, rule *alertDomain.AlertRule) error {
	conditionJSON, err := json.Marshal(rule.Condition)
	if err != nil {
		return fmt.Errorf("序列化条件失败: %w", err)
	}

	query := `
		UPDATE alert_rules
		SET name = $1, description = $2, condition = $3, level = $4, enabled = $5,
		    cooldown_seconds = $6, max_executions = $7, updated_at = $8
		WHERE id = $9
	`

	tag, err := r.store.Exec(query,
		rule.Name,
		rule.Description,
		conditionJSON,
		string(rule.Level),
		rule.Enabled,
		rule.CooldownSeconds,
		rule.MaxExecutions,
		time.Now(),
		rule.ID,
	)

	if err != nil {
		return fmt.Errorf("更新告警规则失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("告警规则不存在: %s", rule.ID)
	}

	return nil
}

// DeleteAlertRule 删除告警规则
func (r *AlertRepositoryImpl) DeleteAlertRule(ctx context.Context, id string) error {
	query := `DELETE FROM alert_rules WHERE id = $1`

	tag, err := r.store.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除告警规则失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("告警规则不存在: %s", id)
	}

	return nil
}
