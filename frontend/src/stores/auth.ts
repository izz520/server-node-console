import { create } from "zustand";
import type { User } from "@/types/domain";

interface AuthState {
  token: string | null;
  expiresAt: string | null;
  user: User | null;
  setSession: (token: string, expiresAt: string, user: User) => void;
  setUser: (user: User) => void;
  clearSession: () => void;
}

const TOKEN_KEY = "singbox_manager_token";
const EXPIRES_AT_KEY = "singbox_manager_expires_at";

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem(TOKEN_KEY),
  expiresAt: localStorage.getItem(EXPIRES_AT_KEY),
  user: null,
  setSession: (token, expiresAt, user) => {
    localStorage.setItem(TOKEN_KEY, token);
    localStorage.setItem(EXPIRES_AT_KEY, expiresAt);
    set({ token, expiresAt, user });
  },
  setUser: (user) => {
    set({ user });
  },
  clearSession: () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(EXPIRES_AT_KEY);
    set({ token: null, expiresAt: null, user: null });
  },
}));
