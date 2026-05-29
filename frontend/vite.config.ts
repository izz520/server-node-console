import { fileURLToPath, URL } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const apiProxyTarget = process.env.API_PROXY_TARGET || "http://localhost:8080";
const previewAllowedHosts = (
  process.env.PREVIEW_ALLOWED_HOSTS ||
  "server.995858.xyz,server2.995858.xyz,server.yasol.me"
)
  .split(",")
  .map((host) => host.trim())
  .filter(Boolean);
const apiProxy = {
  "/api": {
    target: apiProxyTarget,
    changeOrigin: true,
  },
  "/sub/": {
    target: apiProxyTarget,
    changeOrigin: true,
  },
};

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: apiProxy,
  },
  preview: {
    allowedHosts: previewAllowedHosts,
    proxy: apiProxy,
  },
});
