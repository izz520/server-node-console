const configuredApiBaseUrl = import.meta.env.VITE_API_BASE_URL?.toString().trim();

export const API_BASE_URL = configuredApiBaseUrl || "/api/v1";

export const APP_NAME = "sing-box 节点管理平台";
