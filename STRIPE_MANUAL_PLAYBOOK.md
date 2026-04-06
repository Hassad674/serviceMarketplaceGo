# Stripe Manual Test Playbook

Scenarios impossibles à automatiser (dépendent du comportement réel Stripe) —
à lancer manuellement avec Stripe CLI avant chaque release majeure.

## Prérequis

```bash
# Install Stripe CLI
brew install stripe/stripe-cli/stripe   # macOS
# or: scoop install stripe               # Windows
# or: download from https://stripe.com/docs/stripe-cli

# Login (one-time)
stripe login

# Forward webhooks to local backend
stripe listen --forward-to http://localhost:8084/api/v1/stripe/webhook
```

Laisse la commande `stripe listen` tourner dans un terminal dédié — elle va forwarder les events vers ton backend.

---

## Scénarios à tester

### ✅ 1. Account activated (premier succès)

**Setup** : créer un compte connecté via `/fr/payment-info-v2`, compléter le KYC.

**Trigger** :
```bash
# Stripe active automatiquement après KYC complet, pas besoin de trigger
# Mais tu peux forcer :
stripe trigger account.updated --account acct_XXXXX
```

**À vérifier** :
- [ ] Notification "Compte de paiement activé" reçue
- [ ] `AccountStatusCard` passe au vert "Compte entièrement actif"
- [ ] `charges_enabled` et `payouts_enabled` = true dans `/account-status`

---

### ✅ 2. Requirement ajouté post-activation

**Trigger** :
```bash
# Stripe peut ajouter un requirement spontanément (risk review, nouvelles règles)
stripe trigger account.updated --account acct_XXXXX \
  --override "requirements[currently_due][]=individual.verification.document"
```

**À vérifier** :
- [ ] Notification "Information requise" reçue in-app
- [ ] Email envoyé avec lien vers `/fr/payment-info-v2`
- [ ] Push notification (si configuré) avec deep-link
- [ ] `ConnectNotificationBanner` affiche bandeau orange
- [ ] Clic sur le bandeau → `AccountManagement` s'ouvre au bon champ

---

### ✅ 3. Document rejeté (expired)

**Trigger via magic value pendant onboarding** :
1. Dans l'iframe Stripe, upload un fichier nommé `failure_document_expired.png`
2. Stripe déclenche automatiquement `account.updated` avec l'erreur

**Vérifier** :
- [ ] Notification **"Document expiré"** (titre exact en FR)
- [ ] Body : "Le document fourni a expiré. Veuillez en fournir un valide."
- [ ] `requirements.errors[0].code = verification_document_expired`
- [ ] User peut ré-uploader via `AccountManagement`

**Autres magic values documents** :
| Filename | Erreur attendue | Notif title |
|----------|-----------------|-------------|
| `failure_document_expired.png` | expired | "Document expiré" |
| `failure_document_too_blurry.png` | blurry | "Document illisible" |
| `failure_document_name_mismatch.png` | name mismatch | "Informations du document non conformes" |
| `failure_document_fraudulent.png` | fraudulent | "Document refusé" |
| `failure_document_manipulated.png` | altered | "Document refusé" |
| `success_verified.png` | accepted | (aucune notif d'erreur) |

---

### ✅ 4. Account suspendu (past_due)

**Trigger** :
```bash
# Simuler que Stripe a passé un requirement en past_due (délai expiré)
stripe trigger account.updated --account acct_XXXXX \
  --override "requirements[past_due][]=individual.verification.document" \
  --override "requirements[disabled_reason]=requirements.past_due" \
  --override "charges_enabled=false" \
  --override "payouts_enabled=false"
```

**À vérifier** :
- [ ] 3-4 notifications reçues (urgence max) :
  - "Paiements entrants suspendus"
  - "Virements sortants suspendus"
  - "Action urgente — délai dépassé"
  - "Compte restreint par Stripe"
- [ ] `AccountStatusCard` passe au rouge
- [ ] User peut résoudre via `AccountManagement`

---

### ✅ 5. Capability désactivée

**Trigger** :
```bash
stripe trigger capability.updated --account acct_XXXXX \
  --override "capability=card_payments" \
  --override "status=inactive"
```

**À vérifier** :
- [ ] Notification "Paiements entrants suspendus" reçue
- [ ] `charges_enabled` passe à false dans `/account-status`

---

### ✅ 6. IBAN modifié par l'utilisateur

**Trigger manuel** :
1. Va sur `/fr/payment-info-v2`
2. Dans `AccountManagement`, clique sur "Changer compte bancaire"
3. Entre un nouvel IBAN de test : `FR1420041010050500013M02606`
4. Submit

**À vérifier** :
- [ ] Webhook `account.external_account.updated` reçu
- [ ] Nouveau RIB visible dans le dashboard Stripe
- [ ] Log backend : `account.external_account.updated` traité

---

### ✅ 7. UBO ajouté (pour compte company)

**Trigger manuel** :
1. `/fr/payment-info-v2` en mode company
2. `AccountManagement` → section Persons
3. Ajouter un nouveau bénéficiaire effectif (> 25% ownership)
4. Stripe peut demander sa pièce d'identité

**À vérifier** :
- [ ] Nouvelle Person créée dans Stripe
- [ ] `requirements.currently_due` contient `person_XXX.verification.document`
- [ ] Notification "Information requise" déclenchée

---

### ✅ 8. Pending verification (doc en review)

**Trigger** : upload un doc valide et attendre 1-5 minutes.

**À vérifier** :
- [ ] `requirements.pending_verification` contient l'item
- [ ] `AccountStatusCard` affiche "Traitement en cours"
- [ ] Email "Vérification en cours" envoyé (si configuré)

---

### ✅ 9. Compte rejeté par Stripe (fraud detection)

**Trigger** :
```bash
stripe trigger account.updated --account acct_XXXXX \
  --override "requirements[disabled_reason]=rejected.fraud" \
  --override "charges_enabled=false" \
  --override "payouts_enabled=false"
```

**À vérifier** :
- [ ] Notification "Compte restreint par Stripe" avec raison "suspicion de fraude"
- [ ] User peut contacter le support (pas résoudre eux-mêmes)

---

### ✅ 10. Compte réactivé après résolution

**Trigger** : résoudre tous les requirements via `AccountManagement` → attendre webhook.

**À vérifier** :
- [ ] Notification "Compte activé" reçue
- [ ] `charges_enabled` + `payouts_enabled` = true
- [ ] Bandeau orange disparaît

---

## Magic values reference

### DOB
- `1901-01-01` → déclenche verification_document extra
- `2020-01-01` → too_young error
- Date valide (ex: `1990-01-01`) → happy path

### SSN (US)
- `000000000` → verification réussie (full SSN)
- `000000001` → verification declined
- `000000002` → triggers retry
- Last 4 = `0000` → verification failed

### IBAN de test
- France : `FR1420041010050500013M02606`
- Germany : `DE89370400440532013000`
- UK : `GB29NWBK60161331926819`

### Cartes de test
- `4242 4242 4242 4242` → succès
- `4000 0000 0000 0002` → generic decline
- `4000 0000 0000 9995` → insufficient funds
- `4000 0025 0000 3155` → requires 3DS authentication

---

## Checklist avant release

Exécuter **les 10 scénarios** et cocher. Estimation : ~45 minutes pour la totalité.

- [ ] Scénario 1 — Account activated
- [ ] Scénario 2 — Requirement ajouté
- [ ] Scénario 3 — Document rejeté (× 5 magic values)
- [ ] Scénario 4 — Account suspendu (past_due)
- [ ] Scénario 5 — Capability désactivée
- [ ] Scénario 6 — IBAN modifié
- [ ] Scénario 7 — UBO ajouté
- [ ] Scénario 8 — Pending verification
- [ ] Scénario 9 — Compte rejeté (fraud)
- [ ] Scénario 10 — Compte réactivé

**Validation** : tous les scénarios produisent la notification attendue ET l'UI reflète correctement l'état.

---

## Debugging

Si une notification n'arrive pas :

```bash
# Vérifier les webhooks reçus
stripe listen --print-json

# Vérifier les logs backend
tail -f /tmp/embedded-backend.log | grep -E "ERROR|embedded|notif"

# Inspecter l'état d'un compte
curl -s -H "Authorization: Bearer $STRIPE_SECRET_KEY" \
  https://api.stripe.com/v1/accounts/acct_XXXXX | jq '.requirements'
```
