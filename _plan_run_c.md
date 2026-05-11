# WALLET-UNIFY Run C — Web refonte plan

Run 3 of 4. Backend (Run A + Run B) merged into `main` at `76fb5d0a`. Run D
(mobile parity) follows. Scope strictly contained to:

- `web/src/features/wallet/` (refonte unified header, history timeline)
- `web/src/features/referral/` (identity unmask for owner, projection per
  milestone, end-intro modal + badge)
- `web/src/app/[locale]/(app)/wallet/page.tsx`
- `web/src/app/[locale]/(app)/referrals/[id]/page.tsx`
- `web/messages/{fr,en}.json` for i18n strings
- `web/e2e/wallet-unified.spec.ts`
- `web/src/shared/types/api.d.ts` (regenerated from backend golden)

Off-limits (do NOT touch): `web/src/features/proposal/`,
`web/src/features/invoicing/`, `web/src/app/[locale]/(app)/projects/pay/`,
`web/src/app/[locale]/(app)/settings/billing-profile/`, backend Go, mobile,
admin, `web/src/app/[locale]/legal/*`.

## Backend contract reminder (from Run B)

`GET /api/v1/wallet/summary` (envelope-wrapped `{"data": …}`):

```jsonc
{
  "data": {
    "currency": "EUR",
    "total_cents": 0, "available_cents": 0, "escrowed_cents": 0, "transmitted_cents": 0,
    "breakdown": {
      "missions":    { "total_cents": 0, "available_cents": 0, "escrowed_cents": 0, "transmitted_cents": 0 },
      "commissions": { "total_cents": 0, "available_cents": 0, "escrowed_cents": 0, "transmitted_cents": 0 }
    },
    "recent_transactions": [
      { "type": "mission"|"commission", "amount_cents": 0, "currency": "EUR", "status": "...", "mission_title": "...", "occurred_at": "ISO8601", "reference_id": "..." }
    ],
    "next_cursor": "<opaque base64>"
  }
}
```

`POST /api/v1/wallet/withdraw` body `{ amount_cents? }`:

- 200 / 207: `{ "data": { "drained_cents", "missions_cents", "commissions_cents",
  "stripe_transfer_ids", "currency", "errors": [{source, code, message}] } }`
- 422: `{ "error": { "code": "kyc_required", "message": "…" }, "onboarding_url": "…", "redirect": "/payment-info" }`
- 403: `billing_profile_incomplete` (handled by existing
  BillingProfileCompletionModal).

`POST /api/v1/referrals/attributions/{id}/end`:

- 200 (bare body, NOT envelope-wrapped):
  `{ "id": "...", "referral_id": "...", "proposal_id": "...", "ended_at": "ISO8601" }`
- 403/404 on RBAC/not-found.
- Idempotent — second call returns the same payload.

## Decisions

1. **`is_owner` source**: client-side compute via
   `referral.referrer_id === useCurrentUserId()`. The `ReferralResponse`
   already exposes `referrer_id`, `provider_id`, `client_id` to all parties
   (existing pattern reused — see `resolveViewerRole`). No backend extension
   needed (Option 2 from brief).

2. **Per-milestone projection**: use the existing
   `useReferralCommissions(referralId)` hook which fetches one row per
   milestone with `status` + `commission_cents` + `attribution_id`. Filter
   per-attribution in the UI. Projection of "yet-to-be-released" milestones
   relies on `useReferralAttributions(id)`'s `escrow_commission_cents`
   aggregate at the attribution level; commission rows + projection are
   complementary (rows = realized, attribution.escrow = projected).

3. **`ended_at` for the badge**: the response of `EndAttribution` carries
   `ended_at`. `useReferralAttributions` does NOT expose `ended_at` today —
   when the user re-loads the page after ending, we cannot derive the badge
   from the existing payload. Acceptable tradeoff per scope discipline:
   the badge is rendered locally from the mutation response while the page
   is alive; on a page refresh the button comes back. A backend extension
   to surface `ended_at` on the attribution DTO is OUT OF SCOPE (Run C is a
   web refonte only — Run D will mirror; the persistence of the badge can
   ship in a thin follow-up). FLAGGED in final report.

4. **History timeline status mapping**: the unified `recent_transactions`
   carries arbitrary status strings from both legs. We map common statuses
   to four tones (paid/pending/escrowed/failed). Unknown statuses fall back
   to "muted" tone. This is intentional graceful degradation.

5. **Stripe transfer IDs**: surfaced as a 207 sub-detail only when there
   are errors — we never display them in the happy path (no user value).

## File plan

### Hooks (commit 2)

- `web/src/features/wallet/api/wallet-api.ts` — ADD:
  - `WalletSummary`, `WalletSummaryLeg`, `WalletSummaryTransaction` types
  - `WithdrawResult`, `WithdrawError` types
  - `getWalletSummary(cursor?)` → `Promise<WalletSummary>`
  - `withdrawWallet(amount_cents?)` → `Promise<WithdrawResult>`
  - Keep legacy `getWallet`, `requestPayout`, `retryCommission` exports
    (still used by other surfaces; Run C does not remove them).
- `web/src/features/wallet/hooks/use-wallet.ts` — ADD:
  - `useWalletSummary({ cursor? })` keyed `["wallet","summary",{cursor}]`,
    `staleTime: 30_000`.
  - `useWalletWithdraw()` mutation, invalidates `["wallet"]` (broad — covers
    both legacy and summary keys).
- `web/src/features/referral/api/referral-api.ts` — ADD `endAttribution(id)`.
- `web/src/features/referral/hooks/use-referrals.ts` — ADD
  `useEndIntroAttribution()` mutation that invalidates `referralKeys.all`
  and `["wallet","summary"]`.

### Wallet refonte (commits 3 + 4)

- `web/src/features/wallet/components/wallet-unified-page.tsx` — NEW root.
  Replaces the body of `wallet-page.tsx`'s `WalletPage()`. Uses
  `useWalletSummary()`.
- `web/src/features/wallet/components/wallet-unified-header.tsx` — NEW.
  Hero card: title + total + single "Retirer" CTA + 3 cards (escrowed,
  available, transmitted). Uses `WalletPayoutSection`'s gating logic
  internally (KYC modal + billing modal). Single `Retirer` button —
  disabled when `available_cents === 0`.
- `web/src/features/wallet/components/wallet-withdraw-result-modal.tsx` —
  NEW. Renders 207 partial-success detail.
- `web/src/features/wallet/components/wallet-unified-history.tsx` — NEW.
  Renders the merged timeline with type icon + amount + status badge +
  "Charger plus" cursor pagination.
- `web/src/features/wallet/components/wallet-status-badge.tsx` — NEW
  helper to map a status string to a colored pill.
- `web/src/features/wallet/components/wallet-page.tsx` — STAY (legacy
  composition kept for the rest of the page chrome, e.g. the
  `CurrentMonthAggregate` cell). Page-level swap.
- `web/src/app/[locale]/(app)/wallet/page.tsx` — wire to
  `WalletUnifiedPage`.

The old `WalletPayoutSection`, `WalletTransactionsList`,
`WalletCommissionList` files remain in place (mobile + other surfaces
may import them — but `wallet-page.tsx` no longer composes them). They
are NOT deleted under scope discipline; a follow-up can prune.

### Referral page (commits 5 + 6)

- `web/src/features/referral/components/referral-missions-section.tsx` —
  REWORK `AttributionRow` to:
  - Show clear `provider_name` + `client_name` lines (via
    `intro_snapshot` extended with names if exposed) when viewer is owner.
    NOTE: today `IntroSnapshot` does NOT carry names — `referral.provider_id`
    is just a UUID. For the unmask, the existing anonymised cards
    (`AnonymizedProviderCard`, `AnonymizedClientCard`) on the parent
    `ReferralDetailView` already show snapshot fields. We surface NAMES
    only when the viewer is the owner (referrer): show
    "Mise en relation entre **Provider Name** et **Client Name**" — but
    we have no name field in `Referral`. **Pivot to using the existing
    AnonymizedProvider/Client cards' content, conditionally toggling
    masked vs clear rendering.** The AnonymizedProviderCard receives a
    `snapshot` prop today; we wire a new `revealIdentities` prop that
    toggles between the masked SVG bar and a "Public Profile" link with
    the user's display name from the `referrer`-friendly endpoint.

  After re-reading the components: the masking happens INSIDE
  `AnonymizedProviderCard` / `AnonymizedClientCard`. The fix lives there.

- `web/src/features/referral/components/anonymized-provider-card.tsx` —
  ADD `revealed: boolean` prop. When `revealed === true`, render a clear
  link to `/freelances/{provider_id}` (or agencies) with the actual name.
  When `false` (default), keep the existing red-bar mask.
- `web/src/features/referral/components/anonymized-client-card.tsx` —
  same treatment.
- `web/src/features/referral/components/referral-detail-view.tsx` — pass
  `revealed={viewerRole === "referrer"}` to both cards.

NOTE: name lookup. The snapshot does not carry a name. For Run C we
render the existing snapshot fields without a name (no extra fetch).
The "reveal" simply removes the red-bar overlay and replaces the
"Identité masquée" line with a link "Voir le profil" pointing to
`/freelances/{id}` and `/enterprises/{id}`. Acceptable per scope —
final UX detail (showing display name) needs an extra `useUser(id)`
fetch which is unscoped.

- For the C.2.b "per-milestone projection" replacement of `0 €`:
  rework `AttributionRow.CommissionColumn` to optionally render a
  per-milestone breakdown sub-list (toggle-friendly, default hidden to
  keep the row compact). Each row pulls per-milestone commissions via
  the existing `useReferralCommissions` filtered by `attribution_id`.
  States per milestone:
    - `paid` → `+X € reçue` green badge
    - `pending_kyc` / `pending` → `X € en attente` orange badge
    - `failed` → `X € échouée` red badge
    - `cancelled` / `clawed_back` → skip (per brief)
  Plus a synthetic "escrowed" line driven by
  `attribution.escrow_commission_cents` when > 0 (projected funds not
  yet released). Display "≈ X € (en séquestre)" in muted italic.

  Behavioural rule from brief: render NOTHING for milestones with status
  draft/rejected/cancelled — but commission rows don't have those
  statuses (those map to proposal milestone statuses). The mapping in
  the UI is fine because our commission `status` enum already excludes
  "draft" / "rejected" — commissions are only created from approved
  milestones. So the gate is implicit.

### End-intro modal (commit 6)

- `web/src/features/referral/components/end-intro-confirmation-modal.tsx`
  — NEW. Reuses `<Modal>` primitive. Two buttons (annuler / terminer
  définitivement). Calls `useEndIntroAttribution()` mutation.
- `web/src/features/referral/components/end-intro-action.tsx` — NEW.
  Encapsulates the button → modal → badge state machine. Owns the local
  `endedAt: string | null` so the badge replaces the button immediately
  after success (no page refresh).
- Wire into `referral-missions-section.tsx` as the row's trailing action
  (only rendered to referrer).

### i18n + e2e (commit 7)

- `web/messages/fr.json` — add `wallet.unified.*` + `referral.endIntro.*` +
  `wallet.history.*` namespaces. Tutoiement.
- `web/messages/en.json` — mirror EN translations.
- `web/e2e/wallet-unified.spec.ts` — Playwright spec per brief.

## Test plan (vitest)

| File | Cases | Coverage target |
|------|-------|-----------------|
| `wallet-unified-header.test.tsx` | render totals; disabled when avail=0; click triggers mutation | ≥ 90 % |
| `wallet-withdraw-result.test.tsx` | 200 → toast; 207 → modal with errors; 422 → KYC modal opens with onboarding URL | ≥ 90 % |
| `wallet-unified-history.test.tsx` | mixed types render; pagination via cursor; empty state | ≥ 90 % |
| `wallet-status-badge.test.tsx` | maps 4+ statuses to correct tone | 100 % |
| `anonymized-provider-card.test.tsx` (extended) | reveals when `revealed`; masks otherwise | ≥ 90 % |
| `anonymized-client-card.test.tsx` (extended) | reveals when `revealed`; masks otherwise | ≥ 90 % |
| `referral-projected-commissions.test.tsx` | per-status rendering matrix (paid / pending_kyc / failed / cancelled / escrowed) | 100 % |
| `end-intro-confirmation-modal.test.tsx` | open/close; cancel does not mutate; confirm fires mutation | ≥ 90 % |
| `end-intro-action.test.tsx` | shows button by default; badge replaces button after success; renders badge when `ended_at` initial | ≥ 90 % |
| `use-wallet-summary.test.ts` | hook fetches + caches; cursor flows through query key | ≥ 90 % |
| `use-end-intro-attribution.test.ts` | mutation invalidates correct keys; 403/404 propagate | ≥ 90 % |

## UX described (5-6 bullets)

- /wallet hero: ivoire surface, Fraunces title "Portefeuille", display
  total in Geist Mono, single corail pill "Retirer" — disabled state is
  ivory ghost.
- 3 stat cards below: identical layout to `/settings/billing-profile`
  CurrentMonthAggregate (ivory tile + 2-row content).
- Timeline rows: 💼 emoji for mission, 🤝 for commission (lucide icons),
  amount right-aligned tabular Geist Mono, status pill colored. Cursor
  pagination via "Charger plus" ghost button at the bottom.
- Referral /referrals/[id]: when owner, the two anonymised cards swap the
  red-bar overlay for a "Voir le profil →" link (corail). Snapshot fields
  remain visible underneath either way.
- Per-mission projection: sub-list rendered under the attribution row,
  bulleted "Jalon — status badge — amount" lines. Escrow line is muted
  italic; paid is green; pending orange; failed red.
- End-intro button: ghost destructive pill at the bottom of each row
  (referrer-only). Click → modal with French copy + confirm/cancel.
  After confirm, the button is replaced by a green badge "Intro terminée
  le DD/MM/YYYY".

## Commits

1. `_plan_run_c.md` (this file).
2. Hooks + api types regen (regenerated `api.d.ts` via
   `npm run generate-api:offline`).
3. Wallet unified header + 3 cards + Retirer button + tests.
4. Wallet history timeline unified + tests.
5. Referral identity unmask + projection per milestone + tests.
6. End-intro modal + badge + tests.
7. Playwright spec + i18n bundles.

Mobile + admin + backend + legal untouched.
