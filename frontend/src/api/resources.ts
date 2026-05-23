import { request } from "@/api/request";
import type {
  NATPortMapping,
  OperationLog,
  ProtocolNode,
  Server,
  SSHAuthMethod,
  Subscription,
  Task,
  TaskDetail,
  User,
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

export interface NATMappingPayload {
  name: string;
  transport?: string;
  listenPort: number;
  publicPort: number;
  remark?: string;
}

export interface NodeImportPayload {
  mode: "manual" | "link";
  name?: string;
  protocol?: string;
  address?: string;
  port?: number;
  listenPort?: number;
  publicPort?: number | null;
  rawLink?: string;
  remark?: string;
  configJson?: string;
  sensitive?: string;
  displayName?: string;
}

export interface NodeInstallPayload {
  serverId: number;
  name: string;
  protocol: string;
  port?: number;
  uuid?: string;
  realityDomain?: string;
  cdnDomain?: string;
  argoMode?: string;
  argoDomain?: string;
  argoToken?: string;
  namePrefix?: string;
  remark?: string;
}

export interface NodeUpdatePayload {
  name: string;
  protocol: string;
  address: string;
  port: number;
  listenPort?: number;
  publicPort?: number | null;
  remark?: string;
  configJson?: string;
  sensitive?: string;
}

export interface SubscriptionPayload {
  name: string;
  format: string;
  enabled: boolean;
  nodeIds: number[];
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

export async function listNATMappings(serverID: number) {
  const { data } = await request.get<NATPortMapping[]>(
    `/servers/${serverID}/nat-mappings`,
  );
  return data;
}

export async function createNATMapping(
  serverID: number,
  payload: NATMappingPayload,
) {
  const { data } = await request.post<NATPortMapping>(
    `/servers/${serverID}/nat-mappings`,
    payload,
  );
  return data;
}

export async function updateNATMapping(id: number, payload: NATMappingPayload) {
  const { data } = await request.put<NATPortMapping>(
    `/nat-mappings/${id}`,
    payload,
  );
  return data;
}

export async function deleteNATMapping(id: number) {
  await request.delete(`/nat-mappings/${id}`);
}

export async function listNodes() {
  const { data } = await request.get<ProtocolNode[]>("/nodes");
  return data;
}

export async function importNode(payload: NodeImportPayload) {
  const { data } = await request.post<ProtocolNode>("/nodes/import", payload);
  return data;
}

export async function installNode(payload: NodeInstallPayload) {
  const { data } = await request.post<{ node: ProtocolNode; task: Task }>(
    "/nodes/install",
    payload,
  );
  return data;
}

export async function uninstallNode(id: number) {
  const { data } = await request.post<{ node: ProtocolNode; task: Task }>(
    `/nodes/${id}/uninstall`,
  );
  return data;
}

export async function updateNode(id: number, payload: NodeUpdatePayload) {
  const { data } = await request.put<ProtocolNode>(`/nodes/${id}`, payload);
  return data;
}

export async function deleteNode(id: number) {
  await request.delete(`/nodes/${id}`);
}

export async function listSubscriptions() {
  const { data } = await request.get<Subscription[]>("/subscriptions");
  return data;
}

export async function createSubscription(payload: SubscriptionPayload) {
  const { data } = await request.post<Subscription>("/subscriptions", payload);
  return data;
}

export async function updateSubscription(
  id: number,
  payload: SubscriptionPayload,
) {
  const { data } = await request.put<Subscription>(
    `/subscriptions/${id}`,
    payload,
  );
  return data;
}

export async function deleteSubscription(id: number) {
  await request.delete(`/subscriptions/${id}`);
}

export async function resetSubscriptionToken(id: number) {
  const { data } = await request.post<Subscription>(
    `/subscriptions/${id}/reset-token`,
  );
  return data;
}

export async function listTasks() {
  const { data } = await request.get<Task[]>("/tasks");
  return data;
}

export async function getTask(id: number) {
  const { data } = await request.get<TaskDetail>(`/tasks/${id}`);
  return data;
}

export async function listOperationLogs() {
  const { data } = await request.get<OperationLog[]>("/operation-logs");
  return data;
}

export async function listAdminUsers() {
  const { data } = await request.get<User[]>("/admin/users");
  return data;
}

export async function listAdminServers() {
  const { data } = await request.get<Server[]>("/admin/servers");
  return data;
}

export async function listAdminNodes() {
  const { data } = await request.get<ProtocolNode[]>("/admin/nodes");
  return data;
}

export async function listAdminSubscriptions() {
  const { data } = await request.get<Subscription[]>("/admin/subscriptions");
  return data;
}

export async function listAdminTasks() {
  const { data } = await request.get<Task[]>("/admin/tasks");
  return data;
}

export async function getAdminTask(id: number) {
  const { data } = await request.get<TaskDetail>(`/admin/tasks/${id}`);
  return data;
}

export async function listAdminOperationLogs() {
  const { data } = await request.get<OperationLog[]>("/admin/operation-logs");
  return data;
}
