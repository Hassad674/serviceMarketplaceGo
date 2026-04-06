package postgres

const queryInsertReview = `
	INSERT INTO reviews (
		id, proposal_id, reviewer_id, reviewed_id,
		global_rating, timeliness, communication, quality,
		comment, video_url, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4,
		$5, $6, $7, $8,
		$9, $10, $11, $12
	)`

const queryGetReviewByID = `
	SELECT id, proposal_id, reviewer_id, reviewed_id,
		global_rating, timeliness, communication, quality,
		comment, video_url, created_at, updated_at
	FROM reviews
	WHERE id = $1`

const queryListReviewsByReviewedFirst = `
	SELECT id, proposal_id, reviewer_id, reviewed_id,
		global_rating, timeliness, communication, quality,
		comment, video_url, created_at, updated_at
	FROM reviews
	WHERE reviewed_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListReviewsByReviewedWithCursor = `
	SELECT id, proposal_id, reviewer_id, reviewed_id,
		global_rating, timeliness, communication, quality,
		comment, video_url, created_at, updated_at
	FROM reviews
	WHERE reviewed_id = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

const queryAverageRating = `
	SELECT COALESCE(AVG(global_rating), 0), COUNT(*)
	FROM reviews
	WHERE reviewed_id = $1`

const queryHasReviewed = `
	SELECT EXISTS(
		SELECT 1 FROM reviews
		WHERE proposal_id = $1 AND reviewer_id = $2
	)`

const queryUpdateReviewModeration = `
	UPDATE reviews
	SET moderation_status = $2, moderation_score = $3, moderation_labels = $4, updated_at = now()
	WHERE id = $1`
