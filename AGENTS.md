# AGENTS.md

## Cursor Cloud specific instructions

### 项目概述

go_ProFiBus 是一个基于 Go + Vue 3 的工业现场总线数据采集、处理和分析平台。详细架构和功能说明见根目录 `README.md`。

### 服务架构

| 服务 | 技术 | 端口 | 必需 |
|------|------|------|------|
| Go 后端 API | Go 1.24 + Gin | 8080 | 是 |
| Vue 3 前端 | Vite + Vue 3 + Element Plus | 3000 (dev) | 是 |
| PostgreSQL + TimescaleDB | timescale/timescaledb:latest-pg15 | 5432 | 是 |
| Redis | redis:7-alpine | 6379 | 是（docker-compose 要求） |
| Prometheus | prom/prometheus | 9090 | 否 |
| Grafana | grafana/grafana | 3000 | 否 |

### 启动服务

1. **启动 Docker 守护进程**（如未运行）：`sudo dockerd &>/tmp/dockerd.log &`

2. **启动数据库**（需要使用 TimescaleDB 镜像而非普通 PostgreSQL，因为迁移脚本依赖 TimescaleDB 扩展）：
   ```bash
   sudo docker run -d --name profibus-postgres \
     -e POSTGRES_DB=profibus -e POSTGRES_USER=profibus \
     -e POSTGRES_PASSWORD=profibus_secure_password \
     -p 5432:5432 \
     -v /workspace/migrations:/docker-entrypoint-initdb.d \
     timescale/timescaledb:latest-pg15
   ```

3. **启动 Redis**：
   ```bash
   sudo docker run -d --name profibus-redis \
     -p 6379:6379 redis:7-alpine \
     redis-server --requirepass redis_secure_password
   ```

4. **启动 Go 后端**：
   ```bash
   cd /workspace
   go build -o ./bin/profibus ./cmd/server
   DB_HOST=localhost DB_PORT=5432 DB_NAME=profibus DB_USER=profibus \
     DB_PASSWORD=profibus_secure_password API_PORT=8080 API_MODE=debug \
     API_CORS_ENABLED=true API_CORS_ORIGINS="*" \
     ./bin/profibus &>/tmp/profibus.log &
   ```

5. **启动前端开发服务器**：
   ```bash
   cd /workspace/web/dashboard && npx vite --host 0.0.0.0 &>/tmp/vite.log &
   ```

### 重要注意事项

- **迁移脚本中已修复的 PostgreSQL 语法问题**：`006_create_channels.sql` 中 `offset` 是 PostgreSQL 保留字需加引号；`012_create_universal_fusion.sql` 中行内 INDEX 语法不兼容 PostgreSQL；`017_create_rule_templates.sql` 中 JSONB 模板占位符需加引号。这些问题已在本分支修复。
- **默认管理员密码 hash 不匹配**：迁移中预置的 bcrypt hash 与密码 `admin123` 不匹配。首次启动后需手动更新：
  ```sql
  UPDATE users SET password_hash = '$2a$10$O0sUdUe.hTPGA7XXfrcqWO0P.20zRPZ6sd5h8r3Ob7Ype66ofUfp2' WHERE username = 'admin';
  ```
- **API 字段名大小写**：Go 后端返回的 JSON 字段名是大写开头（如 `ID`, `Name`, `Status`），前端某些组件期望小写（如 `id`, `name`, `status`）。设备列表页面的兼容性已在本分支修复。
- **Go vet / 测试**：`examples/` 和 `concurrent/` 目录存在预存的编译错误（引用了不存在的包）。核心包 lint 使用 `go vet ./cmd/... ./api/... ./internal/... ./storage/... ./logger/...`。
- **TypeScript 类型检查**：前端存在预存的 TS 错误（`import.meta.env` 类型、组件属性不匹配等），不影响开发运行。
- **前端 lint/type-check 命令**：`npm run type-check`（在 `web/dashboard/` 目录下）。
- **Go 测试**：`go test ./errors/... ./logger/...` 是有测试文件的核心包。
