import { request } from "@/api/request";
import type { ProtocolNode, Server, Subscription, Task } from "@/types/domain";

export async function listServers() {
  const { data } = await request.get<Server[]>("/servers");
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
