package postgres

const queryInsertReport = `
	INSERT INTO reports (
		id, reporter_id, target_type, target_id, conversation_id,
		reason, description, status, admin_note,
		resolved_at, resolved_by, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8, $9,
		$10, $11, $12, $13
	)`

const queryGetReportByID = `
	SELECT id, reporter_id, target_type, target_id, conversation_id,
		reason, description, status, admin_note,
		resolved_at, resolved_by, created_at, updated_at
	FROM reports
	WHERE id = $1`

const queryListReportsByStatusFirst = `
	SELECT id, reporter_id, target_type, target_id, conversation_id,
		reason, description, status, admin_note,
		resolved_at, resolved_by, created_at, updated_at
	FROM reports
	WHERE status = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListReportsByStatusWithCursor = `
	SELECT id, reporter_id, target_type, target_id, conversation_id,
		reason, description, status, admin_note,
		resolved_at, resolved_by, created_at, updated_at
	FROM reports
	WHERE status = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

const queryListReportsByReporterFirst = `
	SELECT id, reporter_id, target_type, target_id, conversation_id,
		reason, description, status, admin_note,
		resolved_at, resolved_by, created_at, updated_at
	FROM reports
	WHERE reporter_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListReportsByReporterWithCursor = `
	SELECT id, reporter_id, target_type, target_id, conversation_id,
		reason, description, status, admin_note,
		resolved_at, resolved_by, created_at, updated_at
	FROM reports
	WHERE reporter_id = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

const queryListReportsByTarget = `
	SELECT id, reporter_id, target_type, target_id, conversation_id,
		reason, description, status, admin_note,
		resolved_at, resolved_by, created_at, updated_at
	FROM reports
	WHERE target_type = $1 AND target_id = $2
	ORDER BY created_at DESC`

const queryUpdateReportStatus = `
	UPDATE reports
	SET status = $2, admin_note = $3, resolved_by = $4, resolved_at = now(), updated_at = now()
	WHERE id = $1`

const queryHasPendingReport = `
	SELECT EXISTS(
		SELECT 1 FROM reports
		WHERE reporter_id = $1 AND target_type = $2 AND target_id = $3 AND status = 'pending'
	)`
