package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/pendingevent"
	domain "marketplace-backend/internal/domain/proposal"
)

// MilestoneEventPayload is the typed payload of every milestone-scoped
// pending event. Marshalled into the pending_events.payload column at
// schedule time and decoded by the worker handler at fire time.
type MilestoneEventPayload struct {
	ProposalID  uuid.UUID `json:"proposal_id"`
	MilestoneID uuid.UUID `json:"milestone_id"`
}

// scheduleMilestoneAutoApprove queues a milestone_auto_approve event
// to fire after the configured auto-approval delay. Idempotency is
// the handler's responsibility — if the milestone is no longer in
// submitted state when the event fires, the handler is a no-op.
//
// Failures here are logged but don't block the caller — we'd rather
// miss an auto-approval than reject a legitimate provider Submit().
func (s *Service) scheduleMilestoneAutoApprove(ctx context.Context, milestoneID uuid.UUID) {
	if s.pendingEvents == nil {
		return
	}
	event, err := s.buildMilestoneEvent(
		pendingevent.TypeMilestoneAutoApprove,
		milestoneID,
		time.Now().Add(s.autoApprovalDelay),
	)
	if err != nil {
		slog.Error("scheduler: build auto-approve event",
			"milestone_id", milestoneID, "error", err)
		return
	}
	if err := s.pendingEvents.Schedule(ctx, event); err != nil {
		slog.Error("scheduler: schedule auto-approve event",
			"milestone_id", milestoneID, "error", err)
	}
}

// scheduleMilestoneFundReminder queues a fund-reminder event for the
// next milestone of a proposal. Called after a milestone is released
// and there is at least one more milestone in pending_funding state.
func (s *Service) scheduleMilestoneFundReminder(ctx context.Context, milestoneID uuid.UUID) {
	if s.pendingEvents == nil {
		return
	}
	event, err := s.buildMilestoneEvent(
		pendingevent.TypeMilestoneFundReminder,
		milestoneID,
		time.Now().Add(s.fundReminderDelay),
	)
	if err != nil {
		slog.Error("scheduler: build fund-reminder event",
			"milestone_id", milestoneID, "error", err)
		return
	}
	if err := s.pendingEvents.Schedule(ctx, event); err != nil {
		slog.Error("scheduler: schedule fund-reminder event",
			"milestone_id", milestoneID, "error", err)
	}
}

// scheduleProposalAutoClose queues a proposal_auto_close event for
// the proposal after autoCloseDelay. Called after a milestone is
// released so a ghosting client triggers a graceful auto-close
// instead of leaving the proposal in limbo forever.
func (s *Service) scheduleProposalAutoClose(ctx context.Context, proposalID uuid.UUID) {
	if s.pendingEvents == nil {
		return
	}
	payload, err := json.Marshal(MilestoneEventPayload{ProposalID: proposalID})
	if err != nil {
		slog.Error("scheduler: marshal auto-close payload",
			"proposal_id", proposalID, "error", err)
		return
	}
	event, err := pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
		EventType: pendingevent.TypeProposalAutoClose,
		Payload:   payload,
		FiresAt:   time.Now().Add(s.autoCloseDelay),
	})
	if err != nil {
		slog.Error("scheduler: build auto-close event",
			"proposal_id", proposalID, "error", err)
		return
	}
	if err := s.pendingEvents.Schedule(ctx, event); err != nil {
		slog.Error("scheduler: schedule auto-close event",
			"proposal_id", proposalID, "error", err)
	}
}

// scheduleStripeTransfer queues a stripe_transfer outbox event for
// the proposal — the outbox-pattern alternative to calling Stripe
// inline at end-of-project. The worker pops the event, calls
// payments.TransferToProvider, and on failure the entity's backoff
// (1m → 5m → 15m → 1h → 6h, capped at MaxAttempts=5) reschedules
// the retry. This gives exactly-once-on-success semantics: a
// transient Stripe 5xx at completion time no longer leaves the
// platform in an inconsistent state.
//
// Schedules with fires_at = now() so the worker picks it up at the
// next tick. Failures here are logged but don't block the caller —
// the same fallback applies as the legacy direct call.
func (s *Service) scheduleStripeTransfer(ctx context.Context, proposalID uuid.UUID) {
	if s.pendingEvents == nil {
		return
	}
	payload, err := json.Marshal(MilestoneEventPayload{ProposalID: proposalID})
	if err != nil {
		slog.Error("scheduler: marshal stripe transfer payload",
			"proposal_id", proposalID, "error", err)
		return
	}
	event, err := pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
		EventType: pendingevent.TypeStripeTransfer,
		Payload:   payload,
		FiresAt:   time.Now(),
	})
	if err != nil {
		slog.Error("scheduler: build stripe transfer event",
			"proposal_id", proposalID, "error", err)
		return
	}
	if err := s.pendingEvents.Schedule(ctx, event); err != nil {
		slog.Error("scheduler: schedule stripe transfer event",
			"proposal_id", proposalID, "error", err)
	}
}

// ExecuteStripeTransfer is the system-actor entry point used by the
// stripe_transfer handler. Calls the underlying payment processor's
// TransferToProvider with retry semantics handled by the worker (any
// error returned here triggers MarkFailed + backoff at the worker
// level, not a manual loop here).
//
// Idempotent: TransferToProvider in the payment service uses Stripe's
// idempotency key derived from the milestone_id, so a retried call
// after a transient failure does not double-transfer.
func (s *Service) ExecuteStripeTransfer(ctx context.Context, proposalID uuid.UUID) error {
	if s.payments == nil {
		// Simulation mode — no Stripe to call. The worker should
		// still mark the event done so it doesn't retry forever.
		return nil
	}
	return s.payments.TransferToProvider(ctx, proposalID)
}

// buildMilestoneEvent is the common factory for milestone-scoped events.
func (s *Service) buildMilestoneEvent(eventType pendingevent.EventType, milestoneID uuid.UUID, firesAt time.Time) (*pendingevent.PendingEvent, error) {
	// Resolve the proposal id so the handler doesn't need a second
	// repo lookup just to know which proposal a milestone belongs to.
	m, err := s.milestones.GetByID(context.Background(), milestoneID)
	if err != nil {
		return nil, fmt.Errorf("get milestone for event payload: %w", err)
	}
	payload, err := json.Marshal(MilestoneEventPayload{
		ProposalID:  m.ProposalID,
		MilestoneID: milestoneID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
		EventType: eventType,
		Payload:   payload,
		FiresAt:   firesAt,
	})
}

// AutoApproveMilestone is the system-actor path that the scheduler
// worker calls when a submitted milestone has aged past the
// auto-approval delay. It bypasses org auth (no requireOrgIsSide
// check) because there is no human caller, but it preserves every
// other guarantee of CompleteProposal: optimistic locking,
// macro recompute, end-of-project effects on macro completion, and
// the milestone_released or proposal_completed system messages.
//
// Idempotency: if the milestone is no longer in submitted state when
// AutoApproveMilestone runs (e.g. the client approved manually 1
// minute before the timer fired), this method is a no-op and returns
// nil so the worker marks the event done without retrying.
func (s *Service) AutoApproveMilestone(ctx context.Context, milestoneID uuid.UUID) error {
	m, err := s.milestones.GetByID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("get milestone: %w", err)
	}
	// Idempotency: the milestone may have been approved/released
	// manually between the schedule time and the fire time. Either
	// way, nothing for the auto-approve handler to do.
	if m.Status != milestone.StatusSubmitted {
		slog.Info("auto-approve: skipping non-submitted milestone",
			"milestone_id", milestoneID, "status", m.Status)
		return nil
	}

	p, err := s.proposals.GetByID(ctx, m.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	if err := s.withMilestoneLock(ctx, m.ID, func(mm *milestone.Milestone) error {
		// Re-check inside the lock so a concurrent manual approval
		// observable to AnotherWorker doesn't cause a double approve.
		if mm.Status != milestone.StatusSubmitted {
			return nil
		}
		if err := mm.Approve(); err != nil {
			return err
		}
		return mm.Release()
	}); err != nil {
		return fmt.Errorf("auto approve+release milestone: %w", err)
	}

	if err := s.recomputeMacroStatus(ctx, p); err != nil {
		return fmt.Errorf("recompute macro status: %w", err)
	}

	// End-of-project effects only on macro completion (last milestone).
	// Mid-project releases just emit the lighter "milestone_released"
	// signal so the next milestone CTA surfaces in the UI.
	if p.Status == domain.StatusCompleted {
		s.runEndOfProjectEffects(ctx, p)
	} else {
		metadata := buildStatusMetadata(p)
		s.sendProposalMessage(ctx, p.ConversationID, uuid.Nil, "milestone_auto_approved", metadata)
		s.sendNotification(ctx, p.ClientID, "milestone_auto_approved", "Milestone auto-approved",
			"You did not respond within the review window — the milestone was automatically approved and paid to the provider.",
			buildNotificationData(p.ID, p.ConversationID, p.Title))
		s.sendNotification(ctx, p.ProviderID, "milestone_auto_approved", "Milestone auto-approved",
			"The client review window expired — the milestone was automatically approved and paid.",
			buildNotificationData(p.ID, p.ConversationID, p.Title))

		// The next milestone is now waiting for funding. Schedule
		// the fund-reminder + auto-close timers so we nudge the
		// client and gracefully end the project if they ghost.
		if next, nextErr := s.milestones.GetCurrentActive(ctx, p.ID); nextErr == nil && next.Status == milestone.StatusPendingFunding {
			s.scheduleMilestoneFundReminder(ctx, next.ID)
			s.scheduleProposalAutoClose(ctx, p.ID)
		}
	}

	return nil
}

// runEndOfProjectEffects bundles the side effects that fire when a
// proposal reaches macro completion: completion + evaluation_request
// system messages, fraud bonus, KYC first earning, Stripe transfer.
//
// Extracted from CompleteProposal so AutoApproveMilestone can call it
// too — both paths land in the same end state and need the same
// downstream effects.
func (s *Service) runEndOfProjectEffects(ctx context.Context, p *domain.Proposal) {
	metadata := s.buildCompletedMetadata(ctx, p)
	s.sendProposalMessage(ctx, p.ConversationID, uuid.Nil, "proposal_completed", metadata)
	s.sendProposalMessage(ctx, p.ConversationID, uuid.Nil, "evaluation_request", metadata)

	s.sendNotification(ctx, p.ProviderID, "proposal_completed", "Mission completed",
		"Your mission has been marked as complete. Leave a review for the client before the 14-day window closes.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	s.sendNotification(ctx, p.ClientID, "proposal_completed", "Mission completed",
		"The mission is marked as complete. Leave a review for the provider before the 14-day window closes.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	s.awardBonusWithFraudCheck(ctx, p)

	// Phase 7: schedule the Stripe transfer through the pending_events
	// outbox so a transient Stripe 5xx no longer leaves the platform
	// in an inconsistent state. The worker retries with exponential
	// backoff, and TransferToProvider's idempotency key on the
	// milestone_id prevents double-transfer on retry.
	//
	// Falls back to the inline call when pendingEvents is not wired
	// (legacy test setups) so the existing happy path keeps working.
	if s.pendingEvents != nil {
		s.scheduleStripeTransfer(ctx, p.ID)
	} else if s.payments != nil {
		if err := s.payments.TransferToProvider(ctx, p.ID); err != nil {
			slog.Error("end-of-project: failed to transfer to provider",
				"proposal_id", p.ID, "error", err)
		}
	}

	if s.orgs != nil && s.users != nil {
		providerUser, lookupErr := s.users.GetByID(ctx, p.ProviderID)
		if lookupErr == nil && providerUser.OrganizationID != nil {
			if err := s.orgs.SetKYCFirstEarning(ctx, *providerUser.OrganizationID, time.Now()); err != nil {
				slog.Warn("auto-approve: failed to record kyc first earning",
					"provider_id", p.ProviderID,
					"org_id", providerUser.OrganizationID,
					"error", err)
			}
		}
	}
}

// FundReminderForMilestone is the system-actor entry point used by
// the milestone_fund_reminder handler. Sends a reminder notification
// to the client. Idempotent: if the milestone is no longer in
// pending_funding state when the event fires, no notification is sent.
func (s *Service) FundReminderForMilestone(ctx context.Context, milestoneID uuid.UUID) error {
	m, err := s.milestones.GetByID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("get milestone: %w", err)
	}
	if m.Status != milestone.StatusPendingFunding {
		return nil
	}
	p, err := s.proposals.GetByID(ctx, m.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}
	s.sendNotification(ctx, p.ClientID, "milestone_fund_reminder", "Fund the next milestone",
		"The next milestone is waiting for your payment to start.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	return nil
}

// AutoCloseProposal is the system-actor entry point used by the
// proposal_auto_close handler. Closes a proposal whose client has
// failed to fund the next milestone within autoCloseDelay. Cancels
// every pending_funding milestone and recomputes the macro status.
//
// Idempotent: if the proposal is already terminal (completed,
// declined, withdrawn) when the event fires, this method is a no-op.
// If the next milestone has been funded since the schedule time,
// also a no-op.
func (s *Service) AutoCloseProposal(ctx context.Context, proposalID uuid.UUID) error {
	p, err := s.proposals.GetByID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}
	switch p.Status {
	case domain.StatusCompleted, domain.StatusDeclined, domain.StatusWithdrawn:
		return nil
	}

	milestones, err := s.milestones.ListByProposal(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("list milestones: %w", err)
	}

	// Only close if the proposal is genuinely waiting for the next
	// milestone to be funded. If a milestone is in flight (funded /
	// submitted / approved / disputed) the client is still engaged
	// and we leave the proposal alone.
	for _, mm := range milestones {
		switch mm.Status {
		case milestone.StatusFunded, milestone.StatusSubmitted, milestone.StatusApproved, milestone.StatusDisputed:
			return nil
		}
	}

	// Sweep every pending_funding milestone to cancelled.
	for _, mm := range milestones {
		if mm.Status != milestone.StatusPendingFunding {
			continue
		}
		if err := s.withMilestoneLock(ctx, mm.ID, func(target *milestone.Milestone) error {
			return target.Cancel()
		}); err != nil {
			slog.Warn("auto-close: cancel milestone failed",
				"milestone_id", mm.ID, "error", err)
		}
	}

	if err := s.recomputeMacroStatus(ctx, p); err != nil {
		return fmt.Errorf("recompute macro status: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, uuid.Nil, "proposal_auto_closed", metadata)
	s.sendNotification(ctx, p.ClientID, "proposal_auto_closed", "Project auto-closed",
		"The project was automatically closed because the next milestone was not funded in time.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	s.sendNotification(ctx, p.ProviderID, "proposal_auto_closed", "Project auto-closed",
		"The project was automatically closed because the client did not fund the next milestone in time. You keep all already-released milestones.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	return nil
}
