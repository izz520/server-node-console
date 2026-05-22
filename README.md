# 服务器管理与 sing-box 节点订阅管理系统

这是一个根据 `需求文档.md` 和 `PRD.md` 搭建的全栈脚手架。

## 目录

```text
backend/   Go + Gin + GORM + JWT 后端
frontend/  React + Vite + TypeScript + Tailwind CSS 前端
```

## 本地开发

启动 PostgreSQL：

```bash
docker compose up -d postgres
```

启动后端：

```bash
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/api
```

启动前端：

```bash
cd frontend
pnpm install
pnpm dev
```

默认地址：

- 前端：http://localhost:5173
- 后端：http://localhost:8080
- 健康检查：http://localhost:8080/healthz

## 当前状态

- 后端已完成配置、数据库连接、GORM 模型、JWT 中间件和 API 路由骨架。
- 前端已完成 Vite、路由、布局、登录占位页、Dashboard、资源占位页、API client 和状态管理骨架。
- 业务接口目前大多返回 `501 Not Implemented`，下一步可以按模块逐步实现。
