package service

import (
	"context"

	"github.com/google/uuid"
)

// PerMilestoneInvoicer is the tiny port the proposal app service consumes
// to emit a platform_fee invoice the moment a milestone is approved.
// Decouples proposal from the invoicing app implementation so the
// invoicing feature stays fully removable and so the proposal package
// never imports app/invoicing directly.
//
// Contract:
//   - Implementation MUST be idempotent on milestone_id — a second call
//     for the same milestone is a silent no-op.
//   - Implementation MUST be a synchronous best-effort: on error the
//     caller logs and continues so the approval flow itself never rolls
//     back. The monthly safety-net scheduler picks up any milestone the
//     synchronous path missed.
//   - No Premium check is required here — the caller decides whether to
//     skip emission by reading the payment record's PlatformFeeAmount
//     (which is already zero for Premium-waived milestones).
type PerMilestoneInvoicer interface {
	// IssueFromMilestone emits the platform_fee invoice for the milestone
	// identified by milestoneID. The implementation resolves the payment
	// record + provider organization from milestoneID internally so the
	// caller never has to reach across feature boundaries to wire those
	// dependencies in.
	//
	// Returns nil for both success AND for skipped emissions
	// (idempotent replay, premium waiver, no fee). Real I/O failures
	// return a non-nil error which the caller logs without rolling back
	// its own transaction.
	IssueFromMilestone(ctx context.Context, milestoneID uuid.UUID) error
}
