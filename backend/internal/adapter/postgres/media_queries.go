package postgres

const queryInsertMedia = `
	INSERT INTO media (
		id, uploader_id, file_url, file_name, file_type, file_size,
		context, context_id, moderation_status, moderation_labels,
		moderation_score, rekognition_job_id, reviewed_at, reviewed_by, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6,
		$7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16
	)`

const queryGetMediaByID = `
	SELECT id, uploader_id, file_url, file_name, file_type, file_size,
		context, context_id, moderation_status, moderation_labels,
		moderation_score, rekognition_job_id, reviewed_at, reviewed_by, created_at, updated_at
	FROM media
	WHERE id = $1`

const queryGetMediaByJobID = `
	SELECT id, uploader_id, file_url, file_name, file_type, file_size,
		context, context_id, moderation_status, moderation_labels,
		moderation_score, rekognition_job_id, reviewed_at, reviewed_by, created_at, updated_at
	FROM media
	WHERE rekognition_job_id = $1
	LIMIT 1`

const queryUpdateMedia = `
	UPDATE media SET
		moderation_status = $2,
		moderation_labels = $3,
		moderation_score = $4,
		rekognition_job_id = $5,
		reviewed_at = $6,
		reviewed_by = $7,
		updated_at = $8
	WHERE id = $1`

const queryDeleteMedia = `DELETE FROM media WHERE id = $1`
