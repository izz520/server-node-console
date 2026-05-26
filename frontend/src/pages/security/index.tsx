import { useQuery } from "@tanstack/react-query";
import { KeyRound, RotateCcw, ShieldCheck } from "lucide-react";
import { listOperationLogs } from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { cn } from "@/lib/utils";
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
    <div className="space-y-8 py-4 max-w-7xl mx-auto">
      {/* Page Header */}
      <section className="flex flex-col justify-between gap-6 sm:flex-row sm:items-center">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/[0.04] bg-white/[0.02] text-slate-300">
            <KeyRound className="h-4 w-4 text-[#6366f1]" />
          </div>
          <div>
            <h1 className="font-bold text-2xl lg:text-3xl text-slate-100 tracking-tight font-display">
              安全中心
            </h1>
            <p className="mt-1 text-slate-400 text-xs font-semibold">
              监控您账号的关键事务日志与操作历史，敏感私钥与密码出于安全隐去明文展示。
            </p>
          </div>
        </div>
        <Button
          onClick={() => logs.refetch()}
          variant="secondary"
          className="h-9 px-4 text-xs font-semibold self-start sm:self-center"
        >
          <RotateCcw className="h-4 w-4 text-slate-400" />
          <span>刷新记录</span>
        </Button>
      </section>

      {/* Metrics Row */}
      <section className="grid gap-5 md:grid-cols-3">
        <Card className="bg-[#0e1017]/70 border-white/[0.04]">
          <CardContent className="p-6">
            <div className="text-slate-500 text-[10px] font-bold uppercase tracking-wider">
              操作动作总计数
            </div>
            <div className="mt-2.5 font-bold text-3xl text-white tracking-tight font-display">
              {logs.isLoading ? (
                <div className="h-9 w-12 animate-pulse rounded bg-slate-800/60" />
              ) : (
                (logs.data?.length ?? 0)
              )}
            </div>
          </CardContent>
        </Card>
        <Card className="bg-[#0e1017]/70 border-white/[0.04]">
          <CardContent className="p-6">
            <div className="text-slate-500 text-[10px] font-bold uppercase tracking-wider">
              拦截/失败事务
            </div>
            <div className="mt-2.5 font-bold text-3xl text-white tracking-tight font-display">
              {logs.isLoading ? (
                <div className="h-9 w-12 animate-pulse rounded bg-slate-800/60" />
              ) : (
                errorCount
              )}
            </div>
          </CardContent>
        </Card>
        <Card className="bg-[#0e1017]/70 border-white/[0.04]">
          <CardContent className="flex items-center justify-between p-6">
            <div>
              <div className="text-slate-500 text-[10px] font-bold uppercase tracking-wider">
                操作审计范围
              </div>
              <div className="mt-2.5 font-bold text-sm text-slate-200">
                仅限当前认证账号
              </div>
            </div>
            <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-white/[0.04] bg-white/[0.02] text-emerald-400">
              <ShieldCheck className="h-4 w-4" />
            </div>
          </CardContent>
        </Card>
      </section>

      {/* Operation Log Table Card */}
      <Card className="bg-[#0e1017]/70 border-white/[0.04]">
        <CardHeader className="p-5 border-white/[0.04]">
          <div className="font-bold text-slate-100 text-sm tracking-wide">
            操作审计历史流水
          </div>
          <div className="text-[10px] text-slate-500 font-semibold uppercase tracking-wider mt-0.5">
            账号全量关键事务日志归档
          </div>
        </CardHeader>
        <CardContent className="p-0">
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
      <div className="p-12 text-center text-slate-400 text-xs font-semibold animate-pulse border-t border-white/[0.04]">
        正在提取安全日志流水...
      </div>
    );
  }

  if (logs.length === 0) {
    return (
      <div className="p-16 text-center text-slate-500 text-xs font-semibold border-t border-white/[0.04]">
        本月暂无安全敏感操作日志归档。
      </div>
    );
  }

  return (
    <div className="overflow-x-auto border-t border-white/[0.04]">
      <table className="w-full min-w-[760px] border-collapse text-left text-sm">
        <thead>
          <tr className="border-white/[0.04] border-b text-slate-400 text-xs font-semibold uppercase tracking-wider bg-slate-900/10">
            <th className="py-4 px-6 font-medium">执行时间</th>
            <th className="py-4 px-6 font-medium">操作类型</th>
            <th className="py-4 px-6 font-medium">目标资源</th>
            <th className="py-4 px-6 font-medium">资源 Metadata 参数描述</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/[0.04]">
          {logs.map((log) => {
            const isError =
              log.action.endsWith(".failed") || log.action.endsWith("_failed");

            const badgeClass = isError
              ? "border-rose-500/10 bg-rose-500/5 text-rose-400 font-medium"
              : "border-slate-800 bg-slate-900/60 text-slate-300 font-medium";

            return (
              <tr
                className="hover:bg-white/[0.01] transition-colors duration-200"
                key={log.id}
              >
                <td className="whitespace-nowrap py-4 px-6 text-slate-400 font-mono text-xs">
                  {formatDate(log.createdAt)}
                </td>
                <td className="py-4 px-6">
                  <Badge className={badgeClass}>
                    <span
                      className={cn(
                        "mr-1.5 h-1.5 w-1.5 rounded-full shrink-0",
                        isError ? "bg-rose-500" : "bg-slate-400",
                      )}
                    />
                    {actionLabels[log.action] ?? log.action}
                  </Badge>
                </td>
                <td className="py-4 px-6 text-slate-200 font-semibold font-mono text-xs">
                  {log.resource}
                </td>
                <td
                  className="max-w-[360px] py-4 px-6 text-slate-500 font-mono text-xs truncate"
                  title={formatMetadata(log.metadata)}
                >
                  {formatMetadata(log.metadata)}
                </td>
              </tr>
            );
          })}
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
