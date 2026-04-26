-- Rollback du module invoicing. Dev-safety only — forward-only en prod.
-- Ordre : enfants → parents (FK), counter en dernier (pas de FK).
DROP TABLE IF EXISTS invoice_item;
DROP TABLE IF EXISTS credit_note;
DROP TABLE IF EXISTS invoice;
DROP TABLE IF EXISTS billing_profile;
DROP TABLE IF EXISTS invoice_number_counter;
