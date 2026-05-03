# F.5 — Security hardening + CI rigueur + honest update

**Phase:** F.5 — last technical pass before public OSS
**Source:** independent adversarial audit by external Claude agent (post-F.4)
**Effort:** ~2-3j est.
**Tool:** 1 fresh agent dispatched
**Branch:** `feat/f5-security-and-honesty`

## Context

An independent Claude agent ran 5 parallel adversarial audits (backend Go, web Next.js, mobile Flutter, security paranoid, architecture) with REAL tool execution (gosec, go test -race, flutter analyze, npm audit, tsc, quantitative greps). The verdict :
- Most internal "TOP 1%" claims are over-rated by **one tier**
- 8 NEW security findings (S1-S8) the internal audits missed
- 14+ STUPID anti-pattern violations (cross-feature backend imports) that internal audits claimed = 0
- CI uses `continue-on-error: true` everywhere — gates nothing

**The honest verdict** : top 5% solo OSS, top 10-15% vs funded SaaS — NOT "TOP 1% mondial". This brief closes the real gaps + replaces marketing claims with honest documentation.

## Mission — 13 deliverables

### A. Security HIGH (must-close before publish)

#### S1 — RLS WITH CHECK on 8 tables
m.129 added `WITH CHECK` only on `audit_logs`. The 8 other tenant-scoped tables have only `USING` policy — masked today by prod role `BYPASSRLS`, but a future rotation to `NOBYPASSRLS` (documented goal in `backend/docs/rls.md`) silently breaks system writes.

**Fix** : new migration `135_rls_with_check_all_tables.up.sql` adds `WITH CHECK` clauses identical to USING on : `conversations`, `messages`, `invoice`, `proposals`, `proposal_milestones`, `notifications`, `disputes`, `payment_records`. Symmetric down.sql.

**Test** : extend `backend/internal/adapter/postgres/rls_test.go` integration tests to cover INSERT/UPDATE under NOBYPASSRLS test role on all 9 tables.

#### S2 — Refresh-token replay : revoke entire family via `BumpSessionVersion`
`app/auth/service_more.go:46-54` detects replay (already-blacklisted refresh token used) → audits + returns 401 — but does NOT bump session_version. Per RFC OAuth 2.1 §4.13.2, full token family must be revoked on detected replay.

**Fix** : after `s.refreshBlacklist.Has(...)` returns true, call `s.users.BumpSessionVersion(ctx, userID)`. The next access-token validation against session_version will fail → forces full re-login.

**Test** : `service_test.go::TestRefresh_ReplayRevokesEntireFamily` — capture an access+refresh pair, replay the refresh, assert the captured access is now invalid (session_version mismatch).

#### S3 — npm audit fix lockfiles
- `web/package-lock.json` : GHSA-mq59-m269-xvcx (`next` HIGH — Server Actions CSRF bypass via Origin null)
- `admin/package-lock.json` : `lodash` HIGH

**Fix** : `cd web && npm audit fix` + `cd admin && npm audit fix`. If breaking changes are required, document the bump path. If `npm audit` reports no fix path (transitive locks), document the reason.

**Test** : after fix, `npm audit --audit-level=high` returns 0 vulnerabilities.

#### S4 — Stripe error leak — 3 sites remaining
F.4 #7 closed `embedded_handler.go:140, 246`. Audit independent flagged 5 total — remaining sites at lines 173, 180, 208, 235 (verify exact lines on current main).

**Fix** : same `classifyStripeError()` helper applied to the remaining sites.

**Test** : extend existing `stripe_error_sanitizer_test.go` to cover the 3 new call sites.

### B. Security MEDIUM (important — close before publish)

#### S5 — `/register` email enumeration
`app/auth/service.go:199-201` returns `ErrEmailAlreadyExists` (typed) when an email is already registered. Forgot-password is correctly silent. Register breaks the mitigation.

**Fix** : on duplicate email, return generic 200/202 with a neutral message ("Check your email for confirmation"). Send a notification email to the existing-account owner (security signal). The legitimate flow continues via the email link.

**Test** : `service_test.go::TestRegister_DoesNotEnumerate` — register twice with same email, assert response shape identical between fresh + dup, assert security email sent on dup.

#### S6 — IPv6 /64 normalization in rate limiter
`middleware/ratelimit.go:136-159` keys on the full IPv6 — attacker with /64 routed has 2^64 slots.

**Fix** : detect IPv6 in the key extractor, mask to /64 (128 → 64 bits). For IPv4, keep /32. Document in `middleware/ratelimit.go` doc comment.

**Test** : `ratelimit_test.go::TestRateLimit_IPv6_NormalizesTo64` — 65 distinct IPv6 addresses in the same /64 hit the same bucket.

#### S7 — Rate-limit + brute-force fail-CLOSED in production
`ratelimit.go:189-191` and `bruteforce.go:120-124` fail-OPEN on Redis error. A Redis blip silently disables both throttling and login lockout.

**Fix** : env-aware policy. In `cfg.IsProduction()`, fail-CLOSED (return 503 Service Unavailable). In dev, fail-OPEN with `slog.Error` for visibility. Add Prometheus counter `rate_limit_redis_fail_count` for alerting.

**Test** : `ratelimit_test.go::TestRateLimit_FailClosedInProd` + `TestRateLimit_FailOpenInDev` (with mocked cfg).

#### S8 — `verifySessionVersion` fail-CLOSED in production
`middleware/auth.go:184-187 + :237-243` fall back to snapshot on DB/Redis error. An attacker who triggers the DB incident bypasses permission revocation.

**Fix** : same env-aware policy as S7. Production = 503 on lookup failure. Add metric.

**Test** : `auth_test.go::TestAuth_FailClosedInProdOnLookupError`.

### C. Bugs HIGH (genuine attack surface)

#### B1 — DisallowUnknownFields + MaxBytesReader on 13 handlers
13 POST handlers contour the standard pattern of `json.NewDecoder(r.Body).DisallowUnknownFields()` + `http.MaxBytesReader(...)`. Per audit indep : `admin_handler.go:129,188,369` (suspend/ban !), `admin_credit_note_handler.go:117`, `admin_team_handler.go:64,105`, `billing_profile_handler.go:112`, `subscription_handler.go:128,209,236`, `skill_handler.go:156,191`, `health_handler.go:101`.

**Fix** : create a tiny `pkg/decode/json.go` helper `func DecodeBody(r *http.Request, v any, maxBytes int64) error` that wraps the standard pattern. Sweep the 13 sites to use it. Document the convention in `backend/CLAUDE.md`.

**Test** : `pkg/decode/json_test.go` — 5+ cases (over-cap, unknown-field, valid, empty, malformed).

#### B12 — CI hardening (no more theatre)
- `.github/workflows/ci.yml:158-167` ESLint job has `continue-on-error: true || true` — never gates
- `web-build` uses `NEXT_PUBLIC_API_URL: http://localhost:8080` (wrong port — backend is 8083)
- `mobile-analyze` covers only 3 dirs out of ~33
- `gosec` runs with `-no-fail || true` (`security.yml:64`) — never blocks

**Fix** : remove every `continue-on-error: true` and `|| true` that should gate. Promote ESLint, gosec to fail-on-error. Fix port to 8083 (or use `${{ secrets.CI_API_URL }}`). Expand mobile-analyze scope.

**⚠️ Workflow file constraint** : token can't push `.github/workflows/*`. Agent must create the diff as a patch file at `/tmp/ci-hardening.patch` for the user to apply manually via UI GitHub. Document the intent in commit message.

### D. Honest documentation pass

#### D1 — Update audit files
- `auditsecurite.md` : add S1-S8 with severity + CWE + fix steps. Delete items closed by F.1-F.5.
- `auditqualite.md` : add the 14+ cross-feature backend violations. Add the 26/33 mobile features bypassing Clean Architecture.
- `bugacorriger.md` : add B1, B2 (N+1 sites), B3 (38 migrations sans IF NOT EXISTS), B4 (logger cardinality), B5-B11 (mobile + web bugs), B12, B13, B14.
- `auditperf.md` : note the 8 forms with raw useState (not 6), the layout `(app)`/`(public)` "use client" issue.
- `rapportTest.md` : note the 3 race-flake tests, mobile test/integration FAIL on cold compile.
- `ROADMAP_FINALE.md` : update verdict to **"TOP 5% solo OSS, TOP 10-15% vs funded SaaS"** (replace "TOP 1% mondial" claims).

#### D2 — README honest update
Replace "TOP 1% on 6/7 axes" with :
> *Senior-grade engineering primitives (RLS + audit append-only + refresh rotation + magic-byte uploads + SSRF guard + GDPR Art. 15-17 + OTel) actively developed for an open-source B2B marketplace. Top 5% solo OSS audit verdict. Battle-test pending — production usage, chaos engineering, and SLO docs are post-launch goals.*

Honnêteté = top-tier signal. Auto-flatterie = junior signal.

#### D3 — ADR review
- ADR-0001 (Hexagonal) : if it claims "every feature deletable", correct (proposal cascades, moderation depended-on by 6 services)
- ADR-0006 (Feature isolation) : note the 14 cross-feature backend imports that violate it — open as F.6 follow-up
- Other ADRs : factual sanity check vs actual code

#### D4 — `web/CLAUDE.md` correction
Lines describing a "Zustand auth store" that doesn't exist — auth is httpOnly cookie + TanStack Query. Correct or remove.

#### D5 — `backend/CLAUDE.md` ADR contradiction
Line 1166 says `mock/` was abandoned. ADR-0001 says it exists. Pick reality (it doesn't), update ADR-0001.

## Hard constraints (paranoid mode)

- **Validation pipeline before EVERY commit**:
  ```bash
  cd backend && go build ./... && go vet ./... && go test ./... -count=1 -short -race
  cd web && npx tsc --noEmit && npx vitest run
  cd admin && npm install && npx tsc -b --noEmit && npx vitest run
  ```

- **Migration 135 (S1) safety** : run up + down + up locally on the marketplace_go test DB to verify idempotent.
- **`/register` neutral response (S5)** : behavior change — must NOT break existing tests. Update fixtures + tests as needed.
- **CI workflow patch (B12)** : NEVER push `.github/workflows/*`. Create a patch file at `/tmp/ci-hardening.patch` instead. Document in PR description that user must apply via UI GitHub.
- **README honesty (D2)** : the wording matters. Avoid "world-class", "best-in-class", "TOP 1%", "battle-tested" (when not true). Use precise language.

## Tests required per item

- S1 : 8 RLS WITH CHECK integration tests under NOBYPASSRLS role
- S2 : 1 family-revocation test
- S3 : 0 (lockfile bumps)
- S4 : 3 new sanitizer test cases
- S5 : 1 enum-resistance test (+ 1 security-email-on-dup test)
- S6 : 1 IPv6 /64 normalization test
- S7 : 2 fail-closed-prod / fail-open-dev tests
- S8 : 1 fail-closed-prod test
- B1 : 5+ decode helper tests + assertion that 13 handlers use it
- B12 : 0 (CI changes)

Total : ~22 new tests minimum.

## OFF-LIMITS

- LiveKit / call code (never touch)
- `.github/workflows/*` direct push (use patch file)
- Other plans
- Mobile / admin code refactor beyond doc updates (item D1/D4)
- Performance optimisations beyond fixing what's flagged

## Branch ownership

`feat/f5-security-and-honesty` only. Created from main via `git worktree add`.

## Final report (under 1500 words)

Lead with PR URL.

1. Per-deliverable status (S1-S8 + B1 + B12 + D1-D5) with commit ref
2. Tests added (count + names per item)
3. Migration 135 verified up + down green
4. `npm audit --audit-level=high` post-fix : 0 vulnerabilities
5. Validation pipeline output (full per stack)
6. CI patch file location + contents summary
7. Audit file deltas (lines deleted closed items, lines added new findings)
8. README before/after diff for the verdict claim
9. **Honest verdict claim** : "After this merges, the codebase reaches TOP 5% solo OSS / TOP 10-15% vs funded SaaS, with documented gaps in F.6 backlog. Battle-test pending."
10. Branch ownership confirmed
11. F.6 backlog flagged (any item that surfaced but wasn't in scope)

GO. Take time on S1 (multi-table migration) + B1 (sweep 13 handlers). Items D1-D5 are the harder cultural shift — be honest, not flattering.
