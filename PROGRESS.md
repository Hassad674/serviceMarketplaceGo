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
- Permissions dans JWT + session_version pour révocation immédiate
- Notifs : contenu partagé + état personnel, filtrage par permission

---

## Checkpoints

| # | Phase | Statut | Notes |
|---|-------|--------|-------|
| CP1 | Après Phase 1 (backend core) | ⏳ À venir | Tester inscription Agency + Provider |
| CP2 | Après Phase 4 (resource migration) | ⏳ À venir | Vérifier zéro régression |
| CP3 | Après Phase 7 (web) | ⏳ À venir | Tester flow invite complet sur web |
| CP4 | Début Phase 8 (mobile) | ⏳ À venir | Wireless debug code |
| CP5 | Fin Phase 8 (mobile validé) | ⏳ À venir | Tester sur Xiaomi |

---

## Phase 0 — Foundation ✅ TERMINÉE

### Setup
- ✅ Branche `feat/team-management` créée depuis `main` (commit 1117249)
- ✅ DB isolée `marketplace_go_team` créée via `pg_dump | psql` (30 tables copiées)
- ✅ PROGRESS.md initialisé

### Migrations (053-057)
**⚠️ Note** : les numéros partent à 053 (pas 047 comme initialement prévu) — les migrations 047-052 existaient déjà (portfolio + disputes AI budget).

| N° | Fichier | Objet |
|----|---------|-------|
| 053 | `create_organizations` | Table `organizations` + pending_transfer fields + CHECK constraint de cohérence |
| 054 | `create_organization_members` | Table `organization_members` + partial UNIQUE on role='owner' (single-Owner DB-enforced) |
| 055 | `create_organization_invitations` | Table `organization_invitations` + UNIQUE partial on (org, email) WHERE pending |
| 056 | `add_team_fields_to_users` | `users.account_type` + `users.session_version` avec index |
| 057 | `repoint_users_organization_fk` | Change FK `users.organization_id` de `users(id)` vers `organizations(id)` |

**Down migrations** : écrites et testées (toutes reversibles).

### Domain layer (`backend/internal/domain/organization/`)
| Fichier | Lignes | Description |
|---------|--------|-------------|
| `role.go` | ~60 | Role enum (Owner/Admin/Member/Viewer) + IsValid, CanBeInvitedAs, IsElevated |
| `permissions.go` | ~130 | 21 Permission constants + `rolePermissions` map (source de vérité unique) + HasPermission, PermissionsFor |
| `errors.go` | ~50 | 30 sentinels couvrant tous les cas (validation, lifecycle, invariants V1) |
| `organization.go` | ~160 | Organization entity + OrgType + NewOrganization + InitiateTransfer/CancelTransfer/CompleteTransfer |
| `member.go` | ~120 | Member entity + NewMember + ChangeRole + UpdateTitle + HasPermission + CanManageMember |
| `invitation.go` | ~200 | Invitation entity + InvitationStatus + NewInvitation + token generation (32 bytes crypto/rand) + Accept/Cancel/MarkExpired + email validation |

**Total domain** : ~720 lignes réparties sur 6 fichiers (tous sous la limite de 600 lignes).

### Domain tests
| Fichier | Tests | Description |
|---------|-------|-------------|
| `role_test.go` | 4 | IsValid, String, CanBeInvitedAs, IsElevated |
| `permissions_test.go` | 8 | Owner (all 21 perms), Admin (restrictions), Member (capabilities), Viewer (read-only), unknown role |
| `organization_test.go` | 9 | Construction valide/invalide, InitiateTransfer, CancelTransfer, CompleteTransfer (+ erreurs), OrgType |
| `member_test.go` | 10 | Construction, ChangeRole, UpdateTitle, HasPermission, IsOwner, CanManageMember (owner/admin/member/viewer, cross-org) |
| `invitation_test.go` | 12 | Construction, role constraints, email validation, name validation, token uniqueness (100 iter), Accept, Cancel, Expired, MarkExpired, InvitationStatus |

**Total** : **43 tests unitaires, 100% verts.**

### Port layer (interfaces)
- `port/repository/organization_repository.go` — Create, FindByID, FindByOwnerUserID, Update, Delete
- `port/repository/organization_member_repository.go` — Create, FindByID, FindByOrgAndUser, FindOwner, FindUserPrimaryOrg, List, CountByRole, Update, Delete
- `port/repository/organization_invitation_repository.go` — Create, FindByID, FindByToken, FindPendingByOrgAndEmail, List, Update, Delete, ExpireStale

### Validation
- ✅ `go build ./...` — zéro erreur
- ✅ `go test ./internal/domain/organization/...` — 43/43 passent
- ✅ `go test ./internal/domain/...` — TOUS les domaines existants continuent à passer (zéro régression)

---

## Décisions loggées pendant Phase 0

### D1 — Repurpose de `users.organization_id` (au lieu de nouvelle colonne)
**Contexte** : la colonne `users.organization_id` existait déjà depuis la migration 001, avec une self-référence vers `users(id)` (pattern de l'ancienne marketplace). Zéro ligne n'utilise cette colonne.

**Décision** : au lieu d'ignorer la colonne et d'en créer une nouvelle, on repurpose :
- Migration 057 : `DROP CONSTRAINT users_organization_id_fkey; ADD CONSTRAINT ... REFERENCES organizations(id)`
- La colonne garde son nom, son index existant reste valide
- Le champ `User.OrganizationID *uuid.UUID` dans la Go entity n'a pas besoin d'être modifié (Phase 1 l'utilisera directement)

**Justification** : zéro refactor du user repository, semantic preserved ("l'org à laquelle ce user appartient"), minimise la surface de risque.

### D2 — account_type sur users (plutôt que de toucher à role)
**Contexte** : role est NOT NULL avec CHECK (agency|enterprise|provider). Un operator n'est "aucun de ces trois" au sens marketplace.

**Décision** : ajouter `users.account_type` ('marketplace_owner' | 'operator'), et **les operators conservent le role marketplace de leur org** (agency ou enterprise). Par exemple Marie operator de Acme (agency) a `role='agency'` + `account_type='operator'`.

**Justification** : zéro changement au CHECK constraint existant, zéro changement au code existant qui filtre par role, semantic cohérente (Marie "travaille pour une agence"), distinction explicite via le nouveau champ.

### D3 — Single-Owner V1 DB-enforced
**Décision** : plutôt que de s'appuyer uniquement sur le service layer, on ajoute un **partial unique index** `idx_org_members_unique_owner ON organization_members(organization_id) WHERE role = 'owner'`. Impossible d'insérer 2 Owners pour la même org, même en cas de race condition.

**Justification** : defense in depth. Le service layer reste la première ligne, le DB est le filet de sécurité.

### D4 — Resource migrations reportées à Phase 4
**Contexte** : Phase 0 était initialement prévue avec 14 migrations (tables d'org + resource migrations + notifications refactor).

**Décision** : Phase 0 = **5 migrations seulement** (fondation pure). Les resource migrations (`organization_id` sur jobs/proposals/wallets/etc.) partent en Phase 4. Le refactor des notifications part en Phase 5.

**Justification** : Phase 0 doit être atomique et non invasive. Les resource migrations touchent ~10 tables et demandent une étude plus poussée des FK existantes, c'est un risque qu'on isole.

### D5 — Incident rollback shared DB
**Contexte** : la première tentative d'application des migrations a ciblé la DB partagée `marketplace_go` au lieu de la DB isolée, parce que le Makefile charge `.env` via `set -a && . ./.env; set +a` qui override les env vars passées en amont.

**Impact** : shared DB monte à version 57 avec 3 tables vides. Aucune donnée touchée (0 rows affectées).

**Résolution** :
1. Rollback immédiat des 5 migrations sur shared DB via `go run cmd/migrate/main.go down` (répété 5 fois avec DATABASE_URL override)
2. Vérification : shared DB de retour à version 52, FK users.organization_id restaurée vers users(id), 0 users avec organization_id
3. Réapplication correcte sur `marketplace_go_team` via `DATABASE_URL=... go run cmd/migrate/main.go up` (bypass du Makefile)

**Workaround permanent** : pour toute commande migrate qui doit cibler la DB isolée, ne PAS utiliser `make migrate-up` mais `DATABASE_URL="postgres://postgres:postgres@localhost:5435/marketplace_go_team?sslmode=disable" go run cmd/migrate/main.go <cmd>` directement. Cela bypass le chargement de `.env`.

**Leçon apprise** : ajouter `DATABASE_URL` dans l'env de chaque commande migrate, jamais faire confiance au `.env` du repo pour les ops ciblant la DB isolée.

---

## Blockers

_Aucun._

---

## Next step — Phase 1

**Objectif** : wire le domain layer avec le backend existant.

À faire :
1. Ajouter `AccountType` enum + `SessionVersion` field sur le user domain
2. Écrire `adapter/postgres/organization_repository.go` + `organization_member_repository.go` + `organization_invitation_repository.go`
3. Écrire `app/organization/service.go` avec `CreateForOwner(ctx, user)`, `GetByUserID`, `GetMember`
4. Ajouter session_version aux claims JWT dans `adapter/jwt/token_service.go`
5. Ajouter check `session_version` dans middleware `auth.go` (lookup Redis)
6. Modifier le flow d'inscription Agency/Enterprise pour créer auto l'org + self-membership Owner
7. Augmenter `/me` pour retourner le org context + permissions effectives
8. Wire dans `cmd/api/main.go`

**Tests à écrire** :
- Inscription Agency → vérifier org créée + self-membership Owner
- Inscription Provider → vérifier PAS d'org créée
- JWT contient org_id + role + session_version
- /me retourne permissions correctes

**Livrable de fin** : Agency qui s'inscrit obtient son org et peut voir ses permissions via /me. Zero régression sur les Providers.

**Durée estimée** : 2 jours.

---

## Historique des phases

- **Phase 0** (2026-04-10) : Foundation — 5 migrations, 6 domain files (~720 lines), 43 tests passants, 3 port interfaces, branche `feat/team-management` commit XXX
