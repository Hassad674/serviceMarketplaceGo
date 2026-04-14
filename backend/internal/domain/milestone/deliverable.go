package milestone

import (
	"time"

	"github.com/google/uuid"
)

// Deliverable represents a file attached to a specific milestone. Files can
// be added by either party (client or provider) while the milestone is in a
// mutable status — typically pending_funding (brief, specific clauses) or
// funded (provider uploads work-in-progress artefacts).
//
// Proposal-level documents live in a separate type (proposal.ProposalDocument)
// and cover the overall contract. Milestone-level deliverables are scoped to
// the exact step they belong to.
type Deliverable struct {
	ID          uuid.UUID
	MilestoneID uuid.UUID
	Filename    string
	URL         string
	Size        int64
	MimeType    string
	UploadedBy  uuid.UUID
	CreatedAt   time.Time
}

// NewDeliverableInput captures the validated fields required to register a
// deliverable. The URL and storage key are expected to be allocated by the
// adapter (S3/MinIO) before this is called — the domain only stores the
// resulting pointer.
type NewDeliverableInput struct {
	MilestoneID uuid.UUID
	Filename    string
	URL         string
	Size        int64
	MimeType    string
	UploadedBy  uuid.UUID
}

// NewDeliverable builds a validated Deliverable value.
func NewDeliverable(input NewDeliverableInput) (*Deliverable, error) {
	if input.Filename == "" {
		return nil, ErrEmptyTitle // reuse: a deliverable without a filename is as invalid as a milestone without a title
	}
	if input.URL == "" {
		return nil, ErrEmptyDescription
	}
	if input.Size <= 0 {
		return nil, ErrInvalidAmount
	}
	return &Deliverable{
		ID:          uuid.New(),
		MilestoneID: input.MilestoneID,
		Filename:    input.Filename,
		URL:         input.URL,
		Size:        input.Size,
		MimeType:    input.MimeType,
		UploadedBy:  input.UploadedBy,
		CreatedAt:   time.Now(),
	}, nil
}

// IsMutableStatus reports whether deliverables can be added or removed when
// the parent milestone is in the given status. Once the milestone is
// submitted or beyond, deliverables are frozen to preserve evidence.
func IsMutableStatus(status MilestoneStatus) bool {
	return status == StatusPendingFunding || status == StatusFunded
}
