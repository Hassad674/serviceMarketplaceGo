package search

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// indexer.go assembles a SearchDocument from the raw signals the
// search package needs without knowing about the internal layout of
// any feature package. We achieve the decoupling with two moves:
//
//  1. SearchDataRepository (below) is a narrow port the search
//     package OWNS — it lives here, not in the top-level port/
//     directory, because it is an implementation detail of the
//     search engine. Other packages never import it.
//
//  2. The adapter side of SearchDataRepository lives at
//     internal/adapter/postgres/search_document_repository.go. That
//     adapter is free to touch every feature's tables because
//     PostgreSQL is the common data plane — but the search app
//     layer never sees a feature package.
//
// Building one document involves aggregating data from 8 sources
// (profile row, organization row, pricing, skills, rating aggregate,
// earnings aggregate, KYC verification, last_active_at). The
// indexer fetches them all in parallel via errgroup so building
// one document stays under 200ms even on a slow dev DB.

// Raw* types are the plain-old-data structures the adapter populates
// and hands back to the indexer. They intentionally do not reference
// any domain entity so the search package remains feature-agnostic.

// RawActorSignals is everything the indexer needs to populate a
// SearchDocument for a single organization, pre-joined by the
// adapter. Persona is explicit on the struct so the same type can
// carry freelance, agency, or referrer payloads without branching
// inside the indexer.
type RawActorSignals struct {
	OrganizationID          uuid.UUID
	Persona                 Persona
	IsPublished             bool
	DisplayName             string
	Title                   string
	About                   string
	PhotoURL                string
	VideoURL                string
	City                    string
	CountryCode             string
	Latitude                *float64
	Longitude               *float64
	WorkMode                []string
	LanguagesProfessional   []string
	LanguagesConversational []string
	AvailabilityStatus      string
	ExpertiseDomains        []string
	SocialLinksCount        int
	LastActiveAt            time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// RawPricing is the flattened pricing row across personas. Each
// persona has its own pricing table in Postgres but the shape is
// identical enough to share here.
type RawPricing struct {
	Type       string
	MinAmount  *int64
	MaxAmount  *int64
	Currency   string
	Negotiable bool
	HasPricing bool // true when a pricing row actually exists
}

// RawRatingAggregate is the pre-computed (avg, count) pair from the
// reviews table. Zeroed when the actor has no reviews yet.
type RawRatingAggregate struct {
	Average float64
	Count   int
}

// RawEarningsAggregate is the pre-computed (sum, count) pair from
// the released proposal_milestones table.
type RawEarningsAggregate struct {
	TotalAmount       int64
	CompletedProjects int
}

// RawMessagingSignals covers the messaging-driven quality signals:
// response rate and the most recent activity timestamp. Both can be
// zero for inactive accounts.
type RawMessagingSignals struct {
	ResponseRate float64
}

// RawClientHistory captures the proven-work signals derived from
// released proposal milestones (phase 6B). Populated by the adapter
// via a single CTE query per actor. See docs/ranking-v1.md §3.2-4.
//
//   - UniqueClients counts distinct client organisations with ≥1
//     released milestone against the actor. Zero for actors that
//     have never been paid.
//   - RepeatClientRate is the share of unique_clients that returned
//     for ≥2 released projects. Always in [0, 1]. Zero when there
//     are no clients yet (guarded against division-by-zero at the
//     SQL layer).
type RawClientHistory struct {
	UniqueClients    int
	RepeatClientRate float64
}

// RawReviewDiversity captures the three reviewer-diversity signals
// extracted from the reviews table (phase 6B). See
// docs/ranking-v1.md §3.2-3.
//
//   - UniqueReviewers counts distinct reviewer users (not reviews).
//   - MaxReviewerShare is max(count per reviewer) / total_reviews.
//     Zero when there are no reviews; [0, 1] otherwise.
//   - ReviewRecencyFactor is the mean of exp(-age_days / 365) across
//     every review. Recent reviews dominate; 2-year-old reviews barely
//     contribute. Pre-computed at index time so the query hot path
//     never scans the full reviews table.
type RawReviewDiversity struct {
	UniqueReviewers     int
	MaxReviewerShare    float64
	ReviewRecencyFactor float64
}

// RawAccountAge captures the two "how mature is this account"
// signals (phase 6B):
//
//   - LostDisputes counts disputes resolved with a refund
//     (full or partial) where this organisation was the respondent.
//     Feeds negative_signals §5.3.
//   - AccountAgeDays is the integer number of days since the owner
//     user's users.created_at. Drives is_verified_mature §3.2-6
//     and account_age_bonus §3.2-9. Zero for a brand-new account.
type RawAccountAge struct {
	LostDisputes   int
	AccountAgeDays int
}

// SearchDataRepository is the one and only port the indexer depends
// on. It is intentionally coarse: one method per "shape of data"
// rather than one per column, so the Postgres adapter can implement
// each method as a single CTE-powered query and avoid N+1 traps.
type SearchDataRepository interface {
	// LoadActorSignals fetches the core profile + organization +
	// per-persona metadata for one actor. Returns an error if the
	// org does not exist or has no persona-specific row.
	LoadActorSignals(ctx context.Context, orgID uuid.UUID, persona Persona) (*RawActorSignals, error)

	// LoadSkills returns the canonical skill names associated with
	// the actor's persona (ordered as stored).
	LoadSkills(ctx context.Context, orgID uuid.UUID, persona Persona) ([]string, error)

	// LoadPricing returns the persona-specific pricing row, or
	// RawPricing{HasPricing: false} when none has been set.
	LoadPricing(ctx context.Context, orgID uuid.UUID, persona Persona) (*RawPricing, error)

	// LoadRatingAggregate returns the avg+count of completed
	// reviews the actor has received. Zero values if none.
	LoadRatingAggregate(ctx context.Context, orgID uuid.UUID) (*RawRatingAggregate, error)

	// LoadEarningsAggregate returns the total amount + count of
	// released milestones where the actor was the provider.
	LoadEarningsAggregate(ctx context.Context, orgID uuid.UUID) (*RawEarningsAggregate, error)

	// LoadVerificationStatus reports whether the actor has passed
	// KYC (any `approved` kyc_verifications row for the org).
	LoadVerificationStatus(ctx context.Context, orgID uuid.UUID) (bool, error)

	// LoadMessagingSignals computes messaging-driven quality
	// indicators. Phase 1 only populates ResponseRate; future
	// phases may expand the struct without breaking callers.
	LoadMessagingSignals(ctx context.Context, orgID uuid.UUID) (*RawMessagingSignals, error)

	// LoadClientHistory computes unique_clients + repeat_client_rate
	// from released proposal milestones (phase 6B, docs/ranking-v1.md
	// §3.2-4). Returns zero values for actors with no history.
	LoadClientHistory(ctx context.Context, orgID uuid.UUID) (*RawClientHistory, error)

	// LoadReviewDiversity computes unique_reviewers + max_reviewer_share
	// + review_recency_factor from the reviews table (phase 6B,
	// docs/ranking-v1.md §3.2-3). Returns zero values for actors with
	// no reviews.
	LoadReviewDiversity(ctx context.Context, orgID uuid.UUID) (*RawReviewDiversity, error)

	// LoadAccountAge computes lost_disputes_count + account_age_days
	// for one organisation (phase 6B, docs/ranking-v1.md §3.2-6,
	// §3.2-9, §5.3). Zero disputes + zero age for orgs without a
	// traceable owner user (should not happen outside test fixtures).
	LoadAccountAge(ctx context.Context, orgID uuid.UUID) (*RawAccountAge, error)
}

// Indexer converts raw signals into a SearchDocument. Separate from
// the repository so the repository can be swapped (e.g. for a test
// fake) without touching the ranking logic, and so the ranking
// logic can be unit-tested with synthetic raw signals.
type Indexer struct {
	repo       SearchDataRepository
	embedder   EmbeddingsClient
	isFeatured func(orgID uuid.UUID) bool
}

// IndexerOption mutates an Indexer during construction. Exposed so
// callers can opt into the "featured override" hook without a
// bigger constructor signature.
type IndexerOption func(*Indexer)

// WithFeaturedOverride installs a predicate that decides whether
// the `is_featured` boolean on the document is true. Phase 1 does
// not have admin-managed featured flags yet, so the default is
// "never featured"; the hook is wired so phase 4 can plug in a
// Postgres lookup without touching the indexer API.
func WithFeaturedOverride(fn func(orgID uuid.UUID) bool) IndexerOption {
	return func(i *Indexer) { i.isFeatured = fn }
}

// NewIndexer builds an indexer from a repository and an embeddings
// client. Both are required — we refuse to silently inject a nil
// embedder because that would surface as an opaque nil-pointer
// panic deep inside BuildDocument.
func NewIndexer(repo SearchDataRepository, embedder EmbeddingsClient, opts ...IndexerOption) (*Indexer, error) {
	if repo == nil {
		return nil, fmt.Errorf("search indexer: repository is required")
	}
	if embedder == nil {
		return nil, fmt.Errorf("search indexer: embeddings client is required")
	}
	idx := &Indexer{repo: repo, embedder: embedder, isFeatured: defaultNotFeatured}
	for _, opt := range opts {
		opt(idx)
	}
	return idx, nil
}

// defaultNotFeatured is the fall-back predicate when the caller does
// not wire a featured-override — always returns false.
func defaultNotFeatured(_ uuid.UUID) bool { return false }

// indexAggregate holds the concurrent fan-in results of one
// BuildDocument call. Declared as a top-level type so it can be
// passed between the small helper functions below without inflating
// any single function's parameter count.
type indexAggregate struct {
	signals   *RawActorSignals
	skills    []string
	pricing   *RawPricing
	rating    *RawRatingAggregate
	earnings  *RawEarningsAggregate
	kyc       bool
	messaging *RawMessagingSignals
	embed     []float32

	// Ranking V1 aggregates (phase 6B).
	clientHistory   *RawClientHistory
	reviewDiversity *RawReviewDiversity
	accountAge      *RawAccountAge
}

// loadResult is the channel message type used by the fan-in. Named
// so collectResults has a typed signature instead of an anonymous
// struct.
type loadResult struct {
	name string
	err  error
}

// BuildDocument assembles a SearchDocument from the repository's raw
// signals. It runs the seven repo reads concurrently — all are
// independent of each other — and then converts the result into the
// final document via pure functions from ranking.go.
//
// The concurrency model uses a goroutine per read plus a small
// channel-based fan-in pattern that aborts on the first error. We
// avoid `golang.org/x/sync/errgroup` to keep the search package's
// dependency graph minimal.
func (i *Indexer) BuildDocument(ctx context.Context, orgID uuid.UUID, persona Persona) (*SearchDocument, error) {
	if !persona.IsValid() {
		return nil, fmt.Errorf("search indexer: invalid persona %q", persona)
	}
	var agg indexAggregate
	if err := i.fanOutLoad(ctx, orgID, persona, &agg); err != nil {
		return nil, err
	}
	return i.assembleDocument(&agg, persona)
}

// fanOutLoad runs the repo + embeddings calls concurrently. Extracted
// from BuildDocument so the control flow stays linear and the 50-line
// function limit is respected.
//
// Signals + skills are fetched first (sequentially) because both are
// inputs to the embedding text — the embedding goroutine cannot start
// until we know the skills list. Everything else runs in parallel via
// goroutines that push into a buffered channel.
func (i *Indexer) fanOutLoad(ctx context.Context, orgID uuid.UUID, persona Persona, agg *indexAggregate) error {
	signals, err := i.repo.LoadActorSignals(ctx, orgID, persona)
	if err != nil {
		return fmt.Errorf("load actor signals: %w", err)
	}
	agg.signals = signals

	skills, err := i.repo.LoadSkills(ctx, orgID, persona)
	if err != nil {
		return fmt.Errorf("load skills: %w", err)
	}
	agg.skills = skills

	// 9 concurrent reads: 6 legacy + 3 ranking V1 aggregates.
	const parallelReads = 9
	results := make(chan loadResult, parallelReads)

	go func() {
		pricing, err := i.repo.LoadPricing(ctx, orgID, persona)
		agg.pricing = pricing
		results <- loadResult{"pricing", err}
	}()
	go func() {
		rating, err := i.repo.LoadRatingAggregate(ctx, orgID)
		agg.rating = rating
		results <- loadResult{"rating", err}
	}()
	go func() {
		earnings, err := i.repo.LoadEarningsAggregate(ctx, orgID)
		agg.earnings = earnings
		results <- loadResult{"earnings", err}
	}()
	go func() {
		ok, err := i.repo.LoadVerificationStatus(ctx, orgID)
		agg.kyc = ok
		results <- loadResult{"kyc", err}
	}()
	go func() {
		msg, err := i.repo.LoadMessagingSignals(ctx, orgID)
		agg.messaging = msg
		results <- loadResult{"messaging", err}
	}()
	go func() {
		vec, err := i.embedActor(ctx, agg.signals, agg.skills)
		agg.embed = vec
		results <- loadResult{"embedding", err}
	}()
	go func() {
		history, err := i.repo.LoadClientHistory(ctx, orgID)
		agg.clientHistory = history
		results <- loadResult{"client_history", err}
	}()
	go func() {
		diversity, err := i.repo.LoadReviewDiversity(ctx, orgID)
		agg.reviewDiversity = diversity
		results <- loadResult{"review_diversity", err}
	}()
	go func() {
		age, err := i.repo.LoadAccountAge(ctx, orgID)
		agg.accountAge = age
		results <- loadResult{"account_age", err}
	}()

	return collectResults(results, parallelReads)
}
