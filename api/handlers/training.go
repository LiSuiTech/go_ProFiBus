package handlers

import (
	"net/http"
	"strconv"
	"time"

	trainingApp "go_ProFiBus/internal/application/training"
	trainingDomain "go_ProFiBus/internal/domain/training"
	"go_ProFiBus/pkg/interfaces"

	"github.com/gin-gonic/gin"
)

// TrainingHandler ML 模型訓練API處理器
type TrainingHandler struct {
	service *trainingApp.Service
}

// NewTrainingHandler 創建訓練處理器
func NewTrainingHandler(service *trainingApp.Service) *TrainingHandler {
	return &TrainingHandler{service: service}
}

// CreateTask 創建訓練任務
// POST /api/v1/training/tasks
func (h *TrainingHandler) CreateTask(c *gin.Context) {
	var req struct {
		ModelID        string                 `json:"model_id" binding:"required"`
		Name           string                 `json:"name"`
		Description    string                 `json:"description"`
		TrainingType   string                 `json:"training_type" binding:"required"`
		DataSourceType string                 `json:"data_source_type" binding:"required"`
		DataSourceIDs  []string               `json:"data_source_ids" binding:"required"`
		DataFields     []string               `json:"data_fields" binding:"required"`
		Epochs         int                    `json:"epochs"`
		BatchSize      int                    `json:"batch_size"`
		LearningRate   float64                `json:"learning_rate"`
		ValidationSplit float64               `json:"validation_split"`
		Hyperparameters map[string]interface{} `json:"hyperparameters"`
		TrainingConfig  map[string]interface{} `json:"training_config"`
		CreatedBy       string                 `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的請求數據", "details": err.Error()})
		return
	}

	trainingType := trainingDomain.TrainingType(req.TrainingType)
	if trainingType != trainingDomain.TrainingTypeSupervised &&
		trainingType != trainingDomain.TrainingTypeUnsupervised &&
		trainingType != trainingDomain.TrainingTypeReinforcement {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的訓練類型"})
		return
	}

	dataSourceType := trainingDomain.DataSourceType(req.DataSourceType)
	if dataSourceType != trainingDomain.DataSourceTypeDevice &&
		dataSourceType != trainingDomain.DataSourceTypeChannel &&
		dataSourceType != trainingDomain.DataSourceTypeFusion &&
		dataSourceType != trainingDomain.DataSourceTypeExternal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的數據源類型"})
		return
	}

	if len(req.DataSourceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一個數據源ID"})
		return
	}
	if len(req.DataFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一個數據字段"})
		return
	}

	task := trainingDomain.NewTrainingTask(req.ModelID, req.Name, trainingType, dataSourceType)
	task.Description = req.Description
	task.DataSourceIDs = req.DataSourceIDs
	task.DataFields = req.DataFields
	task.CreatedBy = req.CreatedBy

	if req.Epochs > 0 {
		task.Epochs = req.Epochs
	}
	if req.BatchSize > 0 {
		task.BatchSize = req.BatchSize
	}
	if req.LearningRate > 0 {
		task.LearningRate = req.LearningRate
	}
	if req.ValidationSplit > 0 && req.ValidationSplit < 1 {
		task.ValidationSplit = req.ValidationSplit
	}
	if req.Hyperparameters != nil {
		task.Hyperparameters = req.Hyperparameters
	}
	if req.TrainingConfig != nil {
		task.TrainingConfig = req.TrainingConfig
	}

	if err := h.service.CreateTask(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "創建訓練任務失敗", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// ListTasks 列出訓練任務
// GET /api/v1/training/tasks
func (h *TrainingHandler) ListTasks(c *gin.Context) {
	filters := interfaces.TrainingTaskFilters{}

	if modelID := c.Query("model_id"); modelID != "" {
		filters.ModelID = &modelID
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status := trainingDomain.TrainingStatus(statusStr)
		filters.Status = &status
	}
	if typeStr := c.Query("training_type"); typeStr != "" {
		t := trainingDomain.TrainingType(typeStr)
		filters.TrainingType = &t
	}
	if createdBy := c.Query("created_by"); createdBy != "" {
		filters.CreatedBy = &createdBy
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

	tasks, err := h.service.ListTasks(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查詢訓練任務列表失敗", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(tasks),
		"tasks": tasks,
	})
}

// GetTask 獲取訓練任務詳情
// GET /api/v1/training/tasks/:id
func (h *TrainingHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "訓練任務不存在", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// StartTask 啟動訓練任務
// POST /api/v1/training/tasks/:id/start
func (h *TrainingHandler) StartTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.service.StartTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "啟動訓練任務失敗", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// CancelTask 取消訓練任務
// POST /api/v1/training/tasks/:id/cancel
func (h *TrainingHandler) CancelTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.service.CancelTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "取消訓練任務失敗", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetTaskHistory 獲取訓練歷史
// GET /api/v1/training/tasks/:id/history
func (h *TrainingHandler) GetTaskHistory(c *gin.Context) {
	taskID := c.Param("id")

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	history, err := h.service.GetHistory(c.Request.Context(), taskID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查詢訓練歷史失敗", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(history),
		"history": history,
	})
}

