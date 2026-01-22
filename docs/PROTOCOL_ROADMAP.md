# go_ProFiBus 协议支持路线图

## 概述

go_ProFiBus致力于成为全栈工业通信平台，支持从现场层到IT层的所有主流协议。

---

## 协议分类

### 🟢 现场总线协议 (Fieldbus)

| 协议 | 状态 | 优先级 | 说明 |
|------|------|--------|------|
| **PROFIBUS DP/PA** | ✅ **已实现** | ⭐⭐⭐⭐⭐ | 项目核心协议，完整支持Master/Slave、循环数据交换 |
| **Modbus RTU** | ✅ **已实现** | ⭐⭐⭐⭐⭐ | 支持所有功能码、CRC校验 |
| **HART** | 🔜 计划中 | ⭐⭐⭐⭐ | 过程自动化协议 |
| **DeviceNet** | 🔜 计划中 | ⭐⭐⭐ | 基于CAN的现场总线 |
| **AS-i** | 🔜 计划中 | ⭐⭐ | 简单传感器/执行器网络 |
| **Foundation Fieldbus** | 🔜 计划中 | ⭐⭐ | 过程自动化协议 |

### 🟢 工业以太网协议 (Industrial Ethernet)

| 协议 | 状态 | 优先级 | 说明 |
|------|------|--------|------|
| **PROFINET IO** | 🔜 计划中 | ⭐⭐⭐⭐⭐ | PROFIBUS的以太网版本 |
| **EtherCAT** | 🔜 计划中 | ⭐⭐⭐⭐⭐ | 实时以太网，高性能 |
| **EtherNet/IP** | 🔜 计划中 | ⭐⭐⭐⭐ | 罗克韦尔/AB系统 |
| **Modbus TCP** | ✅ **已实现** | ⭐⭐⭐⭐⭐ | Modbus的TCP版本 |
| **POWERLINK** | 🔜 计划中 | ⭐⭐⭐ | B&R的实时以太网 |
| **SERCOS III** | 🔜 计划中 | ⭐⭐ | 运动控制总线 |

### 🟢 IT层协议 (IT Layer)

| 协议 | 状态 | 优先级 | 说明 |
|------|------|--------|------|
| **OPC UA** | ✅ **已实现** | ⭐⭐⭐⭐⭐ | 支持Client模式，Pub/Sub待实现 |
| **MQTT** | ✅ **已实现** | ⭐⭐⭐⭐⭐ | IoT标准协议 |
| **HTTP/REST API** | 🔜 计划中 | ⭐⭐⭐⭐ | RESTful数据源 |
| **数据库 (ODBC)** | 🔜 计划中 | ⭐⭐⭐⭐ | SQL数据库采集 |
| **OPC Classic** | 🔜 计划中 | ⭐⭐⭐ | OPC DA/AE/HDA |
| **WebSocket** | 🔜 计划中 | ⭐⭐⭐ | 实时Web通信 |

### 🟢 车载/物联网协议

| 协议 | 状态 | 优先级 | 说明 |
|------|------|--------|------|
| **CAN bus** | 🔜 计划中 | ⭐⭐⭐⭐ | 汽车/工业控制 |
| **CANopen** | 🔜 计划中 | ⭐⭐⭐ | CAN的高层协议 |
| **LIN bus** | 🔜 计划中 | ⭐⭐ | 低成本车载网络 |
| **FlexRay** | 🔜 计划中 | ⭐⭐ | 高速车载网络 |

### 🟢 楼宇自动化协议

| 协议 | 状态 | 优先级 | 说明 |
|------|------|--------|------|
| **BACnet** | 🔜 计划中 | ⭐⭐⭐ | 楼宇自动化标准 |
| **KNX** | 🔜 计划中 | ⭐⭐⭐ | 智能楼宇 |
| **LonWorks** | 🔜 计划中 | ⭐⭐ | 控制网络协议 |

---

## 已实现协议详情

### ✅ PROFIBUS DP/PA

**实现文件**: `serial/profibus.go`
**示例代码**: `examples/profibus_data_collection/`

**功能特性:**
- ✅ Master/Slave模式
- ✅ 循环数据交换 (Cyclic Data Exchange)
- ✅ 参数化和配置 (Parameterization & Configuration)
- ✅ 诊断功能 (Diagnostics)
- ✅ 数据读取 (ReadInputData)
- ✅ 数据写入 (WriteOutputData)
- ✅ 实时监听 (StartCyclicExchange)
- ✅ FCS校验
- ✅ 支持DP-V0标准

**待增强:**
- ⏳ DP-V1非循环服务 (Acyclic Services)
- ⏳ DP-V2特性
- ⏳ PA (Process Automation)扩展
- ⏳ GSD文件解析
- ⏳ 等时模式 (Isochronous Mode)

**使用示例:**
```go
config := &serial.PROFIBUSConfig{
    Mode:       serial.PROFIBUS_DP,
    Role:       serial.PROFIBUSMaster,
    MasterAddr: 2,
    SerialPort: "/dev/ttyUSB0",
    BaudRate:   19200,
    SlaveAddrs: []byte{3, 4, 5},
}

profibus, _ := serial.NewPROFIBUS(config)
profibus.Open("")

// 读取从站数据
data, _ := profibus.ReadInputData(3)

// 写入从站数据
profibus.WriteOutputData(3, []byte{0x01, 0x02})

// 启动循环数据交换
profibus.StartCyclicExchange(ctx)
```

---

### ✅ Modbus RTU/TCP/ASCII

**实现文件**: `serial/modbus.go`
**示例代码**: `examples/modbus_data_collection/`

**功能特性:**
- ✅ 支持3种模式：RTU、TCP、ASCII
- ✅ Master/Slave角色
- ✅ 所有标准功能码 (0x01-0x10)
- ✅ CRC-16校验 (RTU)
- ✅ LRC校验 (ASCII)
- ✅ MBAP头 (TCP)
- ✅ 读取线圈、寄存器
- ✅ 写入线圈、寄存器
- ✅ 批量读写
- ✅ 超时和重试机制

**使用示例:**
```go
// Modbus RTU
config := &serial.ModbusConfig{
    Mode:       serial.ModbusRTU,
    Role:       serial.ModbusMaster,
    SerialPort: "/dev/ttyUSB0",
    BaudRate:   9600,
}

// Modbus TCP
config := &serial.ModbusConfig{
    Mode:       serial.ModbusTCP,
    Role:       serial.ModbusMaster,
    TCPAddress: "192.168.1.100:502",
}

modbus, _ := serial.NewModbusMaster(config)
modbus.Open("")

// 读取保持寄存器
values, _ := modbus.ReadHoldingRegisters(1, 0, 10)

// 写入寄存器
modbus.WriteSingleRegister(1, 100, 1234)
```

---

### ✅ OPC UA Client

**实现文件**: `serial/opcua.go`
**示例代码**: `examples/opcua_collection/`

**功能特性:**
- ✅ Client模式
- ✅ 订阅模式和轮询模式
- ✅ 多种安全策略
- ✅ 用户认证
- ✅ 节点读写
- ✅ 方法调用

**待增强:**
- ⏳ Server模式
- ⏳ Pub/Sub模式
- ⏳ 信息模型 (Information Model)

---

### ✅ MQTT

**实现文件**: `serial/mqtt.go`
**示例代码**: `examples/mqtt_collection/`

**功能特性:**
- ✅ 发布/订阅
- ✅ QoS 0/1/2
- ✅ TLS/SSL
- ✅ 用户认证
- ✅ 自动重连

---

## 开发优先级 (2024-2026)

### Phase 1: 现场层完善 (Q1 2024) ✅
- [x] Modbus RTU/TCP完整实现
- [x] PROFIBUS DP基础实现
- [x] 统一协议接口设计

### Phase 2: 工业以太网 (Q2-Q3 2024) 🔄
- [ ] PROFINET IO
- [ ] EtherCAT
- [ ] EtherNet/IP
- [ ] PROFIBUS DP-V1/V2增强

### Phase 3: IT层连接 (Q3-Q4 2024) 🔄
- [ ] 数据库接口 (ODBC/JDBC)
- [ ] RESTful API数据源
- [ ] OPC UA Pub/Sub
- [ ] WebSocket

### Phase 4: 扩展协议 (2025)
- [ ] HART
- [ ] DeviceNet
- [ ] CAN/CANopen
- [ ] BACnet
- [ ] KNX

### Phase 5: 高级特性 (2025-2026)
- [ ] 协议网关 (Protocol Gateway)
- [ ] 协议转换器 (Protocol Converter)
- [ ] 协议桥接 (Protocol Bridge)
- [ ] 多协议融合分析

---

## 协议性能对比

| 协议 | 实时性 | 确定性 | 距离 | 带宽 | 复杂度 |
|------|--------|--------|------|------|--------|
| PROFIBUS DP | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 1.2km | 12Mbps | 高 |
| PROFINET | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 100m | 100Mbps | 高 |
| EtherCAT | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 100m | 100Mbps | 高 |
| Modbus RTU | ⭐⭐⭐ | ⭐⭐⭐ | 1.2km | 115Kbps | 低 |
| Modbus TCP | ⭐⭐⭐ | ⭐⭐ | ∞ | 100Mbps | 低 |
| OPC UA | ⭐⭐ | ⭐⭐ | ∞ | 1Gbps | 高 |
| MQTT | ⭐⭐ | ⭐ | ∞ | 1Gbps | 低 |

---

## 协议选择指南

### 离散制造业
推荐：**PROFINET**, **EtherCAT**, **EtherNet/IP**
- 实时性要求高
- 需要运动控制
- 设备多为西门子/贝加莱/罗克韦尔

### 过程自动化
推荐：**PROFIBUS PA**, **HART**, **Modbus RTU**
- 防爆区域
- 长距离传输
- 传感器/变送器为主

### 楼宇自动化
推荐：**BACnet**, **KNX**, **Modbus TCP**
- 暖通空调控制
- 照明控制
- 能源管理

### IoT/云连接
推荐：**MQTT**, **OPC UA**, **HTTP/REST**
- 云平台集成
- 大数据分析
- 远程监控

---

## 贡献指南

欢迎贡献新的协议实现！请参考现有实现：

**实现步骤：**
1. 在`serial/`目录创建协议文件
2. 实现`IndustrialProtocol`接口
3. 提供三大核心功能：读取、监听、管控
4. 创建示例代码
5. 编写单元测试
6. 更新文档

**代码规范：**
- 遵循现有的代码风格
- 完整的注释和文档
- 错误处理要完善
- 提供配置验证

---

## 协议资源

### 标准文档
- **PROFIBUS**: IEC 61158 / IEC 61784
- **PROFINET**: IEC 61158 / IEC 61784
- **EtherCAT**: IEC 61158 / IEC 61784
- **Modbus**: Modbus.org
- **OPC UA**: OPC Foundation

### 测试工具
- **PROFIBUS**: Softing PROFIBUS Tester
- **Modbus**: ModScan, Modbus Poll
- **OPC UA**: UaExpert
- **MQTT**: MQTT.fx, mosquitto_pub/sub

### 开发工具
- **Wireshark**: 协议分析
- **Bus Analyzer**: 总线分析仪
- **Simulators**: 协议模拟器

---

## 更新日志

### 2024-01-22
- ✅ 实现PROFIBUS DP Master基础功能
- ✅ 实现Modbus RTU/TCP/ASCII完整功能
- ✅ 创建统一协议接口 `IndustrialProtocol`
- ✅ 完成示例代码和文档

### 2024-01-21
- ✅ 实现OPC UA Client
- ✅ 实现MQTT协议
- ✅ AI异常检测 + 设备控制集成

---

## 路线图可视化

```
     [现场层]                [控制层]               [IT层]
        ↓                       ↓                    ↓
┌───────────────┐      ┌────────────────┐     ┌──────────────┐
│ PROFIBUS DP   │◄────►│  PLC/SCADA     │◄───►│  OPC UA      │
│ Modbus RTU    │      │  Edge Gateway  │     │  MQTT        │
│ HART          │      │  Protocol      │     │  HTTP/REST   │
│ DeviceNet     │      │  Converter     │     │  Database    │
└───────────────┘      └────────────────┘     └──────────────┘
        ↓                       ↓                    ↓
┌───────────────┐      ┌────────────────┐     ┌──────────────┐
│ PROFINET      │      │  Data Fusion   │     │  Cloud       │
│ EtherCAT      │◄────►│  AI Analysis   │◄───►│  Platform    │
│ EtherNet/IP   │      │  Device Control│     │  Big Data    │
└───────────────┘      └────────────────┘     └──────────────┘
```

---

**项目目标**: 成为最全面的Go语言工业通信库！
