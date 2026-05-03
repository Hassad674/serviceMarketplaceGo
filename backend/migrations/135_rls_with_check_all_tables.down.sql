-- Restore the migration-125 policies (USING only, no explicit WITH CHECK).
-- This rolls back the symmetric WITH CHECK clauses added in 135.up.

BEGIN;

-- 1. conversations
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
    );

-- 2. messages
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
    );

-- 3. invoice
DROP POLICY IF EXISTS invoice_isolation ON invoice;
CREATE POLICY invoice_isolation ON invoice
    USING (
        recipient_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- 4. proposals
DROP POLICY IF EXISTS proposals_isolation ON proposals;
CREATE POLICY proposals_isolation ON proposals
    USING (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- 5. proposal_milestones
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
    );

-- 6. notifications
DROP POLICY IF EXISTS notifications_isolation ON notifications;
CREATE POLICY notifications_isolation ON notifications
    USING (
        user_id = current_setting('app.current_user_id', true)::uuid
    );

-- 7. disputes
DROP POLICY IF EXISTS disputes_isolation ON disputes;
CREATE POLICY disputes_isolation ON disputes
    USING (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- 8. payment_records
DROP POLICY IF EXISTS payment_records_isolation ON payment_records;
CREATE POLICY payment_records_isolation ON payment_records
    USING (
        organization_id = current_setting('app.current_org_id', true)::uuid
    );

COMMIT;
