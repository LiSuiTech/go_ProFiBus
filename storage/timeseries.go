package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// SensorReading 传感器单条读数（用于 WriteSensorReading/QuerySensorReadings，替代已移除的 collector 包）
type SensorReading struct {
	Timestamp  time.Time
	SourceID   string
	Protocol   string
	ParsedData map[string]interface{}
	Quality    float64
}

// WriteSensorReadings 批量写入传感器数据
func (ps *PostgresStore) WriteSensorReadings(readings []map[string]interface{}) error {
	if len(readings) == 0 {
		return nil
	}

	// 使用COPY FROM进行批量插入（性能最优）
	rows := make([][]interface{}, len(readings))
	for i, reading := range readings {
		// 从 reading map 中提取字段
		timestamp, _ := reading["timestamp"].(time.Time)
		sensorID, _ := reading["sensor_id"].(string)
		protocol, _ := reading["protocol"].(string)
		if protocol == "" {
			protocol = "unknown"
		}
		data, _ := reading["data"].(map[string]interface{})
		if data == nil {
			data = reading // 如果没有单独的 data 字段，使用整个 reading
		}
		quality, _ := reading["quality"].(float64)
		if quality == 0 {
			quality = 1.0 // 默认质量
		}

		dataJSON, err := json.Marshal(data)
		if err != nil {
			ps.log.Warn("序列化数据失败: %v", err)
			dataJSON = []byte("{}")
		}

		rows[i] = []interface{}{
			timestamp,
			sensorID,
			protocol,
			dataJSON,
			quality,
		}
	}

	copySource := pgx.CopyFromRows(rows)

	count, err := ps.CopyFrom(
		pgx.Identifier{"sensor_readings"},
		[]string{"time", "sensor_id", "protocol", "data", "quality"},
		copySource,
	)

	if err != nil {
		return fmt.Errorf("批量写入传感器数据失败: %w", err)
	}

	ps.log.Debug("批量写入传感器数据成功: %d 条", count)
	return nil
}

// WriteSensorReading 写入单个传感器数据
func (ps *PostgresStore) WriteSensorReading(sample *SensorReading) error {
	dataJSON, err := json.Marshal(sample.ParsedData)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	query := `
		INSERT INTO sensor_readings (time, sensor_id, protocol, data, quality, source_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = ps.Exec(query,
		sample.Timestamp,
		sample.SourceID,
		sample.Protocol,
		dataJSON,
		sample.Quality,
		sample.SourceID,
	)

	if err != nil {
		return fmt.Errorf("写入传感器数据失败: %w", err)
	}

	return nil
}

// QuerySensorReadings 查询传感器数据。limit <= 0 表示不限制条数。
func (ps *PostgresStore) QuerySensorReadings(
	sensorID string,
	start, end time.Time,
	limit int,
) ([]*SensorReading, error) {
	query := `
		SELECT time, sensor_id, protocol, data, quality
		FROM sensor_readings
		WHERE sensor_id = $1 AND time >= $2 AND time < $3
		ORDER BY time ASC
	`
	args := []interface{}{sensorID, start, end}
	if limit > 0 {
		query += " LIMIT $4"
		args = append(args, limit)
	}

	rows, err := ps.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := make([]*SensorReading, 0)

	for rows.Next() {
		var sample SensorReading
		var dataJSON []byte

		err := rows.Scan(
			&sample.Timestamp,
			&sample.SourceID,
			&sample.Protocol,
			&dataJSON,
			&sample.Quality,
		)
		if err != nil {
			ps.log.Warn("扫描行失败: %v", err)
			continue
		}

		// 反序列化数据
		if err := json.Unmarshal(dataJSON, &sample.ParsedData); err != nil {
			ps.log.Warn("反序列化数据失败: %v", err)
			sample.ParsedData = make(map[string]interface{})
		}

		samples = append(samples, &sample)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果失败: %w", err)
	}

	return samples, nil
}

// QuerySensorReadingsLatest 查询最新的N条传感器数据
func (ps *PostgresStore) QuerySensorReadingsLatest(
	sensorID string,
	limit int,
) ([]*SensorReading, error) {
	query := `
		SELECT time, sensor_id, protocol, data, quality
		FROM sensor_readings
		WHERE sensor_id = $1
		ORDER BY time DESC
		LIMIT $2
	`

	rows, err := ps.Query(query, sensorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := make([]*SensorReading, 0, limit)

	for rows.Next() {
		var sample SensorReading
		var dataJSON []byte

		err := rows.Scan(
			&sample.Timestamp,
			&sample.SourceID,
			&sample.Protocol,
			&dataJSON,
			&sample.Quality,
		)
		if err != nil {
			ps.log.Warn("扫描行失败: %v", err)
			continue
		}

		if err := json.Unmarshal(dataJSON, &sample.ParsedData); err != nil {
			ps.log.Warn("反序列化数据失败: %v", err)
			sample.ParsedData = make(map[string]interface{})
		}

		samples = append(samples, &sample)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果失败: %w", err)
	}

	// 反转切片（从最旧到最新）
	for i, j := 0, len(samples)-1; i < j; i, j = i+1, j-1 {
		samples[i], samples[j] = samples[j], samples[i]
	}

	return samples, nil
}

// AggregateData 聚合查询传感器数据
type AggregationResult struct {
	Bucket    time.Time
	SensorID  string
	FieldName string
	Avg       *float64
	Min       *float64
	Max       *float64
	Count     int64
}

// AggregateSensorData 聚合传感器数据
func (ps *PostgresStore) AggregateSensorData(
	sensorID string,
	fieldName string,
	start, end time.Time,
	intervalMinutes int,
) ([]*AggregationResult, error) {
	query := fmt.Sprintf(`
		SELECT
			time_bucket('%d minutes', time) AS bucket,
			sensor_id,
			AVG((data->>'%s')::numeric) AS avg_value,
			MIN((data->>'%s')::numeric) AS min_value,
			MAX((data->>'%s')::numeric) AS max_value,
			COUNT(*) AS count
		FROM sensor_readings
		WHERE sensor_id = $1 AND time >= $2 AND time < $3
		AND data->>'%s' IS NOT NULL
		GROUP BY bucket, sensor_id
		ORDER BY bucket ASC
	`, intervalMinutes, fieldName, fieldName, fieldName, fieldName)

	rows, err := ps.Query(query, sensorID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*AggregationResult, 0)

	for rows.Next() {
		var result AggregationResult
		result.FieldName = fieldName

		err := rows.Scan(
			&result.Bucket,
			&result.SensorID,
			&result.Avg,
			&result.Min,
			&result.Max,
			&result.Count,
		)
		if err != nil {
			ps.log.Warn("扫描聚合结果失败: %v", err)
			continue
		}

		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历聚合结果失败: %w", err)
	}

	return results, nil
}

// DeleteOldData 删除旧数据（数据保留策略）
func (ps *PostgresStore) DeleteOldData(olderThan time.Time) (int64, error) {
	query := `
		DELETE FROM sensor_readings
		WHERE time < $1
	`

	tag, err := ps.Exec(query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("删除旧数据失败: %w", err)
	}

	rowsAffected := tag.RowsAffected()
	ps.log.Info("删除旧数据: %d 行 (早于 %v)", rowsAffected, olderThan)

	return rowsAffected, nil
}

// GetSensorList 获取传感器列表
func (ps *PostgresStore) GetSensorList() ([]string, error) {
	query := `
		SELECT DISTINCT sensor_id
		FROM sensor_readings
		ORDER BY sensor_id
	`

	rows, err := ps.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sensors := make([]string, 0)

	for rows.Next() {
		var sensorID string
		if err := rows.Scan(&sensorID); err != nil {
			ps.log.Warn("扫描传感器ID失败: %v", err)
			continue
		}
		sensors = append(sensors, sensorID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历传感器列表失败: %w", err)
	}

	return sensors, nil
}

// GetDataCount 获取数据点数量
func (ps *PostgresStore) GetDataCount(sensorID string, start, end time.Time) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM sensor_readings
		WHERE sensor_id = $1 AND time >= $2 AND time < $3
	`

	var count int64
	err := ps.QueryRow(query, sensorID, start, end).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("查询数据点数量失败: %w", err)
	}

	return count, nil
}

// GetDataRange 获取数据时间范围
func (ps *PostgresStore) GetDataRange(sensorID string) (start, end *time.Time, err error) {
	query := `
		SELECT MIN(time), MAX(time)
		FROM sensor_readings
		WHERE sensor_id = $1
	`

	err = ps.QueryRow(query, sensorID).Scan(&start, &end)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("查询数据时间范围失败: %w", err)
	}

	return start, end, nil
}

// CompressChunks 压缩时序数据块（TimescaleDB特有）
func (ps *PostgresStore) CompressChunks(olderThan time.Time) error {
	// 检查是否启用了压缩
	query := `
		SELECT compress_chunk(i)
		FROM show_chunks('sensor_readings', older_than => $1) i
	`

	ctx, cancel := context.WithTimeout(ps.ctx, 5*time.Minute)
	defer cancel()

	rows, err := ps.pool.Query(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("压缩数据块失败: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}

	ps.log.Info("压缩了 %d 个数据块 (早于 %v)", count, olderThan)
	return nil
}
