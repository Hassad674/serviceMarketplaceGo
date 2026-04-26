package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/search"
)

// search_ranking_v1_repository.go implements the 3 new aggregate
// loaders introduced in phase 6B of the ranking engine. Each method
// runs exactly one CTE-powered SELECT so the indexer's fan-out stays
// zero N+1 on the hot path.
//
// The queries live in this dedicated file (rather than alongside the
// original LoadActorSignals / LoadRatingAggregate methods) so the
// base `search_document_repository.go` stays under the 600-line cap
// and the review surface for ranking changes remains focused.
//
// Every query is read-only, uses parameterised arguments, and honours
// the shared `queryTimeout` constant declared in user_repository.go.
//
// See docs/ranking-v1.md §3 for the feature definitions these
// queries feed (proven_work_score, rating_score_diverse, negative
// signals, account_age_bonus).

// LoadClientHistory aggregates the proven-work signals from released
// proposal milestones (phase 6B).
//
// Definition recap:
//   - unique_clients = distinct client orgs that have at least one
//     released milestone against this provider.
//   - repeat_client_rate = share of unique_clients that appear in ≥2
//     distinct proposals (projects).
//
// Schema notes:
//   - `proposals.organization_id` is denormalised to the CLIENT's
//     org (migration 062) — one-per-proposal, nullable when the
//     client is a solo Provider without an organisation. When it
//     is NULL we fall back to the client user's personal org via
//     users.organization_id. Either side can be NULL; the COALESCE
//     picks whichever is populated.
//   - The PROVIDER side is derived from `proposals.provider_id` →
//     `users.organization_id`, which matches the "provider org this
//     search document represents" filter on the caller side.
//
// The query uses a CTE so Postgres plans the row gather once and
// computes both aggregates in one pass. The `provider_projects` CTE
// projects (client_org, proposal_id) pairs with NULL client orgs
// filtered out; `per_client` collapses to (client_org, count). The
// outer SELECT counts unique clients and divides "clients with ≥2
// projects" by the total, with NULLIF shielding the zero case.
//
// See docs/ranking-v1.md §3.2-4.
func (r *SearchDocumentRepository) LoadClientHistory(ctx context.Context, orgID uuid.UUID) (*search.RawClientHistory, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
WITH provider_projects AS (
    SELECT DISTINCT
        COALESCE(p.organization_id, cu.organization_id) AS client_org,
        p.id AS proposal_id
    FROM proposal_milestones pm
    JOIN proposals p ON p.id = pm.proposal_id
    JOIN users provider_user ON provider_user.id = p.provider_id
    LEFT JOIN users cu ON cu.id = p.client_id
    WHERE pm.status = 'released'
      AND provider_user.organization_id = $1
      AND COALESCE(p.organization_id, cu.organization_id) IS NOT NULL
),
per_client AS (
    SELECT client_org, COUNT(*) AS project_count
    FROM provider_projects
    GROUP BY client_org
)
SELECT
    COALESCE(COUNT(*), 0) AS unique_clients,
    COALESCE(
        (SELECT COUNT(*) FROM per_client WHERE project_count >= 2)::float
            / NULLIF(COUNT(*), 0),
        0
    ) AS repeat_client_rate
FROM per_client`

	var uniqueClients int
	var repeatRate sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, orgID).Scan(&uniqueClients, &repeatRate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Defensive: the query aggregates so a row is always
			// produced — but if the driver surfaces ErrNoRows, we
			// treat it as an empty history rather than propagating.
			return &search.RawClientHistory{}, nil
		}
		return nil, fmt.Errorf("search repository: load client history: %w", err)
	}
	return &search.RawClientHistory{
		UniqueClients:    uniqueClients,
		RepeatClientRate: repeatRate.Float64,
	}, nil
}

// LoadReviewDiversity aggregates the reviewer-diversity signals from
// the reviews table (phase 6B).
//
// Definition recap:
//   - unique_reviewers = distinct reviewer users.
//   - max_reviewer_share = max(count per reviewer) / total_reviews.
//   - review_recency_factor = mean of exp(-age_days / 365) across
//     every review.
//
// The query uses two CTEs: `recent` materialises (reviewer, age_days,
// recency_weight) triples for each qualifying review; `per_reviewer`
// collapses that to (reviewer, count). The outer SELECT reads
// unique_reviewers + max_reviewer_share from `per_reviewer` and
// review_recency_factor from `recent` with COALESCE shielding against
// empty-input division. We filter reviews to the public ranking
// scope — published, `client_to_provider`, not hidden by moderation —
// to match the reviews actually exposed on the profile page.
//
// See docs/ranking-v1.md §3.2-3 (especially the diversity factor).
func (r *SearchDocumentRepository) LoadReviewDiversity(ctx context.Context, orgID uuid.UUID) (*search.RawReviewDiversity, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
WITH recent AS (
    SELECT
        reviewer_id,
        EXTRACT(EPOCH FROM (NOW() - created_at)) / 86400.0 AS age_days
    FROM reviews rv
    WHERE rv.reviewed_organization_id = $1
      AND rv.side = 'client_to_provider'
      AND rv.published_at IS NOT NULL
      AND NOT EXISTS (
          SELECT 1 FROM moderation_results mr
           WHERE mr.content_type = 'review'
             AND mr.content_id = rv.id
             AND mr.status IN ('hidden', 'deleted')
             AND mr.reviewed_at IS NULL
      )
),
per_reviewer AS (
    SELECT reviewer_id, COUNT(*) AS review_count
    FROM recent
    GROUP BY reviewer_id
)
SELECT
    COALESCE((SELECT COUNT(*) FROM per_reviewer), 0) AS unique_reviewers,
    COALESCE(
        (SELECT MAX(review_count)::float / NULLIF(SUM(review_count), 0)
         FROM per_reviewer),
        0
    ) AS max_reviewer_share,
    COALESCE(
        (SELECT AVG(EXP(-age_days / 365.0)) FROM recent),
        0
    ) AS review_recency_factor`

	var uniqueReviewers int
	var maxShare, recencyFactor sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, orgID).Scan(&uniqueReviewers, &maxShare, &recencyFactor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &search.RawReviewDiversity{}, nil
		}
		return nil, fmt.Errorf("search repository: load review diversity: %w", err)
	}
	return &search.RawReviewDiversity{
		UniqueReviewers:     uniqueReviewers,
		MaxReviewerShare:    maxShare.Float64,
		ReviewRecencyFactor: recencyFactor.Float64,
	}, nil
}

// LoadAccountAge aggregates the disputes + age signals for one
// organisation (phase 6B).
//
// Definition recap:
//   - lost_disputes = disputes where the org was the respondent AND
//     the resolution was a refund (full or partial).
//   - account_age_days = days since the owner user's `created_at`.
//     The owner is the user whose `id = organizations.owner_user_id`.
//
// The query is a single SELECT over the organisations table joined
// to its owner user, with a correlated subquery counting lost
// disputes. We use provider_organization_id on disputes (the org
// the dispute was filed against) and filter on the resolution
// types declared in docs/ranking-v1.md §5.3 — adjusted to the
// actual domain enum values (`full_refund`, `partial_refund`).
//
// See docs/ranking-v1.md §3.2-6, §3.2-9, §5.3.
func (r *SearchDocumentRepository) LoadAccountAge(ctx context.Context, orgID uuid.UUID) (*search.RawAccountAge, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT
    COALESCE(
        (SELECT COUNT(*)
         FROM disputes d
         WHERE d.provider_organization_id = o.id
           AND d.resolution_type IN ('full_refund', 'partial_refund')),
        0
    ) AS lost_disputes,
    COALESCE(
        GREATEST(0, EXTRACT(EPOCH FROM (NOW() - u.created_at)) / 86400.0),
        0
    )::int AS account_age_days
FROM organizations o
JOIN users u ON u.id = o.owner_user_id
WHERE o.id = $1`

	var lostDisputes int
	var accountAgeDays int
	if err := r.db.QueryRowContext(ctx, query, orgID).Scan(&lostDisputes, &accountAgeDays); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No owner row — treat as a brand-new account with no
			// dispute history. Keeps the indexer's fan-out resilient
			// to test fixtures that omit the owner link.
			return &search.RawAccountAge{}, nil
		}
		return nil, fmt.Errorf("search repository: load account age: %w", err)
	}
	return &search.RawAccountAge{
		LostDisputes:   lostDisputes,
		AccountAgeDays: accountAgeDays,
	}, nil
}
