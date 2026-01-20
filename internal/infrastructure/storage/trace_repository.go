package storage

import (
	"context"
	"fmt"
	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/storage"
	"time"

	"github.com/jackc/pgx/v5"
)

// TraceRepository 追踪数据仓储实现
type TraceRepository struct {
	store *storage.PostgresStore
}

// NewTraceRepository 创建追踪数据仓储
func NewTraceRepository(store *storage.PostgresStore) *TraceRepository {
	return &TraceRepository{
		store: store,
	}
}

// SaveTraceEvent 实现 TraceRepository 接口
func (r *TraceRepository) SaveTraceEvent(ctx context.Context, event *interfaces.TraceEvent) error {
	sql := `
		INSERT INTO trace_events
		(id, pipeline_id, sample_id, component_type, component_id, component_name,
		 action, timestamp, duration_ms, status, error_message, metadata, data_snapshot)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.store.Exec(sql,
		event.ID,
		event.PipelineID,
		event.SampleID,
		event.ComponentType,
		event.ComponentID,
		event.ComponentName,
		event.Action,
		event.Timestamp,
		event.Duration.Milliseconds(),
		event.Status,
		event.Error,
		event.Metadata,
		event.DataSnapshot,
	)

	return err
}

// SaveTraceEventsBatch 实现 TraceRepository 接口 - 批量写入优化
func (r *TraceRepository) SaveTraceEventsBatch(ctx context.Context, events []*interfaces.TraceEvent) error {
	if len(events) == 0 {
		return nil
	}

	// 准备批量插入数据
	rows := make([][]interface{}, 0, len(events))
	for _, event := range events {
		row := []interface{}{
			event.ID,
			event.PipelineID,
			event.SampleID,
			event.ComponentType,
			event.ComponentID,
			event.ComponentName,
			event.Action,
			event.Timestamp,
			event.Duration.Milliseconds(),
			event.Status,
			event.Error,
			event.Metadata,
			event.DataSnapshot,
		}
		rows = append(rows, row)
	}

	// 使用 COPY 批量插入
	copySource := pgx.CopyFromRows(rows)
	tableName := pgx.Identifier{"trace_events"}
	columnNames := []string{
		"id", "pipeline_id", "sample_id", "component_type", "component_id", "component_name",
		"action", "timestamp", "duration_ms", "status", "error_message", "metadata", "data_snapshot",
	}

	count, err := r.store.CopyFrom(tableName, columnNames, copySource)
	if err != nil {
		return fmt.Errorf("batch insert trace events failed: %w", err)
	}

	_ = count // 可以记录日志
	return nil
}

// QueryTraceEvents 实现 TraceRepository 接口
func (r *TraceRepository) QueryTraceEvents(ctx context.Context, filter interfaces.TraceFilter) ([]interfaces.TraceEvent, error) {
	// 构建 SQL 查询
	sql := `
		SELECT id, pipeline_id, sample_id, component_type, component_id, component_name,
		       action, timestamp, duration_ms, status, error_message, metadata, data_snapshot
		FROM trace_events
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argIdx := 1

	// 应用过滤条件
	if filter.PipelineID != nil {
		sql += fmt.Sprintf(" AND pipeline_id = $%d", argIdx)
		args = append(args, *filter.PipelineID)
		argIdx++
	}

	if filter.ComponentType != nil {
		sql += fmt.Sprintf(" AND component_type = $%d", argIdx)
		args = append(args, *filter.ComponentType)
		argIdx++
	}

	if filter.ComponentID != nil {
		sql += fmt.Sprintf(" AND component_id = $%d", argIdx)
		args = append(args, *filter.ComponentID)
		argIdx++
	}

	if filter.SampleID != nil {
		sql += fmt.Sprintf(" AND sample_id = $%d", argIdx)
		args = append(args, *filter.SampleID)
		argIdx++
	}

	if filter.Action != nil {
		sql += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, *filter.Action)
		argIdx++
	}

	if filter.Status != nil {
		sql += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.StartTime != nil {
		sql += fmt.Sprintf(" AND timestamp >= $%d", argIdx)
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil {
		sql += fmt.Sprintf(" AND timestamp <= $%d", argIdx)
		args = append(args, *filter.EndTime)
		argIdx++
	}

	// 排序
	orderBy := filter.OrderBy
	if orderBy == "" {
		orderBy = "timestamp"
	}
	if filter.OrderDesc {
		sql += fmt.Sprintf(" ORDER BY %s DESC", orderBy)
	} else {
		sql += fmt.Sprintf(" ORDER BY %s ASC", orderBy)
	}

	// 限制和偏移
	if filter.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	if filter.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	// 执行查询
	rows, err := r.store.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query trace events failed: %w", err)
	}
	defer rows.Close()

	// 解析结果
	events := make([]interfaces.TraceEvent, 0)
	for rows.Next() {
		var event interfaces.TraceEvent
		var durationMs int64

		err := rows.Scan(
			&event.ID,
			&event.PipelineID,
			&event.SampleID,
			&event.ComponentType,
			&event.ComponentID,
			&event.ComponentName,
			&event.Action,
			&event.Timestamp,
			&durationMs,
			&event.Status,
			&event.Error,
			&event.Metadata,
			&event.DataSnapshot,
		)
		if err != nil {
			return nil, fmt.Errorf("scan trace event failed: %w", err)
		}

		event.Duration = time.Duration(durationMs) * time.Millisecond
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetTraceBySampleID 实现 TraceRepository 接口
func (r *TraceRepository) GetTraceBySampleID(ctx context.Context, sampleID string) ([]interfaces.TraceEvent, error) {
	filter := interfaces.TraceFilter{
		SampleID:  &sampleID,
		OrderBy:   "timestamp",
		OrderDesc: false,
	}

	return r.QueryTraceEvents(ctx, filter)
}

// DeleteOldTraces 实现 TraceRepository 接口
func (r *TraceRepository) DeleteOldTraces(ctx context.Context, before time.Time) error {
	sql := "DELETE FROM trace_events WHERE timestamp < $1"

	tag, err := r.store.Exec(sql, before)
	if err != nil {
		return fmt.Errorf("delete old traces failed: %w", err)
	}

	_ = tag.RowsAffected() // 可以记录日志
	return nil
}

// GetPipelineMetrics 实现 TraceRepository 接口
func (r *TraceRepository) GetPipelineMetrics(ctx context.Context, pipelineID string, start, end time.Time) (*interfaces.PipelineMetrics, error) {
	// 查询总体指标
	sql := `
		SELECT
			COUNT(*) as total_samples,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as success_samples,
			COUNT(CASE WHEN status = 'error' THEN 1 END) as error_samples,
			COALESCE(AVG(duration_ms), 0) as avg_duration_ms,
			COALESCE(MAX(duration_ms), 0) as max_duration_ms,
			COALESCE(MIN(duration_ms), 0) as min_duration_ms
		FROM trace_events
		WHERE pipeline_id = $1
		  AND timestamp >= $2
		  AND timestamp <= $3
		  AND duration_ms IS NOT NULL
	`

	var totalSamples, successSamples, errorSamples int64
	var avgDurationMs, maxDurationMs, minDurationMs float64

	row := r.store.QueryRow(sql, pipelineID, start, end)
	err := row.Scan(&totalSamples, &successSamples, &errorSamples, &avgDurationMs, &maxDurationMs, &minDurationMs)
	if err != nil {
		return nil, fmt.Errorf("query pipeline metrics failed: %w", err)
	}

	metrics := &interfaces.PipelineMetrics{
		PipelineID:     pipelineID,
		StartTime:      start,
		EndTime:        end,
		TotalSamples:   totalSamples,
		SuccessSamples: successSamples,
		ErrorSamples:   errorSamples,
		AvgDuration:    time.Duration(avgDurationMs) * time.Millisecond,
		MaxDuration:    time.Duration(maxDurationMs) * time.Millisecond,
		MinDuration:    time.Duration(minDurationMs) * time.Millisecond,
		ComponentMetrics: make(map[string]*interfaces.ComponentMetrics),
	}

	// 查询各组件指标
	componentSQL := `
		SELECT
			component_id,
			component_type,
			component_name,
			COUNT(*) as event_count,
			COALESCE(AVG(duration_ms), 0) as avg_duration_ms,
			COALESCE(MAX(duration_ms), 0) as max_duration_ms,
			COALESCE(MIN(duration_ms), 0) as min_duration_ms,
			COUNT(CASE WHEN status = 'error' THEN 1 END) as error_count
		FROM trace_events
		WHERE pipeline_id = $1
		  AND timestamp >= $2
		  AND timestamp <= $3
		  AND duration_ms IS NOT NULL
		GROUP BY component_id, component_type, component_name
	`

	rows, err := r.store.Query(componentSQL, pipelineID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query component metrics failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var componentID, componentType, componentName string
		var eventCount, errorCount int64
		var avgMs, maxMs, minMs float64

		err := rows.Scan(&componentID, &componentType, &componentName, &eventCount, &avgMs, &maxMs, &minMs, &errorCount)
		if err != nil {
			return nil, fmt.Errorf("scan component metrics failed: %w", err)
		}

		errorRate := 0.0
		if eventCount > 0 {
			errorRate = float64(errorCount) / float64(eventCount) * 100
		}

		metrics.ComponentMetrics[componentID] = &interfaces.ComponentMetrics{
			ComponentID:   componentID,
			ComponentType: componentType,
			ComponentName: componentName,
			EventCount:    eventCount,
			AvgDuration:   time.Duration(avgMs) * time.Millisecond,
			MaxDuration:   time.Duration(maxMs) * time.Millisecond,
			MinDuration:   time.Duration(minMs) * time.Millisecond,
			ErrorCount:    errorCount,
			ErrorRate:     errorRate,
		}
	}

	return metrics, rows.Err()
}

// 确保实现了接口
var _ interfaces.TraceRepository = (*TraceRepository)(nil)
