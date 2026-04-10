# Team Management V1 — Progress Report

**Branch**: `feat/team-management`
**Isolated DB**: `marketplace_go_team` (localhost:5435)
**Started**: 2026-04-10

---

## Architecture finalisée

- Table `organizations` séparée (pas de `user.id = org`)
- **1 seul Owner** par org (V1), évolutif vers N plus tard
- Transfer ownership = flow 2 étapes avec acceptation (7 jours d'expiration)
- Escape hatch admin plateforme pour les cas extrêmes
- Account types : `marketplace_owner` (Agency/Enterprise/Provider) + `operator` (invité)
- 4 rôles org hardcodés : Owner, Admin, Member, Viewer
- Permissions dans JWT (session_version reporté à Phase 3)
- Notifs : contenu partagé + état personnel, filtrage par permission

---

## Checkpoints

| # | Phase | Statut | Notes |
|---|-------|--------|-------|
| **CP1** | **Après Phase 1 (backend core)** | 🟢 **READY — tu peux valider** | Smoke test OK, voir section CP1 ci-dessous |
| CP2 | Après Phase 4 (resource migration) | ⏳ À venir | Vérifier zéro régression |
| CP3 | Après Phase 7 (web) | ⏳ À venir | Tester flow invite complet sur web |
| CP4 | Début Phase 8 (mobile) | ⏳ À venir | Wireless debug code |
| CP5 | Fin Phase 8 (mobile validé) | ⏳ À venir | Tester sur Xiaomi |

---

## Phase 0 — Foundation ✅ TERMINÉE (commit `d90b8da`)

- 5 migrations (053-057) : organizations, organization_members, organization_invitations, users.account_type + session_version, repoint FK
- Domain layer (6 fichiers, ~720 lignes) : role, permissions, errors, organization, member, invitation
- **43 tests unitaires verts**
- 3 interfaces port

---

## Phase 1 — Backend core ✅ TERMINÉE — PRÊT POUR CP1

### Ce qui a été fait

**1. Extensions des ports services** (additif, non-breaking) :
- `service.TokenClaims` : nouveaux champs `OrganizationID *uuid.UUID` + `OrgRole string`
- `service.AccessTokenInput` : nouvelle struct (paramètre unique pour `GenerateAccessToken`)
- `service.CreateSessionInput` : nouvelle struct (paramètre unique pour `Create`)
- `service.Session` : nouveaux champs `OrganizationID` + `OrgRole`
- Mocks existants mis à jour (3 fichiers), zéro régression

**2. JWT + Session avec org context** :
- `pkg/crypto/jwt.go` : `customClaims` étendu avec `org_id` + `org_role` (avec `omitempty` pour rester compact pour les Providers)
- `adapter/redis/session.go` : `sessionData` stocke org context
- Round-trip validé : Generate → Validate préserve l'org

**3. User domain étendu** :
- Nouvelle enum `AccountType` (`marketplace_owner` | `operator`)
- Nouveau champ `User.AccountType`
- Nouvelle fonction `NewOperator(...)` pour le flow d'invitation (Phase 2)
- `NewUser` défaut à `marketplace_owner`
- `user_repository.go` étendu : account_type ajouté à SELECT/INSERT/UPDATE des 6 méthodes (Create, GetByID, GetByEmail, Update, ListAdmin, RecentSignups)

**4. Middleware auth enrichi** :
- Nouvelles context keys : `ContextKeyOrganizationID` + `ContextKeyOrgRole`
- Nouveaux helpers : `GetOrganizationID(ctx)` + `GetOrgRole(ctx)`
- Le middleware extrait l'org context depuis le cookie session OU depuis les claims JWT Bearer

**5. Postgres adapters pour les 3 tables org** (`adapter/postgres/`) :
- `organization_repository.go` : Create, CreateWithOwnerMembership (atomique), FindByID, FindByOwnerUserID, Update, Delete
- `organization_member_repository.go` : Create, FindByID, FindByOrgAndUser, FindOwner, FindUserPrimaryOrg, List (cursor), CountByRole, Update, Delete
- `organization_invitation_repository.go` : Create, FindByID, FindByToken, FindPendingByOrgAndEmail, List, Update, Delete, ExpireStale

**6. App service `app/organization/service.go`** :
- `Service.CreateForOwner(ctx, user)` : crée org + owner member en 1 transaction
- `Service.ResolveContext(ctx, userID)` : résout org + member + permissions (nil si solo)
- `Service.HasPermission(ctx, userID, perm)` : check permission effective
- Rejette les Providers via `ErrProviderCannotOwnOrg`

**7. Auth service wiring** :
- Nouvelle interface locale `OrgProvisioner` dans le package auth
- Nouveau constructeur `NewServiceWithDeps(ServiceDeps)` avec injection d'`Orgs`
- `Register()` : appelle `orgs.CreateForOwner(u)` pour Agency/Enterprise, skip pour Provider
- `Login()` et `RefreshToken()` : appellent `orgs.ResolveContext(userID)` pour inclure org dans le token
- Helpers `buildAccessInput` et `buildAuthOutput` pour centraliser la logique
- `AuthOutput` étendu avec `OrganizationID` + `OrgRole`

**8. /me endpoint augmenté** :
- `AuthHandler` injecte `orgService *orgapp.Service`
- `Me()` appelle `orgService.ResolveContext(userID)` et renvoie `MeResponse{User, Organization}`
- `sendAuthResponse()` inclut l'org context dans la réponse mobile (token mode) et web (cookie mode)

**9. DTOs réponses** (`handler/dto/response/auth.go`) :
- `UserResponse.AccountType` ajouté
- Nouveau `OrganizationResponse` avec `id, type, owner_user_id, member_role, member_title, permissions[]`
- Nouveau `MeResponse{User, Organization?}` pour `/me`
- `AuthResponse` étendu avec `Organization?` (omitempty pour Providers)
- `NewAuthResponseWithOrg()` helper

**10. Wiring main.go** :
- 3 nouveaux repos injectés (organization, org_member, org_invitation)
- 1 nouveau service injecté (`organizationapp.NewService(...)`)
- `authSvc` construit via `auth.NewServiceWithDeps(ServiceDeps{...})` avec `Orgs: organizationSvc`
- `authHandler := handler.NewAuthHandler(authSvc, organizationSvc, sessionSvc, cookieCfg)`

### Tests

- **9 tests unitaires** pour `app/organization/service.go` (CreateForOwner Agency/Enterprise, rejette Provider, ResolveContext, HasPermission par rôle)
- **Tous les tests backend existants passent** — zéro régression
- **Tests smoke end-to-end manuels** sur la DB isolée :

```
✅ POST /auth/register (role=agency) → 201
   - user créé avec account_type='marketplace_owner'
   - organization créée (type=agency)
   - organization_members ligne avec role='owner'
   - JWT contient org_id + org_role='owner'
   - Response inclut organization.permissions (21 perms Owner)

✅ POST /auth/register (role=provider) → 201
   - user créé avec account_type='marketplace_owner'
   - AUCUNE organisation créée
   - JWT SANS org_id ni org_role
   - Response SANS champ "organization"

✅ GET /auth/me (Agency token) → 200
   - user avec account_type
   - organization complète avec les 21 permissions

✅ GET /auth/me (Provider token) → 200
   - user avec account_type
   - AUCUN champ "organization"
```

### DB state final (isolated DB)

| Table | Count |
|-------|-------|
| users (total) | 933 |
| users (marketplace_owner) | 933 |
| users (operator) | 0 |
| organizations | 1 (créée par le smoke test Agency) |
| organization_members | 1 (l'Owner auto-créé) |
| organization_members (role=owner) | 1 |

Les 931 users pré-existants dans la DB clonée ont été migrés automatiquement en `marketplace_owner` via le default de la migration 056. Aucune donnée perdue.

---

## 🟢 Checkpoint CP1 — À toi de valider

### Ce que tu dois vérifier (5-10 min)

**Objectif** : confirmer qu'une Agency/Enterprise s'inscrit et obtient bien son org automatiquement, et qu'un Provider reste solo inchangé.

### Option A — Tester manuellement via curl sur la DB isolée

Je peux relancer le backend sur port 8084 pointé sur la DB isolée :

```bash
cd backend
DATABASE_URL="postgres://postgres:postgres@localhost:5435/marketplace_go_team?sslmode=disable" \
  PORT=8084 \
  JWT_SECRET="dev-marketplace-secret-key-change-in-production-2024" \
  REDIS_URL="redis://localhost:6380" \
  STORAGE_ENDPOINT="192.168.1.156:9000" \
  STORAGE_ACCESS_KEY="minioadmin" \
  STORAGE_SECRET_KEY="minioadmin" \
  STORAGE_BUCKET="marketplace" \
  STORAGE_USE_SSL="false" \
  STORAGE_PUBLIC_URL="http://192.168.1.156:9000/marketplace" \
  SESSION_TTL="336h" \
  ALLOWED_ORIGINS="http://localhost:3001" \
  go run cmd/api/main.go
```

Puis tester :

```bash
# Agency registration
curl -s -X POST http://localhost:8084/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Auth-Mode: token" \
  -d '{"email":"test-agency@example.com","password":"Pass1234!","first_name":"T","last_name":"T","display_name":"Test Agency","role":"agency"}' | python3 -m json.tool

# Provider registration
curl -s -X POST http://localhost:8084/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Auth-Mode: token" \
  -d '{"email":"test-provider@example.com","password":"Pass1234!","first_name":"T","last_name":"T","role":"provider"}' | python3 -m json.tool

# /me with Agency token
curl -s -H "Authorization: Bearer <agency_access_token>" http://localhost:8084/api/v1/auth/me | python3 -m json.tool
```

### Option B — Faire confiance au smoke test que j'ai déjà fait

J'ai exécuté exactement ce flow avec succès, logs disponibles dans la section "Tests" ci-dessus. Si tu me fais confiance, tu peux juste dire **"go phase 2"** et j'enchaîne.

### Option C — Inspecter le commit et le code manuellement

- Commit Phase 0 : `d90b8da`
- Commit Phase 1 : (en cours de création)
- `git log --oneline -5`
- `git diff d90b8da..HEAD --stat` pour voir le scope du changement

### Ce qui te prouve que CP1 est OK

- [ ] Agency `/auth/register` retourne un champ `organization` avec `member_role='owner'` et un tableau de permissions de 21 éléments
- [ ] Provider `/auth/register` retourne **SANS** champ `organization`
- [ ] Agency `/auth/me` renvoie user + organization
- [ ] Provider `/auth/me` renvoie user sans organization
- [ ] Backend compile (`cd backend && go build ./...`)
- [ ] Tous les tests passent (`cd backend && go test ./...`)

---

## Décisions loggées pendant Phase 1

### D6 — Signature GenerateAccessToken en struct (AccessTokenInput)
**Contexte** : ajouter org context à un token demandait soit 2 params de plus (orgID + orgRole), soit une struct.
**Décision** : struct `AccessTokenInput`. Plus lisible à l'usage (nommage explicite) et extensible (session_version viendra en Phase 3 sans nouveau refactor).
**Impact** : 20+ call sites mis à jour (mocks + tests + 3 calls dans auth/service.go). Zéro régression.

### D7 — Session API similairement via CreateSessionInput
**Décision** : mêmes raisons que D6. `SessionService.Create(ctx, CreateSessionInput)` plus clair.

### D8 — OrgProvisioner interface locale dans le package auth
**Contexte** : le service auth a besoin de déléguer la création d'org sans dépendre directement de `*orgapp.Service`.
**Décision** : interface `OrgProvisioner` définie dans le package `auth`, que `*orgapp.Service` satisfait naturellement. Permet des mocks en test tout en gardant une DI propre.

### D9 — session_version reporté à Phase 3
**Contexte** : la migration 056 a ajouté la colonne, mais wire l'infra complète en Phase 1 ajoutait du scope.
**Décision** : Phase 1 ne propage PAS session_version dans JWT/Session/middleware. Phase 3 (team management) ajoutera tout le flow (incrémenter à chaque changement de rôle + check dans middleware via Redis).
**Justification** : la valeur reste à 0 en DB, rien n'est cassé, la révocation immédiate arrive avec la feature qui en a vraiment besoin.

### D10 — Repurpose vs refactor de user.organization_id
**Décision** : la colonne `users.organization_id` existe depuis migration 001. Migration 057 a repointé son FK vers `organizations(id)`. En Phase 1, je ne la populate PAS — je m'appuie sur `organization_members.FindUserPrimaryOrg(userID)` comme source de vérité unique.
**Justification** : évite la double source de vérité. La colonne reste comme cache optionnel pour des optimisations futures (ex: joindre users avec leur org en 1 query).

### D11 — NewServiceWithDeps (struct constructor) conservé en parallèle de NewService
**Contexte** : tests existants utilisent `NewService(...)` avec 6 params positionnels.
**Décision** : ajouter `NewServiceWithDeps(ServiceDeps)` pour la production sans casser les tests. Mixte V1, on pourra consolider plus tard.

---

## Blockers

_Aucun._

---

## Next step — Phase 2 (après validation CP1)

**Objectif** : invitation flow complet (envoi + acceptation).

À faire en Phase 2 :
1. App service `app/organization/invitation_service.go` avec SendInvitation, ValidateToken, AcceptInvitation, ResendInvitation, CancelInvitation, ListPending
2. Email template Resend "team_invitation" en français (avec redirection vers `hassad.smara69@gmail.com` en dev)
3. Handlers HTTP :
   - `POST /api/v1/organizations/{id}/invitations` (Owner/Admin)
   - `DELETE /api/v1/organizations/{id}/invitations/{invID}` (cancel)
   - `POST /api/v1/organizations/{id}/invitations/{invID}/resend`
   - `GET /api/v1/organizations/{id}/invitations` (list pending)
   - `GET /api/v1/invitations/validate?token=X` (public)
   - `POST /api/v1/invitations/{token}/accept` (public — crée le user operator)
4. Rate limit : 10 invitations/heure/org via Redis sliding window
5. Vérifications pré-envoi : email déjà membre, déjà opérateur ailleurs, etc.
6. Tests unitaires + smoke test manuel (envoi → réception par `hassad.smara69@gmail.com` → création du operator)

**Durée estimée** : 2 jours.
**Checkpoint CP2** : non — CP2 est après Phase 4 (resource migration).

---

## Historique des phases

- **Phase 0** (2026-04-10, commit `d90b8da`) : Foundation — 5 migrations, 6 domain files (~720 lines), 43 tests passants, 3 port interfaces
- **Phase 1** (2026-04-10, commit en cours) : Backend core — JWT/Session/User étendus, 3 postgres adapters, app/organization service, auth flow wire, /me augmenté, main.go wiring, 9 nouveaux tests, smoke test E2E validé
