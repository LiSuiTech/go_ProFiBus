package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	deviceDomain "go_ProFiBus/internal/domain/device"
	"go_ProFiBus/pkg/interfaces"
)

// DeviceHandler 设备管理API处理器
type DeviceHandler struct {
	repo interfaces.DeviceRepository
}

// NewDeviceHandler 创建设备处理器
func NewDeviceHandler(repo interfaces.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

// CreateDevice 创建设备
// POST /api/v1/devices
func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		Description string                 `json:"description"`
		Type        string                 `json:"type" binding:"required"`
		Location    struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
			Z float64 `json:"z"`
		} `json:"location"`
		Area     string                 `json:"area"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 验证设备类型
	deviceType := deviceDomain.DeviceType(req.Type)
	if deviceType != deviceDomain.DeviceTypePLC &&
		deviceType != deviceDomain.DeviceTypeSensor &&
		deviceType != deviceDomain.DeviceTypeInstrument &&
		deviceType != deviceDomain.DeviceTypeSmartDevice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的设备类型"})
		return
	}

	// 创建设备
	device := deviceDomain.NewDevice(uuid.New().String(), req.Name, deviceType)
	device.Description = req.Description
	device.SetLocation(req.Location.X, req.Location.Y, req.Location.Z)
	device.Area = req.Area
	if req.Metadata != nil {
		device.Metadata = req.Metadata
	}

	if err := h.repo.Create(c.Request.Context(), device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建设备失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, device)
}

// GetDevice 获取设备详情
// GET /api/v1/devices/:id
func (h *DeviceHandler) GetDevice(c *gin.Context) {
	deviceID := c.Param("id")

	device, err := h.repo.GetByID(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在", "details": err.Error()})
		return
	}

	// 获取关联的通道ID列表
	channelIDs, _ := h.repo.GetChannels(c.Request.Context(), deviceID)

	c.JSON(http.StatusOK, gin.H{
		"device":      device,
		"channel_ids": channelIDs,
	})
}

// ListDevices 列出设备
// GET /api/v1/devices
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	filters := interfaces.DeviceFilters{}

	// 解析查询参数
	if typeStr := c.Query("type"); typeStr != "" {
		deviceType := deviceDomain.DeviceType(typeStr)
		filters.Type = &deviceType
	}

	if statusStr := c.Query("status"); statusStr != "" {
		status := deviceDomain.DeviceStatus(statusStr)
		filters.Status = &status
	}

	if area := c.Query("area"); area != "" {
		filters.Area = &area
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

	devices, err := h.repo.List(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(devices),
		"devices": devices,
	})
}

// UpdateDevice 更新设备
// PUT /api/v1/devices/:id
func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	deviceID := c.Param("id")

	// 获取现有设备
	device, err := h.repo.GetByID(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在", "details": err.Error()})
		return
	}

	var req struct {
		Name        *string                `json:"name"`
		Description *string                `json:"description"`
		Type        *string                `json:"type"`
		Location    *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
			Z float64 `json:"z"`
		} `json:"location"`
		Area     *string                `json:"area"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 更新字段
	if req.Name != nil {
		device.Name = *req.Name
	}
	if req.Description != nil {
		device.Description = *req.Description
	}
	if req.Type != nil {
		deviceType := deviceDomain.DeviceType(*req.Type)
		device.Type = deviceType
	}
	if req.Location != nil {
		device.SetLocation(req.Location.X, req.Location.Y, req.Location.Z)
	}
	if req.Area != nil {
		device.Area = *req.Area
	}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			device.SetMetadata(k, v)
		}
	}

	if err := h.repo.Update(c.Request.Context(), device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新设备失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, device)
}

// DeleteDevice 删除设备
// DELETE /api/v1/devices/:id
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	deviceID := c.Param("id")

	if err := h.repo.Delete(c.Request.Context(), deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除设备失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "设备删除成功"})
}

// UpdateDeviceStatus 更新设备状态
// PATCH /api/v1/devices/:id/status
func (h *DeviceHandler) UpdateDeviceStatus(c *gin.Context) {
	deviceID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	status := deviceDomain.DeviceStatus(req.Status)
	if status != deviceDomain.DeviceStatusOnline &&
		status != deviceDomain.DeviceStatusOffline &&
		status != deviceDomain.DeviceStatusFault &&
		status != deviceDomain.DeviceStatusMaintenance {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的设备状态"})
		return
	}

	if err := h.repo.UpdateStatus(c.Request.Context(), deviceID, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新设备状态失败", "details": err.Error()})
		return
	}

	device, _ := h.repo.GetByID(c.Request.Context(), deviceID)
	c.JSON(http.StatusOK, device)
}

// AddDeviceChannel 添加设备与通道的关联
// POST /api/v1/devices/:id/channels
func (h *DeviceHandler) AddDeviceChannel(c *gin.Context) {
	deviceID := c.Param("id")

	var req struct {
		ChannelID string `json:"channel_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if err := h.repo.AddChannel(c.Request.Context(), deviceID, req.ChannelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加通道关联失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "通道关联添加成功"})
}

// RemoveDeviceChannel 移除设备与通道的关联
// DELETE /api/v1/devices/:id/channels/:channel_id
func (h *DeviceHandler) RemoveDeviceChannel(c *gin.Context) {
	deviceID := c.Param("id")
	channelID := c.Param("channel_id")

	if err := h.repo.RemoveChannel(c.Request.Context(), deviceID, channelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移除通道关联失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "通道关联移除成功"})
}

// GetDeviceLayout 获取设备布局
// GET /api/v1/devices/layout
func (h *DeviceHandler) GetDeviceLayout(c *gin.Context) {
	area := c.Query("area")

	filters := interfaces.DeviceFilters{}
	if area != "" {
		filters.Area = &area
	}
	filters.Limit = 1000

	devices, err := h.repo.List(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询设备布局失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
	})
}
