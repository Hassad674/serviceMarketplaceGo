# BATCH-OPPORTUNITES — W-12 (feed) + W-13 (détail + candidature) + mobile

> Worktree: `/tmp/mp-opportunites` · Branch: `feat/design-opportunites` · Base: `origin/main` (cdf6beb4)

## Goal
Port the Opportunités flow (freelance side, public route) to Soleil v2:
- **W-12** Opportunités feed: public list at `/(public)/opportunities/page.tsx`
- **W-13** Détail opportunité + candidature: `/(public)/opportunities/[id]/page.tsx` + apply modal
- Mobile parity: opportunity card + detail screen (read existing then port)

This is the freelance discovery surface (most common role). Symmetric with the upcoming "Mes annonces" batch (entreprise side) — that's a SIBLING agent owning `job-list.tsx` etc. — DO NOT touch their files.

## Source design
- JSX: `design/assets/sources/phase1/soleil.jsx` and `soleil-lotC.jsx` (search for `Opportunit` blocks)
- PDF reference: `design/assets/pdf/atelier-web-design-stash.pdf` if present
- Current production:
  - `web/src/app/[locale]/(public)/opportunities/page.tsx`
  - `web/src/app/[locale]/(public)/opportunities/[id]/page.tsx`
  - `web/src/features/job/components/{opportunity-card,opportunity-detail,opportunity-list,apply-modal}.tsx`

## TOUCHABLE files (exhaustive — STAY IN THIS LANE)

### Web
- `web/src/app/[locale]/(public)/opportunities/page.tsx`
- `web/src/app/[locale]/(public)/opportunities/[id]/page.tsx`
- `web/src/features/job/components/opportunity-card.tsx`
- `web/src/features/job/components/opportunity-detail.tsx`
- `web/src/features/job/components/opportunity-list.tsx`
- `web/src/features/job/components/apply-modal.tsx` (apply CTA from W-13)
- `web/src/features/job/components/budget-section.tsx` (read-only — only edit IF the design uses it inside opportunity-detail; do NOT edit if it's only used by entreprise-side `create-job-form` / `edit-job-form` which are SIBLING-owned)
- `web/src/messages/fr.json` and `en.json` — NEW keys ONLY, prefix `opportunities_w12_` and `opportunities_w13_`. DO NOT touch existing keys.

### Mobile
- `mobile/lib/features/job/presentation/screens/opportunity_detail_screen.dart`
- `mobile/lib/features/job/presentation/widgets/opportunity_card.dart`
- (read other mobile job widgets but only edit those two unless a public listing screen exists — list it then edit)
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` — NEW keys ONLY, prefix `opportunities_m`

## OFF-LIMITS — STRICT
- Anything under `backend/`
- `web/src/features/job/components/{job-list,job-details-section,application-list,candidate-card,candidate-detail-panel,candidates-list,create-job-form,edit-job-form,applicant-type-selector,credits-info-modal}.tsx` (these belong to SIBLING agent A3 Mes annonces)
- `web/src/features/proposal/**` (proposal feature — locked)
- `web/src/features/messaging/**`, `web/src/features/notification/**` (other Wave A agents)
- `web/src/shared/components/layouts/**` (anyone touching this conflicts with everyone)
- `web/src/shared/components/chat-widget/**` (Wave B)
- All `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`, `shared/lib/api-client.ts`
- `web/src/features/job/api/**`, `web/src/features/job/hooks/**`
- `mobile/lib/features/job/data/**`, `mobile/lib/features/job/domain/**`
- `mobile/lib/features/messaging/**`, `mobile/lib/features/notification/**`, `mobile/lib/features/proposal/**`
- All `package.json`, `pubspec.yaml`, lockfiles, generated l10n `.dart`
- All existing tests

## Acceptance criteria

### W-12 Opportunités feed (public listing)
- Hero header: Fraunces title with italic corail accent ("Trouve ta prochaine *mission*."), eyebrow font-mono uppercase corail "ATELIER · OPPORTUNITÉS", subtitle 15px tabac.
- Filter row (if backend exposes filters: budget range, expertise, etc.): rounded-full pills, ivoire-soft default / corail-soft active.
- Card grid (1col mobile, 2col tablet, 3col desktop optional but stay close to current behavior — read first):
  - Card: ivoire bg, rounded-2xl, border, shadow-card, padding 20-24px
  - Top: small expertise tag pill (corail-soft) + relative time (Geist Mono mini)
  - Title: Fraunces 18-20px (or as design dictates)
  - Description excerpt: 14px tabac, line-clamp-2
  - Footer: Portrait of poster (24×24) + name + "•" + budget pill (Geist Mono, sapin-soft)
  - Hover: -translate-y-0.5 + border-strong
- Empty state: Soleil card with corail-soft icon, Fraunces title, body, optional CTA.
- Pagination / load-more: keep existing behavior, restyle to corail pill button.

### W-13 Détail opportunité + candidature
- Page: 2-col on desktop (≥ 1024px), 1-col below. Left = main content (Fraunces title, expertise tags, description, budget breakdown, deliverables list). Right = sidebar with "Candidater" CTA card (sticky, ivoire bg, corail pill button) + poster mini-card (Portrait + name + role + view-profile link).
- Apply modal: Soleil-styled — ivoire bg, rounded-2xl, large Fraunces title "Candidater à cette opportunité", form fields (cover letter textarea + budget input + delivery date picker), corail submit pill.
- Keep ALL form behavior identical (zod schema, submit handler, validation rules — read but DO NOT change).

### Mobile parity
- Opportunity card widget: same anatomy as web card (Soleil tokens, rounded 16-20).
- Detail screen: AppBar with Fraunces title, scroll body with same blocks (expertise tags, description, budget, deliverables), sticky bottom corail FilledButton "Candidater" opening a bottom sheet form.
- Bottom sheet form: matches web modal fields. Same submit handler.

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-opportunites

# 1. Scope check (catches drift into A1/A3 lanes)
git diff --name-only origin/main...HEAD | grep -E "^(backend/|.*\.test\.|.*_test\.|features/messaging/|features/notification/|features/proposal/|features/job/components/(job-list|job-details-section|application-list|candidate-|candidates-|create-job-form|edit-job-form|applicant-type-selector|credits-info-modal)|shared/components/chat-widget/|shared/components/layouts/)" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"

# 2. Web
cd web
npm ci
npx tsc --noEmit
npx vitest run src/features/job src/app/\[locale\]/\(public\)/opportunities
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
Same as Notifications batch: zero new hooks, zero test edits, all i18n, no Color(0xFF) magic, ONE squashed commit, no `git config` mutation.

## Push + PR
```bash
git push -u origin feat/design-opportunites
gh pr create --title "[design/web/W-12+W-13+mobile] Port Opportunités feed + détail to Soleil v2" --body "<full report>"
```

## Final report (under 700 words)
Standard structure. Visual diffs at `design/diffs/W-12-W-13/`.
