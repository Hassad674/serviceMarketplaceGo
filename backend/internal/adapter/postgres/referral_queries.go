package postgres

// referralColumns must match the order in scanReferral.
const referralColumns = `
	id, referrer_id, provider_id, client_id,
	rate_pct, duration_months,
	intro_snapshot, intro_snapshot_version,
	intro_message_provider, intro_message_client,
	status, version,
	activated_at, expires_at, last_action_at,
	rejection_reason, rejected_by,
	created_at, updated_at`

const queryInsertReferral = `
	INSERT INTO referrals (
		id, referrer_id, provider_id, client_id,
		rate_pct, duration_months,
		intro_snapshot, intro_snapshot_version,
		intro_message_provider, intro_message_client,
		status, version,
		activated_at, expires_at, last_action_at,
		rejection_reason, rejected_by,
		created_at, updated_at
	) VALUES (
		$1, $2, $3, $4,
		$5, $6,
		$7, $8,
		$9, $10,
		$11, $12,
		$13, $14, $15,
		$16, $17,
		$18, $19
	)`

const queryGetReferralByID = `
	SELECT ` + referralColumns + `
	FROM referrals WHERE id = $1`

const queryUpdateReferral = `
	UPDATE referrals SET
		rate_pct           = $2,
		duration_months    = $3,
		status             = $4,
		version            = $5,
		activated_at       = $6,
		expires_at         = $7,
		last_action_at     = $8,
		rejection_reason   = $9,
		rejected_by        = $10,
		updated_at         = now()
	WHERE id = $1`

const queryFindActiveReferralByCouple = `
	SELECT ` + referralColumns + `
	FROM referrals
	WHERE provider_id = $1 AND client_id = $2
	  AND status IN ('pending_provider', 'pending_referrer', 'pending_client', 'active')
	LIMIT 1`

// listing query templates: actor column is interpolated by the caller via
// fmt.Sprintf because Postgres does not parameterise column names. The actor
// column value comes from a fixed allow-list (referrer_id / provider_id /
// client_id) so there is no SQL injection surface.
const queryListReferralsTemplate = `
	SELECT ` + referralColumns + `
	FROM referrals
	WHERE %s = $1
	  %s
	  %s
	ORDER BY created_at DESC, id DESC
	LIMIT $%d`

const queryInsertNegotiation = `
	INSERT INTO referral_negotiations (
		id, referral_id, version, actor_id, actor_role, action, rate_pct, message, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

const queryListNegotiations = `
	SELECT id, referral_id, version, actor_id, actor_role, action, rate_pct, message, created_at
	FROM referral_negotiations
	WHERE referral_id = $1
	ORDER BY created_at ASC`

// attributionColumns is the single source of truth for the column list
// read by every attribution scan path. Keep in sync with scanAttribution
// in referral_repository.go — the column order is load-bearing.
const attributionColumns = `
	id, referral_id, proposal_id, provider_id, client_id,
	rate_pct_snapshot, attributed_at, ended_at`

const queryInsertAttribution = `
	INSERT INTO referral_attributions (
		id, referral_id, proposal_id, provider_id, client_id,
		rate_pct_snapshot, attributed_at, ended_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (proposal_id) DO NOTHING`

const queryFindAttributionByProposal = `
	SELECT ` + attributionColumns + `
	FROM referral_attributions
	WHERE proposal_id = $1`

const queryFindAttributionByID = `
	SELECT ` + attributionColumns + `
	FROM referral_attributions
	WHERE id = $1`

const queryListAttributionsByReferral = `
	SELECT ` + attributionColumns + `
	FROM referral_attributions
	WHERE referral_id = $1
	ORDER BY attributed_at DESC`

// queryListAttributionsByReferralIDs — batch loader used by the
// apporteur reputation aggregate to avoid an N+1 across the
// referrer's referrals. Ordering matches the single-referral query
// so consumers don't have to re-sort.
const queryListAttributionsByReferralIDs = `
	SELECT ` + attributionColumns + `
	FROM referral_attributions
	WHERE referral_id = ANY($1)
	ORDER BY attributed_at DESC`

// queryEndAttribution ends an active intro attribution. The JOIN to
// referrals enforces RBAC — only the apporteur (parent referral's
// referrer_id) can end the attribution. The `ended_at IS NULL` guard
// makes the UPDATE idempotent: a second call affects zero rows and
// the caller can distinguish "already ended" from "not owned / not
// found" by a follow-up read.
const queryEndAttribution = `
	UPDATE referral_attributions a
	SET ended_at = NOW()
	FROM referrals r
	WHERE a.referral_id = r.id
	  AND a.id = $1
	  AND r.referrer_id = $2
	  AND a.ended_at IS NULL`

// queryGetAttributionEndedStateForReferrer disambiguates the failure
// mode after queryEndAttribution affects zero rows: was the row
// already ended (return ErrAttributionAlreadyEnded) or absent /
// owned by someone else (return ErrAttributionNotFound)?
const queryGetAttributionEndedStateForReferrer = `
	SELECT a.ended_at IS NOT NULL AS already_ended
	FROM referral_attributions a
	JOIN referrals r ON r.id = a.referral_id
	WHERE a.id = $1 AND r.referrer_id = $2`

const queryInsertCommission = `
	INSERT INTO referral_commissions (
		id, attribution_id, milestone_id,
		gross_amount_cents, commission_cents, currency,
		status, stripe_transfer_id, stripe_reversal_id, failure_reason,
		paid_at, clawed_back_at, created_at, updated_at
	) VALUES (
		$1, $2, $3,
		$4, $5, $6,
		$7, $8, $9, $10,
		$11, $12, $13, $14
	)`

const queryUpdateCommission = `
	UPDATE referral_commissions SET
		status              = $2,
		stripe_transfer_id  = $3,
		stripe_reversal_id  = $4,
		failure_reason      = $5,
		paid_at             = $6,
		clawed_back_at      = $7,
		updated_at          = now()
	WHERE id = $1`

const queryFindCommissionByMilestone = `
	SELECT id, attribution_id, milestone_id,
	       gross_amount_cents, commission_cents, currency,
	       status, stripe_transfer_id, stripe_reversal_id, failure_reason,
	       paid_at, clawed_back_at, created_at, updated_at
	FROM referral_commissions
	WHERE milestone_id = $1`

// queryFindCommissionByID — single-row PK lookup used by the wallet
// retry endpoint (D1+D2) and other ownership-checking flows.
const queryFindCommissionByID = `
	SELECT id, attribution_id, milestone_id,
	       gross_amount_cents, commission_cents, currency,
	       status, stripe_transfer_id, stripe_reversal_id, failure_reason,
	       paid_at, clawed_back_at, created_at, updated_at
	FROM referral_commissions
	WHERE id = $1`

// queryFindCommissionByStripeTransferID — used by the Stripe webhook
// handler on transfer.failed (D1+D2) to locate the matching commission
// row from the failed transfer id. Stripe Transfer ids (tr_*) are
// unique platform-wide so the lookup is unambiguous. The matching
// column is indexed in migration 108_create_referral_commissions.up.sql
// via the unique partial index on (stripe_transfer_id) WHERE
// stripe_transfer_id IS NOT NULL.
const queryFindCommissionByStripeTransferID = `
	SELECT id, attribution_id, milestone_id,
	       gross_amount_cents, commission_cents, currency,
	       status, stripe_transfer_id, stripe_reversal_id, failure_reason,
	       paid_at, clawed_back_at, created_at, updated_at
	FROM referral_commissions
	WHERE stripe_transfer_id = $1`

const queryListCommissionsByReferral = `
	SELECT c.id, c.attribution_id, c.milestone_id,
	       c.gross_amount_cents, c.commission_cents, c.currency,
	       c.status, c.stripe_transfer_id, c.stripe_reversal_id, c.failure_reason,
	       c.paid_at, c.clawed_back_at, c.created_at, c.updated_at
	FROM referral_commissions c
	JOIN referral_attributions a ON a.id = c.attribution_id
	WHERE a.referral_id = $1
	ORDER BY c.created_at DESC`

const queryListRecentCommissionsByReferrer = `
	SELECT c.id, c.attribution_id, c.milestone_id,
	       c.gross_amount_cents, c.commission_cents, c.currency,
	       c.status, c.stripe_transfer_id, c.stripe_reversal_id, c.failure_reason,
	       c.paid_at, c.clawed_back_at, c.created_at, c.updated_at
	FROM referral_commissions c
	JOIN referral_attributions a ON a.id = c.attribution_id
	JOIN referrals r ON r.id = a.referral_id
	WHERE r.referrer_id = $1
	ORDER BY c.created_at DESC
	LIMIT $2`

const queryListPendingKYCByReferrer = `
	SELECT c.id, c.attribution_id, c.milestone_id,
	       c.gross_amount_cents, c.commission_cents, c.currency,
	       c.status, c.stripe_transfer_id, c.stripe_reversal_id, c.failure_reason,
	       c.paid_at, c.clawed_back_at, c.created_at, c.updated_at
	FROM referral_commissions c
	JOIN referral_attributions a ON a.id = c.attribution_id
	JOIN referrals r ON r.id = a.referral_id
	WHERE c.status = 'pending_kyc' AND r.referrer_id = $1
	ORDER BY c.created_at ASC`

// queryListPendingCommissions returns commission rows in `pending`
// status (NOT pending_kyc) that have been sitting for at least the
// caller's grace window. Used by the referral scheduler sweeper to
// drain prepared-but-untransferred commissions (CRIT-REF).
//
// The status column is already indexed via the composite (status,
// created_at) lookup pattern present on this table; the WHERE clause
// uses the leading equality on status before the inequality on
// created_at so the planner short-circuits correctly.
const queryListPendingCommissions = `
	SELECT id, attribution_id, milestone_id,
	       gross_amount_cents, commission_cents, currency,
	       status, stripe_transfer_id, stripe_reversal_id, failure_reason,
	       paid_at, clawed_back_at, created_at, updated_at
	FROM referral_commissions
	WHERE status = 'pending' AND created_at < $1
	ORDER BY created_at ASC
	LIMIT $2`

const queryListExpiringIntros = `
	SELECT ` + referralColumns + `
	FROM referrals
	WHERE status IN ('pending_provider', 'pending_referrer', 'pending_client')
	  AND last_action_at < $1
	ORDER BY last_action_at ASC
	LIMIT $2`

const queryListExpiringActives = `
	SELECT ` + referralColumns + `
	FROM referrals
	WHERE status = 'active' AND expires_at < $1
	ORDER BY expires_at ASC
	LIMIT $2`

const queryCountByReferrer = `
	SELECT status, count(*)
	FROM referrals
	WHERE referrer_id = $1
	GROUP BY status`

const querySumCommissionsByReferrer = `
	SELECT c.status, COALESCE(SUM(c.commission_cents), 0)
	FROM referral_commissions c
	JOIN referral_attributions a ON a.id = c.attribution_id
	JOIN referrals r ON r.id = a.referral_id
	WHERE r.referrer_id = $1
	GROUP BY c.status`
