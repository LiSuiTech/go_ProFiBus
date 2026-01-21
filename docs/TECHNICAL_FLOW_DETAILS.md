# go_ProFiBus 技术实现细节：配置监听、数据解析与算法计算

## 📑 目录

1. [配置监听与生效机制](#配置监听与生效机制)
2. [数据样例解析流程](#数据样例解析流程)
3. [算法计算详细流程](#算法计算详细流程)
4. [实际代码示例](#实际代码示例)

---

## 🔄 配置监听与生效机制

### 1. 采集通道配置监听

#### 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                  配置生效流程                                 │
└─────────────────────────────────────────────────────────────┘

用户操作 Web UI
    ↓
PUT /api/v1/channels/{id}
    ↓
┌────────────────────────────────┐
│  ChannelHandler.UpdateChannel  │
│  ├─ 验证输入                   │
│  ├─ 更新数据库                 │
│  ├─ 发布配置变更事件           │
│  └─ 返回响应                   │
└────────────────────────────────┘
    ↓
┌────────────────────────────────┐
│  配置变更通知机制              │
│  （选择其一）                  │
│                                │
│  方式1: Redis Pub/Sub          │
│  ├─ Publish("config:channel")  │
│  └─ 订阅者立即收到通知         │
│                                │
│  方式2: 数据库轮询             │
│  ├─ 定时查询 updated_at        │
│  └─ 每30秒检查一次             │
│                                │
│  方式3: 内存事件总线           │
│  ├─ EventBus.Emit()            │
│  └─ 同进程内立即通知           │
└────────────────────────────────┘
    ↓
┌────────────────────────────────┐
│  Collector 重新加载配置        │
│  ├─ 停止当前采集任务           │
│  ├─ 从数据库读取新配置         │
│  ├─ 重新初始化连接             │
│  └─ 启动新的采集任务           │
└────────────────────────────────┘
```

#### 方式1：Redis Pub/Sub 实时通知（推荐）

```go
// ========================================
// 1. 配置更新时发布事件
// ========================================
// 文件：api/handlers/channel.go

func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
    // ... 更新数据库 ...

    // 发布配置变更事件
    h.redisClient.Publish(ctx, "config:channel:updated", channelID)

    c.JSON(200, channel)
}

// ========================================
// 2. Collector 订阅配置变更
// ========================================
// 文件：collector/collector.go

type Collector struct {
    channelID     string
    config        *ChannelConfig
    redisClient   *redis.Client
    configReloadCh chan string
}

func (c *Collector) Start() {
    // 启动配置监听
    go c.listenConfigChanges()

    // 启动数据采集
    go c.collectLoop()
}

func (c *Collector) listenConfigChanges() {
    // 订阅 Redis 频道
    pubsub := c.redisClient.Subscribe(ctx, "config:channel:updated")
    defer pubsub.Close()

    ch := pubsub.Channel()

    for msg := range ch {
        channelID := msg.Payload

        // 只处理自己的配置变更
        if channelID == c.channelID {
            log.Info("收到配置变更通知，重新加载配置...")

            // 发送重载信号
            c.configReloadCh <- channelID
        }
    }
}

func (c *Collector) collectLoop() {
    ticker := time.NewTicker(c.config.PollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // 正常采集
            c.collectData()

        case <-c.configReloadCh:
            // 收到配置重载信号
            c.reloadConfig()
            // 更新定时器间隔
            ticker.Reset(c.config.PollInterval)

        case <-c.ctx.Done():
            return
        }
    }
}

func (c *Collector) reloadConfig() {
    // 1. 从数据库读取最新配置
    newConfig, err := c.channelRepo.GetChannel(ctx, c.channelID)
    if err != nil {
        log.Error("重载配置失败: %v", err)
        return
    }

    // 2. 关闭旧连接
    if c.connection != nil {
        c.connection.Close()
    }

    // 3. 应用新配置
    c.config = newConfig

    // 4. 重新建立连接
    c.connection, err = c.openConnection(newConfig)
    if err != nil {
        log.Error("重新连接失败: %v", err)
        return
    }

    // 5. 重新加载点位配置
    c.points = newConfig.Points

    log.Info("配置重载完成，新配置已生效")
}
```

#### 方式2：定时轮询检查

```go
// 文件：collector/collector.go

type Collector struct {
    channelID      string
    config         *ChannelConfig
    lastUpdated    time.Time
    checkInterval  time.Duration
}

func (c *Collector) Start() {
    // 启动配置检查
    go c.periodicConfigCheck()

    // 启动数据采集
    go c.collectLoop()
}

func (c *Collector) periodicConfigCheck() {
    ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            c.checkConfigUpdate()

        case <-c.ctx.Done():
            return
        }
    }
}

func (c *Collector) checkConfigUpdate() {
    // 查询数据库中的更新时间
    channel, err := c.channelRepo.GetChannel(ctx, c.channelID)
    if err != nil {
        return
    }

    // 对比更新时间
    if channel.UpdatedAt.After(c.lastUpdated) {
        log.Info("检测到配置更新，重新加载...")

        // 重载配置
        c.reloadConfig()

        // 更新最后检查时间
        c.lastUpdated = channel.UpdatedAt
    }
}
```

---

## 📊 数据样例解析流程

### 1. 从原始字节到结构化数据

#### Modbus RTU 协议解析示例

```
┌─────────────────────────────────────────────────────────────┐
│  物理层：串口接收字节流                                       │
└─────────────────────────────────────────────────────────────┘
    ↓
原始字节: [0x01, 0x03, 0x04, 0x0E, 0x8B, 0x27, 0x8D, 0x3B, 0x4C]
          │     │     │     │─────┴─────│ │─────┴─────│ │───│
          │     │     │     温度寄存器    压力寄存器    CRC
          │     │     数据长度(4字节)
          │     功能码(读取保持寄存器)
          从站地址
    ↓
┌─────────────────────────────────────────────────────────────┐
│  Step 1: 协议层解析                                          │
└─────────────────────────────────────────────────────────────┘

func (c *ModbusCollector) parseModbusResponse(rawBytes []byte) (*ModbusData, error) {
    // 1. 验证 CRC 校验
    if !verifyCRC(rawBytes) {
        return nil, errors.New("CRC校验失败")
    }

    // 2. 提取数据部分
    slaveAddr := rawBytes[0]    // 0x01
    funcCode  := rawBytes[1]    // 0x03
    byteCount := rawBytes[2]    // 0x04 (4字节数据)
    data      := rawBytes[3:3+byteCount] // [0x0E, 0x8B, 0x27, 0x8D]

    // 3. 按寄存器拆分（每个寄存器2字节）
    registers := make([]uint16, byteCount/2)
    for i := 0; i < len(registers); i++ {
        // 大端序（Big-Endian）
        registers[i] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
    }

    // 结果:
    // registers[0] = 0x0E8B = 3723 (温度寄存器)
    // registers[1] = 0x278D = 10125 (压力寄存器)

    return &ModbusData{
        SlaveAddr: slaveAddr,
        Registers: registers,
    }, nil
}
```

#### Step 2: 点位配置映射

```go
// ========================================
// 数据库中的点位配置
// ========================================
// channel_points 表：
// | id | channel_id | name | address | data_type | scale | offset |
// |----|------------|------|---------|-----------|-------|--------|
// | p1 | ch-001     | 温度 | 40001   | float     | 0.1   | -273.15|
// | p2 | ch-001     | 压力 | 40002   | float     | 0.01  | 0      |

func (c *Collector) mapPointsToData(registers []uint16, points []Point) DataSample {
    data := make(map[string]interface{})

    for _, point := range points {
        // 计算寄存器索引（Modbus地址40001对应索引0）
        registerIndex := point.Address - 40001

        if registerIndex < 0 || registerIndex >= len(registers) {
            continue
        }

        // 获取原始值
        rawValue := float64(registers[registerIndex])

        // 应用缩放和偏移
        // 公式: processed_value = raw_value * scale + offset
        processedValue := rawValue * point.Scale + point.Offset

        // 存储到 data map
        data[point.Name] = processedValue
    }

    // 计算结果:
    // 温度: 3723 * 0.1 + (-273.15) = 372.3 - 273.15 = 99.15℃
    // 压力: 10125 * 0.01 + 0 = 101.25 kPa

    return DataSample{
        ID:        generateID(),
        SourceID:  c.channelID,
        Timestamp: time.Now(),
        Data:      data,
        // 结果: {
        //   "温度": 99.15,
        //   "压力": 101.25
        // }
    }
}
```

#### Step 3: 创建 DataSample 对象

```go
func (c *Collector) Collect() (DataSample, error) {
    // 1. 发送 Modbus 读取命令
    request := buildModbusRequest(c.config.SlaveID, 0x03, 40001, 2)
    rawBytes, err := c.port.ReadWrite(request)
    if err != nil {
        return DataSample{}, err
    }

    // 2. 解析协议响应
    modbusData, err := c.parseModbusResponse(rawBytes)
    if err != nil {
        return DataSample{}, err
    }

    // 3. 映射点位配置
    sample := c.mapPointsToData(modbusData.Registers, c.points)

    // 4. 添加元数据
    sample.Metadata = map[string]interface{}{
        "protocol":   "Modbus RTU",
        "slave_id":   c.config.SlaveID,
        "channel_id": c.channelID,
        "quality":    "good",
    }

    // 5. 返回结构化数据
    return sample, nil
    // 输出:
    // DataSample {
    //   ID: "sample-12345",
    //   SourceID: "ch-001",
    //   Timestamp: "2024-01-15T10:30:00Z",
    //   Data: {
    //     "温度": 99.15,
    //     "压力": 101.25
    //   },
    //   Metadata: {
    //     "protocol": "Modbus RTU",
    //     "slave_id": 1,
    //     "quality": "good"
    //   }
    // }
}
```

### 2. 完整数据采集循环

```go
func (c *Collector) Run(ctx context.Context) {
    ticker := time.NewTicker(c.config.PollInterval) // 如 100ms
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // 执行一次采集
            sample, err := c.Collect()
            if err != nil {
                log.Error("采集失败: %v", err)
                c.errorCount++
                continue
            }

            // 发送到 Pipeline
            c.outputChan <- sample

            // 更新统计
            c.successCount++
            c.lastSampleTime = time.Now()

        case <-c.configReloadCh:
            // 重载配置
            c.reloadConfig()
            ticker.Reset(c.config.PollInterval)

        case <-ctx.Done():
            return
        }
    }
}
```

---

## 🧮 算法计算详细流程

### 1. Processor：数据处理器

#### 缩放处理器示例

```go
// ========================================
// ScaleProcessor: 对数据进行缩放和偏移
// ========================================

type ScaleProcessor struct {
    name   string
    config ScaleConfig
}

type ScaleConfig struct {
    Field  string  `json:"field"`   // 要处理的字段名
    Scale  float64 `json:"scale"`   // 缩放系数
    Offset float64 `json:"offset"`  // 偏移量
}

func (p *ScaleProcessor) Process(ctx context.Context, sample DataSample) (DataSample, error) {
    // 1. 获取字段值
    value, exists := sample.Data[p.config.Field]
    if !exists {
        return sample, fmt.Errorf("字段 %s 不存在", p.config.Field)
    }

    // 2. 类型转换
    floatValue, ok := value.(float64)
    if !ok {
        return sample, fmt.Errorf("字段 %s 不是数值类型", p.config.Field)
    }

    // 3. 应用公式
    // 公式: new_value = old_value * scale + offset
    newValue := floatValue * p.config.Scale + p.config.Offset

    // 4. 更新数据
    sample.Data[p.config.Field] = newValue

    // 5. 记录处理历史
    sample.Metadata["scaled_"+p.config.Field] = true

    return sample, nil
}

// ========================================
// 使用示例
// ========================================

// 输入:
// sample.Data = {
//   "raw_temperature": 3723.0  // 原始ADC值
// }

processor := &ScaleProcessor{
    name: "temp-scale",
    config: ScaleConfig{
        Field:  "raw_temperature",
        Scale:  0.1,
        Offset: -273.15,
    },
}

output, _ := processor.Process(ctx, sample)

// 输出:
// output.Data = {
//   "raw_temperature": 99.15  // 3723 * 0.1 - 273.15 = 99.15℃
// }
```

#### 移动平均处理器

```go
// ========================================
// MovingAverageProcessor: 移动平均平滑
// ========================================

type MovingAverageProcessor struct {
    name       string
    config     MAConfig
    buffer     []float64  // 滑动窗口
    bufferSize int
}

type MAConfig struct {
    Field      string `json:"field"`
    WindowSize int    `json:"window_size"`  // 窗口大小
}

func (p *MovingAverageProcessor) Process(ctx context.Context, sample DataSample) (DataSample, error) {
    // 1. 获取当前值
    value := sample.Data[p.config.Field].(float64)

    // 2. 添加到缓冲区
    p.buffer = append(p.buffer, value)

    // 3. 保持窗口大小
    if len(p.buffer) > p.config.WindowSize {
        p.buffer = p.buffer[1:] // 移除最旧的值
    }

    // 4. 计算平均值
    sum := 0.0
    for _, v := range p.buffer {
        sum += v
    }
    average := sum / float64(len(p.buffer))

    // 5. 添加新字段
    sample.Data[p.config.Field+"_ma"] = average

    return sample, nil
}

// ========================================
// 时间序列示例
// ========================================

// T1: temperature = 100.0
//     buffer = [100.0]
//     average = 100.0

// T2: temperature = 102.0
//     buffer = [100.0, 102.0]
//     average = 101.0

// T3: temperature = 98.0
//     buffer = [100.0, 102.0, 98.0]
//     average = 100.0

// T4: temperature = 105.0
//     buffer = [100.0, 102.0, 98.0, 105.0]
//     average = 101.25

// T5: temperature = 97.0 (窗口大小=4，移除100.0)
//     buffer = [102.0, 98.0, 105.0, 97.0]
//     average = 100.5
```

### 2. Analyzer：异常检测算法

#### 阈值检测算法

```go
// ========================================
// ThresholdRule: 阈值规则检测
// ========================================

type ThresholdRule struct {
    ID        string
    Name      string
    Field     string    // 检测字段
    Operator  string    // 操作符: >, <, >=, <=, ==, !=
    Threshold float64   // 阈值
    Severity  string    // 严重级别
}

func (r *ThresholdRule) Evaluate(sample DataSample) (*Event, bool) {
    // 1. 获取字段值
    value := sample.Data[r.Field].(float64)

    // 2. 执行比较
    triggered := false
    switch r.Operator {
    case ">":
        triggered = value > r.Threshold
    case "<":
        triggered = value < r.Threshold
    case ">=":
        triggered = value >= r.Threshold
    case "<=":
        triggered = value <= r.Threshold
    case "==":
        triggered = value == r.Threshold
    case "!=":
        triggered = value != r.Threshold
    }

    // 3. 如果未触发，返回
    if !triggered {
        return nil, false
    }

    // 4. 生成事件
    event := &Event{
        ID:        generateEventID(),
        Type:      "THRESHOLD_EXCEEDED",
        Severity:  r.Severity,
        Message:   fmt.Sprintf("%s 超过阈值: %.2f %s %.2f",
            r.Field, value, r.Operator, r.Threshold),
        Timestamp: time.Now(),
        Metadata: map[string]interface{}{
            "rule_id":       r.ID,
            "rule_name":     r.Name,
            "field":         r.Field,
            "current_value": value,
            "threshold":     r.Threshold,
            "operator":      r.Operator,
        },
    }

    return event, true
}

// ========================================
// 使用示例
// ========================================

rule := &ThresholdRule{
    ID:        "rule-001",
    Name:      "高温告警",
    Field:     "temperature",
    Operator:  ">",
    Threshold: 80.0,
    Severity:  "WARNING",
}

// 输入数据:
sample := DataSample{
    Data: map[string]interface{}{
        "temperature": 99.15,
    },
}

// 执行检测:
event, triggered := rule.Evaluate(sample)

// triggered = true (因为 99.15 > 80.0)
// event = {
//   Type: "THRESHOLD_EXCEEDED",
//   Severity: "WARNING",
//   Message: "temperature 超过阈值: 99.15 > 80.00",
//   Metadata: {
//     "current_value": 99.15,
//     "threshold": 80.0
//   }
// }
```

#### 统计异常检测（Z-Score）

```go
// ========================================
// StatisticalRule: 统计异常检测
// ========================================

type StatisticalRule struct {
    ID           string
    Name         string
    Field        string
    WindowSize   int     // 滑动窗口大小
    StdThreshold float64 // 标准差倍数
    buffer       []float64
}

func (r *StatisticalRule) Evaluate(sample DataSample) (*Event, bool) {
    // 1. 获取当前值
    value := sample.Data[r.Field].(float64)

    // 2. 添加到缓冲区
    r.buffer = append(r.buffer, value)
    if len(r.buffer) > r.WindowSize {
        r.buffer = r.buffer[1:]
    }

    // 3. 需要足够的历史数据
    if len(r.buffer) < r.WindowSize {
        return nil, false
    }

    // 4. 计算统计指标
    mean := calculateMean(r.buffer)
    std := calculateStd(r.buffer, mean)

    // 5. 计算 Z-Score
    // Z-Score = (当前值 - 平均值) / 标准差
    zScore := math.Abs((value - mean) / std)

    // 6. 判断是否异常
    // 如果 Z-Score > 阈值（如3），则认为是异常
    if zScore <= r.StdThreshold {
        return nil, false
    }

    // 7. 生成异常事件
    event := &Event{
        ID:        generateEventID(),
        Type:      "STATISTICAL_ANOMALY",
        Severity:  "WARNING",
        Message:   fmt.Sprintf("%s 统计异常: 当前值 %.2f 偏离均值 %.2f 超过 %.1f 个标准差",
            r.Field, value, mean, r.StdThreshold),
        Timestamp: time.Now(),
        Metadata: map[string]interface{}{
            "rule_id":       r.ID,
            "field":         r.Field,
            "current_value": value,
            "mean":          mean,
            "std":           std,
            "z_score":       zScore,
            "threshold":     r.StdThreshold,
        },
    }

    return event, true
}

// 辅助函数
func calculateMean(values []float64) float64 {
    sum := 0.0
    for _, v := range values {
        sum += v
    }
    return sum / float64(len(values))
}

func calculateStd(values []float64, mean float64) float64 {
    variance := 0.0
    for _, v := range values {
        variance += math.Pow(v-mean, 2)
    }
    variance /= float64(len(values))
    return math.Sqrt(variance)
}

// ========================================
// 数值示例
// ========================================

// 历史数据: [98, 99, 97, 100, 98, 99, 101, 98]
// mean = 98.75
// std = 1.16

// 当前值: 105
// z_score = |105 - 98.75| / 1.16 = 5.39

// 阈值: 3
// 结果: 5.39 > 3, 触发异常！
```

### 3. Pipeline 完整执行流程

```go
// ========================================
// Pipeline.Process(): 完整处理流程
// ========================================

func (p *Pipeline) Process(sample DataSample) error {
    // 记录开始时间
    startTime := time.Now()

    // 1. 记录追踪：数据进入Pipeline
    p.tracer.RecordEvent(TraceEvent{
        PipelineID:    p.id,
        SampleID:      sample.ID,
        ComponentType: "pipeline",
        ComponentID:   p.id,
        Action:        "received",
        Timestamp:     startTime,
        Status:        "success",
    })

    // 2. Processor 链处理
    processedSample := sample
    for _, processor := range p.processors {
        procStart := time.Now()

        // 执行处理
        result, err := processor.Process(ctx, processedSample)
        if err != nil {
            // 记录错误追踪
            p.tracer.RecordEvent(TraceEvent{
                ComponentType: "processor",
                ComponentID:   processor.GetName(),
                Action:        "process",
                Status:        "error",
                Error:         err.Error(),
                Duration:      time.Since(procStart),
            })
            return err
        }

        // 记录成功追踪
        p.tracer.RecordEvent(TraceEvent{
            ComponentType: "processor",
            ComponentID:   processor.GetName(),
            Action:        "process",
            Status:        "success",
            Duration:      time.Since(procStart),
            DataSnapshot:  result.Data,
        })

        processedSample = result
    }

    // 3. Analyzer 分析
    var events []Event
    for _, analyzer := range p.analyzers {
        analyzerStart := time.Now()

        // 执行分析
        detectedEvents, err := analyzer.Analyze(ctx, processedSample)
        if err != nil {
            p.tracer.RecordEvent(TraceEvent{
                ComponentType: "analyzer",
                ComponentID:   analyzer.GetName(),
                Action:        "analyze",
                Status:        "error",
                Error:         err.Error(),
            })
            continue
        }

        // 记录追踪
        p.tracer.RecordEvent(TraceEvent{
            ComponentType: "analyzer",
            ComponentID:   analyzer.GetName(),
            Action:        "analyze",
            Status:        len(detectedEvents) > 0 ? "triggered" : "success",
            Duration:      time.Since(analyzerStart),
            Metadata: map[string]interface{}{
                "events_count": len(detectedEvents),
            },
        })

        events = append(events, detectedEvents...)
    }

    // 4. Sink 输出
    for _, sink := range p.sinks {
        sinkStart := time.Now()

        // 写入数据和事件
        err := sink.Write(ctx, processedSample, events)
        if err != nil {
            p.tracer.RecordEvent(TraceEvent{
                ComponentType: "sink",
                ComponentID:   sink.GetName(),
                Action:        "write",
                Status:        "error",
                Error:         err.Error(),
            })
            continue
        }

        // 记录追踪
        p.tracer.RecordEvent(TraceEvent{
            ComponentType: "sink",
            ComponentID:   sink.GetName(),
            Action:        "write",
            Status:        "success",
            Duration:      time.Since(sinkStart),
        })
    }

    // 5. 记录总体处理时间
    totalDuration := time.Since(startTime)
    p.tracer.RecordEvent(TraceEvent{
        PipelineID:    p.id,
        SampleID:      sample.ID,
        ComponentType: "pipeline",
        Action:        "completed",
        Status:        "success",
        Duration:      totalDuration,
    })

    // 6. 更新统计
    p.stats.SamplesProcessed++
    p.stats.TotalDuration += totalDuration
    p.stats.LastSampleTime = time.Now()

    return nil
}
```

---

## 💻 实际代码示例

### 完整示例：温度监控系统

```go
package main

import (
    "context"
    "time"
    "go_ProFiBus/collector"
    "go_ProFiBus/internal/application/orchestrator"
    "go_ProFiBus/internal/domain/rule"
    "go_ProFiBus/pkg/interfaces"
)

func main() {
    ctx := context.Background()

    // ===================================
    // 1. 配置采集通道（从数据库加载）
    // ===================================
    channel := &Channel{
        ID:         "ch-001",
        Name:       "温度传感器",
        Protocol:   "Modbus",
        DevicePort: "/dev/ttyUSB0",
        Config: ProtocolConfig{
            BaudRate:  115200,
            SlaveID:   1,
            DataBits:  8,
            StopBits:  1,
            Parity:    "None",
        },
        Points: []Point{
            {
                ID:       "p1",
                Name:     "temperature",
                Address:  "40001",
                DataType: "float",
                Scale:    0.1,
                Offset:   -273.15,
                Unit:     "℃",
            },
        },
    }

    // ===================================
    // 2. 创建 Collector
    // ===================================
    collector := collector.NewModbusCollector(channel)

    // ===================================
    // 3. 包装为 DataSource
    // ===================================
    dataSource := NewDataSourceAdapter("sensor-001", "温度传感器", collector)

    // ===================================
    // 4. 创建 Processor链
    // ===================================

    // Processor 1: 数据质量检查
    qualityCheck := &QualityCheckProcessor{
        name: "quality-check",
        config: QualityConfig{
            Field:    "temperature",
            MinValue: -50.0,
            MaxValue: 150.0,
        },
    }

    // Processor 2: 移动平均
    movingAvg := &MovingAverageProcessor{
        name: "moving-average",
        config: MAConfig{
            Field:      "temperature",
            WindowSize: 10,
        },
    }

    // ===================================
    // 5. 创建 Analyzer（从数据库加载规则）
    // ===================================
    ruleEngine := rule.NewRuleEngine()

    // 规则1：高温告警
    highTempRule := &rule.ThresholdRule{
        ID:        "rule-high-temp",
        Name:      "高温告警",
        Field:     "temperature",
        Operator:  ">",
        Threshold: 80.0,
        Severity:  "WARNING",
    }
    ruleEngine.AddRule(highTempRule)

    // 规则2：低温告警
    lowTempRule := &rule.ThresholdRule{
        ID:        "rule-low-temp",
        Name:      "低温告警",
        Field:     "temperature",
        Operator:  "<",
        Threshold: 10.0,
        Severity:  "WARNING",
    }
    ruleEngine.AddRule(lowTempRule)

    // 规则3：统计异常
    statRule := &rule.StatisticalRule{
        ID:           "rule-stat-anomaly",
        Name:         "统计异常检测",
        Field:        "temperature",
        WindowSize:   100,
        StdThreshold: 3.0,
    }
    ruleEngine.AddRule(statRule)

    analyzer := NewRuleEngineAnalyzer("rule-engine", ruleEngine)

    // ===================================
    // 6. 创建 Sink
    // ===================================

    // Sink 1: TimescaleDB
    dbSink := NewTimescaleDBSink(postgresStore)

    // Sink 2: WebSocket推送
    wsSink := NewWebSocketSink(wsHub)

    // ===================================
    // 7. 构建 Pipeline
    // ===================================
    pipeline, err := orchestrator.NewPipelineBuilder("temp-monitor").
        WithSource(dataSource).
        WithProcessor(qualityCheck).
        WithProcessor(movingAvg).
        WithAnalyzer(analyzer).
        WithSink(dbSink).
        WithSink(wsSink).
        Build()

    if err != nil {
        log.Fatalf("构建Pipeline失败: %v", err)
    }

    // ===================================
    // 8. 启动 Pipeline
    // ===================================
    if err := pipeline.Start(ctx); err != nil {
        log.Fatalf("启动Pipeline失败: %v", err)
    }
    defer pipeline.Stop()

    log.Println("Pipeline 已启动，开始监控...")

    // ===================================
    // 9. 运行循环（Pipeline内部会处理）
    // ===================================
    // 数据流：
    // 1. Collector 每100ms采集一次
    // 2. 生成 DataSample:
    //    {
    //      "temperature": 99.15,
    //      "timestamp": "2024-01-15T10:30:00Z"
    //    }
    // 3. QualityCheck: 检查范围 [-50, 150] ✓
    // 4. MovingAverage: 计算10点平均值
    // 5. RuleEngine:
    //    - HighTempRule: 99.15 > 80.0 ✓ 触发
    //    - LowTempRule: 99.15 < 10.0 ✗ 不触发
    //    - StatRule: z_score = 5.2 > 3.0 ✓ 触发
    // 6. DBSink: 写入数据库
    // 7. WSSink: 推送到前端

    // 保持运行
    select {}
}
```

---

## 🎯 总结

### 配置生效流程

```
用户修改配置 → API保存 → 发布事件 → Collector监听 → 重载配置 → 立即生效
               (数据库)   (Redis)    (订阅)     (关闭旧连接)  (下一次采集使用新配置)
```

### 数据解析流程

```
原始字节 → 协议解析 → 寄存器提取 → 点位映射 → 缩放转换 → DataSample
(串口)    (Modbus)   (0x0E8B)   (温度40001)  (*0.1-273.15) (99.15℃)
```

### 算法计算流程

```
DataSample → Processor链 → Analyzer → 事件生成 → Sink输出
           (质量检查)    (规则引擎)  (告警事件)  (数据库+推送)
           (移动平均)    (统计检测)
```

**关键点**：
- ✅ 配置热重载，无需重启
- ✅ Pipeline 流式处理，实时计算
- ✅ 完整的追踪记录，可观测
- ✅ 灵活的规则配置，易扩展