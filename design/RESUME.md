# Design RESUME — current state snapshot

> Read this file FIRST after a context compression or new session.
> Tells you exactly where the chantier is and what to do next.

---

## Last updated

`2026-05-05` — Closing wave landed. 37 / 42 screens shipped, 3 skipped explicitly, 2 remain.

---

## Where we are

**Phase 0 — Foundation** ✅ DONE.

- ✅ Audit existing design rules in repo (Rose Contra superseded)
- ✅ Cleaned 4 CLAUDE.md files (root + web + admin + mobile) — old design system tokens removed, replaced by Soleil v2 references pointing to `design/INDEX.md`
- ✅ Created `design/` scaffold with INDEX, DESIGN_SYSTEM, rules, RESUME, CHANGELOG, agent-templates, scripts, batches/, diffs/, assets/
- ✅ Copied Soleil v2 source assets (4 JSX + phase1/ + 18 HTML + 3 PDFs)
- ✅ Auto-memory entries written
- ✅ `inventory.md` — 42 unique screens (24 web + 18 mobile) with full route-repo mapping
- ✅ `tracking.md` — live status board
- ✅ Phase 0 batch — Soleil v2 tokens (web `globals.css` + admin `index.css` + mobile `app_theme.dart` + `Portrait` primitive). Merged in PR #110.
- ✅ Validation scripts — written, smoke-tested.

**Phase 1 — Calibration** ✅ DONE (PRs #111, #112, #114/#115, #116).
W-01 + W-11 + W-16 (web) + M-01 (mobile) shipped, agent dispatch format validated.

**Phase 2 — Web batches** ✅ DONE (22 / 24 screens shipped).
Auth + onboarding, annonces, opportunités, profil, argent, messagerie, équipe, compte, notifications all on Soleil v2. W-14 Mes candidatures is the only web screen that has not been ported yet. W-17 was a duplicate of W-16 (same page) — explicit SKIP.

**Phase 3 — Mobile batches** ✅ DONE (15 / 18 screens shipped).
M-01..M-04, M-07..M-09, M-12, M-13, M-14, M-15, M-17, M-18, M-19, M-20 all on Soleil v2. Skips:
- **M-06 Détail mission** — covered by `proposal_detail_screen.dart` ported in #136 (proposal-flow). The mobile screen mirrors the W-10/W-15 web page; no separate "détail mission" screen exists.
- **M-15 invoice/ skeleton** — the inventory pointed at `mobile/lib/features/invoice/` which was a near-empty domain skeleton. The real M-15 lives in `mobile/lib/features/invoicing/` and was ported in #134/#140.
- **M-16 Détail paiement** — inventory entry pointed at a screen that never existed in the repo. The PR labelled `mobile/M-16-fix` (#117) actually polished `freelance_profile_screen.dart`, which is M-13. M-16 is therefore explicitly skipped — there is no standalone "détail paiement" screen to port.

Remaining: M-05 Mes candidatures (mirror of W-14).

**Closing wave + mini-fixes** ✅ DONE.
- #117 mobile M-16 fix (polish freelance_profile_screen) — actually targeted M-13 file path
- #122 register fix (W-03 / W-04 step-2 forms)
- #123 compte fix (W-23 toggles + new mobile account screen + drawer)
- #124 profil fix (W-16 width + section order + mobile parity)
- #125 wallet fix (W-18 responsive + mobile police visibility)
- #126 toggle + profile width polish
- #137 system messages port + overflow polish

**Atoms cleanup + search-cast fixes** ✅ DONE.
- #142 Typesense search ID coercion
- #143 SearchDocument String fields defensive coercion

---

## What to do next

**If you're the orchestrator (Hassad / main session)**:

1. Decide whether W-14 Mes candidatures (web) and M-05 Mes candidatures (mobile) need a Soleil v2 port now or can be deferred to a follow-up round. Both screens still wear the legacy chrome.
2. Mobile test compile fix has landed — `flutter analyze test/` is clean for chantier-introduced compile errors. The remaining 23 errors are all pre-Phase 0 issues (`integration_test/*` invalidated by April 28 ApiClient.get signature change, `provider_card.dart` removed in #adfac6c7, `credits_display_test.dart` orphan helper). Track separately.
3. If you have time: fix the legacy `cardShadowHover`-era assertions in `app_theme_test.dart` — they compile clean now but still assert rose-500 / slate-50 values that no longer exist on Soleil v2. Either rewrite the values to Soleil expectations or leave them as documented runtime failures.

**If you're a fresh agent dispatched on a batch**:

1. You should NOT be reading RESUME.md as your starting point. Read your batch file in `design/batches/BATCH-XXX-...md` — it has your specific instructions.
2. RESUME.md is for orchestrator recovery, not agent dispatch.

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
