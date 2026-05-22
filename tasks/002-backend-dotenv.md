# 任务记录：后端自动加载 .env 配置

## 任务状态

已完成

## 任务背景

当前后端配置通过系统环境变量读取。如果直接执行 `go run ./cmd/api`，后端不会自动读取 `backend/.env`，导致仍然使用代码默认的 `DATABASE_DSN`，例如默认用户 `postgres`。用户已经在 `backend/.env` 中配置了本机数据库连接信息，需要后端启动时自动加载该文件。

## 本次任务目标

让后端在启动时自动加载 `.env` 文件，使用户在 `backend` 目录直接执行 `go run ./cmd/api` 时可以使用 `backend/.env` 中的配置。

## 本次调整范围

### 后端

- 增加 `.env` 加载能力。
- 启动时优先读取当前工作目录下的 `.env`。
- 保持系统环境变量优先级：如果 shell 中已设置同名环境变量，应优先使用 shell 环境变量。
- 更新后端 README 的启动说明。

## 不包含范围

- 不修改数据库模型。
- 不修改认证逻辑。
- 不实现自动创建数据库。
- 不修改前端。

## 验收标准

- 在 `backend/.env` 中配置 `DATABASE_DSN` 后，执行 `go run ./cmd/api` 会使用 `.env` 中的数据库连接。
- shell 中已有环境变量时，不被 `.env` 覆盖。
- 后端仍可在没有 `.env` 的情况下使用代码默认配置启动。
- `go test ./...` 通过。

## 风险与待确认

- 建议使用 `github.com/joho/godotenv` 作为 `.env` 加载库。

## 完成记录

- 已引入 `github.com/joho/godotenv`。
- 后端启动时会自动加载当前目录 `.env`。
- 兼容从项目根目录启动时加载 `backend/.env`。
- `.env` 不会覆盖 shell 中已经存在的同名环境变量。
- 已更新后端 README 启动说明。
- 已通过 `go test ./...`。
