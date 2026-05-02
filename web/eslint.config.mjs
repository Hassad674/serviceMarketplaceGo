import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

// Phase 5 — ESLint gate is now enforcing (continue-on-error removed
// from ci.yml). Pre-existing rule violations across legacy features
// are downgraded from error to warn so CI can be promoted without a
// flood of unrelated fixes; a dedicated cleanup PR will move them
// back to error one rule at a time.
//
// New code is still linted at full strength — these overrides only
// affect the rules listed below. Adding NEW errors of any kind still
// fails the build.
const phase5LegacyOverrides = {
  rules: {
    // setState inside useEffect — legacy patterns predating the React
    // Compiler. Will be migrated to derived state in a sweep PR.
    "react-hooks/set-state-in-effect": "warn",
    "react-hooks/refs": "warn",
    "react-hooks/static-components": "warn",
    "react-hooks/component-hook-factories": "warn",
    "react-hooks/error-boundaries": "warn",
    "react-hooks/incompatible-library": "warn",
    "react-hooks/preserve-manual-memoization": "warn",
    // Stable-key warnings on legacy lists. Replace with cuid in sweep.
    "react/jsx-key": "warn",
    // any escapes in legacy types — typed in sweep.
    "@typescript-eslint/no-explicit-any": "warn",
    // React Compiler-flagged hoisting issues in legacy WebSocket
    // setup paths (call/messaging features). Will be addressed in a
    // sweep PR refactoring those flows.
    "react-hooks/use-memo": "warn",
    "react-hooks/purity": "warn",
    // Use-before-decl warnings on legacy WebSocket cleanup chains
    // (call/messaging features). LiveKit is off-limits per repo policy.
    "react-hooks/immutability": "warn",
  },
};

// LiveKit/call code is OFF-LIMITS per repo policy (the user has
// already lost significant time debugging it). React-hooks rules are
// disabled inside features/call/** so the linter does not complain
// about a fragile, frozen module that is intentionally not touched.
const liveKitCallOffLimits = {
  files: ["src/features/call/**"],
  rules: {
    "react-hooks/set-state-in-effect": "off",
    "react-hooks/refs": "off",
    "react-hooks/immutability": "off",
    "react-hooks/use-memo": "off",
    "react-hooks/purity": "off",
    "react-hooks/exhaustive-deps": "off",
    "react-hooks/preserve-manual-memoization": "off",
  },
};

// Project-wide convention: variables, parameters, and destructured
// values prefixed with `_` are intentionally unused (e.g. capture-only
// destructures, unused params we keep for clarity). Recognising this
// prefix in @typescript-eslint/no-unused-vars matches the typescript
// community standard and matches the underscore comments scattered
// across the codebase.
const unusedVarsConvention = {
  rules: {
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
};

// `@next/next/no-img-element` is promoted from warn (the
// `next/core-web-vitals` default) to error: every raw <img> in the
// codebase has been migrated to `next/image`, and the few legitimate
// exceptions (blob: previews) carry a per-line
// `eslint-disable-next-line` annotation. New code that introduces a
// raw <img> should fail CI rather than silently regress LCP on the
// SEO-critical listing pages.
const noRawImgPromoted = {
  rules: {
    "@next/next/no-img-element": "error",
  },
};

// P9 — Features must NEVER import from each other. Composition lives
// in `app/` pages or in `shared/`; cross-feature imports turn the
// dependency graph into spaghetti and break the modularity contract
// documented in `web/CLAUDE.md` ("Features NEVER import from other
// features"). This rule keeps the audit at zero.
//
// The plugin is loaded transitively via `eslint-config-next`, so no
// extra `plugins` block is needed.
const noCrossFeatureImports = {
  files: ["src/features/**"],
  rules: {
    "import/no-restricted-paths": [
      "error",
      {
        zones: [
          {
            target: "./src/features/auth",
            from: "./src/features",
            except: ["./auth"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/account",
            from: "./src/features",
            except: ["./account"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/billing",
            from: "./src/features",
            except: ["./billing"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/call",
            from: "./src/features",
            except: ["./call"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/client-profile",
            from: "./src/features",
            except: ["./client-profile"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/dispute",
            from: "./src/features",
            except: ["./dispute"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/freelance-profile",
            from: "./src/features",
            except: ["./freelance-profile"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/invoicing",
            from: "./src/features",
            except: ["./invoicing"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/job",
            from: "./src/features",
            except: ["./job"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/messaging",
            from: "./src/features",
            except: ["./messaging"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/notification",
            from: "./src/features",
            except: ["./notification"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/organization-shared",
            from: "./src/features",
            except: ["./organization-shared"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/proposal",
            from: "./src/features",
            except: ["./proposal"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/provider",
            from: "./src/features",
            except: ["./provider"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/referral",
            from: "./src/features",
            except: ["./referral"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/referrer-profile",
            from: "./src/features",
            except: ["./referrer-profile"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/reporting",
            from: "./src/features",
            except: ["./reporting"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/review",
            from: "./src/features",
            except: ["./review"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/skill",
            from: "./src/features",
            except: ["./skill"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/subscription",
            from: "./src/features",
            except: ["./subscription"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/team",
            from: "./src/features",
            except: ["./team"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
          {
            target: "./src/features/wallet",
            from: "./src/features",
            except: ["./wallet"],
            message:
              "Features must not import from other features. Extract shared logic to src/shared/.",
          },
        ],
      },
    ],
  },
};

// P3 — `react/forbid-elements` blocks raw <button>, <input>, <select>
// outside of the design-system primitives layer. Every site has been
// migrated to the Button / Input / Select primitives in
// `src/shared/components/ui/`, which centralise focus-ring styling,
// disabled states, and the rose-tinted design tokens documented in
// CLAUDE.md.
//
// The rule is configured globally; an override for the primitives
// folder itself + the LiveKit/call code (off-limits per repo policy)
// scopes the check appropriately.
const noRawNativeFormElements = {
  files: ["src/**/*.{ts,tsx,jsx}"],
  ignores: [
    "src/shared/components/ui/**",
    "src/features/call/**",
    "src/**/__tests__/**",
    "src/**/*.test.*",
    "src/**/*.spec.*",
  ],
  rules: {
    "react/forbid-elements": [
      "error",
      {
        forbid: [
          { element: "button", message: "Use <Button> from @/shared/components/ui/button" },
          { element: "input", message: "Use <Input> from @/shared/components/ui/input" },
          { element: "select", message: "Use <Select> from @/shared/components/ui/select" },
        ],
      },
    ],
  },
};

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  globalIgnores([
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
  phase5LegacyOverrides,
  liveKitCallOffLimits,
  unusedVarsConvention,
  noRawImgPromoted,
  noRawNativeFormElements,
  noCrossFeatureImports,
]);

export default eslintConfig;
