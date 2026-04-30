# Bugs à corriger

**Date** : 2026-04-30 (mise à jour post Phases 1-5Q ; audit précédent : 2026-04-29)
**Branche** : `main` @ `c8284526`

## Méthodologie

Audit exhaustif full-stack focalisé sur les bugs réels (races, state machines, error swallowing, null safety, off-by-one, edge cases, ressources) — pas les missing features. Pour chaque suspicion : localisation + recoupement avec les tests + lecture des call sites + cross-référence avec MEMORY.md pour ne pas re-flagger les items déférés (modération AWS region, FCM push, blocking, subscription user_id legacy).

Les bugs **strictement de sécurité** sont dans `auditsecurite.md` ; les bugs de **performance** dans `auditperf.md`. Ce fichier liste les bugs métier, race conditions, state machines, cohérence de données.

---

## CRITICAL (0)

All 5 v1 critical bugs are closed:

- ~~BUG-01 (ConfirmPayment Stripe verify)~~ closed in PR #33 (`90f4556b`)
- ~~BUG-02 (payment state guards)~~ closed in PR #35 (`913e8450 fix(payment): guard payment_record state transitions`)
- ~~BUG-03 (dispute restore propagation)~~ closed in PR #35 (`9ac5fcaf fix(dispute): propagate proposal update errors after restore` + `506253ac test(dispute): expand RespondToCancellation coverage`)
- ~~BUG-04 (Race Stripe Connect create)~~ closed in PR #33 (`18767345`)
- ~~BUG-05 (Outbox tx)~~ closed in PR #36 (`f4cf7d9d fix(search/outbox): commit profile mutation and reindex event in same tx`) — see BUG-NEW-05 for the residual cooldown drift

---

## HIGH (5)

V1 closed in this round (7 of 12) :

- ~~BUG-06 (WS sendBuffer block)~~ closed in PR #40 (`7306f055`)
- ~~BUG-07 (WS isLast race)~~ closed in PR #40 (`7306f055` same fix)
- ~~BUG-08 (mobile refresh single-flight)~~ closed in PR #31 (`a7497907 fix(mobile): single-flight refresh guard`)
- ~~BUG-09 (records.Update error swallowing locations 1 & 2)~~ closed in PR #35 (`a2ff5029 fix(payment): surface DB errors after Stripe PI re-fetch and transfer fail`) — see BUG-NEW-01 for residual sites 774/782/1008 still silenced
- ~~BUG-10 (webhook stripe_webhook_events table)~~ closed in PR #36 (`512eaa56`) — see BUG-NEW-08 for the post-claim handler-throw issue
- ~~BUG-11 (GetByIDForUpdate misnomer)~~ closed in PR #35 (`ee51f71f refactor(milestone): rename GetByIDForUpdate to GetByIDWithVersion`)
- ~~BUG-12 (embedded invalid_json)~~ closed in PR #40 (`0d8a266d`)
- ~~BUG-15 (context.Background overrides)~~ closed in PR #34 (`04319934`)
- ~~BUG-16 (notification worker pool)~~ closed in PR #36 (`3dbbf747`)
- ~~BUG-17 (upload goroutine ctx)~~ closed in PR #40 (`4ca6adfa fix(upload): track RecordUpload goroutines so SIGTERM drains cleanly`) — see BUG-NEW-02 for the `_ = ctx` discard

### BUG-13 : LiveKit room `maxParticipants=2` + reconnect → 3rd-participant rejection (still open, OFF-LIMITS)
- **Location** : `backend/internal/adapter/livekit/client.go:33-42`
- **Type** : state / UX
- **Trigger** : reconnect WS d'un user en plein appel (Wi-Fi/4G switch). L'ancienne participant entry n'est pas immédiatement nettoyée par LiveKit. Le user revient comme 3rd, rejeté.
- **Impact** : appel coupé, pas de recovery automatique.
- **Fix** : passer à `maxParticipants=4` (marge), OU identifier les participants par `identity` stable + autoriser le re-join à kicker l'ancienne session.

### BUG-14 : LiveKit token sans `CanPublish` / `CanSubscribe` (still open, OFF-LIMITS)
- **Location** : `backend/internal/adapter/livekit/client.go:44-60`
- **Type** : misconfig
- **Trigger** : `VideoGrant{Room, RoomJoin: true}` sans permissions explicites. Selon les versions du SDK les défauts diffèrent → publish refusé.
- **Impact** : appels muets / sans vidéo si LiveKit durcit ses défauts.
- **Fix** : expliciter `CanPublish: stripe.Bool(true)`, `CanSubscribe`, `CanPublishData`.

---

## MEDIUM (5)

### BUG-18 : API response envelope incohérente avec le contrat documenté (still open)
- **Location** : `backend/pkg/response/json.go:43-48`
- **Trigger** : `Error()` sort `{"error": "code", "message": "..."}` au lieu de `{"error": {"code", "message"}, "meta": {request_id}}` requis par CLAUDE.md. `JSON()` n'enveloppe pas dans `data:`.
- **Impact** : frontend doit jongler entre les deux formes ; OpenAPI inconsistante.
- **Fix** : `JSONData(w, status, data)` qui wrap, migrer les handlers.

### ~~BUG-19 (Empty list null vs [])~~ partially closed in PR #40 (`9488c684 fix(api): normalise nil top-level slices to empty arrays`) — only top-level slices normalized; nested slice fields still serialize null (see BUG-NEW-11)

### ~~BUG-20 (audit_repository unmarshal)~~ closed in PR #40 (`07bf8550 fix(audit): WARN+sentinel on corrupt metadata instead of silent swallow`)

### BUG-21 : VIES cache `_ = c.redisClient.Set(...)` (still open)
- **Location** : `backend/internal/adapter/vies/client.go:165`
- **Trigger** : cache write VIES ignoré.
- **Impact** : cache miss à la prochaine vérif TVA — perf, pas correctness.
- **Fix** : log warn.

### ~~BUG-22 (Notification queue Ack)~~ closed in PR #40 (`35475f26 fix(notif-queue): log Ack failures so doubled deliveries are observable`)

### BUG-23 : WS presence broadcast `_ = deps.Hub.broadcastToOthers` (still open)
- **Location** : `backend/internal/adapter/ws/connection.go:263`
- **Trigger** : erreur de marshalling envelope ignorée.
- **Impact** : typing indicator perdu silencieusement.
- **Fix** : log.

### BUG-24 : FCM device tokens jamais marqués stale (still open, deferred — depends on FCM full integration)
- **Location** : `backend/internal/adapter/fcm/` (à compléter quand FCM sera intégré)
- **Trigger** : pas de mécanisme pour invalider les `device_tokens` après échec FCM repeated (UNREGISTERED, INVALID_ARGUMENT).
- **Impact** : notification fan-out gaspille des appels API à des tokens morts.
- **Fix** : sur erreur Firebase, supprimer le row.

### ~~BUG-25 (Mobile FCM tap routing)~~ closed in PR #36 (`21431fae fix(mobile/fcm): tap navigates to relevant screen via rootNavigatorKey`) — see BUG-NEW-15 for cold-launch race

### BUG-26 : Mobile non-null assert `_formKey.currentState!` (still open)
- **Location** : `mobile/lib/features/auth/presentation/screens/login_screen.dart:34`, `register_screen.dart:41`, `agency_register_screen.dart:40`
- **Trigger** : si formulaire detached du tree au moment du tap (race), `currentState` est null → crash.
- **Impact** : edge crash rare.
- **Fix** : `if (_formKey.currentState?.validate() != true) return;`.

### BUG-27 : Search index publisher debounce 5min process-local (still open)
- **Location** : `backend/internal/app/searchindex/publisher.go:128-130`
- **Trigger** : `lastPublish map[debounceKey]time.Time` local au process. N instances backend → debounce N× moins efficace.
- **Impact** : pression légèrement accrue sur Typesense / OpenAI embeddings. Pas critique.
- **Fix** : déplacer dans Redis (`SETNX` avec TTL).

---

## LOW (8)

- **BUG-28** : `tx.Commit` ignoré dans `conversation_repository.go:43` (find existant) — log warn (still open)
- **BUG-29** : `defer tx.Rollback()` partout (~30 sites) perd l'erreur de Rollback — acceptable mais log idéal (still open)
- **BUG-30** : `idx_search_queries_search_id` UNIQUE peut bloquer inserts concurrents sur hot search_id — négligeable
- ~~BUG-31 (Webhook 7d TTL replay)~~ closed in PR #36 (composite Postgres + Redis fast-path)
- **BUG-32** : Migration 074 backfill DO $$ block monolithique — pour les futures migrations bulk-copy splitter en chunks
- **BUG-33** : `_ = json.Marshal(c)` dans `pkg/cursor/Encode` — petit struct toujours marshalable mais convention zéro-swallow → retourner `(string, error)` (still open)
- **BUG-34** : Pas de SSRF protection sur les URLs profil (PhotoURL/VideoURL) — flagged en sécurité (SEC-23) (still open)
- **BUG-35** : Mobile `chat_screen.dart` crée un Dio standalone qui bypass l'auth interceptor (timeouts hardcodés 30s/120s) — bug latent (still open)

---

## Verified shipped (no longer issues)

- ✅ Magic byte validation portfolio image (`UploadPortfolioImage:411-446`) — mais à étendre aux autres uploads (cf. SEC-09)
- ✅ Pagination cursor-based sur SearchProfiles
- ✅ Race condition payment_records — `payment_records.milestone_id UNIQUE` (migration 093)
- ✅ Optimistic concurrency milestones — version column + check WHERE
- ✅ WS conversation seq locking — `queryLockConversation` + `MAX(seq)+1` dans tx
- ✅ Conversation deduplication race — SERIALIZABLE + retry

---

## Bugs recoupés avec d'autres audits (cross-référence)

| Bug | Aussi listé dans |
|---|---|
| BUG-01 (ConfirmPayment fraud) | auditsecurite SEC-02 |
| BUG-04 (Race Stripe Connect create) | auditsecurite SEC-18 |
| BUG-10 (Webhook Postgres unused) | auditsecurite SEC-17 |
| BUG-16 (Notification worker bloquant) | auditperf PERF-B-12 |
| BUG-34 (SSRF profile URLs) | auditsecurite SEC-23 |

---

## TODO/FIXME inventory

| Location | Comment | Severity |
|---|---|---|
| `mobile/.../fcm_service.dart:148` | Use a global navigator key or GoRouter | HIGH (BUG-25) |
| `mobile/.../login_screen.dart:186` | navigate to forgot password | MEDIUM (feature gap) |
| `mobile/.../messaging_ws_service.dart:175` | replace with single-use, short-lived WS token | LOW (security hardening) |
| `mobile/.../referrer_profile_screen.dart:174` | wire referral_deals when backend ships | LOW (feature gap) |
| `backend/.../service_reputation.go:129` | paginate aggregator at >10k referrals | LOW (perf future) |
| `backend/.../referral_wallet.go:33` | group per currency | MEDIUM (multi-currency) |
| `web/.../pricing-format.ts:7` | when agency profile refactored | LOW |
| `backend/.../publisher.go:111-113` | "transactional variant planned for phase 2" | HIGH (BUG-05) |

---

## Top 15 bugs par dangerosité

| # | ID | Effort | Type |
|---|---|---|---|
| 1 | BUG-01 | 1h | Fraud Stripe |
| 2 | BUG-02 | 1h | State machine payment |
| 3 | BUG-03 | 30min | Cohérence dispute/proposal |
| 4 | BUG-04 | 1h | Race Stripe Connect |
| 5 | BUG-05 | 4h | Drift Postgres/Typesense |
| 6 | BUG-06 | 30min | Goroutine deadlock WS |
| 7 | BUG-07 | 30min | Race présence WS |
| 8 | BUG-08 | 1h | Race refresh mobile (bloquant SEC-06) |
| 9 | BUG-09 | 30min | Désynchro payment record |
| 10 | BUG-10 | 2h | Webhook double-traitement |
| 11 | BUG-11 | 1h | Pessimistic lock illusoire |
| 12 | BUG-13 | 1h | LiveKit reconnect |
| 13 | BUG-14 | 15min | LiveKit perms manquantes |
| 14 | BUG-15 | 30min | Context override |
| 15 | BUG-25 | 1h | FCM tap mobile |

**Bundle « stop the bleeding » (~ 1 jour)** = items 1-7 + 14 = ferme les 5 vrais bugs métier critiques + 2 races WS.

---

## Summary (v1 cleanup)

| Severity | Count |
|---|---|
| CRITICAL | 0 |
| HIGH | 2 (LiveKit OFF-LIMITS) |
| MEDIUM | 5 |
| LOW | 7 |
| **Total** | **14** |

(was 35 v1 → 14 remaining + 21 closed across PRs #31-#41 and Phase 0)

---

## v2 — found 2026-04-30

Bugs identified during the post-Phase 5Q sweep. Focus: cross-PR interactions, RLS migration foot-guns, and regressions in Phase 4 N (web RSC) / Phase 2 D-E (state machines, outbox).

### BUG-NEW-01: `RequestPayout` and `RetryFailedTransfer` still silence `records.Update` errors (Severity: HIGH)
- **Location**: `backend/internal/app/payment/service_stripe.go:774, 782, 1008`
- **Type**: error swallowing
- **Trigger**: BUG-09 fixed sites 120 and 365 (the `MarkTransferFailed` after PI re-fetch + the inline failure branch) but missed the same pattern at `RequestPayout` (lines 773-782) and `RetryFailedTransfer` (line 1008). After a successful Stripe `CreateTransfer`, the record is mutated to `MarkTransferred(transferID)` then the persistence is silently swallowed — so the funds moved on Stripe but our DB still says `TransferStatus = pending`. A retry will re-issue another transfer (idempotency-keyed, so Stripe blocks the duplicate, but the DB stays out of sync indefinitely).
- **Impact**: provider's wallet shows escrow funds that have already been transferred → over-counting. Same drift on `MarkTransferFailed` at line 774: a DB blip after a Stripe failure means the next retry can't see the failure flag and computes the wrong "last status".
- **Reproduction**:
  1. Mock `s.records.Update` to return an error after `MarkTransferred(transferID)` is called.
  2. Call `RequestPayout` and observe the function returns success.
  3. Re-fetch the record from DB → `TransferStatus = pending` (stale), but Stripe shows the transfer as completed.
- **Fix sketch**: replicate the BUG-09 pattern: `if uErr := s.records.Update(ctx, r); uErr != nil { slog.Error("payout: failed to persist MarkTransferred — record desynced from Stripe", ...); }`. Same at line 1008 / 774.
- **Test that would catch it**: extend `service_bug09_test.go` with cases that mock `Update` to return an error inside `RequestPayout` and `RetryFailedTransfer`, asserting both that an error is returned (or surfaced) and that the test logger captured a structured `slog.Error`.

### BUG-NEW-02: Upload `RecordUpload` goroutine cancellation context is built then discarded (Severity: HIGH)
- **Location**: `backend/internal/handler/upload_handler.go:243-247`
- **Type**: dead code / cancellation loss
- **Trigger**: BUG-17 fix tracks `RecordUpload` goroutines via `WaitGroup` for SIGTERM drain, AND constructs a `ctx, doneCancel := context.WithCancel(taskCtx)` plus a select goroutine on `shutdown.Done()` to cancel it. But line 247 is `_ = ctx` — the carefully-built cancellation context is never actually passed to `h.recorder.RecordUpload(...)`. The recorder uses its own internal 60s timeout instead. Comment admits: "The media service uses its own context internally — passing one in keeps the public API unchanged today, the tracking + cancellation contract is enforced here at the goroutine boundary."
- **Impact**: SIGTERM still has to wait for each in-flight upload's full 60s timeout before draining. Rekognition/S3 calls won't be aborted on shutdown. The WaitGroup tracks the goroutine but cannot abort the work.
- **Reproduction**: send SIGTERM mid-upload while Rekognition is processing → server shutdown blocks ~60s per in-flight upload.
- **Fix sketch**: extend `MediaRecorder` interface with `RecordUploadCtx(ctx, ...)` that propagates `ctx`. Replace `_ = ctx` with the actual call.
- **Test that would catch it**: integration test that spawns 3 concurrent uploads, sends `cancel()` on `shutdownCtx`, asserts all 3 return within 5s (not 60s).

### BUG-NEW-03: Pending events stuck forever in `processing` status after worker crash (Severity: CRITICAL)
- **Location**: `backend/internal/adapter/postgres/pending_event_queries.go:31`
- **Type**: data loss / stuck rows
- **Trigger**: `queryPopDuePendingEvents` filters `WHERE status IN ('pending', 'failed')`. When `processOne` claims a row (status → `processing`) and the worker crashes (panic, OOM, SIGKILL), the row stays at `processing` indefinitely. No watchdog, no stuck-row reset migration. The next worker pass can never re-claim it.
- **Impact**: a single worker crash mid-batch permanently loses search.reindex events → permanent Postgres/Typesense drift exactly contrary to BUG-05's purpose. Also affects: notification dead-letter handling and any future event types.
- **Reproduction**:
  1. Insert a `pending_events` row with `event_type='search.reindex'`.
  2. Manually update it to `status='processing'`.
  3. Run the worker — it never picks the row up.
- **Fix sketch**: either (a) add `OR (status = 'processing' AND updated_at < now() - interval '5 minutes')` to the WHERE clause to reclaim stale processing rows, OR (b) add a startup hook that resets `processing → failed` on worker boot.
- **Test that would catch it**: integration test inserts a row at `processing` with `updated_at = now() - 10min`, ticks the worker, asserts the row is re-claimed and processed.

### BUG-NEW-04: RLS migration 125 silently breaks ALL existing reads when production DB role is set up (Severity: CRITICAL — deployment time-bomb)
- **Location**: `backend/migrations/125_enable_row_level_security.up.sql` + every repo method that reads from `messages`, `conversations`, `invoice`, `proposals`, `proposal_milestones`, `notifications`, `disputes`, `audit_logs`, `payment_records` directly via `r.db.QueryContext` (i.e. NOT inside `RunInTxWithTenant`).
- **Type**: deployment foot-gun / silent zero-row regression
- **Trigger**: migration 125 enables `FORCE ROW LEVEL SECURITY` on 9 tables. The policies key on `current_setting('app.current_org_id', true)` which returns NULL if unset, evaluating policies to FALSE → all rows filtered. Today the migration owner role bypasses RLS, so production still works. **The moment** the dedicated `marketplace_app NOSUPERUSER NOBYPASSRLS` role is wired (per `backend/docs/rls.md`), every `r.db.QueryContext(ctx, "SELECT * FROM conversations ...")` call returns 0 rows — including admin endpoints, list views, RLS-aware writes that don't go through `RunInTxWithTenant`.
- **Impact**: app appears entirely empty for every user. Catastrophic outage at the moment the prod role is rotated. Admin moderation queries also break.
- **Reproduction**:
  ```sql
  CREATE ROLE test_app NOSUPERUSER NOBYPASSRLS LOGIN PASSWORD 'x';
  GRANT ALL ON ALL TABLES IN SCHEMA public TO test_app;
  -- Connect as test_app:
  SELECT count(*) FROM conversations;  -- returns 0 even though rows exist
  ```
- **Fix sketch**: every read path against a tenant-scoped table MUST be migrated to `RunInTxWithTenant(ctx, orgID, userID, fn)`. Currently only `profile.UpdateProfile` is migrated. Until then, **DO NOT rotate the prod DB role**. Track this as a Phase 6 blocker: either migrate every read path (~3 weeks of work given ~40 repos with reads) or add a temporary `SECURITY DEFINER` function wrapper. Admin endpoints need a separate `BYPASSRLS` role.
- **Test that would catch it**: an integration test that creates the non-superuser role, runs ALL repo Get/List methods that touch RLS tables, asserts none return zero rows for legitimate same-org data.

### BUG-NEW-05: Search publisher cooldown stamps before the outer tx commits — silent drift on rollback (Severity: HIGH)
- **Location**: `backend/internal/app/searchindex/publisher.go:144-173` (`PublishReindexTx`)
- **Type**: cooldown / state drift
- **Trigger**: `PublishReindexTx` flow: (1) `buildReindexEvent` checks `isWithinCooldown`, (2) `events.ScheduleTx` inserts in caller's tx, (3) `recordPublish` stamps `lastPublish` map. If the caller's outer tx rolls back AFTER step 3, the row is wiped but `lastPublish` retains the stamp. The next mutation within 5 min won't re-publish — silent drift. The comment at line 158 acknowledges this as "acceptable" but it's only acceptable when the only transient failure is in the publisher itself; here it can fail when the OUTER tx fails for any reason (e.g. the profile UPDATE rejects).
- **Impact**: profile UPDATE that fails at the last step (commit) yields a profile that "looks unchanged" + a cooldown that suppresses the next 5 minutes of attempts. User saves several times → looks like nothing happens; the search index never reflects until the cooldown expires.
- **Reproduction**: in a unit test, call `PublishReindexTx` inside a tx that rolls back. Wait < 5 min. Call `PublishReindexTx` again with a fresh tx that commits. Observe the second call is suppressed (cooldown).
- **Fix sketch**: register a `tx.AfterCommit(func() { p.recordPublish(key) })` hook OR move the cooldown stamp out of the function and have the caller invoke it explicitly after a successful commit.
- **Test that would catch it**: a test where `RunInTx` rolls back, and the next tx (within cooldown) successfully schedules an event.

### BUG-NEW-06: Stripe webhook handler errors are silenced and treated as success — Stripe drops the event (Severity: HIGH)
- **Location**: `backend/internal/handler/stripe_handler.go:155-216`
- **Type**: state drift / data loss
- **Trigger**: webhook dispatcher claims the event in idempotency BEFORE running the handler. Each handler (`handleSubscriptionCreated`, `handleInvoicePaid`, `handleChargeRefunded`, etc.) logs errors via `slog.Error` but returns silently — the dispatcher always sends `200 OK`. Stripe sees 200, marks event as delivered, and the claim has been recorded. **A handler crash mid-execution cannot be retried** because: (1) Stripe won't retry (200 response), (2) idempotency claim is persistent. The state transition is permanently lost.
- **Impact**: example — a `customer.subscription.created` event whose handler hits a transient DB error (line 277) results in: subscription row not created, but the user is charged on Stripe and our system never reflects it. Customer gets a charge but no Premium activation.
- **Reproduction**: mock `subscriptionSvc.RegisterFromCheckout` to return an error. Send a `customer.subscription.created` webhook. Observe 200 OK + permanent claim → state never repaired.
- **Fix sketch**: the idempotency claim should be GATED on successful handler execution. Move claim AFTER the dispatch, OR use a two-phase commit: insert `stripe_webhook_events` with `status='processing'`, run handler, mark `status='done'` on success; on next replay, the row is `processing` so the handler gets the chance to retry.
- **Test that would catch it**: integration test sends a `subscription.created` event with a failing service mock, then re-sends the same event ID, asserts the second delivery is processed (not deduped as "already claimed").

### BUG-NEW-07: Audit log INSERT will be rejected by RLS once production DB role is wired (Severity: HIGH — deployment time-bomb)
- **Location**: `backend/migrations/125_enable_row_level_security.up.sql:190-193` + `backend/internal/adapter/postgres/audit_repository.go:53` (`Log` uses `r.db` directly, not in tx)
- **Type**: RLS interaction with INSERT
- **Trigger**: the audit_logs RLS policy uses `USING (user_id = current_setting('app.current_user_id', true)::uuid)`. Without an explicit `WITH CHECK` clause, PostgreSQL applies USING to INSERTs as well. The `Log` method runs outside any transaction, so `app.current_user_id` is unset (NULL), and EVERY audit INSERT will fail the RLS check once the prod role is set up.
- **Impact**: `slog.Warn("audit: insert failed", ...)` after every login, suspend, ban — total loss of audit trail in prod.
- **Reproduction**: same as BUG-NEW-04 reproduction; INSERT into `audit_logs` from the non-superuser role without setting `app.current_user_id` → fails with "new row violates row-level security policy".
- **Fix sketch**: either (a) wrap audit Log in `RunInTxWithTenant(ctx, uuid.Nil, actorUserID, ...)`, OR (b) split the RLS policy into separate `FOR SELECT USING (...)` and `FOR INSERT WITH CHECK (true)` policies (admin-only writes; users only read their own).
- **Test that would catch it**: cross-tenant integration test that asserts an admin user can insert an audit row about another user (currently impossible under the current policy).

### BUG-NEW-08: Stripe Connect error messages leak to the API response (Severity: MEDIUM)
- **Location**: `backend/internal/handler/embedded_handler.go:140, 235, 246`
- **Type**: information disclosure
- **Trigger**: `res.Error(w, http.StatusInternalServerError, "stripe_error", err.Error())` and similar pass the raw Stripe SDK error directly to the client. Stripe errors include account IDs, request IDs, Stripe internal request paths, sometimes truncated lookup keys. The `invalid_json` branch at line 140 also passes `jsonErr.Error()` which leaks Go struct field names like `"json: cannot unmarshal number into Go struct field accountSessionRequest.country"`.
- **Impact**: third-party can probe error responses to enumerate internal IDs, struct shapes. Also breaks DTO contract by exposing implementation language hints.
- **Reproduction**: POST malformed body to `/api/v1/payment-info/account-session` → response contains "Go struct field accountSessionRequest.country".
- **Fix sketch**: replace with sanitized messages: `"invalid_json", "request body could not be parsed as JSON"`, `"stripe_error", "the Stripe operation failed"`. Keep details in the slog.Error.
- **Test that would catch it**: handler test posts invalid JSON, asserts the response body does NOT contain "Go struct field" or "json:".

### BUG-NEW-09: Admin audit log entries record the SUSPENDED user as actor instead of the admin (Severity: HIGH)
- **Location**: `backend/internal/app/admin/service.go:237-247, 262-269, 286-294, 347` and `logAudit` at 360-372
- **Type**: audit attribution bug
- **Trigger**: `SuspendUser`, `BanUser`, `UnsuspendUser`, etc. call `s.logAudit(ctx, audit.NewEntryInput{ UserID: &userID, ... })` where `userID` is the SUSPENDED user. The actor (admin) is never set. The intent of `audit_logs.user_id` is the actor (forensic trail of who did what); recording the target instead means the audit table cannot answer "what actions did admin X take?".
- **Impact**: forensic queries return wrong attribution. RGPD Art. 30 records of processing activity are mis-attributed.
- **Reproduction**: admin A suspends user B. Query `SELECT * FROM audit_logs WHERE action='admin.user.suspend' AND user_id = <admin A id>` → empty. Query with user_id = B → returns the row (wrong).
- **Fix sketch**: extract admin user_id from middleware context, set `UserID: &adminUserID` on each `logAudit` call. Move `userID` (the target) to `ResourceID`.
- **Test that would catch it**: handler test that calls `SuspendUser` with admin ctx, then asserts the audit row's `user_id` matches the admin (not the suspended user).

### BUG-NEW-10: `restoreProposalAndDistribute` system messages use uuid.Nil senderID — silently fail to insert (Severity: MEDIUM)
- **Location**: `backend/internal/app/dispute/service_helpers.go:115, 117`
- **Type**: known broken FK / silent loss
- **Trigger**: at line 814 of `service_actions.go` the comment explicitly says: "Use the admin's user ID as sender: the messages table has a FK on sender_id → users(id), so uuid.Nil silently fails the insert." But `restoreProposalAndDistribute` at line 115/117 (the auto-resolution path triggered by RespondToCounterProposal AND admin resolution) passes `uuid.Nil`. The FK violation will WARN log + continue (line 31 of service_helpers.go). Users get the dispute-resolved notification but never the `proposal_completed` / `evaluation_request` system bubbles.
- **Impact**: missing system messages in chat post-dispute resolution. Users see a stale conversation timeline.
- **Reproduction**: resolve a dispute via admin. Check the conversation messages — `proposal_completed` and `evaluation_request` are absent.
- **Fix sketch**: pass the admin/responder user ID through `restoreProposalAndDistribute`, OR introduce a real "system" user with a fixed UUID inserted via migration so the FK is satisfied.
- **Test that would catch it**: integration test that resolves a dispute admin-side, then queries `messages` for the conversation, asserts both `proposal_completed` and `evaluation_request` rows exist.

### BUG-NEW-11: Empty list normalization only handles top-level slices — nested slice fields still serialize null (Severity: MEDIUM)
- **Location**: `backend/pkg/response/json.go:58-67` (`NilSliceToEmpty` with `reflect.Kind() != reflect.Slice` early return) + every DTO with a slice field that is set from a possibly-nil domain field (`backend/internal/handler/dto/response/job.go:43 Skills: j.Skills`, etc.)
- **Type**: contract drift / TS client crash
- **Trigger**: BUG-19 fix wraps top-level encode arg via `NilSliceToEmpty(data)`. But struct DTOs containing slice fields (like `JobResponse.Skills`, `ProfileResponse.Languages`, `TeamMemberResponse.Permissions`) keep `null` when their source `j.Skills` is nil. Web/admin/mobile TS clients calling `.length` on these crash.
- **Impact**: client crashes when loading a Job that has no skills set (legitimate empty case).
- **Reproduction**:
  1. Create a Job with empty `skills` array.
  2. GET `/api/v1/jobs/{id}` → response contains `"skills": null`.
  3. Web client does `job.skills.length` → TypeError.
- **Fix sketch**: either (a) recursively normalise via reflection on Marshal time, (b) introduce per-DTO `nilToEmptyStrings(j.Skills)` conversion (helper exists in `helpers.go` for some DTOs but not job/proposal/etc.), or (c) annotate slice fields with a custom JSON marshaller.
- **Test that would catch it**: handler test that creates a Job with `Skills: nil`, GETs it, asserts response body contains `"skills": []` (not `null`).

### BUG-NEW-12: RSC public listings fall back to `http://localhost:8080` instead of the actual backend port 8083 (Severity: MEDIUM)
- **Location**: `web/src/features/provider/api/search-server.ts:64`
- **Type**: misconfig / dead fallback
- **Trigger**: `const url = ${API_BASE_URL || "http://localhost:8080"}/api/v1/search?...`. The actual backend port (per project memory) is **8083**, not 8080. In any environment where `API_BASE_URL` is unset, the fetch hits a dead port.
- **Impact**: SEO listings (the entire reason for PR #41 perf work) silently render the empty fallback when `API_BASE_URL` is missing. No error logged because of the `try { } catch { return null }` swallow.
- **Reproduction**: unset `NEXT_PUBLIC_API_URL` / `API_BASE_URL`, run `npm run build && npm start`, visit `/agencies` → page renders with no documents.
- **Fix sketch**: change fallback to `http://localhost:8083` to match the actual dev port. OR throw on missing `API_BASE_URL` so misconfigs are loud at build time.
- **Test that would catch it**: a smoke test that runs `next build` with API_BASE_URL unset and asserts a build error or a deterministic fallback hit.

### BUG-NEW-13: Admin `<Suspense>` wraps `<Routes>` — layout shell flashes between every navigation (Severity: MEDIUM, UX)
- **Location**: `admin/src/app/router.tsx:96-122`
- **Type**: UX regression
- **Trigger**: `<Suspense fallback={<RouteSkeleton />}> <Routes> ... </Routes> </Suspense>`. The Suspense boundary wraps the entire Routes tree, including the `<AdminLayout />` parent route. When navigating from `/users` to `/jobs`, the lazy import triggers Suspense, the fallback replaces the entire layout (sidebar + header gone) for the duration of the chunk download.
- **Impact**: flash of full-page skeleton instead of in-content skeleton. Worse on slow networks. Defeats the polish work of PR #41.
- **Reproduction**: throttle network in DevTools to "Slow 3G", navigate between admin pages → entire layout disappears.
- **Fix sketch**: nest the Suspense inside `<AdminLayout />`'s `<Outlet />` consumer:
  ```tsx
  // In AdminLayout:
  <main>
    <Suspense fallback={<RouteSkeleton />}>
      <Outlet />
    </Suspense>
  </main>
  ```
- **Test that would catch it**: Playwright test navigates between two lazy admin routes, asserts the sidebar is continuously present in the DOM during navigation.

### BUG-NEW-14: `MaxBytesReader` cap is reset to nil-writer inside `validateAndBuildKey` (Severity: MEDIUM)
- **Location**: `backend/internal/handler/upload_handler.go:461`
- **Type**: 413 not surfaced
- **Trigger**: each upload handler does `r.Body = http.MaxBytesReader(w, r.Body, maxSize)` THEN calls `validateAndBuildKey` which AGAIN does `r.Body = http.MaxBytesReader(nil, r.Body, maxSize)`. The second call replaces the first reader; the nil writer means no automatic 413 short-circuit — instead the multipart reader keeps reading until it hits the limit, then errors out.
- **Impact**: 413 response is still produced (manually via `isMaxBytesError`) but the writer hint is lost. Subtle behavioural change: previously the `w` arg let `MaxBytesReader` automatically write the 413; now it relies entirely on the manual `isMaxBytesError` branch.
- **Reproduction**: send a multipart POST larger than `maxSize` to any upload endpoint → behaviour is correct today (manual 413) but the redundant double-wrapping is bug-prone.
- **Fix sketch**: drop the inner `r.Body = http.MaxBytesReader(nil, r.Body, maxSize)` line. The handler-level cap is sufficient.
- **Test that would catch it**: handler test verifies that ONLY one MaxBytesReader is in the chain (either via stack inspection or by checking that the 413 path still works without the inner wrap).

### BUG-NEW-15: Mobile FCM cold-launch tap can be silently dropped (Severity: MEDIUM)
- **Location**: `mobile/lib/core/notifications/fcm_service.dart:213-236` (`_navigateFromData`)
- **Type**: race / UX
- **Trigger**: cold-launch tap path waits 100ms then drops if `rootNavigatorKey.currentContext` is still null. On slow Android devices and during heavy first-frame work, 100ms is too short — first frame can take 500-1500ms post-Firebase init. Also, `GoRouter.push(route)` adds onto the stack: if the user is unauthenticated (no token), the destination chat/proposal will overlay the login redirect → confusion.
- **Impact**: tap on push from terminated state ignored on slow devices. Or routed onto unauthenticated screen.
- **Reproduction**:
  1. Force-stop the app.
  2. Tap a push notification.
  3. On a slow device, the navigation drops silently.
- **Fix sketch**: instead of fixed 100ms, listen for the first frame via `WidgetsBinding.instance.addPostFrameCallback` and push then. Also gate on auth state — if unauthenticated, store the intent and push after login.
- **Test that would catch it**: integration test that simulates a cold-launch push tap on a delayed-frame harness, asserts navigation eventually happens.

### BUG-NEW-16: Wallet referral commissions silently swallow DB errors (Severity: MEDIUM)
- **Location**: `backend/internal/app/payment/service_stripe.go:660-700`
- **Type**: error swallowing / UX confusion
- **Trigger**: `if sum, err := s.referralWallet.GetReferrerSummary(...); err == nil { ... }` and `if recent, err := s.referralWallet.RecentCommissions(...); err == nil { ... }` — on transient DB errors, the user sees `wallet.Commissions` as zero/empty even though commissions exist. Comment claims this is intentional ("a broken referral read never takes down the provider-side wallet") but it produces false data without any signal.
- **Impact**: user thinks they have no commissions; refresh fixes it but they may have already disputed a missing payout.
- **Fix sketch**: at minimum `slog.Warn("wallet: failed to load referral summary", "error", err)`. Better: surface a `commissions_partial: true` flag on the response so the UI can show "data temporarily unavailable".
- **Test that would catch it**: handler test mocks `referralWallet.GetReferrerSummary` to return an error, asserts the wallet response includes a `commissions_partial: true` flag.

### BUG-NEW-17: WS broadcastPresenceChange on disconnect uses `context.Background()` instead of WithoutCancel (Severity: LOW)
- **Location**: `backend/internal/adapter/ws/connection.go:153`
- **Type**: inconsistency / minor cancellation gap
- **Trigger**: line 75 uses `bgCtx := context.WithoutCancel(r.Context())` for the OnConnect broadcast (correct pattern). Line 153 (OnDisconnect) uses `context.Background()` — loses any baggage/trace context the request had. Inconsistency.
- **Impact**: structured logs lose trace correlation on disconnect.
- **Fix sketch**: store the original request context on the Client struct, use `WithoutCancel(client.requestCtx)` on disconnect.
- **Test that would catch it**: assertion in a logging test that disconnect logs carry the same request_id as the connect logs for the same connection.

### BUG-NEW-18: `RetryFailedTransfer` directly mutates `record.TransferStatus` without state machine guard (Severity: MEDIUM)
- **Location**: `backend/internal/app/payment/service_stripe.go:992`
- **Type**: state machine bypass
- **Trigger**: `record.TransferStatus = domain.TransferPending` is a raw field assignment — bypasses the `MarkTransferFailed` / `MarkTransferred` / `ApplyDisputeResolution` guarded mutators that BUG-02 added. There's no `MarkTransferPending()` method, no validation that the previous state was `TransferFailed`. A retry on an already-completed transfer would silently revert it to pending and trigger another Stripe transfer (idempotency-keyed, so Stripe blocks, but DB drifts).
- **Impact**: state machine guards aren't truly closed — there's still a back-door. Future refactors that add new transitions will miss this site.
- **Fix sketch**: add `func (r *PaymentRecord) MarkTransferRetrying() error { if r.TransferStatus != TransferFailed { return ErrInvalidStateTransition; } r.TransferStatus = TransferPending; ... }` and use it.
- **Test that would catch it**: state machine test ensures `RetryFailedTransfer` rejects records that aren't in `TransferFailed`.

### BUG-NEW-19: Conversation repo `tx.Commit` error swallowed on existing conversation lookup (Severity: LOW)
- **Location**: `backend/internal/adapter/postgres/conversation_repository.go:43`
- **Type**: error swallowing
- **Trigger**: when an existing conversation is found (line 41), the function does `_ = tx.Commit()` and returns success. If the commit fails (e.g., serialization conflict from another concurrent caller), the caller is told the conversation exists when actually nothing was committed. This is BUG-28 from v1, still open.
- **Impact**: minor — the conversation does exist, just the read tx couldn't commit. Likely no user-visible impact, but masks real DB issues.
- **Fix sketch**: `if err := tx.Commit(); err != nil { slog.Warn("...") }`.

### BUG-NEW-20: Dispute system message sender uuid.Nil silent FK fail (re-flagged scoped to BUG-NEW-10) (Severity: LOW)
Already covered in BUG-NEW-10.

---

## Summary (after v2)

| Severity | v1 remaining | v2 new |
|---|---|---|
| CRITICAL | 0 | 2 (BUG-NEW-03, BUG-NEW-04) |
| HIGH | 2 (BUG-13/14 LiveKit OFF-LIMITS) | 5 (BUG-NEW-01, 02, 05, 06, 07, 09) |
| MEDIUM | 5 | 7 |
| LOW | 7 | 2 |
| **Total** | **14** | **20** |

**Top 5 v2 priorities by deployment risk**:
1. BUG-NEW-04 (RLS blocks all reads when prod role is set up) — must fix before next prod deploy
2. BUG-NEW-03 (worker crash leaves processing rows stuck) — silent data loss
3. BUG-NEW-07 (audit log INSERT blocked under RLS) — total audit trail loss
4. BUG-NEW-06 (Stripe webhook handler errors lose state) — financial state drift
5. BUG-NEW-09 (admin audit attribution wrong) — forensic / RGPD impact
