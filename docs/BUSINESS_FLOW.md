# go_ProFiBus 项目业务逻辑与数据流程完整说明

## 📋 目录

1. [系统架构概览](#系统架构概览)
2. [配置流程](#配置流程)
3. [数据接入与解析流程](#数据接入与解析流程)
4. [数据处理管道](#数据处理管道)
5. [算法配置与调整](#算法配置与调整)
6. [实时监控与可视化](#实时监控与可视化)
7. [完整业务流程示例](#完整业务流程示例)

---

## 🏗️ 系统架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web Dashboard (前端)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │控制面板  │  │采集通道  │  │算法配置  │  │用户管理  │       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└─────────────────────────────────────────────────────────────────┘
                            ↓ REST API + WebSocket ↓
┌─────────────────────────────────────────────────────────────────┐
│                      API Server (Go + Gin)                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │Pipeline  │  │Channel   │  │Config    │  │Auth      │       │
│  │Handler   │  │Handler   │  │Handler   │  │Handler   │       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└─────────────────────────────────────────────────────────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────────┐
│                     Application Layer (应用层)                   │
│  ┌────────────────────────────────────────────────────────┐     │
│  │              Orchestrator (管道编排器)                  │     │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐            │     │
│  │  │Pipeline 1│  │Pipeline 2│  │Pipeline N│            │     │
│  │  └──────────┘  └──────────┘  └──────────┘            │     │
│  └────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Data Processing Pipeline                      │
│                                                                  │
│  DataSource → Processor₁ → ... → Processorₙ → Analyzer → Sink  │
│                                                                  │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐    │
│  │ 采集通道  │→ │ 数据转换  │→ │ 异常检测  │→ │ 数据存储  │    │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────────┐
│                   Infrastructure Layer (基础设施层)              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐               │
│  │TimescaleDB │  │   Redis    │  │  WebSocket │               │
│  │时序数据库  │  │   缓存     │  │  实时推送  │               │
│  └────────────┘  └────────────┘  └────────────┘               │
└─────────────────────────────────────────────────────────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Physical Devices (物理设备)                   │
│  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐            │
│  │Modbus│  │RS-485│  │ CAN  │  │ I2C  │  │ UART │  ...        │
│  └──────┘  └──────┘  └──────┘  └──────┘  └──────┘            │
└─────────────────────────────────────────────────────────────────┘
```

---

## ⚙️ 配置流程

### 1. 采集通道配置（在哪里配置？）

**位置**：Web Dashboard → 采集通道管理页面 (`/channels`)

**配置步骤**：

```
Step 1: 创建采集通道
├─ 访问 http://localhost:8888/channels
├─ 点击 "新增通道"
└─ 填写通道信息：
   ├─ 通道名称：如 "温度传感器1"
   ├─ 描述：设备用途说明
   ├─ 协议类型：选择 Modbus/RS-485/CAN 等
   ├─ 设备名称：如 "温控器A"
   ├─ 设备端口：如 /dev/ttyUSB0 或 COM3
   └─ 协议配置（动态表单）：
      ├─ 波特率：115200
      ├─ 数据位：8
      ├─ 停止位：1
      ├─ 校验位：None
      └─ 从站ID：1 (Modbus)

Step 2: 配置采集点位
├─ 在通道列表中点击 "点位数量"
├─ 点击 "新增点位"
└─ 填写点位信息：
   ├─ 点位名称：如 "当前温度"
   ├─ 地址：40001 (Modbus寄存器地址)
   ├─ 数据类型：float
   ├─ 单位：℃
   ├─ 缩放系数：0.1 (原始值需要缩放)
   └─ 偏移量：-273.15 (开尔文转摄氏度)

Step 3: 启动通道
└─ 点击 "启动" 按钮开始采集
```

**后端存储**：
- 数据库表：`channels` 和 `channel_points`
- 路径：通过 REST API 保存到 PostgreSQL

---

### 2. Pipeline 配置（数据处理管道）

**位置**：代码级配置（未来可通过 Web 界面配置）

**当前配置方式**：

```go
// 方式1: 通过代码配置
pipeline := orchestrator.NewPipelineBuilder("sensor-pipeline").
    WithSource(dataSource).              // 数据源（采集通道）
    WithProcessor(tempConverter).        // 处理器1：温度转换
    WithProcessor(dataFilter).           // 处理器2：数据过滤
    WithAnalyzer(anomalyDetector).       // 分析器：异常检测
    WithSink(dbSink).                    // 输出：数据库存储
    Build()

// 方式2: 通过配置文件（config.yaml）
pipelines:
  - name: "temperature-monitoring"
    source:
      type: "channel"
      channel_id: "channel-001"
    processors:
      - type: "scale"
        params:
          scale: 0.1
          offset: -273.15
      - type: "filter"
        params:
          min: -50
          max: 150
    analyzers:
      - type: "threshold"
        rule_id: "temp-high-alert"
    sinks:
      - type: "timescaledb"
      - type: "websocket"
```

**未来扩展**：
- 通过 Web 界面拖拽配置 Pipeline
- 使用 Phase 3 的配置管理功能动态加载

---

### 3. 算法配置（规则/分析器配置）

**位置**：Web Dashboard → 算法配置页面（Phase 3 功能）

**配置 API**：

```
POST /api/v1/config/rules              # 创建异常检测规则
POST /api/v1/config/analyzers          # 创建分析器配置
POST /api/v1/config/processors         # 创建处理器配置
```

**配置示例**：

```json
// 创建温度阈值规则
POST /api/v1/config/rules
{
  "name": "temp-high-alert",
  "type": "threshold",
  "parameters": {
    "field": "temperature",
    "operator": ">",
    "threshold": 80.0,
    "severity": "warning"
  },
  "enabled": true,
  "priority": 1
}

// 创建统计分析器
POST /api/v1/config/analyzers
{
  "name": "statistical-analyzer",
  "type": "statistical",
  "parameters": {
    "window_size": 100,
    "std_threshold": 3.0
  },
  "rule_ids": ["temp-high-alert"],
  "enabled": true
}
```

---

## 📊 数据接入与解析流程

### 完整数据流

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. 物理设备采集                                                   │
└─────────────────────────────────────────────────────────────────┘
                      ↓
        Modbus RTU / RS-485 / CAN / I2C ...
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ 2. 采集通道（Channel + Collector）                                │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ Collector.Collect()                                    │     │
│  │  ├─ 连接设备端口 (/dev/ttyUSB0)                       │     │
│  │  ├─ 发送读取命令（根据协议）                          │     │
│  │  ├─ 接收原始字节流                                    │     │
│  │  ├─ 协议解析（Modbus/CAN/etc）                        │     │
│  │  └─ 生成 DataSample                                   │     │
│  └────────────────────────────────────────────────────────┘     │
│                                                                  │
│  输出：DataSample {                                              │
│    ID: "sample-001",                                            │
│    SourceID: "channel-001",                                     │
│    Timestamp: 2024-01-15T10:30:00Z,                            │
│    Data: {                                                      │
│      "raw_temperature": 3731,  // 原始值                        │
│      "raw_pressure": 1013      // 原始值                        │
│    }                                                            │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ 3. DataSource 适配器                                             │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ DataSourceAdapter.GetSample()                          │     │
│  │  └─ 包装 Collector，提供统一接口                      │     │
│  └────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ 4. Pipeline 处理 - Processor 链                                  │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ Processor 1: TemperatureConverter                      │     │
│  │  ├─ 读取点位配置（scale=0.1, offset=-273.15）         │     │
│  │  ├─ 计算：temp = 3731 * 0.1 - 273.15 = 99.95℃       │     │
│  │  └─ 更新 DataSample.Data["temperature"] = 99.95       │     │
│  └────────────────────────────────────────────────────────┘     │
│                      ↓                                           │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ Processor 2: DataFilter                                │     │
│  │  ├─ 检查范围：-50℃ < temp < 150℃                      │     │
│  │  ├─ 移除异常值                                         │     │
│  │  └─ 标记质量状态                                       │     │
│  └────────────────────────────────────────────────────────┘     │
│                      ↓                                           │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ Processor 3: MovingAverage                             │     │
│  │  ├─ 窗口大小：10                                       │     │
│  │  ├─ 计算移动平均                                       │     │
│  │  └─ 平滑数据                                           │     │
│  └────────────────────────────────────────────────────────┘     │
│                                                                  │
│  输出：DataSample {                                              │
│    ID: "sample-001",                                            │
│    Data: {                                                      │
│      "temperature": 99.95,      // 已转换                       │
│      "temperature_avg": 98.5,   // 移动平均                     │
│      "quality": "good"           // 质量标记                     │
│    }                                                            │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ 5. Pipeline 处理 - Analyzer（异常检测）                          │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ RuleEngine (规则引擎)                                  │     │
│  │                                                        │     │
│  │  规则1: ThresholdRule                                 │     │
│  │  ├─ IF temperature > 80℃                              │     │
│  │  ├─ THEN 触发告警                                     │     │
│  │  └─ 严重级别: WARNING                                 │     │
│  │                                                        │     │
│  │  规则2: StatisticalRule                               │     │
│  │  ├─ IF |temp - avg| > 3 * std                        │     │
│  │  ├─ THEN 异常检测                                     │     │
│  │  └─ 严重级别: ERROR                                   │     │
│  │                                                        │     │
│  │  执行结果：                                            │     │
│  │  ├─ 规则1触发：temp=99.95 > 80 ✓                     │     │
│  │  ├─ 生成 Event                                        │     │
│  │  └─ 发送告警                                          │     │
│  └────────────────────────────────────────────────────────┘     │
│                                                                  │
│  输出：Event {                                                   │
│    ID: "event-001",                                             │
│    Type: "THRESHOLD_EXCEEDED",                                  │
│    Severity: "WARNING",                                         │
│    Message: "温度超过阈值: 99.95℃ > 80℃",                      │
│    Timestamp: 2024-01-15T10:30:00Z,                            │
│    Metadata: {                                                  │
│      "rule_id": "temp-high-alert",                             │
│      "current_value": 99.95,                                   │
│      "threshold": 80.0                                         │
│    }                                                            │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ 6. Pipeline 处理 - Sink（数据输出）                              │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ Sink 1: TimescaleDB                                    │     │
│  │  ├─ 写入时序数据表 sensor_readings                     │     │
│  │  ├─ 写入事件表 events                                  │     │
│  │  └─ 批量写入优化                                       │     │
│  └────────────────────────────────────────────────────────┘     │
│                      ↓                                           │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ Sink 2: WebSocket                                      │     │
│  │  ├─ 推送实时数据到前端                                 │     │
│  │  ├─ 推送事件/告警                                      │     │
│  │  └─ 连接的客户端立即收到更新                           │     │
│  └────────────────────────────────────────────────────────┘     │
│                      ↓                                           │
│  ┌────────────────────────────────────────────────────────┐     │
│  │ Sink 3: Redis Cache                                    │     │
│  │  ├─ 缓存最新数据                                       │     │
│  │  ├─ 用于快速查询                                       │     │
│  │  └─ TTL: 5分钟                                         │     │
│  └────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ 7. 前端实时显示                                                   │
│                                                                  │
│  Dashboard 收到 WebSocket 消息：                                 │
│  ├─ 更新实时数据图表                                            │
│  ├─ 显示告警通知                                                │
│  ├─ 更新 Pipeline 状态                                          │
│  └─ 记录追踪事件                                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔄 数据处理管道详解

### Pipeline 生命周期

```go
// 1. 创建 Pipeline
pipeline := orchestrator.NewPipelineBuilder("main-pipeline").
    WithSource(dataSource).
    WithProcessor(processor1).
    WithProcessor(processor2).
    WithAnalyzer(analyzer).
    WithSink(sink).
    Build()

// 2. 启动 Pipeline
pipeline.Start(ctx)

// 3. 运行循环
for {
    // 从 DataSource 获取数据
    sample := dataSource.GetSample()

    // 经过 Processor 链处理
    for _, processor := range processors {
        sample = processor.Process(sample)
    }

    // Analyzer 分析
    events := analyzer.Analyze(sample)

    // 发送到 Sink
    for _, sink := range sinks {
        sink.Write(sample, events)
    }

    // 记录追踪
    tracer.RecordEvent(traceEvent)
}

// 4. 停止 Pipeline
pipeline.Stop()
```

### 关键组件说明

| 组件 | 职责 | 输入 | 输出 |
|------|------|------|------|
| **DataSource** | 数据采集 | 物理设备 | DataSample |
| **Processor** | 数据处理/转换 | DataSample | DataSample (modified) |
| **Analyzer** | 异常检测/分析 | DataSample | Event[] |
| **Sink** | 数据输出 | DataSample + Event[] | - |
| **Tracer** | 追踪记录 | 所有操作 | TraceEvent |

---

## 🎯 算法配置与调整

### 1. 运行时算法调整流程

```
┌─────────────────────────────────────────────────────────────────┐
│ Step 1: 通过 Web 界面修改算法参数                                 │
│                                                                  │
│  用户在 Dashboard 中：                                           │
│  ├─ 访问算法配置页面                                            │
│  ├─ 选择已有规则 "temp-high-alert"                             │
│  ├─ 修改阈值：80℃ → 85℃                                        │
│  └─ 点击保存                                                    │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 2: API 调用                                                 │
│                                                                  │
│  PUT /api/v1/config/rules/temp-high-alert                       │
│  {                                                              │
│    "parameters": {                                              │
│      "threshold": 85.0  // 新阈值                               │
│    }                                                            │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 3: 后端更新配置                                             │
│                                                                  │
│  ConfigHandler.UpdateRuleConfig():                              │
│  ├─ 验证新参数                                                  │
│  ├─ 保存到数据库 config_rules 表                               │
│  ├─ 记录配置历史 config_history 表                             │
│  └─ 返回成功响应                                                │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 4: 热重载配置（两种方式）                                    │
│                                                                  │
│  方式1: 定期轮询（推荐）                                         │
│  ├─ Pipeline 每30秒检查配置更新                                 │
│  ├─ 发现新配置后重新加载规则                                    │
│  └─ 不中断数据流                                                │
│                                                                  │
│  方式2: 事件通知                                                 │
│  ├─ 配置更新时发送 Redis Pub/Sub 消息                          │
│  ├─ Pipeline 订阅配置变更事件                                   │
│  └─ 立即重载配置                                                │
└─────────────────────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│ Step 5: 应用新配置                                               │
│                                                                  │
│  RuleEngine.ReloadRules():                                      │
│  ├─ 从数据库读取最新配置                                        │
│  ├─ 创建新的规则实例                                            │
│  ├─ 替换旧规则（原子操作）                                      │
│  └─ 后续数据使用新阈值 85℃                                     │
└─────────────────────────────────────────────────────────────────┘
```

### 2. 算法类型与配置

#### 阈值规则 (Threshold Rule)

```json
{
  "type": "threshold",
  "parameters": {
    "field": "temperature",
    "operator": ">",           // >, <, >=, <=, ==, !=
    "threshold": 80.0,
    "severity": "warning"
  }
}
```

#### 统计规则 (Statistical Rule)

```json
{
  "type": "statistical",
  "parameters": {
    "window_size": 100,        // 滑动窗口大小
    "std_threshold": 3.0,      // 标准差阈值
    "method": "zscore"         // zscore, iqr, mad
  }
}
```

#### 机器学习规则 (ML Rule)

```json
{
  "type": "ml",
  "parameters": {
    "model_path": "/models/anomaly_detector.pkl",
    "threshold": 0.8,
    "features": ["temp", "pressure", "humidity"]
  }
}
```

### 3. 配置版本管理

```sql
-- 配置历史表
SELECT * FROM config_history
WHERE config_id = 'temp-high-alert'
ORDER BY changed_at DESC;

-- 结果：
| version | action  | changed_by | threshold | changed_at          |
|---------|---------|------------|-----------|---------------------|
| 3       | updated | admin      | 85.0      | 2024-01-15 10:30:00 |
| 2       | updated | admin      | 80.0      | 2024-01-10 14:20:00 |
| 1       | created | admin      | 75.0      | 2024-01-01 09:00:00 |
```

**回滚配置**：
```
POST /api/v1/config/rules/temp-high-alert/rollback
{
  "version": 2  // 回滚到版本2
}
```

---

## 📡 实时监控与可视化

### 1. 追踪系统（Tracer）

每个操作都会被记录：

```
Pipeline: main-pipeline
  ├─ [10:30:00.001] DataSource.GetSample() - SUCCESS (2ms)
  ├─ [10:30:00.003] Processor.Process("scale") - SUCCESS (1ms)
  ├─ [10:30:00.004] Processor.Process("filter") - SUCCESS (1ms)
  ├─ [10:30:00.005] Analyzer.Analyze("threshold") - TRIGGERED (3ms)
  │   └─ Event: THRESHOLD_EXCEEDED
  └─ [10:30:00.008] Sink.Write("timescaledb") - SUCCESS (5ms)

Total Duration: 12ms
```

**前端显示**：
- 实时追踪时间线
- 组件性能指标
- 错误率统计

### 2. WebSocket 实时推送

```javascript
// 前端订阅
const ws = new WebSocket('ws://localhost:8080/ws/trace')

ws.onmessage = (event) => {
  const data = JSON.parse(event.data)

  if (data.type === 'trace') {
    // 更新追踪时间线
    updateTraceline(data.trace_event)
  }

  if (data.type === 'event') {
    // 显示告警通知
    showAlert(data.event)
  }

  if (data.type === 'metrics') {
    // 更新性能图表
    updateChart(data.metrics)
  }
}
```

---

## 🎬 完整业务流程示例

### 场景：温度监控系统

#### 第一步：系统配置（一次性）

```
1. 配置采集通道
   ├─ 创建 Modbus RTU 通道
   ├─ 设备端口：/dev/ttyUSB0
   ├─ 波特率：115200
   └─ 从站ID：1

2. 配置采集点位
   ├─ 点位1：温度 (地址 40001, float, 0.1倍, ℃)
   ├─ 点位2：压力 (地址 40002, float, 0.01倍, kPa)
   └─ 点位3：湿度 (地址 40003, float, 1倍, %)

3. 配置处理规则
   ├─ 规则1：温度>80℃ 告警
   ├─ 规则2：压力<90kPa 告警
   └─ 规则3：统计异常检测

4. 创建 Pipeline
   ├─ 数据源：采集通道
   ├─ 处理器：数据转换、过滤、平滑
   ├─ 分析器：规则引擎
   └─ 输出：TimescaleDB + WebSocket

5. 启动 Pipeline
   └─ 点击 "启动" 按钮
```

#### 第二步：运行时数据流

```
T+0s: 设备发送数据
  └─ Modbus RTU: [0x03, 0x01, 0x02, 0x0E, 0x8B, ...]
       (读取寄存器 40001-40003)

T+0.01s: 采集通道解析
  ├─ 解析 Modbus 响应
  ├─ 提取数据：
  │   ├─ 寄存器40001 = 3731 → 373.1℃ (异常！)
  │   ├─ 寄存器40002 = 10130 → 101.3kPa
  │   └─ 寄存器40003 = 65 → 65%
  └─ 生成 DataSample

T+0.02s: Pipeline 处理
  ├─ Processor: 数据转换 ✓
  ├─ Processor: 范围过滤 ✗ (温度超限，标记异常)
  └─ Processor: 移动平均 ✓

T+0.03s: 异常检测
  ├─ 规则1: temp > 80℃ ✓ 触发
  ├─ 生成告警事件
  └─ 严重级别：ERROR

T+0.04s: 数据存储
  ├─ TimescaleDB: 写入数据和事件
  ├─ WebSocket: 推送告警到前端
  └─ Redis: 缓存最新值

T+0.05s: 前端显示
  ├─ Dashboard 收到 WebSocket 消息
  ├─ 弹出告警通知："温度异常：373.1℃"
  ├─ 更新实时图表
  └─ 记录到事件时间线
```

#### 第三步：运维调整

```
场景：发现温度传感器需要重新校准

1. 运维人员操作
   ├─ 访问采集通道管理
   ├─ 编辑点位 "温度"
   ├─ 修改缩放系数：0.1 → 0.01 (修正)
   └─ 保存

2. 系统自动应用
   ├─ 后续数据使用新系数
   ├─ 3731 * 0.01 = 37.31℃ (正常)
   └─ 无需重启 Pipeline

3. 验证结果
   ├─ 查看实时数据图表
   ├─ 温度值恢复正常范围
   └─ 告警解除
```

---

## 🔑 关键配置文件位置

| 配置内容 | 存储位置 | 管理方式 |
|---------|---------|---------|
| **采集通道配置** | PostgreSQL `channels` 表 | Web UI `/channels` |
| **采集点位配置** | PostgreSQL `channel_points` 表 | Web UI `/channels` |
| **规则配置** | PostgreSQL `config_rules` 表 | Web UI 或 API |
| **分析器配置** | PostgreSQL `config_analyzers` 表 | Web UI 或 API |
| **Pipeline 配置** | 代码或 `config.yaml` | 代码或配置文件 |
| **系统配置** | `config.yaml` | 配置文件 |
| **数据库连接** | 环境变量或 `.env` | 环境变量 |

---

## 📚 相关文档

- [ARCHITECTURE.md](./ARCHITECTURE.md) - 架构设计
- [CHANNEL_INTEGRATION_GUIDE.md](./CHANNEL_INTEGRATION_GUIDE.md) - 通道集成指南
- [DATABASE.md](./DATABASE.md) - 数据库设计
- [DEPLOYMENT.md](./DEPLOYMENT.md) - 部署指南
- [PHASE3_IMPLEMENTATION.md](./PHASE3_IMPLEMENTATION.md) - 算法配置系统
- [PHASE4_CONTAINERIZATION.md](./PHASE4_CONTAINERIZATION.md) - 容器化部署

---

## 🎓 总结

### 配置管理总结

| 配置类型 | 在哪里配置 | 如何生效 |
|---------|-----------|---------|
| 采集通道 | Web UI `/channels` | 立即生效，启动通道后开始采集 |
| 采集点位 | Web UI `/channels` | 立即生效，影响数据解析 |
| 检测规则 | Web UI 或 API | 轮询或事件通知，热重载 |
| Pipeline | 代码配置 | 重启 Pipeline 生效 |

### 数据流转总结

```
设备 → 采集通道 → DataSample → Processor链 → Analyzer → Sink
              ↓
          点位配置             规则配置      存储+推送
          (转换/缩放)          (阈值/统计)   (DB/WS)
```

### 算法调整总结

- ✅ **在线调整**：规则参数、阈值、窗口大小
- ✅ **热重载**：无需重启，自动应用
- ✅ **版本管理**：配置历史、审计、回滚
- ✅ **实时生效**：30秒内应用到所有 Pipeline

---

**祝您使用愉快！如有疑问，请参考相关文档或联系技术支持。**
