# Phase 3 实现总结 - 算法配置系统与RBAC

本文档记录了 Phase 3 的完整实现,包括算法配置系统和RBAC(基于角色的访问控制)系统。

## 📋 实现概览

Phase 3 包含两个主要部分:
1. **算法配置系统** - 允许用户配置规则、分析器和处理器
2. **RBAC系统** - 提供完整的用户认证、授权和权限管理

## 🎯 核心功能

### 1. 算法配置系统

#### 1.1 配置接口 (`pkg/interfaces/config.go`)

定义了以下配置类型:
- `RuleConfig` - 异常检测规则配置
- `AnalyzerConfig` - 分析器配置
- `ProcessorConfig` - 数据处理器配置
- `ConfigTemplate` - 可重用的配置模板
- `ConfigHistory` - 配置变更历史

**关键特性**:
- 完整的参数验证
- 版本管理
- 优先级排序(规则)
- 执行顺序(处理器)
- 规则关联(分析器)

#### 1.2 配置验证器 (`internal/domain/config/validator.go`)

提供智能配置验证:
- JSON Schema 验证
- 类型检查(string, number, integer, boolean, object, array)
- 数值范围验证(minimum, maximum)
- 枚举值验证
- 正则表达式模式匹配
- 特定算法类型的业务规则验证

**示例验证规则**:
```go
// 阈值规则验证
- 必需参数: metric, threshold, operator
- 操作符限制: gt, lt, eq, gte, lte, ne

// 统计规则验证
- 必需参数: metric, zscore_threshold
- zscore_threshold > 0
- window_size >= 10

// 集成分析器验证
- 至少需要 2 个规则
- 策略必须是: majority_vote, unanimous, weighted_average
- weighted_average 策略需要 weights 参数
```

#### 1.3 配置存储库 (`internal/infrastructure/storage/config_repository.go`)

实现了完整的 CRUD 操作:
- 规则配置管理
- 分析器配置管理
- 处理器配置管理
- 模板管理
- 历史记录
- 导入/导出功能

**性能优化**:
- 使用 PostgreSQL 索引加速查询
- 批量操作支持
- 自动时间戳更新(触发器)

#### 1.4 配置 API (`api/handlers/config.go`)

提供 RESTful API:

**规则配置**:
- `POST /api/v1/config/rules` - 创建规则
- `GET /api/v1/config/rules` - 列出规则
- `GET /api/v1/config/rules/:id` - 获取规则
- `PUT /api/v1/config/rules/:id` - 更新规则
- `DELETE /api/v1/config/rules/:id` - 删除规则

**分析器配置**: 同上(路径为 `/config/analyzers`)
**处理器配置**: 同上(路径为 `/config/processors`)

**模板管理**:
- `POST /api/v1/config/templates` - 创建模板
- `GET /api/v1/config/templates` - 列出模板
- `GET /api/v1/config/templates/:id` - 获取模板
- `PUT /api/v1/config/templates/:id` - 更新模板
- `DELETE /api/v1/config/templates/:id` - 删除模板

**其他功能**:
- `GET /api/v1/config/history/:type/:id` - 获取配置历史
- `GET /api/v1/config/export/:type` - 导出配置(JSON)
- `POST /api/v1/config/import/:type` - 导入配置
- `POST /api/v1/config/validate/:type` - 验证配置(不保存)

### 2. RBAC系统

#### 2.1 RBAC接口 (`pkg/interfaces/rbac.go`)

定义了完整的RBAC实体:
- `User` - 系统用户
- `Role` - 角色(权限集合)
- `Permission` - 权限(resource:action格式)
- `Session` - 会话管理

**预定义角色**:
```go
const (
    RoleAdmin    = "admin"     // 完全访问权限
    RoleOperator = "operator"  // 管道操作权限
    RoleViewer   = "viewer"    // 只读权限
    RoleEditor   = "editor"    // 编辑配置权限
)
```

**权限格式**: `resource:action`
- 资源: rule, analyzer, processor, pipeline, user, role, config, metrics, trace
- 操作: create, read, update, delete, execute, export, import

#### 2.2 认证服务 (`internal/domain/auth/auth_service.go`)

提供完整的认证功能:
- **登录/登出**: JWT token 生成和验证
- **密码管理**: Bcrypt 哈希
- **会话管理**: Token 过期和刷新
- **安全性**:
  - 密码复杂度要求(最少8位)
  - 会话过期时间(默认24小时)
  - 失败登录保护

#### 2.3 授权服务 (`internal/domain/auth/auth_service.go`)

实现权限检查:
- `CheckPermission()` - 检查用户是否有特定权限
- `GetUserRoles()` - 获取用户所有角色
- `AssignRoleToUser()` - 分配角色给用户
- `RemoveRoleFromUser()` - 从用户移除角色

**权限聚合**: 用户的权限是所有角色权限的并集

#### 2.4 RBAC中间件 (`api/middleware/auth.go`)

提供多种认证/授权中间件:

```go
// 认证中间件 - 验证 JWT token
AuthMiddleware(authService)

// 权限中间件 - 需要特定权限
RequirePermission(authzService, resource, action)

// 角色中间件 - 需要特定角色
RequireRole(authzService, roleID)

// 可选认证 - 不强制要求认证
OptionalAuth(authService)

// 多权限中间件 - 满足任一权限即可
RequireAnyPermission(authzService, permissions)
```

**使用示例**:
```go
// 需要认证和 rule:read 权限
rules := v1.Group("/config/rules")
rules.Use(middleware.AuthMiddleware(authService))
rules.Use(middleware.RequirePermission(authzService, "rule", "read"))
{
    rules.GET("", configHandler.ListRuleConfigs)
}
```

#### 2.5 用户管理 API (`api/handlers/auth.go`)

**认证端点**:
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/logout` - 用户登出
- `POST /api/v1/auth/refresh` - 刷新token
- `GET /api/v1/auth/me` - 获取当前用户信息
- `POST /api/v1/auth/change-password` - 修改密码

**用户管理端点** (需要 admin 权限):
- `POST /api/v1/users` - 创建用户
- `GET /api/v1/users` - 列出用户
- `GET /api/v1/users/:id` - 获取用户
- `PUT /api/v1/users/:id` - 更新用户
- `DELETE /api/v1/users/:id` - 删除用户

**角色管理端点** (需要 admin 权限):
- `POST /api/v1/roles` - 创建角色
- `GET /api/v1/roles` - 列出角色
- `GET /api/v1/roles/:id` - 获取角色
- `PUT /api/v1/roles/:id` - 更新角色
- `DELETE /api/v1/roles/:id` - 删除角色

**用户-角色关联**:
- `POST /api/v1/users/:id/roles/:role_id` - 分配角色
- `DELETE /api/v1/users/:id/roles/:role_id` - 移除角色
- `GET /api/v1/users/:id/roles` - 获取用户角色

## 📊 数据库结构

### 配置表 (`migrations/003_add_config_tables.sql`)

**rule_configs** - 规则配置
```sql
- id (PK)
- name, description
- type (threshold/statistical/ml)
- version
- parameters (JSONB)
- enabled, priority
- created_at, updated_at
- created_by, updated_by
```

**analyzer_configs** - 分析器配置
```sql
- id (PK)
- name, description
- type (statistical/ml/ensemble)
- version
- parameters (JSONB)
- rule_ids (TEXT[])
- enabled
- created_at, updated_at
- created_by, updated_by
```

**processor_configs** - 处理器配置
```sql
- id (PK)
- name, description
- type (filter/transform/aggregate)
- version
- parameters (JSONB)
- "order" (INT) - 处理顺序
- enabled
- created_at, updated_at
- created_by, updated_by
```

**config_templates** - 配置模板
```sql
- id (PK)
- name, description
- type, category
- schema (JSONB) - JSON Schema
- defaults (JSONB)
- created_at, updated_at
```

**config_history** - 配置历史
```sql
- id (SERIAL PK)
- config_id, config_type
- action (created/updated/deleted)
- changed_by, changed_at
- old_value, new_value (JSONB)
- reason
```

**预定义模板**:
- `template-rule-threshold` - 阈值规则模板
- `template-rule-statistical` - 统计规则模板
- `template-analyzer-ensemble` - 集成分析器模板
- `template-processor-filter` - 过滤处理器模板
- `template-processor-transform` - 转换处理器模板

### RBAC表 (`migrations/004_add_rbac_tables.sql`)

**users** - 用户
```sql
- id (PK)
- username (UNIQUE)
- email (UNIQUE)
- password_hash
- full_name
- enabled
- created_at, updated_at
- last_login_at
```

**roles** - 角色
```sql
- id (PK)
- name (UNIQUE)
- description
- permissions (TEXT[]) - 权限数组
- created_at, updated_at
```

**user_roles** - 用户-角色关联表
```sql
- user_id, role_id (Composite PK)
- assigned_at
```

**sessions** - 会话
```sql
- id (PK)
- user_id (FK)
- token (UNIQUE)
- expires_at
- created_at
- ip_address, user_agent
```

**默认用户**:
- 用户名: `admin`
- 密码: `admin123`
- 角色: Admin (完全权限)

**预定义角色**:
1. **Admin** - 系统管理员
   - 所有资源的所有操作权限

2. **Operator** - 操作员
   - 创建/读取/更新规则、分析器、处理器
   - 读取/执行管道
   - 读取配置和指标

3. **Editor** - 编辑者
   - 创建/读取/更新规则、分析器、处理器
   - 只读管道
   - 读取/更新配置

4. **Viewer** - 查看者
   - 所有资源的只读权限

## 🔐 权限矩阵

| 角色 | 配置管理 | 管道操作 | 用户管理 | 导入/导出 |
|------|---------|---------|---------|----------|
| Admin | ✅ 全部 | ✅ 全部 | ✅ 全部 | ✅ 全部 |
| Operator | ✅ 创建/读/更新 | ✅ 读/执行 | ❌ | ❌ |
| Editor | ✅ 创建/读/更新 | 👁️ 只读 | ❌ | ❌ |
| Viewer | 👁️ 只读 | 👁️ 只读 | ❌ | ❌ |

## 🚀 使用示例

### 1. 登录认证

```bash
# 登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'

# 响应
{
  "token": "abc123...",
  "user": {
    "id": "user-admin",
    "username": "admin",
    "email": "admin@profibus.local",
    "full_name": "System Administrator",
    "role_ids": ["role-admin"],
    "enabled": true
  },
  "expires_at": "2024-01-21T10:00:00Z"
}
```

### 2. 创建规则配置

```bash
# 使用token访问受保护的端点
curl -X POST http://localhost:8080/api/v1/config/rules \
  -H "Authorization: Bearer abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Temperature Alert",
    "description": "Alert when temperature exceeds threshold",
    "type": "threshold",
    "version": "1.0.0",
    "parameters": {
      "metric": "temperature",
      "threshold": 75.0,
      "operator": "gt",
      "window_size": 60
    },
    "enabled": true,
    "priority": 1
  }'
```

### 3. 创建分析器配置

```bash
curl -X POST http://localhost:8080/api/v1/config/analyzers \
  -H "Authorization: Bearer abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ensemble Anomaly Detector",
    "description": "Combines multiple rules",
    "type": "ensemble",
    "version": "1.0.0",
    "rule_ids": ["rule-1", "rule-2", "rule-3"],
    "parameters": {
      "strategy": "majority_vote",
      "threshold": 0.6
    },
    "enabled": true
  }'
```

### 4. 导出/导入配置

```bash
# 导出规则配置
curl -X GET http://localhost:8080/api/v1/config/export/rule \
  -H "Authorization: Bearer abc123..." \
  -o rules_backup.json

# 导入规则配置
curl -X POST http://localhost:8080/api/v1/config/import/rule \
  -H "Authorization: Bearer abc123..." \
  -H "Content-Type: application/json" \
  -d @rules_backup.json
```

### 5. 用户管理

```bash
# 创建新用户
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com",
    "password": "SecurePass123",
    "full_name": "John Doe",
    "role_ids": ["role-operator"]
  }'

# 分配角色给用户
curl -X POST http://localhost:8080/api/v1/users/user-123/roles/role-editor \
  -H "Authorization: Bearer abc123..."
```

## 📝 配置模板示例

### 阈值规则模板

```json
{
  "id": "template-rule-threshold",
  "name": "Threshold Rule Template",
  "type": "rule",
  "schema": {
    "type": "object",
    "properties": {
      "metric": {
        "type": "string",
        "description": "Metric field to monitor"
      },
      "threshold": {
        "type": "number",
        "description": "Threshold value"
      },
      "operator": {
        "type": "string",
        "enum": ["gt", "lt", "eq", "gte", "lte"],
        "description": "Comparison operator"
      },
      "window_size": {
        "type": "integer",
        "description": "Time window size in seconds",
        "minimum": 1
      }
    },
    "required": ["metric", "threshold", "operator"]
  },
  "defaults": {
    "operator": "gt",
    "window_size": 60
  }
}
```

## 🔧 API服务器集成

在 `api/server.go` 中完成了以下集成:

1. **添加依赖注入**:
```go
func NewServer(
    config *ServerConfig,
    store *storageOld.PostgresStore,
    tracer interfaces.Tracer,
    traceRepository interfaces.TraceRepository,
    configRepository interfaces.ConfigRepository,
    userRepository interfaces.UserRepository,
    authService interfaces.AuthService,
    authzService interfaces.AuthorizationService,
    orch *orchestrator.Orchestrator,
) (*Server, error)
```

2. **注册认证路由**:
   - 公开端点: `/api/v1/auth/login`
   - 受保护端点: `/api/v1/auth/*` (需要认证)
   - 用户管理: `/api/v1/users` (需要 user:* 权限)
   - 角色管理: `/api/v1/roles` (需要 admin 角色)

3. **保护配置路由**:
   - 所有配置端点都需要认证
   - 读取操作需要 `resource:read` 权限
   - 创建操作需要 `resource:create` 权限
   - 更新操作需要 `resource:update` 权限
   - 删除操作需要 `resource:delete` 权限
   - 导入/导出需要 `config:import` / `config:export` 权限

## 🎨 前端集成准备

### TypeScript类型定义

已在 `web/dashboard/src/types/index.ts` 中添加:
- 配置类型 (RuleConfig, AnalyzerConfig, ProcessorConfig)
- RBAC类型 (User, Role, LoginRequest, LoginResponse)
- API响应类型

### 待实现的前端组件

建议的组件结构:

```
src/
├── views/
│   ├── Login.vue                  # 登录页面
│   ├── config/
│   │   ├── RuleList.vue          # 规则列表
│   │   ├── RuleEdit.vue          # 规则编辑
│   │   ├── AnalyzerList.vue      # 分析器列表
│   │   ├── AnalyzerEdit.vue      # 分析器编辑
│   │   ├── ProcessorList.vue     # 处理器列表
│   │   └── ProcessorEdit.vue     # 处理器编辑
│   └── admin/
│       ├── UserList.vue          # 用户列表
│       ├── UserEdit.vue          # 用户编辑
│       ├── RoleList.vue          # 角色列表
│       └── RoleEdit.vue          # 角色编辑
├── stores/
│   ├── auth.ts                    # 认证状态管理
│   └── config.ts                  # 配置状态管理
└── services/
    ├── auth.ts                    # 认证API服务
    └── config.ts                  # 配置API服务
```

## ✅ 实现清单

### 后端实现

- ✅ 配置接口定义
- ✅ 配置验证器
- ✅ 配置存储库 (CRUD)
- ✅ 配置API处理器
- ✅ 配置数据库迁移
- ✅ RBAC接口定义
- ✅ 用户存储库
- ✅ 认证服务 (登录/登出/token管理)
- ✅ 授权服务 (权限检查)
- ✅ RBAC中间件
- ✅ 用户管理API
- ✅ 角色管理API
- ✅ RBAC数据库迁移
- ✅ API服务器集成

### 前端实现

- ✅ TypeScript类型定义
- ⏳ 登录页面
- ⏳ 配置管理页面
- ⏳ 用户管理页面
- ⏳ 角色管理页面
- ⏳ 权限守卫 (路由)
- ⏳ API服务封装

## 🚦 下一步工作

1. **运行数据库迁移**:
```bash
psql -U postgres -d profibus -f migrations/003_add_config_tables.sql
psql -U postgres -d profibus -f migrations/004_add_rbac_tables.sql
```

2. **更新main.go**:
   - 初始化 ConfigRepository
   - 初始化 UserRepository
   - 初始化 AuthService 和 AuthorizationService
   - 传递给 NewServer()

3. **实现前端组件**:
   - 创建登录页面
   - 创建配置管理页面
   - 创建用户/角色管理页面
   - 实现路由守卫

4. **测试**:
   - 单元测试
   - 集成测试
   - API端点测试
   - 权限测试

## 📖 参考文档

- [API_EXAMPLES.md](./API_EXAMPLES.md) - API使用示例
- [PHASE2_OPTIMIZATIONS.md](./PHASE2_OPTIMIZATIONS.md) - Phase 2优化总结
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 项目架构文档

## 🔒 安全建议

1. **密码策略**:
   - 最少8位
   - 包含大小写字母、数字、特殊字符
   - 定期更换

2. **Token管理**:
   - 使用HTTPS传输
   - 设置合理的过期时间(默认24小时)
   - 实现token刷新机制
   - 登出时清除token

3. **权限最小化**:
   - 只分配必要的权限
   - 定期审查用户权限
   - 使用预定义角色

4. **审计日志**:
   - 配置变更历史已记录
   - 建议记录所有敏感操作
   - 定期审查日志

## 💡 最佳实践

1. **配置管理**:
   - 使用模板快速创建配置
   - 启用配置历史追踪
   - 导出配置做备份
   - 在测试环境验证后再部署到生产环境

2. **用户管理**:
   - 使用强密码
   - 禁用不活跃的用户账户
   - 使用角色而不是直接分配权限
   - 定期审查用户访问权限

3. **部署**:
   - 在生产环境修改默认admin密码
   - 启用HTTPS
   - 配置防火墙规则
   - 定期更新依赖包

## 📧 支持

如有问题或建议,请通过以下方式联系:
- GitHub Issues: https://github.com/yourusername/go_profibus/issues
- 文档: 查看 `docs/` 目录下的其他文档

---

**Phase 3 完成时间**: 2026-01-20
**主要贡献者**: Claude Code Agent
