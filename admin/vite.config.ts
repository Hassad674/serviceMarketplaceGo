import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"
import path from "node:path"

// ADMIN-PERF-01 + ADMIN-PERF-NEW-19 — Vite tuning.
//
//   - `manualChunks` splits the heaviest vendor libraries into
//     dedicated chunks so a routine app rebuild doesn't force every
//     user to re-download recharts (~500 KB) or react-vendor.
//   - `target: "es2022"` keeps the output unminified-friendly for
//     modern admin browsers and trims the polyfill surface.
//   - `chunkSizeWarningLimit: 500` raises the warning threshold so
//     the build doesn't false-alarm on the recharts split (which is
//     intentional and bound by the manualChunks rule).
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  server: {
    port: 5174,
  },
  build: {
    target: "es2022",
    chunkSizeWarningLimit: 500,
    rollupOptions: {
      output: {
        manualChunks: {
          "react-vendor": ["react", "react-dom", "react-router-dom"],
          tanstack: ["@tanstack/react-query", "@tanstack/react-table"],
          charts: ["recharts"],
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
    css: false,
  },
})
