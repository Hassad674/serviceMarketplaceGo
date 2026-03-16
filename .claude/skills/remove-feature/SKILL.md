---
name: remove-feature
description: Safely remove a feature from the entire project (backend Go, web Next.js, mobile Flutter, admin Vite+React). Proves modularity by deleting all feature files, cleaning wiring, and verifying compilation. Never deletes migration files.
user-invocable: true
allowed-tools: Read, Edit, Bash, Grep, Glob, Agent
---

# Remove Feature

Remove the feature: **$ARGUMENTS**

You are safely removing a feature from the B2B Marketplace. This operation proves the modularity promise — every non-core feature must be deletable without breaking any app.

This is a multi-app project:

- **backend/** — Go 1.25, Chi v5, hexagonal architecture
- **web/** — Next.js 16, React 19, Tailwind 4, feature-based
- **mobile/** — Flutter 3.16+, Riverpod 2, Clean Architecture
- **admin/** — Vite + React 19, page-based

---

## GUARD — Core features cannot be removed

The following features are **core** and cannot be removed:
- **auth** — every app depends on authentication
- **user** — users are the foundation of every feature, all tables reference the users table

If `$ARGUMENTS` matches a core feature, **STOP** immediately and explain why it cannot be removed.

---

## STEP 1 — Inventory the feature

Map every file and reference belonging to this feature across ALL apps. Use Glob and Grep to be exhaustive — do not guess.

### Backend files
```
backend/internal/domain/{feature}/                    -> Domain entities, errors
backend/internal/port/repository/{feature}*.go        -> Repository interface(s)
backend/internal/port/service/{feature}*.go           -> Service interface(s) (if any)
backend/internal/app/{feature}/                       -> Application service
backend/internal/adapter/postgres/{feature}*.go       -> PostgreSQL adapter
backend/internal/adapter/{provider}/                  -> External adapters (stripe, resend, minio, etc.)
backend/internal/handler/{feature}*.go                -> HTTP handler(s)
backend/internal/handler/dto/request/{feature}.go     -> Request DTOs
backend/internal/handler/dto/response/{feature}.go    -> Response DTOs
backend/mock/{feature}*.go                            -> Generated mocks
backend/test/**/{feature}*                            -> Integration/E2E tests
```

### Web files
```
web/src/features/{feature}/                           -> Entire feature directory
web/src/app/(dashboard)/**/{feature}/                 -> Dashboard route pages
web/src/app/(public)/**/{feature}/                    -> Public route pages
```

### Mobile files
```
mobile/lib/features/{feature}/                        -> Entire feature directory
mobile/test/features/{feature}/                       -> Feature tests
mobile/test/unit/{feature}/                           -> Unit tests
mobile/test/widget/{feature}/                         -> Widget tests
```

### Admin files
```
admin/src/pages/{Feature}.tsx                         -> Page (PascalCase filename)
```

### Migration files (DO NOT DELETE — inventory only)
```
backend/migrations/*_{feature}*.up.sql
backend/migrations/*_{feature}*.down.sql
```

### Wiring references
```
backend/cmd/api/main.go                              -> DI lines for this feature
backend/internal/handler/router.go                    -> Route registrations
web/src/shared/components/layouts/                    -> Sidebar navigation links
mobile/lib/core/router/app_router.dart                -> Router definitions
admin/src/App.tsx                                     -> Route definitions
admin/src/components/layouts/                         -> Sidebar navigation links
```

---

## STEP 2 — Check for external dependencies

Before deleting anything, verify no other feature depends on this one.

### Backend: search for imports of this feature's packages

Use Grep to find imports matching `domain/{feature}`, `app/{feature}`, `port/repository/{feature}`, `port/service/{feature}`, `adapter/*/{feature}` in files OUTSIDE the feature's own directories.

**Allowed references** (will be cleaned up in Step 3):
- `cmd/api/main.go` — wiring
- `internal/handler/router.go` — routes
- `internal/handler/{feature}*.go` — the feature's own handler

**Forbidden references:**
- Another feature's app service importing this feature's domain or service
- Another feature's handler importing this feature's types
- Another adapter importing this feature's adapter
- A shared package depending on this feature

If forbidden references are found, **STOP** and list every violation. The dependency must be resolved before removal is possible.

### Web: search for cross-feature imports

Use Grep to search all files in `web/src/features/` for imports from `@/features/{feature}/`. Any match in a DIFFERENT feature directory means a dependency violation — **STOP** and report.

### Mobile: search for cross-feature imports

Use Grep to search all files in `mobile/lib/features/` for imports referencing `features/{feature}/`. Any match in a DIFFERENT feature directory means **STOP**.

### Database: check for cross-feature foreign keys

Read all migration `.up.sql` files. If any OTHER feature's migration table has a `REFERENCES` to a table belonging to this feature (excluding the `users` table), **STOP** and report. The FK must be removed first.

---

## STEP 3 — Clean wiring (before deletion)

### 3a. Update `backend/cmd/api/main.go`
Remove:
- Repository instantiation: `{feature}Repo := postgres.New{Feature}Repository(db)`
- Service instantiation: `{feature}Svc := {feature}.NewService(...)`
- Handler instantiation: `{feature}Handler := handler.New{Feature}Handler({feature}Svc)`
- The feature's entry in the `handler.RouterDeps{}` struct
- Related import lines

### 3b. Update `backend/internal/handler/router.go`
Remove:
- The feature's field from `RouterDeps` struct
- The feature's route group / route registrations inside `NewRouter()`
- Related import lines

### 3c. Update web sidebar navigation
Search in `web/src/shared/components/layouts/` for navigation items referencing this feature. Remove the sidebar link/menu item.

### 3d. Update mobile router
Edit `mobile/lib/core/router/app_router.dart`:
- Remove route definitions for this feature's screens
- Remove the feature's tab from the bottom navigation (if applicable)
- Remove related import lines

### 3e. Update admin routing and navigation
Edit `admin/src/App.tsx`:
- Remove the `<Route>` for this feature's page
- Remove import of the page component

Search in `admin/src/components/layouts/` for sidebar links to this feature's page. Remove them.

---

## STEP 4 — Delete feature files

Delete in this order (dependencies first, dependents last):

### Backend
1. `backend/internal/handler/dto/request/{feature}.go`
2. `backend/internal/handler/dto/response/{feature}.go` (if separate file)
3. `backend/internal/handler/{feature}*.go`
4. `backend/internal/app/{feature}/` (entire directory)
5. `backend/internal/adapter/postgres/{feature}*.go`
6. `backend/internal/adapter/{provider}/` (feature-specific external adapters)
7. `backend/internal/port/repository/{feature}*.go`
8. `backend/internal/port/service/{feature}*.go` (if exists)
9. `backend/internal/domain/{feature}/` (entire directory)
10. `backend/mock/{feature}*.go` (generated mocks)
11. `backend/test/**/{feature}*` (integration tests)

### Web
12. `web/src/features/{feature}/` (entire directory)
13. `web/src/app/(dashboard)/**/{feature}/` (route pages)
14. `web/src/app/(public)/**/{feature}/` (route pages)

### Mobile
15. `mobile/lib/features/{feature}/` (entire directory)
16. `mobile/test/features/{feature}/` (tests)
17. `mobile/test/unit/{feature}/` (if exists)
18. `mobile/test/widget/{feature}/` (if exists)

### Admin
19. `admin/src/pages/{Feature}.tsx` (PascalCase)

### Migrations — NEVER DELETE
**CRITICAL: Never delete migration files.** Data schema changes require separate down migrations. Note which migrations belong to this feature so the user can run the down migration:
```
To drop this feature's tables, apply the down migration:
  backend/migrations/NNN_{feature_table}.down.sql
```

---

## STEP 5 — Compile and verify

### Backend
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend && go build ./...
```
Fix any compilation errors. Common issues:
- Unused imports in main.go or router.go — remove them
- Missing field in RouterDeps — update the struct
- Stale references in other handler files — clean up

### Web (if node_modules exist)
```bash
cd /home/hassad/Documents/marketplaceServiceGo/web && [ -d node_modules ] && npx tsc --noEmit
```
Fix any TypeScript errors. Common issues:
- Page importing from deleted feature — delete the page too
- Sidebar component referencing deleted feature type — remove the link

### Mobile (if flutter is available)
```bash
cd /home/hassad/Documents/marketplaceServiceGo/mobile && command -v flutter >/dev/null 2>&1 && flutter analyze
```
Fix any Dart analysis errors. Common issues:
- Router referencing deleted screen — remove the route
- Import of deleted feature — clean up

### Admin (if node_modules exist)
```bash
cd /home/hassad/Documents/marketplaceServiceGo/admin && [ -d node_modules ] && npx tsc --noEmit
```

---

## STEP 6 — Run tests

```bash
# Backend
cd /home/hassad/Documents/marketplaceServiceGo/backend && go test ./... -count=1 -timeout 120s

# Web (if deps installed)
cd /home/hassad/Documents/marketplaceServiceGo/web && [ -d node_modules ] && npx vitest run 2>/dev/null

# Mobile (if flutter available)
cd /home/hassad/Documents/marketplaceServiceGo/mobile && command -v flutter >/dev/null 2>&1 && flutter test

# Admin (if deps installed)
cd /home/hassad/Documents/marketplaceServiceGo/admin && [ -d node_modules ] && npx vitest run 2>/dev/null
```

All remaining tests must pass. If a test outside this feature fails, there was a hidden dependency — fix it.

---

## STEP 7 — Final verification checklist

Run through each item mentally:
- [ ] Backend compiles with zero errors (`go build ./...`)
- [ ] Web compiles with zero errors (`tsc --noEmit`)
- [ ] Mobile analyzes cleanly (`flutter analyze`)
- [ ] Admin compiles with zero errors (`tsc --noEmit`)
- [ ] No dangling imports referencing the removed feature in any app
- [ ] No orphan route pages pointing to deleted components
- [ ] `cmd/api/main.go` and `router.go` are clean
- [ ] Web sidebar has no dead link for this feature
- [ ] Mobile router has no dead route for this feature
- [ ] Admin routing has no dead route for this feature
- [ ] All remaining tests pass across all apps
- [ ] Migration files are preserved (not deleted)

---

## Output

Report:

```
# Removed feature: {feature}

## Deleted files (grouped by app)

### Backend (X files)
  backend/internal/domain/{feature}/         (N files)
  backend/internal/port/repository/{feature}.go
  backend/internal/app/{feature}/            (N files)
  backend/internal/adapter/postgres/{feature}.go
  backend/internal/handler/{feature}.go
  backend/internal/handler/dto/request/{feature}.go
  backend/internal/handler/dto/response/{feature}.go
  backend/mock/{feature}*.go

### Web (X files)
  web/src/features/{feature}/                (N files)
  web/src/app/(dashboard)/{role}/{feature}/  (N files)

### Mobile (X files)
  mobile/lib/features/{feature}/             (N files)
  mobile/test/features/{feature}/            (N files)

### Admin (X files)
  admin/src/pages/{Feature}.tsx

## Modified files
  backend/cmd/api/main.go           — removed {feature} wiring
  backend/internal/handler/router.go — removed {feature} routes
  web/src/shared/components/layouts/ — removed sidebar link
  mobile/lib/core/router/           — removed route
  admin/src/App.tsx                  — removed route

## Migrations to revert (DO NOT DELETE — run down migration)
  backend/migrations/NNN_create_{feature_table}.down.sql

## Dependency issues found
  None (feature was fully independent)
  — OR —
  BLOCKED: {other_feature} imports {feature}. See violations listed above.

## Compilation
  Backend: OK
  Web: OK (or SKIPPED — node_modules not installed)
  Mobile: OK (or SKIPPED — flutter not available)
  Admin: OK (or SKIPPED — node_modules not installed)

## Tests
  Backend: X passed, 0 failed
  Web: X passed, 0 failed (or SKIPPED)
  Mobile: X passed, 0 failed (or SKIPPED)
  Admin: X passed, 0 failed (or SKIPPED)

## Manual steps required
  - Run `backend/migrations/NNN_create_{feature_table}.down.sql` to drop the database tables
  - (any other manual steps)
```
