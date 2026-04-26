-- Module invoicing — factures émises par la marketplace au prestataire.
-- Stripe est PSP only ; toute la facturation légale (numérotation continue,
-- mentions FR / UE B2B / hors UE, immutabilité, archivage 10 ans) vit ici.
--
-- 5 tables :
--   * billing_profile          — infos destinataire (FK org), pré-rempli depuis
--                                Stripe KYC + complété par le user
--   * invoice                  — facture émise (immutable une fois finalized)
--   * invoice_item             — lignes (1+ par invoice, N pour la consolidation
--                                mensuelle des frais transaction)
--   * credit_note              — avoir, séquence séparée (AV-NNNNNN)
--   * invoice_number_counter   — compteur atomique scopé (invoice / credit_note)
--
-- Toutes les tables sont scopées à organization_id (jamais user_id) — règle
-- CLAUDE.md "business state belongs to organizations".

-- ---------------------------------------------------------------------------
-- billing_profile : 1 ligne par organisation, pré-remplie via Stripe KYC,
-- complétée par l'utilisateur. C'est la source des `recipient_snapshot`
-- figés au moment de l'émission.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS billing_profile (
    organization_id           UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,

    -- Profil
    profile_type              TEXT NOT NULL DEFAULT 'individual'
                              CHECK (profile_type IN ('individual', 'business')),
    legal_name                TEXT NOT NULL DEFAULT '',
    trading_name              TEXT NOT NULL DEFAULT '',
    legal_form                TEXT NOT NULL DEFAULT '', -- "EI", "SARL", "SAS", "Ltd", etc.

    -- Identifiants fiscaux
    tax_id                    TEXT NOT NULL DEFAULT '', -- SIRET pour FR (14), tax ID local sinon
    vat_number                TEXT NOT NULL DEFAULT '', -- n° TVA intracom UE (FR12345678901, DE…)
    vat_validated_at          TIMESTAMPTZ,
    vat_validation_payload    JSONB, -- preuve VIES horodatée pour audit

    -- Adresse postale
    address_line1             TEXT NOT NULL DEFAULT '',
    address_line2             TEXT NOT NULL DEFAULT '',
    postal_code               TEXT NOT NULL DEFAULT '',
    city                      TEXT NOT NULL DEFAULT '',
    country                   TEXT NOT NULL DEFAULT '', -- ISO alpha-2

    -- Contact facturation
    invoicing_email           TEXT NOT NULL DEFAULT '',

    -- Sync depuis Stripe KYC. NULL si jamais syncé. Le service s'abstient
    -- d'écraser les champs déjà éditados par l'user (heuristique : le merge
    -- ne touche que les champs vides).
    synced_from_kyc_at        TIMESTAMPTZ,

    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER billing_profile_updated_at
    BEFORE UPDATE ON billing_profile
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- VAT validation lookups par numéro pour le warmup du cache + l'admin.
CREATE INDEX idx_billing_profile_vat ON billing_profile(vat_number)
    WHERE vat_number <> '';

-- ---------------------------------------------------------------------------
-- invoice : facture émise. Immutable dès finalized_at != NULL.
--   * `recipient_snapshot` et `issuer_snapshot` figent l'état au moment de
--     l'émission. Si l'org change d'adresse 2 ans plus tard, les anciennes
--     factures gardent l'adresse de l'époque (obligation légale).
--   * `mentions_rendered` stocke la liste exacte des phrases légales qui ont
--     été rendues sur le PDF — auditable.
--   * `tax_regime` enum est la décision déterministe au moment de l'émission.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS invoice (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number                      TEXT NOT NULL UNIQUE, -- "FAC-000123"

    recipient_organization_id   UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    recipient_snapshot          JSONB NOT NULL, -- snapshot du billing_profile à l'émission
    issuer_snapshot             JSONB NOT NULL, -- snapshot des INVOICE_ISSUER_* env vars

    issued_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    service_period_start        TIMESTAMPTZ NOT NULL, -- début de la prestation facturée
    service_period_end          TIMESTAMPTZ NOT NULL, -- fin de la prestation

    currency                    TEXT NOT NULL DEFAULT 'EUR',
    amount_excl_tax_cents       BIGINT NOT NULL,
    vat_rate                    NUMERIC(5,2) NOT NULL DEFAULT 0,
    vat_amount_cents            BIGINT NOT NULL DEFAULT 0,
    amount_incl_tax_cents       BIGINT NOT NULL,

    tax_regime                  TEXT NOT NULL CHECK (tax_regime IN (
        'fr_franchise_base',     -- FR — auto-entrepreneur en franchise
        'eu_reverse_charge',     -- B2B UE — autoliquidation
        'out_of_scope_eu'        -- hors UE — art. 259-1 CGI
    )),
    mentions_rendered           TEXT[] NOT NULL DEFAULT '{}',

    -- Trace de l'origine. `subscription` = abo Stripe ; `monthly_commission` =
    -- consolidation mensuelle des frais sur transactions.
    source_type                 TEXT NOT NULL CHECK (source_type IN (
        'subscription', 'monthly_commission'
    )),
    -- Lien webhook pour idempotence ; UNIQUE pour s'assurer qu'on ne génère
    -- jamais deux factures pour le même évènement Stripe (NULL OK pour les
    -- factures issues du batch mensuel qui n'ont pas d'event id).
    stripe_event_id             TEXT UNIQUE,
    -- Trace facultative — utile pour les avoirs sur refund
    stripe_payment_intent_id    TEXT,
    stripe_invoice_id           TEXT, -- pour les abos (Stripe émet sa propre invoice côté abo)

    pdf_r2_key                  TEXT, -- emplacement du PDF dans le bucket R2
    status                      TEXT NOT NULL DEFAULT 'draft' CHECK (status IN (
        'draft',          -- créée, PDF pas encore généré
        'issued',         -- finalized_at != NULL, PDF en R2, email envoyé
        'credited'        -- annulée par un avoir
    )),
    finalized_at                TIMESTAMPTZ, -- une fois set, la ligne est read-only

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER invoice_updated_at
    BEFORE UPDATE ON invoice
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Lookups standards : par org (page "Mes factures" paginée), par event Stripe
-- (idempotence webhook), par PaymentIntent (lookup avoir sur refund).
CREATE INDEX idx_invoice_recipient_issued ON invoice(recipient_organization_id, issued_at DESC, id DESC);
CREATE INDEX idx_invoice_payment_intent ON invoice(stripe_payment_intent_id)
    WHERE stripe_payment_intent_id IS NOT NULL;
CREATE INDEX idx_invoice_source_period ON invoice(source_type, service_period_start);

-- ---------------------------------------------------------------------------
-- invoice_item : 1+ lignes par invoice. Pour les abos Premium = 1 ligne. Pour
-- la consolidation mensuelle = N lignes (1 par milestone libéré dans le mois).
--   * milestone_id et payment_record_id sont nullables (toujours NULL pour
--     les abos), permettent le rapprochement et les avoirs ciblés.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS invoice_item (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id          UUID NOT NULL REFERENCES invoice(id) ON DELETE CASCADE,

    description         TEXT NOT NULL, -- "Premium Agence — avril 2026" / "Commission mission XYZ"
    quantity            NUMERIC(10,2) NOT NULL DEFAULT 1,
    unit_price_cents    BIGINT NOT NULL,
    amount_cents        BIGINT NOT NULL, -- quantity * unit_price

    -- Liens optionnels pour traçabilité + idempotence du batch mensuel.
    -- Le batch refuse de re-facturer un payment_record déjà couvert.
    milestone_id        UUID REFERENCES proposal_milestones(id) ON DELETE SET NULL,
    payment_record_id   UUID REFERENCES payment_records(id) ON DELETE SET NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Idempotence batch mensuel : pas deux items pour le même payment_record.
CREATE UNIQUE INDEX idx_invoice_item_payment_record_unique
    ON invoice_item(payment_record_id)
    WHERE payment_record_id IS NOT NULL;
CREATE INDEX idx_invoice_item_invoice ON invoice_item(invoice_id);

-- ---------------------------------------------------------------------------
-- credit_note : avoir — émis sur refund Stripe ou correction admin.
-- Structure quasi identique à invoice, séquence séparée (AV-NNNNNN).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS credit_note (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number                      TEXT NOT NULL UNIQUE, -- "AV-000045"

    original_invoice_id         UUID NOT NULL REFERENCES invoice(id) ON DELETE RESTRICT,
    recipient_organization_id   UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    recipient_snapshot          JSONB NOT NULL,
    issuer_snapshot             JSONB NOT NULL,

    issued_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason                      TEXT NOT NULL DEFAULT '',

    currency                    TEXT NOT NULL DEFAULT 'EUR',
    amount_excl_tax_cents       BIGINT NOT NULL, -- positif : c'est le montant CRÉDITÉ
    vat_rate                    NUMERIC(5,2) NOT NULL DEFAULT 0,
    vat_amount_cents            BIGINT NOT NULL DEFAULT 0,
    amount_incl_tax_cents       BIGINT NOT NULL,

    tax_regime                  TEXT NOT NULL,
    mentions_rendered           TEXT[] NOT NULL DEFAULT '{}',

    stripe_event_id             TEXT UNIQUE, -- charge.refunded event id
    stripe_refund_id            TEXT,

    pdf_r2_key                  TEXT,
    finalized_at                TIMESTAMPTZ,

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER credit_note_updated_at
    BEFORE UPDATE ON credit_note
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_credit_note_recipient_issued ON credit_note(recipient_organization_id, issued_at DESC, id DESC);
CREATE INDEX idx_credit_note_original ON credit_note(original_invoice_id);

-- ---------------------------------------------------------------------------
-- invoice_number_counter : compteur atomique. Une ligne par scope.
--   * scope = 'invoice'    → FAC-NNNNNN
--   * scope = 'credit_note'→ AV-NNNNNN
--   * year  = 0 ⇒ continu à vie (politique actuelle), évolutif vers reset annuel
--     en passant year=YYYY et en élargissant la PK.
-- L'app layer fait `SELECT next_value FOR UPDATE ; UPDATE next_value+1 ;
-- INSERT invoice` dans la même transaction → zéro doublon, zéro trou (sauf
-- rollback explicite, ce qui est OK pour drafts qui ne finalisent pas).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS invoice_number_counter (
    scope        TEXT NOT NULL,
    year         INT NOT NULL DEFAULT 0,
    next_value   BIGINT NOT NULL DEFAULT 1,
    PRIMARY KEY (scope, year)
);

-- Seed initial : on commence à 1 pour les deux séquences.
INSERT INTO invoice_number_counter (scope, year, next_value) VALUES
    ('invoice', 0, 1),
    ('credit_note', 0, 1)
ON CONFLICT (scope, year) DO NOTHING;
