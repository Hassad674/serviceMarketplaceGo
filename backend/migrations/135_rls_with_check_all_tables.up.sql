-- 135_rls_with_check_all_tables.up.sql
--
-- F.5 S1 — RLS WITH CHECK on the 8 tenant-scoped tables that have only
-- a USING policy today.
--
-- Migration 125 enabled RLS on 9 tables. Migration 129 added an explicit
-- WITH CHECK on `audit_logs` (background workers write rows without a
-- tenant context, so WITH CHECK had to be `(true)`). The other 8 tables
-- still have only a USING policy. PostgreSQL defaults a missing
-- WITH CHECK to mirror USING — that means an INSERT or UPDATE has to
-- prove the NEW row matches the USING expression.
--
-- This is masked TODAY because the application database role has
-- BYPASSRLS (see backend/docs/rls.md "Production deployment recipe" —
-- the doc explicitly tracks the goal of rotating to a NOBYPASSRLS role
-- once the policies are battle-tested). Once that rotation lands, every
-- INSERT into the 8 tables would silently fail any row whose tenant
-- columns evaluate the USING expression to NULL or false in the
-- writer's context — for example a system worker writing a notification
-- to a user other than the actor, or the platform issuing an invoice
-- where current_setting('app.current_org_id') is unset.
--
-- The fix: make WITH CHECK explicit and restate the SAME predicate as
-- USING. INSERTs that pass the predicate succeed; INSERTs that don't
-- get rejected by the database — matching the application's own
-- repository-level authorization, so misalignment between the two
-- becomes visible at write time rather than silently leaking on read.
--
-- Tables covered (8):
--   1. conversations          — direct organization_id (or participant)
--   2. messages               — JOIN on conversations.organization_id
--   3. invoice                — recipient_organization_id
--   4. proposals              — client / provider organization_id
--   5. proposal_milestones    — inherited from proposals
--   6. notifications          — user_id (per-recipient)
--   7. disputes               — client / provider organization_id
--   8. payment_records        — organization_id
--
-- audit_logs is intentionally excluded (it has its own WITH CHECK (true)
-- from migration 129 — append-only system writers must be allowed
-- through regardless of context).

BEGIN;

-- ----------------------------------------------------------------------
-- 1. conversations
-- ----------------------------------------------------------------------
DROP POLICY IF EXISTS conversations_isolation ON conversations;

CREATE POLICY conversations_isolation ON conversations
    USING (
        organization_id = current_setting('app.current_org_id', true)::uuid
        OR EXISTS (
            SELECT 1
            FROM   conversation_participants cp
            WHERE  cp.conversation_id = conversations.id
              AND  cp.user_id = current_setting('app.current_user_id', true)::uuid
        )
    )
    WITH CHECK (
        organization_id = current_setting('app.current_org_id', true)::uuid
        OR EXISTS (
            SELECT 1
            FROM   conversation_participants cp
            WHERE  cp.conversation_id = conversations.id
              AND  cp.user_id = current_setting('app.current_user_id', true)::uuid
        )
    );

-- ----------------------------------------------------------------------
-- 2. messages
-- ----------------------------------------------------------------------
DROP POLICY IF EXISTS messages_isolation ON messages;

CREATE POLICY messages_isolation ON messages
    USING (
        EXISTS (
            SELECT 1
            FROM   conversations c
            WHERE  c.id = messages.conversation_id
              AND  (
                    c.organization_id = current_setting('app.current_org_id', true)::uuid
                    OR EXISTS (
                        SELECT 1
                        FROM   conversation_participants cp
                        WHERE  cp.conversation_id = c.id
                          AND  cp.user_id = current_setting('app.current_user_id', true)::uuid
                    )
              )
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1
            FROM   conversations c
            WHERE  c.id = messages.conversation_id
              AND  (
                    c.organization_id = current_setting('app.current_org_id', true)::uuid
                    OR EXISTS (
                        SELECT 1
                        FROM   conversation_participants cp
                        WHERE  cp.conversation_id = c.id
                          AND  cp.user_id = current_setting('app.current_user_id', true)::uuid
                    )
              )
        )
    );

-- ----------------------------------------------------------------------
-- 3. invoice
-- ----------------------------------------------------------------------
DROP POLICY IF EXISTS invoice_isolation ON invoice;

CREATE POLICY invoice_isolation ON invoice
    USING (
        recipient_organization_id = current_setting('app.current_org_id', true)::uuid
    )
    WITH CHECK (
        recipient_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- ----------------------------------------------------------------------
-- 4. proposals
-- ----------------------------------------------------------------------
DROP POLICY IF EXISTS proposals_isolation ON proposals;

CREATE POLICY proposals_isolation ON proposals
    USING (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    )
    WITH CHECK (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- ----------------------------------------------------------------------
-- 5. proposal_milestones
-- ----------------------------------------------------------------------
DROP POLICY IF EXISTS proposal_milestones_isolation ON proposal_milestones;

CREATE POLICY proposal_milestones_isolation ON proposal_milestones
    USING (
        EXISTS (
            SELECT 1
            FROM   proposals p
            WHERE  p.id = proposal_milestones.proposal_id
              AND  (
                    p.client_organization_id = current_setting('app.current_org_id', true)::uuid
                    OR p.provider_organization_id = current_setting('app.current_org_id', true)::uuid
              )
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1
            FROM   proposals p
            WHERE  p.id = proposal_milestones.proposal_id
              AND  (
                    p.client_organization_id = current_setting('app.current_org_id', true)::uuid
                    OR p.provider_organization_id = current_setting('app.current_org_id', true)::uuid
              )
        )
    );

-- ----------------------------------------------------------------------
-- 6. notifications
-- ----------------------------------------------------------------------
DROP POLICY IF EXISTS notifications_isolation ON notifications;

CREATE POLICY notifications_isolation ON notifications
    USING (
        user_id = current_setting('app.current_user_id', true)::uuid
    )
    WITH CHECK (
        user_id = current_setting('app.current_user_id', true)::uuid
    );

-- ----------------------------------------------------------------------
-- 7. disputes
-- ----------------------------------------------------------------------
DROP POLICY IF EXISTS disputes_isolation ON disputes;

CREATE POLICY disputes_isolation ON disputes
    USING (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    )
    WITH CHECK (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- ----------------------------------------------------------------------
-- 8. payment_records
-- ----------------------------------------------------------------------
DROP POLICY IF EXISTS payment_records_isolation ON payment_records;

CREATE POLICY payment_records_isolation ON payment_records
    USING (
        organization_id = current_setting('app.current_org_id', true)::uuid
    )
    WITH CHECK (
        organization_id = current_setting('app.current_org_id', true)::uuid
    );

COMMIT;
