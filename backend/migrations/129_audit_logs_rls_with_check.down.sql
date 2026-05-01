-- Restore the original migration-125 audit_logs_isolation policy
-- (USING only, no explicit WITH CHECK).
BEGIN;

DROP POLICY IF EXISTS audit_logs_isolation ON audit_logs;

CREATE POLICY audit_logs_isolation ON audit_logs
    USING (
        user_id = current_setting('app.current_user_id', true)::uuid
    );

COMMIT;
