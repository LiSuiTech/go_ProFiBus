package interfaces

import (
	"context"
	"encoding/json"
	"time"
)

// AlgorithmConfig represents the base configuration interface for all algorithms
type AlgorithmConfig interface {
	GetID() string
	GetName() string
	GetType() string // "rule", "analyzer", "processor"
	GetVersion() string
	GetParameters() map[string]interface{}
	Validate() error
	ToJSON() ([]byte, error)
}

// RuleConfig represents the configuration for anomaly detection rules
type RuleConfig struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Type        string                 `json:"type" db:"type"` // "threshold", "statistical", "ml"
	Version     string                 `json:"version" db:"version"`
	Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	Priority    int                    `json:"priority" db:"priority"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy   string                 `json:"created_by" db:"created_by"`
	UpdatedBy   string                 `json:"updated_by" db:"updated_by"`
}

// GetID returns the rule ID
func (r *RuleConfig) GetID() string {
	return r.ID
}

// GetName returns the rule name
func (r *RuleConfig) GetName() string {
	return r.Name
}

// GetType returns the algorithm type
func (r *RuleConfig) GetType() string {
	return "rule"
}

// GetVersion returns the rule version
func (r *RuleConfig) GetVersion() string {
	return r.Version
}

// GetParameters returns the rule parameters
func (r *RuleConfig) GetParameters() map[string]interface{} {
	return r.Parameters
}

// Validate validates the rule configuration
func (r *RuleConfig) Validate() error {
	if r.ID == "" {
		return ErrInvalidConfig{Field: "id", Message: "rule ID is required"}
	}
	if r.Name == "" {
		return ErrInvalidConfig{Field: "name", Message: "rule name is required"}
	}
	if r.Type == "" {
		return ErrInvalidConfig{Field: "type", Message: "rule type is required"}
	}
	if r.Version == "" {
		return ErrInvalidConfig{Field: "version", Message: "version is required"}
	}
	if r.Parameters == nil || len(r.Parameters) == 0 {
		return ErrInvalidConfig{Field: "parameters", Message: "parameters are required"}
	}
	return nil
}

// ToJSON serializes the rule config to JSON
func (r *RuleConfig) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// AnalyzerConfig represents the configuration for anomaly analyzers
type AnalyzerConfig struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Type        string                 `json:"type" db:"type"` // "statistical", "ml", "ensemble"
	Version     string                 `json:"version" db:"version"`
	Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
	RuleIDs     []string               `json:"rule_ids" db:"rule_ids"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy   string                 `json:"created_by" db:"created_by"`
	UpdatedBy   string                 `json:"updated_by" db:"updated_by"`
}

// GetID returns the analyzer ID
func (a *AnalyzerConfig) GetID() string {
	return a.ID
}

// GetName returns the analyzer name
func (a *AnalyzerConfig) GetName() string {
	return a.Name
}

// GetType returns the algorithm type
func (a *AnalyzerConfig) GetType() string {
	return "analyzer"
}

// GetVersion returns the analyzer version
func (a *AnalyzerConfig) GetVersion() string {
	return a.Version
}

// GetParameters returns the analyzer parameters
func (a *AnalyzerConfig) GetParameters() map[string]interface{} {
	return a.Parameters
}

// Validate validates the analyzer configuration
func (a *AnalyzerConfig) Validate() error {
	if a.ID == "" {
		return ErrInvalidConfig{Field: "id", Message: "analyzer ID is required"}
	}
	if a.Name == "" {
		return ErrInvalidConfig{Field: "name", Message: "analyzer name is required"}
	}
	if a.Type == "" {
		return ErrInvalidConfig{Field: "type", Message: "analyzer type is required"}
	}
	if a.Version == "" {
		return ErrInvalidConfig{Field: "version", Message: "version is required"}
	}
	if a.Parameters == nil || len(a.Parameters) == 0 {
		return ErrInvalidConfig{Field: "parameters", Message: "parameters are required"}
	}
	return nil
}

// ToJSON serializes the analyzer config to JSON
func (a *AnalyzerConfig) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// ProcessorConfig represents the configuration for data processors
type ProcessorConfig struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Type        string                 `json:"type" db:"type"` // "filter", "transform", "aggregate"
	Version     string                 `json:"version" db:"version"`
	Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
	Order       int                    `json:"order" db:"order"` // Processing order in pipeline
	Enabled     bool                   `json:"enabled" db:"enabled"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy   string                 `json:"created_by" db:"created_by"`
	UpdatedBy   string                 `json:"updated_by" db:"updated_by"`
}

// GetID returns the processor ID
func (p *ProcessorConfig) GetID() string {
	return p.ID
}

// GetName returns the processor name
func (p *ProcessorConfig) GetName() string {
	return p.Name
}

// GetType returns the algorithm type
func (p *ProcessorConfig) GetType() string {
	return "processor"
}

// GetVersion returns the processor version
func (p *ProcessorConfig) GetVersion() string {
	return p.Version
}

// GetParameters returns the processor parameters
func (p *ProcessorConfig) GetParameters() map[string]interface{} {
	return p.Parameters
}

// Validate validates the processor configuration
func (p *ProcessorConfig) Validate() error {
	if p.ID == "" {
		return ErrInvalidConfig{Field: "id", Message: "processor ID is required"}
	}
	if p.Name == "" {
		return ErrInvalidConfig{Field: "name", Message: "processor name is required"}
	}
	if p.Type == "" {
		return ErrInvalidConfig{Field: "type", Message: "processor type is required"}
	}
	if p.Version == "" {
		return ErrInvalidConfig{Field: "version", Message: "version is required"}
	}
	if p.Parameters == nil || len(p.Parameters) == 0 {
		return ErrInvalidConfig{Field: "parameters", Message: "parameters are required"}
	}
	return nil
}

// ToJSON serializes the processor config to JSON
func (p *ProcessorConfig) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// ConfigTemplate represents a reusable configuration template
type ConfigTemplate struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Type        string                 `json:"type" db:"type"` // "rule", "analyzer", "processor"
	Category    string                 `json:"category" db:"category"`
	Schema      ConfigSchema           `json:"schema" db:"schema"`
	Defaults    map[string]interface{} `json:"defaults" db:"defaults"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// ConfigSchema defines the JSON schema for configuration parameters
type ConfigSchema struct {
	Type       string                       `json:"type"`
	Properties map[string]SchemaProperty    `json:"properties"`
	Required   []string                     `json:"required"`
	Examples   []map[string]interface{}     `json:"examples,omitempty"`
}

// SchemaProperty defines a single parameter schema
type SchemaProperty struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
}

// ConfigHistory tracks configuration changes
type ConfigHistory struct {
	ID         string                 `json:"id" db:"id"`
	ConfigID   string                 `json:"config_id" db:"config_id"`
	ConfigType string                 `json:"config_type" db:"config_type"` // "rule", "analyzer", "processor"
	Action     string                 `json:"action" db:"action"`           // "created", "updated", "deleted"
	ChangedBy  string                 `json:"changed_by" db:"changed_by"`
	ChangedAt  time.Time              `json:"changed_at" db:"changed_at"`
	OldValue   map[string]interface{} `json:"old_value" db:"old_value"`
	NewValue   map[string]interface{} `json:"new_value" db:"new_value"`
	Reason     string                 `json:"reason" db:"reason"`
}

// ConfigRepository defines the interface for configuration persistence
type ConfigRepository interface {
	// Rule configuration methods
	CreateRuleConfig(ctx context.Context, config *RuleConfig) error
	GetRuleConfig(ctx context.Context, id string) (*RuleConfig, error)
	ListRuleConfigs(ctx context.Context, enabled *bool, ruleType *string) ([]*RuleConfig, error)
	UpdateRuleConfig(ctx context.Context, config *RuleConfig) error
	DeleteRuleConfig(ctx context.Context, id string) error

	// Analyzer configuration methods
	CreateAnalyzerConfig(ctx context.Context, config *AnalyzerConfig) error
	GetAnalyzerConfig(ctx context.Context, id string) (*AnalyzerConfig, error)
	ListAnalyzerConfigs(ctx context.Context, enabled *bool) ([]*AnalyzerConfig, error)
	UpdateAnalyzerConfig(ctx context.Context, config *AnalyzerConfig) error
	DeleteAnalyzerConfig(ctx context.Context, id string) error

	// Processor configuration methods
	CreateProcessorConfig(ctx context.Context, config *ProcessorConfig) error
	GetProcessorConfig(ctx context.Context, id string) (*ProcessorConfig, error)
	ListProcessorConfigs(ctx context.Context, enabled *bool) ([]*ProcessorConfig, error)
	UpdateProcessorConfig(ctx context.Context, config *ProcessorConfig) error
	DeleteProcessorConfig(ctx context.Context, id string) error

	// Template methods
	CreateTemplate(ctx context.Context, template *ConfigTemplate) error
	GetTemplate(ctx context.Context, id string) (*ConfigTemplate, error)
	ListTemplates(ctx context.Context, configType string) ([]*ConfigTemplate, error)
	UpdateTemplate(ctx context.Context, template *ConfigTemplate) error
	DeleteTemplate(ctx context.Context, id string) error

	// History methods
	RecordHistory(ctx context.Context, history *ConfigHistory) error
	GetConfigHistory(ctx context.Context, configID string, configType string) ([]*ConfigHistory, error)

	// Import/Export methods
	ExportConfigs(ctx context.Context, configType string) ([]byte, error)
	ImportConfigs(ctx context.Context, data []byte, configType string) error
}

// ErrInvalidConfig represents a configuration validation error
type ErrInvalidConfig struct {
	Field   string
	Message string
}

func (e ErrInvalidConfig) Error() string {
	return "invalid configuration: " + e.Field + " - " + e.Message
}

// ErrConfigNotFound represents a configuration not found error
type ErrConfigNotFound struct {
	ID         string
	ConfigType string
}

func (e ErrConfigNotFound) Error() string {
	return e.ConfigType + " configuration not found: " + e.ID
}

// ErrConfigExists represents a duplicate configuration error
type ErrConfigExists struct {
	ID         string
	ConfigType string
}

func (e ErrConfigExists) Error() string {
	return e.ConfigType + " configuration already exists: " + e.ID
}
