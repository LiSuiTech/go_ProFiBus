package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go_ProFiBus/internal/application/workflow"
	"go_ProFiBus/internal/infrastructure/storage"
)

// WorkflowHandler 工作流API处理器
type WorkflowHandler struct {
	engine         *workflow.Engine
	repo           *storage.WorkflowRepositoryImpl
	templateRepo   *storage.WorkflowTemplateRepositoryImpl
}

// NewWorkflowHandler 创建工作流处理器
func NewWorkflowHandler(engine *workflow.Engine, repo *storage.WorkflowRepositoryImpl) *WorkflowHandler {
	return &WorkflowHandler{
		engine: engine,
		repo:   repo,
	}
}

// SetTemplateRepository 设置模板仓储（用于模板相关功能）
func (h *WorkflowHandler) SetTemplateRepository(templateRepo *storage.WorkflowTemplateRepositoryImpl) {
	h.templateRepo = templateRepo
}

// ListWorkflows 列出工作流
// GET /api/v1/workflows
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	workflows, err := h.repo.List(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询工作流列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":     len(workflows),
		"workflows": workflows,
	})
}

// CreateWorkflow 创建工作流
// POST /api/v1/workflows
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	var wf workflow.Workflow
	if err := c.ShouldBindJSON(&wf); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if wf.ID == "" {
		wf.ID = uuid.New().String()
	}

	if err := wf.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作流验证失败", "details": err.Error()})
		return
	}

	if err := h.repo.Save(c.Request.Context(), &wf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建工作流失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wf)
}

// GetWorkflow 获取工作流详情
// GET /api/v1/workflows/:id
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	id := c.Param("id")

	wf, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作流不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wf)
}

// UpdateWorkflow 更新工作流
// PUT /api/v1/workflows/:id
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	id := c.Param("id")

	wf, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作流不存在", "details": err.Error()})
		return
	}

	var updateData workflow.Workflow
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 更新字段
	wf.Name = updateData.Name
	wf.Description = updateData.Description
	wf.Nodes = updateData.Nodes
	wf.Edges = updateData.Edges
	wf.Variables = updateData.Variables
	wf.Status = updateData.Status

	if err := wf.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作流验证失败", "details": err.Error()})
		return
	}

	if err := h.repo.Save(c.Request.Context(), wf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新工作流失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wf)
}

// DeleteWorkflow 删除工作流
// DELETE /api/v1/workflows/:id
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除工作流失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "工作流已删除"})
}

// ExecuteWorkflow 执行工作流
// POST /api/v1/workflows/:id/execute
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}
	c.ShouldBindJSON(&req)

	executionID, err := h.engine.Execute(c.Request.Context(), id, req.Variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "执行工作流失败", "details": err.Error()})
		return
	}

	// 获取执行实例
	execution, err := h.repo.GetExecution(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"execution_id": executionID, "message": "执行已启动"})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// ListExecutions 列出工作流执行历史
// GET /api/v1/workflows/:id/executions
func (h *WorkflowHandler) ListExecutions(c *gin.Context) {
	workflowID := c.Param("id")

	executions, err := h.repo.ListExecutions(c.Request.Context(), workflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询执行历史失败", "details": err.Error()})
		return
	}

	// 应用 limit（如果提供）
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	if len(executions) > limit {
		executions = executions[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"count":      len(executions),
		"executions": executions,
	})
}

// GetExecution 获取执行详情
// GET /api/v1/workflows/:id/executions/:execution_id
func (h *WorkflowHandler) GetExecution(c *gin.Context) {
	executionID := c.Param("execution_id")

	execution, err := h.repo.GetExecution(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "执行记录不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// CancelExecution 取消执行
// POST /api/v1/workflows/:id/executions/:execution_id/cancel
func (h *WorkflowHandler) CancelExecution(c *gin.Context) {
	executionID := c.Param("execution_id")

	if err := h.engine.CancelExecution(c.Request.Context(), executionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "取消执行失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "执行已取消"})
}

// ========== 工作流模板相关 API ==========

// ListTemplates 列出工作流模板
// GET /api/v1/workflows/templates
func (h *WorkflowHandler) ListTemplates(c *gin.Context) {
	if h.templateRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "模板功能未启用"})
		return
	}

	filters := make(map[string]interface{})
	filters["enabled"] = true

	if category := c.Query("category"); category != "" {
		filters["category"] = category
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
// GET /api/v1/workflows/templates/:id
func (h *WorkflowHandler) GetTemplate(c *gin.Context) {
	if h.templateRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "模板功能未启用"})
		return
	}

	templateID := c.Param("id")
	template, err := h.templateRepo.GetTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// CreateWorkflowFromTemplate 从模板创建工作流
// POST /api/v1/workflows/templates/:id/create
func (h *WorkflowHandler) CreateWorkflowFromTemplate(c *gin.Context) {
	if h.templateRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "模板功能未启用"})
		return
	}

	templateID := c.Param("id")
	template, err := h.templateRepo.GetTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在", "details": err.Error()})
		return
	}

	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Variables   map[string]interface{} `json:"variables"` // 模板变量值
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 从模板数据创建工作流
	workflowData := template.WorkflowData
	nodesData, _ := workflowData["nodes"].([]interface{})
	edgesData, _ := workflowData["edges"].([]interface{})
	variablesData, _ := workflowData["variables"].([]interface{})

	// 转换为 Workflow 结构
	wf := &workflow.Workflow{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Status:      "draft",
		CreatedBy:   req.Name, // TODO: 从认证上下文获取
	}

	// 解析节点
	for _, nodeData := range nodesData {
		nodeMap, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}

		node := workflow.Node{
			ID:       getString(nodeMap, "id"),
			Type:     workflow.NodeType(getString(nodeMap, "type")),
			Name:     getString(nodeMap, "name"),
			Config:   getMap(nodeMap, "config"),
			Position: parsePosition(nodeMap["position"]),
		}

		// 替换模板变量（${variable_name}）
		node.Config = replaceTemplateVariables(node.Config, req.Variables)

		// 解析输入输出端口
		if inputs, ok := nodeMap["inputs"].([]interface{}); ok {
			for _, inputData := range inputs {
				if inputMap, ok := inputData.(map[string]interface{}); ok {
					node.Inputs = append(node.Inputs, parseInputPort(inputMap))
				}
			}
		}
		if outputs, ok := nodeMap["outputs"].([]interface{}); ok {
			for _, outputData := range outputs {
				if outputMap, ok := outputData.(map[string]interface{}); ok {
					node.Outputs = append(node.Outputs, parseOutputPort(outputMap))
				}
			}
		}

		wf.Nodes = append(wf.Nodes, node)
	}

	// 解析边
	for _, edgeData := range edgesData {
		if edgeMap, ok := edgeData.(map[string]interface{}); ok {
			edge := workflow.Edge{
				ID:         getString(edgeMap, "id"),
				Source:     getString(edgeMap, "source"),
				Target:     getString(edgeMap, "target"),
				SourcePort: getString(edgeMap, "source_port"),
				TargetPort: getString(edgeMap, "target_port"),
				Condition:  getString(edgeMap, "condition"),
			}
			if paramMapping, ok := edgeMap["param_mapping"].(map[string]interface{}); ok {
				edge.ParamMapping = make(map[string]string)
				for k, v := range paramMapping {
					edge.ParamMapping[k] = v.(string)
				}
			}
			wf.Edges = append(wf.Edges, edge)
		}
	}

	// 解析变量
	for _, varData := range variablesData {
		if varMap, ok := varData.(map[string]interface{}); ok {
			wf.Variables = append(wf.Variables, workflow.Variable{
				Name:        getString(varMap, "name"),
				Type:        getString(varMap, "type"),
				Value:       varMap["value"],
				Description: getString(varMap, "description"),
			})
		}
	}

	// 验证工作流
	if err := wf.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作流验证失败", "details": err.Error()})
		return
	}

	// 保存工作流
	if err := h.repo.Save(c.Request.Context(), wf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建工作流失败", "details": err.Error()})
		return
	}

	// 增加模板使用次数
	h.templateRepo.IncrementUsage(c.Request.Context(), templateID)

	c.JSON(http.StatusCreated, wf)
}

// 辅助函数
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return make(map[string]interface{})
}

func parsePosition(pos interface{}) workflow.Position {
	if posMap, ok := pos.(map[string]interface{}); ok {
		x, _ := posMap["x"].(float64)
		y, _ := posMap["y"].(float64)
		return workflow.Position{X: x, Y: y}
	}
	return workflow.Position{}
}

func parseInputPort(m map[string]interface{}) workflow.InputPort {
	return workflow.InputPort{
		ID:          getString(m, "id"),
		Label:       getString(m, "label"),
		Type:        getString(m, "type"),
		ParamName:   getString(m, "param_name"),
		DataType:    getString(m, "data_type"),
		Required:    getBool(m, "required"),
		Description: getString(m, "description"),
		DefaultValue: m["default_value"],
	}
}

func parseOutputPort(m map[string]interface{}) workflow.OutputPort {
	return workflow.OutputPort{
		ID:          getString(m, "id"),
		Label:       getString(m, "label"),
		Type:        getString(m, "type"),
		ParamName:   getString(m, "param_name"),
		DataType:    getString(m, "data_type"),
		Description: getString(m, "description"),
	}
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// replaceTemplateVariables 替换模板变量
func replaceTemplateVariables(config map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range config {
		switch val := v.(type) {
		case string:
			// 替换 ${variable_name} 格式的变量
			if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
				varName := val[2 : len(val)-1]
				if varValue, ok := variables[varName]; ok {
					result[k] = varValue
				} else {
					result[k] = val // 保持原值
				}
			} else {
				result[k] = val
			}
		case map[string]interface{}:
			result[k] = replaceTemplateVariables(val, variables)
		default:
			result[k] = v
		}
	}
	return result
}
