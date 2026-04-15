package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"marketplace-backend/internal/search"
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
type ServiceDeps struct {
	Freelance PersonaQueryClient
	Agency    PersonaQueryClient
	Referrer  PersonaQueryClient
}

// Service is the application service for the public search query
// path. Methods are safe for concurrent use because every dependency
// they touch is itself concurrent-safe.
type Service struct {
	clients map[search.Persona]PersonaQueryClient
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
	return &Service{clients: clients}
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
// and the service applies sane defaults (page=1, per_page=20).
type QueryInput struct {
	Persona search.Persona
	Query   string
	Filters FilterInput
	SortBy  string
	Page    int
	PerPage int
}

// QueryResult is the typed payload returned to handlers and (via
// JSON) to the frontend SSR fallback. It is a strict subset of the
// raw Typesense response — the embedding vectors are stripped to
// keep the payload small and the field is omitted entirely.
type QueryResult struct {
	Documents      []search.SearchDocument        `json:"documents"`
	Found          int                            `json:"found"`
	OutOf          int                            `json:"out_of"`
	Page           int                            `json:"page"`
	PerPage        int                            `json:"per_page"`
	SearchTimeMs   int                            `json:"search_time_ms"`
	FacetCounts    map[string]map[string]int      `json:"facet_counts"`
	CorrectedQuery string                         `json:"corrected_query,omitempty"`
	Highlights     []map[string]string            `json:"highlights"`
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
func (s *Service) Query(ctx context.Context, input QueryInput) (*QueryResult, error) {
	client, ok := s.clients[input.Persona]
	if !ok {
		return nil, fmt.Errorf("search query: %w: %s", ErrPersonaNotConfigured, input.Persona)
	}

	params := s.buildSearchParams(input)
	raw, err := client.Query(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	return parseQueryResult(raw)
}

// buildSearchParams assembles the Typesense query parameters from
// the input + sane defaults. Extracted so tests can pin the exact
// wire format the service produces.
func (s *Service) buildSearchParams(input QueryInput) search.SearchParams {
	q := strings.TrimSpace(input.Query)
	if q == "" {
		q = "*"
	}
	page := input.Page
	if page <= 0 {
		page = DefaultPage
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
		sortBy = search.DefaultSortBy()
	}

	return search.SearchParams{
		Q:                   q,
		QueryBy:             defaultQueryBy,
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
		NumTypos:       "2,2,1,1",
		MaxFacetValues: 40,
	}
}

// defaultQueryBy is the comma-separated list of fields the query
// term hits. Order matters: Typesense weights earlier fields more
// heavily so display_name + title get the top rank.
//
// IMPORTANT: keep this in sync with the NumTypos field count above.
const defaultQueryBy = "display_name,title,skills_text,city"

// defaultFacetBy is the list of fields the response should compute
// counts for. Persona is intentionally excluded — the scoped client
// already locks it.
const defaultFacetBy = "availability_status,city,country_code,languages_professional," +
	"expertise_domains,skills,work_mode,is_verified,is_top_rated,pricing_currency"
