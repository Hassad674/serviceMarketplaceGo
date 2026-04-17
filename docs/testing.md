# Testing strategy

This document is the single source of truth for how we verify the marketplace. It
describes every test layer, how to run each one locally, how CI invokes them, the
coverage we commit to, and the cost envelope for any test that touches a paid
third-party API.

---

## 1. Philosophy

Three rules govern every decision in this document.

1. **Mock-first.** 95% of our tests run without the network. A deterministic
   `MockEmbeddingsClient` replaces OpenAI, an in-memory store replaces the
   database for unit tests, and a `httptest.NewServer` replaces any HTTP
   upstream. Fast feedback is more important than "real" coverage — if a
   change passes 500 unit tests in 30 seconds, regressions surface before
   the PR is even reviewed.

2. **Live-sparingly.** 5% of our tests hit real external systems — OpenAI
   for embedding quality, a real Typesense container for schema validation,
   a real Postgres container for RLS policies. These tests are expensive
   (cost, time, flakiness) and run only when (a) a human labels the PR with
   `run-e2e` OR (b) the change lands on main.

3. **Paranoid on critical paths.** Security, authorization, and ranking
   logic get table-driven tests covering every documented invariant AND
   adversarial inputs. A regression in the scoped-key generator that lets
   persona A see persona B's results is a fireable offence — we catch it
   with leak tests, not trust.

> Why: the user asked for max coverage so agents can run autonomously for
> hours and self-correct via test failures. Without this discipline, long
> autonomous sessions drift silently.

---

## 2. Test layers

| Layer | Tool | Scope | Local command | CI workflow | Cadence | Coverage | Owner |
|-------|------|-------|---------------|-------------|---------|----------|-------|
| Unit (Go) | `go test` + testify | `internal/**/service_test.go`, `domain/**`, `pkg/**` | `cd backend && go test ./... -count=1` | `ci.yml` → backend-test | every PR | 85%+ on `internal/search`, `internal/app/search*`, `internal/app/searchanalytics` | CI |
| Unit (Web) | vitest + @testing-library | `web/src/**/*.test.ts(x)` | `cd web && npx vitest run` | `ci.yml` → web-test | every PR | 80%+ on new files | CI |
| Unit (Mobile) | `flutter test` | `mobile/test/**/*_test.dart` | `cd mobile && flutter test` | `ci.yml` → mobile-test | every PR | 80%+ on new widgets | CI |
| Integration (backend) | `go test -tags=integration` + real Postgres + real Typesense | `internal/search/integration_test.go`, `internal/adapter/postgres/*_test.go` | `MARKETPLACE_TEST_DATABASE_URL=... TYPESENSE_INTEGRATION_URL=... go test -tags=integration ./...` | `e2e.yml` → backend-integration | PR w/ `run-e2e` + push to main | N/A (correctness, not coverage) | CI |
| E2E (web) | Playwright (chromium) | `web/e2e/**.spec.ts` | `cd web && npx playwright test` | `e2e.yml` → web-e2e | PR w/ `run-e2e` + push to main | N/A | CI |
| Smoke | bash + curl | `scripts/smoke/**` | `./scripts/smoke/security.sh` | `e2e.yml` (manual) | pre-deploy | N/A | Human |
| Perf | k6 | `scripts/perf/k6-search.js` | `./scripts/perf/baseline.sh --compare` | nightly / pre-release | weekly | Trend line, >10% regression fails | Human |
| Security | gosec, govulncheck, trivy, npm audit | various | `cd backend && gosec ./...` | `security.yml` | weekly cron + lockfile PRs | Zero HIGH/CRITICAL | CI |
| Visual regression | Playwright screenshots | `web/e2e/*.spec.ts` with `toHaveScreenshot` | `npx playwright test --update-snapshots` | `e2e.yml` | on design-system PRs | N/A | Human |
| Live golden (AI) | `go test -tags=golden` | `internal/search/golden_test.go` | `OPENAI_EMBEDDINGS_LIVE=true OPENAI_API_KEY=... go test -run Golden ./internal/search/` | manual, never auto | pre-release | Top-3 keyword containment | Human |

The "Owner" column describes who ensures the tests stay green. CI jobs are owned
by whoever merges a PR that breaks them — the red X is the signal. Human-owned
jobs are the engineer's responsibility before a release.

---

## 3. How to run each layer locally

### Unit — backend

```bash
cd backend
go test ./... -count=1                       # full run, ~20s
go test ./internal/search/... -count=1       # scoped
go test ./... -count=1 -race                 # race detector (CI runs this)
go tool cover -func=coverage.out             # read coverage
```

### Unit — web

```bash
cd web
npx vitest run                               # one-shot
npx vitest                                   # watch mode
npx vitest run --coverage                    # with coverage report
```

### Unit — mobile

```bash
cd mobile
flutter pub get
flutter test                                 # unit + widget, no device
flutter test test/features/search            # scoped
```

### Integration — backend

Requires Postgres, Redis, and Typesense running (use `docker compose up -d`):

```bash
cd backend
export MARKETPLACE_TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5435/marketplace_go?sslmode=disable"
export TYPESENSE_INTEGRATION_URL="http://localhost:8108"
export TYPESENSE_INTEGRATION_API_KEY="xyz-dev-master-key-change-in-production"
go test -tags=integration ./... -count=1
```

### E2E — web

Requires a running backend (`make run` in backend/) + web dev server:

```bash
cd web
npx playwright install --with-deps chromium
npx playwright test                          # headless
npx playwright test --headed                 # see the browser
npx playwright test --ui                     # interactive explorer
```

### Smoke

```bash
./scripts/smoke/security.sh                  # phase 5B
./scripts/ci/security-baseline.sh --env local --token $JWT --admin-token $ADMIN_JWT
./scripts/ci/rbac-matrix.sh --base http://localhost:8080 --token $JWT --admin-token $ADMIN_JWT
./scripts/ci/openapi-diff.sh
```

### Perf

```bash
./scripts/perf/baseline.sh --dry-run         # parse-only, no k6 run
./scripts/perf/baseline.sh                   # record once
./scripts/perf/baseline.sh --compare         # compare to last committed baseline
./scripts/perf/baseline.sh --update          # append a new entry to baseline.json
```

### Live golden (AI)

```bash
cd backend
export OPENAI_API_KEY=sk-...
export OPENAI_EMBEDDINGS_LIVE=true
export TYPESENSE_INTEGRATION_URL=http://localhost:8108
export TYPESENSE_INTEGRATION_API_KEY=xyz-dev-master-key-change-in-production
go test -run GoldenSemantic ./internal/search/ -count=1 -v
```

Cost envelope: 14 queries × ~200 tokens each × $0.02/1M = negligible. Even 500
runs during a debug session stay under **$0.10**. We never gate this on CI so
no cost is charged without a human in the loop.

---

## 4. How CI invokes each layer

| Workflow | File | Trigger | Jobs |
|----------|------|---------|------|
| CI | `.github/workflows/ci.yml` | every PR + push to main | backend-lint, backend-test, web-lint, web-test, web-build, mobile-analyze, mobile-test, ci-gate |
| E2E | `.github/workflows/e2e.yml` | PR w/ `run-e2e` label, push to main, manual dispatch | web-e2e, backend-integration |
| Lighthouse | `.github/workflows/lighthouse.yml` | successful Vercel preview deployment | lighthouse |
| Security | `.github/workflows/security.yml` | lockfile PR + weekly cron | backend-gosec, backend-trivy, web-npm-audit, mobile-pub-outdated |
| Drift | `.github/workflows/drift.yml` | hourly cron (staging only) | drift-check |
| Snapshot | `.github/workflows/snapshot.yml` | daily cron (documentation) | snapshot |

Every workflow uses `permissions: read-all` by default and narrows to
`contents:write` / `issues:write` / `pull-requests:write` per job only
when needed. Actions are pinned to major versions (`@v4`, never `@main`).

Concurrency groups auto-cancel stale PR runs. Main-branch runs are never
cancelled — even if a newer commit lands, the older run finishes so we
always have a green/red state for every SHA on main.

---

## 5. Adding a new test

### New Go unit test

1. Create `<file>_test.go` next to the file under test.
2. Use table-driven subtests with `t.Run(tt.name, ...)`.
3. Mock every port via the `mock/` package.
4. Run `go test ./...` locally.
5. If the package is under the coverage gate, verify `go tool cover -func`
   still reports >=85% for that package.

### New web vitest

1. Create `<file>.test.ts(x)` next to the component.
2. Use `@testing-library/react` + `@testing-library/user-event`.
3. Mock TanStack Query hooks via the test provider.
4. Run `npx vitest run <path>`.

### New Playwright spec

1. Add `web/e2e/<feature>.spec.ts`.
2. Use page objects (reusable helpers in `web/e2e/pages/`).
3. Label the PR `run-e2e` so CI runs the suite.
4. Artefacts (trace, video) upload on failure; `actions/upload-artifact@v4`
   keeps them for 14 days.

### New golden AI query

1. Add the `{query, expectedKeywords, persona}` tuple to
   `backend/internal/search/golden_test.go`.
2. Keep the expected set to 3-5 keywords — profile IDs rotate when the
   dataset rebuilds.
3. Document the intent inline (`// why: this tests that "apporteur" is
   synonymised to "business referrer" correctly`).

### New integration test

1. Add `<feature>_integration_test.go` under `internal/search/` or
   `internal/adapter/postgres/`.
2. Gate with `//go:build integration` so it is excluded from the default
   run.
3. Read `MARKETPLACE_TEST_DATABASE_URL` / `TYPESENSE_INTEGRATION_URL`
   and `t.Skip` if unset — never fail the build when a developer runs
   `go test ./...` without containers.

---

## 6. Coverage commitments

| Module | Committed minimum | Rationale |
|--------|-------------------|-----------|
| `backend/internal/search/**` | 85% | core search logic; bugs cost trust |
| `backend/internal/app/search/**` | 85% | filter + cursor + facet glue; cheap to cover |
| `backend/internal/app/searchanalytics/**` | 85% | analytics are append-only; easy to test in isolation |
| `backend/internal/handler/search_handler.go` | 80% | thin layer, but every error branch matters |
| `backend/internal/handler/middleware/redact.go` | 90% | security-critical, regex-heavy |
| `backend/internal/handler/metrics.go` | 80% | instrumentation, not business logic |
| `web/src/shared/lib/search/**` | 80% | TS counterpart of the Go filter builder |
| `web/src/shared/components/search/**` | 80% | user-visible, snapshot-friendly |
| `mobile/lib/shared/search/**` | 80% | parity with web |

Coverage below these thresholds fails CI. Raising a threshold is a one-line
PR against `.github/workflows/ci.yml`. Lowering one requires a paragraph in
the PR description explaining why.

---

## 7. Live OpenAI golden tests

### When to run

- Before every release candidate.
- After every change to `internal/search/embeddings.go` or `synonyms.go`.
- When investigating a "the search returns nothing" bug where mock tests
  pass.

### Cost envelope

- 14 golden queries × ~200 tokens each = 2,800 tokens per run.
- `text-embedding-3-small` costs $0.02 per 1M tokens.
- One run: $0.0000056.
- 10,000 runs (extreme debug session): $0.056.

You can run this test thousands of times without approaching a
noticeable bill. Still — do not run it on a cron. The goal is human
judgement on the output: "did the new synonym set improve or degrade
this query?".

### Adding a query

1. Pick a query a real user would type (French or English).
2. Pick 3-5 keywords the top-3 results should contain.
3. Run the suite locally — if it passes, commit the test. If it fails,
   fix the indexer / synonyms / embedding-text composition before
   committing.

---

## 8. Visual regression

Playwright's `toHaveScreenshot()` captures a pixel-level snapshot. The
first run of a new spec creates the baseline under
`web/e2e/__screenshots__/`. Subsequent runs compare against it.

### Updating snapshots

```bash
cd web
# Update ALL snapshots (nuclear option — review the diff carefully)
npx playwright test --update-snapshots

# Update a single spec
npx playwright test search.spec.ts --update-snapshots
```

### Reviewing snapshot changes

When a PR touches UI, Playwright will produce diff images in
`web/test-results/`. CI uploads them on failure. The reviewer must
verify that every diff is intentional — "looks the same to me" is not
enough, we trust the pixel comparator.

### Stability

We lock the browser user-agent, viewport (`1280x720`), and fonts to
avoid flakes. If a snapshot is flaky without a code change, that is a
configuration bug — do NOT add `toHaveScreenshot({ threshold: 0.2 })` to
paper over it.

---

## 9. Perf

### How baselines work

`docs/perf/baseline.json` is a versioned array. Each entry is one run:

```json
{
  "date": "2026-04-17",
  "commit": "a0741d2",
  "metrics": {
    "p50_ms": 80,
    "p95_ms": 150,
    "p99_ms": 280,
    "throughput_rps": 200,
    "error_rate": 0.001
  }
}
```

`scripts/perf/baseline.sh --compare` diffs the latest run against the
last committed entry. Any metric regression >10% fails the script with
exit code 2. The CI workflow (when a perf job is wired in) posts the
diff as a PR comment.

### When to bump the baseline

- After a genuine, expected improvement ("we added a caching layer,
  p95 dropped 30%" — commit a new entry).
- Never to silence a flaky regression. If the numbers bounce, dig:
  cold vs warm cache, cold VM, insufficient warm-up iterations.

### The k6 script itself

Phase 5B owns `scripts/perf/k6-search.js`. Do not edit it from this
repo root — changes must land via the 5B branch so the baseline /
k6 script evolve together.

---

## 10. Security tests

### RBAC matrix

`scripts/ci/rbac-matrix.sh` exercises the 3x3 grid of {anon, user,
admin} × {public, auth-only, admin-only}. Every cell has an expected
HTTP status code. Any mismatch fails the script with a row-level
`[FAIL]` marker.

The script is the canonical test of:
- `middleware.Auth` — rejects anon with 401
- `middleware.RequireRole("admin")` — rejects user with 403
- handler ownership checks — rejects owner-B reading owner-A's data

### Security baseline

`scripts/ci/security-baseline.sh` enforces:
- Every tested endpoint returns all 5 security headers (CSP, X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy).
- `/api/v1/admin/*` denies anon, user, and expired tokens.
- `/api/v1/search/key` response contains ONLY the whitelisted fields (`key`, `host`, `ttl`, `persona`, and response envelope fields). Any extra field is treated as a potential leak.
- `/api/v1/search/track` returns 204 and does not 5xx even under
  repeated calls.
- Search parameters (`q`, `filter_by`, `sort_by`, `cursor`) accept
  adversarial payloads (XSS, SQLi, path traversal, oversize) without
  emitting a 5xx response.

### Fuzz corpora

The fuzz list lives inline in `security-baseline.sh` so changes are
reviewable in a PR. To add a payload:

1. Edit the `PAYLOADS=(...)` array.
2. Re-run against a local backend (`./scripts/ci/security-baseline.sh`).
3. If the backend returns 5xx, file a bug — do not commit the payload
   until the server is fixed.

### Scoped-key security model

`GET /api/v1/search/key` returns a short-lived (1h TTL) key whose
`filter_by` clause is baked in via HMAC-SHA256. The response body
contains at most five fields: `key`, `host`, `ttl`, `persona`, and the
response envelope's `request_id`. Anything else is a bug — the baseline
script enforces this invariant via a field allowlist.

The master key never leaves the backend. Integration tests
(`internal/search/scoped_client_test.go`) verify that a client armed
with a `persona=freelance` key receives zero hits when querying
`persona:agency` documents, even if it forges a different `filter_by`
in the request. This is the cornerstone of the multi-tenant security
model for search.

---

## 11. Validation pipeline (pre-commit)

Every commit must pass these gates locally before push. The CI
pipeline enforces the same gates — if a step fails in CI but passed
locally, you skipped it.

```bash
# Backend
cd backend
go build ./...
go vet ./...
go test ./... -count=1 -race
gofmt -l . | (! grep .)            # empty output = formatted

# Web
cd web
npx tsc --noEmit
npx vitest run
npx next build

# Mobile
cd mobile
flutter analyze lib/features/search lib/features/profile lib/shared/search
flutter test test/features/search test/shared/search

# Cross-stack
./scripts/ci/openapi-diff.sh       # if backend is running
```

A helper script `./scripts/ci/pre-commit.sh` can be dropped in later
to chain all of the above. For now, the test framework's speed makes
the manual invocation tolerable.
