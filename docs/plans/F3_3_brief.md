# F.3.3 — Quality web/mobile + remaining MEDIUM polish

**Phase:** F.3.3 — post-publish polish
**Source audit:** PR #92 — quality regressions + remaining MEDIUM
**Effort:** 2-3j est.
**Tool:** 1 fresh agent dispatched (after user validates)
**Branch:** `feat/f3-3-quality-web-mobile`

## Problem

Final audit ranked Quality top-15-25% on web/mobile due to:
- **Mobile `dynamic` regression**: 196 → 746 references (3.8× growth) — typed-as-`dynamic` re-introduced post-Freezed regen
- **Mobile `Color(0x...)` regression**: 491 → 573 — theme-token discipline degrading (devs hardcoding hex instead of using token)
- **19 backend files > 600 lines** — CLAUDE.md violations re-introduced

Plus minor:
- `CONTRIBUTING.md:165` cites old path `contract-isolation.spec.ts` (real: `refactor-isolation.spec.ts`)
- 1 pre-existing flaky test `TestProfileCache_Singleflight_CoalescesConcurrentMisses` in `internal/adapter/redis`

## Plan (4 commits)

### Commit 1 — Mobile `dynamic` reduction
- Inventory all 746 `dynamic` references via `grep -rn "dynamic" mobile/lib/ --include="*.dart"`
- Categorize:
  - Freezed-generated files (allowed) → confirm via `*.freezed.dart` filter, exclude
  - Real source `.dart` files → add proper types
- Target: ≤ 200 references in source files (close to original 196)
- Tests: `flutter analyze --fatal-infos` clean post-fix

### Commit 2 — Mobile theme tokens
- Inventory `Color(0x...)` calls in `mobile/lib/`
- Replace hex literals with `AppColors.<token>` from existing `lib/core/theme/app_theme.dart`
- Target: 0 hex `Color(0x...)` outside the theme file itself
- Tests: visual check via `flutter test` widget tests still pass

### Commit 3 — Backend file split (19 files > 600 lines)
- List via `find backend/internal -name "*.go" -exec wc -l {} \; | awk '$1>600' | sort -rn`
- Split each by sub-domain (handler files: by sub-resource; service files: by use case; repo files: by query group)
- Tests pass after each split (zero behaviour change)
- Target: 0 files > 600 lines

### Commit 4 — Minor polish
- `CONTRIBUTING.md:165` typo fix
- `TestProfileCache_Singleflight_CoalescesConcurrentMisses` flake — add `t.Skip` with reason if flaky under -race, OR fix the race if visible (check the test source)
- Any LOW items from audit that are 5-min fixes

## Hard constraints

- **Validation pipeline**: `flutter analyze && flutter test` per commit; backend pipeline per backend commit
- **Zero behaviour change** on file splits — pure structural refactor
- **No new tests required** beyond re-running existing suites — F.3.3 is polish, not feature

## Out-of-scope

- LiveKit (off-limits)
- 41 LOW findings remaining → F.4 if user wants
- Workflow files (token can't push)

## Branch ownership

`feat/f3-3-quality-web-mobile` only.

## Final report

Lead with PR URL.
1. Mobile dynamic 746 → N (target ≤ 200)
2. Mobile Color(0x...) 573 → N (target 0 outside theme)
3. Backend files >600 19 → 0
4. Minor polish items closed
5. Validation pipeline output
6. "Branch ownership confirmed"
