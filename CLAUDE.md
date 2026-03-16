# Marketplace Service

## What is this project

Open-source B2B marketplace connecting agencies, freelancers, enterprises, and business referrers. Not a simple directory or job board — this is a full-featured marketplace with contracts, matching, messaging, payments, and administration.

**Stack**: Next.js 16 (web) + Go 1.25 with Chi v5 (backend) + PostgreSQL 16 (pure SQL, no ORM) + Redis 7 (caching/sessions) + MinIO (object storage)
**Apps**: backend/ (Go API) + web/ (Next.js) + admin/ (Vite + React) + mobile/ (Flutter 3.16+)

This project is meant to showcase professional-grade engineering. Every file, every pattern, every decision should reflect that.

---

## Core philosophy — Modularity above all

This is a marketplace with multiple user roles and complex interactions. Despite that complexity, every feature must remain independently developed, tested, and deployable. A broken billing module must never take down messaging. A new matching algorithm must never require changes in the auth system.

**This means every feature must be fully independent and removable.**

### What this means in practice

**A feature is removable when:**
- Deleting its folder (backend + web) causes ZERO compilation errors elsewhere
- No other feature imports from it directly
- Its database tables can be dropped without breaking other tables
- The app still runs perfectly without it

**The only core modules that everything depends on are:**
- Auth (users must exist for anything to work)
- Database connection and config

Everything else — matching, contracts, messaging, billing, reviews, notifications — is optional. The marketplace can start with just auth and profiles, then grow feature by feature.

### Rules to enforce this

1. **Features never import each other.** If contracts need user data, it receives a `UserRepository` interface via dependency injection — not by importing the user service directly. This way, contracts depends on an interface, not on the user module.

2. **Database tables reference users via foreign key, but no cross-feature foreign keys.** The `contracts` table can reference `users(id)`, but the `messages` table must NOT reference `contracts`. Each feature's tables are self-contained.

3. **Frontend features never import from other features.** Composition happens in `app/` pages only. If a dashboard page needs both profile and contract components, the page imports from both features — the features don't know about each other.

4. **Backend wiring in main.go is explicit.** Every feature is wired with its dependencies in `cmd/api/main.go`. Removing a feature means deleting its lines there. No auto-discovery, no magic registration.

5. **Server Actions are feature-scoped.** Each feature's `actions/` folder only calls API endpoints related to that feature. No cross-feature API calls from within a feature.

6. **Migrations are feature-prefixed.** Each feature's tables are created in their own migration files. Dropping a feature = skipping or reverting its migrations.

### How to verify independence

Before merging any feature, mentally test: "If I delete this feature's entire folder and its lines in main.go, does everything else still compile and run?" If the answer is no, there's a hidden dependency that must be fixed.

---

## Contract-first API design

The backend is the single source of truth for the API contract. Every frontend generates its own types from the backend's OpenAPI schema.

### How it works

1. **Backend** exposes an OpenAPI 3.1 schema at `/api/openapi.json`
2. **Each frontend app** (web/, admin/, mobile/) runs `openapi-typescript` (web/admin) or openapi-generator (Flutter) to generate types from that schema
3. **Each frontend app** uses type-safe API calls against the generated contract
4. **NO shared packages** between apps — each app is fully independent

### Why no shared packages

- Apps evolve at different paces (admin might need different endpoints than web)
- Deployment independence — changing admin must never require rebuilding web
- Each app only generates types for the endpoints it actually uses
- Simpler dependency graph, no monorepo tooling needed

---

## Roles and user types

The marketplace has three primary roles:

| Role | French term | Description |
|------|-------------|-------------|
| **Agency** | Prestataire | Service provider company that employs/manages freelancers |
| **Enterprise** | Client | Business that publishes needs and contracts agencies/freelancers |
| **Provider** | Freelance | Independent professional offering services |

**Provider special case:** A Provider has a `referrer_enabled` boolean field. When toggled on, the Provider also acts as an "apporteur d'affaire" (business referrer) — someone who connects enterprises with agencies/freelancers for a commission. This is a toggle, not a separate role.

---

## Project structure

```
marketplaceServiceGo/
├── backend/           -> Go 1.25 + Chi v5, hexagonal architecture (see backend/CLAUDE.md)
│   ├── cmd/           -> Entry points (api, migrate, seed)
│   ├── internal/      -> Private application code (domain, port, app, adapter, handler)
│   ├── pkg/           -> Public reusable packages
│   ├── migrations/    -> SQL migration files (up/down)
│   ├── mock/          -> Generated mocks from port interfaces
│   └── test/          -> Integration / E2E tests
│
├── web/               -> Next.js 16, React 19, feature-based (see web/CLAUDE.md)
│   └── src/           -> Source code
│
├── admin/             -> Vite + React 19 + Tailwind 4, admin dashboard (see admin/CLAUDE.md)
│   └── src/           -> Source code
│
├── mobile/            -> Flutter 3.16+, Dart (see mobile/CLAUDE.md)
│   └── lib/           -> Source code
│
├── docker-compose.yml -> PostgreSQL 16 + Redis 7 + MinIO
└── CLAUDE.md          -> This file
```

Each major directory has its own CLAUDE.md with specific conventions.

---

## Code quality standards

These limits are non-negotiable. They keep the codebase readable, testable, and reviewable.

| Metric | Limit | Action when exceeded |
|--------|-------|----------------------|
| Lines per file | 600 max | Split into focused files by sub-domain or responsibility |
| Lines per function | 50 max | Extract helper functions or break into pipeline steps |
| Parameters per function | 4 max | Group into a struct or options object |
| Nesting depth | 3 levels max | Use early returns, extract to functions, or invert conditions |
| Cyclomatic complexity | < 10 per function | Simplify conditionals, use lookup tables, or split logic |

**Why these numbers:** Files above 600 lines are impossible to review in a single pass. Functions above 50 lines have too many responsibilities. More than 4 parameters signal a missing abstraction. Deep nesting is the primary source of bugs in conditional logic.

---

## SOLID principles — with concrete examples

### S — Single Responsibility

One service = one domain. One file = one concern.

- `AuthService` handles authentication (login, register, token refresh). It does NOT manage user profiles, send marketing emails, or generate invoices.
- `UserRepository` handles user persistence. It does NOT validate business rules or format HTTP responses.
- If a service starts growing beyond 200 lines, it likely has two responsibilities. Split it.

### O — Open/Closed

Port interfaces allow extension without modification of existing code.

- Adding a new payment provider (Stripe, PayPal, etc.) = create a new adapter file. Zero changes to domain or app layer.
- `internal/adapter/stripe/payment.go` implements `port/service/PaymentService`.
- Switching providers: change ONE line in `cmd/api/main.go`. `payment := stripe.New(cfg)` becomes `payment := paypal.New(cfg)`.

### L — Liskov Substitution

Any implementation of an interface must be a drop-in replacement.

- Any `UserRepository` implementation — Postgres, MySQL, in-memory mock — is interchangeable. The app layer never knows which one it uses.
- If a mock implementation needs special setup that the real one does not, the interface is wrong. Redesign it.

### I — Interface Segregation

Small, focused interfaces. No god interfaces.

- `HasherService` has 2 methods: `Hash(password string) (string, error)` and `Compare(hash, password string) error`. That is the entire contract.
- `StorageService` has 3 methods: `Upload`, `Delete`, `GetPresignedURL`. Not 20 methods covering every possible storage operation.
- If a consumer only uses 2 of 10 methods, the interface is too wide. Split it.

### D — Dependency Inversion

The app layer depends on port interfaces, never on adapter implementations.

- `app/auth/service.go` imports `port/repository` and `port/service` — never `adapter/postgres` or `adapter/redis`.
- All wiring (connecting interfaces to implementations) happens in ONE place: `cmd/api/main.go`.
- Tests inject mocks through the same constructor. No special test wiring needed.

---

## STUPID anti-patterns — what to NEVER do

### Singleton — No global mutable state

Every dependency is passed via constructor injection. No `var db *sql.DB` at package level. No `func GetInstance()`. If you need a shared resource, create it in `main.go` and inject it into every service that needs it.

### Tight Coupling — Features never import each other

Wiring only happens in `cmd/api/main.go`. If `ContractService` needs user data, it receives a `UserRepository` interface — never an import of the user package. A broken user feature should never prevent contract code from compiling.

### Untestability — Every service depends on interfaces

If you cannot write a unit test with a mock in under 5 minutes, the code is too coupled. Every app service takes interfaces via constructor. Every adapter implements a port interface. No concrete types in business logic signatures.

### Premature Optimization — Profile before optimizing

Write correct, readable code first. Only optimize when benchmarks prove a bottleneck. The order is always: make it work, make it right, make it fast. Use `pprof` and `EXPLAIN ANALYZE` — never gut feelings.

### Indescriptive Naming — Names reveal intent

Forbidden names: `data`, `info`, `temp`, `result`, `manager`, `handler2`, `utils`, `helpers`, `misc`. Every variable, function, type, and file name must make its purpose obvious without reading the implementation. `CalculateCommission` not `DoCalc`. `ValidateEmail` not `Check`.

### Duplication — The rule of three

Do not abstract on the first or second occurrence. Copy-paste is acceptable twice. On the THIRD occurrence, extract the shared logic into a reusable function or package. Premature abstraction is worse than duplication.

---

## Performance standards

### Backend API

| Metric | Target | How to measure |
|--------|--------|----------------|
| p95 latency (CRUD) | < 100ms | Structured logging with request duration |
| p95 latency (complex queries) | < 500ms | Structured logging + EXPLAIN ANALYZE |
| Throughput | > 1000 req/s per instance | Load testing with k6 or vegeta |
| Connection pool | Max 25 idle, 50 open | `sql.DB.SetMaxIdleConns` / `SetMaxOpenConns` |

### Database

- Every query touching > 1000 rows must use an index. Verify with `EXPLAIN ANALYZE`.
- Any query exceeding 50ms in dev must be investigated and optimized.
- N+1 queries are forbidden. Use JOINs or batch queries (see backend/CLAUDE.md for details).
- Pagination on every list endpoint. No unbounded `SELECT *`.

### Frontend (web/)

| Core Web Vital | Target |
|----------------|--------|
| LCP (Largest Contentful Paint) | < 2.5s |
| FID (First Input Delay) | < 100ms |
| CLS (Cumulative Layout Shift) | < 0.1 |
| JS bundle (initial load) | < 200KB gzipped |

- Server Components by default. Client Components only when interactivity is required.
- Lazy load below-the-fold content and heavy components.
- Images: always use `next/image` with explicit width/height.

### Mobile (Flutter 3.16+)

| Metric | Target |
|--------|--------|
| Frame rate | 60fps minimum, 120fps on capable devices |
| App cold start | < 2s to interactive |
| App warm start | < 500ms |
| APK size | < 30MB |

- Use `const` constructors wherever possible.
- Avoid rebuilding entire widget trees — use targeted `setState` or state management.
- Profile with Flutter DevTools before optimizing.

---

## Security standards

### Input and queries

- **SQL injection**: Always parameterized queries (`$1, $2, $3`). Never string concatenation. No exceptions.
- **Input validation**: Validate at the boundary (handler layer). Sanitize all user input. Reject unknown fields.
- **XSS**: All user-generated content rendered with proper escaping. React handles this by default, but `dangerouslySetInnerHTML` is forbidden without sanitization.

### Authentication and authorization

- **JWT**: Short-lived access tokens (15 minutes). Refresh token rotation with single-use tokens.
- **Passwords**: bcrypt with cost 12 minimum. Never store plaintext. Never log passwords.
- **RBAC**: Role checks enforced at BOTH handler middleware AND app service layer. Defense in depth.
- **Rate limiting**: Aggressive on auth endpoints (login, register, password reset). Standard on all others.

### Infrastructure

- **CORS**: Explicit allow-list of origins. Never `*` in production. Never reflect the `Origin` header blindly.
- **Secrets**: Never in code, always environment variables. `.env` files are gitignored. No secrets in Docker images.
- **HTTPS**: All production traffic over TLS. HSTS headers enabled.
- **Headers**: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy` configured.

### RGPD / GDPR compliance

- **Data minimization**: Collect only what is necessary. No "nice to have" personal data fields.
- **Right to deletion**: Users can request full account deletion. Cascade delete all personal data.
- **Right to export**: Users can request a data export (JSON format) of all their personal data.
- **Consent**: Explicit opt-in for marketing communications. Track consent timestamps.
- **EU data residency**: Production databases hosted in EU region.
- **Retention**: Define and enforce data retention policies per data type.

### Authentication security (enhanced)

- **Brute force protection**: Maximum 5 failed login attempts per email per 15-minute window, then 30-minute lockout. Tracked in Redis with key `login_attempts:{email}` (TTL 15min) and `login_locked:{email}` (TTL 30min). Return `429 Too Many Requests` when locked.
- **Refresh token rotation**: Every call to `/api/v1/auth/refresh` generates a new access+refresh token pair. The old refresh token is immediately added to a Redis blacklist (TTL = old token's remaining expiry). If a blacklisted refresh token is reused, respond with `401 Unauthorized` — this signals token theft.
- **Token revocation**: Logout invalidates the refresh token by adding it to the Redis blacklist. The access token remains valid until its short TTL (15min) expires — this is acceptable because the window is small and avoids the cost of checking a blacklist on every request.
- **Password requirements**: Enforced at domain level (already done). Minimum 8 characters, at least one uppercase, one lowercase, one digit, one special character.

### Authorization model (who can do what)

Three layers, evaluated in order — every request must pass all three:

1. **Authentication** (JWT middleware): Is this a valid, non-expired token? Extract `user_id` and `role` into context.
2. **Role check** (RequireRole middleware): Does the user's role permit access to this endpoint group?
3. **Ownership check** (handler level): Does the user OWN the specific resource they are trying to read/modify?

Every mutation endpoint must verify the user OWNS the resource or has the `admin` role. Never trust client-side role checks alone — always verify server-side. A user with role `agency` who sends a request to modify another agency's profile must receive `403 Forbidden`, even though the role check passes.

### HTTP security headers (middleware)

Applied globally via `SecurityHeaders` middleware on every response:

```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 0 (rely on CSP instead)
Strict-Transport-Security: max-age=31536000; includeSubDomains
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
```

`X-XSS-Protection` is set to `0` intentionally — the legacy XSS auditor in older browsers can introduce vulnerabilities. Modern protection comes from the `Content-Security-Policy` header.

### Input sanitization

- **All user text input**: Strip HTML tags before storage to prevent stored XSS. Use a allowlist-based sanitizer, not a denylist.
- **File uploads**: Validate MIME type by reading file magic bytes (not just the extension). Enforce max file size (10MB default). Reject executable extensions (`.exe`, `.sh`, `.bat`, `.cmd`, `.ps1`, `.php`, `.jsp`). Store in MinIO with randomized keys — never use the original filename in the storage path.
- **URL inputs**: Validate scheme (`https` only in production, `http` allowed in dev). Reject `javascript:`, `data:`, `vbscript:`, and `file:` URIs. Validate that the host resolves to a public IP (prevent SSRF against internal services).

### Audit logging

- Log all authentication events: `login_success`, `login_failure`, `logout`, `password_reset_request`, `password_reset_complete`, `token_refresh`.
- Log all data mutations: `create`, `update`, `delete` with `user_id`, `resource_type`, `resource_id`, `timestamp`, and `ip_address`.
- Log all permission denials: `authorization_denied` with `user_id`, `attempted_action`, `resource_type`, `resource_id`.
- Store in a dedicated `audit_logs` table. This table is **append-only** — no `UPDATE` or `DELETE` operations, ever. No soft deletes. Application-level DB user should only have `INSERT` and `SELECT` on this table.
- Retention: audit logs are kept indefinitely (legal/compliance requirement). Archive to cold storage after 12 months if volume becomes a concern.

### Rate limiting strategy

All rate limits are enforced via Redis sliding window counters. When a limit is exceeded, return `429 Too Many Requests` with a `Retry-After` header.

```
Global:      100 req/min per IP (covers all endpoints)
Auth:        5 req/min per email (login, register, password reset)
Mutations:   30 req/min per authenticated user (POST, PUT, PATCH, DELETE)
File upload: 10 req/min per authenticated user
```

Rate limit headers included on every response: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.

---

## Data isolation and authorization

### Three-layer security model

```
Request → JWT Auth (who are you?) → Authorization (what role?) → Ownership (is this yours?) → Repository (WHERE user_id = ?) → PostgreSQL (RLS backup)
```

Every layer is a defense-in-depth checkpoint. If the application layer has a bug that skips the ownership check, PostgreSQL RLS prevents cross-tenant data leaks. If RLS is misconfigured, the application-level `WHERE user_id = $1` still filters correctly. No single point of failure.

### Authorization middleware pattern

- `middleware.Auth(tokenService)` — validates JWT, extracts `user_id` + `role`, stores in request context.
- `middleware.RequireRole("agency", "provider")` — checks the role from context against the allowed list. Returns `403 Forbidden` if not matched.
- Handler-level ownership check: after fetching the resource from the database, verify `resource.UserID == requestingUserID`. This cannot be middleware because the resource ID comes from the URL and must be fetched first.

### Repository-level filtering (primary defense)

- ALL repository methods that return user-specific data MUST accept a `userID` parameter.
- `GetMissions(ctx, userID)` not `GetMissions(ctx)` — the SQL query always includes `WHERE user_id = $1`.
- List endpoints: users only see their own resources unless the resource is explicitly marked as public.
- Admin list endpoints use separate repository methods without the `userID` filter, accessed only through admin-gated routes.
- Never construct a query that returns all rows and then filter in Go. The database does the filtering.

### PostgreSQL RLS (defense-in-depth backup)

- Enable RLS on ALL tables except `users` (the `users` table is accessed by auth flows that run before user context is established).
- Policy pattern: `USING (user_id = current_setting('app.current_user_id', true)::uuid)` — the `true` parameter makes `current_setting` return `NULL` instead of erroring when the setting is not set, which causes the policy to deny access (safe default).
- Set `app.current_user_id` via `SET LOCAL` at the beginning of each request's database transaction. `SET LOCAL` scopes the setting to the current transaction only — it cannot leak to other requests sharing the same connection.
- RLS is a BACKUP — application-level checks (repository `WHERE` clauses + handler ownership checks) are the primary defense. RLS catches bugs in application code.
- Superuser and table owner bypass RLS by default. The application database user must NOT be a superuser and must NOT own the tables. Use a separate migration user for DDL.
- Test RLS in integration tests: attempt to read another user's data and verify it returns zero rows.

### What each role can access

| Resource | Owner | Same role | Other roles | Admin |
|----------|-------|-----------|-------------|-------|
| Own profile | Read/Write | — | Read (public fields only) | Read/Write |
| Own missions | Read/Write | — | — | Read/Write |
| Own messages | Read/Write | — | — | Read |
| Own invoices | Read/Write | — | — | Read |
| Other's public profile | Read | Read | Read | Read/Write |
| Other's missions | — | — | — | Read/Write |

"—" means no access. Any attempt returns `403 Forbidden`. Public fields on profiles are: name, company name, description, avatar URL, role, and average rating. Private fields (email, phone, billing info) are never exposed to other users.

### Critical invariants (never violate these)

1. A user must NEVER see another user's messages, invoices, or private profile data (unless admin).
2. A user must NEVER modify another user's resources (unless admin).
3. Every database query for user-specific data MUST include the `user_id` filter — no exceptions.
4. Admin endpoints are gated by `RequireRole("admin")` AND logged in the audit trail.
5. RLS policies must be tested in integration tests to verify they block cross-tenant access.

---

## Accessibility standards (web)

Minimum compliance: **WCAG 2.1 Level AA**.

| Requirement | Implementation |
|-------------|----------------|
| Keyboard navigation | All interactive elements reachable and operable via Tab, Enter, Escape, Arrow keys |
| Focus indicators | Visible focus ring on all focusable elements. Never `outline: none` without replacement. |
| ARIA labels | Custom components (modals, dropdowns, tabs) must have proper `role`, `aria-label`, `aria-expanded`, etc. |
| Color contrast | Minimum 4.5:1 for normal text, 3:1 for large text (WCAG AA) |
| Screen readers | Test with VoiceOver (macOS) or NVDA (Windows). All content must be announced correctly. |
| Alt text | Every `<img>` has meaningful `alt` text. Decorative images use `alt=""`. |
| Form labels | Every input has an associated `<label>`. No placeholder-only inputs. |
| Error messages | Form errors announced to screen readers via `aria-live` regions or `aria-describedby`. |

---

## API versioning strategy

| Rule | Detail |
|------|--------|
| Versioning scheme | URL-based: `/api/v1/`, `/api/v2/` |
| Breaking changes | Always create a new version. Never break existing clients. |
| Deprecation period | Old versions supported for minimum 6 months after successor release. |
| Deprecation signal | `Deprecation` header in responses with sunset date: `Deprecation: true`, `Sunset: 2026-12-01` |
| Non-breaking changes | Additive changes (new fields, new endpoints) do NOT require a new version. |
| Documentation | Each version has its own OpenAPI schema: `/api/v1/openapi.json`, `/api/v2/openapi.json` |

**What counts as breaking:**
- Removing or renaming a field in a response
- Changing a field type
- Removing an endpoint
- Changing error code semantics
- Making a previously optional field required

**What is NOT breaking:**
- Adding a new field to a response
- Adding a new endpoint
- Adding a new optional query parameter
- Adding a new error code

---

## Observability

### Structured logging

All logs are structured JSON via Go's `slog` package. Every log line includes:

```json
{
  "time": "2026-03-16T10:30:00Z",
  "level": "INFO",
  "msg": "request completed",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "POST",
  "path": "/api/v1/auth/register",
  "status": 201,
  "duration_ms": 42,
  "user_id": "optional-if-authenticated"
}
```

- Every request gets a unique `request_id` (UUID v4) via middleware. Propagated in context.
- All log calls include `request_id` for correlation across the request lifecycle.
- Log levels: `DEBUG` (local only), `INFO` (request flow), `WARN` (recoverable issues), `ERROR` (failures requiring attention).
- Never log sensitive data: passwords, tokens, full credit card numbers, personal identifiers.

### Health endpoints

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `GET /health` | Liveness probe. Is the process running? | `200 OK` with `{"status": "ok"}` |
| `GET /ready` | Readiness probe. Can it serve traffic? | `200 OK` if DB + Redis connected, `503` otherwise |

### Metrics (current and planned)

- **Request duration**: Logged per request. Aggregate with log analysis tools.
- **Error rate**: Track 4xx and 5xx rates. Alert on 5xx spike.
- **DB query duration**: Logged per query in adapter layer.
- **Future**: OpenTelemetry traces for distributed tracing across services. Prometheus metrics export.

---

## Development principles

### Test-Driven Development
- Write tests FIRST or alongside code, never after
- Tests are how AI agents self-correct — they run tests, see failures, and fix autonomously
- Target: 80%+ coverage on business logic layers

### Scalability and Maintainability
- Stateless backend — horizontal scaling ready
- Features are isolated — adding/removing has zero side effects
- Adapters are swappable: change provider = one new file + one line change
- Database migrations with up/down for safe rollbacks
- Redis for session/cache — no in-memory state

### AI-Agent Friendliness
- Consistent patterns across all features — learn one, know all
- Small files with single responsibility — fits in agent context window
- Explicit interfaces — agents see exactly what to implement
- Tests as guardrails — agents validate their own work
- CLAUDE.md at each level describes conventions for that area

---

## Code conventions

### SQL and Migrations
- Pure SQL, no ORM, no query builder. Powered by `golang-migrate`.
- Migrations live in `backend/migrations/`: `001_name.up.sql` / `001_name.down.sql`
- All tables: UUID `id`, `created_at`, `updated_at`
- Use `TEXT` not `VARCHAR`, index foreign keys
- No cross-feature foreign keys (only reference `users` table)
- Migrations are immutable once applied in prod — never edit, only create new ones
- Workflow: create migration -> test locally (`make migrate-up`) -> commit -> apply to prod (`DATABASE_URL=<prod> make migrate-up`)

### Git
- Conventional commits: `feat:`, `fix:`, `refactor:`, `chore:`, `test:`, `docs:`
- One feature per commit, atomic changes
- Never commit secrets, .env files, or node_modules

---

## Running locally

```bash
# Infrastructure (PostgreSQL 16, Redis 7, MinIO)
docker compose up -d

# Apply migrations
cd backend && make migrate-up

# Seed data (admin user, default roles)
cd backend && make seed

# Backend (port 8080)
cd backend && make run

# Web frontend (port 3000)
cd web && npm run dev

# Admin panel (port 5173)
cd admin && npm run dev
```

### Ports reference

| Service    | Port  |
|------------|-------|
| Backend    | 8080  |
| Web        | 3000  |
| Admin      | 5173  |
| PostgreSQL | 5434  |
| Redis      | 6380  |
| MinIO API  | 9000  |
| MinIO UI   | 9001  |

---

## Environment variables

- **Backend**: `DATABASE_URL`, `REDIS_URL`, `PORT`, `JWT_SECRET`, `MINIO_*`, service-specific keys. See `backend/.env.example`.
- **Web**: `NEXT_PUBLIC_API_URL` (default: `http://localhost:8080`)
- **Admin**: `VITE_API_URL` (default: `http://localhost:8080`)
- Never commit `.env` files — they are gitignored

---

## Autonomous work process

### Test tools
- **Backend**: Go `testing` package + `github.com/stretchr/testify` (assertions, mocks)
- **Backend integration**: `testcontainers-go` for PostgreSQL/Redis in tests
- **Web unit**: `vitest` + `@testing-library/react` + `@testing-library/jest-dom`
- **Web E2E**: `playwright` (chromium) — tests in `web/e2e/`

### Test -> Fix -> Retest loop (MANDATORY)

Every piece of code you write must be tested. Follow this loop:

```
1. Write implementation code
2. Write unit tests for that code
3. Run tests
   |-- ALL PASS -> continue to next sub-task
   |-- FAIL ->
       4. Read error output carefully
       5. Fix the bug (in code OR test, whichever is actually wrong)
       6. Rerun tests
       (max 3 fix attempts per failing test)
       Still failing after 3 attempts -> blocker policy below
```

**NEVER commit with failing tests. NEVER delete or skip a test to make the suite pass.**

### Commit strategy
- 1 commit per completed task (not per sub-step)
- Before EVERY commit: run the full validation pipeline below
- Never commit broken code on main
- Conventional messages: `feat:`, `fix:`, `test:`, `refactor:`, `chore:`, `docs:`

### Validation pipeline (run before EVERY commit)

```bash
# 1. Backend compilation
cd backend && go build ./...

# 2. Backend tests
cd backend && go test ./... -count=1

# 3. Web compilation (when implemented)
cd web && npx tsc --noEmit

# 4. Web tests (when implemented)
cd web && npx vitest run

# 5. Architecture checks
# - No cross-feature imports in web/src/features/
# - Migration has both .up.sql and .down.sql
# - No hardcoded secrets in source files
```

ALL steps must pass. If any fails -> enter fix loop above -> only commit when ALL green.

### Blocker policy

**A long task is NOT a blocker.** A blocker = same error, 3+ approaches tried, no progress. Implementing matching logic taking 2h is normal. Stuck on the same error for 20 min is a blocker.

**Type A -- Test failure**: max 3 fix attempts per test -> comment `// TODO: fix` -> log `BLOCKED-taskX.md` -> continue other sub-steps

**Type B -- Same error, no progress**: 3+ different approaches tried, nothing works -> log `BLOCKED-taskX.md` with error + all approaches -> skip only the blocked sub-step (not the whole task) -> commit working code -> move on

**Type C -- Compilation failure**: TOP PRIORITY, 10 min to fix -> if unfixable, revert latest changes (`git checkout -- <files>`) -> NEVER leave build broken

---

## Design system

Primary color: **Rose** (#F43F5E) — warm, distinctive, Malt/Airbnb-inspired.

Full design system documentation: `design/DESIGN_SYSTEM.md`. Read it when working on any UI component.

### Token quick reference (always in context)

| Token | Light | Dark |
|-------|-------|------|
| primary | #F43F5E | #FB7185 |
| background | #FFFFFF | #0F172A |
| foreground | #0F172A | #F8FAFC |
| muted | #F1F5F9 | #1E293B |
| border | #E2E8F0 | #334155 |
| success | #22C55E | #4ADE80 |
| warning | #F59E0B | #FBBF24 |
| destructive | #EF4444 | #F87171 |

### Component rules (compact)

- **Buttons**: 5 variants (primary/secondary/outline/ghost/destructive), 3 sizes (sm/md/lg), rounded-md
- **Cards**: white bg, 1px border, rounded-lg, shadow-sm, p-6. Interactive: shadow-md on hover
- **Inputs**: h-10, rounded-md, focus ring-2 ring-primary. Error: border-destructive
- **Avatars**: rounded-full, 5 sizes (24-64px), initials fallback on primary-100
- **Spacing**: 4px base unit. Only: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128
- **Radius**: sm(6px), md(8px), lg(12px), xl(16px), full(9999px)
- **Shadows**: sm (rest), md (hover), lg (modal)
- **Transitions**: 150ms ease-out everywhere
- **Loading**: skeleton matching content shape, NEVER full-page spinner
- **Role badges**: agency (blue), enterprise (purple), provider (rose), admin (slate)
- **Icons**: Lucide (web/admin), 18px inline / 20px buttons / 24px standalone

---

## Compact instructions

When compacting context, prioritize preserving:
1. The current task being worked on (task number and sub-step)
2. Any errors or blockers encountered
3. Files recently created or modified
4. The test->fix->retest loop state (what's failing, what was tried)

After compaction:
1. Re-read task list to recover full progress
2. Run `git log --oneline -10` to see what was already committed
3. Run `cd backend && go build ./...` to verify project compiles
4. Check for `BLOCKED-*.md` files at project root
5. Resume from first unchecked task
