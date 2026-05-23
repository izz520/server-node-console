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
    <div className="space-y-6">
      <section className="flex items-center gap-4">
        <div className="flex h-11 w-11 items-center justify-center rounded-md bg-slate-950 text-white">
          <ShieldCheck className="h-5 w-5" />
        </div>
        <div>
          <h1 className="font-semibold text-2xl text-slate-950">管理员后台</h1>
          <p className="mt-1 text-slate-600 text-sm">
            第一版仅提供只读查看，不允许代用户编辑或删除数据。
          </p>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-6">
        <StatCard label="用户" value={users.data?.length ?? 0} />
        <StatCard label="服务器" value={servers.data?.length ?? 0} />
        <StatCard label="节点" value={nodes.data?.length ?? 0} />
        <StatCard label="订阅" value={subscriptions.data?.length ?? 0} />
        <StatCard label="任务" value={tasks.data?.length ?? 0} />
        <StatCard label="操作日志" value={operationLogs.data?.length ?? 0} />
      </section>

      <section className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">用户</div>
          </CardHeader>
          <CardContent>
            <Table
              columns={["用户名", "邮箱", "角色"]}
              rows={(users.data ?? []).map((user) => [
                user.username,
                user.email,
                user.role,
              ])}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">服务器</div>
          </CardHeader>
          <CardContent>
            <Table
              columns={["用户", "名称", "地址", "状态"]}
              rows={(servers.data ?? []).map((server) => [
                `#${server.userId}`,
                server.name,
                `${server.host}:${server.sshPort}`,
                server.status,
              ])}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">节点</div>
          </CardHeader>
          <CardContent>
            <Table
              columns={["用户", "名称", "协议", "状态"]}
              rows={(nodes.data ?? []).map((node) => [
                `#${node.userId}`,
                node.name,
                node.protocol,
                node.status,
              ])}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">订阅与任务</div>
          </CardHeader>
          <CardContent className="space-y-6">
            <Table
              columns={["用户", "订阅", "格式", "节点数"]}
              rows={(subscriptions.data ?? []).map((subscription) => [
                `#${subscription.userId}`,
                subscription.name,
                subscription.format,
                String(subscription.nodeCount),
              ])}
            />
            <Table
              columns={["用户", "任务", "类型", "状态"]}
              rows={(tasks.data ?? []).map((task) => [
                `#${task.userId}`,
                `#${task.id}`,
                task.type,
                task.status,
              ])}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">操作日志</div>
          </CardHeader>
          <CardContent>
            <Table
              columns={["用户", "动作", "资源"]}
              rows={(operationLogs.data ?? []).map((log) => [
                log.userId ? `#${log.userId}` : "系统",
                log.action,
                log.resource || "-",
              ])}
            />
          </CardContent>
        </Card>
      </section>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <ListChecks className="h-5 w-5 text-slate-700" />
            <div className="font-medium text-slate-950">任务日志详情</div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 xl:grid-cols-[320px_1fr]">
            <div className="space-y-2">
              {taskRows.length === 0 ? (
                <EmptyTable text="暂无任务" />
              ) : (
                taskRows.slice(0, 10).map((task) => (
                  <button
                    className={[
                      "w-full rounded-md border p-3 text-left transition",
                      selectedTaskID === task.id
                        ? "border-slate-950 bg-slate-50"
                        : "border-slate-200 bg-white hover:bg-slate-50",
                    ].join(" ")}
                    key={task.id}
                    onClick={() => setSelectedTaskID(task.id)}
                    type="button"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <span className="font-medium text-slate-950 text-sm">
                        #{task.id} {task.type}
                      </span>
                      <Badge>{task.status}</Badge>
                    </div>
                    <div className="mt-1 text-slate-500 text-xs">
                      用户 #{task.userId} / 服务器 #{task.serverId ?? "-"} /
                      节点 #{task.nodeId ?? "-"}
                    </div>
                  </button>
                ))
              )}
            </div>
            <div>
              {!selectedTaskID ? (
                <EmptyTable text="选择任务查看日志" />
              ) : taskDetail.isLoading ? (
                <EmptyTable text="正在加载日志" />
              ) : taskDetail.data ? (
                <div className="space-y-3">
                  {taskDetail.data.logs.length === 0 ? (
                    <EmptyTable text="当前任务暂无日志" />
                  ) : (
                    taskDetail.data.logs.map((log) => (
                      <div
                        className="rounded-md border border-slate-200 bg-slate-50 p-3"
                        key={log.id}
                      >
                        <div className="mb-2 flex items-center justify-between gap-3 text-xs">
                          <Badge>{log.level}</Badge>
                          <span className="text-slate-500">
                            {new Date(log.createdAt).toLocaleString()}
                          </span>
                        </div>
                        <pre className="whitespace-pre-wrap text-slate-700 text-sm">
                          {log.message}
                        </pre>
                      </div>
                    ))
                  )}
                </div>
              ) : (
                <EmptyTable text="任务日志加载失败" />
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
    <Card>
      <CardContent>
        <div className="text-slate-500 text-sm">{label}</div>
        <div className="mt-2 font-semibold text-2xl text-slate-950">
          {value}
        </div>
      </CardContent>
    </Card>
  );
}

function Table({ columns, rows }: { columns: string[]; rows: string[][] }) {
  if (rows.length === 0) {
    return <EmptyTable text="暂无数据" />;
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[420px] border-collapse text-left text-sm">
        <thead>
          <tr className="border-slate-100 border-b text-slate-500">
            {columns.map((column) => (
              <th className="py-2 pr-3 font-medium" key={column}>
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr className="border-slate-100 border-b" key={row.join("|")}>
              {row.map((cell) => (
                <td className="py-2 pr-3 text-slate-700" key={cell}>
                  <Badge>{cell}</Badge>
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
    <div className="rounded-md border border-dashed border-slate-200 p-6 text-center text-slate-500 text-sm">
      {text}
    </div>
  );
}
