import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import path from "path"

// Aggregate coverage across all 4 god-component decompositions.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: [
      "src/features/wallet/components/__tests__/**/*.test.{ts,tsx}",
      "src/features/messaging/components/__tests__/**/*.test.{ts,tsx}",
      "src/features/messaging/hooks/__tests__/use-message-scroll.test.tsx",
      "src/shared/components/search/__tests__/**/*.test.{ts,tsx}",
      "src/features/invoicing/components/__tests__/billing-profile-form*.test.{ts,tsx}",
      "src/features/invoicing/components/__tests__/billing-section-*.test.{ts,tsx}",
    ],
    coverage: {
      provider: "v8",
      include: [
        // Wallet
        "src/features/wallet/components/wallet-*.tsx",
        // Messaging
        "src/features/messaging/components/message-area.tsx",
        "src/features/messaging/components/message-area-utils.ts",
        "src/features/messaging/components/message-bubble.tsx",
        "src/features/messaging/components/text-message-bubble.tsx",
        "src/features/messaging/hooks/use-message-scroll.ts",
        // Search
        "src/shared/components/search/search-filter-sidebar.tsx",
        "src/shared/components/search/filter-primitives.tsx",
        "src/shared/components/search/filter-section-availability.tsx",
        "src/shared/components/search/filter-section-pricing.tsx",
        "src/shared/components/search/filter-section-location.tsx",
        "src/shared/components/search/filter-section-skills-expertise.tsx",
        "src/shared/components/search/filter-section-rating.tsx",
        // Invoicing
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
