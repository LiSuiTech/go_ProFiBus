package handlers

import (
	"fmt"
	"go_ProFiBus/event"
	"go_ProFiBus/storage"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// EventHandler 事件管理处理器
type EventHandler struct {
	store *storage.PostgresStore
}

// NewEventHandler 创建事件处理器
func NewEventHandler(store *storage.PostgresStore) *EventHandler {
	return &EventHandler{
		store: store,
	}
}

// GetEventsRequest 获取事件列表请求参数
type GetEventsRequest struct {
	Status   string    `form:"status"`   // pending, confirmed, rejected
	Type     string    `form:"type"`     // threshold, statistical, pattern
	Severity *int      `form:"severity"` // 1-5
	Start    time.Time `form:"start" time_format:"2006-01-02T15:04:05Z07:00"`
	End      time.Time `form:"end" time_format:"2006-01-02T15:04:05Z07:00"`
	Limit    int       `form:"limit"`
	Offset   int       `form:"offset"`
}

// UpdateEventRequest 更新事件请求
type UpdateEventRequest struct {
	Status      *string                `json:"status"`      // 状态更新
	Severity    *int                   `json:"severity"`    // 严重程度更新
	Description *string                `json:"description"` // 描述更新
	Metadata    map[string]interface{} `json:"metadata"`    // 元数据更新
}

// GetEvents 获取事件列表
// @Summary 查询事件列表
// @Tags events
// @Accept json
// @Produce json
// @Param status query string false "事件状态: pending, confirmed, rejected"
// @Param type query string false "事件类型: threshold, statistical, pattern"
// @Param severity query int false "严重程度: 1-5"
// @Param start query string false "开始时间 RFC3339格式"
// @Param end query string false "结束时间 RFC3339格式"
// @Param limit query int false "返回记录数" default(100)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/events [get]
func (h *EventHandler) GetEvents(c *gin.Context) {
	var req GetEventsRequest

	// 绑定查询参数
	req.Status = c.Query("status")
	req.Type = c.Query("type")

	// 解析severity
	if severityStr := c.Query("severity"); severityStr != "" {
		var severity int
		if _, err := fmt.Sscanf(severityStr, "%d", &severity); err == nil {
			if severity >= 1 && severity <= 5 {
				req.Severity = &severity
			}
		}
	}

	// 解析时间范围
	if startStr := c.Query("start"); startStr != "" {
		if start, err := time.Parse(time.RFC3339, startStr); err == nil {
			req.Start = start
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		if end, err := time.Parse(time.RFC3339, endStr); err == nil {
			req.End = end
		}
	}

	// 解析limit和offset
	req.Limit = 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			req.Limit = limit
		}
	}

	req.Offset = 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			req.Offset = offset
		}
	}

	// 构建过滤条件
	filters := storage.EventFilters{
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	if req.Status != "" {
		filters.Status = &req.Status
	}
	if req.Type != "" {
		filters.Type = &req.Type
	}
	if req.Severity != nil {
		filters.Severity = req.Severity
	}
	if !req.Start.IsZero() {
		filters.StartTime = &req.Start
	}
	if !req.End.IsZero() {
		filters.EndTime = &req.End
	}

	// 查询事件
	events, err := h.store.QueryEvents(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询事件失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(events),
		"limit":  req.Limit,
		"offset": req.Offset,
		"events": events,
	})
}

// GetEvent 获取单个事件详情
// @Summary 获取事件详情
// @Tags events
// @Accept json
// @Produce json
// @Param event_id path string true "事件ID"
// @Success 200 {object} event.Event
// @Router /api/v1/events/{event_id} [get]
func (h *EventHandler) GetEvent(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "事件ID不能为空"})
		return
	}

	// 查询事件
	evt, err := h.store.GetEventByID(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询事件失败", "details": err.Error()})
		return
	}

	if evt == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "事件不存在"})
		return
	}

	c.JSON(http.StatusOK, evt)
}

// UpdateEvent 更新事件
// @Summary 更新事件信息
// @Tags events
// @Accept json
// @Produce json
// @Param event_id path string true "事件ID"
// @Param event body UpdateEventRequest true "更新内容"
// @Success 200 {object} event.Event
// @Router /api/v1/events/{event_id} [put]
func (h *EventHandler) UpdateEvent(c *gin.Context) {
	eventID := c.Param("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "事件ID不能为空"})
		return
	}

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 验证状态值
	if req.Status != nil {
		validStatuses := map[string]bool{
			string(event.EventPending):   true,
			string(event.EventConfirmed): true,
			string(event.EventRejected):  true,
		}
		if !validStatuses[*req.Status] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态值，必须是: pending, confirmed, rejected"})
			return
		}
	}

	// 验证严重程度
	if req.Severity != nil && (*req.Severity < 1 || *req.Severity > 5) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "严重程度必须在1-5之间"})
		return
	}

	// 更新事件
	updates := make(map[string]interface{})
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Severity != nil {
		updates["severity"] = *req.Severity
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}

	if err := h.store.UpdateEvent(eventID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新事件失败", "details": err.Error()})
		return
	}

	// 返回更新后的事件
	evt, err := h.store.GetEventByID(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询更新后的事件失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, evt)
}

// GetEventStats 获取事件统计信息
// @Summary 获取事件统计
// @Tags events
// @Accept json
// @Produce json
// @Param start query string false "开始时间 RFC3339格式"
// @Param end query string false "结束时间 RFC3339格式"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/events/stats [get]
func (h *EventHandler) GetEventStats(c *gin.Context) {
	// 解析时间范围
	var start, end time.Time
	if startStr := c.Query("start"); startStr != "" {
		var err error
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始时间格式"})
			return
		}
	} else {
		// 默认最近7天
		start = time.Now().AddDate(0, 0, -7)
	}

	if endStr := c.Query("end"); endStr != "" {
		var err error
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束时间格式"})
			return
		}
	} else {
		end = time.Now()
	}

	// 获取统计信息
	stats, err := h.store.GetEventStats(start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计信息失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start": start.Format(time.RFC3339),
		"end":   end.Format(time.RFC3339),
		"stats": stats,
	})
}
