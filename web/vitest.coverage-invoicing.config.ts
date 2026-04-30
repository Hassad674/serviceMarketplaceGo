import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import path from "path"

// Focused coverage config for the billing-profile-form decomposition.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: [
      "src/features/invoicing/components/__tests__/billing-profile-form*.test.{ts,tsx}",
      "src/features/invoicing/components/__tests__/billing-section-*.test.{ts,tsx}",
    ],
    coverage: {
      provider: "v8",
      include: [
        "src/features/invoicing/components/billing-profile-form.tsx",
        "src/features/invoicing/components/billing-profile-form.schema.ts",
        "src/features/invoicing/components/billing-section-legal-identity.tsx",
        "src/features/invoicing/components/billing-section-address.tsx",
        "src/features/invoicing/components/billing-section-fiscal.tsx",
      ],
      exclude: ["**/*.d.ts", "**/*.test.*"],
      reporter: ["text"],
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
