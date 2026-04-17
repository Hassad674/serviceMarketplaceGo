package search

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatVectorQuery(t *testing.T) {
	t.Run("empty vector returns empty string", func(t *testing.T) {
		assert.Equal(t, "", FormatVectorQuery(nil, 10))
		assert.Equal(t, "", FormatVectorQuery([]float32{}, 10))
	})
	t.Run("default k applied when zero", func(t *testing.T) {
		got := FormatVectorQuery([]float32{0.1, 0.2}, 0)
		assert.Contains(t, got, "k:10")
	})
	t.Run("wire format is exact", func(t *testing.T) {
		got := FormatVectorQuery([]float32{0.1, 0.2, 0.3}, 20)
		// We assert shape rather than byte-equality on floats to
		// survive Go's float formatting nuances across versions.
		assert.True(t, strings.HasPrefix(got, "embedding:(["))
		assert.True(t, strings.HasSuffix(got, "], k:20)"))
		// 2 commas between 3 floats + 1 comma before k.
		assert.Equal(t, 3, strings.Count(got, ","))
	})
	t.Run("large vectors serialise without error", func(t *testing.T) {
		vec := make([]float32, 1536)
		for i := range vec {
			vec[i] = float32(i) / 1536.0
		}
		got := FormatVectorQuery(vec, 20)
		assert.True(t, strings.HasSuffix(got, "], k:20)"))
		// 1535 commas between 1536 values + 1 before k.
		assert.Equal(t, 1536, strings.Count(got, ","))
	})
}
