-- Down: unlink payment_records from their synthetic milestones by
-- clearing the column. This is safe to run before the constraint
-- tightening migration 093 has been applied.
UPDATE payment_records SET milestone_id = NULL;
