import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, Link2, Pencil, RotateCcw, Share2, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import { getErrorMessage } from "@/api/errors";
import {
  createClashTemplate,
  createSubscription,
  deleteClashTemplate,
  deleteSubscription,
  listClashTemplates,
  listNodes,
  listSubscriptions,
  resetSubscriptionToken,
  type ClashTemplatePayload,
  type SubscriptionPayload,
  updateClashTemplate,
  updateSubscription,
} from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
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
  content: "mixed-port: 7890\nmode: rule\nproxies: []\nproxy-groups:\n  - name: PROXY\n    type: select\n    proxies:\n      - DIRECT\nrules:\n  - MATCH,PROXY",
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
      setMessage("订阅已保存");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "订阅保存失败"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteSubscription,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
      setMessage("订阅已删除");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "订阅删除失败"));
    },
  });

  const resetTokenMutation = useMutation({
    mutationFn: resetSubscriptionToken,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["subscriptions"] });
      setMessage("订阅 token 已重置");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "重置 token 失败"));
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
      setMessage("Clash 模板已保存");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "Clash 模板保存失败"));
    },
  });

  const deleteTemplateMutation = useMutation({
    mutationFn: deleteClashTemplate,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["clash-templates"] });
      setMessage("Clash 模板已删除");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "Clash 模板删除失败"));
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
  }

  function resetTemplateForm() {
    setTemplateForm(emptyTemplateForm);
    setEditingTemplateID(null);
  }

  async function copySubscriptionURL(url: string) {
    const fullURL = url.startsWith("http")
      ? url
      : `${window.location.origin}${url}`;
    try {
      await navigator.clipboard.writeText(fullURL);
      setMessage("订阅链接已复制");
    } catch {
      setMessage("复制失败，请手动复制订阅链接");
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
  }

  return (
    <div className="grid gap-6 xl:grid-cols-[420px_1fr]">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-slate-950 text-white">
              <Share2 className="h-4 w-4" />
            </div>
            <div>
              <h1 className="font-semibold text-slate-950 text-xl">
                {editingSubscription ? "编辑订阅" : "创建订阅"}
              </h1>
              <p className="text-slate-500 text-sm">
                选择多个节点生成客户端订阅链接
              </p>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={handleSubmit}>
            <Field label="订阅名称">
              <Input
                onChange={(event) =>
                  setForm({ ...form, name: event.target.value })
                }
                placeholder="香港节点订阅"
                required
                value={form.name}
              />
            </Field>
            <Field label="订阅格式">
              <select
                className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
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
              <Field label="Clash 模板">
                <select
                  className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
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
                <Field label="自定义模板">
                  <select
                    className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
                    onChange={(event) =>
                      setForm({
                        ...form,
                        clashTemplateId: Number(event.target.value),
                      })
                    }
                    required
                    value={form.clashTemplateId ?? ""}
                  >
                    <option value="">选择自定义模板</option>
                    {customClashTemplates.map((template) => (
                      <option key={template.id} value={template.id}>
                        {template.name}
                      </option>
                    ))}
                  </select>
                  {customClashTemplates.length === 0 && (
                    <p className="mt-2 text-slate-500 text-xs">
                      先在下方保存一个自定义 Clash 模板。
                    </p>
                  )}
                </Field>
              )}
            <Field label="启用状态">
              <label className="flex h-10 items-center gap-2 rounded-md border border-slate-200 px-3 text-sm">
                <input
                  checked={form.enabled}
                  onChange={(event) =>
                    setForm({ ...form, enabled: event.target.checked })
                  }
                  type="checkbox"
                />
                启用订阅链接
              </label>
            </Field>
            <Field label="包含节点">
              {availableNodes.length === 0 ? (
                <div className="rounded-md border border-dashed border-slate-200 p-4 text-slate-500 text-sm">
                  还没有可用节点，请先添加外部节点。
                </div>
              ) : (
                <div className="max-h-56 space-y-2 overflow-auto rounded-md border border-slate-200 p-2">
                  {availableNodes.map((node) => (
                    <label
                      className="flex cursor-pointer items-center justify-between gap-3 rounded-md px-2 py-2 text-sm hover:bg-slate-50"
                      key={node.id}
                    >
                      <span>
                        <span className="font-medium text-slate-950">
                          {node.name}
                        </span>
                        <span className="ml-2 text-slate-500">
                          {node.protocol} · {node.address}:
                          {node.publicPort || node.port}
                        </span>
                      </span>
                      <input
                        checked={form.nodeIds.includes(node.id)}
                        onChange={() => toggleNode(node.id)}
                        type="checkbox"
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
                placeholder="可选"
                value={form.remark}
              />
            </Field>
            {message && <p className="text-slate-600 text-sm">{message}</p>}
            <div className="flex gap-2">
              <Button
                disabled={
                  saveMutation.isPending ||
                  form.nodeIds.length === 0 ||
                  (form.format === "clash-mihomo" &&
                    form.clashTemplate === "custom" &&
                    !form.clashTemplateId)
                }
                type="submit"
              >
                {saveMutation.isPending
                  ? "保存中..."
                  : editingSubscription
                    ? "保存修改"
                    : "创建订阅"}
              </Button>
              {editingSubscription && (
                <Button onClick={resetForm} type="button" variant="secondary">
                  取消
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="xl:order-last xl:col-span-2">
        <CardHeader>
          <div className="font-medium text-slate-950">Clash 自定义模板</div>
        </CardHeader>
        <CardContent>
          <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,420px)_minmax(0,1fr)]">
            <form className="space-y-3" onSubmit={handleTemplateSubmit}>
              <Field label="模板名称">
                <Input
                  onChange={(event) =>
                    setTemplateForm({
                      ...templateForm,
                      name: event.target.value,
                    })
                  }
                  placeholder="我的 Clash 模板"
                  required
                  value={templateForm.name}
                />
              </Field>
              <Field label="模板 YAML">
                <textarea
                  className="min-h-64 w-full resize-y rounded-md border border-slate-200 bg-white px-3 py-2 font-mono text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
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
                  placeholder="可选"
                  value={templateForm.remark}
                />
              </Field>
              <div className="flex gap-2">
                <Button
                  disabled={saveTemplateMutation.isPending}
                  type="submit"
                >
                  {saveTemplateMutation.isPending
                    ? "保存中..."
                    : editingTemplateID
                      ? "保存模板"
                      : "新增模板"}
                </Button>
                {editingTemplateID && (
                  <Button
                    onClick={resetTemplateForm}
                    type="button"
                    variant="secondary"
                  >
                    取消
                  </Button>
                )}
              </div>
            </form>

            {customClashTemplates.length === 0 ? (
              <div className="rounded-md border border-dashed border-slate-200 p-8 text-center text-slate-500 text-sm">
                还没有自定义 Clash 模板。
              </div>
            ) : (
              <div className="min-w-0 space-y-3">
                {customClashTemplates.map((template) => (
                  <div
                    className="min-w-0 rounded-md border border-slate-200 p-4"
                    key={template.id}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="font-medium text-slate-950">
                          {template.name}
                        </div>
                        {template.remark && (
                          <div className="mt-1 text-slate-500 text-sm">
                            {template.remark}
                          </div>
                        )}
                      </div>
                      <div className="flex gap-2">
                        <Button
                          onClick={() => startEditTemplate(template)}
                          title="编辑模板"
                          variant="secondary"
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          onClick={() => {
                            if (window.confirm("确定删除这个模板吗？")) {
                              deleteTemplateMutation.mutate(template.id);
                            }
                          }}
                          title="删除模板"
                          variant="danger"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                    <pre className="mt-3 max-h-40 max-w-full overflow-auto whitespace-pre rounded-md bg-slate-950 p-3 text-slate-100 text-xs">
                      {template.content}
                    </pre>
                  </div>
                ))}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="font-medium text-slate-950">订阅列表</div>
        </CardHeader>
        <CardContent>
          {subscriptionsQuery.isLoading ? (
            <div className="text-slate-500 text-sm">加载中...</div>
          ) : subscriptions.length === 0 ? (
            <div className="rounded-md border border-dashed border-slate-200 p-8 text-center text-slate-500 text-sm">
              还没有订阅，选择节点后创建一个客户端订阅。
            </div>
          ) : (
            <div className="space-y-3">
              {subscriptions.map((subscription) => (
                <div
                  className="rounded-md border border-slate-200 p-4"
                  key={subscription.id}
                >
                  <div className="flex flex-col justify-between gap-3 md:flex-row md:items-start">
                    <div>
                      <div className="flex items-center gap-2">
                        <div className="font-medium text-slate-950">
                          {subscription.name}
                        </div>
                        <Badge>{subscription.enabled ? "启用" : "禁用"}</Badge>
                        <Badge>{subscription.format}</Badge>
                        {subscription.format === "clash-mihomo" && (
                          <Badge>
                            {clashTemplateLabel(subscription.clashTemplate)}
                          </Badge>
                        )}
                      </div>
                      <div className="mt-2 text-slate-500 text-sm">
                        {subscription.nodeCount} 个节点
                      </div>
                      {subscription.subscriptionUrl && (
                        <div className="mt-2 flex flex-wrap items-center gap-2 text-slate-700 text-sm">
                          <div className="flex min-w-0 items-center gap-2">
                            <Link2 className="h-4 w-4 shrink-0" />
                            <code className="break-all">
                              {subscription.subscriptionUrl}
                            </code>
                          </div>
                          <Button
                            onClick={() =>
                              copySubscriptionURL(
                                subscription.subscriptionUrl ?? "",
                              )
                            }
                            title="复制订阅链接"
                            variant="secondary"
                          >
                            <Copy className="h-4 w-4" />
                          </Button>
                        </div>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <Button
                        onClick={() => {
                          if (
                            window.confirm(
                              "确定重置 token 吗？旧订阅链接会立即失效。",
                            )
                          ) {
                            resetTokenMutation.mutate(subscription.id);
                          }
                        }}
                        title="重置 token"
                        variant="secondary"
                      >
                        <RotateCcw className="h-4 w-4" />
                      </Button>
                      <Button
                        onClick={() => startEdit(subscription)}
                        title="编辑"
                        variant="secondary"
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        onClick={() => {
                          if (window.confirm("确定删除这个订阅吗？")) {
                            deleteMutation.mutate(subscription.id);
                          }
                        }}
                        title="删除"
                        variant="danger"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
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
      <span className="mb-1 block text-slate-700 text-sm">{label}</span>
      {children}
    </div>
  );
}
