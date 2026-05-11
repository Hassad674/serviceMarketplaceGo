# WALLET-UNIFY Run D — Plan

Last run of the WALLET-UNIFY chain. Run A (referrals foundation), B (backend
wallet surfaces), C (web UI) already merged. Run D ships:

1. The 5-LOC backend `AttributionResponse.ended_at` fix (unblocks Run C web).
2. Mobile parity: unified wallet screen + referral identity reveal +
   end-intro modal + projected commissions list, all matching Run C web.
3. Mobile-side i18n via ARB.
4. Tests at every layer (>= 90% coverage on new code).

---

## D.1 — Backend `AttributionResponse.ended_at` (5 LOC)

**Files**:

- `backend/internal/handler/dto/response/referral.go`:
  - Add `EndedAt *string `json:"ended_at,omitempty"`` to
    `AttributionResponse` struct.
  - In `NewAttributionListFromStats`, populate from `r.Attribution.EndedAt`
    using `r.Attribution.EndedAt.UTC().Format(time.RFC3339)` when non-nil.
- `backend/internal/handler/dto/response/referral_attribution_ended_at_test.go`
  (NEW): table-driven test for the mapper covering
    - `EndedAt = nil` → field omitted from JSON output.
    - `EndedAt = non-nil` → field present, RFC3339 formatted, isClient
      strips commission rate but keeps `ended_at`.
- Regenerate OpenAPI golden: `go test ./internal/handler/ -run TestOpenAPISchemaShape_Snapshot -update-openapi-golden`.
- Regenerate web types: `cd web && npm run generate-api:offline`.

**Why pointer to string**: the existing `*time.Time` would emit an empty
struct (`{}` when unmarshalled badly) — using `*string` mirrors the
other timestamp fields already on the DTO and gives `nil` → field
omitted via `omitempty`.

**Test count**: 1 file, 2 table-driven cases + 1 isClient-with-ended_at
case = 3 cases.

---

## D.2 — Mobile wallet unified screen

**Files** (new):

- `mobile/lib/features/wallet/domain/entities/wallet_summary_entity.dart`
  - `WalletSummaryEntity` (currency, total_cents, available_cents,
    escrowed_cents, transmitted_cents, breakdown.missions, breakdown.commissions,
    recent_transactions[], next_cursor).
  - `WalletSummaryLeg` (total/available/escrowed/transmitted cents).
  - `WalletSummaryTransaction` (type, amount_cents, currency, status,
    mission_title?, occurred_at, reference_id).
  - `WithdrawResult` + `WithdrawLegError`.
  - `formatEurCents` helper (kept locally so the file is self-contained).
- `mobile/lib/features/wallet/domain/repositories/wallet_repository.dart`:
  - Add `fetchSummary({String? cursor})` and `withdraw({int? amountCents})`
    methods to the abstract.
- `mobile/lib/features/wallet/data/wallet_repository_impl.dart`:
  - Implement the two new methods against `/api/v1/wallet/summary` (GET)
    and `/api/v1/wallet/withdraw` (POST).
  - On 422 kyc_required → throw `CommissionKYCRequiredException` (reuse
    existing exception type).
- `mobile/lib/features/wallet/presentation/providers/wallet_provider.dart`:
  - Add `walletSummaryProvider` (FutureProvider.family<WalletSummary, String?>
    keyed on cursor).
- `mobile/lib/features/wallet/presentation/widgets/wallet_unified_header.dart` (NEW):
  - HeroCard: Fraunces title, big total amount, single Retirer button.
  - 3 stat cards: en séquestre / disponible / transmis.
- `mobile/lib/features/wallet/presentation/widgets/wallet_unified_history.dart` (NEW):
  - List of rows with type icon (briefcase or handshake), title, amount,
    status badge. "Charger plus" button when next_cursor present.
- `mobile/lib/features/wallet/presentation/widgets/wallet_withdraw_result_sheet.dart` (NEW):
  - Bottom sheet shown on 207 partial-success — breakdown + errors[].

**Replace** `wallet_screen.dart` body to use the new header + history.
Remove the per-row Retirer button on commission rows in the new history
view (commissions become read-only tiles).

### ASCII layout

```
┌─────────────────────────────────────┐
│  💰 Portefeuille                    │   ← Hero card
│  Tes revenus issus des missions...  │
│                                     │
│  TOTAL DES REVENUS                  │
│  3 116 €                            │
│                                     │
│  ┌───────────────────┐              │
│  │  Retirer 410 €    │              │   single CTA
│  └───────────────────┘              │
└─────────────────────────────────────┘

┌──────────┐ ┌──────────┐ ┌──────────┐
│ En       │ │Disponible│ │ Transmis │
│ séquestre│ │          │ │          │
│ 1500 €   │ │  410 €   │ │ 1206 €   │
└──────────┘ └──────────┘ └──────────┘

┌─────────────────────────────────────┐
│ Historique des revenus              │
│ ──────────────────────────────────  │
│ 🤝  Mission X       +1 616 €  Reçu  │
│ 💼  Logo design     +500 €    Reçu  │
│ ...                                  │
│           [Charger plus]            │
└─────────────────────────────────────┘
```

**Test count**:
- `wallet_summary_entity_test.dart` — JSON parse table (full payload,
  empty arrays, missing fields, malformed types) = 4 cases.
- `wallet_unified_header_test.dart` — renders title + total + 3 cards,
  Retirer disabled when available=0, Retirer pending shows spinner = 4 cases.
- `wallet_unified_history_test.dart` — empty state, mission row, commission
  row, charger-plus visible only with next_cursor = 4 cases.
- `wallet_withdraw_result_sheet_test.dart` — 200 toast, 207 sheet
  with errors, 422 dialog with onboarding url = 3 cases.
- `wallet_repository_impl_summary_test.dart` — GET + POST happy paths +
  422 throws KYC exception = 3 cases.

---

## D.3 — Mobile referral identity + projection + end-intro

**Files** (new):

- `mobile/lib/features/referral/presentation/widgets/referral_identity_card.dart`:
  - When `viewerId == referral.referrerId` (the apporteur, "is_owner"
    on web), show clear provider + client names + tap-to-profile link.
  - Otherwise reuse `AnonymizedProviderCard` / `AnonymizedClientCard`.
- `mobile/lib/features/referral/presentation/widgets/projected_commissions_list.dart`:
  - Per-status pill (paid/pending/escrowed/failed).
  - Filters out cancelled/clawed_back rows.
  - Adds a synthetic "≈ X € en séquestre" line when escrow_commission_cents > 0
    on the attribution.
- `mobile/lib/features/referral/presentation/widgets/end_intro_confirmation_dialog.dart`:
  - AlertDialog with title + body + Cancel / Confirmer destructive button.
- `mobile/lib/features/referral/presentation/widgets/end_intro_action.dart`:
  - Button → dialog → mutation → badge state machine.
- Repository: add `endAttribution(String attributionId)` returning the
  ended_at string.
- Provider: add `endIntroAttributionProvider` (one-shot action).
- `referral_detail_screen.dart` — swap `AnonymizedProviderCard` /
  `AnonymizedClientCard` for `ReferralIdentityCard`. Below the missions
  section, render `ProjectedCommissionsList` per attribution. Below
  each attribution, render `EndIntroAction` when the viewer is the
  referrer.

**Test count**:
- `referral_identity_card_test.dart` — viewer is owner (clear), viewer
  is provider (masked client), viewer is client (masked provider) = 3 cases.
- `projected_commissions_list_test.dart` — paid / pending / failed /
  escrow line / empty = 5 cases.
- `end_intro_confirmation_dialog_test.dart` — opens, cancel closes,
  confirm fires callback = 3 cases.
- `end_intro_action_test.dart` — open dialog, confirm → badge appears,
  initialEndedAt → badge directly = 3 cases.

---

## D.4 — i18n (ARB)

`mobile/lib/l10n/app_fr.arb` + `app_en.arb`: mirror web keys.

```
walletUnified.title
walletUnified.subtitle
walletUnified.totalEarned
walletUnified.withdraw
walletUnified.withdrawing
walletUnified.noFunds
walletUnified.card.{escrowed,escrowedHint,available,availableHint,transmitted,transmittedHint}
walletUnified.toast.{success,partial}
walletUnified.result.{title,drained,missionsLine,commissionsLine,errorsHeading,errorMissions,errorCommissions,close}
walletUnified.history.{title,subtitle,loadMore,empty}
walletUnified.history.row.{mission,commission,untitled}
walletUnified.history.status.{paid,pending,escrowed,failed}

referralEndIntro.ctaLabel
referralEndIntro.modal.{title,body,fallbackProvider,fallbackClient,cancel,confirm}
referralEndIntro.badge
referralEndIntro.error.{forbidden,notFound,generic}

referralIdentity.reveal.{providerLink,clientLink}

referralProjection.perMilestoneTitle
referralProjection.status.{escrowed,pending,paid,failed}
referralProjection.empty
```

ARB keys are added at the end of the existing file. `flutter gen-l10n`
regenerates `app_localizations*.dart`.

---

## D.5 — Decision summary

- **Pointer-to-string for EndedAt on AttributionResponse** rather than
  `*time.Time` — mirrors how `paid_at` already does it on `CommissionResponse`,
  emits `nil` → omitted via `omitempty`.
- **Reuse `CommissionKYCRequiredException`** for the 422 branch on
  withdraw — same exception class, same dialog widget. No new exception
  type.
- **Reuse `CommissionKYCRequiredDialog`** widget for the KYC modal.
- **`Wrap` vs `Row` for stat cards** — `GridView.count(3)` would force
  a fixed 1:1 ratio. Use `Row(children: [Expanded, Expanded, Expanded])`
  with `IntrinsicHeight` to match heights, no scroll.
- **Per-attribution end-intro vs whole-referral terminate**: the
  existing `terminate` action is preserved on the parent referral
  (whole intro). The NEW `EndIntroAction` works on ONE attribution.
- **Mobile uses Riverpod, not TanStack Query**, so the cache
  invalidation pattern is `ref.invalidate(walletSummaryProvider)`
  after withdraw success.
- **Authoritative user-id**: read from `ref.watch(authProvider).user?['id']`
  to compare with `referral.referrerId`.

## D.6 — Build verification

```
cd backend && go build ./... && go vet ./... && go test ./internal/handler/...
cd web    && npm run generate-api:offline && npx tsc --noEmit
cd mobile && flutter analyze lib/features/wallet/ lib/features/referral/ test/features/wallet/ test/features/referral/
cd mobile && flutter test test/features/wallet/ test/features/referral/
cd mobile && flutter test --coverage test/features/wallet/ test/features/referral/
cd mobile && flutter build apk --debug --target-platform android-arm64 2>&1 | tail -10
```

ALL green — paste stdout in the final report. No `.skip`, no
`@ts-ignore`, no `// ignore_for_file:` widening.
