import { request } from "@/api/request";
import type { User } from "@/types/domain";

export interface LoginPayload {
  account: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export async function login(payload: LoginPayload) {
  const { data } = await request.post<LoginResponse>("/auth/login", payload);
  return data;
}

export async function getMe() {
  const { data } = await request.get<User>("/me");
  return data;
}
