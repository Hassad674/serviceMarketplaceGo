package service

import (
	"context"

	"github.com/google/uuid"
)

// ReferralAttributorInput is the minimal payload the proposal service hands to
// the referral feature when a new contract is signed. It carries only the IDs
// the referral feature needs to look up an active referral on the couple — the
// proposal feature does not import the referral package, so this struct uses
// primitive types only.
type ReferralAttributorInput struct {
	ProposalID uuid.UUID
	ProviderID uuid.UUID
	ClientID   uuid.UUID
}

// ReferralAttributor is implemented by the referral app service and called by
// the proposal app service when a new proposal is accepted/signed.
//
// Contract:
//   - Implementation MUST be a no-op when no active referral matches the
//     (provider, client) couple. The proposal flow continues regardless.
//   - Implementation MUST be idempotent on proposal_id: calling it twice for
//     the same proposal must not create two attributions, and must not return
//     an error on the second call. The postgres unique index handles this.
//   - Errors returned by this port MUST be logged but MUST NOT block the
//     proposal flow — wire it as a defensive call (`if err != nil { slog.Warn ... }`).
//
// The proposal feature has zero compile-time knowledge of the referral
// package. This interface is the only contract.
type ReferralAttributor interface {
	CreateAttributionIfExists(ctx context.Context, input ReferralAttributorInput) error
}
