# Agent brief template — Mobile batch (Flutter iOS native)

> Copy this file to `design/batches/BATCH-XXX-mobile-<topic>.md`, fill the variables marked `{{...}}`, then dispatch.

---

## Batch identity

- **ID**: `BATCH-XXX`
- **Surface**: Mobile (Flutter 3.16+ — iOS native, Material 3 + Cupertino flavor)
- **Topic**: `{{topic}}`
- **Worktree**: `/tmp/mp-design-mobile-{{batch-id}}`
- **Branch**: `feat/design-mobile-{{batch-id}}-{{topic}}`
- **Base**: `origin/main`

## Screens in this batch

- `{{inventory-id-1}}` — `{{screen-name-1}}`
- `{{inventory-id-2}}` — `{{screen-name-2}}`
- ...

## Source assets

For each screen:
- Soleil JSX implementation: `design/assets/sources/phase1/{{file}}.jsx` lines `{{range}}`
- Mobile layout: `design/assets/pdf/app-native-ios.pdf` page `{{N}}`

The JSX is **inspiration only** — re-implement using Flutter idioms (Material 3 widgets, Cupertino accents, Theme.of(context) tokens).

## Repo mapping

For each screen, the inventory entry lists:
- **Route existante**: `{{go_router_path}}` in `mobile/lib/core/router/app_router.dart`
- **Fichier principal**: `{{path/to/screen.dart}}`
- **Widgets touchables**: `{{whitelist}}`
- **Repository (OFF-LIMITS)**: `{{list — these MUST NOT be edited}}`
- **Features design absentes du repo**: `{{SKIP and FLAG}}`

---

# RULES

[Include verbatim from `design/agent-templates/shared-rules.md`]

# MOBILE-SPECIFIC RULES

## Theme & tokens

The Soleil v2 theme is wired in `mobile/lib/core/theme/soleil_theme.dart` (created in Phase 0). Use it via:

```dart
final theme = Theme.of(context);
final soleil = theme.extension<SoleilColors>()!;

// Colors
theme.colorScheme.primary       // corail #e85d4a
theme.colorScheme.surface       // ivoire #fffbf5
theme.colorScheme.onSurface     // encre #2a1f15
theme.colorScheme.onSurfaceVariant  // tabac #7a6850
theme.colorScheme.outline       // border #f0e6d8
soleil.successSoft              // sapin pâle #e8f2eb
soleil.corailSoft               // corail pâle #fde9e3

// Typography
SoleilTextStyles.displayL       // Fraunces 38-44
SoleilTextStyles.displayM       // Fraunces 30
SoleilTextStyles.body           // Inter Tight 14-15
SoleilTextStyles.mono           // Geist Mono 11-18
```

## Hard rules

- **Never** `Color(0xFF...)` hardcoded in widgets — always go through `colorScheme` or `SoleilColors` extension.
- **Never** inline `TextStyle(fontSize: 17, fontWeight: FontWeight.w600)` with magic numbers — use `SoleilTextStyles.*` constants. Local micro-tweaks via `.copyWith(...)` are OK.
- **Never** put strings in widgets — every label via `AppLocalizations.of(context)` (i18n in `mobile/lib/l10n/app_fr.arb`).
- **Always** prefer `const` constructors. Mobile perf budget = 60fps minimum.
- **Photos** = `Portrait(id: n, size: 48)` widget from `mobile/lib/core/widgets/portrait.dart`. Never `CircleAvatar(child: Text("EM"))`.

## OFF-LIMITS for mobile (in addition to shared list)

- `mobile/lib/core/api/**.dart`
- `mobile/lib/core/network/**.dart` (Dio, interceptors)
- `mobile/lib/features/*/data/**.dart` (data sources, repositories, models with json mapping)
- `mobile/lib/features/*/domain/**.dart` (entities, use cases)
- `mobile/pubspec.yaml`, `pubspec.lock`
- All `*_test.dart` files

## TOUCHABLE for mobile (typical batch)

- `mobile/lib/features/<feature>/presentation/screens/<screen>.dart`
- `mobile/lib/features/<feature>/presentation/widgets/**.dart`
- `mobile/lib/core/theme/soleil_theme.dart` (Phase 0 only)
- `mobile/lib/core/widgets/portrait.dart` (Phase 0 only)
- `mobile/lib/l10n/app_fr.arb` (add i18n keys)

## Validation pipeline (mobile batch)

```bash
cd /tmp/mp-design-mobile-{{batch-id}}

# Analyze + tests on touched dirs
cd mobile
flutter analyze lib/features/{{feature}}/presentation lib/core/theme lib/core/widgets
flutter test test/features/{{feature}}/

cd ..

# Verify backend untouched
cd backend && go build ./... && go vet ./... && cd ..

# Design guardrails
design/scripts/validate-no-regression.sh
```

ALL of these must pass before you commit.

## Visual diff

For each screen:
1. Before changes (on `origin/main`), `flutter screenshot --device-id <id>` and save as `design/diffs/{{screen-id}}/before-mobile.png`.
2. After changes (your branch), same → `after-mobile.png`.
3. `notes.md` for intentional differences.

If a device is not connected, document it in the report and note that `before/after` capture must be done by orchestrator.

## Push + PR

```bash
git push -u origin feat/design-mobile-{{batch-id}}-{{topic}}
gh pr create \
  --title "[design/mobile/{{batch-id}}] {{topic}}" \
  --body "$(cat <<'EOF'
## Batch summary
{{1-paragraph}}

## Screens shipped
- {{inventory-id-1}} {{screen-name-1}}
- ...

## Out-of-scope flagged
- ...

## Validation pipeline output
\`\`\`
{{paste full output}}
\`\`\`

## Files changed (whitelist check)
- Allowed: ...
- Off-limits: NONE

## Visual diffs
- design/diffs/{{screen-id}}/before-mobile.png ↔ after-mobile.png
EOF
)"
```

## Final report (in this batch file)

```markdown
## Outcome
- PR: #{{pr-number}}
- Status: in-review / merged / changes-requested
- Validation pipeline: PASS / FAIL
- Visual diffs reviewed: yes / no
- Screens marked done in tracking.md: yes / no
```
