import { Activity, CheckCircle2, Clock3, Server, Share2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  SUBSCRIPTION_FORMATS,
  SUPPORTED_PROTOCOLS,
} from "@/constants/protocols";

const stats = [
  { label: "服务器", value: "0", icon: Server },
  { label: "协议节点", value: "0", icon: Activity },
  { label: "订阅", value: "0", icon: Share2 },
  { label: "运行任务", value: "0", icon: Clock3 },
];

const workflows = [
  "添加服务器并完成 SSH 连通性测试",
  "选择协议并自动生成安装参数",
  "后端异步执行安装任务并记录日志",
  "组合多个节点生成客户端订阅",
];

export function DashboardPage() {
  return (
    <div className="space-y-6">
      <section className="flex flex-col justify-between gap-4 rounded-lg border border-slate-200 bg-white p-6 md:flex-row md:items-center">
        <div>
          <h1 className="font-semibold text-2xl text-slate-950">
            节点管理工作台
          </h1>
          <p className="mt-2 max-w-2xl text-slate-600 text-sm">
            当前是前端脚手架页面，后续会接入服务器管理、协议安装、任务日志和订阅生成接口。
          </p>
        </div>
        <div className="flex gap-2">
          <Button>添加服务器</Button>
          <Button variant="secondary">导入外部节点</Button>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {stats.map((item) => (
          <Card key={item.label}>
            <CardContent className="flex items-center justify-between">
              <div>
                <div className="text-slate-500 text-sm">{item.label}</div>
                <div className="mt-2 font-semibold text-2xl text-slate-950">
                  {item.value}
                </div>
              </div>
              <div className="flex h-10 w-10 items-center justify-center rounded-md bg-slate-100 text-slate-700">
                <item.icon className="h-5 w-5" />
              </div>
            </CardContent>
          </Card>
        ))}
      </section>

      <section className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">一期核心流程</div>
          </CardHeader>
          <CardContent>
            <div className="grid gap-3 md:grid-cols-2">
              {workflows.map((item) => (
                <div
                  className="flex items-start gap-3 rounded-md border border-slate-100 p-3"
                  key={item}
                >
                  <CheckCircle2 className="mt-0.5 h-4 w-4 text-emerald-600" />
                  <span className="text-slate-700 text-sm">{item}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="font-medium text-slate-950">支持范围</div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <div className="mb-2 text-slate-500 text-sm">协议</div>
              <div className="flex flex-wrap gap-2">
                {SUPPORTED_PROTOCOLS.map((protocol) => (
                  <Badge key={protocol}>{protocol}</Badge>
                ))}
              </div>
            </div>
            <div>
              <div className="mb-2 text-slate-500 text-sm">订阅格式</div>
              <div className="flex flex-wrap gap-2">
                {SUBSCRIPTION_FORMATS.map((format) => (
                  <Badge key={format}>{format}</Badge>
                ))}
              </div>
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
