# Design — Tracking (live status board)

> Source-of-truth status per screen. Update after every batch merge.
> See `inventory.md` for the full mapping (ID ↔ source ↔ route ↔ files ↔ off-limits).

**Legend** :
- ⚪ `not-started` — no batch dispatched yet
- 🟡 `in-progress` — batch dispatched, agent working
- 🔵 `in-review` — PR open, awaiting orchestrator audit
- 🟢 `merged` — PR merged, screen marked done
- ⚫ `skipped` — explicitly skipped (feature absent from repo, etc.)
- 🔴 `blocked` — blocker found, see batch file

---

## Phase 0 · Foundation

| Item | Status | Notes / PR |
|------|--------|------------|
| Foundation scaffold (CLAUDE.md, design/, scripts) | 🟢 | PR #107 (merged) |
| `inventory.md` + `tracking.md` (this file) | 🟡 | Current branch `chore/design-inventory-and-tracking` |
| Phase 0 batch — tokens (web `globals.css` + admin `index.css` + mobile `soleil_theme.dart` + `Portrait` primitive web/mobile) | ⚪ | TBD |

---

## Phase 1 · Calibration (2-3 reference screens, manual review)

> Goal: ship 2-3 representative screens manually with the orchestrator (Hassad + main session) to lock the visual identity before going parallel.

**Proposed candidates** (reorderable by Hassad):

| ID | Screen | Why this one | Status |
|----|--------|--------------|--------|
| W-01 | Connexion | Simple layout, anchors auth flow, validates the editorial right-column pattern | ⚪ |
| W-11 | Dashboard freelance | Content-heavy, exercises sidebar + topbar + stat cards + editorial accent | ⚪ |
| W-16 | Profil prestataire (public) | Most editorial layout (cover, citation pleine page, portfolio gallery, sidebar-stats) — biggest unknown | ⚪ |

If Hassad prefers a smaller calibration set: W-01 + W-11 (2 screens) suffice. The profile (W-16) can be the first Phase 2 batch.

---

## Phase 2 · Web batches

> Each row = one screen unique to web (desktop + responsive share files). Status updates after batch merge.

### 1 · Auth & onboarding (5)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-01 | Connexion | ⚪ | — | — |
| W-02 | Inscription · choix de rôle | ⚪ | — | — |
| W-03 | Inscription · prestataire | ⚪ | — | — |
| W-04 | Inscription · entreprise | ⚪ | — | — |
| W-05 | Stripe Connect | ⚪ | — | — |

### 2 · Entreprise · annonces & projets (5)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-06 | Mes annonces (liste) | ⚪ | — | — |
| W-07 | Détail annonce · description | ⚪ | — | — |
| W-08 | Détail annonce · candidatures | ⚪ | — | — |
| W-09 | Création d'une annonce | ⚪ | — | — |
| W-10 | Détail projet | ⚪ | — | — |

### 3 · Freelance · opportunités & missions (5)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-11 | Tableau de bord prestataire | ⚪ | — | — |
| W-12 | Opportunités (feed) | ⚪ | — | — |
| W-13 | Détail opportunité + candidature | ⚪ | — | — |
| W-14 | Mes candidatures | ⚪ | — | — |
| W-15 | Mission active (livrer jalon) | ⚪ | — | — |

### 4 · Profil prestataire (2)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-16 | Profil public | ⚪ | — | — |
| W-17 | Profil privé (édition) | ⚪ | — | — |

### 5 · Argent · portefeuille & facturation (3)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-18 | Portefeuille | ⚪ | — | — |
| W-19 | Factures | ⚪ | — | — |
| W-20 | Profil de facturation | ⚪ | — | — |

### 6 · Communication & équipe (2)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-21 | Messagerie | ⚪ | — | — |
| W-22 | Équipe & permissions | ⚪ | — | — |

### 7 · Compte & paramètres (1)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| W-23 | Compte (préférences) | ⚪ | — | — |

---

## Phase 3 · Mobile batches

### 1 · Auth (2)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-01 | Connexion | ⚪ | — | — |
| M-02 | Inscription · choix de rôle | ⚪ | — | — |

### 2 · Activité (4)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-03 | Dashboard freelance | ⚪ | — | — |
| M-04 | Dashboard entreprise | ⚪ | — | — |
| M-05 | Mes candidatures | ⚪ | — | — |
| M-06 | Détail mission | ⚪ | — | — |

### 3 · Annonces entreprise (3)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-07 | Mes annonces | ⚪ | — | — |
| M-08 | Détail annonce + candidatures | ⚪ | — | — |
| M-09 | Créer une annonce | ⚪ | — | — |

### 4 · Recherche & profil (2)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-12 | Recherche freelances | ⚪ | — | — |
| M-13 | Profil prestataire | ⚪ | — | — |

### 5 · Argent (3)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-14 | Portefeuille | ⚪ | — | — |
| M-15 | Factures | ⚪ | — | — |
| M-16 | Détail paiement | ⚪ | — | — |

### 6 · Communication (3)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-17 | Conversation active | ⚪ | — | — |
| M-18 | Liste conversations | ⚪ | — | — |
| M-19 | Notifications | ⚪ | — | — |

### 7 · Compte (1)

| ID | Screen | Status | Batch | PR |
|----|--------|--------|-------|-----|
| M-20 | Mon compte | ⚪ | — | — |

---

## Aggregate

| Surface | Total | Done | In progress | Skipped | Remaining |
|---------|-------|------|-------------|---------|-----------|
| Web | 23 | 0 | 0 | 0 | 23 |
| Mobile | 18 | 0 | 0 | 0 | 18 |
| **Total** | **41** | **0** | **0** | **0** | **41** |

---

## Last 5 merged batches

(empty — chantier just started)

---

## Open questions for orchestrator

1. **W-12 Opportunités** : la route est-elle `/(public)/opportunities` (publique pour SEO) ou `/(app)/opportunities` (auth-gated) ? Le design suggère un feed personnalisé pour le freelance connecté → peut-être les deux : la liste publique pour SEO + le feed dans `/(app)/opportunities` ?
2. **W-15 Mission active** : on dirait que c'est juste une variante role-aware de `W-10 Détail projet` (même route, layout adapté selon que l'user est client ou provider). Confirmer pour ne pas faire double emploi.
3. **M-19 Notifications mobile** : web a-t-il une route `/notifications` ? Si oui, ajouter une entrée `W-24` à l'inventory. Sinon, mobile-only OK.
4. **Phase 0 batch** — qui pilote ? Solo agent (briefing strict, perimeter 3 fichiers + le `Portrait` primitive) ou orchestrator-only (Hassad+moi en main session) ? Recommandation : orchestrator-only car ça touche `globals.css` qui est OFF-LIMITS pour les agents par défaut. Override one-shot via `ALLOW_OVERRIDE` est possible mais cher en context-switching.
