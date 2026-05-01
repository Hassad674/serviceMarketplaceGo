-- Phase 5 / area T — DB-level CHECK constraints on enum-style TEXT columns.
--
-- Until now the application validated enum values via Go domain
-- methods (`IsValid()`), but the DB happily accepted anything. A
-- typo in a hand-written SQL UPDATE or a bug in a future repository
-- could write garbage. CHECK constraints catch those at write-time
-- and produce a recognisable error code (23514) callers can map.
--
-- Each `IF NOT EXISTS` guard keeps the migration idempotent on a DB
-- that already has the constraint. The `pg_constraint` lookup below
-- is the standard PG idiom for "create constraint if not present".

BEGIN;

-- proposals.status — values from internal/domain/proposal/entity.go
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'proposals_status_check'
    ) THEN
        ALTER TABLE proposals
            ADD CONSTRAINT proposals_status_check
            CHECK (status IN (
                'pending', 'accepted', 'declined', 'withdrawn',
                'paid', 'active', 'completion_requested',
                'completed', 'disputed'
            ));
    END IF;
END $$;

-- disputes.status — values from internal/domain/dispute/entity.go
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'disputes_status_check'
    ) THEN
        ALTER TABLE disputes
            ADD CONSTRAINT disputes_status_check
            CHECK (status IN (
                'open', 'negotiation', 'escalated', 'resolved', 'cancelled'
            ));
    END IF;
END $$;

-- dispute_counter_proposals.status (in disputes feature)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'dispute_counter_proposals_status_check'
    ) THEN
        ALTER TABLE dispute_counter_proposals
            ADD CONSTRAINT dispute_counter_proposals_status_check
            CHECK (status IN (
                'pending', 'accepted', 'rejected', 'superseded'
            ));
    END IF;
END $$;

-- payment_records.payment_status — values from internal/domain/payment/payment_record.go
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'payment_records_status_check'
    ) THEN
        ALTER TABLE payment_records
            ADD CONSTRAINT payment_records_status_check
            CHECK (status IN (
                'pending', 'succeeded', 'failed', 'refunded'
            ));
    END IF;
END $$;

-- payment_records.transfer_status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'payment_records_transfer_status_check'
    ) THEN
        ALTER TABLE payment_records
            ADD CONSTRAINT payment_records_transfer_status_check
            CHECK (transfer_status IN (
                'pending', 'completed', 'failed'
            ));
    END IF;
END $$;

-- proposal_milestones.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'proposal_milestones_status_check'
    ) THEN
        ALTER TABLE proposal_milestones
            ADD CONSTRAINT proposal_milestones_status_check
            CHECK (status IN (
                'pending_funding', 'funded', 'submitted', 'approved',
                'released', 'disputed', 'cancelled', 'refunded'
            ));
    END IF;
END $$;

-- jobs.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'jobs_status_check'
    ) THEN
        ALTER TABLE jobs
            ADD CONSTRAINT jobs_status_check
            CHECK (status IN ('open', 'closed'));
    END IF;
END $$;

-- subscriptions.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'subscriptions_status_check'
    ) THEN
        ALTER TABLE subscriptions
            ADD CONSTRAINT subscriptions_status_check
            CHECK (status IN (
                'incomplete', 'active', 'past_due', 'canceled', 'unpaid'
            ));
    END IF;
END $$;

-- referrals.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'referrals_status_check'
    ) THEN
        ALTER TABLE referrals
            ADD CONSTRAINT referrals_status_check
            CHECK (status IN (
                'pending_provider', 'pending_referrer', 'pending_client',
                'active', 'rejected', 'expired', 'cancelled', 'terminated'
            ));
    END IF;
END $$;

-- referral_commissions.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'referral_commissions_status_check'
    ) THEN
        ALTER TABLE referral_commissions
            ADD CONSTRAINT referral_commissions_status_check
            CHECK (status IN (
                'pending', 'pending_kyc', 'paid', 'failed',
                'cancelled', 'clawed_back'
            ));
    END IF;
END $$;

-- reports.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'reports_status_check'
    ) THEN
        ALTER TABLE reports
            ADD CONSTRAINT reports_status_check
            CHECK (status IN (
                'pending', 'reviewed', 'resolved', 'dismissed'
            ));
    END IF;
END $$;

-- messages.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'messages_status_check'
    ) THEN
        ALTER TABLE messages
            ADD CONSTRAINT messages_status_check
            CHECK (status IN ('sent', 'delivered', 'read'));
    END IF;
END $$;

-- users.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_status_check'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_status_check
            CHECK (status IN ('active', 'suspended', 'banned'));
    END IF;
END $$;

-- organization_invitations.status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'organization_invitations_status_check'
    ) THEN
        ALTER TABLE organization_invitations
            ADD CONSTRAINT organization_invitations_status_check
            CHECK (status IN ('pending', 'accepted', 'cancelled', 'expired'));
    END IF;
END $$;

-- invoices.status (m.121)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'invoice_status_check'
    ) THEN
        ALTER TABLE invoice
            ADD CONSTRAINT invoice_status_check
            CHECK (status IN ('draft', 'issued', 'credited'));
    END IF;
END $$;

COMMIT;
