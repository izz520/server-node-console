export type UserRole = "user" | "admin";

export type ServerStatus = "normal" | "connection_failed" | "disabled";
export type SSHAuthMethod = "password" | "private_key";

export type NodeStatus =
  | "installing"
  | "install_success"
  | "install_failed"
  | "uninstalling"
  | "uninstalled"
  | "imported";

export type TaskStatus = "queued" | "running" | "success" | "failed";

export interface User {
  id: number;
  username: string;
  email: string;
  role: UserRole;
  createdAt?: string;
}

export interface Server {
  id: number;
  userId: number;
  name: string;
  host: string;
  sshPort: number;
  sshUsername: string;
  authMethod: SSHAuthMethod;
  status: ServerStatus;
  region?: string;
  tags?: string;
  remark?: string;
  hasPassword: boolean;
  hasPrivateKey: boolean;
  lastCheckedAt?: string | null;
  expiresAt?: string | null;
  price?: number;
  billingCycle?: string;
  currency?: string;
}

export interface NATPortMapping {
  id: number;
  serverId: number;
  name: string;
  transport?: string;
  listenPort: number;
  publicPort: number;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ProtocolNode {
  id: number;
  userId: number;
  serverId?: number | null;
  name: string;
  protocol: string;
  address: string;
  port: number;
  listenPort?: number;
  publicPort?: number;
  remark?: string;
  installMethod: "system" | "external";
  status: NodeStatus;
  hasSensitive: boolean;
  chainProxyNodeId?: number | null;
}

export interface Subscription {
  id: number;
  userId: number;
  name: string;
  enabled: boolean;
  format: string;
  clashTemplate?: string;
  clashTemplateId?: number | null;
  nodeIds: number[];
  nodeCount: number;
  token?: string;
  subscriptionUrl?: string;
  remark?: string;
}

export interface ClashTemplate {
  id: number;
  userId: number;
  name: string;
  content: string;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Task {
  id: number;
  userId: number;
  type: "install" | "uninstall" | "ssh_test";
  status: TaskStatus;
  error?: string;
  serverId?: number | null;
  nodeId?: number | null;
  startedAt?: string | null;
  endedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface TaskLog {
  id: number;
  taskId: number;
  level: string;
  message: string;
  createdAt: string;
}

export interface TaskDetail {
  task: Task;
  logs: TaskLog[];
}

export interface OperationLog {
  id: number;
  userId?: number | null;
  action: string;
  resource: string;
  metadata: string;
  createdAt: string;
}
