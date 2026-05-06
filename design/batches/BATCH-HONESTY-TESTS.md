# BATCH-HONESTY-TESTS — fix mobile test compile + tracking honesty

> Worktree: `/tmp/mp-honesty-tests` · Branch: `fix/honesty-and-test-compile` · Base: `origin/main` (49650494)

## Context — V6 audit findings to close

- **H6 — Mobile test suite broken**: `flutter analyze` returns 97 errors in `test/` and `integration_test/`. Specifically `test/core/theme/app_theme_test.dart:109,126,146,154,172` calls `AppColors(...)` constructor with the OLD signature (before Phase 0 added `accentSoft`, `amberSoft`, `borderStrong`, `pink`, `pinkSoft`, `primaryDeep`, `subtleForeground`, `successSoft` as required positional/named params). Plus drift on messaging entity (`otherUserName` / `otherUserRole` were removed but tests still reference them). Result: `flutter test` cold = compile FAIL. This invalidates the "test before commit" rule because per-feature scoped tests pass while the root suite doesn't compile.
- **M1 — tracking.md is 26 PRs in retard**: `design/tracking.md` says "0 done / 21 web remaining / 18 mobile remaining" and `design/RESUME.md` says "Phase 1 NOT STARTED". Reality: **27+ screens shipped** (W-01, W-02, W-06, W-07, W-08, W-09, W-11, W-12, W-13, W-16, W-18, W-19, W-20, W-21, W-22, W-23, W-24, W-05 KYC, W-10/W-15, plus M-01..M-19). The doc tells a lie — fix it.

## Goal — ONE squashed commit
Make `flutter analyze test/` clean again + update `tracking.md`, `RESUME.md`, `CHANGELOG.md` to reflect actual progress honestly.

## TOUCHABLE files (exhaustive)

### Mobile — test compile fix
- `mobile/test/core/theme/app_theme_test.dart` (update `AppColors(...)` constructor calls to match the NEW signature in `mobile/lib/core/theme/app_theme.dart`)
- Any other test file under `mobile/test/` flagged by `flutter analyze` with errors related to:
  - `AppColors` constructor mismatch
  - Messaging entity drift (`otherUserName`, `otherUserRole` removed)
  - Any other compile-time error introduced during Soleil chantier
- If a test asserts on a removed/renamed entity field, update the assertion to reference the new field — DO NOT remove the test, DO NOT skip it
- `mobile/integration_test/**.dart` if compile errors there too

### Design tracking docs
- `design/tracking.md` — update screen statuses to match reality. Run `git log --oneline --since=2026-05-04 -- design/diffs/ web/src/app web/src/features mobile/lib/features` to identify which screens shipped, then flip their status from ⚪ → 🟢 with PR links. Update aggregate counts.
- `design/RESUME.md` — update "Phase X status" markers to reflect that Phase 1 + Phase 2 + closing wave are done. Document M-06 / M-15 / M-13 SKIP rationale.
- `design/CHANGELOG.md` — append entries for the major waves (Phase 0 tokens, Phase 1 calibration, Wave A, Wave B, annonce lifecycle, boucle marketplace, system messages fix, closing wave, atoms cleanup, search-cast fixes). One paragraph per wave, link the PR.

## OFF-LIMITS — STRICT
- ALL `mobile/lib/**` (this batch is test + docs only — DO NOT modify production mobile code)
- ALL `web/**`, `admin/**`, `backend/**` (test + docs only)
- `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`
- `package.json`, `pubspec.yaml`, lockfiles
- `mobile/lib/core/theme/app_theme.dart` itself (the test should match the source, not vice-versa — if `AppColors(...)` constructor changed, update the TEST to match. If the source is wrong, that's a separate problem flagged but not fixed here.)

If `flutter analyze test/` reports errors that require touching production code (`mobile/lib/`), STOP and write a `BLOCKED-honesty-tests.md` at worktree root explaining what needs to change in production code — do NOT modify production.

## Acceptance criteria

### Test compile fix
- `cd mobile && flutter analyze --no-pub test/ integration_test/` → returns "No issues found" (or only pre-existing issues that exist on origin/main BEFORE this PR — verify by checkout-comparison)
- `cd mobile && flutter test --no-pub` → cold suite compiles and runs (failures may persist where they were already failing on origin/main, but compile is clean)

### Documentation honesty
- `design/tracking.md` aggregate table: shows "Done" count = 27+ (verify by counting merged screens in git log), "In progress" = 0, "Remaining" = 0 or only the SKIP-flagged items
- `design/RESUME.md`: ledger reflects current state — Phase 0 done, Phase 1 done, Wave A done, Wave B done, annonce lifecycle done, boucle marketplace done, polish + atoms done. SKIPs documented (M-06, M-15-skeleton, M-13).
- `design/CHANGELOG.md`: chronological entries with PR refs for each wave merged this session (~17-20 PRs)

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-honesty-tests

# 1. Scope check
git diff --name-only origin/main...HEAD | grep -E "^(backend/|web/|admin/|mobile/lib/)" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"

# 2. Mobile analyze on tests
cd mobile
flutter pub get
flutter analyze --no-pub test/ integration_test/ 2>&1 | tail -20

# 3. Mobile cold test compile
flutter test --no-pub --no-test-assets 2>&1 | tail -20 || echo "(see test failures — distinguish pre-existing from regressions)"

# 4. Markdown lint sanity (light)
cd ..
ls -la design/tracking.md design/RESUME.md design/CHANGELOG.md
```

ALL must pass. Fix loop max 3.

## Quality bar
- ZERO touch to production code (`mobile/lib/**`, `web/src/**`, `admin/src/**`, `backend/**`)
- Test SEMANTICS preserved — only update CONSTRUCTOR ARGS or FIELD NAMES that drifted
- Documentation accuracy — verifiable against `git log`, not invented
- ONE squashed commit
- DO NOT modify `git config`

## Push + PR
- Message: `chore(design): fix mobile test compile + update tracking/RESUME/CHANGELOG honestly`
- PR title: `[chore/honesty] Mobile test compile fix + design docs accurate state`

## Final report (under 500 words)
Standard structure. INCLUDE a "Honesty corrections" section listing what the docs claimed vs what reality is.
