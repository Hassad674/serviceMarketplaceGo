# BATCH-PROFIL-FACTURATION — W-20 (web) + mobile billing widget

> Worktree: `/tmp/mp-profil-facturation` · Branch: `feat/design-profil-facturation` · Base: `origin/main` (487cc6e1)

## Goal
Port Profil de facturation (billing profile form: address + SIRET + fiscal + legal identity for issuing invoices) to Soleil v2:
- **W-20** Profil facturation form, modal entry, billing sections + the billing redirect pages (`/billing/success`, `/billing/cancel`) + fee preview
- Mobile parity: `mobile/lib/features/billing/presentation/` (fee preview widget exists; billing profile form on mobile may or may not exist — read first)

## TOUCHABLE files

### Web
- `web/src/features/invoicing/components/billing-profile-form.tsx`
- `web/src/features/invoicing/components/billing-profile-completion-modal.tsx`
- `web/src/features/invoicing/components/billing-section-address.tsx`
- `web/src/features/invoicing/components/billing-section-fiscal.tsx`
- `web/src/features/invoicing/components/billing-section-legal-identity.tsx`
- `web/src/features/invoicing/components/address-autocomplete.tsx`
- `web/src/features/invoicing/components/eu-countries.ts` (small constants helper)
- `web/src/features/billing/components/fee-preview.tsx`
- `web/src/app/[locale]/(app)/billing/success/page.tsx`
- `web/src/app/[locale]/(app)/billing/cancel/page.tsx`
- `web/messages/fr.json` and `en.json` — NEW keys ONLY, prefix `billingProfile_*`

### Mobile
- `mobile/lib/features/billing/presentation/widgets/fee_preview_widget.dart`
- Any other presentation files under `mobile/lib/features/billing/presentation/` (read first)
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` — NEW keys ONLY, prefix `billingProfile_*`

## OFF-LIMITS — STRICT
- `web/src/features/invoicing/components/{invoice-list, current-month-aggregate, missing-fields-copy}.{tsx,ts}` (sibling D2 Factures)
- `web/src/features/invoicing/components/billing-profile-form.schema.ts` (schema — OFF-LIMITS for everyone, even though name suggests it's part of this batch — schema files are OFF-LIMITS by design rules)
- `web/src/app/[locale]/(app)/invoices/**` (sibling D2)
- `web/src/features/proposal/**`, `web/src/features/wallet/**` (siblings + merged)
- `web/src/app/[locale]/(app)/projects/**` (sibling D1 Proposal flow)
- `mobile/lib/features/invoice/**`, `mobile/lib/features/proposal/**`, `mobile/lib/features/mission/**`, `mobile/lib/features/wallet/**`
- `mobile/lib/features/billing/data/**`, `mobile/lib/features/billing/domain/**`, `mobile/lib/features/billing/presentation/providers/**`
- Anything under `backend/`
- All `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`, `shared/lib/api-client.ts`
- `web/src/features/invoicing/api/**`, `web/src/features/invoicing/hooks/**`
- `web/src/features/billing/api/**`, `web/src/features/billing/hooks/**`
- `package.json`, `pubspec.yaml`, lockfiles
- All existing `*.test.tsx`, `*_test.dart`, `__tests__/`
- Generated `app_localizations*.dart`

## Acceptance criteria

### W-20 Billing profile form
- Editorial header in the modal: corail mono eyebrow "ATELIER · PROFIL DE FACTURATION", Fraunces title with italic corail accent ("Renseigne ton *profil de facturation*."), tabac subtitle
- Form sections (collapsed accordion or stacked on a single page — match existing UX):
  - **Identité légale** (`billing-section-legal-identity.tsx`): name + SIRET + raison sociale, Soleil inputs (rounded-xl, sable-foncé border, corail focus)
  - **Adresse** (`billing-section-address.tsx` + `address-autocomplete.tsx`): autocomplete with corail dropdown
  - **Fiscal** (`billing-section-fiscal.tsx`): TVA + régime, Soleil pills if applicable
- Modal frame: ivoire bg, rounded-2xl, border, max-h with internal scroll
- Submit pill: corail rounded-full "Enregistrer", ghost outline "Annuler"
- Form behavior: ALL react-hook-form + zod (loaded from `billing-profile-form.schema.ts` which is OFF-LIMITS) STAYS EXACTLY THE SAME

### Fee preview (web `fee-preview.tsx` + mobile `fee_preview_widget.dart`)
- Soleil card: ivoire bg, rounded-2xl, padding, Fraunces "Aperçu des frais" title, breakdown rows (label + Geist Mono amount), corail accent for total

### Billing redirect pages (`/billing/success` + `/billing/cancel`)
- Soleil hero card: corail-soft icon plate (success: corail check, cancel: amber-soft warning), Fraunces title, body, corail pill CTA back to wallet

## Validation pipeline

```bash
cd /tmp/mp-profil-facturation
git diff --name-only origin/main...HEAD | grep -E "^(backend/|.*\.test\.|.*_test\.|features/proposal/|features/wallet/|features/invoicing/(api|hooks|__tests__|components/(invoice-list|current-month-aggregate|missing-fields-copy)|billing-profile-form\.schema)|features/billing/(api|hooks)|app/\[locale\]/\(app\)/(invoices|projects)/|mobile/lib/features/(invoice|proposal|mission|wallet|billing/(data|domain|presentation/providers)))" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"
cd web && npm ci && npx tsc --noEmit && npx vitest run src/features/invoicing src/features/billing src/app/\[locale\]/\(app\)/billing && npm run build
cd ../mobile && flutter pub get && flutter analyze --no-pub lib/features/billing && flutter test --no-pub test/features/billing/ 2>&1 || echo "(may not exist)"
cd .. && bash design/scripts/check-api-untouched.sh && bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3.

## Quality bar
Same as siblings. ZERO touch to `billing-profile-form.schema.ts`. ZERO new providers/repos.

## Push + PR
- Message: `feat(design/profil-facturation): port W-20 web + mobile billing widget to Soleil v2`
- PR title: `[design/web/W-20+mobile] Port Profil facturation to Soleil v2`

## Final report (under 700 words)
Standard structure. Visual diffs `design/diffs/W-20/`.
