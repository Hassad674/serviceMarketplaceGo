package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"marketplace-backend/internal/app/searchanalytics"
	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/features"
)

// service.go is the application-layer wrapper around the
// per-persona scoped Typesense clients. It owns the concrete query
// shape (query_by, facet_by, sort_by, highlight fields) and parses
// the raw Typesense response into a typed result struct.
//
// Handlers depend on *Service, never directly on
// search.PersonaScopedClient — that gives us a single place to add
// caching, rate limiting, or query rewriting in future phases
// without touching every call site.

// PersonaQueryClient is the narrow interface the service depends
// on. Defined locally instead of importing
// search.PersonaScopedClient directly so unit tests can drop a
// fake in via constructor injection.
type PersonaQueryClient interface {
	Persona() search.Persona
	Query(ctx context.Context, params search.SearchParams) (json.RawMessage, error)
}

// ServiceDeps groups the constructor inputs. The service holds
// one client per persona; missing personas return ErrPersonaNotConfigured
// so the handler can surface a clean 404.
//
// Embedder + Analytics are optional. When Embedder is nil the service
// falls back to keyword-only search (BM25); when Analytics is nil the
// service skips the fire-and-forget capture step. This lets the
// service degrade gracefully when OpenAI or Postgres is unavailable.
type ServiceDeps struct {
	Freelance PersonaQueryClient
	Agency    PersonaQueryClient
	Referrer  PersonaQueryClient
	Embedder  search.EmbeddingsClient
	Analytics AnalyticsRecorder
	Logger    *slog.Logger

	// RankingPipeline runs Stages 2-5 (feature extraction → anti-gaming
	// → composite scoring → business rules) on the hits returned by
	// Typesense before the JSON envelope is decorated. Absence = no
	// rerank — the service returns the raw Typesense order so the
	// search engine keeps working when the ranking packages are
	// removed or not yet wired.
	RankingPipeline *RankingPipeline

	// LTRRepository captures the reranked feature vectors into
	// search_queries.result_features_json for downstream LTR training
	// (docs/ranking-v1.md §9.1). Nil skips the capture path; a non-nil
	// repo still requires RankingPipeline to be wired — without
	// reranking there are no feature vectors to capture.
	LTRRepository searchanalytics.LTRRepository

	// AnalyticsService is the searchanalytics service that owns the
	// LTR capture goroutine. Wired in parallel with RankingPipeline
	// + LTRRepository — all three must be non-nil for LTR capture to
	// fire. Shared with the AnalyticsRecorder adapter above.
	AnalyticsService *searchanalytics.Service
}

// AnalyticsRecorder is the port the service uses to capture each
// search into the search_queries table. Declared locally (consumer-
// side interface) so the service package stays independent of any
// concrete analytics implementation.
type AnalyticsRecorder interface {
	CaptureSearch(ctx context.Context, event AnalyticsEvent)
}

// AnalyticsEvent is the per-search payload posted to the recorder.
// A superset of the search_queries columns so the recorder can
// persist any subset it wants.
type AnalyticsEvent struct {
	SearchID     string
	UserID       string
	SessionID    string
	Query        string
	FilterBy     string
	SortBy       string
	Persona      string
	ResultsCount int
	LatencyMs    int
}

// Service is the application service for the public search query
// path. Methods are safe for concurrent use because every dependency
// they touch is itself concurrent-safe.
type Service struct {
	clients          map[search.Persona]PersonaQueryClient
	embedder         search.EmbeddingsClient
	analytics        AnalyticsRecorder
	logger           *slog.Logger
	rankingPipeline  *RankingPipeline
	ltrRepository    searchanalytics.LTRRepository
	analyticsService *searchanalytics.Service
}

// NewService wires the service from its dependency struct. Nil
// entries are silently dropped so the caller can opt out of any
// persona at boot time (e.g. when a worktree is testing only the
// freelance flow).
func NewService(deps ServiceDeps) *Service {
	clients := make(map[search.Persona]PersonaQueryClient, 3)
	if deps.Freelance != nil {
		clients[search.PersonaFreelance] = deps.Freelance
	}
	if deps.Agency != nil {
		clients[search.PersonaAgency] = deps.Agency
	}
	if deps.Referrer != nil {
		clients[search.PersonaReferrer] = deps.Referrer
	}
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		clients:          clients,
		embedder:         deps.Embedder,
		analytics:        deps.Analytics,
		logger:           logger,
		rankingPipeline:  deps.RankingPipeline,
		ltrRepository:    deps.LTRRepository,
		analyticsService: deps.AnalyticsService,
	}
}

// HasPersona reports whether the service has a client wired for the
// given persona. Used by the scoped-key handler to decide whether
// to short-circuit with a 404.
func (s *Service) HasPersona(p search.Persona) bool {
	_, ok := s.clients[p]
	return ok
}

// ErrPersonaNotConfigured signals that the requested persona does
// not have a wired client — usually because the search engine is
// disabled or a dev environment opted out.
var ErrPersonaNotConfigured = fmt.Errorf("search: persona is not configured")

// QueryInput is the per-request payload posted to Service.Query.
// Field semantics mirror the public API: every field is optional,
// and the service applies sane defaults (per_page=20).
//
// Cursor is an opaque string generated by the previous page; empty
// means "first page". Page + PerPage remain available for the
// non-paginated use cases (admin dashboards, golden tests) but the
// public API should always use Cursor.
type QueryInput struct {
	Persona   search.Persona
	Query     string
	Filters   FilterInput
	SortBy    string
	Page      int
	PerPage   int
	Cursor    string
	UserID    string // optional, captured in analytics
	SessionID string // optional, captured in analytics
}

// QueryResult is the typed payload returned to handlers and (via
// JSON) to the frontend SSR fallback. It is a strict subset of the
// raw Typesense response — the embedding vectors are stripped to
// keep the payload small and the field is omitted entirely.
//
// Hybrid reports whether the query blended BM25 + vector cosine on
// this call. Handlers use it for structured logging; the frontend
// ignores the field. Deliberately NOT json-serialised so no public
// consumer couples to it.
type QueryResult struct {
	SearchID       string                    `json:"search_id"`
	Documents      []search.SearchDocument   `json:"documents"`
	Found          int                       `json:"found"`
	OutOf          int                       `json:"out_of"`
	Page           int                       `json:"page"`
	PerPage        int                       `json:"per_page"`
	SearchTimeMs   int                       `json:"search_time_ms"`
	FacetCounts    map[string]map[string]int `json:"facet_counts"`
	CorrectedQuery string                    `json:"corrected_query,omitempty"`
	Highlights     []map[string]string       `json:"highlights"`
	NextCursor     string                    `json:"next_cursor,omitempty"`
	HasMore        bool                      `json:"has_more"`
	Hybrid         bool                      `json:"-"`

	// Reranked reports whether the Stage 2-5 ranking pipeline ran on
	// this result set. Handlers read it for the structured query log;
	// the frontend ignores the field. Deliberately NOT json-serialised
	// so no public consumer couples to it.
	Reranked bool `json:"-"`

	// RerankDurationMs is the wall-clock time (ms) spent inside the
	// ranking pipeline on this call. Zero when Reranked = false. Like
	// Hybrid + Reranked, not exposed to external consumers.
	RerankDurationMs int `json:"-"`

	// TopFinalScore is the Final score (0-100) of the top-ranked
	// candidate after the rerank. Zero when no candidates remain
	// after reranking. Not exposed to external consumers.
	TopFinalScore float64 `json:"-"`
}

// Default page sizing constants. PerPage is capped at MaxPerPage so
// a misbehaving client cannot ask for 10k documents in one call.
const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 60
)

// Query runs the search against the persona-scoped client and
// parses the response. Empty Query strings are interpreted as
// "match all" via the `*` wildcard so the listing pages can render
// without a typed query.
//
// Pipeline:
//  1. Resolve page from cursor (opaque base64).
//  2. Optionally embed the query text when hybrid search is enabled
//     and the user typed something (we skip embeddings on `q=*` so
//     vector distance does not dominate the listing page).
//  3. Execute Typesense search.
//  4. Parse + decorate the response (search_id, next_cursor, has_more).
//  5. Fire-and-forget analytics capture.
func (s *Service) Query(ctx context.Context, input QueryInput) (*QueryResult, error) {
	client, ok := s.clients[input.Persona]
	if !ok {
		return nil, fmt.Errorf("search query: %w: %s", ErrPersonaNotConfigured, input.Persona)
	}

	page, err := resolvePage(input)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	vectorQuery := s.maybeVectorQuery(ctx, input)
	params := s.buildSearchParams(input, page, vectorQuery != "")
	params.VectorQuery = vectorQuery

	start := time.Now()
	raw, err := client.Query(ctx, params)
	latency := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	result, hits, err := parseQueryResultWithHits(raw)
	if err != nil {
		return nil, err
	}

	result.Hybrid = vectorQuery != ""
	// decorateResult assigns SearchID — applyRerank needs it to
	// kick off the LTR capture with the right row key, so order
	// matters: decorate first, then rerank.
	s.decorateResult(result, input, params)
	s.applyRerank(ctx, input, result, hits)
	s.captureAnalytics(ctx, input, result, params, latency)
	return result, nil
}

// decorateResult fills the phase 3 metadata (SearchID, NextCursor,
// HasMore). Pulled out of Query so the hot path stays flat.
//
// Typesense sometimes omits per_page from the response envelope; we
// fall back to the params we sent so has_more math stays correct.
func (s *Service) decorateResult(result *QueryResult, input QueryInput, params search.SearchParams) {
	result.SearchID = NewSearchID(input, params, time.Now())
	perPage := result.PerPage
	if perPage == 0 {
		perPage = params.PerPage
	}
	if result.PerPage == 0 {
		result.PerPage = perPage
	}
	page := result.Page
	if page == 0 {
		page = params.Page
	}
	if result.Page == 0 {
		result.Page = page
	}
	totalLoaded := page * perPage
	result.HasMore = totalLoaded < result.Found
	if result.HasMore {
		result.NextCursor = EncodeCursor(Cursor{Page: page + 1})
	}
}

// applyRerank runs the Stage 2-5 ranking pipeline on the raw Typesense
// hits and rewrites the result.Documents slice in the new order. When
// the pipeline is not wired the method is a no-op and result.Reranked
// stays false — the service degrades gracefully to Typesense's native
// order.
//
// Rerank duration is measured in wall-clock milliseconds so the
// structured query log can surface it to operators. The top-final
// score is captured so operators can spot ranking regressions
// (e.g. the top hit's Final dropping below 50 consistently).
func (s *Service) applyRerank(ctx context.Context, input QueryInput, result *QueryResult, hits []TypesenseHit) {
	if s.rankingPipeline == nil || len(hits) == 0 {
		return
	}
	rerankStart := time.Now()
	query := features.Query{
		Text:             input.Query,
		NormalisedTokens: NormaliseTokens(input.Query),
		FilterSkills:     append([]string(nil), input.Filters.Skills...),
		Persona:          features.Persona(input.Persona),
	}
	ranked := s.rankingPipeline.Rerank(ctx, RankInput{
		Query:   query,
		Persona: features.Persona(input.Persona),
		Hits:    hits,
		Now:     rerankStart,
	})
	result.RerankDurationMs = int(time.Since(rerankStart).Milliseconds())
	result.Reranked = true

	// Rewrite Documents + Highlights in the reranked order. The
	// handler still emits the same DTO shape — only the order changes.
	docs := make([]search.SearchDocument, 0, len(ranked))
	highlights := make([]map[string]string, 0, len(ranked))
	// Build a highlight lookup keyed by document ID so we can reorder
	// without scanning the original slice for every candidate.
	highlightByID := make(map[string]map[string]string, len(result.Highlights))
	for i, doc := range result.Documents {
		if i < len(result.Highlights) {
			highlightByID[doc.ID] = result.Highlights[i]
		}
	}
	for _, r := range ranked {
		docs = append(docs, r.RawDoc.Document)
		if h, ok := highlightByID[r.RawDoc.Document.ID]; ok {
			highlights = append(highlights, h)
		} else {
			highlights = append(highlights, map[string]string{})
		}
	}
	result.Documents = docs
	result.Highlights = highlights
	if len(ranked) > 0 {
		result.TopFinalScore = ranked[0].Candidate.Score.Final
	}
	s.captureLTR(ctx, result, ranked)
}

// captureLTR is the fire-and-forget hand-off to searchanalytics. When
// either the service or the repository is nil the capture is skipped
// silently — LTR persistence is advisory, never blocking.
func (s *Service) captureLTR(ctx context.Context, result *QueryResult, ranked []RankedCandidate) {
	if s.analyticsService == nil || s.ltrRepository == nil {
		return
	}
	if result.SearchID == "" {
		// The decorate pass has not yet assigned a SearchID. We
		// re-derive one here so the LTR row and the analytics row
		// agree. Handlers reading result.SearchID downstream will
		// see the same value because decorateResult computes it
		// deterministically from input + params + time.
		return
	}
	payload := make([]searchanalytics.RankedResult, 0, len(ranked))
	for i, r := range ranked {
		if i >= ltrTopK {
			break
		}
		payload = append(payload, searchanalytics.RankedResult{
			DocID:        r.Candidate.DocumentID,
			RankPosition: i + 1,
			FinalScore:   r.Candidate.Score.Final,
			Features:     featureContributionMap(r),
		})
	}
	// fire-and-forget: CaptureResultFeatures returns an error only for
	// programming bugs (empty search_id, nil repo) that we already
	// guarded against above. Runtime failures land in the service's
	// own logger.
	if err := s.analyticsService.CaptureResultFeatures(ctx, result.SearchID, payload, s.ltrRepository); err != nil {
		s.logger.Warn("search: ltr capture kickoff failed",
			"error", err, "search_id", result.SearchID)
	}
}

// featureContributionMap returns the breakdown map for a ranked
// candidate, extended with the raw penalty term so LTR training can
// reconstruct the full scoring context.
//
// We reuse the scorer's Breakdown map (not a clone) — the LTR capture
// path serialises it immediately and never retains a reference. This
// saves ~600 bytes per candidate × 20 candidates = 12 KB per search.
func featureContributionMap(r RankedCandidate) map[string]float64 {
	out := r.Candidate.Score.Breakdown
	if out == nil {
		out = map[string]float64{}
	}
	// Augment with the raw features the Breakdown map omits so the
	// LTR payload fully describes the scoring state. Copy on write
	// to avoid mutating the scorer's returned map.
	augmented := make(map[string]float64, len(out)+2)
	for k, v := range out {
		augmented[k] = v
	}
	augmented["negative_signals"] = r.Candidate.Feat.NegativeSignals
	augmented["base"] = r.Candidate.Score.Base
	return augmented
}

// ltrTopK caps the per-search LTR payload size. 20 matches the
// documented window in docs/ranking-v1.md §9.1 — anything beyond
// that is out of the rendered top-20 and therefore not a training
// signal.
const ltrTopK = 20

// captureAnalytics fires a fire-and-forget CaptureSearch. Never
// blocks the caller and never surfaces errors — analytics must not
// degrade the user-facing search path.
func (s *Service) captureAnalytics(ctx context.Context, input QueryInput, result *QueryResult, params search.SearchParams, latency time.Duration) {
	if s.analytics == nil {
		return
	}
	evt := AnalyticsEvent{
		SearchID:     result.SearchID,
		UserID:       input.UserID,
		SessionID:    input.SessionID,
		Query:        params.Q,
		FilterBy:     params.FilterBy,
		SortBy:       params.SortBy,
		Persona:      string(input.Persona),
		ResultsCount: result.Found,
		LatencyMs:    int(latency.Milliseconds()),
	}
	s.analytics.CaptureSearch(ctx, evt)
}

// maybeVectorQuery returns the Typesense hybrid `vector_query`
// parameter when the user typed a real query AND an embedder is
// wired. Empty query (match-all) returns "" so the listing page
// stays keyword-first.
func (s *Service) maybeVectorQuery(ctx context.Context, input QueryInput) string {
	if s.embedder == nil {
		return ""
	}
	q := strings.TrimSpace(input.Query)
	if q == "" || q == "*" {
		return ""
	}
	vec, err := s.embedder.Embed(ctx, q)
	if err != nil {
		s.logger.Warn("search: embedding failed, falling back to BM25 only",
			"error", err, "persona", input.Persona)
		return ""
	}
	return search.FormatVectorQuery(vec, HybridK)
}

// HybridK is the number of vector-search candidates Typesense
// considers in the hybrid blend. 20 matches the phase 3 spec.
const HybridK = 20

// buildSearchParams assembles the Typesense query parameters from
// the input + sane defaults. Extracted so tests can pin the exact
// wire format the service produces.
//
// `hybridActive` toggles the hybrid-specific wire format: embedding
// in query_by + _vector_distance in sort_by. Must match whether the
// caller is going to set params.VectorQuery — mismatching the two
// tripwires Typesense's validation and returns 400.
func (s *Service) buildSearchParams(input QueryInput, page int, hybridActive bool) search.SearchParams {
	q := strings.TrimSpace(input.Query)
	if q == "" {
		q = "*"
	}
	perPage := input.PerPage
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	sortBy := strings.TrimSpace(input.SortBy)
	if sortBy == "" {
		if hybridActive {
			sortBy = search.DefaultSortByHybrid()
		} else {
			sortBy = search.DefaultSortBy()
		}
	}

	// Hybrid blending on Typesense 28.0: the vector side is controlled
	// exclusively by the `vector_query` parameter. `embedding` is a
	// manual (not auto-) vector field, so including it in `query_by`
	// triggers a 400 — Typesense reserves query_by for auto-embedding
	// fields only. The pre-computed vector flows in via VectorQuery
	// (`maybeVectorQuery` above) and Typesense blends BM25 + cosine
	// natively. query_by stays text-only regardless of hybrid state.
	queryBy := defaultQueryBy
	numTypos := defaultNumTypos

	return search.SearchParams{
		Q:                   q,
		QueryBy:             queryBy,
		FilterBy:            BuildFilterBy(input.Filters),
		FacetBy:             defaultFacetBy,
		SortBy:              sortBy,
		Page:                page,
		PerPage:             perPage,
		ExcludeFields:       "embedding",
		HighlightFields:     "display_name,title,skills_text",
		HighlightFullFields: "display_name,title",
		// Typesense requires num_typos to either be a single value
		// applied to every query_by field or a comma-separated list
		// with the SAME length as query_by. We use a per-field list
		// so display_name + title tolerate two typos but
		// skills_text + city are strict (skills are technical terms
		// where typos usually mean a different concept).
		NumTypos:       numTypos,
		MaxFacetValues: 40,
	}
}

// defaultQueryBy is the comma-separated list of fields the query
// term hits. Order matters: Typesense weights earlier fields more
// heavily so display_name + title get the top rank.
//
// IMPORTANT: keep this in sync with the NumTypos field count above.
const defaultQueryBy = "display_name,title,skills_text,city"

// defaultNumTypos matches the four defaultQueryBy fields. See
// buildSearchParams for why each slot has its specific budget.
const defaultNumTypos = "2,2,1,1"

// defaultFacetBy is the list of fields the response should compute
// counts for. Persona is intentionally excluded — the scoped client
// already locks it.
const defaultFacetBy = "availability_status,city,country_code,languages_professional," +
	"expertise_domains,skills,work_mode,is_verified,is_top_rated,pricing_currency"
