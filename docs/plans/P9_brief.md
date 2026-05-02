# P9 — Web cleanup : 33 cross-feature imports + 96 hardcoded API paths → typed client

**Phase:** F.2 HIGH #5
**Source audit:** DRY-FINAL + ARCH-FINAL (`auditqualite.md`, `auditperf.md`)
**Effort:** 2j est.
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p9-web-imports-and-typed-api`

## Problem

Two architectural violations on web/:

### A. Cross-feature imports (33 violations)
Per `web/CLAUDE.md`: "Features NEVER import from other features. Composition happens in `app/` pages only." 33 sites violate this rule. Examples (run discovery to find exact list):
- `features/proposal/components/X.tsx` imports from `features/messaging/...`
- `features/dispute/...` imports from `features/proposal/...`

### B. 96 hardcoded API paths
Sites use `apiClient<T>("/api/v1/proposals/...")` with literal string paths. Should use the typed `apiClient` with auto-generated OpenAPI types. The types file is `src/shared/types/api.d.ts` (gitignored, generated via `npm run generate-api`).

## Decision (LOCKED — user validated)

### A. Cross-feature imports
- Extract shared logic to `shared/` modules (`shared/types/`, `shared/lib/`, `shared/components/`)
- ESLint rule `import/no-restricted-paths` blocks future violations: zone `features/X` → cannot import from `features/Y`

### B. Typed API paths
- Migrate all sites from string-literal paths to typed `apiClient<paths["/api/v1/...]"]>(path)` pattern
- The 96 sites stay as call sites — only the typing improves
- Re-generate api.d.ts (`npm run generate-api`) to ensure latest

## Plan (4 commits)

### Commit 1 — Inventory + ESLint rule
- Run discovery: `grep -rn "from \"@/features/" web/src/features/` → list cross-feature imports per pair
- Run discovery: `grep -rEn "apiClient.*\"/api/v1/" web/src/` → list 96 hardcoded paths per file
- Add ESLint config block:
  ```js
  "import/no-restricted-paths": ["error", {
    zones: [{
      target: "./src/features/A",
      from: "./src/features/B",
      message: "..."
    }, ...]
  }]
  ```
- Tests

### Commit 2 — Cross-feature imports cleanup
- For each violation:
  - Identify shared logic (type, helper, hook, component)
  - Extract to `shared/<category>/<name>`
  - Update both sides to import from `shared/`
- Re-run ESLint to verify 0 violations
- Tests still pass

### Commit 3 — Typed API client migration
- For each file with hardcoded `/api/v1/...` strings:
  - Replace with `apiClient<paths["..."]>(path)` typed call
  - Verify TypeScript compile catches any drift
- Re-generate `api.d.ts` if backend OpenAPI has new endpoints since last regen
- Tests

### Commit 4 — Lint config strict + docs
- Promote ESLint `import/no-restricted-paths` from `warn` to `error` (was already error if step 1 done well)
- Document the convention in `web/CLAUDE.md` if not already
- Tests

## Hard constraints

- **Validation pipeline before EVERY commit**: `npx tsc --noEmit && npx vitest run --changed && npx eslint src/`
- **Zero behaviour change**: pure refactor + typing. UI byte-identical.
- **No new components / no new features** — extract existing only.

## OFF-LIMITS

- LiveKit / call code: `web/src/features/call/` — never touch
- `.github/workflows/*` — token can't push
- Backend / mobile / admin — out of scope
- Other plans

## Branch ownership

`fix/p9-web-imports-and-typed-api` only. Created from main.

## Final report (under 700 words)

PR URL first. Then:
1. Cross-feature imports : 33 → 0 (with shared/ extractions count)
2. Hardcoded API paths : 96 → 0 (typed client adoption)
3. ESLint rule active
4. Validation pipeline output
5. Branch ownership confirmed
