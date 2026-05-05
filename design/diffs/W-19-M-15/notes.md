# W-19 Factures (web) + M-15 Factures (mobile) — Soleil v2 port

## What changed

### Web (W-19)

**`web/src/app/[locale]/(app)/invoices/page.tsx`**
- Replaced the slate `text-2xl font-bold` heading with the Soleil editorial header:
  - corail mono eyebrow `ATELIER · FACTURES`
  - Fraunces serif title `Tes` + italic corail `factures et reçus.`
  - tabac subtitle from `invoicesList.heroSubtitle`
- Server Component now uses `getTranslations("invoicesList")`.
- Layout container width unchanged (`max-w-4xl`); vertical rhythm bumped from
  `space-y-6` to `space-y-8` to match the Soleil editorial grammar.

**`web/src/shared/components/billing-profile/current-month-aggregate.tsx`**
- Card now uses semantic Soleil tokens (`bg-card`, `border-border`,
  `shadow-card`) instead of slate hex shades.
- Icon plate switched from `bg-rose-50/text-rose-600` to corail-soft
  (`bg-primary-soft text-primary`), 11×11 with rounded-2xl.
- Header now exposes a corail mono eyebrow `MOIS EN COURS` and the running
  total in Geist Mono semibold (`font-mono text-[22px]`) on the right.
- Body copy through `useTranslations("invoicesList")`. Plural-aware milestone
  count via ICU MessageFormat.
- Toggle button restyled as a Soleil pill (`rounded-full border border-border`).
- Detail rows live inside a nested rounded-2xl container with divide-y borders.

**`web/src/features/invoicing/components/invoice-list.tsx`**
- Card chrome: `bg-card`/`border-border`/`shadow-card`, `divide-y divide-border`.
- Section header: corail mono eyebrow `Toutes les factures` + tabac subtitle.
- Row anatomy: corail-soft icon plate · Geist Mono invoice-number pill ·
  source-type pill (sapin-soft Payée / amber-soft En attente / muted Avoir) ·
  source label · Geist Mono uppercase relative date · Geist Mono amount ·
  rounded-full download pill (border + icon + label).
- Relative date (`formatRelativeDate`) renders FR conversational labels
  (`à l'instant`, `il y a 12 min`, `il y a 3 h`, `hier`, `il y a 4 j`,
  `1 avr.`).
- Empty state: corail-soft icon plate, Fraunces title `Aucune facture
  archivée`, italic Fraunces body.
- Skeleton: Soleil card with three placeholder rows mirroring the real layout.
- Status pill mapping is derived from `source_type` because the Invoice
  type currently exposes no `status` field (no backend change in this batch).

**`web/messages/{fr,en}.json`** — added a `invoicesList` namespace with the
13 keys consumed by `page.tsx` and `current-month-aggregate.tsx`. No other
top-level namespace touched.

### Mobile (M-15) — SKIP+FLAG

`mobile/lib/features/invoice/` ships only the domain layer
(`domain/entities/invoice.dart`, `domain/repositories/invoice_repository.dart`)
— **no `presentation/` layer, no `data/` layer, no provider**. Per the brief
("SKIP+FLAG mobile and DO NOT create one — that requires new providers/repos
which are OFF-LIMITS"), we did not introduce any presentation widgets. No
mobile file was modified, no l10n key was added, and `flutter analyze
lib/features/invoice` reports `No issues found!`.

## Out-of-scope flagged (NOT shipped this batch)

- **Filter pills (sent / received / all)** — `/api/v1/me/invoices` exposes
  no filter parameter; the `useInvoices` hook accepts only a `cursor`
  argument. Implementing a pill row would require either a new query
  parameter on the backend or a new hook (both OFF-LIMITS). Flagged for a
  follow-up product/backend round.
- **Real `status` field on Invoice** — surfaced as a derivation from
  `source_type` (`subscription → Payée`, `monthly_commission → En attente`,
  `credit_note → Avoir`). Replace with a backend-driven status once the
  invoicing domain models it.
- **Mobile factures screen (M-15)** — no presentation layer exists;
  scaffolding one is OFF-LIMITS for this design batch.
- **Migration of `invoice-list.tsx` literals to `useTranslations`** — gated
  on a test update (the test renders without an `IntlProvider`); deferred
  to a follow-up batch that owns the test file.

## Files modified

```
web/messages/en.json
web/messages/fr.json
web/src/app/[locale]/(app)/invoices/page.tsx
web/src/features/invoicing/components/invoice-list.tsx
web/src/shared/components/billing-profile/current-month-aggregate.tsx
```

## Files NOT modified (off-limits respected)

- `web/src/features/invoicing/api/**` (transport)
- `web/src/features/invoicing/hooks/**` (TanStack Query)
- `web/src/features/invoicing/components/billing-*.tsx` (sibling D3)
- `web/src/features/invoicing/components/{address-autocomplete,eu-countries}.{tsx,ts}` (sibling D3)
- `web/src/features/invoicing/components/billing-profile-form.schema.ts` (schema)
- `web/src/features/invoicing/components/__tests__/*.test.tsx` (tests)
- `web/src/features/billing/**`, `web/src/app/[locale]/(app)/billing/**` (sibling D3)
- `web/src/features/proposal/**`, `web/src/app/[locale]/(app)/projects/**` (sibling D1)
- `mobile/lib/features/invoice/data|domain/**` (data + domain layers)
- `mobile/lib/features/{billing,proposal,mission,wallet}/**` (siblings)
- `backend/**`, all test files, all `package.json`/`pubspec.yaml`

## Visual reference

Source maquette: `design/assets/sources/phase1/soleil.jsx` (editorial header
grammar) + `design/assets/sources/phase1/soleil-app-lot5.jsx` (mobile
settings row referencing `Mes factures`). No dedicated `SoleilInvoices`
mock exists in the source; the implementation reuses the editorial grammar
established by W-12 Opportunités (`design/assets/sources/phase1/soleil-lotC.jsx`).

`before-android.png` / `after-android.png` not captured: this batch ships
zero mobile UI (M-15 is SKIP+FLAG). Web `before.png` / `after.png` would
require a running dev stack against `origin/main`; the orchestrator can
add them when reviewing.
