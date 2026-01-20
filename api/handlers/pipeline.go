package handlers

import (
	"net/http"

	"go_ProFiBus/internal/application/orchestrator"
	"go_ProFiBus/pkg/interfaces"

	"github.com/gin-gonic/gin"
)

// PipelineHandler 管道API处理器
type PipelineHandler struct {
	orchestrator *orchestrator.Orchestrator
}

// NewPipelineHandler 创建管道处理器
func NewPipelineHandler(orch *orchestrator.Orchestrator) *PipelineHandler {
	return &PipelineHandler{
		orchestrator: orch,
	}
}

// PipelineInfo 管道信息
type PipelineInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Status  string `json:"status"`
}

// PipelineTopology 管道拓扑结构
type PipelineTopology struct {
	PipelineID string                   `json:"pipeline_id"`
	Name       string                   `json:"name"`
	Running    bool                     `json:"running"`
	Components PipelineTopologyComponents `json:"components"`
}

// PipelineTopologyComponents 管道组件信息
type PipelineTopologyComponents struct {
	Source     ComponentInfo   `json:"source"`
	Processors []ComponentInfo `json:"processors"`
	Analyzers  []ComponentInfo `json:"analyzers"`
	Sinks      []ComponentInfo `json:"sinks"`
}

// ComponentInfo 组件信息
type ComponentInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// GetPipelines godoc
// @Summary 获取所有管道列表
// @Description 返回系统中所有注册的数据处理管道
// @Tags pipeline
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "管道列表"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/pipelines [get]
func (h *PipelineHandler) GetPipelines(c *gin.Context) {
	status := h.orchestrator.GetStatus()

	pipelines := make([]PipelineInfo, 0)
	for name, pipelineStatus := range status.Pipelines {
		pipelines = append(pipelines, PipelineInfo{
			ID:      name,
			Name:    name,
			Running: pipelineStatus.Running,
			Status:  pipelineStatus.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"pipelines": pipelines,
		"total":     len(pipelines),
		"running":   status.RunningCount,
		"stopped":   status.StoppedCount,
	})
}

// GetPipelineTopology godoc
// @Summary 获取管道拓扑结构
// @Description 返回指定管道的完整拓扑结构，包括所有组件（数据源、处理器、分析器、输出）
// @Tags pipeline
// @Accept json
// @Produce json
// @Param id path string true "管道ID"
// @Success 200 {object} PipelineTopology "管道拓扑"
// @Failure 404 {object} map[string]interface{} "管道未找到"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/pipelines/{id}/topology [get]
func (h *PipelineHandler) GetPipelineTopology(c *gin.Context) {
	pipelineID := c.Param("id")

	pipeline, err := h.orchestrator.GetPipeline(pipelineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Pipeline not found",
			"message": err.Error(),
		})
		return
	}

	status := pipeline.GetStatus()

	// 提取拓扑信息
	topology := PipelineTopology{
		PipelineID: pipelineID,
		Name:       status.Name,
		Running:    status.Running,
		Components: h.extractComponents(pipeline),
	}

	c.JSON(http.StatusOK, topology)
}

// extractComponents 提取管道组件信息
func (h *PipelineHandler) extractComponents(pipeline *orchestrator.Pipeline) PipelineTopologyComponents {
	components := PipelineTopologyComponents{
		Processors: make([]ComponentInfo, 0),
		Analyzers:  make([]ComponentInfo, 0),
		Sinks:      make([]ComponentInfo, 0),
	}

	// 提取 Source 信息
	if source := pipeline.GetSource(); source != nil {
		components.Source = ComponentInfo{
			ID:          source.GetID(),
			Name:        source.GetName(),
			Type:        "source",
			Description: source.GetDescription(),
		}
	}

	// 提取 Processors 信息
	for _, proc := range pipeline.GetProcessors() {
		components.Processors = append(components.Processors, ComponentInfo{
			ID:   proc.GetName(),
			Name: proc.GetName(),
			Type: "processor",
		})
	}

	// 提取 Analyzers 信息
	for _, analyzer := range pipeline.GetAnalyzers() {
		components.Analyzers = append(components.Analyzers, ComponentInfo{
			ID:   analyzer.GetID(),
			Name: analyzer.GetName(),
			Type: "analyzer",
		})
	}

	// 提取 Sinks 信息
	for _, sink := range pipeline.GetSinks() {
		components.Sinks = append(components.Sinks, ComponentInfo{
			ID:   sink.GetName(),
			Name: sink.GetName(),
			Type: "sink",
		})
	}

	return components
}

// GetPipelineStatus godoc
// @Summary 获取管道状态
// @Description 返回指定管道的当前运行状态和统计信息
// @Tags pipeline
// @Accept json
// @Produce json
// @Param id path string true "管道ID"
// @Success 200 {object} map[string]interface{} "管道状态"
// @Failure 404 {object} map[string]interface{} "管道未找到"
// @Router /api/v1/pipelines/{id}/status [get]
func (h *PipelineHandler) GetPipelineStatus(c *gin.Context) {
	pipelineID := c.Param("id")

	pipeline, err := h.orchestrator.GetPipeline(pipelineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Pipeline not found",
			"message": err.Error(),
		})
		return
	}

	status := pipeline.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"pipeline_id":     pipelineID,
		"name":            status.Name,
		"running":         status.Running,
		"status":          status.Status,
		"samples_processed": status.SamplesProcessed,
		"errors":          status.Errors,
		"last_sample_time": status.LastSampleTime,
	})
}

// StartPipeline godoc
// @Summary 启动管道
// @Description 启动指定的数据处理管道
// @Tags pipeline
// @Accept json
// @Produce json
// @Param id path string true "管道ID"
// @Success 200 {object} map[string]interface{} "成功消息"
// @Failure 400 {object} map[string]interface{} "请求错误"
// @Failure 404 {object} map[string]interface{} "管道未找到"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/pipelines/{id}/start [post]
func (h *PipelineHandler) StartPipeline(c *gin.Context) {
	pipelineID := c.Param("id")

	if err := h.orchestrator.StartPipeline(pipelineID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start pipeline",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pipeline started successfully",
		"pipeline_id": pipelineID,
	})
}

// StopPipeline godoc
// @Summary 停止管道
// @Description 停止指定的数据处理管道
// @Tags pipeline
// @Accept json
// @Produce json
// @Param id path string true "管道ID"
// @Success 200 {object} map[string]interface{} "成功消息"
// @Failure 404 {object} map[string]interface{} "管道未找到"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/pipelines/{id}/stop [post]
func (h *PipelineHandler) StopPipeline(c *gin.Context) {
	pipelineID := c.Param("id")

	if err := h.orchestrator.StopPipeline(pipelineID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to stop pipeline",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pipeline stopped successfully",
		"pipeline_id": pipelineID,
	})
}
