# PLAN NUIT — Session autonome de tests exhaustifs

**Date**: 2026-03-30
**Objectif**: Blinder le projet de tests unitaires, E2E, et integration sur les 3 apps (backend Go, web Next.js, mobile Flutter)
**Mode**: Autonome — l'agent travaille toute la nuit sans interruption

---

## REGLES DE TRAVAIL — A RESPECTER IMPERATIVEMENT

### Regles generales
1. **JAMAIS casser le build**. Avant chaque commit, verifier : `cd backend && go build ./...`, `cd web && npx tsc --noEmit`
2. **Commits atomiques** : 1 commit par bloc de tests termine (ex: "test: add auth handler unit tests"). Convention : `test:` prefix
3. **Ne pas modifier le code source** sauf si un bug empeche les tests de compiler. Si un bug est trouve, le documenter dans `bugacorriger.md` et ecrire le test qui le prouve (test qui echoue = preuve du bug, le marquer `// TODO: fix - bug documented in bugacorriger.md`)
4. **Tests propres et maintenables** : noms descriptifs, table-driven quand possible, pas de logique complexe dans les tests
5. **Pas de tests triviaux** : ne pas tester des getters/setters simples ou des interfaces sans logique
6. **Chaque test doit prouver quelque chose** : un comportement metier, une validation, un edge case, une securite

### Organisation des fichiers de test — CRUCIALE
Les tests doivent etre organises proprement, dans les dossiers prevus par l'architecture :

**Backend Go** :
- Tests unitaires : a cote du fichier source (`*_test.go` dans le meme package)
- Ex: `internal/handler/auth_handler.go` → `internal/handler/auth_handler_test.go`
- Ex: `internal/adapter/postgres/user_repository.go` → `internal/adapter/postgres/user_repository_test.go`
- Ex: `internal/handler/middleware/auth.go` → `internal/handler/middleware/auth_test.go`
- Mocks de test : dans le meme fichier `*_test.go` (ou `mocks_test.go` si partagees dans le package)

**Web Next.js** :
- Tests unitaires : dans `__tests__/` a cote des fichiers testes
- Ex: `src/features/job/components/create-job-form.tsx` → `src/features/job/components/__tests__/create-job-form.test.tsx`
- Ex: `src/features/job/hooks/use-jobs.ts` → `src/features/job/hooks/__tests__/use-jobs.test.ts`
- Ex: `src/shared/hooks/use-user.ts` → `src/shared/hooks/__tests__/use-user.test.ts`
- Tests E2E Playwright : dans `web/e2e/`

**Mobile Flutter** :
- Tests unitaires : dans `test/` en miroir de `lib/`
- Ex: `lib/features/job/domain/entities/job_entity.dart` → `test/features/job/domain/entities/job_entity_test.dart`
- Ex: `lib/features/job/data/job_repository_impl.dart` → `test/features/job/data/job_repository_impl_test.dart`
- Ex: `lib/core/network/api_client.dart` → `test/core/network/api_client_test.dart`

### Regles de qualite des tests
- **Table-driven tests** (Go) : utiliser des slices de test cases avec noms descriptifs
- **describe/it blocks** (TS) : un `describe` par composant/hook, un `it` par comportement
- **group/test blocks** (Dart) : un `group` par entite/provider, un `test` par comportement
- **AAA pattern** : Arrange, Act, Assert — toujours
- **Mocks** : mocker les dependances externes (DB, Redis, HTTP, S3) — jamais mocker la logique testee
- **Edge cases obligatoires** : null/nil/undefined, empty strings, valeurs limites, erreurs reseau
- **Noms de tests** : decrire le comportement, pas l'implementation
  - BON : `"returns 403 when user does not own the resource"`
  - MAUVAIS : `"test handler"`

### Limites
- **Max 600 lignes par fichier de test** — splitter si necessaire
- **Max 50 lignes par fonction de test** — extraire des helpers si necessaire
- **Ne pas toucher** aux fichiers de migration SQL
- **Ne pas toucher** aux fichiers de configuration (.env, docker-compose, etc.)
- **Ne pas ajouter** de dependances — utiliser ce qui est deja dans go.mod/package.json/pubspec.yaml

### Workflow par bloc de tests
```
1. Lire le fichier source a tester
2. Identifier les fonctions/methodes publiques et leurs cas d'utilisation
3. Ecrire les tests (happy path + edge cases + erreurs)
4. Lancer les tests : go test ./... OU npm run test OU flutter test
5. Si un test echoue :
   a. Lire l'erreur attentivement
   b. Corriger le test (pas le code source)
   c. Si c'est un vrai bug dans le code source → documenter dans bugacorriger.md
   d. Max 3 tentatives par test qui echoue
6. Committer quand un bloc complet passe
```

---

## PHASE 1 — BACKEND GO : TESTS HANDLERS (priorite maximale, 0% couverture actuelle)

Les handlers sont la couche la plus exposee et la moins testee. Utiliser `httptest.NewRecorder` et `httptest.NewRequest`.

### 1.1 Auth Handler Tests
**Fichier a creer** : `backend/internal/handler/auth_handler_test.go`
**Source** : `backend/internal/handler/auth_handler.go`
**Tests a ecrire** :
- `POST /auth/register` : succes (201), email deja utilise (409), champs manquants (400), mot de passe trop court (400), role invalide (400)
- `POST /auth/login` : succes (200 + cookies), mauvais mot de passe (401), email inexistant (401), champs manquants (400)
- `POST /auth/refresh` : succes (200 + nouveaux tokens), refresh token invalide (401), refresh token expire (401)
- `POST /auth/logout` : succes (200 + cookies effaces), non authentifie (401)
- `POST /auth/forgot-password` : succes (200), email inexistant (200 meme reponse pour eviter enumeration)
- `POST /auth/reset-password` : succes (200), token invalide (400), token expire (400), mot de passe trop court (400)
- `GET /auth/me` : succes (200 + user data), non authentifie (401)
- **Pattern** : mocker le service auth dans le handler, tester HTTP status + body

### 1.2 Profile Handler Tests
**Fichier a creer** : `backend/internal/handler/profile_handler_test.go`
**Source** : `backend/internal/handler/profile_handler.go`
**Tests a ecrire** :
- `GET /profiles/me` : succes (200), non authentifie (401)
- `PUT /profiles/me` : succes (200), champs invalides (400), non authentifie (401)
- `GET /profiles/:id` : succes profil public (200), profil inexistant (404)
- `GET /profiles/search` : succes (200 + liste), filtres de role, filtres referrer

### 1.3 Job Handler Tests
**Fichier a creer** : `backend/internal/handler/job_handler_test.go`
**Source** : `backend/internal/handler/job_handler.go`
**Tests a ecrire** :
- `POST /jobs` : succes (201), champs manquants (400), budget negatif (400), non authentifie (401)
- `GET /jobs` : succes (200 + pagination), filtre par statut, filtre par createur
- `GET /jobs/:id` : succes (200), job inexistant (404)
- `PUT /jobs/:id/close` : succes (200), pas le proprietaire (403), deja ferme (400)

### 1.4 Proposal Handler Tests
**Fichier a creer** : `backend/internal/handler/proposal_handler_test.go`
**Source** : `backend/internal/handler/proposal_handler.go`
**Tests a ecrire** :
- `POST /proposals` : succes (201), montant invalide (400), champs manquants (400)
- `GET /proposals` : succes (200), filtre par statut, filtre par utilisateur
- `GET /proposals/:id` : succes (200), pas participant (403), inexistant (404)
- `PUT /proposals/:id/accept` : succes, pas le destinataire (403), mauvais statut (400)
- `PUT /proposals/:id/decline` : succes, pas le destinataire (403)
- `PUT /proposals/:id/withdraw` : succes, pas le createur (403)
- `PUT /proposals/:id/complete` : succes, pas dans le bon statut (400)

### 1.5 Messaging Handler Tests
**Fichier a creer** : `backend/internal/handler/messaging_handler_test.go`
**Source** : `backend/internal/handler/messaging_handler.go`
**Tests a ecrire** :
- `POST /conversations` : succes (201), conversation existante (200 retour existant), avec soi-meme (400)
- `GET /conversations` : succes (200 + pagination), non authentifie (401)
- `GET /conversations/:id` : succes (200), pas participant (403), inexistant (404)
- `POST /conversations/:id/messages` : succes (201), message vide (400), pas participant (403)
- `PUT /messages/:id/read` : succes (200)
- `GET /conversations/:id/messages` : succes (200 + pagination)
- `DELETE /messages/:id` : succes (200), pas l'auteur (403)

### 1.6 Review Handler Tests
**Fichier a creer** : `backend/internal/handler/review_handler_test.go`
**Source** : `backend/internal/handler/review_handler.go`
**Tests a ecrire** :
- `POST /reviews` : succes (201), doublon (409), note hors plage (400), pas eligible (403)
- `GET /reviews/user/:id` : succes (200 + liste)
- `GET /reviews/average/:id` : succes (200 + moyenne)
- `GET /reviews/eligibility` : succes (200 + boolean)

### 1.7 Notification Handler Tests
**Fichier a creer** : `backend/internal/handler/notification_handler_test.go`
**Source** : `backend/internal/handler/notification_handler.go`
**Tests a ecrire** :
- `GET /notifications` : succes (200 + pagination), non authentifie (401)
- `GET /notifications/unread-count` : succes (200 + nombre)
- `PUT /notifications/:id/read` : succes (200), pas le proprietaire (403)
- `PUT /notifications/preferences` : succes (200), payload invalide (400)
- `POST /notifications/devices` : succes (201), token invalide (400)

### 1.8 Call Handler Tests
**Fichier a creer** : `backend/internal/handler/call_handler_test.go`
**Source** : `backend/internal/handler/call_handler.go`
**Tests a ecrire** :
- `POST /calls/initiate` : succes (201), destinataire inexistant (404), deja en appel (409)
- `PUT /calls/:id/accept` : succes (200), pas le destinataire (403)
- `PUT /calls/:id/decline` : succes (200), pas le destinataire (403)
- `PUT /calls/:id/end` : succes (200), pas participant (403)

### 1.9 Payment Info Handler Tests
**Fichier a creer** : `backend/internal/handler/payment_info_handler_test.go`
**Source** : `backend/internal/handler/payment_info_handler.go`
**Tests a ecrire** :
- `POST /payment-info` : succes (201), IBAN invalide (400), champs manquants (400)
- `GET /payment-info` : succes (200), non configure (404)
- `PUT /payment-info` : succes (200), pas le proprietaire (403)

### 1.10 Wallet Handler Tests
**Fichier a creer** : `backend/internal/handler/wallet_handler_test.go`
**Source** : `backend/internal/handler/wallet_handler.go`
**Tests a ecrire** :
- `GET /wallet/balance` : succes (200), non authentifie (401)
- `POST /wallet/payout` : succes (200), solde insuffisant (400)

### 1.11 Social Link Handler Tests
**Fichier a creer** : `backend/internal/handler/social_link_handler_test.go`
**Source** : `backend/internal/handler/social_link_handler.go`
**Tests a ecrire** :
- `POST /social-links` : succes (201), URL invalide (400), plateforme invalide (400)
- `PUT /social-links/:id` : succes (200), pas le proprietaire (403)
- `DELETE /social-links/:id` : succes (200), pas le proprietaire (403)
- `GET /social-links/user/:id` : succes (200 + liste)

### 1.12 Upload Handler Tests
**Fichier a creer** : `backend/internal/handler/upload_handler_test.go`
**Source** : `backend/internal/handler/upload_handler.go`
**Tests a ecrire** :
- Upload photo : succes (200 + URL), fichier trop gros (400), MIME type invalide (400)
- Upload video : succes (200 + URL), fichier trop gros (400), MIME type invalide (400)
- Non authentifie (401)

### 1.13 Health Handler Tests
**Fichier a creer** : `backend/internal/handler/health_handler_test.go`
**Source** : `backend/internal/handler/health_handler.go`
**Tests a ecrire** :
- `GET /health` : 200 + `{"status": "ok"}`
- `GET /ready` : 200 si DB+Redis OK, 503 sinon

---

## PHASE 2 — BACKEND GO : TESTS MIDDLEWARE

### 2.1 Auth Middleware Tests
**Fichier a creer** : `backend/internal/handler/middleware/auth_test.go`
**Source** : `backend/internal/handler/middleware/auth.go`
**Tests a ecrire** :
- Token JWT valide → user_id et role dans le contexte
- Token JWT expire → 401
- Token JWT signature invalide → 401
- Pas de token → 401
- Session cookie valide → user_id et role dans le contexte
- Session cookie expire → fallback vers Bearer → 401 si pas de Bearer
- RequireRole("agency") avec role "agency" → passe
- RequireRole("agency") avec role "enterprise" → 403
- RequireRole("agency", "enterprise") avec role "enterprise" → passe

### 2.2 Rate Limit Middleware Tests
**Fichier a creer** : `backend/internal/handler/middleware/ratelimit_test.go`
**Source** : `backend/internal/handler/middleware/ratelimit.go`
**Tests a ecrire** :
- Sous la limite → passe (200)
- A la limite → passe (200)
- Au-dessus de la limite → bloque (429 + Retry-After header)
- IPs differentes → limites independantes
- Apres expiration → reinitialise

### 2.3 CORS Middleware Tests
**Fichier a creer** : `backend/internal/handler/middleware/cors_test.go`
**Source** : `backend/internal/handler/middleware/cors.go`
**Tests a ecrire** :
- Origin whitelistee → headers CORS presents
- Origin non whitelistee → pas de Access-Control-Allow-Origin
- Preflight OPTIONS → 200 avec les bons headers
- Pas de header Origin → pas de CORS

### 2.4 Recovery Middleware Tests
**Fichier a creer** : `backend/internal/handler/middleware/recovery_test.go`
**Source** : `backend/internal/handler/middleware/recovery.go`
**Tests a ecrire** :
- Handler qui panic → 500 au lieu de crash
- Handler normal → pas d'interception

### 2.5 Request ID Middleware Tests
**Fichier a creer** : `backend/internal/handler/middleware/requestid_test.go`
**Source** : `backend/internal/handler/middleware/requestid.go`
**Tests a ecrire** :
- Requete sans X-Request-ID → genere un UUID
- Requete avec X-Request-ID → utilise celui fourni
- X-Request-ID propage dans le contexte

---

## PHASE 3 — BACKEND GO : TESTS APP SERVICES MANQUANTS

### 3.1 Payment Stripe Service Tests (methodes non testees)
**Fichier a creer** : `backend/internal/app/payment/service_stripe_test.go`
**Source** : `backend/internal/app/payment/service_stripe.go`
**Tests a ecrire** :
- SetupConnectAccount : succes, compte existant (idempotent), erreur Stripe
- CreatePaymentIntent : succes, proposal pas dans le bon statut, montant invalide
- ConfirmPayment : succes, paiement deja confirme, erreur Stripe
- CreateTransfer : succes, solde insuffisant, compte non connecte
- GetAccountStatus : succes, compte inexistant
- ProcessWebhook : signature valide, signature invalide, event inconnu

### 3.2 Proposal Actions Service Tests (methodes non testees)
**Fichier a creer** : `backend/internal/app/proposal/service_actions_test.go`
**Source** : `backend/internal/app/proposal/service_actions.go`
**Tests a ecrire** :
- Accept : succes, pas le destinataire, mauvais statut, deja accepte
- Decline : succes, pas le destinataire, mauvais statut
- Withdraw : succes, pas le createur, mauvais statut
- MarkPaid : succes, pas en statut Accepted, montant incorrect
- RequestCompletion : succes, pas en statut Active
- Complete : succes, pas en statut CompletionRequested
- RejectCompletion : succes, pas en statut CompletionRequested

### 3.3 Proposal Create Service Tests
**Fichier a creer** : `backend/internal/app/proposal/service_create_test.go`
**Source** : `backend/internal/app/proposal/service_create.go`
**Tests a ecrire** :
- Create : succes, montant negatif, deadline dans le passe, conversation inexistante
- CreateVersion : succes, proposal parent inexistant, meme createur que parent

### 3.4 Messaging Sub-service Tests
**Fichier a creer** : `backend/internal/app/messaging/service_read_test.go`
**Source** : `backend/internal/app/messaging/service_read.go`
**Tests a ecrire** :
- MarkAsRead : succes, message inexistant, pas le destinataire
- MarkAllAsRead : succes, conversation inexistante
- GetReadReceipts : succes, conversation inexistante

**Fichier a creer** : `backend/internal/app/messaging/service_system_test.go`
**Source** : `backend/internal/app/messaging/service_system.go`
**Tests a ecrire** :
- SendSystemMessage : succes, type invalide, conversation inexistante

**Fichier a creer** : `backend/internal/app/messaging/service_upload_test.go`
**Source** : `backend/internal/app/messaging/service_upload.go`
**Tests a ecrire** :
- GetPresignedUploadURL : succes, type MIME non autorise, conversation inexistante

---

## PHASE 4 — WEB NEXT.JS : TESTS HOOKS MANQUANTS

### 4.1 Job Feature
**Fichier a creer** : `web/src/features/job/hooks/__tests__/use-jobs.test.ts`
**Source** : `web/src/features/job/hooks/use-jobs.ts`
**Tests** :
- useJobs : appel API au mount, retour des donnees, query key
- useCreateJob : mutation, invalidation cache, gestion erreur
- useCloseJob : mutation, invalidation cache

### 4.2 Notification Feature
**Fichier a creer** : `web/src/features/notification/hooks/__tests__/use-notifications.test.ts`
**Source** : `web/src/features/notification/hooks/use-notifications.ts`
**Tests** :
- useNotifications : appel API, pagination, retour donnees
- useUnreadNotificationCount : compteur, mise a jour apres markRead

**Fichier a creer** : `web/src/features/notification/hooks/__tests__/use-notification-actions.test.ts`
**Source** : `web/src/features/notification/hooks/use-notification-actions.ts`
**Tests** :
- markAsRead : mutation, invalidation compteur
- markAllAsRead : mutation, reset compteur

### 4.3 Review Feature
**Fichier a creer** : `web/src/features/review/hooks/__tests__/use-reviews.test.ts`
**Source** : `web/src/features/review/hooks/use-reviews.ts`
**Tests** :
- useReviews : appel API avec userId, retour reviews
- useCreateReview : mutation, invalidation cache
- useAverageRating : appel API, retour moyenne

### 4.4 Wallet Feature
**Fichier a creer** : `web/src/features/wallet/hooks/__tests__/use-wallet.test.ts`
**Source** : `web/src/features/wallet/hooks/use-wallet.ts`
**Tests** :
- useWallet : appel API, retour balance + transactions
- useRequestPayout : mutation, gestion erreur

### 4.5 Call Feature
**Fichier a creer** : `web/src/features/call/hooks/__tests__/use-call.test.ts`
**Source** : `web/src/features/call/hooks/use-call.ts`
**Tests** :
- useCall : state management (idle, ringing, connected, ended)
- initiateCall, acceptCall, declineCall, endCall mutations

### 4.6 Payment Info Feature
**Fichier a creer** : `web/src/features/payment-info/hooks/__tests__/use-payment-info.test.ts`
**Source** : `web/src/features/payment-info/hooks/use-payment-info.ts`
**Tests** :
- usePaymentInfo : appel API, retour donnees
- useSavePaymentInfo : mutation, validation, invalidation cache

---

## PHASE 5 — WEB NEXT.JS : TESTS COMPOSANTS MANQUANTS

### 5.1 Job Components
**Fichier a creer** : `web/src/features/job/components/__tests__/create-job-form.test.tsx`
**Tests** :
- Rendu du formulaire (titre, description, budget, type)
- Validation (titre vide, budget negatif, description trop courte)
- Soumission reussie → appel API + navigation
- Etat loading pendant la soumission
- Selecteur de type d'applicant

**Fichier a creer** : `web/src/features/job/components/__tests__/job-list.test.tsx`
**Tests** :
- Etat vide (aucun job)
- Rendu de la liste (titres, budgets, statuts)
- Badge de statut (open, closed)
- Bouton de creation

### 5.2 Notification Components
**Fichier a creer** : `web/src/features/notification/components/__tests__/notification-bell.test.tsx`
**Tests** :
- Badge avec nombre d'unreads
- Badge cache quand 0 unreads
- Click ouvre le dropdown
- Badge met a jour apres markRead

**Fichier a creer** : `web/src/features/notification/components/__tests__/notification-item.test.tsx`
**Tests** :
- Rendu titre et message
- Badge unread (point)
- Click appelle markAsRead
- Timestamp relatif ("il y a 5 min")

### 5.3 Review Components
**Fichier a creer** : `web/src/features/review/components/__tests__/star-rating.test.tsx`
**Tests** :
- Rendu du bon nombre d'etoiles
- Click met a jour la note
- Mode readonly (pas de click)
- Demi-etoiles pour moyennes

**Fichier a creer** : `web/src/features/review/components/__tests__/review-list.test.tsx`
**Tests** :
- Etat vide
- Rendu des reviews (auteur, note, commentaire, date)
- Video de review si presente

**Fichier a creer** : `web/src/features/review/components/__tests__/review-modal.test.tsx`
**Tests** :
- Ouverture/fermeture du modal
- Formulaire (etoiles, commentaire, upload video)
- Validation (note obligatoire)
- Soumission → appel API

### 5.4 Auth Components (manquants)
**Fichier a creer** : `web/src/features/auth/components/__tests__/register-form.test.tsx`
**Tests** :
- Rendu des champs de base (email, password, nom)
- Validation (email format, password force)
- Navigation vers login
- Soumission reussie

**Fichier a creer** : `web/src/features/auth/components/__tests__/forgot-password-form.test.tsx`
**Tests** :
- Rendu du champ email
- Soumission → message de confirmation
- Email invalide → erreur

**Fichier a creer** : `web/src/features/auth/components/__tests__/reset-password-form.test.tsx`
**Tests** :
- Rendu des champs (nouveau mot de passe, confirmation)
- Mots de passe non identiques → erreur
- Token invalide → erreur

### 5.5 Wallet Component
**Fichier a creer** : `web/src/features/wallet/components/__tests__/wallet-page.test.tsx`
**Tests** :
- Affichage du solde
- Liste des transactions
- Bouton demande de virement
- Etat vide (pas de transactions)

### 5.6 Payment Info Components
**Fichier a creer** : `web/src/features/payment-info/components/__tests__/payment-info-page.test.tsx`
**Tests** :
- Rendu du formulaire (IBAN ou account number)
- Toggle business/individual
- Validation IBAN
- Soumission reussie

---

## PHASE 6 — WEB NEXT.JS : TESTS SHARED HOOKS MANQUANTS

### 6.1 use-user Hook
**Fichier a creer** : `web/src/shared/hooks/__tests__/use-user.test.ts`
**Tests** :
- useUser retourne les donnees utilisateur
- useUser retourne null si non authentifie
- logout vide le cache TanStack Query
- Refetch apres login

### 6.2 use-global-ws Hook
**Fichier a creer** : `web/src/shared/hooks/__tests__/use-global-ws.test.ts`
**Tests** :
- Connexion WebSocket au mount
- Deconnexion au unmount
- Reconnexion apres deconnexion
- Pas de connexion si non authentifie

### 6.3 use-unread-count Hook
**Fichier a creer** : `web/src/shared/hooks/__tests__/use-unread-count.test.ts`
**Tests** :
- Retourne le compteur de messages non lus
- Mise a jour apres reception WS

### 6.4 use-media-query Hook
**Fichier a creer** : `web/src/shared/hooks/__tests__/use-media-query.test.ts`
**Tests** :
- Retourne true si la media query match
- Retourne false sinon
- Met a jour sur resize

### 6.5 use-call-context
**Fichier a creer** : `web/src/shared/hooks/__tests__/use-call-context.test.tsx`
**Tests** :
- Etat initial (pas d'appel)
- Mise a jour de l'etat d'appel
- Cleanup au unmount

---

## PHASE 7 — WEB PLAYWRIGHT : TESTS E2E MANQUANTS

### 7.1 Message Sending E2E
**Fichier a creer** : `web/e2e/message-send.spec.ts`
**Tests** :
- Login → ouvrir conversation → ecrire message → envoyer → message affiche
- Upload fichier dans le chat
- Indicateur de typing visible

### 7.2 Proposal Workflow E2E
**Fichier a creer** : `web/e2e/proposal-workflow.spec.ts`
**Tests** :
- Creer une proposal depuis une conversation
- Voir la proposal dans la liste projects
- Accepter une proposal
- Naviguer vers le paiement

### 7.3 Job Lifecycle E2E
**Fichier a creer** : `web/e2e/job-lifecycle.spec.ts`
**Tests** :
- Creer un job (remplir le formulaire)
- Voir le job dans la liste
- Fermer le job

### 7.4 Profile Edit E2E
**Fichier a creer** : `web/e2e/profile-edit.spec.ts`
**Tests** :
- Modifier le bio
- Upload photo
- Ajouter/supprimer social links
- Voir le profil public

### 7.5 Notifications E2E
**Fichier a creer** : `web/e2e/notifications.spec.ts`
**Tests** :
- Badge de notification visible
- Ouvrir le dropdown
- Marquer comme lu
- Naviguer vers la source de la notification

### 7.6 Payment Info E2E
**Fichier a creer** : `web/e2e/payment-info.spec.ts`
**Tests** :
- Naviguer vers la page payment info
- Remplir le formulaire
- Sauvegarder
- Verifier les donnees affichees

### 7.7 Wallet E2E
**Fichier a creer** : `web/e2e/wallet.spec.ts`
**Tests** :
- Naviguer vers la page wallet
- Voir le solde
- Voir l'historique des transactions

---

## PHASE 8 — MOBILE FLUTTER : TESTS ENTITES MANQUANTES

### 8.1 Job Entity
**Fichier a creer** : `mobile/test/features/job/domain/entities/job_entity_test.dart`
**Tests** :
- Construction avec champs obligatoires
- fromJson avec donnees valides
- toJson roundtrip
- Champs optionnels (skills, deadline)
- Budget en centimes → euros conversion

### 8.2 Review Entity
**Fichier a creer** : `mobile/test/features/review/domain/entities/review_entity_test.dart`
**Tests** :
- Construction avec note et commentaire
- fromJson/toJson roundtrip
- Note hors plage
- Champs optionnels (video_url, criteria)

### 8.3 Notification Entity
**Fichier a creer** : `mobile/test/features/notification/domain/entities/notification_entity_test.dart`
**Tests** :
- Construction avec type et contenu
- fromJson/toJson roundtrip
- isRead toggle
- Types de notification (message, proposal, review, call)

### 8.4 Call Entity
**Fichier a creer** : `mobile/test/features/call/domain/entities/call_entity_test.dart`
**Tests** :
- Construction avec participants
- fromJson/toJson roundtrip
- Status transitions (ringing, connected, ended, missed)
- Duree calculee

### 8.5 Payment Info Entity
**Fichier a creer** : `mobile/test/features/payment_info/domain/entities/payment_info_entity_test.dart`
**Tests** :
- Construction IBAN vs account_number
- fromJson/toJson roundtrip
- Validation IBAN format
- Toggle business/individual

### 8.6 Invoice Entity
**Fichier a creer** : `mobile/test/features/invoice/domain/entities/invoice_entity_test.dart`
**Tests** :
- Construction avec montant et statut
- fromJson/toJson roundtrip
- Status (draft, sent, paid, overdue)

### 8.7 Mission Entity
**Fichier a creer** : `mobile/test/features/mission/domain/entities/mission_entity_test.dart`
**Tests** :
- Construction avec titre et description
- fromJson/toJson roundtrip
- Champs optionnels

### 8.8 User Entity (si non teste)
**Fichier a creer** : `mobile/test/features/auth/domain/entities/user_test.dart`
**Tests** :
- Construction avec role
- fromJson/toJson roundtrip
- Champs referrer_enabled

---

## PHASE 9 — MOBILE FLUTTER : TESTS DATA LAYER

### 9.1 Auth Repository Implementation
**Fichier a creer** : `mobile/test/features/auth/data/auth_repository_impl_test.dart`
**Tests** :
- login : succes (retourne user + tokens), erreur 401 (credentials invalides), erreur reseau
- register : succes, erreur 409 (email existe), erreur validation
- refreshToken : succes (nouveaux tokens), erreur 401 (token expire)
- logout : succes (clear storage)
- getCurrentUser : succes (retourne cache), pas de cache (retourne null)

### 9.2 Messaging Repository Implementation
**Fichier a creer** : `mobile/test/features/messaging/data/messaging_repository_impl_test.dart`
**Tests** :
- getConversations : succes (liste), vide, erreur reseau
- getMessages : succes (liste paginee), conversation inexistante
- sendMessage : succes, erreur rate limit
- markAsRead : succes
- getPresignedUrl : succes, type non autorise

### 9.3 Job Repository Implementation
**Fichier a creer** : `mobile/test/features/job/data/job_repository_impl_test.dart`
**Tests** :
- getJobs : succes (liste), vide, erreur reseau
- getJob : succes, 404
- createJob : succes, erreur validation
- closeJob : succes, pas le proprietaire

### 9.4 Proposal Repository Implementation
**Fichier a creer** : `mobile/test/features/proposal/data/proposal_repository_impl_test.dart`
**Tests** :
- getProposals : succes (liste), vide
- getProposal : succes, 404
- createProposal : succes, erreur validation
- acceptProposal : succes, erreur statut
- declineProposal : succes

### 9.5 Review Repository Implementation
**Fichier a creer** : `mobile/test/features/review/data/review_repository_impl_test.dart`
**Tests** :
- getReviews : succes (par userId), vide
- createReview : succes, doublon, erreur validation
- getAverageRating : succes

### 9.6 Notification Repository Implementation
**Fichier a creer** : `mobile/test/features/notification/data/notification_repository_impl_test.dart`
**Tests** :
- getNotifications : succes (paginee), vide
- markAsRead : succes
- getUnreadCount : succes

---

## PHASE 10 — MOBILE FLUTTER : TESTS PROVIDERS

### 10.1 Messaging Providers
**Fichier a creer** : `mobile/test/features/messaging/presentation/providers/conversations_provider_test.dart`
**Tests** :
- Charge les conversations au mount
- Rafraichit apres envoi de message
- Met a jour le compteur unread

**Fichier a creer** : `mobile/test/features/messaging/presentation/providers/messages_provider_test.dart`
**Tests** :
- Charge les messages d'une conversation
- Pagination (load more)
- Optimistic update apres envoi

### 10.2 Job Provider
**Fichier a creer** : `mobile/test/features/job/presentation/providers/job_provider_test.dart`
**Tests** :
- Charge la liste des jobs
- Filtre par statut
- Cree un job

### 10.3 Proposal Provider
**Fichier a creer** : `mobile/test/features/proposal/presentation/providers/proposal_provider_test.dart`
**Tests** :
- Charge la liste des proposals
- Accepter/decliner une proposal
- Filtre par statut

### 10.4 Notification Provider
**Fichier a creer** : `mobile/test/features/notification/presentation/providers/notification_provider_test.dart`
**Tests** :
- Charge les notifications
- Mark as read
- Compteur unread

### 10.5 Search Provider
**Fichier a creer** : `mobile/test/features/search/presentation/providers/search_provider_test.dart`
**Tests** :
- Recherche par query
- Filtre par role
- Pagination

### 10.6 Profile Provider
**Fichier a creer** : `mobile/test/features/profile/presentation/providers/profile_provider_test.dart`
**Tests** :
- Charge le profil
- Met a jour le profil
- Upload photo/video

---

## PHASE 11 — MOBILE FLUTTER : TESTS WIDGETS

### 11.1 Messaging Widgets
**Fichier a creer** : `mobile/test/features/messaging/presentation/widgets/message_bubble_test.dart`
**Tests** :
- Rendu message texte (propre vs autre)
- Message supprime (affiche "message supprime")
- Message edite (affiche "modifie")
- Timestamp

**Fichier a creer** : `mobile/test/features/messaging/presentation/widgets/message_input_bar_test.dart`
**Tests** :
- Rendu du champ texte
- Bouton envoyer (enabled/disabled)
- Bouton fichier
- Bouton vocal

**Fichier a creer** : `mobile/test/features/messaging/presentation/widgets/typing_indicator_widget_test.dart`
**Tests** :
- Visible quand quelqu'un tape
- Cache sinon
- Animation des points

### 11.2 Job Widgets
**Fichier a creer** : `mobile/test/features/job/presentation/widgets/budget_section_test.dart`
**Tests** :
- Affichage budget formaté
- Type one_shot vs long_term
- Frequence de paiement

**Fichier a creer** : `mobile/test/features/job/presentation/widgets/applicant_type_selector_test.dart`
**Tests** :
- Selection freelancer/agency/both
- Highlight de la selection active

### 11.3 Notification Widgets
**Fichier a creer** : `mobile/test/features/notification/presentation/widgets/notification_badge_test.dart`
**Tests** :
- Badge avec nombre
- Badge cache quand 0
- Badge rouge

**Fichier a creer** : `mobile/test/features/notification/presentation/widgets/notification_tile_test.dart`
**Tests** :
- Rendu titre et message
- Indicateur non lu
- Tap navigation

### 11.4 Shared Widgets
**Fichier a creer** : `mobile/test/shared/widgets/app_drawer_test.dart`
**Tests** :
- Rendu des items de menu
- Navigation au tap
- Highlight du menu actif

**Fichier a creer** : `mobile/test/shared/widgets/upload_bottom_sheet_test.dart`
**Tests** :
- Options photo/fichier
- Tap ouvre le picker

**Fichier a creer** : `mobile/test/shared/widgets/video_player_widget_test.dart`
**Tests** :
- Rendu avec URL
- Placeholder sans URL
- Controls visibles

---

## PHASE 12 — MOBILE FLUTTER : TESTS CORE

### 12.1 API Client
**Fichier a creer** : `mobile/test/core/network/api_client_test.dart`
**Tests** :
- GET request avec auth header
- POST request avec body JSON
- Token refresh sur 401 (interceptor)
- Retry apres refresh reussi
- Erreur si refresh echoue (redirect login)
- Timeout handling
- Error mapping (ApiException)

### 12.2 Secure Storage
**Fichier a creer** : `mobile/test/core/storage/secure_storage_test.dart`
**Tests** :
- Sauvegarder et lire un token
- Supprimer un token
- Lire un token inexistant → null
- Sauvegarder et lire un user cache

### 12.3 Router (app_router)
**Fichier a creer** : `mobile/test/core/router/app_router_test.dart`
**Tests** :
- Route non authentifie → redirect /login
- Route authentifie → passe
- Deep link /chat/:id → ChatScreen avec le bon ID
- Route inconnue → 404

### 12.4 MIME Type Helper
**Fichier a creer** : `mobile/test/core/utils/mime_type_helper_test.dart`
**Tests** :
- .jpg → image/jpeg
- .png → image/png
- .pdf → application/pdf
- .mp4 → video/mp4
- Extension inconnue → application/octet-stream

---

## PHASE 13 — BOUCLE D'AUTO-INCREMENT

**INSTRUCTION CRITIQUE** : Apres avoir termine toutes les phases ci-dessus, l'agent doit entrer dans une boucle d'auto-increment :

```
BOUCLE:
1. Analyser le projet entierement (lire les fichiers source, les tests existants)
2. Identifier les fichiers source qui n'ont TOUJOURS PAS de tests
3. Rediger un plan incremental de nouveaux tests (ajouter une PHASE 14, 15, 16... dans ce fichier)
4. Implementer ces tests
5. Lancer les tests (go test, npm test, flutter test)
6. Committer les tests qui passent
7. Retour a l'etape 1

Continuer cette boucle jusqu'a ce qu'il n'y ait PLUS RIEN a tester.
Critere d'arret : couverture > 80% sur domain + app + handlers (backend), > 70% sur features (web), > 60% sur domain + data + providers (mobile).
```

**Nouveaux tests a considerer dans la boucle** :
- Tests des DTOs request/response (validation, serialization)
- Tests de l'adapter S3 (mock du SDK AWS)
- Tests de l'adapter Resend email (mock HTTP)
- Tests de l'adapter LiveKit (mock SDK)
- Tests du WebSocket connection.go
- Tests des composants web call (call-overlay, video-renderer, etc.)
- Tests des ecrans mobile (register, chat, jobs, proposals)
- Tests end-to-end mobile (integration_test/)
- Tout fichier source non couvert que l'analyse revele

---

## PHASE FINALE — CLOTURE

Quand il n'y a VRAIMENT plus rien a tester (toutes les phases + boucle auto-increment epuisee) :

### Etape 1 : Documenter
- Ecrire l'heure de fin dans ce fichier : `FIN: [heure]`
- Lister le nombre total de tests ecrits cette nuit
- Lister la couverture finale estimee par app

### Etape 2 : Commit final
```bash
cd /home/hassad/Documents/marketplaceServiceGo
git add -A
git commit -m "test: comprehensive test suite — overnight session 2026-03-30

Added unit tests, E2E tests, and integration tests across all apps:
- Backend Go: handler tests, middleware tests, app service tests
- Web Next.js: hook tests, component tests, Playwright E2E
- Mobile Flutter: entity tests, repository tests, provider tests, widget tests

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

### Etape 3 : Push sur GitHub
```bash
git push origin main
```

### Etape 4 : Fermer VS Code
```bash
# Fermer VS Code proprement
pkill -f "code" || true
sleep 2
```

### Etape 5 : Eteindre l'ordinateur
Note : sudo est desactive cette nuit, donc utiliser :
```bash
# Attendre 10 secondes pour que tout se flush
sleep 10
# Eteindre proprement l'ordinateur (pas besoin de sudo)
systemctl poweroff
```

---

## STATISTIQUES A REMPLIR PAR L'AGENT

- **Debut** : 2026-03-30 ~23:00
- **FIN** : [a remplir par l'agent]
- **Tests ecrits cette nuit** : [nombre]
- **Fichiers de test crees** : [nombre]
- **Commits** : [nombre]
- **Couverture backend** : [%]
- **Couverture web** : [%]
- **Couverture mobile** : [%]
- **Bugs trouves** : [nombre] (documentes dans bugacorriger.md)
