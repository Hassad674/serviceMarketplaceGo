-- Down: restore the legacy UNIQUE(proposal_id) and revert milestone_id
-- back to nullable. Only safe in local development: if any
-- payment_record has been created via the new N:1 layout (multiple
-- rows per proposal), the ADD UNIQUE(proposal_id) will fail on
-- duplicates and the migration will abort.
ALTER TABLE payment_records
    DROP CONSTRAINT IF EXISTS payment_records_milestone_id_key;

ALTER TABLE payment_records
    ALTER COLUMN milestone_id DROP NOT NULL;

ALTER TABLE payment_records
    ADD CONSTRAINT payment_records_proposal_id_key UNIQUE (proposal_id);
