import path from "node:path";

import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const projectDirectory = import.meta.dirname;

export default defineConfig({
  plugins: [
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
    }),
    react({
      jsxRuntime: "automatic",
    }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@components": path.resolve(projectDirectory, "./src/components"),
      "@features": path.resolve(projectDirectory, "./src/features"),
      "@hooks": path.resolve(projectDirectory, "./src/hooks"),
      "@lib": path.resolve(projectDirectory, "./src/lib"),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": {
        changeOrigin: true,
        target: "http://localhost:8080",
      },
    },
  },
});
