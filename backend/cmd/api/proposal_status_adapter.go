package main

import (
	"context"
	"errors"

	"github.com/google/uuid"

	proposalapp "marketplace-backend/internal/app/proposal"
	proposaldomain "marketplace-backend/internal/domain/proposal"
)

// proposalStatusAdapter bridges proposalapp.Service to the payment
// service's ProposalStatusReader contract. The payment package refuses
// to import proposal directly (feature-independence invariant), so the
// thin shim lives in main where cross-feature wiring is allowed.
//
// Stateless beyond the service pointer, so one instance is shared
// across all RequestPayout calls.
type proposalStatusAdapter struct {
	svc *proposalapp.Service
}

// newProposalStatusAdapter returns an adapter ready to be plugged into
// paymentInfoSvc.SetProposalStatusReader.
func newProposalStatusAdapter(svc *proposalapp.Service) *proposalStatusAdapter {
	return &proposalStatusAdapter{svc: svc}
}

// GetProposalStatus implements service.ProposalStatusReader.
//
// Returns an empty string (not an error) when the proposal cannot be
// found — the payment service treats "unknown" as "do not transfer",
// which is safer than failing the entire payout batch for a single
// orphan record. Real infrastructure failures still propagate.
func (a *proposalStatusAdapter) GetProposalStatus(ctx context.Context, proposalID uuid.UUID) (string, error) {
	p, err := a.svc.GetProposalByID(ctx, proposalID)
	if err != nil {
		if errors.Is(err, proposaldomain.ErrProposalNotFound) {
			return "", nil
		}
		return "", err
	}
	if p == nil {
		return "", nil
	}
	return string(p.Status), nil
}
