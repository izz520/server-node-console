# 任务记录：修复 SSH 命令执行阶段超时失效

## 任务状态

已完成

## 任务背景

真实 AnyTLS 安装验证中，安装任务进入 `running` 后长时间无结束状态。排查发现 `sshclient.RunCommand` 当前把 `Timeout` 主要用于 SSH 连接和认证阶段，远端命令执行阶段没有被 `context` 或 timeout 控制；当远端安装脚本阻塞时，后端任务会长期停留在运行中。

## 本次任务目标

让 SSH 命令执行阶段也受 `Timeout` 控制。超时后关闭 SSH 会话并返回明确错误，让安装/卸载任务能进入失败态，而不是一直卡在 `running`。

## 本次调整范围

### 后端

- 调整 `backend/internal/sshclient/tester.go` 的 `RunCommand`。
- 将连接、认证和命令执行统一纳入同一个超时上下文。
- 超时后关闭 SSH session/client/conn，并返回包含超时时长的错误。

### 验证

- 运行 `go test ./...`。
- 重新跑 AnyTLS 真实安装验证，观察任务是否能在后端超时边界内正确落到失败态或继续完成。

## 不包含范围

- 不实现 SSH 输出流式日志。
- 不新增任务取消 API。
- 不更改远端 argosbx 脚本。

## 验收标准

- `RunCommand` 不会因为远端命令不返回而无限等待。
- 超时错误会被安装/卸载任务记录到任务日志。
- `go test ./...` 通过。

## 执行结果

- `RunCommand` 已将远端命令执行纳入 timeout 上下文。
- 使用真实 SSH 执行 `sleep 10` 并设置 3 秒 timeout，已确认会返回 `run ssh command timeout after 3s`。
- 后端测试通过。
