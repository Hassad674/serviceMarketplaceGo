# Bugs à corriger — Final Deep

**Date** : 2026-05-01 (final audit before public showcase)
**Branche** : `chore/final-audit-deep`
**Méthodologie** : audit exhaustif full-stack focalisé sur les bugs réels (races, state machines, error swallowing, null safety, off-by-one, edge cases, ressources). Pour chaque suspicion : localisation + recoupement avec les tests + lecture des call sites + cross-référence avec MEMORY.md pour ne pas re-flagger les items déférés (modération AWS region, FCM push, blocking, subscription user_id legacy).

Les bugs **strictement de sécurité** sont dans `auditsecurite.md` ; les bugs de **performance** dans `auditperf.md`. Ce fichier liste les bugs métier, race conditions, state machines, cohérence de données.

---

## CRITICAL (1)

### BUG-FINAL-01 (was BUG-NEW-04) : RLS migration breaks ALL legacy GetByID callers under prod role
- **Severity**: 🔴 CRITICAL — deployment time-bomb
- **Location** : `backend/migrations/125_enable_row_level_security.up.sql` + 35 callers in `internal/app/{proposal,dispute,review,referral}/*.go` (see SEC-FINAL-01 for full list).
- **Type** : deployment foot-gun / silent zero-row regression
- **Trigger** : migration 125 enables `FORCE ROW LEVEL SECURITY` on 9 tables. The 8-path PR series migrated each REPO method to wrap reads in `RunInTxWithTenant`, but kept the legacy `GetByID(ctx, id)` signature in place "for system-actor scheduler paths". Today this works because the migration owner role bypasses RLS. The moment production rotates to a dedicated `marketplace_app NOSUPERUSER NOBYPASSRLS` role, every legacy `GetByID` call returns `ErrNotFound` because the policy `USING` evaluates to NULL/false.
- **Impact** : at the moment the prod role rotation happens, ALL proposal/dispute/review actions silently fail with NotFound. Every checkout returns 404. Every dispute action 404s. Total app outage masquerading as a routing bug.
- **Reproduction** :
  ```sql
  CREATE ROLE test_app NOSUPERUSER NOBYPASSRLS LOGIN PASSWORD 'x';
  GRANT ALL ON ALL TABLES IN SCHEMA public TO test_app;
  -- Connect as test_app:
  SELECT count(*) FROM proposals WHERE id = '<existing_proposal_id>';
  -- returns 0 rows
  ```
  Then run any backend service action that calls `proposals.GetByID(ctx, id)` — it returns `ErrProposalNotFound` even though the proposal exists.
- **Fix sketch** : 
  1. Add `GetByIDForOrg(ctx, id, callerOrgID)` to dispute, review, milestone repos (proposal already has it).
  2. Migrate every legacy caller : extract `orgID := mustGetOrgID(ctx)` from middleware context, pass to `GetByIDForOrg`.
  3. Keep the old `GetByID` signature ONLY for explicit system-actor paths (`AutoApproveMilestone`, `AutoCloseProposal`) and gate those behind a privileged DB connection pool that bypasses RLS.
- **Test required** : `rls_caller_audit_test.go` (integration) — create `marketplace_test_app` role with `NOBYPASSRLS`, run every public service action, asserter all return correctly.
- **Effort** : L (3 jours)

---

## HIGH (8)

### BUG-FINAL-02 (was BUG-13) : LiveKit room `maxParticipants=2` + reconnect → 3rd-participant rejection
- **Severity**: 🟠 HIGH (LiveKit OFF-LIMITS per CLAUDE.md)
- **Location** : `backend/internal/adapter/livekit/client.go:33-42`
- **Type** : state / UX
- **Trigger** : reconnect WS d'un user en plein appel (Wi-Fi/4G switch). L'ancienne participant entry n'est pas immédiatement nettoyée par LiveKit. Le user revient comme 3rd, rejeté.
- **Impact** : appel coupé, pas de recovery automatique.
- **Fix** : passer à `maxParticipants=4` (marge), OU identifier les participants par `identity` stable + autoriser le re-join à kicker l'ancienne session.
- **NOTE** : OFF-LIMITS — flag only, do not fix.

### BUG-FINAL-03 (was BUG-14) : LiveKit token sans `CanPublish` / `CanSubscribe`
- **Severity**: 🟠 HIGH (LiveKit OFF-LIMITS per CLAUDE.md)
- **Location** : `backend/internal/adapter/livekit/client.go:44-60`
- **Type** : misconfig
- **Trigger** : `VideoGrant{Room, RoomJoin: true}` sans permissions explicites. Selon les versions du SDK les défauts diffèrent → publish refusé.
- **Impact** : appels muets / sans vidéo si LiveKit durcit ses défauts.
- **Fix** : expliciter `CanPublish: stripe.Bool(true)`, `CanSubscribe`, `CanPublishData`.
- **NOTE** : OFF-LIMITS — flag only, do not fix.

### BUG-FINAL-04 (was BUG-NEW-01 partial) : `RequestPayout` and `RetryFailedTransfer` records.Update silenced at sites 774/782/1008
- **Severity**: 🟠 HIGH
- **Location** : `backend/internal/app/payment/payout_request.go` and `backend/internal/app/payment/payout_transfer.go` — sites refactored from old `service_stripe.go`. The fix at sites 120 and 365 (BUG-09) was applied but the same pattern at lines 774 (`MarkTransferFailed` after retry), 782 (`MarkTransferred` after retry), and 1008 (`RequestPayout` post-success) remained `_ = p.records.Update(ctx, r)` per the original audit. Need to verify post-refactor whether the pattern was carried forward.
- **Type** : error swallowing
- **Trigger** : after a successful Stripe `CreateTransfer`, the record is mutated to `MarkTransferred(transferID)` then the persistence is silently swallowed — funds moved on Stripe but DB still says `TransferStatus = pending`. A retry will re-issue another transfer (idempotency-keyed, so Stripe blocks the duplicate, but the DB stays out of sync indefinitely).
- **Impact** : provider's wallet shows escrow funds that have already been transferred → over-counting. State drift between Stripe and DB.
- **Fix** : replicate the BUG-09 pattern at all 3 sites: `if uErr := s.records.Update(ctx, r); uErr != nil { slog.Error("payout: failed to persist MarkTransferred — record desynced from Stripe", ...); }`.
- **Test required** : extend `payout_bug_new_01_test.go` to cover the 3 sites, mock `Update` to return error, asserter slog.Error is captured.
- **Effort** : S (1-2h)

### BUG-FINAL-05 (was BUG-NEW-02) : Upload `RecordUpload` goroutine cancellation context discarded
- **Severity**: 🟠 HIGH
- **Location** : `backend/internal/handler/upload_handler.go:243-247`
- **Type** : dead code / cancellation loss
- **Trigger** : line 247 is `_ = ctx`. The carefully-built cancellation context is never actually passed to `h.recorder.RecordUpload(...)`. The recorder uses its own internal 60s timeout instead. SIGTERM still has to wait for each in-flight upload's full 60s timeout before draining.
- **Impact** : graceful shutdown blocked ~60s per in-flight upload. Rekognition/S3 calls won't be aborted on shutdown.
- **Fix sketch** : extend `MediaRecorder` interface with `RecordUploadCtx(ctx, ...)` that propagates `ctx`. Replace `_ = ctx` with the actual call.
- **Test required** : integration test that spawns 3 concurrent uploads, sends `cancel()` on `shutdownCtx`, asserter all 3 return within 5s (not 60s).
- **Effort** : S (1-2h)

### BUG-FINAL-06 (was BUG-NEW-05) : Search publisher cooldown stamps before outer tx commits
- **Severity**: 🟠 HIGH
- **Location** : `backend/internal/app/searchindex/publisher.go:144-173` (`PublishReindexTx`)
- **Type** : cooldown / state drift
- **Trigger** : flow is (1) `buildReindexEvent` checks `isWithinCooldown`, (2) `events.ScheduleTx` inserts in caller's tx, (3) `recordPublish` stamps `lastPublish` map. If caller's outer tx rolls back AFTER step 3, the row is wiped but `lastPublish` retains the stamp. The next mutation within 5 min won't re-publish — silent drift.
- **Impact** : profile UPDATE that fails at the last step (commit) yields a profile that "looks unchanged" + a cooldown that suppresses the next 5 minutes of attempts. User saves several times → looks like nothing happens.
- **Fix sketch** : register a tx hook (`tx.AfterCommit`) that calls `p.recordPublish(key)` only on successful commit, OR move the cooldown stamp out of the function and have the caller invoke it after a successful commit.
- **Test required** : test where `RunInTx` rolls back, the next tx (within cooldown) successfully schedules an event.
- **Effort** : S (1-2h)

### BUG-FINAL-07 (was BUG-NEW-11) : Empty list normalization only handles top-level slices
- **Severity**: 🟠 HIGH (TS client crash)
- **Location** : `backend/pkg/response/json.go:58-67` (`NilSliceToEmpty` with `reflect.Kind() != reflect.Slice` early return)
- **Type** : contract drift / TS client crash
- **Trigger** : BUG-19 fix wraps top-level encode arg via `NilSliceToEmpty(data)`. But struct DTOs containing slice fields (like `JobResponse.Skills`, `ProfileResponse.Languages`, `TeamMemberResponse.Permissions`) keep `null` when their source `j.Skills` is nil. Web/admin/mobile TS clients calling `.length` on these crash.
- **Impact** : client crashes when loading a Job that has no skills set (legitimate empty case).
- **Reproduction** : Create a Job with empty `skills` array. GET `/api/v1/jobs/{id}` → response contains `"skills": null`. Web client does `job.skills.length` → TypeError.
- **Fix sketch** : either (a) recursively normalise via reflection on Marshal time, (b) introduce per-DTO `nilToEmptyStrings(j.Skills)` conversion (helper exists in `helpers.go` for some DTOs but not job/proposal/etc.), or (c) annotate slice fields with a custom JSON marshaller.
- **Test required** : handler test that creates a Job with `Skills: nil`, GETs it, asserter response body contains `"skills": []` (not `null`).
- **Effort** : S (1-2h)

### BUG-FINAL-08 (NEW) : `BUG-NEW-18` `RetryFailedTransfer` raw field assignment bypasses state machine
- **Severity**: 🟠 HIGH (also flagged as SEC-FINAL-16 / QUAL-FINAL-B-08)
- **Location** : `backend/internal/app/payment/payout_transfer.go:992`
- **Type** : state machine bypass
- **Trigger** : `record.TransferStatus = domain.TransferPending` is raw — bypasses the `MarkTransferFailed` / `MarkTransferred` / `ApplyDisputeResolution` guarded mutators.
- **Fix** : `func (r *PaymentRecord) MarkTransferRetrying() error { if r.TransferStatus != TransferFailed { return ErrInvalidStateTransition }; r.TransferStatus = TransferPending; return nil }`.
- **Test required** : state machine test ensures `RetryFailedTransfer` rejette records pas en `TransferFailed`.
- **Effort** : XS (30 min)

### BUG-FINAL-09 (NEW) : `BUG-NEW-16` Wallet referral commissions silently swallow DB errors
- **Severity**: 🟠 HIGH (UX confusion + financial)
- **Location** : `backend/internal/app/payment/wallet.go:660-700`
- **Type** : error swallowing / UX confusion
- **Trigger** : `if sum, err := w.referralWallet.GetReferrerSummary(...); err == nil { ... }` and `if recent, err := w.referralWallet.RecentCommissions(...); err == nil { ... }` — on transient DB errors, the user sees `wallet.Commissions` as zero/empty even though commissions exist. Comment claims this is intentional but it produces false data without any signal.
- **Impact** : user thinks they have no commissions ; refresh fixes it but they may have already disputed a missing payout.
- **Fix sketch** : at minimum `slog.Warn("wallet: failed to load referral summary", "error", err)`. Better: surface a `commissions_partial: true` flag on the response so the UI can show "data temporarily unavailable".
- **Test required** : handler test mocks `referralWallet.GetReferrerSummary` to return an error, asserter the wallet response includes a `commissions_partial: true` flag.
- **Effort** : S (1-2h)

---

## MEDIUM (8)

### BUG-FINAL-10 (was BUG-18) : API response envelope incohérente avec contrat documenté
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/pkg/response/json.go:43-48`
- **Trigger** : `Error()` sort `{"error": "code", "message": "..."}` au lieu de `{"error": {"code", "message"}, "meta": {request_id}}` requis par CLAUDE.md. `JSON()` n'enveloppe pas dans `data:`.
- **Impact** : frontend doit jongler entre les deux formes ; OpenAPI inconsistante. Migration vers le contrat documenté nécessite un breaking change ou un dual-format pendant 6 mois.
- **Fix** : créer `JSONData(w, status, data)` qui wrap dans `{data: ...}`, migrer les handlers progressivement, déprécier l'ancienne forme. La forme erreur peut similarly migrer.
- **Effort** : L (2 jours, c'est cross-cutting)

### BUG-FINAL-11 (was BUG-21) : VIES cache `_ = c.redisClient.Set(...)` 
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/adapter/vies/client.go:165`
- **Trigger** : cache write VIES ignoré.
- **Impact** : cache miss à la prochaine vérif TVA — perf, pas correctness.
- **Fix** : log warn.
- **Effort** : XS (5 min)

### BUG-FINAL-12 (was BUG-23) : WS presence broadcast `_ = deps.Hub.broadcastToOthers`
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/adapter/ws/connection.go:263`
- **Trigger** : erreur de marshalling envelope ignorée.
- **Impact** : typing indicator perdu silencieusement.
- **Fix** : log.
- **Effort** : XS (5 min)

### BUG-FINAL-13 (was BUG-24) : FCM device tokens jamais marqués stale (now wired in adapter)
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/adapter/fcm/push.go:75-83`
- **Trigger** : sur erreur Firebase (UNREGISTERED, INVALID_ARGUMENT), warning logged but no `device_tokens` row deleted.
- **Impact** : notification fan-out gaspille des appels API à des tokens morts.
- **Fix** : sur erreur Firebase, supprimer le row. See SEC-FINAL-15.
- **Effort** : S (1-2h)

### BUG-FINAL-14 (was BUG-26) : Mobile non-null assert `_formKey.currentState!`
- **Severity**: 🟡 MEDIUM
- **Location** : `mobile/lib/features/auth/presentation/screens/login_screen.dart:34`, `register_screen.dart:41`, `agency_register_screen.dart:40`
- **Trigger** : si formulaire detached du tree au moment du tap (race), `currentState` est null → crash.
- **Impact** : edge crash rare.
- **Fix** : `if (_formKey.currentState?.validate() != true) return;`.
- **Effort** : XS (15 min)

### BUG-FINAL-15 (was BUG-27) : Search index publisher debounce 5min process-local
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/app/searchindex/publisher.go:128-130`
- **Trigger** : `lastPublish map[debounceKey]time.Time` local au process. N instances backend → debounce N× moins efficace.
- **Impact** : pression légèrement accrue sur Typesense / OpenAI embeddings. Pas critique mais coûteux à scale.
- **Fix** : déplacer dans Redis (`SETNX` avec TTL).
- **Effort** : S (1-2h)

### BUG-FINAL-16 (was BUG-NEW-15) : Mobile FCM cold-launch tap can be silently dropped
- **Severity**: 🟡 MEDIUM
- **Location** : `mobile/lib/core/notifications/fcm_service.dart:213-236`
- **Trigger** : cold-launch tap waits 100ms then drops if `rootNavigatorKey.currentContext` is still null. On slow Android devices, 100ms is too short.
- **Fix sketch** : `WidgetsBinding.instance.addPostFrameCallback` au lieu de fixed 100ms timer. Plus gate sur auth state — if unauthenticated, store the intent and push after login.
- **Test required** : integration test simulating cold-launch push tap on delayed-frame harness.
- **Effort** : S (1-2h)

### BUG-FINAL-17 (NEW) : `BUG-NEW-14` `MaxBytesReader` cap reset to nil-writer inside `validateAndBuildKey`
- **Severity**: 🟡 MEDIUM
- **Location** : `backend/internal/handler/upload_handler.go:466`
- **Trigger** : each upload handler does `r.Body = http.MaxBytesReader(w, r.Body, maxSize)` THEN `validateAndBuildKey` does AGAIN `r.Body = http.MaxBytesReader(nil, r.Body, maxSize)`. The second call replaces the first reader; the nil writer means no automatic 413 short-circuit.
- **Impact** : 413 response is still produced (manually via `isMaxBytesError`) but the redundant double-wrapping is bug-prone.
- **Fix sketch** : drop the inner `r.Body = http.MaxBytesReader(nil, r.Body, maxSize)` line.
- **Effort** : XS (5 min)

---

## LOW (5)

### BUG-FINAL-18 (was BUG-28) : `tx.Commit` ignoré dans `conversation_repository.go:43`
- **Severity**: 🟢 LOW
- **Location** : `backend/internal/adapter/postgres/conversation_repository.go:43`
- **How to fix** : `if err := tx.Commit(); err != nil { slog.Warn("...") }`.
- **Effort** : XS (5 min)

### BUG-FINAL-19 (was BUG-29) : `defer tx.Rollback()` partout perd l'erreur
- **Severity**: 🟢 LOW
- **Location** : ~30 sites
- **How to fix** : `defer func() { if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) { slog.Warn("rollback", "error", rbErr) } }()`.
- **Effort** : S (1-2h)

### BUG-FINAL-20 (was BUG-33) : `_ = json.Marshal(c)` dans `pkg/cursor/Encode`
- **Severity**: 🟢 LOW
- **Location** : `pkg/cursor/cursor.go`
- **How to fix** : retourner `(string, error)`.
- **Effort** : XS (15 min)

### BUG-FINAL-21 (was BUG-NEW-17) : WS broadcastPresenceChange on disconnect uses `context.Background()` instead of WithoutCancel
- **Severity**: 🟢 LOW
- **Location** : `backend/internal/adapter/ws/connection.go:153`
- **Trigger** : line 75 uses `bgCtx := context.WithoutCancel(r.Context())` for the OnConnect broadcast (correct pattern). Line 153 (OnDisconnect) uses `context.Background()` — loses any baggage/trace context.
- **Fix sketch** : store the original request context on the Client struct, use `WithoutCancel(client.requestCtx)` on disconnect.
- **Effort** : XS (10 min)

### BUG-FINAL-22 (NEW) : Stale TODO comment in `messaging_ws_service.dart:175`
- **Severity**: 🟢 LOW
- **Location** : `mobile/lib/features/messaging/data/messaging_ws_service.dart:175`
- **Trigger** : comment says "TODO: replace with single-use, short-lived WS token" — but `ws_token` was migrated in PR #31 (SEC-15). Stale comment.
- **Fix** : delete the TODO.
- **Effort** : XS (1 min)

---

## Verified shipped (no longer issues)

- ✅ Magic byte validation generalised across upload endpoints
- ✅ Pagination cursor-based sur SearchProfiles
- ✅ Race condition payment_records — `payment_records.milestone_id UNIQUE` (m.093)
- ✅ Optimistic concurrency milestones — version column + check WHERE
- ✅ WS conversation seq locking — `queryLockConversation` + `MAX(seq)+1` dans tx
- ✅ Conversation deduplication race — SERIALIZABLE + retry
- ✅ ConfirmPayment Stripe verify (BUG-01 closed PR #33)
- ✅ Payment state guards (BUG-02 closed PR #35)
- ✅ Dispute restore propagation (BUG-03 closed PR #35)
- ✅ Race Stripe Connect create (BUG-04 closed PR #33)
- ✅ Outbox tx (BUG-05 closed PR #36, residual cooldown drift = BUG-FINAL-06)
- ✅ WS sendBuffer block (BUG-06 closed PR #40)
- ✅ WS isLast race (BUG-07 closed PR #40)
- ✅ Mobile refresh single-flight (BUG-08 closed PR #31)
- ✅ records.Update sites 1 & 2 (BUG-09 closed PR #35, residual at sites 774/782/1008 = BUG-FINAL-04)
- ✅ Webhook stripe_webhook_events table (BUG-10 closed PR #36)
- ✅ Webhook handler error → 503 retry (BUG-NEW-06 closed)
- ✅ GetByIDForUpdate misnomer (BUG-11 closed PR #35)
- ✅ Embedded invalid_json (BUG-12 closed PR #40)
- ✅ Context.Background overrides (BUG-15 closed PR #34)
- ✅ Notification worker pool (BUG-16 closed PR #36)
- ✅ Upload goroutine ctx (BUG-17 closed PR #40, residual `_ = ctx` discard = BUG-FINAL-05)
- ✅ Empty list null vs [] (BUG-19 partially closed PR #40, top-level only — nested = BUG-FINAL-07)
- ✅ Audit_repository unmarshal (BUG-20 closed PR #40)
- ✅ Notification queue Ack (BUG-22 closed PR #40)
- ✅ Mobile FCM tap routing (BUG-25 closed PR #36, cold-launch race = BUG-FINAL-16)
- ✅ Webhook 7d TTL replay (BUG-31 closed PR #36)
- ✅ Pending events stuck (BUG-NEW-03 closed via m.128)
- ✅ Audit log INSERT under RLS (BUG-NEW-07 closed via m.129)
- ✅ Admin audit attribution (BUG-NEW-09 closed)
- ✅ System messages uuid.Nil sender FK fail (BUG-NEW-10 closed via m.130 — sender_id nullable)

---

## Bugs cross-référencés avec d'autres audits

| Bug | Aussi listé dans |
|---|---|
| BUG-FINAL-01 | auditsecurite SEC-FINAL-01 |
| BUG-FINAL-02, 03 | LiveKit OFF-LIMITS — flag only |
| BUG-FINAL-08 | auditsecurite SEC-FINAL-16, auditqualite QUAL-FINAL-B-08 |
| BUG-FINAL-13 | auditsecurite SEC-FINAL-15, auditqualite QUAL-FINAL-B-17 |
| BUG-FINAL-15 | auditqualite QUAL-FINAL-B-18 |

---

## Summary

| Severity | Count |
|---|---|
| CRITICAL | 1 (RLS rotation blocker) |
| HIGH | 8 (2 LiveKit OFF-LIMITS) |
| MEDIUM | 8 |
| LOW | 5 |
| **Total** | **22** |

(was 35 v1 + 20 v2 = 55 → 22 remaining + 33 closed across PRs #31-#66 and Phase 0)

**Top 3 deployment-risk priorities** :
1. **BUG-FINAL-01** — RLS migration blocks all reads when prod role rotated. Must fix before next prod role rotation. CRITICAL.
2. **BUG-FINAL-04** — `records.Update` swallowed at 3 sites = financial state drift. HIGH.
3. **BUG-FINAL-07** — Nested slice fields serialize null = TS client crashes on empty lists. HIGH.

**Bundle "stop the bleeding" (~ 2 jours)** = items BUG-FINAL-01 first (3 jours) + items 04-09 + 17 = closes deployment blocker + 6 critical correctness bugs + 1 contract bug.
