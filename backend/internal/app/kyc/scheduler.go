// Package kyc enforces the KYC deadline for marketplace orgs.
//
// After a mission completes and funds become available, the merchant
// org has 14 days to complete Stripe KYC. If they don't, the team's
// wallet is restricted (cannot create proposals, accept proposals, or
// apply to jobs on behalf of that org).
//
// The Scheduler runs as a background goroutine (1h ticker), queries
// orgs with pending funds but no KYC, and emits notifications at day
// 0/3/7/14. The notification is sent to the org's current owner.
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

// kycSchedulerOrgs is the local composite the scheduler needs: it
// reads the KYC-pending list (Reader) and persists the per-tier
// notification stamps via SaveKYCNotificationState (StripeStore).
// Composing locally keeps the wide port out of the dependency graph.
type kycSchedulerOrgs interface {
	repository.OrganizationReader
	repository.OrganizationStripeStore
}

type SchedulerDeps struct {
	Organizations kycSchedulerOrgs
	Records       repository.PaymentRecordRepository
	Notifications portservice.NotificationSender
}

type Scheduler struct {
	orgs          kycSchedulerOrgs
	records       repository.PaymentRecordRepository
	notifications portservice.NotificationSender
}

func NewScheduler(deps SchedulerDeps) *Scheduler {
	return &Scheduler{
		orgs:          deps.Organizations,
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
	orgs, err := s.orgs.ListKYCPending(ctx)
	if err != nil {
		slog.Error("kyc scheduler: list pending orgs", "error", err)
		return
	}
	if len(orgs) == 0 {
		return
	}

	slog.Debug("kyc scheduler: processing", "pending_orgs", len(orgs))

	for _, org := range orgs {
		if org.KYCFirstEarningAt == nil {
			continue
		}
		elapsed := time.Since(*org.KYCFirstEarningAt)
		elapsedDays := int(elapsed.Hours() / 24)

		notified := org.KYCRestrictionNotifiedAt
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

			amount := s.computePendingAmount(ctx, org.ID)

			title := tier.titleEN
			body := tier.bodyEN
			if amount > 0 {
				body = fmt.Sprintf("%s (%d€ pending)", body, amount/100)
			}

			// Notifications target the org owner (inbox is per-user).
			if err := s.notifications.Send(ctx, portservice.NotificationInput{
				UserID: org.OwnerUserID,
				Type:   string(tier.notifType),
				Title:  title,
				Body:   body,
				Data: mustJSON(map[string]any{
					"tier":         tier.key,
					"amount":       amount,
					"days_elapsed": elapsedDays,
					"org_id":       org.ID.String(),
				}),
			}); err != nil {
				slog.Warn("kyc scheduler: send notification failed",
					"org_id", org.ID, "tier", tier.key, "error", err)
				continue
			}

			notified[tier.key] = time.Now()
			changed = true
			slog.Info("kyc scheduler: notification sent",
				"org_id", org.ID, "tier", tier.key, "days_elapsed", elapsedDays)
		}

		if changed {
			if err := s.orgs.SaveKYCNotificationState(ctx, org.ID, notified); err != nil {
				slog.Warn("kyc scheduler: save notification state failed",
					"org_id", org.ID, "error", err)
			}
		}
	}
}

// computePendingAmount sums up all succeeded payments with pending transfers
// for the given organization.
func (s *Scheduler) computePendingAmount(ctx context.Context, orgID uuid.UUID) int64 {
	records, err := s.records.ListByOrganization(ctx, orgID)
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
