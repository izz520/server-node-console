import { useQueries } from "@tanstack/react-query";
import {
  Activity,
  CheckCircle2,
  Clock3,
  Plus,
  Server,
  Share2,
  Upload,
} from "lucide-react";
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
    <div className="space-y-6">
      <section className="flex flex-col justify-between gap-4 md:flex-row md:items-center">
        <div>
          <h1 className="font-semibold text-2xl text-slate-950">
            节点管理工作台
          </h1>
          <p className="mt-2 max-w-2xl text-slate-600 text-sm">
            集中查看服务器、协议节点、订阅和异步任务状态。
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button onClick={() => navigate("/servers")}>
            <Plus className="h-4 w-4" />
            添加服务器
          </Button>
          <Button onClick={() => navigate("/nodes")} variant="secondary">
            <Upload className="h-4 w-4" />
            导入节点
          </Button>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard
          icon={Server}
          label="服务器"
          loading={servers.isLoading}
          value={servers.data?.length ?? 0}
        />
        <StatCard
          icon={Activity}
          label="可用节点"
          loading={nodes.isLoading}
          value={usableNodes}
        />
        <StatCard
          icon={Share2}
          label="订阅"
          loading={subscriptions.isLoading}
          value={subscriptions.data?.length ?? 0}
        />
        <StatCard
          icon={Clock3}
          label="运行任务"
          loading={tasks.isLoading}
          value={runningTasks}
        />
      </section>

      <section className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">最近任务</div>
          </CardHeader>
          <CardContent>
            <RecentTasks
              loading={tasks.isLoading}
              rows={(tasks.data ?? []).slice(0, 6).map((task) => ({
                id: task.id,
                name: `#${task.id}`,
                type: task.type,
                status: taskStatusLabels[task.status],
              }))}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">最近节点</div>
          </CardHeader>
          <CardContent>
            <RecentNodes
              loading={nodes.isLoading}
              rows={(nodes.data ?? []).slice(0, 6).map((node) => ({
                id: node.id,
                name: node.name,
                protocol: node.protocol,
                status: nodeStatusLabels[node.status],
              }))}
            />
          </CardContent>
        </Card>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        <QuickAction
          description="添加 SSH 凭据并完成连通性测试"
          icon={Server}
          label="服务器管理"
          onClick={() => navigate("/servers")}
        />
        <QuickAction
          description="安装协议或导入外部分享链接"
          icon={Activity}
          label="协议节点"
          onClick={() => navigate("/nodes")}
        />
        <QuickAction
          description="组合多个节点生成客户端订阅"
          icon={Share2}
          label="订阅管理"
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
}: {
  icon: typeof Server;
  label: string;
  loading: boolean;
  value: number;
}) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between">
        <div>
          <div className="text-slate-500 text-sm">{label}</div>
          <div className="mt-2 font-semibold text-2xl text-slate-950">
            {loading ? "-" : value}
          </div>
        </div>
        <div className="flex h-10 w-10 items-center justify-center rounded-md bg-slate-100 text-slate-700">
          <Icon className="h-5 w-5" />
        </div>
      </CardContent>
    </Card>
  );
}

function RecentTasks({
  loading,
  rows,
}: {
  loading: boolean;
  rows: Array<{ id: number; name: string; type: string; status: string }>;
}) {
  if (loading) {
    return <EmptyState text="正在加载任务" />;
  }
  if (rows.length === 0) {
    return <EmptyState text="暂无任务" />;
  }
  return (
    <div className="space-y-3">
      {rows.map((row) => (
        <div
          className="flex items-center justify-between rounded-md border border-slate-100 p-3"
          key={row.id}
        >
          <div>
            <div className="font-medium text-slate-800 text-sm">{row.name}</div>
            <div className="mt-1 text-slate-500 text-xs">{row.type}</div>
          </div>
          <Badge>{row.status}</Badge>
        </div>
      ))}
    </div>
  );
}

function RecentNodes({
  loading,
  rows,
}: {
  loading: boolean;
  rows: Array<{ id: number; name: string; protocol: string; status: string }>;
}) {
  if (loading) {
    return <EmptyState text="正在加载节点" />;
  }
  if (rows.length === 0) {
    return <EmptyState text="暂无节点" />;
  }
  return (
    <div className="space-y-3">
      {rows.map((row) => (
        <div
          className="flex items-center justify-between rounded-md border border-slate-100 p-3"
          key={row.id}
        >
          <div>
            <div className="font-medium text-slate-800 text-sm">{row.name}</div>
            <div className="mt-1 text-slate-500 text-xs">{row.protocol}</div>
          </div>
          <Badge>{row.status}</Badge>
        </div>
      ))}
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
      className="flex items-start gap-3 rounded-md border border-slate-200 bg-white p-4 text-left transition hover:border-slate-300 hover:bg-slate-50"
      onClick={onClick}
      type="button"
    >
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-950 text-white">
        <Icon className="h-4 w-4" />
      </div>
      <div>
        <div className="flex items-center gap-2 font-medium text-slate-950 text-sm">
          <CheckCircle2 className="h-4 w-4 text-emerald-600" />
          {label}
        </div>
        <div className="mt-1 text-slate-500 text-xs">{description}</div>
      </div>
    </button>
  );
}

function EmptyState({ text }: { text: string }) {
  return (
    <div className="rounded-md border border-dashed border-slate-200 p-6 text-center text-slate-500 text-sm">
      {text}
    </div>
  );
}
