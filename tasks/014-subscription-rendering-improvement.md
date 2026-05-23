# 任务记录：订阅格式输出增强

## 任务状态

已完成

## 任务背景

`PRD.md` 要求订阅支持 sing-box、Clash/Mihomo、v2rayN、Shadowrocket 和通用 Base64 格式，并且 NAT 场景下优先使用对外访问端口。当前订阅接口已经支持格式分支和 NAT 端口优先，但输出内容仍较基础，需要增强为更接近客户端可用的结构，并在导入分享链接时尽量保留原始链接。

## 本次任务目标

增强订阅渲染逻辑：保留导入分享链接，改进 sing-box JSON 和 Clash/Mihomo YAML 输出，统一 v2rayN、Shadowrocket 和 Base64 的链接生成规则，并增加测试覆盖。

## 本次调整范围

### 后端

- 订阅节点视图携带 `rawLink` 和 `configJson`。
- v2rayN、Shadowrocket、Base64 优先输出导入时的原始分享链接。
- sing-box 输出标准 `outbounds` 结构，支持手动 `configJson` 覆盖。
- Clash/Mihomo 输出 `proxies` 结构，并使用协议映射后的 type。
- 继续保持 NAT 对外端口优先。
- 增加测试覆盖 Base64 原始链接、sing-box 结构、Clash/Mihomo 结构和 NAT 公网端口。

## 不包含范围

- 不完整实现所有协议的客户端专属高级字段。
- 不解析加密保存的敏感字段生成完整协议密钥。
- 不调用第三方订阅转换器。

## 验收标准

- 导入分享链接生成 v2rayN/Shadowrocket/Base64 时优先保持原始链接。
- sing-box 输出 JSON 包含 `outbounds`，且节点包含 `type`、`tag`、`server`、`server_port`。
- Clash/Mihomo 输出 YAML `proxies`。
- NAT 对外端口优先规则通过测试覆盖。
- `go test ./...` 通过。

## 风险与待确认

- 第一版对客户端高级字段采用“原始链接优先、基础字段兜底”的方式，后续可按协议逐个补全专属参数。

## 完成记录

- 订阅节点视图已携带 `rawLink` 和 `configJson`。
- v2rayN、Shadowrocket、Base64 已优先输出导入时的原始分享链接。
- sing-box 已输出 `outbounds`，并支持手动 `configJson` 覆盖。
- Clash/Mihomo 已输出 `proxies` YAML 结构。
- 已增加测试覆盖原始链接、Base64、Clash/Mihomo、sing-box 和 NAT 对外端口。

## 验证结果

- `go test ./...` 通过。
