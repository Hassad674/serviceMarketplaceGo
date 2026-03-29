DROP TABLE IF EXISTS business_persons;
ALTER TABLE payment_info
    DROP COLUMN IF EXISTS is_self_representative,
    DROP COLUMN IF EXISTS is_self_director,
    DROP COLUMN IF EXISTS no_major_owners,
    DROP COLUMN IF EXISTS is_self_executive;
