package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	controlDomain "go_ProFiBus/internal/domain/control"
	controlApp "go_ProFiBus/internal/application/control"
	"go_ProFiBus/pkg/interfaces"
)

// ControlHandler 设备控制API处理器
type ControlHandler struct {
	repo    interfaces.ControlRepository
	service *controlApp.ControlService
}

// NewControlHandler 创建设备控制处理器
func NewControlHandler(
	repo interfaces.ControlRepository,
	service *controlApp.ControlService,
) *ControlHandler {
	return &ControlHandler{
		repo:    repo,
		service: service,
	}
}

// CreateControlPolicy 创建控制策略
// POST /api/v1/control/policies
func (h *ControlHandler) CreateControlPolicy(c *gin.Context) {
	var req struct {
		Name            string                 `json:"name" binding:"required"`
		Description     string                 `json:"description"`
		Enabled         bool                   `json:"enabled"`
		Priority        int                    `json:"priority"`
		ConditionConfig map[string]interface{} `json:"condition_config"`
		ActionConfig    map[string]interface{} `json:"action_config" binding:"required"`
		CooldownSeconds int                    `json:"cooldown_seconds"`
		MaxExecutions   int                    `json:"max_executions"`
		Metadata        map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	policy := controlDomain.NewControlPolicy(uuid.New().String(), req.Name)
	policy.Description = req.Description
	policy.Enabled = req.Enabled
	policy.Priority = req.Priority
	if req.ConditionConfig != nil {
		policy.ConditionConfig = req.ConditionConfig
	}
	if req.ActionConfig != nil {
		policy.ActionConfig = req.ActionConfig
	}
	if req.CooldownSeconds > 0 {
		policy.CooldownSeconds = req.CooldownSeconds
	}
	if req.MaxExecutions > 0 {
		policy.MaxExecutions = req.MaxExecutions
	}
	if req.Metadata != nil {
		policy.Metadata = req.Metadata
	}

	if err := h.repo.CreateControlPolicy(c.Request.Context(), policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建控制策略失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// ListControlPolicies 列出控制策略
// GET /api/v1/control/policies
func (h *ControlHandler) ListControlPolicies(c *gin.Context) {
	filters := interfaces.ControlPolicyFilters{}

	if enabled := c.Query("enabled"); enabled != "" {
		enabledBool := enabled == "true"
		filters.Enabled = &enabledBool
	}

	if limit := c.Query("limit"); limit != "" {
		if limitInt, err := parseInt(limit); err == nil {
			filters.Limit = limitInt
		}
	}
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	if offset := c.Query("offset"); offset != "" {
		if offsetInt, err := parseInt(offset); err == nil {
			filters.Offset = offsetInt
		}
	}

	policies, err := h.repo.ListControlPolicies(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取控制策略列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies, "total": len(policies)})
}

// GetControlPolicy 获取控制策略
// GET /api/v1/control/policies/:id
func (h *ControlHandler) GetControlPolicy(c *gin.Context) {
	id := c.Param("id")

	policy, err := h.repo.GetControlPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "控制策略不存在"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// UpdateControlPolicy 更新控制策略
// PUT /api/v1/control/policies/:id
func (h *ControlHandler) UpdateControlPolicy(c *gin.Context) {
	id := c.Param("id")

	policy, err := h.repo.GetControlPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "控制策略不存在"})
		return
	}

	var req struct {
		Name            string                 `json:"name"`
		Description     string                 `json:"description"`
		Enabled         *bool                  `json:"enabled"`
		Priority        *int                   `json:"priority"`
		ConditionConfig map[string]interface{} `json:"condition_config"`
		ActionConfig    map[string]interface{} `json:"action_config"`
		CooldownSeconds *int                   `json:"cooldown_seconds"`
		MaxExecutions   *int                   `json:"max_executions"`
		Metadata        map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if req.Name != "" {
		policy.Name = req.Name
	}
	if req.Description != "" {
		policy.Description = req.Description
	}
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		policy.Priority = *req.Priority
	}
	if req.ConditionConfig != nil {
		policy.ConditionConfig = req.ConditionConfig
	}
	if req.ActionConfig != nil {
		policy.ActionConfig = req.ActionConfig
	}
	if req.CooldownSeconds != nil {
		policy.CooldownSeconds = *req.CooldownSeconds
	}
	if req.MaxExecutions != nil {
		policy.MaxExecutions = *req.MaxExecutions
	}
	if req.Metadata != nil {
		policy.Metadata = req.Metadata
	}
	policy.UpdatedAt = time.Now()

	if err := h.repo.UpdateControlPolicy(c.Request.Context(), policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新控制策略失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// DeleteControlPolicy 删除控制策略
// DELETE /api/v1/control/policies/:id
func (h *ControlHandler) DeleteControlPolicy(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.DeleteControlPolicy(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除控制策略失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "控制策略已删除"})
}

// CreateControlAction 创建控制动作
// POST /api/v1/control/actions
func (h *ControlHandler) CreateControlAction(c *gin.Context) {
	var req struct {
		PolicyID           string                 `json:"policy_id"`
		DeviceID           string                 `json:"device_id" binding:"required"`
		ActionType         string                 `json:"action_type" binding:"required"`
		Parameters         map[string]interface{} `json:"parameters"`
		Reason             string                 `json:"reason"`
		Severity           int                    `json:"severity"`
		RequireConfirmation bool                  `json:"require_confirmation"`
		Metadata           map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	actionType := controlDomain.ActionType(req.ActionType)
	if !isValidActionType(actionType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的动作类型"})
		return
	}

	action := controlDomain.NewControlAction(uuid.New().String(), req.DeviceID, actionType)
	action.PolicyID = req.PolicyID
	if req.Parameters != nil {
		action.Parameters = req.Parameters
	}
	action.Reason = req.Reason
	if req.Severity > 0 {
		action.Severity = req.Severity
	}
	action.RequireConfirmation = req.RequireConfirmation
	if req.Metadata != nil {
		action.Metadata = req.Metadata
	}

	// 获取用户信息（从认证中间件或请求头）
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "system"
	}
	userName := c.GetString("user_name")
	if userName == "" {
		userName = "System"
	}
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// 执行控制动作
	if err := h.service.ExecuteControlAction(c.Request.Context(), action, userID, userName, ipAddress, userAgent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "执行控制动作失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, action)
}

// ListControlActions 列出控制动作
// GET /api/v1/control/actions
func (h *ControlHandler) ListControlActions(c *gin.Context) {
	filters := interfaces.ControlActionFilters{}

	if policyID := c.Query("policy_id"); policyID != "" {
		filters.PolicyID = &policyID
	}
	if deviceID := c.Query("device_id"); deviceID != "" {
		filters.DeviceID = &deviceID
	}
	if actionType := c.Query("action_type"); actionType != "" {
		at := controlDomain.ActionType(actionType)
		filters.ActionType = &at
	}
	if status := c.Query("status"); status != "" {
		s := controlDomain.ActionStatus(status)
		filters.Status = &s
	}
	if executedBy := c.Query("executed_by"); executedBy != "" {
		filters.ExecutedBy = &executedBy
	}
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters.EndTime = &t
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if limitInt, err := parseInt(limit); err == nil {
			filters.Limit = limitInt
		}
	}
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	if offset := c.Query("offset"); offset != "" {
		if offsetInt, err := parseInt(offset); err == nil {
			filters.Offset = offsetInt
		}
	}

	actions, err := h.repo.ListControlActions(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取控制动作列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"actions": actions, "total": len(actions)})
}

// GetControlAction 获取控制动作
// GET /api/v1/control/actions/:id
func (h *ControlHandler) GetControlAction(c *gin.Context) {
	id := c.Param("id")

	action, err := h.repo.GetControlActionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "控制动作不存在"})
		return
	}

	c.JSON(http.StatusOK, action)
}

// ConfirmControlAction 确认控制动作
// POST /api/v1/control/actions/:id/confirm
func (h *ControlHandler) ConfirmControlAction(c *gin.Context) {
	id := c.Param("id")

	// 获取用户信息
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "system"
	}
	userName := c.GetString("user_name")
	if userName == "" {
		userName = "System"
	}
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	if err := h.service.ConfirmControlAction(c.Request.Context(), id, userID, userName, ipAddress, userAgent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "确认控制动作失败", "details": err.Error()})
		return
	}

	action, _ := h.repo.GetControlActionByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, action)
}

// ListAuditLogs 列出审计日志
// GET /api/v1/control/audit-logs
func (h *ControlHandler) ListAuditLogs(c *gin.Context) {
	filters := interfaces.AuditLogFilters{}

	if actionID := c.Query("action_id"); actionID != "" {
		filters.ActionID = &actionID
	}
	if userID := c.Query("user_id"); userID != "" {
		filters.UserID = &userID
	}
	if eventType := c.Query("event_type"); eventType != "" {
		et := controlDomain.AuditEventType(eventType)
		filters.EventType = &et
	}
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters.EndTime = &t
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if limitInt, err := parseInt(limit); err == nil {
			filters.Limit = limitInt
		}
	}
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	if offset := c.Query("offset"); offset != "" {
		if offsetInt, err := parseInt(offset); err == nil {
			filters.Offset = offsetInt
		}
	}

	logs, err := h.repo.GetAuditLogs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取审计日志失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": len(logs)})
}

// CreateControlPermission 创建控制权限
// POST /api/v1/control/permissions
func (h *ControlHandler) CreateControlPermission(c *gin.Context) {
	var req struct {
		UserID             string                 `json:"user_id" binding:"required"`
		ActionType         string                 `json:"action_type" binding:"required"`
		Enabled            bool                   `json:"enabled"`
		TargetDevices      []string               `json:"target_devices"`
		MaxSeverity        int                    `json:"max_severity"`
		RequireConfirmation bool                  `json:"require_confirmation"`
		AllowedTimeRanges  []controlDomain.TimeRange `json:"allowed_time_ranges"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	actionType := controlDomain.ActionType(req.ActionType)
	if !isValidActionType(actionType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的动作类型"})
		return
	}

	permission := controlDomain.NewControlPermission(uuid.New().String(), req.UserID, actionType)
	permission.Enabled = req.Enabled
	if req.TargetDevices != nil {
		permission.TargetDevices = req.TargetDevices
	}
	if req.MaxSeverity > 0 {
		permission.MaxSeverity = req.MaxSeverity
	}
	permission.RequireConfirmation = req.RequireConfirmation
	if req.AllowedTimeRanges != nil {
		permission.AllowedTimeRanges = req.AllowedTimeRanges
	}

	if err := h.repo.CreateControlPermission(c.Request.Context(), permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建控制权限失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

// ListControlPermissions 列出控制权限
// GET /api/v1/control/permissions
func (h *ControlHandler) ListControlPermissions(c *gin.Context) {
	filters := interfaces.ControlPermissionFilters{}

	if userID := c.Query("user_id"); userID != "" {
		filters.UserID = &userID
	}
	if actionType := c.Query("action_type"); actionType != "" {
		at := controlDomain.ActionType(actionType)
		filters.ActionType = &at
	}
	if enabled := c.Query("enabled"); enabled != "" {
		enabledBool := enabled == "true"
		filters.Enabled = &enabledBool
	}

	if limit := c.Query("limit"); limit != "" {
		if limitInt, err := parseInt(limit); err == nil {
			filters.Limit = limitInt
		}
	}
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	if offset := c.Query("offset"); offset != "" {
		if offsetInt, err := parseInt(offset); err == nil {
			filters.Offset = offsetInt
		}
	}

	permissions, err := h.repo.ListControlPermissions(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取控制权限列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions, "total": len(permissions)})
}

// UpdateControlPermission 更新控制权限
// PUT /api/v1/control/permissions/:id
func (h *ControlHandler) UpdateControlPermission(c *gin.Context) {
	id := c.Param("id")

	// 通过ListControlPermissions查找ID对应的权限
	filters := interfaces.ControlPermissionFilters{Limit: 1000}
	permissions, err := h.repo.ListControlPermissions(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取控制权限失败", "details": err.Error()})
		return
	}

	var permission *controlDomain.ControlPermission
	for _, p := range permissions {
		if p.ID == id {
			permission = p
			break
		}
	}

	if permission == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "控制权限不存在"})
		return
	}

	var req struct {
		Enabled            *bool                      `json:"enabled"`
		TargetDevices      []string                   `json:"target_devices"`
		MaxSeverity        *int                       `json:"max_severity"`
		RequireConfirmation *bool                     `json:"require_confirmation"`
		AllowedTimeRanges  []controlDomain.TimeRange  `json:"allowed_time_ranges"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if req.Enabled != nil {
		permission.Enabled = *req.Enabled
	}
	if req.TargetDevices != nil {
		permission.TargetDevices = req.TargetDevices
	}
	if req.MaxSeverity != nil {
		permission.MaxSeverity = *req.MaxSeverity
	}
	if req.RequireConfirmation != nil {
		permission.RequireConfirmation = *req.RequireConfirmation
	}
	if req.AllowedTimeRanges != nil {
		permission.AllowedTimeRanges = req.AllowedTimeRanges
	}
	permission.UpdatedAt = time.Now()

	if err := h.repo.UpdateControlPermission(c.Request.Context(), permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新控制权限失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permission)
}

// DeleteControlPermission 删除控制权限
// DELETE /api/v1/control/permissions/:id
func (h *ControlHandler) DeleteControlPermission(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.DeleteControlPermission(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除控制权限失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "控制权限已删除"})
}

// Helper functions

func isValidActionType(actionType controlDomain.ActionType) bool {
	validTypes := []controlDomain.ActionType{
		controlDomain.ActionTypeEmergencyStop,
		controlDomain.ActionTypeShutdown,
		controlDomain.ActionTypeStart,
		controlDomain.ActionTypePause,
		controlDomain.ActionTypeResume,
		controlDomain.ActionTypeSetValue,
		controlDomain.ActionTypeCallMethod,
		controlDomain.ActionTypeSendCommand,
		controlDomain.ActionTypeCustom,
	}

	for _, vt := range validTypes {
		if actionType == vt {
			return true
		}
	}
	return false
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
