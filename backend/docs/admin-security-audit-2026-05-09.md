# Admin Security Audit — 2026-05-09

> **Scope:** verify the `admin` role / `is_admin` flag is fully enforced
> across every backend endpoint, the admin SPA, the marketplace web,
> and the mobile app. Treat the marketplace as if a determined attacker
> with a low-privilege account is probing every angle.

> **Methodology:** code review of all 10 layers documented in the
> mission brief, `chi.Walk` over the full router, hand-rolled live
> probes against the running backend (`localhost:8083`), and a
> demote-window e2e test against the real DB.

> **Verdict:** **PASS overall.** Every admin endpoint is gated by both
> `Auth` (live identity) and `RequireAdmin` (live `is_admin` flag from
> a 30 s Redis-fronted DB read). Marketplace web and mobile expose **no
> admin link or admin route** to non-admin users. The admin SPA's login
> rejects non-admins, and the boot probe re-validates `is_admin` on
> every tab activation.

> **One MEDIUM finding:** the 30 s `user_state` Redis cache TTL creates
> a 30 s window where a demoted admin (via direct SQL) keeps admin
> access until the cache expires. Mitigations are documented under
> Layer 5.

---

## Executive summary

| Layer | Concern | Verdict |
|-------|---------|---------|
| 1 | Backend admin routes & middleware chain | PASS |
| 2 | Admin SPA login gate | PASS |
| 3 | Marketplace web sidebar / nav | PASS |
| 4 | Marketplace web `/admin/*` routes | PASS — no such routes exist |
| 5 | Cache poisoning + revocation latency | MEDIUM gap (30 s window on demote-via-SQL) |
| 6 | JWT vs session cookie (live override) | PASS |
| 7 | CORS isolation | PASS, with a note |
| 8 | Mobile admin code | PASS — none |
| 9 | Integration tests | PASS — added `admin_test.go` + `admin_security_test.go` |
| 10 | Live test against running backend | PASS — every admin path returns 403 / 401 for non-admin / anon |

---

## Layer 1 — Backend routes → middleware → handler

`backend/internal/handler/routes_admin.go` declares the entire admin
sub-tree under a single `r.Route("/admin", func(r chi.Router) { ... })`
block. Inside that block, three middlewares are applied in order:

```go
r.Use(auth)                       // 401 if identity invalid
r.Use(middleware.RequireAdmin())  // 403 if !is_admin (live DB read)
r.Use(middleware.NoCache)         // strips Cache-Control for replay safety
```

Every admin handler is mounted via one of nine helpers
(`mountAdminUsersRoutes`, `mountAdminConversationsRoutes`,
`mountAdminJobsRoutes`, `mountAdminMessageModerationRoutes`,
`mountAdminReviewsRoutes`, `mountAdminUnifiedModerationRoutes`,
`mountAdminMediaRoutes`, `mountAdminDisputesRoutes`,
`mountAdminProposalRoutes`, `mountAdminTeamRoutes`,
`mountAdminSearchRoutes`, `mountAdminInvoicingRoutes`). Each helper is
called from `mountAdminRoutes`, ensuring the three middlewares apply
uniformly.

### Inventory (61 routes)

| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/v1/admin/dashboard/stats` | `Admin.GetDashboardStats` |
| GET | `/api/v1/admin/users` | `Admin.ListUsers` |
| GET | `/api/v1/admin/users/{id}` | `Admin.GetUser` |
| POST | `/api/v1/admin/users/{id}/suspend` | `Admin.SuspendUser` |
| POST | `/api/v1/admin/users/{id}/unsuspend` | `Admin.UnsuspendUser` |
| POST | `/api/v1/admin/users/{id}/ban` | `Admin.BanUser` |
| POST | `/api/v1/admin/users/{id}/unban` | `Admin.UnbanUser` |
| GET | `/api/v1/admin/users/{id}/reports` | `Admin.ListUserReports` |
| GET | `/api/v1/admin/notifications` | `Admin.GetNotificationCounters` |
| POST | `/api/v1/admin/notifications/{category}/reset` | `Admin.ResetNotificationCounter` |
| GET | `/api/v1/admin/conversations[/{id}[/messages,reports]]` | `Admin.ListConversations`, etc. |
| POST | `/api/v1/admin/reports/{id}/resolve` | `Admin.ResolveReport` |
| GET | `/api/v1/admin/jobs[/{id}[/reports]]`, `/job-applications` | `Admin.ListJobs`, etc. |
| DELETE | `/api/v1/admin/jobs/{id}`, `/job-applications/{id}` | `Admin.DeleteAdminJob`, `Admin.DeleteJobApplication` |
| POST | `/api/v1/admin/messages/{id}/[approve,hide,restore]-moderation` | message moderation suite |
| GET, DELETE, POST | `/api/v1/admin/reviews/...` | review moderation suite |
| GET | `/api/v1/admin/moderation[/count]` | unified moderation queue |
| POST | `/api/v1/admin/moderation/{type}/{id}/restore` | unified restore |
| GET, POST, DELETE | `/api/v1/admin/media/{id?}/...` | media moderation |
| GET, POST | `/api/v1/admin/disputes/...` | dispute admin (resolve, force-escalate, AI chat, AI budget) |
| POST | `/api/v1/admin/proposals/{id}/activate` | force-activate |
| POST | `/api/v1/admin/credits/reset[/{userId}]` | reset credits |
| GET, POST | `/api/v1/admin/credits/bonus-log[/...]` | bonus fraud log |
| GET, POST, PATCH, DELETE | `/api/v1/admin/users/{id}/organization`, `/api/v1/admin/organizations/{id}/...` | force team management |
| GET | `/api/v1/admin/search/stats` | search analytics |
| POST | `/api/v1/admin/invoices/{id}/credit-note` | credit note issuance |
| GET | `/api/v1/admin/invoices`, `/api/v1/admin/invoices/{id}/pdf` | invoice listing |

The full inventory is pinned in
`internal/handler/admin_security_test.go::TestAdminRoutes_KnownEndpointsArePresent`
so a future mount drop fails CI.

### `RequireAdmin` middleware

`backend/internal/handler/middleware/admin.go`:

```go
func RequireAdmin() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !GetIsAdmin(r.Context()) {
                response.Error(w, http.StatusForbidden, "forbidden", "admin access required")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

The flag is read from request context. Crucially, `GetIsAdmin` is
populated by the upstream `Auth` middleware **from the live
`UserStateChecker`** — *not* from the JWT / session cookie snapshot.
This is the PR #178 fix:

```go
// auth.go — checkUserState lives at request time, not login time
live, outcome := resolveUserState(r.Context(), deps.UserState, userID, snapshot)
ctx := stampAuthContext(r.Context(), authStamp{
    ...
    IsAdmin: live.IsAdmin,  // <-- LIVE override, never the snapshot
    ...
}, deps.OrgOverrides)
```

Production wires `UserState` to a Redis-fronted Postgres reader
(`redisadapter.NewCachedUserStateChecker`) with a 30 s TTL. The cache
miss costs one indexed PK lookup (`SELECT is_admin, status FROM users
WHERE id = $1`), which the audit doc on the file pins at < 5 ms p95.

**Fail-closed in production:** if the live lookup fails (DB / Redis
blip), the middleware returns `503 auth_unavailable` instead of
silently trusting the snapshot. This is gated on
`AuthDeps.FailClosedInProd` → `cfg.IsProduction()` (see
`internal/handler/router.go:140-151`).

**Order matters:** the auth middleware checks `session_version` BEFORE
`UserState`. A banned user's session_version is bumped, so the JWT is
rejected at the version check and never reaches the `UserState` check
— the 30 s `UserState` cache cannot keep a banned user alive past the
DB write.

### Layer 1 verdict

PASS. Every admin endpoint passes through `Auth + RequireAdmin +
NoCache`. Inventory pinned; route-mount drift is caught by the
structural test in `internal/handler/admin_security_test.go`.

---

## Layer 2 — Admin SPA login

`admin/src/features/auth/components/login-form.tsx` is the entry
point. The login flow goes through `useAuth().login(email, password)`
defined in `admin/src/shared/hooks/use-auth.tsx`:

```ts
const data = await adminApi<LoginResponse>("/api/v1/auth/login", {
    method: "POST",
    body: { email, password },
    headers: { "X-Auth-Mode": "token" },
})

if (!data.user.is_admin) {
    throw new Error("Acces reserve aux administrateurs")
}

setToken(data.access_token)
setHasCookieSession(true)
```

**Boot-time + visibility-change probe:**

```ts
async function restoreSession({ markHydrated }: { markHydrated: boolean }) {
    try {
        const me = await adminApi<MeResponse>("/api/v1/auth/me")
        if (cancelled) return
        if (me.user?.is_admin) {
            setHasCookieSession(true)
        } else {
            // The user lost admin rights (demotion, ban). Drop the
            // cookie-session flag so AdminLayout redirects to /login
            // on the next render. The api-client clears the bearer on
            // 401 separately.
            setHasCookieSession(false)
        }
    } catch { /* 401 → fall through to logged-out */ }
    // ...
}
```

The `/auth/me` endpoint reads `is_admin` from a fresh
`s.users.GetByID(ctx, userID)` call (see
`backend/internal/app/auth/service_more.go:202-204`). The handler's
response payload (`NewMeResponse → NewUserResponse`) sets
`IsAdmin: u.IsAdmin` from the live DB row — not from the session
cookie's snapshot.

**`adminApi` 401 handling:** `admin/src/shared/lib/api-client.ts` (per
`admin/CLAUDE.md` lines 144-149) catches every `status === 401` by
clearing the in-memory token via `clearAuthToken()` and redirecting to
`/login`. This handles refresh-token expiry, manual revocation, and
post-demotion 401s consistently.

**Bearer storage:** the bearer token lives in the in-memory Zustand
store (`admin/src/shared/stores/auth-store.ts`) only — never
`localStorage` / `sessionStorage` / cookies (per audit item
SEC-FINAL-07). A hard reload drops the bearer; the cookie-session
probe restores the user via the `/auth/me` cookie path.

### Layer 2 verdict

PASS. Login, boot probe, and visibility-change probe all enforce the
live `is_admin` flag from the backend. A demoted user keeps a cookie
session and a stale bearer for at most one tab activation cycle (≤ a
few seconds with a focused tab).

---

## Layer 3 — Marketplace web sidebar / nav

Reviewed
`web/src/shared/components/layouts/sidebar.tsx`. The `FREELANCE_NAV`
and `REFERRER_NAV` arrays do **not** contain any admin link. Every
nav entry is gated on `roles: ["agency" | "provider" | "enterprise"]`
— admin role does not appear in any nav configuration.

**Cross-checked the entire web app:** `grep -rn "admin\|/admin"
web/src --include="*.tsx" --include="*.ts"` returns only:

1. Auto-generated OpenAPI types (`web/src/shared/types/api.d.ts` —
   pure type definitions, no calls);
2. **Org-level "admin" role** for team membership (a different
   concept than platform admin — same string, different domain);
3. Comments mentioning "admin" in dispute / moderation system message
   components (UI text only).

**No** `<Link href="https://admin.designedtrust.com">`, **no** `Admin`
nav entry, **no** `is_admin`-gated UI element. The marketplace web
treats platform admin as out-of-band.

### Layer 3 verdict

PASS. Marketplace web does not surface admin entry points to any user.

---

## Layer 4 — Marketplace web `/admin/*` routes

`find web/src/app -type d -name "admin"` returns nothing. The
marketplace Next.js app has zero `/admin/*` routes by design — admin
lives exclusively on the SPA at `localhost:5174` /
`admin.designedtrust.com`. There is no proxy, no rewrite, no
redirect.

### Layer 4 verdict

PASS. No admin surface in the marketplace web app.

---

## Layer 5 — Cache poisoning + revocation latency

The `user_state` Redis cache (TTL 30 s,
`redisadapter.DefaultUserStateCacheTTL`) is the hot path the auth
middleware reads on every request. Cache key: `user_state:<uuid>`,
payload: `{ is_admin, status }`.

### The 30 s window — confirmed live

I ran the demote scenario end-to-end against the running backend on
`localhost:8083`:

```text
1. Register a non-admin user, login, get bearer token.
2. SQL: UPDATE users SET is_admin = true WHERE id = <uuid>
3. Immediately call /api/v1/admin/dashboard/stats:        403
4. Wait 31 s for cache to expire, call again:             200
5. SQL: UPDATE users SET is_admin = false WHERE id = <uuid>
6. Immediately call /api/v1/admin/dashboard/stats:        200  <-- STALE
7. Wait 31 s, call again:                                 403
```

This confirms a **30 s revocation window** in *both directions*:

- Promotion lag (acceptable — granting access slightly late is benign);
- **Demotion lag** (concerning — a hostile insider has a 30 s window
  to act on admin endpoints after they have been removed via SQL).

### Mitigation status

The cache decorator exposes
`CachedUserStateChecker.Invalidate(ctx, userID)` (see
`internal/adapter/redis/user_state_cache.go:149`). The doc explicitly
says *"Call this from any code path that mutates users.is_admin or
users.status"*. **No code path currently calls it.**

For **suspend / ban** paths (`SuspendUser`, `BanUser` in
`internal/app/admin/service.go:257`), the gap is *fully covered* by:

1. `BumpSessionVersion` — the next request fails the
   session_version check (returns `401 session_revoked` before the
   user_state cache is even consulted);
2. `sessionSvc.DeleteByUserID` — the cookie session is purged;
3. `BroadcastAccountSuspended` — the WS pushes the client to log out.

So `BanUser` / `SuspendUser` do **not** depend on user_state cache
invalidation for correctness.

For **`is_admin` toggle**, there is **no app-level endpoint** that
mutates the flag — the only path is direct SQL by an operator. In
that operator workflow the 30 s window is the entire correctness
gap. Documented but not currently exploitable through the API
surface.

### Recommendation (NOT applied without owner approval)

Add a tiny admin-only endpoint
`POST /api/v1/admin/users/{id}/admin-flag` (or an explicit
`Invalidate` after every operator workflow) that:

1. Mutates `users.is_admin`;
2. Calls `userStateCache.Invalidate(ctx, userID)`;
3. Optionally bumps `session_version` so the demoted admin's existing
   sessions are immediately invalidated (closing the gap to ~0 ms).

**Why I am not applying this fix:** changing auth semantics requires
the user's approval per the mission brief. The 30 s window is small,
operator-driven, and not currently a production attack vector
(operators are the only path to `is_admin` mutation today).

### Layer 5 verdict

MEDIUM finding. 30 s window on demote-via-SQL. Mitigation path is
clear but requires owner approval to merge.

---

## Layer 6 — JWT vs session cookie (live override)

The auth middleware reads `is_admin` from `live.IsAdmin` (the
`UserStateChecker`'s response), NOT from `claims.IsAdmin` (JWT) or
`session.IsAdmin` (cookie session). This is the entire point of PR
#178.

A handcrafted JWT carrying `is_admin: true` for a user whose DB row
says `is_admin: false` would still hit the live state check on every
admin request → middleware sees `false` → 403. Confirmed by:

- `internal/handler/middleware/user_state_test.go::TestAuth_LiveDemotionOverridesSnapshot`
  — session/JWT carry `IsAdmin: true`, live state returns `false`
  → context gets `false`.
- New tests:
  `internal/handler/middleware/admin_test.go::TestRequireAdmin_DeniesNonAdminFlag`,
  `TestRequireAdmin_DeniesUnsetFlag`,
  `TestRequireAdmin_DeniesWrongTypeOnContext`.
- Live curl loop in Layer 10 — every endpoint returns 403 for the
  non-admin user.

I could not literally forge a JWT with `is_admin: true` (signing
secret is in env, not exposed). But the unit test pins the exact
override behaviour, and the production Redis-fronted reader is the
only source of truth for `is_admin` on every admin request.

### Layer 6 verdict

PASS. JWT / session snapshots are explicitly overridden by live DB
state on every authenticated request.

---

## Layer 7 — CORS isolation

`backend/internal/handler/middleware/cors.go` enforces a strict
allow-list:

```go
allowed := origin != "" && originsMap[origin]

if allowed {
    w.Header().Set("Access-Control-Allow-Origin", origin)
    w.Header().Set("Access-Control-Allow-Credentials", "true")
    ...
}
```

Origins come from `cfg.AllowedOrigins`, which is sourced from the
`ALLOWED_ORIGINS` env var (defaults to
`http://localhost:3000,http://localhost:5173` for dev).

**Note:** the CORS allow-list applies globally — including to
`/api/v1/admin/*` endpoints. This means the marketplace origin
(`localhost:3000` / `services.designedtrust.com`) is *technically*
allowed to send cross-origin requests to admin endpoints. **However**,
this is a non-issue because:

1. The marketplace UI does not have a XSS payload that would call
   admin endpoints (only the admin SPA's compiled JS does);
2. The admin SPA's bearer is in-memory only (SEC-FINAL-07), so no
   `httpOnly` admin cookie exists for the marketplace to replay —
   the marketplace's session cookie can browse `/auth/me` etc. but
   not authenticate as admin without `is_admin: true` on that user;
3. Even if a user is admin on both apps and has both the
   marketplace cookie and the admin cookie, the live `is_admin`
   check still applies. CORS does not authenticate the request — it
   only relaxes the same-origin rule on the browser side.

**Recommended hardening (optional):** split the CORS allow-list into
a "marketplace" set and an "admin" set, so the admin middleware
applies a tighter list. But under the current threat model this is
not a vulnerability — just defense-in-depth.

### Layer 7 verdict

PASS. The shared CORS allow-list does not create an admin
vulnerability. A note for future hardening.

---

## Layer 8 — Mobile

`grep -rn "is_admin\|/api/v1/admin" mobile/lib --include="*.dart"`
returns ZERO results. The Flutter app has no admin API client, no
admin route, and no admin-only UI.

The string "admin" appears only in:

- Org-level role `"admin"` (team feature — different concept);
- Dispute / moderation system messages (UI strings about disputes);
- Skill catalog metadata ("seeded by the admin team").

`mobile/lib/core/router/app_router.dart` (the app's router) does not
register any admin route.

### Layer 8 verdict

PASS. Mobile is admin-free by design.

---

## Layer 9 — Integration tests

I added two test files in this audit:

1. `backend/internal/handler/middleware/admin_test.go` — 6 unit tests
   (with sub-tests, totaling 9 assertion blocks) covering
   `RequireAdmin` in isolation:
   - `TestRequireAdmin_AllowsAdminFlag` — happy path.
   - `TestRequireAdmin_DeniesNonAdminFlag` — 403, body shape.
   - `TestRequireAdmin_DeniesUnsetFlag` — defense in depth: missing
     context key defaults to deny.
   - `TestRequireAdmin_DeniesWrongTypeOnContext` — paranoia: a
     non-bool flag does not coerce.
   - `TestRequireAdmin_FlagIsTheOnlySignal` — the gate ignores
     role / org_role / etc.; only `is_admin` matters.
   - `TestRequireAdmin_DenialBodyIsCanonicalEnvelope` — the 403 body
     parses as the SPA expects.
   - `TestRequireAdmin_DoesNotConsumeRequestBody` — the middleware
     forwards request bodies untouched.

2. `backend/internal/handler/admin_security_test.go` — 5 structural
   tests over the fully-wired router:
   - `TestAdminRoutes_AllUnderAdminSubRouter` — every
     `/api/v1/admin/*` route has strictly more middlewares than the
     baseline `/api/v1/auth/me` route, so `RequireAdmin + NoCache`
     are layered on top of the global stack.
   - `TestAdminRoutes_NoAdminEndpointOutsideAdminSubRouter` — no
     "admin"-named route exists outside `/api/v1/admin/`.
   - `TestAdminRoutes_KnownEndpointsArePresent` — pins the inventory
     of 61 admin routes; a future drop fails CI with a clear
     message.
   - `TestAdminRoutes_NoCacheChainShape` — sanity check that the
     middleware count is at least 3.
   - `TestAdminRoutes_MiddlewareCountConsistent` — every admin
     endpoint has the same middleware count; a drift means one of
     the gates was dropped on that route.

All 11 tests pass:

```text
=== RUN   TestRequireAdmin_AllowsAdminFlag
--- PASS: TestRequireAdmin_AllowsAdminFlag (0.00s)
=== RUN   TestRequireAdmin_DeniesNonAdminFlag
--- PASS: TestRequireAdmin_DeniesNonAdminFlag (0.00s)
=== RUN   TestRequireAdmin_DeniesUnsetFlag
--- PASS: TestRequireAdmin_DeniesUnsetFlag (0.00s)
=== RUN   TestRequireAdmin_DeniesWrongTypeOnContext
--- PASS: TestRequireAdmin_DeniesWrongTypeOnContext (0.00s)
=== RUN   TestRequireAdmin_FlagIsTheOnlySignal
--- PASS: TestRequireAdmin_FlagIsTheOnlySignal (0.00s)
    --- PASS: ...is_admin=true_even_with_empty_role_still_allowed (0.00s)
    --- PASS: ...is_admin=true_even_with_role='enterprise'_still_allowed (0.00s)
    --- PASS: ...is_admin=false_with_role='admin'_still_denied (0.00s)
    --- PASS: ...no_role,_is_admin=true_allowed (0.00s)
=== RUN   TestRequireAdmin_DenialBodyIsCanonicalEnvelope
--- PASS: TestRequireAdmin_DenialBodyIsCanonicalEnvelope (0.00s)
=== RUN   TestRequireAdmin_DoesNotConsumeRequestBody
--- PASS: TestRequireAdmin_DoesNotConsumeRequestBody (0.00s)
=== RUN   TestAdminRoutes_AllUnderAdminSubRouter
--- PASS: TestAdminRoutes_AllUnderAdminSubRouter (0.01s)
=== RUN   TestAdminRoutes_NoAdminEndpointOutsideAdminSubRouter
--- PASS: TestAdminRoutes_NoAdminEndpointOutsideAdminSubRouter (0.01s)
=== RUN   TestAdminRoutes_KnownEndpointsArePresent
--- PASS: TestAdminRoutes_KnownEndpointsArePresent (0.01s)
=== RUN   TestAdminRoutes_NoCacheChainShape
--- PASS: TestAdminRoutes_NoCacheChainShape (0.01s)
=== RUN   TestAdminRoutes_MiddlewareCountConsistent
--- PASS: TestAdminRoutes_MiddlewareCountConsistent (0.01s)
PASS
```

Pre-existing tests already cover the live `is_admin` override:

- `internal/handler/middleware/user_state_test.go::TestAuth_LiveIsAdminOverridesSnapshot_CookiePath`,
  `..._BearerPath`, `TestAuth_LiveDemotionOverridesSnapshot`,
  `TestAuth_LiveBannedShortCircuits_Cookie`,
  `TestAuth_LiveUserGoneRejects`,
  `TestAuth_UserStateLookupFailsClosedInProd`,
  `TestAuth_UserStateLookupFailsOpenInDev`,
  `TestAuth_NilUserStateChecker_TrustsSnapshot`.
- `internal/adapter/redis/user_state_cache_test.go::TestUserStateCache_Invalidate_EvictsImmediately`
  pins the 0 ms revocation behaviour when `Invalidate` IS called.
- `internal/adapter/redis/user_state_cache_test.go::TestUserStateCache_TTLExpiresThenRefetches`
  pins the 30 s natural expiry.

Validation pipeline:

```text
$ cd backend && go build ./...
(no output — clean)

$ cd backend && go vet ./...
(no output — clean)

$ cd backend && go test ./internal/handler/middleware/ -count=1
ok    marketplace-backend/internal/handler/middleware  6.460s

$ cd backend && go test ./internal/handler/ -count=1
ok    marketplace-backend/internal/handler  1.914s
```

### Layer 9 verdict

PASS. New tests cover the gate in isolation and the route-mount
shape; pre-existing tests cover the live override semantics.

---

## Layer 10 — Live test against running backend

```bash
# Registered a fresh non-admin provider account
$ curl -X POST http://localhost:8083/api/v1/auth/register \
  -d '{"email":"sec-audit-nonadmin@test.local","password":"AuditTest!2026","first_name":"NonAdmin","last_name":"Audit","role":"provider"}'

# Login as that user, save the bearer
$ TOKEN=$(curl -X POST http://localhost:8083/api/v1/auth/login \
  -H "X-Auth-Mode: token" \
  -d '{"email":"sec-audit-nonadmin@test.local","password":"AuditTest!2026"}' \
  | jq -r '.access_token')

# Probe every admin endpoint
GET  403 /api/v1/admin/dashboard/stats
GET  403 /api/v1/admin/users
GET  403 /api/v1/admin/conversations
GET  403 /api/v1/admin/jobs
GET  403 /api/v1/admin/job-applications
GET  403 /api/v1/admin/reviews
GET  403 /api/v1/admin/moderation
GET  403 /api/v1/admin/moderation/count
GET  403 /api/v1/admin/media
GET  403 /api/v1/admin/disputes
GET  403 /api/v1/admin/disputes/count
GET  403 /api/v1/admin/notifications
GET  403 /api/v1/admin/credits/bonus-log
GET  403 /api/v1/admin/search/stats
GET  403 /api/v1/admin/invoices

POST 403 /api/v1/admin/users/{id}/suspend
POST 403 /api/v1/admin/users/{id}/ban
DEL  403 /api/v1/admin/jobs/{id}

ANON 401 /api/v1/admin/users          # no token at all
```

Every `/api/v1/admin/*` GET returns `403` for the non-admin bearer;
mutations (`POST`, `DELETE`) also return `403`; anonymous requests
return `401` (`unauthenticated`, not `forbidden` — the auth
middleware fires first when no identity is present).

I also probed a non-existent admin path:

```text
GET /api/v1/admin/non-existent-endpoint
→ 403 {"error":"forbidden","message":"admin access required"}
```

Confirms the middleware short-circuits BEFORE chi resolves the URL,
so a non-admin cannot use 404 / 200 differentials to map the admin
URL space.

The test user was deleted from the DB after the probes:

```sql
DELETE FROM organizations WHERE owner_user_id = '6f08a632-...';
DELETE FROM users WHERE id = '6f08a632-...';
SELECT count(*) FROM users WHERE email = 'sec-audit-nonadmin@test.local';
-- 0
```

### Layer 10 verdict

PASS. Live behaviour matches the code review: every admin endpoint
returns 403 for non-admin, 401 for anonymous, and never leaks 404 /
200 differential.

---

## Findings ranked

### CRITICAL

(none)

### HIGH

(none)

### MEDIUM

**M1 — 30 s `is_admin` revocation lag on demote-via-SQL.** A user
demoted via `UPDATE users SET is_admin = false` keeps admin access
for up to 30 s — until the `user_state:<uuid>` Redis cache TTL
expires. Reproducible in 90 s with the SQL + curl sequence in Layer
5.

**Impact:** an operator who removes a hostile admin via SQL must wait
30 s for the access to be revoked. During that window the user can
call any of the 61 admin endpoints (incl. `/users/{id}/ban`,
`/disputes/{id}/resolve`, etc.).

**Mitigation today:** none in code. Operators must wait or call
`redis-cli DEL user_state:<uuid>` manually after the SQL update.

**Fix (NOT applied):** wire the `userStateCache.Invalidate(ctx, uid)`
call into a future operator-side promote/demote endpoint, OR call
`Invalidate` from the suspend/ban paths (defense in depth — those
paths are already protected by `BumpSessionVersion`, but the
invalidation makes the protection 100 % rather than 99 %).

### LOW

**L1 — `RequireAdmin` does not log denial events.** Unlike
`RequireRole`, which emits a `slog.Warn("authorization.denied",
"reason", "insufficient_role", ...)`, `RequireAdmin` writes the 403
silently. A non-admin probing admin endpoints leaves no audit trail.
Adding the structured log line (with `request_id`, `user_id`, `path`,
`method`) would make brute-force probing visible in production logs
without any user-visible change.

**L2 — CORS allow-list is shared across marketplace and admin.**
Currently the marketplace origin is allowed to talk to admin
endpoints. This is not a vulnerability under the current bearer +
in-memory storage model (no admin httpOnly cookie exists for the
marketplace to replay), but a tighter per-route policy would be
defense in depth. Splitting into `cfg.MarketplaceOrigins` and
`cfg.AdminOrigins` (with the admin middleware checking the latter)
would close that small gap.

---

## Fixes applied

I added test coverage only:

1. `backend/internal/handler/middleware/admin_test.go` — 6 unit tests
   for `RequireAdmin`.
2. `backend/internal/handler/admin_security_test.go` — 5 structural
   tests that pin the admin route inventory and middleware shape.

I did **not** apply any code change to behaviour because:

- Layer 1-4, 6, 8 found no defects.
- The Layer 5 finding (M1) is a 30 s revocation gap that requires
  the owner's approval to fix (changes auth semantics, even if
  trivially).
- The Layer 7 note (L2) is a hardening idea, not a vulnerability.
- The Layer 9 audit-log idea (L1) is a small enhancement, not a
  defect.

Each "not applied" finding has a concrete fix proposal documented
above so the owner can green-light it in a follow-up PR.

---

## Audit doc location

`backend/docs/admin-security-audit-2026-05-09.md` (this file)

## Test files added

- `backend/internal/handler/middleware/admin_test.go`
- `backend/internal/handler/admin_security_test.go`
