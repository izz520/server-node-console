import { useQuery } from "@tanstack/react-query";
import { KeyRound, RotateCcw, ShieldCheck } from "lucide-react";
import { listOperationLogs } from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import type { OperationLog } from "@/types/domain";

const actionLabels: Record<string, string> = {
  "auth.login": "登录",
  "server.create": "添加服务器",
  "server.update": "编辑服务器",
  "server.delete": "删除服务器",
  "server.test_ssh.success": "SSH 测试成功",
  "server.test_ssh.failed": "SSH 测试失败",
  "nat_mapping.create": "添加 NAT 映射",
  "nat_mapping.update": "编辑 NAT 映射",
  "nat_mapping.delete": "删除 NAT 映射",
  "node.import": "导入节点",
  "node.update": "编辑节点",
  "node.delete": "删除节点",
  "node.install.start": "发起安装",
  "node.uninstall.start": "发起卸载",
  "subscription.create": "创建订阅",
  "subscription.update": "编辑订阅",
  "subscription.delete": "删除订阅",
  "subscription.reset_token": "重置订阅 Token",
};

export function SecurityPage() {
  const logs = useQuery({
    queryKey: ["operation-logs"],
    queryFn: listOperationLogs,
  });

  const errorCount =
    logs.data?.filter(
      (item) =>
        item.action.endsWith(".failed") || item.action.endsWith("_failed"),
    ).length ?? 0;

  return (
    <div className="space-y-6">
      <section className="flex flex-col justify-between gap-4 md:flex-row md:items-center">
        <div className="flex items-center gap-4">
          <div className="flex h-11 w-11 items-center justify-center rounded-md bg-slate-950 text-white">
            <KeyRound className="h-5 w-5" />
          </div>
          <div>
            <h1 className="font-semibold text-2xl text-slate-950">安全中心</h1>
            <p className="mt-1 text-slate-600 text-sm">
              查看账号关键操作记录，敏感凭据和 token 不展示明文。
            </p>
          </div>
        </div>
        <Button onClick={() => logs.refetch()} variant="secondary">
          <RotateCcw className="h-4 w-4" />
          刷新
        </Button>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent>
            <div className="text-slate-500 text-sm">操作记录</div>
            <div className="mt-2 font-semibold text-2xl text-slate-950">
              {logs.data?.length ?? 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent>
            <div className="text-slate-500 text-sm">失败事件</div>
            <div className="mt-2 font-semibold text-2xl text-slate-950">
              {errorCount}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center justify-between">
            <div>
              <div className="text-slate-500 text-sm">数据范围</div>
              <div className="mt-2 font-medium text-slate-950">仅当前账号</div>
            </div>
            <ShieldCheck className="h-5 w-5 text-emerald-600" />
          </CardContent>
        </Card>
      </section>

      <Card>
        <CardHeader>
          <div className="font-medium text-slate-950">最近操作</div>
        </CardHeader>
        <CardContent>
          <OperationLogTable logs={logs.data ?? []} loading={logs.isLoading} />
        </CardContent>
      </Card>
    </div>
  );
}

function OperationLogTable({
  logs,
  loading,
}: {
  logs: OperationLog[];
  loading: boolean;
}) {
  if (loading) {
    return (
      <div className="rounded-md border border-dashed border-slate-200 p-6 text-center text-slate-500 text-sm">
        正在加载操作日志
      </div>
    );
  }

  if (logs.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-slate-200 p-6 text-center text-slate-500 text-sm">
        暂无操作日志
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[760px] border-collapse text-left text-sm">
        <thead>
          <tr className="border-slate-100 border-b text-slate-500">
            <th className="py-2 pr-3 font-medium">时间</th>
            <th className="py-2 pr-3 font-medium">动作</th>
            <th className="py-2 pr-3 font-medium">资源</th>
            <th className="py-2 pr-3 font-medium">记录内容</th>
          </tr>
        </thead>
        <tbody>
          {logs.map((log) => (
            <tr className="border-slate-100 border-b" key={log.id}>
              <td className="whitespace-nowrap py-2 pr-3 text-slate-600">
                {formatDate(log.createdAt)}
              </td>
              <td className="py-2 pr-3">
                <Badge>{actionLabels[log.action] ?? log.action}</Badge>
              </td>
              <td className="py-2 pr-3 text-slate-700">{log.resource}</td>
              <td className="max-w-[360px] py-2 pr-3 text-slate-500">
                {formatMetadata(log.metadata)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function formatMetadata(value: string) {
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    const entries = Object.entries(parsed);
    if (entries.length === 0) {
      return "-";
    }
    return entries
      .map(([key, metadataValue]) => `${key}: ${String(metadataValue)}`)
      .join(" / ");
  } catch {
    return value || "-";
  }
}
