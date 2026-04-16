package referral

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
)

// Cancel allows the referrer to abort an intro that has not yet activated.
// Forbidden once status is active — use Terminate for that case.
func (s *Service) Cancel(ctx context.Context, referralID, actorID uuid.UUID) (*referral.Referral, error) {
	r, err := s.loadAndAuthorise(ctx, referralID, actorID, referral.ActorReferrer)
	if err != nil {
		return nil, err
	}
	prev := r.Status
	if err := r.Cancel(actorID); err != nil {
		return nil, err
	}
	if err := s.referrals.Update(ctx, r); err != nil {
		return nil, fmt.Errorf("update referral on cancel: %w", err)
	}
	s.notifyStatusTransition(ctx, r, prev)
	s.postTransitionMessages(ctx, r, prev)
	return r, nil
}

// Terminate ends an active referral. Existing attributions and pending
// commissions stay valid — they will pay out as their milestones are released.
// The exclusivity window simply stops generating NEW attributions.
func (s *Service) Terminate(ctx context.Context, referralID, actorID uuid.UUID) (*referral.Referral, error) {
	r, err := s.loadAndAuthorise(ctx, referralID, actorID, referral.ActorReferrer)
	if err != nil {
		return nil, err
	}
	prev := r.Status
	if err := r.Terminate(actorID); err != nil {
		return nil, err
	}
	if err := s.referrals.Update(ctx, r); err != nil {
		return nil, fmt.Errorf("update referral on terminate: %w", err)
	}
	s.notifyStatusTransition(ctx, r, prev)
	s.postTransitionMessages(ctx, r, prev)
	return r, nil
}

// GetByID is the read-side helper used by the handler. It performs the same
// authorisation check as the respond methods — only the three parties of a
// referral can read it.
func (s *Service) GetByID(ctx context.Context, referralID, viewerID uuid.UUID) (*referral.Referral, error) {
	r, err := s.referrals.GetByID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	if r.ReferrerID != viewerID && r.ProviderID != viewerID && r.ClientID != viewerID {
		return nil, referral.ErrNotAuthorized
	}
	return r, nil
}
