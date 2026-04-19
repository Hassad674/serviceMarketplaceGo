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
)

// Persona identifies which weight table to apply. Mirrored locally
// pending the arrival of internal/search/features (R2-F). Once the
// features package lands, this local alias is replaced in a follow-up
// commit by a re-export of features.Persona. Keep the string values
// byte-identical with the features package so the swap is a no-op.
type Persona string

// Persona values mirror the three public personas defined in
// docs/ranking-v1.md §4. The string literals are the canonical wire
// identifiers used across the search pipeline (filter_by, logs,
// env-var suffixes).
const (
	PersonaFreelance Persona = "freelance"
	PersonaAgency    Persona = "agency"
	PersonaReferrer  Persona = "referrer"
)

// Query captures the ranking inputs a scorer may legitimately inspect.
// Mirrors internal/search/features.Query. Today only Text is read — the
// empty-query branch of §5.2 keys off it. Other fields are carried so
// the signature matches the LTR-ready extension path in §13a.
type Query struct {
	Text string
}

// Features is the 10-field feature vector produced by Stage 2 plus the
// raw anti-gaming counters. Mirrors internal/search/features.Features
// one-for-one so the future rename swap is mechanical. The scorer only
// reads the normalised [0, 1] floats plus NegativeSignals; the Raw*
// fields are untouched here (anti-gaming consumed them upstream in
// Stage 3) and exist for downstream explainability logs.
type Features struct {
	// Normalised features in [0, 1] (IsVerifiedMature is {0, 1}).
	TextMatchScore      float64
	SkillsOverlapRatio  float64
	RatingScoreDiverse  float64
	ProvenWorkScore     float64
	ResponseRate        float64
	IsVerifiedMature    float64
	ProfileCompletion   float64
	LastActiveDaysScore float64
	AccountAgeBonus     float64

	// NegativeSignals is the dispute-driven penalty applied multiplicatively
	// to the positive composite (see §5.3). Bounded to [0, 0.30].
	NegativeSignals float64

	// Raw signals — NOT read by the scorer. Present for logging and
	// for the LTR export path. Names mirror features.Features.
	RawTextMatchBucket  int
	RawUniqueReviewers  int
	RawMaxReviewerShare float64
	RawLostDisputes     int
	RawAccountAgeDays   int
}

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
