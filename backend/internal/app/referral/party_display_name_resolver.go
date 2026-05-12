package referral

import (
	"context"

	"github.com/google/uuid"
)

// PartyDisplayNameResolver resolves the human-readable display name for a
// referral party (provider or client). The apporteur page surfaces it
// in clear when the viewer is the apporteur (owner) so the cards become
// purely informational: "you introduced X to Y", no mystery.
//
// Resolution rules (production adapter in wiring_adapters.go):
//   - If the user owns an organization, the org name wins (an agency
//     or enterprise is the legal/economic entity that operates on the
//     marketplace).
//   - Otherwise, fall back to the user's FullName ("First Last").
//
// Returns the empty string (not an error) when the user is unknown —
// the UI degrades to a placeholder. Defined as a port so the referral
// feature stays decoupled from the org/user concretes.
type PartyDisplayNameResolver interface {
	ResolveDisplayName(ctx context.Context, userID uuid.UUID) (string, error)
}
