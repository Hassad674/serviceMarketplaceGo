package postgres

const jobAppColumns = `id, job_id, applicant_id, applicant_organization_id, applicant_kind, message, video_url, created_at, updated_at`

const queryInsertJobApplication = `
	INSERT INTO job_applications (id, job_id, applicant_id, applicant_organization_id, applicant_kind, message, video_url, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

const queryGetJobApplicationByID = `
	SELECT ` + jobAppColumns + `
	FROM job_applications
	WHERE id = $1`

const queryGetJobApplicationByJobAndApplicant = `
	SELECT ` + jobAppColumns + `
	FROM job_applications
	WHERE job_id = $1 AND applicant_id = $2`

const queryDeleteJobApplication = `
	DELETE FROM job_applications
	WHERE id = $1`

const queryListJobAppsByJobFirst = `
	SELECT ` + jobAppColumns + `
	FROM job_applications
	WHERE job_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListJobAppsByJobWithCursor = `
	SELECT ` + jobAppColumns + `
	FROM job_applications
	WHERE job_id = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

// queryListJobAppsByJobAndKindFirst / WithCursor narrow the candidates
// list to a single applicant_kind. The composite index
// idx_job_applications_job_kind (job_id, applicant_kind, created_at DESC)
// keeps the keyset pagination efficient.
const queryListJobAppsByJobAndKindFirst = `
	SELECT ` + jobAppColumns + `
	FROM job_applications
	WHERE job_id = $1 AND applicant_kind = $2
	ORDER BY created_at DESC, id DESC
	LIMIT $3`

const queryListJobAppsByJobAndKindWithCursor = `
	SELECT ` + jobAppColumns + `
	FROM job_applications
	WHERE job_id = $1 AND applicant_kind = $2
		AND (created_at, id) < ($3, $4)
	ORDER BY created_at DESC, id DESC
	LIMIT $5`

const queryListJobAppsByApplicantOrgFirst = `
	SELECT ` + jobAppColumns + `
	FROM job_applications
	WHERE applicant_organization_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListJobAppsByApplicantOrgWithCursor = `
	SELECT ` + jobAppColumns + `
	FROM job_applications
	WHERE applicant_organization_id = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

const queryCountJobAppsByJob = `
	SELECT COUNT(*) FROM job_applications WHERE job_id = $1`
