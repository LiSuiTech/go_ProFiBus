# Phase 3: 算法配置系统 - 实施计划

## 目标
实现一个灵活的算法配置系统，让用户能够通过 Web 界面动态配置规则引擎、分析器和处理器，无需修改代码即可调整算法参数。

## 核心功能

### 1. 配置管理
- 规则配置（Rule Configuration）
- 分析器配置（Analyzer Configuration）
- 处理器配置（Processor Configuration）
- 配置版本控制
- 配置导入/导出

### 2. 规则类型
- 阈值规则（Threshold Rule）
- 范围规则（Range Rule）
- 变化率规则（Rate of Change Rule）
- 统计规则（Statistical Rule）
- 自定义规则（Custom Rule）

### 3. 配置验证
- 参数类型验证
- 参数范围验证
- 依赖关系验证
- 配置完整性检查

---

## 实施步骤

### Step 1: 设计配置数据模型

#### 1.1 通用配置接口

**文件**: `pkg/interfaces/config.go`

```go
package interfaces

import "time"

// AlgorithmConfig 算法配置接口
type AlgorithmConfig interface {
    GetID() string
    GetName() string
    GetType() string
    Validate() error
    ToJSON() ([]byte, error)
}

// RuleConfig 规则配置
type RuleConfig struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Type        string                 `json:"type"` // threshold, range, rate, statistical
    Enabled     bool                   `json:"enabled"`
    Priority    int                    `json:"priority"`
    Parameters  map[string]interface{} `json:"parameters"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// AnalyzerConfig 分析器配置
type AnalyzerConfig struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Type        string                 `json:"type"` // anomaly, pattern, ml
    Enabled     bool                   `json:"enabled"`
    Parameters  map[string]interface{} `json:"parameters"`
    Rules       []string               `json:"rules"` // Rule IDs
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Type        string                 `json:"type"` // filter, transform, aggregate
    Enabled     bool                   `json:"enabled"`
    Order       int                    `json:"order"`
    Parameters  map[string]interface{} `json:"parameters"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}
```

#### 1.2 配置模板

**文件**: `pkg/interfaces/config_template.go`

```go
// ConfigTemplate 配置模板
type ConfigTemplate struct {
    ID          string              `json:"id"`
    Name        string              `json:"name"`
    Description string              `json:"description"`
    Category    string              `json:"category"` // rule, analyzer, processor
    Type        string              `json:"type"`
    Schema      ConfigSchema        `json:"schema"`
    Example     map[string]interface{} `json:"example"`
}

// ConfigSchema 配置 Schema
type ConfigSchema struct {
    Parameters []ParameterSchema `json:"parameters"`
}

// ParameterSchema 参数 Schema
type ParameterSchema struct {
    Name        string      `json:"name"`
    Type        string      `json:"type"` // string, number, boolean, array, object
    Required    bool        `json:"required"`
    Default     interface{} `json:"default"`
    Min         *float64    `json:"min,omitempty"`
    Max         *float64    `json:"max,omitempty"`
    Options     []string    `json:"options,omitempty"`
    Description string      `json:"description"`
}
```

---

### Step 2: 实现配置存储层

#### 2.1 数据库 Schema

**文件**: `migrations/003_add_config_tables.sql`

```sql
-- 规则配置表
CREATE TABLE IF NOT EXISTS rule_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    parameters JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(name)
);

-- 索引
CREATE INDEX idx_rule_configs_type ON rule_configs(type);
CREATE INDEX idx_rule_configs_enabled ON rule_configs(enabled);

-- 分析器配置表
CREATE TABLE IF NOT EXISTS analyzer_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    parameters JSONB NOT NULL,
    rules TEXT[], -- Rule IDs
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(name)
);

-- 索引
CREATE INDEX idx_analyzer_configs_type ON analyzer_configs(type);
CREATE INDEX idx_analyzer_configs_enabled ON analyzer_configs(enabled);

-- 处理器配置表
CREATE TABLE IF NOT EXISTS processor_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    order_index INTEGER DEFAULT 0,
    parameters JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(name)
);

-- 索引
CREATE INDEX idx_processor_configs_type ON processor_configs(type);
CREATE INDEX idx_processor_configs_order ON processor_configs(order_index);

-- 配置模板表
CREATE TABLE IF NOT EXISTS config_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL, -- rule, analyzer, processor
    type VARCHAR(50) NOT NULL,
    schema JSONB NOT NULL,
    example JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(name)
);

-- 索引
CREATE INDEX idx_config_templates_category ON config_templates(category);
CREATE INDEX idx_config_templates_type ON config_templates(type);

-- 配置历史表（用于版本控制）
CREATE TABLE IF NOT EXISTS config_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL,
    config_type VARCHAR(50) NOT NULL, -- rule, analyzer, processor
    action VARCHAR(20) NOT NULL, -- create, update, delete
    config_data JSONB NOT NULL,
    user_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_config_history_config ON config_history(config_id, config_type);
CREATE INDEX idx_config_history_created ON config_history(created_at DESC);
```

#### 2.2 配置仓储实现

**文件**: `internal/infrastructure/storage/config_repository.go`

```go
package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "go_ProFiBus/pkg/interfaces"
    "go_ProFiBus/storage"
)

type ConfigRepository struct {
    store *storage.PostgresStore
}

func NewConfigRepository(store *storage.PostgresStore) *ConfigRepository {
    return &ConfigRepository{store: store}
}

// Rule Config Methods

func (r *ConfigRepository) CreateRuleConfig(ctx context.Context, config *interfaces.RuleConfig) error {
    sql := `
        INSERT INTO rule_configs
        (id, name, description, type, enabled, priority, parameters, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `

    params, _ := json.Marshal(config.Parameters)

    _, err := r.store.Exec(ctx, sql,
        config.ID,
        config.Name,
        config.Description,
        config.Type,
        config.Enabled,
        config.Priority,
        params,
        config.CreatedAt,
        config.UpdatedAt,
    )

    return err
}

func (r *ConfigRepository) GetRuleConfig(ctx context.Context, id string) (*interfaces.RuleConfig, error) {
    sql := `
        SELECT id, name, description, type, enabled, priority, parameters, created_at, updated_at
        FROM rule_configs
        WHERE id = $1
    `

    var config interfaces.RuleConfig
    var paramsJSON []byte

    err := r.store.QueryRow(ctx, sql, id).Scan(
        &config.ID,
        &config.Name,
        &config.Description,
        &config.Type,
        &config.Enabled,
        &config.Priority,
        &paramsJSON,
        &config.CreatedAt,
        &config.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    if err := json.Unmarshal(paramsJSON, &config.Parameters); err != nil {
        return nil, err
    }

    return &config, nil
}

func (r *ConfigRepository) ListRuleConfigs(ctx context.Context, filter map[string]interface{}) ([]*interfaces.RuleConfig, error) {
    sql := `
        SELECT id, name, description, type, enabled, priority, parameters, created_at, updated_at
        FROM rule_configs
        WHERE 1=1
    `

    args := []interface{}{}
    argIdx := 1

    if typeFilter, ok := filter["type"].(string); ok && typeFilter != "" {
        sql += fmt.Sprintf(" AND type = $%d", argIdx)
        args = append(args, typeFilter)
        argIdx++
    }

    if enabledFilter, ok := filter["enabled"].(bool); ok {
        sql += fmt.Sprintf(" AND enabled = $%d", argIdx)
        args = append(args, enabledFilter)
        argIdx++
    }

    sql += " ORDER BY priority DESC, created_at DESC"

    rows, err := r.store.Query(ctx, sql, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    configs := make([]*interfaces.RuleConfig, 0)

    for rows.Next() {
        var config interfaces.RuleConfig
        var paramsJSON []byte

        err := rows.Scan(
            &config.ID,
            &config.Name,
            &config.Description,
            &config.Type,
            &config.Enabled,
            &config.Priority,
            &paramsJSON,
            &config.CreatedAt,
            &config.UpdatedAt,
        )

        if err != nil {
            return nil, err
        }

        if err := json.Unmarshal(paramsJSON, &config.Parameters); err != nil {
            return nil, err
        }

        configs = append(configs, &config)
    }

    return configs, rows.Err()
}

func (r *ConfigRepository) UpdateRuleConfig(ctx context.Context, config *interfaces.RuleConfig) error {
    sql := `
        UPDATE rule_configs
        SET name = $2, description = $3, type = $4, enabled = $5,
            priority = $6, parameters = $7, updated_at = $8
        WHERE id = $1
    `

    params, _ := json.Marshal(config.Parameters)
    config.UpdatedAt = time.Now()

    result, err := r.store.Exec(ctx, sql,
        config.ID,
        config.Name,
        config.Description,
        config.Type,
        config.Enabled,
        config.Priority,
        params,
        config.UpdatedAt,
    )

    if err != nil {
        return err
    }

    if result.RowsAffected() == 0 {
        return fmt.Errorf("rule config not found: %s", config.ID)
    }

    return nil
}

func (r *ConfigRepository) DeleteRuleConfig(ctx context.Context, id string) error {
    sql := "DELETE FROM rule_configs WHERE id = $1"

    result, err := r.store.Exec(ctx, sql, id)
    if err != nil {
        return err
    }

    if result.RowsAffected() == 0 {
        return fmt.Errorf("rule config not found: %s", id)
    }

    return nil
}

// Similar methods for AnalyzerConfig and ProcessorConfig...
```

---

### Step 3: 实现配置验证器

**文件**: `internal/domain/config/validator.go`

```go
package config

import (
    "fmt"
    "go_ProFiBus/pkg/interfaces"
)

type ConfigValidator struct {
    templates map[string]*interfaces.ConfigTemplate
}

func NewConfigValidator() *ConfigValidator {
    return &ConfigValidator{
        templates: make(map[string]*interfaces.ConfigTemplate),
    }
}

func (v *ConfigValidator) RegisterTemplate(template *interfaces.ConfigTemplate) {
    v.templates[template.Type] = template
}

func (v *ConfigValidator) ValidateRuleConfig(config *interfaces.RuleConfig) error {
    template, ok := v.templates[config.Type]
    if !ok {
        return fmt.Errorf("unknown rule type: %s", config.Type)
    }

    return v.validateParameters(config.Parameters, template.Schema)
}

func (v *ConfigValidator) validateParameters(params map[string]interface{}, schema interfaces.ConfigSchema) error {
    for _, paramSchema := range schema.Parameters {
        value, exists := params[paramSchema.Name]

        // Check required
        if paramSchema.Required && !exists {
            return fmt.Errorf("required parameter missing: %s", paramSchema.Name)
        }

        if !exists {
            continue
        }

        // Check type
        if err := v.validateType(value, paramSchema.Type); err != nil {
            return fmt.Errorf("parameter %s: %w", paramSchema.Name, err)
        }

        // Check range for numbers
        if paramSchema.Type == "number" {
            if num, ok := value.(float64); ok {
                if paramSchema.Min != nil && num < *paramSchema.Min {
                    return fmt.Errorf("parameter %s: value %f below minimum %f",
                        paramSchema.Name, num, *paramSchema.Min)
                }
                if paramSchema.Max != nil && num > *paramSchema.Max {
                    return fmt.Errorf("parameter %s: value %f above maximum %f",
                        paramSchema.Name, num, *paramSchema.Max)
                }
            }
        }

        // Check options
        if len(paramSchema.Options) > 0 {
            if str, ok := value.(string); ok {
                valid := false
                for _, opt := range paramSchema.Options {
                    if str == opt {
                        valid = true
                        break
                    }
                }
                if !valid {
                    return fmt.Errorf("parameter %s: invalid option %s", paramSchema.Name, str)
                }
            }
        }
    }

    return nil
}

func (v *ConfigValidator) validateType(value interface{}, expectedType string) error {
    switch expectedType {
    case "string":
        if _, ok := value.(string); !ok {
            return fmt.Errorf("expected string, got %T", value)
        }
    case "number":
        if _, ok := value.(float64); !ok {
            return fmt.Errorf("expected number, got %T", value)
        }
    case "boolean":
        if _, ok := value.(bool); !ok {
            return fmt.Errorf("expected boolean, got %T", value)
        }
    case "array":
        if _, ok := value.([]interface{}); !ok {
            return fmt.Errorf("expected array, got %T", value)
        }
    case "object":
        if _, ok := value.(map[string]interface{}); !ok {
            return fmt.Errorf("expected object, got %T", value)
        }
    }

    return nil
}
```

---

### Step 4: 实现配置 API

**文件**: `api/handlers/config.go`

```go
package handlers

import (
    "net/http"
    "time"

    "go_ProFiBus/internal/domain/config"
    "go_ProFiBus/internal/infrastructure/storage"
    "go_ProFiBus/pkg/interfaces"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type ConfigHandler struct {
    repository *storage.ConfigRepository
    validator  *config.ConfigValidator
}

func NewConfigHandler(repo *storage.ConfigRepository, validator *config.ConfigValidator) *ConfigHandler {
    return &ConfigHandler{
        repository: repo,
        validator:  validator,
    }
}

// Rule Config Endpoints

func (h *ConfigHandler) CreateRuleConfig(c *gin.Context) {
    var config interfaces.RuleConfig
    if err := c.ShouldBindJSON(&config); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate
    if err := h.validator.ValidateRuleConfig(&config); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Set metadata
    config.ID = uuid.New().String()
    config.CreatedAt = time.Now()
    config.UpdatedAt = time.Now()

    // Save
    if err := h.repository.CreateRuleConfig(c.Request.Context(), &config); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, config)
}

func (h *ConfigHandler) GetRuleConfig(c *gin.Context) {
    id := c.Param("id")

    config, err := h.repository.GetRuleConfig(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Rule config not found"})
        return
    }

    c.JSON(http.StatusOK, config)
}

func (h *ConfigHandler) ListRuleConfigs(c *gin.Context) {
    filter := map[string]interface{}{}

    if typeFilter := c.Query("type"); typeFilter != "" {
        filter["type"] = typeFilter
    }

    if enabledFilter := c.Query("enabled"); enabledFilter != "" {
        filter["enabled"] = enabledFilter == "true"
    }

    configs, err := h.repository.ListRuleConfigs(c.Request.Context(), filter)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "rules": configs,
        "total": len(configs),
    })
}

func (h *ConfigHandler) UpdateRuleConfig(c *gin.Context) {
    id := c.Param("id")

    var config interfaces.RuleConfig
    if err := c.ShouldBindJSON(&config); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    config.ID = id

    // Validate
    if err := h.validator.ValidateRuleConfig(&config); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Update
    if err := h.repository.UpdateRuleConfig(c.Request.Context(), &config); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, config)
}

func (h *ConfigHandler) DeleteRuleConfig(c *gin.Context) {
    id := c.Param("id")

    if err := h.repository.DeleteRuleConfig(c.Request.Context(), id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Rule config deleted"})
}

// Similar endpoints for AnalyzerConfig and ProcessorConfig...
```

---

### Step 5: 前端配置界面

#### 5.1 配置管理页面

**文件**: `web/dashboard/src/views/ConfigManager.vue`

**功能**:
- 配置列表（表格展示）
- 创建/编辑配置（表单）
- 删除配置
- 启用/禁用配置
- 配置预览
- 导入/导出配置

#### 5.2 规则配置组件

**文件**: `web/dashboard/src/components/RuleConfigForm.vue`

**功能**:
- 动态表单生成（基于模板）
- 参数验证
- 实时预览
- 帮助文档

---

## 配置模板示例

### 阈值规则模板

```json
{
  "id": "threshold-rule-template",
  "name": "Threshold Rule",
  "description": "Triggers when a value exceeds a threshold",
  "category": "rule",
  "type": "threshold",
  "schema": {
    "parameters": [
      {
        "name": "field",
        "type": "string",
        "required": true,
        "description": "The field name to monitor"
      },
      {
        "name": "operator",
        "type": "string",
        "required": true,
        "options": [">", ">=", "<", "<=", "==", "!="],
        "default": ">",
        "description": "Comparison operator"
      },
      {
        "name": "threshold",
        "type": "number",
        "required": true,
        "description": "Threshold value"
      },
      {
        "name": "severity",
        "type": "string",
        "required": true,
        "options": ["info", "warning", "error", "critical"],
        "default": "warning",
        "description": "Alert severity level"
      }
    ]
  },
  "example": {
    "field": "temperature",
    "operator": ">",
    "threshold": 80.0,
    "severity": "warning"
  }
}
```

---

## API 端点设计

### Rule Config APIs

```
POST   /api/v1/configs/rules           - Create rule config
GET    /api/v1/configs/rules           - List rule configs
GET    /api/v1/configs/rules/:id       - Get rule config
PUT    /api/v1/configs/rules/:id       - Update rule config
DELETE /api/v1/configs/rules/:id       - Delete rule config
POST   /api/v1/configs/rules/:id/test  - Test rule config
```

### Analyzer Config APIs

```
POST   /api/v1/configs/analyzers        - Create analyzer config
GET    /api/v1/configs/analyzers        - List analyzer configs
GET    /api/v1/configs/analyzers/:id    - Get analyzer config
PUT    /api/v1/configs/analyzers/:id    - Update analyzer config
DELETE /api/v1/configs/analyzers/:id    - Delete analyzer config
```

### Template APIs

```
GET    /api/v1/configs/templates         - List all templates
GET    /api/v1/configs/templates/:type   - Get template by type
```

### Import/Export APIs

```
POST   /api/v1/configs/import            - Import configs (JSON)
GET    /api/v1/configs/export            - Export all configs
GET    /api/v1/configs/export/:type      - Export specific type
```

---

## 时间估算

- Step 1: 设计数据模型 - 2小时
- Step 2: 实现存储层 - 3小时
- Step 3: 实现验证器 - 2小时
- Step 4: 实现 API - 3小时
- Step 5: 前端界面 - 6小时
- 测试和优化 - 2小时

**总计**: ~18小时

---

## 成功标准

- ✅ 支持至少5种规则类型配置
- ✅ 配置验证准确率 100%
- ✅ 配置热更新（无需重启）
- ✅ 配置导入/导出功能
- ✅ 友好的 Web 配置界面
- ✅ 配置版本历史记录

---

## 下一步

完成 Phase 3 后，系统将具备完整的配置管理能力，用户可以通过界面动态调整算法参数，大大提升系统的灵活性和可用性。
