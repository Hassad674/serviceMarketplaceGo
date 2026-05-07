// Package receipt is the read-side aggregate that backs the "Reçus"
// tab in the billing UI.
//
// A Receipt is a per-payment snapshot:
//   - the client and provider organizations involved,
//   - the optional referrer (apporteur d'affaire) commission amount,
//   - the billing identity (legal name, tax id, VAT, address) of every
//     party at the moment the payment cleared,
//   - the gross amount and currency.
//
// Receipts are NOT legal invoices. The Contra-like invoicing model
// (see project_invoicing_model.md) is explicit on this: the platform
// never produces a fiscal invoice on behalf of users. The receipt is a
// transaction record the user transcribes into their own accounting
// tool to produce an actual invoice.
//
// The domain has no persistence responsibility — see
// internal/port/repository/receipt.go for the read port and the
// postgres adapter for the implementation. This package owns:
//   - the Receipt aggregate
//   - the PartyBilling value object (one snapshot per party)
//   - the sentinel errors the app layer maps to HTTP statuses
//
// Package import policy: zero external imports beyond stdlib + the
// google/uuid module already used across all domain packages. No
// imports from port/, app/, adapter/, or any other domain package.
package receipt

import (
	"time"

	"github.com/google/uuid"
)

// PartyBilling is the snapshot of one party's billing identity on a
// receipt. Every field except OrganizationID may be empty when:
//   - the party has not completed their billing profile yet,
//   - the receipt predates the snapshot feature (historical data).
//
// The handler layer renders empty fields as "Profil incomplet" so the
// user can still produce their own invoice from what is available.
type PartyBilling struct {
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

// Receipt represents one transaction between a client org and a
// provider org, with an optional referrer commission. The ID matches
// the underlying payment_record id so callers can navigate from one
// to the other without a second lookup.
//
// MilestoneID is non-nil when the payment was scoped to a specific
// milestone (the modern path — every payment_record after Phase 4 is
// milestone-scoped). It is left nil only on legacy or test rows.
//
// SnapshotAvailable encodes whether the row has a stored
// billing_snapshot JSON. False means the receipt predates the feature
// — Client/Provider/Referrer are all nil, the UI must surface "Reçu
// incomplet — données antérieures à l'introduction de la
// fonctionnalité".
type Receipt struct {
	ID                            uuid.UUID
	PaymentRecordID               uuid.UUID
	ProposalID                    uuid.UUID
	MilestoneID                   uuid.UUID
	AmountCents                   int64
	Currency                      string
	CreatedAt                     time.Time
	Client                        *PartyBilling
	Provider                      *PartyBilling
	Referrer                      *PartyBilling
	ReferrerCommissionAmountCents int64
	SnapshotAvailable             bool
}

// IsParty reports whether the given org appears as client, provider
// or referrer on the receipt. Used by the handler layer for the final
// ownership check (defense in depth — the repository already filters
// at the SQL layer, but a second check at the handler closes any
// future hole if a query path skips the WHERE).
func (r *Receipt) IsParty(orgID uuid.UUID) bool {
	if r == nil || orgID == uuid.Nil {
		return false
	}
	if r.Client != nil && r.Client.OrganizationID == orgID {
		return true
	}
	if r.Provider != nil && r.Provider.OrganizationID == orgID {
		return true
	}
	if r.Referrer != nil && r.Referrer.OrganizationID == orgID {
		return true
	}
	return false
}
