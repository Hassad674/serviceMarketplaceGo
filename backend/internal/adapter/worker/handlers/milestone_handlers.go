// Package handlers provides the concrete EventHandler implementations
// that the phase-6 worker dispatches to. Each handler decodes its
// typed payload from the pending_event row and calls into the proposal
// app service's system-actor methods (AutoApproveMilestone,
// FundReminderForMilestone, AutoCloseProposal).
//
// Handlers are intentionally tiny — they only translate the JSONB
// payload into typed args. All business logic lives in the service.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/domain/pendingevent"
)

// ProposalScheduler is the small subset of the proposal app service
// that the milestone handlers need. Defined as an interface so the
// handler tests can use a mock without dragging in the whole service.
type ProposalScheduler interface {
	AutoApproveMilestone(ctx context.Context, milestoneID interface{ String() string }) error
	FundReminderForMilestone(ctx context.Context, milestoneID interface{ String() string }) error
	AutoCloseProposal(ctx context.Context, proposalID interface{ String() string }) error
}

// MilestoneAutoApproveHandler dispatches a milestone_auto_approve
// event to the proposal service's system-actor approve+release path.
type MilestoneAutoApproveHandler struct {
	svc *proposalapp.Service
}

// NewMilestoneAutoApproveHandler builds the handler from the proposal
// service. The worker registers it under TypeMilestoneAutoApprove.
func NewMilestoneAutoApproveHandler(svc *proposalapp.Service) *MilestoneAutoApproveHandler {
	return &MilestoneAutoApproveHandler{svc: svc}
}

// Handle decodes the milestone_id from the payload and calls
// AutoApproveMilestone. The proposal service is itself idempotent —
// if the milestone has already been approved manually between the
// schedule time and the fire time, it returns nil without acting.
func (h *MilestoneAutoApproveHandler) Handle(ctx context.Context, event *pendingevent.PendingEvent) error {
	var payload proposalapp.MilestoneEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode milestone_auto_approve payload: %w", err)
	}
	return h.svc.AutoApproveMilestone(ctx, payload.MilestoneID)
}

// MilestoneFundReminderHandler dispatches a milestone_fund_reminder
// event to the proposal service's reminder notification path.
type MilestoneFundReminderHandler struct {
	svc *proposalapp.Service
}

// NewMilestoneFundReminderHandler builds the handler from the
// proposal service. The worker registers it under
// TypeMilestoneFundReminder.
func NewMilestoneFundReminderHandler(svc *proposalapp.Service) *MilestoneFundReminderHandler {
	return &MilestoneFundReminderHandler{svc: svc}
}

// Handle decodes the milestone_id from the payload and calls
// FundReminderForMilestone. Idempotent: if the milestone is no
// longer in pending_funding state when the event fires (because
// the client already paid), the service is a no-op.
func (h *MilestoneFundReminderHandler) Handle(ctx context.Context, event *pendingevent.PendingEvent) error {
	var payload proposalapp.MilestoneEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode milestone_fund_reminder payload: %w", err)
	}
	return h.svc.FundReminderForMilestone(ctx, payload.MilestoneID)
}

// ProposalAutoCloseHandler dispatches a proposal_auto_close event
// to the proposal service's auto-close path. Cancels every
// pending_funding milestone and recomputes the macro status.
type ProposalAutoCloseHandler struct {
	svc *proposalapp.Service
}

// NewProposalAutoCloseHandler builds the handler from the proposal
// service. The worker registers it under TypeProposalAutoClose.
func NewProposalAutoCloseHandler(svc *proposalapp.Service) *ProposalAutoCloseHandler {
	return &ProposalAutoCloseHandler{svc: svc}
}

// Handle decodes the proposal_id from the payload and calls
// AutoCloseProposal. Idempotent: if the proposal is already
// terminal or a milestone has been funded since the event was
// scheduled, the service is a no-op.
func (h *ProposalAutoCloseHandler) Handle(ctx context.Context, event *pendingevent.PendingEvent) error {
	var payload proposalapp.MilestoneEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode proposal_auto_close payload: %w", err)
	}
	return h.svc.AutoCloseProposal(ctx, payload.ProposalID)
}

// StripeTransferHandler dispatches a stripe_transfer outbox event
// to the proposal service's ExecuteStripeTransfer path. The handler
// returns the underlying Stripe error verbatim so the worker's
// MarkFailed + backoff retries the call automatically — the entity
// caps retries at MaxAttempts=5 with exponential backoff, after
// which the row stays in failed status for admin inspection.
//
// Idempotency: payments.TransferToProvider uses Stripe's
// idempotency key derived from the milestone_id, so a retried call
// after a transient failure does not double-transfer.
type StripeTransferHandler struct {
	svc *proposalapp.Service
}

// NewStripeTransferHandler builds the handler from the proposal
// service. The worker registers it under TypeStripeTransfer.
func NewStripeTransferHandler(svc *proposalapp.Service) *StripeTransferHandler {
	return &StripeTransferHandler{svc: svc}
}

// Handle decodes the proposal_id from the payload and calls
// ExecuteStripeTransfer. Errors propagate to the worker which
// applies the backoff schedule.
func (h *StripeTransferHandler) Handle(ctx context.Context, event *pendingevent.PendingEvent) error {
	var payload proposalapp.MilestoneEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode stripe_transfer payload: %w", err)
	}
	return h.svc.ExecuteStripeTransfer(ctx, payload.ProposalID)
}
