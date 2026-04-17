package referrerprofile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	proposaldomain "marketplace-backend/internal/domain/proposal"
	referraldomain "marketplace-backend/internal/domain/referral"
	reviewdomain "marketplace-backend/internal/domain/review"
	userdomain "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// userDomain is a local alias so the helpers below don't drag the
// imported name across every signature. Kept private — the
// public surface is the ReputationDeps struct.
type userDomain = userdomain.User

// ReferrerReputation is the aggregated apporteur reputation view:
// a single rating stat pair computed across every reviewed attribution,
// plus a cursor-paginated "projets apportés" history of attributions.
//
// RatingAvg and ReviewCount are summary stats — they are returned once
// on the first page and do NOT change across pagination. Only History
// and NextCursor rotate as the caller pages through.
type ReferrerReputation struct {
	RatingAvg   float64
	ReviewCount int
	History     []ProjectHistoryEntry
	NextCursor  string
}

// ProjectHistoryEntry is one attributed mission in the referrer's
// history. Client identity is intentionally omitted — the B2B working
// relationship stays confidential, the apporteur reputation surface
// only exposes the provider side.
type ProjectHistoryEntry struct {
	ProposalID     uuid.UUID
	ProposalTitle  string
	ProposalStatus string
	ProviderID     uuid.UUID
	ProviderName   string
	Rating         *int
	Comment        string
	ReviewedAt     *time.Time
	CompletedAt    *time.Time
	AttributedAt   time.Time
}

// ReputationDeps groups the four repositories the reputation aggregate
// needs. All are required — a nil on any of them disables the surface
// (GetReferrerReputation returns an empty aggregate).
type ReputationDeps struct {
	Referrals repository.ReferralRepository
	Proposals repository.ProposalRepository
	Reviews   repository.ReviewRepository
	Users     repository.UserBatchReader
}

// WithReputationDeps attaches the reputation aggregate dependencies.
// Kept as a fluent builder so NewService's signature stays stable and
// the feature remains removable — wiring only appears here and in
// cmd/api/main.go.
func (s *Service) WithReputationDeps(deps ReputationDeps) *Service {
	if s == nil {
		return nil
	}
	clone := *s
	clone.referrals = deps.Referrals
	clone.proposals = deps.Proposals
	clone.reviews = deps.Reviews
	clone.users = deps.Users
	return &clone
}

const (
	// defaultHistoryLimit is the default page size for the history
	// slice. Mirrors the project-history default.
	defaultHistoryLimit = 20
	// maxHistoryLimit caps the page size so a hostile caller can't
	// request the full aggregate in one roundtrip.
	maxHistoryLimit = 50
)

// reputationCursor is the opaque pagination token for the history
// slice. Stored as base64(JSON) so it stays decoded/encoded in one
// place and the wire format stays forward-compatible.
type reputationCursor struct {
	// Either CompletedAt or AttributedAt is set — never both — mirroring
	// the two-stage ORDER BY (completed missions first, ongoing after).
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	AttributedAt time.Time  `json:"attributed_at"`
	ProposalID   uuid.UUID  `json:"proposal_id"`
}

// GetReferrerReputation assembles the apporteur reputation aggregate:
// summary rating + cursor-paginated attribution history. Runs in five
// batched queries total, no N+1 — the referrer's referrals load once,
// then attributions / proposals / provider users / reviews each load
// once with a single IN clause.
//
// The scope is naturally bounded by the referral exclusivity window:
// proposals only get attributed when a matching active referral exists
// on the (provider, client) couple at proposal-creation time, so this
// method simply iterates whatever the attributions table already
// contains for the referrer's referrals.
func (s *Service) GetReferrerReputation(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) (ReferrerReputation, error) {
	if s == nil || s.referrals == nil || s.proposals == nil || s.reviews == nil || s.users == nil {
		// Reputation deps not wired — return a defensive empty view
		// instead of panicking, so the feature can be disabled without
		// breaking the profile page.
		return ReferrerReputation{History: []ProjectHistoryEntry{}}, nil
	}
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}

	// TODO(reputation): paginate the aggregator when >10k referrals
	// become plausible. For V1 an apporteur has at most a few dozen
	// referrals — the full scan stays cheap.
	allEntries, ratingAvg, reviewCount, err := s.buildHistoryEntries(ctx, userID)
	if err != nil {
		return ReferrerReputation{}, err
	}

	sortHistoryEntries(allEntries)

	paged, nextCursor, err := paginateHistory(allEntries, cursorStr, limit)
	if err != nil {
		return ReferrerReputation{}, err
	}

	return ReferrerReputation{
		RatingAvg:   ratingAvg,
		ReviewCount: reviewCount,
		History:     paged,
		NextCursor:  nextCursor,
	}, nil
}

// buildHistoryEntries runs the five batch queries and assembles the
// full un-paginated entry list plus the summary stats.
func (s *Service) buildHistoryEntries(ctx context.Context, userID uuid.UUID) ([]ProjectHistoryEntry, float64, int, error) {
	referralIDs, err := s.collectReferralIDs(ctx, userID)
	if err != nil {
		return nil, 0, 0, err
	}
	if len(referralIDs) == 0 {
		return []ProjectHistoryEntry{}, 0, 0, nil
	}

	attributions, err := s.referrals.ListAttributionsByReferralIDs(ctx, referralIDs)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("reputation: list attributions: %w", err)
	}
	if len(attributions) == 0 {
		return []ProjectHistoryEntry{}, 0, 0, nil
	}

	proposalIDs := uniqueProposalIDs(attributions)
	proposals, err := s.proposals.GetByIDs(ctx, proposalIDs)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("reputation: get proposals: %w", err)
	}
	proposalByID := indexProposals(proposals)

	providerIDs := uniqueProviderIDs(attributions)
	users, err := s.users.GetByIDs(ctx, providerIDs)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("reputation: get providers: %w", err)
	}
	providerNameByID := indexProviderNames(users)

	reviewsByProposal, err := s.reviews.GetByProposalIDs(ctx, proposalIDs)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("reputation: get reviews: %w", err)
	}

	entries := make([]ProjectHistoryEntry, 0, len(attributions))
	var ratingSum int
	var ratingCount int
	for _, a := range attributions {
		prop := proposalByID[a.ProposalID]
		// Proposal may have been archived after attribution — skip to
		// keep the surface stable. Extremely rare in practice.
		if prop == nil {
			continue
		}
		entry := buildEntry(a, prop, providerNameByID[a.ProviderID])
		review := clientToProviderReview(reviewsByProposal, a.ProposalID)
		if review != nil {
			applyReview(&entry, review)
			if string(prop.Status) == string(proposaldomain.StatusCompleted) {
				ratingSum += review.GlobalRating
				ratingCount++
			}
		}
		entries = append(entries, entry)
	}

	ratingAvg := 0.0
	if ratingCount > 0 {
		ratingAvg = float64(ratingSum) / float64(ratingCount)
	}
	return entries, ratingAvg, ratingCount, nil
}

// collectReferralIDs loads every referral where userID is the apporteur
// and flattens the ids. Pagination is disabled here because we need the
// full set to compute the aggregate rating — see the V1 TODO above.
func (s *Service) collectReferralIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	const pageSize = 100
	var ids []uuid.UUID
	cursor := ""
	for {
		filter := repository.ReferralListFilter{Cursor: cursor, Limit: pageSize}
		referrals, next, err := s.referrals.ListByReferrer(ctx, userID, filter)
		if err != nil {
			return nil, fmt.Errorf("reputation: list referrals: %w", err)
		}
		for _, r := range referrals {
			ids = append(ids, r.ID)
		}
		if next == "" {
			return ids, nil
		}
		cursor = next
	}
}

// clientToProviderReview returns the one client→provider review for the
// given proposal, or nil if absent or in the wrong direction.
// GetByProposalIDs already filters by published+moderation but NOT by
// side — we enforce the direction here so a provider→client review
// cannot leak into the apporteur score.
func clientToProviderReview(reviews map[uuid.UUID]*reviewdomain.Review, proposalID uuid.UUID) *reviewdomain.Review {
	rv, ok := reviews[proposalID]
	if !ok || rv == nil {
		return nil
	}
	if rv.Side != reviewdomain.SideClientToProvider {
		return nil
	}
	return rv
}

// buildEntry converts an attribution+proposal pair into a history entry.
// Provider name is looked up defensively — a missing user row produces
// an empty string rather than an error so the list stays renderable.
func buildEntry(a *referraldomain.Attribution, prop *proposaldomain.Proposal, providerName string) ProjectHistoryEntry {
	return ProjectHistoryEntry{
		ProposalID:     a.ProposalID,
		ProposalTitle:  prop.Title,
		ProposalStatus: string(prop.Status),
		ProviderID:     a.ProviderID,
		ProviderName:   providerName,
		CompletedAt:    prop.CompletedAt,
		AttributedAt:   a.AttributedAt,
	}
}

// applyReview copies the review fields into the entry. Reviewed_at
// uses PublishedAt when set, otherwise CreatedAt — the public-facing
// timestamp is the reveal moment.
func applyReview(entry *ProjectHistoryEntry, rv *reviewdomain.Review) {
	rating := rv.GlobalRating
	entry.Rating = &rating
	entry.Comment = rv.Comment
	if rv.PublishedAt != nil {
		entry.ReviewedAt = rv.PublishedAt
	} else {
		reviewedAt := rv.CreatedAt
		entry.ReviewedAt = &reviewedAt
	}
}

// sortHistoryEntries orders by completed_at DESC (nulls last) then by
// attributed_at DESC. Matches the pattern used on the freelance
// project history surface.
func sortHistoryEntries(entries []ProjectHistoryEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		ci, cj := entries[i].CompletedAt, entries[j].CompletedAt
		switch {
		case ci != nil && cj != nil:
			if !ci.Equal(*cj) {
				return ci.After(*cj)
			}
		case ci != nil && cj == nil:
			return true
		case ci == nil && cj != nil:
			return false
		}
		if !entries[i].AttributedAt.Equal(entries[j].AttributedAt) {
			return entries[i].AttributedAt.After(entries[j].AttributedAt)
		}
		return entries[i].ProposalID.String() < entries[j].ProposalID.String()
	})
}

// paginateHistory slices the sorted entries by an opaque cursor. Runs
// purely in memory because the aggregate is already loaded in full;
// the cursor is still stable across calls because the sort is
// deterministic.
func paginateHistory(entries []ProjectHistoryEntry, cursorStr string, limit int) ([]ProjectHistoryEntry, string, error) {
	start := 0
	if cursorStr != "" {
		c, err := decodeReputationCursor(cursorStr)
		if err != nil {
			return nil, "", err
		}
		start = findCursorIndex(entries, c)
	}
	if start >= len(entries) {
		return []ProjectHistoryEntry{}, "", nil
	}
	end := start + limit
	hasMore := end < len(entries)
	if end > len(entries) {
		end = len(entries)
	}
	page := entries[start:end]
	nextCursor := ""
	if hasMore {
		nextCursor = encodeReputationCursor(page[len(page)-1])
	}
	// Copy the slice so callers can't accidentally mutate the backing
	// array. Cheap for page-sized slices.
	out := make([]ProjectHistoryEntry, len(page))
	copy(out, page)
	return out, nextCursor, nil
}

// findCursorIndex returns the index of the first entry STRICTLY AFTER
// the cursor. Uses the same sort key as sortHistoryEntries so the
// pagination walk is stable even if two entries share a timestamp.
func findCursorIndex(entries []ProjectHistoryEntry, c reputationCursor) int {
	for i, e := range entries {
		if entryMatchesCursor(e, c) {
			return i + 1
		}
	}
	// Cursor points to something we no longer have (entry was removed,
	// or the caller fabricated a cursor). Return len so we yield an
	// empty page rather than resetting to the start — resetting would
	// make callers see duplicates.
	return len(entries)
}

func entryMatchesCursor(e ProjectHistoryEntry, c reputationCursor) bool {
	if e.ProposalID != c.ProposalID {
		return false
	}
	// Equality on the timestamps guards against a hostile cursor
	// claiming an entry's ID with a different sort key — the walk
	// would then land in the wrong position.
	if c.CompletedAt == nil {
		if e.CompletedAt != nil {
			return false
		}
	} else {
		if e.CompletedAt == nil || !e.CompletedAt.Equal(*c.CompletedAt) {
			return false
		}
	}
	return e.AttributedAt.Equal(c.AttributedAt)
}

func encodeReputationCursor(e ProjectHistoryEntry) string {
	c := reputationCursor{
		CompletedAt:  e.CompletedAt,
		AttributedAt: e.AttributedAt,
		ProposalID:   e.ProposalID,
	}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

func decodeReputationCursor(s string) (reputationCursor, error) {
	var c reputationCursor
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return c, fmt.Errorf("decode cursor: invalid base64: %w", err)
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, fmt.Errorf("decode cursor: invalid json: %w", err)
	}
	return c, nil
}

func uniqueProposalIDs(attributions []*referraldomain.Attribution) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(attributions))
	ids := make([]uuid.UUID, 0, len(attributions))
	for _, a := range attributions {
		if _, ok := seen[a.ProposalID]; ok {
			continue
		}
		seen[a.ProposalID] = struct{}{}
		ids = append(ids, a.ProposalID)
	}
	return ids
}

func uniqueProviderIDs(attributions []*referraldomain.Attribution) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(attributions))
	ids := make([]uuid.UUID, 0, len(attributions))
	for _, a := range attributions {
		if _, ok := seen[a.ProviderID]; ok {
			continue
		}
		seen[a.ProviderID] = struct{}{}
		ids = append(ids, a.ProviderID)
	}
	return ids
}

func indexProposals(list []*proposaldomain.Proposal) map[uuid.UUID]*proposaldomain.Proposal {
	out := make(map[uuid.UUID]*proposaldomain.Proposal, len(list))
	for _, p := range list {
		out[p.ID] = p
	}
	return out
}

// indexProviderNames picks the best human-readable name for each user.
// Preference order: DisplayName → "FirstName LastName" → Email local
// part. An empty result is tolerated — the UI falls back to the
// provider's UUID when rendering.
func indexProviderNames(users []*userDomain) map[uuid.UUID]string {
	out := make(map[uuid.UUID]string, len(users))
	for _, u := range users {
		out[u.ID] = pickDisplayName(u)
	}
	return out
}

func pickDisplayName(u *userDomain) string {
	if u == nil {
		return ""
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.FirstName != "" || u.LastName != "" {
		if u.FirstName == "" {
			return u.LastName
		}
		if u.LastName == "" {
			return u.FirstName
		}
		return u.FirstName + " " + u.LastName
	}
	return u.Email
}
