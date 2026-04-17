package search

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	searchpkg "marketplace-backend/internal/search"
)

func TestEncodeDecodeCursor_RoundTrip(t *testing.T) {
	for _, tc := range []Cursor{
		{Page: 1},
		{Page: 2},
		{Page: 999},
	} {
		encoded := EncodeCursor(tc)
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)
		assert.Equal(t, tc.Page, decoded.Page)
		assert.Equal(t, currentCursorVersion, decoded.Version)
	}
}

func TestDecodeCursor_EmptyReturnsZero(t *testing.T) {
	c, err := DecodeCursor("")
	require.NoError(t, err)
	assert.Equal(t, 0, c.Page)
	assert.Equal(t, currentCursorVersion, c.Version)
}

func TestDecodeCursor_InvalidRejected(t *testing.T) {
	cases := []string{
		"not-base64$",
		"bm90anNvbg", // valid base64 but not JSON
	}
	for _, raw := range cases {
		_, err := DecodeCursor(raw)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCursorInvalid)
	}
}

func TestDecodeCursor_RejectsForeignVersion(t *testing.T) {
	encoded := EncodeCursor(Cursor{Page: 1, Version: 42})
	_, err := DecodeCursor(encoded)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCursorInvalid)
}

func TestResolvePage_PreferCursorOverPage(t *testing.T) {
	input := QueryInput{Cursor: EncodeCursor(Cursor{Page: 5}), Page: 2}
	p, err := resolvePage(input)
	require.NoError(t, err)
	assert.Equal(t, 5, p)
}

func TestResolvePage_FallsBackToPage(t *testing.T) {
	input := QueryInput{Page: 3}
	p, err := resolvePage(input)
	require.NoError(t, err)
	assert.Equal(t, 3, p)
}

func TestResolvePage_DefaultsToFirstPage(t *testing.T) {
	input := QueryInput{}
	p, err := resolvePage(input)
	require.NoError(t, err)
	assert.Equal(t, DefaultPage, p)
}

func TestNewSearchID_StableWithinMinute(t *testing.T) {
	input := QueryInput{
		Persona: searchpkg.PersonaFreelance,
		UserID:  "user-1",
	}
	params := searchpkg.SearchParams{Q: "react", FilterBy: "persona:freelance", SortBy: "rating_score:desc"}
	t0 := time.Date(2026, 4, 17, 10, 30, 5, 0, time.UTC)
	t1 := t0.Add(20 * time.Second)
	t2 := t0.Add(90 * time.Second)

	id0 := NewSearchID(input, params, t0)
	id1 := NewSearchID(input, params, t1)
	id2 := NewSearchID(input, params, t2)

	assert.Equal(t, id0, id1, "same minute bucket must share an id")
	assert.NotEqual(t, id0, id2, "minute boundary must rotate the id")
	assert.Len(t, id0, 24)
}

func TestNewSearchID_VariesByQueryShape(t *testing.T) {
	now := time.Now()
	base := QueryInput{Persona: searchpkg.PersonaFreelance}
	ids := map[string]struct{}{}
	for _, p := range []searchpkg.SearchParams{
		{Q: "react"},
		{Q: "react", FilterBy: "persona:freelance"},
		{Q: "react", SortBy: "rating_score:desc"},
		{Q: "vue"},
	} {
		id := NewSearchID(base, p, now)
		ids[id] = struct{}{}
	}
	assert.Len(t, ids, 4)
}
