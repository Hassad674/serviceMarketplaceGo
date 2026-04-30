-- Phase 5 / area T — index FK columns that aren't backed by an index.
--
-- PostgreSQL does NOT auto-index foreign key columns. Without an
-- index on the child side:
--   - lookups by parent id (the typical "list child rows for a
--     parent" query) do a sequential scan;
--   - parent UPDATEs / DELETEs that propagate to the child also
--     sequential-scan to find rows to update.
--
-- The audit (auditperf.md PERF-B-04) flagged the missing indexes
-- below by cross-referencing pg_constraint (FK columns) with
-- pg_index (existing indexes). All entries in this migration are
-- on tables that did NOT already have a composite or single-column
-- index covering the FK.
--
-- Note on CONCURRENTLY: we use plain CREATE INDEX because
-- golang-migrate wraps every migration in a transaction block and
-- CONCURRENTLY is forbidden in a tx. The tables targeted here are
-- small enough that a brief ACCESS EXCLUSIVE lock during build is
-- acceptable. Future indexes on the heavy tables (messages,
-- payment_records, audit_logs) that need CONCURRENTLY must use
-- the manual workflow described in backend/migrations/README.md.
--
-- Each `IF NOT EXISTS` keeps the migration idempotent on re-runs.

-- counter_proposals.proposer_id — part of disputes feature, lookup by
-- the user who made the offer (e.g. "show my pending counter offers").
CREATE INDEX IF NOT EXISTS idx_counter_proposals_proposer_id
    ON counter_proposals (proposer_id);

-- dispute_evidence.uploader_id — accountability + filtering by uploader.
CREATE INDEX IF NOT EXISTS idx_dispute_evidence_uploader_id
    ON dispute_evidence (uploader_id);

-- dispute_ai_chat_messages.dispute_id — ordered chat retrieval by
-- dispute. Covers the typical "load messages for dispute X in order".
CREATE INDEX IF NOT EXISTS idx_dispute_ai_chat_messages_dispute_id
    ON dispute_ai_chat_messages (dispute_id, created_at);

-- proposal_milestones.proposal_id — ordered milestone retrieval.
-- Covers "list milestones for a proposal in order", which the
-- proposal repository hits on every detail-view fetch.
CREATE INDEX IF NOT EXISTS idx_proposal_milestones_proposal_id_seq
    ON proposal_milestones (proposal_id, sequence);

-- milestone_transitions.milestone_id — audit log of state changes
-- per milestone. Already has a (proposal_id, ...) index but milestone
-- lookup is the more common pattern after the milestone_id is in hand.
CREATE INDEX IF NOT EXISTS idx_milestone_transitions_milestone_id
    ON milestone_transitions (milestone_id, created_at);

-- conversation_read_state.conversation_id — when a conversation is
-- archived/deleted, cascading to the read state requires a scan
-- without this index.
CREATE INDEX IF NOT EXISTS idx_conversation_read_state_conversation_id
    ON conversation_read_state (conversation_id);

-- referral_commissions.referral_id — list commissions for a referral.
-- Tablescans risk explicit when a popular referral has many entries.
CREATE INDEX IF NOT EXISTS idx_referral_commissions_referral_id
    ON referral_commissions (referral_id, created_at);

-- referral_commissions.proposal_id — lookup by proposal (clawback
-- path). NULL when there is no proposal yet, partial index keeps
-- the index small.
CREATE INDEX IF NOT EXISTS idx_referral_commissions_proposal_id
    ON referral_commissions (proposal_id)
    WHERE proposal_id IS NOT NULL;

-- reports.message_id — admin moderation panel lookup of reports
-- targeting a message; partial since most reports are user-scope.
CREATE INDEX IF NOT EXISTS idx_reports_message_id
    ON reports (message_id)
    WHERE message_id IS NOT NULL;

-- reports.conversation_id — same idea for conversation-scoped reports.
CREATE INDEX IF NOT EXISTS idx_reports_conversation_id
    ON reports (conversation_id)
    WHERE conversation_id IS NOT NULL;

-- job_applications.applicant_user_id — list applications by a user.
CREATE INDEX IF NOT EXISTS idx_job_applications_applicant_user_id
    ON job_applications (applicant_user_id, created_at DESC);

-- organization_invitations.target_user_id — invitations addressed to
-- an existing user (vs email-only). Partial since most are email-only.
CREATE INDEX IF NOT EXISTS idx_organization_invitations_target_user_id
    ON organization_invitations (target_user_id)
    WHERE target_user_id IS NOT NULL;

-- organization_invitations.inviter_id — audit / "what did I invite?"
CREATE INDEX IF NOT EXISTS idx_organization_invitations_inviter_id
    ON organization_invitations (inviter_id);
