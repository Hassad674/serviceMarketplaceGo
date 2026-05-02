# 0001. Hexagonal architecture (ports and adapters)

Date: 2026-04-30

## Status

Accepted

## Context

The marketplace backend (`backend/`) implements ~40 features —
authentication, profiles, jobs, proposals, contracts, messaging,
calls, payments, notifications, search, moderation, GDPR, audit.
Each feature has a database surface, an external-service surface
(Stripe, LiveKit, OpenAI, Resend, MinIO/R2, Typesense), and an HTTP
surface. Without a strict layering rule, a feature can quickly grow
direct imports of `database/sql`, the Stripe SDK, Redis primitives
and so on — each one a small coupling that becomes a barrier to:

- **Switching providers**: replacing Stripe with PayPal would
  require touching every payment use-case.
- **Unit testing the domain**: a domain rule (e.g. "a proposal can
  only be activated if the principal has paid") cannot be exercised
  in isolation if the domain code reaches into Postgres directly.
- **Removing a feature**: the project's "delete the folder"
  invariant (`CONTRIBUTING.md §3`) cannot hold if business logic
  imports adapter packages.

The team works AI-augmented; agents need a predictable layout to
know where to put new code without breaking existing modules.

## Decision

We will follow **hexagonal architecture** (ports and adapters,
Alistair Cockburn) with strict directional dependencies:

```
internal/handler  ->  internal/app  ->  internal/domain  <-  internal/port  <-  internal/adapter
```

Concrete rules:

1. `internal/domain/` holds pure entities and value objects. Imports
   only the Go standard library. No `database/sql`, no Stripe, no
   `net/http`.
2. `internal/port/` holds interface contracts split by purpose:
   `port/repository` (DB access), `port/service` (external systems),
   etc.
3. `internal/app/` holds use cases. Depends on `domain` and `port`.
   Never on `adapter`.
4. `internal/adapter/` holds the concrete implementations
   (`adapter/postgres`, `adapter/redis`, `adapter/stripe`,
   `adapter/livekit`, `adapter/s3`, `adapter/openai`,
   `adapter/resend`, `adapter/typesense`). Adapters never import
   each other.
5. `internal/handler/` is HTTP transport. Depends on `app/` and
   the request/response DTOs.
6. **Wiring** — the only place a port meets its adapter — happens
   in `cmd/api/main.go` (now split across `cmd/api/wire_*.go`
   files for line-budget reasons).

A static check enforces the rule: `internal/app/**` files MUST NOT
import any `internal/adapter/...` path. The CI lint catches
violations at PR time.

## Consequences

### Positive

- Switching providers is a one-file change. Replacing Stripe with
  PayPal means writing `internal/adapter/paypal/payment.go` that
  satisfies the existing `port/service.PaymentService` interface,
  then changing one line in `cmd/api/wire_payment.go`. No domain
  or app changes.
- Unit tests on `internal/app` use generated mocks of the port
  interfaces (`backend/mock/`). Tests run fast and exercise pure
  business logic.
- Removing a feature is a folder delete + a few lines stripped
  from `cmd/api/wire_*.go`. The "delete the folder" invariant
  (`CONTRIBUTING.md §3`) is achievable.
- AI agents can scaffold a new feature predictably:
  `domain/foo/`, `port/repository/foo.go`, `app/foo/service.go`,
  `adapter/postgres/foo.go`, `handler/foo_handler.go`,
  `migrations/NNN_create_foo.up.sql`. The `add-feature` slash
  command walks them through this exact layout.

### Negative

- Boilerplate: every domain entity has a port interface and at
  least one adapter implementation. For a tiny CRUD feature the
  ratio looks excessive. We accept this — the modularity payoff
  is realized as the codebase grows.
- New contributors take a session to internalise the directional
  rule. Once they do, the layout is self-evident.
- Wiring code (`cmd/api/main.go` + `wire_*.go`) is large because
  every binding lives there. We split it into per-feature
  `wire_<feature>.go` files to keep the orchestrator readable
  (P2 refactor brought `main.go` from 909 lines to ≤ 300).

## Alternatives considered

- **Layered MVC (controller → service → repository)** — the
  classic rails-style stack. Rejected because the dependency
  direction is unidirectional only by convention; nothing
  prevents a controller importing a Postgres helper. We have
  seen this fail on a previous codebase where 18 months later
  every layer reached into every other.
- **CQRS + DDD bounded contexts** — overkill for a two-developer
  project. We need modularity, not eventual-consistency machinery.
- **Onion architecture** — a near-cousin of hexagonal. Functionally
  equivalent for our needs; we prefer the simpler "domain in the
  middle, adapters at the edge" mental model.

## References

- `backend/CLAUDE.md` lines 51-178 — full SOLID examples that lean
  on this layering.
- `cmd/api/main.go` and `cmd/api/wire_*.go` — the wiring files.
- `backend/mock/` — generated mocks consumed by `internal/app/**`
  tests.
- `internal/adapter/stripe/` and `internal/adapter/livekit/` —
  representative adapter implementations.
- Alistair Cockburn, *Hexagonal Architecture*,
  <https://alistair.cockburn.us/hexagonal-architecture/>.
