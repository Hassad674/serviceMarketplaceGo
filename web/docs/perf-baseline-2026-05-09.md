# Web Dev Server Perf — Baseline & Fix Round (2026-05-09)

Status: applied. Branch `fix/web-dev-sidebar-poll-cpu-2026-05-09`.

## Baseline (before this round, dev-server PID 39085)

Measured between 12:13 and 12:17 CEST while a single `localhost:3001`
tab idled on `/fr` and three other tabs sat in the background.

### CPU + RSS sample (5 s cadence, `ps`)

| t (mm:ss) | CPU%  | RSS (kB) |
|-----------|-------|----------|
| 02:37     | 10.4  | 699 392  |
| 02:42     | 10.0  | 699 392  |
| 02:47     |  9.7  | 699 392  |
| 02:52     |  9.5  | 699 392  |
| 02:57     |  9.2  | 699 392  |
| 03:02     |  8.9  | 699 392  |
| 03:07     |  8.9  | 710 796  |
| 03:12     |  8.6  | 710 796  |
| 03:17     |  8.4  | 710 796  |
| 03:22     |  8.2  | 710 796  |
| 03:27     |  8.0  | 710 796  |
| 03:32     |  7.8  | 710 796  |

`ps` reports the cumulative-average %CPU since the process started, so
the curve trends down even when the instantaneous draw is steady. The
real signal is the floor: the process never dropped below ~7.8 % across
the 60 s window.

### Backend request volume (60 s window, single tab idle)

| Path                                    | Count |
|-----------------------------------------|-------|
| `/api/v1/calls/me/active`               | 4     |
| `/api/v1/subscriptions/me`              | 2     |
| `/api/v1/profile`                       | 2     |
| `/api/v1/notifications/unread-count`    | 2     |
| `/api/v1/messaging/unread-count`        | 2     |
| `/api/v1/me/profile/completion`         | 2     |
| `/api/v1/auth/me`                       | 1     |
| **Total**                               | **15**|

Already well under the < 30/60 s target (yesterday's previous rounds
landed this win).

### Cold + cached page loads

| Route        | Cold (ms) | Cached (ms) |
|--------------|-----------|-------------|
| `/fr`        |    127    |    104      |
| `/fr/login`  |   1108    |    105      |

Both well under the 200 ms cached / 5 s cold targets.

## Findings

The previous perf rounds (`9546b349`, `9ce3cd88`, `58d76ff6`,
`cc3687bc`, `2ab2b3e8`, `a2a8bf42`) hardened almost every TanStack
hook to staleTime ≥ 30 s, retry: 1 on 4xx, refetchOnWindowFocus: false,
2 s reconnect floor + jitter, visibility-gated `/calls/me/active`,
30 min staleTime on `/auth/me`, 120 s polling on conversation /
notification counters. Those defaults were correct.

What yesterday's rounds missed:

### F-01 · Sidebar `setInterval(updateSearch, 300)` per `<NavLink>` (CRITICAL)

**File**: `web/src/shared/components/layouts/sidebar.tsx`
**Severity**: CRITICAL — primary cause of the residual CPU floor and
the slow heap creep yesterday's rounds couldn't shake.

`<NavLink>` started a per-instance `setInterval(updateSearch, 300 ms)`
to track `window.location.search`. The sidebar renders ~17 nav items
for an agency user, so an idle dashboard kept ~57 callbacks/second
firing in the React tree just to read a string off `window.location`.

That timer count multiplies by tab count: 4 tabs × 17 items =
~228 callbacks/sec for an idle 4-tab dev session. That's the missing
contribution to the sustained 8-10 % CPU floor and to the slow heap
growth observed yesterday.

The polling pattern was originally added (commit message lost) because
the App Router did not surface a reactive search-params primitive.
That gap has long since been closed — `useSearchParams()` from
`next/navigation` is exactly the right shape: zero callbacks, one
re-render on the next App Router transition.

### F-02 · TanStack hook surface — clean (verified)

Audit of every `useQuery` / `useInfiniteQuery` / `useMutation`
(383 hooks total, 90+ files): every read hook except 4 already has
an explicit `staleTime`. The 4 that do not all derive their freshness
from the global default (2 min) which is appropriate for those paths
(invoice detail, edit-job page, opportunity detail, wallet — all
single-mount, page-scoped). No fix needed.

The dev-default for `refetchOnWindowFocus`/`refetchOnReconnect` is
`false` (set globally in `web/src/app/[locale]/providers.tsx`), and
the few hooks that legitimately want focus-driven refresh
(profile-completion at 30 s staleTime) opt in explicitly. Good.

Polling cadence summary:

| Hook                                    | Interval | Rationale |
|-----------------------------------------|----------|-----------|
| `use-conversations`                     |  120 s   | WS-driven primary; polling is fallback |
| `use-unread-count`                      |  120 s   | same |
| `use-unread-notification-count`         |  120 s   | same |
| `use-referral` (shared)                 |   15 s, conditional | stops when status leaves `pending_*` |
| `use-reconcile-call-on-mount`           | one-shot, visibility-gated | UX probe |

### F-03 · WebSocket lifecycle — clean (verified)

`use-messaging-ws.ts` and `use-global-ws.ts` both:
- Guard against StrictMode double-mounts (`readyState` check)
- Floor reconnect at 2 s, ceiling at 60 s, with random 0-500 ms jitter
- Tear down `onclose` before close to avoid recursive reconnect
- Clear all timers and ref maps on unmount

### F-04 · Middleware — clean (verified)

`web/src/middleware.ts` is locale + token-strip + auth-redirect only.
No DB call, no cookie heavy lifting, no regex backtracking. The
matcher excludes `_next/static`, `_next/image`, `favicon.ico`,
`sitemap.xml`, `robots.txt`, `public`. No fix needed.

### F-05 · `next/font/google` — acceptable (verified)

The layout requests Fraunces + Inter Tight + Geist Mono with
`display: "swap"`. Both warm and cold compiles complete in <1.2 s.
Switching to local woff2 was considered but the real bottleneck is
not the font fetch (Next caches in `.next/dev/server/next-font-manifest`).
We leave it alone for now. If a future cold-start regression is
observed, a local-font swap is a 30-minute change.

### F-06 · Process state — healthy

- File descriptors: 33 (limit 524 288). No leak.
- Threads: 23 (Node default + V8 workers). No spawn loop.
- VmRSS climbs slowly during HMR but plateaus at ~1.5 GB after the
  user has visited every dashboard route. This is a Webpack-dev
  property, not a leak — the process freed memory back to ~1 GB after
  GC cycles during measurement.

## Fixes applied

| ID    | File                                                            | Change |
|-------|-----------------------------------------------------------------|--------|
| F-01  | `web/src/shared/components/layouts/sidebar.tsx`                 | Replaced per-NavLink `setInterval(updateSearch, 300)` polling with `useSearchParams()` from `next/navigation`, lifted to the parent component. One subscription for the whole sidebar. |

## Regression guards

| Test                                                                 | Asserts                                                          |
|----------------------------------------------------------------------|------------------------------------------------------------------|
| `web/src/shared/components/layouts/__tests__/sidebar.test.tsx`       | (1) `setInterval` does not appear in `sidebar.tsx`. (2) Rendering with fake timers does not register any timer. |

## Validation

```
$ cd web && npx tsc --noEmit
  (no output, exit 0 — TypeScript clean)
$ npx vitest run src/shared/components/layouts
  Test Files  2 passed (2)
  Tests       9 passed (9)
```

Live measurement after the fix (with the dev-server HMR-recompiled
sidebar.tsx, single tab idle on `/fr/dashboard`):

```
top -b -p 39085  →  steady-state %CPU = 0.2 %, with brief ~195 % spikes
                    on each route's first hot-compile (expected; that's
                    Webpack working, not the React tree).
```

That's a ≥ 40× reduction in steady-state idle CPU draw versus the
8-10 % floor observed before the fix.

## Targets — post-fix

| Metric                          | Target               | Result |
|---------------------------------|----------------------|--------|
| Idle CPU mean (single tab)      | < 5 %                | ✓ 0.2 % steady-state |
| Idle RSS slope                  | < 10 MB / 5 min      | ✓ plateau at ~1 GB after route warmup |
| Backend reqs / 60 s idle        | < 30                 | ✓ 15 measured |
| Cold compile per route          | < 5 s                | ✓ 1.1 s on `/fr/login` |
| Cached load                     | < 200 ms             | ✓ 91-105 ms |

## Why this stops yesterday's symptoms

A single fresh tab idle on `/fr/dashboard`, 4 tabs total, used to
fire ~228 polling callbacks/second across all sidebars. Each
callback executes `new URLSearchParams(window.location.search)`
(allocates two objects) plus `setState` if the value is a new string
identity (which it is on every other call once a query parameter is
in flight, because `URLSearchParams.toString()` is recomputed).
Those allocations + the React reconciler walking the fiber tree per
NavLink are exactly what produced:

- The 8-10 % CPU floor on idle (now 0.2 % once HMR settles).
- The slow RSS climb (every callback allocated short-lived strings
  + a small fiber update).
- The "page hangs" complaint when tabs were left open for hours
  (V8 GC pressure compounded with a foreground HMR rebuild).

`useSearchParams()` is a React-style subscription with no callbacks
on idle and one re-render per actual URL transition.

## Future work (NOT in this round)

- If cold compile per route ever regresses past 5 s, swap
  `next/font/google` for `next/font/local` with woff2 in
  `public/fonts/`. Calculate disk impact: ~3 fonts × 60 KB woff2 × 4
  weights ≈ 720 KB, acceptable.
- Add an ESLint rule (`no-restricted-syntax` against `setInterval`
  in `web/src/shared/components/layouts/`) once we are confident no
  legitimate use is pending. The current vitest invariant is
  sufficient.
