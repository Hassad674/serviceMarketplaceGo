package referral

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"marketplace-backend/internal/domain/referral"
)

// IntroSilenceExpiryDays is the number of days a pending intro can sit without
// any action before the cron moves it to expired. 14 days is a generous but
// finite window: long enough to survive holidays and time-zone gaps, short
// enough that the apporteur can reboot a cold lead with a fresh intro rather
// than wait indefinitely.
const IntroSilenceExpiryDays = 14

// ExpirerBatchSize caps the number of rows the cron processes per invocation.
// Keeps one runaway cron from locking the DB under a sudden spike.
const ExpirerBatchSize = 200

// ExpireStaleIntros scans the pending_* referrals whose last_action_at is
// older than IntroSilenceExpiryDays and transitions each to expired. Called
// by the daily cron handler. Returns the number of rows expired so the cron
// can log a summary.
func (s *Service) ExpireStaleIntros(ctx context.Context) (int, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(IntroSilenceExpiryDays) * 24 * time.Hour)
	rows, err := s.referrals.ListExpiringIntros(ctx, cutoff, ExpirerBatchSize)
	if err != nil {
		return 0, fmt.Errorf("list expiring intros: %w", err)
	}

	expired := 0
	for _, r := range rows {
		if err := s.expireRow(ctx, r); err != nil {
			slog.Warn("referral: expire stale intro failed",
				"referral_id", r.ID, "error", err)
			continue
		}
		expired++
	}
	if expired > 0 {
		slog.Info("referral: stale intros expired", "count", expired)
	}
	return expired, nil
}

// ExpireMaturedReferrals scans active referrals whose expires_at has passed
// and transitions them to expired. Existing attributions and in-flight
// commissions are untouched — they will continue to pay out for milestones
// that were already attributed during the exclusivity window.
func (s *Service) ExpireMaturedReferrals(ctx context.Context) (int, error) {
	rows, err := s.referrals.ListExpiringActives(ctx, time.Now().UTC(), ExpirerBatchSize)
	if err != nil {
		return 0, fmt.Errorf("list expiring actives: %w", err)
	}

	expired := 0
	for _, r := range rows {
		if err := s.expireRow(ctx, r); err != nil {
			slog.Warn("referral: expire matured referral failed",
				"referral_id", r.ID, "error", err)
			continue
		}
		expired++
	}
	if expired > 0 {
		slog.Info("referral: matured referrals expired", "count", expired)
	}
	return expired, nil
}

// RunExpirerCycle is the single entry point called by the cron worker. It
// sequentially processes stale intros and matured referrals so a stuck
// iteration on one type doesn't block the other.
func (s *Service) RunExpirerCycle(ctx context.Context) (staleExpired, maturedExpired int, err error) {
	staleExpired, err = s.ExpireStaleIntros(ctx)
	if err != nil {
		return staleExpired, 0, err
	}
	maturedExpired, err = s.ExpireMaturedReferrals(ctx)
	return staleExpired, maturedExpired, err
}

// expireRow transitions a single referral to expired, persists it, and sends
// the notifications. Any error bubbles up so the batch loop can count it.
func (s *Service) expireRow(ctx context.Context, r *referral.Referral) error {
	prev := r.Status
	if err := r.Expire(); err != nil {
		return err
	}
	if err := s.referrals.Update(ctx, r); err != nil {
		return fmt.Errorf("persist expired referral: %w", err)
	}
	s.notifyStatusTransition(ctx, r, prev)
	return nil
}
