# Politique de confidentialité

> Version publique destinée aux utilisateurs
> Dernière mise à jour : 2026-05-11
> Version : 1.0
> Référence : RGPD art. 13 et 14, Loi Informatique et Libertés
> Audience : tous les utilisateurs et visiteurs de `services.designedtrust.com`

Tutoiement utilisé en français — version anglaise disponible sur `/en/legal/politique-confidentialite`.

---

## En résumé

- **Qui collecte tes données ?** Marketplace Service, hébergé en Europe pour la base de données, avec quelques services techniques aux États-Unis encadrés par DPA + clauses contractuelles types.
- **Pourquoi ?** Pour te permettre de t'inscrire, publier ton profil, échanger avec d'autres utilisateurs, recevoir des paiements et bénéficier des fonctionnalités du service.
- **Combien de temps ?** Tant que ton compte est actif, puis 30 jours après suppression — sauf obligations légales (paiement : 10 ans, KYC : 5 ans, audit log : 4 ans).
- **Tes droits ?** Accès, rectification, effacement, portabilité, opposition, limitation, retrait du consentement, à exercer à tout moment via `dpo@designedtrust.com` ou ton `/dashboard/account/gdpr`.
- **Une question ?** Notre Délégué à la Protection des Données te répond sous 30 jours.

---

## 1. Qui est responsable de tes données ?

| Champ | Valeur |
|---|---|
| Nom du service | Marketplace Service |
| URL | `https://services.designedtrust.com` |
| Raison sociale | `[À COMPLÉTER : raison sociale légale]` |
| Forme juridique | `[À COMPLÉTER]` |
| RCS / SIREN | `[À COMPLÉTER]` |
| Siège social | `[À COMPLÉTER]` |
| Représentant légal | `[À COMPLÉTER : nom + fonction]` |
| Directeur de la publication | `[À COMPLÉTER]` |
| Email de contact RGPD | `dpo@designedtrust.com` |
| Délégué à la Protection des Données (DPO) | `[À COMPLÉTER : DPO interne ou prestataire externe]` |

Le **responsable de traitement** est l'entité qui détermine les finalités et les moyens du traitement de tes données. Pour tout ce qui concerne tes données personnelles sur le service, c'est Marketplace Service. Pour les paiements, Stripe agit en parallèle comme responsable de traitement pour ses propres finalités (anti-fraude, conformité PSD2).

---

## 2. Quelles données on collecte et pourquoi

### 2.1 Quand tu crées un compte

| Donnée | Pourquoi | Base légale (RGPD art. 6) |
|---|---|---|
| Email + mot de passe | T'authentifier en toute sécurité | Contrat (art. 6.1.b) |
| Nom, prénom, rôle | Personnaliser ton expérience | Contrat |
| Métadonnées techniques (IP, user-agent, horodatages) | Sécurité, lutte contre la fraude | Intérêt légitime (art. 6.1.f) |

### 2.2 Quand tu remplis ton profil

| Donnée | Pourquoi | Base légale |
|---|---|---|
| Photo / portrait, biographie, expertises, langues, ville | Permettre aux clients de te trouver | Contrat |
| Tarif indicatif, expérience, portfolio | Mettre en valeur ton activité | Contrat |
| Toggle de visibilité publique | Te laisser choisir si tu apparais en recherche | Consentement (art. 6.1.a) |

### 2.3 Quand tu échanges des messages

Tes messages restent strictement entre toi et ton interlocuteur. Aucun intermédiaire (apporteur d'affaires, support) n'a accès aux conversations sans une raison documentée (par exemple un litige déclaré).

| Donnée | Pourquoi | Base légale |
|---|---|---|
| Contenu des messages, pièces jointes | Te permettre de communiquer | Contrat |
| Horodatages, statut de lecture | Confort d'usage (notification, recherche) | Contrat |
| Métadonnées de notification push | T'envoyer des alertes mobiles | Contrat |

### 2.4 Quand tu reçois ou envoies un paiement

Aucune donnée de carte bancaire ne transite par nos serveurs. Tu saisis tes informations directement chez Stripe, qui est notre prestataire de paiement et qui agit aussi comme responsable de traitement pour ses propres exigences anti-fraude.

| Donnée | Pourquoi | Base légale |
|---|---|---|
| Identifiants Stripe (compte, transaction) | Encaisser et reverser les fonds | Contrat + obligation légale (art. 6.1.c) |
| Adresse de facturation, TVA | Émettre les factures réglementaires | Obligation légale |
| Justificatifs KYC (pièce d'identité, vidéo selfie) | Vérifier ton identité avant tout flux financier (LCB-FT) | Obligation légale + consentement explicite (art. 9.2.a + 9.2.g) pour la biométrie |

### 2.5 Quand tu utilises la recherche

Tes requêtes sont enregistrées de manière pseudonymisée pour comprendre l'usage du moteur. Au-delà de 12 mois, ton identifiant est totalement anonymisé.

| Donnée | Pourquoi | Base légale |
|---|---|---|
| Texte saisi, filtres, page consultée | Mesurer la pertinence du moteur | Intérêt légitime |
| Identifiant utilisateur (si connecté) | Personnaliser les suggestions | Intérêt légitime |

### 2.6 Quand tu consultes le site

Sans ton consentement explicite, on n'utilise que les cookies strictement nécessaires (session, langue, sécurité paiement). La mesure d'audience (PostHog, Google Analytics) est désactivée par défaut et ne s'active que si tu cliques sur "Accepter" dans le bandeau de consentement. Tu peux modifier ton choix à tout moment via le centre de préférences en bas de page.

Détail complet : `/cookies` et `/sous-processeurs`.

---

## 3. Combien de temps on garde tes données

Notre politique de conservation suit le principe de minimisation : on ne garde rien plus longtemps que nécessaire.

| Donnée | Durée |
|---|---|
| Données de compte (email, profil) | Toute la durée d'activité + 30 jours après suppression |
| Messages | 3 ans à compter de l'envoi (suppression automatique) |
| Notifications | 90 jours |
| Tokens push (FCM) | 60 jours d'inactivité |
| Requêtes de recherche | 12 mois (puis anonymisation totale) |
| Sessions révoquées | 30 jours après révocation |
| Journal d'audit (table `audit_logs`) | 24 mois en chaud + 24 mois en archive R2 |
| Données de paiement / facturation | 10 ans (Code de commerce art. L.123-22) |
| Justificatifs KYC / biométrie | 5 ans après fin de relation (Code monétaire et financier art. L.561-12) |

Au-delà, les données sont **supprimées définitivement**. Pour la biométrie, seuls les labels statistiques sont conservés — les frames vidéo ne sont jamais stockées.

---

## 4. Qui peut voir tes données

### 4.1 Notre équipe

Notre équipe technique et administrative accède aux données uniquement quand c'est nécessaire (support, modération, sécurité). Chaque accès à des données sensibles est enregistré dans un journal d'audit que tu peux consulter à la demande.

| Rôle | Accès |
|---|---|
| `admin` | Lecture/écriture sur l'ensemble — journalisé |
| Support | Lecture limitée des comptes — journalisé |
| Développement | Données anonymisées en environnement de staging |

### 4.2 Nos sous-traitants

Pour faire fonctionner le service, on s'appuie sur des sous-traitants encadrés par des contrats RGPD (DPA + clauses contractuelles types). La liste complète et tenue à jour est sur `/sous-processeurs`. Voici les principaux :

| Sous-traitant | Rôle | Pays |
|---|---|---|
| Vercel | Hébergement web | États-Unis (DPA + CCT + DPF) |
| Railway | Hébergement API | États-Unis (DPA + CCT) |
| Neon | PostgreSQL | Union européenne |
| Cloudflare R2 | Stockage des médias | États-Unis / UE (DPA + CCT + DPF) |
| Stripe | Paiements et KYC | Irlande + États-Unis (DPA + CCT + DPF) |
| Resend | Emails transactionnels | États-Unis (DPA + CCT + DPF) |
| LiveKit | Appels vidéo | États-Unis (DPA + CCT + DPF) |
| OpenAI | Modération automatique | États-Unis (DPA + CCT + DPF + opt-out training) |
| Anthropic | Analyse IA litiges | États-Unis (DPA + CCT + DPF + opt-out training) |
| AWS Rekognition | Modération biométrie/image | UE eu-west-1 (DPA + CCT) |
| Firebase Cloud Messaging | Push notifications | États-Unis (DPA + CCT + DPF) |
| Typesense | Moteur de recherche | Union européenne |
| PostHog | Analytics produit (opt-in) | Union européenne |
| Google Analytics 4 | Analytics marketing (opt-in) | États-Unis (DPA + CCT + DPF) |

### 4.3 Tes interlocuteurs sur la plateforme

Tes données publiques (profil, notation, portfolio) sont visibles par les autres utilisateurs et visiteurs anonymes. Tes échanges privés (messages, factures, justificatifs) ne sont jamais partagés avec d'autres utilisateurs sans ton accord ou sans obligation légale.

### 4.4 Autorités compétentes

On peut être amené à communiquer certaines données aux autorités sur réquisition légale (police, justice, fisc, autorité de contrôle). Dans ce cas, on te prévient sauf si la loi nous l'interdit, et on conteste systématiquement les demandes manifestement infondées.

---

## 5. Transferts hors UE

Certains de nos sous-traitants opèrent depuis les États-Unis. Pour chaque transfert, on s'appuie sur :

- la **décision d'adéquation EU-US Data Privacy Framework** (décision UE 2023/1795 du 10 juillet 2023) lorsque le sous-traitant est auto-certifié ;
- les **clauses contractuelles types** adoptées par la Commission européenne (décision UE 2021/914 du 4 juin 2021) ;
- des **mesures supplémentaires** lorsque nécessaire : chiffrement TLS de bout en bout, opt-out training pour les modèles IA, minimisation des identifiants envoyés.

Aucune donnée biométrique brute n'est transférée hors de l'Union européenne — l'analyse Rekognition se fait sur la région eu-west-1 (Irlande).

---

## 6. Tes droits

Tu disposes des droits suivants, à exercer à tout moment :

### 6.1 Droit d'accès (art. 15)

Tu peux obtenir la confirmation que tes données sont traitées et en recevoir une copie complète. Sur la plateforme : `/dashboard/account/gdpr` → "Exporter mes données" (format JSON portable).

### 6.2 Droit de rectification (art. 16)

Tu peux corriger toute donnée inexacte ou incomplète via `/dashboard/profile` ou `/dashboard/account`.

### 6.3 Droit à l'effacement (art. 17)

Tu peux demander la suppression de tes données. Sur la plateforme : `/dashboard/account/gdpr` → "Supprimer mon compte". La suppression est définitive sous 30 jours, sauf pour les données soumises à une obligation légale (paiement : 10 ans, KYC : 5 ans).

### 6.4 Droit à la portabilité (art. 20)

Tu peux récupérer tes données dans un format structuré, couramment utilisé et lisible par machine. C'est le même outil que le droit d'accès : `/dashboard/account/gdpr` → export JSON.

### 6.5 Droit à la limitation du traitement (art. 18)

Si tu contestes l'exactitude d'une donnée ou la légitimité d'un traitement, tu peux demander à ce qu'on suspende ce traitement le temps de l'instruction. Écris à `dpo@designedtrust.com`.

### 6.6 Droit d'opposition (art. 21)

Tu peux t'opposer à un traitement fondé sur l'intérêt légitime (par exemple la mesure d'audience pseudonyme, le classement de recherche). Écris à `dpo@designedtrust.com` ou utilise les paramètres dédiés (`/dashboard/profile` → "Rendre mon profil non public").

### 6.7 Droit au retrait du consentement

Pour les traitements fondés sur ton consentement (cookies opt-in, biométrie KYC, communications marketing), tu peux retirer ton accord à tout moment via le centre de préférences en bas de page ou `/dashboard/account/preferences`.

### 6.8 Droit à une revue humaine des décisions automatisées (art. 22)

Pour les trois traitements automatisés que sont la modération IA, le classement de recherche et l'évaluation de risque des paiements, tu peux demander une intervention humaine via `/decisions-automatisees`.

### 6.9 Droit post-mortem

Tu peux définir des directives sur le sort de tes données après ton décès (art. 85 de la loi Informatique et Libertés). Écris à `dpo@designedtrust.com` pour les communiquer.

### 6.10 Délai de réponse

On s'engage à te répondre **sous 30 jours** à compter de la réception de ta demande (art. 12-3 RGPD). En cas de complexité particulière, ce délai peut être prolongé de deux mois, sur information motivée. Si tu n'es pas satisfait de notre réponse, tu peux saisir la **CNIL** (autorité de contrôle française) sur `cnil.fr/fr/plaintes`.

---

## 7. Sécurité

On met en œuvre des mesures techniques et organisationnelles à l'état de l'art :

- chiffrement TLS 1.2+ en transit + chiffrement at-rest sur Postgres, R2 et S3 ;
- authentification par tokens courts (15 min) + rotation single-use + blocklist Redis ;
- mots de passe stockés en bcrypt cost 12 ;
- protection anti brute-force (5 tentatives / 15 min, lockout 30 min) ;
- politique RBAC stricte (Auth → Rôle → Ownership) + Row-Level Security PostgreSQL ;
- journal d'audit append-only pour tous les accès sensibles ;
- modération automatique des contenus avant publication ;
- sauvegardes PITR 7 jours + snapshots quotidiens 30 jours ;
- formation annuelle de l'équipe et revue trimestrielle des accès.

En cas de violation de données, on s'engage à notifier la CNIL sous 72 heures (art. 33 RGPD) et à t'informer si tes droits et libertés sont susceptibles d'être impactés (art. 34 RGPD).

---

## 8. Cookies et traceurs

Le détail complet se trouve sur `/cookies`. En synthèse :

- **Cookies strictement nécessaires** (session, langue, paiement Stripe) — actifs par défaut, exemption art. 82 de la loi Informatique et Libertés.
- **Mesure d'audience produit (PostHog)** — opt-in uniquement, identifiant pseudonyme, instance UE, 13 mois max.
- **Mesure d'audience marketing (Google Analytics 4)** — opt-in uniquement, IP anonymisée, Google Signals OFF, 14 mois max.

Tu modifies ton choix à tout moment via le centre de préférences (bouton "Cookies" en bas de chaque page).

---

## 9. Décisions automatisées

Trois traitements automatisés peuvent affecter ton expérience :

1. **Modération IA des contenus** (OpenAI + AWS Rekognition) — peut bloquer un message, une image ou un média.
2. **Classement de la recherche** (Typesense déterministe) — peut faire varier ton positionnement dans les résultats.
3. **Évaluation de risque des paiements** (Stripe Radar) — peut bloquer une transaction.

Tu as le droit à une intervention humaine, à exprimer ton point de vue et à contester la décision. Détails et formulaire de recours : `/decisions-automatisees`.

---

## 10. Comment nous contacter

| Pour quoi | Où |
|---|---|
| Question générale RGPD | `dpo@designedtrust.com` |
| Exercer un de tes droits | `/dashboard/account/gdpr` ou `dpo@designedtrust.com` |
| Signaler une violation de données | `dpo@designedtrust.com` (objet : "Violation de données") |
| Saisir l'autorité de contrôle | CNIL — `cnil.fr/fr/plaintes` — 3 place de Fontenoy, TSA 80715, 75334 PARIS CEDEX 07 |

---

## 11. Modifications de cette politique

On peut modifier cette politique pour intégrer de nouvelles fonctionnalités, refléter un changement de sous-traitant ou se conformer à une nouvelle obligation légale. Toute modification substantielle te sera notifiée par email **au moins 30 jours** avant son entrée en vigueur. La version actuelle est toujours disponible sur cette page, avec la date de dernière mise à jour en haut.

L'historique des versions est conservé dans Git (`legal/politique-confidentialite.md`).

---

## English version (summary)

This Privacy Policy is available in French as the legally binding version. An English summary is available on request to `dpo@designedtrust.com`. The English routing `/en/legal/politique-confidentialite` renders this same document with English-translated navigation and headings; the body content remains in French for legal accuracy. A full English translation is planned — `[À COMPLÉTER : ETA de la version EN complète]`.

---

> Document validé le `[À COMPLÉTER : date]` par `[À COMPLÉTER : représentant légal]`, en cohérence avec le registre des traitements (`legal/registre.md`) et l'analyse d'impact (`legal/aipd.md`).
