const configuredApiBaseUrl = import.meta.env.VITE_API_BASE_URL?.toString().trim();

export const API_BASE_URL =
  configuredApiBaseUrl ||
  `${window.location.protocol}//${window.location.hostname}:8080/api/v1`;

export const APP_NAME = "sing-box 节点管理平台";
