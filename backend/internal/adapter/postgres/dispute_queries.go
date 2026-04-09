package postgres

const disputeColumns = `
	id, proposal_id, conversation_id, initiator_id, respondent_id,
	client_id, provider_id, reason, description,
	requested_amount, proposal_amount, status,
	resolution_type, resolution_amount_client, resolution_amount_provider,
	resolved_by, resolution_note, ai_summary,
	escalated_at, resolved_at, cancelled_at,
	last_activity_at, respondent_first_reply_at,
	cancellation_requested_by, cancellation_requested_at,
	version, created_at, updated_at`

const queryInsertDispute = `
	INSERT INTO disputes (
		id, proposal_id, conversation_id, initiator_id, respondent_id,
		client_id, provider_id, reason, description,
		requested_amount, proposal_amount, status,
		resolution_type, resolution_amount_client, resolution_amount_provider,
		resolved_by, resolution_note, ai_summary,
		escalated_at, resolved_at, cancelled_at,
		last_activity_at, respondent_first_reply_at,
		cancellation_requested_by, cancellation_requested_at,
		version, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8, $9,
		$10, $11, $12,
		$13, $14, $15,
		$16, $17, $18,
		$19, $20, $21,
		$22, $23,
		$24, $25,
		$26, $27, $28
	)`

const queryGetDisputeByID = `
	SELECT ` + disputeColumns + `
	FROM disputes WHERE id = $1`

const queryGetDisputeByProposalID = `
	SELECT ` + disputeColumns + `
	FROM disputes
	WHERE proposal_id = $1 AND status NOT IN ('resolved', 'cancelled')
	LIMIT 1`

const queryUpdateDispute = `
	UPDATE disputes SET
		status = $2,
		resolution_type = $3, resolution_amount_client = $4, resolution_amount_provider = $5,
		resolved_by = $6, resolution_note = $7, ai_summary = $8,
		escalated_at = $9, resolved_at = $10, cancelled_at = $11,
		last_activity_at = $12, respondent_first_reply_at = $13,
		cancellation_requested_by = $14, cancellation_requested_at = $15,
		version = version + 1, updated_at = NOW()
	WHERE id = $1 AND version = $16`

const queryListDisputesByUserFirst = `
	SELECT ` + disputeColumns + `
	FROM disputes
	WHERE initiator_id = $1 OR respondent_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListDisputesByUserWithCursor = `
	SELECT ` + disputeColumns + `
	FROM disputes
	WHERE (initiator_id = $1 OR respondent_id = $1)
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

const queryListDisputesPendingScheduler = `
	SELECT ` + disputeColumns + `
	FROM disputes
	WHERE status IN ('open', 'negotiation')
		AND last_activity_at < NOW() - INTERVAL '7 days'`

const queryListAllDisputesFirst = `
	SELECT ` + disputeColumns + `
	FROM disputes
	WHERE ($1 = '' OR status = $1)
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListAllDisputesWithCursor = `
	SELECT ` + disputeColumns + `
	FROM disputes
	WHERE ($1 = '' OR status = $1)
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

// Evidence queries
const queryInsertEvidence = `
	INSERT INTO dispute_evidence (id, dispute_id, counter_proposal_id, uploader_id, filename, url, size, mime_type, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

const queryListEvidence = `
	SELECT id, dispute_id, counter_proposal_id, uploader_id, filename, url, size, mime_type, created_at
	FROM dispute_evidence WHERE dispute_id = $1 ORDER BY created_at ASC`

// Counter-proposal queries
const queryInsertCounterProposal = `
	INSERT INTO dispute_counter_proposals (id, dispute_id, proposer_id, amount_client, amount_provider, message, status, responded_at, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

const queryGetCounterProposalByID = `
	SELECT id, dispute_id, proposer_id, amount_client, amount_provider, message, status, responded_at, created_at
	FROM dispute_counter_proposals WHERE id = $1`

const queryUpdateCounterProposal = `
	UPDATE dispute_counter_proposals
	SET status = $2, responded_at = $3
	WHERE id = $1`

const queryListCounterProposals = `
	SELECT id, dispute_id, proposer_id, amount_client, amount_provider, message, status, responded_at, created_at
	FROM dispute_counter_proposals WHERE dispute_id = $1 ORDER BY created_at ASC`

const querySupersedeAllPending = `
	UPDATE dispute_counter_proposals
	SET status = 'superseded'
	WHERE dispute_id = $1 AND status = 'pending'`

// Stats
const queryCountDisputesByUser = `
	SELECT COUNT(*) FROM disputes WHERE initiator_id = $1 OR respondent_id = $1`

const queryCountAllDisputes = `
	SELECT
		COUNT(*),
		COUNT(*) FILTER (WHERE status IN ('open', 'negotiation')),
		COUNT(*) FILTER (WHERE status = 'escalated')
	FROM disputes`
