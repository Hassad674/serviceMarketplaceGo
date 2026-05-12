package main

import (
	"context"

	"github.com/google/uuid"

	paymentapp "marketplace-backend/internal/app/payment"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/system"
)

// milestoneStatusBatchReader is the narrow ports the adapter relies on
// off the full MilestoneRepository — defined inline so the adapter only
// declares what it actually consumes (pure ISP, no leakage of the wider
// milestone surface into wire_payment).
type milestoneStatusBatchReader interface {
	StatusByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]milestonedomain.MilestoneStatus, error)
}

// walletMilestoneStatusAdapter satisfies the
// paymentapp.MilestoneStatusReader port by delegating to the postgres
// MilestoneRepository. It is the thin glue piece that bridges the
// payment wallet (which knows nothing about milestones) to the
// milestone repository (which knows nothing about wallets) — pure
// adapter-layer responsibility, no business logic.
//
// SYSTEM-ACTOR: the wallet aggregation is intrinsically cross-org —
// one provider's wallet may aggregate milestones owned by N different
// client proposals. We MUST tag the context with system.WithSystemActor
// before hitting the RLS-protected proposal_milestones table, otherwise
// the policy denies every row whose parent proposal's stakeholder orgs
// do not match the current setting. The attribution gate has already
// been enforced upstream (the wallet only sees payment_records owned
// by the requesting org), so widening the read here is bounded.
type walletMilestoneStatusAdapter struct {
	repo milestoneStatusBatchReader
}

// newWalletMilestoneStatusAdapter wires the adapter. Safe with nil
// repo — every call returns an empty map and nil error so the wallet
// degrades to the conservative escrow-only branch.
func newWalletMilestoneStatusAdapter(repo milestoneStatusBatchReader) *walletMilestoneStatusAdapter {
	return &walletMilestoneStatusAdapter{repo: repo}
}

// StatusByIDs returns a milestone-id → status map under a system-actor
// context. Errors are propagated to the caller — the wallet layer has
// its own fail-soft path (log + degrade to escrow-only).
func (a *walletMilestoneStatusAdapter) StatusByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]milestonedomain.MilestoneStatus, error) {
	if a == nil || a.repo == nil {
		return map[uuid.UUID]milestonedomain.MilestoneStatus{}, nil
	}
	return a.repo.StatusByIDs(system.WithSystemActor(ctx), ids)
}

// Compile-time assertion that the adapter satisfies the wallet's port.
var _ paymentapp.MilestoneStatusReader = (*walletMilestoneStatusAdapter)(nil)

// Compile-time assertion that the postgres MilestoneRepository
// satisfies the narrow batch reader — pins the contract so a future
// refactor (rename, signature change) breaks the build here rather
// than at runtime.
var _ milestoneStatusBatchReader = (repository.MilestoneRepository)(nil)
