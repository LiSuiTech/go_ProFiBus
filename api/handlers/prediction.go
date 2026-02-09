package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	predictionDomain "go_ProFiBus/internal/domain/prediction"
	"go_ProFiBus/pkg/interfaces"
)

// PredictionHandler 预测分析API处理器
type PredictionHandler struct {
	repo interfaces.PredictionRepository
}

// NewPredictionHandler 创建预测处理器
func NewPredictionHandler(repo interfaces.PredictionRepository) *PredictionHandler {
	return &PredictionHandler{repo: repo}
}

// CreatePrediction 创建预测结果
// POST /api/v1/predictions
func (h *PredictionHandler) CreatePrediction(c *gin.Context) {
	var req struct {
		ModelID        string    `json:"model_id" binding:"required"`
		DeviceID       string    `json:"device_id"`
		ChannelID      string    `json:"channel_id"`
		PredictionType string    `json:"prediction_type" binding:"required"`
		FieldName      string    `json:"field_name"`
		PredictedValue float64   `json:"predicted_value" binding:"required"`
		Confidence     float64   `json:"confidence"`
		TimeRangeStart time.Time `json:"time_range_start" binding:"required"`
		TimeRangeEnd   time.Time `json:"time_range_end" binding:"required"`
		Metadata       map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	predictionType := predictionDomain.PredictionType(req.PredictionType)
	if predictionType != predictionDomain.PredictionTypeForecast &&
		predictionType != predictionDomain.PredictionTypeAnomaly &&
		predictionType != predictionDomain.PredictionTypeTrend &&
		predictionType != predictionDomain.PredictionTypePerformance {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预测类型"})
		return
	}

	prediction := predictionDomain.NewPrediction(req.ModelID, req.DeviceID, predictionType, req.PredictedValue)
	prediction.ChannelID = req.ChannelID
	prediction.FieldName = req.FieldName
	prediction.SetConfidence(req.Confidence)
	prediction.SetTimeRange(req.TimeRangeStart, req.TimeRangeEnd)
	if req.Metadata != nil {
		prediction.Metadata = req.Metadata
	}

	if err := h.repo.CreatePrediction(c.Request.Context(), prediction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建预测结果失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, prediction)
}

// ListPredictions 获取预测结果列表
// GET /api/v1/predictions
func (h *PredictionHandler) ListPredictions(c *gin.Context) {
	filters := interfaces.PredictionFilters{}

	if modelID := c.Query("model_id"); modelID != "" {
		filters.ModelID = &modelID
	}
	if deviceID := c.Query("device_id"); deviceID != "" {
		filters.DeviceID = &deviceID
	}
	if channelID := c.Query("channel_id"); channelID != "" {
		filters.ChannelID = &channelID
	}
	if typeStr := c.Query("type"); typeStr != "" {
		predictionType := predictionDomain.PredictionType(typeStr)
		filters.PredictionType = &predictionType
	}
	if startStr := c.Query("start"); startStr != "" {
		if start, err := time.Parse(time.RFC3339, startStr); err == nil {
			filters.StartTime = &start
		}
	}
	if endStr := c.Query("end"); endStr != "" {
		if end, err := time.Parse(time.RFC3339, endStr); err == nil {
			filters.EndTime = &end
		}
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

	predictions, err := h.repo.ListPredictions(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询预测结果列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":       len(predictions),
		"predictions": predictions,
	})
}

// GetPrediction 获取预测结果详情
// GET /api/v1/predictions/:id
func (h *PredictionHandler) GetPrediction(c *gin.Context) {
	predictionID := c.Param("id")

	prediction, err := h.repo.GetPredictionByID(c.Request.Context(), predictionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预测结果不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// Forecast 趋势预测
// POST /api/v1/predictions/forecast
func (h *PredictionHandler) Forecast(c *gin.Context) {
	var req struct {
		ModelID       string    `json:"model_id" binding:"required"`
		DeviceID      string    `json:"device_id" binding:"required"`
		FieldName     string    `json:"field_name" binding:"required"`
		TimeRangeEnd  time.Time `json:"time_range_end" binding:"required"`
		ForecastSteps int       `json:"forecast_steps"` // 预测步数
		InputData     []float64 `json:"input_data"`     // 输入数据（可选）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	// 检查模型是否存在且已部署
	model, err := h.repo.GetModelByID(c.Request.Context(), req.ModelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预测模型不存在", "details": err.Error()})
		return
	}

	if model.Status != predictionDomain.ModelStatusDeployed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "模型未部署，无法使用"})
		return
	}

	// TODO: 使用ML模型管理器进行实际预测
	// 这里返回模拟数据
	predictions := make([]*predictionDomain.Prediction, 0)
	stepDuration := time.Hour // 默认每小时一步
	if req.ForecastSteps == 0 {
		req.ForecastSteps = 24 // 默认预测24小时
	}

	// 模拟预测值（实际应该调用ML模型）
	baseValue := 100.0
	if len(req.InputData) > 0 {
		// 使用输入数据的平均值作为基准
		sum := 0.0
		for _, v := range req.InputData {
			sum += v
		}
		baseValue = sum / float64(len(req.InputData))
	}

	for i := 0; i < req.ForecastSteps; i++ {
		// 简单的线性趋势模拟
		predictedValue := baseValue + float64(i)*0.5
		
		prediction := predictionDomain.NewPrediction(
			req.ModelID,
			req.DeviceID,
			predictionDomain.PredictionTypeForecast,
			predictedValue,
		)
		prediction.FieldName = req.FieldName
		
		// 根据模型准确度设置置信度
		confidence := 0.85
		if model.Accuracy != nil {
			confidence = *model.Accuracy
		}
		prediction.SetConfidence(confidence)
		
		startTime := req.TimeRangeEnd.Add(time.Duration(i) * stepDuration)
		endTime := startTime.Add(stepDuration)
		prediction.SetTimeRange(startTime, endTime)
		predictions = append(predictions, prediction)
	}

	// 保存预测结果
	for _, pred := range predictions {
		_ = h.repo.CreatePrediction(c.Request.Context(), pred)
	}

	c.JSON(http.StatusOK, gin.H{
		"predictions": predictions,
		"message":     "预测完成",
		"model_name":  model.Name,
		"model_type":  model.Type,
	})
}

// GetPredictionHistory 获取预测历史
// GET /api/v1/predictions/history
func (h *PredictionHandler) GetPredictionHistory(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id参数必填"})
		return
	}

	predictionTypeStr := c.DefaultQuery("type", "forecast")
	predictionType := predictionDomain.PredictionType(predictionTypeStr)

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	predictions, err := h.repo.GetPredictionsByDevice(c.Request.Context(), deviceID, predictionType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询预测历史失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":       len(predictions),
		"predictions": predictions,
	})
}

// CreateModel 创建预测模型
// POST /api/v1/predictions/models
func (h *PredictionHandler) CreateModel(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Type        string `json:"type" binding:"required"`
		Version     string `json:"version"`
		FilePath    string `json:"file_path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	modelType := predictionDomain.ModelType(req.Type)
	if modelType != predictionDomain.ModelTypeLinearRegression &&
		modelType != predictionDomain.ModelTypeNeuralNetwork &&
		modelType != predictionDomain.ModelTypeSVM &&
		modelType != predictionDomain.ModelTypeDecisionTree &&
		modelType != predictionDomain.ModelTypeLSTM &&
		modelType != predictionDomain.ModelTypeCustom {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模型类型"})
		return
	}

	model := predictionDomain.NewPredictionModel(uuid.New().String(), req.Name, modelType)
	model.Description = req.Description
	if req.Version != "" {
		model.Version = req.Version
	}
	model.FilePath = req.FilePath

	if err := h.repo.CreateModel(c.Request.Context(), model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建预测模型失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, model)
}

// ListModels 列出预测模型
// GET /api/v1/predictions/models
func (h *PredictionHandler) ListModels(c *gin.Context) {
	filters := interfaces.ModelFilters{}

	if typeStr := c.Query("type"); typeStr != "" {
		modelType := predictionDomain.ModelType(typeStr)
		filters.Type = &modelType
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status := predictionDomain.ModelStatus(statusStr)
		filters.Status = &status
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

	models, err := h.repo.ListModels(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询预测模型列表失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(models),
		"models": models,
	})
}

// GetModel 获取预测模型详情
// GET /api/v1/predictions/models/:id
func (h *PredictionHandler) GetModel(c *gin.Context) {
	modelID := c.Param("id")

	model, err := h.repo.GetModelByID(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预测模型不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model)
}

// UpdateModel 更新预测模型
// PUT /api/v1/predictions/models/:id
func (h *PredictionHandler) UpdateModel(c *gin.Context) {
	modelID := c.Param("id")

	model, err := h.repo.GetModelByID(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预测模型不存在", "details": err.Error()})
		return
	}

	var req struct {
		Name            *string  `json:"name"`
		Description     *string  `json:"description"`
		Version         *string  `json:"version"`
		FilePath        *string  `json:"file_path"`
		Status          *string  `json:"status"`
		Accuracy        *float64 `json:"accuracy"`
		TrainingSamples *int     `json:"training_samples"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "details": err.Error()})
		return
	}

	if req.Name != nil {
		model.Name = *req.Name
	}
	if req.Description != nil {
		model.Description = *req.Description
	}
	if req.Version != nil {
		model.Version = *req.Version
	}
	if req.FilePath != nil {
		model.FilePath = *req.FilePath
	}
	if req.Status != nil {
		model.Status = predictionDomain.ModelStatus(*req.Status)
	}
	if req.Accuracy != nil {
		model.SetAccuracy(*req.Accuracy)
	}
	if req.TrainingSamples != nil {
		model.TrainingSamples = *req.TrainingSamples
	}

	if err := h.repo.UpdateModel(c.Request.Context(), model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新预测模型失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model)
}

// DeployModel 部署模型
// POST /api/v1/predictions/models/:id/deploy
func (h *PredictionHandler) DeployModel(c *gin.Context) {
	modelID := c.Param("id")

	model, err := h.repo.GetModelByID(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预测模型不存在", "details": err.Error()})
		return
	}

	model.Deploy()

	if err := h.repo.UpdateModel(c.Request.Context(), model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "部署模型失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model)
}

// DeleteModel 删除预测模型
// DELETE /api/v1/predictions/models/:id
func (h *PredictionHandler) DeleteModel(c *gin.Context) {
	modelID := c.Param("id")

	if err := h.repo.DeleteModel(c.Request.Context(), modelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除预测模型失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "预测模型删除成功"})
}

// UploadModelFile 上传模型文件
// POST /api/v1/predictions/models/:id/upload
func (h *PredictionHandler) UploadModelFile(c *gin.Context) {
	modelID := c.Param("id")

	// 检查模型是否存在
	_, err := h.repo.GetModelByID(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预测模型不存在", "details": err.Error()})
		return
	}

	// 解析multipart form
	file, err := c.FormFile("model")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供模型文件", "details": err.Error()})
		return
	}

	// 创建模型存储目录
	modelDir := "models"
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建模型目录失败", "details": err.Error()})
		return
	}

	// 构建文件路径：models/{modelID}/{filename}
	modelPath := filepath.Join(modelDir, modelID, file.Filename)
	if err := os.MkdirAll(filepath.Dir(modelPath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建模型目录失败", "details": err.Error()})
		return
	}

	// 保存文件
	if err := c.SaveUploadedFile(file, modelPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存模型文件失败", "details": err.Error()})
		return
	}

	// 更新模型的文件路径
	model, _ := h.repo.GetModelByID(c.Request.Context(), modelID)
	model.FilePath = modelPath
	if err := h.repo.UpdateModel(c.Request.Context(), model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新模型路径失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         modelID,
		"model_path": modelPath,
		"filename":   file.Filename,
		"size":       file.Size,
		"message":    "模型文件上传成功",
	})
}
