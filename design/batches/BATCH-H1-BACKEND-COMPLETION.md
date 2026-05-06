# BATCH-H1-BACKEND-COMPLETION — close V8 NEW-1 to NEW-6 + ci.yml port fix

> Worktree: `/tmp/mp-h1-backend` · Branch: `fix/h1-backend-completion` · Base: `origin/main` (133a3047)

## Goal — close V8 backend NEW findings + ci.yml port fix, ONE squashed commit

### NEW-1 (MEDIUM) — Complete V5-4: RecordFailure/RecordSuccess WithTimeout
- `backend/internal/adapter/redis/bruteforce.go` — `IsLocked` and `IsIPLocked` are wrapped with `redisCallTimeout` (200ms) (closed in PR #152)
- BUT `RecordFailure` (Lua-script EVAL) and `RecordSuccess` (DEL) are NOT wrapped → same brown-out on write path
- Fix: wrap both with `context.WithTimeout(parentCtx, redisCallTimeout)` like the read functions
- ~5-10 LOC

### NEW-2 (MEDIUM) — Move `CacheError` from adapter to port
- `backend/internal/app/webhookidempotency/claimer.go:35` imports `internal/adapter/redis` to type-assert `*redis.CacheError`
- Violates app→port layer rule (app should never import adapter)
- Fix: move the `CacheError` type to a port (e.g. `backend/internal/port/cache/errors.go` or wherever cache errors should live) so app code can type-assert without importing adapter
- Update the redis adapter to return the port-level error type
- Update the import in claimer.go
- ~10-15 LOC + careful import refactor

### NEW-3 (LOW) — slog.Warn for dropped IDs in NF-12 resolver
- `backend/internal/app/referral/wiring_adapters.go:202-209` (or thereabouts) — when the resolver drops a foreign proposal ID, it logs at `slog.Debug` level
- Bug-or-abuse signal deserves `slog.Warn` with `dropped_ids count` field so it's visible in prod logs
- 1 LOC fix (slog.Debug → slog.Warn) + add `slog.Int("dropped_ids", len(dropped))` field

### NEW-4 (LOW) — Extract generic `coalesceWithDoubleCheck[T]` helper
- The V6-1 fix (double-check INSIDE `singleflight.Group.Do` callback) is duplicated across 4 caches:
  - `backend/internal/adapter/redis/freelance_profile_cache.go`
  - `backend/internal/adapter/redis/expertise_cache.go`
  - `backend/internal/adapter/redis/skill_catalog_cache.go`
  - `backend/internal/adapter/redis/profile_cache.go`
- Rule of three triggered. Extract a generic helper `coalesceWithDoubleCheck[T any](sf *singleflight.Group, key string, load func() (T, error), peek func() (T, bool)) (T, error)` into a new package `backend/pkg/redisutil/` or `backend/internal/adapter/redis/coalesce.go`
- Refactor the 4 caches to use it. Behavior identical, fewer LOC, less duplication
- ~50-80 LOC total (helper + refactor)

### NEW-5 (LOW) — Accept-Encoding in idempotency cache key
- `backend/internal/handler/middleware/idempotency.go` — `Content-Encoding` is now whitelisted (closed in #152) BUT `Accept-Encoding` is not part of the cache key
- Effect: a client that sends `Accept-Encoding: gzip` triggers a cached gzipped response. A second client with same Idempotency-Key but no Accept-Encoding gets the gzipped response anyway → decode fails
- Two options:
  - (a) Include `Accept-Encoding` in the cache key (separate cache entries per encoding)
  - (b) Document explicitly that idempotency replays preserve the original response encoding (callers must handle)
- Pick (a) — safer. Update the key construction to hash include normalized `Accept-Encoding` (e.g. "gzip" or "identity")
- ~10 LOC

### NEW-6 (INFO) — Refresh audit docs
- `auditperf.md`, `auditqualite.md`, `auditsecurite.md`, `bugacorriger.md`, `rapportTest.md` are all frozen at 2026-05-04 13:27 (per `mtime`) despite 4 PRs "close V7 NF-*" in the meantime
- Refresh each doc to reflect current state:
  - List the V7 + V8 fixes shipped (PRs #144 to #152 plus this H1 batch when merged)
  - Update aggregate metrics (gosec 0 issues, race tests green, residual rose -96%, etc.)
  - Mark resolved findings as resolved with PR refs
  - Mark new findings (V8 NEW-1 to NEW-6 + ongoing tier-1 mobile cert pinning, etc.) as deferred or in-progress
- This is honesty hygiene — claim "TOP 5%" without docs reflecting it = dishonest

### ci.yml port fix
- `.github/workflows/ci.yml:226` (or wherever) sets `NEXT_PUBLIC_API_URL: localhost:8080` — should be `localhost:8083` per the project convention
- 1 LOC fix

## TOUCHABLE files

- `backend/internal/adapter/redis/bruteforce.go` (NEW-1)
- `backend/internal/adapter/redis/{freelance_profile,expertise,skill_catalog,profile}_cache.go` (NEW-4 refactor)
- `backend/internal/adapter/redis/coalesce.go` OR `backend/pkg/redisutil/coalesce.go` (NEW-4 helper, NEW file)
- `backend/internal/app/webhookidempotency/claimer.go` (NEW-2 import update)
- `backend/internal/port/cache/errors.go` OR similar (NEW-2 NEW file)
- `backend/internal/adapter/redis/cache_error.go` (NEW-2 — make existing CacheError satisfy port interface or move)
- `backend/internal/app/referral/wiring_adapters.go` (NEW-3 slog.Warn)
- `backend/internal/handler/middleware/idempotency.go` (NEW-5 Accept-Encoding)
- `backend/internal/handler/middleware/idempotency_test.go` (NEW-5 test)
- `auditperf.md`, `auditqualite.md`, `auditsecurite.md`, `bugacorriger.md`, `rapportTest.md` (NEW-6 refresh)
- `.github/workflows/ci.yml` (port fix)

## OFF-LIMITS — STRICT
- ALL web/admin/mobile files
- Database migrations
- `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`
- `package.json`, lockfiles, go.mod (unless strictly needed for NEW-2 import path — unlikely)
- Sibling agents H2 (mobile), H3 (admin) scopes

## Acceptance criteria
- `cd backend && go build ./...` clean
- `cd backend && go vet ./...` clean
- `cd backend && go test ./... -count=1 -timeout=180s` PASS
- `cd backend && go test -race ./... -count=1 -timeout=180s` PASS
- `cd backend && gosec -exclude-dir=mock ./...` 0 issues
- The 4 cache files all use the new `coalesceWithDoubleCheck` helper instead of inline double-check
- `webhookidempotency/claimer.go` no longer imports `internal/adapter/redis`
- Audit docs reflect current state (resolved V7+V8 findings, current open list)
- `ci.yml` port = 8083

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-h1-backend
git diff --name-only origin/main...HEAD | grep -E "^(web/|admin/|mobile/|backend/migrations/)" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"
cd backend
go build ./... 2>&1 | tail
go vet ./... 2>&1 | tail
go test ./... -count=1 -timeout=180s 2>&1 | tail -10
go test -race ./... -count=1 -timeout=240s 2>&1 | tail -10
gosec -exclude-dir=mock ./... 2>&1 | tail -5
cd ..
bash design/scripts/check-api-untouched.sh
```

ALL must pass. Fix loop max 3.

## Quality bar
- ZERO touch to web/admin/mobile
- ZERO migration writes
- ZERO new go.mod dependencies (the generic helper uses existing `golang.org/x/sync/singleflight`)
- ONE squashed commit
- DO NOT modify `git config` — use per-command `-c user.email=...`

## Push + PR
- Message: `fix(backend): close V8 NEW-1 to NEW-6 + ci.yml port — backend completion`
- PR title: `[fix/h1-backend-completion] V8 backend NEW findings + audit docs refresh`

## Final report (under 700 words)
Standard structure + EMPHASIZE which findings CLOSED vs PARTIAL with rationale. Include audit docs delta (which sections were updated).
