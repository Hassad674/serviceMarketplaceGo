ALTER TABLE payment_info
  ADD COLUMN IF NOT EXISTS charges_enabled BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS payouts_enabled BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS stripe_business_type TEXT,
  ADD COLUMN IF NOT EXISTS stripe_country TEXT,
  ADD COLUMN IF NOT EXISTS stripe_display_name TEXT;
