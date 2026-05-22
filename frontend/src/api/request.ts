import axios from "axios";
import { API_BASE_URL } from "@/constants/config";
import { useAuthStore } from "@/stores/auth";

export const request = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30_000,
});

request.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

request.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().clearSession();
    }
    return Promise.reject(error);
  },
);
