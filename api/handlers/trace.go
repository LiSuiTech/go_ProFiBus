package handlers

import (
	"net/http"
	"strconv"
	"time"

	"go_ProFiBus/pkg/interfaces"

	"github.com/gin-gonic/gin"
)

// TraceHandler 追踪API处理器
type TraceHandler struct {
	tracer     interfaces.Tracer
	repository interfaces.TraceRepository
}

// NewTraceHandler 创建追踪处理器
func NewTraceHandler(tracer interfaces.Tracer, repository interfaces.TraceRepository) *TraceHandler {
	return &TraceHandler{
		tracer:     tracer,
		repository: repository,
	}
}

// GetTraces godoc
// @Summary 获取追踪记录
// @Description 查询数据流追踪记录，支持多种过滤条件
// @Tags trace
// @Accept json
// @Produce json
// @Param pipeline_id query string false "管道ID"
// @Param sample_id query string false "样本ID"
// @Param component_type query string false "组件类型 (source, processor, analyzer, sink)"
// @Param component_id query string false "组件ID"
// @Param action query string false "动作 (enter, process, exit, error, skip)"
// @Param status query string false "状态 (success, error, skip)"
// @Param start_time query string false "开始时间 (RFC3339 格式)"
// @Param end_time query string false "结束时间 (RFC3339 格式)"
// @Param limit query int false "限制数量" default(100)
// @Param offset query int false "偏移量" default(0)
// @Param order_by query string false "排序字段" default(timestamp)
// @Param order_desc query bool false "降序排序" default(true)
// @Success 200 {object} map[string]interface{} "追踪记录列表"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/traces [get]
func (h *TraceHandler) GetTraces(c *gin.Context) {
	// 构建过滤器
	filter := interfaces.TraceFilter{
		Limit:     100,
		OrderBy:   "timestamp",
		OrderDesc: true,
	}

	// 解析查询参数
	if pipelineID := c.Query("pipeline_id"); pipelineID != "" {
		filter.PipelineID = &pipelineID
	}

	if sampleID := c.Query("sample_id"); sampleID != "" {
		filter.SampleID = &sampleID
	}

	if componentType := c.Query("component_type"); componentType != "" {
		filter.ComponentType = &componentType
	}

	if componentID := c.Query("component_id"); componentID != "" {
		filter.ComponentID = &componentID
	}

	if action := c.Query("action"); action != "" {
		filter.Action = &action
	}

	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid start_time format",
				"message": "Use RFC3339 format (e.g., 2024-01-20T10:00:00Z)",
			})
			return
		}
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid end_time format",
				"message": "Use RFC3339 format (e.g., 2024-01-20T10:00:00Z)",
			})
			return
		}
		filter.EndTime = &endTime
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid limit",
				"message": "Limit must be between 1 and 1000",
			})
			return
		}
		filter.Limit = limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid offset",
				"message": "Offset must be non-negative",
			})
			return
		}
		filter.Offset = offset
	}

	if orderBy := c.Query("order_by"); orderBy != "" {
		filter.OrderBy = orderBy
	}

	if orderDescStr := c.Query("order_desc"); orderDescStr != "" {
		orderDesc, err := strconv.ParseBool(orderDescStr)
		if err == nil {
			filter.OrderDesc = orderDesc
		}
	}

	// 查询追踪记录
	traces, err := h.tracer.GetTraces(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get traces",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"traces": traces,
		"total":  len(traces),
		"filter": gin.H{
			"pipeline_id":    filter.PipelineID,
			"sample_id":      filter.SampleID,
			"component_type": filter.ComponentType,
			"component_id":   filter.ComponentID,
			"action":         filter.Action,
			"status":         filter.Status,
			"limit":          filter.Limit,
			"offset":         filter.Offset,
		},
	})
}

// GetTraceBySample godoc
// @Summary 根据样本ID获取完整追踪链路
// @Description 获取指定数据样本在整个管道中的完整追踪记录
// @Tags trace
// @Accept json
// @Produce json
// @Param sample_id path string true "样本ID"
// @Success 200 {object} map[string]interface{} "追踪链路"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "样本未找到"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/traces/samples/{sample_id} [get]
func (h *TraceHandler) GetTraceBySample(c *gin.Context) {
	sampleID := c.Param("sample_id")

	if sampleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Sample ID is required",
			"message": "Please provide a valid sample_id",
		})
		return
	}

	// 查询追踪记录
	traces, err := h.repository.GetTraceBySampleID(c.Request.Context(), sampleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get trace by sample",
			"message": err.Error(),
		})
		return
	}

	if len(traces) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Sample not found",
			"message": "No trace records found for the specified sample_id",
		})
		return
	}

	// 构建追踪链路
	traceChain := h.buildTraceChain(traces)

	c.JSON(http.StatusOK, gin.H{
		"sample_id": sampleID,
		"total_events": len(traces),
		"trace_chain": traceChain,
		"events": traces,
	})
}

// buildTraceChain 构建追踪链路
func (h *TraceHandler) buildTraceChain(traces []interfaces.TraceEvent) []map[string]interface{} {
	chain := make([]map[string]interface{}, 0)

	// 按时间排序
	for _, trace := range traces {
		node := map[string]interface{}{
			"component_type": trace.ComponentType,
			"component_id":   trace.ComponentID,
			"component_name": trace.ComponentName,
			"action":         trace.Action,
			"timestamp":      trace.Timestamp.Format(time.RFC3339),
			"duration_ms":    trace.Duration.Milliseconds(),
			"status":         trace.Status,
		}

		if trace.Error != "" {
			node["error"] = trace.Error
		}

		chain = append(chain, node)
	}

	return chain
}

// GetTraceStats godoc
// @Summary 获取追踪统计信息
// @Description 获取指定时间范围内的追踪统计数据
// @Tags trace
// @Accept json
// @Produce json
// @Param pipeline_id query string false "管道ID"
// @Param start_time query string false "开始时间 (RFC3339 格式)"
// @Param end_time query string false "结束时间 (RFC3339 格式)"
// @Success 200 {object} map[string]interface{} "统计信息"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "内部错误"
// @Router /api/v1/traces/stats [get]
func (h *TraceHandler) GetTraceStats(c *gin.Context) {
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

	pipelineID := c.Query("pipeline_id")

	// 构建过滤器
	filter := interfaces.TraceFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     10000, // 最大查询数量
	}

	if pipelineID != "" {
		filter.PipelineID = &pipelineID
	}

	// 查询追踪记录
	traces, err := h.tracer.GetTraces(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get traces",
			"message": err.Error(),
		})
		return
	}

	// 计算统计信息
	stats := h.calculateStats(traces)

	c.JSON(http.StatusOK, gin.H{
		"start_time": startTime.Format(time.RFC3339),
		"end_time":   endTime.Format(time.RFC3339),
		"pipeline_id": pipelineID,
		"stats":      stats,
	})
}

// calculateStats 计算统计信息
func (h *TraceHandler) calculateStats(traces []interfaces.TraceEvent) map[string]interface{} {
	stats := map[string]interface{}{
		"total_events":   len(traces),
		"success_events": 0,
		"error_events":   0,
		"skip_events":    0,
		"by_component_type": make(map[string]int),
		"by_action":      make(map[string]int),
		"avg_duration_ms": 0.0,
	}

	totalDuration := int64(0)

	for _, trace := range traces {
		// 按状态统计
		switch trace.Status {
		case interfaces.StatusSuccess:
			stats["success_events"] = stats["success_events"].(int) + 1
		case interfaces.StatusError:
			stats["error_events"] = stats["error_events"].(int) + 1
		case interfaces.StatusSkip:
			stats["skip_events"] = stats["skip_events"].(int) + 1
		}

		// 按组件类型统计
		byType := stats["by_component_type"].(map[string]int)
		byType[trace.ComponentType]++

		// 按动作统计
		byAction := stats["by_action"].(map[string]int)
		byAction[trace.Action]++

		// 累计耗时
		totalDuration += trace.Duration.Milliseconds()
	}

	// 计算平均耗时
	if len(traces) > 0 {
		stats["avg_duration_ms"] = float64(totalDuration) / float64(len(traces))
	}

	return stats
}
