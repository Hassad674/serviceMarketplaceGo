# Agent batch — orchestrator audit checklist

> Run through this checklist for EVERY agent-dispatched batch before
> merging. ~5 min per batch. Catches drift before it compounds.

---

## 1. PR exists and is mergeable (30s)

```bash
gh pr view <pr-number> --json mergeable,mergeStateStatus,additions,deletions,changedFiles
```

- [ ] `mergeable: "MERGEABLE"` (not "CONFLICTING")
- [ ] `additions` and `deletions` look reasonable for the screen scope (a single screen port should be < 1500 LOC delta total)
- [ ] `changedFiles` count makes sense (1 page + 2-4 components + i18n + maybe layout + tests = 5-15 files)

---

## 2. Files changed — whitelist enforcement (1 min)

```bash
gh pr view <pr-number> --json files | jq -r '.files[].path'
```

For every path in the list, verify it's in the **TOUCHABLE** set per `design/rules.md` §2 + the batch brief. ZERO tolerance for:

- [ ] No file under `backend/`
- [ ] No file matching `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/**.ts`
- [ ] No `shared/lib/api-client.ts`, `api-paths.ts`, `types/api.d.ts`
- [ ] No `middleware.ts`, `next.config.ts`, `package.json`, `package-lock.json`
- [ ] No existing `*.test.ts*`, `*.spec.ts*`, `*_test.go`, `*_test.dart` modified
- [ ] No `mobile/**` for web batches (and vice-versa)

If any forbidden path appears in the diff: **REJECT** and either fix manually or re-dispatch.

---

## 3. Validation pipeline output present in PR body (30s)

The agent's PR body must contain the FULL output of:
- `tsc --noEmit` (clean)
- `vitest run` (touched files, all pass)
- `next build` or `flutter build` (succeeds)
- `check-api-untouched.sh` (no off-limits)
- `check-imports-stable.sh` (0 delta on api/hooks/schemas/zod)

If output is missing or claims success without showing the command output: **REJECT and ask for the output**. The agent might have skipped the pipeline.

---

## 4. Out-of-scope flagged section (1 min)

Read the agent's "Out-of-scope flagged" section. Sanity-check:

- [ ] Section exists (not empty unless explicitly noted "no design feature was absent from repo")
- [ ] Each flagged item maps to something the design source actually shows AND something the repo can't provide
- [ ] No invention spotted: the agent didn't add a new TanStack Query hook, didn't add a new field on an entity, didn't fabricate mock data

If the design clearly shows a section and the agent didn't flag it: **READ the new code and verify** the agent didn't silently invent it. If yes — REJECT.

---

## 5. Hooks consumed delta (30s)

If `check-imports-stable.sh` shows ANY positive delta on `api/`, `hooks/use-*`, or `schemas/`:

- [ ] The agent listed those new consumption sites in the report
- [ ] Each makes sense (e.g., dashboard now consumes `useMessages` to show "conversations en cours" — legitimate; vs. agent randomly imported a hook in a UI component — not legitimate)

A small positive delta is OK if explained. A delta on `from "zod"` is suspicious — usually means a new schema was created, which is OFF-LIMITS.

---

## 6. Visual diff (1 min)

The agent should have added screenshots in `design/diffs/<screen-id>/`:
- `before.png` (from `origin/main`)
- `after.png` (from agent's branch)
- `notes.md` (intentional differences vs source maquette)

If missing: ask the agent or capture yourself. Quick visual check:

- [ ] Soleil v2 identity recognizable (ivoire bg, corail accent, Fraunces serif headings)
- [ ] No legacy gradient (rose→purple), no blue-50 / violet-50 / emerald-50 hardcoded backgrounds
- [ ] Photos via `<Portrait>` (no initials)
- [ ] Layout matches the source maquette structure (don't be pixel-perfect strict on Phase 2 batches; the design source is inspiration not law)

---

## 7. CI status (15s)

```bash
gh pr view <pr-number> --json statusCheckRollup | jq '.statusCheckRollup[] | {name, conclusion}'
```

- [ ] All required checks green (or a known-acceptable subset for Soleil refactor PRs — e.g., E2E might not run on UI-only PRs)
- [ ] No CRITICAL flag from CodeQL / gosec / npm audit

---

## 8. Tracking + CHANGELOG update (1 min)

After merge, you (orchestrator) update:

- [ ] `design/tracking.md`: flip the screen status from 🟡 in-progress → 🟢 merged with PR ref
- [ ] `design/CHANGELOG.md`: append a one-paragraph entry
- [ ] If aggregate tables changed: update them (Done/In progress/Remaining counts)
- [ ] Memory hygiene: if a recurring drift pattern shows up across 2+ batches, add a `feedback_design_*.md` memory entry to anchor the rule

---

## Red flag patterns — auto-reject without further audit

- The agent says "I had to add `useFoo()` to make this work" → **reject**. No new hooks for UI batches.
- The agent says "tests were updated to match the new structure" → **reject**. Existing tests are the contract; if a test fails, fix the underlying string source (i18n key) or the impl, never the test.
- The agent says "I removed the `forgot password` link because the design doesn't show it" → **reject**. The agent removed a feature instead of skipping a design extra. Out-of-scope features are SKIPPED ADDITIONS, not REMOVALS.
- The agent's diff has > 2000 LOC for a single screen port → **reject**, ask for split.
- The agent's PR body has placeholder `[paste output]` text instead of actual output → **reject**, fake validation.

---

## Green flag patterns — fast-merge candidate

- Diff is `+200/-300` or thereabouts (simple visual rewrite of a few components)
- Validation pipeline output pasted verbatim, full
- Out-of-scope section clearly listed with reasons
- New i18n keys added in BOTH `fr.json` AND `en.json`
- Visual diff present
- Agent's "What I'd improve in the brief" section is candid (a vague "everything was clear" is suspicious)

---

## After 5 batches — drift audit

Every 5 merged agent batches, do a quick consistency pass:

```bash
git log --oneline -50 -- web/src/app web/src/features web/src/shared/components
```

Cross-batch checks:

- [ ] All ported screens use the same Portrait sizes / palette assignments per role
- [ ] All ported screens speak French via i18n (no hardcoded text in `.tsx`)
- [ ] All ported screens use consistent radii (rounded-xl / rounded-2xl / rounded-full)
- [ ] No agent re-introduced a legacy class (`gradient-primary-rose`, `bg-rose-500`, etc. — should never appear post-Phase 0)

If drift is found, write a `feedback_design_*.md` memory entry and tighten the next batch brief.
