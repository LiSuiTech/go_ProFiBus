package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yourusername/go_profibus/pkg/interfaces"
)

// Validator provides configuration validation functionality
type Validator struct {
	templates map[string]*interfaces.ConfigTemplate
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{
		templates: make(map[string]*interfaces.ConfigTemplate),
	}
}

// RegisterTemplate registers a configuration template for validation
func (v *Validator) RegisterTemplate(template *interfaces.ConfigTemplate) {
	v.templates[template.ID] = template
}

// ValidateWithTemplate validates a configuration against a template
func (v *Validator) ValidateWithTemplate(config interfaces.AlgorithmConfig, templateID string) error {
	template, exists := v.templates[templateID]
	if !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	// Check if config type matches template type
	if config.GetType() != template.Type {
		return interfaces.ErrInvalidConfig{
			Field:   "type",
			Message: fmt.Sprintf("config type %s does not match template type %s", config.GetType(), template.Type),
		}
	}

	// Validate against schema
	return v.ValidateAgainstSchema(config.GetParameters(), template.Schema)
}

// ValidateAgainstSchema validates parameters against a JSON schema
func (v *Validator) ValidateAgainstSchema(params map[string]interface{}, schema interfaces.ConfigSchema) error {
	// Check required fields
	for _, required := range schema.Required {
		if _, exists := params[required]; !exists {
			return interfaces.ErrInvalidConfig{
				Field:   required,
				Message: fmt.Sprintf("required field %s is missing", required),
			}
		}
	}

	// Validate each property
	for key, value := range params {
		property, exists := schema.Properties[key]
		if !exists {
			// Unknown property - could be warning or error depending on strictness
			continue
		}

		if err := v.validateProperty(key, value, property); err != nil {
			return err
		}
	}

	return nil
}

// validateProperty validates a single property against its schema
func (v *Validator) validateProperty(key string, value interface{}, property interfaces.SchemaProperty) error {
	// Type validation
	if err := v.validateType(key, value, property.Type); err != nil {
		return err
	}

	// Numeric range validation
	if property.Minimum != nil || property.Maximum != nil {
		if err := v.validateNumericRange(key, value, property.Minimum, property.Maximum); err != nil {
			return err
		}
	}

	// Enum validation
	if len(property.Enum) > 0 {
		if err := v.validateEnum(key, value, property.Enum); err != nil {
			return err
		}
	}

	// Pattern validation for strings
	if property.Pattern != "" {
		if err := v.validatePattern(key, value, property.Pattern); err != nil {
			return err
		}
	}

	return nil
}

// validateType validates the type of a value
func (v *Validator) validateType(key string, value interface{}, expectedType string) error {
	var actualType string

	switch value.(type) {
	case string:
		actualType = "string"
	case float64, int, int32, int64, float32:
		actualType = "number"
	case bool:
		actualType = "boolean"
	case map[string]interface{}:
		actualType = "object"
	case []interface{}:
		actualType = "array"
	case nil:
		actualType = "null"
	default:
		actualType = "unknown"
	}

	// Handle integer as a subtype of number
	if expectedType == "integer" {
		switch v := value.(type) {
		case int, int32, int64:
			return nil
		case float64:
			if v == float64(int64(v)) {
				return nil
			}
		}
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: fmt.Sprintf("expected integer, got %s", actualType),
		}
	}

	if actualType != expectedType {
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: fmt.Sprintf("expected type %s, got %s", expectedType, actualType),
		}
	}

	return nil
}

// validateNumericRange validates numeric values are within specified range
func (v *Validator) validateNumericRange(key string, value interface{}, min, max *float64) error {
	var numValue float64

	switch v := value.(type) {
	case float64:
		numValue = v
	case int:
		numValue = float64(v)
	case int32:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	case float32:
		numValue = float64(v)
	default:
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: "value is not numeric",
		}
	}

	if min != nil && numValue < *min {
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: fmt.Sprintf("value %v is less than minimum %v", numValue, *min),
		}
	}

	if max != nil && numValue > *max {
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: fmt.Sprintf("value %v is greater than maximum %v", numValue, *max),
		}
	}

	return nil
}

// validateEnum validates value is one of the allowed enum values
func (v *Validator) validateEnum(key string, value interface{}, allowedValues []string) error {
	strValue, ok := value.(string)
	if !ok {
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: "enum value must be a string",
		}
	}

	for _, allowed := range allowedValues {
		if strValue == allowed {
			return nil
		}
	}

	return interfaces.ErrInvalidConfig{
		Field:   key,
		Message: fmt.Sprintf("value %s is not in allowed values: %s", strValue, strings.Join(allowedValues, ", ")),
	}
}

// validatePattern validates string matches a regex pattern
func (v *Validator) validatePattern(key string, value interface{}, pattern string) error {
	strValue, ok := value.(string)
	if !ok {
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: "pattern validation requires string value",
		}
	}

	matched, err := regexp.MatchString(pattern, strValue)
	if err != nil {
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: fmt.Sprintf("invalid regex pattern: %s", err),
		}
	}

	if !matched {
		return interfaces.ErrInvalidConfig{
			Field:   key,
			Message: fmt.Sprintf("value %s does not match pattern %s", strValue, pattern),
		}
	}

	return nil
}

// ValidateRuleConfig validates a rule configuration
func (v *Validator) ValidateRuleConfig(config *interfaces.RuleConfig) error {
	// Basic validation
	if err := config.Validate(); err != nil {
		return err
	}

	// Type-specific validation
	switch config.Type {
	case "threshold":
		return v.validateThresholdRule(config)
	case "statistical":
		return v.validateStatisticalRule(config)
	case "ml":
		return v.validateMLRule(config)
	default:
		return interfaces.ErrInvalidConfig{
			Field:   "type",
			Message: fmt.Sprintf("unknown rule type: %s", config.Type),
		}
	}
}

// validateThresholdRule validates threshold rule specific parameters
func (v *Validator) validateThresholdRule(config *interfaces.RuleConfig) error {
	params := config.Parameters

	// Check required parameters
	requiredParams := []string{"metric", "threshold", "operator"}
	for _, param := range requiredParams {
		if _, exists := params[param]; !exists {
			return interfaces.ErrInvalidConfig{
				Field:   param,
				Message: fmt.Sprintf("threshold rule requires parameter: %s", param),
			}
		}
	}

	// Validate operator
	operator, ok := params["operator"].(string)
	if !ok {
		return interfaces.ErrInvalidConfig{Field: "operator", Message: "operator must be a string"}
	}

	validOperators := []string{"gt", "lt", "eq", "gte", "lte", "ne"}
	found := false
	for _, valid := range validOperators {
		if operator == valid {
			found = true
			break
		}
	}
	if !found {
		return interfaces.ErrInvalidConfig{
			Field:   "operator",
			Message: fmt.Sprintf("invalid operator: %s (must be one of: %s)", operator, strings.Join(validOperators, ", ")),
		}
	}

	// Validate threshold is numeric
	if _, ok := params["threshold"].(float64); !ok {
		if _, ok := params["threshold"].(int); !ok {
			return interfaces.ErrInvalidConfig{Field: "threshold", Message: "threshold must be numeric"}
		}
	}

	return nil
}

// validateStatisticalRule validates statistical rule specific parameters
func (v *Validator) validateStatisticalRule(config *interfaces.RuleConfig) error {
	params := config.Parameters

	// Check required parameters
	if _, exists := params["metric"]; !exists {
		return interfaces.ErrInvalidConfig{Field: "metric", Message: "statistical rule requires metric parameter"}
	}

	// Validate zscore_threshold if present
	if zscore, exists := params["zscore_threshold"]; exists {
		if zscoreFloat, ok := zscore.(float64); ok {
			if zscoreFloat <= 0 {
				return interfaces.ErrInvalidConfig{Field: "zscore_threshold", Message: "zscore_threshold must be positive"}
			}
		} else {
			return interfaces.ErrInvalidConfig{Field: "zscore_threshold", Message: "zscore_threshold must be numeric"}
		}
	}

	// Validate window_size if present
	if windowSize, exists := params["window_size"]; exists {
		switch v := windowSize.(type) {
		case float64:
			if v < 10 {
				return interfaces.ErrInvalidConfig{Field: "window_size", Message: "window_size must be at least 10"}
			}
		case int:
			if v < 10 {
				return interfaces.ErrInvalidConfig{Field: "window_size", Message: "window_size must be at least 10"}
			}
		default:
			return interfaces.ErrInvalidConfig{Field: "window_size", Message: "window_size must be numeric"}
		}
	}

	return nil
}

// validateMLRule validates machine learning rule specific parameters
func (v *Validator) validateMLRule(config *interfaces.RuleConfig) error {
	params := config.Parameters

	// Check required parameters
	if _, exists := params["model_type"]; !exists {
		return interfaces.ErrInvalidConfig{Field: "model_type", Message: "ml rule requires model_type parameter"}
	}

	// Validate model_type
	modelType, ok := params["model_type"].(string)
	if !ok {
		return interfaces.ErrInvalidConfig{Field: "model_type", Message: "model_type must be a string"}
	}

	validModelTypes := []string{"isolation_forest", "autoencoder", "lstm", "svm"}
	found := false
	for _, valid := range validModelTypes {
		if modelType == valid {
			found = true
			break
		}
	}
	if !found {
		return interfaces.ErrInvalidConfig{
			Field:   "model_type",
			Message: fmt.Sprintf("invalid model_type: %s (must be one of: %s)", modelType, strings.Join(validModelTypes, ", ")),
		}
	}

	return nil
}

// ValidateAnalyzerConfig validates an analyzer configuration
func (v *Validator) ValidateAnalyzerConfig(config *interfaces.AnalyzerConfig) error {
	// Basic validation
	if err := config.Validate(); err != nil {
		return err
	}

	// Type-specific validation
	switch config.Type {
	case "statistical":
		return v.validateStatisticalAnalyzer(config)
	case "ml":
		return v.validateMLAnalyzer(config)
	case "ensemble":
		return v.validateEnsembleAnalyzer(config)
	default:
		return interfaces.ErrInvalidConfig{
			Field:   "type",
			Message: fmt.Sprintf("unknown analyzer type: %s", config.Type),
		}
	}
}

// validateStatisticalAnalyzer validates statistical analyzer parameters
func (v *Validator) validateStatisticalAnalyzer(config *interfaces.AnalyzerConfig) error {
	// Statistical analyzers should have at least one rule
	if len(config.RuleIDs) == 0 {
		return interfaces.ErrInvalidConfig{Field: "rule_ids", Message: "statistical analyzer requires at least one rule"}
	}
	return nil
}

// validateMLAnalyzer validates ML analyzer parameters
func (v *Validator) validateMLAnalyzer(config *interfaces.AnalyzerConfig) error {
	params := config.Parameters

	// Check for model configuration
	if _, exists := params["model_config"]; !exists {
		return interfaces.ErrInvalidConfig{Field: "model_config", Message: "ml analyzer requires model_config parameter"}
	}

	return nil
}

// validateEnsembleAnalyzer validates ensemble analyzer parameters
func (v *Validator) validateEnsembleAnalyzer(config *interfaces.AnalyzerConfig) error {
	params := config.Parameters

	// Ensemble analyzers must have multiple rules
	if len(config.RuleIDs) < 2 {
		return interfaces.ErrInvalidConfig{Field: "rule_ids", Message: "ensemble analyzer requires at least 2 rules"}
	}

	// Check for strategy
	if _, exists := params["strategy"]; !exists {
		return interfaces.ErrInvalidConfig{Field: "strategy", Message: "ensemble analyzer requires strategy parameter"}
	}

	strategy, ok := params["strategy"].(string)
	if !ok {
		return interfaces.ErrInvalidConfig{Field: "strategy", Message: "strategy must be a string"}
	}

	validStrategies := []string{"majority_vote", "unanimous", "weighted_average"}
	found := false
	for _, valid := range validStrategies {
		if strategy == valid {
			found = true
			break
		}
	}
	if !found {
		return interfaces.ErrInvalidConfig{
			Field:   "strategy",
			Message: fmt.Sprintf("invalid strategy: %s (must be one of: %s)", strategy, strings.Join(validStrategies, ", ")),
		}
	}

	// If weighted_average, check for weights
	if strategy == "weighted_average" {
		if _, exists := params["weights"]; !exists {
			return interfaces.ErrInvalidConfig{Field: "weights", Message: "weighted_average strategy requires weights parameter"}
		}
	}

	return nil
}

// ValidateProcessorConfig validates a processor configuration
func (v *Validator) ValidateProcessorConfig(config *interfaces.ProcessorConfig) error {
	// Basic validation
	if err := config.Validate(); err != nil {
		return err
	}

	// Type-specific validation
	switch config.Type {
	case "filter":
		return v.validateFilterProcessor(config)
	case "transform":
		return v.validateTransformProcessor(config)
	case "aggregate":
		return v.validateAggregateProcessor(config)
	default:
		return interfaces.ErrInvalidConfig{
			Field:   "type",
			Message: fmt.Sprintf("unknown processor type: %s", config.Type),
		}
	}
}

// validateFilterProcessor validates filter processor parameters
func (v *Validator) validateFilterProcessor(config *interfaces.ProcessorConfig) error {
	params := config.Parameters

	requiredParams := []string{"field", "operator", "value"}
	for _, param := range requiredParams {
		if _, exists := params[param]; !exists {
			return interfaces.ErrInvalidConfig{
				Field:   param,
				Message: fmt.Sprintf("filter processor requires parameter: %s", param),
			}
		}
	}

	// Validate operator
	operator, ok := params["operator"].(string)
	if !ok {
		return interfaces.ErrInvalidConfig{Field: "operator", Message: "operator must be a string"}
	}

	validOperators := []string{"eq", "ne", "gt", "lt", "gte", "lte", "contains", "regex"}
	found := false
	for _, valid := range validOperators {
		if operator == valid {
			found = true
			break
		}
	}
	if !found {
		return interfaces.ErrInvalidConfig{
			Field:   "operator",
			Message: fmt.Sprintf("invalid operator: %s (must be one of: %s)", operator, strings.Join(validOperators, ", ")),
		}
	}

	return nil
}

// validateTransformProcessor validates transform processor parameters
func (v *Validator) validateTransformProcessor(config *interfaces.ProcessorConfig) error {
	params := config.Parameters

	if _, exists := params["transformations"]; !exists {
		return interfaces.ErrInvalidConfig{Field: "transformations", Message: "transform processor requires transformations parameter"}
	}

	// Validate transformations array
	transformations, ok := params["transformations"].([]interface{})
	if !ok {
		return interfaces.ErrInvalidConfig{Field: "transformations", Message: "transformations must be an array"}
	}

	if len(transformations) == 0 {
		return interfaces.ErrInvalidConfig{Field: "transformations", Message: "transformations array cannot be empty"}
	}

	// Validate each transformation
	for i, t := range transformations {
		transform, ok := t.(map[string]interface{})
		if !ok {
			return interfaces.ErrInvalidConfig{
				Field:   fmt.Sprintf("transformations[%d]", i),
				Message: "transformation must be an object",
			}
		}

		// Check required fields
		for _, field := range []string{"source_field", "target_field", "operation"} {
			if _, exists := transform[field]; !exists {
				return interfaces.ErrInvalidConfig{
					Field:   fmt.Sprintf("transformations[%d].%s", i, field),
					Message: "required field missing",
				}
			}
		}
	}

	return nil
}

// validateAggregateProcessor validates aggregate processor parameters
func (v *Validator) validateAggregateProcessor(config *interfaces.ProcessorConfig) error {
	params := config.Parameters

	requiredParams := []string{"field", "aggregation"}
	for _, param := range requiredParams {
		if _, exists := params[param]; !exists {
			return interfaces.ErrInvalidConfig{
				Field:   param,
				Message: fmt.Sprintf("aggregate processor requires parameter: %s", param),
			}
		}
	}

	// Validate aggregation type
	aggregation, ok := params["aggregation"].(string)
	if !ok {
		return interfaces.ErrInvalidConfig{Field: "aggregation", Message: "aggregation must be a string"}
	}

	validAggregations := []string{"sum", "avg", "min", "max", "count", "stddev"}
	found := false
	for _, valid := range validAggregations {
		if aggregation == valid {
			found = true
			break
		}
	}
	if !found {
		return interfaces.ErrInvalidConfig{
			Field:   "aggregation",
			Message: fmt.Sprintf("invalid aggregation: %s (must be one of: %s)", aggregation, strings.Join(validAggregations, ", ")),
		}
	}

	return nil
}
