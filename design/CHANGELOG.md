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

---

## 2026-05-04 (later) — Pre-work answers + commit hygiene + Android-only

After PR #108 merged, Hassad answered the 4 open questions in `tracking.md`:

1. ✅ W-12 Opportunités: only `/(public)/opportunities` exists — inventory locked to that route.
2. ✅ W-15 Mission active = same page as W-10 (`/projects/[id]`) with role-aware layout. One PR covers both IDs.
3. ✅ Web has `/(app)/notifications/` — added new entry **W-24 Notifications**, coupled with M-19. Inventory now totals 24 web + 18 mobile = 42 IDs (23 PRs because W-10+W-15 = 1 page).
4. ✅ Phase 0 batch ownership = orchestrator (main session). Validated.

Plus two new rules added to the chantier:

- `design/rules.md` §11 — **1 screen = 1 commit** (squash WIP before PR, `feat(design/<surface>/<id>): port <name> to Soleil v2` format)
- `design/rules.md` §12 — **Mobile = Android-only for now** (Hassad on Linux, no Mac/iOS Simulator). Code stays cross-platform; iOS captures added later without refactor.

Both rules also inserted into `design/agent-templates/shared-rules.md` (so every dispatched agent gets them) and into a new auto-memory entry `design_one_screen_one_commit.md`.

Branch: `chore/design-prework-answers`. PR #109.

Next: Phase 0 batch — implement Soleil v2 tokens in `web/src/styles/globals.css` + `admin/src/index.css` + `mobile/lib/core/theme/soleil_theme.dart` + create `Portrait` primitive web/mobile + load fonts. Orchestrator-run in main session.

---

## 2026-05-04 (later) — Phase 0 tokens + Portrait primitive

Foundation batch landed: Soleil v2 tokens are now live in the three apps and the `Portrait` primitive (6 deterministic palettes, SVG-only, no initials/emojis fallback) is available on web and mobile. Web `globals.css` rewires Tailwind tokens to Soleil; admin `index.css` mirrors them; mobile `core/theme/app_theme.dart` exposes the new `AppColors` ThemeExtension with 8 Soleil-specific fields (`subtleForeground`, `primaryDeep`, `accentSoft`, `successSoft`, `pink`, `pinkSoft`, `amberSoft`, `borderStrong`) on top of the 6 legacy aliases. Fraunces + Inter Tight + Geist Mono are loaded via `next/font/google` on web and `google_fonts` on mobile (JetBrains Mono temporarily standing in for Geist Mono pending package update).

Side effect: the legacy `cardShadowHover` getter was renamed to `cardShadowStrong` (1 BoxShadow instead of 2 — calmer Soleil shadow). The mobile test suite picked up compile errors that were addressed in a follow-up honesty PR.

Branch: `chore/design-phase-0-tokens`. PR #110.

---

## 2026-05-04 / 2026-05-05 — Phase 1 calibration

Three reference web screens + one mobile screen ported to Soleil v2 to lock the visual identity and validate the agent dispatch format.

- **W-01 Connexion** (orchestrator-implemented): editorial header, Fraunces title, italic corail accent, ivoire bg, Soleil card form with corail focus. PR #111.
- **W-11 Dashboard freelance** (orchestrator-implemented + Sidebar/Header extraction): full app shell port. Sidebar 280px wide, corail-soft active pill, Fraunces section heads. PR #112.
- **W-16 Profil prestataire** (agent-dispatched, calibration test): two PR sequence — first port (#114 `feat/design-w16-profil`) then a v2 cards restructure (#115 `feat/design-w16-profil-v2`) after Hassad's feedback. Validated that the brief format works for fresh agents.
- **M-01 Connexion mobile** (agent-dispatched): same login flow, Fraunces + corail accents on Pixel 5 emulator. PR #116.

Plus tracking + M-01 prep doc PR #113 between W-11 and W-16.

---

## 2026-05-05 — Auth + onboarding finish (Wave 0bis)

Closing the auth funnel on Soleil v2.

- **W-02 Inscription · choix de rôle** — three role cards (Prestataire / Client / Apporteur d'affaires), corail-soft active pill, Fraunces titles. PR #118.
- **W-23 Compte** — preferences page on Soleil card stack. PR #119.
- **W-18 Portefeuille** — hero card with Fraunces total, Geist Mono amounts, Soleil tones for incoming/outgoing. PR #121.
- **M-02 Inscription · choix de rôle mobile** — mobile mirror of W-02. PR #120.
- **M-16 mobile fix** — 4 polish fixes on `freelance_profile_screen.dart` (Portrait widget replaces initials avatar, corail StadiumBorder CTA, header meta strip with daily rate + availability dot, padding rebalanced). The PR was labelled M-16 but actually targets the M-13 file path (Profil prestataire mobile). PR #117.

---

## 2026-05-05 — Closing fixes wave (Wave A)

Fixes on top of the calibration wave that addressed Hassad's review feedback.

- **#122 register-fix** — W-03 / W-04 step-2 inscription forms (provider/agency/enterprise) ported to Soleil v2 (corail focus, Fraunces section heads, ivoire fields).
- **#123 compte-fix** — W-23 redesigned toggles (bigger pill, no TYPE column), new mobile `account_screen.dart` for M-20 + drawer entry.
- **#124 profil-fix** — W-16 v3 polish: max-w-4xl column on web, ProjectHistorySection lifted to LAST position, mobile parity fixes.
- **#125 wallet-fix** — W-18 responsive layout (md: breakpoint instead of sm:, hero card narrow padding) + mobile WalletHeroCard text visibility on light bg.
- **#126 toggle + profile width** — Soleil polish on the toggle pill geometry + max-w-5xl on profile shells.

---

## 2026-05-05 — Annonce + opportunités lifecycle (Wave A continued)

Full marketplace loop ported.

- **#127 opportunites** — W-12 feed (corail filter chips, Soleil cards) + W-13 détail opportunité + mobile mirror.
- **#128 mes-annonces** — W-06 entreprise list + M-07 mobile mirror.
- **#129 notifications** — W-24 web + M-19 mobile (corail unread dot, Soleil card per notification, Geist Mono timestamps).
- **#131 messagerie-widget** — W-21 page + Widget raccourci on dashboards + M-17 conversation + M-18 list mobile. Soleil bubbles (corail outgoing, ivoire incoming), system messages now visually distinct.
- **#132 creation-annonce** — W-09 web + M-09 mobile, multi-step form on Soleil cards.
- **#133 detail-annonce** — W-07 description + W-08 candidatures + edit + M-08 mobile.
- **#134 factures** — W-19 invoices list + M-15 mobile.
- **#135 profil-facturation** — W-20 billing profile form + mobile billing widget.

---

## 2026-05-05 — Boucle marketplace + system messages

- **#136 proposal-flow** — création + détail + pay + mobile. Single PR covers W-09 proposal create, W-10 client + W-15 provider detail (role-aware proposal-detail-view), payment-mode-toggle, milestone-tracker, payment-simulation. The mobile `proposal_detail_screen.dart` ported here implicitly covers M-06 Détail mission (the inventory entry that pointed at a non-existent standalone "détail mission" screen).
- **#137 system-messages-fix** — port of system messages bubbles to Soleil + overflow text polish on long names.

---

## 2026-05-05 — Closing wave (Wave B)

- **#138 mobile-dashboards** — M-03 freelance + M-04 entreprise dashboards ported (Soleil hero card, mini-stats row, recent activity Fraunces heads).
- **#139 kyc-visual** — W-05 Stripe Connect / KYC pages visual port (corail step indicators, Soleil status pills sapin / amber / corail).
- **#140 mobile-invoicing** — M-15 factures + billing profile screens deeper port (the M-15 lives in `mobile/lib/features/invoicing/`, not in the empty `invoice/` skeleton the inventory referenced).
- **#141 team-search** — W-22 Équipe & permissions (Soleil cards per member, role pills) + M-12 Recherche freelances mobile (filter sheet, Soleil cards).

---

## 2026-05-05 — Search-cast safety net

Tightening the Typesense client after live data exposed type drift.

- **#142 typesense-search-id-cast** — coerce `search_id` + `next_cursor` to String defensively in the SearchDocument decoder so an unexpected int from the server doesn't crash the mobile screen.
- **#143 search-document-string-coercion** — same pattern applied to all SearchDocument String fields (display_name, role, location, etc.). Pure resilience layer; no UI change.

---

## 2026-05-05 — Honesty + mobile test compile fix

V6 audit flagged two regressions:

- The mobile test suite no longer compiled — Phase 0's `AppColors` constructor now requires 8 additional Soleil-specific fields (`accentSoft`, `amberSoft`, `borderStrong`, `pink`, `pinkSoft`, `primaryDeep`, `subtleForeground`, `successSoft`), and the team R7 refactor renamed `otherUserName/otherUserRole` to `otherOrgName/otherOrgType` on `ConversationEntity`. The `messaging_repository_impl_test.dart` still called `startConversation(recipientId:)` instead of the new `recipientOrgId:` parameter. The `cardShadowHover` getter was renamed to `cardShadowStrong` (1 BoxShadow now). Compile errors went from 97 → 23 (the remaining 23 are pre-Phase 0 issues: integration_test invalidated by ApiClient.get adding `Options? options`, `provider_card.dart` removed by #adfac6c7, `credits_display_test.dart` orphan helper).
- `tracking.md` claimed "0 done / 21 web remaining / 18 mobile remaining" while reality was 27+ screens shipped. `RESUME.md` claimed "Phase 1 NOT STARTED" while Phase 1 + 2 + 3 + closing waves were all done. Both updated to reflect reality with PR refs verified against `git log`.

Branch: `fix/honesty-and-test-compile`. Tests + design docs only — zero touch to `mobile/lib/`, `web/src/`, `admin/src/`, or `backend/`.
