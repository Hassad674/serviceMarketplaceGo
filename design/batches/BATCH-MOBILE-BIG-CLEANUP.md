# BATCH-MOBILE-BIG-CLEANUP — final mobile Soleil polish

> Worktree: `/tmp/mp-mobile-big-cleanup` · Branch: `fix/mobile-big-cleanup` · Base: `origin/main` (48b0680e)

## Goal — 4 final mobile cleanup tasks, ONE squashed commit

### Task 1 — AppPalette deprecation in widgets
The audit identified 34+ mobile widgets still using `AppPalette.rose*` / `AppPalette.slate*` / `AppPalette.indigo*` etc. directly instead of going through `colorScheme.*` or the `SoleilColors` ThemeExtension. This is the largest remaining Soleil leakage on mobile.

Audit listed: portfolio-related, dispute, search/filter widgets, provider-card, client-profile-editor and similar.

Action: replace direct `AppPalette.X` usages with semantic tokens:
- `AppPalette.rose500` / `rose600` → `colorScheme.primary` (corail). For "deeper" rose, use `Theme.of(context).extension<AppColors>()!.primaryDeep` (corail-deep #c43a26).
- `AppPalette.slate*` → `colorScheme.outline` / `colorScheme.onSurfaceVariant` / `colorScheme.surfaceContainer` per usage context
- `AppPalette.indigo*` / `AppPalette.violet*` / `AppPalette.purple*` / `AppPalette.fuchsia*` → `colorScheme.primary` (or extension equivalents)
- `AppPalette.amber*` → `colorScheme.tertiary` if available, OR `Theme.of(context).extension<AppColors>()!.amberSoft` for soft pills
- `AppPalette.teal*` / `AppPalette.sky*` → `colorScheme.primary` (Soleil has no cool tones — collapse to corail)
- `AppPalette.gray800` (#333333) → `colorScheme.onSurface`
- `AppPalette.pureRed` (#FF0000) → `colorScheme.error` (corail-deep)

Goal: `git grep -rn "AppPalette\." mobile/lib/` returns 0 hits in `features/` and only AppPalette class definition + tests reference in `mobile/lib/core/theme/app_palette.dart` and any test file.

Bonus: add `@Deprecated('Use colorScheme.* or AppColors extension instead — Soleil v2 migration')` annotation on every `AppPalette.X` accessor in `app_palette.dart` so future code gets a lint warning.

### Task 2 — Theme file split (over 600 LOC cap)
`mobile/lib/core/theme/app_theme.dart` is 622 lines, breaching the 600 LOC cap from CLAUDE.md.

Action: split into focused files:
- `app_theme.dart` — main `AppTheme` class with `light` / `dark` factories (entry point)
- `theme_colors.dart` — `AppColors` ThemeExtension class (the data container)
- `theme_text_styles.dart` — `SoleilTextStyles` class (typography)
- (optional) `theme_components.dart` — Material 3 component themes if there are many overrides

Each file ≤ 400 LOC. Re-exports from `app_theme.dart` so consumers don't need to update imports. Run `flutter analyze` to verify all imports still resolve.

### Task 3 — Geist Mono bundling (no more google_fonts runtime fetch)
Currently `mobile/lib/core/theme/app_theme.dart` substitutes Geist Mono via `google_fonts` runtime fetch (the audit flagged H3). On first launch offline, falls back to Roboto. On metered networks, slow.

Action: bundle Geist Mono as asset:
1. Download Geist Mono Regular + Medium TTF/WOFF from https://github.com/vercel/geist-font (look in `releases/` for latest TTF) or use the `fontsource_geist_mono` Dart package if it exists. PREFERRED: bundle locally as asset to avoid yet another network dep.
2. Add to `mobile/assets/fonts/` directory (create if needed)
3. Register in `mobile/pubspec.yaml`:
   ```yaml
   flutter:
     fonts:
       - family: GeistMono
         fonts:
           - asset: assets/fonts/GeistMono-Regular.ttf
             weight: 400
           - asset: assets/fonts/GeistMono-Medium.ttf
             weight: 500
   ```
4. Replace `google_fonts.GoogleFonts.geistMono(...)` calls (or whatever the current substitution is — read the existing TODO comment around app_theme.dart:489,495) with `TextStyle(fontFamily: 'GeistMono', ...)`
5. Verify `flutter run` shows the proper monospaced font (visible in wallet amounts, invoice numbers)

If you can't obtain the TTF files in this session (network restrictions, etc.), SKIP+FLAG Task 3 and document in PR what's needed for completion. Don't ship a broken font.

### Task 4 — 15 mobile runtime test failures from rose-500
The previous PR #145 flagged 15 pre-existing runtime test failures caused by widgets still using rose-500/600/etc. literal colors. These widgets are exactly what Task 1 fixes — so completing Task 1 should also resolve the 15 runtime test failures naturally (because the widgets will now use Soleil tokens that the tests' theme provides).

Verify after Task 1 + 2: `flutter test test/` cold suite — failures should drop significantly. Document final counts in PR body.

## TOUCHABLE files

### Task 1 (widgets)
- ALL `mobile/lib/features/*/presentation/widgets/**.dart` and `screens/**.dart` files that grep `AppPalette\.` returns
- DO NOT touch `data/`, `domain/`, or `presentation/providers/` files — they shouldn't have `AppPalette` anyway, but if a stray reference exists, leave it and flag

### Task 2 (theme split)
- `mobile/lib/core/theme/app_theme.dart` (split source)
- NEW files: `mobile/lib/core/theme/theme_colors.dart`, `mobile/lib/core/theme/theme_text_styles.dart`, etc.
- `mobile/lib/core/theme/app_palette.dart` (add `@Deprecated` annotations only — don't delete the class yet, would break stragglers)

### Task 3 (Geist Mono)
- `mobile/pubspec.yaml` (add fonts asset registration — THIS IS AN EXCEPTION to the "no pubspec" rule, explicitly authorized for this batch)
- `mobile/assets/fonts/GeistMono-Regular.ttf` (NEW asset, if you can download)
- `mobile/assets/fonts/GeistMono-Medium.ttf` (NEW asset)
- `mobile/lib/core/theme/app_theme.dart` (or `theme_text_styles.dart` after split) — replace `google_fonts.geistMono(...)` calls with `TextStyle(fontFamily: 'GeistMono', ...)`

### i18n
- No new strings expected for this batch.

## OFF-LIMITS — STRICT
- ALL web files
- ALL admin files
- Backend
- `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`
- `mobile/lib/core/theme/app_theme.dart` import path consumers — if you split the file, ensure the import surface stays compatible (re-export from app_theme.dart so external imports don't break)
- `mobile/test/**` for new tests — only update existing tests if widgets they test now use new tokens AND the test asserts on literal class strings. Otherwise leave alone.
- `mobile/lib/features/*/presentation/providers/**.dart` (no Riverpod state changes)
- `package.json`, lockfiles

## Acceptance criteria
- `git grep -rn "AppPalette\." mobile/lib/features/` returns 0 hits
- `git grep -rn "AppPalette\." mobile/lib/` only matches `mobile/lib/core/theme/app_palette.dart` (the deprecated class itself)
- `flutter analyze --no-pub lib/` returns 0 errors (warnings on `@Deprecated` AppPalette usage in non-features code OK if any)
- `app_theme.dart` < 600 LOC (cap)
- `flutter test --no-pub test/` runtime failures drop significantly (target: from 15 → 0 or near-0)
- Geist Mono renders properly OR Task 3 SKIP+FLAG with clear rationale

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-mobile-big-cleanup

# 1. Scope check
git diff --name-only origin/main...HEAD | grep -E "^(backend/|web/|admin/|mobile/(?!(lib/(features|core/theme)|assets/fonts|pubspec\.yaml|test/)))" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"

# 2. AppPalette grep verification
grep -rn "AppPalette\." mobile/lib/features/ | head
# Should be empty

# 3. Theme file size
wc -l mobile/lib/core/theme/*.dart
# All should be < 600

# 4. Mobile validation
cd mobile
flutter pub get
flutter analyze --no-pub lib/ 2>&1 | tail -10
flutter test --no-pub test/ 2>&1 | tail -20

# 5. Design guardrails
cd ..
bash design/scripts/check-api-untouched.sh
bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3.

## Quality bar
- ZERO new providers/repositories
- ZERO touch to data/domain/providers
- ZERO new dependencies (use `Theme.of(context).extension<AppColors>()` for everything)
- ZERO `Color(0xFF...)` magic re-introduced
- ONE squashed commit
- DO NOT modify `git config` — use per-command `git -c user.email=...`

## Push + PR
- Message: `fix(design/mobile): finalize Soleil v2 — AppPalette deprecation + theme split + Geist Mono bundle`
- PR title: `[fix/mobile-big-cleanup] AppPalette deprecation + theme split + Geist Mono`

## Final report (under 700 words)
Standard structure. EMPHASIZE the AppPalette grep counts (before/after) and the runtime test failure count delta.
