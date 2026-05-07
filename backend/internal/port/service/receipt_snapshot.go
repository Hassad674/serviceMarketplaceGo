package service

import (
	"context"

	"github.com/google/uuid"
)

// ReceiptSnapshotResolver builds the per-party billing snapshot
// stored on a payment_records row when a payment is created. The
// snapshot freezes the legal identity of the three parties
// (client / provider / referrer) at the moment the money moves so
// the "Reçus" tab can render historically correct receipts even
// after the parties later edit their billing profiles.
//
// Defined as a narrow, transport-agnostic port so the payment app
// stays decoupled from the invoicing + referral features. The
// wiring in cmd/api/main.go injects a thin adapter that reads the
// org membership + billing profile + active referral attribution.
//
// The resolver MUST never block payment creation: any error returned
// here is logged at WARN level and the snapshot field is left empty.
// Receipts produced from such records are rendered with a "données
// indisponibles" marker — better than blocking the payment flow.
type ReceiptSnapshotResolver interface {
	// ResolveForPayment builds a snapshot for a payment between the
	// client (org behind clientUserID) and the provider (org behind
	// providerUserID), with an optional referrer if an active referral
	// attribution exists for the (provider, client) couple.
	//
	// proposalID is passed through so the resolver can look up the
	// referral attribution by proposal — the canonical link from a
	// payment_records row to a referral commission split.
	ResolveForPayment(ctx context.Context, in ReceiptSnapshotInput) (ReceiptSnapshot, error)

	// MarshalSnapshot serialises a snapshot into the canonical JSONB
	// payload stored on payment_records.billing_snapshot. Owned by
	// the resolver so the payment app does not need to know the wire
	// shape. Returns (nil, nil) for an empty snapshot — the column
	// stays NULL on the row.
	MarshalSnapshot(s ReceiptSnapshot) ([]byte, error)
}

// ReceiptSnapshotInput captures the keys the resolver needs to build
// a snapshot. Grouped into a struct so the port stays under the 4-arg
// project limit.
type ReceiptSnapshotInput struct {
	ClientUserID   uuid.UUID
	ProviderUserID uuid.UUID
	ProposalID     uuid.UUID
}

// ReceiptSnapshot is the resolved view returned by the resolver.
// Cents amounts are int64 (centimes), strings can be empty when the
// underlying party has not completed their billing profile.
type ReceiptSnapshot struct {
	Client                        ReceiptSnapshotParty
	Provider                      ReceiptSnapshotParty
	Referrer                      *ReceiptSnapshotParty // nil when no apporteur on this payment
	ReferrerCommissionAmountCents int64                 // 0 when no referrer
}

// ReceiptSnapshotParty is the slim per-party projection captured at
// payment time. Mirrors the fields displayed in the Reçus modal.
type ReceiptSnapshotParty struct {
	OrganizationID uuid.UUID
	Name           string
	SIRET          string
	VAT            string
	AddressLine1   string
	AddressLine2   string
	City           string
	PostalCode     string
	Country        string
}

// IsEmpty reports whether the snapshot has any data attached. Used
// by the payment adapter to decide whether to write NULL or an
// actual JSONB document.
func (s ReceiptSnapshot) IsEmpty() bool {
	return s.Client.OrganizationID == uuid.Nil &&
		s.Provider.OrganizationID == uuid.Nil &&
		s.Referrer == nil
}
