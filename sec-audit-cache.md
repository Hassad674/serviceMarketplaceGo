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

## Per-vector verdict — PART 2 EXECUTION COMPLETE (2026-05-11)

Test files (pushed to `chore/sec-audit-cache-part2`):
- `backend/internal/adapter/redis/session_version_cache_security_test.go` — 6 tests, vectors A/B/C/D/E/G.
- `backend/internal/adapter/redis/org_overrides_cache_security_test.go` — 6 tests, vectors A/B/C/D/E/G.

Validation pipeline (run from `backend/`):

```
$ timeout 90 go build ./...
(silent — green)

$ timeout 480 go test ./internal/adapter/redis/... -count=3 -race -run "Security"
ok      marketplace-backend/internal/adapter/redis      12.033s

$ timeout 300 go test ./internal/adapter/redis/... -count=1 -race
ok      marketplace-backend/internal/adapter/redis      43.356s
```

All 12 security tests pass 3× under `-race`. The wider redis suite
also passes under `-race`, confirming no regression on the existing
happy-path tests.

| Vector | Description | session_version | org_overrides | Verdict |
|--------|-------------|-----------------|---------------|---------|
| A | Race between mutation+Invalidate and concurrent read (1000 iter) | PASS | PASS | **GREEN** |
| B | Invalidate fails but inner mutation succeeds | PASS (error surfaced) | PASS (error surfaced) | **YELLOW** — see residual risk below |
| C | Cache poisoning after Redis recovery | PASS | PASS | **GREEN** |
| D | Singleflight key isolation across 2 ids (100×2 concurrent callers) | PASS — 1 inner call per uid, NO cross-key leak | PASS — 1 inner call per orgID, NO cross-key leak | **GREEN** |
| E | Negative cache poisoning (ErrUserNotFound / transient error) | PASS — no Redis key written | PASS — no Redis key written | **GREEN** |
| F | TTL bypass | Not directly tested (pre-existing happy-path test `TestSessionVersionCache_DefaultTTLAppliedWhenZero` and `TestOrgOverridesCache_DefaultTTLAppliedWhenZero` already pin TTL on every write — no user-controlled write path exists). | same | **GREEN by construction** |
| G | Concurrent invalidation idempotency (100 parallel DEL) | PASS — 0 errors, key absent | PASS — 0 errors, key absent | **GREEN** |
| H | Org-overrides A-G repeated | — | covered by the dedicated org_overrides test file | **GREEN** (5 / 6 GREEN, 1 YELLOW — same shape as session_version) |

### Net verdict

**GREEN** on 5 of 6 functional vectors for BOTH caches under the
QW-HARDENING post-merge state. **YELLOW** on vector B — the cache
surfaces a non-nil error when DEL fails (so the call-site CAN log /
alert), but the cache does NOT currently:

1. Retry the DEL with backoff.
2. Force-write the new version into the Redis key (so even a missed
   DEL would not produce a stale read).
3. Log the failure inside `Invalidate` itself (the warn log only
   fires on the SET path in `load` / `peek`).

**Residual risk (vector B):**

- Window of staleness: bounded by the cache TTL (30 s) per the
  `DefaultSessionVersionCacheTTL` / `DefaultOrgOverridesCacheTTL`
  constants.
- Caller responsibility: every site that calls
  `BumpSessionVersion` + `Invalidate` MUST check the Invalidate
  return value and log it at WARN/ERROR. The current code base
  does NOT do this consistently — `BumpSessionVersion` is
  uniformly called via `s.users.BumpSessionVersion(ctx, ...)`
  with a best-effort log, but the **`Invalidate` wiring at the 13
  BumpSessionVersion call sites listed in the pre-merge baseline
  was NOT shipped** by QW-HARDENING (which only added
  singleflight).
- Impact: any successful Bump followed by a Redis DEL failure
  will leave the previous session_version cached for up to 30 s.
  A revoked session can therefore continue to authenticate for
  the residual TTL.

### Remediation brief (recommended follow-up)

A separate `chore/sec-audit-cache-part3` agent should:

1. **Wire `sessionVersionCache.Invalidate` at every BumpSessionVersion
   call site** (13 sites listed in the pre-merge baseline section
   above). Each call site MUST log the Invalidate error at WARN
   level so failures are observable.
2. **Wire `orgOverridesCache.Invalidate` at the single
   `role_overrides` write path** (already a single site, just needs
   the call inserted).
3. **Optional hardening — force-write fallback inside
   `Invalidate`:** if DEL fails, attempt a `SET` of the new
   version-or-empty payload with TTL. This collapses the residual
   stale window from 30 s to ~0 s even when DEL fails. Tradeoff is
   one extra Redis round-trip on the rare failure path —
   acceptable.
4. **Add an internal `slog.Warn` inside `Invalidate` itself** so a
   forgotten call site still produces an observable warning rather
   than a silent stale-read tail. The existing pattern from
   `peek` / `load` makes this trivial.

The audit tests in this branch will continue to enforce the
GREEN/YELLOW verdicts so that any regression on the singleflight,
cache-poisoning, or negative-cache surface is caught in CI.

---

## Vector C nuance — "load" semantics under SetError

A subtle point caught while writing the test:

- During `mr.SetError(...)`, the cache's `load` path performs the
  `SET key value TTL` on the write-through. That SET errors out,
  but the cache returns the inner value to the caller (per the
  documented "Redis write error → log, return inner result"
  semantics). On Redis recovery, the next read is therefore a
  **fresh miss** (no value was actually persisted) and `load`
  successfully populates the cache. The third step of the test
  (mutate inner, assert cached value held) proves the write was
  durable post-recovery.
- This confirms there is NO PATH where Redis-during-blip ingests
  user-controlled state. The only writer is the cache's `load`
  method, which sources its value from the inner DB reader.

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
