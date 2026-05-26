import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Copy,
  Link2,
  Pencil,
  Plus,
  RotateCcw,
  Terminal,
  Trash2,
} from "lucide-react";
import { type FormEvent, useState } from "react";
import { getErrorMessage } from "@/api/errors";
import {
  type ClashTemplatePayload,
  createClashTemplate,
  createSubscription,
  deleteClashTemplate,
  deleteSubscription,
  listClashTemplates,
  listNodes,
  listSubscriptions,
  resetSubscriptionToken,
  type SubscriptionPayload,
  updateClashTemplate,
  updateSubscription,
} from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import type { Subscription } from "@/types/domain";

const subscriptionFormats = [
  { label: "sing-box", value: "sing-box" },
  { label: "Clash / Mihomo", value: "clash-mihomo" },
  { label: "v2rayN", value: "v2rayn" },
  { label: "Shadowrocket", value: "shadowrocket" },
  { label: "通用 Base64", value: "base64" },
];

const clashTemplates = [
  { label: "规则模式：国内直连", value: "rule-cn" },
  { label: "全局代理：除局域网外走代理", value: "global-proxy" },
  { label: "自定义模板", value: "custom" },
];

const emptyForm: SubscriptionPayload = {
  name: "",
  format: "sing-box",
  clashTemplate: "rule-cn",
  clashTemplateId: null,
  enabled: true,
  nodeIds: [],
  remark: "",
};

const emptyTemplateForm: ClashTemplatePayload = {
  name: "",
  content:
    "mixed-port: 7890\nmode: rule\nproxies: []\nproxy-groups:\n  - name: PROXY\n    type: select\n    proxies:\n      - DIRECT\nrules:\n  - MATCH,PROXY",
  remark: "",
};

export function SubscriptionsPage() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<SubscriptionPayload>(emptyForm);
  const [editingSubscription, setEditingSubscription] =
    useState<Subscription | null>(null);
  const [templateForm, setTemplateForm] =
    useState<ClashTemplatePayload>(emptyTemplateForm);
  const [editingTemplateID, setEditingTemplateID] = useState<number | null>(
    null,
  );
  const [message, setMessage] = useState("");

  // Dialog State controls
  const [isSubDialogOpen, setIsSubDialogOpen] = useState(false);
  const [isTemplateDialogOpen, setIsTemplateDialogOpen] = useState(false);

  const subscriptionsQuery = useQuery({
    queryKey: ["subscriptions"],
    queryFn: listSubscriptions,
  });

  const nodesQuery = useQuery({
    queryKey: ["nodes"],
    queryFn: listNodes,
  });

  const clashTemplatesQuery = useQuery({
    queryKey: ["clash-templates"],
    queryFn: listClashTemplates,
  });

  const saveMutation = useMutation({
    mutationFn: (payload: SubscriptionPayload) =>
      editingSubscription
        ? updateSubscription(editingSubscription.id, payload)
        : createSubscription(payload),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
      resetForm();
      setIsSubDialogOpen(false);
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "订阅保存失败"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteSubscription,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
    },
    onError: (error) => {
      alert(getErrorMessage(error, "订阅删除失败"));
    },
  });

  const resetTokenMutation = useMutation({
    mutationFn: resetSubscriptionToken,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
      alert("订阅 Token 已重置，旧链接已失效");
    },
    onError: (error) => {
      alert(getErrorMessage(error, "重置 token 失败"));
    },
  });

  const saveTemplateMutation = useMutation({
    mutationFn: (payload: ClashTemplatePayload) =>
      editingTemplateID
        ? updateClashTemplate(editingTemplateID, payload)
        : createClashTemplate(payload),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["clash-templates"] });
      resetTemplateForm();
      setIsTemplateDialogOpen(false);
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "Clash 模板保存失败"));
    },
  });

  const deleteTemplateMutation = useMutation({
    mutationFn: deleteClashTemplate,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["clash-templates"] });
    },
    onError: (error) => {
      alert(getErrorMessage(error, "Clash 模板删除失败"));
    },
  });

  const subscriptions = subscriptionsQuery.data ?? [];
  const nodes = nodesQuery.data ?? [];
  const customClashTemplates = clashTemplatesQuery.data ?? [];
  const availableNodes = nodes.filter((node) =>
    ["imported", "install_success"].includes(node.status),
  );

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    saveMutation.mutate(form);
  }

  function resetForm() {
    setForm(emptyForm);
    setEditingSubscription(null);
    setMessage("");
  }

  function resetTemplateForm() {
    setTemplateForm(emptyTemplateForm);
    setEditingTemplateID(null);
    setMessage("");
  }

  async function copySubscriptionURL(url: string) {
    const fullURL = url.startsWith("http")
      ? url
      : `${window.location.origin}${url}`;
    try {
      await navigator.clipboard.writeText(fullURL);
      alert("订阅链接已成功复制到剪贴板");
    } catch {
      alert("复制失败，请手动选择并复制订阅链接");
    }
  }

  function startEdit(subscription: Subscription) {
    setEditingSubscription(subscription);
    setForm({
      name: subscription.name,
      format: subscription.format,
      clashTemplate: subscription.clashTemplate ?? "rule-cn",
      clashTemplateId: subscription.clashTemplateId ?? null,
      enabled: subscription.enabled,
      nodeIds: subscription.nodeIds,
      remark: subscription.remark ?? "",
    });
    setMessage("");
    setIsSubDialogOpen(true);
  }

  function toggleNode(nodeID: number) {
    const exists = form.nodeIds.includes(nodeID);
    setForm({
      ...form,
      nodeIds: exists
        ? form.nodeIds.filter((id) => id !== nodeID)
        : [...form.nodeIds, nodeID],
    });
  }

  function handleTemplateSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    saveTemplateMutation.mutate(templateForm);
  }

  function startEditTemplate(template: {
    id: number;
    name: string;
    content: string;
    remark?: string;
  }) {
    setEditingTemplateID(template.id);
    setTemplateForm({
      name: template.name,
      content: template.content,
      remark: template.remark ?? "",
    });
    setMessage("");
    setIsTemplateDialogOpen(true);
  }

  return (
    <div className="space-y-10 py-4 max-w-7xl mx-auto">
      {/* 1. Subscriptions Section Header */}
      <section className="flex flex-col justify-between gap-6 sm:flex-row sm:items-center">
        <div>
          <h1 className="font-bold text-2xl lg:text-3xl text-slate-100 tracking-tight font-display">
            客户端订阅
          </h1>
          <p className="mt-1 text-slate-400 text-xs font-semibold">
            将多个部署好的代理节点进行批量打包，支持下发 sing-box、Clash-Mihomo
            等常用客户端格式订阅。
          </p>
        </div>
        <Button
          onClick={() => {
            resetForm();
            setIsSubDialogOpen(true);
          }}
          className="bg-white text-slate-950 hover:bg-slate-100 px-4 h-9 font-semibold text-xs tracking-wide rounded-lg flex items-center gap-1.5 self-start sm:self-center"
        >
          <Plus className="h-4 w-4" />
          创建订阅规则
        </Button>
      </section>

      {/* Subscriptions Cards Grid */}
      <section>
        {subscriptionsQuery.isLoading ? (
          <div className="text-slate-400 text-xs font-semibold animate-pulse py-6">
            正在读取客户端订阅流数据...
          </div>
        ) : subscriptions.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-white/[0.04] p-16 text-center text-slate-500 text-xs font-semibold">
            暂无订阅分发规则。请点击右上角按钮创建一条规则。
          </div>
        ) : (
          <div className="grid gap-5 md:grid-cols-2">
            {subscriptions.map((subscription) => {
              const enabledClass = subscription.enabled
                ? "border-emerald-500/10 bg-emerald-500/5 text-emerald-400 font-medium"
                : "border-rose-500/10 bg-rose-500/5 text-rose-400 font-medium";

              return (
                <Card
                  className="bg-[#0e1017]/70 border-white/[0.04] p-6 shadow-lg shadow-black/20 hover:border-white/[0.08] hover:-translate-y-0.5 flex flex-col justify-between"
                  key={subscription.id}
                >
                  <div>
                    <div className="flex flex-wrap items-start justify-between gap-2.5">
                      <div className="flex items-center gap-2">
                        <div className="font-bold text-slate-200 text-sm tracking-wide">
                          {subscription.name}
                        </div>
                        <Badge className={enabledClass}>
                          <span
                            className={cn(
                              "mr-1 h-1 w-1 rounded-full shrink-0",
                              subscription.enabled
                                ? "bg-emerald-500 animate-pulse"
                                : "bg-rose-500",
                            )}
                          />
                          {subscription.enabled ? "正常分发" : "已禁用"}
                        </Badge>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <Badge className="border-slate-800 bg-slate-900/60 text-slate-400 font-mono text-[9px] px-1.5 py-0">
                          {subscription.format}
                        </Badge>
                        {subscription.format === "clash-mihomo" && (
                          <Badge className="border-violet-500/10 bg-violet-500/5 text-violet-400 text-[9px] px-1.5 py-0">
                            {clashTemplateLabel(subscription.clashTemplate)}
                          </Badge>
                        )}
                      </div>
                    </div>

                    {subscription.remark && (
                      <p className="mt-3 text-slate-400 text-xs font-semibold">
                        备注: {subscription.remark}
                      </p>
                    )}

                    <div className="mt-4 text-slate-500 text-xs font-semibold">
                      封装集成代理节点数:{" "}
                      <span className="text-slate-300 font-bold font-mono">
                        {subscription.nodeCount}
                      </span>{" "}
                      个
                    </div>
                  </div>

                  <div className="mt-6 pt-5 border-t border-white/[0.03]">
                    {subscription.subscriptionUrl && (
                      <div className="flex items-center gap-2">
                        <div className="flex min-w-0 flex-1 items-center gap-2 text-slate-300 text-xs font-medium bg-[#090b11] border border-white/[0.04] rounded-lg px-3 py-2 shadow-inner">
                          <Link2 className="h-3.5 w-3.5 shrink-0 text-slate-500" />
                          <code className="break-all font-mono text-[10px] select-all truncate text-slate-400">
                            {subscription.subscriptionUrl}
                          </code>
                        </div>
                        <Button
                          onClick={() =>
                            copySubscriptionURL(
                              subscription.subscriptionUrl ?? "",
                            )
                          }
                          variant="secondary"
                          className="h-8 w-8 p-0 rounded-lg shrink-0 flex items-center justify-center"
                          title="复制订阅链接"
                        >
                          <Copy className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    )}

                    <div className="mt-4 flex justify-end gap-2">
                      <Button
                        onClick={() => {
                          if (
                            window.confirm(
                              "确定重置订阅链接的 Token 吗？旧的订阅链接将会立刻失效，您需要重新在客户端中导入新链接。",
                            )
                          ) {
                            resetTokenMutation.mutate(subscription.id);
                          }
                        }}
                        variant="secondary"
                        className="h-8 px-2.5 rounded-lg flex items-center gap-1 text-[10px]"
                      >
                        <RotateCcw className="h-3.5 w-3.5 opacity-70" />
                        <span>重置 Token</span>
                      </Button>
                      <Button
                        onClick={() => startEdit(subscription)}
                        variant="secondary"
                        className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                        title="编辑配置"
                      >
                        <Pencil className="h-3.5 w-3.5 text-slate-400 hover:text-white transition-colors" />
                      </Button>
                      <Button
                        onClick={() => {
                          if (window.confirm("确定删除这个订阅吗？")) {
                            deleteMutation.mutate(subscription.id);
                          }
                        }}
                        variant="danger"
                        className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                        title="删除订阅"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                </Card>
              );
            })}
          </div>
        )}
      </section>

      {/* 2. Clash Custom Templates Section Header */}
      <section className="flex flex-col justify-between gap-6 sm:flex-row sm:items-center border-t border-white/[0.04] pt-10">
        <div>
          <h2 className="font-bold text-2xl text-slate-100 tracking-tight font-display">
            Clash 配置模板
          </h2>
          <p className="mt-1 text-slate-400 text-xs font-semibold">
            支持编辑 YAML 策略块。封装 Clash-Mihomo
            订阅时可动态注入自定义分流、连接端口及策略代理组。
          </p>
        </div>
        <Button
          onClick={() => {
            resetTemplateForm();
            setIsTemplateDialogOpen(true);
          }}
          className="bg-white text-slate-950 hover:bg-slate-100 px-4 h-9 font-semibold text-xs tracking-wide rounded-lg flex items-center gap-1.5 self-start sm:self-center"
        >
          <Plus className="h-4 w-4" />
          新增 YAML 模板
        </Button>
      </section>

      {/* Clash Custom Templates Grid */}
      <section>
        {clashTemplatesQuery.isLoading ? (
          <div className="text-slate-400 text-xs font-semibold animate-pulse py-6">
            正在与本地配置库同步...
          </div>
        ) : customClashTemplates.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-white/[0.04] p-16 text-center text-slate-500 text-xs font-semibold">
            暂无自定义配置模板。请点击右上角按钮添加您的第一个 YAML 模板。
          </div>
        ) : (
          <div className="grid gap-6 md:grid-cols-2">
            {customClashTemplates.map((template) => (
              <Card
                className="bg-[#0e1017]/70 border-white/[0.04] p-6 shadow-lg shadow-black/20 hover:border-white/[0.08] hover:-translate-y-0.5 flex flex-col justify-between"
                key={template.id}
              >
                <div>
                  <div className="flex items-center justify-between gap-3 border-b border-white/[0.03] pb-4 mb-4">
                    <div>
                      <div className="font-bold text-slate-200 text-sm tracking-wide flex items-center gap-2">
                        <Terminal className="h-4 w-4 text-[#6366f1]" />
                        <span>{template.name}</span>
                      </div>
                      {template.remark && (
                        <div className="mt-1 text-slate-500 text-[10px] font-semibold">
                          说明: {template.remark}
                        </div>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <Button
                        onClick={() => startEditTemplate(template)}
                        variant="secondary"
                        className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                        title="编辑模板"
                      >
                        <Pencil className="h-3.5 w-3.5 text-slate-400 hover:text-white transition-colors" />
                      </Button>
                      <Button
                        onClick={() => {
                          if (window.confirm("确定删除这个模板吗？")) {
                            deleteTemplateMutation.mutate(template.id);
                          }
                        }}
                        variant="danger"
                        className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                        title="删除模板"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </div>
                </div>
                <pre className="max-h-48 overflow-auto whitespace-pre rounded-lg border border-white/[0.04] bg-[#090b11] p-4 text-emerald-400 font-mono text-[10px] leading-relaxed scrollbar-thin shadow-inner select-all">
                  {template.content}
                </pre>
              </Card>
            ))}
          </div>
        )}
      </section>

      {/* 3. Subscription Dialog Modal */}
      <Dialog
        isOpen={isSubDialogOpen}
        onClose={() => {
          setIsSubDialogOpen(false);
          resetForm();
        }}
        title={editingSubscription ? "编辑订阅规则" : "创建订阅规则"}
        size="md"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <Field label="订阅名称">
            <Input
              onChange={(event) =>
                setForm({ ...form, name: event.target.value })
              }
              placeholder="香港极速线路订阅"
              required
              value={form.name}
            />
          </Field>
          <Field label="输出格式">
            <select
              className="h-9 w-full rounded-lg border border-slate-800 bg-slate-950 px-3 text-xs text-slate-100 outline-none transition-all duration-300 focus:border-white/20 focus:ring-0 cursor-pointer"
              onChange={(event) =>
                setForm({
                  ...form,
                  format: event.target.value,
                  clashTemplate:
                    event.target.value === "clash-mihomo"
                      ? (form.clashTemplate ?? "rule-cn")
                      : "rule-cn",
                  clashTemplateId:
                    event.target.value === "clash-mihomo"
                      ? form.clashTemplateId
                      : null,
                })
              }
              required
              value={form.format}
            >
              {subscriptionFormats.map((format) => (
                <option key={format.value} value={format.value}>
                  {format.label}
                </option>
              ))}
            </select>
          </Field>

          {form.format === "clash-mihomo" && (
            <Field label="Clash 基础策略规则">
              <select
                className="h-9 w-full rounded-lg border border-slate-800 bg-slate-950 px-3 text-xs text-slate-100 outline-none transition-all duration-300 focus:border-white/20 focus:ring-0 cursor-pointer"
                onChange={(event) =>
                  setForm({
                    ...form,
                    clashTemplate: event.target.value,
                    clashTemplateId:
                      event.target.value === "custom"
                        ? (form.clashTemplateId ??
                          customClashTemplates[0]?.id ??
                          null)
                        : null,
                  })
                }
                value={form.clashTemplate ?? "rule-cn"}
              >
                {clashTemplates.map((template) => (
                  <option key={template.value} value={template.value}>
                    {template.label}
                  </option>
                ))}
              </select>
            </Field>
          )}

          {form.format === "clash-mihomo" &&
            form.clashTemplate === "custom" && (
              <Field label="关联的自定义模板">
                <select
                  className="h-9 w-full rounded-lg border border-slate-800 bg-slate-950 px-3 text-xs text-slate-100 outline-none transition-all duration-300 focus:border-white/20 focus:ring-0 cursor-pointer"
                  onChange={(event) =>
                    setForm({
                      ...form,
                      clashTemplateId: Number(event.target.value),
                    })
                  }
                  required
                  value={form.clashTemplateId ?? ""}
                >
                  <option value="">选择已有自定义模板</option>
                  {customClashTemplates.map((template) => (
                    <option key={template.id} value={template.id}>
                      {template.name}
                    </option>
                  ))}
                </select>
                {customClashTemplates.length === 0 && (
                  <p className="mt-2 text-slate-400 text-[10px]">
                    请先在控制台主界面中创建至少一个自定义 Clash 配置模板。
                  </p>
                )}
              </Field>
            )}

          <Field label="链接启用状态">
            <label className="flex h-9 items-center gap-2.5 rounded-lg border border-slate-800 bg-slate-950/40 px-3 text-xs text-slate-300 cursor-pointer hover:border-slate-700 transition duration-200">
              <input
                checked={form.enabled}
                onChange={(event) =>
                  setForm({ ...form, enabled: event.target.checked })
                }
                type="checkbox"
                className="rounded border-slate-800 bg-slate-950 text-slate-100 focus:ring-0 focus:ring-offset-0 h-4 w-4 cursor-pointer"
              />
              <span>是否允许客户端拉取订阅配置</span>
            </label>
          </Field>

          <Field label="打包封装的节点 (至少勾选一个)">
            {availableNodes.length === 0 ? (
              <div className="rounded-xl border border-dashed border-slate-800 p-4 text-center text-slate-400 text-xs font-semibold">
                当前暂无有效安装成功或导入成功的节点，请先在协议节点页面进行创建。
              </div>
            ) : (
              <div className="max-h-56 space-y-1 overflow-auto rounded-lg border border-slate-800 bg-slate-950 p-2 scrollbar-thin">
                {availableNodes.map((node) => (
                  <label
                    className="flex cursor-pointer items-center justify-between gap-3 rounded-lg px-2.5 py-1.5 text-xs font-semibold hover:bg-white/5 transition duration-200"
                    key={node.id}
                  >
                    <span className="truncate pr-2">
                      <span className="font-bold text-slate-200">
                        {node.name}
                      </span>
                      <span className="ml-2 text-slate-500 text-[9px] font-mono">
                        {node.protocol} · {node.address}:
                        {node.publicPort || node.port}
                      </span>
                    </span>
                    <input
                      checked={form.nodeIds.includes(node.id)}
                      onChange={() => toggleNode(node.id)}
                      type="checkbox"
                      className="rounded border-slate-800 bg-slate-950 text-slate-100 focus:ring-0 focus:ring-offset-0 h-3.5 w-3.5 shrink-0 cursor-pointer"
                    />
                  </label>
                ))}
              </div>
            )}
          </Field>

          <Field label="备注">
            <Input
              onChange={(event) =>
                setForm({ ...form, remark: event.target.value })
              }
              placeholder="备注说明"
              value={form.remark}
            />
          </Field>

          {message && (
            <p className="text-rose-400 text-xs font-semibold bg-red-500/5 border border-red-500/10 px-3.5 py-2.5 rounded-lg">
              {message}
            </p>
          )}

          <div className="flex justify-end gap-2.5 pt-2">
            <Button
              onClick={() => {
                setIsSubDialogOpen(false);
                resetForm();
              }}
              type="button"
              variant="secondary"
              className="h-9 px-4 text-xs"
            >
              取消
            </Button>
            <Button
              disabled={
                saveMutation.isPending ||
                form.nodeIds.length === 0 ||
                (form.format === "clash-mihomo" &&
                  form.clashTemplate === "custom" &&
                  !form.clashTemplateId)
              }
              type="submit"
              className="h-9 px-4 text-xs font-semibold bg-white text-slate-950 hover:bg-slate-100"
            >
              {saveMutation.isPending ? "保存中..." : "保存订阅"}
            </Button>
          </div>
        </form>
      </Dialog>

      {/* 4. Clash Template Dialog Modal */}
      <Dialog
        isOpen={isTemplateDialogOpen}
        onClose={() => {
          setIsTemplateDialogOpen(false);
          resetTemplateForm();
        }}
        title={
          editingTemplateID ? "编辑 Clash 配置模板" : "新增 Clash 配置模板"
        }
        size="md"
      >
        <form className="space-y-4" onSubmit={handleTemplateSubmit}>
          <Field label="模板名称">
            <Input
              onChange={(event) =>
                setTemplateForm({
                  ...templateForm,
                  name: event.target.value,
                })
              }
              placeholder="我的 Clash 分流配置"
              required
              value={templateForm.name}
            />
          </Field>
          <Field label="YAML 结构内容 (请确保格式正确)">
            <textarea
              className="min-h-72 w-full resize-y rounded-lg border border-slate-800 bg-slate-950 px-3.5 py-2.5 font-mono text-[10px] text-slate-200 leading-relaxed outline-none transition-all duration-300 focus:border-white/20 shadow-inner"
              onChange={(event) =>
                setTemplateForm({
                  ...templateForm,
                  content: event.target.value,
                })
              }
              required
              value={templateForm.content}
            />
          </Field>
          <Field label="备注">
            <Input
              onChange={(event) =>
                setTemplateForm({
                  ...templateForm,
                  remark: event.target.value,
                })
              }
              placeholder="说明此模板的适用场景"
              value={templateForm.remark}
            />
          </Field>

          {message && (
            <p className="text-rose-400 text-xs font-semibold bg-red-500/5 border border-red-500/10 px-3.5 py-2.5 rounded-lg">
              {message}
            </p>
          )}

          <div className="flex justify-end gap-2.5 pt-2">
            <Button
              onClick={() => {
                setIsTemplateDialogOpen(false);
                resetTemplateForm();
              }}
              type="button"
              variant="secondary"
              className="h-9 px-4 text-xs"
            >
              取消
            </Button>
            <Button
              disabled={saveTemplateMutation.isPending}
              type="submit"
              className="h-9 px-4 text-xs font-semibold bg-white text-slate-950 hover:bg-slate-100"
            >
              {saveTemplateMutation.isPending ? "保存中..." : "保存模板"}
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  );
}

function clashTemplateLabel(value?: string) {
  return (
    clashTemplates.find((template) => template.value === value)?.label ??
    "规则模式：国内直连"
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="block">
      <span className="mb-1.5 block text-slate-500 text-[9px] font-bold uppercase tracking-widest">
        {label}
      </span>
      {children}
    </div>
  );
}
