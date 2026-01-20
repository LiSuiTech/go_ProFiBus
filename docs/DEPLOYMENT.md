# go_ProFiBus Deployment Guide

本文档提供 go_ProFiBus 的完整部署指南,包括 Docker Compose、Kubernetes 和 CI/CD 配置。

## 📋 目录

- [前提条件](#前提条件)
- [Docker Compose 部署](#docker-compose-部署)
- [Kubernetes 部署](#kubernetes-部署)
- [CI/CD 配置](#cicd-配置)
- [监控和日志](#监控和日志)
- [故障排除](#故障排除)

## 🔧 前提条件

### 通用要求
- **Go**: 1.22+ (开发环境)
- **PostgreSQL**: 15+ with TimescaleDB extension
- **Redis**: 7+

### Docker Compose 部署
- **Docker**: 24.0+
- **Docker Compose**: 2.20+
- **最小系统要求**:
  - CPU: 2 cores
  - RAM: 4GB
  - 磁盘: 20GB

### Kubernetes 部署
- **Kubernetes**: 1.27+
- **kubectl**: 匹配集群版本
- **Helm**: 3.12+ (可选)
- **最小集群要求**:
  - 节点: 3+ (生产环境)
  - CPU: 4+ cores 总计
  - RAM: 8GB+ 总计
  - 存储: 支持 PersistentVolume

## 🐳 Docker Compose 部署

### 快速开始

1. **克隆仓库**
```bash
git clone https://github.com/yourusername/go_profibus.git
cd go_profibus
```

2. **配置环境变量**
```bash
cp .env.example .env
# 编辑 .env 文件,修改密码和敏感信息
vim .env
```

3. **构建并启动服务**
```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f profibus

# 检查服务状态
docker-compose ps
```

4. **初始化数据库**
```bash
# 运行数据库迁移
docker-compose exec profibus /app/profibus migrate up

# 或者使用 psql 手动运行
docker-compose exec postgres psql -U profibus -d profibus -f /docker-entrypoint-initdb.d/001_init_schema.sql
```

5. **验证部署**
```bash
# 健康检查
curl http://localhost:8080/health

# 测试 API
curl http://localhost:8080/api/v1/pipelines

# 使用健康检查脚本
./scripts/healthcheck.sh localhost 8080
```

### 服务说明

| 服务 | 端口 | 说明 |
|------|------|------|
| profibus | 8080 | 主应用 REST API |
| profibus | 8081 | Prometheus metrics |
| postgres | 5432 | PostgreSQL 数据库 |
| redis | 6379 | Redis 缓存 |
| prometheus | 9090 | Prometheus 监控 |
| grafana | 3000 | Grafana 可视化 |
| nginx | 80/443 | 反向代理 (可选) |

### 常用命令

```bash
# 停止所有服务
docker-compose down

# 停止并删除卷
docker-compose down -v

# 重启服务
docker-compose restart profibus

# 查看资源使用
docker stats

# 进入容器
docker-compose exec profibus sh

# 备份数据库
docker-compose exec postgres pg_dump -U profibus profibus > backup.sql

# 恢复数据库
docker-compose exec -T postgres psql -U profibus profibus < backup.sql
```

### 生产环境配置

1. **启用 HTTPS (Nginx)**
```bash
# 生成 SSL 证书
mkdir -p deployments/nginx/ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout deployments/nginx/ssl/nginx.key \
  -out deployments/nginx/ssl/nginx.crt

# 启动 nginx
docker-compose --profile production up -d nginx
```

2. **配置持久化存储**
```yaml
# docker-compose.yml
volumes:
  postgres_data:
    driver: local
    driver_opts:
      type: none
      device: /data/postgres
      o: bind
```

3. **配置备份**
```bash
# 创建备份脚本
cat > scripts/backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
docker-compose exec -T postgres pg_dump -U profibus profibus | gzip > "$BACKUP_DIR/profibus_$DATE.sql.gz"
find "$BACKUP_DIR" -name "profibus_*.sql.gz" -mtime +7 -delete
EOF

chmod +x scripts/backup.sh

# 添加到 crontab
crontab -e
# 每天凌晨 2 点备份
0 2 * * * /path/to/scripts/backup.sh
```

## ☸️ Kubernetes 部署

### 准备工作

1. **配置 kubectl**
```bash
# 验证集群连接
kubectl cluster-info
kubectl get nodes
```

2. **创建 namespace**
```bash
kubectl apply -f deployments/kubernetes/namespace.yaml
```

3. **配置 Secrets**
```bash
# 生成新的 secrets
kubectl create secret generic profibus-secrets \
  --from-literal=DB_USER=profibus \
  --from-literal=DB_PASSWORD=$(openssl rand -base64 32) \
  --from-literal=REDIS_PASSWORD=$(openssl rand -base64 32) \
  --from-literal=JWT_SECRET=$(openssl rand -base64 48) \
  -n profibus

# 或使用配置文件(记得修改默认密码!)
kubectl apply -f deployments/kubernetes/secrets.yaml
```

4. **配置 ConfigMap**
```bash
kubectl apply -f deployments/kubernetes/configmap.yaml
```

### 部署步骤

1. **部署数据库层**
```bash
# PostgreSQL
kubectl apply -f deployments/kubernetes/postgres.yaml

# 等待 PostgreSQL 就绪
kubectl wait --for=condition=ready pod -l app=postgres -n profibus --timeout=300s

# Redis
kubectl apply -f deployments/kubernetes/redis.yaml
```

2. **运行数据库迁移**
```bash
# 创建迁移 Job
kubectl create job --from=cronjob/profibus-migration migration-$(date +%s) -n profibus

# 或手动运行
kubectl run -it --rm migrate --image=your-registry/profibus:latest \
  --restart=Never -n profibus -- /app/profibus migrate up
```

3. **部署应用**
```bash
kubectl apply -f deployments/kubernetes/deployment.yaml
```

4. **配置自动扩缩容**
```bash
kubectl apply -f deployments/kubernetes/hpa.yaml
```

5. **配置 Ingress**
```bash
# 安装 Ingress Controller (如果还没有)
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.5/deploy/static/provider/cloud/deploy.yaml

# 部署 Ingress
kubectl apply -f deployments/kubernetes/ingress.yaml
```

6. **验证部署**
```bash
# 检查所有资源
kubectl get all -n profibus

# 检查 Pod 状态
kubectl get pods -n profibus -w

# 查看日志
kubectl logs -f deployment/profibus -n profibus

# 测试服务
kubectl port-forward svc/profibus-service 8080:80 -n profibus
curl http://localhost:8080/health
```

### 常用 Kubernetes 命令

```bash
# 查看 Pod 详情
kubectl describe pod <pod-name> -n profibus

# 进入 Pod
kubectl exec -it <pod-name> -n profibus -- sh

# 查看资源使用
kubectl top pods -n profibus
kubectl top nodes

# 扩缩容
kubectl scale deployment profibus --replicas=5 -n profibus

# 滚动更新
kubectl set image deployment/profibus profibus=your-registry/profibus:v1.1.0 -n profibus
kubectl rollout status deployment/profibus -n profibus

# 回滚
kubectl rollout undo deployment/profibus -n profibus
kubectl rollout history deployment/profibus -n profibus

# 查看事件
kubectl get events -n profibus --sort-by='.lastTimestamp'

# 删除所有资源
kubectl delete namespace profibus
```

### 生产环境最佳实践

1. **资源限制**
```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "1Gi"
    cpu: "500m"
```

2. **Pod 反亲和性**
```yaml
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchExpressions:
        - key: app
          operator: In
          values:
          - profibus
      topologyKey: kubernetes.io/hostname
```

3. **存储类配置**
```yaml
# 使用 SSD 存储类
storageClassName: fast-ssd
```

4. **配置 PodDisruptionBudget**
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: profibus-pdb
  namespace: profibus
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: profibus
```

5. **网络策略**
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: profibus-network-policy
  namespace: profibus
spec:
  podSelector:
    matchLabels:
      app: profibus
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: profibus
    ports:
    - protocol: TCP
      port: 8080
```

## 🔄 CI/CD 配置

### GitHub Actions

已配置的工作流 (`.github/workflows/ci-cd.yml`):

1. **代码质量检查**: golangci-lint, 格式化检查
2. **测试**: 单元测试、集成测试、覆盖率报告
3. **安全扫描**: Trivy、gosec
4. **构建镜像**: 多平台构建 (amd64, arm64)
5. **部署**:
   - Staging: `develop` 分支自动部署
   - Production: `v*` 标签触发部署

### 配置 GitHub Secrets

在 GitHub 仓库设置中添加以下 secrets:

```bash
# Kubernetes配置
KUBE_CONFIG_STAGING       # Staging 集群 kubeconfig (base64编码)
KUBE_CONFIG_PRODUCTION    # Production 集群 kubeconfig (base64编码)

# 容器镜像仓库
REGISTRY_USERNAME         # 镜像仓库用户名
REGISTRY_PASSWORD         # 镜像仓库密码

# 数据库
DB_PASSWORD               # 生产数据库密码

# 其他
CODECOV_TOKEN            # Codecov token
```

### 手动触发部署

```bash
# 创建发布标签
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# 这将触发生产环境部署
```

## 📊 监控和日志

### Prometheus

访问 Prometheus: `http://localhost:9090` (Docker Compose) 或通过 Ingress

**关键指标**:
- `http_requests_total`: API 请求总数
- `http_request_duration_seconds`: 请求延迟
- `go_goroutines`: Goroutine 数量
- `process_cpu_seconds_total`: CPU 使用
- `process_resident_memory_bytes`: 内存使用

### Grafana

访问 Grafana: `http://localhost:3000` (默认 admin/admin)

**预配置面板**:
- 应用性能概览
- API 请求统计
- 数据库性能
- 系统资源使用

### 日志

**Docker Compose**:
```bash
# 查看实时日志
docker-compose logs -f profibus

# 查看最近 100 行
docker-compose logs --tail=100 profibus

# 按时间过滤
docker-compose logs --since 1h profibus
```

**Kubernetes**:
```bash
# 查看 Pod 日志
kubectl logs -f deployment/profibus -n profibus

# 查看多个 Pod 的日志
kubectl logs -f -l app=profibus -n profibus --max-log-requests=10

# 查看前一个容器的日志(崩溃后)
kubectl logs <pod-name> -n profibus --previous
```

**集中式日志方案** (推荐):
- ELK Stack (Elasticsearch, Logstash, Kibana)
- Loki + Grafana
- Datadog
- CloudWatch (AWS)

## 🔧 故障排除

### 常见问题

#### 1. 数据库连接失败

```bash
# 检查数据库状态
docker-compose ps postgres
kubectl get pods -l app=postgres -n profibus

# 检查连接
docker-compose exec profibus nc -zv postgres 5432
kubectl exec -it <pod-name> -n profibus -- nc -zv postgres-service 5432

# 查看数据库日志
docker-compose logs postgres
kubectl logs -l app=postgres -n profibus
```

#### 2. Pod 一直在 CrashLoopBackOff

```bash
# 查看 Pod 事件
kubectl describe pod <pod-name> -n profibus

# 查看日志
kubectl logs <pod-name> -n profibus --previous

# 检查配置
kubectl get configmap profibus-config -n profibus -o yaml
kubectl get secret profibus-secrets -n profibus -o jsonpath='{.data}'
```

#### 3. Ingress 无法访问

```bash
# 检查 Ingress 状态
kubectl get ingress -n profibus
kubectl describe ingress profibus-ingress -n profibus

# 检查 Ingress Controller
kubectl get pods -n ingress-nginx
kubectl logs -l app.kubernetes.io/name=ingress-nginx -n ingress-nginx

# 测试内部服务
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -n profibus -- \
  curl http://profibus-service/health
```

#### 4. 性能问题

```bash
# 检查资源使用
kubectl top pods -n profibus
kubectl top nodes

# 查看 HPA 状态
kubectl get hpa -n profibus
kubectl describe hpa profibus-hpa -n profibus

# 分析慢查询
docker-compose exec postgres psql -U profibus -d profibus \
  -c "SELECT query, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
```

### 调试工具

```bash
# 创建调试 Pod
kubectl run -it --rm debug --image=nicolaka/netshoot --restart=Never -n profibus -- /bin/bash

# 在调试 Pod 中测试
nslookup postgres-service
curl http://profibus-service/health
psql -h postgres-service -U profibus -d profibus
```

## 📚 更多资源

- [Docker 官方文档](https://docs.docker.com/)
- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)

## 🆘 获取帮助

如遇问题,请:
1. 查看本文档的故障排除部分
2. 检查项目 Issues: https://github.com/yourusername/go_profibus/issues
3. 提交新的 Issue 并提供详细信息

---

**最后更新**: 2026-01-20
**作者**: go_ProFiBus Team
