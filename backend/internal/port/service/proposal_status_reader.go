package service

import (
	"context"

	"github.com/google/uuid"
)

// ProposalStatusReader exposes the mission lifecycle status of a proposal
// so the payment feature can gate payouts on "mission completed" without
// ever importing the proposal package (feature-independence invariant).
//
// Implementation contract:
//   - Returns the proposal's Status string exactly as stored on the
//     domain entity (e.g. "active", "completion_requested", "completed").
//   - When the proposal cannot be found, returns an empty string with a
//     nil error. The caller is expected to treat an empty status as
//     "unknown, do not transfer" — erroring here would take the entire
//     payout batch down on a single missing row, which is worse than
//     deferring the transfer.
//   - Any non-nil error is a real infrastructure failure (database
//     outage, timeout) and propagates up to the caller.
type ProposalStatusReader interface {
	GetProposalStatus(ctx context.Context, proposalID uuid.UUID) (string, error)
}
