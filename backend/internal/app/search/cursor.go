package search

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"marketplace-backend/internal/search"
)

// cursor.go owns the opaque base64-encoded cursor format the public
// search API exposes.
//
// Phase 3 is paginated via cursor rather than numeric page to match
// the project-wide pagination convention (see backend/CLAUDE.md). We
// internally translate the cursor back to an integer page because
// Typesense 28.0 still uses page-based pagination — when Typesense
// ships cursor pagination natively (v29+, experimental) we will be
// able to swap the internal representation without touching the
// public API.
//
// The cursor JSON carries the following fields:
//
//	{"page": 2, "v": 1}
//
// `v` is a version byte reserved for future changes — anything other
// than 1 raises ErrCursorInvalid so stale cursors from a prior format
// do not decode to the wrong page number.

// Cursor is the decoded cursor payload. Kept small on purpose:
// future phases may add a sort-stable key if Typesense exposes it.
type Cursor struct {
	Page    int `json:"page"`
	Version int `json:"v"`
}

// ErrCursorInvalid is returned when the cursor cannot be decoded or
// carries an unknown version. Handlers map this to a 400 so clients
// do not retry the same bad value.
var ErrCursorInvalid = fmt.Errorf("search cursor: invalid")

// currentCursorVersion is the wire-format version embedded in every
// cursor emitted today. Bump when the shape evolves and keep the
// legacy branch in DecodeCursor until the 30-day rotation is done.
const currentCursorVersion = 1

// EncodeCursor serialises a Cursor into the opaque base64 form
// exposed on the public API.
func EncodeCursor(c Cursor) string {
	if c.Version == 0 {
		c.Version = currentCursorVersion
	}
	raw, err := json.Marshal(c)
	if err != nil {
		// json.Marshal on a struct cannot fail in practice — guard
		// against future field additions by returning empty string
		// rather than a garbled cursor.
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// DecodeCursor parses an opaque cursor back into its typed form.
// Empty input returns the zero cursor (first page) with no error so
// the caller does not need to branch.
func DecodeCursor(raw string) (Cursor, error) {
	if strings.TrimSpace(raw) == "" {
		return Cursor{Page: 0, Version: currentCursorVersion}, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: not base64", ErrCursorInvalid)
	}
	var c Cursor
	if err := json.Unmarshal(decoded, &c); err != nil {
		return Cursor{}, fmt.Errorf("%w: not json", ErrCursorInvalid)
	}
	if c.Version != currentCursorVersion {
		return Cursor{}, fmt.Errorf("%w: version %d unsupported", ErrCursorInvalid, c.Version)
	}
	if c.Page < 0 {
		return Cursor{}, fmt.Errorf("%w: negative page", ErrCursorInvalid)
	}
	return c, nil
}

// resolvePage returns the 1-indexed Typesense page number to query,
// derived first from the input cursor (if set), then from the
// explicit Page field, then from DefaultPage.
func resolvePage(input QueryInput) (int, error) {
	if input.Cursor != "" {
		c, err := DecodeCursor(input.Cursor)
		if err != nil {
			return 0, err
		}
		if c.Page >= 1 {
			return c.Page, nil
		}
	}
	if input.Page >= 1 {
		return input.Page, nil
	}
	return DefaultPage, nil
}

// NewSearchID returns a deterministic identifier for a search
// request. Callers (notably the click-through handler) store the
// ID alongside each search_queries row so they can update the same
// row when a click event arrives seconds later.
//
// The ID is derived from the query shape + the timestamp's second
// bucket, which is sufficient for the 24-hour click window. We do
// NOT use a random UUID because the same search run from the
// frontend pagination should reuse the same ID across pages — this
// lets the analytics layer join page-2 loads to the same conceptual
// query.
func NewSearchID(input QueryInput, params search.SearchParams, now time.Time) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00", input.Persona)
	fmt.Fprintf(h, "%s\x00", params.Q)
	fmt.Fprintf(h, "%s\x00", params.FilterBy)
	fmt.Fprintf(h, "%s\x00", params.SortBy)
	fmt.Fprintf(h, "%s\x00", input.UserID)
	// Bucket the timestamp to the minute so paginated loads of the
	// same query bucket together. If the user re-runs the query 60+
	// seconds later we treat it as a new session.
	fmt.Fprintf(h, "%d", now.Unix()/60)
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:12])
}
