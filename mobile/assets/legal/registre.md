# Registre des activités de traitement

> Document de conformité — RGPD art. 30
> Responsable de traitement : **Marketplace Service** — `services.designedtrust.com`
> Dernière mise à jour : 2026-05-11
> Version : 1.0
> Statut : `[À COMPLÉTER : validation finale par DPO + responsable légal]`

---

## Sommaire

1. Informations sur le responsable de traitement
2. Méthode et portée du registre
3. Traitement T-01 — Gestion des comptes utilisateurs
4. Traitement T-02 — Profils publics et mise en relation B2B
5. Traitement T-03 — Messagerie temps réel entre utilisateurs
6. Traitement T-04 — Paiements et facturation (Stripe Connect)
7. Traitement T-05 — Recherche de prestataires (Typesense)
8. Traitement T-06 — Vérification d'identité (KYC + biométrie vidéo)
9. Traitement T-07 — Modération automatisée des contenus (texte / image / vidéo)
10. Traitement T-08 — Mesure d'audience produit (PostHog opt-in)
11. Traitement T-09 — Mesure d'audience marketing (Google Analytics 4 opt-in)
12. Traitement T-10 — Support client et résolution des litiges
13. Traitement T-11 — Journal d'audit et traçabilité de sécurité
14. Sous-traitants et transferts hors UE
15. Mesures techniques et organisationnelles communes

---

## 1. Informations sur le responsable de traitement

| Élément | Valeur |
|---|---|
| Nom commercial | Marketplace Service |
| URL de production | `https://services.designedtrust.com` |
| Raison sociale | `[À COMPLÉTER : raison sociale enregistrée]` |
| Forme juridique | `[À COMPLÉTER : SAS / SARL / autre]` |
| Numéro RCS / SIREN | `[À COMPLÉTER]` |
| Adresse postale du siège | `[À COMPLÉTER]` |
| Représentant légal | `[À COMPLÉTER : nom, prénom, fonction]` |
| Délégué à la Protection des Données (DPO) | `[À COMPLÉTER : DPO interne ou externe]` |
| Adresse de contact RGPD | `dpo@designedtrust.com` (variable `NEXT_PUBLIC_DPO_EMAIL`) |

Le présent registre est tenu en application de l'article 30 du Règlement (UE) 2016/679 (RGPD). Il décrit, pour chaque traitement, la finalité, la base légale, les catégories de données, les destinataires, la durée de conservation et les mesures de sécurité associées.

---

## 2. Méthode et portée du registre

- **Périmètre** — toutes les activités opérées par Marketplace Service sur le domaine `services.designedtrust.com` et ses sous-domaines (`api.*`, `cdn.*`).
- **Mise à jour** — au moins annuelle, ainsi qu'à chaque nouveau traitement ou modification substantielle (ajout d'un sous-traitant, changement de finalité, ouverture d'un nouveau pays).
- **Conservation du registre** — pendant 5 ans après la cessation du dernier traitement, conformément aux recommandations de la CNIL.
- **Forme** — document Markdown versionné dans Git (`legal/registre.md`), exportable en Word/PDF pour transmission au DPO ou à la CNIL.

---

## 3. Traitement T-01 — Gestion des comptes utilisateurs

| Élément | Détail |
|---|---|
| **Finalité principale** | Permettre la création, l'authentification, la mise à jour et la suppression de comptes utilisateurs (agences, freelances, entreprises, apporteurs, administrateurs). |
| **Finalités secondaires** | Réinitialisation de mot de passe, rotation des sessions, journalisation de sécurité, droit à l'effacement (RGPD art. 17). |
| **Base légale (RGPD art. 6)** | Exécution du contrat (art. 6.1.b) — pas de service sans compte. |
| **Catégories de personnes concernées** | Utilisateurs inscrits dans les rôles `agency`, `enterprise`, `provider`, `admin`. |
| **Catégories de données** | Identité (nom, prénom), contact (email, téléphone optionnel), authentification (hash bcrypt cost 12), métadonnées techniques (IP, user-agent, horodatages), rôle métier, identifiant organisationnel. |
| **Données sensibles** | Aucune (l'authentification biométrique éventuelle est rattachée au T-06 KYC). |
| **Destinataires internes** | Équipe administrative (rôle `admin`), équipe support de premier niveau (lecture limitée), équipe développement (anonymisée en environnement de staging). |
| **Sous-traitants** | Neon (hébergement PostgreSQL, UE), Railway (hébergement API, US — DPA + SCC), Vercel (hébergement web, US — DPA + SCC), Resend (emails de confirmation/réinitialisation, US — DPA + SCC). |
| **Transferts hors UE** | Oui — Railway, Vercel, Resend (États-Unis). Garanties : Data Processing Agreement signé + clauses contractuelles types (CCT 2021/914) + adhésion au cadre EU-US Data Privacy Framework lorsqu'applicable. |
| **Durée de conservation** | Toute la durée d'activité du compte. À la fermeture : suppression complète sous 30 jours, à l'exception des données nécessaires aux obligations légales (facturation : 10 ans). |
| **Mesures de sécurité spécifiques** | bcrypt cost 12, JWT short-lived (15 min) + rotation refresh, blocklist Redis post-rotation, anti brute-force (5 tentatives / 15 min, lockout 30 min), RLS PostgreSQL sur tables sensibles. |
| **Droits exerçables** | Accès (`/api/v1/me/gdpr/export`), rectification (`/api/v1/me/profile`), effacement (`/api/v1/me/gdpr/delete`), portabilité (export JSON), limitation, opposition. |
| **Référence audit** | `gdpr-audit.md` § 3.1, `internal/domain/retention/policies.go` (user_sessions). |

---

## 4. Traitement T-02 — Profils publics et mise en relation B2B

| Élément | Détail |
|---|---|
| **Finalité principale** | Publier des profils consultables (agences, freelances, apporteurs) et permettre aux entreprises de découvrir des prestataires. |
| **Finalités secondaires** | Affichage de portfolios, lecture des notes/avis, mise en valeur de l'expertise. |
| **Base légale (RGPD art. 6)** | Exécution du contrat (art. 6.1.b) pour l'utilisateur ayant créé le profil. Intérêt légitime (art. 6.1.f) pour la consultation par un tiers prospect. |
| **Catégories de personnes concernées** | Utilisateurs ayant choisi d'activer leur profil public + visiteurs anonymes. |
| **Catégories de données** | Identité professionnelle (nom commercial), photo / portrait, biographie, expertises, langues, taux journalier indicatif, localisation (ville), historique d'expérience, projets publics. |
| **Données sensibles** | Aucune. Les données fiscales (TVA, SIREN) sont privées et rattachées à T-04. |
| **Destinataires** | Visiteurs anonymes et utilisateurs authentifiés. Aucune diffusion à un tiers commercial sans consentement explicite. |
| **Sous-traitants** | Cloudflare R2 (stockage des médias profils, UE/US — DPA + SCC), Typesense (indexation recherche, instance UE), Cloudflare (CDN, US — DPA + SCC). |
| **Transferts hors UE** | Cloudflare (US) — garanties par DPA + CCT + EU-US DPF. |
| **Durée de conservation** | Toute la durée d'activité du profil. À la désactivation : 30 jours de soft-delete (récupération possible), puis suppression définitive. |
| **Mesures de sécurité spécifiques** | Cache public en lecture seule, signature URL R2 pour les médias, modération automatique avant publication (cf. T-07). |
| **Droits exerçables** | Tous les droits RGPD applicables ; rectification immédiate via `/dashboard/profile`. |
| **Référence audit** | `gdpr-audit.md` § 3.2. |

---

## 5. Traitement T-03 — Messagerie temps réel entre utilisateurs

| Élément | Détail |
|---|---|
| **Finalité principale** | Permettre l'échange de messages entre principaux d'une mise en relation B2B (entreprise ↔ prestataire). |
| **Finalités secondaires** | Notification de nouveaux messages, accusés de réception, recherche dans l'historique personnel, modération automatique. |
| **Base légale (RGPD art. 6)** | Exécution du contrat (art. 6.1.b). |
| **Catégories de personnes concernées** | Utilisateurs authentifiés engagés dans une conversation. |
| **Catégories de données** | Contenu des messages (texte, pièces jointes), expéditeur, destinataire, horodatage, statut de lecture, identifiant de conversation. |
| **Données sensibles** | Aucune par défaut — les utilisateurs sont rappelés qu'ils ne doivent pas partager de données sensibles. |
| **Destinataires** | Strictement les deux principaux de la conversation. Aucun apporteur d'affaires n'a accès à une conversation 1-1 entre principaux (cf. `feedback_b2b_confidentiality.md`). L'équipe administrative peut accéder en cas de litige déclaré (cf. T-10). |
| **Sous-traitants** | Neon (base de données, UE), Cloudflare R2 (pièces jointes, UE/US — DPA + SCC), Firebase Cloud Messaging (push, US — DPA + SCC). |
| **Transferts hors UE** | Cloudflare (US), Firebase (US) — garanties par DPA + CCT + EU-US DPF. |
| **Durée de conservation** | **3 ans** à partir de la date d'envoi du message (politique `messages_3y` dans `internal/domain/retention/policies.go`). Suppression automatique par le scheduler de rétention. |
| **Mesures de sécurité spécifiques** | RLS PostgreSQL (politique : seuls expéditeur et destinataire peuvent lire), chiffrement TLS en transit, modération automatique sur l'envoi (cf. T-07), `Permissions-Policy: microphone=(self)` pour messages vocaux. |
| **Droits exerçables** | Accès, rectification (édition de message limitée à 5 min), effacement (à la fermeture du compte), portabilité (export JSON). |
| **Référence audit** | `gdpr-audit.md` § 3.3, `internal/domain/retention/policies.go` (messages_3y). |

---

## 6. Traitement T-04 — Paiements et facturation (Stripe Connect)

| Élément | Détail |
|---|---|
| **Finalité principale** | Encaisser les paiements des clients, reverser les fonds aux prestataires, prélever la commission de la plateforme, émettre factures et reçus. |
| **Finalités secondaires** | Anti-fraude (Stripe Radar), conformité PSD2/SCA, déclarations fiscales obligatoires, gestion des litiges et remboursements. |
| **Base légale (RGPD art. 6)** | Exécution du contrat (art. 6.1.b) + obligation légale (art. 6.1.c) — conservation comptable Code de commerce art. L.123-22. |
| **Catégories de personnes concernées** | Organisations (entreprises clientes, agences, freelances, apporteurs) ayant effectué un paiement ou reçu un versement. |
| **Catégories de données** | Identifiants de paiement Stripe (`stripe_account_id`, `payment_intent_id`), montants, devises, dates de transaction, statut, métadonnées de facturation (raison sociale, adresse), TVA, IBAN (chez Stripe uniquement — jamais stocké côté Marketplace Service), justificatifs KYC (cf. T-06). |
| **Données sensibles** | Aucune au sens RGPD art. 9. Les données financières sont sensibles au sens PSD2 mais ne tombent pas dans la catégorie "particulière" de l'art. 9. |
| **Destinataires** | Équipe administrative (rôle `admin`) pour réconciliation et support facturation. Stripe Inc. en tant que prestataire de services de paiement. Administration fiscale française en cas de demande légale. |
| **Sous-traitants** | Stripe Payments Europe Ltd. (IE) + Stripe Inc. (US) — DPA signé, SCC 2021/914 en place. Resend (envoi des factures, US — DPA + SCC). Neon (stockage des métadonnées, UE). |
| **Transferts hors UE** | Stripe (US), Resend (US) — garanties par DPA + CCT + EU-US DPF. |
| **Durée de conservation** | **10 ans** à compter de la clôture de l'exercice comptable, conformément à l'art. L.123-22 du Code de commerce (obligation comptable) et à l'art. L.102 B du LPF (obligation fiscale). Cette durée prime sur le droit à l'effacement (RGPD art. 17.3.b). |
| **Mesures de sécurité spécifiques** | Aucune donnée de carte ne transite par nos serveurs (champ Stripe Elements). Webhooks Stripe vérifiés par signature HMAC. Audit log de chaque mouvement financier. RLS sur `payment_records`. |
| **Droits exerçables** | Accès, rectification (adresse de facturation), portabilité. **Le droit à l'effacement est limité** par l'obligation légale de conservation comptable. |
| **Référence audit** | `gdpr-audit.md` § 3.4, `STRIPE_MANUAL_PLAYBOOK.md`. |

---

## 7. Traitement T-05 — Recherche de prestataires (Typesense)

| Élément | Détail |
|---|---|
| **Finalité principale** | Permettre à un visiteur ou utilisateur de rechercher un prestataire selon des critères textuels et facettes (compétence, ville, taux, langues). |
| **Finalités secondaires** | Statistiques agrégées d'usage du moteur, amélioration de la pertinence (classement déterministe — pas d'apprentissage opaque, cf. `/decisions-automatisees`). |
| **Base légale (RGPD art. 6)** | Intérêt légitime (art. 6.1.f) — la mise en relation efficace est la raison d'être du service. |
| **Catégories de personnes concernées** | Utilisateurs publiant un profil indexé + utilisateurs effectuant des recherches (anonymisés au-delà de 12 mois). |
| **Catégories de données** | Index : copie publique des profils (cf. T-02). Requêtes : texte saisi, filtres, identifiant de session optionnel, horodatage, identifiant utilisateur si connecté. |
| **Données sensibles** | Aucune. |
| **Destinataires** | Équipe développement (logs agrégés), équipe administrative pour debug. |
| **Sous-traitants** | Typesense Cloud (instance hébergée en UE — DPA en place). |
| **Transferts hors UE** | Aucun. |
| **Durée de conservation** | Index : actualisé en temps réel, miroir des profils publics. Requêtes journalisées (`search_queries`) : **12 mois** au-delà desquels les identifiants `user_id` et `session_id` sont anonymisés (politique `search_queries_12mo_anonymize`). |
| **Mesures de sécurité spécifiques** | Clé API restreinte (read-only pour le frontend), pas de collecte d'IP côté index, journalisation anonymisée après 12 mois. |
| **Droits exerçables** | Effacement immédiat des entrées de recherche à la demande, opposition au traitement statistique. |
| **Référence audit** | `gdpr-audit.md` § 3.5, `internal/domain/retention/policies.go` (search_queries_12mo_anonymize). |

---

## 8. Traitement T-06 — Vérification d'identité (KYC + biométrie vidéo)

| Élément | Détail |
|---|---|
| **Finalité principale** | Vérifier l'identité d'un utilisateur souhaitant recevoir des paiements via Stripe Connect, conformément aux obligations LCB-FT (DSP2, articles L.561-2 et suivants du Code monétaire et financier). |
| **Finalités secondaires** | Lutte contre la fraude à l'identité, conformité KYC Stripe Embedded Components. |
| **Base légale (RGPD art. 6 + 9)** | Obligation légale (art. 6.1.c) — LCB-FT. Données biométriques (vidéo selfie analysée par AWS Rekognition) : intérêt public important (art. 9.2.g) combiné au consentement explicite (art. 9.2.a) recueilli avant la capture. |
| **Catégories de personnes concernées** | Utilisateurs souhaitant recevoir des paiements (rôles `agency`, `provider`, `referrer`). |
| **Catégories de données** | Pièce d'identité (recto-verso), justificatif de domicile, RIB, selfie vidéo (3-5 secondes), labels biométriques retournés par Rekognition (pas de gabarit biométrique stocké côté Marketplace Service), score de confiance. |
| **Données sensibles** | **Oui — données biométriques** (RGPD art. 9.1). Traitement encadré par le considérant 51 et par l'art. 9.2.g + 9.2.a. |
| **Destinataires** | Équipe administrative habilitée (validation manuelle si le score automatisé est inférieur au seuil). Stripe (KYC Embedded). Aucune diffusion à un tiers commercial. |
| **Sous-traitants** | AWS Rekognition (région `eu-west-1`, Irlande — DPA + CCT), AWS S3 (transit avant modération, UE — DPA), AWS SNS/SQS (notification fin de modération, UE — DPA), Stripe (US — DPA + SCC + DPF). |
| **Transferts hors UE** | AWS Rekognition est techniquement en UE mais la société-mère est US — garantie par DPA + CCT 2021/914. Stripe : US — DPF. |
| **Durée de conservation** | Pièce d'identité et selfie : **5 ans** à compter de la fin de la relation contractuelle (obligation LCB-FT, art. L.561-12 du Code monétaire et financier). Score biométrique : 30 jours après validation, puis suppression. |
| **Mesures de sécurité spécifiques** | Chiffrement at-rest R2, accès admin journalisé (audit log), pas de rétention des frames vidéo (analyse à la volée), `Permissions-Policy: camera=(self)`. |
| **Droits exerçables** | Accès, rectification, opposition (impossibilité technique de poursuivre la relation contractuelle Stripe Connect en cas d'opposition). |
| **Référence audit** | `gdpr-audit.md` § 3.6, `MIGRATION_KYC_EMBEDDED.md`. |

---

## 9. Traitement T-07 — Modération automatisée des contenus

| Élément | Détail |
|---|---|
| **Finalité principale** | Détecter et bloquer les contenus interdits (haine, harcèlement, violence, contenu sexuel non consenti, pédopornographie) avant publication ou envoi. |
| **Finalités secondaires** | Protection des utilisateurs, conformité Digital Services Act (DSA — Règlement UE 2022/2065). |
| **Base légale (RGPD art. 6)** | Obligation légale (art. 6.1.c) — DSA + LCEN art. 6 + Code pénal art. 222-33-2-2. Intérêt légitime (art. 6.1.f) pour la modération préventive. |
| **Catégories de personnes concernées** | Tout utilisateur publiant un contenu (texte, image, audio, vidéo). |
| **Catégories de données** | Contenu publié + classification produite par les modèles IA (catégorie, score de confiance). |
| **Données sensibles** | Selon le contenu publié — l'utilisateur reste responsable de ce qu'il transmet. |
| **Destinataires** | Système automatisé exclusivement, sauf en cas de blocage déclenchant un appel humain (RGPD art. 22 — cf. `/decisions-automatisees`). |
| **Sous-traitants** | OpenAI (omni-moderation, US — DPA + SCC + DPF), AWS Rekognition (eu-west-1 — DPA + CCT), Anthropic (analyse litige, US — DPA + SCC). |
| **Transferts hors UE** | OpenAI (US), Anthropic (US) — garanties par DPA + CCT + DPF. |
| **Durée de conservation** | Contenu rejeté : suppression immédiate (ou conservation 7 jours pour appel humain). Classification automatique : 12 mois. |
| **Mesures de sécurité spécifiques** | Pas d'envoi des contenus utilisateur aux modèles à des fins d'entraînement (clauses DPA spécifiques avec OpenAI et Anthropic — opt-out training). Anonymisation des identifiants utilisateurs dans les requêtes vers les modèles. Audit log de chaque décision automatisée. |
| **Droits exerçables** | Droit à une intervention humaine (RGPD art. 22), contestation via le formulaire `/decisions-automatisees`. |
| **Référence audit** | `gdpr-audit.md` § 3.7, page web `/decisions-automatisees`. |

---

## 10. Traitement T-08 — Mesure d'audience produit (PostHog opt-in)

| Élément | Détail |
|---|---|
| **Finalité principale** | Mesurer de manière agrégée l'usage du produit pour améliorer l'expérience utilisateur. |
| **Finalités secondaires** | Identification de bugs par parcours utilisateur, A/B testing de fonctionnalités. |
| **Base légale (RGPD art. 6)** | **Consentement explicite** (art. 6.1.a) — opt-in via la CMP vanilla-cookieconsent. |
| **Catégories de personnes concernées** | Utilisateurs ayant explicitement accepté la mesure d'audience. |
| **Catégories de données** | Identifiant pseudonyme (UUID stocké en localStorage), événements de navigation, durée de session, type d'appareil, version navigateur. |
| **Données sensibles** | Aucune. |
| **Destinataires** | Équipe produit (rôles `admin` et `product`). |
| **Sous-traitants** | PostHog Cloud (instance UE — Irlande — DPA en place). |
| **Transferts hors UE** | Aucun (instance UE-only). |
| **Durée de conservation** | 13 mois maximum (recommandation CNIL pour les cookies de mesure d'audience). |
| **Mesures de sécurité spécifiques** | IP anonymisée côté PostHog, pas de tracking cross-domain, opt-in révocable à tout moment via le centre de préférences. |
| **Droits exerçables** | Retrait du consentement à tout moment, suppression de l'historique sur demande. |
| **Référence audit** | `gdpr-audit.md` § 3.8, `web/src/shared/lib/cookie-consent-config.ts`. |

---

## 11. Traitement T-09 — Mesure d'audience marketing (Google Analytics 4 opt-in)

| Élément | Détail |
|---|---|
| **Finalité principale** | Mesurer l'efficacité des campagnes d'acquisition et la performance des pages publiques. |
| **Base légale (RGPD art. 6)** | **Consentement explicite** (art. 6.1.a) — opt-in via la CMP. |
| **Catégories de personnes concernées** | Visiteurs ayant explicitement accepté la mesure marketing. |
| **Catégories de données** | Identifiant Google (`_ga`, `_ga_*`), pages vues, source de trafic, durée de session, type d'appareil. |
| **Données sensibles** | Aucune. |
| **Destinataires** | Équipe marketing. |
| **Sous-traitants** | Google Ireland Ltd. + Google LLC (US — DPA + SCC + EU-US DPF). |
| **Transferts hors UE** | États-Unis — garanties par DPF (décision d'adéquation UE-US 10 juillet 2023). |
| **Durée de conservation** | 14 mois (paramétré dans GA4). |
| **Mesures de sécurité spécifiques** | IP anonymisée (`anonymizeIp: true`), pas de signaux Google (Google Signals OFF), opt-in révocable. |
| **Droits exerçables** | Retrait du consentement à tout moment via la CMP. |
| **Référence audit** | `gdpr-audit.md` § 3.9. |

---

## 12. Traitement T-10 — Support client et résolution des litiges

| Élément | Détail |
|---|---|
| **Finalité principale** | Répondre aux demandes de support et arbitrer les litiges entre utilisateurs (contestation de paiement, conflit prestataire/client). |
| **Base légale (RGPD art. 6)** | Exécution du contrat (art. 6.1.b) + intérêt légitime (art. 6.1.f). |
| **Catégories de personnes concernées** | Utilisateurs ayant ouvert un ticket ou impliqués dans un litige. |
| **Catégories de données** | Identité, contenu du ticket / pièces jointes, historique de la conversation concernée (si autorisé par les principaux), métadonnées du contrat. |
| **Données sensibles** | Possible selon le contenu transmis. |
| **Destinataires** | Équipe support (rôle `admin`), équipe juridique en cas d'escalade. Anthropic Claude pour assistance à l'analyse (analyse non-identifiante). |
| **Sous-traitants** | Resend (email de réponse, US — DPA), Anthropic (analyse litige, US — DPA + opt-out training). |
| **Transferts hors UE** | Resend, Anthropic (US) — garanties par DPA + CCT + DPF. |
| **Durée de conservation** | Tickets : 3 ans après clôture. Litiges aboutis : 5 ans (en cas de pré-contentieux). |
| **Mesures de sécurité spécifiques** | Audit log de chaque accès admin à une conversation utilisateur, anonymisation des identifiants envoyés à Anthropic, opt-out training. |
| **Droits exerçables** | Accès, rectification, effacement à l'issue de la durée légale. |
| **Référence audit** | `gdpr-audit.md` § 3.10. |

---

## 13. Traitement T-11 — Journal d'audit et traçabilité de sécurité

| Élément | Détail |
|---|---|
| **Finalité principale** | Tracer les actions sensibles (authentification, autorisation, mutations critiques) à des fins de sécurité, d'enquête en cas d'incident et de conformité. |
| **Base légale (RGPD art. 6)** | Obligation légale (art. 6.1.c) + intérêt légitime (art. 6.1.f). |
| **Catégories de personnes concernées** | Tous les utilisateurs (et visiteurs en cas d'incident de sécurité). |
| **Catégories de données** | Identifiant utilisateur, action, ressource cible, horodatage, adresse IP, user-agent, métadonnées de l'événement (sans données personnelles inutiles). |
| **Données sensibles** | Aucune. |
| **Destinataires** | Équipe sécurité (rôle `admin`), équipe RSSI / CISO. Aucune diffusion externe sauf demande judiciaire. |
| **Sous-traitants** | Neon (24 mois en chaud — UE), Cloudflare R2 (24 mois en froid — UE/US — DPA + SCC). |
| **Transferts hors UE** | R2 (US/UE) — garanties par DPA + CCT + DPF. |
| **Durée de conservation** | **24 mois en chaud (PostgreSQL) + 24 mois en froid (R2)** = ~4 ans total. Au-delà : suppression définitive (politique `audit_logs_24mo_archive` puis `audit_logs_archive_to_r2_24mo`). |
| **Mesures de sécurité spécifiques** | Table `audit_logs` append-only (droits PostgreSQL `INSERT` + `SELECT` uniquement), chiffrement at-rest, signature des dumps R2. |
| **Droits exerçables** | Accès limité (l'utilisateur peut accéder à ses propres entrées via `/api/v1/me/gdpr/export`). Le droit à l'effacement est limité par l'obligation légale de traçabilité sécurité. |
| **Référence audit** | `gdpr-audit.md` § 3.11, `internal/domain/retention/policies.go` (audit_logs_24mo_archive + audit_logs_archive_to_r2_24mo). |

---

## 14. Sous-traitants et transferts hors UE

| Sous-traitant | Pays principal | Rôle | Garanties |
|---|---|---|---|
| Vercel Inc. | États-Unis | Hébergement frontend | DPA + CCT 2021/914 + EU-US DPF |
| Railway Corp. | États-Unis | Hébergement backend | DPA + CCT 2021/914 |
| Neon Inc. | Union européenne | PostgreSQL managé | DPA + traitement en UE |
| Cloudflare Inc. | États-Unis (présence UE) | R2 stockage + CDN | DPA + CCT 2021/914 + EU-US DPF |
| Stripe Payments Europe Ltd. | Irlande | Paiement + KYC Embedded | DPA + co-responsable |
| Stripe Inc. | États-Unis | Paiements (rebond US) | DPA + CCT + EU-US DPF |
| Resend Inc. | États-Unis | Email transactionnel | DPA + CCT + EU-US DPF |
| LiveKit Inc. | États-Unis | Appels vidéo WebRTC | DPA + CCT + EU-US DPF |
| OpenAI LLC | États-Unis | Modération texte + embeddings | DPA + CCT + DPF + clauses opt-out training |
| Anthropic PBC | États-Unis | Analyse IA litiges | DPA + CCT + DPF + clauses opt-out training |
| Amazon Web Services (Rekognition, S3, SNS, SQS) | UE (eu-west-1) — société US | Modération visuelle + transit | DPA + CCT 2021/914 + EU-US DPF |
| Google Ireland Ltd. (FCM) | Irlande | Push notifications | DPA + CCT |
| Google LLC (GA4) | États-Unis | Mesure d'audience marketing | DPA + CCT + EU-US DPF |
| Typesense Cloud | Union européenne | Moteur de recherche | DPA + UE-only |
| PostHog Ltd. | Union européenne (Irlande) | Analytics produit | DPA + UE-only |
| VIES (Commission européenne) | UE | Validation TVA intra-UE | Service public — exemption sous-traitance |
| Nominatim (OSM) | Allemagne | Auto-complétion adresses | API publique — minimisation |
| BAN (data.gouv.fr) | France | Auto-complétion adresses FR | API publique — minimisation |
| Photon (komoot.io) | Allemagne | Auto-complétion villes monde | API publique — minimisation |

**Liste publique** : page `/sous-processeurs` (mise à jour à chaque changement, notification aux utilisateurs J-30).

---

## 15. Mesures techniques et organisationnelles communes

### 15.1 Mesures techniques

- **Chiffrement en transit** : TLS 1.2+ obligatoire (HSTS `max-age=31536000; includeSubDomains`).
- **Chiffrement at-rest** : Postgres (Neon), R2 (Cloudflare), S3 (AWS) — clés gérées par les sous-traitants.
- **Authentification** : JWT short-lived (15 min) + refresh token avec rotation single-use + blocklist Redis.
- **Mots de passe** : bcrypt cost 12, exigences 8+ caractères, mixed case + chiffre + spécial.
- **Anti brute-force** : Redis sliding window 5 tentatives / 15 min, lockout 30 min.
- **RBAC** : trois niveaux (Auth → Role → Ownership) appliqués handler-side ET RLS PostgreSQL en filet de sécurité.
- **Headers HTTP de sécurité** : CSP, X-Content-Type-Options, X-Frame-Options, HSTS, Referrer-Policy, Permissions-Policy (cf. `middleware/security_headers.go`).
- **Rate-limiting** : Redis sliding window — 100 req/min global, 5 req/min auth, 30 req/min mutations, 10 req/min upload.
- **Audit log** : table `audit_logs` append-only, 24 mois chaud + 24 mois froid R2.
- **Sauvegardes** : Neon PITR (Point-In-Time Recovery) 7 jours, snapshots quotidiens 30 jours.
- **Sécurité applicative** : validation Zod / Go validate au boundary, parameterized SQL, sanitization HTML, opt-out training IA.
- **Modération automatique** : OpenAI omni-moderation + AWS Rekognition avant publication.
- **Sandbox** : environnements dev / staging / prod cloisonnés, secrets différenciés.

### 15.2 Mesures organisationnelles

- **Politique de sécurité de l'information** : `[À COMPLÉTER : politique formelle signée]`.
- **Accès aux données** : principe du moindre privilège, revue trimestrielle des accès `admin`.
- **Sensibilisation** : `[À COMPLÉTER : formation annuelle RGPD obligatoire pour l'équipe]`.
- **Gestion des incidents** : processus de notification à la CNIL sous 72h en cas de violation (`SECURITY.md`).
- **Audit** : audit annuel interne, audit externe `[À COMPLÉTER : périodicité à définir avec DPO]`.
- **Sous-traitants** : revue annuelle des DPA, notification J-30 en cas d'ajout d'un sous-traitant (cf. `/sous-processeurs`).
- **Continuité d'activité** : plan de continuité en place pour les services critiques (Auth, Paiements, Messagerie).

---

## Annexe — Références juridiques

- **RGPD** — Règlement (UE) 2016/679 du 27 avril 2016.
- **Loi Informatique et Libertés** — Loi n° 78-17 du 6 janvier 1978 modifiée.
- **LCEN** — Loi n° 2004-575 du 21 juin 2004.
- **Code de commerce** — art. L.123-22 (conservation comptable 10 ans).
- **Code monétaire et financier** — art. L.561-2 et suivants (LCB-FT, 5 ans).
- **LPF** — art. L.102 B (obligation fiscale).
- **DSA** — Règlement (UE) 2022/2065 du 19 octobre 2022.
- **Décision d'adéquation EU-US DPF** — Décision UE 2023/1795 du 10 juillet 2023.
- **Clauses contractuelles types** — Décision UE 2021/914 du 4 juin 2021.

---

**Signature du registre — `[À COMPLÉTER : nom, fonction, date]`**
**Validation DPO — `[À COMPLÉTER : nom du DPO, date]`**
