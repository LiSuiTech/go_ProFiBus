package interfaces

import (
	"context"
	"time"
)

// Analyzer 分析器接口
// 对数据进行分析，检测异常、模式等
type Analyzer interface {
	// Analyze 分析数据
	Analyze(ctx context.Context, data DataSample) ([]AnalysisResult, error)

	// GetType 获取分析器类型
	GetType() AnalyzerType

	// GetName 获取分析器名称
	GetName() string

	// Configure 配置分析器
	Configure(config map[string]interface{}) error

	// Close 关闭分析器
	Close() error
}

// AnalyzerType 分析器类型
type AnalyzerType string

const (
	// AnalyzerTypeThreshold 阈值分析
	AnalyzerTypeThreshold AnalyzerType = "threshold"

	// AnalyzerTypeStatistical 统计分析
	AnalyzerTypeStatistical AnalyzerType = "statistical"

	// AnalyzerTypePattern 模式识别
	AnalyzerTypePattern AnalyzerType = "pattern"

	// AnalyzerTypeSimilarity 相似度分析
	AnalyzerTypeSimilarity AnalyzerType = "similarity"

	// AnalyzerTypeML 机器学习
	AnalyzerTypeML AnalyzerType = "ml"
)

// AnalysisResult 分析结果
type AnalysisResult struct {
	// Type 结果类型
	Type string

	// Severity 严重程度 (1-5)
	Severity int

	// Score 分数 (0.0-1.0)
	Score float64

	// Message 描述信息
	Message string

	// Details 详细信息
	Details map[string]interface{}

	// Timestamp 时间戳
	Timestamp time.Time

	// SourceID 数据源ID
	SourceID string
}

// RuleEngine 规则引擎接口
// 管理和执行多条规则
type RuleEngine interface {
	// AddRule 添加规则
	AddRule(rule Rule) error

	// RemoveRule 移除规则
	RemoveRule(ruleID string) error

	// UpdateRule 更新规则
	UpdateRule(ruleID string, rule Rule) error

	// GetRule 获取规则
	GetRule(ruleID string) (Rule, error)

	// ListRules 列出所有规则
	ListRules() []Rule

	// Evaluate 评估规则
	Evaluate(ctx context.Context, data DataSample) ([]RuleResult, error)

	// EvaluateConcurrent 并发评估规则
	EvaluateConcurrent(ctx context.Context, data DataSample) ([]RuleResult, error)
}

// Rule 规则接口
type Rule interface {
	// GetID 获取规则ID
	GetID() string

	// GetName 获取规则名称
	GetName() string

	// GetType 获取规则类型
	GetType() string

	// IsEnabled 是否启用
	IsEnabled() bool

	// Evaluate 评估规则
	Evaluate(data DataSample) (*RuleResult, error)

	// GetConfig 获取规则配置
	GetConfig() map[string]interface{}
}

// RuleResult 规则评估结果
type RuleResult struct {
	// RuleID 规则ID
	RuleID string

	// RuleName 规则名称
	RuleName string

	// Triggered 是否触发
	Triggered bool

	// Severity 严重程度
	Severity int

	// Score 分数
	Score float64

	// Message 消息
	Message string

	// Details 详细信息
	Details map[string]interface{}

	// Timestamp 时间戳
	Timestamp time.Time
}
