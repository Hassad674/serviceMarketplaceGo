package postgres

const jobColumns = `id, creator_id, title, description, skills,
		applicant_type, budget_type, min_budget, max_budget,
		status, created_at, updated_at, closed_at,
		payment_frequency, duration_weeks, is_indefinite,
		description_type, video_url`

// organization_id is resolved at INSERT time from organization_members,
// which is the source of truth for membership. Providers have no row in
// organization_members, so the column stays NULL for them. Agencies/
// Enterprises (including operators acting on behalf of the org) get the
// org denormalized onto the job, which lets operators list the org's
// jobs in later phases without touching the domain entity.
const queryInsertJob = `
	INSERT INTO jobs (
		id, creator_id, organization_id, title, description, skills,
		applicant_type, budget_type, min_budget, max_budget,
		status, created_at, updated_at, closed_at,
		payment_frequency, duration_weeks, is_indefinite,
		description_type, video_url
	) VALUES (
		$1, $2,
		(SELECT organization_id FROM organization_members WHERE user_id = $2 LIMIT 1),
		$3, $4, $5,
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
	SET title = $2,
		description = $3,
		skills = $4,
		applicant_type = $5,
		budget_type = $6,
		min_budget = $7,
		max_budget = $8,
		payment_frequency = $9,
		duration_weeks = $10,
		is_indefinite = $11,
		description_type = $12,
		video_url = $13,
		status = $14,
		closed_at = $15,
		updated_at = $16
	WHERE id = $1`

const queryListJobsByOrgFirst = `
	SELECT ` + jobColumns + `
	FROM jobs
	WHERE organization_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListJobsByOrgWithCursor = `
	SELECT ` + jobColumns + `
	FROM jobs
	WHERE organization_id = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`
