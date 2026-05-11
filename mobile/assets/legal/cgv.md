# Conditions Générales de Vente (CGV)

> Version : 1.0
> Dernière mise à jour : 2026-05-11
> Référence : `legal/cgv.md`
> Statut : **Base à valider par un conseil juridique** — `[À COMPLÉTER : validation cabinet d'avocats + commissaire aux comptes]` avant déploiement public.
> Audience : tous les utilisateurs effectuant ou recevant un paiement via Marketplace Service.

---

## Article 1 — Objet

Les présentes Conditions Générales de Vente (« **CGV** ») définissent les conditions financières et commerciales applicables aux **transactions** effectuées via la plateforme `services.designedtrust.com` (« le **Service** » ou « **Marketplace Service** »).

Les présentes CGV complètent les **Conditions Générales d'Utilisation** (`legal/cgu.md`) et la **Politique de confidentialité** (`/legal/politique-confidentialite`). En cas de contradiction, les CGV prévalent pour les questions financières et commerciales.

Le Service est une **marketplace B2B** dont la finalité est exclusivement professionnelle. Les Utilisateurs déclarent agir à titre professionnel, à l'exclusion de toute relation entre professionnel et consommateur au sens du Code de la consommation.

---

## Article 2 — Définitions

| Terme | Définition |
|---|---|
| **Éditeur** | `[À COMPLÉTER : raison sociale]`, opérateur du Service. |
| **Utilisateur** | Toute personne ayant créé un compte sur le Service, agissant à titre professionnel. |
| **Client** | Utilisateur de rôle `enterprise` qui publie une mission et engage un prestataire. |
| **Prestataire** | Utilisateur de rôle `agency` ou `provider` qui exécute une mission pour un Client. |
| **Apporteur d'affaires** | Utilisateur ayant activé le toggle `referrer_enabled` et qui met en relation un Client et un Prestataire. |
| **Mission** | Opportunité publiée par un Client, en vue d'engager un Prestataire. |
| **Proposal** | Offre commerciale formulée par un Prestataire dans une conversation, contenant prix, délais et livrables. |
| **Contrat** | Engagement contractuel résultant de l'acceptation d'une proposal entre un Client et un Prestataire. |
| **Plateforme** | Le Service dans son ensemble. |
| **Stripe Connect** | Solution technique de paiement opérée par Stripe Payments Europe Ltd. (IE) et Stripe Inc. (US). |

---

## Article 3 — Modèle économique

### 3.1 Service gratuit d'accès

L'inscription et la consultation du Service sont gratuites pour tous les Utilisateurs. La publication d'une mission, la candidature à une mission et l'échange de messages ne donnent lieu à aucune facturation directe.

### 3.2 Commission sur transaction

La rémunération de l'Éditeur provient d'une **commission prélevée sur chaque transaction** transitant par la Plateforme.

| Élément | Taux par défaut | Notes |
|---|---|---|
| Commission Plateforme (côté Client) | `[À COMPLÉTER : ex. 5 % HT]` | Ajoutée au montant facturé par le Prestataire |
| Commission Plateforme (côté Prestataire) | `[À COMPLÉTER : ex. 10 % HT]` | Prélevée sur le montant brut versé |
| Commission Apporteur d'affaires (le cas échéant) | `[À COMPLÉTER : ex. 5 % du HT après commission Plateforme]` | Prélevée sur la part Prestataire |
| Frais Stripe (paiement carte) | Refacturés au coût | `[À COMPLÉTER : voir grille Stripe]` |

Les taux ci-dessus sont indicatifs et peuvent être ajustés par formules promotionnelles ou abonnements premium. **Le taux applicable à une transaction est celui affiché à l'utilisateur au moment de la signature de la proposal**.

### 3.3 Abonnements premium (optionnel)

L'Éditeur peut proposer des abonnements premium (`subscriptions`) à destination des Organisations, qui permettent par exemple :

- une visibilité accrue dans la recherche ;
- la suppression de la commission Plateforme côté Prestataire au-delà d'un seuil de chiffre d'affaires ;
- des fonctionnalités avancées (statistiques, supports prioritaire).

Les modalités, les tarifs et les conditions de résiliation sont définis dans le `[À COMPLÉTER : annexe Premium]` et publiés sur `/pricing`.

### 3.4 Transparence

Tous les montants sont affichés HT et TTC le cas échéant. Les commissions sont précisées avant validation de toute transaction. Aucun frais caché : tout prélèvement apparaît dans `/dashboard/billing` (factures + relevés).

---

## Article 4 — Vérification d'identité (KYC) et activation des paiements

### 4.1 Obligation KYC

Conformément aux articles L.561-2 et suivants du Code monétaire et financier (LCB-FT) et aux exigences de Stripe Connect, **toute Organisation souhaitant recevoir des paiements doit passer une vérification d'identité préalable** (Know Your Customer — KYC).

### 4.2 Pièces demandées

| Type d'entité | Pièces minimales |
|---|---|
| Personne physique | Pièce d'identité officielle (carte d'identité, passeport, titre de séjour), justificatif de domicile de moins de 3 mois, RIB. |
| Personne morale | Extrait Kbis < 3 mois, statuts à jour, pièce d'identité du dirigeant, justificatif de siège, RIB, déclaration des bénéficiaires effectifs. |

Une **vérification biométrique** (vidéo selfie analysée par AWS Rekognition) est exigée pour valider la concordance entre la pièce d'identité et la personne. Détails : `legal/aipd.md` § AIPD-01.

### 4.3 Délais

La validation KYC est traitée sous **5 jours ouvrés** maximum après transmission du dossier complet. En cas de doute, une revue humaine peut prolonger le délai jusqu'à 10 jours ouvrés. L'Utilisateur peut suivre l'avancement dans `/dashboard/account/kyc`.

### 4.4 Refus

En cas de refus, l'Utilisateur peut former un recours documenté auprès de `support@designedtrust.com`. Le recours est traité sous 30 jours. En cas de refus définitif, l'Utilisateur reste libre d'utiliser le Service pour les fonctionnalités gratuites mais ne peut pas être destinataire d'un transfert financier.

---

## Article 5 — Cycle d'une transaction

### 5.1 Schéma général

1. **Publication** d'une mission par le Client.
2. **Proposal** émise par un Prestataire dans la conversation associée (objet, livrables, prix HT, délais, conditions).
3. **Acceptation** de la proposal par le Client : à compter de cet instant, la transaction est engagée.
4. **Provisionnement** du paiement par le Client (carte bancaire Stripe ou virement SEPA), placé en **séquestre** chez Stripe en attente de validation finale.
5. **Exécution** de la prestation par le Prestataire.
6. **Validation** par le Client (clic « livraison conforme ») OU expiration du délai d'opposition (cf. art. 6.2).
7. **Versement** au Prestataire net des commissions.
8. **Émission** de la facture / reçu et journalisation comptable.

### 5.2 Forme de la proposal

La proposal doit comporter :

- description précise des prestations attendues (livrables, périmètre) ;
- prix HT et TTC le cas échéant (TVA applicable selon situation fiscale du Prestataire) ;
- délai d'exécution et jalons si applicable ;
- conditions de validation et de réception ;
- éventuelle structure d'acompte / soldes ;
- mention de l'apporteur d'affaires (le cas échéant) avec accord exprès du Prestataire.

### 5.3 Valeur contractuelle

L'acceptation de la proposal vaut **contrat** entre le Client et le Prestataire au sens des articles 1101 et suivants du Code civil. La conversation et la proposal sont conservées pendant **10 ans** à des fins de preuve (Code de commerce art. L.123-22).

---

## Article 6 — Modalités de paiement

### 6.1 Provisionnement

Le Client provisionne intégralement la transaction au moment de l'acceptation. Le paiement est encaissé par **Stripe Connect** et conservé en séquestre jusqu'à la validation finale.

Moyens acceptés (selon la grille Stripe et la situation du Client) :

- carte bancaire (Visa, Mastercard, American Express selon Stripe) ;
- virement SEPA (`[À COMPLÉTER : seuils SEPA applicables]`) ;
- débit direct SEPA pour les contrats récurrents.

### 6.2 Validation et délai d'opposition

Après livraison annoncée par le Prestataire :

- le Client dispose de **7 jours calendaires** pour valider expressément la livraison ou formuler une opposition motivée ;
- à défaut, la validation est réputée acquise tacitement à l'issue du délai (sauf litige formalisé dans `/dashboard/disputes`).

### 6.3 Reversement au Prestataire

Le versement au Prestataire intervient dans un délai de **1 à 3 jours ouvrés** après validation, sur le compte bancaire associé à son compte Stripe Connect, net des commissions.

### 6.4 Acomptes et jalons

Pour les missions longues, la proposal peut prévoir des acomptes ou un échelonnement par jalons. Chaque jalon est traité comme une transaction indépendante (provisionnement, exécution, validation, reversement).

### 6.5 TVA

Le Prestataire est seul responsable de l'application de la TVA selon sa situation fiscale (assujetti, franchise en base, auto-entrepreneur). L'Éditeur ne fournit pas de conseil fiscal et invite chaque Utilisateur à consulter son expert-comptable.

Pour les opérations intra-UE, la validation du numéro de TVA via **VIES** (Commission européenne) est obligatoire avant l'application du régime d'autoliquidation.

---

## Article 7 — Litiges entre Utilisateurs et procédure de résolution

### 7.1 Résolution amiable

En cas de désaccord sur l'exécution d'un contrat, les Parties s'engagent à tenter une résolution amiable directe via la messagerie du Service avant toute escalade. Cette tentative doit faire l'objet d'au moins **un échange écrit** de chacune des Parties.

### 7.2 Ouverture d'un litige

À défaut de résolution amiable, l'une des Parties peut **ouvrir un litige formel** via `/dashboard/disputes`. Le litige est instruit par l'équipe support de l'Éditeur dans un délai cible de **15 jours ouvrés**.

### 7.3 Décision

Au terme de l'instruction, l'Éditeur peut :

- libérer le séquestre au profit du Prestataire (livraison conforme) ;
- rembourser tout ou partie au Client (livraison non conforme ou non-exécution) ;
- proposer un partage négocié ;
- inviter les Parties à une médiation conventionnelle.

La décision de l'Éditeur ne préjuge pas du droit pour chacune des Parties de saisir un tribunal compétent.

### 7.4 Médiation conventionnelle

Les Parties peuvent recourir à une médiation conventionnelle (`[À COMPLÉTER : référence médiateur]`). Les frais sont en principe partagés à parts égales.

---

## Article 8 — Apport d'affaires

### 8.1 Activation

Un Utilisateur disposant du toggle `referrer_enabled` peut être désigné apporteur d'affaires lors de la mise en relation entre un Client et un Prestataire.

### 8.2 Modalités

- L'apport est **formalisé dans la conversation initiale** entre le Client et le Prestataire, avec accord exprès des trois Parties.
- La commission d'apport est fixée par défaut à `[À COMPLÉTER : ex. 5 % HT]` du montant facturé HT après commission Plateforme, sauf accord particulier consigné dans la proposal.
- La commission est prélevée automatiquement au moment du reversement au Prestataire.
- L'apporteur d'affaires reçoit une facture détaillée mentionnant son rôle et la commission perçue.

### 8.3 Confidentialité

L'apporteur d'affaires **n'a pas accès à la conversation 1-1 entre le Client et le Prestataire** après la mise en relation initiale, sauf accord exprès des deux Parties principales (cf. `feedback_b2b_confidentiality.md`). Cette stricte séparation préserve la confidentialité des échanges commerciaux entre principaux.

### 8.4 Cessation

L'apporteur peut renoncer à sa commission à tout moment. La cessation du toggle `referrer_enabled` n'éteint pas les commissions déjà acquises sur des transactions en cours.

---

## Article 9 — Factures et obligations comptables

### 9.1 Émission

Pour chaque transaction validée, le Service émet automatiquement :

- une **facture du Prestataire vers le Client** mentionnant les éléments légaux (raison sociale, adresses, numéro SIRET, TVA si applicable, mention « TVA non applicable, article 293 B du CGI » pour les micro-entreprises, etc.) ;
- un **reçu de la Plateforme** mentionnant la commission perçue ;
- une **facture de l'apporteur d'affaires** s'il y a lieu.

### 9.2 Disponibilité

Les factures sont accessibles dans `/dashboard/billing` sous format PDF téléchargeable et exportable en bloc (CSV) pour la comptabilité.

### 9.3 Conservation

L'Éditeur conserve les factures pendant **10 ans** conformément à l'article L.123-22 du Code de commerce. Cette obligation prime sur le droit à l'effacement du RGPD pour les seules données comptables.

### 9.4 Obligations fiscales et déclaratives

Chaque Prestataire est seul responsable de :

- ses déclarations de TVA et impôts ;
- sa conformité au dispositif DAC 7 (Directive UE 2021/514) sur la déclaration des revenus tirés de plateformes numériques — l'Éditeur fournit annuellement à chaque Prestataire un **récapitulatif des revenus perçus** et **les communique aux autorités fiscales compétentes** conformément aux obligations légales.

---

## Article 10 — Remboursements et annulations

### 10.1 Annulation avant exécution

Tant que la prestation n'a pas commencé, le Client peut annuler la transaction et obtenir le remboursement intégral du provisionnement. La proposal peut prévoir des **frais d'annulation** ou un **acompte non remboursable**, qui sont alors prélevés conformément à ce qui a été convenu.

### 10.2 Annulation en cours d'exécution

L'annulation en cours d'exécution donne lieu à une négociation entre les Parties ou à un litige formel (cf. art. 7).

### 10.3 Mécanisme de remboursement

Tout remboursement transite par Stripe et est crédité sur le moyen de paiement initial. Les délais Stripe usuels (5-10 jours ouvrés) s'appliquent.

### 10.4 Frais Stripe

En cas de remboursement, les frais Stripe initialement perçus ne sont pas systématiquement restitués par Stripe. Les modalités sont précisées dans la grille Stripe en vigueur.

---

## Article 11 — Garanties et responsabilités

### 11.1 Garanties du Prestataire

Le Prestataire garantit qu'il dispose des qualifications, autorisations professionnelles et assurances nécessaires à l'exécution des prestations qu'il propose. Il s'engage à exécuter ses missions dans les règles de l'art et conformément aux engagements souscrits dans la proposal.

### 11.2 Garanties du Client

Le Client garantit qu'il dispose des autorisations nécessaires pour engager son organisation et que les fonds utilisés sont d'origine licite. Il s'engage à régler les sommes dues conformément à la proposal.

### 11.3 Responsabilité de l'Éditeur

L'Éditeur intervient en qualité d'intermédiaire technique de paiement. Il **ne garantit pas la qualité, le délai, la conformité ou la livraison des prestations** réalisées entre Utilisateurs. Sa responsabilité financière est limitée comme indiqué dans les CGU art. 9.

---

## Article 12 — Comptes inactifs et fonds dormants

Si un compte demeure inactif (aucune connexion, aucune transaction) pendant **24 mois** consécutifs, l'Éditeur peut :

- notifier l'Utilisateur par email d'une procédure de clôture imminente ;
- procéder à la clôture du compte sous 90 jours en l'absence de réponse ;
- restituer les fonds éventuels au profit de l'Utilisateur via Stripe, sous réserve de validation des données bancaires.

Les fonds non réclamés au-delà de **5 ans** sont reversés à la Caisse des Dépôts et Consignations conformément à la loi n° 2014-617 du 13 juin 2014 (dispositif Eckert).

---

## Article 13 — Évolution des CGV

L'Éditeur peut faire évoluer les présentes CGV. Toute modification substantielle (tarif, mécanisme de commission, mode de paiement, règles d'apport) est notifiée par email et bandeau in-app **au moins 30 jours avant** son entrée en vigueur. L'Utilisateur qui refuse la nouvelle version peut résilier son compte avant cette date.

---

## Article 14 — Loi applicable, juridiction et médiation

Les présentes CGV sont régies par le **droit français**. Tout litige relatif à leur interprétation, leur exécution ou leur fin sera, à défaut de résolution amiable préalable, porté devant les juridictions compétentes du ressort du siège social de l'Éditeur.

Les Utilisateurs professionnels peuvent recourir, à leur initiative, à une médiation conventionnelle. `[À COMPLÉTER : référence médiateur agréé]`

---

## Article 15 — Stipulations diverses

- **Convention de preuve** : les enregistrements informatiques (logs Stripe, audit log, journal de paiement) sont opposables et reconnus comme moyen de preuve au sens des articles 1366 et 1368 du Code civil.
- **Indivisibilité** : la nullité d'une clause n'affecte pas la validité des autres.
- **Non-renonciation** : le fait pour l'Éditeur de ne pas exercer un droit ne saurait être interprété comme une renonciation à ce droit.
- **Cession** : l'Utilisateur ne peut céder son compte. L'Éditeur peut transférer les présentes CGV en cas de cession d'activité, sous réserve d'une information préalable des Utilisateurs concernés.

---

**Date d'entrée en vigueur des présentes CGV :** `[À COMPLÉTER : date]`
**Validation juridique :** `[À COMPLÉTER : nom du conseil + date]`
**Référence comptable :** `[À COMPLÉTER : commissaire aux comptes + date]`
