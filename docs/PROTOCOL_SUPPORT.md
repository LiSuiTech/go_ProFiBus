# 多协议数据采集支持

go_ProFiBus 支持多种工业协议和IoT协议的数据采集，包括 MQTT、OPC-UA、Profibus、串口等。

## 支持的协议

### 1. MQTT (消息队列遥测传输)

MQTT是一种轻量级的发布/订阅消息传输协议，广泛应用于IoT场景。

**使用场景：**
- IoT设备数据采集
- 传感器网络
- 实时消息订阅
- 事件驱动数据采集

**配置示例：**
```go
config := &collector.MQTTSourceConfig{
    SourceID:     "mqtt-sensor-network",
    Name:         "工厂传感器MQTT网络",
    Broker:       "tcp://localhost:1883",
    Topics:       []string{"factory/sensors/#"},
    QoS:          1,
    Username:     "user",
    Password:     "pass",
    UseTLS:       true,
    BufferSize:   1000,
}

source, _ := collector.NewMQTTSource(config)
source.Start(ctx)
```

**支持的功能：**
- ✅ QoS 0, 1, 2
- ✅ 主题通配符 (#, +)
- ✅ TLS/SSL加密
- ✅ 用户名密码认证
- ✅ 自动重连
- ✅ 清除会话/持久会话

### 2. OPC-UA (开放平台通信统一架构)

OPC-UA是工业自动化领域的标准通信协议，用于工业控制系统之间的数据交换。

**使用场景：**
- 工业自动化系统
- SCADA数据采集
- PLC数据读取
- 工厂设备监控

**配置示例：**
```go
config := &collector.OPCUASourceConfig{
    SourceID:         "opcua-plc",
    Name:             "工厂PLC系统",
    Endpoint:         "opc.tcp://localhost:4840",
    SecurityPolicy:   "Basic256Sha256",
    SecurityMode:     "SignAndEncrypt",
    Username:         "admin",
    Password:         "admin",
    NodeIDs:          []string{
        "ns=2;s=Temperature.Zone1",
        "ns=2;s=Pressure.Tank1",
    },
    SamplingInterval: 1 * time.Second,
    UseSubscription:  true,
}

source, _ := collector.NewOPCUASource(config)
source.Start(ctx)
```

**支持的功能：**
- ✅ 订阅模式（数据变更推送）
- ✅ 轮询模式（定时读取）
- ✅ 安全策略（None, Basic128Rsa15, Basic256, Basic256Sha256）
- ✅ 安全模式（None, Sign, SignAndEncrypt）
- ✅ 用户名密码认证
- ✅ 证书认证
- ✅ 多节点批量读取

### 3. Profibus（现场总线）

**即将支持...**

### 4. 串口 (Serial Port)

支持RS232/RS485串口通信。

**使用场景：**
- 传统工业设备
- 串口传感器
- PLC串口通信

## 数据流架构

```
[MQTT设备] [OPC-UA设备] [串口设备]
     ↓           ↓           ↓
  [数据源适配器] 统一为 DataSample接口
     ↓           ↓           ↓
      [融合处理器] ← 多源数据融合
            ↓
    [特征提取器] ← 提取ML特征
            ↓
      [ML分析器] ← AI异常检测
            ↓
      [结果输出]
```

## 多协议融合示例

可以同时使用多种协议采集数据，并进行融合处理：

```go
// 1. 创建MQTT数据源
mqttSource, _ := collector.NewMQTTSource(mqttConfig)
mqttSource.Start(ctx)

// 2. 创建OPC-UA数据源
opcuaSource, _ := collector.NewOPCUASource(opcuaConfig)
opcuaSource.Start(ctx)

// 3. 创建融合处理器
fusionProcessor := processor.NewFusionProcessor("multi-protocol", fusion.StrategyWeighted)
fusionProcessor.SetSourceWeight("mqtt-sensor", 0.5)
fusionProcessor.SetSourceWeight("opcua-sensor", 0.5)

// 4. 处理数据
for {
    select {
    case sample := <-mqttSource.GetData():
        fusedSample, _ := fusionProcessor.Process(ctx, sample)
        // 处理融合后的数据...

    case sample := <-opcuaSource.GetData():
        fusedSample, _ := fusionProcessor.Process(ctx, sample)
        // 处理融合后的数据...
    }
}
```

## 完整示例

查看以下示例代码：

- **MQTT采集示例**: `examples/mqtt_collection/main.go`
- **OPC-UA采集示例**: `examples/opcua_collection/main.go`
- **多协议融合示例**: `examples/multi_protocol_fusion/main.go`

## 依赖安装

```bash
# MQTT客户端库
go get github.com/eclipse/paho.mqtt.golang

# OPC-UA客户端库
go get github.com/gopcua/opcua
```

## 测试服务器

### MQTT测试服务器

使用 Mosquitto MQTT broker：
```bash
# 安装Mosquitto
sudo apt-get install mosquitto mosquitto-clients

# 启动服务器
mosquitto -p 1883

# 发布测试消息
mosquitto_pub -t "factory/sensors/temp" -m '{"value": 85.5}'

# 订阅测试
mosquitto_sub -t "factory/sensors/#"
```

### OPC-UA测试服务器

推荐使用以下开源OPC-UA服务器：

1. **Prosys OPC UA Simulation Server**
   - 下载: https://www.prosysopc.com/products/opc-ua-simulation-server/
   - 免费版支持基本功能
   - 提供丰富的模拟节点

2. **open62541**
   - GitHub: https://github.com/open62541/open62541
   - 开源OPC-UA实现
   - 支持Linux/Windows/macOS

## 性能优化

### MQTT优化

1. **合理设置QoS等级**
   - QoS 0: 最多一次（高性能，可能丢失）
   - QoS 1: 至少一次（推荐）
   - QoS 2: 恰好一次（高可靠，低性能）

2. **调整缓冲区大小**
   ```go
   config.BufferSize = 10000 // 高频数据
   ```

3. **使用主题通配符**
   ```go
   Topics: []string{"factory/sensors/#"} // 订阅所有传感器
   ```

### OPC-UA优化

1. **选择合适的采集模式**
   - 订阅模式：数据变更时推送（推荐）
   - 轮询模式：定时读取（兼容性好）

2. **批量读取节点**
   ```go
   NodeIDs: []string{"ns=2;s=Temp1", "ns=2;s=Temp2", ...}
   ```

3. **调整采样间隔**
   ```go
   SamplingInterval: 100 * time.Millisecond // 高频采样
   ```

## 故障排查

### MQTT连接失败

1. 检查broker地址和端口
2. 检查网络连接
3. 验证用户名密码
4. 检查TLS证书

### OPC-UA连接失败

1. 检查服务器端点
2. 验证安全策略和模式
3. 检查证书配置
4. 确认节点ID格式正确

## 扩展开发

要添加新的协议支持，实现以下接口：

```go
type DataSource interface {
    Start(ctx context.Context) error
    Stop() error
    GetData() <-chan DataSample
    GetStatus() SourceStatus
    GetID() string
    GetName() string
}

type DataSourceConfig interface {
    Validate() error
    GetType() string
}
```

参考 `mqtt_source.go` 和 `opcua_source.go` 的实现。
