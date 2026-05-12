// Package referral implements the apport d'affaires (business referral)
// feature: a provider with referrer_enabled=true introduces another provider
// (or agency) to a client (enterprise or agency), negotiates a commission
// rate bilaterally with the provider, and once the client accepts, an
// exclusivity window opens during which any signed proposal between the
// couple generates commission payouts at every milestone release.
//
// Architecture:
//
//	handler → referral.Service → referral.* (domain) ← postgres adapter
//	                  │
//	                  ├──→ MessageSender (port)        — opens 1:1 conv at activation
//	                  ├──→ NotificationSender (port)   — in-app + push notifs
//	                  ├──→ UserRepository (port)       — role + KYC lookups
//	                  ├──→ StripeService (port)        — commission transfer
//	                  └──→ StripeTransferReversal (port) — clawback
//
// The Service exposes use-case methods consumed by the handler AND implements
// the four exposed ports (ReferralAttributor, ReferralCommissionDistributor,
// ReferralClawback, ReferralKYCListener) so external features can call into it
// without importing this package.
package referral

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ServiceDeps groups the dependency interfaces needed by the referral service.
// Grouping into a struct keeps the constructor stable as new ports are added.
//
// Referrals stays on the wide ReferralRepository — the referral
// service straddles all four segregated children (Reader for the many
// lookup paths, Writer for create/update + negotiations,
// AttributionStore for the proposal anchor, CommissionStore for the
// distributor + clawback + kyc-listener flows). Composing locally
// would reproduce the wide port verbatim.
type ServiceDeps struct {
	Referrals        repository.ReferralRepository
	Users            repository.UserReader
	Messages         service.MessageSender
	Notifications    service.NotificationSender
	Stripe           service.StripeService
	Reversals        service.StripeTransferReversalService
	SnapshotProfiles SnapshotProfileLoader
	StripeAccounts   StripeAccountResolver
	OrgMembers       OrgMemberResolver
	ProposalSummaries ProposalSummaryResolver
	// Relationships detects whether the two parties an apporteur is
	// trying to introduce already share a 1:1 conversation. Anti-fraud
	// gate enforced at create time. Optional: when nil, the check is
	// skipped (legacy unit-test behaviour); production wiring always
	// passes a non-nil checker so the gate fires.
	Relationships RelationshipChecker
	// Audits is the append-only audit log repository. Optional: when
	// nil, audit emission is silently skipped — production wiring
	// always passes a non-nil repository so anti-fraud attempts are
	// recorded for forensic review.
	Audits repository.AuditRepository
	// MilestonesByProposal batch-loads milestones for a slice of
	// proposal ids. Used by ProjectedCommissions (Run B WALLET-UNIFY)
	// to compose the projection × commission row picture in one sweep.
	// Optional: when nil, ProjectedCommissions degrades to returning
	// nil (graceful degradation in worktrees that skipped milestone
	// wiring).
	MilestonesByProposal MilestonesByProposalLister
	// OrgMembersLister resolves org id → member user ids. Required by
	// ProjectedCommissions to fan a per-org wallet query onto the
	// underlying users (referrals.referrer_id is a user id). Optional:
	// nil disables projected commissions for the deployment.
	OrgMembersLister OrgMemberLister
	// PartyDisplayNames resolves a user id into a human-readable label
	// (org name or FullName). Used by the handler to expose
	// provider/client display names on the apporteur detail page.
	// Optional: when nil, names default to empty strings (UI degrades).
	PartyDisplayNames PartyDisplayNameResolver
}

// Service is the referral feature's application service. It implements the
// public ReferralAttributor / Distributor / Clawback / KYCListener ports and
// exposes intro lifecycle use cases (create, respond, cancel, terminate).
type Service struct {
	referrals        repository.ReferralRepository
	users            repository.UserReader
	messages         service.MessageSender
	notifications    service.NotificationSender
	stripe           service.StripeService
	reversals        service.StripeTransferReversalService
	snapshotProfiles SnapshotProfileLoader
	stripeAccounts   StripeAccountResolver
	orgMembers       OrgMemberResolver
	proposalSummaries ProposalSummaryResolver
	relationships    RelationshipChecker
	audits           repository.AuditRepository
	// Run B (WALLET-UNIFY) — narrow ports for ProjectedCommissions.
	milestonesByProposal MilestonesByProposalLister
	orgMemberLister      OrgMemberLister
	partyDisplayNames    PartyDisplayNameResolver
}

// ResolvePartyDisplayName exposes the underlying resolver to the
// handler layer so the DTO can attach human-readable provider/client
// labels. Returns the empty string (and no error) when the resolver
// is not wired or the lookup fails — the UI degrades gracefully to a
// placeholder rather than crashing on a missing label.
func (s *Service) ResolvePartyDisplayName(ctx context.Context, userID uuid.UUID) string {
	if s == nil || s.partyDisplayNames == nil {
		return ""
	}
	name, err := s.partyDisplayNames.ResolveDisplayName(ctx, userID)
	if err != nil {
		return ""
	}
	return name
}

// Compile-time assertions that the Service satisfies the eight exposed ports.
var (
	_ service.ReferralAttributor               = (*Service)(nil)
	_ service.ReferralCommissionDistributor    = (*Service)(nil)
	_ service.ReferralCommissionPreparer       = (*Service)(nil)
	_ service.ReferralClawback                 = (*Service)(nil)
	_ service.ReferralKYCListener              = (*Service)(nil)
	_ service.ReferralWalletReader             = (*Service)(nil)
	_ service.ReferralCommissionRetryService   = (*Service)(nil)
	_ service.ReferralTransferFailureListener  = (*Service)(nil)
)

// NewService wires the referral service from its dependency bag.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		referrals:        deps.Referrals,
		users:            deps.Users,
		messages:         deps.Messages,
		notifications:    deps.Notifications,
		stripe:           deps.Stripe,
		reversals:        deps.Reversals,
		snapshotProfiles: deps.SnapshotProfiles,
		stripeAccounts:   deps.StripeAccounts,
		orgMembers:       deps.OrgMembers,
		proposalSummaries: deps.ProposalSummaries,
		relationships:    deps.Relationships,
		audits:           deps.Audits,
		milestonesByProposal: deps.MilestonesByProposal,
		orgMemberLister:      deps.OrgMembersLister,
		partyDisplayNames:    deps.PartyDisplayNames,
	}
}
