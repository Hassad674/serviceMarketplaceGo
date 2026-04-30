# Marketplace Service

[![CI](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/ci.yml)
[![E2E](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/e2e.yml)
[![Security](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/security.yml)
[![Coverage](https://codecov.io/gh/Hassad674/serviceMarketplaceGo/branch/main/graph/badge.svg)](https://codecov.io/gh/Hassad674/serviceMarketplaceGo)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Hassad674/serviceMarketplaceGo)](https://goreportcard.com/report/github.com/Hassad674/serviceMarketplaceGo)

An open-source, full-featured B2B marketplace connecting agencies,
enterprises, freelancers, and business referrers — built end to end
to be a showcase of professional engineering practice. Not a
directory or a job board: contracts, escrow payments, milestones,
disputes, invoicing, real-time messaging, video calls, hybrid
search, and a full admin dashboard, across **four apps** that share
a single contract.

---

## Quick demo

```bash
# 1. Bring up infrastructure (Postgres 16 + Redis 7 + MinIO + Typesense 28)
docker compose up -d

# 2. Schema + seed
cd backend && cp .env.example .env && make migrate-up && make seed

# 3. Run the API (port 8083)
make run
```

In another shell:

```bash
# 4. Run the web app (port 3001)
cd web && npm install && npm run dev

# 5. Optional — admin dashboard (port 5173)
cd admin && npm install && npm run dev
```

Open <http://localhost:3001>, register as an Agency or Enterprise,
and you have the full marketplace running on your laptop. The
mobile app (Flutter) follows the same pattern: `cd mobile && flutter
pub get && flutter run`.

---

## What's inside

| App           | Stack                                    | Source        | Audience                                             |
|---------------|------------------------------------------|---------------|------------------------------------------------------|
| **Backend**   | Go 1.25 + Chi v5 + PostgreSQL 16 + Redis 7 + Typesense 28 | `backend/`    | API server — single source of truth for the contract |
| **Web**       | Next.js 16 + React 19 + Tailwind 4       | `web/`        | End users (agency, enterprise, provider, referrer)   |
| **Admin**     | Vite 7 + React 19 + Tailwind 4           | `admin/`      | Platform staff (moderation, support, billing)        |
| **Mobile**    | Flutter 3.16+ / Dart 3.2+                | `mobile/`     | iOS + Android end users                              |

The backend exposes an OpenAPI 3.1 schema; each frontend generates
its own typed client from it. **No shared packages** between the
four apps — they evolve at their own pace and ship independently.

---

## Engineering choices

The architecture is opinionated. The full deep-dive — diagrams,
sequence flows, security model — lives in [**docs/ARCHITECTURE.md**](docs/ARCHITECTURE.md).
Selected highlights:

- **Hexagonal architecture** with the dependency rule
  `handler -> app -> domain <- port <- adapter` enforced by review
  and by `go vet`. Adapters never import each other; wiring lives
  in exactly one file (`backend/cmd/api/main.go`).
- **Feature isolation invariant**: deleting a feature folder
  (`internal/app/<x>/`, `web/src/features/<x>/`,
  `mobile/lib/features/<x>/`, `admin/src/features/<x>/`) must cause
  zero compile errors elsewhere. Cross-feature data is exchanged
  through injected interfaces, never imports. Enforced by an e2e
  contract test.
- **Org-scoped business state**: every business row owns by
  `organization_id`, not `user_id`. `user_id` is reserved for
  authorship (audit log, `created_by`). A user joining or leaving a
  company never affects what the company owns.
- **Defense in depth on multi-tenancy**: five layers — JWT, role
  middleware, handler ownership check, repository `WHERE org_id =
  $1`, and PostgreSQL **Row-Level Security** with `FORCE ROW LEVEL
  SECURITY` on **9 tenant-scoped tables**. The DB itself rejects
  cross-tenant reads if any layer above leaks.
- **Outbox pattern** for everything async: search reindexes, Stripe
  transfers, push notifications. Events written in the same
  transaction as the business mutation, drained by a background
  worker. At-least-once delivery with idempotent consumers.
- **Hybrid search** with Typesense (BM25) + OpenAI embeddings
  blended in a single query. Per-persona scoped API keys mean a
  bug in the application layer cannot leak another persona's
  results — Typesense itself enforces the filter.
- **Contract-first API** — the backend's OpenAPI schema is the
  source of truth; every frontend generates its types. Breaking
  changes blocked at PR time by `scripts/ci/openapi-diff.sh`.
- **Append-only audit log** with a Postgres role REVOKE'd of UPDATE
  and DELETE — once written, never modified.

---

## Test coverage at a glance

The repo is tested at every layer; the strategy is documented in
full at [**docs/testing.md**](docs/testing.md).

| Layer                    | Files | Cases   | Tool                                |
|--------------------------|-------|---------|-------------------------------------|
| Backend unit             | 333   | 2,634   | `go test` + testify                 |
| Web unit                 | 132   | 1,292   | vitest + @testing-library           |
| Web E2E                  | 43    | 341     | Playwright (chromium)               |
| Admin unit               | 4     | 30      | vitest                              |
| Mobile unit + widget     | 105   | 806     | `flutter test`                      |
| Backend integration      | (tagged `integration`) | — | testcontainers + real Postgres + real Typesense |
| Smoke (CLI + curl)       | `scripts/smoke/`        | — | Bash + jq |
| Perf (k6)                | `scripts/perf/`         | — | k6 |
| Security                 | every PR + weekly       | — | gosec + govulncheck + trivy + npm audit + semgrep |

CI quality gates (in `.github/workflows/ci.yml`):

- **Backend**: `go vet` + `gofmt` (changed files) + `govulncheck`
  (any CVE fails) + `go test -race -coverprofile` with per-package
  coverage thresholds (85% on `internal/search`, 80% elsewhere).
- **Web**: `tsc --noEmit` (hard fail) + `vitest --coverage` (60%
  aggregate gate) + `next build` (no secrets required).
- **Mobile**: `flutter analyze` + `flutter test --coverage` on the
  scoped surfaces.
- **All-green gate**: a final job blocks merges unless every job
  above passed.

`gosec` baseline: from 35+ findings in Phase 1 to **3 documented
false positives**, all annotated inline.

---

## Roadmap teasers

The project is pre-1.0 and built in named phases (the audit history
lives at the repo root in `auditsecurite.md`, `auditperf.md`,
`auditqualite.md`, and `bugacorriger.md`). What is shipped:

- **Phase 0 — quick wins** (2026-04-22): housekeeping and
  documentation tightening.
- **Phase 1 — security critical** (2026-04-23 → 2026-04-26):
  brute-force protection, refresh token rotation with Redis
  blacklist, CSP + HSTS + Permissions-Policy, append-only audit
  log, file upload sanitization. **40+ findings closed.**
- **Phase 1.5 — RLS** (2026-04-27): PostgreSQL Row-Level Security
  on 9 tenant-scoped tables with `FORCE ROW LEVEL SECURITY`.
- **Phase 2 — business bug fixes** (2026-04-25 → 2026-04-28): state
  machine guards on dispute / refund / payout, webhook idempotency
  via durable Postgres source of truth, WebSocket race fixes, FCM
  tap routing, single-flight refresh on mobile.
- **Phase 3 — refactor** (2026-04-26 → 2026-04-29): god-component
  splits (878-line wallet page, 797-line message-area, 758-line
  search-filter-sidebar, 656-line billing-profile-form), feature
  isolation cleanup, shared UI primitives, react-hook-form + zod
  migration.
- **Phase 4 — performance** (2026-04-28 → 2026-04-29): RSC public
  listings + JSON-LD, lazy LiveKit, dynamic admin chunks, dynamic
  sitemap and robots, mobile cold-start cleanup.
- **Phase 5 — tests + DB hardening** (2026-04-29 → 2026-04-30):
  RLS rollout, expanded coverage on admin / KYC / referral, RBAC
  matrix smoke, RLS cross-tenant denial integration tests.
- **Phase 6 — open-source polish** (2026-04-30, this PR):
  documentation, license, code-of-conduct, dependabot, PR + issue
  templates, semgrep + eslint-plugin-security in CI.

What is next:

- Phase 7: OpenTelemetry + Prometheus exporter, structured tracing
  end-to-end.
- Public stable v1 tag with a documented deprecation policy.
- Stripe Embedded payouts UI for end-to-end provider self-service.

---

## Contributing

Patches welcome. Read [**CONTRIBUTING.md**](CONTRIBUTING.md) before
starting — it covers the validation pipeline, branch ownership, the
"delete the folder = compiles" invariant, and the parallel-agent
workflow we use.

For security issues, see [**SECURITY.md**](SECURITY.md). Do **not**
open a public issue.

---

## Documentation index

- [**docs/ARCHITECTURE.md**](docs/ARCHITECTURE.md) — hexagonal
  layering, security model, search engine, payment flow, all with
  diagrams.
- [**docs/DEPLOYMENT.md**](docs/DEPLOYMENT.md) — production
  deployment runbook (Railway, Vercel, Neon, R2, Resend, LiveKit,
  Stripe).
- [**docs/testing.md**](docs/testing.md) — every test layer, every
  cadence, every coverage commitment.
- [**docs/ops.md**](docs/ops.md) — operational runbook (deploy
  order, reindex, key rotation, slow-query triage, incident
  response).
- [**docs/search-engine.md**](docs/search-engine.md) — Typesense
  schema, ranking, scoped key firewall.
- [**docs/ranking-v1.md**](docs/ranking-v1.md) and
  [**docs/ranking-tuning.md**](docs/ranking-tuning.md) — ranking
  spec and the tuning sandbox.
- [**CLAUDE.md**](CLAUDE.md) — top-level conventions for AI agents
  and humans (modularity, SOLID, security, parallel workflow).
- Per-app conventions: [`backend/CLAUDE.md`](backend/CLAUDE.md) ·
  [`web/CLAUDE.md`](web/CLAUDE.md) ·
  [`admin/CLAUDE.md`](admin/CLAUDE.md) ·
  [`mobile/CLAUDE.md`](mobile/CLAUDE.md).

---

## Contact

Maintainer: Hassad Smara — <hassad.smara69@gmail.com>.

For bug reports use the GitHub issue templates. For security issues
follow [SECURITY.md](SECURITY.md).

---

## License

Apache License 2.0. See [LICENSE](LICENSE) for the full text.

Copyright 2026 Hassad Smara.
