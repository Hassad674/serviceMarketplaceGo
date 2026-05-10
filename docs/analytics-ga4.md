# Google Analytics 4 — setup, taxonomy, consent gating

The marketplace ships **GA4** alongside **PostHog**. The two SDKs cover
different concerns and coexist on every conversion event:

- **GA4** — acquisition analytics. Where users come from, which channel
  drives sign-ups and purchases, traffic-source breakdowns. Wired to
  the Search Console for organic visibility.
- **PostHog** — product analytics. What users do once they arrive, full
  funnel attribution, session-level behaviour. EU-hosted for RGPD scope.

Every conversion call site fires **both** SDKs through a single helper
in `web/src/shared/lib/analytics-events.ts`. Feature code never imports
either SDK directly.

## TL;DR

- GA4 SDK: `@next/third-parties/google` (`<GoogleAnalytics gaId>`).
  Loads the gtag script with `next/script` strategy=`afterInteractive`.
- Measurement ID: `G-4424ZE0MTS` (property name: `marketplace-prod`,
  stream URL: `https://services.designedtrust.com`).
- Single consent flag: the existing PostHog cookie banner persists
  `marketplace.analytics.consent` in localStorage. GA4 honours the
  same flag — opted-out users never see the gtag script load.
- Provider mount: `web/src/shared/components/analytics/google-analytics-provider.tsx`,
  wired in `web/src/app/[locale]/providers.tsx`.

## Environment variables

| Var | Surface | Required | Default |
|---|---|---|---|
| `NEXT_PUBLIC_GA_MEASUREMENT_ID` | Web | optional, fail-open | empty |

Empty env = no `<GoogleAnalytics>` rendered = no script injected = no
network call. Local dev without an ID is a complete no-op.

### Action items for prod deploy

- **Vercel** (web): add `NEXT_PUBLIC_GA_MEASUREMENT_ID=G-4424ZE0MTS` to
  the env config of every deployment (production, preview).
- Backend / mobile: GA4 is a web-only SDK. No action.

## Consent flow

```
[user lands] → cookie banner shown (no script loaded)
  ├─ "Accepter" → localStorage["marketplace.analytics.consent"] = "accepted"
  │              → window.dispatchEvent("analytics:consent-changed")
  │              → GoogleAnalyticsProvider re-renders
  │              → <GoogleAnalytics gaId> mounts → gtag.js loads
  │              → events fire to GA4
  └─ "Refuser"  → localStorage["...consent"] = "refused"
                 → GoogleAnalyticsProvider stays empty (no script)
```

Subsequent reloads read the persisted consent and skip the banner.
Cross-tab consent changes propagate via the standard `storage` event.

## Conversion events (GA4 canonical names)

GA4 uses a canonical event taxonomy that lights up out-of-the-box
dashboards (Acquisition, Engagement, Monetisation). Every conversion
call site uses the GA4-canonical name and the matching property
schema, so we get free reports without configuring custom dimensions.

| GA4 event | Trigger | Properties |
|---|---|---|
| `sign_up` | Successful POST `/api/v1/auth/register` | `method` (always `"email"`), `role` (`agency`/`enterprise`/`provider`) |
| `purchase` | Stripe payment succeeded (proposal escrow funded) | `value` (float, EUR), `currency` (`"EUR"`), `transaction_id` (Stripe payment intent id or proposal id), `items` (array with `item_id`, `item_name`, `price`, `quantity`) |
| `generate_lead` | "Send message" clicked on a public profile | `profile_id` (org id of the contacted profile), `persona` (`freelance`/`agency`/`referrer`) |
| `search` | Search bar submit on landing or `/search` | `search_term` (query string), `persona` (active role tab), `filters_count` (number of applied filters) |

Properties are flat — GA4 dashboards only flatten one level. `items` is
the documented exception (GA4 ecommerce schema).

### Mirror in PostHog

The same call sites fire a PostHog event so dashboards on the
PostHog side stay in sync:

| PostHog event | GA4 event |
|---|---|
| `auth.register_completed` | `sign_up` |
| `proposal.payment_succeeded_client` | `purchase` |
| `public_profile.send_message_clicked` | `generate_lead` |
| `search.executed` | `search` |

The backend already captures `auth.user_registered` and
`proposal.payment_succeeded` server-side via the PostHog Go SDK —
those are the source of truth for delivery guarantees. The browser
events are the analytics-friendly client-side mirror.

## CSP

The CSP shipped from `web/src/shared/lib/csp.ts` whitelists every GA4
origin the gtag SDK touches:

- `script-src` adds `https://www.googletagmanager.com`
- `connect-src` adds `https://www.google-analytics.com`,
  `https://*.analytics.google.com`,
  `https://*.googletagmanager.com`
- `img-src` adds `https://www.google-analytics.com` and
  `https://*.analytics.google.com` (1×1 pixel beacons)

A regression test in `web/src/shared/lib/__tests__/csp.test.ts`
locks these origins. If you ever add a new GA4 product (Tag Manager
container, Analytics 360 features), update the test alongside the
CSP.

## Smoke test

After deploying with `NEXT_PUBLIC_GA_MEASUREMENT_ID` set:

1. Open the production site in an incognito window.
2. Click "Accepter" on the cookie banner.
3. In a separate tab, open
   [GA4 dashboard → Reports → Realtime](https://analytics.google.com/).
   Within 30 seconds the active visitors counter should tick up.
4. Trigger one of the four conversion events (e.g., submit the
   landing search bar). Within ~30 seconds the GA4 Realtime "Event
   count by event name" card should show your event.
5. Verify the schema in the GA4 DebugView (Admin → DebugView) — you
   need to install the [GA Debugger Chrome extension](https://chrome.google.com/webstore/detail/google-analytics-debugger/jnkmfdileelhofjcijamephohjechhna)
   first, then the event payload (including `items` for `purchase`)
   appears in real time.

## Architecture notes

### Provider

`web/src/shared/components/analytics/google-analytics-provider.tsx`
is a pure side-effect component (~40 lines). Two gates protect the
render:

- env gate: `getGAConfig()` returns `isEnabled: false` when the var
  is missing.
- consent gate: `readConsent() === "accepted"`. The provider
  subscribes to both `storage` (cross-tab) and a custom
  `analytics:consent-changed` event (same-tab) so flipping consent
  updates the rendered tree without a reload.

When both gates pass, the provider renders
`<GoogleAnalytics gaId={...} />` from `@next/third-parties/google`.
The helper handles script injection via `next/script` with the
`afterInteractive` strategy — no impact on LCP.

### Event helper

`web/src/shared/lib/analytics-events.ts` exposes four helpers:
`trackSignUp`, `trackPurchase`, `trackLead`, `trackSearch`. Each
helper fires both PostHog and GA4 with the event names + properties
documented above. Call sites import the helpers; they never import
`@next/third-parties/google` or `posthog-js` directly.

PostHog respects its own opt-out via
`posthog.has_opted_out_capturing()` checked inside `captureEvent()`.
GA4 respects the provider-level mount gate — when the script is not
loaded, `sendGAEvent()` is a no-op (the gtag queue swallows the
call). Belt-and-suspenders: the wrapper in `lib/ga.ts` also wraps
the call in try/catch.

### Why `@next/third-parties/google` and not raw gtag

- Officially maintained by the Next.js team.
- ~3 KB runtime overhead.
- Uses `next/script` with `afterInteractive` strategy → does not
  block hydration, no LCP regression.
- Type-safe `GAParams` props.
- Centralised loading semantics — easier to audit than handcrafted
  `<script>` tags scattered across pages.
