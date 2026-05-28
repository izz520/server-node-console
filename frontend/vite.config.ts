import { fileURLToPath, URL } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const apiProxyTarget = process.env.API_PROXY_TARGET || "http://localhost:8080";
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
    allowedHosts: ["server.995858.xyz", "server.yasol.me"],
    proxy: apiProxy,
  },
});
