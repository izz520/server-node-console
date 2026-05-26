# 任务记录：Clash/Mihomo 订阅模板选择

## 任务状态

已完成

## 任务背景

Clash/Mihomo 订阅已经升级为完整 YAML 模板，但不同用户使用场景不同：有些需要规则模式国内直连，有些需要全局代理。订阅创建时需要允许用户选择模板，而不是固定输出单一配置。

## 本次任务目标

在订阅配置中新增 Clash/Mihomo 模板选择能力，公共订阅输出时按用户选择的模板生成 YAML。

## 本次调整范围

### 后端

- `Subscription` 新增 `clashTemplate` 字段。
- 创建、编辑、列表、详情响应支持 `clashTemplate`。
- `clashTemplate` 统一规范化：
  - `rule-cn`：规则模式，国内直连。
  - `global-proxy`：全局代理，除局域网外走代理。
- Clash/Mihomo 渲染函数根据模板输出不同的 `mode` 和 `rules`。
- 未传模板或非法模板时回退到 `rule-cn`。

### 前端

- 创建/编辑订阅时，选择 `Clash / Mihomo` 格式后显示“Clash 模板”下拉框。
- 支持选择：
  - 规则模式：国内直连
  - 全局代理：除局域网外走代理
- 订阅列表中展示 Clash 模板标签。

### 测试

- 增加 Clash/Mihomo 模板选择测试。
- 验证 `global-proxy` 输出 `mode: global`，并不包含 `GEOIP,CN,DIRECT`。

## 不包含范围

- 不支持用户自定义 YAML 文本模板。
- 不支持远程规则集模板。
- 不增加模板 CRUD 管理页面。
- 不改变非 Clash/Mihomo 格式的订阅输出。

## 验收标准

- Clash/Mihomo 订阅可以保存模板选择。
- 公共订阅链接按模板输出不同规则。
- 非 Clash 订阅不显示模板选择。
- 后端测试通过。
- 前端构建通过。

## 完成记录

- 已新增订阅级 `clashTemplate`。
- 已新增 `rule-cn` 和 `global-proxy` 两个内置模板。
- 已完成前端模板下拉选择。
- 已完成订阅列表模板展示。
- 已补充模板选择测试。

## 验证结果

- `go test ./internal/handler` 通过。
- `pnpm build` 通过。
