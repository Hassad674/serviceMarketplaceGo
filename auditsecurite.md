# Audit de Sécurité

**Date** : 2026-04-29 (audit précédent : 2026-03-30, obsolète)
**Branche** : `main` @ `a0d268a4`
**Périmètre** : backend Go (~848 fichiers, 123 migrations), web Next.js, admin Vite, mobile Flutter

## Méthodologie

Audit exhaustif full-stack. Chaque finding du précédent audit a été re-vérifié avant report. Concentration sur les portes d'attaque concrètes : auth/sessions, autorisation/RBAC, validation d'entrée, secrets, headers HTTP, CORS, idempotence, RLS, audit log, RGPD, clients (cookies web, secure storage mobile, JWT admin).

---

## CRITICAL (9)

### SEC-01 : Stored XSS via JSON-LD sur les pages publiques de profil
- **Location** : `web/src/app/[locale]/(public)/freelancers/[id]/page.tsx:99`, `clients/[id]/page.tsx`, `referrers/[id]/page.tsx`
- **CWE** : CWE-79 / OWASP A03
- **Description** : `JSON.stringify(payload)` injecté dans `<script type="application/ld+json"...>` via `dangerouslySetInnerHTML`. `JSON.stringify` **n'échappe pas** `</`. Un freelance qui met `</script><script>fetch('/api/v1/...', {credentials:'include'})</script>` dans son champ `about` exécute du JS arbitraire dans l'origine du site sur tout visiteur de sa fiche.
- **Impact** : compromission de session des visiteurs authentifiés, exfiltration RGPD, ATO. Surface = TOUS les visiteurs des pages profil publiques.
- **Fix** : `JSON.stringify(payload).replace(/</g, '\\u003c').replace(/-->/g, '--\\u003e')`. Le commentaire « trusted server data » est faux : `profile.about` est saisi par l'utilisateur.

### SEC-02 : ConfirmPayment permet d'activer une proposal sans paiement Stripe réel
- **Location** : `backend/internal/handler/proposal_handler.go:360-401`, `internal/app/payment/service_stripe.go:137-150`
- **CWE** : CWE-840 / business logic flaw
- **Description** : `MarkPaymentSucceeded` lit le record local et appelle `record.MarkPaid()` **sans consulter Stripe**. Un client avec accès DevTools peut faire passer le record en `succeeded`, le proposal en `active`, et déclencher `RequestCompletion`.
- **Impact** : fraude directe — fonds escrow virtuels jamais encaissés mais transférés au prestataire.
- **Fix** : avant `MarkPaymentSucceeded`, `stripe.PaymentIntents.Get(record.StripePaymentIntentID)` et vérifier `pi.Status == "succeeded"`. Pas de fallback non-vérifié.

### SEC-03 : Middleware `SecurityHeaders` toujours absent
- **Location** : `backend/internal/handler/middleware/` (pas de `security_headers.go`)
- **CWE** : CWE-693 / OWASP A05
- **Description** : Aucun CSP, X-Content-Type-Options, X-Frame-Options, HSTS, Referrer-Policy, Permissions-Policy. Spec backend/CLAUDE.md détaillée mais jamais implémentée. L'audit de mars cochait "shipped" — c'est faux.
- **Impact** : XSS, clickjacking (iframe-able), MIME-sniffing, downgrade HTTP, fuite Referer.
- **Fix** : créer `middleware/security_headers.go` selon spec, l'insérer dans la stack après `Recovery`, avant `CORS`. Ajouter aussi `async headers()` côté `web/next.config.ts` (Vercel n'envoie rien sans).

### SEC-04 : `JWT_SECRET` fallback hardcodé `"dev-secret-change-me"` sans check au démarrage
- **Location** : `backend/internal/config/config.go:74`
- **CWE** : CWE-798 / OWASP A02
- **Description** : Si `JWT_SECRET` non défini en prod, le service boote silencieusement avec le secret hardcodé committé dans le repo public. Aucun check `IsProduction() && JWTSecret == "dev-secret-change-me"`, aucune longueur minimale.
- **Impact** : compromission totale d'un déploiement mal configuré. Repo open-source = fallback public.
- **Fix** : `log.Fatal` en production si `JWTSecret == "dev-secret-change-me"` ou `len < 32`. Idem pour `STORAGE_SECRET_KEY` (`minioadmin`).

### SEC-05 : Suspend/Ban ne bump pas `session_version` → 15 min de fenêtre mobile
- **Location** : `backend/internal/app/admin/service.go:222-265` (`SuspendUser`, `BanUser`)
- **CWE** : CWE-613 / OWASP A07
- **Description** : `invalidateAndNotify` supprime les sessions Redis mais n'appelle PAS `users.BumpSessionVersion`. Le check middleware fait `current == carriedVersion` → match → next. Un mobile JWT reste valide 15 min après ban.
- **Impact** : utilisateur banni (harcèlement, fraude) garde l'accès complet 15 min après l'action admin.
- **Fix** : ajouter `_, _ = s.users.BumpSessionVersion(ctx, userID)` au début de `invalidateAndNotify`. Faire de même dans `ResetPassword`.

### SEC-06 : Refresh token rotation absente — token vol indéfini
- **Location** : `backend/internal/app/auth/service.go:330-364`
- **CWE** : CWE-384 / OWASP A07
- **Description** : `RefreshToken` génère un nouveau pair access+refresh sans (a) blacklister l'ancien, (b) consulter une blacklist, (c) invalider le `jti`. Un refresh volé peut être rejoué en parallèle pendant 7 jours sans détection.
- **Impact** : vol de refresh = persistance d'attaque non détectable.
- **Fix** : Redis `auth:refresh_blacklist:{jti}` avec TTL = remaining lifetime à chaque refresh. Rejeter 401 si `jti` blacklisté + logger comme indicateur de vol. Faire de même dans `Logout`.

### SEC-07 : Brute force protection absente sur `/auth/login`
- **Location** : `backend/internal/handler/auth_handler.go:108-137`, `internal/app/auth/service.go:270-315`
- **CWE** : CWE-307 / OWASP A07
- **Description** : Aucun verrouillage Redis par email (`login_attempts:{email}` / `login_locked:{email}`). Le seul rate limit est IP-based, en mémoire (10 r/s, burst 20). Multiplie par rotation IP. Pas de cap par compte. `ActionLoginFailure` défini mais jamais appelé.
- **Impact** : credential stuffing en masse possible.
- **Fix** : `port/service.BruteForceService` + adapter Redis (sliding window 5/15min, lockout 30min, 429 + `Retry-After`). Idem `/forgot-password` et `/reset-password` (3/h par email).

### SEC-08 : Android `usesCleartextTraffic="true"` en production
- **Location** : `mobile/android/app/src/main/AndroidManifest.xml:8`
- **CWE** : CWE-319 / OWASP A02
- **Description** : HTTP en clair activé pour TOUS les builds. Avec `defaultValue: 'http://192.168.1.156:8083'`, un APK release oublié sans `--dart-define=API_URL=https://...` enverra les credentials en clair.
- **Impact** : MITM trivial sur Wi-Fi public si build prod sans override.
- **Fix** : retirer l'attribut, créer `network_security_config.xml` autorisant cleartext uniquement pour 10.0.2.2 / 192.168.x. Faire échouer le build prod si `API_URL` ne commence pas par `https://`.

### SEC-09 : Upload — Content-Type accepté tel quel sur tous les médias sauf portfolio
- **Location** : `backend/internal/handler/upload_handler.go:82-86, 144-148, 206-210, 265-269` (UploadPhoto, UploadVideo, UploadReferrerVideo, UploadReviewVideo)
- **CWE** : CWE-434 / OWASP A05
- **Description** : Validation magic-bytes (SOI/EOI JPEG, signature/IEND PNG) implémentée seulement dans `UploadPortfolioImage:411-446`. Les autres uploads se contentent de `strings.HasPrefix(contentType, "image/")` qui accepte `image/svg+xml` (XSS) et `video/anything` renommé.
- **Impact secondaire** : `filepath.Ext(header.Filename)` est utilisé tel quel pour la clé S3 → un attaquant uploade `.html`/`.exe`/`.svg` au domaine du bucket public, exécution JS dans l'origine du bucket.
- **Fix** : centraliser `detectMimeFromBytes` (`http.DetectContentType` sur les 512 premiers octets), allowlist explicite (`image/jpeg`, `png`, `webp` ; `video/mp4`, `webm`, `quicktime`), forcer l'extension dérivée du content-type détecté, randomiser entièrement le filename.

---

## HIGH (15)

### SEC-10 : RLS PostgreSQL totalement absente
- **Location** : `backend/migrations/*.sql` — 0 occurrence de `ENABLE ROW LEVEL SECURITY` sur 63 tables
- **CWE** : CWE-639 / OWASP A01
- **Description** : Backend/CLAUDE.md détaille un modèle RLS avec `SET LOCAL app.current_user_id`. Aucune table activée. Aucun `SET LOCAL` dans les repos. Sécurité reposant entièrement sur le filtre `WHERE org_id = $1` applicatif.
- **Impact** : un seul oubli de filtre lors d'un refacto = fuite cross-tenant complète sur messages/invoices/proposals/disputes.
- **Fix** : activer RLS au minimum sur `messages, conversations, invoices, proposals, notifications, wallet_records, disputes, audit_logs`. `FORCE ROW LEVEL SECURITY`. Test d'intégration vérifiant le block cross-tenant.

### SEC-11 : Rate limiter applicatif en mémoire — bypass multi-pod et derrière LB
- **Location** : `backend/internal/handler/middleware/ratelimit.go:16-83`
- **CWE** : CWE-770 / OWASP A04
- **Description** : `map[string]*visitor` local au process. Sur N pods Railway, quota = N × 10 r/s. Clé = `r.RemoteAddr` (host:port) sans parser X-Forwarded-For : tous les utilisateurs partagent l'IP du LB. Pas de différenciation auth/mutation/upload.
- **Impact** : bypass trivial en multi-pod ; brute force sur /login non effectivement limité.
- **Fix** : Redis sliding window (le pattern existe déjà dans `MessagingRateLimiter`). 4 classes : global IP / auth-email / mutation-user / upload-user. Headers `X-RateLimit-*` + `Retry-After`. Parsing X-Forwarded-For côté allowlist proxy.

### SEC-12 : Idempotency middleware absent côté API
- **Location** : pas de `backend/internal/handler/middleware/idempotency.go`. Les `IdempotencyKey` du code = uniquement Stripe SDK side.
- **CWE** : CWE-837
- **Description** : Spec CLAUDE.md détaillée mais inexistante. Un client mobile qui retry sur timeout réseau peut créer 2 proposals, 2 disputes, 2 reviews.
- **Impact** : doubles missions, double notif, double commission.
- **Fix** : middleware `Idempotency-Key` Redis 24h TTL avec capture status+body. Appliquer sur `POST /proposals`, `/disputes`, `/reviews`, `/jobs`, `/reports`.

### SEC-13 : Audit logs définis mais jamais émis pour les événements auth/admin
- **Location** : `backend/internal/app/auth/service.go` complet, `internal/app/admin/service.go`. Migration `078_create_audit_logs` n'a aucun `REVOKE UPDATE, DELETE`.
- **CWE** : CWE-778 / OWASP A09
- **Description** : Constantes `ActionLoginSuccess/Failure/Logout/PasswordReset*/Suspend/Ban/AuthorizationDenied` définies, jamais appelées. Seuls `role_overrides_service.go` et `admin/message_moderation.go` loggent. Pas de check append-only en DB.
- **Impact** : aucune traçabilité forensique de compromission de compte ; non-conforme RGPD art. 30.
- **Fix** : plumber les events dans le service auth + admin/service.go (suspend/ban/unban). Migration `REVOKE UPDATE, DELETE ON audit_logs FROM <app_user>`. Remplacer `ON DELETE SET NULL` par `actor_email_hash` dans metadata pour préserver l'attribution post-RGPD.

### SEC-14 : Web middleware accepte `?token=` query string sur toutes les routes protégées
- **Location** : `web/src/middleware.ts:70-78`
- **CWE** : CWE-598
- **Description** : `token = cookie.session_id || searchParams.get("token")` — ajouté pour le bridge mobile WebView. Un lien externe cliqué fuite le JWT en Referer header. Apparaît aussi dans logs serveur, historique navigateur, screenshots.
- **Impact** : vol de JWT mobile via referrer, log analyzers, partage d'URL. Token exploitable 15 min.
- **Fix** : restreindre aux routes Stripe Embedded uniquement (`/payment-info`, `/subscribe/*`). Forcer `Referrer-Policy: no-referrer` sur ces pages. Côté mobile : POST immédiat à `/auth/web-session` pour échanger contre cookie httpOnly + `window.history.replaceState`.

### SEC-15 : WebSocket auth `?token=` JWT en query (mobile)
- **Location** : `backend/internal/adapter/ws/connection.go:88-95`
- **CWE** : CWE-598
- **Description** : Stratégie 3 du WS auth fait passer le JWT en query string pour mobile. Les LB et reverse proxies (Railway, Cloudflare) loggent les URLs.
- **Impact** : fuite JWT via logs infrastructure.
- **Fix** : étendre `CreateWSToken` au mode bearer mobile (single-use ticket via `/auth/ws-token` puis WS).

### SEC-16 : `ResetPassword` ne révoque pas les sessions existantes
- **Location** : `backend/internal/app/auth/service.go:422-456`
- **CWE** : CWE-613
- **Description** : Reset change le hash mais ne supprime pas les sessions Redis ni ne bumpe `session_version`. Un attaquant ayant un access token + déclenchant un reset garde l'accès.
- **Impact** : reset password n'est pas le « kill switch » attendu.
- **Fix** : après `MarkUsed`, `s.users.BumpSessionVersion(ctx, u.ID)` + `s.sessionSvc.DeleteByUserID(ctx, u.ID)`.

### SEC-17 : Webhook Stripe — table Postgres `stripe_webhook_events` créée mais jamais utilisée
- **Location** : `migrations/089_create_stripe_webhook_events.up.sql`, `internal/handler/stripe_handler.go:136-147`, `internal/adapter/redis/webhook_idempotency.go:50-55`
- **CWE** : CWE-345
- **Description** : Idempotency uniquement Redis. Si Redis tombe, `TryClaim` retourne `(true, err)` (claim conservatoire) → webhook traité. Si Stripe retry pendant la panne, double-traitement possible : double `subscription.created`, double émission facture FAC-NNNNNN, double-fund de jalons.
- **Impact** : pendant une panne Redis, replay côté Stripe = double-écriture business.
- **Fix** : utiliser `stripe_webhook_events` comme source de vérité (INSERT UNIQUE event_id), Redis comme fast-path uniquement.

### SEC-18 : Race condition création compte Stripe Connect
- **Location** : `backend/internal/handler/embedded_handler.go:235-267` (`resolveStripeAccount`)
- **CWE** : CWE-362 (TOCTOU)
- **Description** : Deux requêtes `CreateAccountSession` concurrentes du même org → deux `GetStripeAccount` vides → deux `createStripeCustomAccount` → 2 comptes Stripe créés, l'un orphelin mais référencé Stripe-side.
- **Impact** : comptes Stripe orphelins, KYC potentiellement soumis sur le compte « perdant ».
- **Fix** : `pg_advisory_xact_lock(hashtext(org_id))` ou transaction `SELECT ... FOR UPDATE` sur la ligne org avant check + create.

### SEC-19 : Validation DTO trop minimaliste — pas de longueur, format, plage
- **Location** : `backend/internal/handler/dto/request/*.go` (aucun tag `validate`, aucune méthode `Validate()`)
- **CWE** : CWE-20
- **Description** : Validation = `validator.ValidateRequired` (champ vide). Pas de cap longueur (DoS gros JSON), pas de regex UUID hors `uuid.Parse`, pas de plage sur `MinBudget`/`MaxBudget` (Stripe overflow possible).
- **Impact** : DoS, business abuse, payloads XSS non rejetés en amont.
- **Fix** : `go-playground/validator` avec tags `required,min=1,max=5000` ; valider UUID, plages, longueurs strings dans chaque DTO.

### SEC-20 : Mot de passe — règle « caractère spécial » de CLAUDE.md non appliquée
- **Location** : `backend/internal/domain/user/valueobject.go` (`NewPassword`)
- **CWE** : CWE-521
- **Description** : Exige uppercase + lowercase + digit, mais CLAUDE.md exige aussi un caractère spécial. `Password1` passe.
- **Impact** : pwd plus faibles que la politique annoncée.
- **Fix** : ajouter check `unicode.IsPunct` ou allowlist. Pousser longueur min à 10.

### SEC-21 : Filename utilisateur dans la clé S3 → path/extension control
- **Location** : `backend/internal/handler/upload_handler.go:88-89, 150-151, 212-213, 271-272, 389-390`
- **CWE** : CWE-22 (path traversal-adjacent) + CWE-434
- **Description** : `ext := filepath.Ext(header.Filename)` puis `fmt.Sprintf("profiles/%s/photo_%s%s", orgID, uuid, ext)`. `.html`/`.exe`/`.svg` passent → fichier HTML rendu inline = exécution JS.
- **Impact** : XSS dans l'origine du bucket si servi inline.
- **Fix** : ne jamais utiliser l'extension client. Forcer extension dérivée du Content-Type validé magic-byte.

### SEC-22 : `RequireRole` middleware annoncé mais jamais implémenté
- **Location** : `backend/internal/handler/middleware/admin.go` (RequireAdmin uniquement)
- **CWE** : CWE-285
- **Description** : Spec CLAUDE.md cite `middleware.RequireRole("agency", "provider")`. N'existe pas. Distinction de rôle se fait au niveau service. OK comme défense en profondeur mais pas de garde routeur sur les endpoints role-spécifiques.
- **Impact** : régression possible si un endpoint provider est touché par enterprise.
- **Fix** : ajouter `RequireRole(roles ...string)` ; appliquer au moins sur `POST /jobs` et flux KYC/payout.

### SEC-23 : URLs user-controlled — pas de SSRF protection
- **Location** : `backend/internal/handler/dto/request/job.go` (VideoURL), `internal/domain/profile/social_link.go:88-103`
- **CWE** : CWE-918
- **Description** : `ValidateSocialURL` rejette `javascript:`/`data:` mais accepte `http://10.0.0.1`, `http://localhost`, `http://169.254.169.254` (AWS metadata). Pas fetché aujourd'hui mais futur scraping (OG image) sera vulnérable.
- **Impact** : préparation au SSRF dès qu'un workflow scraping sera ajouté.
- **Fix** : DNS resolution check, rejeter RFC1918 / link-local / loopback. Documenter dans le domain.

### SEC-24 : CORS — `Allow-Credentials: true` même hors allowlist + pas de `Vary: Origin`
- **Location** : `backend/internal/handler/middleware/cors.go:14-35`
- **CWE** : CWE-942 / OWASP A05
- **Description** : `Access-Control-Allow-Credentials: true` inconditionnel. Pas de `Vary: Origin`. Caches partagés peuvent servir une réponse autorisée à un cross-origin malicieux.
- **Impact** : cache poisoning + signal incohérent qui pourrait faciliter un futur bug.
- **Fix** : `w.Header().Add("Vary", "Origin")` toujours ; `Allow-Credentials` uniquement si origin allowlistée.

---

## MEDIUM (10)

### SEC-25 : `RequestID` middleware accepte `X-Request-ID` arbitraire — log injection
- **Location** : `backend/internal/handler/middleware/requestid.go:30-37`
- **CWE** : CWE-117
- **Fix** : valider format UUID v4 strict, sinon régénérer.

### SEC-26 : Permissions middleware fallback statique sur sessions legacy
- **Location** : `backend/internal/handler/middleware/permission.go:46-54`
- **CWE** : CWE-285
- **Description** : Si la session est ancienne (pré-R17, sans permissions), fallback `organization.HasPermission` qui ignore les overrides per-org.
- **Fix** : forcer logout des sessions sans `Permissions`, ou fallback DB.

### SEC-27 : Stripe Connect `account_id` exposé dans `GET /account-status`
- **Location** : `backend/internal/handler/embedded_handler.go:68, 206-208`
- **CWE** : CWE-200
- **Fix** : ne renvoyer que statut (charges_enabled, payouts_enabled, requirements_count). L'ID Stripe est utile uniquement côté serveur.

### SEC-28 : `X-Forwarded-For` accepté sans CIDR allowlist
- **Location** : `backend/internal/handler/role_overrides_handler.go:187-200`
- **CWE** : CWE-348
- **Fix** : `TRUSTED_PROXIES` config CIDR, n'utiliser XFF que si `RemoteAddr ∈ TRUSTED_PROXIES`.

### SEC-29 : GDPR — endpoints `/me/export` et `/me/account` (DELETE) absents
- **Location** : backend complet
- **CWE** : GDPR Art. 15-17
- **Description** : Cascades posées (91 `ON DELETE CASCADE`) mais aucun endpoint utilisateur. Pas de purge Typesense documentée sur user delete.
- **Fix** : endpoints `GET /api/v1/me/export` (JSON) et `DELETE /api/v1/me/account` (soft + scheduled hard + Typesense + R2 purge).

### SEC-30 : Admin token stocké en `localStorage`
- **Location** : `admin/src/lib/api-client.ts:24, 38-42`
- **CWE** : CWE-922
- **Description** : XSS dans une dep admin = siphonage tokens.
- **Fix** : passer en httpOnly cookie + CSRF token séparé.

### SEC-31 : Audit logs `ON DELETE SET NULL` perd l'attribution sur user delete
- **Location** : `backend/migrations/078_create_audit_logs.up.sql:30`
- **CWE** : CWE-778
- **Fix** : capturer `actor_email_hash` (sha256) dans `metadata` à l'écriture du log. UUID anonymisée n'est plus PII RGPD.

### SEC-32 : `Authorization` header — pas de redaction structurée dans slog
- **Location** : `backend/internal/handler/middleware/logger.go:43-54`
- **CWE** : CWE-532
- **Fix** : `slog.HandlerOptions.ReplaceAttr` qui appelle `Redact` sur tous les attributs.

### SEC-33 : Webhook idempotency Redis TTL = 7 jours
- **Location** : `backend/internal/adapter/redis/webhook_idempotency.go:13`
- **Description** : Stripe peut techniquement replay au-delà de 7 jours sur support manuel.
- **Fix** : combiner avec table Postgres (cf. SEC-17).

### SEC-34 : Cookie `user_role` non-httpOnly
- **Location** : `backend/internal/handler/cookie.go:34-44`
- **Description** : Voulu pour rendu UI, OK si l'invariant « pas une décision serveur » est respecté.
- **Fix** : commentaire de garde côté web qui consomme ce cookie.

---

## LOW (6)

- **SEC-35** : `JWT_SECRET` hardcodé dans tests (`pkg/crypto/jwt_test.go`) → générer via `crypto/rand` dans `TestMain`
- **SEC-36** : `Access-Control-Max-Age: 86400` (24h) → réduire à 600
- **SEC-37** : Pas de pipeline CI gosec/semgrep — déjà govulncheck + trivy hebdo, ajouter gosec sur PR
- **SEC-38** : `embedded_handler.go:97` — `_ = json.Unmarshal` ignore l'erreur sur body malformé
- **SEC-39** : Health check Redis dans `/ready` — vérifier qu'il existe vraiment
- **SEC-40** : 15 sites `_ = err` à logger en `slog.Warn` minimum (cf. audit qualité)

---

## Already shipped (vérifié pendant l'audit)

- ✅ Stripe webhook signature verification (`adapter/stripe/webhook.go:16`)
- ✅ Mobile secure storage (`flutter_secure_storage` + Keychain/EncryptedSharedPreferences)
- ✅ Cookie httpOnly + SameSite=Lax + Secure-en-prod
- ✅ JSON `DisallowUnknownFields` (`pkg/validator`)
- ✅ WebSocket short-lived single-use `ws_token` (Redis `GetDel`)
- ✅ Session_version revocation infrastructure (middleware/auth + propagation rôle org)
- ✅ Recovery middleware avec request_id correlation
- ✅ Redact helpers logs (regex bearer/sk-/emails)
- ✅ Magic-bytes validation (`UploadPortfolioImage` uniquement — cf. SEC-09)
- ✅ CORS allowlist explicite (pas de wildcard)
- ✅ Bcrypt cost 12
- ✅ MaxBytesReader sur uploads (5/50/100 MB)
- ✅ Audit log domain + repo + table (mais call sites incomplets — cf. SEC-13)
- ✅ Forgot password ne révèle pas l'existence de l'email
- ✅ Stripe IdempotencyKey sur Transfers/Payouts/PaymentIntents
- ✅ Web `poweredByHeader: false`, pas de localStorage tokens (proxy session cookie)
- ✅ `.env*` gitignorés
- ✅ Pas de SQL string-concat avec input utilisateur côté path utilisateur (sauf SEC-50 dans audit qualité, admin only)
- ✅ Optimistic concurrency milestones (`version` column)
- ✅ Pending events outbox `FOR UPDATE SKIP LOCKED`

---

## Strong points

- Architecture hexagonale strictement respectée — surface d'attaque limitée par DI
- Système `session_version` + middleware Auth à 4 étages bien commenté
- Permissions org-scoped avec override per-org : bonne base RBAC
- Handlers admin systématiquement gardés `RequireAdmin` + `NoCache`
- Stripe Connect rigoureux : IdempotencyKey systématique, signature stricte, account session délégué
- Modération avec fail-closed sur OpenAI 5xx (`moderateDisplayName` refuse l'inscription)
- Pas de SQL string-concat utilisateur sur les paths publics
- Pas de fuite secrets dans les logs (regex Redact en place)

---

## Top 10 fixes ordonnés par ROI

| # | ID | Effort | Impact |
|---|---|---|---|
| 1 | SEC-01 (XSS JSON-LD) | 30 min | CRITICAL — protège tous les visiteurs profil |
| 2 | SEC-02 (ConfirmPayment vérif Stripe) | 1 h | CRITICAL — ferme la fraude directe |
| 3 | SEC-03 (SecurityHeaders middleware) | 1 h | clickjacking + MIME + HSTS d'un coup |
| 4 | SEC-04 (JWT_SECRET fail-fast) | 30 min | prévient compromission catastrophique |
| 5 | SEC-05 + SEC-16 (session_version sur ban/reset) | 30 min | ferme la fenêtre 15 min mobile |
| 6 | SEC-08 (Android cleartext) | 1 h | MITM mobile fix |
| 7 | SEC-09 + SEC-21 (magic bytes + filename) | 3 h | XSS via media |
| 8 | SEC-06 (refresh rotation) | 3 h | ferme le replay 7 jours |
| 9 | SEC-07 (brute force Redis) | 4 h | credential stuffing block |
| 10 | SEC-11 (rate limiter Redis multi-pod) | 4 h | DoS protection réelle |

**Bundle « pré-open-source » (~ 2 jours)** : 1+2+3+4+5+6 ferment 6 vulnérabilités exploitables. Le reste peut suivre en sprint dédié.

---

## Summary

| Severity | Count |
|---|---|
| CRITICAL | 9 |
| HIGH | 15 |
| MEDIUM | 10 |
| LOW | 6 |
| **Total** | **40** |
