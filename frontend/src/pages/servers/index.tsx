import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckCircle2,
  Pencil,
  PlugZap,
  Plus,
  Server,
  Trash2,
  XCircle,
} from "lucide-react";
import { type FormEvent, useMemo, useState } from "react";
import {
  createServer,
  deleteServer,
  listServers,
  type ServerPayload,
  testServerSSH,
  updateServer,
} from "@/api/resources";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { Server as ServerModel, SSHAuthMethod } from "@/types/domain";

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
  normal: "正常",
  connection_failed: "连接失败",
  disabled: "禁用",
};

export function ServersPage() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<ServerPayload>(emptyForm);
  const [editingServer, setEditingServer] = useState<ServerModel | null>(null);
  const [message, setMessage] = useState("");

  const serversQuery = useQuery({
    queryKey: ["servers"],
    queryFn: listServers,
  });

  const saveMutation = useMutation({
    mutationFn: (payload: ServerPayload) =>
      editingServer
        ? updateServer(editingServer.id, payload)
        : createServer(payload),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["servers"] });
      resetForm();
      setMessage("服务器已保存");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "保存失败，请检查 SSH 连接信息"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteServer,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["servers"] });
      setMessage("服务器已删除");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "删除失败"));
    },
  });

  const testMutation = useMutation({
    mutationFn: testServerSSH,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["servers"] });
      setMessage("SSH 连通性测试成功");
    },
    onError: (error) => {
      setMessage(getErrorMessage(error, "SSH 连通性测试失败"));
      queryClient.invalidateQueries({ queryKey: ["servers"] });
    },
  });

  const servers = serversQuery.data ?? [];
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
  }

  return (
    <div className="grid gap-6 xl:grid-cols-[420px_1fr]">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-slate-950 text-white">
              <Server className="h-4 w-4" />
            </div>
            <div>
              <h1 className="font-semibold text-slate-950 text-xl">
                {editingServer ? "编辑服务器" : "添加服务器"}
              </h1>
              <p className="text-slate-500 text-sm">
                保存前会自动测试 SSH 连通性
              </p>
            </div>
          </div>
        </CardHeader>
        <CardContent>
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
                className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
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
                  className="min-h-32 w-full resize-y rounded-md border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 focus:ring-2 focus:ring-slate-100"
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
              <p className="text-slate-500 text-xs">{credentialHint}</p>
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
                placeholder="可选"
                value={form.remark}
              />
            </Field>
            {message && <p className="text-slate-600 text-sm">{message}</p>}
            <div className="flex gap-2">
              <Button disabled={isSaving} type="submit">
                <Plus className="h-4 w-4" />
                {isSaving
                  ? "保存中..."
                  : editingServer
                    ? "保存修改"
                    : "添加服务器"}
              </Button>
              {editingServer && (
                <Button onClick={resetForm} type="button" variant="secondary">
                  取消
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="font-medium text-slate-950">服务器列表</div>
        </CardHeader>
        <CardContent>
          {serversQuery.isLoading ? (
            <div className="text-slate-500 text-sm">加载中...</div>
          ) : servers.length === 0 ? (
            <div className="rounded-md border border-dashed border-slate-200 p-8 text-center text-slate-500 text-sm">
              还没有服务器，先添加一台用于后续安装协议节点。
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[760px] border-collapse text-left text-sm">
                <thead>
                  <tr className="border-slate-100 border-b text-slate-500">
                    <th className="py-3 pr-3 font-medium">名称</th>
                    <th className="py-3 pr-3 font-medium">SSH</th>
                    <th className="py-3 pr-3 font-medium">认证</th>
                    <th className="py-3 pr-3 font-medium">状态</th>
                    <th className="py-3 pr-3 font-medium">地区</th>
                    <th className="py-3 pr-3 text-right font-medium">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {servers.map((server) => (
                    <tr className="border-slate-100 border-b" key={server.id}>
                      <td className="py-3 pr-3">
                        <div className="font-medium text-slate-950">
                          {server.name}
                        </div>
                        {server.tags && (
                          <div className="mt-1 text-slate-500 text-xs">
                            {server.tags}
                          </div>
                        )}
                      </td>
                      <td className="py-3 pr-3 text-slate-700">
                        {server.sshUsername}@{server.host}:{server.sshPort}
                      </td>
                      <td className="py-3 pr-3 text-slate-700">
                        {server.authMethod === "password" ? "密码" : "私钥"}
                      </td>
                      <td className="py-3 pr-3">
                        <StatusBadge status={server.status} />
                      </td>
                      <td className="py-3 pr-3 text-slate-700">
                        {server.region || "-"}
                      </td>
                      <td className="py-3 pr-3">
                        <div className="flex justify-end gap-2">
                          <Button
                            onClick={() => testMutation.mutate(server.id)}
                            title="测试 SSH"
                            variant="secondary"
                          >
                            <PlugZap className="h-4 w-4" />
                          </Button>
                          <Button
                            onClick={() => startEdit(server)}
                            title="编辑"
                            variant="secondary"
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            onClick={() => deleteMutation.mutate(server.id)}
                            title="删除"
                            variant="danger"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
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

function StatusBadge({ status }: { status: ServerModel["status"] }) {
  if (status === "normal") {
    return (
      <Badge className="border-emerald-200 bg-emerald-50 text-emerald-700">
        <CheckCircle2 className="mr-1 h-3 w-3" />
        {statusLabels[status]}
      </Badge>
    );
  }

  if (status === "connection_failed") {
    return (
      <Badge className="border-red-200 bg-red-50 text-red-700">
        <XCircle className="mr-1 h-3 w-3" />
        {statusLabels[status]}
      </Badge>
    );
  }

  return <Badge>{statusLabels[status]}</Badge>;
}

function getErrorMessage(error: unknown, fallback: string) {
  if (typeof error === "object" && error && "response" in error) {
    const response = (
      error as { response?: { data?: { details?: string; error?: string } } }
    ).response;
    return response?.data?.details ?? response?.data?.error ?? fallback;
  }
  return fallback;
}
