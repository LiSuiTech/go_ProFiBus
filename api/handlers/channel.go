package handlers

import (
	"go_ProFiBus/internal/application/channel"
	channelDomain "go_ProFiBus/internal/domain/channel"
	"go_ProFiBus/pkg/interfaces"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChannelHandler 采集通道处理器
type ChannelHandler struct {
	repo    interfaces.ChannelRepository
	manager *channel.Manager
}

// NewChannelHandler 创建采集通道处理器
func NewChannelHandler(repo interfaces.ChannelRepository, manager *channel.Manager) *ChannelHandler {
	return &ChannelHandler{
		repo:    repo,
		manager: manager,
	}
}

// GetProtocols 获取支持的协议列表
// @Summary 获取支持的协议列表
// @Tags channels
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/protocols [get]
func (h *ChannelHandler) GetProtocols(c *gin.Context) {
	protocols := channelDomain.GetSupportedProtocols()
	c.JSON(http.StatusOK, gin.H{
		"protocols": protocols,
	})
}

// GetChannels 获取所有通道
// @Summary 获取所有采集通道
// @Tags channels
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/channels [get]
func (h *ChannelHandler) GetChannels(c *gin.Context) {
	channels, err := h.repo.GetChannels(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channels": channels,
		"total":    len(channels),
	})
}

// GetChannel 获取通道详情
// @Summary 获取单个通道详情
// @Tags channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {object} channelDomain.Channel
// @Router /api/v1/channels/{id} [get]
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	id := c.Param("id")

	ch, err := h.repo.GetChannel(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	c.JSON(http.StatusOK, ch)
}

// CreateChannel 创建通道
// @Summary 创建新的采集通道
// @Tags channels
// @Accept json
// @Produce json
// @Param channel body channelDomain.Channel true "Channel"
// @Success 201 {object} channelDomain.Channel
// @Router /api/v1/channels [post]
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var ch channelDomain.Channel
	if err := c.ShouldBindJSON(&ch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置初始状态
	if ch.Status == "" {
		ch.Status = channelDomain.StatusStopped
	}
	if ch.Enabled == false {
		ch.Enabled = true
	}

	if err := h.repo.CreateChannel(c.Request.Context(), &ch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ch)
}

// UpdateChannel 更新通道
// @Summary 更新采集通道
// @Tags channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param channel body channelDomain.Channel true "Channel"
// @Success 200 {object} channelDomain.Channel
// @Router /api/v1/channels/{id} [put]
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	id := c.Param("id")

	var ch channelDomain.Channel
	if err := c.ShouldBindJSON(&ch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch.ID = id
	if err := h.repo.UpdateChannel(c.Request.Context(), &ch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ch)
}

// DeleteChannel 删除通道
// @Summary 删除采集通道
// @Tags channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/channels/{id} [delete]
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.DeleteChannel(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "channel deleted successfully"})
}

// StartChannel 启动通道
// @Summary 启动采集通道
// @Tags channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/channels/{id}/start [post]
func (h *ChannelHandler) StartChannel(c *gin.Context) {
	id := c.Param("id")

	// TODO: 实现实际的通道启动逻辑
	// 这里需要与采集器集成

	if err := h.repo.UpdateChannelStatus(c.Request.Context(), id, channelDomain.StatusRunning); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 使用通道管理器启动通道
	if err := h.manager.StartChannel(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "channel started successfully",
		"status":  string(channelDomain.StatusRunning),
	})
}

// StopChannel 停止通道
// @Summary 停止采集通道
// @Tags channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/channels/{id}/stop [post]
func (h *ChannelHandler) StopChannel(c *gin.Context) {
	id := c.Param("id")

	// TODO: 实现实际的通道停止逻辑
	// 这里需要与采集器集成

	if err := h.repo.UpdateChannelStatus(c.Request.Context(), id, channelDomain.StatusStopped); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 使用通道管理器停止通道
	if err := h.manager.StopChannel(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "channel stopped successfully",
		"status":  string(channelDomain.StatusStopped),
	})
}

// GetChannelPoints 获取通道的点位
// @Summary 获取通道的所有点位
// @Tags channels
// @Produce json
// @Param id path string true "Channel ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/channels/{id}/points [get]
func (h *ChannelHandler) GetChannelPoints(c *gin.Context) {
	channelID := c.Param("id")

	points, err := h.repo.GetPointsByChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"points": points,
		"total":  len(points),
	})
}

// CreatePoint 创建点位
// @Summary 创建采集点位
// @Tags channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param point body channelDomain.Point true "Point"
// @Success 201 {object} channelDomain.Point
// @Router /api/v1/channels/{id}/points [post]
func (h *ChannelHandler) CreatePoint(c *gin.Context) {
	channelID := c.Param("id")

	var point channelDomain.Point
	if err := c.ShouldBindJSON(&point); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	point.ChannelID = channelID
	if point.Enabled == false {
		point.Enabled = true
	}
	if point.Scale == 0 {
		point.Scale = 1.0
	}

	if err := h.repo.CreatePoint(c.Request.Context(), &point); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, point)
}

// UpdatePoint 更新点位
// @Summary 更新采集点位
// @Tags channels
// @Accept json
// @Produce json
// @Param id path string true "Channel ID"
// @Param point_id path string true "Point ID"
// @Param point body channelDomain.Point true "Point"
// @Success 200 {object} channelDomain.Point
// @Router /api/v1/channels/{id}/points/{point_id} [put]
func (h *ChannelHandler) UpdatePoint(c *gin.Context) {
	pointID := c.Param("point_id")

	var point channelDomain.Point
	if err := c.ShouldBindJSON(&point); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	point.ID = pointID
	if err := h.repo.UpdatePoint(c.Request.Context(), &point); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, point)
}

// DeletePoint 删除点位
// @Summary 删除采集点位
// @Tags channels
// @Produce json
// @Param id path string true "Channel ID"
// @Param point_id path string true "Point ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/channels/{id}/points/{point_id} [delete]
func (h *ChannelHandler) DeletePoint(c *gin.Context) {
	pointID := c.Param("point_id")

	if err := h.repo.DeletePoint(c.Request.Context(), pointID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "point deleted successfully"})
}
