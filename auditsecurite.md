# Audit de Sécurité

**Date** : 2026-04-30 (mise à jour post Phase 1 → 5Q ; audit précédent : 2026-04-29)
**Branche** : `main` @ `c8284526`
**Périmètre** : backend Go (~860 fichiers, 125 migrations), web Next.js, admin Vite, mobile Flutter

## Méthodologie

Audit exhaustif full-stack. Chaque finding du précédent audit a été re-vérifié avant report. Concentration sur les portes d'attaque concrètes : auth/sessions, autorisation/RBAC, validation d'entrée, secrets, headers HTTP, CORS, idempotence, RLS, audit log, RGPD, clients (cookies web, secure storage mobile, JWT admin).

---

## CRITICAL (0)

Tous les findings CRITICAL du round précédent ont été fermés :

- ~~SEC-01 (Stored XSS JSON-LD)~~ closed in PR #33 (`30ca2e56 fix(web/security): escape JSON-LD payloads`)
- ~~SEC-02 (ConfirmPayment fraude Stripe)~~ closed in PR #33 (`90f4556b fix(payment/security): verify Stripe before marking payment succeeded`)
- ~~SEC-03 (SecurityHeaders middleware)~~ closed in PR #32 (`6fd4178b feat(security): SecurityHeaders middleware + Next.js CSP`)
- ~~SEC-04 (JWT_SECRET fail-fast)~~ closed in PR #32 (`2ecf8153 feat(security): fail-fast on default secrets in production`)
- ~~SEC-05 (Suspend/Ban session_version)~~ closed in PR #32 (`98b0e51c feat(security): session invalidation + auth audit emission`)
- ~~SEC-06 (Refresh token rotation)~~ closed in PR #31 (`9d57096a feat(auth): rotate refresh tokens single-use through Redis blacklist`)
- ~~SEC-07 (Brute force /auth/login)~~ closed in PR #31 (`9d9c86b6 feat(auth): brute-force protection on login + password reset`)
- ~~SEC-08 (Android cleartext)~~ closed in PR #32 (`b19a2d28 fix(security/mobile): block cleartext HTTP outside dev LAN`)
- ~~SEC-09 (Upload magic bytes generalisation)~~ closed in PR #33 (`f878d70d fix(upload/security): magic-byte validation + randomized storage keys`)

---

## HIGH (3)

Findings HIGH fermés ce round (12 sur 15) :

- ~~SEC-10 (RLS PostgreSQL absente)~~ closed in PR #39 (`2aa285ae feat(security): enable RLS on 9 tenant-scoped tables`) — voir BUG-NEW-04 pour la dette résiduelle (production DB role)
- ~~SEC-11 (Rate limiter mémoire)~~ closed in PR #31 (`b2e683fc feat(http): Redis sliding-window rate limiter with 4 classes`)
- ~~SEC-13 (Audit logs auth/admin émis)~~ closed in PR #32 (`98b0e51c feat(security): session invalidation + auth audit emission` + `2d697468 feat(security): audit_logs grants migration`) — voir BUG-NEW-09 pour l'attribution actor manquante
- ~~SEC-14 (web middleware ?token= partout)~~ closed in PR #31 (`40ccc13a feat(web): restrict ?token= to mobile WebView bridge routes`)
- ~~SEC-15 (WS JWT en query)~~ closed in PR #31 (`abef14e7 feat(ws): single-use ws_token for mobile, drop legacy JWT-in-URL`)
- ~~SEC-16 (ResetPassword sessions)~~ closed in PR #32 (idem `98b0e51c`)
- ~~SEC-17 (Webhook stripe_webhook_events durable)~~ closed in PR #36 (`512eaa56 fix(webhook): durable idempotency via Postgres source-of-truth + Redis fast-path`) — voir BUG-NEW-08 pour le cas du handler qui throw après claim
- ~~SEC-18 (Race Stripe Connect create)~~ closed in PR #33 (`18767345 fix(embedded): serialise Stripe Connect account creation per org`)
- ~~SEC-19 (Validation DTO)~~ closed in PR #31 (`afaccf9e feat(validator/security): add struct-tag validation across all DTOs`)
- ~~SEC-20 (Password caractère spécial + 10)~~ closed in PR #31 (`6e75220e feat(auth): require special character + 10-char minimum on passwords`)
- ~~SEC-21 (Filename → S3 key)~~ closed in PR #33 (idem SEC-09)
- ~~SEC-24 (CORS Vary + Allow-Credentials conditional)~~ closed in PR #32 (`8b7c312d fix(security): CORS Vary header always + Allow-* conditional`)

### SEC-12 : Idempotency middleware applicatif absent côté API (still open)
- **Location** : pas de `backend/internal/handler/middleware/idempotency.go`. Les `IdempotencyKey` du code = uniquement Stripe SDK side.
- **CWE** : CWE-837
- **Description** : Spec CLAUDE.md détaillée mais inexistante. Un client mobile qui retry sur timeout réseau peut créer 2 proposals, 2 disputes, 2 reviews.
- **Impact** : doubles missions, double notif, double commission.
- **Fix** : middleware `Idempotency-Key` Redis 24h TTL avec capture status+body. Appliquer sur `POST /proposals`, `/disputes`, `/reviews`, `/jobs`, `/reports`.

### SEC-22 : `RequireRole` middleware annoncé mais jamais implémenté (still open)
- **Location** : `backend/internal/handler/middleware/admin.go` (RequireAdmin uniquement)
- **CWE** : CWE-285
- **Description** : Spec CLAUDE.md cite `middleware.RequireRole("agency", "provider")`. N'existe pas. Distinction de rôle se fait au niveau service. OK comme défense en profondeur mais pas de garde routeur sur les endpoints role-spécifiques.
- **Impact** : régression possible si un endpoint provider est touché par enterprise.
- **Fix** : ajouter `RequireRole(roles ...string)` ; appliquer au moins sur `POST /jobs` et flux KYC/payout.

### SEC-23 : URLs user-controlled — pas de SSRF protection (still open)
- **Location** : `backend/internal/handler/dto/request/job.go` (VideoURL), `internal/domain/profile/social_link.go:88-103`
- **CWE** : CWE-918
- **Description** : `ValidateSocialURL` rejette `javascript:`/`data:` mais accepte `http://10.0.0.1`, `http://localhost`, `http://169.254.169.254` (AWS metadata). Pas fetché aujourd'hui mais futur scraping (OG image) sera vulnérable.
- **Impact** : préparation au SSRF dès qu'un workflow scraping sera ajouté.
- **Fix** : DNS resolution check, rejeter RFC1918 / link-local / loopback. Documenter dans le domain.

---

## MEDIUM (9)

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

### SEC-28 : `X-Forwarded-For` accepté sans CIDR allowlist (role_overrides_handler)
- **Location** : `backend/internal/handler/role_overrides_handler.go:187-200`
- **CWE** : CWE-348
- **Description** : ratelimit middleware applique le CIDR allowlist (Phase 1 SEC-11) mais `clientIP()` dans `role_overrides_handler` reste naïf.
- **Fix** : extraire le helper du middleware ratelimit + l'utiliser ici.

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

### ~~SEC-33 : Webhook idempotency Redis TTL = 7 jours~~ closed in PR #36 (composite Postgres + Redis)

### SEC-34 : Cookie `user_role` non-httpOnly
- **Location** : `backend/internal/handler/cookie.go:34-44`
- **Description** : Voulu pour rendu UI, OK si l'invariant « pas une décision serveur » est respecté.
- **Fix** : commentaire de garde côté web qui consomme ce cookie.

---

## LOW (3)

- **SEC-35** : `JWT_SECRET` hardcodé dans tests (`pkg/crypto/jwt_test.go`) → générer via `crypto/rand` dans `TestMain`
- ~~SEC-36 (CORS Max-Age 86400 → 600)~~ closed in Phase 0 (`307a233f chore(cors): reduce Access-Control-Max-Age from 86400 to 600`)
- **SEC-37** : Pas de pipeline CI gosec/semgrep — déjà govulncheck + trivy hebdo, ajouter gosec sur PR (gosec lui-même tournera mais pas en CI auto)
- ~~SEC-38 (embedded_handler.go _ = json.Unmarshal)~~ closed in PR #40 (`0d8a266d fix(embedded): surface 400 invalid_json on malformed account-session body`)
- **SEC-39** : Health check Redis dans `/ready` — vérifier qu'il existe vraiment
- ~~SEC-40 (15 sites `_ = err` à logger)~~ closed in Phase 0 (`b7f018ae chore(backend): log non-fatal swallowed errors with slog.Warn`)

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

## Top fixes restants ordonnés par ROI

| # | ID | Effort | Impact |
|---|---|---|---|
| 1 | SEC-12 (idempotency middleware applicatif) | 1 j | doubles missions/disputes/reviews |
| 2 | SEC-22 (RequireRole middleware) | 4 h | défense en profondeur RBAC |
| 3 | SEC-23 (SSRF protection URLs) | 2 h | préparation au future scraping |
| 4 | SEC-29 (GDPR /me/export + DELETE) | 2 j | conformité RGPD Art. 15-17 |
| 5 | SEC-31 (actor_email_hash) | 1 h | preserve attribution post-RGPD |
| 6 | SEC-30 (admin token httpOnly) | 4 h | XSS admin = no token exfil |
| 7 | SEC-32 (slog ReplaceAttr Redact) | 2 h | structured redaction |
| 8 | SEC-25 (RequestID UUID strict) | 30 min | log injection prevented |
| 9 | SEC-26 (legacy session permissions fallback) | 2 h | RBAC consistency |
| 10 | SEC-28 (XFF allowlist role_overrides) | 1 h | spoofing prevention |

---

## Closed in this round

12 HIGH + 9 CRITICAL + 1 MEDIUM + 2 LOW = **24 findings closed** through PRs #31-#41.

| ID | Closed in | Note |
|---|---|---|
| SEC-01 | PR #33 | XSS JSON-LD escape `</script>` |
| SEC-02 | PR #33 | ConfirmPayment verify Stripe |
| SEC-03 | PR #32 | SecurityHeaders middleware shipped |
| SEC-04 | PR #32 | JWT_SECRET fail-fast in prod |
| SEC-05 | PR #32 | session_version bumped on suspend/ban |
| SEC-06 | PR #31 | Refresh token rotation + Redis blacklist |
| SEC-07 | PR #31 | BruteForce Redis sliding window |
| SEC-08 | PR #32 | Android cleartext blocked outside dev LAN |
| SEC-09 | PR #33 | Magic bytes generalised + filename randomized |
| SEC-10 | PR #39 | RLS PostgreSQL on 9 tenant tables |
| SEC-11 | PR #31 | Rate limiter Redis sliding window 4 classes |
| SEC-13 | PR #32 | Audit logs auth+admin emission + REVOKE grants |
| SEC-14 | PR #31 | Web `?token=` restricted to bridge routes |
| SEC-15 | PR #31 | WS single-use ws_token (no JWT in URL) |
| SEC-16 | PR #32 | ResetPassword bumps session_version |
| SEC-17 | PR #36 | Webhook idempotency durable Postgres + Redis |
| SEC-18 | PR #33 | Stripe Connect race advisory_lock |
| SEC-19 | PR #31 | DTO validation tags everywhere |
| SEC-20 | PR #31 | Password special char + 10-min length |
| SEC-21 | PR #33 | Storage filename randomized (linked to SEC-09) |
| SEC-24 | PR #32 | CORS Vary + conditional Allow-* |
| SEC-33 | PR #36 | webhook idempotency 7d (linked SEC-17) |
| SEC-36 | Phase 0 | Access-Control-Max-Age 86400 → 600 |
| SEC-38 | PR #40 | embedded_handler invalid_json surfaced |
| SEC-40 | Phase 0 | 15 sites `_ = err` logged |

## Summary

| Severity | Count |
|---|---|
| CRITICAL | 0 |
| HIGH | 3 |
| MEDIUM | 9 |
| LOW | 3 |
| **Total** | **15** |

(was 40 before this round → 15 remaining + 24 closed + 1 promoted to BUG-NEW)
