# BATCH-M-01 — Connexion mobile — DRAFT brief

> Status: **draft**, not yet dispatched. Ready to fire as soon as W-16
> calibration is audited and merged.
>
> This is the second calibration batch (the first mobile screen
> ported). If the agent succeeds here, the brief format is also validated
> for mobile and Phase 2 mobile dispatches can proceed in parallel
> with web.

---

## Batch identity

- **ID**: `BATCH-M-01`
- **Surface**: Mobile (Flutter, Android-first per `design/rules.md` §12)
- **Topic**: Connexion (login screen)
- **Worktree**: `/tmp/mp-m01` (orchestrator creates before dispatch)
- **Branch**: `feat/design-m01-mobile-connexion` (from `origin/main`)
- **Base**: `origin/main` post-W-16 merge — so the agent has any updates W-16 may have brought to shared mobile primitives.

## Screens in this batch

- **M-01** Connexion mobile — `mobile/lib/features/auth/presentation/screens/login_screen.dart`

## Source design

- **JSX**: `design/assets/sources/phase1/soleil-app-lot5.jsx` `AppLogin` (around line ~50)
- **Visual proof**: `design/assets/pdf/app-native-ios.pdf` page 3 (left frame: "Bon retour parmi nous")

The Flutter target is **Pixel 5 emulator (Android, 390x844)** — but the layout must stay cross-platform. Don't introduce `Platform.isAndroid` checks for visual reasons; use Material 3 Cupertino-flavored widgets that adapt automatically.

## Repo mapping

### Route existing in the repo
- GoRouter: `/login`
- Screen file: `mobile/lib/features/auth/presentation/screens/login_screen.dart`

### Existing widgets (touchable for the UI restyle)
- The login_screen file itself.
- Any sub-widgets it composes, IF they live under `mobile/lib/features/auth/presentation/` — typically form fields, OAuth buttons.
- DO NOT touch anything under `mobile/lib/features/auth/data/` or `mobile/lib/features/auth/domain/`.

### OFF-LIMITS for M-01
- All `mobile/lib/features/auth/data/**.dart` (repository, data sources, models)
- All `mobile/lib/features/auth/domain/**.dart` (entities, use cases)
- All `mobile/lib/core/api/**.dart`, `mobile/lib/core/network/**.dart`
- `mobile/pubspec.yaml`, `pubspec.lock`
- All existing `*_test.dart` files
- ALL `web/**`, `admin/**`, `backend/**`

### TOUCHABLE
- `mobile/lib/features/auth/presentation/screens/login_screen.dart`
- `mobile/lib/features/auth/presentation/widgets/**.dart` (if sub-widgets exist)
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` (i18n keys)

## Soleil v2 patterns specific to M-01

From the JSX (re-implement in Flutter, not React):

1. **Stack layout** — single column 390-wide, no side hero (hero is web-only).
2. **AtelierMark logo top** — corail rounded square (~52x52) with "A" in white.
3. **Form column** centered ~40px from logo:
   - Headline Fraunces 28-32px: "Bon retour parmi nous." (1 line) — use `SoleilTextStyles.headlineLarge` from the theme
   - Italic Fraunces subtitle 15px: "Connectez-vous pour retrouver vos missions et conversations." — use `SoleilTextStyles.body.copyWith(fontStyle: FontStyle.italic, color: tabac)`
   - Email TextField: filled bg ivoire-soft, border 1.5px, label "E-mail", filled value placeholder "camille.dubois@atelier.fr"
   - Password TextField: same style, label "Mot de passe", trailing IconButton with eye/eye-off
   - Trailing-aligned link: "Mot de passe oublié ?" italic Fraunces small
   - Primary button: full width, rounded-full pill, bg corail, text white, label "Se connecter", height ~52px
   - Divider with italic "ou continuer avec" text label
   - 3 OAuth buttons inline (Google / Apple / LinkedIn) — outlined, rounded-full, equal flex distribution
4. **Footer link**: "Pas encore de compte ? Créer un compte"

### Use the Phase 0 mobile primitives

- `Theme.of(context).colorScheme.primary` → corail
- `Theme.of(context).extension<AppColors>()!.corailSoft` for soft backgrounds
- `SoleilTextStyles.displayL/headlineLarge/body/etc.` from `mobile/lib/core/theme/app_theme.dart`
- `Portrait` widget if needed (likely not on this screen, but it exists)

## Features design absentes du repo — SKIP and FLAG

Likely candidates:

- **OAuth Google / Apple / LinkedIn buttons** — verify the auth feature has OAuth wiring (likely NOT). If absent, SKIP all 3 buttons and the divider, FLAG. The form is fully usable with email/password only.

You will discover what's wired by reading the existing `login_screen.dart`. DO NOT add OAuth provider plumbing — that's a separate feature dispatch.

## Mandatory before EVERY commit — validation pipeline

```bash
cd /tmp/mp-m01

# 1. Backend untouched
git diff --name-only origin/main...HEAD | grep -E "^(backend|web|admin)/" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"

# 2. Flutter analyze + test
cd mobile
flutter analyze --no-pub lib/features/auth/presentation lib/core/theme
flutter test --no-pub test/features/auth/

cd ..

# 3. Design guardrails
bash design/scripts/check-api-untouched.sh
bash design/scripts/check-imports-stable.sh
```

`check-imports-stable.sh` for mobile checks Dart import patterns (api/, data/, domain/) — any positive delta = OFF-LIMITS violation.

## Quality standards (mobile-specific, in addition to shared rules)

- **No `Color(0xFF...)` hardcoded** — go through `colorScheme.*` or `SoleilColors` extension
- **No inline `TextStyle(fontSize: ..., fontWeight: ...)`** with magic numbers — use `SoleilTextStyles.*`
- **`const` constructors** wherever possible (perf budget 60fps)
- **i18n strings** via `AppLocalizations.of(context)` — never hardcoded in widgets
- **No `Platform.isAndroid` checks** for visual reasons — code stays cross-platform per `rules.md` §12

## Visual diff (mobile-specific)

For Android emulator:

```bash
flutter screenshot --device-id <id> --out=design/diffs/M-01/before-android.png  # before changes (origin/main)
# ... agent does the work ...
flutter screenshot --device-id <id> --out=design/diffs/M-01/after-android.png  # after changes
```

If no Android device/emulator available at agent dispatch time: agent flags this and orchestrator captures.

## Push + PR

```bash
git push -u origin feat/design-m01-mobile-connexion
gh pr create --title "[design/mobile/M-01] Port Connexion to Soleil v2" --body "..."
```

PR body uses the same template as web (summary / screens / out-of-scope / validation output / files / visual diff / test plan).

## Final report (under 700 words)

Same structure as web batch report. Plus the candid "What I'd improve in the brief" section — calibration round, the agent's feedback shapes the Phase 2 mobile brief.

---

## When to dispatch this batch

**Trigger**: W-16 (BATCH-CALIBRATION-3 web) is audited and either merged or the corrections are clear. Then:

1. Orchestrator creates worktree: `git worktree add /tmp/mp-m01 -b feat/design-m01-mobile-connexion origin/main`
2. Orchestrator opens this brief draft, fills any remaining placeholders, dispatches via Agent tool
3. After agent completion: audit with `design/batches/AUDIT-CHECKLIST.md`

If W-16 fails badly: fix the brief format here (this draft) before dispatching M-01. Don't propagate a known-bad brief to mobile.
