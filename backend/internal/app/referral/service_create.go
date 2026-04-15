package referral

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/domain/user"
)

// CreateIntroInput is the payload the handler hands to CreateIntro. The
// referrer is determined by the JWT, not by the request body.
type CreateIntroInput struct {
	ReferrerID           uuid.UUID
	ProviderID           uuid.UUID
	ClientID             uuid.UUID
	RatePct              float64
	DurationMonths       int16
	IntroMessageProvider string
	IntroMessageClient   string
	// Snapshot toggles — if non-nil, the referrer explicitly chose what to
	// reveal. If nil, the snapshot builder uses sensible defaults.
	SnapshotToggles *SnapshotToggles
}

// SnapshotToggles is the apporteur's choice of which auto-filled fields to
// keep on the anonymised snapshot. A nil toggle = include the field.
// Mirrored on the request DTO.
type SnapshotToggles struct {
	IncludeExpertise   bool
	IncludeExperience  bool
	IncludeRating      bool
	IncludePricing     bool
	IncludeRegion      bool
	IncludeLanguages   bool
	IncludeAvailability bool
}

// CreateIntro validates the inputs, builds the anonymised snapshot, persists
// the referral in pending_provider state, appends the initial negotiation row,
// and notifies the provider that they have an intro to review. The client is
// NOT notified at this stage — they enter the flow only after the provider
// has agreed (Modèle A: bilateral apporteur ↔ provider negotiation first).
func (s *Service) CreateIntro(ctx context.Context, input CreateIntroInput) (*referral.Referral, error) {
	if err := s.validateActorRoles(ctx, input.ReferrerID, input.ProviderID, input.ClientID); err != nil {
		return nil, err
	}

	snapshot, err := s.buildSnapshot(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("build snapshot: %w", err)
	}

	ref, err := referral.NewReferral(referral.NewReferralInput{
		ReferrerID:           input.ReferrerID,
		ProviderID:           input.ProviderID,
		ClientID:             input.ClientID,
		RatePct:              input.RatePct,
		DurationMonths:       input.DurationMonths,
		IntroSnapshot:        snapshot,
		IntroMessageProvider: input.IntroMessageProvider,
		IntroMessageClient:   input.IntroMessageClient,
	})
	if err != nil {
		return nil, err
	}

	if err := s.referrals.Create(ctx, ref); err != nil {
		return nil, err
	}

	// Append the initial negotiation row so the timeline starts with the
	// referrer's opening proposal.
	nego, err := referral.NewNegotiation(referral.NewNegotiationInput{
		ReferralID: ref.ID,
		Version:    ref.Version,
		ActorID:    input.ReferrerID,
		ActorRole:  referral.ActorReferrer,
		Action:     referral.NegoActionProposed,
		RatePct:    ref.RatePct,
		Message:    input.IntroMessageProvider,
	})
	if err != nil {
		return nil, fmt.Errorf("build initial negotiation: %w", err)
	}
	if err := s.referrals.AppendNegotiation(ctx, nego); err != nil {
		return nil, fmt.Errorf("append initial negotiation: %w", err)
	}

	s.notifyIntroCreated(ctx, ref)
	return ref, nil
}

// validateActorRoles enforces three rules at the application boundary:
//   - the referrer must be a provider with referrer_enabled=true,
//   - the provider party must be either provider or agency,
//   - the client party must be either enterprise or agency.
func (s *Service) validateActorRoles(ctx context.Context, referrerID, providerID, clientID uuid.UUID) error {
	r, err := s.users.GetByID(ctx, referrerID)
	if err != nil {
		return fmt.Errorf("load referrer: %w", err)
	}
	if r.Role != user.RoleProvider {
		return referral.ErrReferrerRequired
	}
	if !r.ReferrerEnabled {
		return referral.ErrReferrerRequired
	}

	p, err := s.users.GetByID(ctx, providerID)
	if err != nil {
		return fmt.Errorf("load provider party: %w", err)
	}
	if p.Role != user.RoleProvider && p.Role != user.RoleAgency {
		return referral.ErrInvalidProviderRole
	}

	c, err := s.users.GetByID(ctx, clientID)
	if err != nil {
		return fmt.Errorf("load client party: %w", err)
	}
	if c.Role != user.RoleEnterprise && c.Role != user.RoleAgency {
		return referral.ErrInvalidClientRole
	}
	return nil
}

// buildSnapshot resolves the auto-filled snapshot fields from the profile
// loader and applies the apporteur's toggles. Loader errors are non-fatal —
// the snapshot just stays empty for that section, the referral creation can
// still proceed (sometimes a thin profile has nothing to pre-fill).
func (s *Service) buildSnapshot(ctx context.Context, input CreateIntroInput) (referral.IntroSnapshot, error) {
	var snapshot referral.IntroSnapshot

	provSnap, err := s.snapshotProfiles.LoadProvider(ctx, input.ProviderID)
	if err != nil && !errors.Is(err, ErrSnapshotProfileMissing) {
		return snapshot, err
	}
	cliSnap, err := s.snapshotProfiles.LoadClient(ctx, input.ClientID)
	if err != nil && !errors.Is(err, ErrSnapshotProfileMissing) {
		return snapshot, err
	}

	snapshot.Provider = applyProviderToggles(provSnap, input.SnapshotToggles)
	snapshot.Client = cliSnap // client side has no per-field toggles for V1
	return snapshot, nil
}

// applyProviderToggles strips the provider snapshot down to the fields the
// apporteur chose to reveal. A nil toggles pointer means "include everything"
// (the default for the V1 flow when the wizard doesn't expose toggles yet).
func applyProviderToggles(in referral.ProviderSnapshot, toggles *SnapshotToggles) referral.ProviderSnapshot {
	if toggles == nil {
		return in
	}
	out := referral.ProviderSnapshot{}
	if toggles.IncludeExpertise {
		out.ExpertiseDomains = in.ExpertiseDomains
	}
	if toggles.IncludeExperience {
		out.YearsExperience = in.YearsExperience
	}
	if toggles.IncludeRating {
		out.AverageRating = in.AverageRating
		out.ReviewCount = in.ReviewCount
	}
	if toggles.IncludePricing {
		out.PricingMinCents = in.PricingMinCents
		out.PricingMaxCents = in.PricingMaxCents
		out.PricingCurrency = in.PricingCurrency
		out.PricingType = in.PricingType
	}
	if toggles.IncludeRegion {
		out.Region = in.Region
	}
	if toggles.IncludeLanguages {
		out.Languages = in.Languages
	}
	if toggles.IncludeAvailability {
		out.AvailabilityState = in.AvailabilityState
	}
	return out
}

// ErrSnapshotProfileMissing is returned by SnapshotProfileLoader when the
// requested user has no profile yet. It is non-fatal: the snapshot just stays
// empty for that section.
var ErrSnapshotProfileMissing = errors.New("snapshot profile missing")
