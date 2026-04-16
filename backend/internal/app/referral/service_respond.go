package referral

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
)

// RespondAsProvider handles the provider's reaction to an intro that is in
// pending_provider state. The provider can Accept (→ pending_client),
// Negotiate (→ pending_referrer with new rate), or Reject (→ rejected).
//
// The prev→new transition is computed BEFORE the domain mutation so
// notifyStatusTransition can route notifications correctly (who just acted,
// who's up next).
func (s *Service) RespondAsProvider(ctx context.Context, ref ResponseInput) (*referral.Referral, error) {
	r, err := s.loadAndAuthorise(ctx, ref.ReferralID, ref.ActorID, referral.ActorProvider)
	if err != nil {
		return nil, err
	}

	prev := r.Status
	switch ref.Action {
	case referral.NegoActionAccepted:
		if err := r.AcceptByProvider(ref.ActorID); err != nil {
			return nil, err
		}
	case referral.NegoActionRejected:
		if err := r.RejectByProvider(ref.ActorID, ref.Message); err != nil {
			return nil, err
		}
	case referral.NegoActionCountered:
		if err := r.NegotiateByProvider(ref.ActorID, ref.NewRatePct); err != nil {
			return nil, err
		}
	default:
		return nil, referral.ErrInvalidTransition
	}

	if err := s.persistResponse(ctx, r, ref, referral.ActorProvider); err != nil {
		return nil, err
	}
	s.notifyStatusTransition(ctx, r, prev)
	return r, nil
}

// RespondAsReferrer handles the referrer's reaction to a provider counter-offer.
// Possible actions: Accept (→ pending_client), Negotiate (→ pending_provider),
// Reject (→ rejected).
func (s *Service) RespondAsReferrer(ctx context.Context, ref ResponseInput) (*referral.Referral, error) {
	r, err := s.loadAndAuthorise(ctx, ref.ReferralID, ref.ActorID, referral.ActorReferrer)
	if err != nil {
		return nil, err
	}

	prev := r.Status
	switch ref.Action {
	case referral.NegoActionAccepted:
		if err := r.AcceptByReferrer(ref.ActorID); err != nil {
			return nil, err
		}
	case referral.NegoActionRejected:
		if err := r.RejectByReferrer(ref.ActorID, ref.Message); err != nil {
			return nil, err
		}
	case referral.NegoActionCountered:
		if err := r.NegotiateByReferrer(ref.ActorID, ref.NewRatePct); err != nil {
			return nil, err
		}
	default:
		return nil, referral.ErrInvalidTransition
	}

	if err := s.persistResponse(ctx, r, ref, referral.ActorReferrer); err != nil {
		return nil, err
	}
	s.notifyStatusTransition(ctx, r, prev)
	return r, nil
}

// RespondAsClient handles the client's binary decision once the rate is
// locked. Only Accept and Reject are allowed — the client never negotiates
// the rate (Modèle A).
func (s *Service) RespondAsClient(ctx context.Context, ref ResponseInput) (*referral.Referral, error) {
	r, err := s.loadAndAuthorise(ctx, ref.ReferralID, ref.ActorID, referral.ActorClient)
	if err != nil {
		return nil, err
	}

	prev := r.Status
	switch ref.Action {
	case referral.NegoActionAccepted:
		if err := r.AcceptByClient(ref.ActorID); err != nil {
			return nil, err
		}
		// Persist the activation BEFORE opening the conversation so the
		// referral row reflects the active state when the system message
		// adapter looks it up.
		if err := s.referrals.Update(ctx, r); err != nil {
			return nil, fmt.Errorf("update referral on client accept: %w", err)
		}
		if err := s.appendNegotiation(ctx, r, ref, referral.ActorClient); err != nil {
			return nil, err
		}
		s.activate(ctx, r, prev)
		return r, nil
	case referral.NegoActionRejected:
		if err := r.RejectByClient(ref.ActorID, ref.Message); err != nil {
			return nil, err
		}
	default:
		return nil, referral.ErrInvalidTransition
	}

	if err := s.persistResponse(ctx, r, ref, referral.ActorClient); err != nil {
		return nil, err
	}
	s.notifyStatusTransition(ctx, r, prev)
	return r, nil
}

// ResponseInput is the unified payload for the three respond methods.
// The handler dispatches to the right method based on the JWT user's role
// against the referral parties.
type ResponseInput struct {
	ReferralID uuid.UUID
	ActorID    uuid.UUID
	Action     referral.NegotiationAction
	NewRatePct float64
	Message    string
}

// NewResponseInput is a small helper used by both tests and the handler to
// build a ResponseInput positionally — keeps call sites concise.
func NewResponseInput(referralID, actorID uuid.UUID, action referral.NegotiationAction, newRate float64, message string) ResponseInput {
	return ResponseInput{
		ReferralID: referralID,
		ActorID:    actorID,
		Action:     action,
		NewRatePct: newRate,
		Message:    message,
	}
}

// loadAndAuthorise fetches the referral by id and verifies the actor is the
// expected party (referrer / provider / client). Centralised here so the three
// respond methods share the same auth logic.
func (s *Service) loadAndAuthorise(ctx context.Context, referralID, actorID uuid.UUID, expected referral.ActorRole) (*referral.Referral, error) {
	r, err := s.referrals.GetByID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	switch expected {
	case referral.ActorReferrer:
		if r.ReferrerID != actorID {
			return nil, referral.ErrNotAuthorized
		}
	case referral.ActorProvider:
		if r.ProviderID != actorID {
			return nil, referral.ErrNotAuthorized
		}
	case referral.ActorClient:
		if r.ClientID != actorID {
			return nil, referral.ErrNotAuthorized
		}
	}
	return r, nil
}

// persistResponse writes the referral state mutation back to the DB and
// records the negotiation event in the audit table. Called by every respond
// method except RespondAsClient/Accept which has its own activation flow.
func (s *Service) persistResponse(ctx context.Context, r *referral.Referral, in ResponseInput, role referral.ActorRole) error {
	if err := s.referrals.Update(ctx, r); err != nil {
		return fmt.Errorf("update referral: %w", err)
	}
	return s.appendNegotiation(ctx, r, in, role)
}

// appendNegotiation records one row in the audit trail. Rate carried with the
// row is the current referral.RatePct (which the state mutation may have just
// changed for a counter-offer).
func (s *Service) appendNegotiation(ctx context.Context, r *referral.Referral, in ResponseInput, role referral.ActorRole) error {
	nego, err := referral.NewNegotiation(referral.NewNegotiationInput{
		ReferralID: r.ID,
		Version:    r.Version,
		ActorID:    in.ActorID,
		ActorRole:  role,
		Action:     in.Action,
		RatePct:    r.RatePct,
		Message:    in.Message,
	})
	if err != nil {
		return fmt.Errorf("build negotiation row: %w", err)
	}
	return s.referrals.AppendNegotiation(ctx, nego)
}
