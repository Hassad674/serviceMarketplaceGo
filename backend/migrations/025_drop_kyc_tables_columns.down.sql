-- Re-add dropped columns to payment_info
ALTER TABLE payment_info
  ADD COLUMN IF NOT EXISTS first_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS date_of_birth TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01',
  ADD COLUMN IF NOT EXISTS nationality TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS address TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS city TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS postal_code TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS is_business BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS business_name TEXT,
  ADD COLUMN IF NOT EXISTS business_address TEXT,
  ADD COLUMN IF NOT EXISTS business_city TEXT,
  ADD COLUMN IF NOT EXISTS business_postal_code TEXT,
  ADD COLUMN IF NOT EXISTS business_country TEXT,
  ADD COLUMN IF NOT EXISTS tax_id TEXT,
  ADD COLUMN IF NOT EXISTS vat_number TEXT,
  ADD COLUMN IF NOT EXISTS role_in_company TEXT,
  ADD COLUMN IF NOT EXISTS phone TEXT,
  ADD COLUMN IF NOT EXISTS activity_sector TEXT,
  ADD COLUMN IF NOT EXISTS is_self_representative BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS is_self_director BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS no_major_owners BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS is_self_executive BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS iban TEXT,
  ADD COLUMN IF NOT EXISTS bic TEXT,
  ADD COLUMN IF NOT EXISTS account_number TEXT,
  ADD COLUMN IF NOT EXISTS routing_number TEXT,
  ADD COLUMN IF NOT EXISTS account_holder TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS bank_country TEXT;

-- Re-create identity_documents table
CREATE TABLE IF NOT EXISTS identity_documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  category TEXT NOT NULL,
  document_type TEXT NOT NULL,
  side TEXT NOT NULL,
  file_key TEXT NOT NULL,
  stripe_file_id TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  rejection_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_identity_documents_user_id ON identity_documents(user_id);

-- Re-create business_persons table
CREATE TABLE IF NOT EXISTS business_persons (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  date_of_birth TIMESTAMPTZ,
  email TEXT,
  phone TEXT,
  address TEXT,
  city TEXT,
  postal_code TEXT,
  title TEXT,
  stripe_person_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_business_persons_user_id ON business_persons(user_id);
