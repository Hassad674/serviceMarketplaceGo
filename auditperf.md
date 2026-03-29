# Audit de Performance

Audit du 2026-03-30. Fichier genere automatiquement.

---

## BACKEND

### PERF-01: SearchProfiles sans pagination (HIGH)
- **Fichier**: `backend/internal/handler/profile_handler.go:83-115`
- **Probleme**: Limite hardcodee a 20, pas de cursor. Impossible de paginer au-dela
- **Impact**: Fonctionnalite de recherche incomplete, pas scalable
- **Fix**: Implementer cursor-based pagination (comme les autres endpoints)

### PERF-02: Upload fichiers charge en memoire (HIGH)
- **Fichier**: `backend/internal/handler/upload_handler.go`
- **Probleme**: ParseMultipartForm charge tout en RAM (jusqu'a 100MB pour review videos)
- **Impact**: OOM sous charge, latence elevee sur gros fichiers
- **Fix**: Streaming multipart vers S3/R2 avec io.Pipe

### PERF-03: Pool Redis non verifie (MEDIUM)
- **Fichier**: `backend/internal/adapter/redis/client.go`
- **Probleme**: Verifier que PoolSize, MinIdleConns, MaxRetries sont configures
- **Impact**: Connexions Redis non optimisees sous charge
- **Fix**: Configurer PoolSize=50, MinIdleConns=10, MaxRetries=3

### PERF-04: Indexes de recherche a verifier (MEDIUM)
- **Fichier**: Migration 006 `add_search_indexes`
- **Probleme**: Verifier que tous les champs utilises dans des WHERE ont un index
- **Indexes manquants potentiels**:
  - `payment_records(proposal_id)` — utilise dans GetByProposalID
  - `notifications(user_id, read)` — utilise dans CountUnread
  - `proposals(conversation_id)` — utilise dans les requetes de messaging
- **Fix**: Auditer chaque requete SELECT avec EXPLAIN ANALYZE

### PERF-05: Pas de cache Redis sur les endpoints lourds (MEDIUM)
- **Probleme**: Pas de cache-aside pour les donnees lues frequemment
- **Endpoints candidats**:
  - `GET /api/v1/profiles/:id` (profils publics) — TTL 5min
  - `GET /api/v1/reviews/average/:userId` — TTL 10min
  - `GET /api/v1/jobs` (liste publique) — TTL 2min
- **Fix**: Implementer pattern cache-aside avec Redis

### PERF-06: Conversation Repository isolation SERIALIZABLE (LOW)
- **Fichier**: `backend/internal/adapter/postgres/conversation_repository.go:27-73`
- **Probleme**: Utilise SERIALIZABLE pour prevenir les conversations dupliquees — correct mais couteux
- **Impact**: Contention sous forte charge sur les conversations
- **Monitoring**: Surveiller les retries de serialization failure

---

## WEB

### PERF-07: Pas de monitoring taille bundle (MEDIUM)
- **Probleme**: Aucune verification de la taille du bundle JS en CI
- **Cible**: < 200KB gzipped (initial load)
- **Fix**: Ajouter @next/bundle-analyzer + check en CI

### PERF-08: Pas de loading.tsx (skeletons) (MEDIUM)
- **Fichier**: `web/src/app/[locale]/`
- **Probleme**: 0 fichier loading.tsx — pas de skeleton loading sur les pages
- **Impact**: Flash de contenu vide pendant le chargement
- **Fix**: Ajouter loading.tsx avec skeletons dans chaque route group

### PERF-09: Imports non optimises potentiels (LOW)
- **Probleme**: Verifier que les barrel imports ne tirent pas des modules entiers
- **Deja configure**: `experimental.optimizePackageImports: ["lucide-react", "clsx", "@tanstack/react-query"]`
- **A verifier**: recharts, @stripe/react-stripe-js (lazy load si pas utilise partout)

---

## MOBILE

### PERF-10: Pas de profiling recent (LOW)
- **Probleme**: Pas de benchmarks Flutter DevTools documentes
- **Cibles**: 60fps min, cold start < 2s, APK < 30MB
- **Fix**: Profiler avec Flutter DevTools et documenter les resultats

---

## BASE DE DONNEES

### PERF-11: Auditer toutes les requetes > 50ms (MEDIUM)
- **Probleme**: Pas de monitoring automatise des requetes lentes
- **Fix**: Ajouter du logging de duree sur chaque requete dans l'adapter PostgreSQL :
```go
start := time.Now()
defer func() {
    duration := time.Since(start)
    if duration > 50*time.Millisecond {
        slog.Warn("slow query", "query", queryName, "duration_ms", duration.Milliseconds())
    }
}()
```

### PERF-12: Connection pool PostgreSQL (OK)
- **Fichier**: `backend/internal/adapter/postgres/db.go:17-20`
- **Config actuelle**: MaxOpen=50, MaxIdle=25, MaxLifetime=30min, MaxIdleTime=5min
- **Statut**: Raisonnable pour le scale actuel. Monitorer si croissance.
