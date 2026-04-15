package referral

import (
	"context"
	"fmt"
	"log/slog"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// activate is called once the client has accepted, the referral row has been
// updated to status=active, and the timeline event has been logged. Its job
// is to side-effect:
//
//  1. Find or create the 1:1 conversation between provider and client. The
//     apporteur is NOT a participant — Modèle A confidentiality.
//  2. Post a system message in that conversation acknowledging the
//     introduction and naming the apporteur.
//  3. Notify the three parties.
//
// All side-effects are best-effort and logged-on-failure: the referral state
// is already persisted, so a downstream failure here must not roll it back.
func (s *Service) activate(ctx context.Context, r *referral.Referral) {
	s.openProviderClientConversation(ctx, r)
	s.notifyClientResponded(ctx, r, true)
}

// openProviderClientConversation finds (or creates) the 1:1 conversation
// between the provider and the client, and posts the activation system
// message inside it. The system message is rendered as
// MessageTypeReferralIntroActivated on the frontend with the apporteur's
// name pulled from the metadata payload.
func (s *Service) openProviderClientConversation(ctx context.Context, r *referral.Referral) {
	if s.messages == nil {
		return
	}

	convID, err := s.messages.FindOrCreateConversation(ctx, service.FindOrCreateConversationInput{
		UserA:   r.ProviderID,
		UserB:   r.ClientID,
		Content: "🤝 Mise en relation activée",
		Type:    string(message.MessageTypeReferralIntroActivated),
	})
	if err != nil {
		slog.Warn("referral: open provider-client conversation failed",
			"referral_id", r.ID, "error", err)
		return
	}

	// FindOrCreateConversation already posts the system message when the
	// conv is freshly created. If the conv already existed (the parties had
	// previously messaged) FindOrCreate also posts the system message
	// because the messaging service implementation always sends it when
	// Content is non-empty — verified in service_system.go. Nothing else
	// to do here.
	_ = convID
	_ = fmt.Sprintf
}
