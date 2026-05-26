import { useQueries, useQuery } from "@tanstack/react-query";
import { ListChecks, ShieldCheck } from "lucide-react";
import { useEffect, useState } from "react";
import {
  getAdminTask,
  listAdminNodes,
  listAdminOperationLogs,
  listAdminServers,
  listAdminSubscriptions,
  listAdminTasks,
  listAdminUsers,
} from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function AdminPage() {
  const [selectedTaskID, setSelectedTaskID] = useState<number | null>(null);
  const [users, servers, nodes, subscriptions, tasks, operationLogs] =
    useQueries({
      queries: [
        { queryKey: ["admin", "users"], queryFn: listAdminUsers },
        { queryKey: ["admin", "servers"], queryFn: listAdminServers },
        { queryKey: ["admin", "nodes"], queryFn: listAdminNodes },
        {
          queryKey: ["admin", "subscriptions"],
          queryFn: listAdminSubscriptions,
        },
        {
          queryKey: ["admin", "tasks"],
          queryFn: listAdminTasks,
          refetchInterval: 5000,
        },
        {
          queryKey: ["admin", "operation-logs"],
          queryFn: listAdminOperationLogs,
        },
      ],
    });
  const taskRows = tasks.data ?? [];

  useEffect(() => {
    if (!selectedTaskID && taskRows.length > 0) {
      setSelectedTaskID(taskRows[0].id);
    }
  }, [selectedTaskID, taskRows]);

  const taskDetail = useQuery({
    queryKey: ["admin", "tasks", selectedTaskID],
    queryFn: () => getAdminTask(selectedTaskID ?? 0),
    enabled: Boolean(selectedTaskID),
    refetchInterval: 5000,
  });

  return (
    <div className="space-y-8 py-4 max-w-7xl mx-auto">
      {/* Page Header */}
      <section className="flex items-center gap-3">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/[0.04] bg-white/[0.02] text-slate-300">
          <ShieldCheck className="h-4 w-4 text-[#6366f1]" />
        </div>
        <div>
          <h1 className="font-bold text-2xl lg:text-3xl text-slate-100 tracking-tight font-display">
            系统管理后台
          </h1>
          <p className="mt-1 text-slate-400 text-xs font-semibold">
            只读监控系统内的全局状态数据，不支持越权编辑或删除其他用户的敏感资源。
          </p>
        </div>
      </section>

      {/* Stats metrics row */}
      <section className="grid gap-4 sm:grid-cols-3 xl:grid-cols-6">
        <StatCard label="全局用户" value={users.data?.length ?? 0} />
        <StatCard label="总服务器" value={servers.data?.length ?? 0} />
        <StatCard label="总部署节点" value={nodes.data?.length ?? 0} />
        <StatCard label="配发订阅" value={subscriptions.data?.length ?? 0} />
        <StatCard label="归档任务" value={tasks.data?.length ?? 0} />
        <StatCard
          label="系统审计日志"
          value={operationLogs.data?.length ?? 0}
        />
      </section>

      {/* Grid Tables */}
      <section className="grid gap-6 xl:grid-cols-2">
        <Card className="bg-[#0e1017]/70 border-white/[0.04]">
          <CardHeader className="p-5 border-white/[0.04]">
            <div className="font-bold text-slate-200 text-sm tracking-wide">
              系统注册用户
            </div>
          </CardHeader>
          <CardContent className="p-5">
            <Table
              columns={["用户名", "电子邮箱", "角色"]}
              rows={(users.data ?? []).map((user) => [
                user.username,
                user.email,
                user.role,
              ])}
            />
          </CardContent>
        </Card>

        <Card className="bg-[#0e1017]/70 border-white/[0.04]">
          <CardHeader className="p-5 border-white/[0.04]">
            <div className="font-bold text-slate-200 text-sm tracking-wide">
              全网物理服务器
            </div>
          </CardHeader>
          <CardContent className="p-5">
            <Table
              columns={[
                "归属用户ID",
                "服务器名称",
                "目标连接端口",
                "主机运行状态",
              ]}
              rows={(servers.data ?? []).map((server) => [
                `User #${server.userId}`,
                server.name,
                `${server.host}:${server.sshPort}`,
                server.status,
              ])}
            />
          </CardContent>
        </Card>

        <Card className="bg-[#0e1017]/70 border-white/[0.04]">
          <CardHeader className="p-5 border-white/[0.04]">
            <div className="font-bold text-slate-200 text-sm tracking-wide">
              多协议代理节点
            </div>
          </CardHeader>
          <CardContent className="p-5">
            <Table
              columns={["归属用户ID", "节点名称", "网络核心协议", "部署状态"]}
              rows={(nodes.data ?? []).map((node) => [
                `User #${node.userId}`,
                node.name,
                node.protocol,
                node.status,
              ])}
            />
          </CardContent>
        </Card>

        <Card className="bg-[#0e1017]/70 border-white/[0.04]">
          <CardHeader className="p-5 border-white/[0.04]">
            <div className="font-bold text-slate-200 text-sm tracking-wide">
              客户端订阅分发及装机任务
            </div>
          </CardHeader>
          <CardContent className="p-5 space-y-6">
            <Table
              columns={["归属用户ID", "订阅别名", "格式类型", "打包节点数"]}
              rows={(subscriptions.data ?? []).map((subscription) => [
                `User #${subscription.userId}`,
                subscription.name,
                subscription.format,
                `${subscription.nodeCount} nodes`,
              ])}
            />
            <div className="border-t border-white/[0.03] pt-5">
              <Table
                columns={["归属用户", "任务序列", "进程类型", "主状态"]}
                rows={(tasks.data ?? []).map((task) => [
                  `User #${task.userId}`,
                  `Task #${task.id}`,
                  task.type,
                  task.status,
                ])}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-[#0e1017]/70 border-white/[0.04] xl:col-span-2">
          <CardHeader className="p-5 border-white/[0.04]">
            <div className="font-bold text-slate-200 text-sm tracking-wide">
              全局系统操作审计流水
            </div>
          </CardHeader>
          <CardContent className="p-5">
            <Table
              columns={["触发主体", "系统审计动作", "受控目标资源"]}
              rows={(operationLogs.data ?? []).map((log) => [
                log.userId ? `User #${log.userId}` : "System 系统守护进程",
                log.action,
                log.resource || "无参数目标",
              ])}
            />
          </CardContent>
        </Card>
      </section>

      {/* Global Task Console Details */}
      <Card className="bg-[#0e1017]/70 border-white/[0.04]">
        <CardHeader className="p-5 border-white/[0.04] flex flex-row items-center justify-between">
          <div className="flex items-center gap-3">
            <ListChecks className="h-4 w-4 text-[#6366f1] animate-pulse" />
            <div className="font-bold text-slate-200 text-sm tracking-wide">
              全局构建任务终端审计
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-6">
          <div className="grid gap-6 xl:grid-cols-[300px_1fr]">
            <div className="space-y-2 max-h-[500px] overflow-y-auto pr-1 scrollbar-thin">
              {taskRows.length === 0 ? (
                <EmptyTable text="暂无部署任务" />
              ) : (
                taskRows.slice(0, 10).map((task) => (
                  <button
                    className={cn(
                      "w-full rounded-xl border p-4 text-left transition-all duration-200 cursor-pointer select-none btn-interactive flex flex-col justify-between",
                      selectedTaskID === task.id
                        ? "border-white/[0.12] bg-white/[0.03] text-white shadow-inner"
                        : "border-white/[0.03] bg-white/[0.01] text-slate-400 hover:bg-white/[0.02] hover:text-slate-200",
                    )}
                    key={task.id}
                    onClick={() => setSelectedTaskID(task.id)}
                    type="button"
                  >
                    <div className="flex items-center justify-between gap-3 w-full">
                      <span
                        className={cn(
                          "text-xs font-bold font-mono tracking-wide",
                          selectedTaskID === task.id
                            ? "text-white"
                            : "text-slate-300",
                        )}
                      >
                        #{task.id} {task.type}
                      </span>
                      <Badge className="border-white/[0.04] bg-white/5 text-slate-400 font-mono text-[9px] px-1.5 py-0">
                        {task.status}
                      </Badge>
                    </div>
                    <div className="mt-2 text-slate-500 text-[9px] font-semibold font-mono leading-normal">
                      用户 #{task.userId} / 服务器 #{task.serverId ?? "-"} /
                      节点 #{task.nodeId ?? "-"}
                    </div>
                  </button>
                ))
              )}
            </div>

            <div className="min-w-0">
              {!selectedTaskID ? (
                <EmptyTable text="请选择一个任务以调取其执行标准流" />
              ) : taskDetail.isLoading ? (
                <EmptyTable text="正在与云主机终端安全通道建立握手..." />
              ) : taskDetail.data ? (
                <div className="space-y-3">
                  {taskDetail.data.logs.length === 0 ? (
                    <EmptyTable text="该任务尚未写入任何日志条目" />
                  ) : (
                    <div className="rounded-xl border border-white/[0.04] bg-[#05060b] font-mono text-[10px] text-slate-300 p-5 shadow-inner min-h-80 max-h-[500px] overflow-y-auto space-y-2.5 terminal-scroll">
                      {taskDetail.data.logs.map((log) => {
                        const isError = log.level === "ERROR";
                        const isWarn = log.level === "WARNING";
                        const levelColor = isError
                          ? "text-rose-400 font-bold"
                          : isWarn
                            ? "text-amber-400 font-bold"
                            : "text-emerald-400 font-semibold";
                        return (
                          <div className="flex items-start gap-3" key={log.id}>
                            <span className="text-slate-600 select-none text-[9px]">
                              [{new Date(log.createdAt).toLocaleString()}]
                            </span>
                            <span className={levelColor}>[{log.level}]</span>
                            <span className="text-slate-300 whitespace-pre-wrap flex-1 leading-relaxed">
                              {log.message}
                            </span>
                          </div>
                        );
                      })}
                      <div className="flex items-center gap-2 mt-4 text-[10px] text-emerald-400/80 font-bold select-none">
                        <span className="animate-pulse">admin@singbox:~$</span>
                        <span className="h-4.5 w-1.5 bg-emerald-400 animate-pulse inline-block" />
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <EmptyTable text="安全日志数据拉取失败" />
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <Card className="bg-[#0e1017]/70 border-white/[0.04] p-5 shadow-lg shadow-black/25">
      <div className="text-slate-500 text-[9px] font-bold uppercase tracking-widest">
        {label}
      </div>
      <div className="mt-2 font-bold text-2xl text-white tracking-tight font-display">
        {value}
      </div>
    </Card>
  );
}

function Table({ columns, rows }: { columns: string[]; rows: string[][] }) {
  if (rows.length === 0) {
    return <EmptyTable text="暂无可以归档的数据" />;
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[420px] border-collapse text-left text-sm">
        <thead>
          <tr className="border-white/[0.04] border-b text-slate-500 text-[9px] font-bold uppercase tracking-wider bg-slate-900/10">
            {columns.map((column) => (
              <th className="py-3 px-4 font-semibold" key={column}>
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-white/[0.04]">
          {rows.map((row) => (
            <tr
              className="hover:bg-white/[0.01] transition-colors duration-200"
              key={row.join("|")}
            >
              {row.map((cell) => (
                <td
                  className="py-3 px-4 text-slate-300 font-mono text-xs"
                  key={cell}
                >
                  <span className="bg-white/5 border border-white/[0.04] text-slate-200 font-semibold px-2 py-0.5 rounded text-[10px] select-all">
                    {cell}
                  </span>
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function EmptyTable({ text }: { text: string }) {
  return (
    <div className="rounded-xl border border-dashed border-white/[0.04] p-10 text-center text-slate-500 text-xs font-semibold">
      {text}
    </div>
  );
}
