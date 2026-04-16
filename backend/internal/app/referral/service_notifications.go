package referral

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// notify is a fire-and-forget wrapper around s.notifications.Send. Notification
// failures must NEVER block the referral state machine — they are observability
// at best. We log a warning and move on.
func (s *Service) notify(ctx context.Context, userID uuid.UUID, t notification.NotificationType, title, body string, data map[string]any) {
	if s.notifications == nil {
		return
	}
	var raw json.RawMessage
	if data != nil {
		b, _ := json.Marshal(data)
		raw = b
	}
	err := s.notifications.Send(ctx, service.NotificationInput{
		UserID: userID,
		Type:   string(t),
		Title:  title,
		Body:   body,
		Data:   raw,
	})
	if err != nil {
		slog.Warn("referral notification failed",
			"user_id", userID, "type", string(t), "error", err)
	}
}

// fanOut delivers a notification to every member of the anchor user's
// organization. Used so agency / enterprise recipients all see the same
// referral event, not just the single user picked from a conversation or
// search result. Falls back to [anchor] when no resolver is configured or
// the anchor has no org.
func (s *Service) fanOut(ctx context.Context, anchor uuid.UUID, t notification.NotificationType, title, body string, data map[string]any) {
	recipients := []uuid.UUID{anchor}
	if s.orgMembers != nil {
		members, err := s.orgMembers.ResolveMemberUserIDs(ctx, anchor)
		if err == nil && len(members) > 0 {
			recipients = members
		}
	}
	for _, uid := range recipients {
		s.notify(ctx, uid, t, title, body, data)
	}
}

// notifyStatusTransition is the single place where every referral lifecycle
// notification is dispatched. Every state machine mutation captures the
// previous status and hands it here, which then selects the set of (party,
// notification type) tuples to send based on the (prev → new) transition.
//
// Each party is expanded via fanOut so agencies and enterprises notify every
// colleague, not just the contact the intro was addressed to. That's the bug
// fix: the client used to miss the "new intro pending your decision" event
// because we only ever notified r.ReferrerID on a provider accept.
func (s *Service) notifyStatusTransition(ctx context.Context, r *referral.Referral, prev referral.Status) {
	data := map[string]any{
		"referral_id": r.ID.String(),
		"rate_pct":    r.RatePct,
	}

	switch r.Status {
	case referral.StatusPendingProvider:
		if prev == "" {
			// Initial intro creation — the provider is now up.
			s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroCreated,
				"Nouvelle recommandation",
				"Un apporteur d'affaires souhaite vous recommander à un client.",
				data)
			return
		}
		if prev == referral.StatusPendingReferrer {
			// Referrer counter-offered the provider's counter-offer.
			s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroNegotiated,
				"Contre-proposition reçue",
				"L'apporteur propose un nouveau taux. Validez ou contre-proposez.",
				data)
		}

	case referral.StatusPendingReferrer:
		// Provider counter-offered — referrer must react.
		s.fanOut(ctx, r.ReferrerID, notification.TypeReferralIntroNegotiated,
			"Contre-proposition reçue",
			"Le prestataire propose un nouveau taux. Validez ou contre-proposez.",
			data)

	case referral.StatusPendingClient:
		// Someone on the apporteur↔provider side just agreed — client is up.
		// Notify the party that just confirmed AND notify the client (all
		// members of the client's org) that a new intro needs their decision.
		switch prev {
		case referral.StatusPendingProvider:
			s.fanOut(ctx, r.ReferrerID, notification.TypeReferralIntroAcceptedByProvider,
				"Prestataire accepté",
				"Le prestataire a accepté votre proposition. Le client va maintenant décider.",
				data)
		case referral.StatusPendingReferrer:
			s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroAcceptedByProvider,
				"Taux validé",
				"L'apporteur a validé votre contre-proposition. Le client va maintenant décider.",
				data)
		}
		s.fanOut(ctx, r.ClientID, notification.TypeReferralIntroCreated,
			"Nouvelle mise en relation",
			"Un apporteur d'affaires vous recommande un prestataire de confiance.",
			data)

	case referral.StatusActive:
		s.fanOut(ctx, r.ReferrerID, notification.TypeReferralIntroActivated,
			"Mise en relation activée",
			"Les deux parties ont accepté. La mise en relation est active.",
			data)
		s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroActivated,
			"Mise en relation activée",
			"Le client a accepté l'introduction. Vous pouvez maintenant échanger.",
			data)
		s.fanOut(ctx, r.ClientID, notification.TypeReferralIntroActivated,
			"Mise en relation activée",
			"Votre mise en relation est active. Vous pouvez maintenant échanger avec le prestataire.",
			data)

	case referral.StatusRejected:
		switch prev {
		case referral.StatusPendingProvider:
			s.fanOut(ctx, r.ReferrerID, notification.TypeReferralIntroRejected,
				"Prestataire a refusé",
				"Le prestataire a refusé votre proposition.",
				data)
		case referral.StatusPendingReferrer:
			s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroRejected,
				"Apporteur a refusé",
				"L'apporteur a refusé votre contre-proposition.",
				data)
		case referral.StatusPendingClient:
			s.fanOut(ctx, r.ReferrerID, notification.TypeReferralIntroRejected,
				"Client a refusé",
				"Le client a refusé la mise en relation.",
				data)
			s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroRejected,
				"Client a refusé",
				"Le client a refusé la mise en relation.",
				data)
		}

	case referral.StatusCancelled:
		// The referrer aborted before activation. Notify every party that
		// was downstream of the cancellation — provider always, and the
		// client only if the intro had already reached their turn.
		s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroCancelled,
			"Mise en relation annulée",
			"L'apporteur a annulé la mise en relation.",
			data)
		if prev == referral.StatusPendingClient {
			s.fanOut(ctx, r.ClientID, notification.TypeReferralIntroCancelled,
				"Mise en relation annulée",
				"L'apporteur a annulé la mise en relation.",
				data)
		}

	case referral.StatusTerminated:
		s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroTerminated,
			"Mise en relation terminée",
			"L'apporteur a mis fin à la mise en relation.",
			data)
		s.fanOut(ctx, r.ClientID, notification.TypeReferralIntroTerminated,
			"Mise en relation terminée",
			"L'apporteur a mis fin à la mise en relation.",
			data)

	case referral.StatusExpired:
		s.fanOut(ctx, r.ReferrerID, notification.TypeReferralIntroExpired,
			"Mise en relation expirée",
			"La mise en relation est arrivée à échéance.",
			data)
		s.fanOut(ctx, r.ProviderID, notification.TypeReferralIntroExpired,
			"Mise en relation expirée",
			"La mise en relation est arrivée à échéance.",
			data)
		// Client only heard about the intro once it reached their turn.
		if prev == referral.StatusPendingClient || prev == referral.StatusActive {
			s.fanOut(ctx, r.ClientID, notification.TypeReferralIntroExpired,
				"Mise en relation expirée",
				"La mise en relation est arrivée à échéance.",
				data)
		}
	}
}

// notifyCommissionPaid fans the "commission received" event out to every
// member of the referrer's org. Agencies where several people share in the
// revenue all deserve to see the income, not just the named apporteur.
func (s *Service) notifyCommissionPaid(ctx context.Context, referralID, referrerID uuid.UUID, commissionCents int64, transferID string) {
	s.fanOut(ctx, referrerID, notification.TypeReferralCommissionPaid,
		"Commission reçue",
		fmt.Sprintf("Vous avez reçu %.2f € de commission.", float64(commissionCents)/100),
		map[string]any{
			"referral_id":      referralID.String(),
			"commission_cents": commissionCents,
			"transfer_id":      transferID,
		})
}

// notifyCommissionPendingKYC fans the "action required" event out to every
// member of the referrer's org so whoever handles compliance sees it.
func (s *Service) notifyCommissionPendingKYC(ctx context.Context, referralID, referrerID uuid.UUID, commissionCents int64) {
	s.fanOut(ctx, referrerID, notification.TypeReferralCommissionPendingKYC,
		"Action requise — KYC à compléter",
		"Une commission est en attente. Complétez votre KYC pour la recevoir.",
		map[string]any{
			"referral_id":      referralID.String(),
			"commission_cents": commissionCents,
		})
}

// notifyCommissionClawedBack fans the clawback event out to every member of
// the referrer's org so the balance drop is visible platform-wide.
func (s *Service) notifyCommissionClawedBack(ctx context.Context, referralID, commissionID, referrerID uuid.UUID, clawbackCents int64, reversalID string) {
	s.fanOut(ctx, referrerID, notification.TypeReferralCommissionClawedBack,
		"Commission reprise",
		fmt.Sprintf("⚠️ %.2f € de commission ont été repris suite à un remboursement.", float64(clawbackCents)/100),
		map[string]any{
			"referral_id":    referralID.String(),
			"commission_id":  commissionID.String(),
			"clawback_cents": clawbackCents,
			"reversal_id":    reversalID,
		})
}
