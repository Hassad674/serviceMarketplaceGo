# BATCH-TEAM-SEARCH — W-22 Équipe + M-12 Recherche freelances (web + mobile)

> Worktree: `/tmp/mp-team-search` · Branch: `feat/design-team-search` · Base: `origin/main` (12af167d)

## Goal
Port two cross-rôle surfaces to Soleil v2:
- **W-22 Équipe & permissions** — at `/team`, agency + entreprise team management (members + invitations + role permissions)
- **M-12 Recherche freelances** — public freelance directory at `/(public)/freelances` (web) + equivalent mobile screen
- Mobile parity for both

## TOUCHABLE files

### Web — Team
- `web/src/app/[locale]/(app)/team/page.tsx`
- `web/src/features/team/components/team-header.tsx`
- `web/src/features/team/components/team-invitations-list.tsx`
- `web/src/features/team/components/invite-member-modal.tsx`
- `web/src/features/team/components/edit-member-modal.tsx`
- `web/src/features/team/components/remove-member-dialog.tsx`
- `web/src/features/team/components/leave-org-dialog.tsx`
- `web/src/features/team/components/role-permissions-editor.tsx`
- `web/src/features/team/components/role-permissions-editor-parts.tsx`
- `web/src/features/team/components/pending-transfer-banner.tsx`
- `web/src/features/team/components/accept-invitation-page.tsx`
- (Read first all team components and any sub-widgets — port what's needed for Soleil)

### Web — Recherche freelances
- `web/src/app/[locale]/(public)/freelances/page.tsx` (and any sub-route page if exists)
- Any existing `freelance-list` / `freelance-card` / search filter components used by the route (likely under `web/src/features/freelance-profile/components/` or `web/src/shared/components/`). Read the page imports to identify them.
- DO NOT modify already-merged Soleil files — only the directory/search-page-specific ones.

### Web — i18n
- `web/messages/fr.json` and `en.json` — NEW keys ONLY, prefixes `team_w22_*` and `freelancesSearch_m12_*`

### Mobile — Team
- `mobile/lib/features/team/presentation/screens/**.dart` (read first)
- `mobile/lib/features/team/presentation/widgets/**.dart`

### Mobile — Search/freelances
- `mobile/lib/features/search/presentation/screens/**.dart`
- `mobile/lib/features/search/presentation/widgets/**.dart`

### Mobile — i18n
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` — NEW keys ONLY, prefixes `team_w22_*` and `freelancesSearch_m12_*`

## OFF-LIMITS — STRICT
- All `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`
- `web/src/features/team/api/**` and `hooks/**`
- `web/src/features/search/**` (api/hooks if any)
- `mobile/lib/features/team/data/**`, `mobile/lib/features/team/domain/**`
- `mobile/lib/features/team/presentation/providers/**` (read but don't modify Riverpod providers)
- `mobile/lib/features/search/data/**`, `mobile/lib/features/search/domain/**`
- `mobile/lib/features/search/presentation/providers/**`
- All other features (`messaging`, `proposal`, `invoicing`, `billing`, `job`, `notification`, `freelance-profile`, `account`, `wallet`, `auth`, `dashboard` — already Soleil)
- `mobile/lib/features/payment_info/**` (sibling A1)
- `mobile/lib/features/dashboard/**` (sibling A3)
- `mobile/lib/features/invoicing/**` (sibling A4)
- `package.json`, `pubspec.yaml`, lockfiles, generated l10n
- All existing tests
- Anything under `backend/`

## Acceptance criteria

### W-22 Team page
- Editorial header: corail eyebrow "ATELIER · ÉQUIPE", Fraunces italic-corail title (e.g. "Tes *coéquipiers et permissions*."), tabac subtitle
- Members list: Soleil card per member (Portrait + name + role pill + actions ghost icon)
- Invitations list (pending): Soleil card with corail-soft border for pending state + accept/decline ghost buttons
- Modals (invite/edit/remove/leave): Soleil ivoire bg, rounded-2xl, Fraunces titles, corail/destructive pills
- Role permissions editor: Soleil grid + checkboxes/toggles in Soleil tokens

### M-12 Recherche freelances (web + mobile)
- Hero header: Fraunces title with italic corail accent ("Trouve les *meilleurs talents*."), eyebrow, subtitle
- Search bar: rounded-full, ivoire bg, corail focus, search icon left, filter button right
- Filter pills (expertise, location, etc.) if exposed: rounded-full Soleil
- Freelance cards: ivoire bg, rounded-2xl, Portrait + name + title + skills pills + budget pill + view profile ghost link
- Empty state Soleil
- Mobile: same anatomy in Flutter idiom

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-team-search
git diff --name-only origin/main...HEAD | grep -E "^(backend/|.*\.test\.|.*_test\.|features/(messaging|proposal|invoicing|billing|job|notification|freelance-profile|account|wallet|auth|dashboard)/|features/team/(api|hooks)/|features/search/(api|hooks)/|app/\[locale\]/\(app\)/(messages|projects|invoices|jobs|notifications|profile|account|billing|payment-info|wallet)/|app/\[locale\]/\(public\)/(opportunities|agencies)/|mobile/lib/features/(messaging|proposal|invoicing|billing|job|notification|freelance_profile|account|wallet|auth|dashboard|payment_info|team/(data|domain|presentation/providers)|search/(data|domain|presentation/providers)))" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"
cd web && npm ci && npx tsc --noEmit && npx vitest run src/features/team src/app/\[locale\]/\(app\)/team src/app/\[locale\]/\(public\)/freelances && npm run build
cd ../mobile && flutter pub get && flutter analyze --no-pub lib/features/team lib/features/search && flutter test --no-pub test/features/team/ test/features/search/ 2>&1 || echo "(may not exist)"
cd .. && bash design/scripts/check-api-untouched.sh && bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3.

## Quality bar
- ZERO new hooks/mutations/repositories
- ZERO touch to existing tests
- All i18n via translations
- ONE squashed commit, no git config drift

## Push + PR
- Message: `feat(design/team-search): port W-22 team + M-12 search to Soleil v2 (web + mobile)`
- PR title: `[design/web/W-22+mobile/M-12] Port Team + Recherche freelances to Soleil v2`

## Final report (under 700 words)
Standard structure.
