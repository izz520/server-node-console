import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Activity, LinkIcon, Pencil, Plus, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import { getErrorMessage } from "@/api/errors";
import {
  deleteNode,
  importNode,
  installNode,
  listNodes,
  listServers,
  type NodeImportPayload,
  type NodeInstallPayload,
  type NodeUpdatePayload,
  uninstallNode,
  updateNode,
} from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { SUPPORTED_PROTOCOLS } from "@/constants/protocols";
import type { ProtocolNode } from "@/types/domain";

type ImportMode = "manual" | "link" | "install";

const emptyManualForm = {
  name: "",
  protocol: "Hysteria2",
  address: "",
  port: 443,
  listenPort: 443,
  publicPort: "",
  remark: "",
  sensitive: "",
};

const emptyLinkForm = {
  rawLink: "",
  displayName: "",
};

const emptyInstallForm = {
  serverId: "",
  name: "",
  protocol: "AnyTLS",
  port: "",
  uuid: "",
  realityDomain: "",
  cdnDomain: "",
  argoDomain: "",
  argoToken: "",
  namePrefix: "",
  remark: "",
};

export function NodesPage() {
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<ImportMode>("manual");
  const [manualForm, setManualForm] = useState(emptyManualForm);
  const [linkForm, setLinkForm] = useState(emptyLinkForm);
  const [installForm, setInstallForm] = useState(emptyInstallForm);
  const [editingNode, setEditingNode] = useState<ProtocolNode | null>(null);
  const [message, setMessage] = useState("");

  const nodesQuery = useQuery({
    queryKey: ["nodes"],
    queryFn: listNodes,
    refetchInterval: 5000,
  });

  const serversQuery = useQuery({
    queryKey: ["servers"],
    queryFn: listServers,
    refetchInterval: 5000,
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      if (editingNode) {
        return updateNode(editingNode.id, buildUpdatePayload(manualForm));
      }
      if (mode === "install") {
        return installNode(buildInstallPayload(installForm)).then(
          (response) => response.node,
        );
      }
      return importNode(buildImportPayload(mode, manualForm, linkForm));
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["nodes"] });
      resetForms();
      setMessage("节点已保存");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "节点保存失败"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteNode,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["nodes"] });
      setMessage("节点已删除");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "节点删除失败"));
    },
  });

  const uninstallMutation = useMutation({
    mutationFn: uninstallNode,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["nodes"] });
      await queryClient.invalidateQueries({ queryKey: ["tasks"] });
      setMessage("卸载任务已创建");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "卸载任务创建失败"));
    },
  });

  const nodes = nodesQuery.data ?? [];

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    saveMutation.mutate();
  }

  function startEdit(node: ProtocolNode) {
    setMode("manual");
    setEditingNode(node);
    setManualForm({
      name: node.name,
      protocol: node.protocol,
      address: node.address,
      port: node.port || node.listenPort || 443,
      listenPort: node.listenPort || node.port || 443,
      publicPort: node.publicPort ? String(node.publicPort) : "",
      remark: node.remark ?? "",
      sensitive: "",
    });
    setMessage("");
  }

  function resetForms() {
    setManualForm(emptyManualForm);
    setLinkForm(emptyLinkForm);
    setInstallForm(emptyInstallForm);
    setEditingNode(null);
    setMode("manual");
  }

  return (
    <div className="grid gap-6 xl:grid-cols-[420px_1fr]">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-slate-950 text-white">
              <Activity className="h-4 w-4" />
            </div>
            <div>
              <h1 className="font-semibold text-slate-950 text-xl">
                {editingNode ? "编辑外部节点" : "添加外部节点"}
              </h1>
              <p className="text-slate-500 text-sm">
                外部节点不会触发服务器安装任务
              </p>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {!editingNode && (
            <div className="mb-4 grid grid-cols-2 gap-2 rounded-md bg-slate-100 p-1">
              <button
                className={modeButtonClass(mode === "manual")}
                onClick={() => setMode("manual")}
                type="button"
              >
                手动填写
              </button>
              <button
                className={modeButtonClass(mode === "link")}
                onClick={() => setMode("link")}
                type="button"
              >
                分享链接
              </button>
              <button
                className={modeButtonClass(mode === "install")}
                onClick={() => setMode("install")}
                type="button"
              >
                系统安装
              </button>
            </div>
          )}

          <form className="space-y-4" onSubmit={handleSubmit}>
            {editingNode || mode === "manual" ? (
              <ManualNodeFields form={manualForm} setForm={setManualForm} />
            ) : mode === "link" ? (
              <LinkNodeFields form={linkForm} setForm={setLinkForm} />
            ) : (
              <InstallNodeFields
                form={installForm}
                servers={serversQuery.data ?? []}
                setForm={setInstallForm}
              />
            )}

            {message && <p className="text-slate-600 text-sm">{message}</p>}
            <div className="flex gap-2">
              <Button disabled={saveMutation.isPending} type="submit">
                {mode === "link" && !editingNode ? (
                  <LinkIcon className="h-4 w-4" />
                ) : (
                  <Plus className="h-4 w-4" />
                )}
                {saveMutation.isPending
                  ? "保存中..."
                  : editingNode
                    ? "保存修改"
                    : mode === "install"
                      ? "发起安装"
                      : "添加节点"}
              </Button>
              {editingNode && (
                <Button onClick={resetForms} type="button" variant="secondary">
                  取消
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="font-medium text-slate-950">协议节点列表</div>
        </CardHeader>
        <CardContent>
          {nodesQuery.isLoading ? (
            <div className="text-slate-500 text-sm">加载中...</div>
          ) : nodes.length === 0 ? (
            <div className="rounded-md border border-dashed border-slate-200 p-8 text-center text-slate-500 text-sm">
              还没有节点，可以先手动添加外部节点或粘贴分享链接导入。
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[760px] border-collapse text-left text-sm">
                <thead>
                  <tr className="border-slate-100 border-b text-slate-500">
                    <th className="py-3 pr-3 font-medium">名称</th>
                    <th className="py-3 pr-3 font-medium">协议</th>
                    <th className="py-3 pr-3 font-medium">地址</th>
                    <th className="py-3 pr-3 font-medium">状态</th>
                    <th className="py-3 pr-3 font-medium">来源</th>
                    <th className="py-3 pr-3 text-right font-medium">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {nodes.map((node) => (
                    <tr className="border-slate-100 border-b" key={node.id}>
                      <td className="py-3 pr-3">
                        <div className="font-medium text-slate-950">
                          {node.name}
                        </div>
                        {node.remark && (
                          <div className="mt-1 text-slate-500 text-xs">
                            {node.remark}
                          </div>
                        )}
                      </td>
                      <td className="py-3 pr-3 text-slate-700">
                        {node.protocol}
                      </td>
                      <td className="py-3 pr-3 text-slate-700">
                        {node.address}:
                        {node.publicPort || node.port || node.listenPort}
                      </td>
                      <td className="py-3 pr-3">
                        <Badge>
                          {node.status === "imported"
                            ? "外部导入"
                            : node.status}
                        </Badge>
                      </td>
                      <td className="py-3 pr-3 text-slate-700">
                        {node.installMethod === "external"
                          ? "外部节点"
                          : "系统安装"}
                      </td>
                      <td className="py-3 pr-3">
                        <div className="flex justify-end gap-2">
                          {node.installMethod === "external" && (
                            <Button
                              onClick={() => startEdit(node)}
                              title="编辑"
                              variant="secondary"
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                          )}
                          {(node.installMethod === "external" ||
                            node.status === "uninstalled") && (
                            <Button
                              onClick={() => {
                                if (window.confirm("确定删除这个节点吗？")) {
                                  deleteMutation.mutate(node.id);
                                }
                              }}
                              title="删除"
                              variant="danger"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          )}
                          {node.installMethod === "system" &&
                            node.status === "install_success" && (
                              <Button
                                onClick={() => {
                                  if (
                                    window.confirm(
                                      "确定卸载这个系统安装节点吗？",
                                    )
                                  ) {
                                    uninstallMutation.mutate(node.id);
                                  }
                                }}
                                title="卸载"
                                variant="danger"
                              >
                                卸载
                              </Button>
                            )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function InstallNodeFields({
  form,
  servers,
  setForm,
}: {
  form: typeof emptyInstallForm;
  servers: Array<{ id: number; name: string; host: string; status: string }>;
  setForm: (form: typeof emptyInstallForm) => void;
}) {
  const normalServers = servers.filter((server) => server.status === "normal");

  return (
    <>
      <Field label="服务器">
        <select
          className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
          onChange={(event) =>
            setForm({ ...form, serverId: event.target.value })
          }
          required
          value={form.serverId}
        >
          <option value="">选择服务器</option>
          {normalServers.map((server) => (
            <option key={server.id} value={server.id}>
              {server.name} · {server.host}
            </option>
          ))}
        </select>
        {normalServers.length === 0 && (
          <p className="mt-2 text-slate-500 text-xs">
            暂无可安装服务器，请先在服务器页面完成 SSH 连通性测试。
          </p>
        )}
      </Field>
      <Field label="节点名称">
        <Input
          onChange={(event) => setForm({ ...form, name: event.target.value })}
          placeholder="AnyTLS 节点"
          required
          value={form.name}
        />
      </Field>
      <Field label="协议">
        <select
          className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
          onChange={(event) =>
            setForm({ ...form, protocol: event.target.value })
          }
          required
          value={form.protocol}
        >
          {SUPPORTED_PROTOCOLS.filter(
            (protocol) => !protocol.includes("Argo"),
          ).map((protocol) => (
            <option key={protocol} value={protocol}>
              {protocol}
            </option>
          ))}
        </select>
      </Field>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="端口">
          <Input
            max={65535}
            min={1}
            onChange={(event) => setForm({ ...form, port: event.target.value })}
            placeholder="留空由后端自动生成"
            type="number"
            value={form.port}
          />
        </Field>
        <Field label="UUID">
          <Input
            onChange={(event) => setForm({ ...form, uuid: event.target.value })}
            placeholder="留空由后端自动生成"
            value={form.uuid}
          />
        </Field>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="Reality 域名">
          <Input
            onChange={(event) =>
              setForm({ ...form, realityDomain: event.target.value })
            }
            placeholder="可选"
            value={form.realityDomain}
          />
        </Field>
        <Field label="CDN Host">
          <Input
            onChange={(event) =>
              setForm({ ...form, cdnDomain: event.target.value })
            }
            placeholder="可选"
            value={form.cdnDomain}
          />
        </Field>
      </div>
      <Field label="备注">
        <Input
          onChange={(event) => setForm({ ...form, remark: event.target.value })}
          placeholder="可选"
          value={form.remark}
        />
      </Field>
    </>
  );
}

function ManualNodeFields({
  form,
  setForm,
}: {
  form: typeof emptyManualForm;
  setForm: (form: typeof emptyManualForm) => void;
}) {
  return (
    <>
      <Field label="节点名称">
        <Input
          onChange={(event) => setForm({ ...form, name: event.target.value })}
          placeholder="香港 Hysteria2"
          required
          value={form.name}
        />
      </Field>
      <Field label="协议">
        <select
          className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
          onChange={(event) =>
            setForm({ ...form, protocol: event.target.value })
          }
          required
          value={form.protocol}
        >
          {SUPPORTED_PROTOCOLS.map((protocol) => (
            <option key={protocol} value={protocol}>
              {protocol}
            </option>
          ))}
        </select>
      </Field>
      <div className="grid gap-3 md:grid-cols-[1fr_110px]">
        <Field label="地址">
          <Input
            onChange={(event) =>
              setForm({ ...form, address: event.target.value })
            }
            placeholder="example.com"
            required
            value={form.address}
          />
        </Field>
        <Field label="端口">
          <Input
            max={65535}
            min={1}
            onChange={(event) => {
              const value = Number(event.target.value);
              setForm({ ...form, port: value, listenPort: value });
            }}
            required
            type="number"
            value={form.port}
          />
        </Field>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="监听端口">
          <Input
            max={65535}
            min={1}
            onChange={(event) =>
              setForm({ ...form, listenPort: Number(event.target.value) })
            }
            required
            type="number"
            value={form.listenPort}
          />
        </Field>
        <Field label="对外端口">
          <Input
            max={65535}
            min={1}
            onChange={(event) =>
              setForm({ ...form, publicPort: event.target.value })
            }
            placeholder="可选"
            type="number"
            value={form.publicPort}
          />
        </Field>
      </div>
      <Field label="敏感参数">
        <textarea
          className="min-h-24 w-full resize-y rounded-md border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
          onChange={(event) =>
            setForm({ ...form, sensitive: event.target.value })
          }
          placeholder="UUID、密码、私钥等；保存后不会明文返回"
          value={form.sensitive}
        />
      </Field>
      <Field label="备注">
        <Input
          onChange={(event) => setForm({ ...form, remark: event.target.value })}
          placeholder="可选"
          value={form.remark}
        />
      </Field>
    </>
  );
}

function LinkNodeFields({
  form,
  setForm,
}: {
  form: typeof emptyLinkForm;
  setForm: (form: typeof emptyLinkForm) => void;
}) {
  return (
    <>
      <Field label="分享链接">
        <textarea
          className="min-h-28 w-full resize-y rounded-md border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
          onChange={(event) =>
            setForm({ ...form, rawLink: event.target.value })
          }
          placeholder="vless://...、vmess://...、hysteria2://..."
          required
          value={form.rawLink}
        />
      </Field>
      <Field label="显示名称">
        <Input
          onChange={(event) =>
            setForm({ ...form, displayName: event.target.value })
          }
          placeholder="可选；留空则使用链接中的名称"
          value={form.displayName}
        />
      </Field>
    </>
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

function buildImportPayload(
  mode: "manual" | "link",
  manualForm: typeof emptyManualForm,
  linkForm: typeof emptyLinkForm,
): NodeImportPayload {
  if (mode === "link") {
    return {
      mode,
      rawLink: linkForm.rawLink,
      displayName: linkForm.displayName,
    };
  }

  return {
    mode,
    name: manualForm.name,
    protocol: manualForm.protocol,
    address: manualForm.address,
    port: Number(manualForm.port),
    listenPort: Number(manualForm.listenPort),
    publicPort: manualForm.publicPort ? Number(manualForm.publicPort) : null,
    remark: manualForm.remark,
    sensitive: manualForm.sensitive,
  };
}

function buildUpdatePayload(form: typeof emptyManualForm): NodeUpdatePayload {
  return {
    name: form.name,
    protocol: form.protocol,
    address: form.address,
    port: Number(form.port),
    listenPort: Number(form.listenPort),
    publicPort: form.publicPort ? Number(form.publicPort) : null,
    remark: form.remark,
    sensitive: form.sensitive,
  };
}

function buildInstallPayload(
  form: typeof emptyInstallForm,
): NodeInstallPayload {
  return {
    serverId: Number(form.serverId),
    name: form.name,
    protocol: form.protocol,
    port: form.port ? Number(form.port) : undefined,
    uuid: form.uuid,
    realityDomain: form.realityDomain,
    cdnDomain: form.cdnDomain,
    argoDomain: form.argoDomain,
    argoToken: form.argoToken,
    namePrefix: form.namePrefix,
    remark: form.remark,
  };
}

function modeButtonClass(active: boolean) {
  return [
    "h-9 rounded-md text-sm font-medium transition",
    active
      ? "bg-white text-slate-950 shadow-sm"
      : "text-slate-600 hover:text-slate-950",
  ].join(" ");
}
