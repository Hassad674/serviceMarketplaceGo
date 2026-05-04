# Design RESUME — current state snapshot

> Read this file FIRST after a context compression or new session.
> Tells you exactly where the chantier is and what to do next.

---

## Last updated

`2026-05-04` — Phase 0 nearly complete (inventory + tracking landed).

---

## Where we are

**Phase 0** — Foundation almost done.

- ✅ Audit existing design rules in repo (Rose Contra superseded)
- ✅ Cleaned 4 CLAUDE.md files (root + web + admin + mobile) — old design system tokens removed, replaced by Soleil v2 references pointing here
- ✅ Created `design/` scaffold (this folder) with INDEX, DESIGN_SYSTEM, rules, RESUME, CHANGELOG, agent-templates, scripts, batches/, diffs/, assets/
- ✅ Copied Soleil v2 source assets (4 JSX + phase1/ + 18 HTML + 3 PDFs)
- ✅ Auto-memory entries written (4: design_system, off_limits, scope_discipline, progress_pointer)
- ✅ `inventory.md` — 41 unique screens (23 web + 18 mobile) with full route-repo mapping
- ✅ `tracking.md` — live status board, all screens currently `not-started`
- ⏳ Phase 0 batch — tokens implementation (web `globals.css` + admin `index.css` + mobile `soleil_theme.dart` + `Portrait` primitive web/mobile). NOT yet dispatched.
- ⏳ Validation scripts — written, syntax-checked, but not yet smoke-tested against a real diff.

**Phase 1** (calibration with 2-3 screens) — NOT STARTED.
Proposed candidates: W-01 Connexion, W-11 Dashboard freelance, W-16 Profil prestataire (public). See `tracking.md` for the rationale.

**Phase 2** (parallel agent batches: 1 web + 1 mobile in background) — NOT STARTED.

---

## What to do next

**If you're the orchestrator (Hassad / main session)**:

1. Validate the inventory by skimming [`inventory.md`](./inventory.md) — confirm the 41 screens are right and that the route mappings match your understanding of the repo.
2. Answer the 4 open questions at the bottom of [`tracking.md`](./tracking.md) (route ambiguities + Phase 0 batch ownership).
3. Pick the 2-3 screens for Phase 1 calibration (default: W-01 + W-11 + W-16, see `tracking.md`).
4. Decide who runs Phase 0 token batch — recommendation: orchestrator (Hassad+main session), because it touches `globals.css` which is OFF-LIMITS for agents by default.
5. After Phase 0 tokens land, dispatch the first calibration batch.

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
