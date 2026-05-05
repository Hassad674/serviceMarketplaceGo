# BATCH-MES-ANNONCES — W-06 (web) + M-07 (mobile)

> Worktree: `/tmp/mp-mes-annonces` · Branch: `feat/design-mes-annonces` · Base: `origin/main` (cdf6beb4)

## Goal
Port Mes annonces (entreprise listing surface) to Soleil v2:
- **W-06** Mes annonces (web): `/(app)/jobs/page.tsx` — listing of an entreprise's posted annonces (status, applicants count, edit/view actions)
- **M-07** Mes annonces (mobile): mirror screen on mobile

This is the entreprise-side mirror of the freelance Opportunités surface (sibling agent owns that). DO NOT touch the public `/(public)/opportunities/` route or its files.

NB: W-07 (annonce detail · description) and W-08 (annonce detail · candidatures) — at `/(app)/jobs/[id]/page.tsx` — are NOT in scope for this batch. Listing only.

## Source design
- JSX: `design/assets/sources/phase1/soleil-lotC.jsx` and `soleil.jsx` (search "Mes annonces" or "JobsList")
- Current production:
  - `web/src/app/[locale]/(app)/jobs/page.tsx`
  - `web/src/features/job/components/job-list.tsx`
  - `mobile/lib/features/job/presentation/` (read first, locate the entreprise listing screen if any — port or build the surface)

## TOUCHABLE files (exhaustive)

### Web
- `web/src/app/[locale]/(app)/jobs/page.tsx`
- `web/src/features/job/components/job-list.tsx`
- `web/src/features/job/components/job-details-section.tsx` (only if used inside `job-list.tsx`. If only used in detail page `[id]/`, leave alone.)
- `web/src/messages/fr.json` and `en.json` — NEW keys ONLY, prefix `myJobs_w06_`. DO NOT touch existing keys.

### Mobile
- The entreprise "Mes annonces" screen path under `mobile/lib/features/job/presentation/screens/` — read existing structure FIRST, identify the screen (if it exists) or surface a port plan. If no existing screen: SKIP+FLAG; do not invent.
- `mobile/lib/features/job/presentation/widgets/` for the list item widget specifically used in entreprise listing.
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` — NEW keys ONLY, prefix `myJobs_m07_`

## OFF-LIMITS — STRICT
- Anything under `backend/`
- `web/src/features/job/components/{opportunity-card,opportunity-detail,opportunity-list,apply-modal}.tsx` (SIBLING agent A2 Opportunités)
- `web/src/features/job/components/{create-job-form,edit-job-form,applicant-type-selector,application-list,candidate-card,candidate-detail-panel,candidates-list,credits-info-modal,budget-section}.tsx` — these belong to W-09 (Création) and W-08 (Candidatures), NOT this batch
- `web/src/app/[locale]/(app)/jobs/[id]/**` (detail page = W-07/W-08, NOT this batch)
- `web/src/app/[locale]/(app)/jobs/create/**` (creation = W-09, NOT this batch)
- `web/src/app/[locale]/(public)/opportunities/**` (SIBLING agent A2)
- `web/src/features/proposal/**`, `web/src/features/messaging/**`, `web/src/features/notification/**`
- `web/src/shared/components/layouts/**`, `web/src/shared/components/chat-widget/**`
- All `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`, `shared/lib/api-client.ts`
- `web/src/features/job/api/**`, `web/src/features/job/hooks/**`
- `mobile/lib/features/job/data/**`, `mobile/lib/features/job/domain/**`
- `mobile/lib/features/messaging/**`, `mobile/lib/features/notification/**`, `mobile/lib/features/proposal/**`
- All `package.json`, `pubspec.yaml`, lockfiles, generated l10n `.dart`
- All existing tests

## Acceptance criteria

### W-06 Mes annonces (web listing)
- Page header: eyebrow "ATELIER · MES ANNONCES" font-mono uppercase corail, Fraunces title with italic corail accent ("Tes *annonces publiées*." or similar), subtitle.
- Top-right action row: corail pill "Publier une annonce" linking to `/jobs/create` (existing route).
- Filter / status pills (Active / Brouillon / Pourvue / Expirée — match existing data shape): rounded-full, ivoire-soft / corail-soft active.
- List of annonce cards:
  - Ivoire bg, rounded-2xl, border, shadow-card, padding 20-24px
  - Top row: status pill (sapin-soft "Active" / amber-soft "Brouillon" / etc.) + relative date
  - Title: Fraunces 18-20px
  - Excerpt: 14px tabac line-clamp-2
  - Footer: applicants count icon + count + "•" + budget pill (Geist Mono, sapin-soft) + "•" + view/edit actions (ghost icon buttons)
  - Hover: -translate-y-0.5 + border-strong
- Empty state: corail-soft icon plate, Fraunces "Tu n'as encore rien publié", body, corail CTA "Publier ta première annonce".

### M-07 Mes annonces mobile
- AppBar: Fraunces "Mes annonces" + filter icon (if filter exists).
- ListView of annonce cards (same anatomy as web).
- FAB or top corail FilledButton "Publier une annonce".
- Empty state: same Soleil card pattern.

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-mes-annonces

# 1. Scope check (catches drift into A1/A2 lanes or detail/create pages)
git diff --name-only origin/main...HEAD | grep -E "^(backend/|.*\.test\.|.*_test\.|features/messaging/|features/notification/|features/proposal/|features/job/components/(opportunity-|apply-modal|create-job-form|edit-job-form|applicant-type-selector|application-list|candidate-|candidates-|credits-info-modal|budget-section)|app/\[locale\]/\(public\)/opportunities/|app/\[locale\]/\(app\)/jobs/(\[id\]|create)/|shared/components/chat-widget/|shared/components/layouts/)" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"

# 2. Web
cd web
npm ci
npx tsc --noEmit
npx vitest run src/features/job src/app/\[locale\]/\(app\)/jobs
npm run build

# 3. Mobile
cd ../mobile
flutter pub get
flutter analyze --no-pub lib/features/job
flutter test --no-pub test/features/job/ 2>&1 || echo "(may not exist)"

# 4. Design guardrails
cd ..
bash design/scripts/check-api-untouched.sh
bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3. Blocker file if stuck.

## Quality bar
Same as Notifications batch.

## Push + PR
```bash
git push -u origin feat/design-mes-annonces
gh pr create --title "[design/web/W-06+mobile/M-07] Port Mes annonces to Soleil v2" --body "<full report>"
```

## Final report (under 700 words)
Standard structure. Visual diffs `design/diffs/W-06-M-07/`.
