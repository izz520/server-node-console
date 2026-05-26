import { useQuery } from "@tanstack/react-query";
import { Clock3, ListChecks } from "lucide-react";
import { useEffect, useState } from "react";
import { getTask, listTasks } from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { Task, TaskStatus } from "@/types/domain";

const taskTypeLabels: Record<string, string> = {
  install: "自动化安装",
  uninstall: "安全卸载",
  ssh_test: "SSH 连接测试",
};

const taskStatusLabels: Record<TaskStatus, string> = {
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
    <div className="grid gap-6 xl:grid-cols-[380px_1fr] py-4 max-w-7xl mx-auto">
      {/* Task Queue Column */}
      <Card className="bg-[#0e1017]/70 border-white/[0.04]">
        <CardHeader className="p-5 border-white/[0.04]">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/[0.04] bg-white/[0.02] text-slate-300">
              <Clock3 className="h-4 w-4 text-[#6366f1]" />
            </div>
            <div>
              <h1 className="font-bold text-slate-200 text-sm tracking-wide font-display">
                部署日志队列
              </h1>
              <p className="text-slate-500 text-[10px] font-semibold mt-0.5">
                查看自动装机部署与网络诊断的任务进程
              </p>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-5">
          {tasksQuery.isLoading ? (
            <div className="text-slate-400 text-xs font-semibold animate-pulse">
              正在同步异步队列...
            </div>
          ) : tasks.length === 0 ? (
            <div className="rounded-xl border border-dashed border-white/[0.04] p-10 text-center text-slate-500 text-xs font-semibold">
              目前还没有任何部署任务日志。
            </div>
          ) : (
            <div className="space-y-2 max-h-[640px] overflow-y-auto pr-1 scrollbar-thin">
              {tasks.map((task) => (
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
                    <div
                      className={cn(
                        "text-xs font-bold font-mono tracking-wide",
                        selectedTaskID === task.id
                          ? "text-white"
                          : "text-slate-300",
                      )}
                    >
                      #{task.id} {taskTypeLabels[task.type] || task.type}
                    </div>
                    <TaskStatusBadge task={task} />
                  </div>
                  <div className="mt-2.5 text-slate-500 text-[9px] font-semibold font-mono">
                    {formatTime(task.createdAt)}
                  </div>
                </button>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Task Logs details */}
      <Card className="bg-[#0e1017]/70 border-white/[0.04]">
        <CardHeader className="p-5 border-white/[0.04] flex flex-row items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/[0.04] bg-white/[0.02] text-slate-300">
              <ListChecks className="h-4 w-4 text-[#6366f1]" />
            </div>
            <div>
              <h2 className="font-bold text-slate-200 text-sm tracking-wide font-display">
                进程终端输出
              </h2>
              <p className="text-slate-500 text-[10px] font-semibold mt-0.5">
                实时接收并解译来自服务器的构建脚本标准流
              </p>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-6">
          {!selectedTaskID ? (
            <div className="rounded-2xl border border-dashed border-white/[0.04] p-16 text-center text-slate-500 text-xs font-semibold">
              请在左侧列表选择一个正在运行或已归档的任务以读取日志详情。
            </div>
          ) : taskDetailQuery.isLoading ? (
            <div className="text-slate-400 text-xs font-semibold animate-pulse">
              正在解密标准输出流...
            </div>
          ) : taskDetailQuery.data ? (
            <div className="space-y-6">
              <TaskSummary task={taskDetailQuery.data.task} />

              <div className="space-y-3">
                <div className="text-slate-400 text-[10px] font-bold uppercase tracking-widest">
                  实时 Standard Output 日志流
                </div>
                {taskDetailQuery.data.logs.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-white/[0.04] p-10 text-center text-slate-500 text-xs font-semibold">
                    当前构建任务尚未输出任何常规日志。
                  </div>
                ) : (
                  <div className="rounded-xl border border-white/[0.04] bg-[#05060b] font-mono text-[10px] text-slate-300 p-5 shadow-inner min-h-96 max-h-[600px] overflow-y-auto space-y-2.5 terminal-scroll">
                    {taskDetailQuery.data.logs.map((log) => {
                      const isError = log.level === "ERROR";
                      const isWarn = log.level === "WARNING";
                      const levelColor = isError
                        ? "text-rose-400 font-bold"
                        : isWarn
                          ? "text-amber-400 font-bold"
                          : "text-emerald-400 font-bold";
                      return (
                        <div className="flex items-start gap-3" key={log.id}>
                          <span className="text-slate-600 select-none text-[9px]">
                            [{formatTime(log.createdAt)}]
                          </span>
                          <span className={levelColor}>[{log.level}]</span>
                          <span className="text-slate-300 whitespace-pre-wrap flex-1 leading-relaxed">
                            {log.message}
                          </span>
                        </div>
                      );
                    })}
                    <div className="flex items-center gap-2 mt-4 text-[10px] text-emerald-400/80 font-bold select-none">
                      <span className="animate-pulse">yasol@singbox:~$</span>
                      <span className="h-4.5 w-1.5 bg-emerald-400 animate-pulse inline-block" />
                    </div>
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="text-rose-400 text-xs font-semibold bg-red-500/5 border border-red-500/10 px-3.5 py-2.5 rounded-lg">
              标准输出文件加载失败。
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function TaskSummary({ task }: { task: Task }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <SummaryItem
        label="动作类型"
        value={taskTypeLabels[task.type] || task.type}
      />
      <SummaryItem
        label="任务状态"
        value={taskStatusLabels[task.status] || task.status}
      />
      <SummaryItem label="物理云主机 ID" value={task.serverId ?? "-"} />
      <SummaryItem label="目标节点 ID" value={task.nodeId ?? "-"} />
      <SummaryItem label="进程派生时间" value={formatTime(task.startedAt)} />
      <SummaryItem label="进程终止时间" value={formatTime(task.endedAt)} />
      {task.error && (
        <div className="sm:col-span-2 lg:col-span-3 rounded-xl border border-red-500/10 bg-red-500/5 p-4 shadow-sm">
          <div className="text-red-400 text-[10px] font-bold uppercase tracking-wider">
            主程序异常退出描述
          </div>
          <div className="mt-1.5 text-slate-300 text-xs font-mono font-semibold select-all break-all leading-normal">
            {task.error}
          </div>
        </div>
      )}
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
    <div className="rounded-xl border border-white/[0.03] bg-white/[0.01] p-4 shadow-inner">
      <div className="text-slate-500 text-[9px] font-bold uppercase tracking-widest">
        {label}
      </div>
      <div className="mt-1 text-slate-300 text-xs font-bold font-mono truncate">
        {value}
      </div>
    </div>
  );
}

function TaskStatusBadge({ task }: { task: Task }) {
  const isSuccess = task.status === "success";
  const isFailed = task.status === "failed";
  const isRunning = task.status === "running";

  const className = isSuccess
    ? "border-emerald-500/10 bg-emerald-500/5 text-emerald-400 font-medium"
    : isFailed
      ? "border-rose-500/10 bg-rose-500/5 text-rose-400 font-medium"
      : isRunning
        ? "border-indigo-500/10 bg-indigo-500/5 text-indigo-400 font-medium"
        : "border-amber-500/10 bg-amber-500/5 text-amber-400 font-medium";

  return (
    <Badge className={className}>
      <span
        className={cn(
          "mr-1.5 h-1.5 w-1.5 rounded-full shrink-0",
          isSuccess
            ? "bg-emerald-500 animate-pulse"
            : isFailed
              ? "bg-rose-500"
              : isRunning
                ? "bg-indigo-500 animate-bounce"
                : "bg-amber-500",
        )}
      />
      {taskStatusLabels[task.status] || task.status}
    </Badge>
  );
}

function formatTime(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Date(value).toLocaleString();
}
