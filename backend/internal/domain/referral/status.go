package referral

// Status represents the lifecycle state of a Referral.
//
// Phase 1 — bilateral negotiation between referrer and provider on the rate:
//
//	pending_provider  → waiting for the provider to respond (initial offer or
//	                    after the referrer counter-offered)
//	pending_referrer  → waiting for the referrer to respond (after the provider
//	                    counter-offered)
//
// Phase 2 — client decision (no negotiation, just yes/no):
//
//	pending_client    → provider has agreed to the terms, waiting for the client
//	                    to accept the introduction (no rate visible to client)
//
// Active phase — exclusivity window is running:
//
//	active            → all parties accepted, attributions are auto-created on
//	                    any proposal signed between provider and client until
//	                    expires_at
//
// Terminal states (no further transitions):
//
//	rejected          → at least one party refused
//	expired           → 14 days of silence in a pending_* state, OR active referral
//	                    matured past expires_at
//	cancelled         → referrer cancelled the intro before activation
//	terminated        → referrer ended the active referral early
type Status string

const (
	StatusPendingProvider Status = "pending_provider"
	StatusPendingReferrer Status = "pending_referrer"
	StatusPendingClient   Status = "pending_client"
	StatusActive          Status = "active"
	StatusRejected        Status = "rejected"
	StatusExpired         Status = "expired"
	StatusCancelled       Status = "cancelled"
	StatusTerminated      Status = "terminated"
)

// IsValid reports whether s is one of the known status values.
func (s Status) IsValid() bool {
	switch s {
	case StatusPendingProvider, StatusPendingReferrer, StatusPendingClient,
		StatusActive, StatusRejected, StatusExpired, StatusCancelled, StatusTerminated:
		return true
	}
	return false
}

// IsTerminal reports whether s is a terminal state (no further transitions allowed).
func (s Status) IsTerminal() bool {
	switch s {
	case StatusRejected, StatusExpired, StatusCancelled, StatusTerminated:
		return true
	}
	return false
}

// IsPending reports whether s is a pre-activation pending state — used by the
// expirer cron to identify intros at risk of expiring after 14 days of silence.
func (s Status) IsPending() bool {
	switch s {
	case StatusPendingProvider, StatusPendingReferrer, StatusPendingClient:
		return true
	}
	return false
}

// LocksCouple reports whether s prevents another referral from being created
// on the same (provider, client) couple. Used by the unique partial index and
// by the create flow to map duplicate-key errors to ErrCoupleLocked.
func (s Status) LocksCouple() bool {
	return s.IsPending() || s == StatusActive
}

// ActorRole identifies the role of an actor in a referral negotiation.
// Distinct from user.Role because the SAME user can be in multiple roles
// across different referrals.
type ActorRole string

const (
	ActorReferrer ActorRole = "referrer"
	ActorProvider ActorRole = "provider"
	ActorClient   ActorRole = "client"
)

// IsValid reports whether r is a known role.
func (r ActorRole) IsValid() bool {
	switch r {
	case ActorReferrer, ActorProvider, ActorClient:
		return true
	}
	return false
}
