export type UserRole = "user" | "admin";

export type ServerStatus = "normal" | "connection_failed" | "disabled";

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
}

export interface Server {
  id: number;
  name: string;
  host: string;
  sshPort: number;
  sshUsername: string;
  status: ServerStatus;
  region?: string;
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
