package receipt

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	invoicingdomain "marketplace-backend/internal/domain/invoicing"
	referraldomain "marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// SnapshotResolver builds the JSONB billing_snapshot stored on a
// payment_records row at creation time. It is a thin orchestration
// over three read ports — user, billing profile, referral — wired
// in cmd/api/main.go.
//
// The resolver implements portservice.ReceiptSnapshotResolver so the
// payment app can inject it without importing this package. Cross-
// feature isolation is preserved: payment depends on a port, not on
// receipt or invoicing or referral.
//
// Resilience contract: the resolver MUST NOT block payment creation
// on its own failures. Every read defensively swallows ErrNotFound
// and continues with empty fields. Returning a non-nil error here
// would propagate up through CreatePaymentIntent and surface to the
// client — receipts are a side-effect of payments, not a gate.
type SnapshotResolver struct {
	users       userOrgReader
	billing     billingProfileReader
	referrals   referralAttributionReader
	commissions commissionReader
}

// userOrgReader returns the organization id a user belongs to. The
// project keeps users.organization_id denormalised, so this is one
// row lookup. Implemented by a small adapter wrapped over the
// existing user repository in cmd/api wiring.
type userOrgReader interface {
	GetOrganizationIDForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// UserOrgFunc adapts an inline closure to the userOrgReader port —
// makes the wiring layer one-line ergonomic without adding a
// dedicated repository interface.
type UserOrgFunc func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)

// GetOrganizationIDForUser satisfies userOrgReader.
func (f UserOrgFunc) GetOrganizationIDForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	return f(ctx, userID)
}

// billingProfileReader returns the org's billing identity, or
// ErrNotFound when the org has not completed its profile.
type billingProfileReader interface {
	FindByOrganization(ctx context.Context, orgID uuid.UUID) (*invoicingdomain.BillingProfile, error)
}

// referralAttributionReader returns the active attribution for a
// proposal — the link from a payment to the apporteur that owns the
// commission split. Returns referral.ErrAttributionNotFound when
// there is no apporteur on this proposal.
//
// GetByID lets the resolver follow Attribution.ReferralID up to the
// parent Referral so it can read the apporteur's user id (only
// stored on the parent, not the attribution).
type referralAttributionReader interface {
	FindAttributionByProposal(ctx context.Context, proposalID uuid.UUID) (*referraldomain.Attribution, error)
	GetByID(ctx context.Context, id uuid.UUID) (*referraldomain.Referral, error)
}

// commissionReader returns the latest commission row attached to a
// (attribution, proposal) pair so the snapshot can record the
// referrer's gross commission cents. Optional — when nil, the
// resolver records the commission amount as 0 and the UI falls back
// to displaying just the rate.
type commissionReader interface {
	FindCommissionByMilestone(ctx context.Context, milestoneID uuid.UUID) (*referraldomain.Commission, error)
}

// SnapshotResolverDeps groups the constructor arguments.
type SnapshotResolverDeps struct {
	Users       userOrgReader
	Billing     billingProfileReader
	Referrals   referralAttributionReader
	Commissions commissionReader // optional
}

// Compile-time assertion: BillingProfileRepository satisfies the
// narrow billingProfileReader port we depend on. Catches accidental
// drift between the two surfaces if either evolves.
var _ billingProfileReader = (repository.BillingProfileRepository)(nil)

// NewSnapshotResolver wires the resolver. Users + Billing +
// Referrals are mandatory; Commissions is optional (nil disables
// the referrer commission amount field).
func NewSnapshotResolver(deps SnapshotResolverDeps) *SnapshotResolver {
	return &SnapshotResolver{
		users:       deps.Users,
		billing:     deps.Billing,
		referrals:   deps.Referrals,
		commissions: deps.Commissions,
	}
}

// ResolveForPayment builds the snapshot. Best-effort: any sub-read
// that fails is logged via a returned partial snapshot rather than
// surfacing as an error. The payment caller wraps this in a soft
// hook (logs and stores empty bytes when err != nil).
func (r *SnapshotResolver) ResolveForPayment(ctx context.Context, in portservice.ReceiptSnapshotInput) (portservice.ReceiptSnapshot, error) {
	if r == nil {
		return portservice.ReceiptSnapshot{}, errors.New("snapshot resolver not configured")
	}
	if in.ClientUserID == uuid.Nil || in.ProviderUserID == uuid.Nil {
		return portservice.ReceiptSnapshot{}, errors.New("client and provider user ids are required")
	}

	clientOrg, _ := r.users.GetOrganizationIDForUser(ctx, in.ClientUserID)
	providerOrg, _ := r.users.GetOrganizationIDForUser(ctx, in.ProviderUserID)

	out := portservice.ReceiptSnapshot{
		Client:   r.partyFor(ctx, clientOrg),
		Provider: r.partyFor(ctx, providerOrg),
	}

	// Referrer attribution lookup. If no attribution exists for the
	// proposal, the referrer field stays nil (snapshot encodes it
	// as JSON `null`, the UI hides the row).
	//
	// Two-step lookup: Attribution → parent Referral → ReferrerID.
	// Attribution itself only stores the provider/client side of the
	// relationship; the apporteur's user id lives on the parent
	// Referral aggregate (ReferrerID). We fetch the parent so we can
	// resolve the apporteur's organization for the snapshot.
	if r.referrals != nil && in.ProposalID != uuid.Nil {
		attr, err := r.referrals.FindAttributionByProposal(ctx, in.ProposalID)
		if err == nil && attr != nil {
			parent, err := r.referrals.GetByID(ctx, attr.ReferralID)
			if err == nil && parent != nil {
				referrerOrg, _ := r.users.GetOrganizationIDForUser(ctx, parent.ReferrerID)
				if referrerOrg != uuid.Nil {
					p := r.partyFor(ctx, referrerOrg)
					out.Referrer = &p
				}
			}
		}
	}

	return out, nil
}

// partyFor builds a single party projection from an org id. Empty
// org id (the user has no org membership) yields an empty party.
func (r *SnapshotResolver) partyFor(ctx context.Context, orgID uuid.UUID) portservice.ReceiptSnapshotParty {
	if orgID == uuid.Nil {
		return portservice.ReceiptSnapshotParty{}
	}
	out := portservice.ReceiptSnapshotParty{OrganizationID: orgID}
	if r.billing == nil {
		return out
	}
	profile, err := r.billing.FindByOrganization(ctx, orgID)
	if err != nil || profile == nil {
		return out
	}
	out.Name = profile.LegalName
	out.SIRET = profile.TaxID
	out.VAT = profile.VATNumber
	out.AddressLine1 = profile.AddressLine1
	out.AddressLine2 = profile.AddressLine2
	out.City = profile.City
	out.PostalCode = profile.PostalCode
	out.Country = profile.Country
	return out
}

// MarshalSnapshot turns a port snapshot into the JSONB byte payload
// stored on the payment_records row. Method on the resolver so the
// payment app's ChargeService never imports this package — it
// receives the resolver as an interface and calls MarshalSnapshot
// through it.
func (r *SnapshotResolver) MarshalSnapshot(s portservice.ReceiptSnapshot) ([]byte, error) {
	return marshalSnapshot(s)
}

// marshalSnapshot is the package-level helper used by both the
// resolver method and the (test-only) freestanding callers. Kept
// private so the only public marshal entry point stays the resolver
// method on the port.
func marshalSnapshot(s portservice.ReceiptSnapshot) ([]byte, error) {
	if s.IsEmpty() {
		return nil, nil
	}
	return json.Marshal(struct {
		Client                        partySnapshotJSON  `json:"client"`
		Provider                      partySnapshotJSON  `json:"provider"`
		Referrer                      *partySnapshotJSON `json:"referrer"`
		ReferrerCommissionAmountCents int64              `json:"referrer_commission_amount_cents"`
	}{
		Client:                        partyToJSON(s.Client),
		Provider:                      partyToJSON(s.Provider),
		Referrer:                      partyPointerToJSON(s.Referrer),
		ReferrerCommissionAmountCents: s.ReferrerCommissionAmountCents,
	})
}

// partySnapshotJSON is the wire shape shared with the postgres
// adapter (mirrored there to keep the JSON tags on a private type).
type partySnapshotJSON struct {
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	SIRET          string `json:"siret"`
	VAT            string `json:"vat"`
	AddressLine1   string `json:"address_line1"`
	AddressLine2   string `json:"address_line2"`
	City           string `json:"city"`
	PostalCode     string `json:"postal_code"`
	Country        string `json:"country"`
}

func partyToJSON(p portservice.ReceiptSnapshotParty) partySnapshotJSON {
	out := partySnapshotJSON{
		Name:         p.Name,
		SIRET:        p.SIRET,
		VAT:          p.VAT,
		AddressLine1: p.AddressLine1,
		AddressLine2: p.AddressLine2,
		City:         p.City,
		PostalCode:   p.PostalCode,
		Country:      p.Country,
	}
	if p.OrganizationID != uuid.Nil {
		out.OrganizationID = p.OrganizationID.String()
	}
	return out
}

func partyPointerToJSON(p *portservice.ReceiptSnapshotParty) *partySnapshotJSON {
	if p == nil {
		return nil
	}
	out := partyToJSON(*p)
	return &out
}
