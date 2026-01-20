# 并发处理框架

## 概述

go_ProFiBus并发框架提供了完整的并发处理能力，支持在多个层级进行并发处理，同时保证数据的先后连贯性。

## 核心组件

### 1. Worker池 (WorkerPool)

通用的worker池框架，支持任务队列和并发执行。

```go
// 创建Worker池
pool := concurrent.NewWorkerPool(concurrent.WorkerPoolConfig{
    Name:        "DataProcessor",
    WorkerCount: 4,
    QueueSize:   1000,
    ErrorHandler: func(err error) {
        log.Printf("任务失败: %v", err)
    },
})

// 启动
pool.Start()
defer pool.Stop()

// 提交任务
task := concurrent.NewSimpleTask("task_1", func() error {
    // 执行任务逻辑
    return nil
})
pool.Submit(task)
```

**特性**：
- 固定数量的worker goroutine
- 任务队列缓冲
- 错误处理回调
- 统计信息（总任务数、完成数、失败数等）

### 2. 有序Worker池 (OrderedWorkerPool)

保证输出顺序的worker池，使用序列号机制维护结果顺序。

```go
orderedPool := concurrent.NewOrderedWorkerPool(concurrent.WorkerPoolConfig{
    Name:        "OrderedProcessor",
    WorkerCount: 4,
    QueueSize:   1000,
})

orderedPool.Start()
defer orderedPool.Stop()

// 提交有序任务
orderedPool.SubmitOrdered(seqNum, task)
```

**特性**：
- 并发执行但有序输出
- 使用Sequencer管理序列号
- 自动缓冲乱序结果

### 3. 动态Worker池 (DynamicWorkerPool)

根据负载自动调整worker数量的worker池。

```go
dynamicPool := concurrent.NewDynamicWorkerPool(
    concurrent.WorkerPoolConfig{
        Name:        "DynamicProcessor",
        WorkerCount: 2, // 初始worker数
        QueueSize:   1000,
    },
    2,  // 最小worker数
    10, // 最大worker数
)

dynamicPool.Start()
// 启动自动伸缩
dynamicPool.StartAutoScale(5 * time.Second)
```

**特性**：
- 队列利用率 > 80% 时扩容
- 队列利用率 < 20% 时缩容
- 最小/最大worker数限制

### 4. 分区有序采集器 (PartitionedOrderedCollector)

按数据源分区的有序采集器，每个数据源独立维护顺序。

```go
collector := concurrent.NewPartitionedOrderedCollector(
    concurrent.PartitionedCollectorConfig{
        WindowSize:     5 * time.Second,  // 时间窗口
        MaxBufferSize:  1000,              // 每分区最大缓冲
        WorkerCount:    4,
        OutputChanSize: 1000,
    },
)

collector.Start()
defer collector.Stop()

// 提交数据样本
collector.Submit(sample)

// 从输出通道读取有序数据
for sample := range collector.GetOutput() {
    // 处理数据
}
```

**架构**：
```
数据源1 → OrderedBuffer1 ┐
数据源2 → OrderedBuffer2 ├→ TimestampMerger → 有序输出
数据源3 → OrderedBuffer3 ┘
```

**关键点**：
- **按源分区**：每个数据源独立的OrderedBuffer
- **时间窗口**：默认5秒窗口，保证窗口内数据有序
- **时间戳归并**：多源数据按时间戳归并排序
- **定期刷新**：每半个窗口刷新一次超时数据

### 5. 批量写入器 (BatchWriter)

批量累积数据并定期或达到阈值时批量写入。

```go
writer := concurrent.NewBatchWriter(concurrent.BatchWriterConfig{
    Name:          "EventWriter",
    BatchSize:     100,               // 批量大小
    FlushInterval: 5 * time.Second,   // 刷新间隔
    WorkerCount:   2,
    WriteFn: func(items []interface{}) error {
        // 批量写入逻辑
        return db.WriteBatch(items)
    },
})

writer.Start()
defer writer.Stop()

// 写入单个项目
writer.Write(item)

// 写入多个项目
writer.WriteBatch(items)

// 强制刷新
writer.Flush()
```

**特性**：
- 自动批量累积（100-1000条）
- 定时刷新（默认5秒）或达到阈值立即刷新
- 异步写入（使用worker池）
- 统计信息（总项目数、批次数、成功/失败写入数）

### 6. 多类型批量写入器 (MultiTypeBatchWriter)

支持不同类型数据分别批量写入。

```go
multiWriter := concurrent.NewMultiTypeBatchWriter()

// 注册不同类型的写入器
sensorWriter := concurrent.NewBatchWriter(sensorConfig)
eventWriter := concurrent.NewBatchWriter(eventConfig)

multiWriter.RegisterWriter("sensors", sensorWriter)
multiWriter.RegisterWriter("events", eventWriter)

// 启动所有写入器
multiWriter.StartAll()
defer multiWriter.StopAll()

// 写入到指定类型
multiWriter.Write("sensors", sensorData)
multiWriter.Write("events", eventData)

// 刷新所有
multiWriter.FlushAll()
```

## 数据顺序保证机制

### 策略1：分区有序（推荐）

**适用场景**：时序数据采集

**原理**：
- 每个数据源独立维护顺序
- 源内绝对有序，跨源按时间戳归并
- 时间窗口（默认5秒）内保证顺序

**优点**：
- 性能优秀，可线性扩展
- 容忍小范围时钟偏差

**实现**：
```go
type OrderedBuffer struct {
    sourceID   string
    buffer     []*DataSample
    windowSize time.Duration  // 5秒窗口
}

// 定期刷新超出窗口的数据
func (ob *OrderedBuffer) Flush() []*DataSample {
    cutoff := time.Now().Add(-windowSize)
    // 排序并返回超出窗口的样本
}
```

### 策略2：全局序列号

**适用场景**：事件处理

**原理**：
- 使用原子递增的全局序列号
- 所有数据共享同一序列
- Sequencer保证输出顺序

**实现**：
```go
type Sequencer struct {
    nextSeq atomic.Uint64
    buffer  map[uint64]interface{}
}

// 阻塞直到下一个序列号的结果可用
func (s *Sequencer) GetNext() (interface{}, bool) {
    // 等待 nextSeq 对应的结果
}
```

## 并发规则评估

规则引擎支持并发评估多条规则，适合规则数量较多的场景。

```go
ruleEngine := anomaly.NewRuleEngine()

// 添加多条规则
ruleEngine.AddRule(rule1)
ruleEngine.AddRule(rule2)
ruleEngine.AddRule(rule3)

// 并发评估（自动为每条规则创建goroutine）
results := ruleEngine.EvaluateConcurrent(sample)
```

**性能对比**：
- 顺序评估：O(N) 时间
- 并发评估：O(1) 时间（理论上，取决于CPU核心数）
- 推荐：规则数 > 5 时使用并发评估

**实现要点**：
```go
func (re *RuleEngine) EvaluateConcurrent(sample *DataSample) []*EvaluationResult {
    // 为每条规则启动goroutine
    for _, rule := range enabledRules {
        go func(r Rule) {
            result := r.Evaluate(sample)
            resultChan <- result
        }(rule)
    }

    // 等待所有结果
    // 收集并返回
}
```

## 使用场景

### 场景1：高吞吐量数据采集

```go
// 创建分区有序采集器
collector := concurrent.NewPartitionedOrderedCollector(config)
collector.Start()

// 多个数据源并发采集
for _, source := range sources {
    go func(s Source) {
        for data := range s.Read() {
            collector.Submit(data)
        }
    }(source)
}

// 统一处理
for sample := range collector.GetOutput() {
    process(sample)
}
```

### 场景2：批量数据持久化

```go
// 创建批量写入器
writer := concurrent.NewBatchWriter(concurrent.BatchWriterConfig{
    BatchSize:     100,
    FlushInterval: 5 * time.Second,
    WriteFn:       db.WriteBatch,
})
writer.Start()

// 数据处理流程
for sample := range dataStream {
    // 处理数据
    processed := process(sample)

    // 批量写入
    writer.Write(processed)
}
```

### 场景3：多规则异常检测

```go
ruleEngine := anomaly.NewRuleEngine()

// 添加大量规则
for _, ruleConfig := range ruleConfigs {
    ruleEngine.AddRule(createRule(ruleConfig))
}

// 实时检测
for sample := range dataStream {
    // 并发评估所有规则
    results := ruleEngine.EvaluateConcurrent(sample)

    if len(results) > 0 {
        handleAnomalies(results)
    }
}
```

## 性能调优

### 1. Worker数量

**原则**：
- CPU密集型：worker数 = CPU核心数
- IO密集型：worker数 = 2-4 × CPU核心数

**示例**：
```go
import "runtime"

workerCount := runtime.NumCPU() // CPU核心数
```

### 2. 批量大小

**推荐值**：
- 时序数据：100-1000条/批
- 事件数据：50-100条/批
- 大对象：10-50条/批

**权衡**：
- 批量越大：吞吐量越高，延迟越高
- 批量越小：延迟越低，吞吐量越低

### 3. 时间窗口

**推荐值**：
- 实时性要求高：1-2秒
- 一般场景：5秒（默认）
- 对延迟不敏感：10-30秒

### 4. 缓冲区大小

**队列大小**：
```go
QueueSize = 吞吐量(样本/秒) × 延迟容忍度(秒)
```

**示例**：
- 500样本/秒，容忍2秒延迟 → 1000

## 监控指标

### Worker池统计

```go
stats := pool.GetStats()

fmt.Printf("总任务数: %d\n", stats.TotalTasks.Load())
fmt.Printf("完成任务: %d\n", stats.CompletedTasks.Load())
fmt.Printf("失败任务: %d\n", stats.FailedTasks.Load())
fmt.Printf("活跃Worker: %d\n", stats.ActiveWorkers.Load())
fmt.Printf("排队任务: %d\n", stats.QueuedTasks.Load())
```

### 批量写入器统计

```go
stats := writer.GetStats()

fmt.Printf("总项目: %d\n", stats.TotalItems.Load())
fmt.Printf("总批次: %d\n", stats.TotalBatches.Load())
fmt.Printf("成功写入: %d\n", stats.TotalWrites.Load())
fmt.Printf("失败写入: %d\n", stats.FailedWrites.Load())
fmt.Printf("最后刷新: %v\n", stats.LastFlush)
```

### 采集器统计

```go
stats := collector.GetStats()

fmt.Printf("分区数: %d\n", stats["partition_count"])
fmt.Printf("分区缓冲: %v\n", stats["partition_sizes"])
```

## 示例程序

完整示例请参考：`examples/concurrent_collection/main.go`

```bash
cd examples/concurrent_collection
go run main.go
```

## 最佳实践

1. **合理设置worker数**：根据任务类型（CPU/IO密集）选择
2. **避免过大缓冲区**：占用内存，增加延迟
3. **监控队列长度**：队列持续满负载说明worker不足
4. **优雅关闭**：调用Stop()等待所有任务完成
5. **错误处理**：设置ErrorHandler回调处理失败任务
6. **批量写入**：优先使用批量写入减少IO次数
7. **时间窗口**：根据实际需求调整，不要过大或过小

## 故障排查

### 问题1：数据乱序

**原因**：时间窗口过小或时钟不同步

**解决**：
- 增大时间窗口（5秒 → 10秒）
- 同步各设备时钟（NTP）

### 问题2：批量写入延迟高

**原因**：批量大小过大

**解决**：
- 减小批量大小（1000 → 100）
- 减小刷新间隔（5秒 → 2秒）

### 问题3：队列满导致阻塞

**原因**：worker处理速度 < 数据产生速度

**解决**：
- 增加worker数量
- 增大队列大小
- 优化处理逻辑

### 问题4：内存占用过高

**原因**：缓冲区过大或未及时刷新

**解决**：
- 减小MaxBufferSize
- 减小时间窗口
- 增加刷新频率

## 后续优化方向

1. **优先级队列**：支持不同优先级的任务
2. **流控机制**：动态限流避免过载
3. **分布式支持**：多节点并发处理
4. **更多统计**：P50/P95/P99延迟统计
5. **可视化监控**：Grafana集成
