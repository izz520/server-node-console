import { useQuery } from "@tanstack/react-query";
import { Clock3, ListChecks } from "lucide-react";
import { useEffect, useState } from "react";
import { getTask, listTasks } from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import type { Task } from "@/types/domain";

const taskTypeLabels = {
  install: "安装",
  uninstall: "卸载",
  ssh_test: "SSH 测试",
};

const taskStatusLabels = {
  queued: "排队中",
  running: "执行中",
  success: "成功",
  failed: "失败",
};

export function TasksPage() {
  const [selectedTaskID, setSelectedTaskID] = useState<number | null>(null);

  const tasksQuery = useQuery({
    queryKey: ["tasks"],
    queryFn: listTasks,
    refetchInterval: 5000,
  });

  const tasks = tasksQuery.data ?? [];

  useEffect(() => {
    if (!selectedTaskID && tasks.length > 0) {
      setSelectedTaskID(tasks[0].id);
    }
  }, [selectedTaskID, tasks]);

  const taskDetailQuery = useQuery({
    queryKey: ["tasks", selectedTaskID],
    queryFn: () => getTask(selectedTaskID ?? 0),
    enabled: Boolean(selectedTaskID),
    refetchInterval: 5000,
  });

  return (
    <div className="grid gap-6 xl:grid-cols-[420px_1fr]">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-slate-950 text-white">
              <Clock3 className="h-4 w-4" />
            </div>
            <div>
              <h1 className="font-semibold text-slate-950 text-xl">任务日志</h1>
              <p className="text-slate-500 text-sm">
                查看安装、卸载和 SSH 测试任务状态
              </p>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {tasksQuery.isLoading ? (
            <div className="text-slate-500 text-sm">加载中...</div>
          ) : tasks.length === 0 ? (
            <div className="rounded-md border border-dashed border-slate-200 p-8 text-center text-slate-500 text-sm">
              暂无任务。后续安装和卸载协议时会在这里显示。
            </div>
          ) : (
            <div className="space-y-2">
              {tasks.map((task) => (
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
                    <div className="font-medium text-slate-950">
                      #{task.id} {taskTypeLabels[task.type]}
                    </div>
                    <TaskStatusBadge task={task} />
                  </div>
                  <div className="mt-2 text-slate-500 text-xs">
                    {formatTime(task.createdAt)}
                  </div>
                </button>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <ListChecks className="h-5 w-5 text-slate-700" />
            <div className="font-medium text-slate-950">任务详情</div>
          </div>
        </CardHeader>
        <CardContent>
          {!selectedTaskID ? (
            <div className="rounded-md border border-dashed border-slate-200 p-8 text-center text-slate-500 text-sm">
              选择一个任务查看日志详情。
            </div>
          ) : taskDetailQuery.isLoading ? (
            <div className="text-slate-500 text-sm">加载中...</div>
          ) : taskDetailQuery.data ? (
            <div className="space-y-5">
              <TaskSummary task={taskDetailQuery.data.task} />
              <div>
                <div className="mb-3 font-medium text-slate-950">日志</div>
                {taskDetailQuery.data.logs.length === 0 ? (
                  <div className="rounded-md border border-dashed border-slate-200 p-6 text-center text-slate-500 text-sm">
                    当前任务暂无日志。
                  </div>
                ) : (
                  <div className="space-y-2">
                    {taskDetailQuery.data.logs.map((log) => (
                      <div
                        className="rounded-md border border-slate-200 bg-slate-50 p-3"
                        key={log.id}
                      >
                        <div className="mb-1 flex items-center justify-between gap-3 text-xs">
                          <Badge>{log.level}</Badge>
                          <span className="text-slate-500">
                            {formatTime(log.createdAt)}
                          </span>
                        </div>
                        <pre className="whitespace-pre-wrap text-slate-700 text-sm">
                          {log.message}
                        </pre>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="text-red-600 text-sm">任务详情加载失败</div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function TaskSummary({ task }: { task: Task }) {
  return (
    <div className="grid gap-3 md:grid-cols-2">
      <SummaryItem label="任务类型" value={taskTypeLabels[task.type]} />
      <SummaryItem label="任务状态" value={taskStatusLabels[task.status]} />
      <SummaryItem label="服务器 ID" value={task.serverId ?? "-"} />
      <SummaryItem label="节点 ID" value={task.nodeId ?? "-"} />
      <SummaryItem label="开始时间" value={formatTime(task.startedAt)} />
      <SummaryItem label="结束时间" value={formatTime(task.endedAt)} />
      {task.error && <SummaryItem label="错误信息" value={task.error} />}
    </div>
  );
}

function SummaryItem({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  return (
    <div className="rounded-md border border-slate-200 p-3">
      <div className="text-slate-500 text-xs">{label}</div>
      <div className="mt-1 text-slate-900 text-sm">{value}</div>
    </div>
  );
}

function TaskStatusBadge({ task }: { task: Task }) {
  const className =
    task.status === "success"
      ? "border-emerald-200 bg-emerald-50 text-emerald-700"
      : task.status === "failed"
        ? "border-red-200 bg-red-50 text-red-700"
        : "border-slate-200 bg-slate-50 text-slate-700";

  return <Badge className={className}>{taskStatusLabels[task.status]}</Badge>;
}

function formatTime(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Date(value).toLocaleString();
}
