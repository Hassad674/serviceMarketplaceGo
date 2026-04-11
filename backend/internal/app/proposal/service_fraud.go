package proposal

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	jobdomain "marketplace-backend/internal/domain/job"
	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/port/repository"
)

const (
	bonusStatusAwarded       = "awarded"
	bonusStatusBlocked       = "blocked"
	bonusStatusPendingReview = "pending_review"

	reasonBelowMinimum      = "below_minimum"
	reasonSameFingerprint   = "same_fingerprint"
	reasonTooFast           = "too_fast"
	reasonTooFrequentDaily  = "too_frequent_daily"
	reasonTooFrequentWeekly = "too_frequent_weekly"

	fastPaymentThreshold     = 2 * time.Minute
	dailyMissionThreshold    = 3
	weeklyMissionThreshold   = 4
)

// awardBonusWithFraudCheck evaluates fraud patterns before awarding bonus credits.
// It always logs the decision in the credit_bonus_log table.
func (s *Service) awardBonusWithFraudCheck(ctx context.Context, p *domain.Proposal) {
	if s.bonusLog == nil {
		// Fallback: no fraud detection configured, award directly (legacy path)
		s.awardBonusDirect(ctx, p.ProviderID)
		return
	}

	status, reason := s.evaluateFraudPatterns(ctx, p)

	creditsAwarded := 0
	if status == bonusStatusAwarded {
		creditsAwarded = jobdomain.BonusPerMission
	}

	entry := &repository.CreditBonusLogEntry{
		ID:                uuid.New(),
		ProviderID:        p.ProviderID,
		ClientID:          p.ClientID,
		ProposalID:        p.ID,
		CreditsAwarded:    creditsAwarded,
		Status:            status,
		BlockReason:       reason,
		ProposalCreatedAt: p.CreatedAt,
		ProposalPaidAt:    time.Now(),
	}

	if err := s.bonusLog.Insert(ctx, entry); err != nil {
		slog.Error("failed to insert credit bonus log",
			"proposal_id", p.ID, "provider_id", p.ProviderID, "error", err)
	}

	if status == bonusStatusAwarded {
		s.awardBonusDirect(ctx, p.ProviderID)
	}
}

// evaluateFraudPatterns checks all fraud rules and returns the resulting status and reason.
func (s *Service) evaluateFraudPatterns(ctx context.Context, p *domain.Proposal) (string, string) {
	// Rule 6: Mission amount below minimum (30 EUR = 3000 cents)
	if p.Amount < int64(jobdomain.MinBonusAmountCent) {
		return bonusStatusBlocked, reasonBelowMinimum
	}

	// Rule 1: Same card fingerprint (skip if not available yet)
	// Card fingerprint comparison is deferred until KYC data is available.
	// The client_card_fingerprint field is stored as empty string for now.

	// Rule 2: Mission created + paid in < 2 minutes
	if !p.CreatedAt.IsZero() {
		elapsed := time.Since(p.CreatedAt)
		if elapsed < fastPaymentThreshold {
			return bonusStatusPendingReview, reasonTooFast
		}
	}

	// Rule 3: 3+ missions with same client in < 24 hours
	dailyCount, err := s.bonusLog.CountByProviderAndClient(
		ctx, p.ProviderID, p.ClientID, time.Now().Add(-24*time.Hour))
	if err != nil {
		slog.Error("failed to count daily bonus log",
			"provider_id", p.ProviderID, "client_id", p.ClientID, "error", err)
		// On error, default to pending review for safety
		return bonusStatusPendingReview, reasonTooFrequentDaily
	}
	if dailyCount >= dailyMissionThreshold {
		return bonusStatusPendingReview, reasonTooFrequentDaily
	}

	// Rule 4: 4+ missions with same client in < 7 days
	weeklyCount, err := s.bonusLog.CountByProviderAndClient(
		ctx, p.ProviderID, p.ClientID, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		slog.Error("failed to count weekly bonus log",
			"provider_id", p.ProviderID, "client_id", p.ClientID, "error", err)
		return bonusStatusPendingReview, reasonTooFrequentWeekly
	}
	if weeklyCount >= weeklyMissionThreshold {
		return bonusStatusPendingReview, reasonTooFrequentWeekly
	}

	// Rule 5: All clean
	return bonusStatusAwarded, ""
}

// awardBonusDirect adds bonus credits without fraud check (used as fallback).
//
// R12 — Credits live on organizations now, so the bonus lands on the
// provider's org (shared by all of its operators) rather than on the
// provider's user row. Resolving the org goes through the existing
// OrganizationRepository dependency — no new wiring needed.
func (s *Service) awardBonusDirect(ctx context.Context, providerID uuid.UUID) {
	if s.credits == nil {
		return
	}
	if s.orgs == nil {
		slog.Warn("skipping bonus credit award: organization repository not configured",
			"provider_id", providerID)
		return
	}
	org, err := s.orgs.FindByUserID(ctx, providerID)
	if err != nil {
		slog.Error("failed to resolve org for bonus credit award",
			"provider_id", providerID, "error", err)
		return
	}
	if err := s.credits.AddBonus(ctx, org.ID, jobdomain.BonusPerMission, jobdomain.MaxTokens); err != nil {
		slog.Error("failed to add bonus credits",
			"provider_id", providerID, "org_id", org.ID, "error", err)
	}
}

// ApproveBonusEntry changes a pending_review entry to awarded and adds credits.
func (s *Service) ApproveBonusEntry(ctx context.Context, entryID uuid.UUID) error {
	if s.bonusLog == nil {
		return nil
	}

	entry, err := s.bonusLog.GetByID(ctx, entryID)
	if err != nil {
		return err
	}
	if entry.Status != bonusStatusPendingReview {
		return domain.ErrInvalidStatus
	}

	if err := s.bonusLog.UpdateStatus(ctx, entryID, bonusStatusAwarded, jobdomain.BonusPerMission); err != nil {
		return err
	}

	s.awardBonusDirect(ctx, entry.ProviderID)
	return nil
}

// RejectBonusEntry changes a pending_review entry to blocked.
func (s *Service) RejectBonusEntry(ctx context.Context, entryID uuid.UUID) error {
	if s.bonusLog == nil {
		return nil
	}

	entry, err := s.bonusLog.GetByID(ctx, entryID)
	if err != nil {
		return err
	}
	if entry.Status != bonusStatusPendingReview {
		return domain.ErrInvalidStatus
	}

	return s.bonusLog.UpdateStatus(ctx, entryID, bonusStatusBlocked, 0)
}

// ListBonusLog returns all bonus log entries with pagination.
func (s *Service) ListBonusLog(ctx context.Context, cursor string, limit int) ([]*repository.CreditBonusLogEntry, string, error) {
	if s.bonusLog == nil {
		return []*repository.CreditBonusLogEntry{}, "", nil
	}
	return s.bonusLog.ListAll(ctx, cursor, limit)
}

// ListPendingBonusLog returns only pending_review bonus log entries.
func (s *Service) ListPendingBonusLog(ctx context.Context, cursor string, limit int) ([]*repository.CreditBonusLogEntry, string, error) {
	if s.bonusLog == nil {
		return []*repository.CreditBonusLogEntry{}, "", nil
	}
	return s.bonusLog.ListPendingReview(ctx, cursor, limit)
}
