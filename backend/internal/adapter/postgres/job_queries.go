package postgres

const jobColumns = `id, creator_id, title, description, skills,
		applicant_type, budget_type, min_budget, max_budget,
		status, created_at, updated_at, closed_at,
		payment_frequency, duration_weeks, is_indefinite,
		description_type, video_url`

const queryInsertJob = `
	INSERT INTO jobs (
		id, creator_id, title, description, skills,
		applicant_type, budget_type, min_budget, max_budget,
		status, created_at, updated_at, closed_at,
		payment_frequency, duration_weeks, is_indefinite,
		description_type, video_url
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8, $9,
		$10, $11, $12, $13,
		$14, $15, $16,
		$17, $18
	)`

const queryGetJobByID = `
	SELECT ` + jobColumns + `
	FROM jobs
	WHERE id = $1`

const queryUpdateJob = `
	UPDATE jobs
	SET status = $2,
		closed_at = $3,
		updated_at = $4
	WHERE id = $1`

const queryListJobsByCreatorFirst = `
	SELECT ` + jobColumns + `
	FROM jobs
	WHERE creator_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListJobsByCreatorWithCursor = `
	SELECT ` + jobColumns + `
	FROM jobs
	WHERE creator_id = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`
