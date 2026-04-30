import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import path from "path"

// Focused coverage config for the recent-PR surface in web/src/shared:
// json-ld, upload-api, search-api, expertise/city/skeleton extracted
// modules, and use-search hook. Used by the test-coverage agent to
// pick the exact files that need a coverage push without dragging in
// unrelated app code.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: [
      "src/shared/lib/__tests__/**/*.test.{ts,tsx}",
      "src/shared/lib/search/__tests__/**/*.test.{ts,tsx}",
      "src/shared/lib/location/**/*.test.{ts,tsx}",
      "src/shared/components/expertise/__tests__/**/*.test.{ts,tsx}",
      "src/shared/components/location/__tests__/**/*.test.{ts,tsx}",
      "src/shared/components/ui/__tests__/**/*.test.{ts,tsx}",
    ],
    coverage: {
      provider: "v8",
      include: [
        "src/shared/lib/json-ld.ts",
        "src/shared/lib/upload-api.ts",
        "src/shared/lib/search/search-api.ts",
        "src/shared/lib/location/city-search.ts",
        "src/shared/components/expertise/expertise-editor.tsx",
        "src/shared/components/location/city-autocomplete.tsx",
        "src/shared/components/ui/skeleton-block.tsx",
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
