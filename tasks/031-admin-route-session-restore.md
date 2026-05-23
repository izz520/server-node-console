# 任务记录：管理员路由刷新时等待会话恢复

## 任务状态

已完成

## 任务背景

前端管理员路由通过 `user.role === "admin"` 判断权限。刷新 `/admin` 页面时，本地 token 会立即恢复，但用户信息需要通过 `SessionBootstrap` 请求 `/me` 或刷新 token 后再写入 store。在 user 还未恢复前，`RequireAdmin` 会误判为非管理员并重定向到概览页。

## 本次任务目标

修复管理员路由守卫的会话恢复竞态：当存在 token 但用户信息尚未恢复时，先保持等待状态，不提前重定向。

## 本次调整范围

### 前端

- `RequireAdmin` 同时读取 token 和 user。
- 当 token 存在但 user 为 `null` 时，渲染空等待状态。
- user 恢复后再判断管理员角色。

## 不包含范围

- 不改变后端管理员权限校验。
- 不持久化 user 到 localStorage。
- 不新增全局 loading 页面。

## 验收标准

- 管理员刷新 `/admin` 不会被提前踢回概览页。
- 普通用户访问 `/admin` 仍会重定向到概览页。
- `pnpm check`、`pnpm build` 通过。

## 风险与待确认

- 等待状态暂时为空白；后续可统一增加全局加载态。

## 完成记录

- `RequireAdmin` 已在 token 存在但 user 尚未恢复时等待。
- 管理员刷新 `/admin` 不会在会话恢复前被提前重定向。
- 普通用户在 user 恢复后仍会被重定向到概览页。

## 验证结果

- `pnpm check` 通过。
- `pnpm build` 通过。
