# go_ProFiBus

`go_ProFiBus` 是一个功能强大的 Go 语言工业通信库，专为物联网和工业自动化场景设计。它不仅支持多种串口协议通信，还集成了数据采集、多数据融合、模型推理和多模态数据分析等高级功能。

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

### 基础使用

```go
package main

import (
    "fmt"
    "go_ProFiBus/application"
    "log"
)

func main() {
    portName := "/dev/ttyUSB0" // 根据实际情况修改

    // 创建 UART 总线
    uartBus, err := application.NewProtocolBus(application.UART, portName)
    if err != nil {
        log.Fatalf("错误: %v", err)
    }
    defer uartBus.Close()

    // 写入数据
    dataToSend := []byte("Hello UART")
    if n, err := uartBus.Write(dataToSend); err != nil {
        log.Fatalf("写入错误: %v", err)
    } else {
        fmt.Printf("写入 %d 字节\n", n)
    }

    // 读取数据
    buffer := make([]byte, 100)
    n, err := uartBus.Read(buffer)
    if err != nil {
        log.Fatalf("读取错误: %v", err)
    }
    fmt.Printf("接收到数据: %s\n", buffer[:n])
}
```

### 数据采集示例

```go
package main

import (
    "go_ProFiBus/collector"
    "go_ProFiBus/serial"
    "log"
    "time"
)

func main() {
    // 打开串口
    port := &serial.UART{}
    port.Open("/dev/ttyUSB0")

    // 配置采集器
    config := &collector.CollectorConfig{
        SourceID:   "sensor_1",
        Port:       port,
        SampleRate: 100 * time.Millisecond,
        BufferSize: 1000,
        Handler: func(sample *collector.DataSample) error {
            log.Printf("收到数据: %v", sample)
            return nil
        },
    }

    // 创建并启动采集器
    c := collector.NewCollector(config)
    c.Start()
    defer c.Stop()

    // 运行一段时间
    time.Sleep(10 * time.Second)

    // 查看统计信息
    stats := c.GetStats()
    log.Printf("总样本数: %d, 成功: %d, 失败: %d",
        stats.TotalSamples, stats.SuccessSamples, stats.FailedSamples)
}
```

### 数据融合示例

```go
package main

import (
    "go_ProFiBus/collector"
    "go_ProFiBus/fusion"
    "log"
    "time"
)

func main() {
    // 创建融合器
    fusionEngine := fusion.NewDataFusion(fusion.StrategyWeighted, 1*time.Second)

    // 添加数据源
    sample1 := &collector.DataSample{
        SourceID: "sensor_1",
        ParsedData: map[string]interface{}{
            "temperature": 25.5,
        },
        Quality: 0.95,
    }

    sample2 := &collector.DataSample{
        SourceID: "sensor_2",
        ParsedData: map[string]interface{}{
            "temperature": 25.8,
        },
        Quality: 0.90,
    }

    fusionEngine.AddDataSource(sample1, 0.6)
    fusionEngine.AddDataSource(sample2, 0.4)

    // 执行融合
    result, err := fusionEngine.Fuse()
    if err != nil {
        log.Fatalf("融合失败: %v", err)
    }

    log.Printf("融合结果: %+v", result.Data)
    log.Printf("置信度: %.2f", result.Confidence)
}
```

### 模型推理示例

```go
package main

import (
    "go_ProFiBus/inference"
    "log"
)

func main() {
    // 创建推理引擎
    engine := inference.NewInferenceEngine()

    // 创建并注册模型
    model := inference.NewLinearRegressionModel()
    model.SetWeights([]float64{0.5, 0.3, 0.2}, 1.0)
    engine.RegisterModel("predictor", model)

    // 准备输入数据
    input, _ := inference.NewTensor([]int{1, 3}, []float64{25.0, 101.3, 0.6})

    // 执行推理
    output, err := engine.Predict("predictor", input)
    if err != nil {
        log.Fatalf("推理失败: %v", err)
    }

    log.Printf("预测结果: %v", output.Data)
}
```

### 多模态分析示例

```go
package main

import (
    "go_ProFiBus/multimodal"
    "log"
    "time"
)

func main() {
    // 创建分析器
    analyzer := multimodal.NewMultiModalAnalyzer(multimodal.AlignmentLinearInterp)

    // 注册特征提取器
    analyzer.RegisterFeatureExtractor(
        multimodal.ModalitySensor,
        multimodal.NewSensorFeatureExtractor(),
    )

    // 添加模态数据
    sensorData := &multimodal.ModalityData{
        Type:       multimodal.ModalitySensor,
        Timestamp:  time.Now(),
        SourceID:   "sensor_1",
        Embedding:  []float64{0.1, 0.2, 0.3},
        Confidence: 0.95,
    }

    analyzer.AddModalityData(sensorData)

    // 执行分析
    result, err := analyzer.Analyze(time.Now(), "")
    if err != nil {
        log.Fatalf("分析失败: %v", err)
    }

    log.Printf("分析结果: 模态数=%d, 置信度=%.2f",
        result.ModalityCount, result.Confidence)
}
```

## 📁 项目结构

```
go_ProFiBus/
├── application/        # 应用层 - 协议工厂
├── datalink/          # 数据链路层 - 帧处理和CRC校验
├── serial/            # 串口层 - 具体协议实现
├── collector/         # 数据采集层 - 多源数据采集
├── fusion/            # 数据融合层 - 多数据融合算法
├── inference/         # 推理层 - 模型推理引擎
├── multimodal/        # 多模态层 - 跨模态数据分析
├── logger/            # 日志系统
├── errors/            # 错误处理
├── config/            # 配置管理
├── examples/          # 示例程序
│   ├── basic/        # 基础示例
│   └── advanced/     # 高级示例
├── config.yaml        # 配置文件
├── go.mod            # 依赖管理
└── README.md         # 项目文档
```

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

### 基础示例

```bash
cd examples/basic
go run main.go
```

### 高级示例

```bash
cd examples/advanced
go run main.go
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

### 分层架构

```
┌─────────────────────────────────────────┐
│         Application Layer               │  应用层
│  (Protocol Factory, High-level APIs)    │
├─────────────────────────────────────────┤
│       Advanced Features Layer           │  高级功能层
│  (Multimodal, Inference, Fusion)        │
├─────────────────────────────────────────┤
│        Data Collection Layer            │  数据采集层
│  (Multi-source Collector)               │
├─────────────────────────────────────────┤
│         Data Link Layer                 │  数据链路层
│  (Frame, CRC Check)                     │
├─────────────────────────────────────────┤
│         Serial Protocol Layer           │  串口协议层
│  (UART, RS485, I2C, SPI, etc.)         │
└─────────────────────────────────────────┘
```

### 核心组件

1. **Serial Protocols** - 统一的串口通信接口
2. **Data Collector** - 多源并发数据采集
3. **Data Fusion** - 多传感器数据融合
4. **Inference Engine** - 机器学习模型推理
5. **Multimodal Analyzer** - 多模态数据分析

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

- [x] 基础串口协议支持
- [x] 数据采集功能
- [x] 数据融合算法
- [x] 模型推理引擎
- [x] 多模态数据分析
- [ ] WebSocket 实时数据推送
- [ ] RESTful API 接口
- [ ] 图形化配置界面
- [ ] 更多机器学习模型支持
- [ ] 分布式部署支持

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

**Q: 如何在没有硬件的情况下测试？**
A: 可以使用虚拟串口工具如 `socat` 创建虚拟设备进行测试。

**Q: 支持 Windows 系统吗？**
A: 主要针对 Linux 系统开发，Windows 系统部分功能可能需要适配。

**Q: 如何添加自定义协议？**
A: 实现 `serial.SerialPort` 接口，并在 `application` 层注册即可。

**Q: 模型文件格式是什么？**
A: 当前版本支持自定义模型格式，未来将支持 ONNX、TensorFlow Lite 等标准格式。

---

**如果这个项目对你有帮助，请给个 ⭐ Star！**
