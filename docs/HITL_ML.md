# 人在环中的机器学习（Human-in-the-Loop ML）系统

## 概述

本项目实现了完整的人在环中的机器学习系统，用于工业物联网场景的异常检测和事件管理。系统通过人工标注不断优化模型，形成良性循环。

## 系统架构

```
┌──────────────────────────────────────────────────────────────┐
│                     数据采集层                                 │
│              (Collector - 多协议数据采集)                       │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                    异常检测层                                  │
│         (Anomaly Detection - 规则引擎 + 模式匹配)              │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐      │
│  │  规则引擎    │  │  相似度匹配   │  │  事件检测器     │      │
│  │ RuleEngine  │  │  Similarity  │  │ EventDetector  │      │
│  └─────────────┘  └──────────────┘  └────────────────┘      │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                    事件管理层                                  │
│              (Event Management - 事件记录与管理)               │
│  ┌─────────────┐  ┌──────────────┐                           │
│  │  事件管理器   │  │  循环缓冲区   │                           │
│  │EventManager │  │CircularBuffer│                           │
│  └─────────────┘  └──────────────┘                           │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                    人工标注层                                  │
│         (Annotation - 人工审核与标注工作流)                    │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐      │
│  │ 标注工作流   │  │  标注员管理   │  │  标注任务队列   │      │
│  │  Workflow   │  │  Annotator   │  │  TaskQueue     │      │
│  └─────────────┘  └──────────────┘  └────────────────┘      │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                    事件库                                      │
│              (Event Store - 已标注事件存储)                    │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐      │
│  │  内存索引    │  │  文件存储     │  │  相似度搜索     │      │
│  │   Index     │  │    JSONL     │  │   Similarity   │      │
│  └─────────────┘  └──────────────┘  └────────────────┘      │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                    模型优化层                                  │
│          (Model Optimization - 模型训练与更新)                 │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐      │
│  │  数据准备    │  │  模型训练     │  │  持续学习循环   │      │
│  │DataPreparation│ │  Training    │  │ContinuousLoop  │      │
│  └─────────────┘  └──────────────┘  └────────────────┘      │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                    模型部署                                    │
│             (Inference Engine - 模型推理)                      │
│              更新后的模型返回检测层                             │
└──────────────────────────────────────────────────────────────┘
```

## 工作流程

### 1. 异常检测阶段

```go
// 创建规则引擎
ruleEngine := anomaly.NewRuleEngine()

// 添加阈值规则
rule := anomaly.NewThresholdRule(
    "temp_high",
    "温度过高",
    "temperature",
    ">",
    80.0,
    anomaly.SeverityWarning,
)
ruleEngine.AddRule(rule)

// 创建事件检测器
detector := event.NewEventDetector(
    ruleEngine,
    patternMatcher,
    0.7, // 相似度阈值
    5*time.Second, // 上下文窗口
)

// 检测异常
events, _ := detector.Detect(sample)
```

### 2. 人工标注阶段

```go
// 创建标注工作流
workflow := event.NewAnnotationWorkflow(eventManager, eventStore)

// 注册标注员
annotator := &event.Annotator{
    ID:   "annotator_001",
    Name: "张工程师",
    Role: "资深工程师",
}
workflow.RegisterAnnotator(annotator)

// 创建标注任务
task, _ := workflow.CreateAnnotationTask(eventID, annotatorID, priority)

// 标注事件
labels := map[string]interface{}{
    "category": "temperature_anomaly",
    "severity": "high",
}
workflow.AnnotateEvent(eventID, annotatorID, true, "确认异常", labels)
```

### 3. 事件库管理

```go
// 创建事件库
eventStore := event.NewEventStore("data/events", true)

// 保存已确认的事件（自动完成）
// 已确认的事件会自动保存到事件库

// 查找相似事件
similarEvents := eventStore.GetSimilarEvents(event, 0.7, 10)

// 按时间范围查询
events := eventStore.GetEventsByTimeRange(startTime, endTime)
```

### 4. 模型优化阶段

```go
// 创建模型优化器
optimizer := event.NewModelOptimizer(eventStore, inferenceEngine, config)

// 准备训练数据
dataset, _ := optimizer.PrepareTrainingData()

// 训练模型
result, _ := optimizer.TrainModel("anomaly_classifier", dataset)

// 查看训练结果
fmt.Printf("准确率: %.2f%%\n", result.FinalAccuracy*100)
```

### 5. 持续学习循环

```go
// 创建持续学习循环
learningLoop := event.NewContinuousLearningLoop(
    optimizer,
    "anomaly_classifier",
    24*time.Hour, // 每24小时检查一次
)

// 启动持续学习
learningLoop.Start()

// 系统会自动：
// 1. 检查是否有足够的新标注数据
// 2. 自动重新训练模型
// 3. 更新推理引擎中的模型
// 4. 提高检测准确率
```

## 核心组件

### 1. 规则引擎 (RuleEngine)

支持多种规则类型：

- **阈值规则** (ThresholdRule): 单值阈值判断
- **范围规则** (RangeRule): 数值范围判断
- **变化率规则** (RateOfChangeRule): 检测突变
- **统计规则** (StatisticalRule): 基于均值和标准差

### 2. 相似度匹配 (Similarity)

支持多种相似度算法：

- **欧氏距离** (Euclidean)
- **余弦相似度** (Cosine)
- **皮尔逊相关** (Pearson)
- **动态时间规整** (DTW)
- **曼哈顿距离** (Manhattan)
- **切比雪夫距离** (Chebyshev)

### 3. 事件检测器 (EventDetector)

- 结合规则引擎和模式匹配
- 提取事件上下文（前后数据）
- 循环缓冲区保存历史数据
- 事件监听器机制

### 4. 标注工作流 (AnnotationWorkflow)

- 标注员管理
- 标注任务队列
- 权限控制
- 统计信息

### 5. 事件库 (EventStore)

- 内存索引 + 文件持久化
- 按类型、时间范围查询
- 相似事件搜索
- 统计分析

### 6. 模型优化器 (ModelOptimizer)

- 自动数据准备
- 训练集/验证集分割
- 早停机制
- 自动模型更新

## 配置参数

### 异常检测配置

```yaml
anomaly_detection:
  threshold: 0.7              # 相似度阈值
  context_window: "5s"        # 上下文窗口
  similarity_metric: "cosine" # 相似度算法
```

### 标注工作流配置

```yaml
annotation:
  auto_create_tasks: true     # 自动创建标注任务
  task_priority: 1            # 默认任务优先级
  review_required: false      # 是否需要二次审核
```

### 模型优化配置

```yaml
optimization:
  min_training_samples: 100   # 最小训练样本数
  retraining_interval: "24h"  # 重训练间隔
  validation_ratio: 0.2       # 验证集比例
  learning_rate: 0.01         # 学习率
  max_epochs: 100             # 最大训练轮数
  early_stopping_rounds: 10   # 早停轮数
  auto_update: true           # 自动更新模型
```

## 使用示例

完整示例请参考：`examples/hitl_ml/main.go`

```bash
cd examples/hitl_ml
go run main.go
```

## 数据流图

```
原始数据 → 异常检测 → 待标注事件
                ↓
            人工标注
                ↓
         确认/拒绝事件
                ↓
        保存到事件库 ← 相似度匹配
                ↓
           模型训练
                ↓
           模型更新
                ↓
        返回异常检测 ← 持续循环
```

## 性能指标

- **检测延迟**: < 10ms (单样本)
- **相似度计算**: < 5ms (DTW), < 1ms (余弦)
- **模型训练**: 取决于样本数量和模型复杂度
- **事件存储**: 支持百万级事件

## 扩展性

### 1. 自定义规则

```go
type CustomRule struct {
    anomaly.BaseRule
    // 自定义字段
}

func (cr *CustomRule) Evaluate(sample *collector.DataSample) (bool, float64, string) {
    // 自定义逻辑
}
```

### 2. 自定义相似度算法

```go
func customSimilarity(a, b []float64) float64 {
    // 自定义算法
}
```

### 3. 自定义模型

```go
type CustomModel struct {
    // 模型参数
}

func (cm *CustomModel) Predict(input *inference.Tensor) (*inference.Tensor, error) {
    // 自定义推理逻辑
}
```

## 最佳实践

1. **规则设计**: 从简单规则开始，逐步完善
2. **阈值设置**: 初期设置较低阈值，随着数据积累逐步提高
3. **标注质量**: 确保标注准确性，定期审核
4. **模型更新**: 定期检查模型性能，及时更新
5. **数据清理**: 定期清理低质量事件

## 常见问题

**Q: 如何提高检测准确率？**
A: 1) 积累更多标注数据 2) 优化规则参数 3) 调整相似度阈值

**Q: 误报率太高怎么办？**
A: 1) 提高检测阈值 2) 增加上下文验证 3) 使用更严格的规则

**Q: 模型训练失败？**
A: 1) 检查训练样本数量 2) 调整学习率 3) 增加训练轮数

**Q: 如何处理大量待标注事件？**
A: 1) 优先标注高分数事件 2) 批量标注相似事件 3) 多人协作标注

## 后续优化方向

1. **深度学习模型**: 集成LSTM、Transformer等模型
2. **主动学习**: 智能选择最有价值的样本进行标注
3. **迁移学习**: 利用预训练模型
4. **强化学习**: 自动调整检测参数
5. **联邦学习**: 多设备协同训练
6. **在线学习**: 实时更新模型

## 参考资料

- [异常检测综述](https://arxiv.org/abs/2009.04352)
- [主动学习](https://arxiv.org/abs/2009.00236)
- [人在环中的机器学习](https://arxiv.org/abs/2108.00941)
