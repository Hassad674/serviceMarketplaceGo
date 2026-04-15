package search

import (
	"context"
	"fmt"
	"strings"
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
}

// Indexer converts raw signals into a SearchDocument. Separate from
// the repository so the repository can be swapped (e.g. for a test
// fake) without touching the ranking logic, and so the ranking
// logic can be unit-tested with synthetic raw signals.
type Indexer struct {
	repo      SearchDataRepository
	embedder  EmbeddingsClient
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
// The signals load is sequential because the embedding input
// depends on its output; everything else runs in parallel via
// goroutines that push into a buffered channel.
func (i *Indexer) fanOutLoad(ctx context.Context, orgID uuid.UUID, persona Persona, agg *indexAggregate) error {
	signals, err := i.repo.LoadActorSignals(ctx, orgID, persona)
	if err != nil {
		return fmt.Errorf("load actor signals: %w", err)
	}
	agg.signals = signals

	results := make(chan loadResult, 7)

	go func() {
		skills, err := i.repo.LoadSkills(ctx, orgID, persona)
		agg.skills = skills
		results <- loadResult{"skills", err}
	}()
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
		vec, err := i.embedActor(ctx, agg.signals)
		agg.embed = vec
		results <- loadResult{"embedding", err}
	}()

	return collectResults(results, 7)
}

// collectResults drains the results channel and returns the first
// non-nil error, wrapped with the step name for observability.
func collectResults(results chan loadResult, expected int) error {
	var firstErr error
	for k := 0; k < expected; k++ {
		r := <-results
		if r.err != nil && firstErr == nil {
			firstErr = fmt.Errorf("indexer %s: %w", r.name, r.err)
		}
	}
	return firstErr
}

// embedActor builds the text input passed to the embeddings API and
// returns the resulting vector. Extracted so BuildDocument stays
// short and so the text-composition rules can be unit-tested.
func (i *Indexer) embedActor(ctx context.Context, s *RawActorSignals) ([]float32, error) {
	text := ComposeEmbeddingText(s)
	if text == "" {
		// An entirely empty profile has no meaningful vector;
		// rather than send blank text to OpenAI we skip the
		// embedding. The document will still index — just without
		// semantic search coverage — and the next profile update
		// will re-trigger the reindex with a populated text.
		return nil, nil
	}
	return i.embedder.Embed(ctx, text)
}

// ComposeEmbeddingText is the exported helper that converts raw
// profile signals into the text input for the embeddings API. Kept
// exported + pure so golden tests can replay the exact same input.
func ComposeEmbeddingText(s *RawActorSignals) string {
	if s == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if v := strings.TrimSpace(s.Title); v != "" {
		parts = append(parts, v)
	}
	if v := strings.TrimSpace(s.About); v != "" {
		parts = append(parts, v)
	}
	if len(s.ExpertiseDomains) > 0 {
		parts = append(parts, strings.Join(s.ExpertiseDomains, ", "))
	}
	return strings.TrimSpace(strings.Join(parts, ". "))
}

// assembleDocument builds the SearchDocument from the populated
// aggregate. Every numerical feature goes through the ranking.go
// helpers so the ranking formulas have a single source of truth.
func (i *Indexer) assembleDocument(agg *indexAggregate, persona Persona) (*SearchDocument, error) {
	s := agg.signals
	if s == nil {
		return nil, fmt.Errorf("search indexer: signals are nil")
	}

	completion := ProfileCompletionScore(CompletionInput{
		HasPhoto:         s.PhotoURL != "",
		HasAbout:         strings.TrimSpace(s.About) != "",
		HasTitle:         strings.TrimSpace(s.Title) != "",
		HasVideo:         s.VideoURL != "",
		ExpertiseCount:   len(s.ExpertiseDomains),
		SkillsCount:      len(agg.skills),
		HasPricing:       agg.pricing != nil && agg.pricing.HasPricing,
		HasLocation:      s.City != "" && s.CountryCode != "",
		SocialLinksCount: s.SocialLinksCount,
		LanguagesCount:   len(s.LanguagesProfessional),
	})

	doc := &SearchDocument{
		ID:                      s.OrganizationID.String(),
		Persona:                 persona,
		IsPublished:             s.IsPublished,
		DisplayName:             s.DisplayName,
		Title:                   s.Title,
		PhotoURL:                s.PhotoURL,
		City:                    s.City,
		CountryCode:             s.CountryCode,
		WorkMode:                nilToEmpty(s.WorkMode),
		LanguagesProfessional:   nilToEmpty(s.LanguagesProfessional),
		LanguagesConversational: nilToEmpty(s.LanguagesConversational),
		AvailabilityStatus:      s.AvailabilityStatus,
		AvailabilityPriority:    AvailabilityPriority(s.AvailabilityStatus),
		ExpertiseDomains:        nilToEmpty(s.ExpertiseDomains),
		Skills:                  nilToEmpty(agg.skills),
		SkillsText:              strings.Join(agg.skills, " "),
		ProfileCompletionScore:  int32(completion),
		LastActiveAt:            s.LastActiveAt.Unix(),
		IsFeatured:              i.isFeatured(s.OrganizationID),
	}

	applyLocation(doc, s)
	applyPricing(doc, agg.pricing)
	applyRating(doc, agg.rating)
	applyEarnings(doc, agg.earnings)
	applyMessaging(doc, agg.messaging)
	doc.IsVerified = agg.kyc

	if agg.embed != nil {
		doc.Embedding = agg.embed
	}
	doc.SetTimestamps(s.CreatedAt, s.UpdatedAt)

	if err := doc.Validate(); err != nil {
		return nil, fmt.Errorf("search indexer: assembled document invalid: %w", err)
	}
	return doc, nil
}

// applyLocation copies the geopoint into the document if both
// coordinates are present. Typesense expects `[lat, lng]` — any
// other order breaks the geo filter.
func applyLocation(doc *SearchDocument, s *RawActorSignals) {
	if s.Latitude == nil || s.Longitude == nil {
		return
	}
	doc.Location = []float64{*s.Latitude, *s.Longitude}
}

// applyPricing flattens a RawPricing into the document's pricing
// fields. When no pricing exists, the fields stay zero/empty and
// the JSON omitempty tags keep them out of the payload.
func applyPricing(doc *SearchDocument, p *RawPricing) {
	if p == nil || !p.HasPricing {
		return
	}
	doc.PricingType = p.Type
	doc.PricingMinAmount = p.MinAmount
	doc.PricingMaxAmount = p.MaxAmount
	doc.PricingCurrency = p.Currency
	doc.PricingNegotiable = p.Negotiable
}

// applyRating runs the Bayesian rating through the ranking helper
// and records the derived top-rated badge.
func applyRating(doc *SearchDocument, r *RawRatingAggregate) {
	if r == nil {
		return
	}
	doc.RatingAverage = r.Average
	doc.RatingCount = int32(r.Count)
	doc.RatingScore = BayesianRatingScore(r.Average, r.Count)
	doc.IsTopRated = IsTopRated(r.Average, r.Count)
}

// applyEarnings records the two earnings signals.
func applyEarnings(doc *SearchDocument, e *RawEarningsAggregate) {
	if e == nil {
		return
	}
	doc.TotalEarned = e.TotalAmount
	doc.CompletedProjects = int32(e.CompletedProjects)
}

// applyMessaging records the messaging-driven response rate. Nil is
// valid (inactive account) and maps to zero.
func applyMessaging(doc *SearchDocument, m *RawMessagingSignals) {
	if m == nil {
		return
	}
	doc.ResponseRate = m.ResponseRate
}

// nilToEmpty turns a nil slice into an empty slice so the serialised
// JSON payload uses `[]` instead of `null`. Typesense accepts both,
// but `[]` makes the wire format easier to diff in integration
// tests.
func nilToEmpty[T any](in []T) []T {
	if in == nil {
		return []T{}
	}
	return in
}
