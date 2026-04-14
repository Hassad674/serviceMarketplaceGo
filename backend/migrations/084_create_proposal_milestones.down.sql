DROP INDEX IF EXISTS idx_proposal_milestones_submitted_auto_approve;
DROP INDEX IF EXISTS idx_proposal_milestones_status;
DROP INDEX IF EXISTS idx_proposal_milestones_proposal;
DROP TRIGGER IF EXISTS proposal_milestones_updated_at ON proposal_milestones;
DROP TABLE IF EXISTS proposal_milestones;
