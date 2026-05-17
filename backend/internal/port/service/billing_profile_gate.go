package service

import (
	"context"

	"github.com/google/uuid"
)

// BillingProfileGate is the tiny port the payout sub-service consumes
// to learn whether a provider organization's billing profile is
// complete enough to legally emit the platform_fee invoice that the
// transfer triggers (fix/invoicing-defer-till-transfer).
//
// It exists so the payment app package never imports app/invoicing
// directly — the invoicing feature stays fully removable. Production
// wires the invoicing app service (through a thin adapter) into
// PayoutService via SetBillingProfileGate; when invoicing is disabled
// the gate is left nil.
//
// Contract:
//   - IsBillingProfileComplete returns (true, nil) when the org's
//     billing profile passes the invoicing completeness rule.
//   - Returns (false, nil) when the profile is missing or incomplete.
//   - Returns a non-nil error only on a genuine I/O failure (DB blip,
//     etc.). The caller decides the posture on error — for the
//     auto-transfer decision the conservative choice is "treat as
//     NOT complete" so a milestone never auto-transfers on partial
//     information (the funds stay Available and the provider drains
//     them manually once their profile is in order).
type BillingProfileGate interface {
	// IsBillingProfileComplete reports whether the org identified by
	// organizationID has a billing profile complete enough to support
	// the transfer-time platform_fee invoice.
	IsBillingProfileComplete(ctx context.Context, organizationID uuid.UUID) (bool, error)
}
