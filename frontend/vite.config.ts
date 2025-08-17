import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  server: {
    allowedHosts: true,
    proxy: {
      "/api": {
        target: process.env.BACKEND_TARGET || "http://localhost:8080",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, "/v1"),
      },
    },
    cors: true,
    headers: {
      "Access-Control-Allow-Origin": "*",
    },
    hmr: {
      overlay: false,
    },
  },
  css: {
    postcss: "./postcss.config.js",
  },
  plugins: [react(), tailwindcss()],
});
