-- Drop the custom KYC tables replaced by Stripe Embedded Components.
-- All Stripe account data now lives on the users table (migration 040).
--
-- Order matters: business_persons + identity_documents reference users
-- (cascade on delete). payment_info is standalone. test_embedded_accounts
-- was a stop-gap used during the Embedded migration — now superseded by
-- users.stripe_account_id.

DROP TABLE IF EXISTS business_persons CASCADE;
DROP TABLE IF EXISTS identity_documents CASCADE;
DROP TABLE IF EXISTS payment_info CASCADE;
DROP TABLE IF EXISTS test_embedded_accounts CASCADE;
