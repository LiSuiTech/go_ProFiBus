package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	alertDomain "go_ProFiBus/internal/domain/alert"
	templateDomain "go_ProFiBus/internal/domain/rule_template"
	ruleTemplateApp "go_ProFiBus/internal/application/rule_template"
	"go_ProFiBus/internal/infrastructure/storage"
)

// RuleTemplateHandler 规则模板API处理器
type RuleTemplateHandler struct {
	templateRepo *storage.RuleTemplateRepositoryImpl
	testService  *ruleTemplateApp.RuleTestService
	alertRepo    *storage.AlertRepositoryImpl
}

// NewRuleTemplateHandler 创建规则模板处理器
func NewRuleTemplateHandler(
	templateRepo *storage.RuleTemplateRepositoryImpl,
	testService *ruleTemplateApp.RuleTestService,
	alertRepo *storage.AlertRepositoryImpl,
) *RuleTemplateHandler {
	return &RuleTemplateHandler{
		templateRepo: templateRepo,
		testService:  testService,
		alertRepo:    alertRepo,
	}
}

// ListTemplates 列出规则模板
// GET /api/v1/rule-templates
func (h *RuleTemplateHandler) ListTemplates(c *gin.Context) {
	filters := make(map[string]interface{})
	filters["enabled"] = true

	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}
	if ruleType := c.Query("rule_type"); ruleType != "" {
		filters["rule_type"] = ruleType
	}
	if tag := c.Query("tag"); tag != "" {
		filters["tag"] = tag
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	filters["limit"] = limit

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	filters["offset"] = offset

	templates, err := h.templateRepo.ListTemplates(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询模板列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":     len(templates),
		"templates": templates,
	})
}

// GetTemplate 获取模板详情
// GET /api/v1/rule-templates/:id
func (h *RuleTemplateHandler) GetTemplate(c *gin.Context) {
	templateID := c.Param("id")
	template, err := h.templateRepo.GetTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// CreateRuleFromTemplate 从模板创建告警规则
// POST /api/v1/rule-templates/:id/create-rule
func (h *RuleTemplateHandler) CreateRuleFromTemplate(c *gin.Context) {
	templateID := c.Param("id")
	template, err := h.templateRepo.GetTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在", "details": err.Error()})
		return
	}

	var req struct {
		Name            string                 `json:"name" binding:"required"`
		Description     string                 `json:"description"`
		Level           string                 `json:"level" binding:"required"`
		Variables       map[string]interface{} `json:"variables"` // 模板变量值
		Enabled         bool                   `json:"enabled"`
		CooldownSeconds int                    `json:"cooldown_seconds"`
		MaxExecutions   int                    `json:"max_executions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 应用变量到条件模板
	condition := h.applyVariables(template.ConditionTemplate, req.Variables)

	// 确定告警级别（从请求或模板输出配置）
	level := req.Level
	if level == "" {
		if levelFromTemplate, ok := template.OutputConfig["level"].(string); ok {
			level = levelFromTemplate
		} else {
			level = "warning"
		}
	}

	alertLevel := alertDomain.AlertLevel(level)
	if alertLevel != alertDomain.AlertLevelInfo &&
		alertLevel != alertDomain.AlertLevelWarning &&
		alertLevel != alertDomain.AlertLevelError &&
		alertLevel != alertDomain.AlertLevelCritical {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的告警级别"})
		return
	}

	// 创建告警规则
	rule := alertDomain.NewAlertRule(uuid.New().String(), req.Name, condition, alertLevel)
	rule.Description = req.Description
	rule.Enabled = req.Enabled
	if req.CooldownSeconds > 0 {
		rule.CooldownSeconds = req.CooldownSeconds
	}
	if req.MaxExecutions > 0 {
		rule.MaxExecutions = req.MaxExecutions
	}

	if err := h.alertRepo.CreateAlertRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建告警规则失败", "details": err.Error()})
		return
	}

	// 增加模板使用次数
	h.templateRepo.IncrementUsage(c.Request.Context(), templateID)

	c.JSON(http.StatusCreated, rule)
}

// TestTemplate 测试模板
// POST /api/v1/rule-templates/:id/test
func (h *RuleTemplateHandler) TestTemplate(c *gin.Context) {
	templateID := c.Param("id")

	var req struct {
		Variables map[string]interface{} `json:"variables"` // 模板变量值
		TestData  map[string]interface{} `json:"test_data" binding:"required"` // 测试数据
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 测试模板
	testResult, err := h.testService.TestTemplate(c.Request.Context(), templateID, req.Variables, req.TestData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "测试失败", "details": err.Error()})
		return
	}

	// 保存测试结果
	if err := h.templateRepo.SaveTestResult(c.Request.Context(), testResult); err != nil {
		// 记录错误但不影响返回
		c.JSON(http.StatusOK, gin.H{
			"test_result": testResult,
			"warning":     "保存测试结果失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"test_result": testResult,
	})
}

// TestRule 测试规则配置
// POST /api/v1/rule-templates/test-rule
func (h *RuleTemplateHandler) TestRule(c *gin.Context) {
	var req struct {
		RuleConfig map[string]interface{} `json:"rule_config" binding:"required"` // 规则配置
		TestData   map[string]interface{} `json:"test_data" binding:"required"`  // 测试数据
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 测试规则
	testResult, err := h.testService.TestRule(c.Request.Context(), req.RuleConfig, req.TestData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "测试失败", "details": err.Error()})
		return
	}

	// 保存测试结果
	if err := h.templateRepo.SaveTestResult(c.Request.Context(), testResult); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"test_result": testResult,
			"warning":     "保存测试结果失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"test_result": testResult,
	})
}

// ListTestResults 列出测试结果
// GET /api/v1/rule-templates/test-results
func (h *RuleTemplateHandler) ListTestResults(c *gin.Context) {
	filters := make(map[string]interface{})

	if ruleID := c.Query("rule_id"); ruleID != "" {
		filters["rule_id"] = ruleID
	}
	if templateID := c.Query("template_id"); templateID != "" {
		filters["template_id"] = templateID
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}
	filters["limit"] = limit

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	filters["offset"] = offset

	results, err := h.templateRepo.ListTestResults(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询测试结果失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":        len(results),
		"test_results": results,
	})
}

// applyVariables 应用变量到模板
func (h *RuleTemplateHandler) applyVariables(template map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range template {
		result[k] = h.replaceVariables(v, variables)
	}
	return result
}

// replaceVariables 替换变量占位符
func (h *RuleTemplateHandler) replaceVariables(value interface{}, variables map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// 替换 ${variable_name} 格式的变量
		if len(v) > 4 && v[:2] == "${" && v[len(v)-1:] == "}" {
			varName := v[2 : len(v)-1]
			if varValue, ok := variables[varName]; ok {
				return varValue
			}
		}
		return v
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = h.replaceVariables(val, variables)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = h.replaceVariables(val, variables)
		}
		return result
	default:
		return v
	}
}
