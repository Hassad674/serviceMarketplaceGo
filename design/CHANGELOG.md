# Design CHANGELOG

Session-by-session log. Newest first. One entry per orchestrator action that changes the state of the chantier.

---

## 2026-05-04 — Phase 0 setup (foundation)

- Audited existing design rules across the repo. The "Rose Contra/Stitch/Airbnb" system in `design/DESIGN_SYSTEM.md` (827 lines) and the matching sections in `CLAUDE.md` (root, web, admin, mobile) are fully superseded by Soleil v2.
- Cleaned 4 CLAUDE.md files: removed all references to Rose primary, gradient-primary, shadow-glow, glass effects from the previous direction. Each now points to `design/INDEX.md`.
- Rewrote `design/DESIGN_SYSTEM.md` with the Soleil v2 token table (palette ivoire & corail, Fraunces + Inter Tight + Geist Mono, radii, spacing, shadows, components, motion, iconography, French language conventions).
- Created the `design/` scaffold: INDEX, RESUME, CHANGELOG, rules, agent-templates (web + mobile + shared), scripts (validate-no-regression, check-api-untouched, check-imports-stable), batches/, diffs/, assets/sources/, assets/pdf/.
- Copied source assets: 4 JSX files (`design-canvas`, `screens-editorial`, `screens-studio`, `system-cards`), 28 phase1/ files, 18 Atelier HTML files, 3 PDFs (web-desktop, web-responsive, app-native-ios).
- Wrote 4 auto-memory entries to anchor the chantier across sessions.
- Branch: `chore/design-foundation-soleil-v2`. PR: TBD (after this commit is pushed).

Next: write `design/inventory.md` (the 64 screens with mapping design ↔ route-repo). Orchestrator-only work, ~1-2h. Then Phase 1 calibration on 2-3 screens.

---

## 2026-05-04 (later) — Inventory + tracking landed

- Wrote `design/inventory.md` — 41 unique screens (23 web shared between desktop+responsive, 18 mobile Flutter), each with full mapping: design source (jsx file + lines + PDF page), route existante (Next.js or GoRouter), fichier principal, components touchables, OFF-LIMITS hooks/api/schemas, features design absentes du repo to skip, mobile parity coupling.
- Wrote `design/tracking.md` — live status board organized by phase (0: foundation, 1: calibration, 2: web batches, 3: mobile batches), all entries currently `not-started`. Aggregate counters at the bottom. Includes 4 open questions for orchestrator.
- Updated `RESUME.md` — reflects inventory+tracking complete; next steps clarified.
- Branch: `chore/design-inventory-and-tracking`. PR: TBD.

Next: Phase 0 token batch (orchestrator-runs since it touches `globals.css` OFF-LIMITS), then Phase 1 calibration on W-01 + W-11 + W-16.

---

## 2026-05-04 (later) — SOURCES doc + source locations memory

- Wrote `design/SOURCES.md` — complete reference: where the assets come from (3 levels: versioned in repo / Hassad's local Téléchargements / Claude Design canvas URLs), how to use each source file, when to refetch from external. Includes the exact prompts for Claude Design canvas refetch.
- Updated `design/INDEX.md` to reference the new SOURCES.md.
- Wrote auto-memory entry `design_source_locations.md` (reference type) — anchors the source paths and URLs so the chantier can survive a context compression.

This was a follow-up after Hassad noticed that the source paths and Claude Design refetch commands were nowhere documented despite their usefulness as a fail-safe.
