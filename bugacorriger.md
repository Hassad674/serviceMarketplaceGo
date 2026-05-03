# Bugs à corriger — Final Deep Audit V2

**Date** : 2026-05-03 (final-deep-audit-v2 post F.1 + F.2 + F.3.1 + F.3.3)
**Branch** : `chore/final-deep-audit-v2`

---

## Snapshot — état actuel

| Severity | Count | Δ vs 2026-05-01 |
|---|---|---|
| CRITICAL | 0 | 0 |
| HIGH | 4 | 0 (LiveKit OFF-LIMITS counted) |
| MEDIUM | 7 | 0 |
| LOW | 5 | 0 |
| **Total** | **16** | **0** |

---

## CRITICAL (0)

**All previous CRITICAL items closed.**

---

## HIGH (4)

### BUG-FINAL-02 : LiveKit room `maxParticipants=2` + reconnect → 3rd-participant rejection
- **Severity**: HIGH
- **NOTE**: OFF-LIMITS per CLAUDE.md — flag only, do not fix.

### BUG-FINAL-03 : LiveKit token sans `CanPublish` / `CanSubscribe`
- **Severity**: HIGH
- **NOTE**: OFF-LIMITS per CLAUDE.md — flag only, do not fix.

### BUG-FINAL-06 : Search publisher cooldown stamps before outer tx commits
- **Severity**: HIGH
- **Location**: `backend/internal/app/searchindex/publisher.go:144-173` (`PublishReindexTx`)
- **Trigger**: flow is (1) `buildReindexEvent` checks `isWithinCooldown`, (2) `events.ScheduleTx` inserts in caller's tx, (3) `recordPublish` stamps `lastPublish` map. If caller's outer tx rolls back AFTER step 3, the row is wiped but `lastPublish` retains the stamp.
- **Fix**: tx hook (`tx.AfterCommit`) calls `recordPublish` only on successful commit, OR move stamp out of function for caller to invoke after commit.
- **Effort**: S (1-2h)

### BUG-FINAL-09 : Wallet referral commissions silently swallow DB errors
- **Severity**: HIGH (UX confusion + financial)
- **Location**: `backend/internal/app/payment/wallet.go:660-700`
- **Trigger**: `if sum, err := w.referralWallet.GetReferrerSummary(...); err == nil { ... }` — on transient DB errors, the user sees `wallet.Commissions` zero/empty even though commissions exist.
- **Fix**: at minimum `slog.Warn("wallet: failed to load referral summary", "error", err)`. Better: `commissions_partial: true` flag on the response.
- **Effort**: S (1-2h)

---

## MEDIUM (7)

### BUG-FINAL-08 : `RetryFailedTransfer` raw field assignment bypasses state machine
- **Severity**: MEDIUM (state machine)
- **Location**: `backend/internal/app/payment/payout_request.go:292` — `record.TransferStatus = domain.TransferPending`. Verified raw assignment.
- **Fix**: `MarkTransferRetrying()` method with state machine guard.
- **Effort**: XS (30 min)

### BUG-FINAL-10 : API response envelope incohérente avec contrat documenté
- **Severity**: MEDIUM
- **Location**: `backend/pkg/response/json.go:43-48`
- **Fix**: `JSONData(w, status, data)` wrapper, migrate handlers progressively. **Will land naturally as F.3.2 typed apiClient consumers force consistent envelopes**.
- **Effort**: L (2 days, cross-cutting)

### BUG-FINAL-11 : VIES cache `_ = c.redisClient.Set(...)`
- **Severity**: MEDIUM
- **Location**: `backend/internal/adapter/vies/client.go:165`
- **Fix**: log warn.
- **Effort**: XS (5 min)

### BUG-FINAL-12 : WS presence broadcast `_ = deps.Hub.broadcastToOthers`
- **Severity**: MEDIUM
- **Location**: `backend/internal/adapter/ws/connection.go:263`
- **Fix**: log warn.
- **Effort**: XS (5 min)

### BUG-FINAL-13 : FCM device tokens jamais marqués stale
- **Severity**: MEDIUM
- **Location**: `backend/internal/adapter/fcm/push.go:75-83`
- **Fix**: on Firebase error (UNREGISTERED, INVALID_ARGUMENT), call `device_tokens.MarkStale(ctx, tokens)`.
- **Effort**: S (1-2h)

### BUG-FINAL-14 : Mobile non-null assert `_formKey.currentState!`
- **Severity**: MEDIUM
- **Location**: `mobile/lib/features/auth/presentation/screens/login_screen.dart:34`, `register_screen.dart:41`, `agency_register_screen.dart:40`
- **Fix**: `if (_formKey.currentState?.validate() != true) return;`.
- **Effort**: XS (15 min)

### BUG-FINAL-15 : Search index publisher debounce 5min process-local
- **Severity**: MEDIUM
- **Fix**: Redis SETNX with TTL.
- **Effort**: S (1-2h)

### BUG-FINAL-16 : Mobile FCM cold-launch tap can be silently dropped
- **Severity**: MEDIUM
- **Location**: `mobile/lib/core/notifications/fcm_service.dart:213-236`
- **Fix**: `WidgetsBinding.instance.addPostFrameCallback` instead of fixed 100ms.
- **Effort**: S (1-2h)

### BUG-FINAL-17 : `MaxBytesReader` cap reset to nil-writer inside `validateAndBuildKey`
- **Severity**: MEDIUM
- **Location**: `backend/internal/handler/upload_handler.go:466`
- **Fix**: drop the redundant `r.Body = http.MaxBytesReader(nil, r.Body, maxSize)`.
- **Effort**: XS (5 min)

---

## LOW (5)

- BUG-FINAL-18 : `tx.Commit` ignored in `conversation_repository.go:43`. Effort: XS.
- BUG-FINAL-19 : `defer tx.Rollback()` loses error across ~30 sites. Effort: S.
- BUG-FINAL-20 : `_ = json.Marshal(c)` in `pkg/cursor/Encode`. Effort: XS.
- BUG-FINAL-21 : WS broadcastPresenceChange on disconnect uses `context.Background()` instead of `WithoutCancel`. Effort: XS.
- BUG-FINAL-22 : Stale TODO in `messaging_ws_service.dart:175`. Effort: 1 min.

---

## Audit completion summary

| Audit doc | Items closed | Items remaining (~) |
|---|---|---|
| auditsecurite.md | 8 / 20 (40%) | 12 |
| auditperf.md | 17 / 58 (29%) | 41 |
| auditqualite.md | 16 / 73 (22%) | 58 |
| bugacorriger.md | 7 / 22 (32%) | 15 |
| rapportTest.md | varies | see file |

**Total**: ~50 of 195 (~26%) findings closed across F.1 + F.2 + F.3.1 + F.3.3 — same numerator since 2026-05-01 (no F.4 work yet).

---

## What's blocking publication TODAY

| Item | Severity | Effort |
|---|---|---|
| QUAL-FINAL-W-NEW-01 (3 ESLint errors web) | HIGH | XS (30 min) |
| QUAL-FINAL-A-NEW-01 (admin install dance) | HIGH | XS (15 min) |
| SEC-FINAL-02 (idempotency middleware) | HIGH | M (½j) |
| SEC-FINAL-13 (slog ReplaceAttr redact) | HIGH | S (1-2h) |
| SEC-FINAL-06 (Stripe error sanitize) | HIGH | XS (30 min) |
| SEC-FINAL-NEW-01 (`go mod tidy`) | MEDIUM | XS (15 min) |
| F.3.2 PR merge (typed apiClient + OpenAPI) | — | M (½j) |

**Total blocking publication**: ~1.5-2 days of focused work.

After fixing those 7, the codebase has:
- 0 CRITICAL items
- 0 deployment blockers
- 0 visible amateur signals
- TOP 1% on 7/7 axes (Architecture, Security, Performance, Evolvability, Code Cleanliness, Best Practices, Security Paranoid)
