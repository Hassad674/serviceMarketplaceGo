# F.3.2 — DRY web cleanup (gated on backend OpenAPI exposure)

**Phase:** F.3.2 — post-publish OR before-publish polish (user choice)
**Source audit:** PR #92 final verification — top-10% DRY due to web regression
**Effort:** 1-1.5j est. (backend OpenAPI ~½j + web sweep ~1j)
**Tool:** 1 fresh agent dispatched (after user validates)
**Branch:** `feat/f3-2-dry-web-cleanup`

## Problem

Final audit (PR #92) flagged **DRY top-10%** because of:
- **467 hardcoded `apiClient<T>("/api/v1/...")` literal paths** in `web/src/` (regression vs the audited 96)
- 311 similar in `mobile/lib/` (lower priority — mobile has typed Dio per-feature)
- 3 pre-existing ESLint errors in `web/src/features/` (display-name + forbid-elements)

The fix is **gated on the backend exposing OpenAPI 3.1** at `/api/openapi.json` so `npm run generate-api` can produce typed `paths` from the schema.

## Plan (3 stages)

### Stage 1 — Backend OpenAPI exposure (~½j)

Decision (LOCKED — agent must follow):
- Use **chi-router introspection** if available (e.g. via `chi.Walk` + handler annotations) OR a simple hand-built schema in `internal/handler/openapi.go`
- Reject swaggo/swag (heavy, annotation-driven, drift-prone)
- Endpoint: `GET /api/openapi.json` returns OpenAPI 3.1 schema (public, no auth) describing every public + authenticated route
- Schema includes: paths, methods, request bodies, response shapes (DTOs), error envelope contract, security schemes (bearer + cookie)
- Use `getkin/kin-openapi` Go lib for type-safe construction OR build the schema from chi.Walk + manually-curated DTO types
- Snapshot test: `internal/handler/openapi_test.go::TestOpenAPISchemaShape` asserts a stable shape (regenerate snapshot when intentional changes happen)

### Stage 2 — Web `npm run generate-api` (~1h)

- Update `web/package.json::scripts.generate-api` to point at `http://localhost:8083/api/openapi.json` (NOT 8080 — port confusion previously)
- Run `npm run generate-api` → produces `web/src/shared/types/api.d.ts`
- Commit `api.d.ts` (override the gitignore line — convention shift like P12 mobile artefacts)
- Document in `web/CLAUDE.md` that the file is regenerated on demand and committed

### Stage 3 — Web sweep `apiClient<T>(path)` → `apiClient<paths[X]>(path)` (~1j)

- For each of the 467 sites: replace string-literal path with `paths["/api/v1/..."]` typed call
- ESLint rule `@typescript-eslint/no-explicit-any` set to `error` on all `apiClient` call sites (drives the migration)
- For sites where the typed shape causes a real type mismatch (drift between BE and FE): document, fix (probably a stale FE expectation), verify
- 3 pre-existing ESLint errors fixed in passing if trivial (else flag for F.4)

### Stage 4 — Mobile follow-up (skipped or F.3.3)

311 mobile API calls — mobile uses Dio with per-feature typed services. Already typed at the Dart level. Less urgent. Stays in F.3.3 if at all.

## Hard constraints

- **Validation pipeline before EVERY commit**:
  ```bash
  cd backend && go build ./... && go vet ./... && go test ./... -count=1 -short
  cd web && npx tsc --noEmit && npx vitest run
  ```
- **Zero behaviour change** on the API surface. Pure typing migration on web.
- **No new dependencies** unless absolutely required (kin-openapi if chosen)
- **OpenAPI schema is the source of truth** — drift between BE and FE = breaking change visible in CI

## Out-of-scope flags

- ESLint pre-existing 3 errors → fix if trivial during web sweep, else F.4
- Mobile API typing → F.3.3 if user wants

## Branch ownership

`feat/f3-2-dry-web-cleanup` only.

## Final report

Lead with PR URL.
1. Backend OpenAPI endpoint live (yes/no, schema size)
2. Web `api.d.ts` generated + committed (yes)
3. Sites migrated 467 → 0
4. ESLint errors before/after
5. Validation pipeline output
6. "Branch ownership confirmed"
