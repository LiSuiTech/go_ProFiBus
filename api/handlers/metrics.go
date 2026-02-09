package handlers

import (
	"net/http"
	"time"

	"go_ProFiBus/internal/application/orchestrator"
	"go_ProFiBus/pkg/interfaces"

	"github.com/gin-gonic/gin"
)

// MetricsHandler 性能指标API处理器
type MetricsHandler struct {
	repository  interfaces.TraceRepository
	orchestrator *orchestrator.Orchestrator
}

// NewMetricsHandler 创建性能指标处理器
func NewMetricsHandler(repository interfaces.TraceRepository, orch *orchestrator.Orchestrator) *MetricsHandler {
	return &MetricsHandler{
		repository:  repository,
		orchestrator: orch,
	}
}

// GetPipelineMetrics godoc
// @Summary 获取管道性能指标
// @Description 获取指定管道在指定时间范围内的性能指标，包括吞吐量、延迟、错误率等
// @Tags metrics
// @Accept json
// @Produce json
// @Param id path string true "管道ID"
// @Param start_time query string false "开始时间 (RFC3339 格式，默认为1小时前)"
// @Param end_time query string false "结束时间 (RFC3339 格式，默认为当前时间)"
// @Success 200 {object} interfaces.PipelineMetrics "管道性能指标"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "管道未找到"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/pipelines/{id}/metrics [get]
func (h *MetricsHandler) GetPipelineMetrics(c *gin.Context) {
	pipelineID := c.Param("id")

	if pipelineID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Pipeline ID is required",
			"message": "Please provide a valid pipeline_id",
		})
		return
	}

	// 默认查询最近1小时的数据
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	// 解析查询参数
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid start_time format",
				"message": "Use RFC3339 format (e.g., 2024-01-20T10:00:00Z)",
			})
			return
		}
		startTime = parsedTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_time format",
				"message": "Use RFC3339 format (e.g., 2024-01-20T10:00:00Z)",
			})
			return
		}
		endTime = parsedTime
	}

	// 获取性能指标
	metrics, err := h.repository.GetPipelineMetrics(c.Request.Context(), pipelineID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get pipeline metrics",
			"message": err.Error(),
		})
		return
	}

	if metrics == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No metrics found",
			"message": "No metrics data found for the specified pipeline and time range",
		})
		return
	}

	// 格式化响应
	response := formatMetricsResponse(metrics)

	c.JSON(http.StatusOK, response)
}

// GetSystemMetrics godoc
// @Summary 获取系统整体性能指标
// @Description 获取所有管道的汇总性能指标
// @Tags metrics
// @Accept json
// @Produce json
// @Param start_time query string false "开始时间 (RFC3339 格式，默认为1小时前)"
// @Param end_time query string false "结束时间 (RFC3339 格式，默认为当前时间)"
// @Success 200 {object} map[string]interface{} "系统性能指标"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/metrics/system [get]
func (h *MetricsHandler) GetSystemMetrics(c *gin.Context) {
	// 默认查询最近1小时的数据
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	// 解析查询参数
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid start_time format",
				"message": "Use RFC3339 format (e.g., 2024-01-20T10:00:00Z)",
			})
			return
		}
		startTime = parsedTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_time format",
				"message": "Use RFC3339 format (e.g., 2024-01-20T10:00:00Z)",
			})
			return
		}
		endTime = parsedTime
	}

	// 获取所有管道的指标并汇总
	var totalPipelines int
	var totalSamples int64
	var successSamples int64
	var errorSamples int64
	var pipelineCount int

	// 如果 orchestrator 可用，从管道状态获取统计
	if h.orchestrator != nil {
		pipelines := h.orchestrator.GetAllPipelines()
		totalPipelines = len(pipelines)
		pipelineCount = totalPipelines

		for _, pipeline := range pipelines {
			status := pipeline.GetStatus()
			totalSamples += status.SamplesProcessed
			errorSamples += status.Errors
		}
		successSamples = totalSamples - errorSamples
	}

	// 从 TraceRepository 获取追踪数据统计
	if h.repository != nil {
		// 获取所有管道的追踪指标
		// 注意：这需要 TraceRepository 支持列出所有管道
		// 目前先使用管道状态数据
	}

	// 计算平均持续时间（如果有数据）
	avgDurationMs := int64(0)
	if pipelineCount > 0 {
		avgDurationMs = totalDuration.Milliseconds() / int64(pipelineCount)
	}

	// 计算吞吐量
	duration := endTime.Sub(startTime).Seconds()
	throughputPerSec := 0.0
	if duration > 0 {
		throughputPerSec = float64(totalSamples) / duration
	}

	c.JSON(http.StatusOK, gin.H{
		"start_time": startTime.Format(time.RFC3339),
		"end_time":   endTime.Format(time.RFC3339),
		"system_metrics": gin.H{
			"total_pipelines":   totalPipelines,
			"total_samples":     totalSamples,
			"success_samples":   successSamples,
			"error_samples":     errorSamples,
			"avg_duration_ms":   avgDurationMs,
			"throughput_per_sec": throughputPerSec,
		},
	})
}

// GetComponentMetrics godoc
// @Summary 获取组件性能指标
// @Description 获取指定组件在指定时间范围内的性能指标
// @Tags metrics
// @Accept json
// @Produce json
// @Param pipeline_id query string true "管道ID"
// @Param component_id query string true "组件ID"
// @Param start_time query string false "开始时间 (RFC3339 格式，默认为1小时前)"
// @Param end_time query string false "结束时间 (RFC3339 格式，默认为当前时间)"
// @Success 200 {object} map[string]interface{} "组件性能指标"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "组件未找到"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/metrics/component [get]
func (h *MetricsHandler) GetComponentMetrics(c *gin.Context) {
	pipelineID := c.Query("pipeline_id")
	componentID := c.Query("component_id")

	if pipelineID == "" || componentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameters",
			"message": "Both pipeline_id and component_id are required",
		})
		return
	}

	// 默认查询最近1小时的数据
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	// 解析查询参数
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid start_time format",
				"message": "Use RFC3339 format (e.g., 2024-01-20T10:00:00Z)",
			})
			return
		}
		startTime = parsedTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_time format",
				"message": "Use RFC3339 format (e.g., 2024-01-20T10:00:00Z)",
			})
			return
		}
		endTime = parsedTime
	}

	// 获取管道性能指标
	metrics, err := h.repository.GetPipelineMetrics(c.Request.Context(), pipelineID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get metrics",
			"message": err.Error(),
		})
		return
	}

	// 提取特定组件的指标
	if metrics == nil || metrics.ComponentMetrics == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No metrics found",
			"message": "No metrics data found for the specified component",
		})
		return
	}

	componentMetric, exists := metrics.ComponentMetrics[componentID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Component not found",
			"message": "No metrics data found for the specified component_id",
		})
		return
	}

	// 格式化响应
	response := map[string]interface{}{
		"pipeline_id":    pipelineID,
		"component_id":   componentMetric.ComponentID,
		"component_type": componentMetric.ComponentType,
		"component_name": componentMetric.ComponentName,
		"start_time":     metrics.StartTime.Format(time.RFC3339),
		"end_time":       metrics.EndTime.Format(time.RFC3339),
		"metrics": gin.H{
			"event_count":    componentMetric.EventCount,
			"avg_duration_ms": componentMetric.AvgDuration.Milliseconds(),
			"max_duration_ms": componentMetric.MaxDuration.Milliseconds(),
			"min_duration_ms": componentMetric.MinDuration.Milliseconds(),
			"error_count":    componentMetric.ErrorCount,
			"error_rate":     componentMetric.ErrorRate,
		},
	}

	c.JSON(http.StatusOK, response)
}

// formatMetricsResponse 格式化性能指标响应
func formatMetricsResponse(metrics *interfaces.PipelineMetrics) map[string]interface{} {
	response := map[string]interface{}{
		"pipeline_id":  metrics.PipelineID,
		"start_time":   metrics.StartTime.Format(time.RFC3339),
		"end_time":     metrics.EndTime.Format(time.RFC3339),
		"summary": gin.H{
			"total_samples":     metrics.TotalSamples,
			"success_samples":   metrics.SuccessSamples,
			"error_samples":     metrics.ErrorSamples,
			"success_rate":      calculateSuccessRate(metrics.SuccessSamples, metrics.TotalSamples),
			"avg_duration_ms":   metrics.AvgDuration.Milliseconds(),
			"max_duration_ms":   metrics.MaxDuration.Milliseconds(),
			"min_duration_ms":   metrics.MinDuration.Milliseconds(),
			"throughput_per_sec": calculateThroughput(metrics.TotalSamples, metrics.StartTime, metrics.EndTime),
		},
	}

	// 格式化组件指标
	if metrics.ComponentMetrics != nil {
		components := make([]map[string]interface{}, 0, len(metrics.ComponentMetrics))
		for _, compMetric := range metrics.ComponentMetrics {
			components = append(components, map[string]interface{}{
				"component_id":    compMetric.ComponentID,
				"component_type":  compMetric.ComponentType,
				"component_name":  compMetric.ComponentName,
				"event_count":     compMetric.EventCount,
				"avg_duration_ms": compMetric.AvgDuration.Milliseconds(),
				"max_duration_ms": compMetric.MaxDuration.Milliseconds(),
				"min_duration_ms": compMetric.MinDuration.Milliseconds(),
				"error_count":     compMetric.ErrorCount,
				"error_rate":      compMetric.ErrorRate,
			})
		}
		response["components"] = components
	}

	return response
}

// calculateSuccessRate 计算成功率
func calculateSuccessRate(successSamples, totalSamples int64) float64 {
	if totalSamples == 0 {
		return 0.0
	}
	return float64(successSamples) / float64(totalSamples) * 100.0
}

// calculateThroughput 计算吞吐量（样本数/秒）
func calculateThroughput(totalSamples int64, startTime, endTime time.Time) float64 {
	duration := endTime.Sub(startTime).Seconds()
	if duration <= 0 {
		return 0.0
	}
	return float64(totalSamples) / duration
}
