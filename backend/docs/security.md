# Security

This document captures the marketplace backend's security
hardening posture. Every entry here is implementation-anchored —
change the code and update the doc in the same commit.

## Slowloris guard (PERF-FINAL-B-01 / P10)

The HTTP server is configured with a 5-second `ReadHeaderTimeout`
in `cmd/api/wire_serve.go::buildHTTPServer`. This caps the time a
client can take to send the request **headers** before the server
closes the connection.

### Why it matters

Without a `ReadHeaderTimeout`, the Go `http.Server` will wait
indefinitely for the headers to complete. A malicious client can
exploit this by sending headers one byte at a time, holding the
connection open and consuming a connection-pool slot. With enough
parallel slow clients (a few hundred goroutines on a single laptop
is plenty), the server runs out of slots and legitimate traffic
gets refused — the classic "slowloris" attack named after the slow
loris primate's deliberate movements.

### What it does NOT cover

`ReadHeaderTimeout` covers the **headers only**. The body window is
governed by `ReadTimeout` (15s) and per-handler deadlines. This
distinction matters: a legitimate slow upload (a 2MB file over a
poor mobile connection that takes 8s to complete) MUST still
succeed. The slowloris guard only triggers when the **headers**
take longer than 5s — which never happens for a real client whose
headers fit in one TCP packet.

| Knob                  | Value | Covers              |
| --------------------- | ----- | ------------------- |
| `ReadHeaderTimeout`   | 5s    | Headers only        |
| `ReadTimeout`         | 15s   | Body (non-streaming) |
| `WriteTimeout`        | 0     | (Disabled — needed for long-lived WebSocket) |
| `IdleTimeout`         | 60s   | Keep-alive idle window |

The 5s value is a tradeoff: tight enough to short-circuit a
slowloris attack quickly, generous enough to never trip a real
client even on a 2G connection.

### Tested behaviour

`cmd/api/wire_serve_test.go` covers four scenarios:

1. `TestBuildHTTPServer_Timeouts` — single source of truth for the
   timeout values
2. `TestBuildHTTPServer_SlowlorisHeader_Aborts` — drip-feeds a
   request line beyond `ReadHeaderTimeout`, asserts the handler
   does NOT execute
3. `TestBuildHTTPServer_LegitimateSlowBody_Succeeds` — sends headers
   fast + 2MB body over 2s, asserts 200 OK (proves the guard does
   NOT touch the body window)
4. `TestBuildHTTPServer_FastRequest_Succeeds` — smoke test on the
   common path

## Mutation rate limit (SEC-FINAL / P10)

Every authenticated `POST` / `PUT` / `PATCH` / `DELETE` on `/api/v1`
is throttled to **30 requests per minute per user** using a
sliding-window Redis-backed limiter (see SEC-11 documentation in
`internal/handler/middleware/ratelimit.go`).

P10 closes one gap in the existing limiter: anonymous mutations
(login, register, password-reset requests sent without a session)
used to short-circuit the limiter because `UserKey` returned false
for unauthenticated requests. Now the wiring uses
`MutationOnly(UserOrIPKey(rl))` which falls back to the client IP
when no user_id is present in context.

### Bucket key derivation

| Request state            | Bucket key       |
| ------------------------ | ---------------- |
| Authenticated + mutation | `user:<uuid>`    |
| Anonymous + mutation     | `ip:<addr>`      |
| Authenticated + GET      | (skipped — `MutationOnly` short-circuits before the limiter runs) |

The `user:` / `ip:` namespace prefixes keep the two bucket families
isolated. An authenticated user running 30 mutations does NOT
consume the IP bucket of someone else later hitting an anonymous
endpoint from the same IP.

### Response headers

Every response — both inside and over the cap — carries the
limiter's headers so a well-behaved client can self-throttle
without having to parse error bodies:

| Header                | Meaning |
| --------------------- | ------- |
| `X-RateLimit-Limit`   | The cap (30) |
| `X-RateLimit-Remaining` | Requests left in the current window |
| `X-RateLimit-Reset`   | Unix epoch when the window resets |
| `Retry-After`         | Seconds to wait — **only on 429 responses** |

### Tested behaviour

`internal/handler/middleware/ratelimit_test.go` covers:

- `TestMutationRateLimit_31stMutationReturns429` — 30 mutations OK,
  31st returns 429 with `Retry-After`
- `TestMutationRateLimit_30MutationsPlusGETPasses` — read traffic
  is never throttled by the mutation cap
- `TestMutationRateLimit_IPFallback_AnonymousMutationsThrottled` —
  anonymous POSTs from the same IP get the same 30/min cap
- `TestMutationRateLimit_IPFallback_DifferentIPsIndependent` — the
  IP bucket is per-IP, not global
- `TestMutationRateLimit_AuthAndAnonShareNothing` — `user:` and
  `ip:` namespaces stay isolated

### Failure modes

Redis blip → fail-open. The limiter logs the error and lets the
request through. The auth middleware + RBAC + handler-level checks
remain the primary security layer; the rate limit is a coarse
"spike absorber", not a security gate.

## Other rate limits

Documented in `internal/handler/middleware/ratelimit.go`:

| Class    | Limit         | Key   | Endpoints |
| -------- | ------------- | ----- | --------- |
| Global   | 100/min/IP    | IP    | All `/api/v1/*` |
| Mutation | 30/min/user   | User or IP | `/api/v1/*` POST/PUT/PATCH/DELETE |
| Upload   | 10/min/user   | User  | Multipart upload endpoints |
| Auth     | 5/min/email   | email | `/auth/login`, `/auth/forgot-password` (handled by `BruteForceService`, not the standard limiter) |
