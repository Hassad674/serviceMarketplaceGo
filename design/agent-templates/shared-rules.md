# Agent — shared rules (included by every batch brief)

> This file is included verbatim into every batch brief. Common rules across web, admin, and mobile agents. Surface-specific rules live in `web-batch-brief.md` and `mobile-batch-brief.md`.

---

## Mandatory reads (in this order, before writing any code)

1. The CLAUDE.md of your surface: `web/CLAUDE.md` OR `admin/CLAUDE.md` OR `mobile/CLAUDE.md`.
2. The repo root `CLAUDE.md`.
3. `design/INDEX.md` (entry point).
4. `design/rules.md` (hard rules — read it twice).
5. `design/DESIGN_SYSTEM.md` (token reference).
6. The entries in `design/inventory.md` for the screens in your batch (only those entries, not the whole file).
7. The source assets your batch lists (in `design/assets/sources/`) — JSX files for visual reference, PDFs for layout proof.

---

## Branch ownership rule

You OWN the branch listed in your batch file. Never:
- Switch to another branch.
- Touch `/home/hassad/serviceMarketplaceGo/` (the user's main checkout).
- `git rebase` on `main` without explicit confirmation from the orchestrator.
- Push commits authored by anyone other than yourself.

You always:
- Run `git branch --show-current` first thing — confirm it matches the batch file's expected branch.
- Run `git status` — confirm working tree is clean (no leftover changes from a prior agent).

---

## OFF-LIMITS files

A complete list lives in `design/rules.md` §2. The short version:

- ZERO files under `backend/` may appear in your diff.
- No `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/*.ts`.
- No `shared/lib/api-client.ts` (web/admin) or `core/api/**.dart` (mobile).
- No `package.json` / `pubspec.yaml` / lockfiles.
- No existing test files.
- No `next.config.ts` / `vite.config.ts` / `middleware.ts`.

If your batch file says "Whitelisted exceptions: <list>", you may touch ONLY those listed files. Anything else outside the touchable set = round failed.

---

## Scope discipline (the cardinal rule)

Implement EXACTLY the screens listed in your batch, ni plus, ni moins.

If you spot an issue elsewhere ("ah, this other screen has a typo / missing alt-text"), DO NOT fix it. Add it to the batch report under `## Out-of-scope flagged`. The orchestrator handles it in a separate batch.

If a UI section in the design needs a feature absent from the repo (e.g., the "Atelier Premium" CTA, a "Cette semaine" editorial card, a saved-search), SKIP that section, FLAG it in the report, and continue. Do NOT invent a backend, hook, or zod schema to fill the gap.

---

## i18n — never hardcode user-visible strings

Every visible French string passes through the i18n layer:
- Web/admin: add to `web/messages/fr.json` (and `en.json` for parity if a key already exists in en).
- Mobile: add to `mobile/lib/l10n/app_fr.arb`.

If a string is already in the catalog under a different key, REUSE the key. Don't duplicate.

---

## Validation pipeline — run BEFORE every commit

Every commit on your branch MUST be preceded by:

```bash
design/scripts/validate-no-regression.sh
```

This script runs:
1. Backend build/vet/test (no backend file should be in your diff)
2. Web `tsc --noEmit` + `vitest run` (or admin equivalent)
3. Mobile `flutter analyze` + `flutter test` (scoped)
4. `check-api-untouched.sh` — diff vs origin/main, fail if OFF-LIMITS touched
5. `check-imports-stable.sh` — count imports of `*/api/*`, `*/hooks/*`, `zod` — fail if changed

If any step fails, you DO NOT commit. You fix the underlying issue. NEVER skip a step or comment-out a test.

---

## Visual diff — screenshots in your PR

For every screen you ship, capture before/after screenshots and save in `design/diffs/<screen-id>/`:
- `before.png` — pre-change (from `origin/main` — easiest to capture before you start)
- `after.png` — post-change (from your branch)
- `notes.md` — short description of intentional differences vs the source maquette

For mobile, use `flutter screenshot` against a connected emulator/device for both states.

---

## Final report (in your PR body, mandatory)

Use this structure:

```markdown
## Batch summary

[1 paragraph — what was done]

## Screens shipped

- `<inventory-id>` <screen name> — link to design/diffs/<screen-id>/
- ...

## Out-of-scope flagged (NOT implemented this batch)

- <screen-id> · <section> — feature absent from repo (<reason>). Flagged for orchestrator.
- ...

## Validation pipeline output

```text
[paste full output of design/scripts/validate-no-regression.sh]
```

## Files changed (whitelist check)

- Allowed: <list of paths under web/src/features/*/components/, web/src/styles/, etc.>
- Off-limits: NONE (or, if exception was granted in batch brief, list the file + cite the override line)

## Tests added

- <count> new tests in <files>

## Open questions for orchestrator

- <if any>
```

If any deliverable is partial (⚠️) or failed (❌), explain WHY. Don't paper over gaps.
