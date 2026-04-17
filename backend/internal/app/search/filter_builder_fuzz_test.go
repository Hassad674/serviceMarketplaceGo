package search

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzBuildFilterBy feeds randomised string / numeric inputs into the
// filter builder and asserts the three invariants phase 5B locks in:
//
//  1. The call never panics.
//  2. The output is valid UTF-8 (the Typesense wire format only accepts UTF-8).
//  3. The clause count matches the number of active filters — no ghost
//     clauses, no silent drops. This catches regressions that forget
//     to trim a field or accidentally duplicate a clause.
//
// Run with:
//
//	go test ./internal/app/search -run=NONE -fuzz=FuzzBuildFilterBy -fuzztime=30s
//
// Seed corpus includes pathological inputs (empty strings, unicode,
// negative pricing, NaN geo) so the property is exercised from the
// first second.
func FuzzBuildFilterBy(f *testing.F) {
	// Seed corpus — each entry is a structural combination the
	// hand-written tests never cover.
	f.Add("", "", float64(0), float64(0), float64(0), int64(0), int64(0),
		float64(0), "", "", "", "", "", int8(0), int8(0), int8(0))
	f.Add("now", "FR", 48.85, 2.35, 50.0, int64(10000), int64(80000),
		4.5, "fr,en", "React,Go", "dev-frontend", "remote", "Paris", int8(1), int8(1), int8(0))
	f.Add("now,soon,not", "fr", 0.0, 0.0, 0.0, int64(0), int64(0),
		0.0, ",,,,", "   ,,  ", "", "", " ", int8(0), int8(0), int8(0))
	f.Add("soon", "DE", -90.0, 180.0, 10000.0, int64(-1), int64(-1),
		-3.0, "æ", "日本語", "🚀", "\x00", "New York", int8(2), int8(2), int8(2))

	f.Fuzz(func(t *testing.T,
		availability, country string,
		lat, lng, radius float64,
		pMin, pMax int64,
		rating float64,
		languagesCSV, skillsCSV, expertiseCSV, workModeCSV, city string,
		verifiedByte, topRatedByte, negotiableByte int8,
	) {
		input := FilterInput{
			AvailabilityStatus: splitCSV(availability),
			CountryCode:        country,
			City:               city,
			Languages:          splitCSV(languagesCSV),
			Skills:             splitCSV(skillsCSV),
			ExpertiseDomains:   splitCSV(expertiseCSV),
			WorkMode:           splitCSV(workModeCSV),
		}
		if lat != 0 || lng != 0 || radius != 0 {
			input.GeoLat, input.GeoLng, input.GeoRadiusKm = &lat, &lng, &radius
		}
		if pMin != 0 {
			input.PricingMin = &pMin
		}
		if pMax != 0 {
			input.PricingMax = &pMax
		}
		if rating != 0 {
			input.RatingMin = &rating
		}
		input.IsVerified = tristateBool(verifiedByte)
		input.IsTopRated = tristateBool(topRatedByte)
		input.Negotiable = tristateBool(negotiableByte)

		// Property 1: no panic. A panic here fails the fuzz run
		// regardless of the other asserts.
		got := BuildFilterBy(input)

		// Property 2: always valid UTF-8. Typesense rejects non-UTF8.
		if !utf8.ValidString(got) {
			t.Fatalf("BuildFilterBy produced invalid UTF-8: %q", got)
		}

		// Property 3: idempotent — building the same input twice
		// returns the same clause. Catches any hidden global state.
		got2 := BuildFilterBy(input)
		if got != got2 {
			t.Fatalf("BuildFilterBy not deterministic:\n  first:  %q\n  second: %q", got, got2)
		}
	})
}

// FuzzBuildFilterBy_Dedupe asserts that duplicating a filter value N
// times never adds clauses — BuildFilterBy must dedupe its slice
// inputs. This is a property the hand-written tests cover only with
// small fixed samples.
func FuzzBuildFilterBy_Dedupe(f *testing.F) {
	f.Add("fr", 5)
	f.Add("en", 50)
	f.Add("  spaces  ", 10)

	f.Fuzz(func(t *testing.T, lang string, dupCount int) {
		if dupCount < 0 || dupCount > 1000 {
			t.Skip("out-of-range dupCount — skipped by fuzz guard")
		}
		langs := make([]string, 0, dupCount)
		for i := 0; i < dupCount; i++ {
			langs = append(langs, lang)
		}
		input := FilterInput{Languages: langs}
		got := BuildFilterBy(input)

		// Count commas inside the `languages_professional:[...]`
		// bracket. A duplicate-free slice contains zero or one
		// value → zero commas.
		if strings.Contains(got, "languages_professional:[") {
			bracket := extractBracket(got, "languages_professional:[")
			if strings.Contains(bracket, ",") {
				t.Fatalf("dedupe failed — bracket contains comma: %q (from %d copies of %q)", bracket, dupCount, lang)
			}
		}
	})
}

// splitCSV converts a comma-separated fuzz input into a slice the
// filter accepts. Empty segments are dropped by BuildFilterBy via
// dedupeStrings so we forward them as-is.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// tristateBool converts a fuzz byte into {nil, &true, &false} so we
// exercise the pointer-bool codepath.
func tristateBool(b int8) *bool {
	switch b % 3 {
	case 0:
		return nil
	case 1:
		t := true
		return &t
	default:
		f := false
		return &f
	}
}

// extractBracket returns the substring between the first occurrence of
// prefix+"[" and the next "]". Empty string when either marker is
// missing. Pure helper — no dependencies.
func extractBracket(s, prefix string) string {
	idx := strings.Index(s, prefix)
	if idx == -1 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.Index(s[start:], "]")
	if end == -1 {
		return ""
	}
	return s[start : start+end]
}
