// ESLint 9 flat config for the admin app (Vite + React + TypeScript).
//
// Scope is intentionally minimal: TypeScript recommendations + React-Hooks
// rules + React-Refresh (Vite-friendly HMR guard). The repo's project-wide
// convention recognises `_`-prefixed identifiers as intentionally unused —
// mirrored here from `web/eslint.config.mjs` so behaviour stays consistent
// across apps.
//
// Generated artefacts (`dist/`, `coverage/`, `node_modules/`) and the
// committed OpenAPI types (`src/shared/types/api.d.ts`) are ignored
// globally — they are produced by build tooling and should not be linted.

import js from "@eslint/js"
import globals from "globals"
import reactHooks from "eslint-plugin-react-hooks"
import reactRefresh from "eslint-plugin-react-refresh"
import tseslint from "typescript-eslint"

export default tseslint.config(
  {
    ignores: [
      "dist/**",
      "coverage/**",
      "node_modules/**",
      "src/shared/types/api.d.ts",
    ],
  },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommended,
    ],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: {
        ...globals.browser,
        ...globals.es2021,
      },
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
      // Project-wide: identifiers prefixed with `_` are intentionally
      // unused (capture-only destructures, kept-for-clarity params).
      // Matches the convention used in `web/eslint.config.mjs`.
      "@typescript-eslint/no-unused-vars": [
        "warn",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          caughtErrorsIgnorePattern: "^_",
          destructuredArrayIgnorePattern: "^_",
        },
      ],
    },
  },
  {
    // Test files use Vitest globals (describe/it/expect) and may include
    // intentional `any` for fixture helpers. Loosen the rules accordingly.
    files: [
      "**/*.test.{ts,tsx}",
      "**/*.spec.{ts,tsx}",
      "**/__tests__/**/*.{ts,tsx}",
      "src/setupTests.ts",
    ],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
    },
  },
)
