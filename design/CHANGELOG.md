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
