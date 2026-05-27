import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowUpRight,
  Cpu,
  LinkIcon,
  LoaderCircle,
  Pencil,
  Plus,
  Server,
  Trash2,
} from "lucide-react";
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
import { Card } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Dialog } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  getInstallProtocolFieldConfig,
  type InstallField,
  SUPPORTED_PROTOCOLS,
} from "@/constants/protocols";
import { cn } from "@/lib/utils";
import { useToastStore } from "@/stores/toast";
import type { ProtocolNode } from "@/types/domain";

type ImportMode = "link" | "install";
type ConfirmAction = {
  title: string;
  description: string;
  confirmLabel: string;
  onConfirm: () => void;
};

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
  publicPort: "",
  uuid: "",
  realityDomain: "",
  cdnDomain: "",
  argoDomain: "",
  argoToken: "",
  namePrefix: "",
  remark: "",
  shareLink: "",
};

/** Frontend parser: extract fields from a share link to pre-fill install form */
function _parseShareLinkToInstall(
  link: string,
  current: typeof emptyInstallForm,
): typeof emptyInstallForm {
  const trimmed = link.trim();
  if (!trimmed) return { ...current, shareLink: "" };

  // vmess:// is base64-encoded JSON
  if (trimmed.startsWith("vmess://")) {
    try {
      const encoded = trimmed.replace("vmess://", "");
      const decoded = atob(encoded);
      const json = JSON.parse(decoded);
      return {
        ...current,
        shareLink: trimmed,
        name: current.name || (json.ps as string) || `vmess-${json.add}`,
        protocol: "Vmess-ws",
        port: current.port || String(json.port || ""),
        uuid: current.uuid || (json.id as string) || "",
      };
    } catch {
      return { ...current, shareLink: trimmed };
    }
  }

  // Standard URI: vless://, hysteria2://, hy2://, tuic://, ss://, trojan://, socks5://
  try {
    const url = new URL(trimmed);
    const scheme = url.protocol.replace(":", "").toLowerCase();

    const protocolMap: Record<string, string> = {
      vless: "Vless-tcp-reality-vision",
      trojan: "AnyTLS",
      ss: "Shadowsocks-2022",
      hysteria2: "Hysteria2",
      hy2: "Hysteria2",
      tuic: "Tuic",
      socks: "Socks5",
      socks5: "Socks5",
    };

    const protocol = protocolMap[scheme] || current.protocol;
    const fragment = decodeURIComponent(url.hash.replace("#", ""));
    const name = current.name || fragment || `${scheme}-${url.hostname}`;
    const port = current.port || url.port || "";
    // For vless/trojan the userinfo is typically the UUID
    const uuid =
      current.uuid ||
      (scheme === "vless" || scheme === "trojan" ? url.username : "");

    return {
      ...current,
      shareLink: trimmed,
      name,
      protocol,
      port,
      uuid,
    };
  } catch {
    return { ...current, shareLink: trimmed };
  }
}

export function NodesPage() {
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<ImportMode>("install");
  const [manualForm, setManualForm] = useState(emptyManualForm);
  const [linkForm, setLinkForm] = useState(emptyLinkForm);
  const [installForm, setInstallForm] = useState(emptyInstallForm);
  const [editingNode, setEditingNode] = useState<ProtocolNode | null>(null);
  const [confirmAction, setConfirmAction] = useState<ConfirmAction | null>(
    null,
  );
  const [message, setMessage] = useState("");
  const [isNodeDialogOpen, setIsNodeDialogOpen] = useState(false);
  const addToast = useToastStore((state) => state.addToast);

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
      return importNode(buildImportPayload(linkForm));
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["nodes"] });
      resetForms();
      setIsNodeDialogOpen(false);
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "节点保存失败"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteNode,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["nodes"] });
    },
    onError: (error) => {
      addToast(getErrorMessage(error, "节点删除失败"), "error");
    },
  });

  const uninstallMutation = useMutation({
    mutationFn: (id: number) =>
      uninstallNode(id, { deleteAfterUninstall: true }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["nodes"] });
      await queryClient.invalidateQueries({ queryKey: ["tasks"] });
      addToast("卸载并删除任务已创建，请前往任务日志查看进度", "success");
    },
    onError: (error) => {
      addToast(getErrorMessage(error, "卸载任务创建失败"), "error");
    },
  });

  const nodes = nodesQuery.data ?? [];
  const visibleNodes = nodes.filter((node) => node.status !== "uninstalled");
  const servers = serversQuery.data ?? [];

  const findServerName = (node: ProtocolNode) => {
    if (node.serverId) {
      const server = servers.find((s) => s.id === node.serverId);
      if (server) return server.name;
    }
    // Fallback: match by IP address or hostname
    const matched = servers.find(
      (s) =>
        s.host === node.address ||
        s.host.includes(node.address) ||
        node.address.includes(s.host),
    );
    if (matched) return matched.name;
    return null;
  };

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    saveMutation.mutate();
  }

  function startEdit(node: ProtocolNode) {
    if (node.status === "uninstalled") {
      return;
    }
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
    setIsNodeDialogOpen(true);
  }

  const editingSystemNode = editingNode?.installMethod === "system";

  function resetForms() {
    setManualForm(emptyManualForm);
    setLinkForm(emptyLinkForm);
    setInstallForm(emptyInstallForm);
    setEditingNode(null);
    setMode("install");
    setMessage("");
  }

  return (
    <div className="space-y-8 py-4 max-w-7xl mx-auto">
      {/* Page Header */}
      <section className="flex flex-col justify-between gap-6 sm:flex-row sm:items-center">
        <div>
          <h1 className="font-bold text-2xl lg:text-3xl text-slate-100 tracking-tight font-display">
            协议节点
          </h1>
          <p className="mt-1 text-slate-400 text-xs font-semibold">
            部署或贴入各种高性能网络代理节点，支持节点状态健康检查与系统级一键装机部署。
          </p>
        </div>
        <Button
          onClick={() => {
            resetForms();
            setIsNodeDialogOpen(true);
          }}
          className="bg-white text-slate-950 hover:bg-slate-100 px-4 h-9 font-semibold text-xs tracking-wide rounded-lg flex items-center gap-1.5 self-start sm:self-center"
        >
          <Plus className="h-4 w-4" />
          添加协议节点
        </Button>
      </section>

      {/* Main Grid Nodes Layout */}
      <section>
        {nodesQuery.isLoading ? (
          <div className="text-slate-400 text-xs font-semibold animate-pulse py-10">
            正在拉取多协议节点数据...
          </div>
        ) : visibleNodes.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-white/[0.04] p-16 text-center text-slate-500 text-xs font-semibold">
            还没有任何协议节点。请点击右上角按钮新建或粘贴链接导入节点。
          </div>
        ) : (
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {visibleNodes.map((node) => {
              const isSuccess =
                node.status === "install_success" || node.status === "imported";
              const isFailed = node.status === "install_failed";
              const isProgress = ["installing", "uninstalling"].includes(
                node.status,
              );
              const canEdit =
                node.installMethod === "external" ||
                (node.installMethod === "system" &&
                  node.status === "install_success");
              const canDelete =
                node.installMethod === "external" ||
                node.status === "install_success" ||
                node.status === "install_failed";
              const deleteRequiresUninstall =
                node.installMethod === "system" &&
                node.status === "install_success";
              const hasActions = canEdit || canDelete;

              return (
                <Card
                  className={cn(
                    "bg-[#0e1017]/70 border-white/[0.04] shadow-lg shadow-black/20 flex flex-col justify-between",
                    "hover:border-white/[0.08] hover:-translate-y-0.5",
                  )}
                  key={node.id}
                >
                  {/* Card Content Top */}
                  <div className="p-6 pb-4 flex-1">
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex items-center gap-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/[0.04] bg-white/[0.02] text-slate-300 shadow-inner shrink-0">
                          <Cpu className="h-4 w-4 text-[#6366f1]" />
                        </div>
                        <div>
                          <div className="font-bold text-slate-200 text-sm tracking-wide">
                            {node.name}
                          </div>
                          <div className="flex items-center gap-1.5 mt-1">
                            <Badge className="border-slate-800 bg-slate-900/60 text-slate-400 font-mono text-[9px] px-1.5 py-0 rounded-md">
                              {node.protocol}
                            </Badge>
                            {node.installMethod === "system" ? (
                              <Badge className="border-violet-500/10 bg-violet-500/5 text-violet-400 text-[9px] px-1.5 py-0 rounded-md">
                                系统部署
                              </Badge>
                            ) : (
                              <Badge className="border-slate-800 bg-slate-900/60 text-slate-500 text-[9px] px-1.5 py-0 rounded-md">
                                外部导入
                              </Badge>
                            )}
                          </div>
                        </div>
                      </div>

                      {/* Connection status light badge */}
                      <StatusDot
                        status={node.status}
                        isSuccess={isSuccess}
                        isFailed={isFailed}
                        isProgress={isProgress}
                      />
                    </div>

                    <div className="mt-6 space-y-2 border-t border-white/[0.03] pt-4">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-slate-500 font-semibold text-[10px] uppercase tracking-wider">
                          网络接入端
                        </span>
                        <span className="font-mono text-slate-300 text-[11px] flex items-center gap-1">
                          <span>
                            {node.address}:{node.listenPort || node.port}
                          </span>
                          <ArrowUpRight className="h-3 w-3 text-slate-500" />
                        </span>
                      </div>

                      {node.publicPort && (
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-slate-500 font-semibold text-[10px] uppercase tracking-wider">
                            对外公网订阅端口
                          </span>
                          <span className="font-mono text-slate-300 text-[11px]">
                            {node.publicPort}
                          </span>
                        </div>
                      )}

                      {findServerName(node) && (
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-slate-500 font-semibold text-[10px] uppercase tracking-wider">
                            部署物理机
                          </span>
                          <Badge className="border-white/[0.04] bg-white/[0.02] text-slate-300 text-[10px] px-2 py-0.5 flex items-center gap-1.5 font-semibold">
                            <Server className="h-3 w-3 text-slate-500" />
                            <span>{findServerName(node)}</span>
                          </Badge>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Card Content Bottom Actions */}
                  {(hasActions || node.remark) && (
                    <div className="p-6 pt-4 border-t border-white/[0.03] bg-white/[0.01]">
                      {node.remark && (
                        <p className="text-[10px] text-slate-500 font-semibold mb-4 leading-normal">
                          备注: {node.remark}
                        </p>
                      )}

                      {hasActions && (
                        <div className="flex justify-end gap-2">
                          {canEdit && (
                            <Button
                              onClick={() => startEdit(node)}
                              variant="secondary"
                              className="flex-1 h-8 rounded-lg flex items-center justify-center gap-1 text-[10px]"
                            >
                              <Pencil className="h-3.5 w-3.5 opacity-70" />
                              <span>编辑参数</span>
                            </Button>
                          )}
                          {canDelete && (
                            <Button
                              onClick={() =>
                                setConfirmAction({
                                  title: deleteRequiresUninstall
                                    ? "卸载并删除节点"
                                    : "删除节点",
                                  description: deleteRequiresUninstall
                                    ? "确定删除这个系统部署节点吗？系统会先在服务器上卸载核心，卸载成功后自动删除节点记录并移除相关订阅关联。"
                                    : "确定删除这个节点吗？删除后它会从节点列表和相关订阅中移除。",
                                  confirmLabel: deleteRequiresUninstall
                                    ? "卸载并删除"
                                    : "删除节点",
                                  onConfirm: () =>
                                    deleteRequiresUninstall
                                      ? uninstallMutation.mutate(node.id)
                                      : deleteMutation.mutate(node.id),
                                })
                              }
                              variant="danger"
                              className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                              title="删除节点"
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          )}
                        </div>
                      )}
                    </div>
                  )}
                </Card>
              );
            })}
          </div>
        )}
      </section>

      {/* Node dialog form */}
      <Dialog
        isOpen={isNodeDialogOpen}
        onClose={() => {
          setIsNodeDialogOpen(false);
          resetForms();
        }}
        title={editingNode ? "编辑协议节点" : "添加协议节点"}
        size="md"
      >
        {!editingNode && (
          <div className="mb-5 grid grid-cols-2 gap-1 rounded-lg bg-slate-950 border border-slate-800 p-1 shrink-0">
            <button
              className={modeButtonClass(mode === "install")}
              onClick={() => setMode("install")}
              type="button"
            >
              系统安装
            </button>
            <button
              className={modeButtonClass(mode === "link")}
              onClick={() => setMode("link")}
              type="button"
            >
              分享链接
            </button>
          </div>
        )}

        <form className="space-y-4" onSubmit={handleSubmit}>
          {editingNode ? (
            <ManualNodeFields
              form={manualForm}
              lockedCore={editingSystemNode}
              setForm={setManualForm}
            />
          ) : mode === "link" ? (
            <LinkNodeFields form={linkForm} setForm={setLinkForm} />
          ) : (
            <InstallNodeFields
              form={installForm}
              servers={serversQuery.data ?? []}
              setForm={setInstallForm}
            />
          )}

          {message && (
            <p className="text-rose-400 text-xs font-semibold bg-red-500/5 border border-red-500/10 px-3.5 py-2.5 rounded-lg">
              {message}
            </p>
          )}

          <div className="flex justify-end gap-2.5 pt-2">
            <Button
              onClick={() => {
                setIsNodeDialogOpen(false);
                resetForms();
              }}
              type="button"
              variant="secondary"
              className="h-9 px-4 text-xs"
            >
              取消
            </Button>
            <Button
              disabled={saveMutation.isPending}
              type="submit"
              className="h-9 px-4 text-xs font-semibold bg-white text-slate-950 hover:bg-slate-100"
            >
              {mode === "link" && !editingNode ? (
                <LinkIcon className="h-4 w-4 shrink-0" />
              ) : (
                <Plus className="h-4 w-4 shrink-0" />
              )}
              {saveMutation.isPending
                ? "提交中..."
                : editingNode
                  ? "保存修改"
                  : mode === "install"
                    ? "发起安装"
                    : "导入节点"}
            </Button>
          </div>
        </form>
      </Dialog>
      <ConfirmDialog
        isOpen={Boolean(confirmAction)}
        title={confirmAction?.title ?? ""}
        description={confirmAction?.description ?? ""}
        confirmLabel={confirmAction?.confirmLabel}
        isPending={deleteMutation.isPending || uninstallMutation.isPending}
        onClose={() => setConfirmAction(null)}
        onConfirm={() => {
          confirmAction?.onConfirm();
          setConfirmAction(null);
        }}
      />
    </div>
  );
}

function StatusDot({
  status,
  isSuccess,
  isFailed,
  isProgress,
}: {
  status: string;
  isSuccess: boolean;
  isFailed: boolean;
  isProgress: boolean;
}) {
  const dotColor = isSuccess
    ? "bg-emerald-500"
    : isFailed
      ? "bg-rose-500"
      : isProgress
        ? "bg-indigo-500"
        : "bg-slate-500";

  const label =
    status === "imported"
      ? "导入成功"
      : status === "install_success"
        ? "已就绪"
        : status === "install_failed"
          ? "故障"
          : status === "installing"
            ? "部署中"
            : status === "uninstalling"
              ? "卸载中"
              : "已卸载";

  const borderClass = isSuccess
    ? "border-emerald-500/10 bg-emerald-500/5 text-emerald-400"
    : isFailed
      ? "border-rose-500/10 bg-rose-500/5 text-rose-400"
      : "border-slate-800 bg-slate-900/60 text-slate-400";
  const isRunningTask = status === "installing" || status === "uninstalling";

  return (
    <Badge className={cn("px-2 py-0.5", borderClass)}>
      {isRunningTask ? (
        <LoaderCircle className="mr-1.5 h-3 w-3 shrink-0 animate-spin text-indigo-400" />
      ) : (
        <span
          className={cn(
            "mr-1.5 h-1.5 w-1.5 rounded-full shrink-0",
            dotColor,
            (isSuccess || isProgress) && "animate-pulse",
          )}
        />
      )}
      <span>{label}</span>
    </Badge>
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
  const fieldConfig = getInstallProtocolFieldConfig(form.protocol);
  const visibleFields = new Set<InstallField>(fieldConfig.fields);
  const requiredFields = new Set<InstallField>(fieldConfig.requiredFields);
  const isVisible = (field: InstallField) => visibleFields.has(field);
  const isRequired = (field: InstallField) => requiredFields.has(field);

  const updateProtocol = (protocol: string) => {
    const nextConfig = getInstallProtocolFieldConfig(protocol);
    const nextVisibleFields = new Set<InstallField>(nextConfig.fields);
    setForm({
      ...form,
      protocol,
      uuid: nextVisibleFields.has("uuid") ? form.uuid : "",
      realityDomain: nextVisibleFields.has("realityDomain")
        ? form.realityDomain
        : "",
      cdnDomain: nextVisibleFields.has("cdnDomain") ? form.cdnDomain : "",
      argoDomain: nextVisibleFields.has("argoDomain") ? form.argoDomain : "",
      argoToken: nextVisibleFields.has("argoToken") ? form.argoToken : "",
      namePrefix: nextVisibleFields.has("namePrefix") ? form.namePrefix : "",
    });
  };

  return (
    <>
      <Field label="目标物理服务器" required>
        <Select
          value={form.serverId}
          onValueChange={(value) => setForm({ ...form, serverId: value })}
        >
          <SelectTrigger>
            <SelectValue
              placeholder="选择目标主机"
              displayValue={
                form.serverId
                  ? (() => {
                      const s = normalServers.find(
                        (srv) => String(srv.id) === form.serverId,
                      );
                      return s ? `${s.name} · ${s.host}` : undefined;
                    })()
                  : undefined
              }
            />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">选择目标主机</SelectItem>
            {normalServers.map((server) => (
              <SelectItem key={server.id} value={String(server.id)}>
                {server.name} · {server.host}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {normalServers.length === 0 && (
          <p className="mt-2 text-slate-400 text-[10px] leading-normal">
            暂无可安装的服务器，请确保至少有一台服务器在“服务器管理”中显示“正常”状态。
          </p>
        )}
      </Field>
      <Field label="节点名称" required>
        <Input
          onChange={(event) => setForm({ ...form, name: event.target.value })}
          placeholder="AnyTLS 自动化部署节点"
          required
          value={form.name}
        />
      </Field>
      <Field label="底层核心协议" required>
        <Select value={form.protocol} onValueChange={updateProtocol}>
          <SelectTrigger>
            <SelectValue displayValue={form.protocol} />
          </SelectTrigger>
          <SelectContent>
            {SUPPORTED_PROTOCOLS.map((protocol) => (
              <SelectItem key={protocol} value={protocol}>
                {protocol}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
      {(isVisible("port") || isVisible("publicPort")) && (
        <div className="grid gap-3 md:grid-cols-2">
          {isVisible("port") && (
            <Field label="安装/监听端口" required={isRequired("port")}>
              <Input
                max={65535}
                min={1}
                onChange={(event) =>
                  setForm({ ...form, port: event.target.value })
                }
                placeholder="留空则由后端自动生成"
                required={isRequired("port")}
                type="number"
                value={form.port}
              />
            </Field>
          )}
          {isVisible("publicPort") && (
            <Field label="对外公网订阅端口" required={isRequired("publicPort")}>
              <Input
                max={65535}
                min={1}
                onChange={(event) =>
                  setForm({ ...form, publicPort: event.target.value })
                }
                placeholder="留空则复用监听端口"
                required={isRequired("publicPort")}
                type="number"
                value={form.publicPort}
              />
            </Field>
          )}
        </div>
      )}
      {isVisible("uuid") && (
        <Field
          label={fieldConfig.uuidLabel ?? "自定义 UUID / 密钥"}
          required={isRequired("uuid")}
        >
          <Input
            onChange={(event) => setForm({ ...form, uuid: event.target.value })}
            placeholder={
              fieldConfig.uuidPlaceholder ?? "留空则由后端生成强随机 UUID"
            }
            required={isRequired("uuid")}
            value={form.uuid}
          />
        </Field>
      )}
      {(isVisible("realityDomain") || isVisible("cdnDomain")) && (
        <div className="grid gap-3 md:grid-cols-2">
          {isVisible("realityDomain") && (
            <Field
              label="Reality 伪装域名"
              required={isRequired("realityDomain")}
            >
              <Input
                onChange={(event) =>
                  setForm({ ...form, realityDomain: event.target.value })
                }
                placeholder="留空使用脚本默认域名"
                required={isRequired("realityDomain")}
                value={form.realityDomain}
              />
            </Field>
          )}
          {isVisible("cdnDomain") && (
            <Field label="CDN 优选 Host" required={isRequired("cdnDomain")}>
              <Input
                onChange={(event) =>
                  setForm({ ...form, cdnDomain: event.target.value })
                }
                placeholder="可选"
                required={isRequired("cdnDomain")}
                value={form.cdnDomain}
              />
            </Field>
          )}
        </div>
      )}
      {(isVisible("argoDomain") || isVisible("argoToken")) && (
        <div className="grid gap-3 md:grid-cols-2">
          {isVisible("argoDomain") && (
            <Field label="Argo 固定域名" required={isRequired("argoDomain")}>
              <Input
                onChange={(event) =>
                  setForm({ ...form, argoDomain: event.target.value })
                }
                placeholder="如 tunnel.example.com"
                required={isRequired("argoDomain")}
                value={form.argoDomain}
              />
            </Field>
          )}
          {isVisible("argoToken") && (
            <Field label="Argo Tunnel Token" required={isRequired("argoToken")}>
              <Input
                onChange={(event) =>
                  setForm({ ...form, argoToken: event.target.value })
                }
                placeholder="Cloudflare Tunnel token"
                required={isRequired("argoToken")}
                value={form.argoToken}
              />
            </Field>
          )}
        </div>
      )}
      {isVisible("namePrefix") && (
        <Field label="节点名称前缀" required={isRequired("namePrefix")}>
          <Input
            onChange={(event) =>
              setForm({ ...form, namePrefix: event.target.value })
            }
            placeholder="留空则使用节点名称"
            required={isRequired("namePrefix")}
            value={form.namePrefix}
          />
        </Field>
      )}
      {isVisible("remark") && (
        <Field label="备注" required={isRequired("remark")}>
          <Input
            onChange={(event) =>
              setForm({ ...form, remark: event.target.value })
            }
            placeholder="可选备注描述"
            required={isRequired("remark")}
            value={form.remark}
          />
        </Field>
      )}
    </>
  );
}

function ManualNodeFields({
  form,
  lockedCore = false,
  setForm,
}: {
  form: typeof emptyManualForm;
  lockedCore?: boolean;
  setForm: (form: typeof emptyManualForm) => void;
}) {
  return (
    <>
      <Field label="节点名称" required>
        <Input
          onChange={(event) => setForm({ ...form, name: event.target.value })}
          placeholder="香港 Hysteria2"
          required
          value={form.name}
        />
      </Field>
      <Field label="传输协议" required>
        <Select
          value={form.protocol}
          onValueChange={(value) => setForm({ ...form, protocol: value })}
        >
          <SelectTrigger disabled={lockedCore}>
            <SelectValue displayValue={form.protocol} />
          </SelectTrigger>
          <SelectContent>
            {SUPPORTED_PROTOCOLS.map((protocol) => (
              <SelectItem key={protocol} value={protocol}>
                {protocol}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
      <div className="grid gap-3 md:grid-cols-[1fr_110px]">
        <Field label="连接地址/IP" required>
          <Input
            onChange={(event) =>
              setForm({ ...form, address: event.target.value })
            }
            placeholder="example.com"
            required
            value={form.address}
          />
        </Field>
        <Field label="默认端口" required>
          <Input
            max={65535}
            min={1}
            onChange={(event) => {
              const value = Number(event.target.value);
              setForm({ ...form, port: value, listenPort: value });
            }}
            disabled={lockedCore}
            required
            type="number"
            value={form.port}
          />
        </Field>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="底层监听端口" required>
          <Input
            max={65535}
            min={1}
            onChange={(event) =>
              setForm({ ...form, listenPort: Number(event.target.value) })
            }
            disabled={lockedCore}
            required
            type="number"
            value={form.listenPort}
          />
        </Field>
        <Field label="公网映射/订阅端口">
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
      {!lockedCore && (
        <Field label="节点机密参数 (如 UUID、Password 等)">
          <textarea
            className="min-h-24 w-full resize-y rounded-lg border border-white/[0.06] bg-slate-950 px-3.5 py-2.5 text-xs text-slate-100 font-mono outline-none transition-all duration-300 focus:border-white/20"
            onChange={(event) =>
              setForm({ ...form, sensitive: event.target.value })
            }
            placeholder="请输入节点的连接密码、UUID 或私钥等敏感参数；保存后出于安全将不会明文返回"
            value={form.sensitive}
          />
        </Field>
      )}
      <Field label="备注">
        <Input
          onChange={(event) => setForm({ ...form, remark: event.target.value })}
          placeholder="添加备注"
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
      <Field
        label="节点分享链接 (支持 vless://, vmess://, hysteria2://)"
        required
      >
        <textarea
          className="min-h-28 w-full resize-y rounded-lg border border-white/[0.06] bg-slate-950 px-3.5 py-2.5 text-xs text-slate-100 font-mono outline-none transition-all duration-300 focus:border-white/20"
          onChange={(event) =>
            setForm({ ...form, rawLink: event.target.value })
          }
          placeholder="粘贴您的分享链接..."
          required
          value={form.rawLink}
        />
      </Field>
      <Field label="显示名称">
        <Input
          onChange={(event) =>
            setForm({ ...form, displayName: event.target.value })
          }
          placeholder="留空则提取链接中的真实名称"
          value={form.displayName}
        />
      </Field>
    </>
  );
}

function Field({
  label,
  required = false,
  children,
}: {
  label: string;
  required?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className="block">
      <span className="mb-1.5 flex items-center gap-1.5 text-slate-500 text-[9px] font-bold uppercase tracking-widest">
        <span>{label}</span>
        <span
          className={cn(
            "rounded border px-1 py-0 text-[9px] leading-4 tracking-normal",
            required
              ? "border-rose-500/20 bg-rose-500/10 text-rose-300"
              : "border-white/[0.06] bg-white/[0.03] text-slate-500",
          )}
        >
          {required ? "必填" : "可选"}
        </span>
      </span>
      {children}
    </div>
  );
}

function buildImportPayload(linkForm: typeof emptyLinkForm): NodeImportPayload {
  return {
    mode: "link",
    rawLink: linkForm.rawLink,
    displayName: linkForm.displayName,
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

// biome-ignore lint/suspicious/noExplicitAny: internal mapper
function buildInstallPayload(form: any): NodeInstallPayload {
  return {
    serverId: Number(form.serverId),
    name: form.name,
    protocol: form.protocol,
    port: form.port ? Number(form.port) : undefined,
    publicPort: form.publicPort ? Number(form.publicPort) : null,
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
  return cn(
    "h-9 flex-1 rounded-md text-xs font-semibold tracking-wide transition-all duration-200 cursor-pointer select-none",
    active
      ? "bg-white text-slate-950 font-bold"
      : "text-slate-400 hover:text-slate-200 hover:bg-white/5",
  );
}
