package search

import (
	"testing"

	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// audit_fixtures_test.go owns the shared scaffolding used by every
// audit scenario:
//   - Time anchor + days-ago helper for deterministic activity math.
//   - Helpers to look up fixtures by ID and to project them into
//     TypesenseHit slices the rerank pipeline consumes.
//   - newAuditPipeline: a fully-wired RankingPipeline with a pinned
//     RandSeed and a recording antigaming logger.
//
// The actual fixture corpora live next door, one persona per file:
//   - audit_fixtures_freelance_test.go
//   - audit_fixtures_agency_test.go
//   - audit_fixtures_referrer_test.go
//
// Why split per persona: each fixture group is ~270 lines on its own;
// keeping them in a single file would breach the 600-line cap from
// CLAUDE.md and make per-persona diffs harder to review.

// auditNowUnix is the reference instant used by every audit
// scenario. Pinned to 2026-05-01 12:00:00 UTC so test outputs are
// reproducible across machines + git history. Choosing a date in
// the past prevents drift when this file is read many years from
// now.
const auditNowUnix int64 = 1_777_766_400 // 2026-05-01T12:00:00Z

// daysAgoUnix returns a Unix timestamp `days` days before
// auditNowUnix. The audit uses days as the readable unit because
// every spec formula (§3.2-8 last_active, §3.2-9 account_age) is
// defined in days.
func daysAgoUnix(days int) int64 {
	return auditNowUnix - int64(days)*86400
}

// findFixture returns the fixture with the given Document ID.
// Used by scenario assertions that name a profile by ID rather
// than scanning slices manually.
func findFixture(t *testing.T, all []search.SearchDocument, id string) search.SearchDocument {
	t.Helper()
	for _, d := range all {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("fixture not found: %q", id)
	return search.SearchDocument{}
}

// allFixturesByID returns the union of every persona fixture keyed
// by Document ID. Audit assertions that need to look up a profile
// across the whole catalogue use this helper.
func allFixturesByID() map[string]search.SearchDocument {
	out := make(map[string]search.SearchDocument, 30)
	for _, d := range freelanceFixtures {
		out[d.ID] = d
	}
	for _, d := range agencyFixtures {
		out[d.ID] = d
	}
	for _, d := range referrerFixtures {
		out[d.ID] = d
	}
	return out
}

// fixturesForPersona returns the slice associated with a given
// persona. Tests use it to pick the right cohort without an inline
// switch.
func fixturesForPersona(persona search.Persona) []search.SearchDocument {
	switch persona {
	case search.PersonaFreelance:
		return freelanceFixtures
	case search.PersonaAgency:
		return agencyFixtures
	case search.PersonaReferrer:
		return referrerFixtures
	}
	return nil
}

// auditPipeline builds a fully-configured RankingPipeline using the
// production defaults but with a deterministic RandSeed so audit
// scenarios produce stable orderings across CI runs.
//
// The recording logger is exposed so scenarios that assert on
// fired anti-gaming penalties can read the captured events.
type auditPipeline struct {
	pipeline *RankingPipeline
	agLogger *antigaming.RecordingLogger
}

// newAuditPipeline returns a fresh pipeline + logger pair. The
// logger is recreated per call so tests stay independent — sharing
// the same RecordingLogger between scenarios would cause cross-test
// pollution.
func newAuditPipeline(t *testing.T) *auditPipeline {
	t.Helper()
	fcfg := features.DefaultConfig()
	agCfg := antigaming.DefaultConfig()
	scCfg := scorer.DefaultConfig()
	require.NoError(t, scCfg.Validate(), "default scorer config must validate")
	rlCfg := rules.DefaultConfig()
	rlCfg.RandSeed = 7 // pinned across audit suite
	logger := &antigaming.RecordingLogger{}

	ext := features.NewDefaultExtractor(fcfg)
	ag := antigaming.NewPipeline(agCfg, antigaming.NoopLinkedReviewersDetector{}, logger)
	rer := scorer.NewWeightedScorer(scCfg)
	br := rules.NewBusinessRules(rlCfg)
	return &auditPipeline{
		pipeline: NewRankingPipeline(ext, ag, rer, br),
		agLogger: logger,
	}
}

// hitsFromFixtures wraps the fixtures into TypesenseHit entries.
// `bucketByID` is an explicit map of doc-ID → text-match bucket so
// scenarios can simulate exactly what Typesense would compute for a
// given query without invoking the real engine.
//
// IDs missing from `bucketByID` get bucket 0 (the empty-query path).
func hitsFromFixtures(docs []search.SearchDocument, bucketByID map[string]int) []TypesenseHit {
	out := make([]TypesenseHit, 0, len(docs))
	for _, d := range docs {
		out = append(out, TypesenseHit{
			Document:        d,
			TextMatchBucket: bucketByID[d.ID],
		})
	}
	return out
}

// uniformBuckets returns a bucket map assigning the same bucket to
// every fixture in the slice. Useful for "no-text-match-bias"
// scenarios where we want every other ranking signal to drive the
// order.
func uniformBuckets(docs []search.SearchDocument, bucket int) map[string]int {
	out := make(map[string]int, len(docs))
	for _, d := range docs {
		out[d.ID] = bucket
	}
	return out
}
