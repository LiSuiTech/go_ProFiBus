# Phase 2 API 使用示例

本文档提供了 Phase 2 新增 REST API 的使用示例。

## 目录

1. [管道管理 API](#管道管理-api)
2. [追踪数据 API](#追踪数据-api)
3. [性能指标 API](#性能指标-api)
4. [WebSocket 实时追踪](#websocket-实时追踪)

## 基础信息

**Base URL**: `http://localhost:8080`
**API Version**: `v1`
**Content-Type**: `application/json`

---

## 管道管理 API

### 1. 获取所有管道列表

**Endpoint**: `GET /api/v1/pipelines`

**请求示例**:
```bash
curl -X GET http://localhost:8080/api/v1/pipelines
```

**响应示例**:
```json
{
  "pipelines": [
    {
      "id": "sensor-pipeline-1",
      "name": "sensor-pipeline-1",
      "running": true,
      "status": "running"
    },
    {
      "id": "sensor-pipeline-2",
      "name": "sensor-pipeline-2",
      "running": false,
      "status": "stopped"
    }
  ],
  "total": 2,
  "running": 1,
  "stopped": 1
}
```

---

### 2. 获取管道拓扑结构

**Endpoint**: `GET /api/v1/pipelines/{id}/topology`

**请求示例**:
```bash
curl -X GET http://localhost:8080/api/v1/pipelines/sensor-pipeline-1/topology
```

**响应示例**:
```json
{
  "pipeline_id": "sensor-pipeline-1",
  "name": "sensor-pipeline-1",
  "running": true,
  "components": {
    "source": {
      "id": "sensor-001",
      "name": "温度传感器",
      "type": "source",
      "description": "采集温度数据"
    },
    "processors": [
      {
        "id": "temp-converter",
        "name": "temp-converter",
        "type": "processor"
      }
    ],
    "analyzers": [
      {
        "id": "rule-engine",
        "name": "rule-engine",
        "type": "analyzer"
      }
    ],
    "sinks": [
      {
        "id": "db-sink",
        "name": "db-sink",
        "type": "sink"
      }
    ]
  }
}
```

---

### 3. 获取管道状态

**Endpoint**: `GET /api/v1/pipelines/{id}/status`

**请求示例**:
```bash
curl -X GET http://localhost:8080/api/v1/pipelines/sensor-pipeline-1/status
```

**响应示例**:
```json
{
  "pipeline_id": "sensor-pipeline-1",
  "name": "sensor-pipeline-1",
  "running": true,
  "status": "running",
  "samples_processed": 12543,
  "errors": 23,
  "last_sample_time": "2024-01-20T10:30:45Z"
}
```

---

### 4. 启动管道

**Endpoint**: `POST /api/v1/pipelines/{id}/start`

**请求示例**:
```bash
curl -X POST http://localhost:8080/api/v1/pipelines/sensor-pipeline-1/start
```

**响应示例**:
```json
{
  "message": "Pipeline started successfully",
  "pipeline_id": "sensor-pipeline-1"
}
```

---

### 5. 停止管道

**Endpoint**: `POST /api/v1/pipelines/{id}/stop`

**请求示例**:
```bash
curl -X POST http://localhost:8080/api/v1/pipelines/sensor-pipeline-1/stop
```

**响应示例**:
```json
{
  "message": "Pipeline stopped successfully",
  "pipeline_id": "sensor-pipeline-1"
}
```

---

## 追踪数据 API

### 1. 查询追踪记录

**Endpoint**: `GET /api/v1/traces`

**查询参数**:
| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `pipeline_id` | string | 管道ID | `sensor-pipeline-1` |
| `sample_id` | string | 样本ID | `sample-123` |
| `component_type` | string | 组件类型 | `source`, `processor`, `analyzer`, `sink` |
| `component_id` | string | 组件ID | `temp-converter` |
| `action` | string | 动作 | `enter`, `process`, `exit`, `error` |
| `status` | string | 状态 | `success`, `error`, `skip` |
| `start_time` | string | 开始时间 (RFC3339) | `2024-01-20T10:00:00Z` |
| `end_time` | string | 结束时间 (RFC3339) | `2024-01-20T11:00:00Z` |
| `limit` | int | 限制数量 (1-1000) | `100` |
| `offset` | int | 偏移量 | `0` |
| `order_by` | string | 排序字段 | `timestamp` |
| `order_desc` | bool | 降序排序 | `true` |

**请求示例**:
```bash
# 查询指定管道的最近100条追踪记录
curl -X GET "http://localhost:8080/api/v1/traces?pipeline_id=sensor-pipeline-1&limit=100"

# 查询指定时间范围内的错误记录
curl -X GET "http://localhost:8080/api/v1/traces?status=error&start_time=2024-01-20T10:00:00Z&end_time=2024-01-20T11:00:00Z"

# 查询特定组件的追踪记录
curl -X GET "http://localhost:8080/api/v1/traces?component_type=processor&component_id=temp-converter"
```

**响应示例**:
```json
{
  "traces": [
    {
      "id": "trace-001",
      "pipeline_id": "sensor-pipeline-1",
      "sample_id": "sample-123",
      "component_type": "processor",
      "component_id": "temp-converter",
      "component_name": "temp-converter",
      "action": "process",
      "timestamp": "2024-01-20T10:30:45.123Z",
      "duration": 2500000,
      "status": "success",
      "metadata": {
        "input_temp_f": 77.0,
        "output_temp_c": 25.0
      }
    }
  ],
  "total": 1,
  "filter": {
    "pipeline_id": "sensor-pipeline-1",
    "limit": 100,
    "offset": 0
  }
}
```

---

### 2. 根据样本ID获取完整追踪链路

**Endpoint**: `GET /api/v1/traces/samples/{sample_id}`

**请求示例**:
```bash
curl -X GET http://localhost:8080/api/v1/traces/samples/sample-123
```

**响应示例**:
```json
{
  "sample_id": "sample-123",
  "total_events": 5,
  "trace_chain": [
    {
      "component_type": "source",
      "component_id": "sensor-001",
      "component_name": "温度传感器",
      "action": "enter",
      "timestamp": "2024-01-20T10:30:45.100Z",
      "duration_ms": 5,
      "status": "success"
    },
    {
      "component_type": "processor",
      "component_id": "temp-converter",
      "component_name": "temp-converter",
      "action": "process",
      "timestamp": "2024-01-20T10:30:45.110Z",
      "duration_ms": 2,
      "status": "success"
    },
    {
      "component_type": "analyzer",
      "component_id": "rule-engine",
      "component_name": "rule-engine",
      "action": "process",
      "timestamp": "2024-01-20T10:30:45.115Z",
      "duration_ms": 8,
      "status": "success"
    },
    {
      "component_type": "sink",
      "component_id": "db-sink",
      "component_name": "db-sink",
      "action": "process",
      "timestamp": "2024-01-20T10:30:45.125Z",
      "duration_ms": 15,
      "status": "success"
    }
  ],
  "events": [...]
}
```

---

### 3. 获取追踪统计信息

**Endpoint**: `GET /api/v1/traces/stats`

**查询参数**:
| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `pipeline_id` | string | 管道ID (可选) | `sensor-pipeline-1` |
| `start_time` | string | 开始时间 (可选，默认1小时前) | `2024-01-20T10:00:00Z` |
| `end_time` | string | 结束时间 (可选，默认当前时间) | `2024-01-20T11:00:00Z` |

**请求示例**:
```bash
# 查询最近1小时的统计信息
curl -X GET http://localhost:8080/api/v1/traces/stats

# 查询指定管道和时间范围的统计
curl -X GET "http://localhost:8080/api/v1/traces/stats?pipeline_id=sensor-pipeline-1&start_time=2024-01-20T10:00:00Z"
```

**响应示例**:
```json
{
  "start_time": "2024-01-20T09:30:45Z",
  "end_time": "2024-01-20T10:30:45Z",
  "pipeline_id": "sensor-pipeline-1",
  "stats": {
    "total_events": 10523,
    "success_events": 10234,
    "error_events": 289,
    "skip_events": 0,
    "by_component_type": {
      "source": 2543,
      "processor": 2543,
      "analyzer": 2543,
      "sink": 2543,
      "pipeline": 351
    },
    "by_action": {
      "enter": 2543,
      "process": 7629,
      "exit": 351,
      "error": 0
    },
    "avg_duration_ms": 12.5
  }
}
```

---

## 性能指标 API

### 1. 获取管道性能指标

**Endpoint**: `GET /api/v1/pipelines/{id}/metrics`

**查询参数**:
| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `start_time` | string | 开始时间 (可选，默认1小时前) | `2024-01-20T10:00:00Z` |
| `end_time` | string | 结束时间 (可选，默认当前时间) | `2024-01-20T11:00:00Z` |

**请求示例**:
```bash
# 查询管道最近1小时的性能指标
curl -X GET http://localhost:8080/api/v1/pipelines/sensor-pipeline-1/metrics

# 查询指定时间范围的性能指标
curl -X GET "http://localhost:8080/api/v1/pipelines/sensor-pipeline-1/metrics?start_time=2024-01-20T10:00:00Z&end_time=2024-01-20T11:00:00Z"
```

**响应示例**:
```json
{
  "pipeline_id": "sensor-pipeline-1",
  "start_time": "2024-01-20T09:30:45Z",
  "end_time": "2024-01-20T10:30:45Z",
  "summary": {
    "total_samples": 2543,
    "success_samples": 2498,
    "error_samples": 45,
    "success_rate": 98.23,
    "avg_duration_ms": 32,
    "max_duration_ms": 125,
    "min_duration_ms": 15,
    "throughput_per_sec": 42.38
  },
  "components": [
    {
      "component_id": "sensor-001",
      "component_type": "source",
      "component_name": "温度传感器",
      "event_count": 2543,
      "avg_duration_ms": 5,
      "max_duration_ms": 12,
      "min_duration_ms": 3,
      "error_count": 0,
      "error_rate": 0.0
    },
    {
      "component_id": "temp-converter",
      "component_type": "processor",
      "component_name": "temp-converter",
      "event_count": 2543,
      "avg_duration_ms": 2,
      "max_duration_ms": 5,
      "min_duration_ms": 1,
      "error_count": 0,
      "error_rate": 0.0
    },
    {
      "component_id": "rule-engine",
      "component_type": "analyzer",
      "component_name": "rule-engine",
      "event_count": 2543,
      "avg_duration_ms": 8,
      "max_duration_ms": 45,
      "min_duration_ms": 3,
      "error_count": 23,
      "error_rate": 0.9
    },
    {
      "component_id": "db-sink",
      "component_type": "sink",
      "component_name": "db-sink",
      "event_count": 2498,
      "avg_duration_ms": 15,
      "max_duration_ms": 78,
      "min_duration_ms": 8,
      "error_count": 22,
      "error_rate": 0.88
    }
  ]
}
```

---

### 2. 获取组件性能指标

**Endpoint**: `GET /api/v1/metrics/component`

**查询参数**:
| 参数 | 类型 | 必填 | 说明 | 示例 |
|------|------|------|------|------|
| `pipeline_id` | string | 是 | 管道ID | `sensor-pipeline-1` |
| `component_id` | string | 是 | 组件ID | `temp-converter` |
| `start_time` | string | 否 | 开始时间 (默认1小时前) | `2024-01-20T10:00:00Z` |
| `end_time` | string | 否 | 结束时间 (默认当前时间) | `2024-01-20T11:00:00Z` |

**请求示例**:
```bash
curl -X GET "http://localhost:8080/api/v1/metrics/component?pipeline_id=sensor-pipeline-1&component_id=temp-converter"
```

**响应示例**:
```json
{
  "pipeline_id": "sensor-pipeline-1",
  "component_id": "temp-converter",
  "component_type": "processor",
  "component_name": "temp-converter",
  "start_time": "2024-01-20T09:30:45Z",
  "end_time": "2024-01-20T10:30:45Z",
  "metrics": {
    "event_count": 2543,
    "avg_duration_ms": 2,
    "max_duration_ms": 5,
    "min_duration_ms": 1,
    "error_count": 0,
    "error_rate": 0.0
  }
}
```

---

### 3. 获取系统整体性能指标

**Endpoint**: `GET /api/v1/metrics/system`

**查询参数**:
| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `start_time` | string | 开始时间 (可选，默认1小时前) | `2024-01-20T10:00:00Z` |
| `end_time` | string | 结束时间 (可选，默认当前时间) | `2024-01-20T11:00:00Z` |

**请求示例**:
```bash
curl -X GET http://localhost:8080/api/v1/metrics/system
```

**响应示例**:
```json
{
  "start_time": "2024-01-20T09:30:45Z",
  "end_time": "2024-01-20T10:30:45Z",
  "system_metrics": {
    "total_pipelines": 0,
    "total_samples": 0,
    "success_samples": 0,
    "error_samples": 0,
    "avg_duration_ms": 0,
    "throughput_per_sec": 0
  },
  "note": "System metrics aggregation not yet implemented"
}
```

---

## WebSocket 实时追踪

### 连接 WebSocket

**Endpoint**: `ws://localhost:8080/ws/trace`

**JavaScript 示例**:
```javascript
// 创建 WebSocket 连接
const ws = new WebSocket('ws://localhost:8080/ws/trace');

// 连接建立
ws.onopen = function(event) {
  console.log('WebSocket connected');
};

// 接收消息
ws.onmessage = function(event) {
  const traceEvent = JSON.parse(event.data);
  console.log('Received trace event:', traceEvent);

  // 处理追踪事件
  handleTraceEvent(traceEvent);
};

// 连接错误
ws.onerror = function(error) {
  console.error('WebSocket error:', error);
};

// 连接关闭
ws.onclose = function(event) {
  console.log('WebSocket closed:', event.code, event.reason);

  // 重连逻辑
  setTimeout(() => {
    console.log('Reconnecting...');
    connectWebSocket();
  }, 3000);
};

// 处理追踪事件
function handleTraceEvent(event) {
  // 更新拓扑图中的节点状态
  updateNodeStatus(event.component_id, event.status);

  // 显示数据流动画
  if (event.action === 'process') {
    animateDataFlow(event);
  }

  // 更新性能指标
  if (event.duration) {
    updatePerformanceMetrics(event.component_id, event.duration);
  }

  // 处理错误
  if (event.status === 'error') {
    showError(event);
  }
}
```

**Python 示例**:
```python
import asyncio
import websockets
import json

async def connect_websocket():
    uri = "ws://localhost:8080/ws/trace"

    async with websockets.connect(uri) as websocket:
        print("WebSocket connected")

        try:
            async for message in websocket:
                trace_event = json.loads(message)
                print(f"Received trace event: {trace_event}")

                # 处理追踪事件
                handle_trace_event(trace_event)

        except websockets.ConnectionClosed:
            print("WebSocket connection closed")

def handle_trace_event(event):
    # 处理追踪事件
    print(f"Component: {event['component_name']}")
    print(f"Action: {event['action']}")
    print(f"Status: {event['status']}")
    print(f"Duration: {event.get('duration', 0) / 1000000:.2f}ms")

# 运行
asyncio.run(connect_websocket())
```

---

## 错误处理

所有 API 在出错时会返回标准的错误响应：

**错误响应格式**:
```json
{
  "error": "错误类型",
  "message": "详细错误信息"
}
```

**常见状态码**:
| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源未找到 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

---

## 完整示例：监控管道

以下是一个完整的监控管道的示例流程：

```bash
#!/bin/bash

BASE_URL="http://localhost:8080/api/v1"

# 1. 获取所有管道
echo "=== 获取所有管道 ==="
curl -X GET "$BASE_URL/pipelines"
echo -e "\n"

# 2. 获取特定管道的拓扑结构
echo "=== 获取管道拓扑 ==="
curl -X GET "$BASE_URL/pipelines/sensor-pipeline-1/topology"
echo -e "\n"

# 3. 获取管道状态
echo "=== 获取管道状态 ==="
curl -X GET "$BASE_URL/pipelines/sensor-pipeline-1/status"
echo -e "\n"

# 4. 获取管道性能指标
echo "=== 获取性能指标 ==="
curl -X GET "$BASE_URL/pipelines/sensor-pipeline-1/metrics"
echo -e "\n"

# 5. 查询最近的追踪记录
echo "=== 查询追踪记录 ==="
curl -X GET "$BASE_URL/traces?pipeline_id=sensor-pipeline-1&limit=10"
echo -e "\n"

# 6. 获取追踪统计
echo "=== 获取追踪统计 ==="
curl -X GET "$BASE_URL/traces/stats?pipeline_id=sensor-pipeline-1"
echo -e "\n"
```

---

## 下一步

- 查看 [ARCHITECTURE.md](../ARCHITECTURE.md) 了解系统架构
- 查看 [PHASE2_PLAN.md](../PHASE2_PLAN.md) 了解 Phase 2 实施计划
- 前端 Dashboard 开发请参考 Vue 3 组件实现
