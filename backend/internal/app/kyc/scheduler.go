// Package kyc enforces the KYC deadline for providers/agencies.
//
// After a mission completes and funds become available, the provider has
// 14 days to complete Stripe KYC. If they don't, their account is
// restricted (cannot create proposals, accept proposals, or apply to jobs).
//
// The Scheduler runs as a background goroutine (1h ticker), queries users
// with pending funds but no KYC, and emits notifications at day 0/3/7/14.
package kyc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	notifdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// Notification tiers: days elapsed since first available funds.
var tiers = []struct {
	key       string
	minDays   int
	notifType notifdomain.NotificationType
	titleEN   string
	bodyEN    string
}{
	{
		key: "day0", minDays: 0,
		notifType: notifdomain.TypeKYCReminder,
		titleEN:   "Funds available",
		bodyEN:    "You have funds pending. Set up your payment info to receive them.",
	},
	{
		key: "day3", minDays: 3,
		notifType: notifdomain.TypeKYCReminder,
		titleEN:   "Reminder: pending funds",
		bodyEN:    "You have funds waiting. Set up your payment info to receive them.",
	},
	{
		key: "day7", minDays: 7,
		notifType: notifdomain.TypeKYCReminder,
		titleEN:   "Last reminder before restriction",
		bodyEN:    "Your account will be restricted in 7 days if you don't set up your payment info.",
	},
	{
		key: "day14", minDays: 14,
		notifType: notifdomain.TypeKYCRestriction,
		titleEN:   "Account restricted",
		bodyEN:    "Your account is now restricted. You cannot apply to jobs or create proposals until you set up your payment info.",
	},
}

type SchedulerDeps struct {
	Users         repository.UserRepository
	Records       repository.PaymentRecordRepository
	Notifications portservice.NotificationSender
}

type Scheduler struct {
	users         repository.UserRepository
	records       repository.PaymentRecordRepository
	notifications portservice.NotificationSender
}

func NewScheduler(deps SchedulerDeps) *Scheduler {
	return &Scheduler{
		users:         deps.Users,
		records:       deps.Records,
		notifications: deps.Notifications,
	}
}

// Run blocks until ctx is cancelled. Ticks every interval + runs immediately.
// interval controls the tick frequency (e.g. 1 minute in dev, 1 hour in prod).
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	s.tick(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	users, err := s.users.GetKYCPendingUsers(ctx)
	if err != nil {
		slog.Error("kyc scheduler: get pending users", "error", err)
		return
	}
	if len(users) == 0 {
		return
	}

	slog.Debug("kyc scheduler: processing", "pending_users", len(users))

	for _, u := range users {
		if u.KYCFirstEarningAt == nil {
			continue
		}
		elapsed := time.Since(*u.KYCFirstEarningAt)
		elapsedDays := int(elapsed.Hours() / 24)

		notified := u.KYCRestrictionNotifiedAt
		if notified == nil {
			notified = make(map[string]time.Time)
		}

		changed := false
		for _, tier := range tiers {
			if elapsedDays < tier.minDays {
				break // tiers are ordered, no need to check further
			}
			if _, already := notified[tier.key]; already {
				continue
			}

			// Compute pending amount for the notification body
			amount := s.computePendingAmount(ctx, u.ID)

			title := tier.titleEN
			body := tier.bodyEN
			if amount > 0 {
				body = fmt.Sprintf("%s (%d€ pending)", body, amount/100)
			}

			if err := s.notifications.Send(ctx, portservice.NotificationInput{
				UserID: u.ID,
				Type:   string(tier.notifType),
				Title:  title,
				Body:   body,
				Data:   mustJSON(map[string]any{"tier": tier.key, "amount": amount, "days_elapsed": elapsedDays}),
			}); err != nil {
				slog.Warn("kyc scheduler: send notification failed",
					"user_id", u.ID, "tier", tier.key, "error", err)
				continue
			}

			notified[tier.key] = time.Now()
			changed = true
			slog.Info("kyc scheduler: notification sent",
				"user_id", u.ID, "tier", tier.key, "days_elapsed", elapsedDays)
		}

		if changed {
			if err := s.users.SaveKYCNotificationState(ctx, u.ID, notified); err != nil {
				slog.Warn("kyc scheduler: save notification state failed",
					"user_id", u.ID, "error", err)
			}
		}
	}
}

// computePendingAmount sums up all succeeded payments with pending transfers
// for the given provider.
func (s *Scheduler) computePendingAmount(ctx context.Context, userID uuid.UUID) int64 {
	records, err := s.records.ListByProviderID(ctx, userID)
	if err != nil {
		return 0
	}
	var total int64
	for _, r := range records {
		if r.Status == "succeeded" && r.TransferStatus == "pending" {
			total += r.ProviderPayout
		}
	}
	return total
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
