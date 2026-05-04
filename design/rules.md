# Design — Hard rules (non-negotiable)

> Every UI agent reads this BEFORE touching anything. Violation = batch round annulé, regardless of how good the visual outcome looks.

---

## 1. The cardinal rule — zero regression

This chantier is **purely UI / organisation**. The agent is shipping the visual identity Soleil v2 onto already-functional screens. The agent is **NOT** allowed to:

- Change any API call.
- Change any backend endpoint, schema, or migration.
- Touch performance optimizations (memo, useMemo, dynamic imports) unless mandated by the design.
- Drop or rewrite tests "to make them pass".
- Bump dependencies.
- Change routing.

Result: **the app must behave EXACTLY the same after the agent's PR**. Same login flow, same data on screen, same errors handled the same way. The only thing that changes is how it LOOKS.

---

## 2. OFF-LIMITS files (no exceptions)

The agent's PR diff MUST NOT touch these paths — `validate-no-regression.sh` enforces this.

### Backend

```
backend/**         (zero file allowed in the diff — this is a frontend chantier)
```

### Web — data + transport layer

```
web/src/features/*/api/**.ts            (HTTP client wrappers per feature)
web/src/features/*/hooks/use-*.ts       (TanStack Query hooks, mutations)
web/src/features/*/schemas/**.ts        (zod schemas)
web/src/shared/lib/api-client.ts        (transport)
web/src/shared/lib/api-paths.ts         (typed paths)
web/src/shared/types/api.d.ts           (OpenAPI-generated types)
web/middleware.ts                       (auth/routing middleware)
web/next.config.ts                      (build config)
web/package.json                        (deps)
web/package-lock.json
```

### Admin — same families

```
admin/src/features/*/api/**.ts
admin/src/features/*/hooks/use-*.ts
admin/src/features/*/schemas/**.ts
admin/src/shared/lib/api-client.ts
admin/src/shared/types/api.d.ts
admin/package.json
admin/package-lock.json
admin/vite.config.ts
```

### Mobile — same families

```
mobile/lib/core/api/**.dart             (Dio client)
mobile/lib/core/network/**.dart         (interceptors)
mobile/lib/features/*/data/**.dart      (data sources, repositories)
mobile/lib/features/*/domain/**.dart    (entities, use cases)
mobile/pubspec.yaml
mobile/pubspec.lock
```

### Tests (any change to a test file is forbidden)

```
**/*.test.ts*                           (web vitest)
**/*.spec.ts*
**/*_test.go                            (backend Go)
**/*_test.dart                          (mobile)
backend/test/**
web/e2e/**                              (Playwright)
```

If a test fails BECAUSE of the agent's UI change (e.g., a `getByText("Soleil v2 nouveau label")` no longer matches), the agent's job is to **fix the underlying string source** (i18n key in `messages/fr.json`) so the test continues to query the same value, not edit the test.

### Whitelisted exceptions

A file in the OFF-LIMITS list can be touched ONLY if the batch brief explicitly allows it (e.g., Phase 0 batch is allowed to touch `globals.css` to set up tokens). The override is one-shot per batch and named in the batch file.

---

## 3. Scope discipline — design has features, repo has features

The Soleil v2 source includes features that **may not exist in the repo**:

- "Atelier Premium" subscription tier (sidebar bottom CTA in `soleil.jsx`)
- "Cette semaine chez Atelier" editorial card (dashboard hero)
- Saved-search filters
- "Trio portraits flottants" decorative blocks
- Some metric chips that may not have a backend stat behind

**Rule**: if a UI section needs data the repo cannot provide,

1. SKIP the section entirely — do not render an empty card, do not render a stub.
2. Replace it with a sensible neighbour layout adjustment (e.g., expand the next column to fill the row).
3. FLAG in the batch report under `## Out-of-scope flagged (NOT implemented)` with the section name + screen + reason.

The orchestrator decides later whether to spec a real feature for it. **The agent does NOT invent backend, hooks, or zod schemas to fill the gap.**

Decorative purely-visual elements (blobs, gradients, illustrations) are always allowed — they need no data.

---

## 4. i18n is mandatory

Every user-visible French string in the design must be added to the i18n catalog, NEVER hardcoded in `.tsx`/`.dart`.

Web: `web/messages/fr.json` (and `en.json` for English fallback). Mobile: `mobile/lib/l10n/app_fr.arb`.

If a string is already in the catalog under a different key, **reuse the key**. Do not duplicate.

---

## 5. Server vs Client (Next.js) — don't downgrade

In `web/`, do NOT add `"use client"` to a Server Component just because the design suggests an interactive feel. If the screen has zero state and zero event handlers, it stays a Server Component (massive perf benefit).

If the design genuinely requires client interactivity (modal, tab toggle), wrap **only the interactive subtree** in a client component, leave the rest server.

---

## 6. Reorganisation = soft migration

Moving a component within the codebase is allowed. Renaming a component is allowed. Both come with one constraint:

- If the component is publicly exported (importable from another feature or `app/`), keep a **re-export** at the old path with `// deprecated, kept for 1 release — moved to <new path>`. Hard removal happens in a follow-up PR after 1 release.

This avoids breaking imports across batches that run in parallel.

---

## 7. Validation pipeline — paste output or round annulé

Before EVERY commit, the agent runs:

```bash
design/scripts/validate-no-regression.sh
```

This runs:
1. `go build ./... && go vet ./... && go test ./... -count=1` (backend untouched check)
2. `tsc --noEmit && vitest run` (web)
3. Admin: `tsc --noEmit && vitest run` if applicable
4. `flutter analyze && flutter test` (mobile, scoped to touched dirs)
5. `check-api-untouched.sh` — diff vs origin/main, fails if any OFF-LIMITS path is modified
6. `check-imports-stable.sh` — counts imports of `*/api/*`, `*/hooks/use-*`, `zod` schemas — fails if a counter changed

The script is **noisy on purpose** — it must obviously fail loud rather than silently pass. The agent pastes the entire stdout in the PR body under `## Validation pipeline output`.

---

## 8. Visual diff — screenshots in PR

For every screen the agent ships, include in `design/diffs/<screen-id>/`:

- `before.png` — captured against `origin/main` (instructions: spin up dev server pre-change, screenshot)
- `after.png` — captured against the agent's branch
- `notes.md` — what differs intentionally vs source maquette

This is the orchestrator's visual review surface. ~30 sec per screen to audit.

For mobile, screenshots from a connected device or simulator (the agent runs `flutter screenshot` after each screen).

---

## 9. Commit + PR conventions

- One PR per batch.
- Commit per logical step (e.g., "feat(web/auth): port login screen to Soleil v2", "chore(web): rename old auth-form atoms").
- PR title: `[design/<surface>/<batch-id>] <topic>` — example: `[design/web/B-005] Port auth screens to Soleil v2`.
- PR body MUST include:
  - Screens shipped (with `inventory.md` ID)
  - Off-scope flagged (sections skipped + why)
  - Validation pipeline output
  - Files changed summary (matched against OFF-LIMITS check)
  - Links to `design/diffs/<screen>/before.png` and `after.png`
- After merge, orchestrator updates `tracking.md` + appends to `CHANGELOG.md`.

---

## 10. When in doubt — ask, don't invent

If the brief is ambiguous, the source maquette unclear, or a feature mapping uncertain: STOP. Append a question to the batch file under `## Open questions` and return to orchestrator. The cost of a 5-min Q&A is far below the cost of a wrong implementation.

The agent never:
- Picks a different palette because "the maquette one is muddy"
- Invents a route that "would make sense"
- Adds a state or hook to fill design polish (animations, optimistic UI) without explicit greenlight

The agent always:
- Reproduces the visual identity exactly (within the limit of repo features)
- Calls out gaps in the report
- Stays inside the whitelist of touchable files

---

## 11. One screen = one commit (commit hygiene)

Each screen ID (`W-XX` or `M-XX`) ships in **exactly one final commit** on the batch branch, with the message format:

```
feat(design/<surface>/<id>): port <screen-name> to Soleil v2
```

Examples:
- `feat(design/web/W-01): port Connexion to Soleil v2`
- `feat(design/mobile/M-03): port Dashboard freelance to Soleil v2`
- `feat(design/web/W-10+W-15): port Détail projet (role-aware client + provider) to Soleil v2` (combined when 2 IDs share one page)

### Why

- 1 commit = 1 reviewable unit. The PR reviewer can navigate by commit, identify which file changes go with which screen, and revert one screen's port without losing the others if a regression is found.
- Cross-batch traceability: `git log --grep="design/web/W-01"` returns exactly the commit that ported W-01. Useful for follow-up bug hunts.
- `tracking.md` updates: when a screen flips to 🟢 merged, the commit SHA gets pinned in the row. One commit = one SHA = clean reference.

### How

During the batch, the agent may make multiple intermediate WIP commits ("wip: layout draft", "wip: pull tokens"). **Before opening the PR**, those WIP commits are squashed into the single canonical commit per screen:

```bash
git rebase -i origin/main
# squash all "wip:" commits into the canonical "feat(design/...)" commit
# keep one feat commit per screen ID (or per combined ID pair)
```

If a batch ships 3 screens (e.g., the auth lot W-01 + W-02 + W-03), the final history on the branch is 3 commits — one per screen, in inventory order. Not 1 mega-commit, not 17 micro-commits.

### Exceptions

- **Tooling commits** are OK to keep separate from the screen commits: e.g., a `chore(design/web): add Portrait primitive helper` if the agent needs to extract a small helper. Keep these in their own commit, before the screen commit that consumes them.
- **Tests-only commits** are OK to keep separate: `test(design/web/W-01): regression assertions for new layout`. Many small test commits are fine, they don't pollute the screen-commit history.

The CI checks the PR's commit message format and fails fast if a screen ID is referenced in 0 or 2+ commits without explanation. (The check is on the orchestrator's TODO; for now it's a manual review item.)

---

## 12. Mobile testing — Android-only for now

Hassad's local environment runs Linux. **No Mac, no iOS Simulator** as of 2026-05-04. Mobile work proceeds with these constraints:

- **Validation device** = Android emulator (AVD) or a physical Android via wireless debug. Default target = Pixel 5 emulator (matches the design viewport 390×844).
- **Screenshot diffs** = `before-android.png` and `after-android.png` in `design/diffs/<screen-id>/`. Don't fabricate iOS captures — they'll be added when a Mac is available.
- **Code stays cross-platform**. Don't use Cupertino-only widgets where Material 3 has equivalents. Don't hardcode platform checks like `if (Platform.isAndroid)` for visual reasons. The Soleil v2 theme is platform-agnostic (the iOS frame in the design source is inspirational, not literal).
- **iOS-specific features** that the design references (status bar styling, safe areas with iOS-typical inset, navigation back-swipe gestures) are still implemented — Material 3 + the right `MediaQuery.padding` handles them gracefully on both platforms. Don't no-op them just because we test on Android.
- **CI**: `flutter test` and `flutter analyze` are platform-agnostic, no special handling needed. The mobile-test job in CI will continue to pass without iOS-specific paths.

When a Mac is available later, no refactor needed. iOS Simulator captures simply get added to existing `design/diffs/` folders, and an iOS run is added to CI. The Soleil v2 implementation is portable as-is.
