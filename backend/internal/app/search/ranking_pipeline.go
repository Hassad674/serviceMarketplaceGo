package search

import (
	"context"
	"strings"
	"time"
	"unicode"

	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// ranking_pipeline.go composes the four Stage 2-5 ranking packages
// into the concrete pipeline wired into the query service.
//
// Stage 2 (feature extraction) + Stage 3 (anti-gaming) + Stage 4
// (composite scoring) + Stage 5 (business rules) execute in order on
// the candidate list returned by Typesense's hybrid retrieval.
//
// The pipeline is a PURE function of its inputs — no I/O, no
// database, no embedder calls. Every dependency is injected at
// construction time and the Rerank call itself is deterministic when
// the rules config's RandSeed is non-zero.
//
// See docs/ranking-v1.md §2.1 for the end-to-end diagram.

// RankingPipeline is the concrete pipeline object. All fields are
// set once at construction time and read-only afterwards so the
// same instance is safe to share across goroutines. The per-call
// randomness + mutation lives on local values inside Rerank (via
// BusinessRules.Apply) so the pipeline itself carries no state.
type RankingPipeline struct {
	extractor  features.Extractor
	antigaming *antigaming.Pipeline
	scorer     scorer.Reranker
	rules      *rules.BusinessRules
}

// NewRankingPipeline wires the four Stage 2-5 packages together. All
// arguments must be non-nil — the pipeline has no meaningful zero
// value. Callers that want to opt out of re-ranking pass nil through
// to appsearch.Service and the service keeps Typesense's raw order.
func NewRankingPipeline(
	extractor features.Extractor,
	ag *antigaming.Pipeline,
	rer scorer.Reranker,
	br *rules.BusinessRules,
) *RankingPipeline {
	return &RankingPipeline{
		extractor:  extractor,
		antigaming: ag,
		scorer:     rer,
		rules:      br,
	}
}

// RankInput is the per-request payload consumed by Rerank.
//
// Query is the raw + tokenised query plus the active persona. The
// pipeline builds a features.Query out of it so the extractor stays
// dependency-free.
//
// Hits is the ordered list of Typesense hits. The pipeline never
// resizes this slice past the TopN boundary inside
// BusinessRules.Apply but the caller must treat the slice as
// consumed after Rerank returns.
//
// Now is the reference instant used by the last-active + account-age
// extractors. Injecting it keeps the pipeline deterministic under
// test — two calls with identical inputs produce identical outputs.
type RankInput struct {
	Query   features.Query
	Persona features.Persona
	Hits    []TypesenseHit
	Now     time.Time
}

// RankedCandidate is the per-hit output of Rerank. The Candidate is
// the full business-rules view (feature vector + score + availability
// tier) ; RawDoc is preserved so the handler can still emit the
// SearchDocument DTO it produced before re-ranking landed.
type RankedCandidate struct {
	Candidate rules.Candidate
	RawDoc    TypesenseHit
}

// Rerank runs Stage 2 → 3 → 4 → 5 and returns at most TopN
// candidates in business-rule order.
//
// The function is defensively guarded against nil pipeline
// components — a freshly constructed RankingPipeline can be called
// safely from day-one code. Missing pieces degrade silently :
//   - nil extractor  → every feature is zero, scorer outputs zero
//     composites, rules return the input order unchanged.
//   - nil antigaming → penalties are not applied (scorer still runs).
//   - nil scorer     → every RankedScore is zero but rules still run
//     so tier + availability ordering survives.
//   - nil rules      → returns the scored candidates verbatim
//     without applying diversity / rising talent / noise.
//
// Empty Hits returns a non-nil, zero-length slice.
func (p *RankingPipeline) Rerank(ctx context.Context, in RankInput) []RankedCandidate {
	if p == nil {
		return nil
	}
	if len(in.Hits) == 0 {
		return []RankedCandidate{}
	}
	nowUnix := p.resolveNow(in).Unix()

	// Stages 2-4: feature extraction → anti-gaming → composite scoring.
	candidates := p.scoreCandidates(ctx, in, nowUnix)

	// Stage 5: business rules (tier sort, noise, diversity, rising
	// talent, featured, truncate).
	reranked := p.applyRules(ctx, candidates, in.Persona)

	return p.zipWithRaws(reranked, in.Hits)
}

// resolveNow returns the reference instant used by the last-active
// + account-age extractors. Falls back to rankingNow() when the
// caller left in.Now as the zero value.
func (p *RankingPipeline) resolveNow(in RankInput) time.Time {
	if in.Now.IsZero() {
		return rankingNow()
	}
	return in.Now
}

// scoreCandidates runs Stages 2 → 3 → 4 for every hit and returns the
// rules-ready Candidate slice. Split out of Rerank to keep the call
// site short enough to audit in a single page.
func (p *RankingPipeline) scoreCandidates(ctx context.Context, in RankInput, nowUnix int64) []rules.Candidate {
	candidates := make([]rules.Candidate, 0, len(in.Hits))
	for _, hit := range in.Hits {
		lite := hit.ToSearchDocumentLite(nowUnix)
		feat := p.extract(in.Query, lite)
		p.applyAntiGaming(ctx, &feat, lite, hit, nowUnix)
		score := p.score(ctx, in.Query, feat, in.Persona)
		candidates = append(candidates, rules.Candidate{
			DocumentID:         hit.Document.ID,
			OrganizationID:     hit.Document.OrganizationID,
			Persona:            rules.Persona(hit.Document.Persona),
			Feat:               feat,
			Score:              score,
			AvailabilityStatus: hit.Document.AvailabilityStatus,
			PrimarySkill:       primarySkillOf(hit.Document.Skills),
			AccountAgeDays:     int(hit.Document.AccountAgeDays),
			IsFeatured:         hit.Document.IsFeatured,
			IsVerified:         hit.Document.IsVerified,
		})
	}
	return candidates
}

// zipWithRaws pairs each reranked Candidate back with its TypesenseHit.
// Apply may reorder and truncate so a positional mapping no longer
// works — we index by DocumentID instead.
func (p *RankingPipeline) zipWithRaws(reranked []rules.Candidate, raws []TypesenseHit) []RankedCandidate {
	rawByID := make(map[string]TypesenseHit, len(raws))
	for _, h := range raws {
		rawByID[h.Document.ID] = h
	}
	out := make([]RankedCandidate, 0, len(reranked))
	for _, c := range reranked {
		out = append(out, RankedCandidate{
			Candidate: c,
			RawDoc:    rawByID[c.DocumentID],
		})
	}
	return out
}

// extract is the nil-safe wrapper around the feature extractor. A
// nil extractor returns a zero Features — the rest of the pipeline
// still runs on a well-formed but empty vector.
func (p *RankingPipeline) extract(q features.Query, lite features.SearchDocumentLite) features.Features {
	if p.extractor == nil {
		return features.Features{}
	}
	return p.extractor.Extract(q, lite)
}

// applyAntiGaming builds the RawSignals bundle from the lite doc +
// the query and runs the five rules. A nil pipeline is a silent
// no-op — useful for tests and for opting individual features out
// of the anti-gaming layer without tearing out the rerank.
func (p *RankingPipeline) applyAntiGaming(
	ctx context.Context,
	f *features.Features,
	lite features.SearchDocumentLite,
	hit TypesenseHit,
	nowUnix int64,
) {
	if p.antigaming == nil {
		return
	}
	raw := antigaming.RawSignals{
		ProfileID:              hit.Document.OrganizationID,
		Persona:                lite.Persona,
		Text:                   strings.ToLower(strings.TrimSpace(lite.SkillsText)),
		RecentReviewTimestamps: nil, // populated once the adapter lands
		TotalReviewCount:       int(hit.Document.RatingCount),
		ReviewerIDs:            nil, // populated by linked-account detector
		NowUnix:                nowUnix,
		AccountAgeDays:         int(lite.AccountAgeDays),
	}
	p.antigaming.Apply(ctx, f, raw)
}

// score is the nil-safe wrapper around the reranker.
func (p *RankingPipeline) score(
	ctx context.Context,
	q features.Query,
	f features.Features,
	persona features.Persona,
) scorer.RankedScore {
	if p.scorer == nil {
		return scorer.RankedScore{}
	}
	return p.scorer.Score(ctx, q, f, persona)
}

// applyRules is the nil-safe wrapper around the business rules. When
// rules are missing we still truncate at the TopN default so callers
// always receive a bounded slice.
func (p *RankingPipeline) applyRules(
	ctx context.Context,
	candidates []rules.Candidate,
	persona features.Persona,
) []rules.Candidate {
	if p.rules == nil {
		if len(candidates) > defaultPipelineTopN {
			return candidates[:defaultPipelineTopN]
		}
		return candidates
	}
	return p.rules.Apply(ctx, candidates, persona)
}

// defaultPipelineTopN mirrors rules.DefaultConfig().TopN. Declared
// here so the nil-rules fallback still respects the 20-candidate
// window documented in docs/ranking-v1.md §8.
const defaultPipelineTopN = 20

// primarySkillOf returns the first non-empty skill after trimming.
// The diversity rule reads this field; an empty slice means the
// candidate cannot collide with any other in the diversity check.
func primarySkillOf(skills []string) string {
	for _, s := range skills {
		t := strings.TrimSpace(s)
		if t != "" {
			return t
		}
	}
	return ""
}

// NormaliseTokens splits a query string into the lowercased, de-duplicated
// token list expected by features.Query.NormalisedTokens. Whitespace +
// punctuation are the separators; empty tokens are dropped; order is
// preserved (first-seen). Exposed for the query service and for tests
// that construct features.Query directly.
//
// Keeping the implementation here (rather than inside features/) lets
// the features package stay dependency-free while the query layer can
// evolve the tokenisation rules over time without rippling into the
// extractor package.
func NormaliseTokens(raw string) []string {
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	if len(fields) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}
