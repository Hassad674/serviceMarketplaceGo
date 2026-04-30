package searchanalytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// ltr_capture.go extends the search analytics surface with the
// LTR-ready feature-vector persistence (§9.1 of docs/ranking-v1.md).
//
// Design invariants:
//
//   - Capture is FIRE-AND-FORGET, same as CaptureSearch. A failed
//     INSERT on result_features_json never surfaces to the user.
//   - Idempotency is enforced by result_vector_sha: writing the same
//     ranking twice for the same search_id produces the same hash,
//     and the UPDATE is a no-op when rows match by sha.
//   - The payload is marshalled with sorted keys so the hash is
//     stable across Go maps (Go's JSON encoder already sorts map
//     keys, but the per-result features are laid out as a struct
//     to keep ordering locked).
//
// R3-W will wire this into the query service. This file exposes the
// methods in isolation so Round 2 can merge without touching
// app/search.

// LTRRepository is the write-side port for LTR feature vectors.
// Implemented by the Postgres adapter; tests use an in-memory fake.
//
// Kept narrow so future backends (object storage export, Kafka
// topic) can adopt it without inheriting the entire analytics
// surface.
type LTRRepository interface {
	// AttachResultFeatures updates the search_queries row matching
	// searchID with the serialised feature-vector payload + its SHA.
	// Returns LTRErrNotFound when no row exists (e.g. the capture
	// ran after the search_queries row was evicted).
	AttachResultFeatures(ctx context.Context, searchID, payloadJSON, sha string) error
}

// LTRErrNotFound is returned by AttachResultFeatures when the
// target search_queries row cannot be located.
var LTRErrNotFound = errors.New("search analytics: ltr row not found")

// RankedResult is the per-document payload logged for LTR training.
// Field names match the JSON contract documented in §9.1 so future
// consumers can parse without a schema lookup.
type RankedResult struct {
	DocID        string             `json:"doc_id"`
	RankPosition int                `json:"rank_position"`
	FinalScore   float64            `json:"final_score"`
	Features     map[string]float64 `json:"features"`
}

// CaptureResultFeatures serialises the 20-doc result vector and
// attaches it to the search_queries row. Returns a synchronous error
// when the search_id is missing — that is a programming bug, not a
// fire-and-forget condition. Actual persistence runs on a detached
// goroutine with a bounded deadline.
//
// The persisted shape (verbatim from §9.1):
//
//	[
//	  {"doc_id": "...", "rank_position": 1, "final_score": 87.3, "features": {...}},
//	  {"doc_id": "...", "rank_position": 2, "final_score": 85.0, "features": {...}},
//	  ...
//	]
//
// rank_position is 1-indexed (not 0-indexed) so the logged values
// match the display rank users see on the card grid.
func (s *Service) CaptureResultFeatures(
	ctx context.Context,
	searchID string,
	results []RankedResult,
	ltrRepo LTRRepository,
) error {
	if s == nil {
		return nil
	}
	if searchID == "" {
		return fmt.Errorf("searchanalytics: capture result features: empty search_id")
	}
	if ltrRepo == nil {
		// Nothing to persist against — treat as a soft no-op. The
		// service still runs the capture goroutine against its own
		// repo if the caller swapped interfaces.
		return fmt.Errorf("searchanalytics: capture result features: nil ltr repository")
	}

	payloadJSON, sha, err := EncodeResultPayload(results)
	if err != nil {
		return fmt.Errorf("searchanalytics: encode: %w", err)
	}

	// Detach from the search request context so cancellation does
	// not propagate, but keep the trace identifiers via WithoutCancel
	// so the LTR attach can be correlated to the originating search.
	// gosec G118: parent is request-scoped, not context.Background().
	parent := context.WithoutCancel(ctx)
	go func(payload, hash, sID string) {
		bgCtx, cancel := context.WithTimeout(parent, 3*time.Second)
		defer cancel()
		if err := ltrRepo.AttachResultFeatures(bgCtx, sID, payload, hash); err != nil {
			s.logger.Warn("searchanalytics: ltr attach failed",
				"error", err,
				"search_id", sID,
				"payload_sha", hash[:12])
		}
	}(payloadJSON, sha, searchID)

	return nil
}

// EncodeResultPayload serialises the ranked results into the canonical
// JSON blob + its SHA-256 fingerprint. Exposed as a free function so
// unit tests can verify the on-wire shape without spinning up a
// Service.
//
// Canonicalisation rules:
//   - Results are sorted by rank_position ascending (defensive — we
//     trust the caller, but lock it here so the hash is position-
//     stable even if the input is pre-shuffled).
//   - Feature keys are sorted alphabetically inside each result by
//     the encoder (Go's json already sorts map keys).
//   - Floats are marshalled with the default Go precision.
//
// The resulting SHA is hex-encoded (64 chars) so it fits comfortably
// in a TEXT column.
func EncodeResultPayload(results []RankedResult) (payloadJSON, sha string, err error) {
	if len(results) == 0 {
		return "[]", emptySha(), nil
	}

	sorted := make([]RankedResult, len(results))
	copy(sorted, results)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].RankPosition < sorted[j].RankPosition
	})

	b, err := json.Marshal(sorted)
	if err != nil {
		return "", "", fmt.Errorf("marshal: %w", err)
	}
	sum := sha256.Sum256(b)
	return string(b), hex.EncodeToString(sum[:]), nil
}

// emptySha returns the sha256 of the literal `[]` payload — used so
// the hash of an empty ranking is deterministic.
func emptySha() string {
	sum := sha256.Sum256([]byte("[]"))
	return hex.EncodeToString(sum[:])
}
