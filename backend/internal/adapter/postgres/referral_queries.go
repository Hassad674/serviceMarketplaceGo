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

const queryInsertAttribution = `
	INSERT INTO referral_attributions (
		id, referral_id, proposal_id, provider_id, client_id, rate_pct_snapshot, attributed_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (proposal_id) DO NOTHING`

const queryFindAttributionByProposal = `
	SELECT id, referral_id, proposal_id, provider_id, client_id, rate_pct_snapshot, attributed_at
	FROM referral_attributions
	WHERE proposal_id = $1`

const queryListAttributionsByReferral = `
	SELECT id, referral_id, proposal_id, provider_id, client_id, rate_pct_snapshot, attributed_at
	FROM referral_attributions
	WHERE referral_id = $1
	ORDER BY attributed_at DESC`

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

const queryListCommissionsByReferral = `
	SELECT c.id, c.attribution_id, c.milestone_id,
	       c.gross_amount_cents, c.commission_cents, c.currency,
	       c.status, c.stripe_transfer_id, c.stripe_reversal_id, c.failure_reason,
	       c.paid_at, c.clawed_back_at, c.created_at, c.updated_at
	FROM referral_commissions c
	JOIN referral_attributions a ON a.id = c.attribution_id
	WHERE a.referral_id = $1
	ORDER BY c.created_at DESC`

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
