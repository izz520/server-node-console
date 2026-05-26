import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Globe,
  Network,
  Pencil,
  PlugZap,
  Plus,
  Server,
  Tag,
  Trash2,
} from "lucide-react";
import { type FormEvent, useMemo, useState } from "react";
import { getErrorMessage } from "@/api/errors";
import {
  createNATMapping,
  createServer,
  deleteNATMapping,
  deleteServer,
  listNATMappings,
  listServers,
  type NATMappingPayload,
  type ServerPayload,
  testServerSSH,
  updateNATMapping,
  updateServer,
} from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type {
  NATPortMapping,
  Server as ServerModel,
  SSHAuthMethod,
} from "@/types/domain";

const emptyForm: ServerPayload = {
  name: "",
  host: "",
  sshPort: 22,
  sshUsername: "root",
  authMethod: "password",
  password: "",
  privateKey: "",
  region: "",
  tags: "",
  remark: "",
};

const statusLabels = {
  normal: "运行中",
  connection_failed: "连接失败",
  disabled: "已禁用",
};

const emptyNATForm: NATMappingPayload = {
  name: "",
  transport: "TCP",
  listenPort: 8000,
  publicPort: 9000,
  remark: "",
};

export function ServersPage() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<ServerPayload>(emptyForm);
  const [natForm, setNATForm] = useState<NATMappingPayload>(emptyNATForm);
  const [editingServer, setEditingServer] = useState<ServerModel | null>(null);
  const [editingMapping, setEditingMapping] = useState<NATPortMapping | null>(
    null,
  );
  const [selectedServerID, setSelectedServerID] = useState<number | null>(null);
  const [message, setMessage] = useState("");
  const [natMessage, setNATMessage] = useState("");

  // Dialog State controls
  const [isServerDialogOpen, setIsServerDialogOpen] = useState(false);
  const [isNATDialogOpen, setIsNATDialogOpen] = useState(false);

  const serversQuery = useQuery({
    queryKey: ["servers"],
    queryFn: listServers,
  });

  const selectedServer = serversQuery.data?.find(
    (server) => server.id === selectedServerID,
  );

  const natMappingsQuery = useQuery({
    queryKey: ["nat-mappings", selectedServerID],
    queryFn: () => listNATMappings(selectedServerID ?? 0),
    enabled: Boolean(selectedServerID),
  });

  const saveMutation = useMutation({
    mutationFn: (payload: ServerPayload) =>
      editingServer
        ? updateServer(editingServer.id, payload)
        : createServer(payload),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["servers"] });
      resetForm();
      setIsServerDialogOpen(false);
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "保存失败，请检查 SSH 连接信息"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteServer,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["servers"] });
    },
    onError: (error) => {
      alert(getErrorMessage(error, "删除失败"));
    },
  });

  const testMutation = useMutation({
    mutationFn: testServerSSH,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["servers"] });
      alert("SSH 连通性测试成功");
    },
    onError: (error) => {
      alert(getErrorMessage(error, "SSH 连通性测试失败"));
      queryClient.invalidateQueries({ queryKey: ["servers"] });
    },
  });

  const saveNATMutation = useMutation({
    mutationFn: (payload: NATMappingPayload) => {
      if (editingMapping) {
        return updateNATMapping(editingMapping.id, payload);
      }
      if (!selectedServerID) {
        throw new Error("请先选择服务器");
      }
      return createNATMapping(selectedServerID, payload);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["nat-mappings", selectedServerID],
      });
      resetNATForm();
      setNATMessage("NAT 映射已保存");
    },
    onError: (error) => {
      setNATMessage(getErrorMessage(error, "保存 NAT 映射失败"));
    },
  });

  const deleteNATMutation = useMutation({
    mutationFn: deleteNATMapping,
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["nat-mappings", selectedServerID],
      });
      setNATMessage("NAT 映射已删除");
    },
    onError: (error) => {
      setNATMessage(getErrorMessage(error, "删除 NAT 映射失败"));
    },
  });

  const servers = serversQuery.data ?? [];
  const natMappings = natMappingsQuery.data ?? [];
  const isSaving = saveMutation.isPending;
  const credentialHint = useMemo(() => {
    if (!editingServer) {
      return "";
    }
    if (form.authMethod === "password" && editingServer.hasPassword) {
      return "已保存密码；留空则继续使用原密码";
    }
    if (form.authMethod === "private_key" && editingServer.hasPrivateKey) {
      return "已保存私钥；留空则继续使用原私钥";
    }
    return "";
  }, [editingServer, form.authMethod]);

  function resetForm() {
    setForm(emptyForm);
    setEditingServer(null);
    setMessage("");
  }

  function resetNATForm() {
    setNATForm(emptyNATForm);
    setEditingMapping(null);
    setNATMessage("");
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    saveMutation.mutate({
      ...form,
      sshPort: Number(form.sshPort),
      password: form.authMethod === "password" ? form.password : "",
      privateKey: form.authMethod === "private_key" ? form.privateKey : "",
    });
  }

  function startEdit(server: ServerModel) {
    setEditingServer(server);
    setForm({
      name: server.name,
      host: server.host,
      sshPort: server.sshPort,
      sshUsername: server.sshUsername,
      authMethod: server.authMethod,
      password: "",
      privateKey: "",
      region: server.region ?? "",
      tags: server.tags ?? "",
      remark: server.remark ?? "",
    });
    setMessage("");
    setIsServerDialogOpen(true);
  }

  function handleNATSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNATMessage("");
    saveNATMutation.mutate({
      ...natForm,
      listenPort: Number(natForm.listenPort),
      publicPort: Number(natForm.publicPort),
    });
  }

  function startEditMapping(mapping: NATPortMapping) {
    setEditingMapping(mapping);
    setNATForm({
      name: mapping.name,
      transport: mapping.transport || "TCP",
      listenPort: mapping.listenPort,
      publicPort: mapping.publicPort,
      remark: mapping.remark ?? "",
    });
    setNATMessage("");
  }

  return (
    <div className="space-y-8 py-4 max-w-7xl mx-auto">
      {/* Page Header */}
      <section className="flex flex-col justify-between gap-6 sm:flex-row sm:items-center">
        <div>
          <h1 className="font-bold text-2xl lg:text-3xl text-slate-100 tracking-tight font-display">
            物理服务器
          </h1>
          <p className="mt-1 text-slate-400 text-xs font-semibold">
            托管并运维节点所承载的底层云主机，支持 NAT 网络映射及 SSH
            安全密钥测试。
          </p>
        </div>
        <Button
          onClick={() => {
            resetForm();
            setIsServerDialogOpen(true);
          }}
          className="bg-white text-slate-950 hover:bg-slate-100 px-4 h-9 font-semibold text-xs tracking-wide rounded-lg flex items-center gap-1.5 self-start sm:self-center"
        >
          <Plus className="h-4 w-4" />
          添加服务器
        </Button>
      </section>

      {/* Main Grid Rack Layout */}
      <section>
        {serversQuery.isLoading ? (
          <div className="text-slate-400 text-xs font-semibold animate-pulse py-10">
            正在与云端服务器进行心跳同步...
          </div>
        ) : servers.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-white/[0.04] p-16 text-center text-slate-500 text-xs font-semibold">
            还没有任何物理服务器。请点击右上角按钮添加您的第一台云主机。
          </div>
        ) : (
          <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
            {servers.map((server) => (
              <Card
                className="bg-[#0e1017]/70 border-white/[0.04] shadow-lg shadow-black/20 hover:border-white/[0.08] hover:-translate-y-0.5 flex flex-col justify-between"
                key={server.id}
              >
                {/* Card Top */}
                <div className="p-6 pb-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-center gap-2.5">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/[0.04] bg-white/[0.02] text-slate-300 shadow-inner shrink-0">
                        <Server className="h-4 w-4" />
                      </div>
                      <div>
                        <div className="font-bold text-slate-200 text-sm tracking-wide">
                          {server.name}
                        </div>
                        <div className="text-[10px] text-slate-500 font-semibold font-mono mt-0.5">
                          ID: #{server.id}
                        </div>
                      </div>
                    </div>
                    <StatusBadge status={server.status} />
                  </div>

                  <div className="mt-6 space-y-2 border-t border-white/[0.03] pt-4">
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-slate-500 font-semibold text-[10px] uppercase tracking-wider">
                        SSH 凭据地址
                      </span>
                      <span className="font-mono text-slate-300 text-[11px]">
                        {server.sshUsername}@{server.host}:{server.sshPort}
                      </span>
                    </div>

                    <div className="flex items-center justify-between text-xs">
                      <span className="text-slate-500 font-semibold text-[10px] uppercase tracking-wider">
                        鉴权机制
                      </span>
                      <span className="text-slate-300 font-medium">
                        {server.authMethod === "password"
                          ? "密码认证"
                          : "SSH 私钥"}
                      </span>
                    </div>

                    {server.region && (
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-slate-500 font-semibold text-[10px] uppercase tracking-wider">
                          部署属地
                        </span>
                        <div className="flex items-center gap-1 text-slate-300 font-medium">
                          <Globe className="h-3 w-3 text-slate-500" />
                          <span>{server.region}</span>
                        </div>
                      </div>
                    )}
                  </div>
                </div>

                {/* Card Bottom Actions */}
                <div className="p-6 pt-4 border-t border-white/[0.03] bg-white/[0.01]">
                  {/* Tags remark block */}
                  {(server.tags || server.remark) && (
                    <div className="flex flex-wrap items-center gap-1.5 mb-4">
                      {server.tags?.split(",").map((tag) => (
                        <Badge
                          className="border-slate-800 bg-slate-900/60 text-slate-400 font-mono text-[9px] px-1.5"
                          key={tag}
                        >
                          <Tag className="mr-1 h-2.5 w-2.5 text-slate-500" />
                          {tag.trim()}
                        </Badge>
                      ))}
                      {server.remark && (
                        <div className="text-[10px] text-slate-500 font-medium truncate flex-1 text-right">
                          {server.remark}
                        </div>
                      )}
                    </div>
                  )}

                  {/* Buttons rack */}
                  <div className="flex gap-2">
                    <Button
                      onClick={() => {
                        setSelectedServerID(server.id);
                        resetNATForm();
                        setIsNATDialogOpen(true);
                      }}
                      variant="secondary"
                      className="flex-1 h-8 rounded-lg flex items-center justify-center gap-1.5 text-[10px]"
                    >
                      <Network className="h-3.5 w-3.5 opacity-70" />
                      <span>NAT 端口</span>
                    </Button>
                    <Button
                      onClick={() => testMutation.mutate(server.id)}
                      variant="secondary"
                      className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                      title="SSH 连通性测试"
                    >
                      <PlugZap className="h-3.5 w-3.5 text-slate-400 hover:text-white transition-colors" />
                    </Button>
                    <Button
                      onClick={() => startEdit(server)}
                      variant="secondary"
                      className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                      title="编辑服务器"
                    >
                      <Pencil className="h-3.5 w-3.5 text-slate-400 hover:text-white transition-colors" />
                    </Button>
                    <Button
                      onClick={() => {
                        if (window.confirm("确定删除这台服务器吗？")) {
                          deleteMutation.mutate(server.id);
                        }
                      }}
                      variant="danger"
                      className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                      title="删除服务器"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}
      </section>

      {/* Add / Edit Server Dialog */}
      <Dialog
        isOpen={isServerDialogOpen}
        onClose={() => {
          setIsServerDialogOpen(false);
          resetForm();
        }}
        title={editingServer ? "编辑服务器" : "添加服务器"}
        size="md"
      >
        <form className="space-y-4" onSubmit={handleSubmit}>
          <Field label="服务器名称">
            <Input
              onChange={(event) =>
                setForm({ ...form, name: event.target.value })
              }
              placeholder="香港 NAT 01"
              required
              value={form.name}
            />
          </Field>
          <div className="grid gap-3 md:grid-cols-[1fr_110px]">
            <Field label="主机地址/IP">
              <Input
                onChange={(event) =>
                  setForm({ ...form, host: event.target.value })
                }
                placeholder="127.0.0.1"
                required
                value={form.host}
              />
            </Field>
            <Field label="SSH 端口">
              <Input
                max={65535}
                min={1}
                onChange={(event) =>
                  setForm({ ...form, sshPort: Number(event.target.value) })
                }
                required
                type="number"
                value={form.sshPort}
              />
            </Field>
          </div>
          <Field label="SSH 用户名">
            <Input
              onChange={(event) =>
                setForm({ ...form, sshUsername: event.target.value })
              }
              required
              value={form.sshUsername}
            />
          </Field>
          <Field label="认证方式">
            <select
              className="h-9 w-full rounded-lg border border-white/[0.06] bg-slate-950 px-3 text-xs text-slate-100 outline-none transition-all duration-300 focus:border-white/20 focus:ring-0 cursor-pointer"
              onChange={(event) =>
                setForm({
                  ...form,
                  authMethod: event.target.value as SSHAuthMethod,
                })
              }
              value={form.authMethod}
            >
              <option value="password">密码</option>
              <option value="private_key">私钥</option>
            </select>
          </Field>
          {form.authMethod === "password" ? (
            <Field label="SSH 密码">
              <Input
                onChange={(event) =>
                  setForm({ ...form, password: event.target.value })
                }
                placeholder={credentialHint || "请输入 SSH 密码"}
                required={!editingServer?.hasPassword}
                type="password"
                value={form.password}
              />
            </Field>
          ) : (
            <Field label="SSH 私钥">
              <textarea
                className="min-h-32 w-full resize-y rounded-lg border border-white/[0.06] bg-slate-950 px-3.5 py-2 font-mono text-xs text-slate-100 outline-none transition-all duration-300 focus:border-white/20"
                onChange={(event) =>
                  setForm({ ...form, privateKey: event.target.value })
                }
                placeholder={credentialHint || "粘贴无 passphrase 的私钥"}
                required={!editingServer?.hasPrivateKey}
                value={form.privateKey}
              />
            </Field>
          )}
          {credentialHint && (
            <p className="text-slate-400 text-[10px] font-semibold">
              {credentialHint}
            </p>
          )}
          <div className="grid gap-3 md:grid-cols-2">
            <Field label="地区">
              <Input
                onChange={(event) =>
                  setForm({ ...form, region: event.target.value })
                }
                placeholder="Hong Kong"
                value={form.region}
              />
            </Field>
            <Field label="标签">
              <Input
                onChange={(event) =>
                  setForm({ ...form, tags: event.target.value })
                }
                placeholder="nat, hk"
                value={form.tags}
              />
            </Field>
          </div>
          <Field label="备注">
            <Input
              onChange={(event) =>
                setForm({ ...form, remark: event.target.value })
              }
              placeholder="可选备注说明"
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
                setIsServerDialogOpen(false);
                resetForm();
              }}
              type="button"
              variant="secondary"
              className="h-9 px-4 text-xs"
            >
              取消
            </Button>
            <Button
              disabled={isSaving}
              type="submit"
              className="h-9 px-4 text-xs font-semibold bg-white text-slate-950 hover:bg-slate-100"
            >
              {isSaving ? "保存中..." : "保存服务器"}
            </Button>
          </div>
        </form>
      </Dialog>

      {/* NAT Mapping Dialog */}
      <Dialog
        isOpen={isNATDialogOpen}
        onClose={() => {
          setIsNATDialogOpen(false);
          setSelectedServerID(null);
          resetNATForm();
        }}
        title={`NAT 端口映射 - ${selectedServer?.name || "加载中..."}`}
        size="wide"
      >
        <div className="grid gap-8 lg:grid-cols-[340px_1fr]">
          {/* Left panel: Add/Edit NAT mapping Form */}
          <div className="border-white/[0.04] lg:border-r lg:pr-8">
            <h3 className="font-bold text-slate-200 text-xs tracking-wider uppercase mb-4">
              {editingMapping ? "编辑映射规则" : "添加映射规则"}
            </h3>
            <form className="space-y-4" onSubmit={handleNATSubmit}>
              <Field label="映射名称">
                <Input
                  onChange={(event) =>
                    setNATForm({ ...natForm, name: event.target.value })
                  }
                  placeholder="AnyTLS 映射"
                  required
                  value={natForm.name}
                />
              </Field>
              <Field label="协议类型">
                <select
                  className="h-9 w-full rounded-lg border border-white/[0.06] bg-slate-950 px-3 text-xs text-slate-100 outline-none transition-all duration-300 focus:border-white/20 focus:ring-0 cursor-pointer"
                  onChange={(event) =>
                    setNATForm({
                      ...natForm,
                      transport: event.target.value,
                    })
                  }
                  value={natForm.transport}
                >
                  <option value="TCP">TCP</option>
                  <option value="UDP">UDP</option>
                </select>
              </Field>
              <div className="grid gap-3 grid-cols-2">
                <Field label="实际监听端口">
                  <Input
                    max={65535}
                    min={1}
                    onChange={(event) =>
                      setNATForm({
                        ...natForm,
                        listenPort: Number(event.target.value),
                      })
                    }
                    required
                    type="number"
                    value={natForm.listenPort}
                  />
                </Field>
                <Field label="对外访问端口">
                  <Input
                    max={65535}
                    min={1}
                    onChange={(event) =>
                      setNATForm({
                        ...natForm,
                        publicPort: Number(event.target.value),
                      })
                    }
                    required
                    type="number"
                    value={natForm.publicPort}
                  />
                </Field>
              </div>
              <Field label="备注">
                <Input
                  onChange={(event) =>
                    setNATForm({ ...natForm, remark: event.target.value })
                  }
                  placeholder="可选"
                  value={natForm.remark}
                />
              </Field>
              {natMessage && (
                <p className="text-slate-300 text-xs font-semibold bg-slate-900 border border-white/[0.04] px-3 py-2 rounded-lg">
                  {natMessage}
                </p>
              )}
              <div className="flex gap-2 pt-1">
                <Button
                  disabled={saveNATMutation.isPending}
                  type="submit"
                  className="flex-1 h-9 text-xs font-semibold bg-white text-slate-950 hover:bg-slate-100"
                >
                  {saveNATMutation.isPending
                    ? "保存中..."
                    : editingMapping
                      ? "保存规则"
                      : "添加映射"}
                </Button>
                {editingMapping && (
                  <Button
                    onClick={resetNATForm}
                    type="button"
                    variant="secondary"
                    className="h-9 text-xs"
                  >
                    取消
                  </Button>
                )}
              </div>
            </form>
          </div>

          {/* Right panel: current NAT mappings list */}
          <div className="min-w-0 flex flex-col">
            <h3 className="font-bold text-slate-200 text-xs tracking-wider uppercase mb-4">
              已映射规则列表
            </h3>
            {natMappingsQuery.isLoading ? (
              <div className="p-8 text-slate-400 text-xs font-semibold animate-pulse">
                正在加载 NAT 映射列表...
              </div>
            ) : natMappings.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-white/[0.04] p-12 text-center text-slate-500 text-xs font-semibold flex-1 flex flex-col items-center justify-center">
                当前服务器还没有 NAT 端口映射规则。请在左侧表单中进行添加。
              </div>
            ) : (
              <div className="overflow-x-auto flex-1">
                <table className="w-full min-w-[480px] border-collapse text-left text-sm">
                  <thead>
                    <tr className="border-white/[0.04] border-b text-slate-400 text-[10px] font-bold uppercase tracking-wider">
                      <th className="py-3 px-4 font-semibold">名称</th>
                      <th className="py-3 px-4 font-semibold">协议</th>
                      <th className="py-3 px-4 font-semibold">映射关系</th>
                      <th className="py-3 px-4 text-right font-semibold">
                        操作
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-white/[0.04]">
                    {natMappings.map((mapping) => (
                      <tr
                        className="hover:bg-white/[0.01] transition-colors duration-200"
                        key={mapping.id}
                      >
                        <td className="py-3 px-4">
                          <div className="font-semibold text-slate-200 text-xs">
                            {mapping.name}
                          </div>
                          {mapping.remark && (
                            <div className="mt-0.5 text-slate-500 text-[9px]">
                              {mapping.remark}
                            </div>
                          )}
                        </td>
                        <td className="py-3 px-4">
                          <Badge className="border-white/[0.04] bg-white/5 text-slate-300 font-mono text-[9px] px-1.5 py-0">
                            {mapping.transport || "TCP"}
                          </Badge>
                        </td>
                        <td className="py-3 px-4 text-slate-300 font-mono text-xs">
                          {mapping.listenPort}{" "}
                          <span className="text-slate-600">→</span>{" "}
                          {mapping.publicPort}
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex justify-end gap-2">
                            <Button
                              onClick={() => startEditMapping(mapping)}
                              variant="secondary"
                              className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                              title="编辑映射"
                            >
                              <Pencil className="h-3.5 w-3.5" />
                            </Button>
                            <Button
                              onClick={() =>
                                window.confirm("确定删除这条 NAT 映射吗？") &&
                                deleteNATMutation.mutate(mapping.id)
                              }
                              variant="danger"
                              className="h-8 w-8 p-0 rounded-lg flex items-center justify-center"
                              title="删除映射"
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </Dialog>
    </div>
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

function StatusBadge({ status }: { status: ServerModel["status"] }) {
  if (status === "normal") {
    return (
      <Badge className="border-emerald-500/10 bg-emerald-500/5 text-emerald-400 font-medium">
        <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-emerald-500 shrink-0 animate-pulse" />
        {statusLabels[status]}
      </Badge>
    );
  }

  if (status === "connection_failed") {
    return (
      <Badge className="border-rose-500/10 bg-rose-500/5 text-rose-400 font-medium">
        <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-rose-500 shrink-0" />
        {statusLabels[status]}
      </Badge>
    );
  }

  return (
    <Badge className="border-slate-800 bg-slate-900/60 text-slate-400 font-medium">
      <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-slate-500 shrink-0" />
      {statusLabels[status]}
    </Badge>
  );
}
