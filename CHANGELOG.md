# Changelog

All notable changes to **Marketplace Service** are documented in this
file.

The format is based on [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

Releases are tagged on the `main` branch — see the
[GitHub releases page](https://github.com/Hassad674/serviceMarketplaceGo/releases).

Each entry is grouped by the change type:

- **Added** — new features or endpoints.
- **Changed** — changes to existing functionality (non-breaking).
- **Deprecated** — APIs or behaviours that will be removed in a
  future release.
- **Removed** — APIs or behaviours dropped in this release.
- **Fixed** — bug fixes.
- **Security** — fixes for vulnerabilities or hardening of an
  existing surface.

## [Unreleased]

Nothing yet — open a PR.

## [1.0.0-rc.1] - 2026-05-02

The first release candidate covering the Final 3 (F.3.1) publish-ready
gate. Adds the open-source-grade hygiene the project lacked
(architecture decision records, semver discipline, pre-commit hooks)
and closes the last three HIGH security findings from the Final
verification audit.

### Added

- **Architecture Decision Records** in `docs/adr/`. Eight initial
  ADRs in Michael Nygard format covering hexagonal architecture,
  org-scoped business state, RLS + system-actor split, async Stripe
  webhooks, OTLP tracing, feature isolation, RGPD soft-delete, and
  cursor pagination.
- **`middleware.RequireRole(roles...)`** — defense-in-depth role gate
  on every `/admin` route. Layered with the existing `RequireAdmin`
  flag check; either gate failing returns 403. Audit-logs every
  denial with `slog.Warn("authorization.denied", ...)`.
- **Pre-commit hooks** (`.githooks/pre-commit`) — bash, no husky.
  Runs `gofmt -d` on changed Go files, `tsc --noEmit` if any
  TypeScript changed, and `flutter analyze` if any Dart changed.
  Setup with `scripts/install-git-hooks.sh`. Documented in
  `CONTRIBUTING.md` with the `--no-verify` escape hatch.
- **`CHANGELOG.md`** (this file).

### Security

- **SEC-FINAL-07** — admin auth bearer token moved from
  `localStorage` to an in-memory Zustand store. localStorage is
  XSS-readable; on the admin surface a token leak is platform-wide
  catastrophic. Page reload re-uses the existing httpOnly session
  cookie via `/auth/me` boot probe — no UX regression.
- **SEC-FINAL-04** — `ValidateSocialURL` hardened against SSRF:
  - 13 CIDRs explicitly denied (RFC1918, loopback, link-local
    169.254.x — including AWS/GCP/Azure metadata —, multicast,
    CGNAT, IPv6 ULA + link-local + multicast).
  - DNS-rebinding mitigation via `net.LookupIP` + every-IP check.
  - Decimal/octal/hex IP encodings rejected before DNS fires.
  - `javascript:`, `data:`, `vbscript:`, `file:`, `gopher:`
    schemes rejected.
- **SEC-FINAL-03** — `RequireRole` middleware (see Added) closes
  the long-documented gap where `/admin` routes relied on a
  handler-level flag check with no router-side gate.

### Changed

- The `routes.golden` snapshot now records 60+ admin routes with
  `mw=10` instead of `mw=9`, reflecting the new `RequireRole`
  middleware in the chain. Non-admin routes are unaffected.

## [0.9.0] - 2026-04-29 (`v0.9-kyc-custom-final`)

Last pre-1.0 milestone — the F.1 (security) and F.2 (performance +
observability) gates closed across 11 plan items (P1-P12, with
P12 being mobile build_runner remediation).

### Added

- **OpenTelemetry tracing** end-to-end (P11): SDK + OTLP exporter
  with no-op fallback, inbound HTTP, `database/sql`, Redis, and
  outbound HTTP all wrapped. Graceful shutdown drains the WS hub
  before flushing the exporter.
- **Async Stripe webhooks** (P8): events enqueued in
  `pending_events` outbox + dedicated worker. Stripe always sees
  a sub-millisecond ack.
- **GDPR Article 17 soft-delete + 30-day window** (P5): web,
  mobile, and admin all wired. Account export endpoint included
  (Article 20).
- **Mutation rate limit** (P10): covers anonymous traffic via IP
  fallback; 30 req/min on POST/PUT/PATCH/DELETE for authenticated
  users, 100 req/min global.
- **Slowloris guard** via `ReadHeaderTimeout=5s` on the HTTP server.
- **Slow-query structured log** with 50 ms / 500 ms thresholds.
- **Web Server Components and primitives migration** (P3):
  Button, Input, Card, Select, Modal primitives shipped; raw HTML
  elements (`<button>`, `<input>`, `<select>`) banned by ESLint.
- **`<img>` to `next/image`** migration (P4) for LCP improvement.
- **Mobile build_runner artefacts committed** (P12): Freezed +
  json_serializable artefacts now in git for parity with the
  contributor flow.

### Changed

- `main.go` reduced from 909 lines to ≤300 by extracting per-feature
  wires into `cmd/api/wire_<feature>.go` (P2).
- 6 god repositories narrowed via Interface Segregation Principle
  (P7): `UserRepository`, `OrganizationRepository`,
  `ProposalRepository`, `DisputeRepository`, `MessageRepository`,
  `ReferralRepository`. Consumers now depend on segregated child
  interfaces.
- Web features no longer cross-import each other: ESLint enforces
  `import/no-restricted-paths` zones (P9).

### Performance

- Messaging list path reads denormalized `last_message_*` columns
  on `conversations` (P6) — replaces a per-conversation join, cuts
  list latency by ~40 %.

### Security

- Row-Level Security on 9 tenant-scoped tables with the
  system-actor escape hatch for cross-tenant background jobs.
- Brute-force protection on login + password reset.
- Refresh-token rotation through a Redis blacklist (single-use
  refresh tokens).
- `SecurityHeaders` middleware (CSP, HSTS, X-Frame-Options, etc.).
- Default-secret fail-fast in production boot.
- Single-use `ws_token` for mobile WebSocket auth (drops
  legacy JWT-in-URL).

## [0.8.x and earlier]

Pre-1.0 development. Highlights condensed:

- **Feature build-out** — auth, profiles, jobs, proposals,
  contracts, messaging (with WebSocket presence + read receipts),
  reviews, reports, payments (Stripe Connect Custom + Embedded
  KYC), wallet + payouts, calls (LiveKit), notifications,
  moderation queue, search engine (Typesense + custom ranker),
  invoicing, disputes, team membership + invitations, referrals.
- **Mobile parity** — Flutter app (Clean Architecture + Riverpod
  + Freezed) mirrors every web feature.
- **Admin panel** — Vite + React 19, dashboard / moderation
  queue / disputes / invoices.
- **Backend infrastructure** — Go 1.25 + Chi v5, hexagonal
  architecture, golang-migrate, Redis sessions, MinIO/R2 storage.
- **CI/CD** — GitHub Actions running `go test`, `vitest`,
  `flutter analyze` + `flutter test`, ESLint, Playwright.

The full pre-1.0 history is preserved in the git log and tagged
under `v0.9-kyc-custom-final`.

[Unreleased]: https://github.com/Hassad674/serviceMarketplaceGo/compare/v1.0.0-rc.1...HEAD
[1.0.0-rc.1]: https://github.com/Hassad674/serviceMarketplaceGo/compare/v0.9-kyc-custom-final...v1.0.0-rc.1
[0.9.0]: https://github.com/Hassad674/serviceMarketplaceGo/releases/tag/v0.9-kyc-custom-final
