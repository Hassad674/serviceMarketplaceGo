# W-20 · Profil de facturation — visual diff notes

## Source
- `design/assets/sources/phase1/soleil-lotB.jsx` § `SoleilBillingProfile`

## Touched files

### Web (W-20)
- `web/src/features/invoicing/components/billing-profile-form.tsx`
- `web/src/features/invoicing/components/billing-section-address.tsx`
- `web/src/features/invoicing/components/billing-section-fiscal.tsx`
- `web/src/features/invoicing/components/billing-section-legal-identity.tsx`
- `web/src/features/invoicing/components/address-autocomplete.tsx`
- `web/src/shared/components/billing-profile/billing-profile-completion-modal.tsx`
- `web/src/shared/components/billing/fee-preview.tsx`
- `web/src/app/[locale]/(app)/billing/success/page.tsx`
- `web/src/app/[locale]/(app)/billing/cancel/page.tsx`
- `web/messages/fr.json` and `en.json` (new keys, prefix `billingProfile_*`)

### Mobile (fee preview widget)
- `mobile/lib/features/billing/presentation/widgets/fee_preview_widget.dart`

## What changed visually

### Billing profile form (`/settings/billing-profile`, `variant="page"`)
- Editorial Soleil header above the form: corail mono eyebrow
  (`ATELIER · PROFIL DE FACTURATION`), Fraunces 26/32px display title
  with italic corail accent ("Renseigne ton *profil de facturation.*"),
  tabac subtitle.
- All section cards switched from slate/dark-mode classes to Soleil
  tokens (`bg-surface`, `border-border`, `text-foreground`,
  `text-muted-foreground`). Section heading is now Fraunces 18px
  (`font-serif`).
- Inputs: `rounded-xl`, sable border, corail focus ring
  (`focus:border-primary focus:ring-primary/15`).
- Profile-type radio: corail-bordered tile with corail-soft fill on
  selected, 18px ring + 8px corail dot mirroring the maquette. Native
  radio is now `sr-only` for accessibility while the styled tile
  carries the visual.
- Address autocomplete: sable border, corail focus, corail-soft hover
  on dropdown rows. Disabled-state hint uses sable dashed border on
  ivoire bg.
- "Pré-remplir depuis Stripe" + "Valider mon n° TVA" pills →
  rounded-full, sable border, corail-soft hover.
- Save button → corail rounded-full pill with corail-deep hover
  shadow; Cancel/back to ghost outline pill.
- Missing-fields banner → amber-soft fill with warning icon.
- Save error → destructive token text; Save success → success token
  text.

### Billing profile completion modal (gate)
- Modal renders inside the shared `Modal` primitive (already ivoire
  rounded-2xl).
- Warning banner switched to amber-soft + warning icon.
- Missing-fields list: bullet markers now corail (`bg-primary`).
- "Plus tard" → ghost outline pill; "Compléter mon profil" → corail
  rounded-full pill with arrow.

### Fee preview (`<FeePreview />` shared)
- Card frame: ivoire `bg-surface`, `rounded-2xl`, sable border, p-6.
- Header: 40×40 corail-soft (or sapin-soft when subscribed) icon disc,
  Fraunces 18px title.
- Tier rows: rounded-2xl outer, sable divider, corail-tinted active
  row (corail-soft bg + corail-deep label + 4px corail left bar).
- Numerals (`fee_cents`, totals) all in `font-mono`.
- Premium notice → success-soft fill.
- Skeletons → ivoire shimmer on sable bg.

### `/billing/success` and `/billing/cancel`
- Centered ivoire hero card (rounded-2xl, sable border) replacing the
  old slate/emerald gradients.
- Eyebrow + Fraunces title with italic corail accent on the key word.
- Success → corail-soft icon disc with corail-deep glyph; pending →
  same disc with spinner; timeout → amber-soft + warning glyph;
  cancel → amber-soft + warning glyph.
- CTA → corail rounded-full pill with `ArrowRight` glyph (timeout
  variant kept ghost outline because it's a soft "go back to wait"
  rather than the primary path).
- All visible strings now flow through `useTranslations("billingProfile")`
  with new keys prefixed `billingProfile_w20_*`.

### Mobile fee preview widget
- Card: ivoire `colorScheme.surface`, `radius2xl` border, `AppColors`
  border tokens (no `Color(0xFF...)` literals).
- Header: 36px corail-soft (`accentSoft`) disc with corail-deep
  receipt icon, Fraunces (`SoleilTextStyles.titleMedium`) "Platform
  fees" + caption tabac subtitle.
- Tier grid: rounded-xl outer with sable border, divider rows,
  corail-soft active background + 4px corail-deep left bar +
  corail-deep label/amount on the active tier.
- Numerals (`monoLarge`) for amounts. Net amount in `monoLarge` corail
  for the single-amount mode, plain `mono` for milestone rows.
- Skeleton + error states use Soleil radius tokens (`radiusLg`).
- Strings stay English-only per `mobile/CLAUDE.md` ("English-language
  UI strings"); no new ARB keys were added.

## Behaviour preserved (zero regression)

- All react-hook-form + zod logic loaded from
  `billing-profile-form.schema.ts` (OFF-LIMITS) is unchanged.
- All update/sync/validate mutations are unchanged.
- All loading / error / empty states render through the same code
  paths.
- Role gate (`viewer_is_provider`) on the fee preview is unchanged —
  still fails closed.
- Zero new hooks, mutations, or repositories.
- `address-autocomplete.tsx` BAN endpoint logic + debounce + dropdown
  state machine untouched.

## Out-of-scope flagged (NOT implemented)

None. The maquette only shows the form structure already present in
the repo.

## Validation

- `npx tsc --noEmit`: clean.
- `npx vitest run`: 2022/2022 passing.
- `npm run build`: success (only the standard Stripe SSR ConnectJS log).
- `flutter analyze --no-pub lib/features/billing`: no issues.
- `flutter test`: no test exists under `test/features/billing/` (per
  brief: "may not exist").
- Scope check: `scoped clean`.

## Screenshots

`before-*.png` / `after-*.png` not produced in this round (Linux-only
host without a Stripe sandbox to drive `/billing/success` end-to-end).
The Soleil tokens used here are the same ones validated in W-09 / W-07
batches that already shipped to main.
