package main

// ranking_signals.go seeds the downstream marketplace data needed to
// exercise the 7 ranking V1 signals introduced in phase 6B:
//
//   - Released proposal milestones  → unique_clients_count,
//                                      repeat_client_rate
//   - Published reviews             → unique_reviewers_count,
//                                      max_reviewer_share,
//                                      review_recency_factor
//   - Resolved disputes (refund)    → lost_disputes_count
//   - Owner user.created_at shift   → account_age_days
//
// The seeder is invoked from the existing seedAllPersonas call chain
// and only runs once the persona profiles are in place. Distributions
// are tuned so the indexer produces a meaningful stress-test dataset
// for the downstream ranking pipeline:
//
//   - ~5 % of freelancers receive 1 full_refund dispute (negative
//     signals path)
//   - ~3 % of freelancers get heavy reviewer concentration
//     (max_reviewer_share > 0.7) to exercise the diversity factor
//   - account_age spreads: 15 % < 30 days, 50 % 30-365 days, 35 % >
//     365 days — so is_verified_mature + account_age_bonus both have
//     signal
//   - ~60 % of freelancers ship at least one released project, of
//     which 25 % have a repeat client (>= 2 projects with the same
//     client_org)
//
// All rows are anchored on seed-generated organisation IDs, which
// means `wipePreviousSeed` cascades through every downstream row
// without extra DELETE statements (FK CASCADE on
// proposal_milestones/proposals/disputes).

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// seedRankingV1Signals walks the seeded freelance list and writes a
// realistic set of downstream rows (reviews, released milestones,
// disputes) plus patches the owner user's `created_at` to spread
// account_age_days across the documented distribution.
//
// See docs/ranking-v1.md §3 for the ranking features these signals
// feed.
func seedRankingV1Signals(ctx context.Context, db *sql.DB, counts personaCounts, r *rand.Rand) error {
	// Pre-build a pool of "client" freelancers: anyone from the
	// seeded set can become a reviewer/client against anyone else.
	// We use freelance + agency IDs so clients are real orgs.
	clients := make([]clientRef, 0, counts.freelance+counts.agency)
	for i := 0; i < counts.freelance; i++ {
		orgID := deterministicUUID(fmt.Sprintf("seedsearch-freelance-org-%d", i))
		userID := deterministicUUID(fmt.Sprintf("seedsearch-freelance-user-%d", i))
		clients = append(clients, clientRef{orgID: orgID, userID: userID})
	}
	for i := 0; i < counts.agency; i++ {
		orgID := deterministicUUID(fmt.Sprintf("seedsearch-agency-org-%d", i))
		userID := deterministicUUID(fmt.Sprintf("seedsearch-agency-user-%d", i))
		clients = append(clients, clientRef{orgID: orgID, userID: userID})
	}
	if len(clients) == 0 {
		return nil
	}

	for i := 0; i < counts.freelance; i++ {
		providerOrgID := deterministicUUID(fmt.Sprintf("seedsearch-freelance-org-%d", i))
		providerUserID := deterministicUUID(fmt.Sprintf("seedsearch-freelance-user-%d", i))

		if err := applyAccountAgeDistribution(ctx, db, providerUserID, i, r); err != nil {
			return fmt.Errorf("seed account age #%d: %w", i, err)
		}
		if err := applyProjectHistory(ctx, db, providerOrgID, providerUserID, i, clients, r); err != nil {
			return fmt.Errorf("seed project history #%d: %w", i, err)
		}
		if err := applyReviewDistribution(ctx, db, providerOrgID, providerUserID, i, clients, r); err != nil {
			return fmt.Errorf("seed reviews #%d: %w", i, err)
		}
		if err := applyDisputeDistribution(ctx, db, providerOrgID, providerUserID, i, clients, r); err != nil {
			return fmt.Errorf("seed disputes #%d: %w", i, err)
		}
	}
	return nil
}

// clientRef captures (org, owner user) for a candidate client in the
// fixture set. The client role is played by freelance + agency orgs;
// enterprise accounts would be ideal but the seeder currently only
// creates the three marketplace personas, so we reuse existing
// organisations as client substitutes. This does not skew the 7
// signals we care about because the aggregations only count
// DISTINCT client orgs per provider — never read any profile-side
// signal from those clients.
type clientRef struct {
	orgID  uuid.UUID
	userID uuid.UUID
}

// applyAccountAgeDistribution patches the owner user's created_at to
// match the documented spread (15 % < 30 days, 50 % 30-365 days,
// 35 % > 365 days). Controlled by the deterministic RNG so reruns
// produce the same distribution.
func applyAccountAgeDistribution(ctx context.Context, db *sql.DB, userID uuid.UUID, index int, r *rand.Rand) error {
	bucket := r.Intn(100)
	var ageDays int
	switch {
	case bucket < 15:
		ageDays = r.Intn(30) // < 30 days (fresh, not mature)
	case bucket < 65:
		ageDays = 30 + r.Intn(335) // 30-365 days (established)
	default:
		ageDays = 365 + r.Intn(1460) // 1-5 years
	}
	_ = index // kept for potential future keyed-debug logging
	_, err := db.ExecContext(ctx,
		`UPDATE users SET created_at = NOW() - ($1::text || ' days')::interval WHERE id = $2`,
		fmt.Sprint(ageDays), userID)
	return err
}

// applyProjectHistory creates 0-5 released milestones for the
// provider, spread across 0-4 distinct client organisations. About
// 25 % of providers with ≥2 projects get a repeat client.
func applyProjectHistory(ctx context.Context, db *sql.DB, providerOrgID, providerUserID uuid.UUID, index int, clients []clientRef, r *rand.Rand) error {
	// 40 % of providers have no track record yet (cold start).
	if r.Intn(100) < 40 {
		return nil
	}
	projectCount := 1 + r.Intn(5)      // 1-5 completed projects
	repeatHeavy := r.Intn(100) < 25    // 25 % get a repeat client
	pickedClients := make([]clientRef, 0, projectCount)
	for p := 0; p < projectCount; p++ {
		var c clientRef
		if repeatHeavy && p > 0 && len(pickedClients) > 0 {
			// Reuse the first client to create a repeat.
			c = pickedClients[0]
		} else {
			c = clients[r.Intn(len(clients))]
			// Avoid self-referral: a provider cannot be their own client.
			if c.orgID == providerOrgID {
				c = clients[(r.Intn(len(clients))+1)%len(clients)]
			}
			pickedClients = append(pickedClients, c)
		}
		if err := insertReleasedProject(ctx, db, providerOrgID, providerUserID, c, index, p); err != nil {
			return err
		}
	}
	return nil
}

// insertReleasedProject writes one (conversation, proposal,
// released milestone) chain. The conversation and milestone rows
// exist purely to satisfy FK constraints on proposals/milestones.
func insertReleasedProject(ctx context.Context, db *sql.DB, providerOrgID, providerUserID uuid.UUID, client clientRef, providerIdx, projectIdx int) error {
	label := fmt.Sprintf("project-%d-%d", providerIdx, projectIdx)
	convID := deterministicUUID("seedsearch-conv-" + label)
	proposalID := deterministicUUID("seedsearch-proposal-" + label)
	milestoneID := deterministicUUID("seedsearch-milestone-" + label)

	if _, err := db.ExecContext(ctx,
		`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)
		 ON CONFLICT (id) DO NOTHING`, convID, client.orgID); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
		                        title, description, amount, status, organization_id)
		 VALUES ($1, $2, $3, $4, $3, $4, $5, $6, 100000, 'completed', $7)
		 ON CONFLICT (id) DO NOTHING`,
		proposalID, convID, client.userID, providerUserID,
		fmt.Sprintf("Mission %s", label),
		fmt.Sprintf("Delivered scope for %s", label),
		client.orgID); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO proposal_milestones (id, proposal_id, sequence, title, description, amount, status, released_at)
		 VALUES ($1, $2, 1, $3, 'Scope', 100000, 'released', NOW())
		 ON CONFLICT (id) DO NOTHING`,
		milestoneID, proposalID,
		fmt.Sprintf("Milestone %s", label)); err != nil {
		return err
	}
	return nil
}

// applyReviewDistribution writes published client-to-provider
// reviews with a variety of reviewer-diversity profiles:
//   - 40 % no reviews (cold start)
//   - 3  % heavy concentration (1 reviewer posts ≥ 4 reviews →
//     max_reviewer_share > 0.7)
//   - 10 % moderate (2-3 reviewers, 1-2 each)
//   - 47 % balanced (3-6 reviewers, 1 review each)
//
// Review ages span 0-800 days to exercise review_recency_factor.
func applyReviewDistribution(ctx context.Context, db *sql.DB, providerOrgID, providerUserID uuid.UUID, index int, clients []clientRef, r *rand.Rand) error {
	bucket := r.Intn(100)
	var pattern reviewPattern
	switch {
	case bucket < 40:
		return nil // no reviews
	case bucket < 43:
		pattern = reviewPattern{reviewers: 1, reviewsPerReviewer: 4 + r.Intn(4)} // heavy concentration
	case bucket < 53:
		pattern = reviewPattern{reviewers: 2 + r.Intn(2), reviewsPerReviewer: 1 + r.Intn(2)}
	default:
		pattern = reviewPattern{reviewers: 3 + r.Intn(4), reviewsPerReviewer: 1}
	}

	seq := 0
	for rIdx := 0; rIdx < pattern.reviewers; rIdx++ {
		client := clients[(index+rIdx+1)%len(clients)]
		if client.orgID == providerOrgID {
			client = clients[(index+rIdx+2)%len(clients)]
		}
		for k := 0; k < pattern.reviewsPerReviewer; k++ {
			ageDays := r.Intn(800)
			seq++
			if err := insertPublishedReview(ctx, db, providerOrgID, providerUserID, client, index, seq, ageDays, r); err != nil {
				return err
			}
		}
	}
	return nil
}

type reviewPattern struct {
	reviewers          int
	reviewsPerReviewer int
}

// insertPublishedReview writes one published client-to-provider
// review. Rating is concentrated around 4-5 (realistic distribution
// from prior e-commerce research).
func insertPublishedReview(ctx context.Context, db *sql.DB, providerOrgID, providerUserID uuid.UUID, client clientRef, providerIdx, seq, ageDays int, r *rand.Rand) error {
	label := fmt.Sprintf("review-%d-%d", providerIdx, seq)
	convID := deterministicUUID("seedsearch-revconv-" + label)
	proposalID := deterministicUUID("seedsearch-revproposal-" + label)
	reviewID := deterministicUUID("seedsearch-review-" + label)

	rating := 4
	if roll := r.Intn(100); roll < 60 {
		rating = 5
	} else if roll < 85 {
		rating = 4
	} else if roll < 95 {
		rating = 3
	} else {
		rating = 1 + r.Intn(2)
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)
		 ON CONFLICT (id) DO NOTHING`, convID, client.orgID); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
		                        title, description, amount, status, organization_id)
		 VALUES ($1, $2, $3, $4, $3, $4, 'review-seed', 'scope', 1, 'completed', $5)
		 ON CONFLICT (id) DO NOTHING`,
		proposalID, convID, client.userID, providerUserID, client.orgID); err != nil {
		return err
	}
	// reviews has a UNIQUE (proposal_id, reviewer_id) constraint.
	// Use ON CONFLICT to guarantee idempotent reruns on the same seed.
	_, err := db.ExecContext(ctx,
		`INSERT INTO reviews (id, proposal_id, reviewer_id, reviewed_id,
		                      global_rating, timeliness, communication, quality, comment,
		                      reviewer_organization_id, reviewed_organization_id, side,
		                      moderation_status, moderation_score, title_visible,
		                      created_at, published_at)
		 VALUES ($1, $2, $3, $4, $5, $5, $5, $5, 'Clean work',
		         $6, $7, 'client_to_provider',
		         'clean', 0, true,
		         NOW() - ($8::text || ' days')::interval,
		         NOW() - ($8::text || ' days')::interval)
		 ON CONFLICT (proposal_id, reviewer_id) DO NOTHING`,
		reviewID, proposalID, client.userID, providerUserID, rating,
		client.orgID, providerOrgID, fmt.Sprint(ageDays))
	return err
}

// applyDisputeDistribution writes exactly one full_refund dispute
// against ~5 % of providers and one partial_refund against ~2 %.
// The rest get none. The dispute is attached to a freshly seeded
// proposal + milestone so the FK chain stays valid.
func applyDisputeDistribution(ctx context.Context, db *sql.DB, providerOrgID, providerUserID uuid.UUID, index int, clients []clientRef, r *rand.Rand) error {
	roll := r.Intn(100)
	var resolutionType string
	switch {
	case roll < 5:
		resolutionType = "full_refund"
	case roll < 7:
		resolutionType = "partial_refund"
	default:
		return nil // 93 % of providers never lose a dispute
	}

	client := clients[(index+7)%len(clients)]
	if client.orgID == providerOrgID {
		client = clients[(index+8)%len(clients)]
	}
	label := fmt.Sprintf("dispute-%d", index)
	convID := deterministicUUID("seedsearch-dispconv-" + label)
	proposalID := deterministicUUID("seedsearch-dispproposal-" + label)
	milestoneID := deterministicUUID("seedsearch-dispmilestone-" + label)
	disputeID := deterministicUUID("seedsearch-dispute-" + label)

	if _, err := db.ExecContext(ctx,
		`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)
		 ON CONFLICT (id) DO NOTHING`, convID, client.orgID); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO proposals (id, conversation_id, sender_id, recipient_id, client_id, provider_id,
		                        title, description, amount, status, organization_id)
		 VALUES ($1, $2, $3, $4, $3, $4, 'dispute-seed', 'scope', 100000, 'disputed', $5)
		 ON CONFLICT (id) DO NOTHING`,
		proposalID, convID, client.userID, providerUserID, client.orgID); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO proposal_milestones (id, proposal_id, sequence, title, description, amount, status)
		 VALUES ($1, $2, 1, 'disputed milestone', 'Scope', 100000, 'disputed')
		 ON CONFLICT (id) DO NOTHING`,
		milestoneID, proposalID); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO disputes (id, proposal_id, conversation_id,
		                       initiator_id, respondent_id,
		                       client_id, provider_id,
		                       client_organization_id, provider_organization_id,
		                       milestone_id,
		                       reason, description, requested_amount, proposal_amount,
		                       status, resolution_type, resolved_at)
		 VALUES ($1, $2, $3, $4, $5, $4, $5, $6, $7, $8,
		         'quality_issue', 'Seed — not real', 100000, 100000,
		         'resolved', $9, NOW())
		 ON CONFLICT (id) DO NOTHING`,
		disputeID, proposalID, convID, client.userID, providerUserID,
		client.orgID, providerOrgID, milestoneID, resolutionType)
	return err
}

// wipeRankingV1Signals removes all downstream seed data written by
// seedRankingV1Signals. Called from the main wipePreviousSeed flow
// so reruns are idempotent.
func wipeRankingV1Signals(ctx context.Context, db *sql.DB) error {
	// Use email-based filter to stay consistent with the rest of the
	// wipe logic: every row we inserted is anchored to one of the
	// seedsearch org IDs, whose owner user email ends with
	// @search.seed.
	const suffix = `%@search.seed`

	// Find all orgs tied to the seed.
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM organizations WHERE owner_user_id IN (
			SELECT id FROM users WHERE email LIKE $1
		)`, suffix)
	if err != nil {
		return err
	}
	defer rows.Close()
	var orgIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		orgIDs = append(orgIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(orgIDs) == 0 {
		return nil
	}

	// Cleanup in FK order: disputes → milestones → proposals →
	// reviews → conversations. `organization_id` on proposals is
	// the anchor because that column is always populated on rows
	// we insert. We cast to text[] so `ANY` works on the pq.Array
	// wrapper without requiring a uuid[] build step.
	orgStrings := make([]string, len(orgIDs))
	for i, id := range orgIDs {
		orgStrings[i] = id.String()
	}
	orgArray := pq.Array(orgStrings)

	cleanups := []string{
		`DELETE FROM disputes WHERE provider_organization_id::text = ANY($1::text[])
		                        OR client_organization_id::text = ANY($1::text[])`,
		`DELETE FROM reviews WHERE reviewer_organization_id::text = ANY($1::text[])
		                        OR reviewed_organization_id::text = ANY($1::text[])`,
		`DELETE FROM proposal_milestones WHERE proposal_id IN (
			SELECT id FROM proposals WHERE organization_id::text = ANY($1::text[]))`,
		`DELETE FROM proposals WHERE organization_id::text = ANY($1::text[])`,
		`DELETE FROM conversations WHERE organization_id::text = ANY($1::text[])`,
	}
	for _, q := range cleanups {
		if _, err := db.ExecContext(ctx, q, orgArray); err != nil {
			return fmt.Errorf("wipe ranking signals %q: %w", q, err)
		}
	}
	return nil
}
