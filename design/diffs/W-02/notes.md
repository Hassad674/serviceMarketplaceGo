# W-02 — Inscription · choix de rôle (Soleil v2)

## Source

- JSX: `design/assets/sources/phase1/soleil-lotE.jsx` `SoleilSignupRole`
  (lines 133-238)
- PDF: `design/assets/pdf/web-desktop.pdf` page 6
- Visual proof: `design/assets/sources/phase1/soleil-lotE.html` (canvas)

## Intentional differences vs source maquette

- **3 cards = repo's existing 3 roles**, not the source's
  Freelance / Entreprise / Apporteur split. The repo bundles the
  apporteur d'affaires inside the provider role (via the
  `referrer_enabled` toggle on the provider profile). Splitting them
  into a separate registration card would require a backend change to
  the register schema — out of scope for a UI-only batch. Cards
  preserve the existing `roleAgency` / `roleFreelance` /
  `roleEnterprise` strings so the e2e test in
  `web/e2e/auth.spec.ts` keeps matching.

- **Highlighted card = Freelance** (middle). Source highlights the
  Freelance card by default; we keep that visual treatment because
  the role labelled `roleFreelance` covers both freelance AND
  apporteur d'affaires in this repo, which is the most common signup
  path. The selection is purely decorative — the cards are
  navigation links, not radio inputs.

- **Visual on the highlighted card = `<Portrait id={1} />`** (Soleil
  primitive), matching the source's portrait usage on the selected
  card. The two neutral cards use a 76×76 gradient square + lucide
  icon (`Building2` for Agency, `Sparkles` for Enterprise to keep
  the palette warm).

- **No "Continuer en tant que freelance" CTA** at the bottom of the
  page (present in source). The role cards themselves are the
  navigation in this repo — adding a second CTA would conflict with
  the navigation contract pinned by the e2e test.

- **No "NOUVEAU" badge** on a third card (no third card; see first
  item).

## Files changed

- `web/src/app/[locale]/(auth)/register/page.tsx` — full Soleil v2
  rewrite (top bar, editorial header, 3-card grid).
- `web/messages/fr.json` — added `auth.W02_*` keys (eyebrow, title
  prefix/accent, subtitle, per-card eyebrow / desc / 3 bullets).
- `web/messages/en.json` — same keys, English copy.

## Files NOT touched (whitelist preserved)

- No backend file.
- No `web/src/features/auth/api/**`, `hooks/use-*.ts`, or
  `schemas/**`.
- No test file.
- No `middleware.ts`, `next.config.ts`, `package.json`,
  `package-lock.json`.

## Tests

- 34 test files / 313 tests pass via
  `npx vitest run --changed origin/main`.
- e2e contract in `web/e2e/auth.spec.ts` `Role selection (/register)`
  still satisfied — strings (`Créer un compte`, `Agence`,
  `Freelance / Apporteur d'affaire`, `Entreprise`, `Se connecter`)
  unchanged, sub-route navigation (`/register/{agency,provider,
  enterprise}`) preserved.

## Screenshots

- `before.png` — captured against `origin/main` (manual capture
  required by reviewer; instructions: spin up dev server pre-merge
  and screenshot `/register`).
- `after.png` — captured on `feat/design-w02-register` (same
  instructions).

Both captures are TODO for the human reviewer; the agent runs in a
sandbox with no display server.
