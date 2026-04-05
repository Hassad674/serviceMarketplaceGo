# Migration KYC custom → Stripe Embedded Components

**Branche** : `feat/test-stripe-embedded` (worktree isolé)
**Date** : 2026-04-05
**Status** : Plan de migration rédigé, **aucune suppression effectuée**.

---

## Résumé

Le worktree `test-embedded` livre une implémentation **production-ready** de la KYC
via Stripe Connect Embedded Components :

- Route `/payment-info-v2` (web)
- Handler backend `EmbeddedHandler` (`internal/handler/embedded_handler.go`)
- Notifier intelligent diff-based (`internal/app/embedded/`)
- 82+ tests unitaires (47 handler + 35 notifier) + 25+ E2E Playwright

La présente doc explique **quoi supprimer / conserver** pour finaliser la migration
sur `main`.

---

## Fichiers à supprimer (~2500 lignes)

### Frontend (web/src)

#### Code de formulaire custom KYC
- `web/src/features/payment-info/components/payment-info-page.tsx`
- `web/src/features/payment-info/components/identity-verification-section.tsx`
- `web/src/features/payment-info/components/dynamic-section.tsx`
- `web/src/features/payment-info/components/business-persons-section.tsx`
- `web/src/features/payment-info/components/activity-sector-select.tsx`
- `web/src/features/payment-info/components/country-selector.tsx` *(le custom, pas celui de payment-info-v2)*
- `web/src/features/payment-info/components/country-select.tsx`
- `web/src/features/payment-info/hooks/use-payment-info.ts`
- `web/src/features/payment-info/hooks/use-identity-documents.ts`
- `web/src/features/payment-info/hooks/use-country-fields.ts`
- `web/src/features/payment-info/hooks/__tests__/*`
- `web/src/features/payment-info/api/payment-info-api.ts`
- `web/src/features/payment-info/api/identity-document-api.ts`
- `web/src/features/payment-info/lib/country-states.ts`
- `web/src/features/payment-info/types.ts`
- `web/src/app/[locale]/(app)/payment-info/page.tsx` *(remplacée par `/payment-info-v2`)*

#### Banner custom requirements
- `web/src/features/payment-info/components/stripe-requirements-banner.tsx`
  *(remplacé par `<ConnectNotificationBanner />`)*

#### E2E tests custom (obsolètes)
- `web/e2e/kyc-flow-multi.spec.ts`
- Tous les autres tests E2E qui ciblent `/payment-info` custom

**Estimation frontend** : ~1800 lignes à supprimer.

### Backend (backend/internal)

#### Handlers custom
- `backend/internal/handler/payment_info_handler.go`
- `backend/internal/handler/payment_info_handler_test.go`
- `backend/internal/handler/identity_document_handler.go`
- `backend/internal/handler/identity_document_handler_test.go`
- `backend/internal/handler/dto/request/payment_info.go`
- `backend/internal/handler/dto/response/payment_info.go`
- `backend/internal/handler/dto/response/identity_document.go`

#### App layer custom
- `backend/internal/app/payment/service_country_fields.go`
- `backend/internal/app/payment/service_identity.go`
- `backend/internal/app/payment/service_identity_test.go`
- `backend/internal/app/payment/service_requirements.go` *(remplacé par Notifier)*
- `backend/internal/app/payment/service_requirements_test.go`

**Conserver** : `service_stripe.go` et `service.go` (utilisés par proposal/wallet).

#### Adapters
- `backend/internal/adapter/stripe/country_spec.go`
- `backend/internal/adapter/stripe/person.go`
- `backend/internal/adapter/stripe/file_upload.go`
- `backend/internal/adapter/stripe/account.go` *(partiellement — conserver Get/Update)*
- `backend/internal/adapter/redis/country_spec_cache.go`
- `backend/internal/adapter/postgres/identity_document_repository.go`

#### Domain / ports
- `backend/internal/domain/payment/country_spec.go`
- `backend/internal/domain/payment/identity_document.go`
- `backend/internal/port/repository/identity_document_repository.go`
- `backend/internal/port/service/country_spec_service.go`

**Conserver** :
- `backend/internal/domain/payment/entity.go` → PaymentInfo domain encore utilisé
- `backend/internal/domain/payment/requirements.go` → utilisé par Notifier-like fonctions
- `backend/internal/port/service/stripe_service.go` → PaymentIntent, Transfer, Webhook

#### Routes à retirer dans `router.go`
```go
// À retirer :
r.Get("/", deps.PaymentInfo.GetPaymentInfo)
r.Put("/", deps.PaymentInfo.SavePaymentInfo)
r.Get("/status", deps.PaymentInfo.GetPaymentInfoStatus)
r.Get("/requirements", deps.PaymentInfo.GetRequirements)
r.Get("/country-fields", deps.PaymentInfo.GetCountryFields)
r.Post("/account-link", deps.PaymentInfo.CreateAccountLink)

// À conserver :
r.Post("/account-session", deps.Embedded.CreateAccountSession)
r.Delete("/account-session", deps.Embedded.ResetAccount)
r.Get("/account-status", deps.Embedded.GetAccountStatus)
```

**Estimation backend** : ~3500 lignes à supprimer.

### Database

- **NE PAS** dropper la table `payment_info` tout de suite — elle contient les
  `stripe_account_id` existants des utilisateurs production.
- Migration à créer : renommer `test_embedded_accounts` → `embedded_accounts` (ou
  fusionner avec `payment_info` en gardant uniquement les colonnes essentielles).

**Schema final suggéré** (`payment_info` ou `embedded_accounts`) :
```sql
CREATE TABLE embedded_accounts (
    user_id           UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    stripe_account_id TEXT NOT NULL UNIQUE,
    country           TEXT NOT NULL,
    last_state        JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Toutes les colonnes suivantes deviennent inutiles (gérées par Stripe) :
- `extra_fields`, `individual_data`, `business_data`
- `dob`, `address_*`, `iban`, `phone`, `email`
- `persons`, `documents`, etc.

---

## Fichiers à CONSERVER

### Frontend
- `web/src/app/[locale]/(app)/payment-info-v2/**` (la nouvelle implémentation)
- `web/src/app/[locale]/(app)/test-embedded/**` (page test pour QA/dev)
- Notifications hooks (`use-notifications.ts`, etc.) — réutilisés par Embedded

### Backend
- `backend/internal/handler/embedded_handler.go`
- `backend/internal/app/embedded/**`
- `backend/internal/adapter/stripe/payment_intent.go` (charge creation)
- `backend/internal/adapter/stripe/transfer.go` (payouts)
- `backend/internal/adapter/stripe/webhook.go`
- `backend/internal/handler/stripe_handler.go`
- `backend/migrations/038_create_test_embedded_accounts.*.sql`
- `backend/migrations/039_add_last_state_to_test_embedded_accounts.*.sql`
- Tests notification domain (déjà mis à jour pour les types Embedded)

---

## Étapes de migration recommandées

### Phase 1 — Parallèle (semaine 1)
- ✅ Déployer `/payment-info-v2` en parallèle de `/payment-info` classique
- ✅ Rediriger les nouveaux comptes vers v2
- Laisser les comptes existants sur v1 pour le moment

### Phase 2 — Migration des comptes existants (semaine 2-3)
- Pour chaque `payment_info.stripe_account_id` existant :
  - Vérifier que le compte Stripe est toujours actif
  - Créer une ligne `embedded_accounts` avec le même `stripe_account_id`
  - Supprimer la ligne `payment_info` correspondante
- Migration SQL one-shot écrite + testée sur db de staging

### Phase 3 — Suppression code custom (semaine 4)
- Supprimer tous les fichiers listés ci-dessus
- Mettre à jour `router.go` pour retirer les routes obsolètes
- `go build ./... && go test ./...` doit passer

### Phase 4 — Suppression route v1 (semaine 5)
- Rediriger `/payment-info` → `/payment-info-v2` (301)
- Supprimer `web/src/app/[locale]/(app)/payment-info/page.tsx`
- Renommer `/payment-info-v2` → `/payment-info`

### Phase 5 — Cleanup DB (semaine 6)
- Migration : `DROP TABLE payment_info` (si fusion dans `embedded_accounts`)
- Ou `ALTER TABLE payment_info DROP COLUMN extra_fields, DROP COLUMN ...`

---

## Rollback plan

Si Embedded ne convient pas après déploiement :

1. **Code** : `git revert` les commits de migration. Le worktree isolé permet
   un revert propre sans affecter le reste du système.
2. **Data** : tant que la phase 5 n'est pas faite, `payment_info` contient
   toutes les données nécessaires pour réactiver le flow custom.
3. **Stripe** : les comptes connectés Stripe sont indépendants du choix UI —
   ils restent valides et utilisables par le flow custom.

---

## Checklist pré-merge vers `main`

- [ ] Tous les tests passent (`go test ./... && npx playwright test`)
- [ ] Migrations DB testées sur staging
- [ ] Notifications testées end-to-end (Stripe CLI `trigger account.updated`)
- [ ] Smoke test : 5 pays différents (FR, US, DE, SG, GB) onboardent sans erreur
- [ ] Edge cases testés : document rejeté, requirement ajouté, compte suspendu
- [ ] PR review par au moins 1 autre dev
- [ ] Dashboard Stripe production vérifié manuellement
- [ ] Rollback plan validé sur staging

---

## Bénéfices attendus après migration

| Métrique | Avant (custom) | Après (Embedded) |
|----------|---------------|------------------|
| Lignes de code KYC | ~5300 | ~1700 (-68%) |
| Pays supportés | 7 (avec bugs) | 45 (battle-tested) |
| Maintenance compliance | Manuelle | Auto (Stripe updates) |
| Time-to-support-new-country | 3-5 jours dev | `country: "XX"` |
| Edge cases par compte | 10-15 bugs/mois | ~0 (géré par Stripe) |
| Traductions | 1 langue (fr) | 40+ auto |
