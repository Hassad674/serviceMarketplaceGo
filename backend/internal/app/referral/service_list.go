package referral

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/repository"
)

// ListByReferrer returns the paginated list of referrals where the given
// user is the apporteur. Thin pass-through to the repository — the filter
// values are validated by the handler layer.
func (s *Service) ListByReferrer(ctx context.Context, referrerID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return s.referrals.ListByReferrer(ctx, referrerID, filter)
}

// ListIncomingForProvider returns the paginated list of referrals where
// the given user is the provider party.
func (s *Service) ListIncomingForProvider(ctx context.Context, providerID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return s.referrals.ListIncomingForProvider(ctx, providerID, filter)
}

// ListIncomingForClient returns the paginated list of referrals where the
// given user is the client party.
func (s *Service) ListIncomingForClient(ctx context.Context, clientID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return s.referrals.ListIncomingForClient(ctx, clientID, filter)
}

// ListNegotiations returns the audit trail of negotiation events for a
// given referral. The handler is expected to first verify the caller is
// one of the three parties before calling this.
func (s *Service) ListNegotiations(ctx context.Context, referralID uuid.UUID) ([]*referral.Negotiation, error) {
	return s.referrals.ListNegotiations(ctx, referralID)
}
