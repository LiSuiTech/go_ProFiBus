package storage

import (
	"context"
	"fmt"
	"go_ProFiBus/pkg/interfaces"
	"go_ProFiBus/storage"
	"time"

	"github.com/jackc/pgx/v5"
)

// TimeSeriesRepositoryImpl 时序数据仓储实现
type TimeSeriesRepositoryImpl struct {
	store *storage.PostgresStore
}

// NewTimeSeriesRepository 创建时序数据仓储
func NewTimeSeriesRepository(store *storage.PostgresStore) *TimeSeriesRepositoryImpl {
	return &TimeSeriesRepositoryImpl{
		store: store,
	}
}

// Store 实现 Repository 接口
func (r *TimeSeriesRepositoryImpl) Store(ctx context.Context, data interface{}) error {
	sample, ok := data.(interfaces.DataSample)
	if !ok {
		return fmt.Errorf("data must be a DataSample")
	}

	return r.WriteSamples(ctx, []interfaces.DataSample{sample})
}

// Query 实现 Repository 接口
func (r *TimeSeriesRepositoryImpl) Query(ctx context.Context, query interfaces.Query) ([]interface{}, error) {
	// 基础查询实现
	filters := query.GetFilters()
	limit := query.GetLimit()
	offset := query.GetOffset()

	// 构建SQL查询
	sql := "SELECT source_id, timestamp, data, quality FROM sensor_data WHERE 1=1"
	args := make([]interface{}, 0)
	argIdx := 1

	// 应用过滤器
	if sourceID, ok := filters["source_id"].(string); ok {
		sql += fmt.Sprintf(" AND source_id = $%d", argIdx)
		args = append(args, sourceID)
		argIdx++
	}

	if startTime, ok := filters["start_time"].(time.Time); ok {
		sql += fmt.Sprintf(" AND timestamp >= $%d", argIdx)
		args = append(args, startTime)
		argIdx++
	}

	if endTime, ok := filters["end_time"].(time.Time); ok {
		sql += fmt.Sprintf(" AND timestamp <= $%d", argIdx)
		args = append(args, endTime)
		argIdx++
	}

	// 排序
	if query.GetOrderDesc() {
		sql += fmt.Sprintf(" ORDER BY %s DESC", query.GetOrderBy())
	} else {
		sql += fmt.Sprintf(" ORDER BY %s ASC", query.GetOrderBy())
	}

	// 限制和偏移
	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		sql += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
	}

	rows, err := r.store.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]interface{}, 0)
	for rows.Next() {
		var sourceID string
		var timestamp time.Time
		var data map[string]interface{}
		var quality float64

		if err := rows.Scan(&sourceID, &timestamp, &data, &quality); err != nil {
			return nil, err
		}

		// 这里需要实际的 DataSample 实现
		// results = append(results, ...)
	}

	return results, rows.Err()
}

// Update 实现 Repository 接口
func (r *TimeSeriesRepositoryImpl) Update(ctx context.Context, id string, data interface{}) error {
	return fmt.Errorf("update not supported for time series data")
}

// Delete 实现 Repository 接口
func (r *TimeSeriesRepositoryImpl) Delete(ctx context.Context, id string) error {
	sql := "DELETE FROM sensor_data WHERE id = $1"
	_, err := r.store.Exec(sql, id)
	return err
}

// Health 实现 Repository 接口
func (r *TimeSeriesRepositoryImpl) Health() error {
	return r.store.Ping()
}

// Close 实现 Repository 接口
func (r *TimeSeriesRepositoryImpl) Close() error {
	return r.store.Close()
}

// WriteSamples 实现 TimeSeriesRepository 接口
func (r *TimeSeriesRepositoryImpl) WriteSamples(ctx context.Context, samples []interfaces.DataSample) error {
	if len(samples) == 0 {
		return nil
	}

	// 使用 COPY 批量插入
	rows := make([][]interface{}, 0, len(samples))
	for _, sample := range samples {
		row := []interface{}{
			sample.GetSourceID(),
			sample.GetTimestamp(),
			sample.GetData(),
			sample.GetQuality(),
		}
		rows = append(rows, row)
	}

	// 使用 CopyFrom 批量写入
	copySource := pgx.CopyFromRows(rows)
	tableName := pgx.Identifier{"sensor_data"}
	columnNames := []string{"source_id", "timestamp", "data", "quality"}

	_, err := r.store.CopyFrom(tableName, columnNames, copySource)
	return err
}

// QueryByTimeRange 实现 TimeSeriesRepository 接口
func (r *TimeSeriesRepositoryImpl) QueryByTimeRange(
	ctx context.Context,
	sourceID string,
	start, end time.Time,
) ([]interfaces.DataSample, error) {
	sql := `
		SELECT source_id, timestamp, data, quality
		FROM sensor_data
		WHERE source_id = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp ASC
	`

	rows, err := r.store.Query(sql, sourceID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := make([]interfaces.DataSample, 0)
	// 这里需要扫描并创建 DataSample 实例
	// TODO: 实现行扫描逻辑

	return samples, rows.Err()
}

// QueryLatest 实现 TimeSeriesRepository 接口
func (r *TimeSeriesRepositoryImpl) QueryLatest(
	ctx context.Context,
	sourceID string,
	limit int,
) ([]interfaces.DataSample, error) {
	sql := `
		SELECT source_id, timestamp, data, quality
		FROM sensor_data
		WHERE source_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := r.store.Query(sql, sourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := make([]interfaces.DataSample, 0)
	// TODO: 实现行扫描逻辑

	return samples, rows.Err()
}

// Aggregate 实现 TimeSeriesRepository 接口
func (r *TimeSeriesRepositoryImpl) Aggregate(
	ctx context.Context,
	query interfaces.AggregateQuery,
) ([]interfaces.AggregateResult, error) {
	// 构建聚合查询
	aggFunc := query.AggFunc
	if aggFunc == "" {
		aggFunc = "avg"
	}

	// 使用 time_bucket 进行时间分组（TimescaleDB功能）
	sql := fmt.Sprintf(`
		SELECT
			time_bucket($1, timestamp) AS bucket,
			%s(CAST(data->>'value' AS DOUBLE PRECISION)) AS agg_value,
			COUNT(*) AS count
		FROM sensor_data
		WHERE source_id = $2 AND timestamp >= $3 AND timestamp <= $4
		GROUP BY bucket
		ORDER BY bucket ASC
	`, aggFunc)

	rows, err := r.store.Query(sql, query.Interval, query.SourceID, query.StartTime, query.EndTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]interfaces.AggregateResult, 0)
	for rows.Next() {
		var bucket time.Time
		var value float64
		var count int

		if err := rows.Scan(&bucket, &value, &count); err != nil {
			return nil, err
		}

		results = append(results, interfaces.AggregateResult{
			Timestamp: bucket,
			Values:    map[string]float64{"value": value},
			Count:     count,
		})
	}

	return results, rows.Err()
}

// DeleteOldData 实现 TimeSeriesRepository 接口
func (r *TimeSeriesRepositoryImpl) DeleteOldData(ctx context.Context, before time.Time) error {
	sql := "DELETE FROM sensor_data WHERE timestamp < $1"
	_, err := r.store.Exec(sql, before)
	return err
}

// 确保实现了接口
var _ interfaces.TimeSeriesRepository = (*TimeSeriesRepositoryImpl)(nil)
