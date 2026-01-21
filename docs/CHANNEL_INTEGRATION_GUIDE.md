# 采集通道管理功能集成指南

本文档说明如何将采集通道管理功能集成到 go_ProFiBus 项目中。

## 已完成的文件

### 后端文件

1. **Domain Model**
   - `internal/domain/channel/channel.go` - 通道、点位、协议定义

2. **Repository Interface**
   - `pkg/interfaces/channel_repository.go` - 仓储接口定义

3. **Repository Implementation**
   - `internal/infrastructure/storage/channel_repository.go` - PostgreSQL 实现

4. **API Handler**
   - `api/handlers/channel.go` - HTTP 处理器

5. **Database Migration**
   - `migrations/006_create_channels.sql` - 数据库表创建

### 前端文件

1. **Types**
   - `web/dashboard/src/types/index.ts` - TypeScript 类型定义（已更新）

2. **API Service**
   - `web/dashboard/src/services/channelApi.ts` - API 调用服务

3. **Views**
   - `web/dashboard/src/views/Channels.vue` - 采集通道管理页面

4. **Router**
   - `web/dashboard/src/router/index.ts` - 路由配置（已更新）

5. **App Component**
   - `web/dashboard/src/App.vue` - 导航菜单（已更新）

## 后端集成步骤

### 步骤 1: 修改 api/server.go

#### 1.1 在 Server 结构体中添加字段

在 `api/server.go` 的 `Server` 结构体中添加：

```go
type Server struct {
    // ... 现有字段 ...
    channelRepository interfaces.ChannelRepository  // 添加这行
}
```

#### 1.2 修改 NewServer 函数签名

在 `NewServer` 函数参数列表中添加：

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
    channelRepository interfaces.ChannelRepository,  // 添加这行
    orch *orchestrator.Orchestrator,
) (*Server, error) {
```

#### 1.3 在 NewServer 函数中初始化字段

在 `NewServer` 函数的 `server` 结构体初始化部分添加：

```go
server := &Server{
    // ... 现有字段 ...
    channelRepository: channelRepository,  // 添加这行
}
```

#### 1.4 在 registerRoutes 函数中注册路由

在 `registerRoutes` 函数的 `v1` 路由组中添加：

```go
// 采集通道管理路由 (Phase 4)
if s.channelRepository != nil {
    channelHandler := handlers.NewChannelHandler(s.channelRepository)

    // 协议列表（公开）
    v1.GET("/protocols", channelHandler.GetProtocols)

    channels := v1.Group("/channels")
    // 应用认证中间件（如果RBAC启用）
    if s.authService != nil {
        channels.Use(middleware.AuthMiddleware(s.authService))
    }
    {
        channels.GET("", channelHandler.GetChannels)
        channels.GET("/:id", channelHandler.GetChannel)
        channels.POST("", channelHandler.CreateChannel)
        channels.PUT("/:id", channelHandler.UpdateChannel)
        channels.DELETE("/:id", channelHandler.DeleteChannel)
        channels.POST("/:id/start", channelHandler.StartChannel)
        channels.POST("/:id/stop", channelHandler.StopChannel)

        // 点位管理
        channels.GET("/:id/points", channelHandler.GetChannelPoints)
        channels.POST("/:id/points", channelHandler.CreatePoint)
        channels.PUT("/:id/points/:point_id", channelHandler.UpdatePoint)
        channels.DELETE("/:id/points/:point_id", channelHandler.DeletePoint)
    }
}
```

### 步骤 2: 修改主程序入口

在您的主程序文件（通常是 `cmd/server/main.go` 或类似文件）中：

#### 2.1 导入必要的包

```go
import (
    // ... 现有导入 ...
    "go_ProFiBus/internal/infrastructure/storage"
)
```

#### 2.2 创建 Channel Repository 实例

```go
// 创建 Channel Repository
channelRepository := storage.NewChannelRepository(postgresStore)
```

#### 2.3 传递给 NewServer

```go
server, err := api.NewServer(
    serverConfig,
    postgresStore,
    tracer,
    traceRepository,
    configRepository,
    userRepository,
    authService,
    authzService,
    channelRepository,  // 添加这个参数
    orch,
)
```

### 步骤 3: 运行数据库迁移

确保运行数据库迁移脚本来创建必要的表：

```bash
psql -U profibus -d profibus -f migrations/006_create_channels.sql
```

或者如果您使用自动迁移工具，确保它能够检测到新的迁移文件。

## API 端点说明

### 协议管理

- `GET /api/v1/protocols` - 获取支持的协议列表

### 通道管理

- `GET /api/v1/channels` - 获取所有通道
- `GET /api/v1/channels/:id` - 获取单个通道详情
- `POST /api/v1/channels` - 创建通道
- `PUT /api/v1/channels/:id` - 更新通道
- `DELETE /api/v1/channels/:id` - 删除通道
- `POST /api/v1/channels/:id/start` - 启动通道
- `POST /api/v1/channels/:id/stop` - 停止通道

### 点位管理

- `GET /api/v1/channels/:id/points` - 获取通道的所有点位
- `POST /api/v1/channels/:id/points` - 创建点位
- `PUT /api/v1/channels/:id/points/:point_id` - 更新点位
- `DELETE /api/v1/channels/:id/points/:point_id` - 删除点位

## 前端访问

启动前端开发服务器后，访问：

```
http://localhost:8888/channels
```

或者如果使用 Docker Compose，访问：

```
http://localhost:8888/channels
```

## 支持的协议

当前支持以下协议：

1. **UART** - 通用异步收发器
2. **CAN** - 控制器局域网络
3. **USB** - 通用串行总线
4. **1-Wire** - 单线通信协议
5. **Modbus** - 工业通信协议
6. **RS-232** - 串行通信标准
7. **RS-485** - 差分串行接口
8. **I2C** - 互连集成电路
9. **SPI** - 串行外设接口

每个协议都有其特定的配置字段，前端会根据选择的协议动态显示相应的配置表单。

## 数据模型

### Channel（通道）

```go
type Channel struct {
    ID          string        // 通道ID
    Name        string        // 通道名称
    Description string        // 描述
    Protocol    ProtocolType  // 协议类型
    DeviceName  string        // 设备名称
    DevicePort  string        // 设备端口
    Status      ChannelStatus // 状态（running/stopped/error）
    Config      ProtocolConfig // 协议配置
    Enabled     bool          // 是否启用
    CreatedAt   time.Time     // 创建时间
    UpdatedAt   time.Time     // 更新时间
    Points      []Point       // 点位列表
}
```

### Point（点位）

```go
type Point struct {
    ID          string    // 点位ID
    ChannelID   string    // 所属通道ID
    Name        string    // 点位名称
    Description string    // 描述
    Address     string    // 点位地址
    DataType    DataType  // 数据类型（int/float/bool/string/bytes）
    Unit        string    // 单位
    Scale       float64   // 缩放系数
    Offset      float64   // 偏移量
    Enabled     bool      // 是否启用
    CreatedAt   time.Time // 创建时间
    UpdatedAt   time.Time // 更新时间
}
```

## 未来扩展

### 与采集器集成

当前的 `StartChannel` 和 `StopChannel` 方法仅更新状态，需要与实际的采集器集成：

```go
// TODO: 在 api/handlers/channel.go 中实现
func (h *ChannelHandler) StartChannel(c *gin.Context) {
    // 1. 根据通道配置创建采集器实例
    // 2. 启动采集器
    // 3. 更新状态为 running
    // 4. 返回成功响应
}
```

### 实时数据监控

可以扩展以支持实时数据监控：

1. 在通道启动时，将采集到的数据通过 WebSocket 推送到前端
2. 在前端添加实时数据显示组件
3. 支持数据图表和趋势分析

### 点位数据存储

可以将采集到的点位数据存储到 TimescaleDB：

1. 定义数据表结构
2. 在采集器中添加数据存储逻辑
3. 提供历史数据查询 API

## 测试建议

1. **单元测试**：为 Repository 和 Handler 编写单元测试
2. **集成测试**：测试完整的 API 调用流程
3. **E2E 测试**：测试前端页面的完整交互流程

## 常见问题

### Q: 如何添加新的协议？

A: 在 `internal/domain/channel/channel.go` 的 `GetSupportedProtocols()` 函数中添加新的协议定义。

### Q: 如何自定义点位的数据处理？

A: 使用 `Scale` 和 `Offset` 字段：`processed_value = raw_value * scale + offset`

### Q: 如何处理协议特定的配置？

A: 使用 `ProtocolConfig` 结构体的灵活字段，前端会根据 `ProtocolInfo.config_fields` 动态生成表单。

## 完成标志

✅ 后端 Domain Model 创建完成
✅ 后端 Repository 实现完成
✅ 后端 Handler 实现完成
✅ 数据库迁移脚本创建完成
✅ 前端类型定义完成
✅ 前端 API 服务完成
✅ 前端页面组件完成
✅ 路由和导航完成
⏳ 后端路由集成待完成（需手动集成）
⏳ 数据库迁移执行待完成
⏳ 实际采集器集成待完成

## 参考文档

- [ARCHITECTURE.md](../ARCHITECTURE.md) - 项目架构说明
- [DATABASE.md](./DATABASE.md) - 数据库设计
- [DEPLOYMENT.md](./DEPLOYMENT.md) - 部署指南
