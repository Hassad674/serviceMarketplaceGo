# Audit Qualite et Refactoring

Audit du 2026-03-30. Fichier genere automatiquement.

---

## REFACTORING NECESSAIRE

### REF-01: Splitter les fichiers de test > 600 lignes (CRITICAL)
- `backend/internal/app/messaging/service_test.go` — 1129 lignes → splitter en 3
- `backend/internal/app/auth/service_test.go` — 935 lignes → splitter en 3
- **Convention**: max 600 lignes par fichier

### REF-02: conversation_repository.go approche la limite (MEDIUM)
- `backend/internal/adapter/postgres/conversation_repository.go` — 596 lignes
- Preparer le split si ca grandit

---

## COUVERTURE DE TEST ACTUELLE

| App | Fichiers testes | Total fichiers | Couverture |
|-----|----------------|----------------|------------|
| Backend domain | 11/18 | 61% |
| Backend app | 9/16 | 56% |
| Backend handler | 1/25 | 4% |
| Backend adapter postgres | 0/19 | 0% |
| Backend adapter redis | 0/6 | 0% |
| Backend adapter externe | 1/11 | 9% |
| Backend pkg | 5/5 | 100% |
| Web features | 8/16 | 50% |
| Web shared | 4/11 | 36% |
| Mobile | 13/113 | 11.5% |

**Cible**: 80%+ sur la logique metier (domain + app + handlers)

---

## MAUVAISES PRATIQUES DETECTEES

### QUAL-01: Erreurs ignorees avec `_ = err` (HIGH)
- Plusieurs endroits dans le code backend ignorent des erreurs
- Chaque `_ = err` devrait au minimum logger l'erreur
- Pattern minimum: `slog.Warn("non-critical error", "error", err)`

### QUAL-02: Pas de error boundaries React (HIGH)
- 0 fichier error.tsx dans web/src/app/
- Chaque route group devrait avoir un error boundary

### QUAL-03: Pas de loading states React (MEDIUM)
- 0 fichier loading.tsx dans web/src/app/
- Chaque route group devrait avoir un skeleton loading

### QUAL-04: Shared UI components vide (MEDIUM)
- `web/src/shared/components/ui/` est vide
- Pas de composants shadcn/ui scaffoldes
- Potentiel de duplication de code UI

### QUAL-05: Mockito disponible mais non utilise (mobile) (LOW)
- pubspec.yaml a mockito ^5.4.4 mais les tests utilisent des Fake classes manuelles
- Considerer mockito + @GenerateMocks pour une meilleure couverture

### QUAL-06: Pas de CI/CD (HIGH)
- Aucun pipeline GitHub Actions
- Tests non executes automatiquement
- Pas de check de build, lint, securite
- **Impact**: Regressions non detectees

---

## DETTE TECHNIQUE

1. **Integration tests backend**: 0 tests avec testcontainers — toute la couche adapter/postgres non testee
2. **E2E Playwright**: 9 specs ecrites mais jamais executees en CI
3. **OpenAPI generation**: Configuree mais pas automatisee
4. **Admin panel**: 7/10 pages sont des shells vides
5. **Mobile**: 45% complete, beaucoup de features presentation-only
6. **Notifications push**: Phase 6+ non implementee (FCM, Firebase mobile, queue worker)
