# go_ProFiBus 部署说明

本文档说明如何在不同环境中部署 go_ProFiBus。

## 方式一：Docker Compose（推荐用于开发与小规模部署）

```bash
git clone https://github.com/YouEvanLi/go_ProFiBus.git
cd go_ProFiBus
cp .env.example .env   # 配置环境变量
docker-compose up -d
```

- 首次运行前请确保本机已安装 Docker 与 Docker Compose。
- 可选：在项目根目录创建 `configs` 目录并放入 `config.yaml`，compose 会将其挂载到容器内 `/app/configs`。

## 方式二：Kubernetes（推荐用于生产）

```bash
# 部署到 Kubernetes 集群
kubectl apply -f deployments/kubernetes/

# 查看部署状态
kubectl get pods -n profibus
```

部署前请根据实际环境修改：

- `deployments/kubernetes/secrets.yaml`：数据库、Redis、JWT 等敏感信息。
- `deployments/kubernetes/deployment.yaml`：镜像地址 `your-registry/profibus:latest` 改为你的镜像仓库与标签。
- `deployments/kubernetes/ingress.yaml`：域名与 TLS 配置。

## 健康检查

部署完成后可使用项目自带的健康检查脚本：

```bash
./scripts/healthcheck.sh <主机或域名> <端口>
# 例如：./scripts/healthcheck.sh localhost 8080
```

## 架构图与更多文档

- 系统总体架构、技术分层、核心数据流等图示见本目录下的 PNG 文件。
- 项目整体介绍与本地开发说明见仓库根目录 [README.md](../README.md)。
