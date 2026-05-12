# Plan — Referral Apporteur Page Improvements

## Scope (exact)

On `/fr/referrals/[id]` (apporteur owner view):

1. **Identity cards** — when `revealed === true` (apporteur owner):
   - Render only the display name in a clean, minimalist card (Soleil v2 Fraunces for the name).
   - REMOVE the `Voir le profil` button (no link).
   - REMOVE the `Identité visible (tu es l'apporteur)` eyebrow label.
   - REMOVE the masking explainer text `L'apporteur a choisi de ne révéler...` and the masked-field grid.
   - Masked view (other viewers): unchanged.

2. **Mission list** — for each row in `referral-missions-section`:
   - Show `total_amount_cents` formatted as `1 230 €` near the title.
   - Keep existing projection badges + status pill + milestone progress.

## Display-name resolution (decision)

- **Provider display name**:
  - If the provider user has an organization of type `agency`, use `organizations.name`.
  - Otherwise (individual freelancer / no org), use `users.first_name + " " + users.last_name` (FullName).
- **Client display name**:
  - Look up the client user's organization (`enterprise` or `agency`), use `organizations.name`.
  - Fallback to user FullName if no org.

## Files to touch

### Backend (Go)
- `backend/internal/handler/dto/response/referral.go` — extend `ReferralResponse` with `ProviderDisplayName`, `ClientDisplayName`. Extend `AttributionResponse` with `TotalAmountCents`.
- `backend/internal/handler/referral_handler.go` — wire the new fields (Get + ListAttributions).
- `backend/internal/app/referral/service.go` — extend `ServiceDeps` with a `PartyDisplayNameResolver` port + extend `AttributionWithStats` with `TotalAmountCents`.
- `backend/internal/app/referral/service_list.go` — fill `TotalAmountCents` from proposal summary.
- `backend/internal/app/referral/proposal_summary_resolver.go` — add `AmountCents` field to `ProposalSummary`.
- `backend/internal/app/referral/wiring_adapters.go` — populate `AmountCents` from `proposal.Amount`. Add `PartyDisplayNameResolver` port + concrete impl reading users + organizations.
- `backend/cmd/api/main.go` — wire the new resolver.
- `backend/internal/handler/testdata/openapi.golden.json` — regenerate.

### Web (TypeScript)
- `web/src/shared/types/referral.ts` — add `provider_display_name`, `client_display_name` to `Referral`; add `total_amount_cents` to `ReferralAttribution`.
- `web/src/shared/types/api.d.ts` — regenerate.
- `web/src/features/referral/components/anonymized-provider-card.tsx` — when `revealed`, render only the display name card.
- `web/src/features/referral/components/anonymized-client-card.tsx` — same.
- `web/src/features/referral/components/referral-detail-view.tsx` — pass `displayName` prop to cards.
- `web/src/features/referral/components/referral-missions-section.tsx` — render total amount alongside title.
- `web/messages/{fr,en}.json` — drop `referralIdentity.reveal.{providerLink,clientLink}`; add `referralMissions.totalAmount`.

### Mobile (Flutter)
- `mobile/lib/features/referral/presentation/widgets/referral_identity_card.dart` — drop the eyebrow label + fallback fields when owner.
- `mobile/lib/features/referral/presentation/widgets/referral_missions_section.dart` — render total amount.
- `mobile/lib/features/referral/domain/entities/referral_entity.dart` — add fields (provider_display_name, client_display_name, total_amount_cents).
- `mobile/lib/l10n/app_{fr,en}.arb` — drop unused, add new total amount key.

### Tests
- Web vitest: extend `anonymized-card-reveal.test.tsx` for the new revealed mode + add a `referral-missions-section` mission total test.
- Backend: add test in handler test verifying `provider_display_name`, `client_display_name`, `total_amount_cents` in response shape.
- Mobile: widget tests for simplified cards + mission row with total.

## Test count target

- Backend: 3 new test cases (display names individual/org cases; mission total cents).
- Web: 4-5 new test cases (revealed-only-name for provider + client, masked unchanged, mission total visible, no button/no badge).
- Mobile: 3 new widget tests (simplified card revealed, masked unchanged, mission row with total).

## Coverage target

≥ 90% on all touched code per instructions.

## Commits (logical)

1. `_plan_referral_apporteur.md` (this)
2. Backend: display names + mission totals in DTO + tests + golden refresh + types regen
3. Web: simplified identity cards + mission total + tests
4. Mobile: parity + tests
