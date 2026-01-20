# go_ProFiBus 架构文档

## 项目概述

go_ProFiBus 是一个工业现场总线数据采集、处理和分析系统，支持多种协议（RS485, I2C, SPI, Modbus等）的数据采集和实时异常检测。

**当前版本**: Phase 1 重构完成，Phase 2 进行中

## 架构演进

### 历史架构（Legacy - 已弃用）

项目最初采用简单的分层架构，将功能按类型分包：

```
根目录/
├── collector/     # 数据采集
├── anomaly/       # 异常检测
├── event/         # 事件管理
├── fusion/        # 数据融合
├── inference/     # ML推理
└── storage/       # 数据存储
```

**问题**：
- 包之间紧耦合，难以测试
- 缺乏接口抽象，扩展困难
- 混乱的依赖关系
- 重复的数据模型定义

### 当前架构（DDD + Clean Architecture）

Phase 1 重构引入了领域驱动设计（DDD）和整洁架构：

```
go_ProFiBus/
├── pkg/interfaces/              # 公共接口层（可被外部使用）
│   ├── datasource.go            # 数据源抽象
│   ├── processor.go             # 处理器抽象
│   ├── analyzer.go              # 分析器抽象
│   ├── repository.go            # 仓储抽象
│   ├── plugin.go                # 插件系统
│   └── tracer.go                # 追踪器（Phase 2）
│
├── internal/                    # 内部实现（不对外暴露）
│   ├── domain/                  # 领域层 - 业务实体
│   │   ├── datasample/          # 数据样本实体
│   │   ├── event/               # 事件实体
│   │   └── rule/                # 规则实体
│   │
│   ├── application/             # 应用层 - 业务逻辑编排
│   │   ├── orchestrator/        # 管道编排器
│   │   │   ├── pipeline.go      # 数据处理管道
│   │   │   ├── orchestrator.go  # 多管道管理
│   │   │   └── builder.go       # 流式构建器
│   │   ├── processor/
│   │   │   └── chain.go         # 处理器链
│   │   └── tracing/             # 追踪服务（Phase 2）
│   │       └── tracer.go
│   │
│   ├── infrastructure/          # 基础设施层 - 技术实现
│   │   ├── collector/
│   │   │   └── datasource_adapter.go  # 数据源适配器
│   │   ├── storage/
│   │   │   ├── repository_impl.go     # 仓储实现
│   │   │   └── trace_repository.go    # 追踪仓储
│   │   └── analyzer/
│   │       └── rule_engine_adapter.go # 规则引擎适配器
│   │
│   └── interfaces/              # 接口层 - 外部交互
│       └── websocket/           # WebSocket服务（Phase 2）
│           ├── hub.go
│           ├── client.go
│           └── handler.go
│
├── [Legacy Packages - Deprecated] # 旧代码，保留用于向后兼容
│   ├── collector/               # ⚠️ 已弃用，使用 pkg/interfaces
│   ├── anomaly/                 # ⚠️ 已弃用，使用 internal/domain/rule
│   ├── event/                   # ⚠️ 已弃用，使用 internal/domain/event
│   ├── fusion/                  # ⚠️ 已弃用，使用 orchestrator
│   ├── inference/               # ⚠️ 已弃用，使用 plugin系统
│   └── storage/                 # ⚠️ 部分已迁移到 internal/infrastructure/storage
│
└── [Infrastructure & Tools]     # 基础设施和工具
    ├── serial/                  # 串口协议实现
    ├── logger/                  # 日志系统
    ├── errors/                  # 错误处理
    ├── config/                  # 配置管理
    ├── concurrent/              # 并发工具
    ├── api/                     # REST API服务
    ├── examples/                # 使用示例
    ├── migrations/              # 数据库迁移
    └── docs/                    # 文档
```

## 核心设计原则

### 1. 依赖倒置（Dependency Inversion）

高层模块（application）不依赖低层模块（infrastructure），都依赖抽象（interfaces）：

```go
// ✅ 正确：依赖接口
type Pipeline struct {
    source    interfaces.DataSource    // 接口
    processors []interfaces.Processor  // 接口
    analyzers []interfaces.Analyzer    // 接口
}

// ❌ 错误：依赖具体实现
type OldPipeline struct {
    collector *collector.Collector  // 具体实现
    engine    *anomaly.RuleEngine   // 具体实现
}
```

### 2. 接口隔离（Interface Segregation）

细粒度接口，按职责分离：

```go
type DataSource interface {
    Start(ctx context.Context) error
    Stop() error
    GetData() <-chan DataSample
    GetStatus() SourceStatus
}

type Processor interface {
    Process(ctx context.Context, input DataSample) (DataSample, error)
    GetName() string
}

type Analyzer interface {
    Analyze(ctx context.Context, data DataSample) ([]AnalysisResult, error)
    GetType() AnalyzerType
}
```

### 3. 单一职责（Single Responsibility）

每个模块职责明确：

- **Domain**: 仅包含业务实体和业务规则
- **Application**: 编排业务流程，不包含技术细节
- **Infrastructure**: 处理技术实现（数据库、网络、文件等）

### 4. 开闭原则（Open-Closed）

对扩展开放，对修改关闭：

```go
// 添加新功能：实现接口即可，无需修改核心代码
type MyCustomProcessor struct{}

func (p *MyCustomProcessor) Process(ctx context.Context, input interfaces.DataSample) (interfaces.DataSample, error) {
    // 自定义处理逻辑
    return input, nil
}

// 集成到管道
pipeline := orchestrator.NewPipelineBuilder("my-pipeline").
    WithProcessor(MyCustomProcessor{}).  // 直接使用
    Build()
```

## 分层详解

### Interface Layer（接口层）- pkg/interfaces/

**职责**: 定义系统的抽象契约

**核心接口**:
- `DataSource`: 数据源抽象，支持多种协议
- `DataSample`: 数据样本接口
- `Processor`: 数据处理器
- `Analyzer`: 异常检测分析器
- `Repository`: 数据存储仓储
- `Plugin`: 插件系统，支持动态加载

**特点**:
- ✅ 可被外部包导入使用
- ✅ 只包含接口定义，无具体实现
- ✅ 稳定，很少改变

### Domain Layer（领域层）- internal/domain/

**职责**: 业务实体和业务规则

**核心实体**:
- `DataSample`: 数据样本实体（实现 interfaces.DataSample）
- `Event`: 事件实体（异常、告警等）
- `Rule`: 规则实体（阈值、统计等）

**特点**:
- ✅ 不依赖其他层
- ✅ 包含业务逻辑
- ✅ 使用私有字段 + Getter/Setter封装

### Application Layer（应用层）- internal/application/

**职责**: 业务流程编排和协调

**核心组件**:
- `Pipeline`: 数据处理管道（Source → Processors → Analyzers → Sinks）
- `Orchestrator`: 多管道管理器
- `PipelineBuilder`: 流式API构建器
- `Tracer`: 数据流追踪器（Phase 2）

**特点**:
- ✅ 依赖接口，不依赖具体实现
- ✅ 编排领域对象完成业务流程
- ✅ 无技术细节

### Infrastructure Layer（基础设施层）- internal/infrastructure/

**职责**: 技术实现和适配

**核心适配器**:
- `DataSourceAdapter`: 将 collector.Collector 适配为 interfaces.DataSource
- `RuleEngineAnalyzer`: 将 anomaly.RuleEngine 适配为 interfaces.Analyzer
- `TimeSeriesRepository`: TimescaleDB 时序数据仓储实现
- `TraceRepository`: 追踪数据仓储实现

**特点**:
- ✅ 实现接口层定义的接口
- ✅ 包含技术细节（数据库、网络、文件等）
- ✅ 可替换（如切换数据库）

### Interface Layer（对外接口层）- internal/interfaces/

**职责**: 系统对外交互

**核心组件**:
- `WebSocket Hub`: 实时数据推送
- `REST API Handlers`: HTTP接口

**特点**:
- ✅ 处理外部请求
- ✅ 数据格式转换
- ✅ 认证授权

## 数据流示例

### 完整的数据处理流程

```
1. 数据采集
   Serial Port → Collector → DataSourceAdapter → DataSample

2. 数据处理管道
   DataSample
     → Processor1 (数据转换)
     → Processor2 (数据清洗)
     → Analyzer (异常检测)
     → Event (如果检测到异常)
     → Sink (存储到数据库)

3. 追踪和可视化（Phase 2）
   每个步骤 → Tracer.TraceDataFlow()
              → WebSocket Hub
              → Vue 3 Dashboard (实时显示)
```

### 代码示例

```go
// 1. 创建数据源（使用适配器）
collector := collector.NewCollector(serialConfig)
dataSource := infracollector.NewDataSourceAdapter("sensor-001", "温度传感器", collector)

// 2. 创建处理器
tempConverter := NewTemperatureProcessor()
dataFilter := NewFilterProcessor()

// 3. 创建分析器
ruleEngine := anomaly.NewRuleEngine()
ruleEngine.AddRule(tempRule)
analyzer := analyzer.NewRuleEngineAnalyzer("rule-engine", ruleEngine)

// 4. 创建输出
repo := storage.NewTimeSeriesRepository(pgStore)
sink := NewRepositorySink(repo)

// 5. 构建管道
pipeline, _ := orchestrator.NewPipelineBuilder("main-pipeline").
    WithSource(dataSource).
    WithProcessor(tempConverter).
    WithProcessor(dataFilter).
    WithAnalyzer(analyzer).
    WithSink(sink).
    Build()

// 6. 启动
orch := orchestrator.NewOrchestrator()
orch.AddPipeline(pipeline)
orch.StartAll()
```

## 迁移指南

### 从旧代码迁移到新架构

#### 使用数据采集

**旧代码**:
```go
import "go_ProFiBus/collector"

c := collector.NewCollector(config)
c.Start()
sample := <-c.GetDataChannel()
```

**新代码**:
```go
import (
    "go_ProFiBus/pkg/interfaces"
    infracollector "go_ProFiBus/internal/infrastructure/collector"
)

collector := collector.NewCollector(config)
dataSource := infracollector.NewDataSourceAdapter("source-1", "My Source", collector)
dataSource.Start(ctx)
sample := <-dataSource.GetData()
```

#### 异常检测

**旧代码**:
```go
import "go_ProFiBus/anomaly"

engine := anomaly.NewRuleEngine()
engine.AddRule(rule)
evals := engine.Evaluate(sample)
```

**新代码**:
```go
import (
    "go_ProFiBus/pkg/interfaces"
    "go_ProFiBus/internal/infrastructure/analyzer"
)

engine := anomaly.NewRuleEngine()
analyzer := analyzer.NewRuleEngineAnalyzer("analyzer-1", engine)
results, _ := analyzer.Analyze(ctx, sample)
```

#### 使用管道（推荐方式）

**新代码（最佳实践）**:
```go
import (
    "go_ProFiBus/internal/application/orchestrator"
)

pipeline := orchestrator.NewPipelineBuilder("pipeline-1").
    WithSource(dataSource).
    WithAnalyzer(analyzer).
    WithSink(sink).
    Build()

pipeline.Start(ctx)
```

## 技术栈

### 后端
- **语言**: Go 1.22+
- **数据库**: PostgreSQL 14+ with TimescaleDB extension
- **驱动**: pgx/v5（高性能PostgreSQL驱动）
- **Web框架**: Gin（REST API）
- **WebSocket**: Gorilla WebSocket

### 前端（Phase 2）
- **框架**: Vue 3 + Composition API
- **状态管理**: Pinia
- **构建工具**: Vite
- **可视化**: D3.js / Cytoscape.js
- **图表**: ECharts
- **UI组件**: Element Plus

### 工具
- **依赖管理**: Go Modules
- **迁移**: SQL文件 + TimescaleDB
- **日志**: 自定义logger包
- **错误处理**: 自定义errors包

## 性能优化

### 1. 批量写入
使用 `pgx.CopyFrom` 批量写入时序数据，性能提升 10-100倍：

```go
func (r *TimeSeriesRepositoryImpl) WriteSamples(samples []DataSample) error {
    copySource := pgx.CopyFromRows(rows)
    return r.store.CopyFrom("timeseries_data", columnNames, copySource)
}
```

### 2. 并发处理
- Pipeline 使用 goroutine 并发处理多个数据样本
- RuleEngine 支持并发评估规则：`EvaluateConcurrent()`

### 3. 通道缓冲
数据通道默认缓冲 1000 个样本，减少阻塞。

### 4. 追踪开销控制（Phase 2）
追踪系统设计目标：
- 追踪开销 < 5% CPU
- 异步写入数据库
- 批量保存追踪事件

## 测试策略

### 单元测试
- Domain层：测试业务逻辑
- Application层：使用Mock测试管道逻辑
- Infrastructure层：测试适配器

### 集成测试
- Pipeline完整流程测试
- WebSocket推送测试
- 数据库读写测试

### 当前覆盖率
⚠️ **需要改进**: 当前测试覆盖率仅 2.8%（2/71文件）

**优先级**:
1. 核心包测试（collector, anomaly, event）
2. 管道测试（orchestrator）
3. 适配器测试（infrastructure）

## 部署架构

### Docker容器化（Phase 4）

```
┌──────────────────────────────────────┐
│          Load Balancer / Nginx       │
└──────────────────────────────────────┘
           │                    │
   ┌───────▼────────┐  ┌────────▼──────┐
   │  API Server 1  │  │  API Server 2 │
   │  (REST+WS)     │  │  (REST+WS)    │
   └───────┬────────┘  └────────┬──────┘
           │                    │
   ┌───────▼──────────────────────────┐
   │      PostgreSQL + TimescaleDB    │
   │      (时序数据 + 事件存储)         │
   └──────────────────────────────────┘
```

## 未来规划

### Phase 2: 数据流可视化（进行中）
- [x] Tracer接口定义
- [x] WebSocket Hub实现
- [x] 追踪数据库表
- [ ] Vue 3 Dashboard
- [ ] 拓扑图可视化
- [ ] 实时性能指标

### Phase 3: 算法配置系统
- [ ] 基于 ConfigSchema 的表单生成
- [ ] 拖拽式工作流编辑器
- [ ] Plugin Registry 动态加载
- [ ] 算法市场

### Phase 4: 容器化和部署
- [ ] Docker多阶段构建
- [ ] docker-compose编排
- [ ] Kubernetes部署配置
- [ ] CI/CD流水线

## 常见问题（FAQ）

### Q: 为什么保留旧代码包？
A: 旧包通过适配器模式桥接到新架构，保证向后兼容。未来版本将逐步移除。

### Q: 应该使用哪个Event包？
A: 新代码使用 `internal/domain/event`，旧代码使用 `event` 包（已标记为Deprecated）。

### Q: 如何扩展新的数据源？
A: 实现 `pkg/interfaces.DataSource` 接口即可，无需修改核心代码。

### Q: 如何添加自定义处理器？
A: 实现 `pkg/interfaces.Processor` 接口，然后通过 PipelineBuilder 添加到管道。

### Q: 性能监控和追踪会影响性能吗？
A: 追踪系统设计开销 < 5%，且可选（不启用 Tracer 则无开销）。

## 贡献指南

### 添加新功能
1. 在 `pkg/interfaces/` 定义接口
2. 在 `internal/domain/` 实现领域实体
3. 在 `internal/application/` 编排业务逻辑
4. 在 `internal/infrastructure/` 实现技术细节
5. 添加测试
6. 更新文档

### 代码风格
- 遵循 Go 官方风格指南
- 接口命名：动词或能力（如 DataSource, Processor）
- 实现命名：接口名 + Impl（如 DataSourceAdapter）
- 包注释：说明包的职责和用法
- 导出函数：必须有文档注释

## 参考资料

- [Phase 1 重构总结](./PHASE1_SUMMARY.md)
- [Phase 2 实施计划](./PHASE2_PLAN.md)
- [数据库设计](./docs/DATABASE.md)
- [Go Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Domain-Driven Design](https://domainlanguage.com/ddd/)

---

**最后更新**: 2026-01-20
**文档版本**: 1.0
**项目阶段**: Phase 1 完成，Phase 2 进行中
