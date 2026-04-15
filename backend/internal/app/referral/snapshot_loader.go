package referral

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
)

// SnapshotProfileLoader is a thin read-only port the referral service uses to
// fetch the data needed to assemble an IntroSnapshot at create time. It is
// implemented in cmd/api/main.go as a small adapter that pulls together the
// profile + freelance_pricing + skills repositories — the referral service
// itself never imports those features.
//
// Defined IN the referral package (not in port/service) because it is an
// implementation detail of how this feature builds its snapshots, not a
// general-purpose port other features should consume.
type SnapshotProfileLoader interface {
	// LoadProvider returns the safe-to-reveal provider attributes for a given
	// user. Implementations return zero-value fields when data is missing
	// rather than an error, so the apporteur can still create an intro for a
	// thin profile.
	LoadProvider(ctx context.Context, userID uuid.UUID) (referral.ProviderSnapshot, error)

	// LoadClient returns the safe-to-reveal client attributes. Most fields
	// are populated by the apporteur via the creation wizard rather than
	// auto-filled, but the loader still resolves industry/region/size from
	// the org for a sensible default.
	LoadClient(ctx context.Context, userID uuid.UUID) (referral.ClientSnapshot, error)
}
