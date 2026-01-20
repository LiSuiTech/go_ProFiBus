# go_ProFiBus

`go_ProFiBus` 是一个基于 DDD（领域驱动设计）和整洁架构的工业现场总线数据采集、处理和分析系统，专为物联网和工业自动化场景设计。

**当前版本**: Phase 1 重构完成，Phase 2 进行中

## 🏗️ 架构亮点

- ✅ **DDD 分层架构** - 清晰的领域层、应用层、基础设施层分离
- ✅ **接口驱动** - 面向接口编程，易于扩展和测试
- ✅ **管道模式** - 灵活的数据处理管道（Source → Processors → Analyzers → Sinks）
- ✅ **实时追踪** - 数据流可视化和性能监控（Phase 2）
- ✅ **插件系统** - 支持动态加载算法和处理器

**详细架构文档**: [ARCHITECTURE.md](./ARCHITECTURE.md)
**Phase 1 总结**: [PHASE1_SUMMARY.md](./PHASE1_SUMMARY.md)
**Phase 2 计划**: [PHASE2_PLAN.md](./PHASE2_PLAN.md)

## ✨ 核心特性

### 🔌 多协议支持
支持 9 种主流工业通信协议：
- **UART** - 通用异步收发器
- **CAN** - 控制器局域网络
- **USB** - 通用串行总线
- **1-Wire** - 单线通信协议
- **Modbus** - 工业通信协议
- **RS-232** - 串行通信标准
- **RS-485** - 差分串行接口
- **I2C** - 互连集成电路
- **SPI** - 串行外设接口

### 📊 数据采集
- 多源并发数据采集
- 可配置采样率和缓冲区
- 自动重试机制
- 数据质量评估
- 实时统计信息

### 🔀 数据融合
支持多种融合策略：
- 平均融合
- 加权融合
- 卡尔曼滤波
- 移动平均
- 指数移动平均

时序数据处理：
- 时间同步
- 线性插值
- 异常值检测

### 🧠 模型推理
- 线性回归模型
- 神经网络模型
- 自定义模型支持
- 批量推理
- 数据预处理管道

### 🎭 多模态融合分析
支持多种模态数据：
- 时序数据
- 传感器数据
- 图像数据
- 音频数据
- 文本数据
- 视频数据
- 事件数据

特性：
- 多模态数据对齐
- 特征提取
- 跨模态融合
- 流式分析

### 🛠️ 其他特性
- 统一的接口设计
- YAML 配置文件支持
- 完善的错误处理
- 结构化日志系统
- 线程安全设计

## 📦 安装

确保你已经安装了 Go 1.22 或更高版本。

```bash
git clone https://github.com/YouEvanLi/go_ProFiBus.git
cd go_ProFiBus
go mod tidy
```

## 🚀 快速开始

### 使用管道处理数据（推荐方式）

```go
package main

import (
    "context"
    "go_ProFiBus/collector"
    "go_ProFiBus/internal/application/orchestrator"
    infracollector "go_ProFiBus/internal/infrastructure/collector"
    infraanalyzer "go_ProFiBus/internal/infrastructure/analyzer"
    "go_ProFiBus/anomaly"
    "log"
)

func main() {
    ctx := context.Background()

    // 1. 创建数据源（使用适配器包装现有 Collector）
    collectorInstance := collector.NewCollector(config)
    dataSource := infracollector.NewDataSourceAdapter(
        "sensor-001",
        "温度传感器",
        collectorInstance,
    )

    // 2. 创建分析器（异常检测）
    ruleEngine := anomaly.NewRuleEngine()
    tempRule := anomaly.NewThresholdRule(
        "temp_high",
        "温度过高",
        "temperature",
        ">",
        80.0,
        anomaly.SeverityWarning,
    )
    ruleEngine.AddRule(tempRule)
    analyzer := infraanalyzer.NewRuleEngineAnalyzer("rule-engine", ruleEngine)

    // 3. 创建输出（可选：添加数据库存储等）
    sink := NewConsoleSink() // 自定义实现

    // 4. 构建管道
    pipeline, err := orchestrator.NewPipelineBuilder("main-pipeline").
        WithSource(dataSource).
        WithAnalyzer(analyzer).
        WithSink(sink).
        Build()

    if err != nil {
        log.Fatalf("构建管道失败: %v", err)
    }

    // 5. 启动管道
    if err := pipeline.Start(ctx); err != nil {
        log.Fatalf("启动管道失败: %v", err)
    }
    defer pipeline.Stop()

    log.Println("管道运行中...")
    select {} // 保持运行
}
```

### 实现自定义处理器

```go
package main

import (
    "context"
    "go_ProFiBus/pkg/interfaces"
)

// 自定义温度转换处理器
type TemperatureConverter struct {
    name string
}

func NewTemperatureConverter() *TemperatureConverter {
    return &TemperatureConverter{name: "temp-converter"}
}

// 实现 Processor 接口
func (p *TemperatureConverter) Process(ctx context.Context, input interfaces.DataSample) (interfaces.DataSample, error) {
    data := input.GetData()

    // 将华氏度转换为摄氏度
    if tempF, ok := data["temperature_f"].(float64); ok {
        tempC := (tempF - 32) * 5 / 9
        data["temperature_c"] = tempC
    }

    return input, nil
}

func (p *TemperatureConverter) GetName() string {
    return p.name
}

func (p *TemperatureConverter) GetConfig() interfaces.ProcessorConfig {
    return nil
}

func (p *TemperatureConverter) Initialize(config interfaces.ProcessorConfig) error {
    return nil
}

func (p *TemperatureConverter) Close() error {
    return nil
}

// 在管道中使用
func main() {
    pipeline := orchestrator.NewPipelineBuilder("my-pipeline").
        WithSource(dataSource).
        WithProcessor(NewTemperatureConverter()). // 添加自定义处理器
        WithAnalyzer(analyzer).
        Build()
}
```

### 使用多管道编排器

```go
package main

import (
    "context"
    "go_ProFiBus/internal/application/orchestrator"
    "log"
)

func main() {
    ctx := context.Background()

    // 创建编排器
    orch := orchestrator.NewOrchestrator()

    // 添加多个管道
    pipeline1, _ := orchestrator.NewPipelineBuilder("sensor-pipeline-1").
        WithSource(sensor1Source).
        WithAnalyzer(analyzer1).
        Build()

    pipeline2, _ := orchestrator.NewPipelineBuilder("sensor-pipeline-2").
        WithSource(sensor2Source).
        WithAnalyzer(analyzer2).
        Build()

    orch.AddPipeline(pipeline1)
    orch.AddPipeline(pipeline2)

    // 启动所有管道
    if err := orch.StartAll(ctx); err != nil {
        log.Fatalf("启动失败: %v", err)
    }
    defer orch.StopAll()

    // 监控错误
    go func() {
        for err := range orch.MonitorErrors() {
            log.Printf("管道错误: %v", err)
        }
    }()

    // 查看状态
    status := orch.GetStatus()
    log.Printf("运行中的管道: %d", status.RunningCount)
}
```

### 数据持久化到 TimescaleDB

```go
package main

import (
    "context"
    "go_ProFiBus/storage"
    infraStorage "go_ProFiBus/internal/infrastructure/storage"
    "go_ProFiBus/pkg/interfaces"
    "log"
)

func main() {
    ctx := context.Background()

    // 创建 PostgreSQL 连接
    pgStore, err := storage.NewPostgresStore(
        "localhost", 5432, "profibus_db", "user", "password",
    )
    if err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    defer pgStore.Close()

    // 创建时序数据仓储
    repo := infraStorage.NewTimeSeriesRepository(pgStore)

    // 批量写入数据（高性能）
    samples := []interfaces.DataSample{
        dataSample1,
        dataSample2,
        dataSample3,
    }

    if err := repo.WriteSamples(ctx, samples); err != nil {
        log.Fatalf("写入失败: %v", err)
    }

    // 查询时间范围数据
    results, _ := repo.QueryByTimeRange(
        ctx,
        "sensor-001",
        startTime,
        endTime,
    )

    log.Printf("查询到 %d 条记录", len(results))
}
```


## 📁 项目结构

```
go_ProFiBus/
├── pkg/interfaces/              # 公共接口层（推荐使用）
│   ├── datasource.go            # 数据源抽象
│   ├── processor.go             # 处理器抽象
│   ├── analyzer.go              # 分析器抽象
│   ├── repository.go            # 仓储抽象
│   ├── plugin.go                # 插件系统
│   └── tracer.go                # 追踪器
│
├── internal/                    # 内部实现（新架构）
│   ├── domain/                  # 领域层 - 业务实体
│   ├── application/             # 应用层 - 业务逻辑编排
│   │   └── orchestrator/        # 管道编排器 ⭐
│   ├── infrastructure/          # 基础设施层 - 适配器
│   └── interfaces/              # 外部接口 - WebSocket等
│
├── [Legacy - 兼容旧代码]         # ⚠️ 已弃用，仅用于向后兼容
│   ├── collector/               # → 使用 pkg/interfaces
│   ├── anomaly/                 # → 使用 internal/domain/rule
│   ├── event/                   # → 使用 internal/domain/event
│   ├── fusion/                  # → 使用 orchestrator
│   └── inference/               # → 使用 plugin 系统
│
├── [基础设施和工具]
│   ├── serial/                  # 串口协议实现
│   ├── logger/                  # 日志系统
│   ├── errors/                  # 错误处理
│   ├── config/                  # 配置管理
│   ├── storage/                 # 数据存储
│   ├── api/                     # REST API
│   └── examples/                # 使用示例
│
├── ARCHITECTURE.md              # 架构文档 ⭐
├── PHASE1_SUMMARY.md            # Phase 1 总结
├── PHASE2_PLAN.md               # Phase 2 计划
└── README.md                    # 项目文档
```

**⚠️ 重要提示**: 根目录下的 `collector/`, `anomaly/`, `event/`, `fusion/`, `inference/` 等包已被标记为 **Deprecated**。新代码请使用 `internal/` 和 `pkg/interfaces/` 下的新架构。详见 [ARCHITECTURE.md](./ARCHITECTURE.md)。

## ⚙️ 配置文件

项目支持 YAML 格式的配置文件。示例配置 `config.yaml`:

```yaml
system:
  name: "go_ProFiBus"
  version: "1.0.0"
  environment: "dev"
  debug: true

logging:
  level: "INFO"
  enable_file: true
  file_path: "logs/profibus.log"

protocols:
  - id: "rs485_main"
    type: "RS485"
    port: "/dev/ttyUSB0"
    baud_rate: 115200
    enabled: true

collector:
  buffer_size: 1000
  default_sample_rate: "100ms"
  enable_cache: true

fusion:
  strategy: "weighted"
  time_window: "1s"

multimodal:
  alignment: "linear_interp"
  analyze_interval: "1s"
```

## 🧪 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./logger
go test ./errors

# 运行测试并查看覆盖率
go test -cover ./...
```

## 📚 示例程序

### Pipeline 示例（推荐）

```bash
cd examples/pipeline
go run main.go
```

### 并发采集示例

```bash
cd examples/concurrent_collection
go run main.go
```

### WebSocket 追踪示例（Phase 2）

```bash
cd examples/rest_api
go run websocket_trace_example.go
```

## 🔧 配置选项

### 串口配置

可以通过以下选项配置串口参数：

```go
import "go_ProFiBus/serial"

// 设置波特率
serial.WithBaudRate(115200)

// 设置数据位
serial.WithDataBits(8)

// 设置校验位
serial.WithParity(serial.ParityNone)  // None/Odd/Even

// 设置停止位
serial.WithStopBits(1)

// 设置 I2C 地址
serial.WithAddress(0x48)
```

### 日志配置

```go
import "go_ProFiBus/logger"

// 设置日志级别
logger.SetLevel(logger.INFO)

// 启用文件日志
log := logger.GetLogger()
log.EnableFileLog("app.log")
```

## 🏗️ 架构设计

### DDD 分层架构

```
┌─────────────────────────────────────────┐
│      Interface Layer (pkg/interfaces)   │  接口定义层
│    (DataSource, Processor, Analyzer)    │
├─────────────────────────────────────────┤
│      Application Layer (internal/app)   │  应用层
│    (Pipeline, Orchestrator, Builder)    │  业务逻辑编排
├─────────────────────────────────────────┤
│      Domain Layer (internal/domain)     │  领域层
│    (DataSample, Event, Rule)            │  业务实体
├─────────────────────────────────────────┤
│  Infrastructure Layer (internal/infra)  │  基础设施层
│  (Adapters, Repository, WebSocket)      │  技术实现
├─────────────────────────────────────────┤
│      [Legacy Packages - Deprecated]     │  旧代码（向后兼容）
│    (collector, anomaly, event, etc.)    │
└─────────────────────────────────────────┘
```

### 数据流管道

```
Serial Port
    ↓
DataSource (Adapter)
    ↓
Pipeline [
    Processor 1 (数据转换)
    →
    Processor 2 (数据过滤)
    →
    Analyzer (异常检测)
    →
    Sink (存储)
]
    ↓
TimescaleDB / Event Store
    ↓
WebSocket (实时推送)
    ↓
Vue 3 Dashboard
```

### 核心组件

1. **Pipeline** - 数据处理管道编排器 ⭐
2. **Orchestrator** - 多管道管理器
3. **Adapters** - 新旧代码桥接适配器
4. **Repository** - 数据仓储抽象
5. **Tracer** - 数据流追踪和可视化（Phase 2）

详细架构说明请参考：[ARCHITECTURE.md](./ARCHITECTURE.md)

## 🤝 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

该项目使用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [sigurn/crc16](https://github.com/sigurn/crc16) - CRC16 校验库
- [golang.org/x/sys](https://golang.org/x/sys) - 系统调用支持
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析库

## 📧 联系方式

- 作者: lixiaolong
- 项目链接: [https://github.com/YouEvanLi/go_ProFiBus](https://github.com/YouEvanLi/go_ProFiBus)

## 🗺️ 路线图

### Phase 1: 核心架构重构 ✅ 已完成
- [x] DDD 分层架构设计
- [x] 接口抽象层（pkg/interfaces）
- [x] Pipeline 数据处理框架
- [x] 适配器模式桥接新旧代码
- [x] TimescaleDB 时序数据存储

### Phase 2: 数据流可视化 🚧 进行中
- [x] Tracer 接口和实现
- [x] WebSocket 实时推送
- [x] 追踪数据库设计
- [ ] Vue 3 可视化 Dashboard
- [ ] 拓扑图和性能指标展示

### Phase 3: 算法配置系统 📋 计划中
- [ ] 基于 ConfigSchema 的表单生成
- [ ] 拖拽式工作流编辑器
- [ ] Plugin Registry 动态加载
- [ ] 算法市场

### Phase 4: 容器化部署 📋 计划中
- [ ] Docker 多阶段构建
- [ ] docker-compose 编排
- [ ] Kubernetes 部署配置
- [ ] CI/CD 流水线

## 📊 性能指标

- 支持并发采集多达 100+ 数据源
- 单通道采样率可达 10kHz
- 数据融合延迟 < 10ms
- 模型推理时间 < 5ms (CPU)

## ⚠️ 注意事项

1. 某些协议需要 root 权限或特定的设备权限
2. 在没有实际硬件的情况下，部分功能可能无法完全测试
3. 建议在生产环境使用前进行充分测试
4. 部分高级功能需要配置相应的模型文件

## 🔍 常见问题

**Q: 新旧架构如何选择？**
A: 新代码必须使用 `pkg/interfaces` 和 `internal/` 下的新架构。旧包（collector, anomaly, event等）已标记为 Deprecated，仅用于向后兼容。

**Q: 如何添加自定义数据源？**
A: 实现 `pkg/interfaces.DataSource` 接口，然后通过 PipelineBuilder 集成到管道中。参考 `internal/infrastructure/collector/datasource_adapter.go`。

**Q: 如何添加自定义处理器？**
A: 实现 `pkg/interfaces.Processor` 接口，无需修改核心代码即可使用。

**Q: 为什么要重构架构？**
A: 旧架构存在紧耦合、难以测试、扩展困难等问题。新架构基于 DDD 和整洁架构，更易维护和扩展。详见 [ARCHITECTURE.md](./ARCHITECTURE.md)。

**Q: Phase 2 的可视化功能何时完成？**
A: 后端追踪功能已完成，Vue 3 Dashboard 正在开发中。预计 Phase 2 整体完成时间约 22 小时工作量。

---

**如果这个项目对你有帮助，请给个 ⭐ Star！**

## 📖 更多文档

- **[架构文档](./ARCHITECTURE.md)** - 详细的架构设计和迁移指南 ⭐
- **[Phase 1 总结](./PHASE1_SUMMARY.md)** - Phase 1 重构成果和技术细节
- **[Phase 2 计划](./PHASE2_PLAN.md)** - Phase 2 数据流可视化实施计划
- **[数据库设计](./docs/DATABASE.md)** - TimescaleDB 时序数据库设计

