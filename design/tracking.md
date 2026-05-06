# Design — Tracking (live status board)

> Source-of-truth status per screen. Update after every batch merge.
> See `inventory.md` for the full mapping (ID ↔ source ↔ route ↔ files ↔ off-limits).

**Legend** :
- ⚪ `not-started` — no batch dispatched yet
- 🟡 `in-progress` — batch dispatched, agent working
- 🔵 `in-review` — PR open, awaiting orchestrator audit
- 🟢 `merged` — PR merged, screen marked done
- ⚫ `skipped` — explicitly skipped (covered by another screen, feature absent, etc.)
- 🔴 `blocked` — blocker found, see batch file

---

## Phase 0 · Foundation

| Item | Status | Notes / PR |
|------|--------|------------|
| Foundation scaffold (CLAUDE.md, design/, scripts) | 🟢 | PR #107 |
| `inventory.md` + `tracking.md` | 🟢 | PR #108 |
| Pre-work answers + 1-screen-1-commit + Android-only rules | 🟢 | PR #109 |
| Phase 0 batch — Soleil v2 tokens (web `globals.css` + admin `index.css` + mobile `app_theme.dart` + `Portrait` primitive web/mobile) | 🟢 | PR #110 |

---

## Phase 1 · Calibration (3 reference screens + 1 mobile)

> Goal: ship 3 representative web screens + 1 mobile screen to lock the visual identity, the brief format, and validate that an agent can carry the load before going parallel.

| ID | Screen | Mode | Status | PR |
|----|--------|------|--------|-----|
| W-01 | Connexion | Orchestrator-implemented, manual | 🟢 | #111 |
| W-11 | Dashboard freelance | Orchestrator-implemented, manual + Sidebar/Header extraction | 🟢 | #112 |
| W-16 | Profil prestataire | **Agent-dispatched** (calibration: tests if the brief format holds against a fresh agent) | 🟢 | #114, #115 (v2) |
| M-01 | Connexion mobile | Agent-dispatched | 🟢 | #116 |

---

## Phase 2 · Web batches

> Each row = one screen unique to web (desktop + responsive share files). Status updates after batch merge.

### 1 · Auth & onboarding (5)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-01 | Connexion | 🟢 | calibration-1 | #111 |
| W-02 | Inscription · choix de rôle | 🟢 | w02-register | #118 |
| W-03 | Inscription · prestataire | 🟢 | register-fix | #122 |
| W-04 | Inscription · entreprise | 🟢 | register-fix | #122 |
| W-05 | Stripe Connect · KYC | 🟢 | kyc-visual | #139 |

### 2 · Entreprise · annonces & projets (5)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-06 | Mes annonces (liste) | 🟢 | mes-annonces | #128 |
| W-07 | Détail annonce · description | 🟢 | detail-annonce | #133 |
| W-08 | Détail annonce · candidatures | 🟢 | detail-annonce | #133 |
| W-09 | Création d'une annonce | 🟢 | creation-annonce | #132 |
| W-10 | Détail projet | 🟢 | proposal-flow | #136 |

### 3 · Freelance · opportunités & missions (5)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-11 | Tableau de bord prestataire | 🟢 | calibration-2 | #112 |
| W-12 | Opportunités (feed) | 🟢 | opportunites | #127 |
| W-13 | Détail opportunité + candidature | 🟢 | opportunites | #127 |
| W-14 | Mes candidatures | ⚪ | — | — |
| W-15 | Mission active (livrer jalon) | 🟢 | proposal-flow (covered by W-10 page) | #136 |

### 4 · Profil prestataire (2)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-16 | Profil public + privé | 🟢 | calibration-3 + v2 cards + width-fix | #114, #115, #124 |
| W-17 | (covered by W-16) | ⚫ | (same page) | — |

### 5 · Argent · portefeuille & facturation (3)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-18 | Portefeuille | 🟢 | wallet + responsive-fix | #121, #125 |
| W-19 | Factures | 🟢 | factures | #134 |
| W-20 | Profil de facturation | 🟢 | profil-facturation | #135 |

### 6 · Communication & équipe (2)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-21 | Messagerie | 🟢 | messagerie-widget | #131 |
| W-22 | Équipe & permissions | 🟢 | team-search | #141 |

### 7 · Compte & paramètres (1)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-23 | Compte (préférences) | 🟢 | w23-account + compte-fix | #119, #123 |

### 8 · Notifications (1)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-24 | Notifications | 🟢 | notifications | #129 |

---

## Phase 3 · Mobile batches

### 1 · Auth (2)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-01 | Connexion | 🟢 | m01-mobile-connexion | #116 |
| M-02 | Inscription · choix de rôle | 🟢 | m02-mobile-signup-role | #120 |

### 2 · Activité (4)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-03 | Dashboard freelance | 🟢 | mobile-dashboards | #138 |
| M-04 | Dashboard entreprise | 🟢 | mobile-dashboards | #138 |
| M-05 | Mes candidatures | ⚪ | — | — |
| M-06 | Détail mission | ⚫ | covered by proposal_detail_screen (mobile mirror of W-10/W-15) | #136 |

### 3 · Annonces entreprise (3)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-07 | Mes annonces | 🟢 | mes-annonces | #128 |
| M-08 | Détail annonce + candidatures | 🟢 | detail-annonce | #133 |
| M-09 | Créer une annonce | 🟢 | creation-annonce | #132 |

### 4 · Recherche & profil (2)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-12 | Recherche freelances | 🟢 | team-search | #141 |
| M-13 | Profil prestataire | 🟢 | covered by mobile/M-16-fix (PR mis-named, target was freelance_profile_screen which is M-13) | #117 |

### 5 · Argent (3)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-14 | Portefeuille | 🟢 | wallet-fix (mobile WalletHero + balance row) | #125 |
| M-15 | Factures | 🟢 | factures + mobile-invoicing (lives in `invoicing/`, not `invoice/` skeleton) | #134, #140 |
| M-16 | Détail paiement | ⚫ | inventory entry pointed at a non-existent screen — superseded by the M-13 polish that landed under the M-16 label. No standalone "détail paiement" screen exists in the repo. | — |

### 6 · Communication (3)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-17 | Conversation active | 🟢 | messagerie-widget + system-messages-fix | #131, #137 |
| M-18 | Liste conversations | 🟢 | messagerie-widget | #131 |
| M-19 | Notifications | 🟢 | notifications | #129 |

### 7 · Compte (1)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-20 | Mon compte | 🟢 | compte-fix (drawer + new account screen) | #123 |

---

## Aggregate

| Surface | Total | Done 🟢 | Skipped ⚫ | Remaining ⚪ |
|---------|-------|---------|------------|---------------|
| Web | 24 | 22 | 1 (W-17 = W-16) | 1 (W-14) |
| Mobile | 18 | 15 | 2 (M-06 covered, M-16 superseded) | 1 (M-05) |
| **Total** | **42** | **37** | **3** | **2** |

Note pratique : W-10 et W-15 partagent la même page (proposal-detail-view) → un seul PR #136 couvre les deux IDs côté web. W-16 et W-17 partagent la même feature (public/privé) → un seul PR #114/#115 couvre les deux. Côté mobile, M-06 est couvert par le proposal_detail_screen porté dans #136, et M-16 (détail paiement) n'existait pas comme écran standalone dans le repo — l'inventory pointait vers une référence qui n'a jamais été implémentée.

---

## Last 5 merged batches

| Date | PR | Wave | Screens |
|------|-----|------|---------|
| 2026-05-05 | #141 | team-search | W-22 + M-12 |
| 2026-05-05 | #140 | mobile-invoicing | M-15 (invoicing/) |
| 2026-05-05 | #139 | kyc-visual | W-05 KYC |
| 2026-05-05 | #138 | mobile-dashboards | M-03 + M-04 |
| 2026-05-05 | #137 | system-messages-fix | M-17 polish |

---

## Open questions for orchestrator

(All four open questions answered 2026-05-04. Nothing pending.)

### Resolved

- ✅ **W-12 Opportunités** — confirmed: only `/(public)/opportunities` exists in the repo. Inventory locked to that route.
- ✅ **W-15 Mission active** — confirmed: same page as W-10, role-aware variants (provider sees milestone-submit actions, client sees milestone-validate actions). One PR covers both IDs.
- ✅ **M-19 Notifications mobile** — confirmed: web has `/(app)/notifications/` route. Added W-24 to the inventory, coupled with M-19.
- ✅ **Phase 0 batch ownership** — orchestrator runs in main session (recommended by Claude, validated by Hassad). Touches `globals.css` + creates `Portrait` primitive — too sensitive to delegate.

---

## Mobile testing constraint (current session)

Hassad runs Linux (no Mac, no iOS Simulator). Mobile validation goes through:
- **Android emulator** (AVD) for development screenshots and golden tests
- **Android wireless debug** for on-device validation when connected

Code stays cross-platform (Flutter), so iOS support is preserved structurally — the constraint is purely on the screenshot/diff workflow. When Hassad gets a Mac, no refactor is needed; iOS Simulator captures will just be added to the diff folders.

**Implication for batch briefs**: mobile agents capture `before-android.png` / `after-android.png` (not `before-mobile.png`). The screen target stays 390×844 (Pixel 5 emulator matches this).
