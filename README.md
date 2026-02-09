# go_ProFiBus

`go_ProFiBus` 是一个基于 DDD（领域驱动设计）和整洁架构的工业现场总线数据采集、处理和分析系统，专为物联网和工业自动化场景设计。

**当前版本**: Phase 4 完成 - 生产就绪 🚀

[![Docker](https://img.shields.io/badge/Docker-Ready-blue)](./Dockerfile)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-blue)](./deployments/kubernetes/)
[![CI/CD](https://img.shields.io/badge/CI%2FCD-GitHub_Actions-green)](/.github/workflows/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

## 🏗️ 架构亮点

- ✅ **DDD 分层 + 整洁架构**：`domain / application / infrastructure / interfaces` 清晰分层
- ✅ **工业设备一站式平台**：设备管理、告警中心、预测分析、数据管理、反向控制、工作流统一在一个系统中
- ✅ **Workflow 引擎（DAG）**：类似 Dify 的可视化工作流，支持设备数据源、告警输出、设备控制等节点
- ✅ **通用数据融合**：支持单设备多维度、单设备单点、多设备多源的灵活数据融合配置
- ✅ **ML 模型插件系统**：统一的模型加载 / 推理接口，支持多种模型类型与 JSON 模型文件
- ✅ **模型训练接口**：后端提供训练任务管理与模拟训练流程，便于后续接入真实训练框架
- ✅ **通用规则引擎**：规则模板库 + 规则测试（Dry Run），支持快速创建和验证告警 / 控制规则
- ✅ **数据管理中心**：数据清洗、归档策略、生命周期管理一体化
- ✅ **实时数据流 & 大屏**：WebSocket 推送设备数据，前端多维度实时曲线和统计
- ✅ **容器化 + 监控 + CI/CD**：Docker / Kubernetes 部署，Prometheus + Grafana 监控，GitHub Actions 流水线


## ✨ 核心功能模块

### 1. 设备与通道管理
- 设备管理：设备信息、状态、健康度、位置（区域 / 产线 / 车间）管理
- 设备通道：设备与采集通道映射，支持多种协议（UART / CAN / USB / Modbus / RS-232 / RS-485 / I2C / SPI / 1-Wire）
- 设备数据字段 & 数据源：为每个设备配置结构化字段和数据来源
- Web 管理界面：设备列表、详情、状态、健康度可视化

### 2. 设备布局可视化
- 通过 Vue + SVG / D3.js 展示设备在厂区 / 产线上的拓扑布局
- 支持根据设备状态上色、显示运行 / 故障 / 维护等状态
- 可作为实时监控大屏的基础组件

### 3. 告警中心 & 通用规则引擎
- 告警管理：告警规则、告警记录、统计视图（按级别 / 状态）
- 告警规则：基于 JSON 条件的规则定义，支持冷却时间、最大执行次数等
- **规则模板库**：
  - 阈值规则、异常检测、趋势分析、复合条件、变化率等模板
  - 可视化变量配置（字段名、阈值、方法等）
  - 一键从模板创建告警规则
- **规则测试（Dry Run）**：
  - 提交样例数据，实时评估规则是否触发
  - 返回触发状态、Z 分数、趋势值等诊断信息

### 4. 预测分析 & ML 模型
- 预测结果管理：预测任务、结果历史、可视化图表
- 模型管理：创建 / 更新 / 删除 / 部署预测模型，上传模型文件
- **ML 模型插件系统**（后端）：
  - 支持线性回归、神经网络、SVM、决策树、LSTM 等多类型模型
  - 使用 JSON 文件描述模型参数与结构，统一加载与推理流程
- **模型训练接口**：
  - 训练任务：任务状态、进度、历史记录
  - 后端目前提供「模拟训练」流程，方便日后接入实际 ML 平台

### 5. 通用数据融合系统
- 支持多种数据来源：
  - 单设备多维度（如：温度 / 振动 / 转速）
  - 单设备单点
  - 多设备、多通道
- 可配置融合策略与权重，结果可在 Workflow、告警 / 预测等模块中复用
- 与实时数据流打通，可在前端直接看到融合后曲线

### 6. 数据管理中心
- **数据清洗**：
  - 去重、异常值过滤、缺失值填充、标准化、平滑处理、数据验证
  - 基于规则配置的通用清洗引擎
  - 前端提供「清洗预览（Dry Run）」视图：单条样例数据清洗前后对比
- **归档策略**：
  - 按数据源类型 / ID 配置归档策略
  - 保留天数 / 归档天数 / 压缩开关 / 执行间隔
  - 归档记录与统计视图
- **生命周期管理**：
  - 热 / 温 / 冷存储天数配置
  - 删除 / 压缩阈值
  - 判断某条数据应该处于哪种存储层级

### 7. 设备反向控制
- 控制策略：基于条件和动作定义设备控制逻辑
- 控制权限 / 审核 / 审计：谁在什么时候对哪台设备做了什么操作
- 与 Workflow、告警、预测模块联动，可实现「检测 → 决策 → 控制」闭环

### 8. Workflow 工作流引擎
- DAG 工作流：节点 + 边 + 变量的图结构
- 核心节点：
  - 设备数据源（Device Source）
  - 规则检测 / ML 分析
  - 条件分支 / 变量设置
  - 告警输出、设备控制
- 多输入 / 多输出参数映射：前端可视化配置边上的 `param_mapping`
- 工作流模板库：预置常用监控 / 控制 / 数据处理流程，一键生成工作流

### 9. 实时数据流 & Web Dashboard
- WebSocket 通道：`/ws/data` 实时推送设备 / 融合数据
- 前端「实时数据流」页面：
  - 设备 / 字段过滤
  - 多条曲线实时刷新
  - 质量指标 / 数据量统计
- Vue 3 + Element Plus + ECharts 构建的管理控制台

## 📦 安装

### 方式 1: Docker Compose（推荐）

```bash
git clone https://github.com/YouEvanLi/go_ProFiBus.git
cd go_ProFiBus
cp .env.example .env  # 配置环境变量
docker-compose up -d
```

### 方式 2: Kubernetes

```bash
# 部署到 Kubernetes 集群
kubectl apply -f deployments/kubernetes/

# 查看部署状态
kubectl get pods -n profibus
```

### 方式 3: 本地开发

确保你已经安装了 Go 1.22 或更高版本。

```bash
git clone https://github.com/YouEvanLi/go_ProFiBus.git
cd go_ProFiBus
go mod tidy
```

## 🚀 快速开始

### 使用 Docker Compose（最快方式）

```bash
# 1. 启动所有服务
docker-compose up -d

# 2. 查看服务状态
docker-compose ps

# 3. 查看应用日志
docker-compose logs -f profibus

# 4. 访问服务
# Dashboard: http://localhost:8888 (Vue 3 管理平台) ⭐
# API: http://localhost:8080
# Metrics: http://localhost:8081/metrics
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000 (admin/admin)

# 5. 停止服务
docker-compose down
```

### 使用 Web Dashboard 管理系统

访问 Dashboard：http://localhost:8888

#### 1. 配置采集通道

```
导航到 "采集通道" 页面
  ↓
点击 "新增通道"
  ↓
填写配置：
  - 通道名称：温度传感器1
  - 协议类型：Modbus
  - 设备端口：/dev/ttyUSB0
  - 波特率：115200
  - 从站ID：1
  ↓
保存并启动通道
```

#### 2. 配置采集点位

```
点击通道的 "点位数量"
  ↓
点击 "新增点位"
  ↓
填写配置：
  - 点位名称：当前温度
  - 地址：40001 (Modbus 寄存器)
  - 数据类型：float
  - 缩放系数：0.1
  - 偏移量：-273.15
  - 单位：℃
  ↓
保存配置
```

#### 3. 查看实时数据

```
返回 "控制面板"
  ↓
查看 Pipeline 列表和状态
  ↓
点击 Pipeline 查看详情：
  - 拓扑图：可视化数据流
  - 性能指标：吞吐量、耗时、成功率
  - 实时追踪：数据处理过程
```

#### 4. 配置算法规则

```
访问算法配置页面（如已实现）
  ↓
创建检测规则：
  - 规则名称：高温告警
  - 类型：阈值规则
  - 字段：temperature
  - 条件：> 80℃
  - 严重级别：WARNING
  ↓
保存并应用规则
```

### 使用 Kubernetes

```bash
# 1. 部署应用
kubectl apply -f deployments/kubernetes/

# 2. 查看 Pod 状态
kubectl get pods -n profibus

# 3. 查看服务
kubectl get svc -n profibus

# 4. 端口转发访问服务
kubectl port-forward -n profibus svc/profibus-service 8080:8080

# 5. 查看日志
kubectl logs -n profibus -l app=profibus -f

# 6. 水平扩容
kubectl scale deployment profibus -n profibus --replicas=5
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

### Phase 2: 数据流可视化 ✅ 已完成
- [x] Tracer 接口和实现
- [x] WebSocket 实时推送
- [x] 追踪数据库设计
- [x] Vue 3 可视化 Dashboard
- [x] REST API endpoints
- [x] 高级可视化组件
  - D3.js 拓扑图
  - ECharts 性能图表
  - 实时追踪时间线

### Phase 3: 算法配置系统 ✅ 已完成
- [x] 基于 ConfigSchema 的表单生成
- [x] 拖拽式工作流编辑器
- [x] Plugin Registry 动态加载
- [x] 算法配置 REST API
- [x] RBAC 权限管理系统
  - 用户管理
  - 角色管理
  - 权限控制
  - JWT 认证
- [x] 配置版本管理和审计
- [x] 配置热重载机制

### Phase 4: 容器化部署 ✅ 已完成
- [x] Docker 多阶段构建（镜像优化至 40MB）
- [x] docker-compose 编排（PostgreSQL + Redis + 监控）
- [x] Kubernetes 部署配置（生产级配置）
  - StatefulSet（数据库）
  - Deployment（应用）
  - HPA（自动扩缩容）
  - Ingress（外部访问）
- [x] CI/CD 流水线（GitHub Actions）
  - 自动化测试
  - 安全扫描（Trivy + gosec）
  - 多平台构建（amd64 + arm64）
  - 自动部署（Staging + Production）
- [x] Prometheus + Grafana 监控
- [x] Vue 3 Dashboard 容器化
- [x] 采集通道管理功能（Web UI + Backend）
  - 9 种工业协议支持
  - 点位配置和映射
  - 配置热重载
  - 实时状态监控

## 🎨 Web Dashboard 功能模块

### 当前支持的功能页面

| 模块 | 页面路径 | 功能描述 |
|------|---------|---------|
| **控制面板** | `/` | Pipeline 列表、实时追踪、统计信息 |
| **Pipeline 详情** | `/pipeline/:id` | 拓扑可视化、性能指标、组件状态 |
| **采集通道管理** | `/channels` | 设备配置、协议选择、点位管理 ⭐ |

### 核心可视化组件

- **TopologyGraph** - Pipeline 拓扑图（D3.js）
- **TraceTimeline** - 追踪事件时间线
- **MetricsChart** - 性能指标图表（ECharts）
- **ThroughputChart** - 吞吐量趋势图

### 实时功能

- ✅ WebSocket 实时数据推送
- ✅ 实时告警通知
- ✅ 动态图表更新
- ✅ Pipeline 状态监控

### 技术栈

- **前端框架**: Vue 3 + TypeScript
- **UI 组件**: Element Plus
- **可视化**: ECharts + D3.js
- **状态管理**: Pinia
- **路由**: Vue Router
- **HTTP**: Axios

## 📊 性能指标

- 支持并发采集多达 100+ 数据源
- 单通道采样率可达 10kHz
- 数据融合延迟 < 10ms
- 模型推理时间 < 5ms (CPU)
- Docker 镜像大小：40MB（优化 95%）
- 启动时间：< 5 秒

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

**Q: 如何部署到生产环境？**
A: 推荐使用 Kubernetes 部署。详细步骤请参考 [DEPLOYMENT.md](./docs/DEPLOYMENT.md)。支持 Docker Compose 用于开发和小规模部署。

**Q: 如何监控应用性能？**
A: 系统集成了 Prometheus + Grafana。应用在 8081 端口暴露 `/metrics` endpoint，Prometheus 自动采集指标，Grafana 提供可视化面板。

**Q: 如何配置采集通道？**
A: 访问 Dashboard 的"采集通道"页面，可以通过 Web 界面配置设备、选择协议、设置点位。支持 9 种工业通信协议，配置后可实时生效。详见 [采集通道集成指南](./docs/CHANNEL_INTEGRATION_GUIDE.md)。

**Q: 配置修改后如何生效？**
A: 系统支持配置热重载。通过 Web UI 修改配置后，会通过 Redis Pub/Sub 或定时轮询通知采集器，采集器自动重载配置，无需重启服务。详见 [技术实现细节](./docs/TECHNICAL_FLOW_DETAILS.md)。

**Q: 如何查看系统业务流程？**
A: 请参考 [业务流程说明](./docs/BUSINESS_FLOW.md)，包含完整的配置流程、数据接入与解析、算法计算等详细说明。

---

**如果这个项目对你有帮助，请给个 ⭐ Star！**

## 📖 更多文档

### 🎯 核心文档
- **[架构文档](./ARCHITECTURE.md)** - 详细的架构设计和迁移指南 ⭐
- **[业务流程说明](./docs/BUSINESS_FLOW.md)** - 完整业务逻辑和数据流程 📊
- **[技术实现细节](./docs/TECHNICAL_FLOW_DETAILS.md)** - 配置监听、数据解析、算法计算 🔧
- **[部署指南](./docs/DEPLOYMENT.md)** - 完整的部署和运维指南 🚀
- **[数据库设计](./docs/DATABASE.md)** - TimescaleDB 时序数据库设计

### 📋 Phase 实施文档
- **[Phase 1 总结](./PHASE1_SUMMARY.md)** - Phase 1 重构成果和技术细节
- **[Phase 2 计划](./PHASE2_PLAN.md)** - Phase 2 数据流可视化实施计划
- **[Phase 3 实施](./docs/PHASE3_IMPLEMENTATION.md)** - 算法配置系统和 RBAC 实施详情
- **[Phase 4 容器化](./docs/PHASE4_CONTAINERIZATION.md)** - 容器化和部署完整实施方案

### 🔌 功能集成文档
- **[采集通道集成指南](./docs/CHANNEL_INTEGRATION_GUIDE.md)** - 采集通道管理功能集成步骤 ⭐

### 📚 推荐阅读顺序

**新手入门**：
1. README.md（本文档）
2. ARCHITECTURE.md（了解架构）
3. 快速开始（Docker Compose 部署）
4. BUSINESS_FLOW.md（理解业务流程）

**开发者**：
1. ARCHITECTURE.md（架构设计）
2. TECHNICAL_FLOW_DETAILS.md（技术实现）
3. CHANNEL_INTEGRATION_GUIDE.md（功能集成）
4. Phase 实施文档（历史演进）

**运维人员**：
1. DEPLOYMENT.md（部署指南）
2. BUSINESS_FLOW.md（系统配置）
3. DATABASE.md（数据库管理）

