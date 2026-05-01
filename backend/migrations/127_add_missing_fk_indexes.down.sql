-- Rollback for 127_add_missing_fk_indexes.up.sql.

DROP INDEX IF EXISTS idx_dispute_counter_proposals_proposer_id;
DROP INDEX IF EXISTS idx_dispute_evidence_uploader_id;
DROP INDEX IF EXISTS idx_dispute_ai_chat_messages_dispute_id;
DROP INDEX IF EXISTS idx_proposal_milestones_proposal_id_seq;
DROP INDEX IF EXISTS idx_milestone_transitions_milestone_id;
DROP INDEX IF EXISTS idx_conversation_read_state_conversation_id;
DROP INDEX IF EXISTS ;
DROP INDEX IF EXISTS ;
DROP INDEX IF EXISTS ;
DROP INDEX IF EXISTS idx_reports_conversation_id;
DROP INDEX IF EXISTS idx_job_applications_applicant_id;
DROP INDEX IF EXISTS ;
DROP INDEX IF EXISTS idx_organization_invitations_invited_by_user_id;
