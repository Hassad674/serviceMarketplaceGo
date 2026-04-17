package handler

import (
	"net/http"
	"net/url"
	"testing"

	appsearch "marketplace-backend/internal/app/search"
)

// FuzzParseFilterInput feeds the handler-level query-string parser
// arbitrary inputs and asserts two invariants:
//
//  1. The parser never panics, regardless of input shape.
//  2. Passing the parser's output through BuildFilterBy never panics
//     either — i.e. the two layers agree on what counts as a valid
//     FilterInput.
//
// This is the last defensive layer between an HTTP request and
// Typesense. If the fuzzer finds a crash, we have a 500 risk.
func FuzzParseFilterInput(f *testing.F) {
	f.Add("")
	f.Add("availability=now&pricing_min=1000&rating_min=4")
	f.Add("geo_lat=invalid&geo_lng=2.35")
	f.Add("skills=,,,")
	f.Add("availability=\x00\x01\xfe")
	f.Add("country=FR&verified=true&top_rated=1")
	f.Add("pricing_min=not-a-number&pricing_max=9999999999999999999999")
	f.Add("geo_lat=nan&geo_lng=inf&geo_radius_km=-1")

	f.Fuzz(func(t *testing.T, rawQuery string) {
		// Parsing the query string may itself error — we stop the
		// test early in that case because the handler short-circuits
		// before reaching parseFilterInput. Equivalent to Chi's
		// behaviour: an invalid URL never reaches our handler.
		parsedValues, err := url.ParseQuery(rawQuery)
		if err != nil {
			t.Skip("ignored — url.ParseQuery rejects input")
		}
		// Build the request by hand with a known-clean URL so that
		// httptest's own input validation cannot panic on control
		// characters — once the request is in front of our handler
		// the query is a valid url.Values. This matches the real
		// request flow: Go's http server already validated the URI.
		req := &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/api/v1/search", RawQuery: parsedValues.Encode()},
		}
		input := parseFilterInput(req)

		// Round-trip through the filter builder — this is the path
		// the production code always takes.
		got := appsearch.BuildFilterBy(input)

		// Idempotence: a second call must produce identical output.
		got2 := appsearch.BuildFilterBy(input)
		if got != got2 {
			t.Fatalf("BuildFilterBy not idempotent for query %q:\n  first:  %q\n  second: %q", rawQuery, got, got2)
		}
	})
}
