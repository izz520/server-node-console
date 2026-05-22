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
}

export interface ProtocolNode {
  id: number;
  name: string;
  protocol: string;
  listenPort?: number;
  publicPort?: number;
  status: NodeStatus;
}

export interface Subscription {
  id: number;
  name: string;
  enabled: boolean;
  format: string;
  nodeCount: number;
}

export interface Task {
  id: number;
  type: "install" | "uninstall" | "ssh_test";
  status: TaskStatus;
  error?: string;
  createdAt: string;
}
