# Design — Inventory (41 écrans uniques)

> Mapping exhaustif design Soleil v2 ↔ routes/fichiers du repo.
> **Orchestrator-only**: ne jamais déléguer ce fichier à un agent.
>
> Total: **23 écrans web** (desktop + responsive partagent le même `.tsx` via Tailwind breakpoints) + **18 écrans mobile Flutter** = 41 écrans uniques.

---

## Conventions

Chaque entrée :
- **ID** : `W-XX` (web) ou `M-XX` (mobile). Stable, ne change jamais.
- **Source design** : pointer vers le fichier JSX d'origine + lignes + page PDF.
- **Route existante** : path Next.js ou GoRouter. `—` si la route n'existe pas et doit être créée (à signaler à l'orchestrator).
- **Fichier principal** : `page.tsx` web ou `*_screen.dart` mobile. Le point d'entrée.
- **Components touchables** : whitelist explicite. Tout fichier hors de cette liste = OFF-LIMITS.
- **OFF-LIMITS** : hooks/api/schemas qui ne doivent JAMAIS être touchés par cet écran.
- **Features design absentes** : sections du design qui n'ont pas de backing repo — agent SKIP + FLAG.
- **Mobile parity** : ID couplé à dispatcher en parallèle si applicable.
- **Status** : voir `tracking.md` (vit là).

Les valeurs `Components touchables` et `OFF-LIMITS` peuvent être affinées au moment du brief batch — l'inventory pose un cadre, le brief le serre.

---

# WEB · 23 écrans (desktop 1440px + responsive 390px, mêmes fichiers)

## 1 · Auth & onboarding (5 écrans)

### W-01 · Connexion (login)

- **Source** : `phase1/soleil-lotE.jsx` `SoleilLogin` · desktop PDF p.3 · responsive PDF p.3
- **Route existante** : `/[locale]/(auth)/login`
- **Fichier principal** : `web/src/app/[locale]/(auth)/login/page.tsx`
- **Components touchables** : `web/src/features/auth/components/login-form.tsx`, autres `web/src/features/auth/components/login-*.tsx`, `web/src/app/[locale]/(auth)/layout.tsx`
- **OFF-LIMITS** : `web/src/features/auth/api/auth-api.ts`, `web/src/features/auth/hooks/use-login.ts`, `web/src/features/auth/schemas/login.schema.ts`
- **Features design absentes** : Login Apple/Google buttons (à vérifier — peut-être déjà câblés via OAuth)
- **Mobile parity** : `M-01`
- **Notes** : split 2 colonnes 50/50 en desktop, single column en mobile. Visuel rose corail + 3 portraits flottants à droite.

### W-02 · Inscription · choix de rôle

- **Source** : `phase1/soleil-lotE.jsx` `SoleilSignupRole` · desktop PDF p.4
- **Route existante** : `/[locale]/(auth)/register` (la page racine sans `/agency`/`/provider`/`/enterprise`)
- **Fichier principal** : `web/src/app/[locale]/(auth)/register/page.tsx`
- **Components touchables** : `web/src/features/auth/components/role-selection-*.tsx`, `register-stepper.tsx`
- **OFF-LIMITS** : `web/src/features/auth/api/*`, `web/src/features/auth/hooks/use-register-*.ts`, schemas
- **Features design absentes** : "Les deux" recommandé (combo prestataire+entreprise) — vérifier si supporté par backend register flow
- **Mobile parity** : `M-02`

### W-03 · Inscription · prestataire (provider)

- **Source** : `phase1/soleil-lotE.jsx` `SoleilSignupFreelance` · desktop PDF p.5
- **Route existante** : `/[locale]/(auth)/register/provider`
- **Fichier principal** : `web/src/app/[locale]/(auth)/register/provider/page.tsx`
- **Components touchables** : `web/src/features/auth/components/provider-register-*.tsx`, formulaire stepped 3 étapes
- **OFF-LIMITS** : api/, hooks/, schemas/ d'auth
- **Features design absentes** : à recroiser au brief
- **Mobile parity** : `M-02` (consolidé)

### W-04 · Inscription · entreprise

- **Source** : `phase1/soleil-lotE.jsx` `SoleilSignupCompany` · desktop PDF
- **Route existante** : `/[locale]/(auth)/register/enterprise`
- **Fichier principal** : `web/src/app/[locale]/(auth)/register/enterprise/page.tsx`
- **Components touchables** : `web/src/features/auth/components/enterprise-register-*.tsx`
- **OFF-LIMITS** : api/, hooks/, schemas/ d'auth
- **Mobile parity** : `M-02` (consolidé)

### W-05 · Stripe Connect (initial + urgent)

- **Source** : `phase1/soleil-lotE.jsx` `SoleilStripeConnect` · desktop PDF
- **Route existante** : à confirmer — probablement `/[locale]/(app)/payment-info` ou un sous-écran de `/profile`
- **Fichier principal** : `web/src/app/[locale]/(app)/payment-info/page.tsx`
- **Components touchables** : `web/src/features/payment-info/components/*.tsx` (composants Stripe Connect onboarding)
- **OFF-LIMITS** : api/Stripe (`web/src/features/payment-info/api/*`), hooks/, schemas/
- **Features design absentes** : "urgent state" (banner rouge si KYC bloque les paiements) — vérifier au brief
- **Mobile parity** : pas de M (mobile gère Stripe via WebView)

---

## 2 · Entreprise · annonces & projets (5 écrans)

### W-06 · Mes annonces (liste)

- **Source** : `phase1/soleil-lotA.jsx` `SoleilJobsList` · desktop PDF
- **Route existante** : `/[locale]/(app)/jobs`
- **Fichier principal** : `web/src/app/[locale]/(app)/jobs/page.tsx`
- **Components touchables** : `web/src/features/job/components/jobs-list.tsx`, `job-card.tsx`, `jobs-filter.tsx`
- **OFF-LIMITS** : `web/src/features/job/api/*`, `web/src/features/job/hooks/use-jobs.ts`, schemas
- **Features design absentes** : à confirmer (filtres avancés type "saved searches"?)
- **Mobile parity** : `M-07`

### W-07 · Détail annonce · description

- **Source** : `phase1/soleil-lotA.jsx` `SoleilJobDetailDesc`
- **Route existante** : `/[locale]/(app)/jobs/[id]` (tab description)
- **Fichier principal** : `web/src/app/[locale]/(app)/jobs/[id]/page.tsx`
- **Components touchables** : `web/src/features/job/components/job-detail-*.tsx`, `job-description-tab.tsx`
- **OFF-LIMITS** : api/, hooks/use-job.ts, schemas/
- **Mobile parity** : `M-08` (couplé avec W-08)

### W-08 · Détail annonce · candidatures

- **Source** : `phase1/soleil-lotA.jsx` `SoleilJobDetailCands`
- **Route existante** : `/[locale]/(app)/jobs/[id]` (tab candidatures)
- **Fichier principal** : `web/src/app/[locale]/(app)/jobs/[id]/page.tsx` (même page, tab différent)
- **Components touchables** : `web/src/features/job/components/job-candidates-tab.tsx`, `candidate-card.tsx`
- **OFF-LIMITS** : api/, hooks/use-candidates.ts, schemas/
- **Mobile parity** : `M-08`

### W-09 · Création d'une annonce

- **Source** : `phase1/soleil-lotA.jsx` `SoleilJobCreate` · stepper multi-étapes
- **Route existante** : `/[locale]/(app)/jobs/create`
- **Fichier principal** : `web/src/app/[locale]/(app)/jobs/create/page.tsx`
- **Components touchables** : `web/src/features/job/components/job-create-stepper.tsx`, `job-create-step-*.tsx`
- **OFF-LIMITS** : `web/src/features/job/api/create-job.ts`, hooks/use-create-job.ts, schemas
- **Features design absentes** : "+ Apporteur" (referral attribution lors de la création) — à vérifier
- **Mobile parity** : `M-09`

### W-10 · Détail projet (stepper + frais)

- **Source** : `phase1/soleil-lotA.jsx` `SoleilProjectDetail`
- **Route existante** : `/[locale]/(app)/projects/[id]`
- **Fichier principal** : `web/src/app/[locale]/(app)/projects/[id]/page.tsx`
- **Components touchables** : `web/src/features/proposal/components/proposal-detail-view.tsx`, `milestone-tracker.tsx`, `proposal-actions-panel.tsx`, `proposal-stepper.tsx`
- **OFF-LIMITS** : `web/src/features/proposal/api/*`, hooks/, schemas/
- **Features design absentes** : structure existante très proche, juste re-skin
- **Mobile parity** : `M-06`

---

## 3 · Freelance · opportunités & missions (5 écrans)

### W-11 · Tableau de bord prestataire

- **Source** : `phase1/soleil-lotC.jsx` `SoleilFreelancerDashboard` · soleil.jsx `SoleilDashboard` (référence générique)
- **Route existante** : `/[locale]/(app)/dashboard`
- **Fichier principal** : `web/src/app/[locale]/(app)/dashboard/page.tsx`
- **Components touchables** : `web/src/features/dashboard/components/*` (existe?), à confirmer + `web/src/shared/components/layouts/dashboard-shell.tsx`
- **OFF-LIMITS** : tous les hooks de stats
- **Features design absentes** : "Cette semaine chez Atelier" (card éditoriale blog) — SKIP, "Atelier Premium" CTA sidebar — SKIP, témoignage avec quote (Lemon Aviation) — décoratif OK
- **Mobile parity** : `M-03`

### W-12 · Opportunités (feed)

- **Source** : `phase1/soleil-lotC.jsx` `SoleilOpportunities` · format "Find" cards humaines
- **Route existante** : `/[locale]/(public)/opportunities` ou `/[locale]/(app)/opportunities`?
- **Fichier principal** : `web/src/app/[locale]/(public)/opportunities/page.tsx`
- **Components touchables** : `web/src/features/job/components/opportunities-list.tsx`, `opportunity-card.tsx`, `opportunities-filter.tsx`
- **OFF-LIMITS** : `web/src/features/job/api/opportunities.ts`, hooks/, schemas/
- **Mobile parity** : `M-13`

### W-13 · Détail opportunité + candidature

- **Source** : `phase1/soleil-lotC.jsx` `SoleilOpportunityDetail`
- **Route existante** : `/[locale]/(public)/opportunities/[id]`
- **Fichier principal** : `web/src/app/[locale]/(public)/opportunities/[id]/page.tsx`
- **Components touchables** : `web/src/features/job/components/opportunity-detail-*.tsx`, `apply-form.tsx`
- **OFF-LIMITS** : api/, hooks/use-apply.ts, schemas/
- **Mobile parity** : `M-13`

### W-14 · Mes candidatures

- **Source** : `phase1/soleil-lotC.jsx` `SoleilMyApplications`
- **Route existante** : `/[locale]/(app)/my-applications`
- **Fichier principal** : `web/src/app/[locale]/(app)/my-applications/page.tsx`
- **Components touchables** : `web/src/features/job/components/my-applications-list.tsx`, `application-status-badge.tsx`
- **OFF-LIMITS** : api/, hooks/use-my-applications.ts, schemas/
- **Mobile parity** : `M-05`

### W-15 · Mission active (livrer jalon)

- **Source** : `phase1/soleil-lotC.jsx` `SoleilFreelancerProject`
- **Route existante** : `/[locale]/(app)/projects/[id]` (vue côté provider, déjà couvert par W-10 mais avec layout adapté)
- **Fichier principal** : même que W-10 (`web/src/app/[locale]/(app)/projects/[id]/page.tsx`) — la vue provider/client diffère par les actions disponibles, pas par la page
- **Components touchables** : `web/src/features/proposal/components/milestone-submit-form.tsx`, `proposal-detail-view.tsx`
- **OFF-LIMITS** : api/, hooks/, schemas/
- **Mobile parity** : `M-06`

---

## 4 · Profil prestataire (2 écrans)

### W-16 · Profil public

- **Source** : `phase1/soleil-lotD.jsx` `SoleilProfile` (isPrivate=false) · soleil.jsx ligne 382 (ref)
- **Route existante** : `/[locale]/(public)/freelancers/[id]` (et `/agencies/[id]`, `/referrers/[id]`)
- **Fichier principal** : `web/src/app/[locale]/(public)/freelancers/[id]/page.tsx`
- **Components touchables** : `web/src/features/freelance-profile/components/profile-header.tsx`, `profile-tabs.tsx`, `profile-portfolio.tsx`, `profile-reviews.tsx`, `profile-stats-sidebar.tsx`
- **OFF-LIMITS** : `web/src/features/freelance-profile/api/*`, hooks/use-public-profile.ts, schemas/
- **Features design absentes** : "Citation" en italique (probablement `headline` du profile) — à vérifier qu'on a un champ
- **Mobile parity** : `M-12`

### W-17 · Profil privé (édition)

- **Source** : `phase1/soleil-lotD.jsx` `SoleilProfile` (isPrivate=true)
- **Route existante** : `/[locale]/(app)/profile`
- **Fichier principal** : `web/src/app/[locale]/(app)/profile/page.tsx`
- **Components touchables** : `web/src/features/freelance-profile/components/profile-edit-*.tsx`, sections d'édition
- **OFF-LIMITS** : api/, hooks/use-update-profile.ts, schemas/
- **Mobile parity** : `M-12`

---

## 5 · Argent · portefeuille & facturation (3 écrans)

### W-18 · Portefeuille

- **Source** : `phase1/soleil-lotB.jsx` `SoleilWallet`
- **Route existante** : `/[locale]/(app)/wallet`
- **Fichier principal** : `web/src/app/[locale]/(app)/wallet/page.tsx`
- **Components touchables** : `web/src/features/billing/components/wallet-summary.tsx`, `wallet-transactions.tsx`, `payout-action.tsx` (à vérifier les noms exacts)
- **OFF-LIMITS** : `web/src/features/billing/api/wallet.ts`, hooks/use-wallet.ts, schemas/
- **Mobile parity** : `M-14`

### W-19 · Factures

- **Source** : `phase1/soleil-lotB.jsx` `SoleilInvoices`
- **Route existante** : `/[locale]/(app)/invoices`
- **Fichier principal** : `web/src/app/[locale]/(app)/invoices/page.tsx`
- **Components touchables** : `web/src/features/invoicing/components/invoices-list.tsx`, `invoice-card.tsx`, `invoice-filters.tsx`
- **OFF-LIMITS** : `web/src/features/invoicing/api/*`, hooks/use-invoices.ts, schemas/
- **Mobile parity** : `M-15`

### W-20 · Profil de facturation

- **Source** : `phase1/soleil-lotB.jsx` `SoleilBillingProfile`
- **Route existante** : `/[locale]/(app)/billing` (à confirmer — peut-être `/account/billing-profile` ou similaire)
- **Fichier principal** : `web/src/app/[locale]/(app)/billing/page.tsx` ou sous-page
- **Components touchables** : `web/src/features/invoicing/components/billing-profile-form.tsx`
- **OFF-LIMITS** : api/, hooks/use-billing-profile.ts, schemas/
- **Mobile parity** : pas de M direct (consolidé dans M-14 ou M-15)

---

## 6 · Communication & équipe (2 écrans)

### W-21 · Messagerie

- **Source** : `phase1/soleil-lotF.jsx` `SoleilMessagerie` · soleil.jsx `SoleilMessages` (ref)
- **Route existante** : `/[locale]/(app)/messages`
- **Fichier principal** : `web/src/app/[locale]/(app)/messages/page.tsx`
- **Components touchables** : `web/src/features/messaging/components/conversation-list.tsx`, `chat-thread.tsx`, `message-bubble.tsx`, `proposal-card-in-chat.tsx`, `chat-input.tsx`
- **OFF-LIMITS** : `web/src/features/messaging/api/*`, hooks/use-messages.ts + use-conversations.ts, schemas, websocket integration
- **Features design absentes** : Phone/Video buttons dans header chat (LiveKit existe — on a déjà les calls, OK), "Démarrer un projet" CTA dans le header chat (relié au proposal flow — OK)
- **Mobile parity** : `M-16` (split list) + `M-17` (thread)

### W-22 · Équipe & permissions

- **Source** : `phase1/soleil-lotF.jsx` `SoleilTeam`
- **Route existante** : `/[locale]/(app)/team`
- **Fichier principal** : `web/src/app/[locale]/(app)/team/page.tsx`
- **Components touchables** : `web/src/features/team/components/team-list.tsx`, `member-row.tsx`, `invite-form.tsx`
- **OFF-LIMITS** : `web/src/features/team/api/*`, hooks/, schemas/
- **Mobile parity** : pas de M direct (mobile pas prévu pour la gestion d'équipe v1)

---

## 7 · Compte & paramètres (1 écran)

### W-23 · Compte (préférences)

- **Source** : `phase1/soleil-lotE.jsx` `SoleilAccount`
- **Route existante** : `/[locale]/(app)/account` (ou `/settings`?)
- **Fichier principal** : `web/src/app/[locale]/(app)/account/page.tsx` + `web/src/app/[locale]/(app)/settings/page.tsx`
- **Components touchables** : `web/src/features/account/components/*.tsx`, `delete-account-card.tsx`, sections paramètres
- **OFF-LIMITS** : api/, hooks/, schemas/ d'account
- **Mobile parity** : `M-18`

---

# MOBILE · 18 écrans (Flutter, iOS-first 390×844)

## 1 · Auth (2 écrans)

### M-01 · Connexion

- **Source** : `phase1/soleil-app-lot5.jsx` `AppLogin` · mobile PDF p.3 (left frame)
- **Route existante** : GoRouter `/login`
- **Fichier principal** : `mobile/lib/features/auth/presentation/screens/login_screen.dart`
- **Widgets touchables** : `mobile/lib/features/auth/presentation/widgets/*.dart` (form fields, OAuth buttons)
- **OFF-LIMITS** : `mobile/lib/features/auth/data/**`, `mobile/lib/features/auth/domain/**`, `mobile/lib/core/network/**`
- **Mobile parity** : couples avec `W-01`

### M-02 · Inscription · choix de rôle

- **Source** : `phase1/soleil-app-lot5.jsx` `AppSignupRole` · mobile PDF p.3 (right frame)
- **Route existante** : GoRouter `/register` (+ enterprise/agency/provider sub-routes)
- **Fichier principal** : `mobile/lib/features/auth/presentation/screens/role_selection_screen.dart` + `register_screen.dart` + `agency_register_screen.dart` + `enterprise_register_screen.dart`
- **Widgets touchables** : screens d'inscription + widgets de formulaires
- **OFF-LIMITS** : data/, domain/, network/
- **Notes** : les 3 routes web (`W-02`, `W-03`, `W-04`) sont consolidées en M-02 sur mobile

---

## 2 · Activité (dashboard) (4 écrans)

### M-03 · Dashboard freelance

- **Source** : `phase1/soleil-app-lot1.jsx` `AppDashboardFreelance`
- **Route existante** : GoRouter `/dashboard` (rendu différent selon role)
- **Fichier principal** : `mobile/lib/features/dashboard/presentation/screens/dashboard_screen.dart` (à confirmer — vérifier ls)
- **Widgets touchables** : `mobile/lib/features/dashboard/presentation/widgets/*.dart`
- **OFF-LIMITS** : data/, domain/, hooks vers API
- **Mobile parity** : `W-11`

### M-04 · Dashboard entreprise

- **Source** : `phase1/soleil-app-lot1.jsx` `AppDashboardEntreprise`
- **Route existante** : `/dashboard` (variante role=enterprise)
- **Fichier principal** : même fichier que M-03, layout différent selon role provider
- **Mobile parity** : pas de W direct (web a `W-11` pour freelance, le dashboard entreprise sur web est probablement `/dashboard` aussi mais variant)

### M-05 · Mes candidatures

- **Source** : `phase1/soleil-app-lot1.jsx` `AppCandidatures`
- **Route existante** : `/my-applications`
- **Fichier principal** : `mobile/lib/features/job/presentation/screens/my_applications_screen.dart`
- **Widgets touchables** : `mobile/lib/features/job/presentation/widgets/application_*.dart`
- **OFF-LIMITS** : data/, domain/
- **Mobile parity** : `W-14`

### M-06 · Détail mission (livrer jalon)

- **Source** : `phase1/soleil-app-lot1.jsx` `AppMissionDetail`
- **Route existante** : `/projects/:id` (GoRouter)
- **Fichier principal** : `mobile/lib/features/proposal/presentation/screens/proposal_detail_screen.dart`
- **Widgets touchables** : milestone widgets, proposal status panel
- **OFF-LIMITS** : data/, domain/, network/
- **Mobile parity** : `W-10` + `W-15`

---

## 3 · Annonces (entreprise) (3 écrans)

### M-07 · Mes annonces

- **Source** : `phase1/soleil-app-lot2.jsx` `AppAnnonces`
- **Route existante** : `/jobs`
- **Fichier principal** : `mobile/lib/features/job/presentation/screens/jobs_screen.dart`
- **Widgets touchables** : `mobile/lib/features/job/presentation/widgets/job_card.dart`, etc.
- **OFF-LIMITS** : data/, domain/
- **Mobile parity** : `W-06`

### M-08 · Détail annonce + candidatures

- **Source** : `phase1/soleil-app-lot2.jsx` `AppAnnonceDetail`
- **Route existante** : `/jobs/:id`
- **Fichier principal** : `mobile/lib/features/job/presentation/screens/job_detail_screen.dart` + `candidates_screen.dart` + `candidate_detail_screen.dart`
- **Widgets touchables** : job detail tabs, candidate widgets
- **Mobile parity** : `W-07` + `W-08`

### M-09 · Créer une annonce

- **Source** : `phase1/soleil-app-lot2.jsx` `AppAnnonceCreation`
- **Route existante** : `/jobs/create`
- **Fichier principal** : `mobile/lib/features/job/presentation/screens/create_job_screen.dart`
- **Widgets touchables** : stepper widgets
- **OFF-LIMITS** : data/, domain/
- **Mobile parity** : `W-09`

---

## 4 · Recherche & profil prestataire (2 écrans)

### M-12 · Recherche freelances (sidebar item) ← maps to "AppRecherche"

- **Source** : `phase1/soleil-app.jsx` `AppRecherche`
- **Route existante** : `/search`
- **Fichier principal** : `mobile/lib/features/search/presentation/screens/search_screen.dart`
- **Widgets touchables** : `mobile/lib/features/search/presentation/widgets/freelance_card.dart`, search filters
- **OFF-LIMITS** : `mobile/lib/features/search/data/**`, domain/, network/
- **Mobile parity** : pas de W direct (web utilise `/agencies`, `/freelancers`, `/referrers` séparément — voir si un écran search global existe)

### M-13 · Profil prestataire

- **Source** : `phase1/soleil-app.jsx` `AppProfil`
- **Route existante** : `/freelancer/:id` (ou `/profile/public/:id`)
- **Fichier principal** : `mobile/lib/features/freelance_profile/presentation/screens/freelance_public_profile_screen.dart` (public) + `freelance_profile_screen.dart` (privé)
- **Widgets touchables** : profile header, tabs, sections
- **OFF-LIMITS** : data/, domain/
- **Mobile parity** : `W-16` (public) + `W-17` (privé)

---

## 5 · Argent (3 écrans)

### M-14 · Portefeuille

- **Source** : `phase1/soleil-app-lot3.jsx` `AppWallet`
- **Route existante** : `/wallet`
- **Fichier principal** : `mobile/lib/features/wallet/presentation/screens/wallet_screen.dart`
- **Widgets touchables** : wallet widgets
- **OFF-LIMITS** : data/, domain/, network/
- **Mobile parity** : `W-18`

### M-15 · Factures

- **Source** : `phase1/soleil-app-lot3.jsx` `AppFactures`
- **Route existante** : à confirmer — `/invoices` GoRouter
- **Fichier principal** : `mobile/lib/features/invoicing/presentation/screens/*.dart` (chercher invoices_screen.dart si existe; sinon billing_profile_screen.dart est le seul)
- **Widgets touchables** : invoice list/card widgets
- **OFF-LIMITS** : data/, domain/
- **Mobile parity** : `W-19`

### M-16 · Détail paiement (fee breakdown)

- **Source** : `phase1/soleil-app-lot3.jsx` `AppPaiementDetail`
- **Route existante** : à vérifier — peut être un dialogue modal ou sous-écran
- **Fichier principal** : à confirmer
- **OFF-LIMITS** : data/, domain/
- **Mobile parity** : pas de W direct (sur web le détail est intégré dans la page proposal)

---

## 6 · Communication (3 écrans)

### M-17 · Conversation active

- **Source** : `phase1/soleil-app-lot4.jsx` `AppMessagerie`
- **Route existante** : `/messages/:conversationId`
- **Fichier principal** : `mobile/lib/features/messaging/presentation/screens/chat_screen.dart`
- **Widgets touchables** : chat bubble widgets, input field, attachment picker
- **OFF-LIMITS** : data/, domain/, network/, websocket
- **Mobile parity** : `W-21` (thread part)

### M-18 · Liste conversations

- **Source** : `phase1/soleil-app-lot4.jsx` `AppConversations`
- **Route existante** : `/messages`
- **Fichier principal** : `mobile/lib/features/messaging/presentation/screens/messaging_screen.dart`
- **Widgets touchables** : conversation row widgets
- **OFF-LIMITS** : data/, domain/
- **Mobile parity** : `W-21` (list part)

### M-19 · Notifications

- **Source** : `phase1/soleil-app-lot4.jsx` `AppNotifications`
- **Route existante** : `/notifications`
- **Fichier principal** : `mobile/lib/features/notification/presentation/screens/notification_screen.dart`
- **Widgets touchables** : notification row widgets
- **OFF-LIMITS** : data/, domain/, push notification handlers
- **Mobile parity** : pas de W direct (web a peut-être `/notifications` route — à vérifier)

---

## 7 · Compte (1 écran)

### M-20 · Mon compte

- **Source** : `phase1/soleil-app-lot5.jsx` `AppCompte`
- **Route existante** : `/account`
- **Fichier principal** : `mobile/lib/features/account/presentation/screens/*.dart` (chercher account_screen.dart si existe; sinon delete_account_screen.dart est le seul)
- **Widgets touchables** : settings sections, profile preview
- **OFF-LIMITS** : data/, domain/
- **Mobile parity** : `W-23`

---

# Mapping desktop ↔ responsive ↔ mobile

| Web (desktop+responsive) | Mobile |
|---|---|
| W-01 Connexion | M-01 |
| W-02 / W-03 / W-04 Inscription | M-02 (consolidé) |
| W-05 Stripe Connect | (pas de M, géré via WebView) |
| W-06 Mes annonces | M-07 |
| W-07 / W-08 Détail annonce | M-08 |
| W-09 Créer annonce | M-09 |
| W-10 / W-15 Détail projet | M-06 |
| W-11 Dashboard freelance | M-03 |
| (— pas de W) Dashboard entreprise | M-04 |
| W-12 Opportunités feed | (M-12 search?) |
| W-13 Détail opportunité | (cf. M-12/M-13) |
| W-14 Mes candidatures | M-05 |
| W-16 / W-17 Profil prestataire | M-13 |
| W-18 Portefeuille | M-14 |
| W-19 Factures | M-15 |
| W-20 Profil de facturation | (consolidé) |
| W-21 Messagerie | M-17 + M-18 |
| W-22 Équipe | (pas de M) |
| W-23 Compte | M-20 |

---

# Notes sur les features design absentes du repo

À recouper systématiquement au moment du brief batch (le brief liste explicitement les features à skip pour cet écran). Inventaire global :

- **« Cette semaine chez Atelier »** — card éditoriale sur le dashboard (W-11). Feature blog/content qui n'existe pas. SKIP.
- **« Atelier Premium »** — CTA bottom de la sidebar (toutes pages avec sidebar). Subscription tier. À vérifier — si on a `web/src/features/billing/...` qui couvre subscription, OK ; sinon SKIP le CTA.
- **« Saved searches »** — chips filtres "favoris" dans Find. SKIP si pas backed.
- **« Trio portraits flottants »** dans le hero login — purement décoratif, OK à ship (3 `<Portrait/>` rotated).
- **« 1 247 freelances vérifiés »** — text dynamique. Si search a un compteur, wirer dessus ; sinon SKIP le compteur.
- **« Top 5% »** badge dans la carte profil "Vérifié par Atelier" — feature gamification non-existante. SKIP.
- **« Taux de réembauche »** dans "En quelques chiffres" du profil — métrique potentiellement non calculée par le backend. Au brief: vérifier que le hook profile retourne bien ce champ.

---

# Status

Voir `tracking.md` pour le board live des écrans (status par ID).
