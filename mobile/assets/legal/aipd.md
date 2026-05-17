# Analyse d'Impact relative à la Protection des Données (AIPD)

> Document de conformité — RGPD art. 35
> Responsable de traitement : **Marketplace Service**
> Dernière mise à jour : 2026-05-11
> Version : 1.0
> Statut : document tenu par l'éditeur (micro-entreprise non soumise à l'obligation de désigner un DPO au sens de l'art. 37 RGPD) ; révisé à chaque évolution des traitements.

---

## Sommaire

1. Contexte et méthodologie
2. AIPD-01 — Vérification biométrique d'identité (AWS Rekognition)
3. AIPD-02 — Modération automatisée par modèles d'IA tiers (OpenAI + Anthropic)
4. AIPD-03 — Profilage commercial et matching prestataire/client
5. Synthèse des risques résiduels
6. Avis du DPO et décision

---

## 1. Contexte et méthodologie

### 1.1 Pourquoi une AIPD ?

Conformément à l'article 35 du RGPD, une AIPD est obligatoire lorsque le traitement présente un risque élevé pour les droits et libertés des personnes physiques. La CNIL publie une liste de traitements pour lesquels une AIPD est requise (délibération n° 2018-327 du 11 octobre 2018), et une liste de traitements pour lesquels elle n'est pas requise (délibération n° 2019-118 du 12 septembre 2019).

Marketplace Service opère trois traitements identifiés à risque élevé qui requièrent chacun une AIPD distincte :

| AIPD | Traitement | Critère CNIL applicable |
|---|---|---|
| AIPD-01 | Vérification biométrique d'identité (vidéo selfie + AWS Rekognition) | Données biométriques + identification de personnes physiques (art. 9.1 RGPD) |
| AIPD-02 | Modération automatisée par modèles d'IA tiers | Profilage à grande échelle + décisions automatisées (art. 22) |
| AIPD-03 | Profilage commercial pour matching prestataire/client | Évaluation systématique et exhaustive d'aspects personnels (art. 35.3.a) |

### 1.2 Méthodologie

L'AIPD suit la méthode CNIL (guide PIA 1/2/3) :

1. **Description** du traitement (finalités, données, acteurs, technologies, supports).
2. **Évaluation** de la nécessité et de la proportionnalité au regard des principes RGPD.
3. **Identification des risques** pour la vie privée (accès illégitime, modification non désirée, disparition).
4. **Mesures envisagées** pour traiter les risques (techniques, organisationnelles).
5. **Avis** du DPO.
6. **Décision** du responsable de traitement.

Chaque risque est coté : **Vraisemblance** (négligeable / limitée / importante / maximale) × **Gravité** (négligeable / limitée / importante / maximale).

---

## 2. AIPD-01 — Vérification biométrique d'identité

### 2.1 Description du traitement

| Élément | Détail |
|---|---|
| **Finalité** | Vérifier l'identité d'un utilisateur souhaitant recevoir des paiements via Stripe Connect (obligations LCB-FT). |
| **Acteurs** | Marketplace Service (responsable), AWS Rekognition (sous-traitant), Stripe (KYC Embedded). |
| **Personnes concernées** | Utilisateurs `agency`, `provider`, `referrer` souhaitant être payés. |
| **Données traitées** | Pièce d'identité (recto-verso), selfie vidéo 3-5 s, labels biométriques retournés par Rekognition (`face_match_score`, `liveness_score`), score global de confiance. |
| **Technologies** | Capture caméra navigateur (`getUserMedia`) + upload chiffré vers S3 transit (eu-west-1) → SQS notification → Rekognition (eu-west-1) → comparaison facial avec la pièce d'identité. |
| **Supports** | Données : R2 (chiffré at-rest), bases relationnelles Neon (UE). |
| **Mode de collecte** | Active — l'utilisateur est explicitement informé et doit consentir avant la capture. |

### 2.2 Nécessité et proportionnalité

| Critère | Évaluation |
|---|---|
| **Finalité légitime** | OUI — obligation LCB-FT (Code monétaire et financier art. L.561-2) qui impose la vérification d'identité avant de manipuler des flux financiers. |
| **Données minimales** | OUI — seuls les labels et le score de Rekognition sont conservés. Les frames vidéo ne sont pas conservées (analyse à la volée). Pas de gabarit biométrique persisté côté Marketplace Service. |
| **Durée minimale** | OUI — durée alignée sur l'obligation LCB-FT (5 ans après fin de relation). |
| **Exactitude** | OUI — un seuil de confiance déclenche une revue humaine en cas de doute. |
| **Transparence** | OUI — page `/decisions-automatisees` + bandeau de consentement avant la capture. |
| **Alternatives explorées** | OUI — vérification manuelle pure (impossible à grande échelle) ; envoi de pièces sans vidéo (insuffisant face à la fraude à l'identité de synthèse / deepfake) ; KYC tiers humain (latence + coût). Le KYC biométrique avec revue humaine en cas de doute est l'équilibre optimal. |

**Conclusion** — le traitement est nécessaire et proportionné au regard de l'obligation LCB-FT.

### 2.3 Risques identifiés

| # | Risque | Vraisemblance | Gravité | Mesures envisagées |
|---|---|---|---|---|
| R1 | Vol des frames vidéo en transit (interception réseau) | Limitée | Importante | TLS 1.2+, certificats pinned, durée de vie URL signée 15 min |
| R2 | Accès non autorisé aux pièces d'identité stockées | Limitée | Maximale | Chiffrement at-rest R2, RBAC strict (rôle `admin` + ownership check), audit log de chaque accès, isolation par bucket |
| R3 | Faux positif → personne refusée à tort | Importante | Limitée | Revue humaine systématique en dessous d'un seuil de confiance ; possibilité de re-soumission ; canal de support dédié |
| R4 | Faux négatif → fraude à l'identité passe la barrière | Limitée | Importante | Score combiné Rekognition + vérification Stripe + détection liveness ; revue manuelle sur signaux risque additionnels |
| R5 | Biais algorithmique (Rekognition moins précis sur certaines populations) | Importante | Importante | Revue humaine obligatoire en cas de score limite ; suivi statistique trimestriel des taux de rejet par profil ; recours possible via formulaire `/decisions-automatisees` |
| R6 | Transfert hors UE des données biométriques | Limitée | Importante | Rekognition est en eu-west-1 (Irlande) ; société-mère US couverte par DPA + CCT 2021/914 + EU-US DPF |
| R7 | Réutilisation des données à des fins d'entraînement IA chez le sous-traitant | Limitée | Maximale | DPA AWS Rekognition explicite : pas d'utilisation des inputs clients à des fins d'entraînement (clause contractuelle) |
| R8 | Non-respect du droit à l'effacement post-relation | Limitée | Importante | Suppression automatique 5 ans après la fin de la relation (politique programmée), audit trimestriel |

### 2.4 Mesures retenues

**Techniques :**
- Chiffrement TLS en transit + at-rest.
- Pas de stockage des frames vidéo (analyse à la volée).
- URL signées 15 min pour les pièces d'identité, RBAC strict.
- Seuil de confiance ajusté avec sur-revue humaine.
- Détection de liveness pour bloquer les deepfakes.
- Audit log append-only de chaque décision.

**Organisationnelles :**
- Procédure documentée de revue humaine dans les 72h.
- Formation annuelle de l'équipe admin habilitée.
- Recours via `/decisions-automatisees` + DPO.
- Statistiques trimestrielles de taux de rejet pour détecter un biais (R5).

**Contractuelles :**
- DPA AWS Rekognition signé avec clause opt-out training.
- DPA Stripe pour la liaison KYC Embedded.

### 2.5 Risque résiduel

Après mesures, les risques résiduels sont **acceptables** : la combinaison de la revue humaine, du seuil de confiance et du droit de recours préserve les droits des personnes, tout en répondant à l'obligation légale LCB-FT.

---

## 3. AIPD-02 — Modération automatisée par modèles d'IA tiers

### 3.1 Description du traitement

| Élément | Détail |
|---|---|
| **Finalité** | Détecter et bloquer automatiquement les contenus interdits (haine, harcèlement, violence, sexuel non consenti, pédopornographie). Obligation DSA + LCEN. |
| **Acteurs** | Marketplace Service (responsable), OpenAI (omni-moderation, US), Anthropic (analyse litige, US), AWS Rekognition (image/vidéo, UE). |
| **Personnes concernées** | Tout utilisateur publiant un message, image, audio, vidéo ou portfolio. |
| **Données traitées** | Contenu publié (texte / média) + métadonnées de la classification (catégorie, score, decision). |
| **Technologies** | Appel HTTPS POST aux APIs OpenAI / Rekognition avec le contenu en clair, opt-out training activé contractuellement. Anthropic utilisé uniquement pour l'analyse a posteriori des litiges. |
| **Supports** | Sortie de modération : Postgres Neon (table `moderation_decisions`). Contenu rejeté : suppression immédiate ou 7 jours pour appel humain. |

### 3.2 Nécessité et proportionnalité

| Critère | Évaluation |
|---|---|
| **Finalité légitime** | OUI — obligation DSA (Règlement UE 2022/2065) pour les plateformes intermédiaires de retrait des contenus illicites dans les meilleurs délais. |
| **Données minimales** | PARTIEL — l'envoi du contenu à un modèle tiers est nécessaire mais maximaliste. Mitigation : pas d'identifiants utilisateurs envoyés, pas d'historique transmis. |
| **Durée minimale** | OUI — les contenus rejetés sont supprimés sous 7 jours (délai d'appel), les classifications sont conservées 12 mois. |
| **Transparence** | OUI — page `/decisions-automatisees` détaille les trois systèmes (modération, classement, paiement). |
| **Droit à une revue humaine** | OUI — formulaire de recours documenté avec délai de réponse 30 jours (RGPD art. 12-3). |
| **Alternatives explorées** | OUI — modération purement humaine (impossible à grande échelle + retard d'action sur contenus dangereux) ; modération par règles simples (efficacité insuffisante face à la créativité des contenus interdits). |

**Conclusion** — nécessaire, proportionné, encadré par un droit de recours humain.

### 3.3 Risques identifiés

| # | Risque | Vraisemblance | Gravité | Mesures envisagées |
|---|---|---|---|---|
| R1 | Fausse-positivité (contenu légitime bloqué) | Importante | Limitée | Recours humain sous 30 jours ; conservation 7 jours du contenu pour permettre l'appel ; seuils ajustables |
| R2 | Fausse-négativité (contenu interdit passe) | Importante | Importante | Double vérification (OpenAI + Rekognition pour les médias) ; signalement utilisateur facilité ; modération humaine sur signal |
| R3 | Réutilisation par OpenAI/Anthropic à des fins d'entraînement | Limitée | Importante | Clause opt-out training dans les DPA des deux fournisseurs ; revue annuelle de la conformité |
| R4 | Fuite du contenu utilisateur via les logs des sous-traitants | Limitée | Importante | Pas d'identifiants utilisateurs transmis ; DPA OpenAI/Anthropic limitant la rétention à 30 jours ; chiffrement TLS |
| R5 | Biais discriminatoire (modèles entraînés sur des corpus non représentatifs) | Importante | Importante | Revue humaine sur appel ; statistiques trimestrielles par catégorie de rejet ; possibilité d'élargir les fournisseurs |
| R6 | Transfert hors UE | Limitée | Limitée | DPA + CCT 2021/914 + EU-US DPF pour OpenAI/Anthropic ; Rekognition en eu-west-1 |
| R7 | Non-respect du droit à une intervention humaine (RGPD art. 22) | Limitée | Importante | Formulaire `/decisions-automatisees` + DPO + délai garanti 30 jours |
| R8 | Modèle obsolète (drift) — détection insuffisante de nouvelles formes de contenus illicites | Importante | Limitée | Veille mensuelle, mise à jour des prompts/seuils, revue qualité trimestrielle |

### 3.4 Mesures retenues

**Techniques :**
- Anonymisation des identifiants utilisateurs avant envoi aux modèles.
- Conservation 7 jours des contenus rejetés (pour appel) puis suppression.
- Double-modération (OpenAI + Rekognition) pour les médias.
- Audit log de chaque décision automatique.

**Organisationnelles :**
- Formulaire `/decisions-automatisees` + réponse humaine sous 30 jours.
- Statistiques mensuelles de taux de rejet par type de contenu.
- Veille technique trimestrielle des fournisseurs.
- Procédure de retrait DSA documentée : signalement via le bouton « Signaler » présent sur chaque profil, message, proposition et conversation, accusé de réception sous 48 h et instruction sous 10 jours ouvrés (DSA art. 16-17), voie d'appel par email à hassadsmara@designedtrust.com.

**Contractuelles :**
- DPA OpenAI + opt-out training (Zero Data Retention API quand disponible).
- DPA Anthropic + opt-out training.
- DPA AWS Rekognition.

### 3.5 Risque résiduel

Après mesures, les risques résiduels sont **acceptables**. Le recours humain garanti et la transparence sur `/decisions-automatisees` préservent les droits de la personne. Le risque de biais (R5) reste **à surveiller activement** via les statistiques trimestrielles.

---

## 4. AIPD-03 — Profilage commercial et matching prestataire/client

### 4.1 Description du traitement

| Élément | Détail |
|---|---|
| **Finalité** | Mettre en avant les prestataires les plus pertinents pour un client donné, en fonction de critères textuels, historiques d'activité et signaux d'engagement. |
| **Acteurs** | Marketplace Service (responsable), Typesense Cloud (sous-traitant, instance UE). |
| **Personnes concernées** | Utilisateurs disposant d'un profil public indexé (`agency`, `provider`, `referrer`). |
| **Données traitées** | Copie publique du profil (titre, expertises, ville, langues, taux indicatif), historique d'activité (fraîcheur), signaux d'engagement (taux de réponse, complétude du profil), notation moyenne. |
| **Technologies** | Typesense — moteur de recherche full-text déterministe + scoring multi-critère. **Pas d'apprentissage automatique opaque** : les pondérations sont documentées en interne et publiées dans `/decisions-automatisees`. |
| **Supports** | Index Typesense (instance UE-only). |

### 4.2 Nécessité et proportionnalité

| Critère | Évaluation |
|---|---|
| **Finalité légitime** | OUI — la mise en relation efficace est la raison d'être de la plateforme. Intérêt légitime du responsable + bénéfice direct pour l'utilisateur. |
| **Données minimales** | OUI — seules les données publiques du profil et les signaux d'engagement publics sont utilisés. Pas de croisement avec des données externes. |
| **Pas de décision automatisée affectant la personne** | OUI — le classement influence la visibilité mais n'a pas d'effet juridique ni similaire significatif (un humain reste maître du choix final). Sortie hors du périmètre art. 22 selon la doctrine CNIL. |
| **Transparence** | OUI — pondérations publiées dans `/decisions-automatisees`. Pas d'IA opaque. |
| **Alternatives explorées** | OUI — classement alphabétique (insuffisant pour la mise en relation efficace) ; classement par ancienneté (favorise les comptes anciens au détriment des nouveaux) ; classement aléatoire (UX dégradée). |

**Conclusion** — nécessaire et proportionné. Le caractère déterministe du classement et la transparence des pondérations sont essentiels.

### 4.3 Risques identifiés

| # | Risque | Vraisemblance | Gravité | Mesures envisagées |
|---|---|---|---|---|
| R1 | Effet de bord économique pour un prestataire mal classé (perte de revenu) | Importante | Limitée | Transparence des critères + possibilité d'améliorer son profil ; alerte sur les signaux dégradants |
| R2 | Détournement du classement (manipulation des signaux d'engagement) | Limitée | Limitée | Détection des comportements anormaux (audit log) ; pondération conservatrice des signaux manipulables |
| R3 | Biais structurel (un type de profil systématiquement défavorisé) | Limitée | Importante | Pondérations publiées ; revue annuelle des écarts ; recours humain via `/decisions-automatisees` |
| R4 | Drift opaque du classement après mise à jour des pondérations | Limitée | Limitée | Toute mise à jour des pondérations est versionnée et documentée publiquement |
| R5 | Atteinte au droit à l'opposition (un utilisateur ne souhaite plus apparaître dans la recherche) | Limitée | Limitée | Désindexation immédiate sur opposition (toggle "Rendre mon profil non-public") |

### 4.4 Mesures retenues

**Techniques :**
- Classement déterministe (pas de ML opaque).
- Pondérations versionnées et publiques.
- Toggle de visibilité dans `/dashboard/profile`.
- Désindexation immédiate sur opposition (< 5 min).

**Organisationnelles :**
- Revue annuelle des écarts de classement par catégorie.
- Statistiques publiques globales (`/statistics`).
- Procédure documentée de modification des pondérations.

### 4.5 Risque résiduel

Après mesures, les risques résiduels sont **acceptables**. Le caractère déterministe et public du classement, combiné au droit d'opposition, préserve les droits des personnes.

---

## 5. Synthèse des risques résiduels

| AIPD | Niveau de risque avant mesures | Niveau de risque après mesures |
|---|---|---|
| AIPD-01 — KYC biométrique | Élevé | Faible-modéré (R5 biais à surveiller) |
| AIPD-02 — Modération IA | Élevé | Modéré (R5 biais à surveiller activement) |
| AIPD-03 — Matching | Modéré | Faible |

Aucun des trois traitements ne présente de **risque résiduel élevé** après mesures. La consultation préalable de la CNIL (RGPD art. 36) n'est donc pas requise. Une revue annuelle de cette AIPD est planifiée.

---

## 6. Avis du DPO et décision

### 6.1 Avis du DPO

Avis du responsable de traitement (l'éditeur étant entrepreneur individuel, il n'y a pas de DPO distinct au sens de l'art. 37 RGPD ; l'analyse ci-dessous est conduite et arrêtée par le responsable de traitement) :

- L'analyse a-t-elle correctement identifié les risques ? Oui — les risques d'accès non autorisé, de fuite de données et de réidentification sont identifiés et couverts par les mesures décrites.
- Les mesures retenues sont-elles proportionnées ? Oui — chiffrement, minimisation, contrôle d'accès, durées de conservation bornées et journalisation d'audit sont proportionnés aux finalités.
- Recommandations complémentaires : revoir l'analyse à chaque nouveau traitement ou évolution majeure du service.
- Avis : favorable à la mise en œuvre des traitements, sous réserve du maintien des mesures décrites.

Analyse conduite et arrêtée par Hassad SMARA, responsable de traitement — 17 mai 2026.

### 6.2 Décision du responsable de traitement

Le responsable de traitement décide de mettre en œuvre les traitements décrits, les mesures de protection étant jugées proportionnées aux risques identifiés.

Hassad SMARA, responsable de traitement (entrepreneur individuel) — 17 mai 2026.

### 6.3 Périodicité de revue

- Revue annuelle de l'AIPD.
- Revue immédiate en cas de :
  - changement de sous-traitant principal (Rekognition, OpenAI, Anthropic),
  - changement de finalité,
  - incident de sécurité majeur,
  - décision CNIL ou jurisprudence européenne affectant l'un des traitements.

---

## Annexe — Méthodologie CNIL référencée

- Guide CNIL — Analyses d'impact relatives à la protection des données (PIA), version mise à jour 2024.
- Délibération CNIL n° 2018-327 du 11 octobre 2018 — liste des traitements pour lesquels une AIPD est requise.
- Délibération CNIL n° 2019-118 du 12 septembre 2019 — liste des traitements pour lesquels une AIPD n'est pas requise.
- Groupe de l'Article 29 (EDPB) — lignes directrices WP248 sur l'AIPD (octobre 2017, révisées 2018).
