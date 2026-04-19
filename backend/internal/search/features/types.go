// Package features implements Stage 2 of the ranking V1 pipeline described in
// `docs/ranking-v1.md` §3 — Feature Extraction.
//
// Scope
//
// Given a user query and a Typesense search document, the package produces a
// Features value : a ten-component scoring vector whose every coordinate lives
// in [0, 1] (plus a bounded penalty term). Extractors are pure functions of
// (Query, SearchDocumentLite, Config) — no I/O, no mutable package-level state,
// no database. The only external input is the immutable Config loaded once at
// startup from environment variables.
//
// Each extractor has a one-to-one mapping with a section of
// `docs/ranking-v1.md` §3.2 :
//
//   - TextMatchScore       §3.2-1
//   - SkillsOverlapRatio   §3.2-2
//   - RatingScoreDiverse   §3.2-3 (Bayesian × diversity × recency + cold-start floor)
//   - ProvenWorkScore      §3.2-4
//   - ResponseRate         §3.2-5
//   - IsVerifiedMature     §3.2-6
//   - ProfileCompletion    §3.2-7
//   - LastActiveDaysScore  §3.2-8
//   - AccountAgeBonus      §3.2-9
//   - NegativeSignals      §5.3 (bounded penalty in [0, DisputePenaltyCap])
//
// Contract for downstream agents
//
// The Features struct is a frozen contract for Round 2 of the ranking rollout.
// The scorer (R2-S) and the anti-gaming pipeline (this agent, next module)
// both consume the exact shape declared below. Changing a field type, renaming
// a field, or removing a raw signal is a breaking change and must be discussed
// with the orchestrator before landing.
//
// Performance
//
// Extractors are allocation-light and run fully in-memory. Target latency on
// a single document is < 2 µs so the re-ranking of 200 candidates stays well
// under the 50 ms p95 budget carved out for Stages 2-5 in §2.1.
package features

// Persona identifies which weight table the scorer should apply. The three
// values are the exact wire strings used throughout the codebase — see
// `internal/search/schema.go` for the canonical Persona type on the search
// side. We redeclare the constants here to keep the features package
// dependency-free (it must remain importable by the scorer without dragging
// the whole search package along).
type Persona string

const (
	PersonaFreelance Persona = "freelance"
	PersonaAgency    Persona = "agency"
	PersonaReferrer  Persona = "referrer"
)

// IsValid reports whether the persona is one of the three recognised values.
// Used by the extractor entrypoint to fail fast on malformed input.
func (p Persona) IsValid() bool {
	switch p {
	case PersonaFreelance, PersonaAgency, PersonaReferrer:
		return true
	}
	return false
}

// Query is the per-request input fed to every extractor.
//
// Text is the raw query string exactly as it arrived from the user. It is used
// for logging + for the empty-query detection path in the scorer.
//
// NormalisedTokens is the canonical lowercased and de-duplicated token list of
// Text — built once at query boundary so extractors never re-tokenise. Must
// stay deterministic (order preserved) so property tests can compare runs.
//
// FilterSkills is the set of skill-chip values selected from the sidebar.
// Combined with NormalisedTokens via union to produce the query-side skill
// set consumed by SkillsOverlapRatio.
//
// Persona is the persona of the listing page that fired the query. Extractors
// that behave differently per persona (e.g. skills_overlap_ratio returning 0
// for referrers) branch on this field.
type Query struct {
	Text             string
	NormalisedTokens []string
	FilterSkills     []string
	Persona          Persona
}

// Features is the ten-component scoring vector produced per (query, doc) pair.
//
// Every positive-contribution field is guaranteed to be in [0, 1]. The single
// penalty term — NegativeSignals — is guaranteed to be in [0, DisputePenaltyCap]
// (default 0.30).
//
// The Raw* fields expose a few signals in their un-normalised form so the
// anti-gaming pipeline and the explainability UI can read them without
// re-querying the document. They are never summed into the score directly.
type Features struct {
	// Positive contributions (weighted sum ∈ [0, 1])
	TextMatchScore      float64
	SkillsOverlapRatio  float64
	RatingScoreDiverse  float64
	ProvenWorkScore     float64
	ResponseRate        float64
	IsVerifiedMature    float64 // {0, 1}
	ProfileCompletion   float64
	LastActiveDaysScore float64
	AccountAgeBonus     float64

	// Multiplicative penalty ∈ [0, DisputePenaltyCap] (default 0.30).
	NegativeSignals float64

	// Raw signals surfaced for the anti-gaming pipeline + explainability UI.
	RawTextMatchBucket  int
	RawUniqueReviewers  int
	RawMaxReviewerShare float64
	RawLostDisputes     int
	RawAccountAgeDays   int
}

// SearchDocumentLite is a read-only copy of the subset of
// `search.SearchDocument` fields the extractors depend on. Keeping it local
// preserves feature-package independence : the scorer, the indexer, and the
// anti-gaming pipeline can import features without pulling the entire search
// package (which itself depends on the Typesense client, the embeddings
// client, etc.).
//
// The scorer converts a `search.SearchDocument` to a `SearchDocumentLite`
// via a single helper in its own package. This avoids a circular dependency
// between `features` and `search`.
type SearchDocumentLite struct {
	// Identity
	OrganizationID string
	Persona        Persona

	// Expertise / content used by text-match + skills-overlap extractors.
	Skills     []string
	SkillsText string
	About      string

	// Quality signals feeding ratings / proven-work / response / verified /
	// completion / activity / age / negative extractors.
	RatingAverage          float64
	RatingCount            int32
	CompletedProjects      int32
	ProfileCompletionScore int32
	LastActiveAt           int64 // Unix epoch seconds, 0 if unknown
	ResponseRate           float64
	IsVerified             bool

	// Ranking V1 signals from phase 6B.
	UniqueClientsCount   int32
	RepeatClientRate     float64
	UniqueReviewersCount int32
	MaxReviewerShare     float64
	ReviewRecencyFactor  float64
	LostDisputesCount    int32
	AccountAgeDays       int32

	// NowUnix is the request-time reference used by the activity extractor.
	// Kept on the doc copy (rather than fetched inside the extractor) so the
	// features are a pure function of their inputs — critical for
	// deterministic tests + future LTR training.
	NowUnix int64

	// TextMatchBucket is the [0, 10] bucketed score Typesense returns via
	// `_text_match` when `buckets:10` is requested. The query layer fills
	// this field during candidate retrieval. Zero when the value is unknown
	// (empty-query path or non-matching candidate).
	TextMatchBucket int
}

// Extractor is the interface the composite Extract entrypoint satisfies. Kept
// for the benefit of the scorer + future LTR swap — the reranker can depend on
// an Extractor interface and inject a different implementation under test.
type Extractor interface {
	Extract(query Query, doc SearchDocumentLite) Features
}
