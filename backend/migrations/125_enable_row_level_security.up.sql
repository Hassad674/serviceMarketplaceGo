-- 125_enable_row_level_security.up.sql
--
-- Phase 5 Agent Q — SEC-10 fix: enable PostgreSQL Row-Level Security on
-- the 9 tenant-scoped tables that hold business state.
--
-- RLS is the BACKUP defense layer behind the application-level
-- WHERE org_id = $1 / WHERE user_id = $1 filters. With RLS enabled, a
-- single missed filter in repository code can no longer leak
-- another tenant's rows — Postgres itself rejects rows that do not
-- match the policy.
--
-- Policy pattern:
--   * Every policy uses current_setting('app.current_org_id', true)
--     (or 'app.current_user_id' for per-user tables). The "true" arg
--     makes current_setting return NULL when the setting is unset,
--     which causes the USING expression to evaluate to NULL → row is
--     filtered out. This is the SAFE DEFAULT: forgetting to call
--     SetCurrentOrg/SetCurrentUser denies access rather than granting
--     access.
--
--   * FORCE ROW LEVEL SECURITY is added so the table OWNER does not
--     bypass the policy. Without FORCE, the migration user (who owns
--     the table) sees everything, which would defeat the test setup.
--     The application database user must NOT be a superuser AND must
--     NOT own the tables — see backend/docs/rls.md for the prod
--     deployment recipe.
--
-- Tables covered (9):
--   1. messages                — via JOIN on conversations.organization_id
--   2. conversations           — direct organization_id (denormalised)
--   3. invoice                 — recipient_organization_id
--   4. proposals               — client_organization_id OR provider_organization_id
--   5. proposal_milestones     — inherited from proposals via JOIN
--   6. notifications           — user_id (per-recipient, NOT org-scoped)
--   7. disputes                — client_organization_id OR provider_organization_id
--   8. audit_logs              — user_id (per-actor, NOT org-scoped, append-only already from 124)
--   9. payment_records         — organization_id
--
-- The 'users' table is INTENTIONALLY excluded — auth flows (login,
-- register, password reset) run BEFORE the user/org context is
-- established, so an RLS policy keyed on app.current_user_id would
-- block these flows entirely. The same exclusion is documented in
-- backend/CLAUDE.md.

BEGIN;

-- ---------------------------------------------------------------------------
-- 1. conversations — direct organization_id column.
--
-- Conversations between two providers (no org) keep organization_id NULL.
-- The policy admits these rows ONLY when the tenant context is also
-- unset — but since we want providers to see their own conversations,
-- we add a dedicated escape hatch via app.current_user_id matching ANY
-- participant. For the pure org-membership check, NULL conversations
-- are denied to authenticated org users (they would never match any
-- legitimate query anyway).
-- ---------------------------------------------------------------------------
ALTER TABLE conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversations FORCE ROW LEVEL SECURITY;

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

-- ---------------------------------------------------------------------------
-- 2. messages — no direct org column. The owning conversation has
-- organization_id, AND every message is sent by a user who is either a
-- participant of the conversation or a member of the conversation's
-- org. The policy admits rows whose conversation belongs to the
-- caller's org OR whose conversation has the caller as a participant.
-- This matches the existing application-level access rule.
-- ---------------------------------------------------------------------------
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages FORCE ROW LEVEL SECURITY;

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

-- ---------------------------------------------------------------------------
-- 3. invoice — recipient_organization_id.
-- Single-side ownership: the recipient org sees its own invoices. The
-- platform issuer is a static label (env vars), not an org row.
-- ---------------------------------------------------------------------------
ALTER TABLE invoice ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoice FORCE ROW LEVEL SECURITY;

CREATE POLICY invoice_isolation ON invoice
    USING (
        recipient_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- ---------------------------------------------------------------------------
-- 4. proposals — two-sided ownership.
-- Both the client side and the provider side see the proposal: both
-- orgs are stakeholders in the same business transaction.
-- ---------------------------------------------------------------------------
ALTER TABLE proposals ENABLE ROW LEVEL SECURITY;
ALTER TABLE proposals FORCE ROW LEVEL SECURITY;

CREATE POLICY proposals_isolation ON proposals
    USING (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- ---------------------------------------------------------------------------
-- 5. proposal_milestones — inherits from proposals via FK JOIN.
-- The milestone is visible to the same orgs that can see the parent
-- proposal. Cheaper than denormalising both org ids onto every
-- milestone row.
-- ---------------------------------------------------------------------------
ALTER TABLE proposal_milestones ENABLE ROW LEVEL SECURITY;
ALTER TABLE proposal_milestones FORCE ROW LEVEL SECURITY;

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

-- ---------------------------------------------------------------------------
-- 6. notifications — per-user (recipient).
-- Notifications are scoped to a single user, not an org. A solo
-- provider has no org but still receives notifications, so the policy
-- keys on app.current_user_id (NOT app.current_org_id).
-- ---------------------------------------------------------------------------
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications FORCE ROW LEVEL SECURITY;

CREATE POLICY notifications_isolation ON notifications
    USING (
        user_id = current_setting('app.current_user_id', true)::uuid
    );

-- ---------------------------------------------------------------------------
-- 7. disputes — two-sided ownership.
-- Both client and provider orgs see the dispute (just like proposals).
-- ---------------------------------------------------------------------------
ALTER TABLE disputes ENABLE ROW LEVEL SECURITY;
ALTER TABLE disputes FORCE ROW LEVEL SECURITY;

CREATE POLICY disputes_isolation ON disputes
    USING (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- ---------------------------------------------------------------------------
-- 8. audit_logs — per-actor (user_id).
-- Audit rows are forensic records of an actor's actions. Each user
-- sees only their own audit trail. Admin endpoints use a separate
-- privileged path that bypasses RLS via a dedicated DB role (out of
-- scope for this round — tracked as follow-up).
--
-- Note: append-only enforcement (REVOKE UPDATE, DELETE) is already
-- in migration 124. RLS adds the per-tenant read filter on top.
-- ---------------------------------------------------------------------------
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;

CREATE POLICY audit_logs_isolation ON audit_logs
    USING (
        user_id = current_setting('app.current_user_id', true)::uuid
    );

-- ---------------------------------------------------------------------------
-- 9. payment_records — organization_id (single-side: the client org).
-- Payment record ownership is scoped to the org that paid. The
-- provider's view of money received goes through the proposal/
-- milestone path, which is already tenant-isolated.
-- ---------------------------------------------------------------------------
ALTER TABLE payment_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_records FORCE ROW LEVEL SECURITY;

CREATE POLICY payment_records_isolation ON payment_records
    USING (
        organization_id = current_setting('app.current_org_id', true)::uuid
    );

COMMIT;
