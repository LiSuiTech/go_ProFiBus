# Phase 2: 数据流可视化系统 - 实施计划

## 目标
实现实时的数据流可视化系统，让用户能够直观地看到数据在Pipeline中的流动过程、处理状态和性能指标。

## 技术栈

### 后端
- **Go**: 数据流追踪和WebSocket服务
- **PostgreSQL**: 存储追踪数据和拓扑配置
- **WebSocket**: 实时推送数据流事件

### 前端
- **Vue 3**: 使用Composition API
- **Pinia**: 状态管理
- **Vue Router**: 路由管理
- **Vite**: 构建工具
- **D3.js 或 Cytoscape.js**: 拓扑图可视化
- **ECharts**: 性能指标图表
- **Element Plus**: UI组件库
- **Native WebSocket**: 实时通信

## 实施步骤

### Step 1: 后端 - 数据流追踪器 (Tracer)

#### 1.1 创建 Tracer 接口
**文件**: `pkg/interfaces/tracer.go`

```go
package interfaces

import (
    "context"
    "time"
)

// Tracer 数据流追踪器接口
type Tracer interface {
    // TraceDataFlow 追踪数据流经某个组件
    TraceDataFlow(ctx context.Context, event *TraceEvent) error

    // GetTraces 获取追踪记录
    GetTraces(ctx context.Context, filter TraceFilter) ([]TraceEvent, error)

    // Subscribe 订阅追踪事件
    Subscribe() <-chan TraceEvent

    // Close 关闭追踪器
    Close() error
}

// TraceEvent 追踪事件
type TraceEvent struct {
    ID           string                 // 事件ID
    PipelineID   string                 // 管道ID
    SampleID     string                 // 数据样本ID
    ComponentType string                // 组件类型: source, processor, analyzer, sink
    ComponentID  string                 // 组件ID
    Action       string                 // 动作: enter, process, exit, error
    Timestamp    time.Time              // 时间戳
    Duration     time.Duration          // 处理耗时
    Status       string                 // 状态: success, error
    Error        string                 // 错误信息
    Metadata     map[string]interface{} // 元数据
}

// TraceFilter 追踪过滤器
type TraceFilter struct {
    PipelineID    *string
    ComponentType *string
    StartTime     *time.Time
    EndTime       *time.Time
    Limit         int
}
```

#### 1.2 实现 Tracer
**文件**: `internal/application/tracing/tracer.go`

```go
package tracing

import (
    "context"
    "go_ProFiBus/pkg/interfaces"
    "sync"
)

type TracerImpl struct {
    events    []interfaces.TraceEvent
    mu        sync.RWMutex
    subscribers []chan interfaces.TraceEvent
    repository interfaces.TraceRepository
}

func NewTracer(repo interfaces.TraceRepository) *TracerImpl {
    return &TracerImpl{
        events:      make([]interfaces.TraceEvent, 0),
        subscribers: make([]chan interfaces.TraceEvent, 0),
        repository:  repo,
    }
}

// 实现 Tracer 接口方法...
```

#### 1.3 集成 Tracer 到 Pipeline
**修改文件**: `internal/application/orchestrator/pipeline.go`

在 Pipeline 的 processSample 方法中添加追踪：

```go
func (p *Pipeline) processSample(ctx context.Context, sample interfaces.DataSample) error {
    sampleID := generateSampleID(sample)

    // 追踪进入 Pipeline
    if p.tracer != nil {
        p.tracer.TraceDataFlow(ctx, &interfaces.TraceEvent{
            PipelineID:    p.name,
            SampleID:      sampleID,
            ComponentType: "pipeline",
            ComponentID:   p.name,
            Action:        "enter",
            Timestamp:     time.Now(),
        })
    }

    // 处理器链追踪
    for _, processor := range p.processors {
        startTime := time.Now()
        processedSample, err := processor.Process(ctx, sample)
        duration := time.Since(startTime)

        if p.tracer != nil {
            status := "success"
            errMsg := ""
            if err != nil {
                status = "error"
                errMsg = err.Error()
            }

            p.tracer.TraceDataFlow(ctx, &interfaces.TraceEvent{
                PipelineID:    p.name,
                SampleID:      sampleID,
                ComponentType: "processor",
                ComponentID:   processor.GetName(),
                Action:        "process",
                Timestamp:     time.Now(),
                Duration:      duration,
                Status:        status,
                Error:         errMsg,
            })
        }

        if err != nil {
            return err
        }
        sample = processedSample
    }

    // 类似地追踪 Analyzer 和 Sink...
}
```

### Step 2: 数据库 Schema 扩展

#### 2.1 创建迁移文件
**文件**: `migrations/002_add_tracing.sql`

```sql
-- 追踪事件表
CREATE TABLE IF NOT EXISTS trace_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id VARCHAR(255) NOT NULL,
    sample_id VARCHAR(255) NOT NULL,
    component_type VARCHAR(50) NOT NULL,
    component_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    duration_ms INTEGER,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 索引优化查询
CREATE INDEX idx_trace_events_pipeline ON trace_events(pipeline_id);
CREATE INDEX idx_trace_events_sample ON trace_events(sample_id);
CREATE INDEX idx_trace_events_timestamp ON trace_events(timestamp DESC);
CREATE INDEX idx_trace_events_component ON trace_events(component_type, component_id);

-- 使用 TimescaleDB 将表转换为 hypertable（如果需要）
SELECT create_hypertable('trace_events', 'timestamp', if_not_exists => TRUE);

-- 拓扑配置表（存储Pipeline配置用于前端渲染）
CREATE TABLE IF NOT EXISTS pipeline_topology (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 性能指标聚合视图
CREATE MATERIALIZED VIEW IF NOT EXISTS pipeline_metrics AS
SELECT
    pipeline_id,
    component_type,
    component_id,
    DATE_TRUNC('minute', timestamp) as time_bucket,
    COUNT(*) as event_count,
    AVG(duration_ms) as avg_duration,
    MAX(duration_ms) as max_duration,
    MIN(duration_ms) as min_duration,
    COUNT(CASE WHEN status = 'error' THEN 1 END) as error_count
FROM trace_events
GROUP BY pipeline_id, component_type, component_id, time_bucket;

CREATE UNIQUE INDEX ON pipeline_metrics (pipeline_id, component_type, component_id, time_bucket);
```

#### 2.2 实现 TraceRepository
**文件**: `internal/infrastructure/storage/trace_repository.go`

```go
package storage

import (
    "context"
    "go_ProFiBus/pkg/interfaces"
    "go_ProFiBus/storage"
)

type TraceRepository struct {
    store *storage.PostgresStore
}

func NewTraceRepository(store *storage.PostgresStore) *TraceRepository {
    return &TraceRepository{store: store}
}

func (r *TraceRepository) SaveTraceEvent(ctx context.Context, event *interfaces.TraceEvent) error {
    sql := `
        INSERT INTO trace_events
        (pipeline_id, sample_id, component_type, component_id, action,
         timestamp, duration_ms, status, error_message, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
    _, err := r.store.Exec(sql,
        event.PipelineID,
        event.SampleID,
        event.ComponentType,
        event.ComponentID,
        event.Action,
        event.Timestamp,
        event.Duration.Milliseconds(),
        event.Status,
        event.Error,
        event.Metadata,
    )
    return err
}

// 批量写入优化
func (r *TraceRepository) SaveTraceEventsBatch(ctx context.Context, events []*interfaces.TraceEvent) error {
    // 使用 COPY 批量插入
    // ...
}
```

### Step 3: WebSocket 推送服务

#### 3.1 创建 WebSocket Hub
**文件**: `internal/interfaces/websocket/hub.go`

```go
package websocket

import (
    "go_ProFiBus/pkg/interfaces"
    "sync"
)

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan interfaces.TraceEvent
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan interfaces.TraceEvent, 1000),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()

        case event := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.send <- event:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
            h.mu.RUnlock()
        }
    }
}

func (h *Hub) BroadcastTraceEvent(event interfaces.TraceEvent) {
    h.broadcast <- event
}
```

#### 3.2 WebSocket Handler
**文件**: `internal/interfaces/websocket/handler.go`

```go
package websocket

import (
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "net/http"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // 生产环境需要验证origin
    },
}

func ServeWs(hub *Hub, c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }

    client := &Client{
        hub:  hub,
        conn: conn,
        send: make(chan interfaces.TraceEvent, 256),
    }

    client.hub.register <- client

    go client.writePump()
    go client.readPump()
}
```

#### 3.3 集成到 API Server
**修改文件**: `api/server.go`

```go
import (
    "go_ProFiBus/internal/interfaces/websocket"
)

func SetupRouter(tracer interfaces.Tracer) *gin.Engine {
    router := gin.Default()

    // WebSocket Hub
    hub := websocket.NewHub()
    go hub.Run()

    // 订阅 Tracer 事件并广播
    go func() {
        eventChan := tracer.Subscribe()
        for event := range eventChan {
            hub.BroadcastTraceEvent(event)
        }
    }()

    // WebSocket endpoint
    router.GET("/ws/trace", func(c *gin.Context) {
        websocket.ServeWs(hub, c)
    })

    // REST API endpoints
    api := router.Group("/api/v1")
    {
        api.GET("/pipelines", GetPipelines)
        api.GET("/pipelines/:id/topology", GetPipelineTopology)
        api.GET("/traces", GetTraces)
        api.GET("/metrics", GetMetrics)
    }

    return router
}
```

### Step 4: Vue 3 Dashboard 前端

#### 4.1 初始化项目
```bash
cd web
npm create vite@latest dashboard -- --template vue
cd dashboard
npm install
npm install pinia vue-router axios d3 element-plus @element-plus/icons-vue
```

#### 4.2 项目结构
```
web/dashboard/
├── src/
│   ├── components/
│   │   ├── TopologyGraph.vue      # 拓扑图组件
│   │   ├── MetricsChart.vue       # 性能指标图表
│   │   ├── TraceTimeline.vue      # 追踪时间线
│   │   └── PipelineStatus.vue     # 管道状态卡片
│   ├── views/
│   │   ├── Dashboard.vue          # 主仪表板
│   │   ├── PipelineDetail.vue     # 管道详情
│   │   └── TraceExplorer.vue      # 追踪浏览器
│   ├── stores/
│   │   ├── pipeline.ts            # Pipeline 状态
│   │   ├── trace.ts               # Trace 状态
│   │   └── websocket.ts           # WebSocket 连接
│   ├── services/
│   │   ├── api.ts                 # API 客户端
│   │   └── websocket.ts           # WebSocket 客户端
│   ├── router/
│   │   └── index.ts
│   ├── App.vue
│   └── main.ts
├── package.json
└── vite.config.ts
```

#### 4.3 WebSocket 服务
**文件**: `web/dashboard/src/services/websocket.ts`

```typescript
import type { TraceEvent } from '@/types'

export class WebSocketService {
  private ws: WebSocket | null = null
  private listeners: ((event: TraceEvent) => void)[] = []

  connect(url: string) {
    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      console.log('WebSocket connected')
    }

    this.ws.onmessage = (event) => {
      const traceEvent = JSON.parse(event.data) as TraceEvent
      this.listeners.forEach(listener => listener(traceEvent))
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    this.ws.onclose = () => {
      console.log('WebSocket disconnected')
      // 重连逻辑
      setTimeout(() => this.connect(url), 3000)
    }
  }

  subscribe(listener: (event: TraceEvent) => void) {
    this.listeners.push(listener)
  }

  disconnect() {
    this.ws?.close()
  }
}
```

#### 4.4 拓扑图组件
**文件**: `web/dashboard/src/components/TopologyGraph.vue`

```vue
<template>
  <div class="topology-container">
    <svg ref="svgRef" width="100%" height="600"></svg>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import * as d3 from 'd3'
import type { Pipeline, TraceEvent } from '@/types'

const props = defineProps<{
  pipeline: Pipeline
  traces: TraceEvent[]
}>()

const svgRef = ref<SVGSVGElement | null>(null)

onMounted(() => {
  renderTopology()
})

watch(() => props.traces, () => {
  updateActiveNodes()
})

function renderTopology() {
  if (!svgRef.value) return

  const svg = d3.select(svgRef.value)
  const width = svgRef.value.clientWidth
  const height = 600

  // 清空
  svg.selectAll('*').remove()

  // 创建节点数据
  const nodes = [
    { id: 'source', type: 'source', label: props.pipeline.source.name },
    ...props.pipeline.processors.map((p, i) => ({
      id: `processor-${i}`,
      type: 'processor',
      label: p.name
    })),
    ...props.pipeline.analyzers.map((a, i) => ({
      id: `analyzer-${i}`,
      type: 'analyzer',
      label: a.name
    })),
    ...props.pipeline.sinks.map((s, i) => ({
      id: `sink-${i}`,
      type: 'sink',
      label: s.name
    }))
  ]

  // 创建连接线
  const links = []
  for (let i = 0; i < nodes.length - 1; i++) {
    links.push({ source: nodes[i].id, target: nodes[i + 1].id })
  }

  // D3 力导向图
  const simulation = d3.forceSimulation(nodes)
    .force('link', d3.forceLink(links).id(d => d.id))
    .force('charge', d3.forceManyBody().strength(-300))
    .force('center', d3.forceCenter(width / 2, height / 2))

  // 绘制连接线
  const link = svg.append('g')
    .selectAll('line')
    .data(links)
    .join('line')
    .attr('stroke', '#999')
    .attr('stroke-width', 2)

  // 绘制节点
  const node = svg.append('g')
    .selectAll('circle')
    .data(nodes)
    .join('circle')
    .attr('r', 20)
    .attr('fill', d => getNodeColor(d.type))

  // 节点标签
  const label = svg.append('g')
    .selectAll('text')
    .data(nodes)
    .join('text')
    .text(d => d.label)
    .attr('text-anchor', 'middle')
    .attr('dy', 40)

  // 更新位置
  simulation.on('tick', () => {
    link
      .attr('x1', d => d.source.x)
      .attr('y1', d => d.source.y)
      .attr('x2', d => d.target.x)
      .attr('y2', d => d.target.y)

    node
      .attr('cx', d => d.x)
      .attr('cy', d => d.y)

    label
      .attr('x', d => d.x)
      .attr('y', d => d.y)
  })
}

function getNodeColor(type: string) {
  const colors = {
    source: '#67C23A',
    processor: '#409EFF',
    analyzer: '#E6A23C',
    sink: '#F56C6C'
  }
  return colors[type] || '#909399'
}

function updateActiveNodes() {
  // 根据最新的 trace 事件高亮节点
  // 实现节点动画效果
}
</script>

<style scoped>
.topology-container {
  background: #f5f5f5;
  border-radius: 8px;
  padding: 20px;
}
</style>
```

#### 4.5 Pinia Store
**文件**: `web/dashboard/src/stores/trace.ts`

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { TraceEvent } from '@/types'
import { WebSocketService } from '@/services/websocket'

export const useTraceStore = defineStore('trace', () => {
  const traces = ref<TraceEvent[]>([])
  const wsService = new WebSocketService()

  function connect() {
    wsService.connect('ws://localhost:8080/ws/trace')
    wsService.subscribe((event: TraceEvent) => {
      traces.value.push(event)
      // 保持最近 1000 条
      if (traces.value.length > 1000) {
        traces.value.shift()
      }
    })
  }

  function disconnect() {
    wsService.disconnect()
  }

  return {
    traces,
    connect,
    disconnect
  }
})
```

### Step 5: API Endpoints

#### 5.1 Pipeline 拓扑 API
**文件**: `api/handlers/topology.go`

```go
package handlers

import (
    "github.com/gin-gonic/gin"
    "go_ProFiBus/internal/application/orchestrator"
    "net/http"
)

type TopologyHandler struct {
    orchestrator *orchestrator.Orchestrator
}

func NewTopologyHandler(orch *orchestrator.Orchestrator) *TopologyHandler {
    return &TopologyHandler{orchestrator: orch}
}

func (h *TopologyHandler) GetPipelineTopology(c *gin.Context) {
    pipelineID := c.Param("id")

    pipeline, err := h.orchestrator.GetPipeline(pipelineID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Pipeline not found"})
        return
    }

    status := pipeline.GetStatus()

    topology := map[string]interface{}{
        "pipeline_id": pipelineID,
        "name":        status.Name,
        "running":     status.Running,
        "components": map[string]interface{}{
            "source": map[string]interface{}{
                "id":     pipeline.source.GetID(),
                "name":   pipeline.source.GetName(),
                "status": status.SourceStatus,
            },
            "processors": extractProcessorInfo(pipeline.processors),
            "analyzers":  extractAnalyzerInfo(pipeline.analyzers),
            "sinks":      extractSinkInfo(pipeline.sinks),
        },
    }

    c.JSON(http.StatusOK, topology)
}
```

#### 5.2 Trace 查询 API
**文件**: `api/handlers/trace.go`

```go
func (h *TraceHandler) GetTraces(c *gin.Context) {
    filter := interfaces.TraceFilter{
        Limit: 100,
    }

    if pipelineID := c.Query("pipeline_id"); pipelineID != "" {
        filter.PipelineID = &pipelineID
    }

    traces, err := h.tracer.GetTraces(c.Request.Context(), filter)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"traces": traces})
}
```

## 测试计划

### 单元测试
- Tracer 实现测试
- WebSocket Hub 测试
- Repository 测试

### 集成测试
- Pipeline + Tracer 集成测试
- WebSocket 推送测试
- 前后端联调测试

### 性能测试
- 追踪开销测试（目标: <5% 性能影响）
- WebSocket 并发连接测试（目标: 支持 100+ 连接）
- 数据库写入性能测试

## 成功标准

- ✅ 数据流实时可视化
- ✅ 支持多管道同时监控
- ✅ 追踪开销 < 5%
- ✅ WebSocket 支持 100+ 并发连接
- ✅ 拓扑图交互流畅（60 FPS）
- ✅ 历史追踪数据可查询

## 时间估算

- Step 1: 后端 Tracer - 4 小时
- Step 2: 数据库扩展 - 2 小时
- Step 3: WebSocket 服务 - 3 小时
- Step 4: Vue 3 Dashboard - 8 小时
- Step 5: API Endpoints - 2 小时
- 测试和优化 - 3 小时

**总计**: ~22 小时

## 下一步
完成 Phase 2 后，继续 Phase 3 算法配置系统的开发。
