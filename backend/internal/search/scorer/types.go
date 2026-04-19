// Package scorer implements Stage 4 of the Ranking V1 pipeline
// (composite scoring). It turns a 9-feature vector produced by
// internal/search/features into a RankedScore triple (Base, Adjusted,
// Final) plus a per-feature breakdown for explainability.
//
// The package is intentionally free of I/O: Score is pure arithmetic
// suitable for being called 200 times per search request under a 50ms
// p95 budget. See docs/ranking-v1.md §4 (weights) and §5 (composite
// formula) for the normative specification.
package scorer

import (
	"context"

	"marketplace-backend/internal/search/features"
)

// Persona is a type alias for features.Persona. Declared here so callers
// can keep importing scorer without also importing features, while the
// feature package stays the single source of truth for the three
// canonical persona identifiers used across the ranking pipeline.
type Persona = features.Persona

// Re-exported persona constants — byte-identical to features.Persona*.
// Keeping them here means every ranking stage can use the shortest
// import path for its local concerns without leaking the layering.
const (
	PersonaFreelance = features.PersonaFreelance
	PersonaAgency    = features.PersonaAgency
	PersonaReferrer  = features.PersonaReferrer
)

// Query is a type alias for features.Query. The scorer only reads
// Query.Text today (for the empty-query redistribution branch of §5.2);
// the other fields (NormalisedTokens, FilterSkills, Persona) are
// carried through unchanged so future LTR scorers can consume them
// without a signature change.
type Query = features.Query

// Features is a type alias for features.Features. The scorer reads the
// nine normalised [0, 1] floats plus NegativeSignals. The Raw* fields
// are passed through for downstream explainability logs — they are
// never summed into the score.
type Features = features.Features

// RankedScore is the triple emitted per candidate. Base is the purely
// positive composite, Adjusted is Base × (1 − NegativeSignals), Final
// is Adjusted scaled to [0, 100] for the UI. Breakdown carries every
// feature's per-persona contribution so an admin dashboard can explain
// why a profile ranked the way it did.
type RankedScore struct {
	Base      float64            // [0, 1]
	Adjusted  float64            // [0, 1]
	Final     float64            // [0, 100]
	Breakdown map[string]float64 // per-feature contributions
}

// Reranker is the pluggable scoring contract. V1 is WeightedScorer;
// V2 will be LTRScorer with identical signature (see §9.3).
type Reranker interface {
	Score(ctx context.Context, q Query, f Features, persona Persona) RankedScore
}

// Breakdown keys — stable string identifiers used for explainability
// logs and the future admin dashboard. Changing any of these is a
// breaking change for downstream consumers and the LTR training
// pipeline.
const (
	BreakdownTextMatch      = "text_match"
	BreakdownSkillsOverlap  = "skills_overlap"
	BreakdownRating         = "rating"
	BreakdownProvenWork     = "proven_work"
	BreakdownResponseRate   = "response_rate"
	BreakdownVerifiedMature = "verified_mature"
	BreakdownCompletion     = "completion"
	BreakdownLastActive     = "last_active"
	BreakdownAccountAge     = "account_age"
)
