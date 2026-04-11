package postgres

// organization_id is resolved at INSERT time from organization_members,
// keyed on the client (the business side of the proposal). Agencies/
// Enterprises have a row there and get denormalized. Provider-only
// proposals keep NULL.
const queryInsertProposal = `
	INSERT INTO proposals (
		id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		active_dispute_id, last_dispute_id,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at, organization_id
	) VALUES (
		$1, $2, $3, $4,
		$5, $6, $7, $8,
		$9, $10, $11,
		$12, $13, $14,
		$15, $16,
		$17, $18, $19, $20,
		$21, $22,
		(SELECT organization_id FROM organization_members WHERE user_id = $12 LIMIT 1)
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
// denormalized in phase 4) and the provider's side (resolved via a
// JOIN on users.organization_id, the R1 column). Every operator of
// the same org sees the same active projects — the Stripe Dashboard
// shared-workspace model.
const queryListActiveProjectsByOrgFirst = `
	SELECT p.id, p.conversation_id, p.sender_id, p.recipient_id,
		p.title, p.description, p.amount, p.deadline,
		p.status, p.parent_id, p.version,
		p.client_id, p.provider_id, p.metadata,
		p.active_dispute_id, p.last_dispute_id,
		p.accepted_at, p.declined_at, p.paid_at, p.completed_at,
		p.created_at, p.updated_at
	FROM proposals p
	LEFT JOIN users provider_user ON provider_user.id = p.provider_id
	WHERE (p.organization_id = $1 OR provider_user.organization_id = $1)
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
	LEFT JOIN users provider_user ON provider_user.id = p.provider_id
	WHERE (p.organization_id = $1 OR provider_user.organization_id = $1)
		AND p.status IN ('paid', 'active', 'completion_requested', 'completed', 'disputed')
		AND (p.created_at, p.id) < ($2, $3)
	ORDER BY p.created_at DESC, p.id DESC
	LIMIT $4`

// ListCompletedByOrg is provider-side only (the "my completed
// deliverables" view used by public project-history). Keyed on the
// provider's org resolved via users.
const queryListCompletedByOrgFirst = `
	SELECT p.id, p.conversation_id, p.sender_id, p.recipient_id,
		p.title, p.description, p.amount, p.deadline,
		p.status, p.parent_id, p.version,
		p.client_id, p.provider_id, p.metadata,
		p.active_dispute_id, p.last_dispute_id,
		p.accepted_at, p.declined_at, p.paid_at, p.completed_at,
		p.created_at, p.updated_at
	FROM proposals p
	JOIN users provider_user ON provider_user.id = p.provider_id
	WHERE provider_user.organization_id = $1
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
	JOIN users provider_user ON provider_user.id = p.provider_id
	WHERE provider_user.organization_id = $1
		AND p.status = 'completed'
		AND p.completed_at IS NOT NULL
		AND (p.completed_at, p.id) < ($2, $3)
	ORDER BY p.completed_at DESC, p.id DESC
	LIMIT $4`

const queryGetProposalDocuments = `
	SELECT id, proposal_id, filename, url, size, mime_type, created_at
	FROM proposal_documents
	WHERE proposal_id = $1
	ORDER BY created_at ASC`

const queryInsertProposalDocument = `
	INSERT INTO proposal_documents (id, proposal_id, filename, url, size, mime_type, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
