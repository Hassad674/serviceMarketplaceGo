# Design — Atelier · Direction Soleil v2

> Single source of truth for the visual identity of the Atelier marketplace.
> Read this file first. Every UI agent (web, admin, mobile) must navigate from here.

---

## What is "Soleil v2"

The visual direction shipped from a dedicated Claude Design session. Palette ivoire & corail, typographie Fraunces (display) + Inter Tight (UI) + Geist Mono (numbers). Crafted maquettes for desktop, responsive web, and native iOS/Flutter app. Editorial tone, French language (tutoiement), photos via stylized SVG portraits.

**This replaces all earlier visual rules** (the "Rose Contra/Stitch/Airbnb" system referenced in older docs is fully superseded as of 2026-05-04).

---

## Documents in this folder

| File | Purpose | When to read |
|------|---------|--------------|
| [`INDEX.md`](./INDEX.md) | This file. Entry point. | Always first. |
| [`DESIGN_SYSTEM.md`](./DESIGN_SYSTEM.md) | Tokens (colors hex, fonts, radii, spacings, components). | Before touching any token or primitive. |
| [`rules.md`](./rules.md) | Non-negotiable rules (off-limits files, scope discipline, validation pipeline). | Before every batch. |
| [`inventory.md`](./inventory.md) | The 64 screens with mapping design ↔ route-repo + status. | Before picking a batch. |
| [`tracking.md`](./tracking.md) | Live progress board (status per screen). | To pick the next batch. |
| [`RESUME.md`](./RESUME.md) | Snapshot of where we are right now. | After a context compression / new session. |
| [`CHANGELOG.md`](./CHANGELOG.md) | Session-by-session log. | To audit history. |
| [`agent-templates/`](./agent-templates/) | Brief templates for dispatched agents. | When dispatching a new batch. |
| [`scripts/`](./scripts/) | Validation scripts (no-regression, api-untouched, imports-stable). | Run before every commit. |
| [`batches/`](./batches/) | One file per dispatched batch (brief + status + audit). | To review a specific batch. |
| [`diffs/`](./diffs/) | Before/after screenshots per screen. | For visual review. |
| [`assets/`](./assets/) | Source designs (4 JSX + 18 HTML + 3 PDFs). | When briefing an agent on a screen. |

---

## How to use this folder

### As the orchestrator (Hassad / Claude main session)

1. Pick the next batch from [`tracking.md`](./tracking.md).
2. Open [`inventory.md`](./inventory.md) for the screens in that batch — note the `Route existante`, `Components touchables`, `Off-limits`, and `Features design absentes`.
3. Open the matching template in [`agent-templates/`](./agent-templates/), fill in the variables.
4. Create `batches/BATCH-XXX-<surface>-<topic>.md` with brief + dispatch metadata.
5. Dispatch the agent on a fresh worktree + branch.
6. After agent reports back, audit (5 min), update `tracking.md`, append to `CHANGELOG.md`.

### As a dispatched agent (you)

1. **Read** `web/CLAUDE.md` (or `admin/`, `mobile/` per your surface) — your stack rules.
2. **Read** `design/INDEX.md` (this file).
3. **Read** `design/rules.md` — the hard rules.
4. **Read** `design/DESIGN_SYSTEM.md` — the tokens.
5. **Read** `design/inventory.md` only the entries for your batch.
6. **Read** the source assets your batch references (in `design/assets/`).
7. Implement, with paranoid validation.
8. Run `design/scripts/validate-no-regression.sh` before EVERY commit.
9. Open the PR, paste the script output in the body, link the batch file.

### After a context compression (you, again)

1. Read `design/RESUME.md` — what's the current state.
2. `git log --oneline -20 -- design/` — recent activity.
3. `gh pr list --state open` — pending work.
4. Resume from the first not-started screen in `tracking.md`.

---

## Anti-pattern: do NOT copy the design literally

The Claude Design output shows features that **may not exist in the repo** ("Atelier Premium" subscription tier, "Cette semaine chez Atelier" editorial card, saved-search filters, etc.). The agent's job is **NOT to invent backend or hooks** for those.

**Rule**: if a UI section maps to data the repo cannot provide, the agent SKIPS that section, FLAGS it in the batch report under "Out-of-scope flagged", and continues with the rest. The orchestrator decides later whether to spec a real feature for it.

See [`rules.md`](./rules.md) §3 for the full discipline.

---

## Stack confirmation

| Surface | Design says | Repo has | Status |
|---------|-------------|----------|--------|
| Web Desktop (1440px) | React + Tailwind | Next.js 16 + React 19 + Tailwind 4 | ✅ aligned |
| Responsive Web (390px) | Same web codebase, breakpoints | Next.js (responsive of `web/`) | ✅ aligned |
| App Native iOS (390×844) | Flutter + Material 3 | Flutter 3.16+ Material 3 | ✅ aligned |

The Soleil v2 source files in `assets/sources/*.jsx` are **inspiration only** — they use vanilla React with inline styles. We re-implement using the repo's idiomatic patterns (Tailwind utility classes for web/admin, Material 3 + theme extension for mobile).
