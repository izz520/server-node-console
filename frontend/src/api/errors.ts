export function getErrorMessage(error: unknown, fallback: string) {
  if (typeof error === "object" && error && "response" in error) {
    const response = (
      error as { response?: { data?: { details?: string; error?: string } } }
    ).response;
    const message = response?.data?.details ?? response?.data?.error;
    return localizeError(message) ?? fallback;
  }
  return fallback;
}

const errorMessages: Record<string, string> = {
  "check server references failed": "检查服务器关联数据失败",
  "check server nodes failed": "检查服务器节点失败",
  "check server NAT mappings failed": "检查服务器 NAT 映射失败",
  "create node failed": "创建节点失败",
  "create server failed": "创建服务器失败",
  "create subscription failed": "创建订阅失败",
  "delete node failed": "删除节点失败",
  "delete server failed": "删除服务器失败",
  "delete subscription failed": "删除订阅失败",
  "encryption is not configured": "加密配置未正确设置",
  "generate subscription token failed": "生成订阅 token 失败",
  "invalid account or password": "账号或密码不正确",
  "invalid request": "请求参数不正确",
  "node not found": "节点不存在或无权访问",
  "one or more nodes are invalid": "一个或多个节点不可用",
  "server has nodes; uninstall or delete nodes first":
    "服务器下仍有节点，请先卸载或删除节点",
  "server has NAT mappings; delete NAT mappings first":
    "服务器下仍有 NAT 映射，请先删除 NAT 映射",
  "server has nodes referenced by subscriptions; remove them from subscriptions first":
    "服务器节点仍被订阅引用，请先从订阅中移除",
  "server is not ready for installation":
    "服务器当前不可安装，请先完成 SSH 连通性测试",
  "server is not ready for uninstallation":
    "服务器当前不可卸载，请先完成 SSH 连通性测试",
  "server not found": "服务器不存在或无权访问",
  "ssh connection failed": "SSH 连接失败",
  "subscription disabled": "订阅已禁用",
  "subscription must include at least one node": "订阅至少需要包含一个节点",
  "subscription not found": "订阅不存在或已失效",
  "system installed node must be uninstalled before deletion":
    "系统安装节点需要先卸载，不能直接删除",
  "unsupported import mode": "不支持的导入方式",
  "unsupported subscription format": "不支持的订阅格式",
  "update node failed": "更新节点失败",
  "update server failed": "更新服务器失败",
  "update subscription failed": "更新订阅失败",
  "user not found": "用户不存在",
  "username and email are required": "用户名和邮箱不能为空",
  "username or email already exists": "用户名或邮箱已被使用",
};

function localizeError(message?: string) {
  if (!message) {
    return undefined;
  }
  return errorMessages[message] ?? message;
}
