package repository

import (
	"context"

	"github.com/google/uuid"
)

// ExpertiseRepository persists the ordered list of domain
// specializations declared by a provider organization.
//
// The interface is deliberately small (ListByOrganization,
// ListByOrganizationIDs, Replace) — the frontend model treats the
// whole list as a single atomic value, so there is no need for
// per-row add/remove/reorder methods. Replace does the full
// transactional swap in one round-trip.
//
// Keys are plain strings at this layer (validated beforehand by
// the app service via expertise.IsValidKey). Keeping them as
// strings avoids leaking the domain/expertise package into port/,
// preserving the rule that port interfaces stay as stable and
// narrow as possible.
type ExpertiseRepository interface {
	// ListByOrganization returns the organization's declared
	// expertise keys in display order (ascending position). An
	// empty slice (never nil) is returned when the organization
	// has not declared anything yet — callers marshal that
	// directly to "[]" in the JSON response.
	ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]string, error)

	// ListByOrganizationIDs batch-loads expertise for multiple
	// organizations at once, keyed by organization_id. Used by
	// list endpoints (profile search, job applicant lists) to
	// avoid the N+1 read pattern. Organizations without any
	// declared expertise do not appear in the map — the caller
	// must treat a missing key as an empty list.
	ListByOrganizationIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]string, error)

	// Replace atomically swaps the full set of domain keys for
	// the organization. The incoming slice's order is preserved
	// as the display order (index 0 = position 0). Implementations
	// MUST perform the DELETE+INSERT inside a single database
	// transaction so concurrent readers never observe a partial
	// write. An empty slice is valid and clears the list.
	Replace(ctx context.Context, orgID uuid.UUID, domainKeys []string) error
}
