# F.3.1 — Direct path to TOP-1% global (3 security HIGH + 3 maintainability polish)

**Phase:** F.3.1 (chemin direct top-1% sur les 7 axes)
**Source audit:** PR #92 (final verification audit) + 7e axe maintainability
**Effort:** ~3-4h réel attendu
**Tool:** 1 fresh agent dispatched
**Branch:** `feat/f3-1-publish-ready`

## Goal

Close the 3 HIGH security findings flagged by the final audit + add the 3 maintainability polish items (ADRs + CHANGELOG + pre-commit hooks) so the codebase reaches **TOP 1% on all 7 axes** and is publish-ready as open-source.

## 6 deliverables (LOCKED — user validated)

### 1. SEC-FINAL-07 — Admin token in `localStorage` → memory-only

**Problem** : `admin/src/shared/lib/api-client.ts:25` stores the admin auth token in `localStorage`. Any XSS in an admin transitive dep = instant token siphon. The user's other products (open-source) reading the same code can be probed for the same flaw.

**Fix decision** : token in **React in-memory state via Zustand store** (NOT persisted). Page reload requires re-login (acceptable for admin-only surface — not a daily user flow). Refresh token mechanism stays via httpOnly cookie if any.

**Implementation** :
- `admin/src/shared/lib/api-client.ts` : drop `localStorage.getItem("token")` → read from Zustand `useAuthStore`
- `admin/src/shared/stores/auth-store.ts` : in-memory only, NO `persist` middleware
- Page reload : intercept `401` → redirect to login (already handled? verify)
- Backward-compat : on app boot, attempt cookie-based session restore via existing `/api/v1/auth/me` if cookie present
- Tests : `admin/src/shared/__tests__/auth-store.test.tsx` — token read/write, no persist after reload simulation

### 2. SEC-FINAL-04 — SSRF via `ValidateSocialURL`

**Problem** : `backend/internal/domain/profile/social_link.go:88` validates URLs but accepts:
- `http://10.0.0.1` (private RFC1918)
- `http://169.254.169.254` (cloud metadata — AWS/GCP/Azure)
- `[::1]`, `127.0.0.1` (loopback)
- `0177.0.0.1` (octal-encoded loopback)
- `2130706433` (decimal-encoded loopback)

**Fix decision** : explicit denylist of:
- All RFC1918 ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
- Loopback (127.0.0.0/8, ::1/128)
- Link-local (169.254.0.0/16, fe80::/10)
- Multicast (224.0.0.0/4, ff00::/8)
- Reserved (0.0.0.0/8)

Reject `javascript:` `data:` `vbscript:` `file:` schemes.

Resolve hostname via `net.LookupIP` and validate every returned IP (DNS rebinding mitigation).

**Test required** : `social_link_test.go::TestValidateSocialURL_RejectsSSRFVectors` — 15+ cases including each form of encoding (decimal, octal, hex, IPv6 mapped).

### 3. SEC-FINAL-03 — `RequireRole` middleware referenced but absent

**Problem** : 2 handler comments reference `middleware.RequireRole` but `grep -rn "RequireRole" backend/internal/handler/middleware/` returns 0. Defense-in-depth gap.

**Fix decision** : implement `RequireRole(roles ...string)` middleware in `backend/internal/handler/middleware/authorization.go` that:
- Reads role from context (set by `Auth` middleware)
- Returns `403 Forbidden` with code `insufficient_role` if not matched
- Logs the denial event via `slog.Warn`

Wire it on the 2 routes that reference it (find via grep) + audit all admin routes to ensure they ALL use it (currently maybe relying on handler-level checks).

**Test required** : `middleware/authorization_test.go::TestRequireRole_*` — 8 cases (matched, mismatched, missing context, multiple allowed roles, role hierarchy if any).

### 4. ADR system (Architecture Decision Records)

**Problem** : top-1% projects have ADRs documenting WHY architectural decisions were made. Currently absent.

**Decision** : create `docs/adr/` with the Markdown ADR template (Michael Nygard format). Write 8 ADRs based on real decisions visible in the codebase:

1. `0001-hexagonal-architecture.md` — Why hexagonal/ports-and-adapters
2. `0002-org-scoped-business-state.md` — Why business state on `organization_id`, not `user_id`
3. `0003-postgresql-rls-with-system-actor.md` — Why RLS + system-actor split, not just app-level checks
4. `0004-stripe-webhooks-async-via-outbox.md` — Why pending_events + worker, not synchronous dispatch
5. `0005-opentelemetry-otlp-no-vendor-lock-in.md` — Why OTLP exporter standard
6. `0006-feature-isolation-no-cross-feature-imports.md` — Why ESLint-enforced
7. `0007-soft-delete-30-days-rgpd-window.md` — Why 30 days vs instant
8. `0008-cursor-pagination-no-offset.md` — Why cursor over OFFSET

Each ADR: Context / Decision / Consequences / Alternatives considered / References.

### 5. CHANGELOG.md + SemVer release `v1.0.0-rc.1`

**Problem** : 1 git tag (`v0.9-kyc-custom-final`), no SemVer discipline, no CHANGELOG.md.

**Fix** :
- Create `CHANGELOG.md` in Keep-a-Changelog format (https://keepachangelog.com/)
- Group by version: `[Unreleased]`, `[1.0.0-rc.1]` (today), and historical sections derived from `git log --oneline | grep -E "^[a-f0-9]+ (feat|fix)"`
- Tag `v1.0.0-rc.1` after the F.3.1 PR merges (note in PR description, not in this branch — the user creates the tag manually after merge)

### 6. Pre-commit hooks

**Problem** : no client-side enforcement of conventions. CI catches but slow feedback loop.

**Decision** : simple bash hook `.githooks/pre-commit` (NOT husky — keeps deps minimal):
- Run `gofmt -d` on changed `.go` files (fail on diff)
- Run `npx tsc --noEmit` if any `web/` or `admin/` `.ts`/`.tsx` changed
- Run `flutter analyze` on changed mobile files
- Skip via `git commit --no-verify` (escape hatch documented in CONTRIBUTING.md)

Setup script `scripts/install-git-hooks.sh` symlinks `.githooks/*` to `.git/hooks/`. Document in CONTRIBUTING.md so contributors run it once.

## Plan (6 commits, atomic)

1. `fix(admin/security): move auth token to in-memory Zustand store`
2. `fix(profile/security): SSRF guard on ValidateSocialURL — block private/loopback/metadata IPs`
3. `feat(middleware/security): implement RequireRole + wire on admin routes`
4. `docs(adr): add 8 architecture decision records`
5. `docs: add CHANGELOG.md + bump version metadata`
6. `chore: add pre-commit hooks + install script`

## Hard constraints

- **Validation pipeline before EVERY commit**:
  ```bash
  cd backend && go build ./... && go vet ./... && go test ./... -count=1 -short -race
  cd web && npx tsc --noEmit && npx vitest run
  cd admin && npx tsc -b --noEmit && npx vitest run
  cd mobile && flutter analyze lib && flutter test
  ```
- **Tests for security fixes** (paranoid): each of #1, #2, #3 must have a test that FAILS on origin/main and PASSES on the branch (demonstrate via stash trick in PR description).
- **No scope creep** : exactly the 6 deliverables above. Other audit findings (DRY 467 paths, mobile dynamic regression, etc.) → flag in PR description for F.3.2/F.3.3.
- **Branch ownership** : only `feat/f3-1-publish-ready`.

## Tests required

- Security #1: 4 tests (write/read, no-persist-after-reload, 401 redirect, cookie restore on boot)
- Security #2: 15 tests (each SSRF vector + valid URL pass-through)
- Security #3: 8 tests (matched/mismatched/missing/multiple/admin/owner/etc.)
- ADRs: 0 tests (docs)
- CHANGELOG: 0 tests
- Pre-commit hooks: 1 test (`scripts/test-pre-commit.sh` simulates a `gofmt` violation, asserts hook rejects)

## OFF-LIMITS

- LiveKit / call code, workflow files, other plans
- Other audit findings beyond the 6 above (flag, don't fix)

## Final report (under 800 words)

Lead with PR URL.

1. Each of the 6 deliverables : status (done / partial / blocked)
2. Tests added (count + names)
3. Coverage delta on touched files
4. Validation pipeline output
5. **Top-1% verdict claim**: "After this PR merges, the codebase achieves TOP 1% on Architecture / Security / Maintainability. Other axes per audit." — justified with evidence
6. F.3.2/F.3.3 backlog flagged for follow-up
7. "Branch ownership confirmed: only worked on `feat/f3-1-publish-ready`"

GO. Quality > speed. The user is publishing this open-source — every line will be public-scrutinized.
