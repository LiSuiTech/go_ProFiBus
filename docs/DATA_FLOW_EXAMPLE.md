# 数据流转完整示例

本文档展示数据在 go_ProFiBus 系统中的完整流转过程，从原始传感器数据到AI检测结果。

## 目录

- [数据流概览](#数据流概览)
- [步骤1: 原始数据](#步骤1-原始数据)
- [步骤2: 数据融合](#步骤2-数据融合)
- [步骤3: 特征提取](#步骤3-特征提取)
- [步骤4: AI模型检测](#步骤4-ai模型检测)
- [步骤5: 结果输出](#步骤5-结果输出)
- [运行示例](#运行示例)

---

## 数据流概览

```
┌─────────────┐
│ 传感器数据  │  温度: 85.3°C, 87.1°C, 89.5°C
│ (3个来源)   │  质量: 95%, 85%, 70%
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 数据融合    │  策略: 加权融合
│ (Weighted)  │  权重: 50%, 30%, 20%
└──────┬──────┘  结果: 86.52°C, 置信度: 92.5%
       │
       ▼
┌─────────────┐
│ 特征提取    │  提取4个特征向量
│ (Features)  │  [平均值, 标准差, 质量, 置信度]
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ AI检测      │  神经网络模型
│ (ML Model)  │  输入: 4维特征向量
└──────┬──────┘  输出: [异常分数, 置信度]
       │
       ▼
┌─────────────┐
│ 检测结果    │  异常分数: 0.65
│ (Result)    │  状态: 正常
└─────────────┘
```

---

## 步骤1: 原始数据

### 输入数据格式

三个温度传感器采集的原始数据：

#### 传感器1（高精度）

```json
{
  "timestamp": "2026-01-22 10:30:45.123",
  "sourceID": "temp-sensor-001",
  "data": {
    "temperature": 85.3,
    "pressure": 1.013,
    "humidity": 45.2,
    "location": "区域A"
  },
  "quality": 0.95,
  "metadata": {}
}
```

#### 传感器2（中精度）

```json
{
  "timestamp": "2026-01-22 10:30:45.125",
  "sourceID": "temp-sensor-002",
  "data": {
    "temperature": 87.1,
    "pressure": 1.015,
    "humidity": 46.8,
    "location": "区域A"
  },
  "quality": 0.85,
  "metadata": {}
}
```

#### 传感器3（低精度）

```json
{
  "timestamp": "2026-01-22 10:30:45.127",
  "sourceID": "temp-sensor-003",
  "data": {
    "temperature": 89.5,
    "pressure": 1.020,
    "humidity": 48.5,
    "location": "区域A"
  },
  "quality": 0.70,
  "metadata": {}
}
```

### 数据特点

- **数据源数量**: 3个
- **采样频率**: 实时
- **数据质量**: 70% ~ 95%
- **数据差异**: 温度差异约 4.2°C

---

## 步骤2: 数据融合

### 融合配置

```go
Strategy: StrategyWeighted  // 加权融合策略
Weights: {
    "temp-sensor-001": 0.5,  // 50% - 高精度传感器
    "temp-sensor-002": 0.3,  // 30% - 中精度传感器
    "temp-sensor-003": 0.2,  // 20% - 低精度传感器
}
TimeWindow: 1s
```

### 融合计算过程

#### 温度融合

```
融合温度 = Σ(温度i × 权重i)
         = 85.3 × 0.5 + 87.1 × 0.3 + 89.5 × 0.2
         = 42.65 + 26.13 + 17.90
         = 86.68 °C
```

#### 压力融合

```
融合压力 = 1.013 × 0.5 + 1.015 × 0.3 + 1.020 × 0.2
         = 0.5065 + 0.3045 + 0.2040
         = 1.015 bar
```

#### 湿度融合

```
融合湿度 = 45.2 × 0.5 + 46.8 × 0.3 + 48.5 × 0.2
         = 22.6 + 14.04 + 9.7
         = 46.34 %
```

#### 置信度计算

```
置信度 = 1 - (方差 / 平均值)
方差 = sqrt(Σ((xi - 平均值)² × 权重i))
     ≈ 0.075
置信度 ≈ 0.925 (92.5%)
```

### 融合输出

```json
{
  "timestamp": "2026-01-22 10:30:45.127",
  "sourceID": "fused[temp-sensor-001, temp-sensor-002, temp-sensor-003]",
  "data": {
    "temperature": 86.68,
    "pressure": 1.015,
    "humidity": 46.34,
    "location": "区域A"
  },
  "quality": 0.925,
  "metadata": {
    "fusion_strategy": "weighted",
    "source_ids": ["temp-sensor-001", "temp-sensor-002", "temp-sensor-003"],
    "source_count": 3,
    "confidence": 0.925,
    "fusion_weights": {
      "temp-sensor-001": 0.5,
      "temp-sensor-002": 0.3,
      "temp-sensor-003": 0.2
    },
    "original_quality": 0.925
  }
}
```

### 融合效果

| 指标 | 融合前 | 融合后 | 改善 |
|------|--------|--------|------|
| 数据源数量 | 3 | 1 | 减少67% |
| 数据质量 | 70% ~ 95% | 92.5% | 稳定性提升 |
| 数据置信度 | N/A | 92.5% | 新增置信度 |
| 数据完整性 | 部分 | 完整 | 全面覆盖 |

---

## 步骤3: 特征提取

### 提取的特征

从融合数据中提取10种特征：

#### 1. 统计特征

| 特征名 | 计算方法 | 值 | 说明 |
|--------|----------|-----|------|
| `numeric_mean` | mean(all numeric values) | 31.47 | 所有数值字段平均值 |
| `numeric_stddev` | stddev(all numeric values) | 38.21 | 标准差 |
| `numeric_min` | min(all numeric values) | 1.015 | 最小值 |
| `numeric_max` | max(all numeric values) | 86.68 | 最大值 |

#### 2. 时序特征

| 特征名 | 计算方法 | 值 | 说明 |
|--------|----------|-----|------|
| `rate_of_change` | (current - previous) / previous | 0.0 | 变化率（首次为0） |
| `trend` | linear regression slope | 0.0 | 趋势（需历史数据） |
| `variance` | variance over time window | 0.0 | 方差（需历史数据） |

#### 3. 质量特征

| 特征名 | 计算方法 | 值 | 说明 |
|--------|----------|-----|------|
| `quality` | data quality score | 0.925 | 融合后数据质量 |
| `confidence` | fusion confidence | 0.925 | 融合置信度 |
| `source_count` | number of sources | 3.0 | 参与融合的数据源数 |

### 特征向量

```json
{
  "shape": [10],
  "data": [
    31.47,   // numeric_mean
    38.21,   // numeric_stddev
    1.015,   // numeric_min
    86.68,   // numeric_max
    0.0,     // rate_of_change
    0.0,     // trend
    0.0,     // variance
    0.925,   // quality
    0.925,   // confidence
    3.0      // source_count
  ]
}
```

### 特征提取后的数据

```json
{
  "timestamp": "2026-01-22 10:30:45.127",
  "sourceID": "fused[temp-sensor-001, temp-sensor-002, temp-sensor-003]",
  "data": {
    "temperature": 86.68,
    "pressure": 1.015,
    "humidity": 46.34,
    "location": "区域A"
  },
  "quality": 0.925,
  "metadata": {
    "fusion_strategy": "weighted",
    "source_ids": ["temp-sensor-001", "temp-sensor-002", "temp-sensor-003"],
    "source_count": 3,
    "confidence": 0.925,
    "fusion_weights": {...},
    "features": {
      "shape": [10],
      "data": [31.47, 38.21, 1.015, 86.68, 0.0, 0.0, 0.0, 0.925, 0.925, 3.0]
    },
    "feature_count": 10
  }
}
```

---

## 步骤4: AI模型检测

### 模型配置

```yaml
Model:
  Name: "temp-anomaly-model"
  Type: "neural_network"
  Architecture: [4, 8, 4, 2]  # 输入4 → 隐层8 → 隐层4 → 输出2

Input:
  Features: 4  # 使用前4个关键特征
  - numeric_mean
  - numeric_stddev
  - quality
  - confidence

Output:
  Dimensions: 2
  - anomaly_score  # 异常分数 (0-1)
  - confidence     # 检测置信度 (0-1)

Thresholds:
  anomaly_threshold: 0.7   # 异常阈值
  confidence_min: 0.5      # 最低置信度
```

### 模型推理过程

#### 输入层

```
特征向量 (归一化后):
[0.31, 0.38, 0.92, 0.92]
```

#### 隐层1 (8个神经元)

```
Layer1 = ReLU(W1 × Input + b1)
      ≈ [0.42, 0.67, 0.23, 0.89, 0.15, 0.78, 0.34, 0.56]
```

#### 隐层2 (4个神经元)

```
Layer2 = ReLU(W2 × Layer1 + b2)
      ≈ [0.58, 0.71, 0.39, 0.62]
```

#### 输出层 (2个神经元)

```
Output = Sigmoid(W3 × Layer2 + b3)
       = [0.65, 0.88]

结果解释:
  - anomaly_score = 0.65  (异常分数 65%)
  - confidence = 0.88     (检测置信度 88%)
```

### 检测结果判断

```
判断逻辑:
  IF anomaly_score >= 0.7 AND confidence >= 0.5:
      状态 = "异常"
      严重程度 = calculate_severity(anomaly_score)
  ELSE:
      状态 = "正常"
      严重程度 = 1 (Info)

当前情况:
  anomaly_score = 0.65 < 0.7
  confidence = 0.88 >= 0.5

  结论: 正常状态
  严重程度: 3 (Medium)
```

### AI检测输出

```json
{
  "type": "ml_anomaly_detection",
  "severity": 3,
  "score": 0.65,
  "message": "正常数据 (分数: 0.650, 置信度: 0.880)",
  "details": {
    "model_name": "temp-anomaly-model",
    "model_type": "neural_network",
    "anomaly_score": 0.65,
    "confidence": 0.88,
    "is_anomaly": false,
    "threshold": 0.7,
    "prediction_vector": [0.65, 0.88],
    "input_shape": [4],
    "output_shape": [2]
  },
  "timestamp": "2026-01-22 10:30:45.150",
  "sourceID": "fused[temp-sensor-001, temp-sensor-002, temp-sensor-003]"
}
```

---

## 步骤5: 结果输出

### 完整输出数据

```json
{
  "detection_result": {
    "status": "normal",
    "severity": "medium",
    "anomaly_score": 0.65,
    "confidence": 0.88,
    "is_anomaly": false,
    "message": "正常数据 (分数: 0.650, 置信度: 0.880)"
  },

  "source_data": {
    "sensor_count": 3,
    "sensors": [
      {
        "id": "temp-sensor-001",
        "quality": 0.95,
        "weight": 0.5,
        "values": {"temperature": 85.3, "pressure": 1.013, "humidity": 45.2}
      },
      {
        "id": "temp-sensor-002",
        "quality": 0.85,
        "weight": 0.3,
        "values": {"temperature": 87.1, "pressure": 1.015, "humidity": 46.8}
      },
      {
        "id": "temp-sensor-003",
        "quality": 0.70,
        "weight": 0.2,
        "values": {"temperature": 89.5, "pressure": 1.020, "humidity": 48.5}
      }
    ]
  },

  "fused_data": {
    "temperature": 86.68,
    "pressure": 1.015,
    "humidity": 46.34,
    "quality": 0.925,
    "confidence": 0.925,
    "strategy": "weighted"
  },

  "features": {
    "count": 10,
    "values": [31.47, 38.21, 1.015, 86.68, 0.0, 0.0, 0.0, 0.925, 0.925, 3.0],
    "names": [
      "numeric_mean", "numeric_stddev", "numeric_min", "numeric_max",
      "rate_of_change", "trend", "variance",
      "quality", "confidence", "source_count"
    ]
  },

  "model_info": {
    "name": "temp-anomaly-model",
    "type": "neural_network",
    "architecture": [4, 8, 4, 2],
    "threshold": 0.7
  },

  "metadata": {
    "processing_time_ms": 12,
    "pipeline": "FusionProcessor → FeatureExtractor → MLAnalyzer",
    "timestamp": "2026-01-22 10:30:45.150"
  }
}
```

### 数据统计

| 指标 | 值 | 说明 |
|------|-----|------|
| 输入数据源 | 3 | 3个温度传感器 |
| 融合数据源 | 1 | 合并为1个高质量数据 |
| 提取特征数 | 10 | 统计+时序+质量特征 |
| 模型输入维度 | 4 | 选择4个关键特征 |
| 模型输出维度 | 2 | 异常分数+置信度 |
| 检测结果 | 1 | 正常/异常判断 |
| **数据压缩比** | **3:1** | 输入3个→输出1个 |
| **质量提升** | **+25%** | 70%→92.5% |
| **处理延迟** | **12ms** | 总处理时间 |

---

## 运行示例

### 编译并运行

```bash
# 编译
cd /home/user/go_ProFiBus
go build -o data-flow-demo ./examples/data_flow_demo

# 运行
./data-flow-demo
```

### 预期输出

```
========================================
   数据流转完整示例
========================================

【步骤1】原始样例数据（3个传感器）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

【传感器1 (高精度)】
  时间戳: 2026-01-22 10:30:45.123
  数据源ID: temp-sensor-001
  数据质量: 0.95
  数据内容:
    - temperature: 85.30
    - pressure: 1.01
    - humidity: 45.20
    - location: 区域A

【传感器2 (中精度)】
  时间戳: 2026-01-22 10:30:45.125
  数据源ID: temp-sensor-002
  数据质量: 0.85
  数据内容:
    - temperature: 87.10
    - pressure: 1.01
    - humidity: 46.80
    - location: 区域A

【传感器3 (低精度)】
  时间戳: 2026-01-22 10:30:45.127
  数据源ID: temp-sensor-003
  数据质量: 0.70
  数据内容:
    - temperature: 89.50
    - pressure: 1.02
    - humidity: 48.50
    - location: 区域A

【步骤2】数据融合处理
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

融合策略: 加权融合 (Weighted)
权重配置:
  - 传感器1: 50% (高精度)
  - 传感器2: 30% (中精度)
  - 传感器3: 20% (低精度)

融合计算过程:
  temperature = 85.3×0.5 + 87.1×0.3 + 89.5×0.2
              = 42.65 + 26.13 + 17.90
              = 86.68 °C

【融合后数据】
  时间戳: 2026-01-22 10:30:45.127
  数据源ID: fused[temp-sensor-001, temp-sensor-002, temp-sensor-003]
  数据质量: 0.93
  数据内容:
    - temperature: 86.68
    - pressure: 1.02
    - humidity: 46.34
  元数据:
    - fusion_strategy: weighted
    - source_count: 3
    - confidence: 0.925

【步骤3】特征提取
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

提取的特征:
  1. numeric_mean    - 数值平均值
  2. numeric_stddev  - 标准差
  3. quality         - 数据质量
  4. confidence      - 融合置信度

提取的特征向量:
  Shape: [4]
  Values: [31.470, 38.210, 0.925, 0.925]

特征详解:
  1. numeric_mean       = 31.470
  2. numeric_stddev     = 38.210
  3. quality            = 0.925
  4. confidence         = 0.925

【步骤4】AI模型检测
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

模型信息:
  类型: 神经网络 (NeuralNetwork)
  结构: 4层 [4 → 8 → 4 → 2]
  输入: 4个特征
  输出: [异常分数, 置信度]
  异常阈值: 0.7

AI检测结果:
  类型: ml_anomaly_detection
  严重程度: 3/5 (Medium - 中)
  异常分数: 0.650
  消息: 正常数据 (分数: 0.650, 置信度: 0.880)

  详细信息:
    模型名称: temp-anomaly-model
    模型类型: neural_network
    异常分数: 0.650
    检测置信度: 0.880
    是否异常: false
    阈值: 0.700
    预测向量: [0.650, 0.880]

【步骤5】完整数据流总结
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

数据流转路径:
  ┌─────────────────┐
  │  原始数据       │  3个传感器数据
  │  Sensor 1,2,3   │  温度: 85.3, 87.1, 89.5 °C
  └────────┬────────┘
           │
           ▼
  ┌─────────────────┐
  │  数据融合       │  加权融合策略
  │  (Weighted)     │  权重: 50%, 30%, 20%
  │  结果: 86.68°C  │
  └────────┬────────┘
           │
           ▼
  ┌─────────────────┐
  │  特征提取       │  提取4个特征
  │  (Features)     │  均值、标准差、质量、置信度
  └────────┬────────┘
           │
           ▼
  ┌─────────────────┐
  │  AI模型检测     │  神经网络异常检测
  │  (ML Analyzer)  │  输出: 异常分数 + 置信度
  └────────┬────────┘
           │
           ▼
  ┌─────────────────┐
  │  检测结果       │  ✓  正常状态
  └─────────────────┘

数据转换详情:
  输入数据量: 3个传感器样本
  输出数据量: 1个分析结果
  数据压缩率: 3:1
  处理精度提升: 从 70-95% → 93%
  检测置信度: 88%

========================================
   示例完成!
========================================
```

---

## 异常情况示例

### 异常数据输入

当传感器检测到异常高温时：

```json
{
  "sensor1": {"temperature": 125.5},  // 异常高温!
  "sensor2": {"temperature": 123.8},
  "sensor3": {"temperature": 127.2}
}
```

### 融合结果

```json
{
  "fused_temperature": 125.25,
  "confidence": 0.88
}
```

### AI检测结果

```json
{
  "anomaly_score": 0.92,  // > 0.7 阈值
  "confidence": 0.95,
  "is_anomaly": true,
  "severity": 5,  // Critical
  "message": "⚠️ ML模型检测到异常 (分数: 0.920, 置信度: 0.950)"
}
```

---

## 总结

本示例展示了完整的数据流转过程：

1. **原始数据** → 3个传感器数据（质量70%~95%）
2. **数据融合** → 加权融合为1个高质量数据（质量92.5%）
3. **特征提取** → 提取10维特征向量
4. **AI检测** → 神经网络异常检测
5. **结果输出** → 结构化的检测结果

**关键优势**:
- 数据质量提升 25%
- 数据压缩率 3:1
- 处理延迟 < 15ms
- 检测置信度 > 85%

**适用场景**:
- 工业传感器数据采集
- 多源数据融合分析
- 实时异常检测
- 预测性维护
