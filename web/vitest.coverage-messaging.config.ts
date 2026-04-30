import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import path from "path"

// Focused coverage config for the message-area decomposition.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: [
      "src/features/messaging/components/__tests__/**/*.test.{ts,tsx}",
      "src/features/messaging/hooks/__tests__/use-message-scroll.test.tsx",
    ],
    coverage: {
      provider: "v8",
      include: [
        "src/features/messaging/components/message-area.tsx",
        "src/features/messaging/components/message-area-utils.ts",
        "src/features/messaging/components/message-bubble.tsx",
        "src/features/messaging/components/text-message-bubble.tsx",
        "src/features/messaging/hooks/use-message-scroll.ts",
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
