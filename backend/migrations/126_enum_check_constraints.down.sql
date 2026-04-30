-- Rollback for 126_enum_check_constraints.up.sql.
--
-- Drops every CHECK constraint added by the up migration. Uses
-- IF EXISTS so this is safe to re-run on a partially rolled-back
-- environment.

BEGIN;

ALTER TABLE proposals                   DROP CONSTRAINT IF EXISTS proposals_status_check;
ALTER TABLE disputes                    DROP CONSTRAINT IF EXISTS disputes_status_check;
ALTER TABLE counter_proposals           DROP CONSTRAINT IF EXISTS counter_proposals_status_check;
ALTER TABLE payment_records             DROP CONSTRAINT IF EXISTS payment_records_payment_status_check;
ALTER TABLE payment_records             DROP CONSTRAINT IF EXISTS payment_records_transfer_status_check;
ALTER TABLE proposal_milestones         DROP CONSTRAINT IF EXISTS proposal_milestones_status_check;
ALTER TABLE jobs                        DROP CONSTRAINT IF EXISTS jobs_status_check;
ALTER TABLE subscriptions               DROP CONSTRAINT IF EXISTS subscriptions_status_check;
ALTER TABLE referrals                   DROP CONSTRAINT IF EXISTS referrals_status_check;
ALTER TABLE referral_commissions        DROP CONSTRAINT IF EXISTS referral_commissions_status_check;
ALTER TABLE reports                     DROP CONSTRAINT IF EXISTS reports_status_check;
ALTER TABLE messages                    DROP CONSTRAINT IF EXISTS messages_status_check;
ALTER TABLE users                       DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE organization_invitations    DROP CONSTRAINT IF EXISTS organization_invitations_status_check;
ALTER TABLE invoices                    DROP CONSTRAINT IF EXISTS invoices_status_check;

COMMIT;
