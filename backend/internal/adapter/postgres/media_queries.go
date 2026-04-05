package postgres

const queryInsertMedia = `
	INSERT INTO media (
		id, uploader_id, file_url, file_name, file_type, file_size,
		context, context_id, moderation_status, moderation_labels,
		moderation_score, reviewed_at, reviewed_by, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6,
		$7, $8, $9, $10,
		$11, $12, $13, $14, $15
	)`

const queryGetMediaByID = `
	SELECT id, uploader_id, file_url, file_name, file_type, file_size,
		context, context_id, moderation_status, moderation_labels,
		moderation_score, reviewed_at, reviewed_by, created_at, updated_at
	FROM media
	WHERE id = $1`

const queryUpdateMedia = `
	UPDATE media SET
		moderation_status = $2,
		moderation_labels = $3,
		moderation_score = $4,
		reviewed_at = $5,
		reviewed_by = $6,
		updated_at = $7
	WHERE id = $1`

const queryDeleteMedia = `DELETE FROM media WHERE id = $1`
