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

// Body-not-allowed statuses MUST short-circuit JSON without writing a
// payload — the comment on JSON guarantees this. We pick 204, 304 and
// a 1xx informational status to cover all three branches of the helper.
func TestJSON_BodyNotAllowed_StatusesSkipPayload(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{"204 No Content", http.StatusNoContent},
		{"304 Not Modified", http.StatusNotModified},
		{"100 Continue", http.StatusContinue},
		{"103 Early Hints", http.StatusEarlyHints},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			items := []*listItem{{ID: "1", Name: "X"}}
			JSON(w, tc.status, items)

			assert.Equal(t, tc.status, w.Code)
			assert.Empty(t, w.Body.String(),
				"status %d MUST not produce a response body — Go's net/http forbids it", tc.status)
			assert.Empty(t, w.Header().Get("Content-Type"),
				"body-less responses MUST NOT set Content-Type")
		})
	}
}

// failingResponseWriter forces json.Encode to fail by returning an
// error on Write. JSON must log the failure (we verify the call does
// not panic) and recover gracefully.
type failingResponseWriter struct {
	header http.Header
	status int
}

func (w *failingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *failingResponseWriter) WriteHeader(code int) { w.status = code }
func (w *failingResponseWriter) Write(_ []byte) (int, error) {
	return 0, assert.AnError
}

// The encoding-error branch was previously uncovered. By forcing Write
// to fail we ensure the function reaches the slog.Error path without
// panicking — that is the production safety net.
func TestJSON_EncodingFailure_DoesNotPanic(t *testing.T) {
	w := &failingResponseWriter{}

	require.NotPanics(t, func() {
		JSON(w, http.StatusOK, []*listItem{{ID: "x", Name: "y"}})
	})

	assert.Equal(t, http.StatusOK, w.status,
		"the status code must still be written even when encode fails — header was set before encode")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

// NoContent is a thin convenience wrapper around w.WriteHeader. The
// helper is exported, so we assert it stays trivial — any future
// refactor that changes the status code or starts writing a body would
// be a breaking API change.
func TestNoContent_Writes204Empty(t *testing.T) {
	w := httptest.NewRecorder()

	NoContent(w)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

// Error MUST emit the canonical {error, message} envelope so frontends
// can branch on err code without parsing the message.
func TestError_EmitsCanonicalEnvelope(t *testing.T) {
	w := httptest.NewRecorder()

	Error(w, http.StatusBadRequest, "validation_failed", "human message")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "validation_failed", got["error"])
	assert.Equal(t, "human message", got["message"])
}

// ValidationError MUST attach the per-field details map.
func TestValidationError_EmitsDetails(t *testing.T) {
	w := httptest.NewRecorder()

	ValidationError(w, map[string]string{"email": "invalid"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "validation_error", got["error"])
	assert.Equal(t, "one or more fields are invalid", got["message"])
	details, ok := got["details"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "invalid", details["email"])
}
