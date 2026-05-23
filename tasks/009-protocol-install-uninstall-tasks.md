# 任务记录：协议安装与卸载异步任务

## 任务状态

已完成

## 任务背景

根据 `PRD.md`，用户可以选择协议和服务器，系统通过 SSH 自动登录服务器执行安装脚本；安装和卸载过程耗时较长，需要以后端任务形式执行，并在前端协议列表显示状态和任务日志。当前服务器管理、节点管理、任务日志基础查询已完成，可以实现协议安装和卸载任务。

## 本次任务目标

实现系统安装协议节点的任务创建、后台执行、SSH 脚本执行、任务日志记录和节点状态流转。

## 本次调整范围

### 后端

- 实现安装协议接口：`POST /api/v1/nodes/install`
- 实现卸载协议接口：`POST /api/v1/nodes/:id/uninstall`
- 安装接口创建节点记录和安装任务，节点初始状态为 `installing`。
- 卸载接口创建卸载任务，节点状态更新为 `uninstalling`。
- 后端使用 goroutine 异步执行任务。
- 任务日志记录 SSH 连接、脚本生成、脚本输出、成功/失败原因。
- 安装成功后节点状态更新为 `install_success`。
- 安装失败后节点状态更新为 `install_failed`。
- 卸载成功后节点状态更新为 `uninstalled`，后续订阅不再包含该节点。
- 卸载失败后记录错误并保留节点状态。
- 支持第一期 argosbx 协议变量映射：
  - AnyTLS -> `anpt`
  - Any-reality -> `arpt`
  - Vless-xhttp-reality-vision-enc -> `xhpt`
  - Vless-tcp-reality-vision -> `vlpt`
  - Vless-xhttp-vision-enc -> `vxpt`
  - Vless-ws-vision-enc -> `vwpt`
  - Shadowsocks-2022 -> `sspt`
  - Hysteria2 -> `hypt`
  - Tuic -> `tupt`
  - Socks5 -> `sopt`
  - Vmess-ws -> `vmpt`
- 支持可选参数：端口、uuid、reality 域名、CDN 域名、Argo 配置、节点名称前缀。

### 前端

- 协议节点页面增加“系统安装”模式。
- 用户可选择服务器、协议和参数后发起安装。
- 安装后节点列表立即出现 `installing` 状态。
- 已安装节点支持点击卸载。
- 卸载后节点状态显示 `uninstalling` / `uninstalled`。

## 不包含范围

- 不完整解析 argosbx 安装输出中的所有真实节点链接。
- 不实现任务重试。
- 不实现 WebSocket 实时日志推送；前端通过刷新/重新查询查看状态。
- 不实现 passphrase 私钥。

## 验收标准

- 用户可以选择自己的服务器发起协议安装。
- 安装接口立即返回节点和任务信息。
- 后端异步执行安装任务并写入任务日志。
- 安装任务成功/失败后更新任务状态和节点状态。
- 用户可以卸载系统安装节点。
- 卸载任务成功后节点状态为 `uninstalled`。
- 用户不能安装到其他用户服务器，也不能卸载其他用户节点。
- 前端协议节点页可以发起安装和卸载。
- `go test ./...`、`pnpm check`、`pnpm build` 通过。

## 风险与待确认

- 第一版将 argosbx 脚本输出保存到任务日志，但节点订阅参数先基于用户输入的协议、服务器地址和端口生成，后续可增强为解析脚本输出链接。
- 远程安装依赖目标服务器可访问 GitHub raw；如果目标服务器网络无法访问，任务会失败并记录日志。

## 完成记录

- 已实现安装协议接口 `POST /api/v1/nodes/install`。
- 已实现卸载协议接口 `POST /api/v1/nodes/:id/uninstall`。
- 安装和卸载都会创建任务，并使用 goroutine 异步执行。
- 已实现 argosbx 协议变量映射和安装/卸载脚本生成。
- 已实现 SSH 远程命令执行，并将输出写入任务日志。
- 安装成功/失败会更新任务状态和节点状态。
- 卸载成功会将节点标记为 `uninstalled`，卸载失败会记录错误。
- 前端协议节点页已增加“系统安装”模式和卸载入口。
- 已通过 `go test ./...`、`pnpm check`、`pnpm build`。
