import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    // Embedded by internal/web via go:embed all:dist
    outDir: "../internal/web/dist",
    emptyOutDir: true,
    sourcemap: true,
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:7432",
      "/health": "http://127.0.0.1:7432",
    },
  },
});
