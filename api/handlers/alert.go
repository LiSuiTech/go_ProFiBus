package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	alertDomain "go_ProFiBus/internal/domain/alert"
	"go_ProFiBus/pkg/interfaces"
)

// AlertHandler 告警管理API处理器
type AlertHandler struct {
	repo interfaces.AlertRepository
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler(repo interfaces.AlertRepository) *AlertHandler {
	return &AlertHandler{repo: repo}
}

// ListAlerts 获取告警列表
// GET /api/v1/alerts
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	filters := interfaces.AlertFilters{}

	// 解析查询参数
	if ruleID := c.Query("rule_id"); ruleID != "" {
		filters.RuleID = &ruleID
	}
	if deviceID := c.Query("device_id"); deviceID != "" {
		filters.DeviceID = &deviceID
	}
	if channelID := c.Query("channel_id"); channelID != "" {
		filters.ChannelID = &channelID
	}
	if levelStr := c.Query("level"); levelStr != "" {
		level := alertDomain.AlertLevel(levelStr)
		filters.Level = &level
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status := alertDomain.AlertStatus(statusStr)
		filters.Status = &status
	}
	if startStr := c.Query("start"); startStr != "" {
		if start, err := time.Parse(time.RFC3339, startStr); err == nil {
			filters.StartTime = &start
		}
	}
	if endStr := c.Query("end"); endStr != "" {
		if end, err := time.Parse(time.RFC3339, endStr); err == nil {
			filters.EndTime = &end
		}
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	filters.Limit = limit

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	filters.Offset = offset

	alerts, err := h.repo.ListAlerts(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询告警列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(alerts),
		"alerts": alerts,
	})
}

// GetAlert 获取告警详情
// GET /api/v1/alerts/:id
func (h *AlertHandler) GetAlert(c *gin.Context) {
	alertID := c.Param("id")

	alert, err := h.repo.GetAlertByID(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "告警不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// AcknowledgeAlert 确认告警
// POST /api/v1/alerts/:id/acknowledge
func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if err := h.repo.AcknowledgeAlert(c.Request.Context(), alertID, req.AcknowledgedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "确认告警失败", "details": err.Error()})
		return
	}

	alert, _ := h.repo.GetAlertByID(c.Request.Context(), alertID)
	c.JSON(http.StatusOK, alert)
}

// ResolveAlert 解决告警
// POST /api/v1/alerts/:id/resolve
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")

	var req struct {
		ResolvedBy string `json:"resolved_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if err := h.repo.ResolveAlert(c.Request.Context(), alertID, req.ResolvedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解决告警失败", "details": err.Error()})
		return
	}

	alert, _ := h.repo.GetAlertByID(c.Request.Context(), alertID)
	c.JSON(http.StatusOK, alert)
}

// GetAlertStats 获取告警统计
// GET /api/v1/alerts/stats
func (h *AlertHandler) GetAlertStats(c *gin.Context) {
	filters := interfaces.AlertFilters{}

	if deviceID := c.Query("device_id"); deviceID != "" {
		filters.DeviceID = &deviceID
	}
	if startStr := c.Query("start"); startStr != "" {
		if start, err := time.Parse(time.RFC3339, startStr); err == nil {
			filters.StartTime = &start
		}
	}
	if endStr := c.Query("end"); endStr != "" {
		if end, err := time.Parse(time.RFC3339, endStr); err == nil {
			filters.EndTime = &end
		}
	}

	stats, err := h.repo.GetAlertStats(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取告警统计失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CreateAlertRule 创建告警规则
// POST /api/v1/alerts/rules
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
	var req struct {
		Name            string                 `json:"name" binding:"required"`
		Description     string                 `json:"description"`
		Condition       map[string]interface{} `json:"condition" binding:"required"`
		Level           string                 `json:"level" binding:"required"`
		Enabled         bool                   `json:"enabled"`
		CooldownSeconds int                    `json:"cooldown_seconds"`
		MaxExecutions   int                    `json:"max_executions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	level := alertDomain.AlertLevel(req.Level)
	if level != alertDomain.AlertLevelInfo &&
		level != alertDomain.AlertLevelWarning &&
		level != alertDomain.AlertLevelError &&
		level != alertDomain.AlertLevelCritical {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的告警级别"})
		return
	}

	rule := alertDomain.NewAlertRule(uuid.New().String(), req.Name, req.Condition, level)
	rule.Description = req.Description
	rule.Enabled = req.Enabled
	if req.CooldownSeconds > 0 {
		rule.CooldownSeconds = req.CooldownSeconds
	}
	if req.MaxExecutions > 0 {
		rule.MaxExecutions = req.MaxExecutions
	}

	if err := h.repo.CreateAlertRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建告警规则失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListAlertRules 列出告警规则
// GET /api/v1/alerts/rules
func (h *AlertHandler) ListAlertRules(c *gin.Context) {
	var enabled *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		if e, err := strconv.ParseBool(enabledStr); err == nil {
			enabled = &e
		}
	}

	rules, err := h.repo.ListAlertRules(c.Request.Context(), enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询告警规则列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(rules),
		"rules": rules,
	})
}

// GetAlertRule 获取告警规则详情
// GET /api/v1/alerts/rules/:id
func (h *AlertHandler) GetAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	rule, err := h.repo.GetAlertRuleByID(c.Request.Context(), ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "告警规则不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateAlertRule 更新告警规则
// PUT /api/v1/alerts/rules/:id
func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	rule, err := h.repo.GetAlertRuleByID(c.Request.Context(), ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "告警规则不存在", "details": err.Error()})
		return
	}

	var req struct {
		Name            *string                `json:"name"`
		Description     *string                `json:"description"`
		Condition       map[string]interface{} `json:"condition"`
		Level           *string                `json:"level"`
		Enabled         *bool                  `json:"enabled"`
		CooldownSeconds *int                   `json:"cooldown_seconds"`
		MaxExecutions   *int                   `json:"max_executions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Condition != nil {
		rule.Condition = req.Condition
	}
	if req.Level != nil {
		level := alertDomain.AlertLevel(*req.Level)
		rule.Level = level
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.CooldownSeconds != nil {
		rule.CooldownSeconds = *req.CooldownSeconds
	}
	if req.MaxExecutions != nil {
		rule.MaxExecutions = *req.MaxExecutions
	}

	if err := h.repo.UpdateAlertRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新告警规则失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteAlertRule 删除告警规则
// DELETE /api/v1/alerts/rules/:id
func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	if err := h.repo.DeleteAlertRule(c.Request.Context(), ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除告警规则失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "告警规则删除成功"})
}
