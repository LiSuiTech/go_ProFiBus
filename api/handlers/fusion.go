package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	fusionDomain "go_ProFiBus/internal/domain/fusion"
	fusionApp "go_ProFiBus/internal/application/fusion"
	"go_ProFiBus/pkg/interfaces"
)

// FusionHandler 通用融合API处理器
type FusionHandler struct {
	repo         interfaces.FusionRepository
	fusionService *fusionApp.UniversalFusionService
}

// NewFusionHandler 创建融合处理器
func NewFusionHandler(repo interfaces.FusionRepository, fusionService *fusionApp.UniversalFusionService) *FusionHandler {
	return &FusionHandler{
		repo:         repo,
		fusionService: fusionService,
	}
}

// CreateDataSource 创建数据源
// POST /api/v1/fusion/data-sources
func (h *FusionHandler) CreateDataSource(c *gin.Context) {
	var req struct {
		SourceName  string                 `json:"source_name" binding:"required"`
		SourceType  string                 `json:"source_type" binding:"required"`
		DeviceID    string                 `json:"device_id"`
		ChannelID   string                 `json:"channel_id"`
		FieldName   string                 `json:"field_name"`
		SourceConfig map[string]interface{} `json:"source_config"`
		FusionWeight float64                `json:"fusion_weight"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	sourceType := fusionDomain.SourceType(req.SourceType)
	if sourceType != fusionDomain.SourceTypeDeviceField &&
		sourceType != fusionDomain.SourceTypeDevice &&
		sourceType != fusionDomain.SourceTypeChannel &&
		sourceType != fusionDomain.SourceTypeExternal &&
		sourceType != fusionDomain.SourceTypeCalculated {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的数据源类型"})
		return
	}

	source := fusionDomain.NewDataSource(uuid.New().String(), req.SourceName, sourceType)
	source.DeviceID = req.DeviceID
	source.ChannelID = req.ChannelID
	source.FieldName = req.FieldName
	if req.SourceConfig != nil {
		source.SourceConfig = req.SourceConfig
	}
	if req.FusionWeight > 0 {
		source.SetFusionWeight(req.FusionWeight)
	}

	if err := h.repo.CreateDataSource(c.Request.Context(), source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建数据源失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, source)
}

// ListDataSources 列出数据源
// GET /api/v1/fusion/data-sources
func (h *FusionHandler) ListDataSources(c *gin.Context) {
	filters := interfaces.DataSourceFilters{}

	if typeStr := c.Query("type"); typeStr != "" {
		sourceType := fusionDomain.SourceType(typeStr)
		filters.SourceType = &sourceType
	}
	if deviceID := c.Query("device_id"); deviceID != "" {
		filters.DeviceID = &deviceID
	}
	if channelID := c.Query("channel_id"); channelID != "" {
		filters.ChannelID = &channelID
	}
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		filters.Enabled = &enabled
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	filters.Limit = limit

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	filters.Offset = offset

	sources, err := h.repo.ListDataSources(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询数据源列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(sources),
		"sources": sources,
	})
}

// CreateFusionConfig 创建融合配置
// POST /api/v1/fusion/configs
func (h *FusionHandler) CreateFusionConfig(c *gin.Context) {
	var req struct {
		Name          string             `json:"name" binding:"required"`
		Description   string             `json:"description"`
		FusionStrategy string            `json:"fusion_strategy"`
		TimeWindowMs  int                `json:"time_window_ms"`
		MinSources    int                `json:"min_sources"`
		SourceWeights map[string]float64 `json:"source_weights"`
		FieldWeights  map[string]float64 `json:"field_weights"`
		OutputFields  []string           `json:"output_fields"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	config := fusionDomain.NewFusionConfig(uuid.New().String(), req.Name)
	config.Description = req.Description
	if req.FusionStrategy != "" {
		config.SetFusionStrategy(req.FusionStrategy)
	}
	if req.TimeWindowMs > 0 {
		config.TimeWindowMs = req.TimeWindowMs
	}
	if req.MinSources > 0 {
		config.MinSources = req.MinSources
	}
	if req.SourceWeights != nil {
		for sourceID, weight := range req.SourceWeights {
			config.SetSourceWeight(sourceID, weight)
		}
	}
	if req.FieldWeights != nil {
		for field, weight := range req.FieldWeights {
			config.SetFieldWeight(field, weight)
		}
	}
	if req.OutputFields != nil {
		config.OutputFields = req.OutputFields
	}

	if err := h.repo.CreateFusionConfig(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建融合配置失败", "details": err.Error()})
		return
	}

	// 加载配置到服务
	_ = h.fusionService.LoadFusionConfig(c.Request.Context(), config.ID)

	c.JSON(http.StatusCreated, config)
}

// ListFusionConfigs 列出融合配置
// GET /api/v1/fusion/configs
func (h *FusionHandler) ListFusionConfigs(c *gin.Context) {
	filters := interfaces.FusionConfigFilters{}

	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		filters.Enabled = &enabled
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	filters.Limit = limit

	configs, err := h.repo.ListFusionConfigs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询融合配置列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(configs),
		"configs": configs,
	})
}

// AddSourceToConfig 添加数据源到融合配置
// POST /api/v1/fusion/configs/:config_id/sources/:source_id
func (h *FusionHandler) AddSourceToConfig(c *gin.Context) {
	configID := c.Param("config_id")
	sourceID := c.Param("source_id")

	var req struct {
		Weight float64 `json:"weight"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Weight = 1.0 // 默认权重
	}

	if req.Weight <= 0 {
		req.Weight = 1.0
	}

	if err := h.repo.AddSourceToConfig(c.Request.Context(), configID, sourceID, req.Weight); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加数据源失败", "details": err.Error()})
		return
	}

	// 重新加载配置
	_ = h.fusionService.LoadFusionConfig(c.Request.Context(), configID)

	c.JSON(http.StatusOK, gin.H{"message": "数据源已添加到融合配置"})
}

// SubmitData 提交数据到数据源
// POST /api/v1/fusion/data-sources/:source_id/data
func (h *FusionHandler) SubmitData(c *gin.Context) {
	sourceID := c.Param("source_id")

	var req struct {
		Data    map[string]interface{} `json:"data" binding:"required"`
		Quality float64                `json:"quality"`
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

	if err := h.fusionService.SubmitData(c.Request.Context(), sourceID, req.Data, quality); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交数据失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "数据提交成功"})
}

// GetFusionResults 获取融合结果
// GET /api/v1/fusion/configs/:config_id/results
func (h *FusionHandler) GetFusionResults(c *gin.Context) {
	configID := c.Param("config_id")

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
		start = time.Now().Add(-24 * time.Hour)
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

	results, err := h.fusionService.GetFusionResults(c.Request.Context(), configID, start, end, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询融合结果失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(results),
		"results": results,
	})
}

// GetLatestFusionResult 获取最新融合结果
// GET /api/v1/fusion/configs/:config_id/results/latest
func (h *FusionHandler) GetLatestFusionResult(c *gin.Context) {
	configID := c.Param("config_id")

	result, err := h.fusionService.GetLatestFusionResult(c.Request.Context(), configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "融合结果不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
