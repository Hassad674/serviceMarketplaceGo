# Bugs et problemes a corriger

Audit du 2026-03-30. Fichier genere automatiquement.

---

## CRITICAL

### BUG-01: Erreurs silencieuses dans le service de paiement Stripe
- **Fichier**: `backend/internal/app/payment/service_stripe.go`
- **Lignes**: ~38-39, ~69, ~74
- **Probleme**: `_ = s.records.Update(ctx, existing)` — les erreurs de mise a jour des records de paiement sont ignorees silencieusement
- **Impact**: Les records de paiement peuvent ne pas refleter l'etat Stripe reel. Problemes de reconciliation billing
- **Fix**: Logger l'erreur meme si on ne fait pas echouer la requete :
```go
if err := s.records.Update(ctx, existing); err != nil {
    slog.Warn("failed to update payment record", "error", err, "record_id", existing.ID)
}
```

### BUG-02: Race condition sur la creation de paiement par proposal
- **Fichier**: `backend/internal/app/payment/service_stripe.go`
- **Lignes**: ~20-49
- **Probleme**: Si deux requetes concurrentes tentent de creer un paiement pour la meme proposal :
  1. Les deux verifient `GetByProposalID` → not found
  2. Les deux appellent `Create` → la deuxieme echoue
  3. Fallback vers `createPaymentIntentFromExisting` avec potentiellement des params differents
- **Impact**: Payment intents en double ou inconsistants
- **Fix**: Ajouter une contrainte UNIQUE sur `proposal_id` dans la table `payment_records` + gerer le conflit

---

## HIGH

### BUG-03: Fichiers de test depassant 1000 lignes
- **Fichiers**:
  - `backend/internal/app/messaging/service_test.go` — 1129 lignes
  - `backend/internal/app/auth/service_test.go` — 935 lignes
- **Probleme**: Depassent largement la limite de 600 lignes du projet
- **Fix**: Splitter en fichiers plus petits (happy path, edge cases, etc.)

### BUG-04: Validation de fichiers upload insuffisante
- **Fichier**: `backend/internal/handler/upload_handler.go`
- **Lignes**: ~53-57, ~107-110
- **Probleme**: Verifie seulement `strings.HasPrefix(contentType, "image/")` — permet les SVG (image/svg+xml) qui peuvent contenir du JavaScript
- **Impact**: XSS stocke via injection SVG
- **Fix**: Valider les magic bytes du fichier, n'autoriser que JPEG/PNG/WebP pour les images

### BUG-05: Pas de pagination sur SearchProfiles
- **Fichier**: `backend/internal/handler/profile_handler.go`
- **Lignes**: ~83-115
- **Probleme**: `limit := 20` hardcode, pas de cursor pagination. Impossible de naviguer au-dela des 20 premiers resultats
- **Fix**: Implementer cursor-based pagination comme les autres endpoints

### BUG-06: Upload de fichiers > 10MB charge tout en memoire
- **Fichier**: `backend/internal/handler/upload_handler.go`
- **Probleme**: `ParseMultipartForm` charge l'integralite du fichier en memoire (max 100MB pour les videos de review)
- **Impact**: OOM possible sur uploads volumineux
- **Fix**: Utiliser du streaming/chunked upload pour fichiers > 10MB

### BUG-07: Validation URL manquante sur les profils
- **Fichier**: `backend/internal/handler/profile_handler.go`
- **Lignes**: ~44-81
- **Probleme**: PhotoURL, PresentationVideoURL, ReferrerVideoURL acceptent des URLs arbitraires sans validation de protocole ou domaine
- **Impact**: SSRF potentiel, XSS stocke, contenu trompeur
- **Fix**: Ajouter validation URL dans `pkg/validator` — whitelister protocole (https) et domaine (storage propre)

---

## MEDIUM

### BUG-08: CORS Access-Control-Allow-Credentials inconditionnel
- **Fichier**: `backend/internal/handler/middleware/cors.go`
- **Lignes**: ~14-34
- **Probleme**: Le header `Access-Control-Allow-Credentials: true` est set meme quand l'origin n'est pas whitelistee
- **Fix**: Ne setter que quand l'origin est autorisee

### BUG-09: Rate limiter en memoire sans persistance
- **Fichier**: `backend/internal/handler/middleware/ratelimit.go`
- **Lignes**: ~23-32
- **Probleme**: Map en memoire qui croit sans limite, reset au restart. Pas de rate limiting Redis pour les endpoints sensibles
- **Fix**: Migrer vers Redis sliding window pour les endpoints auth

### BUG-10: Erreurs de broadcast ignorees dans le messaging
- **Fichier**: `backend/internal/app/messaging/service_helpers.go`
- **Lignes**: ~37-73
- **Probleme**: Les erreurs de broadcast sont logguees mais jamais retournees — l'utilisateur ne voit pas ses messages se propager en temps reel si le broadcast echoue
- **Impact**: UX degradee silencieusement
- **Fix**: Ajouter monitoring/alerting sur les echecs de broadcast

### BUG-11: 0 error.tsx et 0 loading.tsx dans le web
- **Fichier**: `web/src/app/`
- **Probleme**: Aucun error boundary ni loading state au niveau des route groups
- **Impact**: Erreurs non captees, pas de skeleton loading
- **Fix**: Ajouter error.tsx et loading.tsx dans chaque route group

---

## LOW

### BUG-12: Logging d'erreurs inconsistant
- **Fichier**: Tout le backend
- **Probleme**: Certaines erreurs logguees avec contexte complet, d'autres avec info minimale
- **Fix**: Standardiser le format de log d'erreur

### BUG-13: Secrets JWT hardcodes dans les tests
- **Fichier**: `backend/pkg/crypto/jwt_test.go`
- **Lignes**: ~12, ~84-85
- **Probleme**: Secrets de test visibles dans le repo — risque d'utilisation accidentelle en production
- **Fix**: Generer des secrets aleatoires dans le setup de test
