# REST API 文档

go_ProFiBus REST API提供完整的传感器数据查询、事件管理和规则管理功能。

## 目录
- [快速开始](#快速开始)
- [API端点](#api端点)
- [传感器数据API](#传感器数据api)
- [事件管理API](#事件管理api)
- [规则管理API](#规则管理api)
- [错误处理](#错误处理)
- [性能优化](#性能优化)

## 快速开始

### 环境要求
- Go 1.22+
- PostgreSQL 14+ with TimescaleDB extension
- 已运行的go_ProFiBus数据采集系统

### 安装TimescaleDB

**Ubuntu/Debian:**
```bash
# 添加TimescaleDB仓库
sudo apt-get install gnupg postgresql-common apt-transport-https lsb-release wget
sudo /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh
echo "deb https://packagecloud.io/timescale/timescaledb/ubuntu/ $(lsb_release -c -s) main" | sudo tee /etc/apt/sources.list.d/timescaledb.list
wget --quiet -O - https://packagecloud.io/timescale/timescaledb/gpgkey | sudo apt-key add -

# 安装
sudo apt-get update
sudo apt-get install timescaledb-2-postgresql-14

# 配置
sudo timescaledb-tune
sudo systemctl restart postgresql
```

**macOS (Homebrew):**
```bash
brew install timescaledb
```

### 数据库设置

```bash
# 创建数据库
createdb profibus

# 连接并启用TimescaleDB扩展
psql profibus
profibus=# CREATE EXTENSION IF NOT EXISTS timescaledb;

# 执行迁移脚本
psql profibus < migrations/001_initial_schema.sql
```

### 启动API服务器

**方式1：使用示例程序**
```bash
cd examples/rest_api
go run main.go
```

**方式2：自定义启动**
```go
package main

import (
    "go_ProFiBus/api"
    "go_ProFiBus/storage"
)

func main() {
    // 创建数据库连接
    store, _ := storage.NewPostgresStore(&storage.PostgresConfig{
        ConnString: "host=localhost port=5432 user=postgres dbname=profibus sslmode=disable",
    })
    defer store.Close()

    // 创建并启动API服务器
    server, _ := api.NewServer(nil, store) // nil使用默认配置
    server.Start()

    // 等待关闭信号...
}
```

**环境变量配置:**
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=profibus
export API_HOST=0.0.0.0
export API_PORT=8080
export GIN_MODE=release  # debug, release, test
```

## API端点

### 基础端点

#### 健康检查
```http
GET /health
```

**响应:**
```json
{
  "status": "healthy",
  "database": true,
  "time": "2024-01-15T10:30:00Z"
}
```

#### Ping测试
```http
GET /ping
```

**响应:**
```json
{
  "message": "pong",
  "time": "2024-01-15T10:30:00Z"
}
```

## 传感器数据API

### 1. 查询传感器历史数据

```http
GET /api/v1/sensors/:sensor_id/readings?start=<time>&end=<time>&limit=<int>
```

**路径参数:**
- `sensor_id` (string): 传感器ID

**查询参数:**
- `start` (string, 必需): 开始时间，RFC3339格式，例如 `2024-01-15T00:00:00Z`
- `end` (string, 必需): 结束时间，RFC3339格式
- `limit` (int, 可选): 返回记录数限制，默认1000，最大10000

**示例请求:**
```bash
curl "http://localhost:8080/api/v1/sensors/sensor-1/readings?start=2024-01-15T00:00:00Z&end=2024-01-15T23:59:59Z&limit=100"
```

**响应:**
```json
{
  "sensor_id": "sensor-1",
  "start": "2024-01-15T00:00:00Z",
  "end": "2024-01-15T23:59:59Z",
  "count": 100,
  "readings": [
    {
      "timestamp": "2024-01-15T10:30:15Z",
      "sensor_id": "sensor-1",
      "source_id": "source-1",
      "protocol": "MODBUS_RTU",
      "data": {
        "temperature": 25.5,
        "pressure": 101.3,
        "humidity": 60.2
      },
      "quality": 1.0
    }
  ]
}
```

### 2. 批量写入传感器数据

```http
POST /api/v1/sensors/readings
```

**请求体:**
```json
{
  "readings": [
    {
      "timestamp": "2024-01-15T10:30:15Z",
      "sensor_id": "sensor-1",
      "source_id": "source-1",
      "protocol": "MODBUS_RTU",
      "data": {
        "temperature": 25.5,
        "pressure": 101.3
      },
      "quality": 1.0
    }
  ]
}
```

**限制:**
- 单次最多写入1000条数据
- readings数组不能为空

**示例请求:**
```bash
curl -X POST "http://localhost:8080/api/v1/sensors/readings" \
  -H "Content-Type: application/json" \
  -d '{
    "readings": [
      {
        "timestamp": "2024-01-15T10:30:15Z",
        "sensor_id": "sensor-1",
        "protocol": "MODBUS_RTU",
        "data": {"temperature": 25.5},
        "quality": 1.0
      }
    ]
  }'
```

**响应:**
```json
{
  "message": "传感器数据写入成功",
  "count": 1
}
```

### 3. 数据聚合查询

```http
GET /api/v1/sensors/:sensor_id/aggregation?field=<string>&start=<time>&end=<time>&interval=<int>
```

**路径参数:**
- `sensor_id` (string): 传感器ID

**查询参数:**
- `field` (string, 必需): 要聚合的字段名（如 `temperature`）
- `start` (string, 必需): 开始时间
- `end` (string, 必需): 结束时间
- `interval` (int, 必需): 聚合间隔（分钟）

**示例请求:**
```bash
curl "http://localhost:8080/api/v1/sensors/sensor-1/aggregation?field=temperature&start=2024-01-15T00:00:00Z&end=2024-01-15T23:59:59Z&interval=60"
```

**响应:**
```json
{
  "sensor_id": "sensor-1",
  "field": "temperature",
  "start": "2024-01-15T00:00:00Z",
  "end": "2024-01-15T23:59:59Z",
  "interval": 60,
  "count": 24,
  "results": [
    {
      "bucket": "2024-01-15T00:00:00Z",
      "avg": 25.5,
      "min": 23.2,
      "max": 27.8,
      "count": 60
    }
  ]
}
```

## 事件管理API

### 1. 查询事件列表

```http
GET /api/v1/events?status=<string>&type=<string>&severity=<int>&start=<time>&end=<time>&limit=<int>&offset=<int>
```

**查询参数（全部可选）:**
- `status` (string): 事件状态过滤 (`pending`, `confirmed`, `rejected`)
- `type` (string): 事件类型过滤 (`threshold`, `statistical`, `pattern`)
- `severity` (int): 严重程度过滤 (1-5)
- `start` (string): 开始时间
- `end` (string): 结束时间
- `limit` (int): 返回记录数，默认100，最大1000
- `offset` (int): 偏移量，默认0

**示例请求:**
```bash
curl "http://localhost:8080/api/v1/events?status=pending&severity=4&limit=50"
```

**响应:**
```json
{
  "count": 10,
  "limit": 50,
  "offset": 0,
  "events": [
    {
      "id": "evt-12345",
      "type": "threshold",
      "status": "pending",
      "timestamp": "2024-01-15T10:30:00Z",
      "severity": 4,
      "score": 0.95,
      "description": "温度超过阈值",
      "metadata": {
        "sensor_id": "sensor-1",
        "threshold": 30.0,
        "actual": 35.5
      }
    }
  ]
}
```

### 2. 获取事件详情

```http
GET /api/v1/events/:event_id
```

**路径参数:**
- `event_id` (string): 事件ID

**示例请求:**
```bash
curl "http://localhost:8080/api/v1/events/evt-12345"
```

**响应:**
```json
{
  "id": "evt-12345",
  "type": "threshold",
  "status": "pending",
  "timestamp": "2024-01-15T10:30:00Z",
  "severity": 4,
  "score": 0.95,
  "description": "温度超过阈值",
  "metadata": {
    "sensor_id": "sensor-1",
    "threshold": 30.0,
    "actual": 35.5
  },
  "source_id": "source-1",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### 3. 更新事件

```http
PUT /api/v1/events/:event_id
```

**路径参数:**
- `event_id` (string): 事件ID

**请求体（所有字段可选）:**
```json
{
  "status": "confirmed",
  "severity": 5,
  "description": "已确认为真实异常",
  "metadata": {
    "confirmed_by": "user-123",
    "notes": "需要立即处理"
  }
}
```

**示例请求:**
```bash
curl -X PUT "http://localhost:8080/api/v1/events/evt-12345" \
  -H "Content-Type: application/json" \
  -d '{"status": "confirmed", "severity": 5}'
```

**响应:**
```json
{
  "id": "evt-12345",
  "status": "confirmed",
  "severity": 5,
  "...": "..."
}
```

### 4. 事件统计

```http
GET /api/v1/events/stats?start=<time>&end=<time>
```

**查询参数:**
- `start` (string, 可选): 开始时间，默认最近7天
- `end` (string, 可选): 结束时间，默认当前时间

**示例请求:**
```bash
curl "http://localhost:8080/api/v1/events/stats?start=2024-01-08T00:00:00Z"
```

**响应:**
```json
{
  "start": "2024-01-08T00:00:00Z",
  "end": "2024-01-15T10:30:00Z",
  "stats": {
    "total": 1250,
    "by_status": {
      "pending": 450,
      "confirmed": 680,
      "rejected": 120
    },
    "by_type": {
      "threshold": 800,
      "statistical": 350,
      "pattern": 100
    },
    "by_severity": {
      "1": 200,
      "2": 300,
      "3": 400,
      "4": 250,
      "5": 100
    }
  }
}
```

## 规则管理API

### 1. 查询规则列表

```http
GET /api/v1/rules?enabled=<bool>&type=<string>
```

**查询参数（全部可选）:**
- `enabled` (boolean): 是否只返回启用的规则
- `type` (string): 规则类型过滤

**示例请求:**
```bash
curl "http://localhost:8080/api/v1/rules?enabled=true"
```

**响应:**
```json
{
  "count": 5,
  "rules": [
    {
      "id": "rule-12345",
      "name": "高温报警",
      "type": "threshold",
      "description": "温度超过30度触发",
      "config": {
        "field": "temperature",
        "operator": ">",
        "threshold": 30.0
      },
      "enabled": true,
      "created_at": "2024-01-10T08:00:00Z"
    }
  ]
}
```

### 2. 获取规则详情

```http
GET /api/v1/rules/:rule_id
```

**路径参数:**
- `rule_id` (string): 规则ID

**示例请求:**
```bash
curl "http://localhost:8080/api/v1/rules/rule-12345"
```

### 3. 创建规则

```http
POST /api/v1/rules
```

**请求体:**
```json
{
  "name": "高温报警",
  "type": "threshold",
  "description": "温度超过30度触发",
  "config": {
    "field": "temperature",
    "operator": ">",
    "threshold": 30.0
  },
  "enabled": true
}
```

**规则类型:**
- `threshold`: 阈值规则
- `statistical`: 统计规则
- `pattern`: 模式匹配规则
- `similarity`: 相似度规则
- `ml`: 机器学习规则

**示例请求:**
```bash
curl -X POST "http://localhost:8080/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "高温报警",
    "type": "threshold",
    "config": {"field": "temperature", "operator": ">", "threshold": 30.0},
    "enabled": true
  }'
```

**响应:**
```json
{
  "id": "rule-67890",
  "name": "高温报警",
  "type": "threshold",
  "...": "..."
}
```

### 4. 更新规则

```http
PUT /api/v1/rules/:rule_id
```

**路径参数:**
- `rule_id` (string): 规则ID

**请求体（所有字段可选）:**
```json
{
  "name": "超高温报警",
  "description": "温度超过35度触发",
  "config": {
    "threshold": 35.0
  },
  "enabled": false
}
```

**示例请求:**
```bash
curl -X PUT "http://localhost:8080/api/v1/rules/rule-12345" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

### 5. 删除规则

```http
DELETE /api/v1/rules/:rule_id
```

**路径参数:**
- `rule_id` (string): 规则ID

**示例请求:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/rules/rule-12345"
```

**响应:**
```json
{
  "message": "规则删除成功",
  "rule_id": "rule-12345"
}
```

## 错误处理

所有API错误响应遵循统一格式：

```json
{
  "error": "错误描述",
  "details": "详细错误信息（可选）"
}
```

### HTTP状态码

- `200 OK`: 请求成功
- `201 Created`: 资源创建成功
- `204 No Content`: OPTIONS预检请求
- `400 Bad Request`: 请求参数错误
- `404 Not Found`: 资源不存在
- `500 Internal Server Error`: 服务器内部错误
- `503 Service Unavailable`: 服务不可用（如数据库连接失败）

### 常见错误

**400 Bad Request:**
```json
{
  "error": "无效的时间格式，请使用RFC3339格式"
}
```

**404 Not Found:**
```json
{
  "error": "事件不存在"
}
```

**500 Internal Server Error:**
```json
{
  "error": "查询传感器数据失败",
  "details": "pq: relation \"sensor_readings\" does not exist"
}
```

## 性能优化

### 1. 数据库索引

所有查询字段已添加索引：
```sql
-- 传感器数据索引
CREATE INDEX idx_sensor_readings_sensor_id_time ON sensor_readings (sensor_id, time DESC);
CREATE INDEX idx_sensor_readings_time ON sensor_readings (time DESC);

-- 事件索引
CREATE INDEX idx_events_status ON events (status);
CREATE INDEX idx_events_type ON events (type);
CREATE INDEX idx_events_timestamp ON events (timestamp DESC);
```

### 2. TimescaleDB优化

- **分区**: 数据按7天自动分区
- **压缩**: 7天后自动压缩（90%+存储节省）
- **保留策略**: 90天后自动删除

### 3. API性能指标

**目标:**
- P95响应时间 < 100ms
- 吞吐量 > 1000 req/s
- 批量写入: 10000+ samples/s

**优化建议:**
- 使用批量写入API (POST /sensors/readings)
- 合理设置limit参数（建议100-1000）
- 使用时间范围过滤减少查询数据量
- 使用聚合API代替大量原始数据查询

### 4. 连接池配置

```go
storeConfig := &storage.PostgresConfig{
    MaxConnections: 20,  // 最大连接数
    MinConnections: 5,   // 最小连接数
    ConnString:     "...",
}
```

## CORS配置

默认允许所有源(`*`)访问API。生产环境应配置具体域名：

```go
apiConfig := &api.ServerConfig{
    EnableCORS:   true,
    AllowOrigins: []string{
        "https://example.com",
        "https://app.example.com",
    },
}
```

## 完整示例

### Python客户端示例

```python
import requests
from datetime import datetime, timedelta

# API基础URL
BASE_URL = "http://localhost:8080"

# 1. 健康检查
resp = requests.get(f"{BASE_URL}/health")
print(resp.json())

# 2. 查询传感器数据
end = datetime.now()
start = end - timedelta(hours=1)
resp = requests.get(f"{BASE_URL}/api/v1/sensors/sensor-1/readings", params={
    "start": start.isoformat() + "Z",
    "end": end.isoformat() + "Z",
    "limit": 100
})
data = resp.json()
print(f"查询到 {data['count']} 条数据")

# 3. 批量写入数据
readings = [
    {
        "timestamp": datetime.now().isoformat() + "Z",
        "sensor_id": "sensor-1",
        "protocol": "MODBUS_RTU",
        "data": {"temperature": 25.5},
        "quality": 1.0
    }
]
resp = requests.post(f"{BASE_URL}/api/v1/sensors/readings", json={"readings": readings})
print(resp.json())

# 4. 查询事件
resp = requests.get(f"{BASE_URL}/api/v1/events", params={"status": "pending"})
events = resp.json()
print(f"待处理事件: {events['count']} 个")

# 5. 更新事件状态
if events['count'] > 0:
    event_id = events['events'][0]['id']
    resp = requests.put(f"{BASE_URL}/api/v1/events/{event_id}", json={
        "status": "confirmed",
        "severity": 5
    })
    print("事件已确认")

# 6. 创建规则
rule = {
    "name": "高温报警",
    "type": "threshold",
    "config": {
        "field": "temperature",
        "operator": ">",
        "threshold": 30.0
    },
    "enabled": True
}
resp = requests.post(f"{BASE_URL}/api/v1/rules", json=rule)
print(f"规则已创建: {resp.json()['id']}")
```

### JavaScript/Node.js客户端示例

```javascript
const axios = require('axios');

const BASE_URL = 'http://localhost:8080';

async function main() {
  // 健康检查
  const health = await axios.get(`${BASE_URL}/health`);
  console.log(health.data);

  // 查询传感器数据
  const end = new Date().toISOString();
  const start = new Date(Date.now() - 3600000).toISOString();
  const readings = await axios.get(`${BASE_URL}/api/v1/sensors/sensor-1/readings`, {
    params: { start, end, limit: 100 }
  });
  console.log(`查询到 ${readings.data.count} 条数据`);

  // 批量写入
  await axios.post(`${BASE_URL}/api/v1/sensors/readings`, {
    readings: [{
      timestamp: new Date().toISOString(),
      sensor_id: 'sensor-1',
      protocol: 'MODBUS_RTU',
      data: { temperature: 25.5 },
      quality: 1.0
    }]
  });
  console.log('数据已写入');

  // 查询事件
  const events = await axios.get(`${BASE_URL}/api/v1/events`, {
    params: { status: 'pending' }
  });
  console.log(`待处理事件: ${events.data.count} 个`);
}

main().catch(console.error);
```

## 故障排查

### 数据库连接失败

```
错误: 数据库健康检查失败: connection refused
```

**解决方法:**
1. 确认PostgreSQL已启动: `sudo systemctl status postgresql`
2. 检查连接参数（host, port, user, password）
3. 检查防火墙: `sudo ufw allow 5432/tcp`
4. 检查pg_hba.conf配置

### TimescaleDB扩展未安装

```
错误: pq: extension "timescaledb" does not exist
```

**解决方法:**
```bash
psql profibus -c "CREATE EXTENSION timescaledb;"
```

### 表不存在

```
错误: pq: relation "sensor_readings" does not exist
```

**解决方法:**
```bash
psql profibus < migrations/001_initial_schema.sql
```

## 安全建议

1. **使用HTTPS**: 生产环境必须使用HTTPS
2. **认证**: 添加JWT或OAuth2认证（当前版本未实现）
3. **速率限制**: 防止API滥用
4. **输入验证**: 所有API已实现基础验证
5. **CORS配置**: 限制允许的源域名
6. **SQL注入防护**: 使用pgx参数化查询（已实现）

## 下一步

- [ ] 添加JWT认证中间件
- [ ] 实现WebSocket实时数据推送
- [ ] 添加Swagger/OpenAPI文档
- [ ] 实现GraphQL接口
- [ ] 添加速率限制中间件
- [ ] 实现数据导出功能（CSV, Excel）
