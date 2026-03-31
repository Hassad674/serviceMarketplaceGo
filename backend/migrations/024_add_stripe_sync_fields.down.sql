ALTER TABLE payment_info
  DROP COLUMN IF EXISTS charges_enabled,
  DROP COLUMN IF EXISTS payouts_enabled,
  DROP COLUMN IF EXISTS stripe_business_type,
  DROP COLUMN IF EXISTS stripe_country,
  DROP COLUMN IF EXISTS stripe_display_name;
