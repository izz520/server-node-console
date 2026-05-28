# 服务器管理与 sing-box 节点订阅管理系统

这是一个根据 `需求文档.md` 和 `PRD.md` 实现的多用户 SaaS 控制台，用于管理服务器、通过 SSH 安装/卸载 sing-box 协议节点，并生成多客户端订阅链接。

## 目录

```text
backend/   Go + Gin + GORM + JWT 后端
frontend/  React + Vite + TypeScript + Tailwind CSS 前端
tasks/     每次功能变更的任务记录文档
```

## 已实现能力

- 用户注册、登录、JWT 自动续期。
- 普通用户数据隔离。
- 管理员只读后台。
- 服务器管理、SSH 凭据加密保存、SSH 连通性测试。
- NAT 端口映射记录。
- 系统安装/卸载协议节点的异步任务和任务日志。
- 外部节点手动添加和分享链接导入。
- 订阅创建、编辑、删除、启用/禁用、token 重置。
- sing-box、Clash/Mihomo、v2rayN、Shadowrocket、Base64 订阅输出。
- 操作日志和安全中心。

## Docker Compose 部署

推荐用 Docker Compose 直接启动数据库、后端和前端。

### 1. 准备环境变量

在项目根目录创建 `.env`：

```env
POSTGRES_PASSWORD=你的数据库密码
JWT_SECRET=随机长字符串
ENCRYPTION_KEY=随机长字符串
```

生成随机字符串可以用：

```bash
openssl rand -hex 32
```

说明：

- `POSTGRES_PASSWORD` 是 PostgreSQL 容器初始化数据库用户时使用的密码。
- `JWT_SECRET` 用于签发和校验登录 token，部署后不要随意修改。
- `ENCRYPTION_KEY` 用于加密服务器 SSH 凭据等敏感数据，部署后不要随意修改，否则旧数据可能无法解密。

其他配置都有默认值，通常不需要填写：

- 数据库用户默认：`postgres`
- 数据库名默认：`singbox_manager`
- 数据库端口默认只监听本机：`127.0.0.1:5432`
- 后端默认监听公网：`0.0.0.0:8080`
- 前端默认监听公网：`0.0.0.0:4173`
- 跨域默认允许全部来源：`CORS_ALLOWED_ORIGINS=*`

### 2. 启动服务

在项目根目录执行：

```bash
docker compose up -d --build
```

第一次启动会下载镜像和安装依赖，耗时会稍长。

### 3. 访问地址

前端：

```text
http://服务器IP:4173
```

本机测试：

```text
http://localhost:4173
```

后端健康检查：

```bash
curl http://localhost:8080/healthz
```

前端默认会自动请求当前访问主机的后端端口：

```text
http://当前访问IP或域名:8080/api/v1
```

### 4. 常用命令

查看服务状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f backend
docker compose logs -f frontend
```

停止服务：

```bash
docker compose down
```

更新代码后重新部署：

```bash
docker compose up -d --build
```

### 5. 可选配置

只有需要改端口或绑定地址时，才需要在 `.env` 里增加这些变量：

```env
POSTGRES_BIND=127.0.0.1:5432
BACKEND_BIND=0.0.0.0:8080
FRONTEND_BIND=0.0.0.0:4173
```

如果后端通过单独域名访问，可以指定：

```env
VITE_API_BASE_URL=https://api.example.com/api/v1
```

### 6. 节点可用性检查

节点页面支持两种可用性检查：

- 单个节点检查：点击节点卡片里的检查按钮。
- 全部节点检查：点击节点列表顶部的“全部检查”。

检查会在后端临时启动 Mihomo，通过节点访问：

```text
https://cp.cloudflare.com/generate_204
```

成功后会显示：

- 节点是否可用
- 服务端到该节点的响应耗时
- 出口 IP
- 出口国家/地区

响应耗时只代表后端服务器发起检查时的链路表现，不代表用户本地客户端的实际延迟。

后端 Docker 镜像会内置 Mihomo。如果你自己构建后端镜像，可以通过构建参数指定版本：

```bash
docker build --build-arg MIHOMO_VERSION=v1.19.22 -t your-backend-image ./backend
```

如果源码方式启动后端，可以把 Mihomo 可执行文件放在项目内：

```text
backend/bin/mihomo
```

如果使用自定义 Mihomo 路径，可以在后端环境变量里指定：

```env
MIHOMO_BIN=/usr/local/bin/mihomo
```

如果需要和本地 Mihomo 客户端的 url-test 配置完全对齐，可以指定相同的检查 URL：

```env
PROXY_TEST_URL=https://cp.cloudflare.com/generate_204
```

## 本地开发

### 1. 准备 PostgreSQL

使用 Docker Compose：

```bash
docker compose up -d postgres
```

或使用本机 PostgreSQL。数据库需要先存在，后端启动时会自动迁移表结构，但不会自动创建数据库本身。

```bash
createdb -h 127.0.0.1 -p 5432 -U <你的数据库用户> singbox_manager
```

### 2. 配置后端环境变量

```bash
cd backend
cp .env.example .env
```

如果使用本机 PostgreSQL，请按实际用户修改 `DATABASE_DSN`，例如：

```text
DATABASE_DSN=host=127.0.0.1 user=yasol dbname=singbox_manager port=5432 sslmode=disable TimeZone=Asia/Shanghai
```

后端启动时会自动加载当前目录或 `backend/.env`。Shell 环境变量优先级高于 `.env`。

### 3. 启动后端

```bash
cd backend
go run ./cmd/api
```

默认地址：

- 后端：http://localhost:8080
- 健康检查：http://localhost:8080/healthz

### 4. 启动前端

```bash
cd frontend
pnpm install
pnpm dev
```

默认地址：

- 前端：http://localhost:5173

## 常用验证

后端：

```bash
cd backend
go test ./...
```

前端：

```bash
cd frontend
pnpm check
pnpm build
```
# server-node-console
