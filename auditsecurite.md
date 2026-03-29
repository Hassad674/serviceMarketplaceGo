# Audit de Securite

Audit du 2026-03-30. Fichier genere automatiquement.

---

## CRITICAL

### SEC-01: Security Headers manquants
- **Probleme**: Pas de middleware SecurityHeaders dans le backend
- **Headers manquants**:
  - `Content-Security-Policy`
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Permissions-Policy: camera=(), microphone=(), geolocation=()`
  - `X-XSS-Protection: 0`
- **Impact**: Vulnerable a XSS, MIME sniffing, clickjacking
- **Fix**: Creer middleware SecurityHeaders dans `backend/internal/handler/middleware/security_headers.go`

---

## HIGH

### SEC-02: Rate limiting insuffisant sur les endpoints auth
- **Fichier**: `backend/internal/handler/middleware/ratelimit.go`
- **Probleme**: Rate limiter en memoire, pas de protection brute force Redis-based
- **Requis par CLAUDE.md**: 5 tentatives/15min par email, lockout 30min
- **Fix**: Implementer Redis sliding window + lockout

### SEC-03: Pas de protection CSRF
- **Probleme**: Pas de tokens CSRF sur les endpoints state-changing (POST/PUT/DELETE)
- **Impact**: Cross-site request forgery sur les utilisateurs authentifies
- **Mitigation actuelle**: SameSite=Lax sur les cookies (protection partielle)
- **Fix**: Pour les requetes cross-origin, ajouter validation CSRF token

### SEC-04: Validation uploads — pas de magic bytes
- **Fichier**: `backend/internal/handler/upload_handler.go`
- **Probleme**: Verifie Content-Type header seulement, pas le contenu reel du fichier
- **Impact**: Upload de fichiers malveillants deguises (SVG avec JS, exe renomme)
- **Fix**: Lire les premiers octets du fichier pour valider le type reel

### SEC-05: URLs de profil non validees
- **Fichier**: `backend/internal/handler/profile_handler.go`
- **Probleme**: PhotoURL, VideoURL acceptent n'importe quelle URL
- **Impact**: SSRF, stored XSS via URLs malveillantes
- **Fix**: Whitelister protocole (https) + domaine (storage propre) + rejeter `javascript:`, `data:`

### SEC-06: Ownership checks a verifier dans tous les handlers
- **Fichiers**: Tous les handlers de mutation
- **Probleme**: Verifier que CHAQUE endpoint de mutation verifie la propriete de la ressource
- **Pattern requis**:
```go
if resource.UserID != userID && role != "admin" {
    response.Error(w, http.StatusForbidden, "forbidden", "you do not own this resource")
    return
}
```

---

## MEDIUM

### SEC-07: CORS Access-Control-Allow-Credentials sans condition
- **Fichier**: `backend/internal/handler/middleware/cors.go`
- **Fix**: Ne setter que quand l'origin est whitelistee

### SEC-08: Auth middleware — fallthrough silencieux
- **Fichier**: `backend/internal/handler/middleware/auth.go`
- **Probleme**: Si session invalide, tombe silencieusement vers le check Bearer au lieu de fail fast
- **Fix**: Logger les echecs d'authentification pour audit trail

### SEC-09: Pas de rate limiting sur password reset et email verification
- **Probleme**: Ces endpoints sensibles n'ont pas de rate limiting specifique
- **Fix**: 3 requetes/heure par email

### SEC-10: Validation DTO incomplète
- **Fichier**: `backend/internal/handler/dto/request/auth.go`
- **Probleme**: Pas de validation de format email, pas de validation force mot de passe au niveau DTO
- **Fix**: Ajouter des tags de validation struct ou valider dans le handler

---

## LOW

### SEC-11: Secrets JWT hardcodes dans les tests
- **Fichier**: `backend/pkg/crypto/jwt_test.go`
- **Risque**: Faible, mais mauvaise pratique

### SEC-12: Pas de CI/CD pour scan securite
- **Probleme**: Aucun pipeline GitHub Actions avec scan SAST
- **Fix**: Ajouter gosec, trivy, ou dependabot
