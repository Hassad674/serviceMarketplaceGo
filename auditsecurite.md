# Audit de Sécurité — Final Deep

**Date** : 2026-05-01 (final audit before public showcase)
**Branche** : `chore/final-audit-deep`
**Périmètre** : backend Go (~622 .go fichiers prod, 131 migrations), web Next.js, admin Vite, mobile Flutter
**Méthodologie** : OWASP Top 10 (2021) sweep + auth/sessions/RBAC drill-down + RLS migration audit + supply chain check. Chaque finding cite file:line précis et propose un fix concret. Cross-référence avec PRs #31-#66 fusionnés et `MEMORY.md` pour ne pas re-flagger les items déférés.

---

## Snapshot — état actuel après PRs #31-#66

| Severity | Count |
|---|---|
| CRITICAL | 1 |
| HIGH | 6 |
| MEDIUM | 9 |
| LOW | 4 |
| **Total** | **20** |

**Closed since previous round (24 items)** : XSS JSON-LD, SecurityHeaders middleware, JWT_SECRET fail-fast, session_version revocation, refresh token rotation, brute force, Android cleartext, magic-byte uploads, RLS PostgreSQL m.125, rate limiter Redis sliding window, audit logs auth/admin emission, web ?token= bridge-only, WS single-use ws_token, ResetPassword session bumping, webhook idempotency durable Postgres+Redis, Stripe Connect race advisory_lock, DTO validation tags, password special-char, CORS Vary + conditional Allow-*, embedded invalid_json, audit attribution admin (BUG-NEW-09), audit_logs RLS WITH CHECK m.129, Stripe webhook handler error → 503 retry (BUG-NEW-06).

---

## CRITICAL (1)

### SEC-FINAL-01 : 35 legacy `.GetByID()` callers in app layer break under prod RLS rotation
- **Severity**: 🔴 CRITICAL — deployment time-bomb
- **CWE** : CWE-285 (improper authorization), CWE-863 (incorrect authorization)
- **Location** : 35 sites identified by `grep -rn ".GetByID(" backend/internal/app/ --include="*.go" | grep -v "GetByIDForOrg|GetByIDWithVersion"`. Examples:
  - `backend/internal/app/proposal/service_actions.go:20, 72, 101, 165, 260, 296`
  - `backend/internal/app/dispute/service_actions.go:119, 269, 353, 437, 478, 528, 554, 600, 686, 767, 792, 838, 849`
  - `backend/internal/app/proposal/service_scheduler.go:109, 153, 305, 325`
  - `backend/internal/app/review/service.go:86, 280`
  - `backend/internal/app/referral/wiring_adapters.go:129`
- **Why it matters** : the 8-path PR series (BUG-NEW-04 paths 1-8) migrated each REPO method to wrap reads in `RunInTxWithTenant`, but kept the legacy `GetByID(ctx, id)` signature in place for "system-actor scheduler paths". 35 APP callers still use that legacy signature. Today they work because the migration owner role bypasses RLS. The moment production rotates to a dedicated `marketplace_app NOSUPERUSER NOBYPASSRLS` role (per `backend/docs/rls.md`), every one of these calls returns `ErrProposalNotFound` / `ErrDisputeNotFound` because the policy `USING` evaluates to NULL/false on rows the caller's `app.current_org_id` doesn't match.
- **Impact** : at the moment the prod role rotation happens, ALL proposal/dispute/review actions silently fail with NotFound. Every checkout returns 404. Every dispute action 404s. Total app outage masquerading as a routing bug.
- **How to fix** : 
  1. For every legacy caller, extract `orgID := mustGetOrgID(ctx)` from middleware context.
  2. Migrate to `proposals.GetByIDForOrg(ctx, id, orgID)` (already exists on proposal_repository).
  3. Add the `GetByIDForOrg` method on dispute_repository, review_repository, milestone_repository (only proposal has it today).
  4. Keep the old `GetByID` signature only for explicit system-actor call sites (`proposal/service_scheduler.go:AutoApproveMilestone`, `AutoCloseProposal`) where there is no caller org — and gate those behind a privileged DB connection pool.
- **Test required** : `rls_caller_audit_test.go` (integration) — create `marketplace_test_app` role with `NOBYPASSRLS`, run every public service action through the role, asserter all return correctly. The test should fail today on 35 sites.
- **Effort** : L (3 jours)

---

## HIGH (6)

### SEC-FINAL-02 (was SEC-12) : Idempotency middleware applicatif absent côté API
- **Severity**: 🟠 HIGH
- **CWE** : CWE-837 (improper enforcement of behavioral workflow)
- **Location** : pas de `backend/internal/handler/middleware/idempotency.go`. Les `IdempotencyKey` du code = uniquement Stripe SDK side.
- **Why it matters** : un client mobile qui retry sur timeout réseau peut créer 2 proposals, 2 disputes, 2 reviews. Stripe transferts protégés mais pas les business actions.
- **How to fix** : 
  1. Middleware `Idempotency-Key` Redis 24h TTL. Capture `(method, path, key) -> (status, body)`.
  2. Si la même clé arrive avant 24h, retourner la réponse cachée (bypass handler).
  3. Si la même clé arrive avec un body différent → `409 Conflict idempotency_key_collision`.
  4. Appliquer sur `POST /proposals`, `POST /disputes`, `POST /reviews`, `POST /jobs`, `POST /reports`, `POST /referral-actions`.
- **Test required** : `idempotency_test.go` — POST 2× avec même `Idempotency-Key` retourne le même body, exactly 1 row inséré. Test 409 sur body différent.
- **Effort** : M (½j)

### SEC-FINAL-03 (was SEC-22) : `RequireRole` middleware annoncé mais jamais implémenté
- **Severity**: 🟠 HIGH
- **CWE** : CWE-285
- **Location** : `backend/internal/handler/middleware/admin.go` (RequireAdmin uniquement). Le commentaire dans `proposal_admin_handler.go:21` dit "gated by RequireRole(\"admin\") in the router" mais le middleware n'existe pas.
- **Why it matters** : spec CLAUDE.md cite `middleware.RequireRole("agency", "provider")`. N'existe pas. Distinction de rôle se fait au niveau service. OK comme défense en profondeur mais pas de garde routeur sur les endpoints role-spécifiques. Régression possible si un endpoint provider est touché par enterprise.
- **How to fix** : 
```go
func RequireRole(roles ...string) func(http.Handler) http.Handler {
    allowed := make(map[string]bool, len(roles))
    for _, r := range roles {
        allowed[r] = true
    }
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role, ok := GetUserRole(r.Context())
            if !ok || !allowed[role] {
                res.Error(w, http.StatusForbidden, "forbidden", "role not authorized for this endpoint")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```
- **Test required** : `require_role_test.go` table-driven — agency → /jobs allowed, provider → /jobs 403, etc.
- **Effort** : S (1-2h)

### SEC-FINAL-04 (was SEC-23) : URLs user-controlled — pas de SSRF protection
- **Severity**: 🟠 HIGH
- **CWE** : CWE-918 (Server-Side Request Forgery)
- **Location** : 
  - `backend/internal/domain/profile/social_link.go:87-103` — `ValidateSocialURL`
  - `backend/internal/handler/dto/request/job.go` (VideoURL field)
  - `backend/internal/handler/dto/request/portfolio.go` (image URLs imported externally)
- **Why it matters** : `ValidateSocialURL` rejette `javascript:`/`data:` mais accepte `http://10.0.0.1`, `http://localhost`, `http://169.254.169.254` (AWS metadata), `http://[::1]`, `http://2130706433` (decimal localhost). Pas fetché aujourd'hui mais futur scraping (OG image, link preview) sera vulnérable. Le token rotation IAM, les credentials EC2, des services internes — tous exposés via SSRF dès qu'un workflow scraping ajoute un `http.Get(userURL)`.
- **How to fix** : 
```go
func ValidateSocialURL(rawURL string) error {
    parsed, err := url.ParseRequestURI(rawURL)
    if err != nil { return ErrInvalidURL }
    if parsed.Scheme != "https" { return ErrInvalidURL } // strict https in prod
    if parsed.Host == "" { return ErrInvalidURL }
    
    // Resolve and check it's a public IP
    ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), parsed.Hostname())
    if err != nil { return ErrInvalidURL }
    for _, ip := range ips {
        if ip.IP.IsPrivate() || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() ||
           ip.IP.IsLinkLocalMulticast() || ip.IP.IsMulticast() || ip.IP.IsUnspecified() {
            return ErrPrivateIPRejected
        }
    }
    return nil
}
```
- **Test required** : table-driven `social_link_test.go` rejet sur tous les RFC1918 + link-local + 169.254.169.254 + IPv6 loopback + decimal/octal encodings.
- **Effort** : S (1-2h)

### SEC-FINAL-05 (was SEC-29) : GDPR — endpoints `/me/export` et `/me/account` (DELETE) absents
- **Severity**: 🟠 HIGH (compliance — RGPD Art. 15-17)
- **CWE** : conformité RGPD Art. 15 (right of access), Art. 17 (right to erasure)
- **Location** : backend complet — pas de `users_handler.go::ExportMyData` ni `DeleteAccount`. Cascades posées (91 `ON DELETE CASCADE`) mais aucun endpoint utilisateur exposé. Pas de purge Typesense documentée sur user delete.
- **Why it matters** : un projet B2B qui collecte du business data sans implémenter Art. 15-17 est inéligible à passer un audit RGPD côté entreprise (les enterprises clientes vont refuser le contrat). Pour être éligible Open-Source européen, c'est une checkbox.
- **How to fix** :
  1. `GET /api/v1/me/export` retourne un JSON avec toutes les données personnelles : profile, organizations memberships, proposals, disputes, reviews, messages (anonymized other parties), invoices, audit logs (own actor rows).
  2. `DELETE /api/v1/me/account` (soft delete + scheduled hard purge sous 30 jours) :
     - Soft : `users.deleted_at = now()`, anonymize `email, first_name, last_name, display_name` to deterministic hash.
     - Scheduled job dans `pending_events` : purge Typesense documents, purge MinIO assets (avatar, video, portfolio), hard delete users row après 30j.
     - Cascades existantes prennent le relais pour proposals/disputes/messages/notifications.
- **Test required** : `gdpr_test.go` — register user, créer profile + 1 proposal + 5 messages, GET /me/export retourne tous les éléments. DELETE /me/account → anonymize. Re-GET /me/export → 401. Vérifier Typesense doc purgé via mock.
- **Effort** : L (2 jours)

### SEC-FINAL-06 (was BUG-NEW-08) : Stripe Connect error messages leak to API response
- **Severity**: 🟠 HIGH (information disclosure)
- **CWE** : CWE-209 (information exposure through error message)
- **Location** : `backend/internal/handler/embedded_handler.go:140, 173, 180, 246`
- **Why it matters** : `res.Error(w, http.StatusInternalServerError, "stripe_error", err.Error())` passe le raw Stripe SDK error directement au client. Stripe errors include account IDs, request IDs, Stripe internal request paths. Le `invalid_json` branche à line 140 leak des Go struct field names : `"json: cannot unmarshal number into Go struct field accountSessionRequest.country"` — un attaquant peut énumérer la shape interne.
- **How to fix** : remplacer par sanitized messages :
  - `"invalid_json", "request body could not be parsed as JSON"` (no jsonErr details)
  - `"stripe_error", "the Stripe operation failed"` (no err.Error())
  - Garder details dans `slog.Error("...", "error", err)` pour debugging serveur.
- **Test required** : handler test posts invalid JSON, asserter response body NE CONTIENT PAS "Go struct field" ni "json:". Mock un Stripe error, asserter response body NE CONTIENT PAS "acct_" ni "rseq_".
- **Effort** : XS (30 min)

### SEC-FINAL-07 (was SEC-30) : Admin token stocké en `localStorage`
- **Severity**: 🟠 HIGH
- **CWE** : CWE-922 (insecure storage of sensitive information)
- **Location** : `admin/src/lib/api-client.ts:24, 38-42`
- **Why it matters** : XSS dans une dep admin (transitive vuln npm) = siphonage instant des admin tokens. Le projet admin a 0 cross-feature et 0 `any` mais reste vulnérable côté supply chain — un seul package compromis = full admin take-over.
- **How to fix** : passer en httpOnly cookie + CSRF token séparé dans header `X-CSRF-Token`. Le cookie httpOnly carrie l'auth, le CSRF token (read from `<meta name="csrf-token">` rendered server-side) protect contre les cross-site requests.
- **Test required** : asserter via document.cookie en JS qu'aucun token n'est lisible. Asserter qu'un `fetch` cross-origin sans CSRF token est rejeté 403.
- **Effort** : M (½j)

---

## MEDIUM (9)

### SEC-FINAL-08 (was SEC-25) : `RequestID` middleware accepte `X-Request-ID` arbitraire — log injection
- **Severity**: 🟡 MEDIUM
- **CWE** : CWE-117 (improper output neutralization for logs)
- **Location** : `backend/internal/handler/middleware/requestid.go:30-37`
- **How to fix** : 
```go
incoming := r.Header.Get("X-Request-ID")
if incoming != "" {
    if _, err := uuid.Parse(incoming); err != nil {
        incoming = "" // fall through to regenerate
    }
}
if incoming == "" {
    incoming = uuid.New().String()
}
```
- **Test required** : `requestid_test.go` — header `X-Request-ID: foo\nbar` (log injection) → regenerated UUID, original swallowed.
- **Effort** : XS (15 min)

### SEC-FINAL-09 (was SEC-26) : Permissions middleware fallback statique sur sessions legacy
- **Severity**: 🟡 MEDIUM
- **CWE** : CWE-285
- **Location** : `backend/internal/handler/middleware/permission.go:46-54`
- **How to fix** : forcer logout des sessions sans `Permissions` field (bumper `session_version` une fois en prod), OU fallback DB (charge les permissions de l'org membership row).
- **Effort** : S (1-2h)

### SEC-FINAL-10 (was SEC-27) : Stripe Connect `account_id` exposé dans `GET /account-status`
- **Severity**: 🟡 MEDIUM
- **CWE** : CWE-200 (information exposure)
- **Location** : `backend/internal/handler/embedded_handler.go:87, 92, 251`
- **Why it matters** : DTO `embeddedAccountStatusResponse.AccountID` retourne le Stripe account ID au client. C'est utile uniquement côté serveur (pour les API calls Stripe). Le frontend n'en a pas besoin pour render le status.
- **How to fix** : retirer `AccountID` des deux DTOs publics. Ne renvoyer que `(charges_enabled, payouts_enabled, requirements_count, disabled_reason, country, business_type)`. Garder l'ID interne à la DB seulement.
- **Test required** : handler test asserte response body ne contient pas `"account_id"` ni `"acct_"`.
- **Effort** : XS (15 min)

### SEC-FINAL-11 (was SEC-28) : `X-Forwarded-For` accepté sans CIDR allowlist (role_overrides_handler)
- **Severity**: 🟡 MEDIUM
- **CWE** : CWE-348 (use of less trusted source)
- **Location** : `backend/internal/handler/role_overrides_handler.go:189-203`
- **Why it matters** : ratelimit middleware applique le CIDR allowlist (Phase 1 SEC-11) mais `clientIP()` dans `role_overrides_handler` reste naïf — fait `r.Header.Get("X-Forwarded-For")` sans vérifier que le `r.RemoteAddr` est dans l'allowlist proxy. Un attaquant peut spoof son IP dans les role override audit logs.
- **How to fix** : extraire le helper du middleware ratelimit (`extractRealIP(r, trustedCIDRs)`) en `pkg/httputil/realip.go` partagé. L'utiliser ici.
- **Effort** : S (1-2h)

### SEC-FINAL-12 (was SEC-31) : Audit logs `ON DELETE SET NULL` perd l'attribution sur user delete
- **Severity**: 🟡 MEDIUM
- **CWE** : CWE-778 (insufficient logging)
- **Location** : `backend/migrations/078_create_audit_logs.up.sql:30`
- **Why it matters** : quand un user delete son compte (RGPD), tous ses audit_logs perdent leur `user_id` (SET NULL). On ne peut plus tracer qui a fait quoi en cas d'enquête.
- **How to fix** : capturer `actor_email_hash` (sha256 sur l'email avant deletion) dans `metadata` JSONB à l'écriture du log. UUID anonymisée + hash de l'email permettent un audit forensique sans PII directe.
- **Test required** : `audit_repository_test.go::TestActorEmailHashStored` — write audit row, delete user, asserter row.user_id == NULL et row.metadata->>'actor_email_hash' != "".
- **Effort** : S (1-2h)

### SEC-FINAL-13 (was SEC-32) : `Authorization` header — pas de redaction structurée dans slog
- **Severity**: 🟡 MEDIUM
- **CWE** : CWE-532 (insertion of sensitive information into log)
- **Location** : `backend/internal/handler/middleware/logger.go:43-54` — utilise `slog.NewJSONHandler` standard sans `ReplaceAttr`.
- **Why it matters** : pkg/redact existe (regex bearer/sk-/emails) mais n'est appliqué qu'à des sites manuels. Tout `slog.Info("...", "headers", r.Header)` accidentel fuite les bearer tokens.
- **How to fix** : `slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: ..., ReplaceAttr: redact.SlogReplaceAttr})` qui appelle `pkg/redact.Redact` sur chaque attribut string.
- **Test required** : `logger_test.go` — log un attribut `authorization=Bearer foobar`, capture l'output, asserter `"Bearer [REDACTED]"` apparaît au lieu de `"Bearer foobar"`.
- **Effort** : S (1-2h)

### SEC-FINAL-14 (was SEC-34) : Cookie `user_role` non-httpOnly
- **Severity**: 🟡 MEDIUM (defense-in-depth, design choice)
- **Location** : `backend/internal/handler/cookie.go:42-51`
- **Why it matters** : voulu pour rendu UI client-side, OK si l'invariant "pas une décision serveur" est respecté. Mais aucun lint/grep CI vérifie que le code web ne fait pas de décision d'autorisation basée sur ce cookie.
- **How to fix** : ajouter une lint rule `web/eslint.config.mjs` qui interdit `cookies.get('user_role')` hors d'un fichier marqué `@allow-user-role-cookie`. Centraliser dans `web/src/shared/lib/auth-cookie-ui.ts` qui assert "UI-only".
- **Effort** : S (1-2h)

### SEC-FINAL-15 (NEW) : FCM device tokens jamais marqués stale
- **Severity**: 🟡 MEDIUM (resource exhaustion + UX)
- **CWE** : CWE-400 (uncontrolled resource consumption)
- **Location** : `backend/internal/adapter/fcm/push.go:75-83`
- **Why it matters** : sur erreur Firebase (UNREGISTERED, INVALID_ARGUMENT), `slog.Warn("fcm send failed for token", ...)` mais aucune action sur la DB. Notification fan-out gaspille des appels API à des tokens morts. Quotas Firebase consommés inutilement.
- **How to fix** : 
```go
if response.FailureCount > 0 {
    var staleIndexes []int
    for i, resp := range response.Responses {
        if resp.Error != nil {
            if messaging.IsUnregistered(resp.Error) || messaging.IsInvalidArgument(resp.Error) {
                staleIndexes = append(staleIndexes, i)
            }
        }
    }
    if len(staleIndexes) > 0 && s.deviceTokens != nil {
        staleTokens := make([]string, len(staleIndexes))
        for j, i := range staleIndexes { staleTokens[j] = tokens[i] }
        if err := s.deviceTokens.MarkStale(ctx, staleTokens); err != nil {
            slog.Warn("fcm: failed to mark stale tokens", "error", err)
        }
    }
}
```
Plus le repo `device_tokens.MarkStale(ctx, tokens []string) error` qui set `is_stale = true` et purge après 7 jours.
- **Test required** : mock FCM client retourning `IsUnregistered=true` for token X, asserter device_tokens.MarkStale appelé avec X.
- **Effort** : S (1-2h)

### SEC-FINAL-16 (NEW) : `RetryFailedTransfer` directly mutates `record.TransferStatus` without state machine guard
- **Severity**: 🟡 MEDIUM (state machine bypass)
- **CWE** : CWE-841 (improper enforcement of behavioral workflow)
- **Location** : `backend/internal/app/payment/payout_transfer.go:992` (was service_stripe.go in old layout) — `record.TransferStatus = domain.TransferPending`.
- **Why it matters** : raw field assignment bypass `MarkTransferFailed` / `MarkTransferred` / `ApplyDisputeResolution` guarded mutators. Pas de `MarkTransferRetrying()` method. Une retry sur transfer déjà completed peut silently revert à pending et trigger un duplicate Stripe transfer (idempotency-keyed mais DB drifte).
- **How to fix** : ajouter `func (r *PaymentRecord) MarkTransferRetrying() error { if r.TransferStatus != TransferFailed { return ErrInvalidStateTransition }; r.TransferStatus = TransferPending; return nil }` et l'utiliser ici.
- **Test required** : state machine test — `RetryFailedTransfer` rejette records qui ne sont pas en `TransferFailed`.
- **Effort** : XS (30 min)

---

## LOW (4)

- **SEC-FINAL-17** : `JWT_SECRET` hardcodé dans tests (`pkg/crypto/jwt_test.go`) → générer via `crypto/rand` dans `TestMain`. Effort: XS.
- **SEC-FINAL-18** : Pas de pipeline CI gosec/semgrep — déjà govulncheck + trivy hebdo, ajouter gosec sur PR. Effort: XS.
- **SEC-FINAL-19** : Health check Redis dans `/ready` — vérifier qu'il existe vraiment et qu'il fail-fast si Redis down. Effort: XS.
- **SEC-FINAL-20** (NEW) : Conversation `tx.Commit` ignored on existing-conversation lookup (BUG-NEW-19). `_ = tx.Commit()` at `conversation_repository.go:43`. Effort: XS.

---

## OWASP Top 10 (2021) coverage matrix

| OWASP | Status | Notes |
|---|---|---|
| A01 Broken Access Control | 🟠 (35 GetByID callers) | SEC-FINAL-01 deployment blocker |
| A02 Cryptographic Failures | ✅ | bcrypt 12, JWT short-lived, HSTS |
| A03 Injection | ✅ | parameterized everywhere, gosec sweep PR #34 |
| A04 Insecure Design | 🟡 | idempotency missing (SEC-FINAL-02) |
| A05 Security Misconfiguration | 🟠 | Slowloris (PERF-FINAL-B-01), no `RequireRole` (SEC-FINAL-03) |
| A06 Vulnerable & Outdated Components | ✅ | govulncheck + trivy weekly |
| A07 Identification & Auth Failures | ✅ | brute force, refresh rotation, session_version |
| A08 Software & Data Integrity | 🟡 | webhook idempotency OK after PR #36, Stripe sig OK |
| A09 Logging & Monitoring Failures | 🟡 | SEC-FINAL-13 (slog redact), SEC-FINAL-12 (actor_email_hash) |
| A10 Server-Side Request Forgery | 🟠 | SEC-FINAL-04 (SSRF social URLs) |

## Already shipped (verified during this audit)

- ✅ Stripe webhook signature verification (`adapter/stripe/webhook.go:16`)
- ✅ Mobile secure storage (`flutter_secure_storage` + Keychain/EncryptedSharedPreferences)
- ✅ Cookie httpOnly + SameSite=Lax + Secure-en-prod (session_id)
- ✅ JSON `DisallowUnknownFields` (`pkg/validator`)
- ✅ WebSocket short-lived single-use `ws_token` (Redis `GetDel`)
- ✅ Session_version revocation infrastructure (middleware/auth + propagation rôle org)
- ✅ Recovery middleware avec request_id correlation
- ✅ Magic-bytes validation generalised (`UploadPortfolioImage` + autres uploads)
- ✅ CORS allowlist explicite (pas de wildcard)
- ✅ Bcrypt cost 12
- ✅ MaxBytesReader sur uploads (5/50/100 MB)
- ✅ Audit log domain + repo + table m.078, append-only enforced m.124
- ✅ Forgot password ne révèle pas l'existence de l'email
- ✅ Stripe IdempotencyKey sur Transfers/Payouts/PaymentIntents
- ✅ Web `poweredByHeader: false`, pas de localStorage tokens (proxy session cookie)
- ✅ `.env*` gitignorés
- ✅ Optimistic concurrency milestones (`version` column)
- ✅ Pending events outbox `FOR UPDATE SKIP LOCKED`
- ✅ RLS m.125 sur 9 tables tenant-scoped + audit_logs WITH CHECK m.129
- ✅ Webhook handler error → 503 retry + idempotency release (BUG-NEW-06)
- ✅ Audit attribution admin (BUG-NEW-09 closed)
- ✅ pending_events stale recovery m.128

---

## Strong points

- Architecture hexagonale strictement respectée — surface d'attaque limitée par DI
- Système `session_version` + middleware Auth à 4 étages bien commenté
- Permissions org-scoped avec override per-org : bonne base RBAC
- Handlers admin systématiquement gardés `RequireAdmin` + `NoCache`
- Stripe Connect rigoureux : IdempotencyKey systématique, signature stricte, account session délégué
- Modération avec fail-closed sur OpenAI 5xx (`moderateDisplayName` refuse l'inscription)
- Pas de SQL string-concat utilisateur sur les paths publics (gosec sweep clean)
- Pas de fuite secrets dans les logs sur les sites couverts par redact

---

## Top 10 fixes restants ordonnés par ROI

| # | ID | Severity | Effort | Impact |
|---|---|---|---|---|
| 1 | SEC-FINAL-01 | CRITICAL | L (3j) | Pre-prod RLS rotation blocker |
| 2 | SEC-FINAL-02 | HIGH | M | doubles missions/disputes/reviews |
| 3 | SEC-FINAL-04 | HIGH | S | SSRF preparation (future scraping) |
| 4 | SEC-FINAL-03 | HIGH | S | RBAC defense-in-depth |
| 5 | SEC-FINAL-05 | HIGH | L | RGPD Art. 15-17 compliance |
| 6 | SEC-FINAL-06 | HIGH | XS | Stripe error info leak |
| 7 | SEC-FINAL-07 | HIGH | M | Admin XSS = no token exfil |
| 8 | SEC-FINAL-13 | MEDIUM | S | Structured slog redaction |
| 9 | SEC-FINAL-12 | MEDIUM | S | Audit attribution post-RGPD |
| 10 | SEC-FINAL-15 | MEDIUM | S | FCM stale tokens cleanup |
