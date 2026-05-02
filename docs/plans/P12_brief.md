# P12 — Mobile build_runner + subscription DTOs Freezed + 48 broken tests

**Phase:** F.2 HIGH (chosen as F.2 #1 — quick win, unblocks rest of mobile work)
**Source audit:** QUAL-FINAL (mobile broken tests) + flagged out-of-scope from earlier P5/P63 reports
**Effort:** 1j est. mécanique
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p12-mobile-build-runner`

## Problem

Mobile codebase uses Freezed + json_serializable for immutable data classes (DTOs, entities, responses). The generated `*.freezed.dart` and `*.g.dart` files are missing or stale. Result:
- `flutter analyze lib/features/subscription/` reports compile errors (`Subscription.billingCycle` undefined etc.)
- ~48 mobile tests fail to compile because they reference the missing generated types
- CI mobile job (when wired) would fail
- Any future agent touching mobile features hits the same wall immediately

The fix is **purely mechanical**: run `build_runner build --delete-conflicting-outputs` and commit the generated artefacts.

## Decision: commit generated files (yes)

Two options exist :
- (a) Commit the `*.freezed.dart` + `*.g.dart` artefacts (current convention in this repo per the existing `*.freezed.dart` files)
- (b) Gitignore them and run build_runner in CI

This project uses **option (a)** — confirmed by `git ls-files | grep "\.freezed\.dart$"` returning many committed files. The agent commits generated outputs so CI doesn't need a build_runner step (faster CI, simpler config).

## Discovery (do first)

```bash
cd /tmp/mp-p12-mobile/mobile
# Check Flutter version
flutter --version

# Inventory pre-existing generated files
git ls-files | grep -E "\.(freezed|g)\.dart$" | wc -l

# Find files declaring Freezed/json_serializable that need generation
grep -rn "part '.*\.freezed\.dart';" lib/ | wc -l
grep -rn "part '.*\.g\.dart';" lib/ | wc -l

# Pre-fix analyzer error count
flutter analyze lib 2>&1 | grep -E "error " | wc -l

# Pre-fix test failure count
flutter test 2>&1 | grep -E "Some tests failed|All tests passed" | tail -1
flutter test 2>&1 | grep -cE "FAILED:" || true
```

Note the pre-fix numbers in PR description.

## Fix steps

```bash
cd /tmp/mp-p12-mobile/mobile

# 1. Ensure deps up to date
flutter pub get

# 2. Run build_runner — delete-conflicting-outputs is mandatory because
#    stale generated files may exist from old branches.
dart run build_runner build --delete-conflicting-outputs

# 3. Verify generation succeeded:
flutter analyze lib 2>&1 | tail -20

# 4. Run full test suite:
flutter test
```

If `build_runner build` itself fails (e.g., Freezed annotation typo, missing dep), the agent must:
- Read the error
- Fix the source file (NOT the generated file — fix what produces the broken output)
- Re-run build_runner
- Document the fix in the commit body

## Hard constraints (paranoid mode)

- **Validation pipeline before EVERY commit**:
  ```bash
  cd /tmp/mp-p12-mobile/mobile
  flutter pub get
  flutter analyze lib  # 0 errors on touched files (warnings/info OK if pre-existing)
  flutter test          # the 48 previously-broken tests must now compile + pass
  ```

- **Don't introduce new code** beyond regenerating files. If a `.freezed.dart` or `.g.dart` requires a `freezed`/`json_serializable` annotation that's missing in the source `.dart` file, ADD that annotation minimally and document why.

- **Don't broaden scope** — if you find unrelated test failures (network errors, fake_api_client drift, etc.), flag them in PR description, do NOT silently fix them.

- **Race-flake `TestFreelanceCache_Singleflight_Coalesces`** — flagged out-of-scope, don't touch.

- **One commit per feature regen group** if the generated diff is large (e.g. `subscription/`, `invoicing/`, `billing/` separately). Otherwise one single commit `chore(mobile): regenerate freezed + json_serializable artefacts` is acceptable.

## Tests required

No new test files needed (P12 is a fix, not a feature). The validation is:
- 48 previously-broken tests now pass
- `flutter analyze lib/features/subscription/` clean (was reporting errors)
- `flutter analyze lib/features/invoicing/`, `lib/features/billing/` clean if they were affected
- `flutter test` overall failure count drops to 0 (or down to pre-existing flakes documented out-of-scope)

## OFF-LIMITS

- LiveKit / call code: `mobile/lib/features/call/`, `mobile/test/features/call/`. Never touch.
- `.github/workflows/*` — token can't push.
- Other plans (P6, P7, P8, P9, P10, P11) — never touch.
- Backend / web / admin — out of scope.
- `lib/features/subscription/` source files — only ADD missing annotations if absolutely required for build_runner to succeed; never refactor.

## Branch ownership

Agent creates `fix/p12-mobile-build-runner` from clean `main` via `git worktree add`. Never touches another branch.

## Final report (under 500 words)

Lead with PR URL.

1. Pre-fix vs post-fix counts:
   - analyzer errors on touched files (was X → 0)
   - test failures (was Y → 0 or pre-existing flake list)
   - generated files added/updated (count)
2. build_runner output last 20 lines (paste)
3. Any source-file annotation fixes needed (list with rationale)
4. Validation pipeline output (full paste)
5. "Branch ownership confirmed: only worked on `fix/p12-mobile-build-runner`"
6. Out-of-scope items flagged

GO. This should be quick — pure regeneration unless there's a real annotation bug.
