package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dataManagementDomain "go_ProFiBus/internal/domain/datamanagement"
	dataManagementApp "go_ProFiBus/internal/application/datamanagement"
	"go_ProFiBus/pkg/interfaces"
)

// DataManagementHandler 数据管理API处理器
type DataManagementHandler struct {
	repo            interfaces.DataManagementRepository
	cleaningService *dataManagementApp.DataCleaningService
	archiveService  *dataManagementApp.DataArchiveService
}

// NewDataManagementHandler 创建数据管理处理器
func NewDataManagementHandler(
	repo interfaces.DataManagementRepository,
	cleaningService *dataManagementApp.DataCleaningService,
	archiveService *dataManagementApp.DataArchiveService,
) *DataManagementHandler {
	return &DataManagementHandler{
		repo:            repo,
		cleaningService: cleaningService,
		archiveService:  archiveService,
	}
}

// CreateCleaningRule 创建清洗规则
// POST /api/v1/data-management/cleaning-rules
func (h *DataManagementHandler) CreateCleaningRule(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		Description string                 `json:"description"`
		RuleType    string                 `json:"rule_type" binding:"required"`
		Config      map[string]interface{} `json:"config"`
		Priority    int                    `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	ruleType := dataManagementDomain.CleaningRuleType(req.RuleType)
	if ruleType != dataManagementDomain.CleaningRuleTypeDeduplicate &&
		ruleType != dataManagementDomain.CleaningRuleTypeOutlierFilter &&
		ruleType != dataManagementDomain.CleaningRuleTypeMissingFill &&
		ruleType != dataManagementDomain.CleaningRuleTypeNormalize &&
		ruleType != dataManagementDomain.CleaningRuleTypeSmooth &&
		ruleType != dataManagementDomain.CleaningRuleTypeValidate {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的规则类型"})
		return
	}

	rule := dataManagementDomain.NewCleaningRule(uuid.New().String(), req.Name, ruleType)
	rule.Description = req.Description
	if req.Config != nil {
		rule.Config = req.Config
	}
	rule.Priority = req.Priority

	if err := h.repo.CreateCleaningRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建清洗规则失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListCleaningRules 列出清洗规则
// GET /api/v1/data-management/cleaning-rules
func (h *DataManagementHandler) ListCleaningRules(c *gin.Context) {
	filters := interfaces.CleaningRuleFilters{}

	if typeStr := c.Query("type"); typeStr != "" {
		ruleType := dataManagementDomain.CleaningRuleType(typeStr)
		filters.RuleType = &ruleType
	}
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		filters.Enabled = &enabled
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	filters.Limit = limit

	rules, err := h.repo.ListCleaningRules(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询清洗规则列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(rules),
		"rules": rules,
	})
}

// CreateArchivePolicy 创建归档策略
// POST /api/v1/data-management/archive-policies
func (h *DataManagementHandler) CreateArchivePolicy(c *gin.Context) {
	var req struct {
		Name              string `json:"name" binding:"required"`
		Description       string `json:"description"`
		SourceType        string `json:"source_type" binding:"required"`
		SourceID          string `json:"source_id"`
		RetentionDays     int    `json:"retention_days"`
		ArchiveAfterDays  int    `json:"archive_after_days"`
		CompressionEnabled bool  `json:"compression_enabled"`
		ArchiveLocation   string `json:"archive_location"`
		RunIntervalHours  int    `json:"run_interval_hours"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	policy := dataManagementDomain.NewArchivePolicy(uuid.New().String(), req.Name, req.SourceType)
	policy.Description = req.Description
	policy.SourceID = req.SourceID
	if req.RetentionDays > 0 {
		policy.RetentionDays = req.RetentionDays
	}
	if req.ArchiveAfterDays > 0 {
		policy.ArchiveAfterDays = req.ArchiveAfterDays
	}
	if req.RunIntervalHours > 0 {
		policy.RunIntervalHours = req.RunIntervalHours
	}
	policy.CompressionEnabled = req.CompressionEnabled
	policy.ArchiveLocation = req.ArchiveLocation

	if err := h.repo.CreateArchivePolicy(c.Request.Context(), policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建归档策略失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// ListArchivePolicies 列出归档策略
// GET /api/v1/data-management/archive-policies
func (h *DataManagementHandler) ListArchivePolicies(c *gin.Context) {
	filters := interfaces.ArchivePolicyFilters{}

	if sourceType := c.Query("source_type"); sourceType != "" {
		filters.SourceType = &sourceType
	}
	if sourceID := c.Query("source_id"); sourceID != "" {
		filters.SourceID = &sourceID
	}
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		filters.Enabled = &enabled
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	filters.Limit = limit

	policies, err := h.repo.ListArchivePolicies(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询归档策略列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(policies),
		"policies": policies,
	})
}

// ExecuteArchive 执行归档
// POST /api/v1/data-management/archive-policies/:id/execute
func (h *DataManagementHandler) ExecuteArchive(c *gin.Context) {
	policyID := c.Param("id")

	if err := h.archiveService.ExecuteArchive(c.Request.Context(), policyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "执行归档失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "归档执行成功"})
}

// GetArchiveStats 获取归档统计
// GET /api/v1/data-management/archive-policies/:id/stats
func (h *DataManagementHandler) GetArchiveStats(c *gin.Context) {
	policyID := c.Param("id")

	startStr := c.DefaultQuery("start", "")
	endStr := c.DefaultQuery("end", "")

	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始时间格式"})
			return
		}
	} else {
		start = time.Now().AddDate(0, 0, -30) // 默认30天前
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束时间格式"})
			return
		}
	} else {
		end = time.Now()
	}

	stats, err := h.archiveService.GetArchiveStats(c.Request.Context(), policyID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询归档统计失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ListArchiveRecords 列出归档记录
// GET /api/v1/data-management/archive-records
func (h *DataManagementHandler) ListArchiveRecords(c *gin.Context) {
	filters := interfaces.ArchiveRecordFilters{}

	if policyID := c.Query("policy_id"); policyID != "" {
		filters.PolicyID = &policyID
	}
	if sourceType := c.Query("source_type"); sourceType != "" {
		filters.SourceType = &sourceType
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status := dataManagementDomain.ArchiveStatus(statusStr)
		filters.Status = &status
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	filters.Limit = limit

	records, err := h.repo.ListArchiveRecords(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询归档记录列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(records),
		"records": records,
	})
}

// CreateLifecycleConfig 创建生命周期配置
// POST /api/v1/data-management/lifecycle-configs
func (h *DataManagementHandler) CreateLifecycleConfig(c *gin.Context) {
	var req struct {
		SourceType         string `json:"source_type" binding:"required"`
		SourceID           string `json:"source_id" binding:"required"`
		HotStorageDays     int    `json:"hot_storage_days"`
		WarmStorageDays    int    `json:"warm_storage_days"`
		ColdStorageDays    int    `json:"cold_storage_days"`
		DeleteAfterDays    *int   `json:"delete_after_days"`
		CompressionAfterDays int  `json:"compression_after_days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	config := dataManagementDomain.NewLifecycleConfig(uuid.New().String(), req.SourceType, req.SourceID)
	if req.HotStorageDays > 0 {
		config.HotStorageDays = req.HotStorageDays
	}
	if req.WarmStorageDays > 0 {
		config.WarmStorageDays = req.WarmStorageDays
	}
	if req.ColdStorageDays > 0 {
		config.ColdStorageDays = req.ColdStorageDays
	}
	if req.DeleteAfterDays != nil {
		config.DeleteAfterDays = req.DeleteAfterDays
	}
	if req.CompressionAfterDays > 0 {
		config.CompressionAfterDays = req.CompressionAfterDays
	}

	if err := h.repo.CreateLifecycleConfig(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建生命周期配置失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetLifecycleConfig 获取生命周期配置
// GET /api/v1/data-management/lifecycle-configs/:source_type/:source_id
func (h *DataManagementHandler) GetLifecycleConfig(c *gin.Context) {
	sourceType := c.Param("source_type")
	sourceID := c.Param("source_id")

	config, err := h.repo.GetLifecycleConfig(c.Request.Context(), sourceType, sourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "生命周期配置不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// CleanData 清洗数据
// POST /api/v1/data-management/clean
func (h *DataManagementHandler) CleanData(c *gin.Context) {
	var req struct {
		SourceType string                 `json:"source_type" binding:"required"`
		SourceID   string                 `json:"source_id" binding:"required"`
		Data       map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	cleanedData, wasCleaned, err := h.cleaningService.CleanData(c.Request.Context(), req.SourceType, req.SourceID, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据清洗失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cleaned_data": cleanedData,
		"was_cleaned":  wasCleaned,
	})
}
