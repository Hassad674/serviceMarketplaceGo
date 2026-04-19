package antigaming

import (
	"strings"
	"unicode"

	"marketplace-backend/internal/search/features"
)

// StuffingDetection bundles the detector output. Having it as a named type
// (rather than three return values) means property tests can assert on it
// directly.
type StuffingDetection struct {
	Detected        bool
	MaxRepetition   int
	DistinctRatio   float64
	TotalTokenCount int
}

// detectStuffing implements the rule described in `docs/ranking-v1.md` §7.1.
//
//	tokens         = tokenise(text)
//	token_counts   = histogram(tokens)
//	max_repetition = max(token_counts.values())
//	distinct_ratio = |distinct(tokens)| / |tokens|
//
//	stuffing_detected = max_repetition > MaxTokenRepetition
//	                  OR distinct_ratio < MinDistinctRatio
//
// Tokenisation is whitespace + punctuation based ; everything lowercased.
// Texts shorter than 5 tokens never trigger the rule (short about-blocks
// would otherwise false-positive on any normal writing).
func detectStuffing(text string, cfg Config) StuffingDetection {
	tokens := tokenise(text)
	total := len(tokens)
	if total < 5 {
		return StuffingDetection{TotalTokenCount: total}
	}

	counts := make(map[string]int, total)
	for _, t := range tokens {
		counts[t]++
	}

	maxRep := 0
	for _, c := range counts {
		if c > maxRep {
			maxRep = c
		}
	}
	distinct := float64(len(counts)) / float64(total)

	detected := false
	if maxRep > cfg.MaxTokenRepetition {
		detected = true
	}
	if distinct < cfg.MinDistinctRatio {
		detected = true
	}

	return StuffingDetection{
		Detected:        detected,
		MaxRepetition:   maxRep,
		DistinctRatio:   distinct,
		TotalTokenCount: total,
	}
}

// tokenise splits the input on any non-letter/digit rune + lowercases every
// token. Empty tokens are dropped. Deterministic + allocation-light.
func tokenise(s string) []string {
	if s == "" {
		return nil
	}
	lower := strings.ToLower(s)
	out := make([]string, 0, len(lower)/4)
	start := -1
	for i, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			out = append(out, lower[start:i])
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, lower[start:])
	}
	return out
}

// stuffingRule halves text_match_score when the document text is stuffed.
// The rule mutates the passed-in features in place + returns a Penalty
// describing what happened (for logging).
func stuffingRule(f *features.Features, raw RawSignals, cfg Config) *Penalty {
	det := detectStuffing(raw.Text, cfg)
	if !det.Detected {
		return nil
	}
	// Use the stronger of the two signals as the logged detection value.
	detectionValue := det.DistinctRatio
	threshold := cfg.MinDistinctRatio
	if det.MaxRepetition > cfg.MaxTokenRepetition {
		detectionValue = float64(det.MaxRepetition)
		threshold = float64(cfg.MaxTokenRepetition)
	}
	f.TextMatchScore *= cfg.StuffingPenalty
	return &Penalty{
		Rule:           RuleKeywordStuffing,
		ProfileID:      raw.ProfileID,
		Persona:        raw.Persona,
		DetectionValue: detectionValue,
		Threshold:      threshold,
		PenaltyFactor:  cfg.StuffingPenalty,
	}
}
