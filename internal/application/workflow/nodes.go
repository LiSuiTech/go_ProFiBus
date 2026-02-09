package workflow

import (
	"context"
	"fmt"
	"strconv"

	"go_ProFiBus/pkg/interfaces"
)

// DataSourceNodeExecutor 数据源节点执行器
type DataSourceNodeExecutor struct {
	dataSourceFactory func(config map[string]interface{}) (interfaces.DataSource, error)
}

// NewDataSourceNodeExecutor 创建数据源节点执行器
func NewDataSourceNodeExecutor(factory func(config map[string]interface{}) (interfaces.DataSource, error)) *DataSourceNodeExecutor {
	return &DataSourceNodeExecutor{
		dataSourceFactory: factory,
	}
}

func (e *DataSourceNodeExecutor) GetNodeType() NodeType {
	return NodeTypeDataSource
}

func (e *DataSourceNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["source_id"]; !ok {
		return fmt.Errorf("source_id is required")
	}
	return nil
}

func (e *DataSourceNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	if e.dataSourceFactory == nil {
		return nil, fmt.Errorf("data source factory not configured")
	}

	// 创建数据源
	source, err := e.dataSourceFactory(node.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create data source: %w", err)
	}

	// 启动数据源
	if err := source.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start data source: %w", err)
	}
	defer source.Stop()

	// 读取数据
	dataChan := source.GetData()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case sample := <-dataChan:
		return map[string]interface{}{
			"data":    sample.GetData(),
			"quality": sample.GetQuality(),
		}, nil
	}
}

// RuleDetectionNodeExecutor 规则检测节点执行器
type RuleDetectionNodeExecutor struct {
	ruleEngine func(config map[string]interface{}) (interfaces.RuleEngine, error)
}

// NewRuleDetectionNodeExecutor 创建规则检测节点执行器
func NewRuleDetectionNodeExecutor(engineFactory func(config map[string]interface{}) (interfaces.RuleEngine, error)) *RuleDetectionNodeExecutor {
	return &RuleDetectionNodeExecutor{
		ruleEngine: engineFactory,
	}
}

func (e *RuleDetectionNodeExecutor) GetNodeType() NodeType {
	return NodeTypeRuleDetection
}

func (e *RuleDetectionNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["rule_id"]; !ok {
		return fmt.Errorf("rule_id is required")
	}
	return nil
}

func (e *RuleDetectionNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// 获取输入数据
	data, ok := inputs["data"]
	if !ok {
		return nil, fmt.Errorf("input data not found")
	}

	// 这里需要将数据转换为 DataSample
	// 简化实现，实际需要根据具体数据结构转换
	result := map[string]interface{}{
		"detected": true,
		"score":    0.8,
		"data":     data,
	}

	return result, nil
}

// ConditionNodeExecutor 条件分支节点执行器
type ConditionNodeExecutor struct{}

func NewConditionNodeExecutor() *ConditionNodeExecutor {
	return &ConditionNodeExecutor{}
}

func (e *ConditionNodeExecutor) GetNodeType() NodeType {
	return NodeTypeCondition
}

func (e *ConditionNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["condition"]; !ok {
		return fmt.Errorf("condition is required")
	}
	return nil
}

func (e *ConditionNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// 获取条件表达式
	condition, ok := node.Config["condition"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid condition format")
	}

	// 简单的条件求值（实际应该使用表达式引擎）
	result := e.evaluateCondition(condition, inputs, variables)

	return map[string]interface{}{
		"result": result,
		"condition": condition,
	}, nil
}

func (e *ConditionNodeExecutor) evaluateCondition(condition string, inputs map[string]interface{}, variables map[string]interface{}) bool {
	// 简化实现：支持简单的比较表达式
	// 例如: "value > 10", "status == 'active'"
	// 实际应该使用表达式引擎如 expr 或 cel-go

	// 这里提供一个非常简化的实现
	if condition == "true" {
		return true
	}
	if condition == "false" {
		return false
	}

	// 尝试从变量中获取值
	if val, ok := variables[condition]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}

	return false
}

// VariableSetNodeExecutor 变量设置节点执行器
type VariableSetNodeExecutor struct{}

func NewVariableSetNodeExecutor() *VariableSetNodeExecutor {
	return &VariableSetNodeExecutor{}
}

func (e *VariableSetNodeExecutor) GetNodeType() NodeType {
	return NodeTypeVariableSet
}

func (e *VariableSetNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["variable_name"]; !ok {
		return fmt.Errorf("variable_name is required")
	}
	return nil
}

func (e *VariableSetNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	varName, ok := node.Config["variable_name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid variable_name")
	}

	// 获取值（可以从输入或配置中获取）
	var value interface{}
	if val, ok := node.Config["value"]; ok {
		value = val
	} else if val, ok := inputs["value"]; ok {
		value = val
	} else {
		return nil, fmt.Errorf("value not found")
	}

	return map[string]interface{}{
		"value": value,
		"variable_name": varName,
	}, nil
}

// OutputNodeExecutor 输出节点执行器
type OutputNodeExecutor struct{}

func NewOutputNodeExecutor() *OutputNodeExecutor {
	return &OutputNodeExecutor{}
}

func (e *OutputNodeExecutor) GetNodeType() NodeType {
	return NodeTypeOutput
}

func (e *OutputNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (e *OutputNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// 输出节点将所有输入和变量作为输出
	output := make(map[string]interface{})
	output["inputs"] = inputs
	output["variables"] = variables

	return output, nil
}

// TransformNodeExecutor 数据转换节点执行器
type TransformNodeExecutor struct{}

func NewTransformNodeExecutor() *TransformNodeExecutor {
	return &TransformNodeExecutor{}
}

func (e *TransformNodeExecutor) GetNodeType() NodeType {
	return NodeTypeTransform
}

func (e *TransformNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["transform"]; !ok {
		return fmt.Errorf("transform configuration is required")
	}
	return nil
}

func (e *TransformNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// 获取转换配置
	transform, ok := node.Config["transform"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid transform configuration")
	}

	// 应用转换
	result := make(map[string]interface{})
	for k, v := range inputs {
		if transformType, ok := transform[k].(string); ok {
			switch transformType {
			case "multiply":
				if factor, ok := transform[k+"_factor"].(float64); ok {
					if num, err := toFloat64(v); err == nil {
						result[k] = num * factor
					}
				}
			case "add":
				if offset, ok := transform[k+"_offset"].(float64); ok {
					if num, err := toFloat64(v); err == nil {
						result[k] = num + offset
					}
				}
			default:
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result, nil
}

// FilterNodeExecutor 数据过滤节点执行器
type FilterNodeExecutor struct{}

func NewFilterNodeExecutor() *FilterNodeExecutor {
	return &FilterNodeExecutor{}
}

func (e *FilterNodeExecutor) GetNodeType() NodeType {
	return NodeTypeFilter
}

func (e *FilterNodeExecutor) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["filter"]; !ok {
		return fmt.Errorf("filter configuration is required")
	}
	return nil
}

func (e *FilterNodeExecutor) Execute(ctx context.Context, node *Node, inputs map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// 获取过滤配置
	filter, ok := node.Config["filter"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid filter configuration")
	}

	// 应用过滤
	result := make(map[string]interface{})
	for k, v := range inputs {
		if filterRule, ok := filter[k].(map[string]interface{}); ok {
			// 检查过滤条件
			if shouldInclude(v, filterRule) {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result, nil
}

// toFloat64 转换为 float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert to float64")
	}
}

// shouldInclude 判断是否应该包含该值
func shouldInclude(value interface{}, filterRule map[string]interface{}) bool {
	// 简化实现
	if min, ok := filterRule["min"].(float64); ok {
		if num, err := toFloat64(value); err == nil {
			if num < min {
				return false
			}
		}
	}
	if max, ok := filterRule["max"].(float64); ok {
		if num, err := toFloat64(value); err == nil {
			if num > max {
				return false
			}
		}
	}
	return true
}
