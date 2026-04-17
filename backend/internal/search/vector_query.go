package search

import (
	"strconv"
	"strings"
)

// vector_query.go owns the tiny but shape-sensitive helper that
// turns a raw `[]float32` embedding into the Typesense hybrid-query
// parameter:
//
//	embedding:([0.12, 0.34, ...], k:20)
//
// Typesense is strict about this format — any deviation (square
// brackets missing, extra spaces, scientific notation without
// padding) rejects the query with a terse 400. Keeping the helper
// pure + exported means the exact wire format can be pinned in a
// unit test, both in Go and (via a parity test) in TypeScript.

// FormatVectorQuery serialises a vector + top-k into the Typesense
// `vector_query` string. Returns "" when the vector is nil/empty so
// callers can safely compose the result without nil-checking.
func FormatVectorQuery(vec []float32, k int) string {
	if len(vec) == 0 {
		return ""
	}
	if k <= 0 {
		k = 10
	}
	var b strings.Builder
	// Rough size estimate: 10 chars per float + 2 per separator.
	b.Grow(len(vec) * 12)
	b.WriteString("embedding:([")
	for i, f := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteString("], k:")
	b.WriteString(strconv.Itoa(k))
	b.WriteByte(')')
	return b.String()
}
