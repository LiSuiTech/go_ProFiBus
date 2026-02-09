package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	deviceDomain "go_ProFiBus/internal/domain/device"
	deviceApp "go_ProFiBus/internal/application/device"
	"go_ProFiBus/pkg/interfaces"
)

// DeviceDataHandler 设备数据API处理器
type DeviceDataHandler struct {
	dataRepo      interfaces.DeviceDataRepository
	fusionService *deviceApp.DataFusionService
}

// NewDeviceDataHandler 创建设备数据处理器
func NewDeviceDataHandler(dataRepo interfaces.DeviceDataRepository, fusionService *deviceApp.DataFusionService) *DeviceDataHandler {
	return &DeviceDataHandler{
		dataRepo:      dataRepo,
		fusionService: fusionService,
	}
}

// CreateDataField 创建数据字段
// POST /api/v1/devices/:device_id/data-fields
func (h *DeviceDataHandler) CreateDataField(c *gin.Context) {
	deviceID := c.Param("device_id")

	var req struct {
		FieldName    string   `json:"field_name" binding:"required"`
		FieldType    string   `json:"field_type" binding:"required"`
		Unit         string   `json:"unit"`
		MinValue     *float64 `json:"min_value"`
		MaxValue     *float64 `json:"max_value"`
		DefaultValue *float64 `json:"default_value"`
		Description  string   `json:"description"`
		FusionWeight float64  `json:"fusion_weight"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	fieldType := deviceDomain.FieldType(req.FieldType)
	if fieldType != deviceDomain.FieldTypeFloat &&
		fieldType != deviceDomain.FieldTypeInt &&
		fieldType != deviceDomain.FieldTypeString &&
		fieldType != deviceDomain.FieldTypeBool {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的字段类型"})
		return
	}

	field := deviceDomain.NewDataField(uuid.New().String(), deviceID, req.FieldName, fieldType)
	field.Unit = req.Unit
	field.MinValue = req.MinValue
	field.MaxValue = req.MaxValue
	field.DefaultValue = req.DefaultValue
	field.Description = req.Description
	if req.FusionWeight > 0 {
		field.SetFusionWeight(req.FusionWeight)
	}

	if err := h.dataRepo.CreateDataField(c.Request.Context(), field); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建数据字段失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, field)
}

// ListDataFields 列出数据字段
// GET /api/v1/devices/:device_id/data-fields
func (h *DeviceDataHandler) ListDataFields(c *gin.Context) {
	deviceID := c.Param("device_id")

	fields, err := h.dataRepo.GetDataFieldsByDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询数据字段列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(fields),
		"fields": fields,
	})
}

// CreateDataSource 创建数据源
// POST /api/v1/devices/:device_id/data-sources
func (h *DeviceDataHandler) CreateDataSource(c *gin.Context) {
	deviceID := c.Param("device_id")

	var req struct {
		SourceName   string            `json:"source_name" binding:"required"`
		SourceType   string            `json:"source_type" binding:"required"`
		ChannelID    string            `json:"channel_id"`
		FieldMapping map[string]string `json:"field_mapping"`
		FusionWeight float64           `json:"fusion_weight"`
		SampleRate   int               `json:"sample_rate"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	sourceType := deviceDomain.SourceType(req.SourceType)
	if sourceType != deviceDomain.SourceTypeSensor &&
		sourceType != deviceDomain.SourceTypeChannel &&
		sourceType != deviceDomain.SourceTypeCalculated &&
		sourceType != deviceDomain.SourceTypeExternal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的数据源类型"})
		return
	}

	source := deviceDomain.NewDataSource(uuid.New().String(), deviceID, req.SourceName, sourceType)
	source.ChannelID = req.ChannelID
	if req.FieldMapping != nil {
		source.SetFieldMapping(req.FieldMapping)
	}
	if req.FusionWeight > 0 {
		source.SetFusionWeight(req.FusionWeight)
	}
	if req.SampleRate > 0 {
		source.SampleRate = req.SampleRate
	}

	if err := h.dataRepo.CreateDataSource(c.Request.Context(), source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建数据源失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, source)
}

// ListDataSources 列出数据源
// GET /api/v1/devices/:device_id/data-sources
func (h *DeviceDataHandler) ListDataSources(c *gin.Context) {
	deviceID := c.Param("device_id")

	sources, err := h.dataRepo.GetDataSourcesByDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询数据源列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(sources),
		"sources": sources,
	})
}

// SubmitDeviceData 提交设备数据
// POST /api/v1/devices/:device_id/data
func (h *DeviceDataHandler) SubmitDeviceData(c *gin.Context) {
	deviceID := c.Param("device_id")

	var req struct {
		SourceID string                 `json:"source_id" binding:"required"`
		Data     map[string]interface{} `json:"data" binding:"required"`
		Quality  float64                `json:"quality"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	quality := req.Quality
	if quality <= 0 {
		quality = 1.0
	}
	if quality > 1 {
		quality = 1.0
	}

	// 提交到融合服务
	if err := h.fusionService.AddDataSample(c.Request.Context(), deviceID, req.SourceID, req.Data, quality); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交数据失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "数据提交成功"})
}

// GetFusedData 获取融合数据
// GET /api/v1/devices/:device_id/fused-data
func (h *DeviceDataHandler) GetFusedData(c *gin.Context) {
	deviceID := c.Param("device_id")

	startStr := c.DefaultQuery("start", "")
	endStr := c.DefaultQuery("end", "")
	limitStr := c.DefaultQuery("limit", "100")

	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始时间格式"})
			return
		}
	} else {
		start = time.Now().Add(-24 * time.Hour) // 默认24小时前
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束时间格式"})
			return
		}
	} else {
		end = time.Now()
	}

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	data, err := h.fusionService.GetFusedData(c.Request.Context(), deviceID, start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询融合数据失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(data),
		"data":  data,
	})
}

// GetLatestFusedData 获取最新融合数据
// GET /api/v1/devices/:device_id/fused-data/latest
func (h *DeviceDataHandler) GetLatestFusedData(c *gin.Context) {
	deviceID := c.Param("device_id")

	data, err := h.fusionService.GetLatestFusedData(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "融合数据不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// UpdateFusionConfig 更新融合配置
// PUT /api/v1/devices/:device_id/fusion-config
func (h *DeviceDataHandler) UpdateFusionConfig(c *gin.Context) {
	deviceID := c.Param("device_id")

	config, err := h.dataRepo.GetFusionConfigByDevice(c.Request.Context(), deviceID)
	if err != nil {
		// 如果不存在，创建新配置
		config = deviceDomain.NewFusionConfig(uuid.New().String(), deviceID)
	}

	var req struct {
		FusionStrategy string             `json:"fusion_strategy"`
		TimeWindowMs   int                `json:"time_window_ms"`
		MinSources     int                `json:"min_sources"`
		FieldWeights   map[string]float64 `json:"field_weights"`
		SourceWeights  map[string]float64 `json:"source_weights"`
		Enabled        *bool              `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if req.FusionStrategy != "" {
		config.SetFusionStrategy(req.FusionStrategy)
	}
	if req.TimeWindowMs > 0 {
		config.TimeWindowMs = req.TimeWindowMs
	}
	if req.MinSources > 0 {
		config.MinSources = req.MinSources
	}
	if req.FieldWeights != nil {
		for field, weight := range req.FieldWeights {
			config.SetFieldWeight(field, weight)
		}
	}
	if req.SourceWeights != nil {
		for source, weight := range req.SourceWeights {
			config.SetSourceWeight(source, weight)
		}
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if config.ID == "" {
		if err := h.dataRepo.CreateFusionConfig(c.Request.Context(), config); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建融合配置失败", "details": err.Error()})
			return
		}
	} else {
		if err := h.dataRepo.UpdateFusionConfig(c.Request.Context(), config); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新融合配置失败", "details": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, config)
}

// GetFusionConfig 获取融合配置
// GET /api/v1/devices/:device_id/fusion-config
func (h *DeviceDataHandler) GetFusionConfig(c *gin.Context) {
	deviceID := c.Param("device_id")

	config, err := h.dataRepo.GetFusionConfigByDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "融合配置不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}
