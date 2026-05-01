package postgres

// organization_id is resolved at INSERT time from organization_members,
// keyed on the client (the business side of the proposal). Agencies/
// Enterprises have a row there and get denormalized. Provider-only
// proposals keep NULL.
//
// Migration 115 adds the symmetrical client_organization_id and
// provider_organization_id columns. Their values are resolved from
// users.organization_id (the R1 source of truth): the client side
// mirrors what the legacy organization_id column captures, and the
// provider side uses the provider's user → org mapping so the new
// client-profile read paths can aggregate without a JOIN.
const queryInsertProposal = `
	INSERT INTO proposals (
		id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		active_dispute_id, last_dispute_id,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at, organization_id,
		client_organization_id, provider_organization_id
	) VALUES (
		$1, $2, $3, $4,
		$5, $6, $7, $8,
		$9, $10, $11,
		$12, $13, $14,
		$15, $16,
		$17, $18, $19, $20,
		$21, $22,
		(SELECT organization_id FROM organization_members WHERE user_id = $12 LIMIT 1),
		(SELECT organization_id FROM users WHERE id = $12),
		(SELECT organization_id FROM users WHERE id = $13)
	)`

const queryGetProposalByID = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		active_dispute_id, last_dispute_id,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	FROM proposals
	WHERE id = $1`

// queryGetProposalsByIDs — batch loader used by the apporteur
// reputation aggregate to avoid an N+1 across attributions.
const queryGetProposalsByIDs = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		active_dispute_id, last_dispute_id,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	FROM proposals
	WHERE id = ANY($1)`

const queryUpdateProposal = `
	UPDATE proposals
	SET status = $2,
		accepted_at = $3,
		declined_at = $4,
		paid_at = $5,
		completed_at = $6,
		metadata = $7,
		active_dispute_id = $8,
		last_dispute_id = $9,
		updated_at = $10
	WHERE id = $1`

const queryGetLatestVersion = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		active_dispute_id, last_dispute_id,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	FROM proposals
	WHERE id = $1 OR parent_id = $1
	ORDER BY version DESC
	LIMIT 1`

const queryListByConversation = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		active_dispute_id, last_dispute_id,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	FROM proposals
	WHERE conversation_id = $1
	ORDER BY created_at DESC`

// ListActiveProjectsByOrg queries filter by the caller's organization
// from BOTH perspectives: the client's side (proposals.organization_id
// denormalized in phase 4) and the provider's side (the
// provider_organization_id column added by migration 115). Every
// operator of the same org sees the same active projects — the Stripe
// Dashboard shared-workspace model.
//
// PERF-B-08: the previous version of these queries reached the
// provider's org by JOINing users on provider_id, which forced
// Postgres into a BitmapOr + nested-loop plan and added 50–150 ms p50
// once proposals crossed ~100k rows. The denormalized
// provider_organization_id column was added explicitly to drop that
// JOIN. Migration 131 adds the matching composite partial index
// idx_proposals_provider_org_status_created.
const queryListActiveProjectsByOrgFirst = `
	SELECT p.id, p.conversation_id, p.sender_id, p.recipient_id,
		p.title, p.description, p.amount, p.deadline,
		p.status, p.parent_id, p.version,
		p.client_id, p.provider_id, p.metadata,
		p.active_dispute_id, p.last_dispute_id,
		p.accepted_at, p.declined_at, p.paid_at, p.completed_at,
		p.created_at, p.updated_at
	FROM proposals p
	WHERE (p.organization_id = $1 OR p.provider_organization_id = $1)
		AND p.status IN ('paid', 'active', 'completion_requested', 'completed', 'disputed')
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $2`

const queryListActiveProjectsByOrgWithCursor = `
	SELECT p.id, p.conversation_id, p.sender_id, p.recipient_id,
		p.title, p.description, p.amount, p.deadline,
		p.status, p.parent_id, p.version,
		p.client_id, p.provider_id, p.metadata,
		p.active_dispute_id, p.last_dispute_id,
		p.accepted_at, p.declined_at, p.paid_at, p.completed_at,
		p.created_at, p.updated_at
	FROM proposals p
	WHERE (p.organization_id = $1 OR p.provider_organization_id = $1)
		AND p.status IN ('paid', 'active', 'completion_requested', 'completed', 'disputed')
		AND (p.created_at, p.id) < ($2, $3)
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $4`

// ListCompletedByOrg is provider-side only (the "my completed
// deliverables" view used by public project-history). Keyed on the
// denormalized provider_organization_id column (migration 115).
//
// PERF-B-08: dropped the JOIN on users — the new partial index
// idx_proposals_provider_org_completed (migration 131) backs the
// completed-at ordering.
const queryListCompletedByOrgFirst = `
	SELECT p.id, p.conversation_id, p.sender_id, p.recipient_id,
		p.title, p.description, p.amount, p.deadline,
		p.status, p.parent_id, p.version,
		p.client_id, p.provider_id, p.metadata,
		p.active_dispute_id, p.last_dispute_id,
		p.accepted_at, p.declined_at, p.paid_at, p.completed_at,
		p.created_at, p.updated_at
	FROM proposals p
	WHERE p.provider_organization_id = $1
		AND p.status = 'completed'
		AND p.completed_at IS NOT NULL
	ORDER BY p.completed_at DESC, p.id DESC
	LIMIT $2`

const queryListCompletedByOrgWithCursor = `
	SELECT p.id, p.conversation_id, p.sender_id, p.recipient_id,
		p.title, p.description, p.amount, p.deadline,
		p.status, p.parent_id, p.version,
		p.client_id, p.provider_id, p.metadata,
		p.active_dispute_id, p.last_dispute_id,
		p.accepted_at, p.declined_at, p.paid_at, p.completed_at,
		p.created_at, p.updated_at
	FROM proposals p
	WHERE p.provider_organization_id = $1
		AND p.status = 'completed'
		AND p.completed_at IS NOT NULL
		AND (p.completed_at, p.id) < ($2, $3)
	ORDER BY p.completed_at DESC, p.id DESC
	LIMIT $4`

// queryIsOrgAuthorizedForProposal returns TRUE if the proposal has the
// given org on either side — client side (proposals.organization_id)
// OR provider side (proposals.provider_organization_id, denormalized
// in migration 115). Mirrors exactly the two-sided predicate used by
// queryListActiveProjectsByOrgFirst so the read/list views stay in
// sync.
//
// PERF-B-08: dropped the JOIN on users to use the denormalized column.
const queryIsOrgAuthorizedForProposal = `
	SELECT EXISTS (
		SELECT 1
		FROM proposals p
		WHERE p.id = $1
			AND (p.organization_id = $2 OR p.provider_organization_id = $2)
	)`

const queryGetProposalDocuments = `
	SELECT id, proposal_id, filename, url, size, mime_type, created_at
	FROM proposal_documents
	WHERE proposal_id = $1
	ORDER BY created_at ASC`

const queryInsertProposalDocument = `
	INSERT INTO proposal_documents (id, proposal_id, filename, url, size, mime_type, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

// querySumPaidByClientOrg aggregates the total amount (in cents) the
// given organization has spent as the client. Keyed on the
// denormalized client_organization_id column (migration 115) so the
// plan stays a partial-index scan. Counts proposals that reached the
// paid stage or beyond — i.e. the money actually left the client.
const querySumPaidByClientOrg = `
	SELECT COALESCE(SUM(amount), 0)::bigint
	FROM proposals
	WHERE client_organization_id = $1
	  AND status IN ('paid', 'active', 'completion_requested', 'completed', 'disputed')`

// queryListCompletedByClientOrg returns the org's most recent completed
// deals as the client — the symmetric counterpart of
// queryListCompletedByOrg*. Uses the dedicated partial index
// idx_proposals_client_org_completed.
const queryListCompletedByClientOrg = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		active_dispute_id, last_dispute_id,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	FROM proposals
	WHERE client_organization_id = $1
		AND status = 'completed'
		AND completed_at IS NOT NULL
	ORDER BY completed_at DESC, id DESC
	LIMIT $2`
