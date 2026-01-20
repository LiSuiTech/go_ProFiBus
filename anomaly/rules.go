package anomaly

import (
	"fmt"
	"go_ProFiBus/collector"
	"go_ProFiBus/errors"
	"go_ProFiBus/logger"
	"math"
	"sync"
	"time"
)

// RuleType 规则类型
type RuleType int

const (
	RuleThreshold     RuleType = iota // 阈值规则
	RuleRange                          // 范围规则
	RuleRateOfChange                   // 变化率规则
	RuleStatistical                    // 统计规则
	RulePattern                        // 模式匹配规则
	RuleCustom                         // 自定义规则
)

// Severity 严重程度
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

var severityNames = map[Severity]string{
	SeverityInfo:     "信息",
	SeverityWarning:  "警告",
	SeverityError:    "错误",
	SeverityCritical: "严重",
}

// Rule 异常检测规则接口
type Rule interface {
	GetID() string
	GetName() string
	GetType() RuleType
	GetSeverity() Severity
	Evaluate(sample *collector.DataSample) (bool, float64, string)
	IsEnabled() bool
	SetEnabled(enabled bool)
}

// BaseRule 基础规则
type BaseRule struct {
	ID          string
	Name        string
	Description string
	Type        RuleType
	Severity    Severity
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	mu          sync.RWMutex
}

func (br *BaseRule) GetID() string       { return br.ID }
func (br *BaseRule) GetName() string     { return br.Name }
func (br *BaseRule) GetType() RuleType   { return br.Type }
func (br *BaseRule) GetSeverity() Severity { return br.Severity }

func (br *BaseRule) IsEnabled() bool {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.Enabled
}

func (br *BaseRule) SetEnabled(enabled bool) {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.Enabled = enabled
	br.UpdatedAt = time.Now()
}

// ThresholdRule 阈值规则
type ThresholdRule struct {
	BaseRule
	FieldName string
	Operator  string  // >, <, >=, <=, ==, !=
	Threshold float64
}

// NewThresholdRule 创建阈值规则
func NewThresholdRule(id, name, field, operator string, threshold float64, severity Severity) *ThresholdRule {
	return &ThresholdRule{
		BaseRule: BaseRule{
			ID:        id,
			Name:      name,
			Type:      RuleThreshold,
			Severity:  severity,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		FieldName: field,
		Operator:  operator,
		Threshold: threshold,
	}
}

// Evaluate 评估规则
func (tr *ThresholdRule) Evaluate(sample *collector.DataSample) (bool, float64, string) {
	if !tr.IsEnabled() {
		return false, 0, ""
	}

	value, ok := sample.ParsedData[tr.FieldName].(float64)
	if !ok {
		return false, 0, ""
	}

	var violated bool
	var score float64

	switch tr.Operator {
	case ">":
		violated = value > tr.Threshold
		score = (value - tr.Threshold) / tr.Threshold
	case "<":
		violated = value < tr.Threshold
		score = (tr.Threshold - value) / tr.Threshold
	case ">=":
		violated = value >= tr.Threshold
		score = (value - tr.Threshold) / tr.Threshold
	case "<=":
		violated = value <= tr.Threshold
		score = (tr.Threshold - value) / tr.Threshold
	case "==":
		violated = math.Abs(value-tr.Threshold) < 1e-6
		score = 1.0
	case "!=":
		violated = math.Abs(value-tr.Threshold) >= 1e-6
		score = math.Abs(value-tr.Threshold) / tr.Threshold
	}

	if violated {
		message := fmt.Sprintf("%s: %s %s %.2f (当前值: %.2f)",
			tr.Name, tr.FieldName, tr.Operator, tr.Threshold, value)
		return true, math.Abs(score), message
	}

	return false, 0, ""
}

// RangeRule 范围规则
type RangeRule struct {
	BaseRule
	FieldName string
	MinValue  float64
	MaxValue  float64
}

// NewRangeRule 创建范围规则
func NewRangeRule(id, name, field string, min, max float64, severity Severity) *RangeRule {
	return &RangeRule{
		BaseRule: BaseRule{
			ID:        id,
			Name:      name,
			Type:      RuleRange,
			Severity:  severity,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		FieldName: field,
		MinValue:  min,
		MaxValue:  max,
	}
}

// Evaluate 评估规则
func (rr *RangeRule) Evaluate(sample *collector.DataSample) (bool, float64, string) {
	if !rr.IsEnabled() {
		return false, 0, ""
	}

	value, ok := sample.ParsedData[rr.FieldName].(float64)
	if !ok {
		return false, 0, ""
	}

	if value < rr.MinValue {
		score := (rr.MinValue - value) / (rr.MaxValue - rr.MinValue)
		message := fmt.Sprintf("%s: %s 低于下限 %.2f (当前值: %.2f)",
			rr.Name, rr.FieldName, rr.MinValue, value)
		return true, score, message
	}

	if value > rr.MaxValue {
		score := (value - rr.MaxValue) / (rr.MaxValue - rr.MinValue)
		message := fmt.Sprintf("%s: %s 超出上限 %.2f (当前值: %.2f)",
			rr.Name, rr.FieldName, rr.MaxValue, value)
		return true, score, message
	}

	return false, 0, ""
}

// RateOfChangeRule 变化率规则
type RateOfChangeRule struct {
	BaseRule
	FieldName     string
	MaxRate       float64  // 最大变化率（每秒）
	TimeWindow    time.Duration
	previousValue *float64
	previousTime  *time.Time
}

// NewRateOfChangeRule 创建变化率规则
func NewRateOfChangeRule(id, name, field string, maxRate float64, window time.Duration, severity Severity) *RateOfChangeRule {
	return &RateOfChangeRule{
		BaseRule: BaseRule{
			ID:        id,
			Name:      name,
			Type:      RuleRateOfChange,
			Severity:  severity,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		FieldName:  field,
		MaxRate:    maxRate,
		TimeWindow: window,
	}
}

// Evaluate 评估规则
func (roc *RateOfChangeRule) Evaluate(sample *collector.DataSample) (bool, float64, string) {
	if !roc.IsEnabled() {
		return false, 0, ""
	}

	value, ok := sample.ParsedData[roc.FieldName].(float64)
	if !ok {
		return false, 0, ""
	}

	currentTime := sample.Timestamp

	if roc.previousValue == nil || roc.previousTime == nil {
		roc.previousValue = &value
		roc.previousTime = &currentTime
		return false, 0, ""
	}

	timeDiff := currentTime.Sub(*roc.previousTime).Seconds()
	if timeDiff < roc.TimeWindow.Seconds() {
		return false, 0, ""
	}

	valueDiff := value - *roc.previousValue
	rate := math.Abs(valueDiff / timeDiff)

	roc.previousValue = &value
	roc.previousTime = &currentTime

	if rate > roc.MaxRate {
		score := rate / roc.MaxRate
		message := fmt.Sprintf("%s: %s 变化率过快 %.2f/s (最大: %.2f/s)",
			roc.Name, roc.FieldName, rate, roc.MaxRate)
		return true, score, message
	}

	return false, 0, ""
}

// StatisticalRule 统计规则（基于标准差）
type StatisticalRule struct {
	BaseRule
	FieldName     string
	WindowSize    int
	SigmaMultiple float64 // 几倍标准差
	values        []float64
	mu            sync.Mutex
}

// NewStatisticalRule 创建统计规则
func NewStatisticalRule(id, name, field string, windowSize int, sigmaMultiple float64, severity Severity) *StatisticalRule {
	return &StatisticalRule{
		BaseRule: BaseRule{
			ID:        id,
			Name:      name,
			Type:      RuleStatistical,
			Severity:  severity,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		FieldName:     field,
		WindowSize:    windowSize,
		SigmaMultiple: sigmaMultiple,
		values:        make([]float64, 0, windowSize),
	}
}

// Evaluate 评估规则
func (sr *StatisticalRule) Evaluate(sample *collector.DataSample) (bool, float64, string) {
	if !sr.IsEnabled() {
		return false, 0, ""
	}

	value, ok := sample.ParsedData[sr.FieldName].(float64)
	if !ok {
		return false, 0, ""
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	// 添加新值
	sr.values = append(sr.values, value)
	if len(sr.values) > sr.WindowSize {
		sr.values = sr.values[1:]
	}

	// 需要足够的历史数据
	if len(sr.values) < sr.WindowSize/2 {
		return false, 0, ""
	}

	// 计算均值和标准差
	mean := 0.0
	for _, v := range sr.values {
		mean += v
	}
	mean /= float64(len(sr.values))

	variance := 0.0
	for _, v := range sr.values {
		diff := v - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(len(sr.values)))

	// 检查当前值是否偏离均值超过 N 倍标准差
	deviation := math.Abs(value - mean)
	threshold := sr.SigmaMultiple * stddev

	if deviation > threshold {
		score := deviation / threshold
		message := fmt.Sprintf("%s: %s 偏离均值 %.2f 倍标准差 (当前: %.2f, 均值: %.2f, 标准差: %.2f)",
			sr.Name, sr.FieldName, deviation/stddev, value, mean, stddev)
		return true, score, message
	}

	return false, 0, ""
}

// RuleEngine 规则引擎
type RuleEngine struct {
	rules  map[string]Rule
	mu     sync.RWMutex
	log    *logger.Logger
	stats  *EngineStats
}

// EngineStats 引擎统计
type EngineStats struct {
	TotalEvaluations uint64
	TotalViolations  uint64
	mu               sync.RWMutex
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: make(map[string]Rule),
		log:   logger.GetLogger(),
		stats: &EngineStats{},
	}
}

// AddRule 添加规则
func (re *RuleEngine) AddRule(rule Rule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.rules[rule.GetID()]; exists {
		return errors.Newf(errors.ErrInvalidParam, "规则已存在: %s", rule.GetID())
	}

	re.rules[rule.GetID()] = rule
	re.log.Info("添加规则: %s (%s)", rule.GetName(), rule.GetID())
	return nil
}

// RemoveRule 移除规则
func (re *RuleEngine) RemoveRule(ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.rules[ruleID]; !exists {
		return errors.Newf(errors.ErrInvalidParam, "规则不存在: %s", ruleID)
	}

	delete(re.rules, ruleID)
	re.log.Info("移除规则: %s", ruleID)
	return nil
}

// GetRule 获取规则
func (re *RuleEngine) GetRule(ruleID string) (Rule, bool) {
	re.mu.RLock()
	defer re.mu.RUnlock()
	rule, exists := re.rules[ruleID]
	return rule, exists
}

// ListRules 列出所有规则
func (re *RuleEngine) ListRules() []Rule {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rules := make([]Rule, 0, len(re.rules))
	for _, rule := range re.rules {
		rules = append(rules, rule)
	}
	return rules
}

// EvaluationResult 评估结果
type EvaluationResult struct {
	RuleID      string
	RuleName    string
	Severity    Severity
	Violated    bool
	Score       float64
	Message     string
	Sample      *collector.DataSample
	EvaluatedAt time.Time
}

// Evaluate 评估单个样本
func (re *RuleEngine) Evaluate(sample *collector.DataSample) []*EvaluationResult {
	re.mu.RLock()
	defer re.mu.RUnlock()

	results := make([]*EvaluationResult, 0)

	for _, rule := range re.rules {
		if !rule.IsEnabled() {
			continue
		}

		violated, score, message := rule.Evaluate(sample)

		re.stats.mu.Lock()
		re.stats.TotalEvaluations++
		if violated {
			re.stats.TotalViolations++
		}
		re.stats.mu.Unlock()

		if violated {
			result := &EvaluationResult{
				RuleID:      rule.GetID(),
				RuleName:    rule.GetName(),
				Severity:    rule.GetSeverity(),
				Violated:    true,
				Score:       score,
				Message:     message,
				Sample:      sample,
				EvaluatedAt: time.Now(),
			}
			results = append(results, result)

			re.log.Warn("[规则违规] %s", message)
		}
	}

	return results
}

// GetStats 获取统计信息
func (re *RuleEngine) GetStats() EngineStats {
	re.stats.mu.RLock()
	defer re.stats.mu.RUnlock()
	return *re.stats
}

// EnableRule 启用规则
func (re *RuleEngine) EnableRule(ruleID string) error {
	rule, exists := re.GetRule(ruleID)
	if !exists {
		return errors.Newf(errors.ErrInvalidParam, "规则不存在: %s", ruleID)
	}
	rule.SetEnabled(true)
	return nil
}

// DisableRule 禁用规则
func (re *RuleEngine) DisableRule(ruleID string) error {
	rule, exists := re.GetRule(ruleID)
	if !exists {
		return errors.Newf(errors.ErrInvalidParam, "规则不存在: %s", ruleID)
	}
	rule.SetEnabled(false)
	return nil
}

// EvaluateConcurrent 并发评估规则（性能优化版本）
// 使用goroutine并发评估多条规则，适合规则数量较多的场景
func (re *RuleEngine) EvaluateConcurrent(sample *collector.DataSample) []*EvaluationResult {
	re.mu.RLock()
	enabledRules := make([]Rule, 0, len(re.rules))
	for _, rule := range re.rules {
		if rule.IsEnabled() {
			enabledRules = append(enabledRules, rule)
		}
	}
	re.mu.RUnlock()

	if len(enabledRules) == 0 {
		return nil
	}

	// 创建结果通道
	resultChan := make(chan *EvaluationResult, len(enabledRules))
	var wg sync.WaitGroup

	// 并发评估每条规则
	for _, rule := range enabledRules {
		wg.Add(1)
		go func(r Rule) {
			defer wg.Done()

			violated, score, message := r.Evaluate(sample)

			re.stats.mu.Lock()
			re.stats.TotalEvaluations++
			if violated {
				re.stats.TotalViolations++
			}
			re.stats.mu.Unlock()

			if violated {
				result := &EvaluationResult{
					RuleID:      r.GetID(),
					RuleName:    r.GetName(),
					Severity:    r.GetSeverity(),
					Violated:    true,
					Score:       score,
					Message:     message,
					Sample:      sample,
					EvaluatedAt: time.Now(),
				}
				resultChan <- result
			}
		}(rule)
	}

	// 等待所有评估完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	results := make([]*EvaluationResult, 0)
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}
