-- Reverse of 144_create_automated_decision_appeals.up.sql

BEGIN;

DROP INDEX IF EXISTS idx_automated_decision_appeals_status_created_at;
DROP INDEX IF EXISTS idx_automated_decision_appeals_user_id;
DROP TABLE IF EXISTS automated_decision_appeals;

COMMIT;
