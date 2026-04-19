package fixtures

// search_ranking_v1_signals.go holds the fixture helpers that seed
// the downstream rows (released milestones, reviews, disputes,
// owner-age shifts) the 7 phase 6B ranking signals read. Extracted
// into its own file so search_profiles.go stays under the 600-line
// cap and the ranking surface is reviewable in isolation.
//
// Every insert is idempotent via ON CONFLICT DO NOTHING + a
// deterministic UUID derived from the provider index — reruns of
// SeedSearchProfiles produce the same rows.
//
// See docs/ranking-v1.md §3 for the feature definitions each helper
// exercises.

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// seedRankingSignals produces the downstream rows (released
// milestones, reviews, disputes, owner-age shift) needed to populate
// the 7 signals introduced in phase 6B. Distributions are
// deliberately modest for the 200-profile synthetic set so unit
// tests run fast — the realistic stress distribution lives in
// cmd/seed-search instead.
//
// Rules:
//   - Every 3rd freelance gets 2 released projects across 2 distinct
//     clients so unique_clients_count / repeat_client_rate have a
//     non-zero representative.
//   - Every 5th freelance gets 2 reviews from 1 reviewer (max_share
//     1.0) to exercise the diversity factor path.
//   - Every 7th freelance gets 2 reviews from 2 reviewers + 1
//     recent, 1 old (recency factor cross-path).
//   - Every 11th freelance gets a full_refund dispute so
//     lost_disputes_count is non-zero on at least one document.
//   - Freelance indexes 0,1,2 get specific account_age_days markers
//     (10, 400, 730) so downstream account_age_bonus tests have
//     distinct anchors.
func seedRankingSignals(ctx context.Context, db *sql.DB, seeded *SeededProfiles) error {
	if len(seeded.Freelance) == 0 {
		return nil
	}
	clientPool := make([]uuid.UUID, 0, len(seeded.Agency)+len(seeded.Freelance))
	clientPool = append(clientPool, seeded.Agency...)
	clientPool = append(clientPool, seeded.Freelance...)
	if len(clientPool) < 2 {
		return nil
	}

	for idx, providerOrg := range seeded.Freelance {
		if err := applyFixtureAccountAge(ctx, db, providerOrg, idx); err != nil {
			return err
		}
		if idx%3 == 0 {
			if err := insertFixtureReleasedProjects(ctx, db, providerOrg, clientPool, idx); err != nil {
				return err
			}
		}
		if idx%5 == 0 {
			if err := insertFixtureReviews(ctx, db, providerOrg, clientPool, idx, 1, 2); err != nil {
				return err
			}
		}
		if idx%7 == 0 {
			if err := insertFixtureReviews(ctx, db, providerOrg, clientPool, idx, 2, 1); err != nil {
				return err
			}
		}
		if idx%11 == 0 {
			if err := insertFixtureLostDispute(ctx, db, providerOrg, clientPool, idx); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyFixtureAccountAge shifts the owner user's created_at so the
// first three fixtures anchor known values (10 / 400 / 730 days ago)
// and the rest keep today's created_at (age 0) — good enough to test
// the path without complicating the fixture generator.
func applyFixtureAccountAge(ctx context.Context, db *sql.DB, orgID uuid.UUID, idx int) error {
	ages := []int{10, 400, 730}
	if idx >= len(ages) {
		return nil
	}
	_, err := db.ExecContext(ctx,
		`UPDATE users SET created_at = NOW() - ($1::text || ' days')::interval
		 WHERE organization_id = $2`,
		fmt.Sprint(ages[idx]), orgID)
	return err
}

// insertFixtureReleasedProjects writes 2 released milestones against
// 2 distinct client orgs for the provider. Deterministic via
// idx-based UUIDs.
func insertFixtureReleasedProjects(ctx context.Context, db *sql.DB, providerOrg uuid.UUID, clientPool []uuid.UUID, idx int) error {
	providerUserID, err := fetchFixtureUserForOrg(ctx, db, providerOrg)
	if err != nil {
		return err
	}
	clientA := clientPool[(idx+1)%len(clientPool)]
	clientB := clientPool[(idx+2)%len(clientPool)]
	if clientA == providerOrg {
		clientA = clientPool[(idx+3)%len(clientPool)]
	}
	if clientB == providerOrg {
		clientB = clientPool[(idx+4)%len(clientPool)]
	}
	for k, clientOrg := range []uuid.UUID{clientA, clientB} {
		if err := insertOneReleasedProject(ctx, db, providerOrg, providerUserID, clientOrg, idx, k); err != nil {
			return err
		}
	}
	return nil
}

// insertOneReleasedProject writes a (conversation, proposal,
// released milestone) chain. Split from insertFixtureReleasedProjects
// so the parent stays under the 50-line function cap.
func insertOneReleasedProject(ctx context.Context, db *sql.DB, providerOrg, providerUserID, clientOrg uuid.UUID, idx, k int) error {
	clientUserID, err := fetchFixtureUserForOrg(ctx, db, clientOrg)
	if err != nil {
		return err
	}
	label := fmt.Sprintf("fx-proj-%d-%d", idx, k)
	convID := deterministicUUID("conv-" + label)
	proposalID := deterministicUUID("prop-" + label)
	milestoneID := deterministicUUID("mile-" + label)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)
		 ON CONFLICT (id) DO NOTHING`, convID, clientOrg); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
		                        title, description, amount, status, organization_id)
		 VALUES ($1, $2, $3, $4, $3, $4, $5, 'Fixture project', 100000, 'completed', $6)
		 ON CONFLICT (id) DO NOTHING`,
		proposalID, convID, clientUserID, providerUserID, label, clientOrg); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO proposal_milestones (id, proposal_id, sequence, title, description, amount, status, released_at)
		 VALUES ($1, $2, 1, 'Milestone', 'Scope', 100000, 'released', NOW())
		 ON CONFLICT (id) DO NOTHING`,
		milestoneID, proposalID)
	return err
}

// insertFixtureReviews writes `reviewers × reviewsEach` published
// reviews against the provider. Deterministic via idx.
func insertFixtureReviews(ctx context.Context, db *sql.DB, providerOrg uuid.UUID, clientPool []uuid.UUID, idx, reviewers, reviewsEach int) error {
	providerUserID, err := fetchFixtureUserForOrg(ctx, db, providerOrg)
	if err != nil {
		return err
	}
	for r := 0; r < reviewers; r++ {
		reviewerOrg := clientPool[(idx+r+5)%len(clientPool)]
		if reviewerOrg == providerOrg {
			reviewerOrg = clientPool[(idx+r+6)%len(clientPool)]
		}
		if err := insertReviewerSet(ctx, db, providerOrg, providerUserID, reviewerOrg, idx, r, reviewsEach); err != nil {
			return err
		}
	}
	return nil
}

// insertReviewerSet writes `count` published reviews from a single
// reviewer against the provider. Separated so insertFixtureReviews
// stays under the 50-line function cap.
func insertReviewerSet(ctx context.Context, db *sql.DB, providerOrg, providerUserID, reviewerOrg uuid.UUID, idx, r, count int) error {
	reviewerUserID, err := fetchFixtureUserForOrg(ctx, db, reviewerOrg)
	if err != nil {
		return err
	}
	for k := 0; k < count; k++ {
		label := fmt.Sprintf("fx-rev-%d-%d-%d", idx, r, k)
		convID := deterministicUUID("conv-" + label)
		proposalID := deterministicUUID("prop-" + label)
		reviewID := deterministicUUID("rev-" + label)
		if _, err := db.ExecContext(ctx,
			`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)
			 ON CONFLICT (id) DO NOTHING`, convID, reviewerOrg); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
			                        title, description, amount, status, organization_id)
			 VALUES ($1, $2, $3, $4, $3, $4, 'fixture review', 'scope', 1, 'completed', $5)
			 ON CONFLICT (id) DO NOTHING`,
			proposalID, convID, reviewerUserID, providerUserID, reviewerOrg); err != nil {
			return err
		}
		ageDays := (k + 1) * 60
		if _, err := db.ExecContext(ctx,
			`INSERT INTO reviews (id, proposal_id, reviewer_id, reviewed_id,
			                      global_rating, timeliness, communication, quality, comment,
			                      reviewer_organization_id, reviewed_organization_id, side,
			                      moderation_status, moderation_score, title_visible,
			                      created_at, published_at)
			 VALUES ($1, $2, $3, $4, 5, 5, 5, 5, 'Fixture review',
			         $5, $6, 'client_to_provider',
			         'clean', 0, true,
			         NOW() - ($7::text || ' days')::interval,
			         NOW() - ($7::text || ' days')::interval)
			 ON CONFLICT (proposal_id, reviewer_id) DO NOTHING`,
			reviewID, proposalID, reviewerUserID, providerUserID,
			reviewerOrg, providerOrg, fmt.Sprint(ageDays)); err != nil {
			return err
		}
	}
	return nil
}

// insertFixtureLostDispute writes one full_refund dispute against
// the provider so lost_disputes_count tests have coverage. Uses
// a fresh (conversation, proposal, milestone, dispute) chain to
// satisfy FK invariants.
func insertFixtureLostDispute(ctx context.Context, db *sql.DB, providerOrg uuid.UUID, clientPool []uuid.UUID, idx int) error {
	providerUserID, err := fetchFixtureUserForOrg(ctx, db, providerOrg)
	if err != nil {
		return err
	}
	clientOrg := clientPool[(idx+9)%len(clientPool)]
	if clientOrg == providerOrg {
		clientOrg = clientPool[(idx+10)%len(clientPool)]
	}
	clientUserID, err := fetchFixtureUserForOrg(ctx, db, clientOrg)
	if err != nil {
		return err
	}
	label := fmt.Sprintf("fx-disp-%d", idx)
	convID := deterministicUUID("conv-" + label)
	proposalID := deterministicUUID("prop-" + label)
	milestoneID := deterministicUUID("mile-" + label)
	disputeID := deterministicUUID("disp-" + label)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)
		 ON CONFLICT (id) DO NOTHING`, convID, clientOrg); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
		                        title, description, amount, status, organization_id)
		 VALUES ($1, $2, $3, $4, $3, $4, 'fx-dispute', 'scope', 100000, 'disputed', $5)
		 ON CONFLICT (id) DO NOTHING`,
		proposalID, convID, clientUserID, providerUserID, clientOrg); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO proposal_milestones (id, proposal_id, sequence, title, description, amount, status)
		 VALUES ($1, $2, 1, 'fx-disputed-step', 'Scope', 100000, 'disputed')
		 ON CONFLICT (id) DO NOTHING`,
		milestoneID, proposalID); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO disputes (id, proposal_id, conversation_id,
		                       initiator_id, respondent_id,
		                       client_id, provider_id,
		                       client_organization_id, provider_organization_id,
		                       milestone_id,
		                       reason, description, requested_amount, proposal_amount,
		                       status, resolution_type, resolved_at)
		 VALUES ($1, $2, $3, $4, $5, $4, $5, $6, $7, $8,
		         'quality_issue', 'Fixture dispute', 100000, 100000,
		         'resolved', 'full_refund', NOW())
		 ON CONFLICT (id) DO NOTHING`,
		disputeID, proposalID, convID, clientUserID, providerUserID,
		clientOrg, providerOrg, milestoneID)
	return err
}

// fetchFixtureUserForOrg returns the owner user's ID for the given
// org. Returns an error when the row is missing — callers treat
// this as a fixture invariant violation.
func fetchFixtureUserForOrg(ctx context.Context, db *sql.DB, orgID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := db.QueryRowContext(ctx,
		`SELECT id FROM users WHERE organization_id = $1 LIMIT 1`, orgID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("fetch user for org %s: %w", orgID, err)
	}
	return userID, nil
}
