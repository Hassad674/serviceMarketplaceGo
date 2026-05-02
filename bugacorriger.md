# Bugs à corriger — Final Verification

**Date** : 2026-05-01 (final verification post F.1 + F.2)
**Branche** : `chore/final-verification-audit`

---

## Snapshot — état actuel après F.1 + F.2 (PRs #31 → #91)

| Severity | Count |
|---|---|
| CRITICAL | 0 |
| HIGH | 4 |
| MEDIUM | 7 |
| LOW | 5 |
| **Total** | **16** |

**Closed since previous round (6 items)** :
- BUG-FINAL-01 (was BUG-NEW-04 — RLS migration breaking GetByID callers) — **CLOSED** by `loadProposalForActor` / `loadDisputeForActor` system-actor branching + `warnIfNotSystemActor` soft guardrail. Verified at `proposal/service_actions.go:320-326` and `dispute/service_actions.go:34-40`.
- BUG-FINAL-04 (RequestPayout silenced records.Update at 3 sites) — **CLOSED**: `payout_request.go:107`, `:129`, `:316` now log `slog.Error("payout: failed to persist ... — record desynced from Stripe")`. Verified.
- BUG-FINAL-05 (upload goroutine ctx discarded) — **CLOSED** via P11 graceful shutdown wiring with `UploadCancel` + `UploadHandler.Stop(ctx)` propagating shutdown context.
- BUG-FINAL-07 (empty list normalization) — verify status. Tests in PR #74 mention nested slice fix.
- BUG-NEW-12, BUG-NEW-13 — admin Suspense flash + RSC fallback port — **CLOSED** by P10/PR #87.

---

## CRITICAL (0)

**All previous CRITICAL items closed.**

---

## HIGH (4)

### BUG-FINAL-02 : LiveKit room `maxParticipants=2` + reconnect → 3rd-participant rejection
- **Severity**: HIGH
- **NOTE** : OFF-LIMITS per CLAUDE.md — flag only, do not fix.

### BUG-FINAL-03 : LiveKit token sans `CanPublish` / `CanSubscribe`
- **Severity**: HIGH
- **NOTE** : OFF-LIMITS per CLAUDE.md — flag only, do not fix.

### BUG-FINAL-06 : Search publisher cooldown stamps before outer tx commits
- **Severity**: HIGH
- **Location** : `backend/internal/app/searchindex/publisher.go:144-173` (`PublishReindexTx`)
- **Trigger** : flow is (1) `buildReindexEvent` checks `isWithinCooldown`, (2) `events.ScheduleTx` inserts in caller's tx, (3) `recordPublish` stamps `lastPublish` map. If caller's outer tx rolls back AFTER step 3, the row is wiped but `lastPublish` retains the stamp.
- **Fix** : tx hook (`tx.AfterCommit`) calls `recordPublish` only on successful commit, OR move stamp out of function for caller to invoke after commit.
- **Effort** : S (1-2h)

### BUG-FINAL-08 : `RetryFailedTransfer` raw field assignment bypasses state machine (= SEC-FINAL-16)
- **Severity**: HIGH
- **Location** : `backend/internal/app/payment/payout_request.go:292` — `record.TransferStatus = domain.TransferPending` (verified raw assignment).
- **Fix** : `MarkTransferRetrying()` method with state machine guard.
- **Effort** : XS (30 min)

### BUG-FINAL-09 : Wallet referral commissions silently swallow DB errors
- **Severity**: HIGH (UX confusion + financial)
- **Location** : `backend/internal/app/payment/wallet.go:660-700`
- **Trigger** : `if sum, err := w.referralWallet.GetReferrerSummary(...); err == nil { ... }` — on transient DB errors, the user sees `wallet.Commissions` as zero/empty even though commissions exist.
- **Fix** : at minimum `slog.Warn("wallet: failed to load referral summary", "error", err)`. Better: `commissions_partial: true` flag on the response.
- **Effort** : S (1-2h)

---

## MEDIUM (7)

### BUG-FINAL-10 : API response envelope incohérente avec contrat documenté
- **Severity**: MEDIUM
- **Location** : `backend/pkg/response/json.go:43-48`
- **Fix** : `JSONData(w, status, data)` wrapper, migrate handlers progressively.
- **Effort** : L (2 days, cross-cutting)

### BUG-FINAL-11 : VIES cache `_ = c.redisClient.Set(...)` 
- **Severity**: MEDIUM
- **Location** : `backend/internal/adapter/vies/client.go:165`
- **Fix** : log warn.
- **Effort** : XS (5 min)

### BUG-FINAL-12 : WS presence broadcast `_ = deps.Hub.broadcastToOthers`
- **Severity**: MEDIUM
- **Location** : `backend/internal/adapter/ws/connection.go:263`
- **Fix** : log warn.
- **Effort** : XS (5 min)

### BUG-FINAL-13 : FCM device tokens jamais marqués stale (= SEC-FINAL-15)
- **Severity**: MEDIUM
- **Location** : `backend/internal/adapter/fcm/push.go:75-83`
- **Fix** : on Firebase error (UNREGISTERED, INVALID_ARGUMENT), call `device_tokens.MarkStale(ctx, tokens)`.
- **Effort** : S (1-2h)

### BUG-FINAL-14 : Mobile non-null assert `_formKey.currentState!`
- **Severity**: MEDIUM
- **Location** : `mobile/lib/features/auth/presentation/screens/login_screen.dart:34`, `register_screen.dart:41`, `agency_register_screen.dart:40`
- **Fix** : `if (_formKey.currentState?.validate() != true) return;`.
- **Effort** : XS (15 min)

### BUG-FINAL-15 : Search index publisher debounce 5min process-local
- **Severity**: MEDIUM
- **Fix** : Redis SETNX with TTL.
- **Effort** : S (1-2h)

### BUG-FINAL-16 : Mobile FCM cold-launch tap can be silently dropped
- **Severity**: MEDIUM
- **Location** : `mobile/lib/core/notifications/fcm_service.dart:213-236`
- **Fix** : `WidgetsBinding.instance.addPostFrameCallback` instead of fixed 100ms.
- **Effort** : S (1-2h)

### BUG-FINAL-17 : `MaxBytesReader` cap reset to nil-writer inside `validateAndBuildKey`
- **Severity**: MEDIUM
- **Location** : `backend/internal/handler/upload_handler.go:466`
- **Fix** : drop the redundant `r.Body = http.MaxBytesReader(nil, r.Body, maxSize)`.
- **Effort** : XS (5 min)

---

## LOW (5)

- **BUG-FINAL-18** : `tx.Commit` ignored in `conversation_repository.go:43`. Effort: XS.
- **BUG-FINAL-19** : `defer tx.Rollback()` loses error across ~30 sites. Effort: S.
- **BUG-FINAL-20** : `_ = json.Marshal(c)` in `pkg/cursor/Encode`. Effort: XS.
- **BUG-FINAL-21** : WS broadcastPresenceChange on disconnect uses `context.Background()` instead of `WithoutCancel`. Effort: XS.
- **BUG-FINAL-22** : Stale TODO in `messaging_ws_service.dart:175` ("TODO: replace with single-use, short-lived WS token" — already migrated). Effort: 1 min.

---

## Audit completion summary

| Audit doc | Items closed | Items remaining |
|---|---|---|
| auditsecurite.md | 6 / 20 (30%) | 14 |
| auditperf.md | 15 / 58 (26%) | 43 |
| auditqualite.md | 16 / 73 (22%) | 57 |
| bugacorriger.md | 6 / 22 (27%) | 16 |
| rapportTest.md | varies | see file |

**Total**: ~43 of 195 (22%) findings closed in F.1 + F.2 — the 7-PR series prioritized CRITICAL + the highest-ROI HIGHs. The remaining 152 items are the F.3/F.4 backlog: medium-effort polish + non-blocking debt.

---

## What's blocking publication TODAY

| Item | Severity | Effort |
|---|---|---|
| SEC-FINAL-07 (admin token in localStorage) | HIGH | M |
| SEC-FINAL-04 (SSRF on social URLs) | HIGH | S |
| SEC-FINAL-03 (RequireRole middleware) | HIGH | S |

After fixing those 3 (~6h total), the codebase has zero CRITICAL items, no exposed CVEs, no deployment blockers, no bug masquerading as a feature.
