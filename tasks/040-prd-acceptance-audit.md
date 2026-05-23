# 任务记录：PRD 一期验收对照审计

## 任务状态

已完成

## 任务背景

项目已连续实现用户、服务器、节点、订阅、任务、管理员、安全日志等核心模块。为了确认“按照 PRD 循环执行直到功能全部实现”的目标，需要对 `PRD.md` 的总体验收标准做一次明确对照，标记已实现项和仅待后续迭代的非阻塞项。

## 本次任务目标

生成 PRD 一期验收审计文档，逐项对照当前实现，并识别是否仍存在需要继续开发的第一期功能缺口。

## 本次调整范围

### 文档

- 新增 `tasks/040-prd-acceptance-audit.md` 的验收对照内容。
- 按 PRD 总体验收标准列出实现状态。
- 标记后续迭代/待确认事项，不混入第一期完成范围。

## 不包含范围

- 不新增业务功能。
- 不替代自动化测试。

## 验收标准

- 文档能清楚说明 PRD 一期功能覆盖情况。
- 如发现第一期缺口，后续继续开任务实现。

## 风险与待确认

- 审计基于当前代码和 PRD 文档，不包含真实远端服务器 SSH 安装的人工验收。

## PRD 一期验收对照

| PRD 验收项 | 当前状态 | 证据 |
| --- | --- | --- |
| 用户可以注册、登录并进入系统 | 已实现 | `backend/internal/handler/handler.go`、`frontend/src/pages/login`、`frontend/src/pages/register` |
| 普通用户只能管理自己的服务器、节点和订阅 | 已实现 | 各资源查询/写入均按 `user_id` 过滤；后端测试覆盖跨用户访问 |
| 管理员可以只读查看所有用户数据 | 已实现 | `backend/internal/handler/admin_handler.go`、`frontend/src/pages/admin` |
| 管理员无法编辑或删除用户数据 | 已实现 | 管理员路由仅 GET，写接口仍走普通用户归属校验 |
| 添加服务器前完成 SSH 连通性测试 | 已实现 | `CreateServer` 调用 `testSSH` 成功后保存 |
| SSH 密码和私钥加密保存，页面不显示明文 | 已实现 | `backend/internal/security`、服务器响应仅返回 `hasPassword/hasPrivateKey` |
| 用户可以手动触发 SSH 连通性测试 | 已实现 | `POST /servers/:id/test-ssh` 和前端服务器页 |
| NAT 端口映射记录 | 已实现 | NAT CRUD 接口和服务器页 NAT 面板 |
| NAT 场景订阅优先使用对外端口 | 已实现 | `subscriptionNodeViews` 使用 `node.PublicPort` 优先 |
| 用户可以发起协议安装任务 | 已实现 | `POST /nodes/install`、节点页系统安装表单 |
| 安装任务异步执行并显示任务状态/日志 | 已实现 | `runInstallTask` goroutine、任务页轮询详情 |
| 安装成功节点可以加入订阅，失败节点不进入有效订阅 | 已实现 | 订阅节点校验只允许 `imported/install_success` |
| 用户可以卸载已安装节点 | 已实现 | `POST /nodes/:id/uninstall` |
| 卸载通过 SSH 执行脚本，成功后标记已卸载 | 已实现 | `runUninstallTask`、状态更新为 `uninstalled` |
| 已卸载节点不进入后续订阅内容 | 已实现 | 订阅渲染仅查询 `imported/install_success` |
| 外部节点手动添加 | 已实现 | `POST /nodes/import` manual 模式 |
| 外部节点分享链接导入 | 已实现 | `POST /nodes/import` link 模式，支持常见 URL 和 VMess |
| 创建包含多个节点的订阅 | 已实现 | `subscriptions` + `subscription_nodes` |
| 订阅支持 sing-box、Clash/Mihomo、v2rayN、Shadowrocket、Base64 | 已实现 | `renderSubscription` 分格式输出 |
| 每个订阅独立随机 token | 已实现 | 32 字节随机 token，哈希查询，加密保存 |
| 重置 token 后旧链接失效 | 已实现 | token hash 更新，测试覆盖旧链接 404 |
| 删除订阅后 token 失效 | 已实现 | 软删除订阅后公共查询 404，测试覆盖 |
| 禁用订阅后链接不可正常使用 | 已实现 | 公共订阅返回 403 |
| 操作日志记录关键操作 | 已实现 | 登录、服务器、NAT、节点、任务、订阅操作日志 |
| 管理员不能查看敏感明文 | 已实现 | 管理员订阅列表不返回 token；SSH/节点敏感字段不返回 |

## 第一期待确认但不阻塞项

- 用户禁用功能：PRD 标记为后续迭代。
- 登录失败次数限制和账号锁定：PRD 标记为待确认。
- 安装失败后编辑参数重试：PRD 标记为待确认。
- 卸载失败独立状态：PRD 标记为待确认，当前失败后保留安装成功状态并写任务错误日志。
- 订阅禁用响应格式：当前实现为 `403`，符合“不可正常使用”验收。
- 完整协议高级参数：当前采用原始链接优先和基础字段兜底，后续可按协议逐个增强。

## 完成记录

- 已按 PRD 总体验收标准完成一期功能对照。
- 未发现仍需立即补齐的第一期阻塞功能缺口。
- 已明确记录待确认/后续迭代项，避免误判为当前范围未完成。

## 验证结果

- 文档审计任务，无需执行编译验证。
