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
