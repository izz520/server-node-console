# 任务记录：Clash/Mihomo 完整订阅模板

## 任务状态

已完成

## 任务背景

此前 `clash-mihomo` 订阅只输出 `proxies:` 片段，缺少 Clash/Mihomo 客户端通常需要的全局配置、DNS、代理组和规则。用户使用 Clash 格式订阅时，拿到的 YAML 不够完整，客户端导入后可能无法按预期直接使用。

## 本次任务目标

将 Clash/Mihomo 订阅输出从代理片段升级为完整 YAML 配置模板，并将已有节点字段填入代理、代理组和规则中。

## 本次调整范围

### 后端

- `clash-mihomo` 输出完整 YAML 骨架：
  - `mixed-port`
  - `allow-lan`
  - `mode`
  - `log-level`
  - `dns`
  - `proxies`
  - `proxy-groups`
  - `rules`
- 代理组包含：
  - `PROXY` 手动选择组
  - `AUTO` 延迟测试组
- 规则默认包含：
  - `GEOIP,LAN,DIRECT`
  - `GEOIP,CN,DIRECT`
  - `MATCH,PROXY`
- YAML 字符串使用 JSON 风格安全引号，避免节点名里的特殊字符破坏 YAML。
- AnyTLS、Hysteria2、Tuic、VLESS、VMess 在有可用敏感字段时补充密码或 UUID。
- 敏感参数解析兼容 JSON 和 `key=value` / `key:value` 文本。

### 测试

- 更新 Clash/Mihomo 订阅测试，断言完整模板字段。
- 增加 AnyTLS Clash/Mihomo 输出包含 `password` 的测试。

## 不包含范围

- 不完整实现所有协议的专属 Clash/Mihomo 高级参数。
- 不接入远程规则集或第三方订阅转换器。
- 不保证所有 Clash 分支客户端都支持 AnyTLS；本项目目标格式按 Mihomo 方向输出。
- 不改变 sing-box、v2rayN、Shadowrocket、Base64 的输出路径。

## 验收标准

- `clash-mihomo` 订阅包含完整 YAML 骨架。
- 输出包含 `proxies`、`proxy-groups`、`rules`。
- 节点名称被加入代理组。
- NAT 公网端口仍优先输出。
- AnyTLS 节点在有 UUID 时输出 `password` 字段。
- 后端 handler 测试通过。

## 完成记录

- 已新增 `renderClashMihomo` 完整模板渲染。
- 已新增节点级 `clashProxyLines`。
- 已新增敏感参数兼容解析。
- 已补充 Clash/Mihomo 完整模板测试。
- 已补充 AnyTLS Clash/Mihomo 密码字段测试。

## 验证结果

- `go test ./internal/handler` 通过。
