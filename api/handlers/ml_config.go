package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go_ProFiBus/fusion"
	"go_ProFiBus/internal/application/processor"
	"go_ProFiBus/internal/infrastructure/analyzer"
)

// MLConfigHandler ML配置API处理器
type MLConfigHandler struct {
	analyzers        map[string]*analyzer.MLAnalyzer
	fusionProcessors map[string]*processor.FusionProcessor
}

// NewMLConfigHandler 创建ML配置处理器
func NewMLConfigHandler() *MLConfigHandler {
	return &MLConfigHandler{
		analyzers:        make(map[string]*analyzer.MLAnalyzer),
		fusionProcessors: make(map[string]*processor.FusionProcessor),
	}
}

// RegisterRoutes 注册路由
func (h *MLConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	// ML分析器配置
	router.GET("/ml/analyzers", h.ListAnalyzers)
	router.POST("/ml/analyzers", h.CreateAnalyzer)
	router.GET("/ml/analyzers/:id", h.GetAnalyzer)
	router.PUT("/ml/analyzers/:id", h.UpdateAnalyzer)
	router.DELETE("/ml/analyzers/:id", h.DeleteAnalyzer)
	router.GET("/ml/analyzers/:id/stats", h.GetAnalyzerStats)

	// 融合处理器配置
	router.GET("/ml/fusion", h.ListFusionProcessors)
	router.POST("/ml/fusion", h.CreateFusionProcessor)
	router.GET("/ml/fusion/:id", h.GetFusionProcessor)
	router.PUT("/ml/fusion/:id", h.UpdateFusionProcessor)
	router.DELETE("/ml/fusion/:id", h.DeleteFusionProcessor)

	// 配置Schema
	router.GET("/ml/schema/analyzer", h.GetAnalyzerSchema)
	router.GET("/ml/schema/fusion", h.GetFusionSchema)

	// 模型管理
	router.GET("/ml/models", h.ListModels)
	router.POST("/ml/models/:id/upload", h.UploadModel)
}

// === ML分析器API ===

// ListAnalyzers 获取所有ML分析器
func (h *MLConfigHandler) ListAnalyzers(c *gin.Context) {
	analyzers := make([]map[string]interface{}, 0)

	for id, analyzer := range h.analyzers {
		stats := analyzer.GetStats()
		analyzers = append(analyzers, map[string]interface{}{
			"id":             id,
			"name":           analyzer.GetName(),
			"type":           analyzer.GetType(),
			"threshold":      analyzer.GetThreshold(),
			"anomaly_rate":   analyzer.GetAnomalyRate(),
			"total_samples":  stats.TotalSamples,
			"anomalies":      stats.AnomaliesDetected,
			"accuracy":       stats.ModelAccuracy,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"analyzers": analyzers,
		"count":     len(analyzers),
	})
}

// CreateAnalyzer 创建ML分析器
func (h *MLConfigHandler) CreateAnalyzer(c *gin.Context) {
	var req struct {
		ID                  string                 `json:"id" binding:"required"`
		Name                string                 `json:"name" binding:"required"`
		ModelName           string                 `json:"model_name" binding:"required"`
		ModelType           string                 `json:"model_type"`
		ModelPath           string                 `json:"model_path"`
		AnomalyThreshold    float64                `json:"anomaly_threshold"`
		ConfidenceMin       float64                `json:"confidence_min"`
		UseFeatureExtractor bool                   `json:"use_feature_extractor"`
		Features            []string               `json:"features"`
		ModelConfig         map[string]interface{} `json:"model_config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 检查ID是否已存在
	if _, exists := h.analyzers[req.ID]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Analyzer with this ID already exists"})
		return
	}

	// 创建分析器
	mlAnalyzer := analyzer.NewMLAnalyzer(req.Name, req.ModelName)

	// 配置分析器
	config := map[string]interface{}{
		"model_name":            req.ModelName,
		"model_type":            req.ModelType,
		"model_path":            req.ModelPath,
		"anomaly_threshold":     req.AnomalyThreshold,
		"confidence_min":        req.ConfidenceMin,
		"use_feature_extractor": req.UseFeatureExtractor,
		"features":              req.Features,
		"model_config":          req.ModelConfig,
	}

	if err := mlAnalyzer.Configure(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure analyzer: " + err.Error()})
		return
	}

	// 保存分析器
	h.analyzers[req.ID] = mlAnalyzer

	c.JSON(http.StatusCreated, gin.H{
		"id":      req.ID,
		"name":    req.Name,
		"message": "ML Analyzer created successfully",
	})
}

// GetAnalyzer 获取单个ML分析器
func (h *MLConfigHandler) GetAnalyzer(c *gin.Context) {
	id := c.Param("id")

	analyzer, exists := h.analyzers[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analyzer not found"})
		return
	}

	stats := analyzer.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"id":           id,
		"name":         analyzer.GetName(),
		"type":         analyzer.GetType(),
		"threshold":    analyzer.GetThreshold(),
		"anomaly_rate": analyzer.GetAnomalyRate(),
		"stats": gin.H{
			"total_samples":       stats.TotalSamples,
			"anomalies_detected":  stats.AnomaliesDetected,
			"avg_inference_time":  stats.AvgInferenceTime.String(),
			"last_inference_time": stats.LastInferenceTime.String(),
			"model_accuracy":      stats.ModelAccuracy,
		},
	})
}

// UpdateAnalyzer 更新ML分析器配置
func (h *MLConfigHandler) UpdateAnalyzer(c *gin.Context) {
	id := c.Param("id")

	analyzer, exists := h.analyzers[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analyzer not found"})
		return
	}

	var req struct {
		AnomalyThreshold float64 `json:"anomaly_threshold"`
		ConfidenceMin    float64 `json:"confidence_min"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 更新配置
	if req.AnomalyThreshold > 0 {
		analyzer.SetThreshold(req.AnomalyThreshold)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Analyzer updated successfully",
	})
}

// DeleteAnalyzer 删除ML分析器
func (h *MLConfigHandler) DeleteAnalyzer(c *gin.Context) {
	id := c.Param("id")

	if _, exists := h.analyzers[id]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analyzer not found"})
		return
	}

	// 关闭并删除
	h.analyzers[id].Close()
	delete(h.analyzers, id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Analyzer deleted successfully",
	})
}

// GetAnalyzerStats 获取分析器统计信息
func (h *MLConfigHandler) GetAnalyzerStats(c *gin.Context) {
	id := c.Param("id")

	analyzer, exists := h.analyzers[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analyzer not found"})
		return
	}

	stats := analyzer.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"total_samples":          stats.TotalSamples,
		"anomalies_detected":     stats.AnomaliesDetected,
		"anomaly_rate":           analyzer.GetAnomalyRate(),
		"avg_inference_time_ms":  stats.AvgInferenceTime.Milliseconds(),
		"last_inference_time_ms": stats.LastInferenceTime.Milliseconds(),
		"model_accuracy":         stats.ModelAccuracy,
	})
}

// === 融合处理器API ===

// ListFusionProcessors 获取所有融合处理器
func (h *MLConfigHandler) ListFusionProcessors(c *gin.Context) {
	processors := make([]map[string]interface{}, 0)

	for id, proc := range h.fusionProcessors {
		processors = append(processors, map[string]interface{}{
			"id":   id,
			"name": proc.GetName(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"processors": processors,
		"count":      len(processors),
	})
}

// CreateFusionProcessor 创建融合处理器
func (h *MLConfigHandler) CreateFusionProcessor(c *gin.Context) {
	var req struct {
		ID            string             `json:"id" binding:"required"`
		Name          string             `json:"name" binding:"required"`
		Strategy      string             `json:"strategy"`
		SourceWeights map[string]float64 `json:"source_weights"`
		TimeWindow    string             `json:"time_window"`
		BufferSize    int                `json:"buffer_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 检查ID是否已存在
	if _, exists := h.fusionProcessors[req.ID]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Fusion processor with this ID already exists"})
		return
	}

	// 解析策略
	strategy := parseStrategyFromString(req.Strategy)

	// 创建融合处理器
	fusionProc := processor.NewFusionProcessor(req.Name, strategy)

	// 设置权重
	for sourceID, weight := range req.SourceWeights {
		fusionProc.SetSourceWeight(sourceID, weight)
	}

	// 保存
	h.fusionProcessors[req.ID] = fusionProc

	c.JSON(http.StatusCreated, gin.H{
		"id":      req.ID,
		"name":    req.Name,
		"message": "Fusion processor created successfully",
	})
}

// GetFusionProcessor 获取单个融合处理器
func (h *MLConfigHandler) GetFusionProcessor(c *gin.Context) {
	id := c.Param("id")

	proc, exists := h.fusionProcessors[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fusion processor not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   id,
		"name": proc.GetName(),
	})
}

// UpdateFusionProcessor 更新融合处理器
func (h *MLConfigHandler) UpdateFusionProcessor(c *gin.Context) {
	id := c.Param("id")

	proc, exists := h.fusionProcessors[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fusion processor not found"})
		return
	}

	var req struct {
		SourceWeights map[string]float64 `json:"source_weights"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 更新权重
	for sourceID, weight := range req.SourceWeights {
		proc.SetSourceWeight(sourceID, weight)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Fusion processor updated successfully",
	})
}

// DeleteFusionProcessor 删除融合处理器
func (h *MLConfigHandler) DeleteFusionProcessor(c *gin.Context) {
	id := c.Param("id")

	if _, exists := h.fusionProcessors[id]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fusion processor not found"})
		return
	}

	delete(h.fusionProcessors, id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Fusion processor deleted successfully",
	})
}

// === 配置Schema ===

// GetAnalyzerSchema 获取ML分析器配置Schema (用于前端表单生成)
func (h *MLConfigHandler) GetAnalyzerSchema(c *gin.Context) {
	schema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title":   "ML Analyzer Configuration",
		"type":    "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"title":       "分析器ID",
				"description": "唯一标识符",
				"minLength":   1,
			},
			"name": map[string]interface{}{
				"type":        "string",
				"title":       "分析器名称",
				"description": "用于显示的名称",
				"minLength":   1,
			},
			"model_name": map[string]interface{}{
				"type":        "string",
				"title":       "模型名称",
				"description": "AI模型的名称",
			},
			"model_type": map[string]interface{}{
				"type":        "string",
				"title":       "模型类型",
				"description": "选择AI模型类型",
				"enum": []string{
					"neural_network",
					"linear_regression",
					"logistic_regression",
					"decision_tree",
					"svm",
					"knn",
					"custom",
				},
				"default": "neural_network",
			},
			"model_path": map[string]interface{}{
				"type":        "string",
				"title":       "模型路径",
				"description": "训练好的模型文件路径",
			},
			"anomaly_threshold": map[string]interface{}{
				"type":        "number",
				"title":       "异常阈值",
				"description": "超过此分数视为异常 (0-1)",
				"minimum":     0.0,
				"maximum":     1.0,
				"default":     0.7,
			},
			"confidence_min": map[string]interface{}{
				"type":        "number",
				"title":       "最低置信度",
				"description": "低于此置信度的结果将被忽略 (0-1)",
				"minimum":     0.0,
				"maximum":     1.0,
				"default":     0.5,
			},
			"use_feature_extractor": map[string]interface{}{
				"type":        "boolean",
				"title":       "使用特征提取器",
				"description": "是否自动提取特征",
				"default":     true,
			},
			"features": map[string]interface{}{
				"type":        "array",
				"title":       "特征列表",
				"description": "要提取的特征",
				"items": map[string]interface{}{
					"type": "string",
					"enum": []string{
						"numeric_mean",
						"numeric_stddev",
						"numeric_min",
						"numeric_max",
						"rate_of_change",
						"trend",
						"variance",
						"quality",
						"confidence",
						"source_count",
					},
				},
				"default": []string{"numeric_mean", "numeric_stddev", "quality", "confidence"},
			},
		},
		"required": []string{"id", "name", "model_name", "model_type"},
	}

	c.JSON(http.StatusOK, schema)
}

// GetFusionSchema 获取融合处理器配置Schema
func (h *MLConfigHandler) GetFusionSchema(c *gin.Context) {
	schema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title":   "Fusion Processor Configuration",
		"type":    "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"title":       "处理器ID",
				"description": "唯一标识符",
				"minLength":   1,
			},
			"name": map[string]interface{}{
				"type":        "string",
				"title":       "处理器名称",
				"minLength":   1,
			},
			"strategy": map[string]interface{}{
				"type":        "string",
				"title":       "融合策略",
				"description": "选择数据融合算法",
				"enum": []string{
					"average",
					"weighted",
					"kalman",
					"bayesian",
					"dempster_shafer",
					"time_sync",
					"interpolation",
					"extrapolation",
					"moving_average",
					"exponential_sma",
				},
				"default": "weighted",
			},
			"source_weights": map[string]interface{}{
				"type":        "object",
				"title":       "数据源权重",
				"description": "各数据源的权重 (键为数据源ID, 值为权重0-1)",
				"additionalProperties": map[string]interface{}{
					"type":    "number",
					"minimum": 0.0,
					"maximum": 1.0,
				},
			},
			"time_window": map[string]interface{}{
				"type":        "string",
				"title":       "时间窗口",
				"description": "数据保留时间窗口 (如: 1s, 500ms, 1m)",
				"default":     "1s",
				"pattern":     "^[0-9]+(ns|us|ms|s|m|h)$",
			},
			"buffer_size": map[string]interface{}{
				"type":        "integer",
				"title":       "缓冲区大小",
				"description": "历史数据缓冲区大小",
				"minimum":     1,
				"maximum":     10000,
				"default":     100,
			},
		},
		"required": []string{"id", "name", "strategy"},
	}

	c.JSON(http.StatusOK, schema)
}

// === 模型管理 ===

// ListModels 获取可用模型列表
func (h *MLConfigHandler) ListModels(c *gin.Context) {
	models := []map[string]interface{}{
		{
			"id":          "anomaly-nn-v1",
			"name":        "异常检测神经网络 v1",
			"type":        "neural_network",
			"description": "用于工业传感器异常检测的神经网络模型",
			"accuracy":    0.92,
			"input_size":  10,
			"output_size": 2,
		},
		{
			"id":          "anomaly-svm-v1",
			"name":        "异常检测SVM v1",
			"type":        "svm",
			"description": "基于支持向量机的异常检测模型",
			"accuracy":    0.88,
			"input_size":  10,
			"output_size": 1,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"models": models,
		"count":  len(models),
	})
}

// UploadModel 上传模型文件
func (h *MLConfigHandler) UploadModel(c *gin.Context) {
	id := c.Param("id")

	// 解析multipart form
	file, err := c.FormFile("model")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No model file provided"})
		return
	}

	// TODO: 保存模型文件到存储
	_ = file

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Model uploaded successfully",
	})
}

// === 辅助函数 ===

func parseStrategyFromString(strategy string) fusion.FusionStrategy {
	switch strategy {
	case "average":
		return fusion.StrategyAverage
	case "weighted":
		return fusion.StrategyWeighted
	case "kalman":
		return fusion.StrategyKalman
	case "bayesian":
		return fusion.StrategyBayesian
	case "dempster_shafer":
		return fusion.StrategyDempsterShafer
	case "time_sync":
		return fusion.StrategyTimeSync
	case "interpolation":
		return fusion.StrategyInterpolation
	case "extrapolation":
		return fusion.StrategyExtrapolation
	case "moving_average":
		return fusion.StrategyMovingAverage
	case "exponential_sma":
		return fusion.StrategyExponentialSMA
	default:
		return fusion.StrategyWeighted
	}
}
