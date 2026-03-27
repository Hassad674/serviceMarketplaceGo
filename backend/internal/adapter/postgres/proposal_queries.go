package postgres

const queryInsertProposal = `
	INSERT INTO proposals (
		id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	) VALUES (
		$1, $2, $3, $4,
		$5, $6, $7, $8,
		$9, $10, $11,
		$12, $13, $14,
		$15, $16, $17, $18,
		$19, $20
	)`

const queryGetProposalByID = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
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
		updated_at = $8
	WHERE id = $1`

const queryGetLatestVersion = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
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
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	FROM proposals
	WHERE conversation_id = $1
	ORDER BY created_at DESC`

const queryListActiveProjectsFirst = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	FROM proposals
	WHERE (client_id = $1 OR provider_id = $1)
		AND status IN ('paid', 'active', 'completed')
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListActiveProjectsWithCursor = `
	SELECT id, conversation_id, sender_id, recipient_id,
		title, description, amount, deadline,
		status, parent_id, version,
		client_id, provider_id, metadata,
		accepted_at, declined_at, paid_at, completed_at,
		created_at, updated_at
	FROM proposals
	WHERE (client_id = $1 OR provider_id = $1)
		AND status IN ('paid', 'active', 'completed')
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

const queryGetProposalDocuments = `
	SELECT id, proposal_id, filename, url, size, mime_type, created_at
	FROM proposal_documents
	WHERE proposal_id = $1
	ORDER BY created_at ASC`

const queryInsertProposalDocument = `
	INSERT INTO proposal_documents (id, proposal_id, filename, url, size, mime_type, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
