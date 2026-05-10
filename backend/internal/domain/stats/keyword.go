package stats

// KeywordRow describes one entry in the "top keywords visitors typed
// before viewing the profile" panel.
//
//   - Keyword       — the lowercased, trimmed search query string.
//   - Count         — number of times that query led to a view of the
//                     profile in the requested window.
//   - AvgPosition   — average search rank at which the profile
//                     appeared for that keyword. 0 when no row had
//                     a non-null search_position.
type KeywordRow struct {
	Keyword     string
	Count       int
	AvgPosition float64
}

// ClampLimit returns the user-supplied keyword limit clamped to the
// allowed range. Default is 10, max is 100. The handler uses this so
// a missing/invalid query param does not error out — we render the
// default instead. Validate explicitly when you need a 400 response.
func ClampLimit(in int) int {
	if in <= 0 {
		return 10
	}
	if in > 100 {
		return 100
	}
	return in
}
