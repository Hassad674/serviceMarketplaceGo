package referral

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/repository"
)

// ─── SnapshotProfileLoader adapters ───────────────────────────────────────

// ThinSnapshotLoader is the default SnapshotProfileLoader implementation. It
// looks up the provider's freelance profile and returns a safe-to-reveal
// snapshot. The client-side snapshot is left blank for V1 — the apporteur
// fills in the creation wizard with industry/budget/size manually.
//
// Defined here (not in the handler layer) because it is an implementation
// detail of how the referral feature builds its snapshots from existing
// persona tables. Takes a FreelanceProfileRepository instead of the full
// org+skill machinery so the referral feature stays loosely coupled.
type ThinSnapshotLoader struct {
	freelanceProfiles repository.FreelanceProfileRepository
}

// NewThinSnapshotLoader constructs a ThinSnapshotLoader from the freelance
// profile repository. Safe to call with nil — the loader will return empty
// snapshots rather than error, which lets the referral feature start even
// when the freelance persona tables have not been populated yet.
func NewThinSnapshotLoader(freelanceProfiles repository.FreelanceProfileRepository) *ThinSnapshotLoader {
	return &ThinSnapshotLoader{freelanceProfiles: freelanceProfiles}
}

// LoadProvider returns an empty snapshot for V1. A future iteration will
// resolve the freelance_profile row by user id and fill expertise, pricing
// and availability from it.
func (l *ThinSnapshotLoader) LoadProvider(ctx context.Context, userID uuid.UUID) (referral.ProviderSnapshot, error) {
	return referral.ProviderSnapshot{}, nil
}

// LoadClient returns an empty snapshot — the apporteur supplies client-side
// fields via the creation wizard in V1.
func (l *ThinSnapshotLoader) LoadClient(ctx context.Context, userID uuid.UUID) (referral.ClientSnapshot, error) {
	return referral.ClientSnapshot{}, nil
}

// ─── StripeAccountResolver adapters ───────────────────────────────────────

// OrgStripeAccountResolver resolves a user's Stripe Connect account id via
// the organization repository. Since phase R5, Stripe accounts are owned
// by the organization (the merchant of record), so a user id resolves to
// the Stripe account through their owned org.
//
// Returns empty string (not an error) when no account id is attached —
// that's the signal for the distributor to park the commission as
// pending_kyc, not a failure.
type OrgStripeAccountResolver struct {
	orgs repository.OrganizationRepository
}

// NewOrgStripeAccountResolver wires the resolver. Safe with nil orgs
// (returns empty account id and nil error).
func NewOrgStripeAccountResolver(orgs repository.OrganizationRepository) *OrgStripeAccountResolver {
	return &OrgStripeAccountResolver{orgs: orgs}
}

// ResolveStripeAccountID loads the user's org and returns its Stripe
// account id, or empty string when unavailable.
func (r *OrgStripeAccountResolver) ResolveStripeAccountID(ctx context.Context, userID uuid.UUID) (string, error) {
	if r.orgs == nil {
		return "", nil
	}
	accountID, _, err := r.orgs.GetStripeAccountByUserID(ctx, userID)
	if err != nil {
		// Soft failure — the caller parks the commission and retries later.
		return "", nil
	}
	return accountID, nil
}

// ─── OrgMemberResolver adapters ───────────────────────────────────────────

// OrgDirectoryMemberResolver resolves the list of user ids that share an
// organization with the given user, so referral notifications fan out to
// every member of an agency or enterprise (and not just the contact that
// happened to be named on the intro).
//
// Always includes the anchor user id in the returned slice, even when the
// user has no org row — this is the single-user fallback.
type OrgDirectoryMemberResolver struct {
	orgs    repository.OrganizationRepository
	members repository.OrganizationMemberRepository
}

// NewOrgDirectoryMemberResolver wires the resolver. Safe with nil
// dependencies — the resolver degrades to single-recipient fan-out.
func NewOrgDirectoryMemberResolver(
	orgs repository.OrganizationRepository,
	members repository.OrganizationMemberRepository,
) *OrgDirectoryMemberResolver {
	return &OrgDirectoryMemberResolver{orgs: orgs, members: members}
}

// ResolveMemberUserIDs returns the organization members for the given user,
// or [userID] as a fallback when the user has no org / the lookup fails.
func (r *OrgDirectoryMemberResolver) ResolveMemberUserIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	fallback := []uuid.UUID{userID}
	if r.orgs == nil || r.members == nil {
		return fallback, nil
	}
	org, err := r.orgs.FindByUserID(ctx, userID)
	if err != nil || org == nil {
		return fallback, nil
	}
	byOrg, err := r.members.ListMemberUserIDsByOrgIDs(ctx, []uuid.UUID{org.ID})
	if err != nil {
		return fallback, nil
	}
	ids, ok := byOrg[org.ID]
	if !ok || len(ids) == 0 {
		return fallback, nil
	}
	// Guarantee the anchor user is present (cheap dedup).
	seen := make(map[uuid.UUID]struct{}, len(ids)+1)
	out := make([]uuid.UUID, 0, len(ids)+1)
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if _, ok := seen[userID]; !ok {
		out = append(out, userID)
	}
	return out, nil
}
