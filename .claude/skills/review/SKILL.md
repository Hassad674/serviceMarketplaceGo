---
name: review
description: Comprehensive code quality, security, performance, and architecture audit across all apps (backend Go, web Next.js, mobile Flutter, admin Vite+React). Use before committing, before merging, or to audit existing code. Specify scope as backend, web, mobile, admin, all, a feature name, or a file path.
user-invocable: true
allowed-tools: Read, Bash, Grep, Glob, Agent
---

# Review

Review target: **$ARGUMENTS**

You are the code reviewer for the B2B Marketplace. This is a multi-app project:

- **backend/** — Go 1.25, Chi v5, hexagonal architecture
- **web/** — Next.js 16, React 19, Tailwind 4, feature-based
- **mobile/** — Flutter 3.16+, Riverpod 2, Clean Architecture
- **admin/** — Vite + React 19, page-based

Check every aspect below and produce an actionable report.

---

## STEP 0 — Determine review scope

### If `$ARGUMENTS` is empty — review uncommitted changes:
```bash
cd /home/hassad/Documents/marketplaceServiceGo
git diff --name-only HEAD
git diff --name-only --staged
```
Read every changed file. Determine which app(s) they belong to and apply the relevant checks.

### If `$ARGUMENTS` is a scope keyword (`backend`, `web`, `mobile`, `admin`, `all`):
- `backend` — review all files in `backend/internal/` and `backend/cmd/`
- `web` — review all files in `web/src/`
- `mobile` — review all files in `mobile/lib/`
- `admin` — review all files in `admin/src/`
- `all` — review all four apps

For each app, focus on recently modified files first:
```bash
cd /home/hassad/Documents/marketplaceServiceGo
git log --oneline -20 --name-only -- {app_dir}
```

### If `$ARGUMENTS` is a feature name (e.g., "mission", "invoice", "review"):
Read all files belonging to that feature across ALL apps:
- `backend/internal/domain/{feature}/`
- `backend/internal/port/repository/{feature}*.go`
- `backend/internal/port/service/{feature}*.go`
- `backend/internal/app/{feature}/`
- `backend/internal/adapter/postgres/{feature}*.go`
- `backend/internal/handler/{feature}*.go`
- `backend/internal/handler/dto/request/{feature}.go`
- `backend/internal/handler/dto/response/{feature}.go`
- `web/src/features/{feature}/`
- `mobile/lib/features/{feature}/`
- `admin/src/pages/{Feature}.tsx` (PascalCase)

### If `$ARGUMENTS` is a file path:
Read that specific file and its related files (test file, interface, DTO, etc.).

### If `$ARGUMENTS` is a commit range (e.g., "HEAD~3..HEAD"):
```bash
git diff $ARGUMENTS --name-only
```
Read all files in the diff.

---

## REVIEW 1 — Security

### 1a. SQL injection (backend)
Search for string concatenation in SQL queries:
- `fmt.Sprintf` near SQL keywords (`SELECT`, `INSERT`, `UPDATE`, `DELETE`, `WHERE`)
- String `+` concatenation building queries
- Template literals in queries

**CRITICAL if found.** All queries must use `$1, $2, $3` parameterized placeholders.

### 1b. XSS (web/admin)
- No `dangerouslySetInnerHTML` without DOMPurify or equivalent sanitization
- No `eval()` or `new Function()` with user input
- User-generated content properly escaped

### 1c. Authentication and authorization (backend)
- All protected endpoints behind `middleware.Auth(tokenService)`
- Role-restricted endpoints use `middleware.RequireRole("role")`
- Endpoints that modify user data verify ownership (user can only edit their own resources)
- Admin-only endpoints check admin role

### 1d. Input validation (all)
For every handler/endpoint that accepts user input:
- Backend: request body decoded and validated via DTO `Validate()` before use
- Web: forms validated with zod schema before submission
- Mobile: input validated before API call
- Are required fields checked? Are string lengths bounded? Are IDs validated as UUID?

### 1e. Secrets exposure (all)
- No hardcoded API keys, passwords, tokens, JWT secrets in source code
- No secrets in error messages returned to clients
- Sensitive fields (password_hash, tokens) not included in API responses
- No `.env` files committed (check `.gitignore`)
- Backend: verify `HandleDomainError` never leaks internal details in production

### 1f. CORS (backend)
- Verify `middleware.CORS()` uses explicit allow-list from config, not wildcard `*`
- Check `AllowedOrigins` in config — must be specific domains, not `*`

### 1g. Dependencies (all)
```bash
# Backend
cd /home/hassad/Documents/marketplaceServiceGo/backend && go list -m all | head -30

# Web
cd /home/hassad/Documents/marketplaceServiceGo/web && npm audit --audit-level=high 2>/dev/null | tail -20

# Admin
cd /home/hassad/Documents/marketplaceServiceGo/admin && npm audit --audit-level=high 2>/dev/null | tail -20

# Mobile
cd /home/hassad/Documents/marketplaceServiceGo/mobile && flutter pub outdated 2>/dev/null | head -20
```
Flag any known high/critical vulnerabilities.

---

## REVIEW 2 — Performance

### 2a. N+1 queries (backend)
Look for patterns where a list is fetched, then each item triggers another query:
```go
// BAD: N+1
items, _ := repo.List(ctx, offset, limit)
for _, item := range items {
    related, _ := otherRepo.FindByItemID(ctx, item.ID)  // N extra queries!
}
```
Suggest JOINs or batch queries instead.

### 2b. Missing database indexes (backend)
For every `WHERE` clause and `JOIN` condition in adapter SQL, verify a corresponding index exists in migrations.
Check that all foreign key columns are indexed.

### 2c. Unbounded queries (backend)
- Every `SELECT` that returns multiple rows must have `LIMIT`
- Pagination must be enforced (no "fetch all" endpoints)
- Large text fields should not be included in list queries if not needed

### 2d. Context timeouts (backend)
- All database calls should use `context.Context` with a timeout
- Long-running operations should have explicit timeout context:
```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

### 2e. Bundle size (web/admin)
- Check for heavy imports in client components (moment.js, lodash full bundle)
- Verify route-level code splitting is working (no giant shared bundles)
- No unnecessary `"use client"` — check if Server Component would work (web only)
- No `useEffect` fetching data that could be a Server Action or RSC (web only)

### 2f. Core Web Vitals (web)
- All images use `next/image` with explicit `width` and `height` (no layout shift)
- No raw `<img>` tags
- Lazy load below-the-fold content
- Fonts loaded via `next/font` with `font-display: swap`

### 2g. Widget rebuilds (mobile)
- Check for unnecessary rebuilds: `ConsumerWidget` watching too many providers
- Verify `const` constructors used where possible
- No heavy computation in `build()` methods — should be in providers/use cases
- Large lists use `ListView.builder`, not `Column(children: list.map(...))`

---

## REVIEW 3 — Architecture compliance

### 3a. Dependency direction (backend)
The absolute rule: `handler -> app -> domain <- port <- adapter`
- **domain/** imports NOTHING except Go stdlib — search for external imports
- **port/** imports only domain/
- **app/** imports domain/ and port/ (interfaces only), never adapter/
- **adapter/** never imports another adapter
- **handler/** imports app/ and dto/, never domain/ directly for responses

```bash
# Verify domain purity
cd /home/hassad/Documents/marketplaceServiceGo/backend
grep -r '"marketplace-backend/internal/' internal/domain/ --include="*.go" | grep -v "_test.go"
```
Any match here is a CRITICAL violation.

### 3b. Feature isolation (all)
- Backend: no cross-feature imports (one feature's app/ importing another feature's domain/)
- Web: no imports from `@/features/X/` inside `src/features/Y/` — composition happens in `app/` pages only
- Mobile: no imports from one feature into another — share via `core/` only
- Admin: pages are self-contained, no cross-page imports

```bash
# Web cross-feature imports
cd /home/hassad/Documents/marketplaceServiceGo/web
grep -rn "from.*@/features/" src/features/ --include="*.ts" --include="*.tsx" | grep -v "from.*@/features/\($(basename $(dirname $0))\)"
```

### 3c. Handler discipline (backend)
- Handlers are thin: decode request, validate, call service, encode response
- No business logic in handlers (no conditionals on domain state, no direct DB calls)
- No HTTP status codes in app layer
- No domain errors swallowed silently

### 3d. DTO discipline (backend)
- Handlers never pass domain entities directly to JSON encoder
- Response DTOs strip sensitive fields (password_hash, internal IDs)
- Request DTOs only contain fields needed for that operation
- Response uses `response.UserFromDomain(u)` or equivalent mapper

### 3e. Layer violations (mobile)
- `presentation/` never imports from `data/` directly
- `domain/` has zero external package imports (pure Dart only)
- `data/` implements domain repository interfaces
- Providers depend on domain use cases, not data layer directly

### 3f. Error handling chain (backend)
- Domain defines typed sentinel errors
- App layer returns domain errors, wraps with context: `fmt.Errorf("doing X: %w", err)`
- Handler maps domain errors to HTTP via `response.HandleDomainError(w, err)`
- No error swallowed silently (no `_ = someFunc()` on important operations)

---

## REVIEW 4 — Code quality

### 4a. TypeScript strict mode (web/admin)
- No `any` type — suggest the correct type or `unknown` with narrowing
- No unsafe `as` casts without a comment explaining why
- No `// @ts-ignore` or `// @ts-expect-error` without justification
- Use `type` imports for type-only imports

### 4b. Go idioms (backend)
- Error handling: `if err != nil` immediately after the call
- No naked returns in functions with named return values
- No `panic()` in library code (only in main for fatal startup)
- Exported functions have a doc comment
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`

### 4c. Dart idioms (mobile)
- Use `final` for all variables that are not reassigned
- Prefer `const` constructors for immutable widgets
- No `print()` in production code — use a logger
- Null safety: no unnecessary `!` (bang operator)

### 4d. Naming conventions
- **Go**: PascalCase exports, camelCase locals, snake_case files
- **TypeScript**: PascalCase components, camelCase functions, kebab-case files
- **Dart**: PascalCase classes, camelCase functions, snake_case files
- **Database**: snake_case tables and columns
- Clear descriptive names — no `x`, `tmp`, `data`, `info` (unless obvious from context)

### 4e. File and function size (all)
- No file exceeds 600 lines
- No function/method exceeds 50 lines
- Component JSX does not exceed 200 lines (web/admin)
- Component props do not exceed 4 (web/admin) — use composition or config object

### 4f. Dead code (all)
- Unused variables, imports, functions
- Commented-out code blocks
- Unreachable code after return/panic/throw

### 4g. Duplication (all)
- Same logic repeated 3+ times should be extracted
- Copy-pasted blocks exceeding 10 lines
- Identical SQL queries in different adapter methods (backend)
- Similar components that should share a base (web/admin)

---

## REVIEW 5 — Test quality

### 5a. Coverage gaps
- Backend: every `service.go` needs `service_test.go`, every `entity.go` needs `entity_test.go`
- Web: key components and hooks should have `.test.ts` / `.test.tsx`
- Mobile: domain entities and use cases should have tests
- Identify untested critical paths (auth flows, payment logic, permission checks)

### 5b. Test quality
- Tests assert meaningful behavior (not just "no panic" or "renders without error")
- Edge cases covered: empty input, nil/null, not found, duplicate, unauthorized
- Tests are independent — no shared mutable state between tests
- Go: table-driven tests for multiple scenarios
- No `t.Skip()` or `.skip()` without explanation

### 5c. Test naming
- Go: `TestServiceName_MethodName_Scenario`
- TypeScript: `describe('ComponentName')` -> `it('should do X when Y')`
- Dart: `group('ClassName')` -> `test('should do X when Y')`

### 5d. Mock quality
- Mocks match real behavior (same error types, same return shapes)
- No mocks that always succeed — test failure paths too
- Backend: mocks implement port interfaces faithfully

---

## Report format

```
# Code Review — {target}

## Summary
- Critical: X issues
- Warning: X issues
- Info: X suggestions
- Clean: X checks passed

## Critical

### [CRITICAL] {category} in {file}:{line}
```{language}
{offending code}
```
**Fix:** {specific remediation}

## Warnings

### [WARN] {category} in {file}:{line}
{description of the issue}
**Fix:** {specific remediation}

## Info

### [INFO] {category} in {file}:{line}
{description}

## Clean

- [OK] No hardcoded secrets found across all apps
- [OK] All backend handlers validate input via DTOs
- [OK] Domain layer has zero external imports
- [OK] No cross-feature imports in web/
- [OK] Response DTOs strip sensitive fields
- [OK] All queries use parameterized placeholders
```

### Severity levels:
- **CRITICAL** — Security vulnerability or data loss risk. Must fix before merge.
- **WARNING** — Performance issue, missing test, or architecture smell. Should fix.
- **INFO** — Style, naming, or minor improvement. Nice to fix.
- **CLEAN** — Explicitly confirm checks that passed (builds confidence).
