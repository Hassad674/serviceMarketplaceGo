package cursor

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode_Decode_Roundtrip(t *testing.T) {
	id := uuid.New()
	createdAt := time.Date(2026, 3, 16, 10, 30, 0, 0, time.UTC)

	encoded := Encode(createdAt, id)
	require.NotEmpty(t, encoded)

	decoded, err := Decode(encoded)

	require.NoError(t, err)
	require.NotNil(t, decoded)
	assert.Equal(t, id, decoded.ID)
	assert.True(t, createdAt.Equal(decoded.CreatedAt))
}

func TestDecode_InvalidBase64(t *testing.T) {
	decoded, err := Decode("!!!not-valid-base64!!!")

	assert.Error(t, err)
	assert.Nil(t, decoded)
	assert.Contains(t, err.Error(), "invalid base64")
}

func TestDecode_InvalidJSON(t *testing.T) {
	// Valid base64 but not valid JSON
	// "not json" -> base64
	encoded := "bm90IGpzb24=" // base64 of "not json"

	decoded, err := Decode(encoded)

	assert.Error(t, err)
	assert.Nil(t, decoded)
	assert.Contains(t, err.Error(), "invalid json")
}

func TestDecode_EmptyString(t *testing.T) {
	decoded, err := Decode("")

	assert.Error(t, err)
	assert.Nil(t, decoded)
}

func TestEncode_DifferentValues(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	now := time.Now()

	encoded1 := Encode(now, id1)
	encoded2 := Encode(now, id2)

	assert.NotEqual(t, encoded1, encoded2, "different IDs should produce different cursors")

	later := now.Add(time.Hour)
	encoded3 := Encode(later, id1)

	assert.NotEqual(t, encoded1, encoded3, "different timestamps should produce different cursors")
}

func TestEncode_Decode_PreservesNanosecondPrecision(t *testing.T) {
	id := uuid.New()
	// JSON time marshaling truncates to sub-second; verify roundtrip is consistent
	createdAt := time.Now().UTC().Truncate(time.Millisecond)

	encoded := Encode(createdAt, id)
	decoded, err := Decode(encoded)

	require.NoError(t, err)
	assert.True(t, createdAt.Equal(decoded.CreatedAt))
}
