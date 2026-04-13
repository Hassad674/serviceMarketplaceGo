package repository

import (
	"context"

	"github.com/google/uuid"

	domainskill "marketplace-backend/internal/domain/skill"
)

// ProfileSkillRepository is the persistence contract for the M2M
// relation between organizations and entries of the skills catalog.
// Implementations live in adapter/ and must not leak SQL or driver-
// specific types across this boundary.
//
// The interface is deliberately small: the frontend model treats the
// list of skills attached to an organization as a single atomic value
// (the full set is replaced on save), so there is no need for per-row
// add / remove / reorder methods. ReplaceForOrg performs the full
// transactional swap in one round-trip.
type ProfileSkillRepository interface {
	// ListByOrgID returns all skills declared by the organization,
	// ordered by position ASC. An empty slice (never nil) is returned
	// when the organization has not declared anything yet — callers
	// marshal that directly to "[]" in the JSON response.
	ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error)

	// ListByOrgIDs is the batch variant of ListByOrgID, used by list
	// endpoints (discovery / search) that need to decorate many
	// profiles with their declared skills in a single database
	// roundtrip. The returned map is keyed by organization ID and
	// contains a (non-nil but possibly empty) slice for every org ID
	// passed in — implementations MUST NOT omit keys for orgs that
	// have zero skills, so callers can range over the input slice
	// without nil-checks. Per-org slices are ordered by position ASC.
	//
	// N+1 prevention: the adapter must execute a single query.
	ListByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*domainskill.ProfileSkill, error)

	// ReplaceForOrg atomically replaces the organization's skills
	// with the provided list. The caller is responsible for assigning
	// contiguous, 0-indexed Position values before invoking this
	// method. Implementations MUST perform the DELETE + INSERT inside
	// a single database transaction so concurrent readers never
	// observe a partial write. An empty slice is valid and clears
	// the list entirely.
	ReplaceForOrg(ctx context.Context, orgID uuid.UUID, skills []*domainskill.ProfileSkill) error

	// CountByOrg returns the number of skills currently attached to
	// the organization. Used by the service layer to enforce per-org-
	// type limits (MaxSkillsForOrgType) on incremental operations and
	// by list endpoints to display per-profile counters without
	// fetching the full set.
	CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)

	// DeleteAllByOrg removes every skill attached to the organization.
	// Used on explicit user-initiated clear ("reset my skills"). For
	// cascade-on-org-delete the DB-level ON DELETE CASCADE on the
	// FK does the job — this method is for application-level wipes.
	DeleteAllByOrg(ctx context.Context, orgID uuid.UUID) error
}
