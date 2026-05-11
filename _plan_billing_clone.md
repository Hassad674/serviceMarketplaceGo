# BILLING-IDENTITY-CLONE — plan

Branch: `feat/billing-identity-clone`
Worktree: `/home/hassad/serviceMarketplaceGo/.claude/worktrees/agent-ad25a6fe6fdaafc51`

## Objective

The client payment page at `/fr/projects/pay` currently renders a bad
inline mini-form (`PaymentBillingIdentitySection`) capturing only
SIRET/TVA/legal_name above the Stripe Payment Element. Replace it with
the full prestataire `BillingProfileForm` (pays + adresse + identité +
identifiants fiscaux), pre-filled from the backend, with a compact
read-only summary when complete. Mirror on mobile (Flutter).

The cleanup deletes the mini-form and its helper `persistInlineBillingIdentity`,
plus the matching tests.

## Discoveries / decisions

- `BillingProfileForm` already lives in `web/src/features/invoicing` —
  reusable via the shared `BillingProfileInlineModal` bridge or by
  importing it directly from an `app/` page. Within the proposal
  feature however we must NOT import features/invoicing — feature →
  feature is forbidden. We therefore wrap the form inside a new shared
  component (`@/shared/components/billing-profile/billing-profile-embed.tsx`)
  that the proposal feature imports. Same architectural pattern as the
  existing `BillingProfileInlineModal`.
- `useBillingProfile()` lives in shared (`@/shared/hooks/billing-profile/...`),
  reachable from both shared and the proposal feature.
- The backend already returns `is_complete` on the profile snapshot —
  no new completeness helper is needed for the page logic. The brief
  asks for `checkBillingProfileComplete`; we'll add it as a TINY pure
  helper in `web/src/shared/lib/billing-profile/billing-profile-complete.ts`
  that mirrors the backend's contract (mostly defensive, used by tests).
- Mobile: `billing_profile_inline_sheet.dart` already exists and the
  mobile payment screen already opens it on a 412. The brief wants the
  embedded form (not a sheet behind a CTA). On mobile we replace the
  CTA + sheet pattern with an inline embedded form widget, mirroring
  web.

## Files DELETED

- `web/src/features/proposal/components/payment-billing-identity-section.tsx`
- `web/src/features/proposal/components/__tests__/payment-billing-identity-section.test.tsx`

## Files MODIFIED

- `web/src/features/proposal/components/payment-simulation.tsx`
  - Remove imports of `PaymentBillingIdentitySection` and
    `persistInlineBillingIdentity`.
  - Remove the `useRef<PaymentBillingIdentityValues>`, `handleBillingChange`,
    and the `persistInlineBillingIdentity` step in `handleSubmit`.
  - Remove the `<PaymentBillingIdentitySection onChange={...} />` JSX.
  - Add `BillingProfileEmbed` from the new shared component above
    the Stripe Elements wrapper. When the profile is not complete, hide
    the Stripe Elements UI and show only the embed in edit mode. When
    complete, render the embed in read-only summary mode.
- `web/src/features/proposal/components/__tests__/payment-simulation.test.tsx`
  - Add mocks for the new `BillingProfileEmbed` and `useBillingProfile`
    so the existing tests still pass (they assume the legacy mini-form
    is absent).

## Files ADDED

- `web/src/shared/components/billing-profile/billing-profile-embed.tsx`
  - New client component that renders one of two states:
    1. `summary` — compact read-only card with pays + adresse + entité +
       identifiants fiscaux + "Modifier" button (BillingProfileSummary
       internal component).
    2. `form` — full `BillingProfileForm` (imported from `features/invoicing`,
       same pattern as `BillingProfileInlineModal`), collapses back to
       summary after successful save.
  - Owns the edit/view toggle state. Defaults to summary when profile
    `is_complete`, to form when incomplete.
- `web/src/shared/components/billing-profile/billing-profile-summary.tsx`
  - Pure presentational component rendering the saved profile as a
    Soleil v2 card. Displays pays, adresse, entité légale, SIRET, TVA.
    Single "Modifier" CTA.
- `web/src/shared/lib/billing-profile/billing-profile-complete.ts`
  - Pure `checkBillingProfileComplete(profile)` helper. Required:
    legal_name, country, address_line1, postal_code, city. Required if
    profile_type==='business': tax_id. Returns boolean.
- `web/src/shared/components/billing-profile/__tests__/billing-profile-embed.test.tsx`
- `web/src/shared/components/billing-profile/__tests__/billing-profile-summary.test.tsx`
- `web/src/shared/lib/billing-profile/__tests__/billing-profile-complete.test.ts`
- `web/e2e/payment-billing-clone.spec.ts` — Playwright e2e (gated on
  Stripe-less simulation mode + backend running; if backend isn't
  available we degrade to a smoke test with `describe.skip` per the
  existing pattern).

## Mobile changes

- `mobile/lib/features/proposal/presentation/screens/payment_simulation_screen.dart`
  - Replace the "OutlinedButton.icon (_openBillingSheet)" CTA + sheet
    with an inline embedded `BillingProfileEmbed` widget that mirrors
    web behaviour: read-only summary when complete, full form when
    incomplete, "Modifier" toggle.
  - Keep the existing 412 retry path intact (BillingProfileInlineSheet
    remains as fallback for race conditions).
- `mobile/lib/features/invoicing/presentation/widgets/billing_profile_embed.dart`
  - New widget that hosts the existing `BillingProfileForm` (or a
    summary card) per the same pattern as web. Mirrors the inline sheet
    but rendered inline rather than in a bottom sheet.
- Tests in `mobile/test/features/invoicing/` and
  `mobile/test/features/proposal/`.

## i18n keys ADDED

- web `messages/fr.json` + `messages/en.json`:
  - `proposal.billingEmbed.summaryTitle` — "Identité de facturation"
  - `proposal.billingEmbed.editCta` — "Modifier"
  - `proposal.billingEmbed.country` — "Pays"
  - `proposal.billingEmbed.address` — "Adresse"
  - `proposal.billingEmbed.entity` — "Entité légale"
  - `proposal.billingEmbed.tax` — "Identifiants fiscaux"
  - `proposal.billingEmbed.completePromptTitle` — "Renseigne ton identité de facturation"
  - `proposal.billingEmbed.completePromptBody` — "Avant de confirmer le paiement, complète les informations qui apparaîtront sur ton reçu."
- mobile ARB equivalents (FR-only since mobile is FR-only billing).

## Hard constraints respected

- Reuse existing components/hooks — `BillingProfileForm`,
  `useBillingProfile`, and the shared types are NOT forked.
- File size ≤ 600 LOC, function ≤ 50 LOC.
- All user-visible strings via `useTranslations` / ARB.
- Soleil v2 tokens only.
- Server Components stay server; the embed needs "use client" because
  it owns toggle state and calls TanStack Query.
- No `// @ts-ignore`, no test deletions to pass.
- No new npm dependency.
- No changes to backend, migrations, /legal/, LiveKit, admin SPA.
- `BillingProfileInlineModal` stays as the 412 fallback path — kept
  for race conditions where the sibling tab clobbers the profile.

## Out-of-scope (flagged, NOT done)

- Country-aware exotic switch (brief: FR-only + EU vat optional like
  today).
- VIES live validation rewrite — uses the existing `useValidateVAT`
  inside the form.
- Migration. Nothing on DB.

## Commit plan

1. `_plan_billing_clone.md` (this file).
2. Cleanup — delete bad files + remove imports from payment-simulation
   (compiles broken until step 3).
3. Add `checkBillingProfileComplete` + `BillingProfileSummary` + tests.
4. Add `BillingProfileEmbed` + wire into payment-simulation + vitest.
5. Playwright e2e + i18n keys.
6. Mobile parity (`billing_profile_embed.dart` + screen wire + tests).

Each commit goes through the validation pipeline before push.

## Run B coordination

Run B is on `feat/wallet-unify-run-b` editing
`backend/internal/app/referral/wallet_reader.go` +
`backend/internal/handler/wallet_handler.go::Summary/Withdraw`. Zero
overlap — we touch web (`features/proposal`, `shared/components/billing-profile`,
`shared/lib/billing-profile`, `app/[locale]/(app)/projects/pay/`), mobile,
and i18n. No backend Go files touched.
