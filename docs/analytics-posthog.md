# PostHog analytics — setup, taxonomy, dashboards

The marketplace ships a single PostHog project (`serviceMarketplace`,
project id `175990`, EU region) that captures events from **all three
surfaces** of the product: the Go backend (server-side), the Next.js
web app (browser SDK), and the Flutter mobile app (native SDK).

## TL;DR

- Same public project token everywhere: `phc_qQXNZTfWaFcMF8HrJpDYStBwqNBvtkXAD3NVw2H9bpyy`
- EU host: `https://eu.posthog.com` (RGPD-friendly).
- Backend captures the events that must never be lost (registration,
  payment success). Web + mobile capture user-experience signals
  (search, click-throughs, profile views).
- All three surfaces identify users on login, attach the
  `organization` group, and reset on logout. Dashboards filter by
  organization plan, role, etc. without us shipping the dimension
  on every event.

## Why PostHog

- **One platform, three SDKs**, identical event taxonomy across
  surfaces — no per-channel translations to maintain.
- **EU host**: data stays in Ireland → RGPD scope tight, no SCC
  needed for the marketplace.
- **Free tier** generous enough for dev + staging.
- **Open source** core in case the project ever needs to self-host.

## Environment variables

| Var | Surface | Required | Default |
|---|---|---|---|
| `POSTHOG_PROJECT_KEY` | Backend | optional, fail-open | empty |
| `POSTHOG_HOST` | Backend | optional | `https://eu.posthog.com` |
| `NEXT_PUBLIC_POSTHOG_KEY` | Web | optional, fail-open | empty |
| `NEXT_PUBLIC_POSTHOG_HOST` | Web | optional | `https://eu.posthog.com` |
| `--dart-define=POSTHOG_PROJECT_KEY=...` | Mobile | optional | empty |
| `--dart-define=POSTHOG_HOST=...` | Mobile | optional | `https://eu.posthog.com` |

The project token is **public by design** — PostHog ships the same
value to the backend (server-side capture) and the browser/mobile
SDKs.

### Action items for prod deploy

- **Vercel** (web): add `NEXT_PUBLIC_POSTHOG_KEY` and
  `NEXT_PUBLIC_POSTHOG_HOST` to the env config of every deployment
  (production, preview).
- **Railway** (backend): add `POSTHOG_PROJECT_KEY` and
  `POSTHOG_HOST` to the API service env.
- **Mobile CI**: add `--dart-define=POSTHOG_PROJECT_KEY=...` and
  `--dart-define=POSTHOG_HOST=...` to the `flutter build` command in
  the release pipeline.

## Smoke test

After every deploy, run the backend smoke test:

```bash
cd backend
POSTHOG_PROJECT_KEY=phc_qQXNZTfWaFcMF8HrJpDYStBwqNBvtkXAD3NVw2H9bpyy \
POSTHOG_HOST=https://eu.posthog.com \
POSTHOG_DEBUG=true \
go run ./cmd/posthog-smoke
# expected output: "ok"
```

The PostHog UI's activity feed should show the
`smoke_test.backend` event for distinct id `smoke-backend` within 2
seconds.

## Event taxonomy

Naming follows `domain.action_in_past_tense`. Properties are flat
(no nested objects — PostHog dashboards only flatten one level
automatically).

### Captured server-side (backend) — guaranteed delivery

| Event | Trigger location | Properties |
|---|---|---|
| `auth.user_registered` | `internal/handler/auth_handler.go` Register() success | `role`, `source` |
| `auth.login_succeeded` | `internal/handler/auth_handler.go` Login() success | `role`, `source` (group: `organization`) |
| `proposal.payment_succeeded` | `internal/handler/stripe_handler_more.go` after `payment_intent.succeeded` webhook | `payment_intent_id`, `proposal_id`, `amount`, `client_total`, `currency`, `provider_id`, `source` (idempotency: Stripe event id) |

### Captured client-side (web + mobile) — surface-specific

The browser/mobile SDKs auto-capture `$pageview` and lifecycle
events. Custom captures are wired progressively as features adopt
the `useAnalytics()` hook (web) / `PostHogService.instance.capture()`
(mobile). The core taxonomy planned (and reserved in dashboards):

| Event | Surfaces | Properties |
|---|---|---|
| `landing.search_submitted` | web, mobile | `query`, `persona` |
| `landing.cta_clicked` | web | `cta_id` |
| `public_profile.viewed` | web, mobile | `profile_id`, `persona` |
| `public_profile.send_message_clicked` | web, mobile | `profile_id` |
| `auth.register_started` | web, mobile | `role` |
| `auth.email_verified` | web, mobile | — |
| `profile.completion_changed` | web, mobile | `percent`, `persona` |
| `proposal.payment_initiated` | web, mobile | `proposal_id`, `amount` |
| `messaging.message_sent` | web, mobile | `conversation_id`, `has_attachment` |
| `search.executed` | web, mobile | `query`, `persona`, `filters_applied` |
| `job.application_submitted` | web, mobile | `job_id`, `applicant_kind` |
| `referral.intro_created` | web | `referral_id` |
| `subscription.upgraded` | backend (Stripe webhook) | `plan` |

## Identify + group flow

All three surfaces follow the same shape on login:

```
posthog.identify(user.id, { email, role, email_verified, referrer_enabled })
posthog.group("organization", org.id, { type, plan, member_role })
```

On logout:
```
posthog.reset()
```

Dashboards can therefore filter by:
- `event.role = agency` — events from agencies only
- `group:organization.plan = premium` — only paying orgs
- `group:organization.type = enterprise` — only enterprise clients

## RGPD posture

- **EU data residency**: events ship to `eu.posthog.com` →
  Ireland. No SCC needed.
- **Opt-in by default**: the SDKs are configured with
  `opt_out_capturing_by_default: true`. The cookie banner (web) /
  in-app toggle (mobile) flips the SDK on only after the user
  agrees.
- **No PII in events**: distinct id is the user UUID; properties
  carry roles, plans, ids — never email body, message text, or
  uploaded media references.
- **IP anonymisation**: the web SDK sets `ip: false` so PostHog's
  GeoIP enrichment runs on the request header but the raw IP is
  never persisted on the event.
- **Right to deletion**: PostHog's `/api/projects/.../persons/...`
  endpoint is wired into the GDPR purge cron path (TODO: add a
  scheduled job to also wipe the PostHog person on user delete —
  flagged for follow-up, not in this PR's scope).

## Architecture notes

### Backend hexagonal split

```
port/service/analytics_service.go   AnalyticsService interface
adapter/posthog/client.go           PostHog implementation
adapter/noop/analytics_service.go   No-op fallback
cmd/api/wire_infra.go               buildAnalyticsService(cfg)
```

When `POSTHOG_PROJECT_KEY` is empty the wiring picks the no-op
adapter and logs `posthog: project key missing — analytics
disabled` once at boot. Capture sites unconditionally call
`.Capture()`; the noop adapter silently drops events.

### Web feature isolation

`shared/lib/posthog.ts` is the single import surface. Feature code
calls `useAnalytics()` from `shared/hooks/use-analytics.ts` — never
imports `posthog-js` directly. This keeps features removable
without breaking analytics, and lets a future swap (Mixpanel,
Amplitude) touch one file.

### Mobile singleton

`PostHogService.instance` is a singleton because the underlying
posthog_flutter plugin is a singleton at the platform layer.
Riverpod tests can wrap it via a `Provider` override; production
code calls the static instance directly.

## Dashboard recipes

The PostHog UI dashboards to create (after first events flow):

1. **Conversion funnel** — landing → register → proposal payment.
   Funnel insight on (`landing.cta_clicked`, `auth.register_started`,
   `auth.user_registered`, `proposal.payment_succeeded`).
2. **Daily active orgs** — trend insight on
   `auth.login_succeeded` aggregated by `group:organization.id`.
3. **Search → engagement** — trend on `search.executed` followed
   by `public_profile.viewed` within 1 hour. Drop-off rate.
4. **Plan upgrades** — trend insight on `subscription.upgraded`,
   property breakdown by `plan`.
5. **Mobile vs web split** — trend insight on
   `messaging.message_sent` broken down by `$lib` (auto-set by SDK
   to `posthog-js`, `posthog-go`, or `posthog-flutter`).
