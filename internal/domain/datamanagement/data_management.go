package datamanagement

import (
	"time"
)

// CleaningRule 数据清洗规则
type CleaningRule struct {
	ID          string
	Name        string
	Description string
	RuleType    CleaningRuleType
	Enabled     bool
	Config      map[string]interface{}
	Priority    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CleaningRuleType 清洗规则类型
type CleaningRuleType string

const (
	CleaningRuleTypeDeduplicate  CleaningRuleType = "deduplicate"   // 去重
	CleaningRuleTypeOutlierFilter CleaningRuleType = "outlier_filter" // 异常值过滤
	CleaningRuleTypeMissingFill   CleaningRuleType = "missing_fill"   // 缺失值填充
	CleaningRuleTypeNormalize     CleaningRuleType = "normalize"       // 标准化
	CleaningRuleTypeSmooth        CleaningRuleType = "smooth"         // 平滑处理
	CleaningRuleTypeValidate      CleaningRuleType = "validate"       // 数据验证
)

// NewCleaningRule 创建清洗规则
func NewCleaningRule(id, name string, ruleType CleaningRuleType) *CleaningRule {
	return &CleaningRule{
		ID:        id,
		Name:      name,
		RuleType:  ruleType,
		Enabled:   true,
		Config:    make(map[string]interface{}),
		Priority:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ArchivePolicy 数据归档策略
type ArchivePolicy struct {
	ID                string
	Name              string
	Description       string
	SourceType        string
	SourceID          string
	RetentionDays     int
	ArchiveAfterDays  int
	CompressionEnabled bool
	ArchiveLocation   string
	Enabled           bool
	LastRunAt         *time.Time
	NextRunAt         *time.Time
	RunIntervalHours  int
	Metadata          map[string]interface{}
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewArchivePolicy 创建归档策略
func NewArchivePolicy(id, name, sourceType string) *ArchivePolicy {
	now := time.Now()
	nextRun := now.Add(24 * time.Hour)
	return &ArchivePolicy{
		ID:                id,
		Name:              name,
		SourceType:        sourceType,
		RetentionDays:     365,
		ArchiveAfterDays:  30,
		CompressionEnabled: true,
		Enabled:           true,
		NextRunAt:         &nextRun,
		RunIntervalHours:  24,
		Metadata:          make(map[string]interface{}),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// ShouldRun 检查是否应该执行
func (p *ArchivePolicy) ShouldRun() bool {
	if !p.Enabled {
		return false
	}
	if p.NextRunAt == nil {
		return true
	}
	return time.Now().After(*p.NextRunAt)
}

// UpdateNextRun 更新下次执行时间
func (p *ArchivePolicy) UpdateNextRun() {
	now := time.Now()
	p.LastRunAt = &now
	nextRun := now.Add(time.Duration(p.RunIntervalHours) * time.Hour)
	p.NextRunAt = &nextRun
	p.UpdatedAt = now
}

// ArchiveRecord 数据归档记录
type ArchiveRecord struct {
	ID           string
	PolicyID     string
	SourceType   string
	SourceID     string
	StartTime    time.Time
	EndTime      time.Time
	RecordCount  int64
	ArchiveSize  int64
	ArchivePath  string
	Status       ArchiveStatus
	ErrorMessage string
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// ArchiveStatus 归档状态
type ArchiveStatus string

const (
	ArchiveStatusPending   ArchiveStatus = "pending"
	ArchiveStatusRunning   ArchiveStatus = "running"
	ArchiveStatusCompleted ArchiveStatus = "completed"
	ArchiveStatusFailed    ArchiveStatus = "failed"
)

// NewArchiveRecord 创建归档记录
func NewArchiveRecord(id, policyID, sourceType string) *ArchiveRecord {
	return &ArchiveRecord{
		ID:         id,
		PolicyID:   policyID,
		SourceType: sourceType,
		StartTime:  time.Now(),
		Status:     ArchiveStatusPending,
		CreatedAt:  time.Now(),
	}
}

// Complete 完成归档
func (r *ArchiveRecord) Complete(recordCount, archiveSize int64, archivePath string) {
	now := time.Now()
	r.EndTime = now
	r.RecordCount = recordCount
	r.ArchiveSize = archiveSize
	r.ArchivePath = archivePath
	r.Status = ArchiveStatusCompleted
	r.CompletedAt = &now
}

// Fail 归档失败
func (r *ArchiveRecord) Fail(errorMessage string) {
	now := time.Now()
	r.EndTime = now
	r.Status = ArchiveStatusFailed
	r.ErrorMessage = errorMessage
	r.CompletedAt = &now
}

// CleaningRecord 数据清洗记录
type CleaningRecord struct {
	ID            string
	RuleID        string
	SourceType    string
	SourceID      string
	ProcessedCount int64
	CleanedCount  int64
	RemovedCount  int64
	FilledCount   int64
	StartTime     time.Time
	EndTime       *time.Time
	Status        CleaningStatus
	ErrorMessage  string
	CreatedAt     time.Time
}

// CleaningStatus 清洗状态
type CleaningStatus string

const (
	CleaningStatusPending   CleaningStatus = "pending"
	CleaningStatusRunning   CleaningStatus = "running"
	CleaningStatusCompleted CleaningStatus = "completed"
	CleaningStatusFailed    CleaningStatus = "failed"
)

// NewCleaningRecord 创建清洗记录
func NewCleaningRecord(id, ruleID, sourceType string) *CleaningRecord {
	return &CleaningRecord{
		ID:         id,
		RuleID:     ruleID,
		SourceType: sourceType,
		StartTime:  time.Now(),
		Status:     CleaningStatusPending,
		CreatedAt:  time.Now(),
	}
}

// Complete 完成清洗
func (r *CleaningRecord) Complete(processed, cleaned, removed, filled int64) {
	now := time.Now()
	r.ProcessedCount = processed
	r.CleanedCount = cleaned
	r.RemovedCount = removed
	r.FilledCount = filled
	r.Status = CleaningStatusCompleted
	r.EndTime = &now
}

// Fail 清洗失败
func (r *CleaningRecord) Fail(errorMessage string) {
	now := time.Now()
	r.Status = CleaningStatusFailed
	r.ErrorMessage = errorMessage
	r.EndTime = &now
}

// LifecycleConfig 数据生命周期配置
type LifecycleConfig struct {
	ID                 string
	SourceType         string
	SourceID           string
	HotStorageDays     int // 热存储天数
	WarmStorageDays    int // 温存储天数
	ColdStorageDays    int // 冷存储天数
	DeleteAfterDays    *int // 删除天数（nil=不删除）
	CompressionAfterDays int // 压缩天数
	Enabled            bool
	Metadata           map[string]interface{}
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// NewLifecycleConfig 创建生命周期配置
func NewLifecycleConfig(id, sourceType, sourceID string) *LifecycleConfig {
	return &LifecycleConfig{
		ID:                 id,
		SourceType:         sourceType,
		SourceID:           sourceID,
		HotStorageDays:     7,
		WarmStorageDays:    30,
		ColdStorageDays:    365,
		CompressionAfterDays: 30,
		Enabled:            true,
		Metadata:           make(map[string]interface{}),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

// GetStorageType 根据数据年龄获取存储类型
func (c *LifecycleConfig) GetStorageType(dataAgeDays int) string {
	if dataAgeDays <= c.HotStorageDays {
		return "hot"
	}
	if dataAgeDays <= c.WarmStorageDays {
		return "warm"
	}
	if dataAgeDays <= c.ColdStorageDays {
		return "cold"
	}
	return "archived"
}

// ShouldDelete 检查是否应该删除
func (c *LifecycleConfig) ShouldDelete(dataAgeDays int) bool {
	if c.DeleteAfterDays == nil {
		return false
	}
	return dataAgeDays > *c.DeleteAfterDays
}

// ShouldCompress 检查是否应该压缩
func (c *LifecycleConfig) ShouldCompress(dataAgeDays int) bool {
	return dataAgeDays > c.CompressionAfterDays
}
