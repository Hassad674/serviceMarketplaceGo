package invoicing

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/service"
)

// BillingProfileGateAdapter satisfies service.BillingProfileGate by
// delegating to the invoicing Service.IsBillingProfileComplete read,
// dropping the missing-fields slice the auto-transfer decision does
// not need (it only cares about the boolean — the manual "Retirer"
// flow is the one that surfaces the missing fields in a modal).
//
// Wired in cmd/api (wire_invoicing.go) AFTER the Service is built; the
// payment PayoutService receives this adapter via SetBillingProfileGate
// so the payment package never imports the invoicing app package and
// the invoicing feature stays fully removable.
type BillingProfileGateAdapter struct {
	svc *Service
}

// NewBillingProfileGateAdapter constructs the adapter.
func NewBillingProfileGateAdapter(svc *Service) *BillingProfileGateAdapter {
	return &BillingProfileGateAdapter{svc: svc}
}

// Compile-time interface satisfaction.
var _ service.BillingProfileGate = (*BillingProfileGateAdapter)(nil)

// IsBillingProfileComplete implements service.BillingProfileGate.
// Returns (false, nil) when the adapter or service is not wired so a
// missing dependency degrades to "not complete" — the conservative
// choice for the money-out auto-transfer decision (the caller defers
// the transfer and the funds stay Available rather than being moved on
// partial information).
func (a *BillingProfileGateAdapter) IsBillingProfileComplete(ctx context.Context, organizationID uuid.UUID) (bool, error) {
	if a == nil || a.svc == nil {
		return false, nil
	}
	complete, _, err := a.svc.IsBillingProfileComplete(ctx, organizationID)
	if err != nil {
		return false, err
	}
	return complete, nil
}
