# BATCH-MOBILE-DASHBOARDS — M-03 + M-04 (mobile parity with W-11 web)

> Worktree: `/tmp/mp-mobile-dashboards` · Branch: `feat/design-mobile-dashboards` · Base: `origin/main` (12af167d)

## Goal
Port the mobile dashboards (M-03 freelance + M-04 entreprise) to Soleil v2 — bring them up to parity with the web W-11 dashboard already merged.

Visual references:
- Web W-11 implementation: `web/src/app/[locale]/(app)/dashboard/page.tsx` (already Soleil — read for reference)
- Soleil mobile primitives: `SoleilTextStyles`, `AppColors` extension, Material 3 + custom Soleil theme
- Existing mobile screen: `mobile/lib/features/dashboard/presentation/screens/dashboard_screen.dart`

## TOUCHABLE files

### Mobile
- `mobile/lib/features/dashboard/presentation/screens/dashboard_screen.dart` (the main entry — likely role-aware)
- `mobile/lib/features/dashboard/presentation/screens/referrer_dashboard_screen.dart` (referrer mode)
- `mobile/lib/features/dashboard/presentation/widgets/dashboard_atoms.dart` and any other widgets
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` — NEW keys ONLY, prefix `mobileDashboard_*`

## OFF-LIMITS — STRICT
- ALL web files (this is mobile only)
- `mobile/lib/features/dashboard/data/**`, `mobile/lib/features/dashboard/domain/**`
- `mobile/lib/features/dashboard/presentation/providers/**` (read existing providers but don't modify them)
- All other mobile features (`messaging`, `proposal`, `invoicing`, `billing`, `job`, `notification`, `freelance_profile`, `account`, `wallet`, `auth`, `team`, `search`, `payment_info` — sibling agents own these or already merged)
- `package.json`, `pubspec.yaml`, lockfiles, generated `app_localizations*.dart` (regenerate via flutter gen-l10n is fine)
- All existing `*_test.dart`
- Anything under `backend/`

## Acceptance criteria

### M-03 Dashboard freelance
Mirror W-11's structure in Flutter idiom:
- AppBar: Fraunces "Bonjour {firstName}" or similar editorial greeting
- Editorial greeting block at top: corail mono uppercase eyebrow + Fraunces italic-corail welcome message
- Stats grid: 3 cards in a row (or 2 cols on narrow): "Missions actives", "Messages non lus", "Revenu mensuel"
  - Each card: ivoire `colorScheme.surface`, rounded 20-24, border `colorScheme.outline`, corail-soft icon disc, Geist Mono numbers
- Subtle section dividers using Fraunces titles with italic corail accent (e.g. "Mes *missions du moment*", "Mes *opportunités*")
- Optional sections (only if existing data layer exposes them — SKIP+FLAG otherwise): mini list of recent missions, opportunités feed teaser, Atelier Premium teaser

### M-04 Dashboard entreprise
Same anatomy as M-03 but with entreprise stats:
- "Projets actifs", "Messages non lus", "Budget total"
- Editorial greeting role-aware
- Use `dashboard_screen.dart` role-switching logic (it's already there; just restyle)

### Referrer dashboard variant
- 4 stats: "Filleuls", "Missions actives", "Missions complétées", "Commissions"
- Same visual identity

### Cross-cutting
- All `Color(0xFF...)` magic forbidden — use `colorScheme.*` or `AppColors` extension
- All inline `TextStyle(fontSize:..., fontWeight:...)` magic forbidden — use `SoleilTextStyles.*`
- All `const` constructors where possible (60fps perf budget)
- All user-visible strings via `AppLocalizations.of(context)` — no hardcoded FR

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-mobile-dashboards
git diff --name-only origin/main...HEAD | grep -E "^(backend/|web/|admin/|.*\.test\.|.*_test\.|mobile/lib/features/(messaging|proposal|invoicing|billing|job|notification|freelance_profile|account|wallet|auth|team|search|payment_info)/|mobile/lib/features/dashboard/(data|domain|presentation/providers)/|package\.json|pubspec)" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"
cd mobile && flutter pub get && flutter analyze --no-pub lib/features/dashboard && flutter test --no-pub test/features/dashboard/ 2>&1 || echo "(may not exist)"
cd .. && bash design/scripts/check-api-untouched.sh && bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3.

## Quality bar
- ZERO new providers/repositories
- ZERO touch to existing tests
- All i18n via AppLocalizations
- No `Color(0xFF...)` magic, no inline TextStyle magic
- ONE squashed commit, no git config drift

## Push + PR
- Message: `feat(design/mobile-dashboards): port M-03 freelance + M-04 entreprise to Soleil v2`
- PR title: `[design/mobile/M-03+M-04] Port mobile dashboards to Soleil v2`

## Final report (under 500 words)
Standard structure: summary / files / out-of-scope / validation output VERBATIM / brief feedback.
