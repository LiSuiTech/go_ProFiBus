package storage

import (
	"encoding/json"
	"fmt"
	"go_ProFiBus/event"
	"time"

	"github.com/jackc/pgx/v5"
)

// SaveEvent 保存事件
func (ps *PostgresStore) SaveEvent(evt *event.Event) error {
	metadataJSON, err := json.Marshal(evt.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	query := `
		INSERT INTO events (id, type, status, timestamp, severity, score, description, metadata, source_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			description = EXCLUDED.description,
			metadata = EXCLUDED.metadata
	`

	_, err = ps.Exec(query,
		evt.ID,
		evt.Type,
		string(evt.Status),
		evt.Timestamp,
		evt.Severity,
		evt.Score,
		evt.Description,
		metadataJSON,
		evt.SourceID,
	)

	if err != nil {
		return fmt.Errorf("保存事件失败: %w", err)
	}

	ps.log.Debug("保存事件成功: %s", evt.ID)
	return nil
}

// SaveEvents 批量保存事件
func (ps *PostgresStore) SaveEvents(events []*event.Event) error {
	if len(events) == 0 {
		return nil
	}

	rows := make([][]interface{}, len(events))
	for i, evt := range events {
		metadataJSON, err := json.Marshal(evt.Metadata)
		if err != nil {
			ps.log.Warn("序列化元数据失败: %v", err)
			metadataJSON = []byte("{}")
		}

		rows[i] = []interface{}{
			evt.ID,
			evt.Type,
			string(evt.Status),
			evt.Timestamp,
			evt.Severity,
			evt.Score,
			evt.Description,
			metadataJSON,
			evt.SourceID,
		}
	}

	copySource := pgx.CopyFromRows(rows)

	count, err := ps.CopyFrom(
		pgx.Identifier{"events"},
		[]string{"id", "type", "status", "timestamp", "severity", "score", "description", "metadata", "source_id"},
		copySource,
	)

	if err != nil {
		return fmt.Errorf("批量保存事件失败: %w", err)
	}

	ps.log.Debug("批量保存事件成功: %d 条", count)
	return nil
}

// GetEvent 获取事件
func (ps *PostgresStore) GetEvent(eventID string) (*event.Event, error) {
	query := `
		SELECT id, type, status, timestamp, severity, score, description, metadata, source_id
		FROM events
		WHERE id = $1
	`

	var evt event.Event
	var metadataJSON []byte
	var status string

	err := ps.QueryRow(query, eventID).Scan(
		&evt.ID,
		&evt.Type,
		&status,
		&evt.Timestamp,
		&evt.Severity,
		&evt.Score,
		&evt.Description,
		&metadataJSON,
		&evt.SourceID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("事件不存在: %s", eventID)
		}
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}

	evt.Status = event.EventStatus(status)

	if err := json.Unmarshal(metadataJSON, &evt.Metadata); err != nil {
		ps.log.Warn("反序列化元数据失败: %v", err)
		evt.Metadata = make(map[string]interface{})
	}

	return &evt, nil
}

// EventFilters 事件过滤器
type EventFilters struct {
	Status    *event.EventStatus
	Type      *event.EventType
	StartTime *time.Time
	EndTime   *time.Time
	SourceID  *string
	Limit     int
	Offset    int
}

// QueryEvents 查询事件
func (ps *PostgresStore) QueryEvents(filters EventFilters) ([]*event.Event, error) {
	query := `
		SELECT id, type, status, timestamp, severity, score, description, metadata, source_id
		FROM events
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
		argIndex++
	}

	if filters.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, *filters.Type)
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	if filters.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, *filters.SourceID)
		argIndex++
	}

	query += " ORDER BY timestamp DESC"

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

	rows, err := ps.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*event.Event, 0)

	for rows.Next() {
		var evt event.Event
		var metadataJSON []byte
		var status string

		err := rows.Scan(
			&evt.ID,
			&evt.Type,
			&status,
			&evt.Timestamp,
			&evt.Severity,
			&evt.Score,
			&evt.Description,
			&metadataJSON,
			&evt.SourceID,
		)
		if err != nil {
			ps.log.Warn("扫描事件失败: %v", err)
			continue
		}

		evt.Status = event.EventStatus(status)

		if err := json.Unmarshal(metadataJSON, &evt.Metadata); err != nil {
			ps.log.Warn("反序列化元数据失败: %v", err)
			evt.Metadata = make(map[string]interface{})
		}

		events = append(events, &evt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历事件失败: %w", err)
	}

	return events, nil
}

// UpdateEventStatus 更新事件状态
func (ps *PostgresStore) UpdateEventStatus(
	eventID string,
	status event.EventStatus,
	reviewer string,
	notes string,
) error {
	query := `
		UPDATE events
		SET status = $1,
		    metadata = jsonb_set(
		        jsonb_set(metadata, '{reviewer}', to_jsonb($2::text)),
		        '{review_notes}', to_jsonb($3::text)
		    )
		WHERE id = $4
	`

	tag, err := ps.Exec(query, string(status), reviewer, notes, eventID)
	if err != nil {
		return fmt.Errorf("更新事件状态失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("事件不存在: %s", eventID)
	}

	ps.log.Debug("更新事件状态成功: %s -> %s", eventID, status)
	return nil
}

// DeleteEvent 删除事件
func (ps *PostgresStore) DeleteEvent(eventID string) error {
	query := `DELETE FROM events WHERE id = $1`

	tag, err := ps.Exec(query, eventID)
	if err != nil {
		return fmt.Errorf("删除事件失败: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("事件不存在: %s", eventID)
	}

	ps.log.Debug("删除事件成功: %s", eventID)
	return nil
}

// GetEventCount 获取事件数量
func (ps *PostgresStore) GetEventCount(filters EventFilters) (int64, error) {
	query := `SELECT COUNT(*) FROM events WHERE 1=1`

	args := make([]interface{}, 0)
	argIndex := 1

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*filters.Status))
		argIndex++
	}

	if filters.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, *filters.Type)
		argIndex++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp < $%d", argIndex)
		args = append(args, *filters.EndTime)
		argIndex++
	}

	if filters.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argIndex)
		args = append(args, *filters.SourceID)
		argIndex++
	}

	var count int64
	err := ps.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("查询事件数量失败: %w", err)
	}

	return count, nil
}

// GetEventStats 获取事件统计
type EventStats struct {
	TotalEvents      int64
	PendingEvents    int64
	ConfirmedEvents  int64
	RejectedEvents   int64
	EventsByType     map[string]int64
	EventsBySeverity map[int]int64
}

func (ps *PostgresStore) GetEventStats(start, end time.Time) (*EventStats, error) {
	stats := &EventStats{
		EventsByType:     make(map[string]int64),
		EventsBySeverity: make(map[int]int64),
	}

	// 总事件数
	query := `SELECT COUNT(*) FROM events WHERE timestamp >= $1 AND timestamp < $2`
	if err := ps.QueryRow(query, start, end).Scan(&stats.TotalEvents); err != nil {
		return nil, fmt.Errorf("查询总事件数失败: %w", err)
	}

	// 按状态统计
	query = `
		SELECT status, COUNT(*)
		FROM events
		WHERE timestamp >= $1 AND timestamp < $2
		GROUP BY status
	`

	rows, err := ps.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}

		switch event.EventStatus(status) {
		case event.EventStatusPending:
			stats.PendingEvents = count
		case event.EventStatusConfirmed:
			stats.ConfirmedEvents = count
		case event.EventStatusRejected:
			stats.RejectedEvents = count
		}
	}

	// 按类型统计
	query = `
		SELECT type, COUNT(*)
		FROM events
		WHERE timestamp >= $1 AND timestamp < $2
		GROUP BY type
	`

	rows, err = ps.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			continue
		}
		stats.EventsByType[eventType] = count
	}

	// 按严重程度统计
	query = `
		SELECT severity, COUNT(*)
		FROM events
		WHERE timestamp >= $1 AND timestamp < $2
		GROUP BY severity
	`

	rows, err = ps.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var severity int
		var count int64
		if err := rows.Scan(&severity, &count); err != nil {
			continue
		}
		stats.EventsBySeverity[severity] = count
	}

	return stats, nil
}
