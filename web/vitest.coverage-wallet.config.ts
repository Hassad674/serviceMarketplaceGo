import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import path from "path"

// Focused coverage config for the wallet decomposition (Phase 3).
// Restricts vitest to the wallet test suite AND restricts the v8
// coverage report to the new wallet sub-components — gives us a
// clean, readable per-file coverage table to gate on the ≥90%
// threshold called out in the agent brief.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/features/wallet/components/__tests__/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      include: ["src/features/wallet/components/wallet-*.tsx"],
      exclude: ["**/*.d.ts", "**/*.test.*"],
      reporter: ["text", "text-summary"],
      reportOnFailure: true,
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@i18n": path.resolve(__dirname, "./i18n"),
    },
  },
})
