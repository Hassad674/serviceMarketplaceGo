package referral

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/domain/referral"
)

// activate is called once the client has accepted, the referral row has been
// updated to status=active, and the timeline event has been logged. Its job
// is to side-effect:
//
//  1. Find or create the 1:1 conversation between provider and client. The
//     apporteur is NOT a participant — Modèle A confidentiality.
//  2. Post a system message in that conversation acknowledging the
//     introduction and naming the apporteur.
//  3. Notify the three parties (fanned out to every org member).
//
// All side-effects are best-effort and logged-on-failure: the referral state
// is already persisted, so a downstream failure here must not roll it back.
func (s *Service) activate(ctx context.Context, r *referral.Referral, prev referral.Status) {
	// Open the provider ↔ client conv first — this is the activation
	// handshake message. It is the ONE conversation the apporteur is
	// NOT a participant of (Modèle A confidentiality).
	s.openProviderClientConversation(ctx, r)
	// Then fan out notifications and post mirrored system messages in
	// the apporteur ↔ provider and apporteur ↔ client conv pairs.
	s.notifyStatusTransition(ctx, r, prev)
	s.postTransitionMessages(ctx, r, prev)
}

// openProviderClientConversation finds (or creates) the 1:1 conversation
// between the provider and the client, and posts the activation system
// message inside it. Posts with rate STRIPPED (client never sees it) and
// the full identity envelope so the interactive widget can link back to
// the referral page.
func (s *Service) openProviderClientConversation(ctx context.Context, r *referral.Referral) {
	if s.messages == nil {
		return
	}
	meta := baseMetadata(r)
	meta.IncludeRate = false
	meta.PrevStatus = string(referral.StatusPendingClient)
	s.postReferralSystemMessage(ctx, r.ProviderID, r.ClientID, uuid.Nil,
		message.MessageTypeReferralIntroActivated,
		"🤝 Mise en relation activée",
		meta)
}
