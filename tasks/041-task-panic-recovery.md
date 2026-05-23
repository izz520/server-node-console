# 任务记录：异步任务 panic 恢复

## 任务状态

已完成

## 任务背景

协议安装/卸载任务通过 goroutine 异步执行。Go 中 goroutine 内未恢复的 panic 会导致整个进程退出。安装/卸载涉及 SSH、脚本生成、外部命令输出等不确定因素，需要确保异常能落到任务失败状态和任务日志，而不是打崩 API 服务。

## 本次任务目标

为安装和卸载任务执行入口增加 panic recover，将异常转换为任务失败并写入日志。

## 本次调整范围

### 后端

- `runInstallTask` 增加 `defer recover`。
- `runUninstallTask` 增加 `defer recover`。
- panic 时调用对应失败处理逻辑，记录错误信息。

## 不包含范围

- 不新增任务重试。
- 不改变正常错误处理流程。

## 验收标准

- 安装/卸载任务内部 panic 不会导致进程退出。
- panic 会使任务进入失败状态并写入错误日志。
- `go test ./...` 通过。

## 风险与待确认

- recover 只能保护任务 goroutine 内部，不能替代完整的任务队列系统。

## 完成记录

- `runInstallTask` 已增加 panic recover。
- `runUninstallTask` 已增加 panic recover。
- panic 会进入对应失败处理逻辑并写入任务日志。

## 验证结果

- `go test ./...` 通过。
