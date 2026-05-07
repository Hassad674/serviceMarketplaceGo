package referral

import (
	"context"

	"github.com/google/uuid"
)

// RelationshipChecker is the read-only port the referral service uses to
// detect whether two users (the provider party and the client party of an
// intro) already share a direct business relationship — currently encoded as
// a 1:1 conversation between them.
//
// Anti-fraud rationale: an apporteur d'affaires earns a commission for
// connecting two parties that did not know each other. If the parties have
// already exchanged messages on the platform (i.e. a conversation row exists
// linking the two user ids), they are already in relation and an intro
// between them is rejected at create time. Conversation existence
// transitively covers proposals and payments — both flows live inside an
// existing conversation, so checking conversations alone is the strict
// version of the rule.
//
// Defined IN the referral package (not in port/service) because it is an
// implementation detail of how the referral feature enforces the
// anti-fraud invariant — not a general-purpose port other features should
// consume. Wired in cmd/api/main.go from the messaging adapter.
//
// Implementations MUST be order-insensitive: the same boolean must be
// returned for (userA, userB) and (userB, userA).
type RelationshipChecker interface {
	// AreInRelation returns true when a 1:1 conversation already exists
	// between userA and userB. Returns false (and a nil error) when no
	// conversation links the two parties. Errors are reserved for
	// infrastructure failures (DB unreachable, malformed query).
	//
	// A nil RelationshipChecker MUST be tolerated by the caller as
	// "feature disabled" so unit tests that do not wire the messaging
	// adapter keep working — the production wiring is the only one that
	// guarantees the check fires.
	AreInRelation(ctx context.Context, userA, userB uuid.UUID) (bool, error)
}
