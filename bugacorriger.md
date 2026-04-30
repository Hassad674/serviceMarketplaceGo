# Bugs à corriger

**Date** : 2026-04-29 (audit précédent : 2026-03-30, obsolète)
**Branche** : `main` @ `a0d268a4`

## Méthodologie

Audit exhaustif full-stack focalisé sur les bugs réels (races, state machines, error swallowing, null safety, off-by-one, edge cases, ressources) — pas les missing features. Pour chaque suspicion : localisation + recoupement avec les tests + lecture des call sites + cross-référence avec MEMORY.md pour ne pas re-flagger les items déférés (modération AWS region, FCM push, blocking, subscription user_id legacy).

Les bugs **strictement de sécurité** sont dans `auditsecurite.md` ; les bugs de **performance** dans `auditperf.md`. Ce fichier liste les bugs métier, race conditions, state machines, cohérence de données.

---

## CRITICAL (5)

### BUG-01 : ConfirmPayment permet d'activer une proposal sans paiement Stripe réel (= aussi SEC-02)
- **Location** : `backend/internal/handler/proposal_handler.go:360-401`, `internal/app/payment/service_stripe.go:137-150`
- **Type** : business logic flaw / fraud
- **Trigger** : `MarkPaymentSucceeded` lit le record local et appelle `record.MarkPaid()` sans consulter Stripe. Un client peut faire passer le record en `succeeded` côté client → proposal `active` → `RequestCompletion`.
- **Impact** : fraude directe — fonds escrow virtuels jamais encaissés mais transférés au prestataire.
- **Fix** : avant `MarkPaymentSucceeded`, `stripe.PaymentIntents.Get(record.StripePaymentIntentID)` et vérifier `pi.Status == "succeeded"`.

### BUG-02 : `ApplyDisputeResolution` / `MarkRefunded` / `MarkFailed` sans guards d'état
- **Location** : `backend/internal/domain/payment/payment_record.go:98-141`
- **Type** : state machine
- **Trigger** : ces 3 méthodes ne vérifient pas l'état actuel. Un record `transferred` peut être ré-appliqué une dispute resolution qui écrase `ProviderPayout`. Un retry/replay de webhook → écrasement comptable.
- **Impact** : la dispute resolution peut écraser `ProviderPayout` à 0 et "perdre" l'argent du prestataire.
- **Fix** : `MarkRefunded` exige `Status == Succeeded` ; `MarkFailed` exige `Status == Pending` ; `ApplyDisputeResolution` exige `Status == Succeeded && TransferStatus != Completed`.

### BUG-03 : Erreurs silencieuses sur restauration de proposal après cancel/respond de litige
- **Location** : `backend/internal/app/dispute/service_actions.go:471, 534`
- **Type** : error swallowing / state divergence
- **Trigger** : `_ = s.proposals.Update(ctx, p)` après `RestoreFromDispute`. Si l'UPDATE échoue (DB blip, conflit version), le proposal reste en `disputed` alors que la dispute est `cancelled`.
- **Impact** : paire dispute/proposal incohérente — mission gelée, pas de recovery automatique. Utilisateur ne comprend pas pourquoi.
- **Fix** : propager l'erreur ou minimum `slog.Error` + métrique + enqueue `pendingevent` pour rattraper.

### BUG-04 : Race condition création compte Stripe Connect (= aussi SEC-18)
- **Location** : `backend/internal/handler/embedded_handler.go:235-267` (`resolveStripeAccount`)
- **Type** : race / TOCTOU
- **Trigger** : 2 requêtes `CreateAccountSession` concurrentes du même org → 2 `GetStripeAccount` vides → 2 comptes Stripe créés, le premier orphelin.
- **Impact** : comptes Stripe orphelins inactifs côté DB mais référencés Stripe-side. KYC potentiellement soumis sur le "perdant".
- **Fix** : `pg_advisory_xact_lock(hashtext(org_id))` ou `SELECT ... FOR UPDATE` sur la ligne org avant check + create.

### BUG-05 : Outbox event publish hors transaction → drift Postgres/Typesense permanent
- **Location** : `backend/internal/app/searchindex/publisher.go:114-154` et callers (`internal/app/freelanceprofile/service.go:74-82`, `internal/app/profile/service.go:104`)
- **Type** : outbox / data drift
- **Trigger** : services profil committent leur Update DB, puis `publishReindex` HORS transaction. Si Schedule échoue (DB blip), erreur swallowée (`slog.Warn`) → profil mis à jour, jamais réindexé. Commentaire publisher : "phase 2 once the need arises".
- **Impact** : drift permanent jusqu'à reindex complet manuel ou autre update qui re-trigger.
- **Fix** : la mutation profil + l'INSERT dans `pending_events` doivent être dans la même transaction. Le worker drainera ensuite.

---

## HIGH (12)

### BUG-06 : WS sendBuffer-bloquant → deadlock du readPump
- **Location** : `backend/internal/adapter/ws/connection.go:261, 317`
- **Type** : goroutine deadlock
- **Trigger** : `client.Send <- envelope` dans le readPump. Channel buffered à 64. Si writePump est lent (mobile background, Wi-Fi/4G switch), buffer plein → readPump bloque jusqu'à pongWait timeout (60s). Aucun unregister automatique.
- **Impact** : connexion WS zombie, présence "online" indéfinie ; cumulativement leak de Client objects + writePump goroutines.
- **Fix** : pattern existant ligne 167-171 : `select { case client.Send <- envelope: default: slog.Warn("send buffer full"); /* drop */ }`.

### BUG-07 : WS `isLast` race — présence offline déclenchée à tort
- **Location** : `backend/internal/adapter/ws/connection.go:101-114`
- **Type** : race
- **Trigger** : `isLast := deps.Hub.ConnectionCount(client.UserID) <= 1` calculé AVANT `unregister`. Une nouvelle connexion peut s'enregistrer entre les deux ; ou inversement, deux fermetures concurrentes voient chacune `<=1`.
- **Impact** : flash "online → offline → online" dans l'UI ; broadcast contradictoire.
- **Fix** : `removeClient` retourne `wasLast bool` sous le même lock, `SetOffline` que sur cette valeur.

### BUG-08 : Refresh token reentrancy — Mobile peut rafraîchir 2× en concurrent
- **Location** : `mobile/lib/core/network/api_client.dart:67-84`
- **Type** : race
- **Trigger** : 2 requêtes API simultanées qui retournent 401. Le `_onError` interceptor lance 2 `_tryRefreshToken` en parallèle. Si rotation activée (cf. SEC-06), le second échoue avec "blacklisted" → user déconnecté à tort. Sans rotation : `saveTokens` concurrent peut corrompre le secure storage.
- **Impact** : déconnexions intempestives quand rotation sera livrée. **Bloquant pour le fix de SEC-06**.
- **Fix** : single-flight pattern : `Future<bool>? _refreshInFlight` mémorisé sur ApiClient.

### BUG-09 : `createPaymentIntentFromExisting` ignore erreurs de update payment_record
- **Location** : `backend/internal/app/payment/service_stripe.go:120-123, 292`
- **Type** : error swallowing
- **Trigger** : si on récupère un PI dont l'ID a changé Stripe-side (rare), `_ = s.records.Update(ctx, existing)` swallow. Idem `MarkTransferFailed` ligne 292.
- **Impact** : désynchro payment_record vs Stripe. Le record continue avec l'ancien `StripePaymentIntentID`, transfert/refund cible un PI fantôme.
- **Fix** : minimum `slog.Error("failed to update record after PI re-fetch", ...)` ; idéalement remonter l'erreur car la persistance d'un nouveau PI ID est critique.

### BUG-10 : Webhook `stripe_webhook_events` table créée mais jamais utilisée (= aussi SEC-17)
- **Location** : `migrations/089_create_stripe_webhook_events.up.sql`, `internal/handler/stripe_handler.go:136-147`, `internal/adapter/redis/webhook_idempotency.go:50-55`
- **Type** : idempotency / data loss
- **Trigger** : idempotency uniquement Redis. Si Redis tombe, `TryClaim` retourne `(true, err)` ("claim conservatoire") → webhook traité plusieurs fois si Stripe retry.
- **Impact** : pendant panne Redis, double-traitement de `subscription.created`, double émission facture FAC-NNNNNN, double-fund de jalons.
- **Fix** : utiliser la table Postgres comme source de vérité (INSERT UNIQUE event_id), Redis fast-path.

### BUG-11 : `GetByIDForUpdate` libère le lock immédiatement
- **Location** : `backend/internal/adapter/postgres/milestone_repository.go:91-114`
- **Type** : pessimistic lock illusoire
- **Trigger** : `BeginTx → SELECT FOR UPDATE → tx.Commit()` — le commit relâche le lock. Le caller (`withMilestoneLock`) lit la version, mutate en mémoire, puis Update re-vérifie la version. La sécurité repose entièrement sur le `WHERE version = $2` dans Update.
- **Impact** : pas de bug aujourd'hui (optimistic concurrency rattrape) mais le `FOR UPDATE` ne fait rien d'utile et le commentaire ligne 88-89 ("brief transaction window is intentional") masque que c'est de l'optimistic, pas du pessimistic. Si quelqu'un retire le check version (refacto), la race redevient violente.
- **Fix** : conserver la transaction ouverte et passer le `*sql.Tx` (vrai pessimistic), OU renommer `GetByIDWithVersion` et supprimer le `FOR UPDATE`.

### BUG-12 : Empty JSON body unmarshalled silently (Stripe Embedded)
- **Location** : `backend/internal/handler/embedded_handler.go:96-99`
- **Type** : error swallowing
- **Trigger** : `_ = json.Unmarshal(body, &req)`. Body malformé → req vide → erreur générique "country is required" qui n'explique pas que le client a envoyé du JSON invalide.
- **Impact** : UX dégradée, debug compliqué.
- **Fix** : vérifier l'erreur, retourner 400 `invalid_json` quand body non-vide mais malformé.

### BUG-13 : LiveKit room `maxParticipants=2` + reconnect → 3rd-participant rejection
- **Location** : `backend/internal/adapter/livekit/client.go:33-42`
- **Type** : state / UX
- **Trigger** : reconnect WS d'un user en plein appel (Wi-Fi/4G switch). L'ancienne participant entry n'est pas immédiatement nettoyée par LiveKit. Le user revient comme 3rd, rejeté.
- **Impact** : appel coupé, pas de recovery automatique.
- **Fix** : passer à `maxParticipants=4` (marge), OU identifier les participants par `identity` stable + autoriser le re-join à kicker l'ancienne session.

### BUG-14 : LiveKit token sans `CanPublish` / `CanSubscribe`
- **Location** : `backend/internal/adapter/livekit/client.go:44-60`
- **Type** : misconfig
- **Trigger** : `VideoGrant{Room, RoomJoin: true}` sans permissions explicites. Selon les versions du SDK les défauts diffèrent → publish refusé.
- **Impact** : appels muets / sans vidéo si LiveKit durcit ses défauts.
- **Fix** : expliciter `CanPublish: stripe.Bool(true)`, `CanSubscribe`, `CanPublishData`.

### BUG-15 : `context.Background()` override silencieux
- **Location** : `internal/search/antigaming/pipeline.go:74` (overwrite ctx d'entrée), `internal/app/proposal/service_scheduler.go:109` (Background dans méthode qui reçoit ctx)
- **Type** : timeout/cancellation loss
- **Trigger** : la cancellation upstream est silencieusement ignorée.
- **Impact** : si client annule la requête (back), goroutine continue d'attendre.
- **Fix** : `context.WithTimeout(ctx, 30*time.Second)` à partir du ctx reçu.

### BUG-16 : Notification worker single-threaded + `time.Sleep(delay)` bloquant (= aussi PERF-B-12)
- **Location** : `backend/internal/app/notification/worker.go:121-143`
- **Type** : throughput / latency
- **Trigger** : 1 notif qui timeout 3× bloque le worker 7s. Tous les jobs en file derrière attendent.
- **Impact** : burst de 100 notifs après inactivité → p99 multi-secondes.
- **Fix** : ré-enqueue avec `available_at = now() + delay`, ou paralléliser N=3-5 workers.

### BUG-17 : Photo upload — goroutine `media.RecordUpload` détachée sans context lié
- **Location** : `backend/internal/handler/upload_handler.go:113, 175, 237, 282, 400, 487`
- **Type** : goroutine leak / lost work
- **Trigger** : `go h.mediaSvc.RecordUpload(...)` détaché. Le service crée son propre context (60s timeout), mais pas de tracking ni cancellation lors d'un shutdown gracieux.
- **Impact** : work perdu si le process est tué pendant l'upload (downloads + moderation Rekognition tronqués).
- **Fix** : passer un context shared (`app.Context`) ou worker queue.

---

## MEDIUM (10)

### BUG-18 : API response envelope incohérente avec le contrat documenté
- **Location** : `backend/pkg/response/json.go:43-48`
- **Trigger** : `Error()` sort `{"error": "code", "message": "..."}` au lieu de `{"error": {"code", "message"}, "meta": {request_id}}` requis par CLAUDE.md. `JSON()` n'enveloppe pas dans `data:`.
- **Impact** : frontend doit jongler entre les deux formes ; OpenAPI inconsistante.
- **Fix** : `JSONData(w, status, data)` qui wrap, migrer les handlers.

### BUG-19 : Empty list responses serialise à `null` au lieu de `[]`
- **Location** : divers handlers de liste
- **Trigger** : `nil` slice Go → `null` JSON. CLAUDE.md mandate `data: []`.
- **Impact** : clients TS qui font `.length` sur null crashent.
- **Fix** : dans chaque handler de liste, `if items == nil { items = []*Type{} }`.

### BUG-20 : `_ = json.Unmarshal(metadata, ...)` dans audit_repository
- **Location** : `backend/internal/adapter/postgres/audit_repository.go:248`
- **Trigger** : metadata corrompue → unmarshal fail → entry rendue avec metadata vide silencieusement.
- **Fix** : minimum `slog.Warn`.

### BUG-21 : VIES cache `_ = c.redisClient.Set(...)`
- **Location** : `backend/internal/adapter/vies/client.go:165`
- **Trigger** : cache write VIES ignoré.
- **Impact** : cache miss à la prochaine vérif TVA — perf, pas correctness.
- **Fix** : log warn.

### BUG-22 : Notification queue `Ack` sans erreur retournée
- **Location** : `backend/internal/adapter/redis/notification_queue.go:96, 103`
- **Trigger** : Ack qui échoue = message redélivré.
- **Impact** : notifications doublées au prochain cycle si Redis blip.
- **Fix** : log + métrique.

### BUG-23 : WS presence broadcast `_ = deps.Hub.broadcastToOthers`
- **Location** : `backend/internal/adapter/ws/connection.go:197`
- **Trigger** : erreur de marshalling envelope ignorée.
- **Impact** : typing indicator perdu silencieusement.
- **Fix** : log.

### BUG-24 : FCM device tokens jamais marqués stale
- **Location** : `backend/internal/adapter/fcm/` (à compléter quand FCM sera intégré)
- **Trigger** : pas de mécanisme pour invalider les `device_tokens` après échec FCM repeated (UNREGISTERED, INVALID_ARGUMENT).
- **Impact** : notification fan-out gaspille des appels API à des tokens morts.
- **Fix** : sur erreur Firebase, supprimer le row.

### BUG-25 : Mobile FCM tap ne navigue pas
- **Location** : `mobile/lib/core/notifications/fcm_service.dart:140-152`
- **Trigger** : `_navigateFromData` est un `debugPrint` + TODO.
- **Impact** : tap sur push ouvre l'app sur le dernier écran, pas sur la conversation/proposal pertinente.
- **Fix** : injecter `GoRouter` global, router selon `data['type']` (proposal → `/projects/detail/:id`, message → `/chat/:id`, review → `/profile`).

### BUG-26 : Mobile non-null assert `_formKey.currentState!`
- **Location** : `mobile/lib/features/auth/presentation/screens/login_screen.dart:34`, `register_screen.dart:41`, `agency_register_screen.dart:40`
- **Trigger** : si formulaire detached du tree au moment du tap (race), `currentState` est null → crash.
- **Impact** : edge crash rare.
- **Fix** : `if (_formKey.currentState?.validate() != true) return;`.

### BUG-27 : Search index publisher debounce 5min process-local
- **Location** : `backend/internal/app/searchindex/publisher.go:128-130`
- **Trigger** : `lastPublish map[debounceKey]time.Time` local au process. N instances backend → debounce N× moins efficace.
- **Impact** : pression légèrement accrue sur Typesense / OpenAI embeddings. Pas critique.
- **Fix** : déplacer dans Redis (`SETNX` avec TTL).

---

## LOW (8)

- **BUG-28** : `tx.Commit` ignoré dans `conversation_repository.go:43` (find existant) — log warn
- **BUG-29** : `defer tx.Rollback()` partout (~30 sites) perd l'erreur de Rollback — acceptable mais log idéal
- **BUG-30** : `idx_search_queries_search_id` UNIQUE peut bloquer inserts concurrents sur hot search_id — négligeable
- **BUG-31** : Webhook idempotency Redis TTL = 7 jours, replay 8e jour passe — combiner avec table Postgres (cf. BUG-10)
- **BUG-32** : Migration 074 backfill DO $$ block monolithique — pour les futures migrations bulk-copy splitter en chunks
- **BUG-33** : `_ = json.Marshal(c)` dans `pkg/cursor/Encode` — petit struct toujours marshalable mais convention zéro-swallow → retourner `(string, error)`
- **BUG-34** : Pas de SSRF protection sur les URLs profil (PhotoURL/VideoURL) — flagged en sécurité (SEC-23) mais aussi un futur bug si scraping ajouté
- **BUG-35** : Mobile `chat_screen.dart` crée un Dio standalone qui bypass l'auth interceptor (timeouts hardcodés 30s/120s) — bug latent

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

## Summary

| Severity | Count |
|---|---|
| CRITICAL | 5 |
| HIGH | 12 |
| MEDIUM | 10 |
| LOW | 8 |
| **Total** | **35** |
