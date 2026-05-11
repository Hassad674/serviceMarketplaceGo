# SEC-AUDIT-CACHE — Adversarial audit of QW-HARDENING (placeholder)

**Status**: BLOCKED waiting for QW-HARDENING merge to main
**Branch**: `chore/sec-audit-cache`
**Date started**: 2026-05-11
**Agent**: SEC-AUDIT-CACHE (worktree `agent-ab72a671b9e526c92`)

---

## Why this is paused

The QW-HARDENING agent (`a351aa60c6358f644`, branch `feat/qw-hardening`) currently has only the placeholder commit `20747fc3 wip(cache-hardening): in flight` on its branch. No real code change is on `feat/qw-hardening` or merged to `origin/main`.

The user's brief explicitly says: *"Wait for `git fetch origin && git log --oneline main` to show QW-HARDENING's commits BEFORE you start adversarial testing. If not yet merged, write `sec-audit-cache.md` with current findings + plan, push, and STOP."*

This file is the deliverable for the "STOP" branch.

---

## Audit scope (post-merge)

QW-HARDENING is expected to deliver:

1. **Cache invalidation wiring** — `cache.Invalidate(userID)` (and the org-overrides equivalent) at every `BumpSessionVersion` / `role_overrides` mutation call site.
2. **Singleflight protection** against cache stampede on:
   - `backend/internal/adapter/redis/session_version_cache.go`
   - `backend/internal/adapter/redis/org_overrides_cache.go`

This audit must prove each invariant or document the residual risk.

---

## Pre-merge baseline (current main, commit `d4e35d50`)

### `session_version_cache.go` (already on main)

Already implements:
- Cache-aside read (Redis GET → inner DB fallback) with 30s TTL.
- Write-through on miss.
- `Invalidate(ctx, userID)` method (Redis DEL, idempotent on missing key).
- Error semantics: `ErrUserNotFound` and transient inner errors are **NOT** cached (good).
- Malformed payload triggers refresh (existing test pins it).

Missing (QW-HARDENING is expected to add):
- Singleflight around the inner-call path so 100 concurrent misses for `uid_X` collapse to one DB read.
- Active wiring of `Invalidate` at all 13 production `BumpSessionVersion` call sites listed below.

### `org_overrides_cache.go` (already on main)

Same shape as session_version_cache. Same gaps.

### BumpSessionVersion call sites (13 production)

```
internal/app/organization/transfer_service.go:235     — old owner on org transfer
internal/app/organization/transfer_service.go:239     — new owner on org transfer
internal/app/organization/role_overrides_service.go:275 — on role permissions matrix update (per affected user)
internal/app/auth/service_more.go:76                  — on logout-all
internal/app/auth/service_more.go:408                 — on password change
internal/app/auth/service_account.go:196              — on account deletion
internal/app/organization/admin_overrides.go:67       — admin force-bump
internal/app/organization/admin_overrides.go:102      — admin force-bump (variant)
internal/app/organization/admin_overrides.go:202      — admin org transfer (old owner)
internal/app/organization/admin_overrides.go:206      — admin org transfer (new owner)
internal/app/organization/membership_service.go:165   — member kick/leave
internal/app/organization/membership_service.go:281   — role change
internal/app/organization/membership_service.go:332   — leave-self
internal/app/admin/service.go:355                     — admin ban
```

QW-HARDENING must touch each site (or wire a wrapper) so that every bump is paired with `sessionVersionCache.Invalidate(ctx, userID)`. Audit must verify this by grep — any bump without a paired Invalidate is a RED flag.

### `role_overrides` mutation sites

```
internal/adapter/postgres/organization_repository.go:250 — UPDATE organizations SET role_overrides = $2
internal/app/organization/role_overrides_service.go:UpdateRoleOverrides — single write path
internal/handler/role_overrides_handler.go:UpdateMatrix — handler entry
```

Single write path → single Invalidate call needed. Audit must verify the call is present on the handler/service path.

---

## Attack vectors — test plan

For each, the test will live in `backend/internal/adapter/redis/*_security_test.go`. All tests will run under `go test ... -race`.

### A. Read/write race between bump+invalidate and concurrent read

**Setup**: G1 calls `BumpSessionVersion(uid)` followed by `cache.Invalidate(uid)`. G2 calls `GetSessionVersion(uid)` racing in parallel.

**Invariant**: AFTER G1's `Invalidate` returns, no subsequent G2 read may return the pre-bump version. The window in which G2 could observe the stale value is bounded by [G1.Bump-start, G1.Invalidate-return].

**Implementation**: 1000 iterations. After each iteration, await G1 fully (Bump+Invalidate), then issue a fresh `GetSessionVersion` and assert it equals the bumped value. The intermediate reads during the window are inherently allowed to be stale, but **after-the-fact** reads must reflect the new version.

**Verdict criteria**: GREEN if 0 stale reads observed post-Invalidate. RED otherwise.

### B. Invalidate fails but Bump succeeds

**Setup**: Inject a Redis client whose `DEL` returns a network error. Call the production code path that bumps + invalidates.

**Invariant**: The bump must NOT be rolled back (DB is authoritative). The failure must be logged at ERROR or WARN level. Stale cache entry persists for up to TTL (30s).

**Required behavior** (per brief): *At minimum log at ERROR level. Ideally fall back to a "force-write the new version" Redis SET.*

**Implementation**: Wrap miniredis client with a flaky-DEL decorator. Trigger the path. Capture slog output. Assert:
- The bump persisted in the test DB.
- A log line at WARN/ERROR was emitted.
- (Soft) the cache key is either deleted OR force-rewritten with the new version.

**Verdict criteria**:
- GREEN if Bump persists AND failure logged AND force-write fallback is implemented.
- YELLOW if Bump persists AND failure logged but no force-write — residual 30s stale window. Acceptable per current doctrine.
- RED if Bump persists silently with no log, OR Bump is rolled back.

### C. Cache poisoning after Redis recovery

**Setup**: Redis client returning network errors → cache reads fall through to DB → user sees correct state. Then Redis "recovers" (errors stop). Confirm no stale value can be injected.

**Invariant**: The cache content can only come from the inner DB read, never from user-controlled state.

**Implementation**: Toggle a flaky-Redis decorator from "always-fail" to "always-ok". Confirm next read is a miss (no stale data) and that the value written matches the DB.

**Verdict criteria**: GREEN if next read after recovery sources from DB; no path allows user-supplied data into the cache.

### D. Singleflight identity confusion

**Setup**: 100 concurrent `GetSessionVersion(uid_A)` + 100 concurrent `GetSessionVersion(uid_B)`. Inner reader has a 50ms artificial delay and counts calls per uid.

**Invariant**: Singleflight coalesces per-key (exactly 1 inner call per uid). Each caller receives the correct value for the uid they asked for. NO cross-key value leak.

**Implementation**:
- Use `sync.WaitGroup` to start all 200 goroutines simultaneously.
- Inner reader returns deterministic value (`versionForUID(uid)`).
- After: assert `callsFor[uid_A] == 1` and `callsFor[uid_B] == 1`.
- Assert every caller received the correct value (compare by uid).

**Verdict criteria**: GREEN if 1 inner call per key AND every caller got correct value. RED if cross-key value leak OR > 1 inner call per key.

NOTE: If QW-HARDENING did NOT add singleflight (e.g., relied solely on Invalidate), then expect `callsFor[uid_A] == 100` — that's a DIFFERENT verdict (no perf bug, but no stampede protection either). Flag YELLOW.

### E. Negative cache poisoning

**Setup**: Inner returns `user.ErrUserNotFound`. Call `GetSessionVersion(uid)`.

**Invariant**: The cache MUST NOT store a key for a not-found user. Next call MUST hit the inner again.

**Implementation**: Already covered by `TestSessionVersionCache_DoesNotCacheUserNotFound` — re-verify under `-race` and add a singleflight-aware variant: 100 concurrent calls for a non-existent uid; assert `callsFor[uid] <= 100` (no upper bound forced) and Redis key absent.

**Verdict criteria**: GREEN if no Redis key written and subsequent calls re-hit inner. RED if any Redis key exists.

### F. TTL bypass

**Setup**: Plant a cache entry with no TTL via direct miniredis manipulation.

**Invariant**: TTL is set by the cache code, not by user input — so no external influence is possible. The test confirms the cache always writes with the configured TTL.

**Implementation**: Prime the cache, read TTL via `mr.TTL(key)`, assert ~30s. Then directly write a no-TTL key, call `GetSessionVersion`, assert the cache replaces it on next write-through with TTL set.

**Verdict criteria**: GREEN if every cache write goes through `Set(ctx, k, v, ttl)`. (Note: only relevant write path is in the cache code; verify by grep — no other site SETs `session_version:*`.)

### G. Concurrent invalidation idempotency

**Setup**: Spawn 100 goroutines each calling `cache.Invalidate(uid)` for the same uid. Then 1 final goroutine reads.

**Invariant**: Redis DEL on missing key is a noop (returns 0). 100 concurrent DELs must not produce error and final state must be cache-empty.

**Implementation**: Already trivially safe. Test confirms no error from any of the 100 Invalidate calls; final Redis key absent.

**Verdict criteria**: GREEN.

### H. Org-overrides — repeat A-G

Identical attack matrix on `CachedOrgOverridesResolver`. The only structural difference is:
- Payload is JSON-marshalled `RoleOverrides` (map of permissions) — adds a marshal/unmarshal step.
- Inner error semantics are "fail-open" (middleware trusts session snapshot on resolver error) so negative-cache risk is the same.

Same test plan, swap types accordingly.

---

## Per-vector verdict (placeholder)

| Vector | Description | Status |
|--------|-------------|--------|
| A | Race between bump and read | PENDING |
| B | Invalidate fails but bump succeeds | PENDING |
| C | Cache poisoning after Redis recovery | PENDING |
| D | Singleflight identity confusion | PENDING |
| E | Negative cache poisoning | PENDING |
| F | TTL bypass attack | PENDING |
| G | Concurrent invalidation idempotency | PENDING |
| H | Org-overrides (A-G repeated) | PENDING |

---

## Resumption checklist for follow-up agent

When QW-HARDENING merges, the resuming agent must:

1. `git fetch origin && git log --oneline origin/main -20` — confirm a `Merge feat: QW-HARDENING ...` commit exists.
2. `git switch chore/sec-audit-cache && git rebase origin/main` — bring this audit up to the post-merge state.
3. Re-read `backend/internal/adapter/redis/session_version_cache.go` and `org_overrides_cache.go` — diff against the baseline captured above.
4. Re-grep every `BumpSessionVersion` call site in the list above and confirm each is paired with an Invalidate.
5. Author the two test files per the plan in section "Attack vectors — test plan":
   - `backend/internal/adapter/redis/session_version_cache_security_test.go`
   - `backend/internal/adapter/redis/org_overrides_cache_security_test.go`
6. Run `timeout 90 go build ./...` and `timeout 180 go test ./internal/adapter/redis/... -count=1 -race -run "Security"`.
7. Fill the per-vector verdict table and produce the final report.

All non-trivial code analysis already done is captured above so the resuming agent does NOT need to re-discover the call-site list or baseline file contents.
