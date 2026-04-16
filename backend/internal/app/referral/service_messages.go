package referral

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// referralMessageMetadata is the JSON payload attached to every referral
// system message. The frontend uses it to render an interactive card with
// role-based action buttons (accept / reject / negotiate) without having
// to fetch the referral first.
//
// Rate is included ONLY when the message is posted into a conv where every
// participant is allowed to see it (apporteur ↔ provider). The client never
// sees the rate — Modèle A confidentiality.
type referralMessageMetadata struct {
	ReferralID string  `json:"referral_id"`
	NewStatus  string  `json:"new_status"`
	PrevStatus string  `json:"prev_status,omitempty"`
	RatePct    float64 `json:"rate_pct,omitempty"`
	IncludeRate bool   `json:"-"`
	ReferrerID string `json:"referrer_id"`
	ProviderID string `json:"provider_id"`
	ClientID   string `json:"client_id"`
}

// marshalMetadata returns the JSON encoding, honouring IncludeRate to strip
// the rate from conv pairs where the client is a participant.
func (m referralMessageMetadata) marshal() json.RawMessage {
	if !m.IncludeRate {
		m.RatePct = 0
	}
	b, err := json.Marshal(m)
	if err != nil {
		// Marshal of a plain struct with primitive fields is unreachable
		// in practice — guard anyway so the caller never sees a nil
		// payload it might render as null.
		return json.RawMessage(`{}`)
	}
	return b
}

// postReferralSystemMessage is the common path every Phase-B hook goes
// through: find-or-create the 1:1 conversation between the two users,
// then inject a system message with rich metadata. The apporteur is the
// sender whenever they are a participant of the target conv — otherwise
// the platform (uuid.Nil) is the system actor, same convention as every
// other cross-feature system-message path in the codebase.
//
// Failures are logged but never returned — referral state is persisted
// before we call this, a downstream messaging outage must not roll it
// back.
func (s *Service) postReferralSystemMessage(ctx context.Context, userA, userB, senderID uuid.UUID, msgType message.MessageType, content string, meta referralMessageMetadata) {
	if s.messages == nil {
		return
	}

	convID, err := s.messages.FindOrCreateConversation(ctx, service.FindOrCreateConversationInput{
		UserA: userA,
		UserB: userB,
	})
	if err != nil {
		slog.Warn("referral: find/create conversation failed",
			"referral_id", meta.ReferralID, "user_a", userA, "user_b", userB, "error", err)
		return
	}

	if err := s.messages.SendSystemMessage(ctx, service.SystemMessageInput{
		ConversationID: convID,
		SenderID:       senderID,
		Content:        content,
		Type:           string(msgType),
		Metadata:       meta.marshal(),
	}); err != nil {
		slog.Warn("referral: send system message failed",
			"referral_id", meta.ReferralID, "type", string(msgType), "error", err)
	}
}

// baseMetadata builds the identity envelope shared by every referral
// system message. Callers fill in NewStatus / PrevStatus / IncludeRate
// per-transition.
func baseMetadata(r *referral.Referral) referralMessageMetadata {
	return referralMessageMetadata{
		ReferralID: r.ID.String(),
		NewStatus:  string(r.Status),
		RatePct:    r.RatePct,
		ReferrerID: r.ReferrerID.String(),
		ProviderID: r.ProviderID.String(),
		ClientID:   r.ClientID.String(),
	}
}

// postTransitionMessages routes a state transition to the right conv
// pairs:
//
//   - apporteur ↔ provider     : always receives an update (they own the
//                                 negotiation). Rate IS included — both
//                                 are allowed to see it.
//   - apporteur ↔ client       : receives an update from pending_client
//                                 onwards. Rate is STRIPPED (Modèle A).
//   - provider   ↔ client      : receives the activation message at
//                                 active; apporteur is never a
//                                 participant of this conv (B2B
//                                 confidentiality).
//
// All three conv pairs are independent: posting into one never touches
// the others. Downstream failure on one conv does not block the others
// either.
func (s *Service) postTransitionMessages(ctx context.Context, r *referral.Referral, prev referral.Status) {
	switch r.Status {
	case referral.StatusPendingProvider:
		if prev == "" {
			// Intro creation. Only the apporteur ↔ provider conv exists
			// in this phase — the client is not yet in the loop.
			meta := baseMetadata(r)
			meta.IncludeRate = true
			meta.PrevStatus = string(prev)
			s.postReferralSystemMessage(ctx, r.ReferrerID, r.ProviderID, r.ReferrerID,
				message.MessageTypeReferralIntroSent,
				"🤝 Nouvelle proposition d'apport d'affaires",
				meta)
			return
		}
		if prev == referral.StatusPendingReferrer {
			// Referrer counter-counter-offered. Negotiation round.
			s.postNegotiationMessage(ctx, r, prev, r.ReferrerID)
		}

	case referral.StatusPendingReferrer:
		// Provider counter-offered.
		s.postNegotiationMessage(ctx, r, prev, r.ProviderID)

	case referral.StatusPendingClient:
		// Apporteur↔provider conv: both parties agreed, record it.
		metaProv := baseMetadata(r)
		metaProv.IncludeRate = true
		metaProv.PrevStatus = string(prev)
		s.postReferralSystemMessage(ctx, r.ReferrerID, r.ProviderID, r.ReferrerID,
			message.MessageTypeReferralIntroSent,
			"✅ Taux validé — en attente du client",
			metaProv)

		// Apporteur↔client conv: NEW post, client is now up. Rate stripped.
		metaCli := baseMetadata(r)
		metaCli.IncludeRate = false
		metaCli.PrevStatus = string(prev)
		s.postReferralSystemMessage(ctx, r.ReferrerID, r.ClientID, r.ReferrerID,
			message.MessageTypeReferralIntroSent,
			"🤝 Nouvelle mise en relation",
			metaCli)

	case referral.StatusActive:
		// Apporteur ↔ provider conv: activation confirmation (with rate).
		metaProv := baseMetadata(r)
		metaProv.IncludeRate = true
		metaProv.PrevStatus = string(prev)
		s.postReferralSystemMessage(ctx, r.ReferrerID, r.ProviderID, r.ReferrerID,
			message.MessageTypeReferralIntroActivated,
			"🎉 Mise en relation activée",
			metaProv)

		// Apporteur ↔ client conv: activation confirmation (no rate).
		metaCli := baseMetadata(r)
		metaCli.IncludeRate = false
		metaCli.PrevStatus = string(prev)
		s.postReferralSystemMessage(ctx, r.ReferrerID, r.ClientID, r.ReferrerID,
			message.MessageTypeReferralIntroActivated,
			"🎉 Mise en relation activée",
			metaCli)

		// Provider ↔ client conv is posted by openProviderClientConversation.
		// Apporteur is NOT a participant — that's the confidentiality
		// barrier of Modèle A.

	case referral.StatusRejected, referral.StatusCancelled,
		referral.StatusTerminated, referral.StatusExpired:
		s.postClosureMessages(ctx, r, prev)
	}
}

// postNegotiationMessage posts one ReferralIntroNegotiated message into
// the apporteur ↔ provider conv whenever either side sends a counter
// offer. The sender is the user who just acted.
func (s *Service) postNegotiationMessage(ctx context.Context, r *referral.Referral, prev referral.Status, sender uuid.UUID) {
	meta := baseMetadata(r)
	meta.IncludeRate = true
	meta.PrevStatus = string(prev)
	s.postReferralSystemMessage(ctx, r.ReferrerID, r.ProviderID, sender,
		message.MessageTypeReferralIntroNegotiated,
		"💬 Contre-proposition de taux",
		meta)
}

// postClosureMessages posts ReferralIntroClosed into every conv pair
// where the referral was visible. Apporteur ↔ provider always. Apporteur
// ↔ client only when the intro had reached the client (pending_client
// or active). Provider ↔ client only once the intro had activated.
func (s *Service) postClosureMessages(ctx context.Context, r *referral.Referral, prev referral.Status) {
	title, body := closureCopy(r.Status)

	metaProv := baseMetadata(r)
	metaProv.IncludeRate = true
	metaProv.PrevStatus = string(prev)
	s.postReferralSystemMessage(ctx, r.ReferrerID, r.ProviderID, uuid.Nil,
		message.MessageTypeReferralIntroClosed, title+" — "+body, metaProv)

	if prev == referral.StatusPendingClient || prev == referral.StatusActive {
		metaCli := baseMetadata(r)
		metaCli.IncludeRate = false
		metaCli.PrevStatus = string(prev)
		s.postReferralSystemMessage(ctx, r.ReferrerID, r.ClientID, uuid.Nil,
			message.MessageTypeReferralIntroClosed, title+" — "+body, metaCli)
	}

	if prev == referral.StatusActive {
		metaPC := baseMetadata(r)
		metaPC.IncludeRate = false
		metaPC.PrevStatus = string(prev)
		s.postReferralSystemMessage(ctx, r.ProviderID, r.ClientID, uuid.Nil,
			message.MessageTypeReferralIntroClosed, title+" — "+body, metaPC)
	}
}

// closureCopy returns a human-readable title/body pair for a terminal
// status. Used by the fallback renderer; the frontend component mostly
// relies on metadata.new_status to pick the right visual treatment.
func closureCopy(s referral.Status) (title, body string) {
	switch s {
	case referral.StatusRejected:
		return "🛑 Mise en relation refusée", "L'une des parties a refusé la proposition."
	case referral.StatusCancelled:
		return "🛑 Mise en relation annulée", "L'apporteur a annulé la proposition."
	case referral.StatusTerminated:
		return "🛑 Mise en relation terminée", "L'apporteur a mis fin à la mise en relation."
	case referral.StatusExpired:
		return "⏳ Mise en relation expirée", "La mise en relation est arrivée à échéance."
	default:
		return "Mise en relation clôturée", ""
	}
}
