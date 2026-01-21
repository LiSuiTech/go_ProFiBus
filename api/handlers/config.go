package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/go_profibus/internal/domain/config"
	"github.com/yourusername/go_profibus/pkg/interfaces"
)

// ConfigHandler handles configuration-related HTTP requests
type ConfigHandler struct {
	repository interfaces.ConfigRepository
	validator  *config.Validator
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(repository interfaces.ConfigRepository, validator *config.Validator) *ConfigHandler {
	return &ConfigHandler{
		repository: repository,
		validator:  validator,
	}
}

// ========== Rule Configuration Handlers ==========

// CreateRuleConfig creates a new rule configuration
// POST /api/v1/config/rules
func (h *ConfigHandler) CreateRuleConfig(c *gin.Context) {
	var ruleConfig interfaces.RuleConfig
	if err := c.ShouldBindJSON(&ruleConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Generate ID if not provided
	if ruleConfig.ID == "" {
		ruleConfig.ID = "rule-" + uuid.New().String()
	}

	// Set default version if not provided
	if ruleConfig.Version == "" {
		ruleConfig.Version = "1.0.0"
	}

	// Validate configuration
	if err := h.validator.ValidateRuleConfig(&ruleConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	// Create in repository
	if err := h.repository.CreateRuleConfig(c.Request.Context(), &ruleConfig); err != nil {
		if _, ok := err.(interfaces.ErrConfigExists); ok {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rule config: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ruleConfig)
}

// GetRuleConfig retrieves a rule configuration by ID
// GET /api/v1/config/rules/:id
func (h *ConfigHandler) GetRuleConfig(c *gin.Context) {
	id := c.Param("id")

	ruleConfig, err := h.repository.GetRuleConfig(c.Request.Context(), id)
	if err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get rule config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ruleConfig)
}

// ListRuleConfigs lists all rule configurations
// GET /api/v1/config/rules
func (h *ConfigHandler) ListRuleConfigs(c *gin.Context) {
	// Parse query parameters
	var enabled *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabledVal := enabledStr == "true"
		enabled = &enabledVal
	}

	ruleConfigs, err := h.repository.ListRuleConfigs(c.Request.Context(), enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rule configs: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rules": ruleConfigs,
		"count": len(ruleConfigs),
	})
}

// UpdateRuleConfig updates an existing rule configuration
// PUT /api/v1/config/rules/:id
func (h *ConfigHandler) UpdateRuleConfig(c *gin.Context) {
	id := c.Param("id")

	var ruleConfig interfaces.RuleConfig
	if err := c.ShouldBindJSON(&ruleConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Ensure ID matches URL parameter
	ruleConfig.ID = id

	// Validate configuration
	if err := h.validator.ValidateRuleConfig(&ruleConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	// Update in repository
	if err := h.repository.UpdateRuleConfig(c.Request.Context(), &ruleConfig); err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rule config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ruleConfig)
}

// DeleteRuleConfig deletes a rule configuration
// DELETE /api/v1/config/rules/:id
func (h *ConfigHandler) DeleteRuleConfig(c *gin.Context) {
	id := c.Param("id")

	if err := h.repository.DeleteRuleConfig(c.Request.Context(), id); err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rule config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule configuration deleted successfully"})
}

// ========== Analyzer Configuration Handlers ==========

// CreateAnalyzerConfig creates a new analyzer configuration
// POST /api/v1/config/analyzers
func (h *ConfigHandler) CreateAnalyzerConfig(c *gin.Context) {
	var analyzerConfig interfaces.AnalyzerConfig
	if err := c.ShouldBindJSON(&analyzerConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if analyzerConfig.ID == "" {
		analyzerConfig.ID = "analyzer-" + uuid.New().String()
	}

	if analyzerConfig.Version == "" {
		analyzerConfig.Version = "1.0.0"
	}

	if err := h.validator.ValidateAnalyzerConfig(&analyzerConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	if err := h.repository.CreateAnalyzerConfig(c.Request.Context(), &analyzerConfig); err != nil {
		if _, ok := err.(interfaces.ErrConfigExists); ok {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create analyzer config: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, analyzerConfig)
}

// GetAnalyzerConfig retrieves an analyzer configuration by ID
// GET /api/v1/config/analyzers/:id
func (h *ConfigHandler) GetAnalyzerConfig(c *gin.Context) {
	id := c.Param("id")

	analyzerConfig, err := h.repository.GetAnalyzerConfig(c.Request.Context(), id)
	if err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analyzer config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, analyzerConfig)
}

// ListAnalyzerConfigs lists all analyzer configurations
// GET /api/v1/config/analyzers
func (h *ConfigHandler) ListAnalyzerConfigs(c *gin.Context) {
	var enabled *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabledVal := enabledStr == "true"
		enabled = &enabledVal
	}

	analyzerConfigs, err := h.repository.ListAnalyzerConfigs(c.Request.Context(), enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list analyzer configs: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analyzers": analyzerConfigs,
		"count":     len(analyzerConfigs),
	})
}

// UpdateAnalyzerConfig updates an existing analyzer configuration
// PUT /api/v1/config/analyzers/:id
func (h *ConfigHandler) UpdateAnalyzerConfig(c *gin.Context) {
	id := c.Param("id")

	var analyzerConfig interfaces.AnalyzerConfig
	if err := c.ShouldBindJSON(&analyzerConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	analyzerConfig.ID = id

	if err := h.validator.ValidateAnalyzerConfig(&analyzerConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	if err := h.repository.UpdateAnalyzerConfig(c.Request.Context(), &analyzerConfig); err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update analyzer config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, analyzerConfig)
}

// DeleteAnalyzerConfig deletes an analyzer configuration
// DELETE /api/v1/config/analyzers/:id
func (h *ConfigHandler) DeleteAnalyzerConfig(c *gin.Context) {
	id := c.Param("id")

	if err := h.repository.DeleteAnalyzerConfig(c.Request.Context(), id); err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete analyzer config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "analyzer configuration deleted successfully"})
}

// ========== Processor Configuration Handlers ==========

// CreateProcessorConfig creates a new processor configuration
// POST /api/v1/config/processors
func (h *ConfigHandler) CreateProcessorConfig(c *gin.Context) {
	var processorConfig interfaces.ProcessorConfig
	if err := c.ShouldBindJSON(&processorConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if processorConfig.ID == "" {
		processorConfig.ID = "processor-" + uuid.New().String()
	}

	if processorConfig.Version == "" {
		processorConfig.Version = "1.0.0"
	}

	if err := h.validator.ValidateProcessorConfig(&processorConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	if err := h.repository.CreateProcessorConfig(c.Request.Context(), &processorConfig); err != nil {
		if _, ok := err.(interfaces.ErrConfigExists); ok {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create processor config: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, processorConfig)
}

// GetProcessorConfig retrieves a processor configuration by ID
// GET /api/v1/config/processors/:id
func (h *ConfigHandler) GetProcessorConfig(c *gin.Context) {
	id := c.Param("id")

	processorConfig, err := h.repository.GetProcessorConfig(c.Request.Context(), id)
	if err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get processor config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, processorConfig)
}

// ListProcessorConfigs lists all processor configurations
// GET /api/v1/config/processors
func (h *ConfigHandler) ListProcessorConfigs(c *gin.Context) {
	var enabled *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabledVal := enabledStr == "true"
		enabled = &enabledVal
	}

	processorConfigs, err := h.repository.ListProcessorConfigs(c.Request.Context(), enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list processor configs: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"processors": processorConfigs,
		"count":      len(processorConfigs),
	})
}

// UpdateProcessorConfig updates an existing processor configuration
// PUT /api/v1/config/processors/:id
func (h *ConfigHandler) UpdateProcessorConfig(c *gin.Context) {
	id := c.Param("id")

	var processorConfig interfaces.ProcessorConfig
	if err := c.ShouldBindJSON(&processorConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	processorConfig.ID = id

	if err := h.validator.ValidateProcessorConfig(&processorConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	if err := h.repository.UpdateProcessorConfig(c.Request.Context(), &processorConfig); err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update processor config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, processorConfig)
}

// DeleteProcessorConfig deletes a processor configuration
// DELETE /api/v1/config/processors/:id
func (h *ConfigHandler) DeleteProcessorConfig(c *gin.Context) {
	id := c.Param("id")

	if err := h.repository.DeleteProcessorConfig(c.Request.Context(), id); err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete processor config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "processor configuration deleted successfully"})
}

// ========== Template Handlers ==========

// CreateTemplate creates a new configuration template
// POST /api/v1/config/templates
func (h *ConfigHandler) CreateTemplate(c *gin.Context) {
	var template interfaces.ConfigTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if template.ID == "" {
		template.ID = "template-" + uuid.New().String()
	}

	if err := h.repository.CreateTemplate(c.Request.Context(), &template); err != nil {
		if _, ok := err.(interfaces.ErrConfigExists); ok {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create template: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// GetTemplate retrieves a configuration template by ID
// GET /api/v1/config/templates/:id
func (h *ConfigHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")

	template, err := h.repository.GetTemplate(c.Request.Context(), id)
	if err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get template: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// ListTemplates lists all configuration templates
// GET /api/v1/config/templates
func (h *ConfigHandler) ListTemplates(c *gin.Context) {
	configType := c.Query("type")

	templates, err := h.repository.ListTemplates(c.Request.Context(), configType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list templates: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"count":     len(templates),
	})
}

// UpdateTemplate updates an existing configuration template
// PUT /api/v1/config/templates/:id
func (h *ConfigHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")

	var template interfaces.ConfigTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	template.ID = id

	if err := h.repository.UpdateTemplate(c.Request.Context(), &template); err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update template: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// DeleteTemplate deletes a configuration template
// DELETE /api/v1/config/templates/:id
func (h *ConfigHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")

	if err := h.repository.DeleteTemplate(c.Request.Context(), id); err != nil {
		if _, ok := err.(interfaces.ErrConfigNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete template: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "template deleted successfully"})
}

// ========== History Handlers ==========

// GetConfigHistory retrieves the change history for a configuration
// GET /api/v1/config/history/:config_type/:config_id
func (h *ConfigHandler) GetConfigHistory(c *gin.Context) {
	configType := c.Param("config_type")
	configID := c.Param("config_id")

	history, err := h.repository.GetConfigHistory(c.Request.Context(), configID, configType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"count":   len(history),
	})
}

// ========== Import/Export Handlers ==========

// ExportConfigs exports all configurations of a specific type
// GET /api/v1/config/export/:config_type
func (h *ConfigHandler) ExportConfigs(c *gin.Context) {
	configType := c.Param("config_type")

	data, err := h.repository.ExportConfigs(c.Request.Context(), configType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export configs: " + err.Error()})
		return
	}

	// Set headers for file download
	c.Header("Content-Disposition", "attachment; filename="+configType+"_configs.json")
	c.Data(http.StatusOK, "application/json", data)
}

// ImportConfigs imports configurations from JSON
// POST /api/v1/config/import/:config_type
func (h *ConfigHandler) ImportConfigs(c *gin.Context) {
	configType := c.Param("config_type")

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body: " + err.Error()})
		return
	}

	if err := h.repository.ImportConfigs(c.Request.Context(), data, configType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import configs: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "configurations imported successfully"})
}

// ValidateConfig validates a configuration without saving it
// POST /api/v1/config/validate/:config_type
func (h *ConfigHandler) ValidateConfig(c *gin.Context) {
	configType := c.Param("config_type")

	var err error

	switch configType {
	case "rule":
		var ruleConfig interfaces.RuleConfig
		if err := c.ShouldBindJSON(&ruleConfig); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}
		err = h.validator.ValidateRuleConfig(&ruleConfig)

	case "analyzer":
		var analyzerConfig interfaces.AnalyzerConfig
		if err := c.ShouldBindJSON(&analyzerConfig); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}
		err = h.validator.ValidateAnalyzerConfig(&analyzerConfig)

	case "processor":
		var processorConfig interfaces.ProcessorConfig
		if err := c.ShouldBindJSON(&processorConfig); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}
		err = h.validator.ValidateProcessorConfig(&processorConfig)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown config type: " + configType})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "configuration is valid",
	})
}
