# Design RESUME — current state snapshot

> Read this file FIRST after a context compression or new session.
> Tells you exactly where the chantier is and what to do next.

---

## Last updated

`2026-05-04` — Phase 0 setup in progress (this commit).

---

## Where we are

**Phase 0** — Foundation in progress.

- ✅ Audit existing design rules in repo (Rose Contra superseded)
- ✅ Cleaned 4 CLAUDE.md files (root + web + admin + mobile) — old design system tokens removed, replaced by Soleil v2 references pointing here
- ✅ Created `design/` scaffold (this folder) with INDEX, DESIGN_SYSTEM, rules, RESUME, CHANGELOG, agent-templates, scripts, batches/, diffs/, assets/
- ✅ Copied Soleil v2 source assets (4 JSX + phase1/ + 18 HTML + 3 PDFs)
- ⏳ `inventory.md` — not yet written (TODO: 64 screens × mapping route-repo). This is the next big chunk of orchestrator work, ~1-2h, NOT to be delegated.
- ⏳ `tracking.md` — not yet written (depends on inventory.md being done first)
- ⏳ Auto-memory entries — to be written in the same session as scaffold
- ⏳ Validation scripts — written but not yet smoke-tested against a real diff

**Phase 1** (calibration with 2-3 screens) — NOT STARTED.

**Phase 2** (parallel agent batches) — NOT STARTED.

---

## What to do next

**If you're the orchestrator (Hassad / main session)**:

1. Validate the scaffold by reading [`INDEX.md`](./INDEX.md), [`DESIGN_SYSTEM.md`](./DESIGN_SYSTEM.md), [`rules.md`](./rules.md).
2. Confirm the OFF-LIMITS list in `rules.md` §2 matches the actual repo file structure.
3. Greenlight the orchestrator to write `inventory.md` (the 64 screens with mapping). This file is sensitive — orchestrator-only, no agent.
4. Once `inventory.md` is done, pick the 2-3 screens for Phase 1 calibration.
5. Run Phase 0 token implementation (web globals.css + admin index.css + mobile soleil_theme.dart) — small batch, can be a single agent or done by orchestrator.

**If you're a fresh agent dispatched on a batch**:

1. You should NOT be reading RESUME.md as your starting point. Read your batch file in `design/batches/BATCH-XXX-...md` — it has your specific instructions.
2. RESUME.md is for orchestrator recovery, not agent dispatch.

---

## Open questions / TODO for orchestrator

- [ ] Confirm that the Soleil v2 fonts (Fraunces, Inter Tight, Geist Mono) are loaded in the existing web `next/font` setup or need new entries.
- [ ] Confirm the mobile `google_fonts` package is already in `pubspec.yaml` or needs adding (allowed only in Phase 0 token batch).
- [ ] Decide if the admin app gets the same Soleil treatment or stays "minimal admin chrome" (the maquettes don't cover admin specifically).
- [ ] Validation scripts (`validate-no-regression.sh` etc.) — smoke-test on a no-op diff to confirm exit codes.

---

## Recovery commands

```bash
# Where am I in the chantier?
cat design/RESUME.md design/tracking.md 2>/dev/null

# What was done recently in design/?
git log --oneline -20 -- design/

# Any open PRs?
gh pr list --state open --label design

# All batches dispatched?
ls design/batches/ 2>/dev/null
```
