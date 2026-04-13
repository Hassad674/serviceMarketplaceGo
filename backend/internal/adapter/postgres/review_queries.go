package postgres

// queryInsertReview inserts a new review row. published_at is always
// NULL at insert time: the reveal logic (either atomic pair reveal or
// the lazy auto-publish on read) is responsible for populating it.
const queryInsertReview = `
	INSERT INTO reviews (
		id, proposal_id, reviewer_id, reviewed_id,
		reviewer_organization_id, reviewed_organization_id,
		side,
		global_rating, timeliness, communication, quality,
		comment, video_url, title_visible, created_at, updated_at,
		published_at
	) VALUES (
		$1, $2, $3, $4,
		$5, $6,
		$7,
		$8, $9, $10, $11,
		$12, $13, $14, $15, $16,
		NULL
	)`

const queryGetReviewByID = `
	SELECT id, proposal_id, reviewer_id, reviewed_id,
		reviewer_organization_id, reviewed_organization_id,
		side,
		global_rating, timeliness, communication, quality,
		comment, video_url, title_visible, created_at, updated_at,
		published_at
	FROM reviews
	WHERE id = $1`

// queryListReviewsByReviewedOrgFirst — public org profile list. Only
// client→provider reviews that are both non-hidden and published are
// returned; blind submissions and provider-side reviews stay private.
const queryListReviewsByReviewedOrgFirst = `
	SELECT id, proposal_id, reviewer_id, reviewed_id,
		reviewer_organization_id, reviewed_organization_id,
		side,
		global_rating, timeliness, communication, quality,
		comment, video_url, title_visible, created_at, updated_at,
		published_at
	FROM reviews
	WHERE reviewed_organization_id = $1
		AND side = 'client_to_provider'
		AND published_at IS NOT NULL
		AND moderation_status <> 'hidden'
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListReviewsByReviewedOrgWithCursor = `
	SELECT id, proposal_id, reviewer_id, reviewed_id,
		reviewer_organization_id, reviewed_organization_id,
		side,
		global_rating, timeliness, communication, quality,
		comment, video_url, title_visible, created_at, updated_at,
		published_at
	FROM reviews
	WHERE reviewed_organization_id = $1
		AND side = 'client_to_provider'
		AND published_at IS NOT NULL
		AND moderation_status <> 'hidden'
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

// queryAverageRatingByOrg — aggregate only the published, non-hidden,
// client→provider reviews so the average stays consistent with what the
// public list surface displays.
const queryAverageRatingByOrg = `
	SELECT COALESCE(AVG(global_rating), 0), COUNT(*)
	FROM reviews
	WHERE reviewed_organization_id = $1
		AND side = 'client_to_provider'
		AND published_at IS NOT NULL
		AND moderation_status <> 'hidden'`

const queryHasReviewed = `
	SELECT EXISTS(
		SELECT 1 FROM reviews
		WHERE proposal_id = $1 AND reviewer_id = $2
	)`

const queryUpdateReviewModeration = `
	UPDATE reviews
	SET moderation_status = $2, moderation_score = $3, moderation_labels = $4, updated_at = now()
	WHERE id = $1`

// queryReviewsByProposalIDs — batch loader used by the project history
// service. Excludes hidden and unpublished reviews so blind submissions
// never leak into the provider's project history view.
const queryReviewsByProposalIDs = `
	SELECT id, proposal_id, reviewer_id, reviewed_id,
		reviewer_organization_id, reviewed_organization_id,
		side,
		global_rating, timeliness, communication, quality,
		comment, video_url, title_visible, created_at, updated_at,
		published_at
	FROM reviews
	WHERE proposal_id = ANY($1)
		AND published_at IS NOT NULL
		AND moderation_status <> 'hidden'`

// queryCountReviewsForProposal — returns the total row count and the
// published row count for the reveal decision. Runs inside the reveal
// transaction right after the INSERT.
const queryCountReviewsForProposal = `
	SELECT
		COUNT(*) AS total,
		COUNT(published_at) AS published
	FROM reviews
	WHERE proposal_id = $1`

// queryRevealPendingReviews — flips any unpublished review on the given
// proposal to published_at = NOW(). Idempotent: already-published rows
// are skipped by the WHERE clause.
const queryRevealPendingReviews = `
	UPDATE reviews
	SET published_at = NOW(), updated_at = NOW()
	WHERE proposal_id = $1 AND published_at IS NULL`

// queryAutoPublishDeadlineElapsed — lazy auto-publish sweep executed
// before public reads. Touches only reviews whose proposal completion
// is older than 14 days and which are still pending. This amortizes the
// deadline reveal across reads and removes the need for a cron worker.
const queryAutoPublishDeadlineElapsed = `
	UPDATE reviews
	SET published_at = NOW(), updated_at = NOW()
	WHERE published_at IS NULL
	  AND proposal_id IN (
	    SELECT id FROM proposals
	    WHERE completed_at IS NOT NULL
	      AND completed_at + interval '14 days' < NOW()
	  )`
