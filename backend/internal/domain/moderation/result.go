package moderation

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ContentType identifies what kind of user-generated text a moderation
// row refers to. Values must stay in sync with the strings used by the
// admin UI dropdowns and the postgres queries that join moderation_results
// to the source table — they are part of the API contract, not internal
// labels.
type ContentType string

const (
	// Existing content types (Phase 1).
	ContentTypeMessage ContentType = "message"
	ContentTypeReview  ContentType = "review"

	// Phase 2 — sync blocking content types (creation refused if toxic).
	ContentTypeUserDisplayName ContentType = "user_display_name"
	ContentTypeProfileTitle    ContentType = "profile_title"
	ContentTypeProfileAbout    ContentType = "profile_about"
	ContentTypeJobTitle        ContentType = "job_title"
	ContentTypeJobDescription  ContentType = "job_description"

	// Phase 2 — async content types (post-create moderation).
	ContentTypeJobApplicationMessage ContentType = "job_application_message"
	ContentTypeProposalDescription   ContentType = "proposal_description"
)

// ErrContentBlocked is returned by the moderation service when the
// caller asked for blocking mode and the content scored above the
// configured threshold. Handlers translate this sentinel into HTTP 422.
// The error is intentionally generic — the specific field/content_type
// is carried in the wrapping app-layer error.
var ErrContentBlocked = errors.New("moderation: content blocked")

// ErrInvalidContentType is returned by NewResult when content_type is
// empty. We do not enforce membership in the constants list above so
// that future content types do not require domain edits to land — the
// admin UI only shows what it understands and treats unknowns as
// generic items, but the storage layer accepts any non-empty string.
var ErrInvalidContentType = errors.New("moderation: content_type required")

// ErrResultNotFound signals that GetByContent did not find a row for
// the supplied (content_type, content_id). Callers usually treat this
// as "not yet moderated" rather than a hard error.
var ErrResultNotFound = errors.New("moderation: result not found")

// Result is one moderation decision row, mapping 1:1 to a row in
// moderation_results. The constructor enforces invariants that the
// SQL UNIQUE constraint cannot — empty strings, nil UUIDs, etc.
//
// Two timestamps because they answer different questions:
//   - DecidedAt: when did the moderation engine produce this verdict?
//   - ReviewedAt: when did a human admin override or confirm it?
//
// ReviewedBy + ReviewedAt are written together by admin actions
// (Approve, Restore). They are nil for fresh auto-decisions.
type Result struct {
	ID           uuid.UUID
	ContentType  ContentType
	ContentID    uuid.UUID
	AuthorUserID *uuid.UUID // nil for system-generated content (rare)
	Status       Status
	Score        float64
	Labels       []byte // JSON-encoded []TextModerationLabel
	Reason       string
	DecidedAt    time.Time
	ReviewedBy   *uuid.UUID
	ReviewedAt   *time.Time
}

// NewResultInput keeps the constructor under the project's 4-arg
// limit while making each field nameable at the call site.
type NewResultInput struct {
	ContentType  ContentType
	ContentID    uuid.UUID
	AuthorUserID *uuid.UUID
	Status       Status
	Score        float64
	Labels       []byte
	Reason       string
}

// NewResult builds a Result with sane defaults. DecidedAt is set to
// "now" — callers who need to backfill a historical decision must
// override the field on the returned struct before passing it to the
// repository.
func NewResult(in NewResultInput) (*Result, error) {
	if strings.TrimSpace(string(in.ContentType)) == "" {
		return nil, ErrInvalidContentType
	}
	if in.ContentID == uuid.Nil {
		return nil, errors.New("moderation: content_id required")
	}
	labels := in.Labels
	if labels == nil {
		labels = []byte("[]")
	}
	return &Result{
		ID:           uuid.New(),
		ContentType:  in.ContentType,
		ContentID:    in.ContentID,
		AuthorUserID: in.AuthorUserID,
		Status:       in.Status,
		Score:        in.Score,
		Labels:       labels,
		Reason:       in.Reason,
		DecidedAt:    time.Now().UTC(),
	}, nil
}
