-- 125_enable_row_level_security.down.sql
--
-- Reverses migration 125: drops every RLS policy and disables RLS on
-- the 9 tenant-scoped tables. Run only on disposable databases — the
-- shared production DB must never have RLS rolled back (it is a hard
-- security regression).

BEGIN;

DROP POLICY IF EXISTS payment_records_isolation       ON payment_records;
ALTER TABLE payment_records       NO FORCE ROW LEVEL SECURITY;
ALTER TABLE payment_records       DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS audit_logs_isolation            ON audit_logs;
ALTER TABLE audit_logs            NO FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_logs            DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS disputes_isolation              ON disputes;
ALTER TABLE disputes              NO FORCE ROW LEVEL SECURITY;
ALTER TABLE disputes              DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS notifications_isolation         ON notifications;
ALTER TABLE notifications         NO FORCE ROW LEVEL SECURITY;
ALTER TABLE notifications         DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS proposal_milestones_isolation   ON proposal_milestones;
ALTER TABLE proposal_milestones   NO FORCE ROW LEVEL SECURITY;
ALTER TABLE proposal_milestones   DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS proposals_isolation             ON proposals;
ALTER TABLE proposals             NO FORCE ROW LEVEL SECURITY;
ALTER TABLE proposals             DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS invoice_isolation               ON invoice;
ALTER TABLE invoice               NO FORCE ROW LEVEL SECURITY;
ALTER TABLE invoice               DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS messages_isolation              ON messages;
ALTER TABLE messages              NO FORCE ROW LEVEL SECURITY;
ALTER TABLE messages              DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS conversations_isolation         ON conversations;
ALTER TABLE conversations         NO FORCE ROW LEVEL SECURITY;
ALTER TABLE conversations         DISABLE ROW LEVEL SECURITY;

COMMIT;
