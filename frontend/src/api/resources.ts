import { request } from "@/api/request";
import type {
  ProtocolNode,
  Server,
  SSHAuthMethod,
  Subscription,
  Task,
} from "@/types/domain";

export interface ServerPayload {
  name: string;
  host: string;
  sshPort: number;
  sshUsername: string;
  authMethod: SSHAuthMethod;
  password?: string;
  privateKey?: string;
  region?: string;
  tags?: string;
  remark?: string;
}

export async function listServers() {
  const { data } = await request.get<Server[]>("/servers");
  return data;
}

export async function createServer(payload: ServerPayload) {
  const { data } = await request.post<Server>("/servers", payload);
  return data;
}

export async function updateServer(id: number, payload: ServerPayload) {
  const { data } = await request.put<Server>(`/servers/${id}`, payload);
  return data;
}

export async function deleteServer(id: number) {
  await request.delete(`/servers/${id}`);
}

export async function testServerSSH(id: number) {
  const { data } = await request.post<{ status: string; server: Server }>(
    `/servers/${id}/test-ssh`,
  );
  return data;
}

export async function listNodes() {
  const { data } = await request.get<ProtocolNode[]>("/nodes");
  return data;
}

export async function listSubscriptions() {
  const { data } = await request.get<Subscription[]>("/subscriptions");
  return data;
}

export async function listTasks() {
  const { data } = await request.get<Task[]>("/tasks");
  return data;
}
