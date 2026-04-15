package referral

import (
	"context"
	"encoding/json"
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

func (s *Service) notifyIntroCreated(ctx context.Context, r *referral.Referral) {
	data := map[string]any{"referral_id": r.ID.String(), "rate_pct": r.RatePct}
	s.notify(ctx, r.ProviderID, notification.TypeReferralIntroCreated,
		"Nouvelle recommandation",
		"Un apporteur d'affaires souhaite vous recommander à un client.",
		data)
}

func (s *Service) notifyProviderResponded(ctx context.Context, r *referral.Referral, accepted bool) {
	if accepted {
		s.notify(ctx, r.ReferrerID, notification.TypeReferralIntroAcceptedByProvider,
			"Provider accepté",
			"Le prestataire a accepté votre proposition. Le client va maintenant décider.",
			map[string]any{"referral_id": r.ID.String()})
	} else {
		s.notify(ctx, r.ReferrerID, notification.TypeReferralIntroRejected,
			"Provider refusé",
			"Le prestataire a refusé votre proposition.",
			map[string]any{"referral_id": r.ID.String()})
	}
}

func (s *Service) notifyClientResponded(ctx context.Context, r *referral.Referral, accepted bool) {
	if accepted {
		s.notify(ctx, r.ReferrerID, notification.TypeReferralIntroAcceptedByClient,
			"Mise en relation activée",
			"Les deux parties ont accepté. La mise en relation est active.",
			map[string]any{"referral_id": r.ID.String()})
		s.notify(ctx, r.ProviderID, notification.TypeReferralIntroActivated,
			"Mise en relation activée",
			"Le client a accepté l'introduction. Vous pouvez maintenant échanger.",
			map[string]any{"referral_id": r.ID.String()})
	} else {
		s.notify(ctx, r.ReferrerID, notification.TypeReferralIntroRejected,
			"Client a refusé",
			"Le client a refusé la mise en relation.",
			map[string]any{"referral_id": r.ID.String()})
		s.notify(ctx, r.ProviderID, notification.TypeReferralIntroRejected,
			"Mise en relation annulée",
			"Le client n'a pas donné suite à la proposition.",
			map[string]any{"referral_id": r.ID.String()})
	}
}

func (s *Service) notifyNegotiated(ctx context.Context, r *referral.Referral, awaitsActor uuid.UUID) {
	s.notify(ctx, awaitsActor, notification.TypeReferralIntroNegotiated,
		"Contre-proposition reçue",
		"Une nouvelle proposition de taux a été envoyée. Validez ou contre-proposez.",
		map[string]any{"referral_id": r.ID.String(), "rate_pct": r.RatePct})
}

func (s *Service) notifyCancelled(ctx context.Context, r *referral.Referral) {
	for _, uid := range []uuid.UUID{r.ProviderID, r.ClientID} {
		s.notify(ctx, uid, notification.TypeReferralIntroCancelled,
			"Mise en relation annulée",
			"L'apporteur a annulé son intro.",
			map[string]any{"referral_id": r.ID.String()})
	}
}

func (s *Service) notifyTerminated(ctx context.Context, r *referral.Referral) {
	for _, uid := range []uuid.UUID{r.ProviderID, r.ClientID} {
		s.notify(ctx, uid, notification.TypeReferralIntroTerminated,
			"Mise en relation terminée",
			"L'apporteur a mis fin à la mise en relation.",
			map[string]any{"referral_id": r.ID.String()})
	}
}

func (s *Service) notifyExpired(ctx context.Context, r *referral.Referral) {
	for _, uid := range []uuid.UUID{r.ReferrerID, r.ProviderID, r.ClientID} {
		s.notify(ctx, uid, notification.TypeReferralIntroExpired,
			"Mise en relation expirée",
			"La mise en relation est arrivée à échéance.",
			map[string]any{"referral_id": r.ID.String()})
	}
}
