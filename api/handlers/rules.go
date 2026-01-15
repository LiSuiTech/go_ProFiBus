package handlers

import (
	"go_ProFiBus/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RuleHandler 规则管理处理器
type RuleHandler struct {
	store *storage.PostgresStore
}

// NewRuleHandler 创建规则处理器
func NewRuleHandler(store *storage.PostgresStore) *RuleHandler {
	return &RuleHandler{
		store: store,
	}
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"` // threshold, statistical, pattern, similarity, ml
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config" binding:"required"`
	Enabled     *bool                  `json:"enabled"`
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Enabled     *bool                  `json:"enabled"`
}

// GetRules 获取规则列表
// @Summary 获取所有规则
// @Tags rules
// @Accept json
// @Produce json
// @Param enabled query boolean false "是否只返回启用的规则"
// @Param type query string false "规则类型过滤"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/rules [get]
func (h *RuleHandler) GetRules(c *gin.Context) {
	// 解析查询参数
	var enabledFilter *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		enabledFilter = &enabled
	}

	typeFilter := c.Query("type")

	// 查询规则
	rules, err := h.store.GetRules(enabledFilter, typeFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询规则失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(rules),
		"rules": rules,
	})
}

// GetRule 获取单个规则详情
// @Summary 获取规则详情
// @Tags rules
// @Accept json
// @Produce json
// @Param rule_id path string true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/rules/{rule_id} [get]
func (h *RuleHandler) GetRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则ID不能为空"})
		return
	}

	// 查询规则
	rule, err := h.store.GetRuleByID(ruleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询规则失败", "details": err.Error()})
		return
	}

	if rule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// CreateRule 创建新规则
// @Summary 创建规则
// @Tags rules
// @Accept json
// @Produce json
// @Param rule body CreateRuleRequest true "规则信息"
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/rules [post]
func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req CreateRuleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 验证规则类型
	validTypes := map[string]bool{
		"threshold":   true,
		"statistical": true,
		"pattern":     true,
		"similarity":  true,
		"ml":          true,
	}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的规则类型，必须是: threshold, statistical, pattern, similarity, ml"})
		return
	}

	// 设置默认启用状态
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	// 创建规则
	ruleID, err := h.store.CreateRule(req.Name, req.Type, req.Description, req.Config, enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建规则失败", "details": err.Error()})
		return
	}

	// 返回创建的规则
	rule, err := h.store.GetRuleByID(ruleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询创建的规则失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// UpdateRule 更新规则
// @Summary 更新规则
// @Tags rules
// @Accept json
// @Produce json
// @Param rule_id path string true "规则ID"
// @Param rule body UpdateRuleRequest true "更新内容"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/rules/{rule_id} [put]
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则ID不能为空"})
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Config != nil {
		updates["config"] = req.Config
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	// 更新规则
	if err := h.store.UpdateRule(ruleID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新规则失败", "details": err.Error()})
		return
	}

	// 返回更新后的规则
	rule, err := h.store.GetRuleByID(ruleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询更新后的规则失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule 删除规则
// @Summary 删除规则
// @Tags rules
// @Accept json
// @Produce json
// @Param rule_id path string true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/rules/{rule_id} [delete]
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则ID不能为空"})
		return
	}

	// 删除规则
	if err := h.store.DeleteRule(ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除规则失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "规则删除成功",
		"rule_id": ruleID,
	})
}
