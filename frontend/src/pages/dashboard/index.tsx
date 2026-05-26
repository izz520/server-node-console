import { useQueries } from "@tanstack/react-query";
import { Activity, Clock3, Plus, Server, Share2, Upload } from "lucide-react";
import { useNavigate } from "react-router-dom";
import {
  listNodes,
  listServers,
  listSubscriptions,
  listTasks,
} from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { NodeStatus, TaskStatus } from "@/types/domain";

const nodeStatusLabels: Record<NodeStatus, string> = {
  installing: "安装中",
  install_success: "安装成功",
  install_failed: "安装失败",
  uninstalling: "卸载中",
  uninstalled: "已卸载",
  imported: "外部导入",
};

const taskStatusLabels: Record<TaskStatus, string> = {
  queued: "排队中",
  running: "执行中",
  success: "成功",
  failed: "失败",
};

export function DashboardPage() {
  const navigate = useNavigate();
  const [servers, nodes, subscriptions, tasks] = useQueries({
    queries: [
      { queryKey: ["servers"], queryFn: listServers },
      { queryKey: ["nodes"], queryFn: listNodes, refetchInterval: 5000 },
      { queryKey: ["subscriptions"], queryFn: listSubscriptions },
      { queryKey: ["tasks"], queryFn: listTasks, refetchInterval: 5000 },
    ],
  });

  const runningTasks =
    tasks.data?.filter((task) => ["queued", "running"].includes(task.status))
      .length ?? 0;
  const usableNodes =
    nodes.data?.filter((node) =>
      ["imported", "install_success"].includes(node.status),
    ).length ?? 0;

  return (
    <div className="space-y-10 py-4 max-w-7xl mx-auto">
      {/* Dashboard Top Header Section */}
      <section className="flex flex-col justify-between gap-6 sm:flex-row sm:items-center">
        <div>
          <h1 className="font-bold text-2xl lg:text-3xl text-slate-100 tracking-tight font-display">
            管理工作台
          </h1>
          <p className="mt-1 text-slate-400 text-xs font-semibold">
            实时查看及调控服务器、代理节点、客户端订阅以及系统装机异步任务。
          </p>
        </div>
        <div className="flex gap-2.5">
          <Button
            onClick={() => navigate("/servers")}
            className="bg-white text-slate-950 hover:bg-slate-100 px-4 h-9 font-semibold text-xs tracking-wide rounded-lg flex items-center gap-1.5"
          >
            <Plus className="h-4 w-4" />
            添加物理服务器
          </Button>
          <Button
            onClick={() => navigate("/nodes")}
            variant="secondary"
            className="h-9 px-4 text-xs font-semibold"
          >
            <Upload className="h-4 w-4 text-slate-400" />
            一键导入节点
          </Button>
        </div>
      </section>

      {/* Numerical Metrics Stats Row */}
      <section className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          icon={Server}
          label="物理服务器"
          loading={servers.isLoading}
          value={servers.data?.length ?? 0}
          description="正常对接的物理云主机"
        />
        <StatCard
          icon={Activity}
          label="可用代理节点"
          loading={nodes.isLoading}
          value={usableNodes}
          description="正常连通的协议节点数"
        />
        <StatCard
          icon={Share2}
          label="下发订阅集"
          loading={subscriptions.isLoading}
          value={subscriptions.data?.length ?? 0}
          description="已配发的客户端拉取链接"
        />
        <StatCard
          icon={Clock3}
          label="后台队列任务"
          loading={tasks.isLoading}
          value={runningTasks}
          description="正在执行中的异步线程"
        />
      </section>

      {/* Grid of Task Logs & Recent Nodes */}
      <section className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between border-white/[0.04] p-5">
            <div>
              <div className="font-bold text-slate-100 text-sm tracking-wide">
                最新部署记录
              </div>
              <div className="text-[10px] text-slate-500 font-semibold uppercase tracking-wider mt-0.5">
                异步装机任务执行流
              </div>
            </div>
            <Button
              onClick={() => navigate("/tasks")}
              variant="ghost"
              className="h-8 px-2.5 text-[11px] font-semibold text-slate-400 hover:text-slate-200"
            >
              查看全部
            </Button>
          </CardHeader>
          <CardContent className="p-5">
            <RecentTasks
              loading={tasks.isLoading}
              rows={(tasks.data ?? []).slice(0, 5).map((task) => ({
                id: task.id,
                name: `Task #${task.id}`,
                type: task.type,
                status: task.status,
              }))}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between border-white/[0.04] p-5">
            <div>
              <div className="font-bold text-slate-100 text-sm tracking-wide">
                最新部署节点
              </div>
              <div className="text-[10px] text-slate-500 font-semibold uppercase tracking-wider mt-0.5">
                节点协议状态清单
              </div>
            </div>
            <Button
              onClick={() => navigate("/nodes")}
              variant="ghost"
              className="h-8 px-2.5 text-[11px] font-semibold text-slate-400 hover:text-slate-200"
            >
              查看全部
            </Button>
          </CardHeader>
          <CardContent className="p-5">
            <RecentNodes
              loading={nodes.isLoading}
              rows={(nodes.data ?? []).slice(0, 5).map((node) => ({
                id: node.id,
                name: node.name,
                protocol: node.protocol,
                status: node.status,
              }))}
            />
          </CardContent>
        </Card>
      </section>

      {/* Grid of Interactive Quick Action Buttons */}
      <section className="grid gap-5 md:grid-cols-3">
        <QuickAction
          description="配置底层 SSH 凭据，远程执行节点安装前置测试环境"
          icon={Server}
          label="物理服务器控制台"
          onClick={() => navigate("/servers")}
        />
        <QuickAction
          description="一键自动化编译安装 AnyTLS 节点，或手动贴入分享链接导入"
          icon={Activity}
          label="协议节点部署中心"
          onClick={() => navigate("/nodes")}
        />
        <QuickAction
          description="自定义 Mihomo YAML 分流策略组，配发客户端拉取订阅"
          icon={Share2}
          label="节点订阅配发控制"
          onClick={() => navigate("/subscriptions")}
        />
      </section>
    </div>
  );
}

function StatCard({
  icon: Icon,
  label,
  loading,
  value,
  description,
}: {
  icon: typeof Server;
  label: string;
  loading: boolean;
  value: number;
  description: string;
}) {
  return (
    <Card className="hover:-translate-y-0.5 bg-[#0e1017]/70 border-white/[0.04] p-6 shadow-lg shadow-black/20 flex flex-col justify-between min-h-36">
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="text-slate-500 text-[10px] font-bold uppercase tracking-wider">
            {label}
          </div>
          <div className="mt-2 font-bold text-3xl text-white tracking-tight font-display">
            {loading ? (
              <div className="h-9 w-12 animate-pulse rounded-lg bg-slate-800/60" />
            ) : (
              value
            )}
          </div>
        </div>
        <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-white/[0.04] bg-white/[0.02] text-slate-400">
          <Icon className="h-4 w-4" />
        </div>
      </div>
      <div className="text-[10px] text-slate-500 font-semibold mt-4">
        {description}
      </div>
    </Card>
  );
}

function RecentTasks({
  loading,
  rows,
}: {
  loading: boolean;
  rows: Array<{ id: number; name: string; type: string; status: TaskStatus }>;
}) {
  if (loading) {
    return <EmptyState text="正在读取异步任务队列..." />;
  }
  if (rows.length === 0) {
    return <EmptyState text="暂无执行任务日志" />;
  }
  return (
    <div className="space-y-2">
      {rows.map((row) => {
        const isSuccess = row.status === "success";
        const isFailed = row.status === "failed";
        const isRunning = row.status === "running";

        const badgeClass = isSuccess
          ? "border-emerald-500/10 bg-emerald-500/5 text-emerald-400 font-medium"
          : isFailed
            ? "border-rose-500/10 bg-rose-500/5 text-rose-400 font-medium"
            : isRunning
              ? "border-indigo-500/10 bg-indigo-500/5 text-indigo-400 font-medium"
              : "border-amber-500/10 bg-amber-500/5 text-amber-400 font-medium";

        return (
          <div
            className="flex items-center justify-between rounded-xl border border-white/[0.03] bg-white/[0.01] px-4 py-3.5 hover:bg-white/[0.02] transition-colors duration-200"
            key={row.id}
          >
            <div>
              <div className="font-semibold text-slate-200 text-xs font-mono">
                {row.name}
              </div>
              <div className="mt-0.5 text-slate-500 text-[10px] font-bold uppercase tracking-wide">
                {row.type}
              </div>
            </div>
            <Badge className={badgeClass}>
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
              {taskStatusLabels[row.status]}
            </Badge>
          </div>
        );
      })}
    </div>
  );
}

function RecentNodes({
  loading,
  rows,
}: {
  loading: boolean;
  rows: Array<{
    id: number;
    name: string;
    protocol: string;
    status: NodeStatus;
  }>;
}) {
  if (loading) {
    return <EmptyState text="正在同步边缘节点..." />;
  }
  if (rows.length === 0) {
    return <EmptyState text="暂无可读取协议节点" />;
  }
  return (
    <div className="space-y-2">
      {rows.map((row) => {
        const isSuccess = ["imported", "install_success"].includes(row.status);
        const isFailed = row.status === "install_failed";
        const isProgress = ["installing", "uninstalling"].includes(row.status);

        const badgeClass = isSuccess
          ? "border-emerald-500/10 bg-emerald-500/5 text-emerald-400 font-medium"
          : isFailed
            ? "border-rose-500/10 bg-rose-500/5 text-rose-400 font-medium"
            : isProgress
              ? "border-indigo-500/10 bg-indigo-500/5 text-indigo-400 font-medium animate-pulse"
              : "border-slate-800 bg-slate-900/60 text-slate-400 font-medium";

        return (
          <div
            className="flex items-center justify-between rounded-xl border border-white/[0.03] bg-white/[0.01] px-4 py-3.5 hover:bg-white/[0.02] transition-colors duration-200"
            key={row.id}
          >
            <div>
              <div className="font-semibold text-slate-200 text-xs">
                {row.name}
              </div>
              <div className="mt-0.5 text-slate-500 text-[10px] font-bold uppercase tracking-wide font-mono">
                {row.protocol}
              </div>
            </div>
            <Badge className={badgeClass}>
              <span
                className={cn(
                  "mr-1.5 h-1.5 w-1.5 rounded-full shrink-0",
                  isSuccess
                    ? "bg-emerald-500 animate-pulse"
                    : isFailed
                      ? "bg-rose-500"
                      : isProgress
                        ? "bg-indigo-500 animate-pulse"
                        : "bg-slate-500",
                )}
              />
              {nodeStatusLabels[row.status]}
            </Badge>
          </div>
        );
      })}
    </div>
  );
}

function QuickAction({
  description,
  icon: Icon,
  label,
  onClick,
}: {
  description: string;
  icon: typeof Server;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      className="flex items-start gap-4 rounded-xl border border-white/[0.04] bg-[#0e1017]/50 p-5 text-left transition-all duration-200 hover:-translate-y-0.5 hover:bg-[#0e1017]/90 hover:border-white/[0.08] hover:shadow-xl shadow-black/40 group cursor-pointer select-none"
      onClick={onClick}
      type="button"
    >
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-white/[0.06] bg-slate-900 text-slate-400 group-hover:bg-[#6366f1] group-hover:text-white group-hover:border-transparent transition-all duration-200">
        <Icon className="h-4 w-4" />
      </div>
      <div>
        <div className="flex items-center gap-1.5 font-bold text-slate-200 text-xs tracking-wide">
          <span>{label}</span>
        </div>
        <div className="mt-1.5 text-slate-500 text-[10px] font-semibold leading-relaxed">
          {description}
        </div>
      </div>
    </button>
  );
}

function EmptyState({ text }: { text: string }) {
  return (
    <div className="rounded-xl border border-dashed border-white/[0.04] p-10 text-center text-slate-500 text-xs font-semibold">
      {text}
    </div>
  );
}
