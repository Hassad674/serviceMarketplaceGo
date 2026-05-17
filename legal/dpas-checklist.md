# Sous-processeurs — Checklist DPA (Art. 28 RGPD)

> **Statut du document** : version initiale auto-générée à partir du code (`.env.example`, `go.mod`, `package.json`, `pubspec.yaml`) le **2026-05-11**.
> **Responsable de traitement** : Hassad SMARA (Marketplace Service).
> **Contact RGPD** : `hassadsmara@designedtrust.com` (à mettre à jour avec une adresse `dpo@…` ou `privacy@…` une fois le domaine arrêté — variable `GDPR_CONTACT_EMAIL`).
> **Référence transverse** : audit RGPD interne (transferts extra-UE / Schrems II) + planning RGPD interne.

L'app traite des données personnelles à travers les sous-processeurs listés ci-dessous. Chaque ligne correspond à un fournisseur **réellement intégré** (présence vérifiée dans `backend/go.mod`, `web/package.json`, `mobile/pubspec.yaml`, `backend/internal/config/config.go` ou `backend/.env.example`). Les fournisseurs absents du code ne sont pas listés — la liste évolue avec le code.

Pour chaque sous-processeur :
1. Le contrôleur (toi) doit signer un DPA (Data Processing Agreement) au sens de l'Art. 28 RGPD **avant la mise en production réelle** auprès d'utilisateurs européens.
2. Pour tout transfert hors UE, vérifier le mécanisme (Data Privacy Framework — DPF, Standard Contractual Clauses — SCC, ou décision d'adéquation).
3. Archiver le PDF signé dans `legal/signed-dpas/<vendor>-YYYY.pdf` et mettre à jour la colonne **Statut**.

---

## A. Tableau exhaustif des sous-processeurs

| # | Service | Catégorie | Données traitées | Localisation | Transfert hors UE ? | URL DPA / Demande | Statut | Notes |
|---|---|---|---|---|---|---|---|---|
| 1 | **Vercel** | Hébergement web (Next.js) | IP, user-agent, cookies session, contenu des requêtes HTTP | US (edge global) | Oui — DPF + SCC fallback | <https://vercel.com/legal/dpa> | À demander | DPA auto-signé via le dashboard Team (Settings → Security → Data Processing Addendum). Choisir région Frankfurt (`fra1`) pour les fonctions serverless quand possible. |
| 2 | **Railway** | Hébergement backend (Go) | Toutes les données passant par l'API (profils, missions, contrats, messages, paiements, audit logs) | US par défaut, EU disponible (region `eu-west1`) | Oui — SCC | <https://railway.com/legal/dpa> ou contact `legal@railway.app` | À demander | **À COMPLÉTER** : vérifier que le projet prod est sur `eu-west1` (région Frankfurt). Demander le DPA via support ticket si pas signable en self-service. |
| 3 | **Neon** | Base de données PostgreSQL | Intégralité des données applicatives (`DATABASE_URL`) | EU (eu-central-1 Frankfurt) si configuré | Non si EU region, sinon SCC | <https://neon.com/legal/dpa> | À demander | **À COMPLÉTER** : confirmer que le projet Neon prod est en `aws-eu-central-1`. DPA signable depuis Console → Settings → Legal. |
| 4 | **Cloudflare R2** | Object storage (S3-compatible) | Avatars, justificatifs KYC, pièces jointes messages, exports RGPD, archives audit logs (`STORAGE_*`, `STORAGE_AUDIT_COLD_BUCKET`) | Global (replication par région) | Oui — DPF + SCC | <https://www.cloudflare.com/cloudflare-customer-dpa/> | À demander | DPA Cloudflare unifié couvre R2 + CDN + DNS. Activable en self-service. Configurer `Jurisdiction: EU` sur le bucket R2 pour épingler la localisation. |
| 5 | **Cloudflare CDN/DNS** | CDN + DNS frontal | IP visiteurs, user-agent, headers HTTP | Global (anycast) | Oui — DPF + SCC | <https://www.cloudflare.com/cloudflare-customer-dpa/> | À demander | **Couvert par le même DPA que R2** (point 4). Mention séparée car le rôle de sous-traitant pour le CDN/DNS est distinct du stockage. |
| 6 | **Stripe (Payments + Connect)** | Paiement, KYC vendeur, virements | Nom, email, IBAN, pièce d'identité, transactions, métadonnées Connect, identifiants entreprise | US + Irlande (Stripe Payments Europe Ltd pour les sujets UE) | Oui — DPF + SCC + entité Stripe Ireland | <https://stripe.com/legal/dpa> | À demander | DPA accepté électroniquement depuis Dashboard → Settings → Compliance → Data Processing Agreement. Activer "Stripe Payments Europe Ltd" comme contrepartie pour les comptes UE. |
| 7 | **Resend** | Email transactionnel | Email destinataire, contenu (templates auth, notifications, factures) | US (AWS us-east-1) | Oui — DPF (à vérifier) + SCC | <https://resend.com/legal/dpa> | À demander | **À COMPLÉTER** : confirmer que Resend est inscrit au DPF actif. Si non : exiger SCC explicites. DPA téléchargeable depuis Dashboard → Settings → Legal. |
| 8 | **OpenAI** | Modération texte + embeddings de recherche | Texte saisi par l'utilisateur (descriptions missions, messages soumis à modération) + tokens d'embedding | US | Oui — DPF + SCC | <https://openai.com/policies/data-processing-addendum/> | À demander | DPA signable via le portail (Settings → Organization → Data Controls → Data Processing Addendum). **Activer** "Do not train on my data" (déjà activé pour les comptes API). Réviser périodiquement. |
| 9 | **Anthropic (Claude)** | AI analyzer (résumé de litiges, chat dispute, Haiku 4.5) | Messages bruts utilisateurs envoyés au modèle (`backend/internal/adapter/anthropic/analyzer.go`) | US | Oui — DPF | <https://www.anthropic.com/legal/commercial-terms> (DPA en addendum) | À demander | DPA disponible via console.anthropic.com → Organization → Data Processing Agreement. **Vérifier** que "Do not train on my data" est ON (toggle par défaut pour les comptes commerciaux). |
| 10 | **AWS Rekognition** | Modération image/vidéo | Images et vidéos uploadées par les utilisateurs (avatars, médias contrats) | Configurable — `REKOGNITION_REGION` défaut `eu-west-1` (Irlande) | Non si EU | <https://aws.amazon.com/service-terms/> (DPA inclus dans le Customer Agreement) | À demander | DPA AWS unifié couvre Rekognition + S3 + SQS + SNS. Téléchargeable via AWS Artifact (console.aws.amazon.com/artifact). **Vérifier** que toutes les ressources prod sont en `eu-west-1` ou `eu-central-1`. |
| 11 | **AWS S3 (Rekognition transit)** | Stockage temporaire pour la modération vidéo | Vidéos à analyser (passage transitoire avant suppression) | `eu-west-1` (configurable) | Non si EU | <https://aws.amazon.com/service-terms/> | À demander | Couvert par le même DPA AWS (point 10). Configurer lifecycle policy: suppression auto < 7 jours. |
| 12 | **AWS SNS** | Notifications de fin d'analyse Rekognition | Identifiant de média, statut analyse | `eu-west-1` (configurable) | Non si EU | <https://aws.amazon.com/service-terms/> | À demander | Couvert par le même DPA AWS (point 10). Aucune donnée personnelle directe (seuls des IDs). |
| 13 | **AWS SQS** | File d'attente résultats Rekognition | Identifiant de média, statut analyse | `eu-west-1` (configurable) | Non si EU | <https://aws.amazon.com/service-terms/> | À demander | Couvert par le même DPA AWS (point 10). Aucune donnée personnelle directe. |
| 14 | **LiveKit Cloud** | Vidéo-conférence WebRTC | Flux audio/vidéo temps réel (P2P + SFU), métadonnées de salle | US (régions EU disponibles) | Oui — mécanisme à vérifier | <https://livekit.io/legal> ou `legal@livekit.io` | À demander | **À COMPLÉTER (Medium 🔴)** : LiveKit n'est pas listé au DPF publiquement — exiger SCC explicites au moment du DPA. Si refusé, pivoter sur un fournisseur EU (Daily.co EU, Jitsi self-host). Configurer projet sur région EU (`eu-fra`). |
| 15 | **Typesense Cloud** | Moteur de recherche hybride | Index des profils, missions, services (champs publics + dérivés) | Configurable — choisir EU | Oui si non-EU — SCC | <https://typesense.org/legal/> ou ticket support | À demander | **À COMPLÉTER** : si self-hosted, pas de sous-processeur. Si Typesense Cloud, vérifier `TYPESENSE_HOST` pointe vers une région EU et demander le DPA via support. |
| 16 | **Google Firebase Cloud Messaging (FCM)** | Push notifications mobile | Tokens FCM (identifiant device), payload notification | US | Oui — DPF (Google LLC) | <https://firebase.google.com/support/privacy> + <https://cloud.google.com/terms/data-processing-addendum> | À demander | DPA Google unifié pour FCM + Crashlytics. Acceptable via console.firebase.google.com → Project Settings → Legal. |
| 17 | **Firebase Crashlytics** | Crash reports mobile (Flutter) | Stack traces, identifiant device, version OS, logs en clair pré-crash | US | Oui — DPF (Google LLC) | <https://firebase.google.com/support/privacy> | À demander | Couvert par le même DPA Google (point 16). **Désactiver** la collecte d'IP utilisateur si non strictement nécessaire (`setCrashlyticsCollectionEnabled` toggle utilisateur). |
| 18 | **PostHog (EU)** | Product analytics (web + mobile + serveur) | Événements applicatifs, identifiant utilisateur pseudonyme, properties | EU (Frankfurt — `eu.posthog.com`) | Non | <https://posthog.com/dpa> | À demander | DPA signable en self-service depuis posthog.com → Settings → Legal. Confirmer que `POSTHOG_HOST=https://eu.posthog.com` (déjà imposé dans `.env.example`). |
| 19 | **Google Analytics 4 (GA4)** | Web analytics (optionnel) | IP tronquée, pages vues, events anonymisés | US (Region 1 EU possible) | Oui — DPF + SCC | <https://business.safety.google/dpa/> | À demander | **À COMPLÉTER (Medium 🔴)** : la CNIL a historiquement contesté GA4 (avant DPF). Décision à prendre : (a) conserver GA4 + activer Region 1 + IP truncation + signal opt-in via bannière cookies, ou (b) supprimer GA4 et garder PostHog-EU seul. Variable `NEXT_PUBLIC_GA_MEASUREMENT_ID` vide actuellement = pas de risque tant que non configurée. |
| 20 | **GitHub** | Hébergement code source | Code applicatif (pas de données personnelles utilisateur en principe) | US | N/A (pas un sous-processeur de données utilisateur) | <https://github.com/customer-terms/github-data-protection-agreement> | À demander | **Mention seulement** : GitHub n'est pas sous-processeur au sens RGPD tant qu'aucune donnée personnelle d'utilisateur final n'est commitée. **Vérifier** qu'aucun dump SQL / fixture / log d'utilisateur réel n'est dans l'historique git. |
| 21 | **MinIO** | Object storage local (dev) | Aucune donnée prod | Local (Docker compose) | N/A | N/A | N/A (dev only) | MinIO tourne dans `docker-compose.yml` pour le dev local uniquement. En production, le stockage est Cloudflare R2 (point 4). Aucun DPA nécessaire. |

### Récapitulatif

| Mesure | Valeur |
|---|---|
| Sous-processeurs réellement intégrés (DPA requis) | **19** (lignes 1-19) |
| Mention sans DPA (code seul) | 1 (GitHub) |
| Dev-only (pas de prod) | 1 (MinIO) |
| **Total entrées** | **21** |
| Transferts extra-UE (zones grises Schrems II) | Lignes 1, 2, 4, 5, 6, 7, 8, 9, 14, 16, 17, 19 |
| Risque élevé identifié 🔴 | LiveKit (#14), GA4 (#19) |

---

## B. Supply chain — pour mémoire, hors RGPD

Les services suivants servent à la livraison logicielle. Ils **ne traitent pas de données personnelles utilisateur** et ne sont donc pas sous-processeurs au sens Art. 28 :

- **Docker Hub** — registry d'images de build (postgres, redis, minio).
- **npm registry** — packages JavaScript/TypeScript du web et de l'admin.
- **Go module proxy (proxy.golang.org)** — modules Go du backend.
- **pub.dev** — packages Dart/Flutter du mobile.
- **Vercel CLI / Railway CLI / GitHub Actions** — outillage CI/CD.

Aucun DPA requis. À documenter dans une éventuelle SBOM (Software Bill of Materials) si besoin de traçabilité supply-chain.

---

## C. Plan d'action pour le contrôleur (Hassad)

**Échéance cible : J+30 avant tout déploiement public d'envergure** (lancement marketing, ouverture publique des inscriptions, etc.). Pour la phase actuelle de production "soft launch", la signature des DPAs reste légalement obligatoire mais peut être effectuée en parallèle de l'usage.

### Étape 1 — Signatures self-service immédiates (J+1 à J+7)

Ces DPAs sont signables en quelques clics depuis le dashboard du provider :

1. **Vercel** — Dashboard → Team Settings → Security → "Data Processing Addendum" → signer. Télécharger PDF.
2. **Cloudflare** (R2 + CDN + DNS) — Dashboard → Notifications → DPA. Couvre simultanément R2 + Cloudflare core.
3. **Stripe** — Dashboard → Settings → Compliance → Data Processing Agreement. Sélectionner "Stripe Payments Europe Ltd" comme contrepartie UE.
4. **OpenAI** — Settings → Organization → Data Controls → DPA. Activer "Do not train on my data" simultanément.
5. **Anthropic** — Console → Organization → Data Processing Agreement.
6. **PostHog** — Settings → Legal → DPA. Confirmer projet sur instance EU.
7. **Firebase / Google** (FCM + Crashlytics) — Firebase Console → Project Settings → "Google Cloud Terms" → accepter le DPA Google unifié.
8. **AWS** (Rekognition + S3 + SNS + SQS) — AWS Artifact → "AWS GDPR Data Processing Addendum" → accepter.
9. **Resend** — Dashboard → Settings → Legal → DPA.
10. **Neon** — Console → Settings → Legal → DPA. Confirmer région `aws-eu-central-1`.

Pour chaque DPA signé :

```bash
mkdir -p legal/signed-dpas
# Renommer le PDF téléchargé selon la convention :
mv ~/Downloads/dpa.pdf legal/signed-dpas/<vendor>-2026-MM-DD.pdf
# Mettre à jour la colonne Statut du tableau de cette checklist : "Signé YYYY-MM-DD".
```

### Étape 2 — Demandes par ticket support (J+7 à J+21)

Ces providers nécessitent une demande explicite :

11. **Railway** — Ticket support `legal@railway.app` : demander le "GDPR DPA with SCC". Vérifier en même temps que le projet est sur région EU.
12. **LiveKit** — Email `legal@livekit.io` : demander DPA + SCC explicites. **Action de contingence** : si refus de SCC, évaluer la migration vers Daily.co EU ou Jitsi self-hosted (voir `gdpr-audit.md` §3).
13. **Typesense Cloud** — Si utilisé en prod : ticket support pour DPA + confirmation de la région. Si self-hosted, ignorer.

### Étape 3 — Décisions produit avant signature (J+1)

14. **Google Analytics 4** — Décider : conserver ou retirer.
    - **Si conserver** : activer Region 1 EU + IP truncation + ne charger le script qu'après opt-in cookies, puis signer le DPA Google (souvent unifié avec Firebase si même org).
    - **Si retirer** : supprimer toute trace de `NEXT_PUBLIC_GA_MEASUREMENT_ID` du code et de Vercel.
    - Recommandation `gdpr-audit.md` : retirer GA4, garder PostHog-EU seul.

### Étape 4 — Vérifications complémentaires (J+30)

15. **Confirmer les régions** de chaque service prod (idéalement EU partout sauf US-only inévitables) :

    ```text
    Railway      : eu-west1 ?
    Neon         : aws-eu-central-1 ?
    Cloudflare R2: Jurisdiction EU ?
    AWS          : eu-west-1 partout (Rekognition, S3, SNS, SQS) ?
    LiveKit      : projet eu-fra ?
    Vercel       : fonctions serverless fra1 ?
    Typesense    : nœud EU ?
    ```

16. **Mettre à jour la page publique** `web/src/app/[locale]/legal/sous-processeurs/page.tsx` (à créer dans une autre tâche RGPD) avec la même liste, dans un format consommable par les utilisateurs.

17. **Notifier les utilisateurs existants** par email si la liste de sous-processeurs change (Art. 28(2) RGPD — droit d'objection raisonnable).

### Risques en cas de non-signature

| Risque | Détail |
|---|---|
| **Sanction CNIL** | Sans DPA signé, le contrôleur est responsable des fautes du sous-traitant. Amende possible jusqu'à 4% du CA mondial ou 20M€ (Art. 83 RGPD). |
| **Notification de violation** | Sans DPA, le sous-traitant n'est pas obligé contractuellement de te notifier en cas de violation, ce qui empêche de respecter le délai de 72h vers la CNIL (Art. 33). |
| **Refus de prise en charge** | Un utilisateur qui exerce ses droits RGPD peut saisir la CNIL ; sans DPA documenté, la défense est très affaiblie. |
| **Schrems II** | Sans SCC ou DPF documenté pour les transferts US, transfert illicite — la CNIL exige preuve écrite. |

---

## D. Maintenance et procédure

### Revisite périodique

- **Tous les 6 mois** (calendrier : 2026-11-11, 2027-05-11, …) — re-passer chaque ligne :
  1. Le service est-il toujours utilisé ?
  2. Le DPA a-t-il été mis à jour côté provider (nouveau brouillon proposé) ?
  3. La région prod est-elle toujours la bonne ?
  4. Le mécanisme de transfert est-il toujours valide (DPF actif ? nouvelles SCC ?) ?
- **À chaque ajout d'un nouveau provider** : la checklist doit être mise à jour AVANT la mise en prod.

### Procédure d'ajout d'un nouveau sous-processeur

Tout ajout de sous-processeur tiers (nouveau SaaS, nouvelle dépendance qui appelle un service externe) suit cette séquence stricte :

1. **Évaluation préalable** — qualifier les données envoyées (catégorie + sensibilité) et la nécessité réelle. Si une alternative EU existe à coût équivalent, la privilégier.
2. **DPA signé AVANT la mise en prod** — pas de premier appel API en production sans DPA.
3. **Mise à jour de `legal/dpas-checklist.md`** — nouvelle ligne dans le tableau §A.
4. **Mise à jour de `gdpr-audit.md` §3** si transfert hors UE.
5. **Mise à jour de la page publique** `/legal/sous-processeurs` (au moins 30 jours avant le premier appel, sauf force majeure).
6. **Mise à jour de la politique de confidentialité** si la catégorie de sous-traitant est nouvelle.
7. **Notification aux utilisateurs existants** (Art. 28(2) — droit d'objection).
8. **Stocker le PDF signé** dans `legal/signed-dpas/<vendor>-YYYY-MM-DD.pdf` et committer (le PDF, pas les secrets éventuels du dashboard).

### Procédure de retrait d'un sous-processeur

Si un sous-processeur est supprimé du stack :

1. Supprimer les appels code + variables d'environnement Railway/Vercel correspondants.
2. Marquer la ligne de la checklist comme `Retiré YYYY-MM-DD` (ne pas supprimer — garder l'historique).
3. Demander la suppression des données du sous-processeur via leur procédure (Art. 28(3)(g) RGPD).
4. Conserver le DPA signé dans `legal/signed-dpas/` pour traçabilité de la période d'usage.

### Suivi des incidents sous-traitant

Si un sous-processeur signale une violation :

1. Recevoir la notification (le DPA doit imposer un délai max — typiquement 24-72h).
2. Évaluer si des données personnelles d'utilisateurs sont concernées.
3. Si oui : déclencher la procédure interne (voir `docs/runbook-violation.md` à créer — référencé `gdpr-roadmap.md` C8) : notification CNIL sous 72h + utilisateurs concernés si risque élevé.
4. Documenter dans le registre des violations.

---

## Annexes

### Annexe 1 — Variables d'environnement par sous-processeur

| Sous-processeur | Variables backend | Variables web | Variables mobile |
|---|---|---|---|
| Vercel | (déployement) | (déployement) | — |
| Railway | `PORT`, `ENV`, (toutes) | — | — |
| Neon | `DATABASE_URL`, `DATABASE_URL_ADMIN` | — | — |
| Cloudflare R2 | `STORAGE_*`, `STORAGE_AUDIT_COLD_BUCKET` | — | — |
| Stripe | `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PUBLISHABLE_KEY` | `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | — |
| Resend | `RESEND_API_KEY`, `RESEND_DEV_REDIRECT_TO`, `EMAIL_FROM` | — | — |
| OpenAI | `OPENAI_API_KEY`, `OPENAI_EMBEDDINGS_MODEL` | — | — |
| Anthropic | `ANTHROPIC_API_KEY` | — | — |
| AWS Rekognition + S3 + SNS + SQS | `REKOGNITION_REGION`, `REKOGNITION_ROLE_ARN`, `REKOGNITION_ENABLED`, `SNS_TOPIC_ARN`, `SQS_QUEUE_URL` | — | — |
| LiveKit | `LIVEKIT_URL`, `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET` | — | — |
| Typesense | `TYPESENSE_HOST`, `TYPESENSE_API_KEY` | `NEXT_PUBLIC_TYPESENSE_HOST` | — |
| Firebase FCM | `FCM_CREDENTIALS_PATH` | — | `google-services.json` |
| Firebase Crashlytics | — | — | `google-services.json` |
| PostHog | `POSTHOG_PROJECT_KEY`, `POSTHOG_HOST` | `NEXT_PUBLIC_POSTHOG_KEY`, `NEXT_PUBLIC_POSTHOG_HOST` | `--dart-define=POSTHOG_PROJECT_KEY` |
| GA4 | — | `NEXT_PUBLIC_GA_MEASUREMENT_ID` | — |

### Annexe 2 — Références internes

- [`gdpr-audit.md`](../gdpr-audit.md) — audit RGPD général (§3 Transferts extra-UE).
- [`gdpr-roadmap.md`](../gdpr-roadmap.md) — plan de mise en conformité.
- `backend/internal/config/config.go` — configuration applicative (source de vérité pour les env vars).
- `backend/.env.example` — variables documentées avec leur rôle.

### Annexe 3 — Convention de nommage des PDFs signés

```text
legal/signed-dpas/<vendor>-YYYY-MM-DD.pdf
```

Exemples :
- `legal/signed-dpas/stripe-2026-05-15.pdf`
- `legal/signed-dpas/vercel-2026-05-12.pdf`
- `legal/signed-dpas/aws-2026-05-20.pdf` (couvre Rekognition + S3 + SNS + SQS)
- `legal/signed-dpas/cloudflare-2026-05-13.pdf` (couvre R2 + CDN + DNS)
- `legal/signed-dpas/google-2026-05-18.pdf` (couvre Firebase FCM + Crashlytics ; éventuellement GA4)

Conserver également la **version texte du DPA** (`.md` ou `.txt`) si fournie par le provider — facilite la recherche et le diff lors des renouvellements.

---

*Document généré le 2026-05-11 par l'agent D5. À actualiser à chaque revisite semestrielle ou ajout/retrait de sous-processeur.*
