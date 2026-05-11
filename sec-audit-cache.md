# SEC-AUDIT-CACHE — Adversarial audit of QW-HARDENING

**Status**: in flight — placeholder (QW-HARDENING not yet merged to main)
**Branch**: `chore/sec-audit-cache`
**Date started**: 2026-05-11

## Context

QW-HARDENING (branch `feat/qw-hardening`, agent `a351aa60c6358f644`) is wiring `cache.Invalidate(userID)` on every `BumpSessionVersion` call site, plus singleflight protection against cache stampede on:
- `backend/internal/adapter/redis/session_version_cache.go`
- `backend/internal/adapter/redis/org_overrides_cache.go`

This audit's job: **prove or disprove each security invariant on the post-merge code**.

## Per-vector status (placeholder — will fill once merged)

| Vector | Description | Status |
|--------|-------------|--------|
| A | Race between bump and read | PENDING |
| B | Invalidate fails but bump succeeds | PENDING |
| C | Cache poisoning after Redis recovery | PENDING |
| D | Singleflight identity confusion | PENDING |
| E | Negative cache poisoning | PENDING |
| F | TTL bypass attack | PENDING |
| G | Concurrent invalidation idempotency | PENDING |
| H | Org-overrides — repeat A-G | PENDING |

## Plan

1. Wait for `feat/qw-hardening` to merge to `main`. Probe every ~5min.
2. Once merged, rebase this branch on `main` and read the patched source files.
3. Write adversarial tests in:
   - `backend/internal/adapter/redis/session_version_cache_security_test.go`
   - `backend/internal/adapter/redis/org_overrides_cache_security_test.go`
4. Run `timeout 120 go test ./internal/adapter/redis/... -count=1 -race -run "Security"`.
5. Report per-vector verdict + flag any RED.
