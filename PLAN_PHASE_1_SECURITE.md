# Plan détaillé — Phase 1 Sécurité

**Date** : 2026-04-29
**Objectif** : fermer toutes les vulnérabilités CRITICAL + HIGH de `auditsecurite.md` + bugs business critiques liés sécurité de `bugacorriger.md`.
**Stratégie** : 3 agents en parallèle, worktrees isolés, branches dédiées, tests E2E paranoïaques pour zéro retest manuel.

## ETA

| Élément | Estimation |
|---|---|
| Agent A (headers/sessions/audit) | 2 jours |
| Agent B (auth hardening) | 2.5 jours |
| Agent C (inputs/uploads/fraud) | 1.5 jours |
| **Wall time (parallèle)** | **~2.5 jours** |
| Validation cross-agent + smoke | 0.5 jour |
| **Total** | **~3 jours** |

## Findings traités (18) vs déférés

**Inclus Phase 1** : SEC-01, SEC-02 (=BUG-01), SEC-03, SEC-04, SEC-05+16, SEC-06 (+BUG-08), SEC-07, SEC-08, SEC-09+21, SEC-11, SEC-13, SEC-14, SEC-15, SEC-19, SEC-20, SEC-24

**Déférés** :
- SEC-36 ✅ fait Phase 0
- SEC-22 (RequireRole middleware) → Phase 3 refactor (defense-in-depth, pas critical)
- SEC-23 (SSRF profile URLs) → Phase 3 (pas de scraping actif)
- SEC-25 → SEC-40 (MEDIUM/LOW) → Phases ultérieures

---

## 🔵 Agent A — Headers, secrets, sessions, audit logging (2 jours)

**Branche** : `feat/security-headers-sessions`
**Worktree** : `/tmp/mp-phase-1-agent-a` (créé manuellement, hors `/home/hassad/serviceMarketplaceGo`)
**DB dédiée** : OUI (1 nouvelle migration audit_logs grants) — `marketplace_go_phase1_a`

### Scope

#### SEC-03 : Middleware `SecurityHeaders` (backend) + headers Next.js (web)
- **Crée** `backend/internal/handler/middleware/security_headers.go` avec :
  - `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `X-XSS-Protection: 0`
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains` (en prod uniquement)
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Permissions-Policy: camera=(), microphone=(), geolocation=()`
- **Wire** dans `backend/internal/handler/router.go` après `Recovery`, avant `CORS`
- **Web** : `web/next.config.ts` `async headers()` avec CSP étendue (autoriser Stripe, R2, MinIO, LiveKit) + autres headers identiques

#### SEC-04 : `JWT_SECRET` fail-fast au démarrage
- **Modifie** `backend/internal/config/config.go` :
  - En prod (`ENV=production`) : `log.Fatal` si `JWTSecret == "dev-secret-change-me"` ou `len(JWTSecret) < 32`
  - En dev : `slog.Warn` bruyant si fallback utilisé
- Idem `STORAGE_SECRET_KEY` (`minioadmin`) en prod

#### SEC-05 + SEC-16 : Bumper `session_version` sur Suspend/Ban/Reset
- **Modifie** `backend/internal/app/admin/service.go:invalidateAndNotify` : ajouter `s.users.BumpSessionVersion(ctx, userID)` avant le delete sessions
- **Modifie** `backend/internal/app/auth/service.go:ResetPassword` : après `MarkUsed`, appeler `BumpSessionVersion` + `sessionSvc.DeleteByUserID`

#### SEC-08 : Android cleartext
- **Modifie** `mobile/android/app/src/main/AndroidManifest.xml` : retirer `android:usesCleartextTraffic="true"` du `<application>`
- **Crée** `mobile/android/app/src/main/res/xml/network_security_config.xml` autorisant cleartext uniquement pour `10.0.2.2`, `192.168.0.0/16`, `localhost`
- **Référence** dans manifest : `android:networkSecurityConfig="@xml/network_security_config"`
- **Build-time check** dans `mobile/lib/core/network/api_client.dart` : `assert(kDebugMode || apiUrl.startsWith('https://'))` pour fail le build prod si cleartext

#### SEC-24 : CORS `Vary: Origin` + `Allow-Credentials` conditionnel
- **Modifie** `backend/internal/handler/middleware/cors.go` :
  - `w.Header().Add("Vary", "Origin")` toujours
  - `Access-Control-Allow-Credentials: true` uniquement si origin allowlistée
  - `Access-Control-Allow-Methods/Headers` aussi conditionnels (cohérence)

#### SEC-13 : Émettre les events audit log auth + grants DB
- **Crée** `backend/migrations/124_audit_logs_grants.up.sql` :
  ```sql
  -- Make audit_logs append-only at the database level.
  REVOKE UPDATE, DELETE ON audit_logs FROM PUBLIC;
  -- App user has only INSERT + SELECT (verified after deploy).
  ```
  Plus `_down.sql` symétrique (re-grant).
- **Modifie** `backend/internal/app/auth/service.go` : appeler `auditrepo.Log` pour :
  - `Login` (success + failure avec metadata `{email, reason, ip}`)
  - `Logout`
  - `RefreshToken` (success + reused-blacklisted = `token_reuse_detected`)
  - `RequestPasswordReset`, `ResetPassword`
- **Modifie** `backend/internal/app/admin/service.go` : audit pour `Suspend`, `Ban`, `Unban`, `ForceTransferOwnership`

### Tests Agent A (paranoïaque)

#### Backend Go
- `internal/handler/middleware/security_headers_test.go` (NEW) :
  - Table-driven : 7 headers, chaque header présent + valeur exacte
  - HSTS uniquement en prod (env-based)
  - Test sur OPTIONS (preflight) + GET + POST
- `internal/config/config_test.go` (extend) :
  - Fail-fast prod avec `JWT_SECRET=""`, `JWT_SECRET="dev-secret-change-me"`, `len < 32`
  - Pas de fail en dev (juste warn capturé via test logger)
- `internal/app/admin/service_test.go` (extend) :
  - `Suspend(targetID)` → `BumpSessionVersion` appelé sur targetID + sessions Redis purgées + audit log row écrit
  - Idem `Ban`, `Unban`, `ForceTransferOwnership`
- `internal/app/auth/service_test.go` (extend) :
  - Login success → audit row `login_success`
  - Login failure × 5 → 5 audit rows `login_failure` avec metadata.reason
  - Logout → audit row `logout`
  - PasswordReset → audit row + sessions purgées + session_version bumped
- `internal/handler/middleware/cors_test.go` (extend) :
  - `Vary: Origin` toujours présent
  - `Allow-Credentials` absent quand origin non-allowlistée

#### Web (Vitest + Playwright)
- `web/e2e/security/headers.spec.ts` (NEW Playwright) :
  - Visit `/`, fetch `/api/v1/health`, assert presence des 7 security headers + valeurs
  - HSTS attendu uniquement si NODE_ENV=production (skip en dev)
  - Visit `/dashboard` (auth required), assert headers présents là aussi
- `web/src/__tests__/middleware.test.ts` (NEW) :
  - CSP cible inclut `*.stripe.com`, `*.r2.cloudflarestorage.com`, `livekit.cloud`

#### Mobile (widget + integration)
- `mobile/test/core/network/api_client_security_test.dart` (NEW) :
  - `assert` cleartext URL en release mode → throws
  - https URL en release mode → OK

### Migration safety (Agent A)

L'agent doit :
1. `createdb -p 5435 -T marketplace_go marketplace_go_phase1_a`
2. Configurer `.env` du worktree avec cette DB
3. `make migrate-up` pour appliquer 124
4. Tester REVOKE en `psql` : tenter UPDATE → échec attendu, INSERT → succès
5. Après merge sur main, l'utilisateur applique sur la DB partagée
6. `dropdb` sur la copie avant fin

### Exit criteria Agent A

- [ ] `go build ./... && go vet ./... && go test ./... -count=1 -race` 100% green
- [ ] `cd web && npx tsc --noEmit && npx vitest run` 100% green
- [ ] Migration 124 applique + rollback green sur DB dédiée
- [ ] Manual curl `-I /api/v1/health` montre les 7 security headers
- [ ] Playwright e2e/security/headers passe

---

## 🟢 Agent B — Auth hardening (2.5 jours)

**Branche** : `feat/security-auth-hardening`
**Worktree** : `/tmp/mp-phase-1-agent-b`
**DB dédiée** : NON (pas de migration applicative) — DB partagée OK
**Dépend de** : Agent A pour le merge audit_logs grants (sinon non-bloquant)

### Scope

#### BUG-08 : Mobile single-flight refresh (PREREQUIS de SEC-06)
- **Modifie** `mobile/lib/core/network/api_client.dart` :
  - Champ privé `Future<bool>? _refreshInFlight`
  - `_tryRefreshToken` retourne le Future en cours si déjà en vol
  - Test : 2 calls API simultanés qui retournent 401 → 1 seul appel `/auth/refresh`

#### SEC-06 : Refresh token rotation + Redis blacklist
- **Crée** `backend/internal/adapter/redis/refresh_blacklist.go` :
  - `Blacklist(ctx, jti, ttl)` → `SET token_blacklist:{jti} 1 EX <remaining>`
  - `IsBlacklisted(ctx, jti)` → `EXISTS`
- **Modifie** `backend/internal/app/auth/service.go:RefreshToken` :
  - Vérifier `IsBlacklisted(jti)` → si oui, log `token_reuse_detected` audit + 401
  - Générer nouvelle paire access+refresh
  - `Blacklist(oldJti, oldRemainingTTL)` AVANT de retourner
- **Modifie** `Logout` : blacklister le refresh token courant
- **Modifie** `pkg/crypto/jwt.go` : s'assurer que `jti` (UUID v4) est inclus dans chaque token

#### SEC-07 : `BruteForceService` Redis
- **Crée** `backend/internal/port/service/bruteforce.go` :
  ```go
  type BruteForceService interface {
      IsLocked(ctx context.Context, email string) (bool, error)
      RecordFailure(ctx context.Context, email string) error
      RecordSuccess(ctx context.Context, email string) error
      RetryAfter(ctx context.Context, email string) (time.Duration, error)
  }
  ```
- **Crée** `backend/internal/adapter/redis/bruteforce.go` :
  - `login_attempts:{email}` counter, TTL 15min, INCR atomique
  - `login_locked:{email}` flag, TTL 30min, set quand counter ≥ 5
  - `RecordSuccess` supprime les deux clés
- **Modifie** `backend/internal/handler/auth_handler.go:Login` :
  - Avant validation pwd : `IsLocked(email)` → 429 + `Retry-After`
  - Après validation : `RecordSuccess` ou `RecordFailure`
- **Idem** sur `RequestPasswordReset` (3/h par email) et `ResetPassword`

#### SEC-11 : Rate limiter Redis sliding window 4 classes
- **Réécrit** `backend/internal/handler/middleware/ratelimit.go` :
  - Backend Redis (clés `ratelimit:{class}:{key}:{window}`)
  - 4 classes : `global` (100/min/IP), `auth` (5/min/email — délègue à BruteForceService), `mutation` (30/min/user), `upload` (10/min/user)
  - Headers `X-RateLimit-Limit/Remaining/Reset` + `Retry-After` sur 429
  - Parsing `X-Forwarded-For` derrière allowlist `TRUSTED_PROXIES` CIDR

#### SEC-14 : Web `?token=` query string restriction
- **Modifie** `web/src/middleware.ts` :
  - `?token=` accepté UNIQUEMENT sur `/payment-info`, `/subscribe/*`, `/billing/embed*`
  - Sur ces routes : POST immédiat à `/auth/web-session` pour échanger en cookie httpOnly + `redirect()` qui efface la query string
  - Sur toutes les autres routes : `?token=` ignoré, redirection `/login` si pas de cookie session

#### SEC-15 : Mobile WS auth single-use ws_token
- **Modifie** `backend/internal/handler/auth_handler.go` : `POST /api/v1/auth/ws-token` accepte JWT Bearer mobile et retourne ws_token (déjà existant pour web — étendre au mobile)
- **Modifie** `mobile/lib/features/messaging/data/messaging_ws_service.dart` : appeler `/auth/ws-token` puis ouvrir WS avec ce ticket
- **Modifie** `backend/internal/adapter/ws/connection.go` : retirer la stratégie 3 (`?token=` JWT)

#### SEC-20 : Password politique « caractère spécial »
- **Modifie** `backend/internal/domain/user/valueobject.go:NewPassword` :
  - Ajouter check `unicode.IsPunct(c)` ou allowlist `!@#$%^&*()...`
  - Pousser longueur min de 8 → 10
- **Update** UI register/reset password (web + mobile) : règle affichée aux users

### Tests Agent B (paranoïaque, EXHAUSTIFS)

#### Backend Go
- `internal/adapter/redis/refresh_blacklist_test.go` (NEW, testcontainers Redis) :
  - `Blacklist` puis `IsBlacklisted` → true
  - TTL respecté : `IsBlacklisted` après TTL → false
  - Concurrent blacklist + check : pas de race
- `internal/adapter/redis/bruteforce_test.go` (NEW, testcontainers Redis) :
  - 4 fails → not locked
  - 5e fail → locked + `RetryAfter > 0`
  - 6e attempt même avec good password → still locked (handler test)
  - Lockout TTL 30min
  - Counter TTL 15min reset après success
- `internal/app/auth/service_test.go` (extend EXHAUSTIVELY) :
  - **Refresh rotation** :
    - Refresh once → 200 + new pair, old jti blacklisted
    - Refresh same old token twice → 2nd call 401 + audit `token_reuse_detected`
    - Refresh after logout → 401
  - **Brute force** :
    - 5 fails → 6e call 429 + `Retry-After`
    - Success après 4 fails → counter reset
- `internal/handler/middleware/ratelimit_test.go` (NEW, testcontainers Redis) :
  - 4 classes : 100 hits global / 5 auth / 30 mutation / 10 upload
  - 429 + `X-RateLimit-Limit`, `Remaining`, `Reset` headers
  - `X-Forwarded-For` accepté seulement si `RemoteAddr ∈ TRUSTED_PROXIES`
  - Multi-pod simulation : 2 instances Redis-backed → quota partagé (vs in-memory qui doublait)
- `internal/domain/user/valueobject_test.go` (extend) :
  - `Password!23` ≥ 10 chars + special → OK
  - `Password123` (no special) → ErrInvalidPassword
  - `Pass!1` (< 10 chars) → ErrInvalidPassword

#### Web Playwright (e2e)
- `web/e2e/security/login-bruteforce.spec.ts` (NEW) :
  - 5 logins ratés sur même email → 6e tentative = 429 + UI "Trop de tentatives, réessayez dans 30 min"
  - Wait 31 min ou inject Redis del → re-test : login OK
- `web/e2e/security/refresh-rotation.spec.ts` (NEW) :
  - Login → save refresh token → refresh → save new pair
  - Reuse old refresh → 401 + automatic logout client-side
- `web/e2e/security/token-leakage.spec.ts` (NEW) :
  - Visit `/dashboard?token=ABC` → ignored, redirect `/login`
  - Visit `/payment-info?token=ABC` → cookie set, query stripped
- `web/e2e/security/csp-violation.spec.ts` (NEW) :
  - Page de test injecte `<img src="evil.com/steal">` → bloqué par CSP
  - Console capture le `report-only` violation

#### Mobile (widget + integration)
- `mobile/test/core/network/api_client_singleflight_test.dart` (NEW) :
  - 2 calls API simultanés retournent 401 → 1 seul `/auth/refresh` réseau
  - Use Dio mocker / fake_async pour vérifier ordre
- `mobile/integration_test/security/refresh_flow_test.dart` (NEW) :
  - Real backend, login → 401 forcé → refresh single-flight → continuation OK

### Exit criteria Agent B

- [ ] Pipeline backend + web + mobile green
- [ ] Playwright security suite complète green
- [ ] Manual smoke : login 6× wrong → 429 dans Postman
- [ ] Manual smoke : refresh token replay → 401 + audit log row visible

---

## 🟠 Agent C — Inputs, uploads, fraud (1.5 jours)

**Branche** : `feat/security-inputs-uploads`
**Worktree** : `/tmp/mp-phase-1-agent-c`
**DB dédiée** : NON
**Dépend de** : indépendant des autres agents

### Scope

#### SEC-01 : XSS JSON-LD escape
- **Crée** `web/src/shared/lib/json-ld.ts` :
  ```ts
  export function safeJsonLd(payload: unknown): string {
    return JSON.stringify(payload)
      .replace(/</g, "\\u003c")
      .replace(/-->/g, "--\\u003e")
      .replace(/ /g, "\\u2028")
      .replace(/ /g, "\\u2029")
  }
  ```
- **Modifie** `web/src/app/[locale]/(public)/freelancers/[id]/page.tsx`, `clients/[id]/page.tsx`, `referrers/[id]/page.tsx` : `dangerouslySetInnerHTML={{ __html: safeJsonLd(payload) }}`

#### SEC-02 / BUG-01 : ConfirmPayment vérifie Stripe avant `MarkPaymentSucceeded`
- **Modifie** `backend/internal/app/payment/service_stripe.go:ConfirmPayment` (ou wherever `MarkPaymentSucceeded` est appelé) :
  - Avant : `pi, err := s.stripe.PaymentIntents.Get(ctx, record.StripePaymentIntentID, nil)`
  - Si `err != nil || pi.Status != "succeeded"` → return `domain.ErrPaymentNotConfirmed`
  - Seulement après : `record.MarkPaid()`
- Audit log emission : `payment_confirm_attempt_unverified` si l'API client tente sans vrai PI

#### SEC-09 + SEC-21 : Magic bytes généralisés + filename randomisé
- **Refactor** `backend/internal/handler/upload_handler.go` :
  - Extraire helper `detectMimeFromBytes(bytes []byte) (mimeType, ext string, ok bool)` qui :
    - Lit les 512 premiers octets via `http.DetectContentType`
    - Whitelist par scope :
      - photo → `image/jpeg, image/png, image/webp` uniquement
      - video → `video/mp4, video/webm, video/quicktime` uniquement
      - document (KYC) → `application/pdf, image/jpeg, image/png` uniquement
    - SVG, HTML, exe → REJETÉ
  - Appliquer dans `UploadPhoto`, `UploadVideo`, `UploadReferrerVideo`, `UploadReviewVideo`, `UploadPortfolioImage`, `UploadPortfolioVideo`, `UploadIdentityDoc`
  - Filename de stockage : `<uuid>.{ext_from_magic}` — JAMAIS `header.Filename`
  - Reject si magic bytes détectent un type différent du Content-Type déclaré

#### SEC-19 : DTO validation avec `go-playground/validator`
- **Add** `github.com/go-playground/validator/v10` à `go.mod`
- **Refactor** `backend/pkg/validator/validator.go` :
  - Wrapper `Validate(struct any) error` qui retourne erreurs détaillées
  - Conversion vers `domain.ValidationError` avec `field`, `rule`, `message`
- **Tag** TOUS les DTOs `backend/internal/handler/dto/request/*.go` :
  - Strings : `validate:"required,min=1,max=N"` (N par champ)
  - UUIDs : `validate:"omitempty,uuid"`
  - Emails : `validate:"required,email"`
  - URLs : `validate:"omitempty,url"`
  - Plages : `validate:"gte=0,lte=999999999"` pour budgets/montants
- Cap longueurs raisonnables : title 200, description 5000, comment 2000, etc.

### Tests Agent C (paranoïaque)

#### Backend Go
- `internal/handler/upload_handler_test.go` (extend EXHAUSTIVELY) :
  - Table-driven 12+ scénarios :
    - JPEG valid → 201 + key randomized
    - PNG valid → 201
    - WebP valid → 201
    - SVG (`image/svg+xml` Content-Type) → 415
    - HTML déguisé en `image/png` → 415 (magic bytes catch)
    - .exe renommé `.png` → 415
    - Vidéo MP4 valid → 201
    - Vidéo > maxSize → 413
    - Texte vide → 400
    - PDF dans endpoint photo → 415
    - Filename avec `../../etc/passwd` → key randomisé, pas de traversal
- `internal/app/payment/service_stripe_test.go` (extend) :
  - `ConfirmPayment` avec PaymentIntent status `requires_payment_method` → erreur
  - status `processing` → erreur (pas encore succeeded)
  - status `succeeded` → OK
  - PI inexistant Stripe-side → erreur
  - PI ID null → erreur
- `pkg/validator/validator_test.go` (extend) :
  - Table-driven : champ requis vide, longueur trop courte, longueur trop longue, format email invalide, UUID invalide, montant négatif → tous rejets
- `internal/handler/dto/request/auth_test.go` (NEW) :
  - Register avec email `not-an-email` → 400 `validation_error`
  - Password manquant → 400
  - Tous les DTOs ont au moins 1 test de validation négatif

#### Web Vitest + Playwright
- `web/src/shared/lib/__tests__/json-ld.test.ts` (NEW) :
  - `</script>` → `</script>`
  - `<script>` → `<script>`
  - `  ` → escaped
  - Roundtrip avec JSON.parse identique
- `web/e2e/security/xss-jsonld.spec.ts` (NEW Playwright) :
  - Setup : créer profil avec `about = "</script><script>window.__pwned=true</script>"`
  - Visit `/freelancers/<id>` → `window.__pwned` undefined
  - Inspect HTML : `<script type="application/ld+json">` contient `</script>` (escaped)
- `web/e2e/security/upload-abuse.spec.ts` (NEW) :
  - Upload SVG via UI → erreur affichée
  - Upload .html déguisé → erreur

### Exit criteria Agent C

- [ ] Pipeline backend + web green
- [ ] Playwright xss-jsonld + upload-abuse green
- [ ] Manual : payload `</script>` dans about → no execution dans le browser
- [ ] Manual : SVG upload via API → 415

---

## ⚙️ Pipeline de validation cross-agent (avant merge final)

Une fois les 3 agents mergés sur leurs branches respectives :

1. **Orchestrateur (moi) merge dans cet ordre** :
   - Agent A (base : headers + sessions + audit)
   - Agent B (auth hardening, dépend du blacklist Redis qui est nouveau)
   - Agent C (indépendant)

2. **Smoke E2E global** (créé par moi après les 3 merges) :
   - `scripts/smoke/security-phase-1.sh` :
     - curl `-I` → 7 security headers
     - login 6× wrong → 429
     - login OK → refresh → reuse → 401
     - upload SVG → 415
     - profile.about avec `</script>` → escaped HTML
     - admin Suspend user → audit log row visible

3. **Final regression smoke** :
   - Backend : `go build ./... && go test ./... -count=1 -race`
   - Web : `npx vitest run && npx tsc --noEmit && npx playwright test e2e/security/`
   - Mobile : `flutter analyze && flutter test test/core test/integration_test/security/`

---

## 📋 Brief commun à TOUS les agents (à inclure dans chaque prompt)

```
## HARD RULES

1. NEVER touch /call/, /livekit/ — off-limits per LiveKit rule
2. Branch ownership: create your own from main, never reuse another agent's
3. Migration safety: if your scope touches migrations, use isolated DB copy
4. Scope discipline: implement exactly your scope, flag out-of-scope items
5. Test paranoia: tests BEFORE code on non-trivial scope, ≥90% coverage on
   touched files, table-driven, edge cases, fuzz where applicable
6. Validation pipeline mandatory before EVERY commit — paste output in report
7. End-of-phase report MUST include 3 retest lists (Critical/Moderate/Low)

## OUTPUT

Final report markdown ≤ 2500 words including:
- Per-item status (✅ done / ⚠️ partial / ❌ skipped + reason)
- Files added/modified
- Tests added (count + names)
- Validation pipeline output (real paste)
- Manual smoke commands (curl/Playwright snippets to verify)
- Critical / Moderate / Low retest lists
- Out-of-scope flags
```

---

## 🎯 Décisions à valider AVANT dispatch

1. **OK pour ce scope ?** (18 findings traités, 22 déférés explicitement)
2. **OK pour 3 agents en parallèle** ? (vs 1 séquentiel = 6 jours, ici 3 jours wall time)
3. **OK pour les nouvelles deps** ? `go-playground/validator/v10` côté backend (small, mainstream, ~5 imports stable)
4. **OK pour le mode plan obligatoire** sur chaque agent ? (ils me retournent leur plan détaillé avant de coder, je relaye, tu valides 5 min, ils codent)
5. **Tests E2E paranoïaques** : ce niveau te convient ? (~30 nouveaux tests Go + 8 Playwright + 5 mobile = ~43 tests sur 3 jours)

Une fois validé, je dispatche les 3 agents et je te ping à chaque retour.
