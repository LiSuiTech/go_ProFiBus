# WebSocket 实时追踪系统

## 概述

本系统实现了基于WebSocket的实时数据流追踪功能，允许前端实时监控Pipeline中数据的流动情况。

## 架构组件

### 1. 核心组件

#### Tracer (内部/application/tracing/tracer.go)
- 负责收集和记录追踪事件
- 提供发布-订阅机制
- 批量写入数据库（100条或1秒刷新）
- 支持多个订阅者

#### WebSocket Hub (internal/interfaces/websocket/hub.go)
- 管理所有WebSocket连接
- 广播追踪事件到所有客户端
- 支持客户端过滤

#### WebSocket Client (internal/interfaces/websocket/client.go)
- 管理单个WebSocket连接
- 实现读写pump模式
- 支持客户端自定义过滤器
- 自动心跳检测（60秒）

#### WebSocket Handler (internal/interfaces/websocket/handler.go)
- HTTP到WebSocket的升级处理
- 连接初始化

### 2. 集成流程

```
Pipeline Component
       ↓
    Tracer.TraceDataFlow()
       ↓
  Tracer.Subscribe()  → TraceEvent Channel
       ↓
  Server.bridgeTracerToWebSocket()
       ↓
  Hub.BroadcastEvent()
       ↓
  Client (WebSocket) → 前端
```

## API 端点

### WebSocket 端点

**URL:** `ws://localhost:8080/ws/trace`

**连接方式:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/trace');

ws.onmessage = function(event) {
    const traceEvent = JSON.parse(event.data);
    console.log('Received trace event:', traceEvent);
};
```

### 事件格式

追踪事件JSON结构：
```json
{
  "id": "uuid-string",
  "pipeline_id": "pipeline-001",
  "sample_id": "sample-123",
  "component_type": "processor",
  "component_id": "proc-001",
  "component_name": "DataValidator",
  "action": "process",
  "timestamp": "2025-01-16T10:30:45.123Z",
  "duration_ms": 150,
  "status": "success",
  "error_message": "",
  "metadata": {
    "key": "value"
  },
  "data_snapshot": {
    "temperature": 42.5,
    "unit": "°C"
  }
}
```

### 客户端过滤器

客户端可以发送过滤器来只接收特定事件：

```javascript
const filter = {
  "pipeline_ids": ["pipeline-001"],
  "component_ids": ["proc-001", "proc-002"],
  "component_type": "processor",
  "sample_ids": ["sample-123"]
};

ws.send(JSON.stringify(filter));
```

**过滤器字段:**
- `pipeline_ids`: 只接收指定Pipeline的事件
- `component_ids`: 只接收指定Component的事件
- `component_type`: 只接收指定类型的Component（source, processor, analyzer, sink）
- `sample_ids`: 只接收指定Sample的事件

## 使用示例

### 1. 后端集成

```go
package main

import (
    "go_ProFiBus/api"
    "go_ProFiBus/internal/application/tracing"
    "go_ProFiBus/internal/infrastructure/storage"
)

func main() {
    // 1. 创建数据库连接
    store, _ := storage.NewPostgresStore(dbConfig)

    // 2. 创建TraceRepository
    traceRepo := storage.NewTraceRepository(store)

    // 3. 创建Tracer
    tracer := tracing.NewTracer(traceRepo, 100)

    // 4. 创建API服务器（自动集成WebSocket）
    server, _ := api.NewServer(serverConfig, store, tracer)

    // 5. 启动服务器
    server.Start()
}
```

### 2. Pipeline中使用Tracer

```go
// 在Pipeline中记录追踪事件
func (p *Pipeline) processSample(sample DataSample) {
    ctx := context.Background()

    // 记录进入事件
    p.tracer.TraceDataFlow(ctx, &interfaces.TraceEvent{
        PipelineID:    p.id,
        SampleID:      sample.ID,
        ComponentType: "processor",
        ComponentID:   "proc-001",
        ComponentName: "DataValidator",
        Action:        "enter",
        Timestamp:     time.Now(),
        Status:        "success",
    })

    // 执行处理逻辑
    result, duration, err := processor.Process(sample)

    // 记录处理结果
    status := "success"
    if err != nil {
        status = "error"
    }

    p.tracer.TraceDataFlow(ctx, &interfaces.TraceEvent{
        PipelineID:    p.id,
        SampleID:      sample.ID,
        ComponentType: "processor",
        ComponentID:   "proc-001",
        ComponentName: "DataValidator",
        Action:        "process",
        Timestamp:     time.Now(),
        Duration:      duration,
        Status:        status,
        Error:         err.Error(),
        DataSnapshot: map[string]interface{}{
            "input":  sample.Data,
            "output": result,
        },
    })
}
```

### 3. 前端连接

参见 `examples/websocket_client.html` 完整示例。

```html
<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Trace Viewer</title>
</head>
<body>
    <div id="events"></div>

    <script>
        const ws = new WebSocket('ws://localhost:8080/ws/trace');

        ws.onopen = () => {
            console.log('Connected to WebSocket');
        };

        ws.onmessage = (event) => {
            const traceEvent = JSON.parse(event.data);
            displayEvent(traceEvent);
        };

        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        function displayEvent(event) {
            const div = document.createElement('div');
            div.textContent = `[${event.timestamp}] ${event.component_name}: ${event.action} - ${event.status}`;
            document.getElementById('events').appendChild(div);
        }
    </script>
</body>
</html>
```

## 性能特性

### 1. 缓冲写入
- Tracer使用100条事件缓冲
- 每1秒自动刷新到数据库
- 缓冲区满时立即刷新

### 2. WebSocket优化
- 256条消息的发送缓冲区
- 自动心跳检测（54秒间隔）
- 60秒连接超时
- 512KB最大消息大小

### 3. 背压控制
- 客户端消息队列满时自动跳过事件
- 避免慢客户端影响整体性能

### 4. 多客户端支持
- 支持多个客户端同时连接
- 每个客户端独立过滤器
- 客户端断开不影响其他客户端

## 数据库Schema

追踪事件存储在TimescaleDB hypertable中：

```sql
CREATE TABLE trace_events (
    id UUID PRIMARY KEY,
    pipeline_id VARCHAR(255) NOT NULL,
    sample_id VARCHAR(255) NOT NULL,
    component_type VARCHAR(50) NOT NULL,
    component_id VARCHAR(255) NOT NULL,
    component_name VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    duration_ms INTEGER,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    metadata JSONB,
    data_snapshot JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

SELECT create_hypertable('trace_events', 'timestamp');
```

## 安全考虑

### 1. Origin验证
当前配置允许所有来源，生产环境需要修改：

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        // 生产环境应该验证Origin
        origin := r.Header.Get("Origin")
        return origin == "https://yourdomain.com"
    },
}
```

### 2. 认证
建议添加JWT认证：

```go
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
    // 验证JWT token
    token := r.URL.Query().Get("token")
    if !validateToken(token) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // 升级连接
    conn, err := upgrader.Upgrade(w, r, nil)
    // ...
}
```

### 3. 速率限制
建议添加连接速率限制防止DoS攻击。

## 监控指标

可以添加以下Prometheus指标：

- `websocket_connections_total`: 当前连接数
- `websocket_messages_sent_total`: 发送的消息总数
- `websocket_messages_dropped_total`: 丢弃的消息总数
- `tracer_events_total`: 追踪事件总数
- `tracer_buffer_size`: 当前缓冲区大小

## 故障排查

### 连接失败
1. 检查服务器是否启动：`curl http://localhost:8080/health`
2. 检查WebSocket端点：使用浏览器开发者工具查看Network标签
3. 检查防火墙设置

### 事件丢失
1. 检查客户端过滤器设置
2. 检查Tracer订阅是否正常
3. 查看服务器日志

### 性能问题
1. 减少事件频率
2. 增加缓冲区大小
3. 优化数据库写入

## 下一步开发

Phase 2剩余任务：
- [ ] Step 4: 集成Tracer到Pipeline
- [ ] Step 5: REST API端点（拓扑查询、历史追踪查询）
- [ ] Step 6: Vue 3 Dashboard前端
- [ ] Step 7: 集成测试

详见 `/root/.claude/plans/composed-exploring-clarke.md`
