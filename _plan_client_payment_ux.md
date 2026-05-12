# Plan — fix/client-payment-ux

Two contextual UX bugs on the client payment page `/projects/pay/...`.

## Bug 1 — Double navbar on desktop

The page `(app)/projects/pay/page.tsx` inherits the dashboard shell
(`(app)/layout.tsx` → `<DashboardShell>` with sidebar + top header).
A checkout page must be focused: minimal header (logo only), no sidebar.

### Approach

Add a route segment `layout.tsx` inside `(app)/projects/pay/` that
overrides the parent layout. Next.js nested layouts compose, but
because the parent already wraps children with `DashboardShell`, the
nested layout still gets wrapped. **Solution**: instead of replacing
the shell at the segment level, the parent `(app)/layout.tsx` already
supports an "embedded" branch via `?embedded=true` — but using URL
params is brittle. Cleaner: detect the `/projects/pay` path inside the
existing `(app)/layout.tsx` and render the minimal shell when the path
matches. This keeps a single source of truth for the dashboard shell
and avoids fighting the route group nesting.

After re-read: Next.js DOES let a nested `layout.tsx` opt out of the
parent shell via a Route Group at the segment. But the cleanest path
given the existing `?embedded=true` short-circuit is to extend the
parent layout with a path check. We'll add the `/projects/pay`
detection alongside the embedded query param so the same minimal
shell renders.

**Final decision**: keep the parent `(app)/layout.tsx` as the single
gateway. Add a `usePathname` check: if the path matches
`/<locale>/projects/pay` (or starts with it), render the new
`PaymentCheckoutShell` (logo + back-to-dashboard link, no sidebar).
This avoids duplicating the parent's auth-protected branch and lets
the existing middleware enforce auth.

The new `PaymentCheckoutShell` component lives under
`web/src/shared/components/layouts/payment-checkout-shell.tsx`.

## Bug 2 — "Pré-remplir depuis Stripe" button shown to clients

The button is rendered inside `BillingProfileForm` (line 142). It is
useful for prestataires (their Connect KYC has data) but meaningless
for clients (no Connect account).

### Approach

Add a `showStripePrefill?: boolean` prop on `BillingProfileForm`
(defaults to `true` for backwards compat with
`/settings/billing-profile`). Pipe it through `BillingProfileEmbed`
via a new `showStripePrefill?: boolean` prop. On the embed instance
inside `payment-simulation.tsx`, pass `showStripePrefill={false}`.

The synced indicator (`SyncedFromStripeIndicator`) on the left of the
row is also irrelevant for clients — when `showStripePrefill={false}`,
the whole flex row is hidden, since the synced indicator is the only
other content in that row and equally pestraire-centric.

## Mobile parity

Mobile equivalent exists in `mobile/lib/features/invoicing/presentation/widgets/billing_profile_form.dart`
(line 166) — `BillingStripeSyncRow` (label "Sync depuis Stripe").
Mirror the web prop: add `showStripePrefill: bool` (default true) on
`BillingProfileForm` widget and `BillingProfileEmbed` widget,
threaded down to the `BillingStripeSyncRow`. Pass `false` from
`payment_simulation_screen.dart`.

Mobile has no "navbar" issue — the screen is full-screen by default.
No layout work needed on mobile.

## Files modified

### Web
1. `web/src/shared/components/layouts/payment-checkout-shell.tsx` — NEW minimal shell (logo + back link).
2. `web/src/app/[locale]/(app)/layout.tsx` — branch to checkout shell on `/projects/pay`.
3. `web/src/features/invoicing/components/billing-profile-form.tsx` — add `showStripePrefill` prop.
4. `web/src/shared/components/billing-profile/billing-profile-embed.tsx` — add `showStripePrefill` prop (thread through).
5. `web/src/features/proposal/components/payment-simulation.tsx` — pass `showStripePrefill={false}`.
6. `web/messages/{fr,en}.json` — i18n keys for checkout shell ("backToDashboard").
7. Test files (new + extended).

### Mobile
8. `mobile/lib/features/invoicing/presentation/widgets/billing_profile_form.dart` — add `showStripePrefill` flag.
9. `mobile/lib/features/invoicing/presentation/widgets/billing_profile_embed.dart` — add `showStripePrefill` flag.
10. `mobile/lib/features/proposal/presentation/screens/payment_simulation_screen.dart` — pass `false`.
11. Test files (extended).

## Prop name decision

`showStripePrefill: boolean` (default `true`). Rationale: positive
flag (no double negative), explicit, default preserves backwards
compatibility for the prestataire context.

## Mobile decision

Button exists — `BillingStripeSyncRow` in `billing_form_status.dart`,
mounted unconditionally by `BillingProfileForm`. Will be hidden in
client checkout context via the same `showStripePrefill` flag.

## Test count

- New: `payment-checkout-shell.test.tsx` (≥3 cases: renders children, no sidebar, back link present).
- Extended: `billing-profile-form.test.tsx` (+3 cases: prefill shown by default; hidden when `showStripePrefill={false}`; SyncedFromStripeIndicator also hidden).
- Extended: `billing-profile-embed.test.tsx` (+2 cases: prop threaded through; default true).
- Extended: `payment-simulation.test.tsx` (+1 case: embed receives `showStripePrefill={false}`).
- Extended: `payment-billing-clone.spec.ts` Playwright (+2 cases: no sidebar present; no prefill button visible).
- Extended: `billing_profile_form_test.dart` (+2 cases mobile: shown by default; hidden when false).
- Extended: `billing_profile_embed_test.dart` (+1 case mobile: prop threaded).

Total: ~14 new test cases. Coverage target ≥ 90% on touched files.
