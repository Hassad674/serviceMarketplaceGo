import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import path from "path"

// Focused coverage config for the search-filter-sidebar decomposition.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/shared/components/search/__tests__/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      include: [
        "src/shared/components/search/search-filter-sidebar.tsx",
        "src/shared/components/search/filter-primitives.tsx",
        "src/shared/components/search/filter-section-availability.tsx",
        "src/shared/components/search/filter-section-pricing.tsx",
        "src/shared/components/search/filter-section-location.tsx",
        "src/shared/components/search/filter-section-skills-expertise.tsx",
        "src/shared/components/search/filter-section-rating.tsx",
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
