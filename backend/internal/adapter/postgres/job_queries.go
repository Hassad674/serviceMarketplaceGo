package postgres

const queryInsertJob = `
	INSERT INTO jobs (
		id, creator_id, title, description, skills,
		applicant_type, budget_type, min_budget, max_budget,
		status, created_at, updated_at, closed_at
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8, $9,
		$10, $11, $12, $13
	)`

const queryGetJobByID = `
	SELECT id, creator_id, title, description, skills,
		applicant_type, budget_type, min_budget, max_budget,
		status, created_at, updated_at, closed_at
	FROM jobs
	WHERE id = $1`

const queryUpdateJob = `
	UPDATE jobs
	SET status = $2,
		closed_at = $3,
		updated_at = $4
	WHERE id = $1`

const queryListJobsByCreatorFirst = `
	SELECT id, creator_id, title, description, skills,
		applicant_type, budget_type, min_budget, max_budget,
		status, created_at, updated_at, closed_at
	FROM jobs
	WHERE creator_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListJobsByCreatorWithCursor = `
	SELECT id, creator_id, title, description, skills,
		applicant_type, budget_type, min_budget, max_budget,
		status, created_at, updated_at, closed_at
	FROM jobs
	WHERE creator_id = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`
