package referral

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// CreateAttributionIfExists implements service.ReferralAttributor.
//
// Called by the proposal feature after a new proposal is signed. Looks up an
// active referral on the (provider, client) couple; if one exists AND its
// exclusivity window has not expired, creates the matching attribution row.
//
// Idempotent: ON CONFLICT (proposal_id) DO NOTHING in the postgres adapter
// turns a second call on the same proposal into a silent no-op. Errors are
// returned but the proposal flow is expected to swallow them — see the
// ReferralAttributor port docstring.
func (s *Service) CreateAttributionIfExists(ctx context.Context, in service.ReferralAttributorInput) error {
	r, err := s.referrals.FindActiveByCouple(ctx, in.ProviderID, in.ClientID)
	if errors.Is(err, referral.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find referral by couple: %w", err)
	}

	// Belt-and-braces check on the exclusivity window. The DB query already
	// filters non-terminal statuses but does not check expires_at, so an
	// active referral whose window has matured (and the cron has not yet
	// run) must NOT generate a new attribution.
	if !r.IsExclusivityActive(time.Now().UTC()) {
		return nil
	}

	att, err := referral.NewAttribution(referral.NewAttributionInput{
		ReferralID:      r.ID,
		ProposalID:      in.ProposalID,
		ProviderID:      in.ProviderID,
		ClientID:        in.ClientID,
		RatePctSnapshot: r.RatePct,
	})
	if err != nil {
		return fmt.Errorf("build attribution: %w", err)
	}
	if err := s.referrals.CreateAttribution(ctx, att); err != nil {
		return fmt.Errorf("persist attribution: %w", err)
	}

	slog.Info("referral attribution created",
		"referral_id", r.ID,
		"proposal_id", in.ProposalID,
		"rate_pct", r.RatePct,
	)
	return nil
}
