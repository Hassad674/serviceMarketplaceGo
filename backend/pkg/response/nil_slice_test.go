package response

// BUG-19 — list endpoints used to return JSON `null` when the result
// set was empty (Go nil slice → JSON null). TS clients across web,
// admin and mobile call `.length` / `.map` on the response and crash
// on `null`. response.JSON now normalises the top-level slice argument
// to an empty slice of the same element type so the wire format is
// `[]` instead of `null`. These tests guard the contract.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// --- NilSliceToEmpty unit tests ---

func TestNilSliceToEmpty_NilArgIsPassedThrough(t *testing.T) {
	got := NilSliceToEmpty(nil)
	assert.Nil(t, got, "literal nil interface stays nil — non-slice rules")
}

func TestNilSliceToEmpty_NilSliceOfStructPointers(t *testing.T) {
	var items []*listItem
	got := NilSliceToEmpty(items)

	out, ok := got.([]*listItem)
	require.True(t, ok, "result must keep the original element type")
	assert.NotNil(t, out)
	assert.Len(t, out, 0)
}

func TestNilSliceToEmpty_NilSliceOfStrings(t *testing.T) {
	var ids []string
	got := NilSliceToEmpty(ids)

	out, ok := got.([]string)
	require.True(t, ok)
	assert.NotNil(t, out)
	assert.Len(t, out, 0)
}

func TestNilSliceToEmpty_NilSliceOfMaps(t *testing.T) {
	var rows []map[string]any
	got := NilSliceToEmpty(rows)

	out, ok := got.([]map[string]any)
	require.True(t, ok)
	assert.NotNil(t, out)
	assert.Len(t, out, 0)
}

func TestNilSliceToEmpty_PopulatedSliceIsUnchanged(t *testing.T) {
	items := []*listItem{{ID: "a", Name: "Alice"}, {ID: "b", Name: "Bob"}}
	got := NilSliceToEmpty(items)

	out, ok := got.([]*listItem)
	require.True(t, ok)
	assert.Len(t, out, 2)
	assert.Equal(t, "Alice", out[0].Name)
}

func TestNilSliceToEmpty_NonSliceIsUnchanged(t *testing.T) {
	type wrapper struct {
		Foo string `json:"foo"`
	}
	in := wrapper{Foo: "bar"}
	got := NilSliceToEmpty(in)

	out, ok := got.(wrapper)
	require.True(t, ok)
	assert.Equal(t, "bar", out.Foo)
}

func TestNilSliceToEmpty_EmptySliceIsUnchanged(t *testing.T) {
	in := []string{}
	got := NilSliceToEmpty(in)

	out, ok := got.([]string)
	require.True(t, ok)
	assert.Len(t, out, 0)
}

// Map at top-level is not normalised (only slices are): a nil map
// renders to JSON `null` and that is the legacy semantic for
// `response.JSON(w, 200, map[...])` callers.
func TestNilSliceToEmpty_NilMapIsUnchanged(t *testing.T) {
	var m map[string]any
	got := NilSliceToEmpty(m)
	// Reflect would reveal the nil-ness; we don't reach for it.
	_, ok := got.(map[string]any)
	assert.True(t, ok)
}

// --- JSON writer integration tests ---

func TestJSON_NilSliceTopLevel_RendersAsEmptyArray(t *testing.T) {
	w := httptest.NewRecorder()
	var items []*listItem // nil

	JSON(w, http.StatusOK, items)

	assert.Equal(t, http.StatusOK, w.Code)
	body := bytes.TrimSpace(w.Body.Bytes())
	assert.Equal(t, "[]", string(body),
		"BUG-19: nil slice MUST encode to `[]`, never `null`")
}

func TestJSON_NilSliceOfMapsTopLevel_RendersAsEmptyArray(t *testing.T) {
	w := httptest.NewRecorder()
	var rows []map[string]any

	JSON(w, http.StatusOK, rows)

	body := bytes.TrimSpace(w.Body.Bytes())
	assert.Equal(t, "[]", string(body))
}

func TestJSON_PopulatedSlice_StillEncodes(t *testing.T) {
	w := httptest.NewRecorder()
	items := []*listItem{{ID: "1", Name: "A"}}

	JSON(w, http.StatusOK, items)

	var got []listItem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Len(t, got, 1)
	assert.Equal(t, "A", got[0].Name)
}

// The legacy `JSON(w, 200, nil)` idiom — used for endpoints with no
// payload — must still produce `null` so non-list callers are not
// disturbed by BUG-19.
func TestJSON_LiteralNil_PreservesLegacyNullSemantic(t *testing.T) {
	w := httptest.NewRecorder()

	JSON(w, http.StatusOK, nil)

	assert.Equal(t, "null\n", w.Body.String())
}

// Single-resource responses (envelope objects) must NOT be normalised
// — the data field can legitimately be a slice OR an object, and the
// caller controls the shape.
func TestJSON_StructWithNilSliceField_SliceFieldStaysNull(t *testing.T) {
	type envelope struct {
		Items []*listItem `json:"items"`
	}
	w := httptest.NewRecorder()

	JSON(w, http.StatusOK, envelope{Items: nil})

	// Nested nil slices ARE NOT normalised — that is the documented
	// scope of NilSliceToEmpty. Handlers that want `items: []` for
	// a struct field call NilSliceToEmpty on the field before
	// composing the envelope.
	body := bytes.TrimSpace(w.Body.Bytes())
	assert.Equal(t, `{"items":null}`, string(body))
}

// The BUG-19 helper is exported so callers building envelopes with
// nested slices can opt in to normalisation.
func TestJSON_NilSliceToEmpty_CalledByCaller_ProducesEmptyArray(t *testing.T) {
	type envelope struct {
		Items any `json:"items"`
	}
	w := httptest.NewRecorder()

	var items []*listItem // nil
	env := envelope{Items: NilSliceToEmpty(items)}

	JSON(w, http.StatusOK, env)

	body := bytes.TrimSpace(w.Body.Bytes())
	assert.Equal(t, `{"items":[]}`, string(body))
}
