# Marketplace Service

[![CI](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/ci.yml)
[![E2E](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/e2e.yml)
[![Security](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/Hassad674/serviceMarketplaceGo/actions/workflows/security.yml)
[![Codecov](https://codecov.io/gh/Hassad674/serviceMarketplaceGo/branch/main/graph/badge.svg)](https://codecov.io/gh/Hassad674/serviceMarketplaceGo)

Open-source B2B marketplace connecting agencies, freelancers, enterprises,
and business referrers — built as a complete showcase of professional
engineering practice.

**Stack**: Next.js 16 (web) · Go 1.25 + Chi v5 (backend) · PostgreSQL 16
(pure SQL, no ORM) · Redis 7 · MinIO · Typesense 28.0 (search)

**Apps**: `backend/` (Go API) · `web/` (Next.js) · `admin/` (Vite + React)
· `mobile/` (Flutter 3.16+)

---

## Quick start

```bash
# Infrastructure (PostgreSQL 16, Redis 7, MinIO, Typesense)
docker compose up -d

# Apply migrations
cd backend && make migrate-up

# Seed data (admin user, default roles)
cd backend && make seed

# Backend (port 8080)
cd backend && make run

# Web frontend (port 3000)
cd web && npm install && npm run dev

# Admin panel (port 5173)
cd admin && npm install && npm run dev
```

Port reference: 8080 backend · 3000 web · 5173 admin · 5434 postgres ·
6380 redis · 9000/9001 minio · 8108 typesense.

---

## Testing

This project is tested at every layer — unit, integration, E2E, smoke,
perf, and security. Every layer has a corresponding command, cadence,
and coverage commitment.

See [**docs/testing.md**](docs/testing.md) for the full strategy, test
matrix, coverage commitments, live-OpenAI cost envelope, and
instructions to add a new test at any layer.

Quick reference:

| Layer | Command |
|-------|---------|
| Backend unit | `cd backend && go test ./... -count=1 -race` |
| Web unit | `cd web && npx vitest run` |
| Mobile unit | `cd mobile && flutter test` |
| Backend integration | `cd backend && go test -tags=integration ./...` |
| Web E2E | `cd web && npx playwright test` |
| Security smoke | `./scripts/ci/security-baseline.sh --env local` |
| RBAC matrix | `./scripts/ci/rbac-matrix.sh --env local` |
| Perf baseline | `./scripts/perf/baseline.sh --compare` |

CI runs all of these on every PR. See `.github/workflows/` for the
workflow definitions and [docs/testing.md §4](docs/testing.md#4-how-ci-invokes-each-layer)
for the CI-to-layer mapping.

---

## Operations

Runbook for deploy order, reindex, key rotation, snapshot, restore,
drift alerts, slow-query triage, and incident response lives in
[**docs/ops.md**](docs/ops.md).

---

## Architecture

- `CLAUDE.md` — top-level conventions (modularity, SOLID, security).
- `backend/CLAUDE.md` — hexagonal layering, port/adapter rules, SQL +
  migration conventions, error handling chain.
- `web/CLAUDE.md` — feature-based structure, Server vs Client
  Components, TanStack Query patterns.
- `admin/CLAUDE.md` — Vite + React 19 + Tailwind 4 conventions.
- `mobile/CLAUDE.md` — Clean Architecture layers, Riverpod providers,
  Freezed entities.
- `docs/search-engine.md` — Typesense + OpenAI embedding search
  architecture (phase 4).

---

## Contributing

1. Start from `main`, create your own branch (`feat/<name>`).
2. One feature per commit; conventional commit messages (`feat:`,
   `fix:`, `refactor:`, `chore:`, `test:`, `docs:`).
3. Run the full validation pipeline locally before every commit
   ([docs/testing.md §11](docs/testing.md#11-validation-pipeline-pre-commit)).
4. Never merge code that fails CI or has lower coverage than the
   module commitment ([docs/testing.md §6](docs/testing.md#6-coverage-commitments)).

See `CLAUDE.md` for the complete rules around branch ownership,
migration safety, and parallel work.

---

## License

(to be confirmed — placeholder)
