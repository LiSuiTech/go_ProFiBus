# 数据库集成文档

## 概述

go_ProFiBus 使用 **TimescaleDB** (PostgreSQL扩展) 作为统一的数据库解决方案，支持：
- **时序数据存储**：传感器采样数据（高性能）
- **关系数据存储**：事件、规则、标注等业务数据

## 为什么选择TimescaleDB？

1. **统一管理**：单一数据库，SQL统一
2. **自动分区**：按时间自动分区（Hypertable）
3. **高性能压缩**：7天前数据自动压缩，节省90%+存储
4. **PostgreSQL生态**：完全兼容PostgreSQL，工具丰富
5. **易于运维**：备份、恢复、监控都很成熟

## 数据库架构

### 表结构

#### 1. sensor_readings (时序数据表)

```sql
CREATE TABLE sensor_readings (
    time        TIMESTAMPTZ NOT NULL,  -- 采样时间
    sensor_id   TEXT NOT NULL,          -- 传感器ID
    protocol    TEXT,                   -- 通信协议
    data        JSONB,                  -- 采样数据（JSON格式）
    quality     FLOAT,                  -- 数据质量
    source_id   TEXT                    -- 数据源ID
);
```

**特性**：
- TimescaleDB超表（Hypertable）
- 7天自动分区（chunk）
- 7天后数据自动压缩
- 支持90天数据保留策略（可配置）

**索引**：
- `(sensor_id, time DESC)` - 传感器时间序列查询
- `(source_id)` - 按数据源查询
- `GIN (data)` - JSON数据查询

#### 2. events (事件表)

```sql
CREATE TABLE events (
    id          TEXT PRIMARY KEY,
    type        TEXT NOT NULL,
    status      TEXT NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    severity    INTEGER,
    score       FLOAT,
    description TEXT,
    metadata    JSONB DEFAULT '{}',
    source_id   TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
```

**状态类型**：
- `pending` - 待标注
- `confirmed` - 已确认
- `rejected` - 已拒绝

**索引**：
- `(timestamp DESC)` - 时间查询
- `(status, timestamp DESC)` - 按状态和时间查询
- `(type)`, `(severity)`, `(source_id)` - 其他字段查询
- `GIN (metadata)` - JSON元数据查询

#### 3. rules (规则表)

```sql
CREATE TABLE rules (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    config      JSONB NOT NULL,
    enabled     BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
```

#### 4. annotations (标注表)

```sql
CREATE TABLE annotations (
    id           TEXT PRIMARY KEY,
    event_id     TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    annotator_id TEXT NOT NULL,
    confirmed    BOOLEAN,
    annotation   TEXT,
    labels       JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
```

#### 5. annotators (标注员表)

```sql
CREATE TABLE annotators (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    role        TEXT,
    permissions TEXT[],
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

## 安装和配置

### 1. 安装PostgreSQL和TimescaleDB

#### Docker方式（推荐）

```bash
docker run -d \
  --name timescaledb \
  -p 5432:5432 \
  -e POSTGRES_PASSWORD=yourpassword \
  -e POSTGRES_DB=profibus \
  timescale/timescaledb:latest-pg16
```

#### 手动安装（Ubuntu/Debian）

```bash
# 添加TimescaleDB仓库
sudo sh -c "echo 'deb https://packagecloud.io/timescale/timescaledb/ubuntu/ $(lsb_release -c -s) main' > /etc/apt/sources.list.d/timescaledb.list"
wget --quiet -O - https://packagecloud.io/timescale/timescaledb/gpgkey | sudo apt-key add -

# 安装
sudo apt update
sudo apt install timescaledb-2-postgresql-16

# 配置PostgreSQL
sudo timescaledb-tune

# 重启PostgreSQL
sudo systemctl restart postgresql
```

### 2. 创建数据库

```bash
# 连接到PostgreSQL
psql -U postgres

# 创建数据库
CREATE DATABASE profibus;

# 连接到数据库
\c profibus

# 启用TimescaleDB扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

### 3. 运行迁移脚本

```bash
cd migrations
psql -U postgres -d profibus -f 001_initial_schema.sql
```

### 4. 验证安装

```sql
-- 查看TimescaleDB版本
SELECT extversion FROM pg_extension WHERE extname = 'timescaledb';

-- 查看超表
SELECT * FROM timescaledb_information.hypertables;

-- 查看所有表
\dt
```

## Go代码集成

### 1. 基本连接

```go
package main

import (
    "go_ProFiBus/storage"
    "log"
)

func main() {
    // 创建数据库连接
    config := &storage.PostgresConfig{
        Host:            "localhost",
        Port:            5432,
        Database:        "profibus",
        User:            "postgres",
        Password:        "yourpassword",
        MaxConnections:  10,
        MinConnections:  2,
    }

    store, err := storage.NewPostgresStore(config)
    if err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    defer store.Close()

    // 测试连接
    if err := store.Ping(); err != nil {
        log.Fatalf("数据库连接不可用: %v", err)
    }

    log.Println("数据库连接成功！")
}
```

### 2. 写入时序数据

```go
import (
    "go_ProFiBus/collector"
    "go_ProFiBus/storage"
    "time"
)

// 写入单个样本
sample := &collector.DataSample{
    Timestamp: time.Now(),
    SourceID:  "sensor_001",
    Protocol:  "RS485",
    ParsedData: map[string]interface{}{
        "temperature": 25.5,
        "pressure":    101.3,
    },
    Quality: 0.95,
}

err := store.WriteSensorReading(sample)

// 批量写入（推荐）
samples := []*collector.DataSample{sample1, sample2, sample3}
err := store.WriteSensorReadings(samples)
```

### 3. 查询时序数据

```go
// 按时间范围查询
start := time.Now().Add(-24 * time.Hour)
end := time.Now()

samples, err := store.QuerySensorReadings("sensor_001", start, end)

// 查询最新N条
latest, err := store.QuerySensorReadingsLatest("sensor_001", 100)

// 聚合查询（每5分钟的平均值）
results, err := store.AggregateSensorData(
    "sensor_001",
    "temperature",
    start,
    end,
    5, // 5分钟间隔
)
```

### 4. 事件操作

```go
import "go_ProFiBus/event"

// 保存事件
evt := &event.Event{
    ID:          "evt_001",
    Type:        event.EventTypeAnomaly,
    Status:      event.EventStatusPending,
    Timestamp:   time.Now(),
    Severity:    2,
    Score:       0.85,
    Description: "温度异常",
    SourceID:    "sensor_001",
    Metadata:    map[string]interface{}{
        "temperature": 35.0,
    },
}

err := store.SaveEvent(evt)

// 查询事件
status := event.EventStatusPending
filters := storage.EventFilters{
    Status: &status,
    Limit:  100,
}

events, err := store.QueryEvents(filters)

// 更新事件状态
err := store.UpdateEventStatus(
    "evt_001",
    event.EventStatusConfirmed,
    "annotator_001",
    "确认异常",
)
```

### 5. 使用事务

```go
err := store.WithTx(func(tx pgx.Tx) error {
    // 在事务中执行多个操作
    _, err := tx.Exec(ctx, "INSERT INTO events ...")
    if err != nil {
        return err // 自动回滚
    }

    _, err = tx.Exec(ctx, "UPDATE rules ...")
    if err != nil {
        return err // 自动回滚
    }

    return nil // 自动提交
})
```

### 6. 集成批量写入器

```go
import "go_ProFiBus/concurrent"

// 创建批量写入器
writer := concurrent.NewBatchWriter(concurrent.BatchWriterConfig{
    Name:          "SensorData",
    BatchSize:     100,
    FlushInterval: 5 * time.Second,
    WorkerCount:   2,
    WriteFn: func(items []interface{}) error {
        // 转换为DataSample切片
        samples := make([]*collector.DataSample, len(items))
        for i, item := range items {
            samples[i] = item.(*collector.DataSample)
        }

        // 批量写入数据库
        return store.WriteSensorReadings(samples)
    },
})

writer.Start()
defer writer.Stop()

// 写入数据
writer.Write(sample)
```

## 性能优化

### 1. 批量插入

使用`COPY FROM`而不是逐条`INSERT`：

```go
// ❌ 慢 - 逐条插入
for _, sample := range samples {
    store.WriteSensorReading(sample)
}

// ✅ 快 - 批量插入（10-100倍性能提升）
store.WriteSensorReadings(samples)
```

### 2. 索引优化

```sql
-- 为常用查询创建复合索引
CREATE INDEX idx_sensor_time ON sensor_readings (sensor_id, time DESC);

-- 为JSONB字段创建表达式索引
CREATE INDEX idx_temp ON sensor_readings ((data->>'temperature'));
```

### 3. 数据压缩

```sql
-- 手动压缩7天前的数据
SELECT compress_chunk(i)
FROM show_chunks('sensor_readings', older_than => INTERVAL '7 days') i;

-- 查看压缩率
SELECT
    pg_size_pretty(before_compression_total_bytes) as before,
    pg_size_pretty(after_compression_total_bytes) as after,
    100 - (after_compression_total_bytes * 100 / before_compression_total_bytes) as compression_ratio
FROM timescaledb_information.compression_settings
WHERE hypertable_name = 'sensor_readings';
```

### 4. 连接池配置

```go
config := &storage.PostgresConfig{
    MaxConnections:  20,              // 最大连接数
    MinConnections:  5,               // 最小连接数
    MaxConnLifetime: 1 * time.Hour,   // 连接最大生命周期
    MaxConnIdleTime: 30 * time.Minute, // 连接最大空闲时间
    HealthCheckPeriod: 1 * time.Minute, // 健康检查间隔
}
```

### 5. 分区策略

```sql
-- 调整chunk间隔（默认7天）
SELECT set_chunk_time_interval('sensor_readings', INTERVAL '1 day');

-- 对于高频数据，可以使用更小的间隔
SELECT set_chunk_time_interval('sensor_readings', INTERVAL '6 hours');
```

## 数据保留策略

### 自动删除旧数据

```sql
-- 保留90天数据
SELECT add_retention_policy('sensor_readings', INTERVAL '90 days');

-- 取消保留策略
SELECT remove_retention_policy('sensor_readings');
```

### 手动清理

```go
// 删除30天前的数据
olderThan := time.Now().Add(-30 * 24 * time.Hour)
count, err := store.DeleteOldData(olderThan)
log.Printf("删除了 %d 条旧数据", count)
```

## 监控和维护

### 1. 监控查询

```sql
-- 查看表大小
SELECT
    hypertable_name,
    pg_size_pretty(hypertable_size(format('%I.%I', hypertable_schema, hypertable_name))) as size
FROM timescaledb_information.hypertables;

-- 查看chunk信息
SELECT
    chunk_name,
    range_start,
    range_end,
    pg_size_pretty(total_bytes) as size
FROM timescaledb_information.chunks
WHERE hypertable_name = 'sensor_readings'
ORDER BY range_start DESC
LIMIT 10;

-- 查看压缩状态
SELECT
    chunk_schema,
    chunk_name,
    compression_status,
    before_compression_total_bytes,
    after_compression_total_bytes
FROM timescaledb_information.compressed_chunks
WHERE hypertable_name = 'sensor_readings';
```

### 2. Go代码监控

```go
// 获取数据库统计
stats := store.GetStats()
fmt.Printf("总连接: %d\n", stats.TotalConnections)
fmt.Printf("活跃连接: %d\n", stats.ActiveConnections)
fmt.Printf("空闲连接: %d\n", stats.IdleConnections)
fmt.Printf("总查询数: %d\n", stats.TotalQueries)
fmt.Printf("失败查询: %d\n", stats.FailedQueries)
fmt.Printf("最后查询: %v\n", stats.LastQueryTime)

// 健康检查
if !store.IsHealthy() {
    log.Println("数据库连接异常！")
}
```

### 3. 备份和恢复

```bash
# 备份
pg_dump -U postgres -Fc profibus > profibus_backup.dump

# 恢复
pg_restore -U postgres -d profibus profibus_backup.dump

# 只备份schema
pg_dump -U postgres -s profibus > schema.sql
```

## 常见问题

### Q1: 如何提高写入性能？

A:
1. 使用批量写入（COPY FROM）
2. 增大批量大小（100-1000条）
3. 增加连接池大小
4. 调整PostgreSQL配置（shared_buffers, work_mem等）

### Q2: 数据压缩后如何查询？

A: TimescaleDB会自动解压缩，查询方式完全一样，无需修改代码。

### Q3: 如何选择chunk间隔？

A:
- 高频数据（秒级）：1-6小时
- 中频数据（分钟级）：1天
- 低频数据（小时级）：7天

### Q4: 内存占用过高怎么办？

A:
1. 启用数据压缩
2. 减小chunk间隔
3. 调整shared_buffers（不超过总内存的25%）
4. 设置数据保留策略

### Q5: 如何迁移到其他时序数据库？

A: storage包提供了抽象接口，只需实现新的存储后端即可切换。

## 最佳实践

1. **批量操作**：始终使用批量写入
2. **索引优化**：为常用查询字段创建索引
3. **数据压缩**：7天后自动压缩
4. **保留策略**：根据需求设置（30-90天）
5. **连接池**：合理配置最大/最小连接数
6. **监控告警**：监控数据库大小、查询性能
7. **定期维护**：定期VACUUM、ANALYZE
8. **备份策略**：每日增量备份 + 每周全量备份

## 参考资料

- [TimescaleDB官方文档](https://docs.timescale.com/)
- [PostgreSQL官方文档](https://www.postgresql.org/docs/)
- [pgx驱动文档](https://github.com/jackc/pgx)
