# Security Policy

The marketplace is a payments-handling B2B platform; we treat security as
a first-class concern, not a checkbox. This document explains how to
report a vulnerability, which versions receive fixes, and what we have
already done in-tree to keep the attack surface small.

---

## Reporting a vulnerability

**Please do not open a public GitHub issue for security problems.**
Instead, send a private report to:

- **Email**: `hassad.smara69@gmail.com`
- **Subject prefix**: `[SECURITY] <one-line summary>`
- **Encryption (optional)**: include your PGP key in the first message
  if you want subsequent replies encrypted; we will respond in kind.

Include in your report:

1. A description of the issue and the impact you believe it has.
2. The exact endpoint, file, or commit affected (a permalink to a line
   in `main` is ideal).
3. A minimal reproduction — request payload, expected vs. actual
   response, or a small script.
4. Your assessment of severity (CVSS optional but appreciated).
5. Whether you intend to disclose publicly and on what timeline.

### Response timeline

| Step | Target |
|------|--------|
| Initial acknowledgement of receipt | within **72 hours** |
| Triage + first technical response | within **5 business days** |
| Patch released for CRITICAL / HIGH | within **14 days** of triage |
| Patch released for MEDIUM / LOW | within **60 days** of triage |
| Public disclosure (CVE if applicable) | coordinated, after patch |

If we miss any of these targets we will say so explicitly in the
thread and explain why. Silence is not the answer.

We do **not** currently run a paid bug bounty programme. We will
publicly acknowledge any reporter who wants credit (in the release
notes and in `SECURITY.md`'s "Hall of fame" once we have entries).

---

## Supported versions

The project ships from `main`. We provide security fixes only for
the most recent minor release line — older releases must be upgraded.

| Version | Released | Security fixes |
|---------|----------|----------------|
| `main` (rolling) | continuous | yes |
| `1.x` (latest minor) | TBD at first tag | yes, until next minor + 60 days |
| `< 1.x` | — | no — please upgrade |

When the first stable tag is cut, this table will be updated with
concrete dates. Until then, deploy from `main` at a known commit SHA
and pin via a Git tag in your fork.

---

## Out-of-scope

The following are **not** considered vulnerabilities for the purposes
of this policy:

- Findings on third-party services we depend on (Stripe, LiveKit,
  Cloudflare R2, Resend) — please report those upstream.
- Issues that require physical access to the user's device.
- Self-XSS where the attacker is the same user already authenticated.
- Missing security headers on documentation-only routes.
- Volumetric DoS (we do not have rate-budget guarantees against
  motivated attackers; we do enforce per-IP and per-user limits — see
  below).
- Findings that only reproduce in development mode (`ENV=development`)
  with `STORAGE_USE_SSL=false` and similar relaxed settings.

---

## Track record (what is already in-tree)

We have shipped six audit phases between February and April 2026. Each
was orchestrated through parallel agents and validated against
`go test`, `vitest`, and `flutter test` before merging. The numbers
below are from the `auditsecurite.md` and `rapportTest.md` artefacts at
the project root and from the migration log in `backend/migrations/`.

### Closed findings (Phase 1 — security)

- **40+** security findings closed across `auditsecurite.md` (9
  CRITICAL, 15 HIGH, 10 MEDIUM, 6 LOW).
- **Brute force protection**: 5 failed logins per email per 15 min
  triggers a 30 min lockout in Redis. Implementation in
  `backend/internal/app/auth/bruteforce_*.go` and tests in
  `internal/app/auth/bruteforce_test.go`.
- **Refresh token rotation**: every `/api/v1/auth/refresh` mints a new
  pair, blacklists the old `jti` in Redis (TTL = remaining lifetime);
  reuse returns 401. Code: `internal/app/auth/service.go`,
  `internal/adapter/redis/token_blacklist.go`.
- **Audit log append-only**: `audit_logs` is gated by an
  application-level Postgres role with INSERT/SELECT only — no UPDATE,
  no DELETE. Migration `124_audit_logs_grants.up.sql` documents the
  REVOKE.
- **HTTP security headers** middleware: CSP, HSTS, X-Frame-Options,
  X-Content-Type-Options, Referrer-Policy, Permissions-Policy. Code in
  `internal/handler/middleware/security_headers.go`. The CSP rules are
  reproduced in `CLAUDE.md` for traceability.
- **Webhook idempotency**: Stripe webhooks dedupe via the
  `stripe_webhook_events` table (UNIQUE on `event_id`) with a Redis
  fast-path for hot events.
- **Streaming uploads**: multipart bodies are processed via
  `MultipartReader` so a 100MB file no longer pins 100MB of heap.
- **CORS**: explicit allow-list, no wildcards in production,
  `Vary: Origin` set correctly, and credentials only when both the
  origin matches and the route opts in.

### Closed findings (Phase 1.5 — defense in depth)

- **Row-Level Security** (`SEC-10`): `migrations/125_enable_row_level_security.up.sql`
  enables `ROW LEVEL SECURITY` plus `FORCE ROW LEVEL SECURITY` on **9
  tenant-scoped tables** — `conversations`, `messages`, `invoice`,
  `proposals`, `proposal_milestones`, `notifications`, `disputes`,
  `audit_logs`, `payment_records`. Policies key on
  `current_setting('app.current_org_id', true)` (or `app.current_user_id`
  for per-actor tables) with `true` so an unset context returns NULL
  and rows are filtered out — fail-closed by default.
- **Tenant context helpers**: `port/repository.TxRunner.RunInTxWithTenant`
  + `internal/adapter/postgres/tenant_context.go` set the GUC at the
  beginning of every business transaction. Cross-tenant denial proven
  by the integration tests in `internal/adapter/postgres/rls_isolation_test.go`.
- **Production DB user split**: documented in `backend/docs/rls.md` —
  the application user is non-superuser and does not own the tables;
  DDL runs under a separate migration role that bypasses RLS only at
  schema-change time.

### Continuous scanning (CI)

| Tool | When it runs | Severity gate | Workflow |
|------|--------------|----------------|----------|
| `gosec` (Go static) | every PR + weekly | HIGH/CRITICAL fail with allowlist | `.github/workflows/security.yml` |
| `govulncheck` (Go CVE DB) | every PR | any vulnerable symbol fails the build | `.github/workflows/ci.yml` |
| `trivy fs` (Go deps) | every PR + weekly + push to main | HIGH/CRITICAL → SARIF to Code Scanning | `.github/workflows/security.yml` |
| `trivy config` (Dockerfile) | every PR if Dockerfile changed | HIGH/CRITICAL | `.github/workflows/security.yml` |
| `npm audit` (web + admin) | every PR + weekly | HIGH advisory | `.github/workflows/security.yml` |
| `flutter pub outdated` | weekly | informational | `.github/workflows/security.yml` |
| `semgrep r/golang.security` | every PR touching `backend/**` | HIGH severity fails | `.github/workflows/security.yml` |
| `eslint-plugin-security` (web + admin) | every PR | warnings surfaced in review | `.github/workflows/security.yml` |
| Playwright e2e (SQL injection / open redirect / G120) | PRs with `run-e2e` label + push to main | failure blocks deploy | `.github/workflows/e2e.yml` |
| RBAC matrix smoke (`./scripts/ci/rbac-matrix.sh`) | PRs with `run-e2e` label | any 200 where 403 expected fails | `.github/workflows/e2e.yml` |

The `gosec` baseline went from **35+ findings** in Phase 1 to **3
documented false positives** — annotated inline with `#nosec` and
explained in `docs/ops.md`. We treat any new finding as a regression.

GitHub Code Scanning, Dependabot, and Secret Scanning are configured
on the repo. The Dependabot config (`.github/dependabot.yml`)
auto-opens PRs for Go, npm, pub, and GitHub Actions updates.

---

## Defense in depth

We do not rely on any single layer:

1. **Authentication** — short-lived JWT access tokens (15 min) plus
   rotated refresh tokens (7 days) with Redis blacklist on logout.
2. **Authorization** — three checkpoints per request: JWT validation,
   role middleware, handler-level ownership check.
3. **Repository filters** — every query for tenant-scoped data
   includes `WHERE organization_id = $1` or
   `WHERE user_id = $1`. The DB does the filtering, not Go.
4. **PostgreSQL RLS** — fail-closed backstop on the 9 tables above.
   A bug in repository filtering cannot leak rows because the
   database itself rejects them.
5. **Audit log** — append-only, REVOKE'd of UPDATE/DELETE, captures
   `login_*`, `logout`, `password_reset_*`, `token_refresh`,
   `authorization_denied`, and all data mutations.

If any one layer has a bug, the next one catches it. RLS testing is
mandatory — see `internal/adapter/postgres/rls_isolation_test.go`.

---

## Disclosing fixed vulnerabilities

Once a fix lands on `main`:

1. We open a security advisory on GitHub (private until disclosure).
2. We request a CVE through GitHub's CNA if the issue qualifies.
3. We publish a `SECURITY.md` advisory entry (linked from the
   release notes) with: affected versions, reporter credit (with
   permission), CVSS vector, and mitigation steps for users who
   cannot upgrade immediately.
4. We tag the fix commit with `security/<advisory-id>` for grep-ability.

---

## Hall of fame

_Empty for now — be the first._

We will list reporters here (with permission and a link of their
choice) once the first external advisory is resolved.
