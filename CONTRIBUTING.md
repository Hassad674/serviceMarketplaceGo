# Contributing

Thanks for taking the time to read this. The marketplace is designed
so that a focused contributor — human or AI — can ship a well-isolated
feature in an evening without breaking unrelated parts of the
codebase. The rules below are what make that possible. Follow them and
your PR review will be quick.

This file is the canonical contributor guide. The deeper conventions
live in the per-app `CLAUDE.md` files (`backend/CLAUDE.md`,
`web/CLAUDE.md`, `admin/CLAUDE.md`, `mobile/CLAUDE.md`) — read the one
for the app you are touching before opening a PR.

---

## Table of contents

1. [Local development setup](#1-local-development-setup)
2. [Architecture cheatsheet](#2-architecture-cheatsheet)
3. [The "delete the folder" invariant](#3-the-delete-the-folder-invariant)
4. [Validation pipeline](#4-validation-pipeline)
5. [Code quality rules](#5-code-quality-rules)
6. [Conventional commits](#6-conventional-commits)
7. [Branch naming and parallel workflow](#7-branch-naming-and-parallel-workflow)
8. [Database migrations](#8-database-migrations)
9. [Pull request template](#9-pull-request-template)
10. [Reporting bugs and security issues](#10-reporting-bugs-and-security-issues)

---

## 1. Local development setup

You need Docker, Go 1.25, Node 20, and (for the mobile app) Flutter
3.16+. The full stack runs on standard ports — see the table at the
bottom of this section.

### One-time setup

```bash
git clone https://github.com/Hassad674/serviceMarketplaceGo.git
cd serviceMarketplaceGo

# Wire local pre-commit hooks (gofmt, tsc, flutter analyze).
# Skip with `git commit --no-verify` if a hook is wrong for your
# situation — see `.githooks/pre-commit` for the full list.
./scripts/install-git-hooks.sh

# Infrastructure: PostgreSQL 16 + Redis 7 + MinIO + Typesense 28
docker compose up -d

# Backend env (copy and edit)
cp backend/.env.example backend/.env

# Apply schema and seed roles + admin user
cd backend
make migrate-up
make seed
cd ..
```

### Run the four apps

```bash
# Backend (8083 by default — see backend/.env)
cd backend && make run

# Web (Next.js, 3001)
cd web && npm install && npm run dev

# Admin (Vite + React, 5173)
cd admin && npm install && npm run dev

# Mobile (Flutter — connect a device first, then)
cd mobile && flutter pub get && flutter run
```

### Ports reference

| Service                | Port |
|------------------------|------|
| Backend (Go API)       | 8083 |
| Web (Next.js)          | 3001 |
| Admin (Vite)           | 5173 |
| PostgreSQL             | 5435 |
| Redis                  | 6380 |
| MinIO API              | 9000 |
| MinIO Console          | 9001 |
| Typesense              | 8108 |
| DBGate (Postgres GUI)  | 8085 |

The defaults differ from `CLAUDE.md`'s 8080/3000 in places — those are
the canonical "production" ports; the dev stack uses 8083/3001/5435 to
avoid conflicts with whatever else you may be running locally. Check
each `.env.example` for the source of truth.

---

## 2. Architecture cheatsheet

The backend is hexagonal. The dependency rule is absolute and never
broken:

```
handler -> app -> domain <- port <- adapter
```

- `internal/domain/` — pure entities and value objects, **only**
  imports from the Go standard library.
- `internal/port/` — interface contracts (`port/repository`,
  `port/service`).
- `internal/app/` — use cases, depends only on `domain` and `port`.
- `internal/adapter/` — concrete implementations (`postgres`, `redis`,
  `stripe`, `livekit`, `s3`, `openai`, `resend`…). Adapters never
  import each other.
- `internal/handler/` — HTTP transport, depends on `app/` and `dto/`.

Wiring (the only place adapters meet app services) is `cmd/api/main.go`.
Switching from Stripe to PayPal is a one-line change there.

The web app uses a **feature-based** layout under `web/src/features/`.
Features never import each other; composition happens in `app/`
pages. Shared primitives (Button, Input, the i18n provider) live in
`web/src/shared/`.

The mobile app uses **Clean Architecture** (`lib/features/<name>/{data,
domain, presentation}/`) with Riverpod for state and Freezed for
entities. The same feature isolation rule applies.

The deep dive — diagrams, sequence flows, security model — lives in
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Read it before
adding a new feature.

---

## 3. The "delete the folder" invariant

This is the single most important rule in the project. Every feature
must be removable.

**A feature is removable when:**

1. Deleting its folder (e.g. `backend/internal/app/contract/`,
   `backend/internal/handler/contract_handler.go`,
   `web/src/features/contract/`, `mobile/lib/features/contract/`,
   `admin/src/features/contracts/`) causes ZERO compilation errors
   elsewhere.
2. No other feature imports from it directly. Cross-feature data is
   exchanged via injected interfaces wired in `cmd/api/main.go` only.
3. Its database tables can be dropped without breaking other tables.
   No cross-feature foreign keys — only references to `users(id)` and
   `organizations(id)` are allowed.
4. The app still runs perfectly without it.

**Practical consequences:**

- If `ContractService` needs user data, it accepts a
  `UserRepository` interface in its constructor — never an import of
  `internal/app/user/`.
- A web page that needs both the profile widget and the contract
  widget is the place where they meet — never inside either feature.
- A new mobile screen that needs notification state subscribes to the
  notification feature's exposed Riverpod provider; it does not reach
  into private files.

Before opening a PR, mentally test: "If I delete this feature's
entire folder and its lines in `main.go`, does everything else still
compile?" If the answer is no, refactor until it does.

The repo includes a contract test that asserts isolation
(`web/e2e/contract-isolation.spec.ts`). Adding a cross-feature import
will fail CI.

---

## 4. Validation pipeline

Run **all** of this before every commit. CI will run it again on the
PR; do not skip it locally.

### Backend

```bash
cd backend
go build ./...
go vet ./...
go test ./... -count=1 -race
```

The `-race` flag is required — the test suite is race-clean and we
intend to keep it that way.

### Web

```bash
cd web
npx tsc --noEmit
npx vitest run
npx eslint .
```

Coverage gate (CI-enforced): 60% statements aggregate, 80%+ on new
files.

### Admin

```bash
cd admin
npx tsc --noEmit
npx vitest run
```

### Mobile

```bash
cd mobile
flutter pub get
flutter analyze
flutter test
```

`flutter analyze` must be warning-free on the files you touched. Info
lints in legacy code are tracked in a backlog and being cleaned up;
do not add new info-level violations.

### Integration / E2E (run before merging large PRs)

```bash
# Backend integration (real Postgres + Typesense via docker compose)
cd backend
MARKETPLACE_TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5435/marketplace_go?sslmode=disable" \
TYPESENSE_INTEGRATION_URL="http://localhost:8108" \
TYPESENSE_INTEGRATION_API_KEY="xyz-dev-master-key-change-in-production" \
go test -tags=integration ./...

# Web Playwright
cd web
npx playwright install --with-deps chromium
npx playwright test

# Smoke
./scripts/smoke/run-all.sh
```

The full strategy — every layer, every cadence, the live OpenAI
budget envelope — is documented in [`docs/testing.md`](docs/testing.md).

---

## 5. Code quality rules

These are non-negotiable. They keep the codebase reviewable.

| Metric                     | Limit | Action when exceeded |
|----------------------------|-------|----------------------|
| Lines per file             | 600   | Split by sub-domain or responsibility |
| Lines per function         | 50    | Extract helper functions |
| Parameters per function    | 4     | Group into a struct or options object |
| Nesting depth              | 3     | Use early returns or extract |
| Cyclomatic complexity      | < 10  | Simplify conditionals or split |

### Language-specific bans

**Go:**
- No global mutable state. No `var x = NewSomething()` at package
  level. Everything is constructor-injected.
- No `any` / `interface{}` returns from public APIs (we use
  generics where polymorphism is needed).
- No SQL string concatenation. Always parameterized queries
  (`$1, $2`).

**TypeScript (web + admin):**
- No `any` — `unknown` if you must, then narrow. ESLint rule
  `@typescript-eslint/no-explicit-any` is on.
- No `dangerouslySetInnerHTML` without an explicit sanitizer.
- Server Components are the default. `"use client"` only when an
  effect, listener, or browser API is needed — and never at a layout
  boundary if a leaf could carry the directive instead.

**Dart (mobile):**
- No `dynamic` in DTO/entity layers — Freezed + `JsonSerializable`
  handle this.
- No `print()` in production code. Use `debugPrint` or the structured
  `logger` in `lib/shared/logger`.
- `const` constructors wherever the analyzer suggests one.
- No `ref.read` inside `build()` — `ref.watch`, or move it to
  `initState`/callbacks.

### Naming

Names reveal intent. The following are forbidden across all four
apps: `data`, `info`, `temp`, `result`, `manager`, `helper`, `utils`,
`misc`, `handler2`. Use `CalculateCommission`, not `DoCalc`.
`ValidateEmail`, not `Check`.

### SOLID and the STUPID anti-patterns

The full SOLID examples live in `backend/CLAUDE.md` lines 51-178.
Anti-patterns to avoid (singleton, tight coupling, untestability,
premature optimization, indescriptive naming, duplication) are
detailed at lines 234-258 of the root `CLAUDE.md`.

---

## 6. Conventional commits

Every commit follows the [Conventional Commits](https://www.conventionalcommits.org/)
format:

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

**Allowed types** (no others): `feat`, `fix`, `refactor`, `chore`,
`test`, `docs`, `perf`, `style`, `revert`.

**Examples** (taken from `git log`):

```
feat(security): enable RLS on 9 tenant-scoped tables (SEC-10)
fix(invoicing/mobile): download PDF through authenticated ApiClient
refactor(messaging): split 797-line message-area into 4 focused units
test(security): cross-tenant denial integration tests for migration 125
docs(security): document RLS policies, helpers, and prod DB-user requirements
```

One commit = one logical change. If your work spans backend + web +
mobile, three commits with the same intent are fine — keep each one
green by itself so a `git revert <sha>` is always safe.

The body explains the *why*, not the *what*. The diff already shows
the what.

---

## 7. Branch naming and parallel workflow

**`main` is protected.** It must always compile and have a green CI.
You never push directly to `main`.

### Branch types

| Prefix      | Meaning                                  |
|-------------|------------------------------------------|
| `feat/<x>`  | New feature or vertical slice            |
| `fix/<x>`   | Bug fix                                  |
| `refactor/<x>` | Pure refactor, no behaviour change    |
| `test/<x>`  | Adding tests with no production change   |
| `docs/<x>`  | Documentation only                       |
| `wip/<x>`   | Long-running work-in-progress branches   |

### Parallel agent workflow

The repo is built to be worked on by several agents (or humans) in
parallel without stepping on each other:

1. Each task gets its own **git worktree** + branch off `main`.
2. If the task touches migrations, it MUST also use an isolated DB
   copy (`createdb -p 5435 marketplace_go_<scope> -T marketplace_go`).
   Never run `migrate-down` on the shared DB. See `CLAUDE.md` lines
   575-602 for the full rule.
3. Branch ownership rule: **an agent (or contributor) never commits
   on a branch they did not create themselves.** This is enforced
   by convention; we have lost work to it before.
4. PRs go through GitHub. CI must be green to merge. We squash-merge
   to keep `main` history linear and reverts trivial.

If you are doing this with AI agents, brief them with the four
phrases the project considers mandatory: scope discipline (ni plus
ni moins), self-validation (run the pipeline before commit), branch
ownership (own your branch), and the parallel migration rule
(isolated DB).

---

## 8. Database migrations

Pure SQL, powered by `golang-migrate`. Files live in
`backend/migrations/<NNN>_<name>.up.sql` and `<NNN>_<name>.down.sql`.

Rules:

1. **Both files mandatory.** A migration without a `down.sql` will be
   rejected.
2. **Immutable once on `main`.** If you applied a migration in prod
   and then realised it was wrong, fix it forward with a new
   migration (`<NNN+1>_fix_<thing>.up.sql`). Never edit a merged
   migration.
3. **Idempotent**: use `IF [NOT] EXISTS` so re-running is safe.
4. **Conventions**: UUID primary keys, `created_at TIMESTAMPTZ NOT
   NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT
   now()`, `TEXT` over `VARCHAR(...)`, index every foreign key.
5. **No cross-feature foreign keys.** A `proposals` table can
   reference `users(id)` and `organizations(id)` but **not**
   `messages(id)`. This preserves the deletability invariant.
6. **RLS for tenant-scoped tables.** New tables holding
   organization-scoped business state must follow the pattern in
   `migrations/125_enable_row_level_security.up.sql`.

To create a migration:

```bash
cd backend
make migrate-create NAME=add_thing
# Edit the two generated files
make migrate-up    # Apply on local
make migrate-down  # Test the rollback
make migrate-up    # Re-apply
```

Verify the schema:

```bash
psql "$DATABASE_URL" -c "\d proposals"
```

---

## 9. Pull request template

When you open a PR, GitHub will pre-fill the description with
`.github/PULL_REQUEST_TEMPLATE.md`. Keep it filled in — reviewers
need it.

The template asks for:

- **Summary** — 1 to 3 bullets explaining the *why*.
- **Test plan** — a checklist of how you verified the change.
- **Linked issues** — `Closes #N` if applicable.

For larger PRs, include screenshots (web/admin) or a screen recording
(mobile). Visual regressions are easier to catch in review than in
prod.

---

## 10. Reporting bugs and security issues

- **Bugs**: open a GitHub issue using the "Bug report" template
  (`.github/ISSUE_TEMPLATE/bug.yml`). Include reproduction steps,
  expected and actual behaviour, and the affected app(s).
- **Feature requests**: use `.github/ISSUE_TEMPLATE/feature.yml`.
- **Security vulnerabilities**: do **not** open a public issue. Email
  `hassad.smara69@gmail.com` (see `SECURITY.md` for the full
  policy and our 72h ack target).

---

Thanks again for contributing. The bar is high on purpose — the
codebase is meant to be a showcase of professional-grade engineering,
and that only works if every PR holds it up.
