package handlers

import (
	"go_ProFiBus/collector"
	"go_ProFiBus/storage"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SensorHandler 传感器数据处理器
type SensorHandler struct {
	store *storage.PostgresStore
}

// NewSensorHandler 创建传感器处理器
func NewSensorHandler(store *storage.PostgresStore) *SensorHandler {
	return &SensorHandler{
		store: store,
	}
}

// GetSensorReadingsRequest 获取传感器读数请求参数
type GetSensorReadingsRequest struct {
	SensorID string    `uri:"sensor_id" binding:"required"`
	Start    time.Time `form:"start" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	End      time.Time `form:"end" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	Limit    int       `form:"limit" binding:"omitempty,min=1,max=10000"`
}

// PostSensorReadingsRequest 批量写入传感器读数请求
type PostSensorReadingsRequest struct {
	Readings []*collector.DataSample `json:"readings" binding:"required,dive"`
}

// AggregationRequest 传感器数据聚合请求
type AggregationRequest struct {
	SensorID string    `uri:"sensor_id" binding:"required"`
	Field    string    `form:"field" binding:"required"`
	Start    time.Time `form:"start" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	End      time.Time `form:"end" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	Interval int       `form:"interval" binding:"required,min=1"` // 聚合间隔（分钟）
}

// GetSensorReadings 获取传感器读数
// @Summary 获取传感器历史读数
// @Tags sensors
// @Accept json
// @Produce json
// @Param sensor_id path string true "传感器ID"
// @Param start query string true "开始时间 RFC3339格式"
// @Param end query string true "结束时间 RFC3339格式"
// @Param limit query int false "返回记录数限制" default(1000)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sensors/{sensor_id}/readings [get]
func (h *SensorHandler) GetSensorReadings(c *gin.Context) {
	var req GetSensorReadingsRequest

	// 绑定路径参数
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的传感器ID"})
		return
	}

	// 绑定查询参数
	startStr := c.Query("start")
	endStr := c.Query("end")
	limitStr := c.DefaultQuery("limit", "1000")

	// 解析时间
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始时间格式，请使用RFC3339格式"})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束时间格式，请使用RFC3339格式"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit必须在1-10000之间"})
		return
	}

	// 验证时间范围
	if end.Before(start) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "结束时间必须晚于开始时间"})
		return
	}

	// 查询数据库
	readings, err := h.store.QuerySensorReadings(req.SensorID, start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询传感器数据失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sensor_id": req.SensorID,
		"start":     start.Format(time.RFC3339),
		"end":       end.Format(time.RFC3339),
		"count":     len(readings),
		"readings":  readings,
	})
}

// PostSensorReadings 批量写入传感器读数
// @Summary 批量写入传感器数据
// @Tags sensors
// @Accept json
// @Produce json
// @Param readings body PostSensorReadingsRequest true "传感器读数数组"
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/sensors/readings [post]
func (h *SensorHandler) PostSensorReadings(c *gin.Context) {
	var req PostSensorReadingsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 验证读数数量
	if len(req.Readings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "readings数组不能为空"})
		return
	}

	if len(req.Readings) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单次最多写入1000条数据"})
		return
	}

	// 写入数据库
	if err := h.store.WriteSensorReadings(req.Readings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入传感器数据失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "传感器数据写入成功",
		"count":   len(req.Readings),
	})
}

// GetSensorAggregation 获取传感器数据聚合结果
// @Summary 获取传感器数据的时间聚合统计
// @Tags sensors
// @Accept json
// @Produce json
// @Param sensor_id path string true "传感器ID"
// @Param field query string true "要聚合的字段名"
// @Param start query string true "开始时间 RFC3339格式"
// @Param end query string true "结束时间 RFC3339格式"
// @Param interval query int true "聚合间隔（分钟）"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sensors/{sensor_id}/aggregation [get]
func (h *SensorHandler) GetSensorAggregation(c *gin.Context) {
	var req AggregationRequest

	// 绑定路径参数
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的传感器ID"})
		return
	}

	// 绑定查询参数
	req.Field = c.Query("field")
	startStr := c.Query("start")
	endStr := c.Query("end")
	intervalStr := c.Query("interval")

	if req.Field == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "field参数不能为空"})
		return
	}

	// 解析时间
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始时间格式"})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束时间格式"})
		return
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interval必须是大于0的整数"})
		return
	}

	// 验证时间范围
	if end.Before(start) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "结束时间必须晚于开始时间"})
		return
	}

	// 查询聚合数据
	results, err := h.store.AggregateSensorData(req.SensorID, req.Field, start, end, interval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "聚合查询失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sensor_id": req.SensorID,
		"field":     req.Field,
		"start":     start.Format(time.RFC3339),
		"end":       end.Format(time.RFC3339),
		"interval":  interval,
		"count":     len(results),
		"results":   results,
	})
}
