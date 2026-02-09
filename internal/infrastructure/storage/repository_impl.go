package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"go_ProFiBus/internal/domain/datasample"
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
	sql := "SELECT source_id, time as timestamp, data, quality FROM sensor_readings WHERE 1=1"
	args := make([]interface{}, 0)
	argIdx := 1

	// 应用过滤器
	if sourceID, ok := filters["source_id"].(string); ok {
		sql += fmt.Sprintf(" AND source_id = $%d", argIdx)
		args = append(args, sourceID)
		argIdx++
	}

	if startTime, ok := filters["start_time"].(time.Time); ok {
		sql += fmt.Sprintf(" AND time >= $%d", argIdx)
		args = append(args, startTime)
		argIdx++
	}

	if endTime, ok := filters["end_time"].(time.Time); ok {
		sql += fmt.Sprintf(" AND time <= $%d", argIdx)
		args = append(args, endTime)
		argIdx++
	}

	// 排序（默认按时间排序）
	orderBy := query.GetOrderBy()
	if orderBy == "" {
		orderBy = "time"
	}
	if query.GetOrderDesc() {
		sql += fmt.Sprintf(" ORDER BY %s DESC", orderBy)
	} else {
		sql += fmt.Sprintf(" ORDER BY %s ASC", orderBy)
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
	// 时序数据通常不按 ID 删除，而是按时间范围删除
	return fmt.Errorf("delete by id not supported for time series data, use DeleteOldData instead")
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
		// 序列化数据为 JSON
		dataJSON, err := json.Marshal(sample.GetData())
		if err != nil {
			return fmt.Errorf("序列化数据失败: %w", err)
		}
		
		row := []interface{}{
			sample.GetSourceID(),
			sample.GetTimestamp(),
			dataJSON, // JSONB 字段需要 JSON bytes
			sample.GetQuality(),
		}
		rows = append(rows, row)
	}

	// 使用 CopyFrom 批量写入
	copySource := pgx.CopyFromRows(rows)
	tableName := pgx.Identifier{"sensor_readings"}
	columnNames := []string{"source_id", "time", "data", "quality"}

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
		SELECT source_id, time as timestamp, data, quality
		FROM sensor_readings
		WHERE source_id = $1 AND time >= $2 AND time <= $3
		ORDER BY time ASC
	`

	rows, err := r.store.Query(sql, sourceID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := make([]interfaces.DataSample, 0)
	
	for rows.Next() {
		var sourceID string
		var timestamp time.Time
		var dataJSON []byte
		var quality float64

		if err := rows.Scan(&sourceID, &timestamp, &dataJSON, &quality); err != nil {
			return nil, fmt.Errorf("扫描行失败: %w", err)
		}

		// 解析 JSON 数据
		var data map[string]interface{}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("解析数据 JSON 失败: %w", err)
		}

		// 创建 DataSample 实例
		sample := datasample.NewDataSampleWithTime(sourceID, timestamp, data)
		sample.SetQuality(quality)
		samples = append(samples, sample)
	}

	return samples, rows.Err()
}

// QueryLatest 实现 TimeSeriesRepository 接口
func (r *TimeSeriesRepositoryImpl) QueryLatest(
	ctx context.Context,
	sourceID string,
	limit int,
) ([]interfaces.DataSample, error) {
	sql := `
		SELECT source_id, time as timestamp, data, quality
		FROM sensor_readings
		WHERE source_id = $1
		ORDER BY time DESC
		LIMIT $2
	`

	rows, err := r.store.Query(sql, sourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := make([]interfaces.DataSample, 0)
	
	for rows.Next() {
		var sourceID string
		var timestamp time.Time
		var dataJSON []byte
		var quality float64

		if err := rows.Scan(&sourceID, &timestamp, &dataJSON, &quality); err != nil {
			return nil, fmt.Errorf("扫描行失败: %w", err)
		}

		// 解析 JSON 数据
		var data map[string]interface{}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("解析数据 JSON 失败: %w", err)
		}

		// 创建 DataSample 实例
		sample := datasample.NewDataSampleWithTime(sourceID, timestamp, data)
		sample.SetQuality(quality)
		samples = append(samples, sample)
	}

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
			time_bucket($1, time) AS bucket,
			%s(CAST(data->>'value' AS DOUBLE PRECISION)) AS agg_value,
			COUNT(*) AS count
		FROM sensor_readings
		WHERE source_id = $2 AND time >= $3 AND time <= $4
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
	sql := "DELETE FROM sensor_readings WHERE time < $1"
	_, err := r.store.Exec(sql, before)
	return err
}

// 确保实现了接口
var _ interfaces.TimeSeriesRepository = (*TimeSeriesRepositoryImpl)(nil)
