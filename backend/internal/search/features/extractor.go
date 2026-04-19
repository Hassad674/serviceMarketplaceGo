package features

// ExtractorFunc adapts a bare function into the Extractor interface. Used by
// the scorer when it wants to inject a stub implementation under test.
type ExtractorFunc func(query Query, doc SearchDocumentLite) Features

// Extract implements Extractor on ExtractorFunc.
func (f ExtractorFunc) Extract(query Query, doc SearchDocumentLite) Features {
	return f(query, doc)
}

// DefaultExtractor is the production extractor. It holds the Config handed
// down at startup and dispatches each feature to its dedicated extract_*.go
// function. The struct is immutable after construction so it is safe to share
// across goroutines without locking.
type DefaultExtractor struct {
	cfg Config
}

// NewDefaultExtractor returns a ready-to-use extractor. Panics are avoided —
// an invalid Config still produces deterministic zeros + the validation
// pipeline will have rejected the config at startup.
func NewDefaultExtractor(cfg Config) *DefaultExtractor {
	return &DefaultExtractor{cfg: cfg}
}

// Extract runs all ten feature extractors + the penalty term on (query, doc)
// and returns the composite Features struct. The result is a pure function of
// its inputs — no I/O, no randomness, no package-level state — which is the
// property the property tests rely on.
//
// Ordering of the calls is irrelevant (each extractor is independent) but the
// code stays in the spec-order defined in §3.1 so future readers can grep
// through the file top-to-bottom + find the formula matching the doc.
func (e *DefaultExtractor) Extract(query Query, doc SearchDocumentLite) Features {
	textScore, bucket := ExtractTextMatch(query, doc, e.cfg)
	penalty, rawDisputes := ExtractNegativeSignals(doc, e.cfg)

	return Features{
		TextMatchScore:      textScore,
		SkillsOverlapRatio:  ExtractSkillsOverlap(query, doc, e.cfg),
		RatingScoreDiverse:  ExtractRatingDiverse(doc, e.cfg),
		ProvenWorkScore:     ExtractProvenWork(query.Persona, doc, e.cfg),
		ResponseRate:        ExtractResponseRate(doc),
		IsVerifiedMature:    ExtractVerifiedMature(doc, e.cfg),
		ProfileCompletion:   ExtractProfileCompletion(doc),
		LastActiveDaysScore: ExtractLastActiveDays(doc, e.cfg),
		AccountAgeBonus:     ExtractAccountAgeBonus(doc, e.cfg),
		NegativeSignals:     penalty,

		RawTextMatchBucket:  bucket,
		RawUniqueReviewers:  int(doc.UniqueReviewersCount),
		RawMaxReviewerShare: clamp01(doc.MaxReviewerShare),
		RawLostDisputes:     rawDisputes,
		RawAccountAgeDays:   int(doc.AccountAgeDays),
	}
}

// Config returns a copy of the extractor's Config for the anti-gaming
// pipeline (it may reuse env thresholds that live alongside formula
// parameters). Returning by value avoids aliasing.
func (e *DefaultExtractor) Config() Config { return e.cfg }
