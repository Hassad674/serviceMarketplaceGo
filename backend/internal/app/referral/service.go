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
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ServiceDeps groups the dependency interfaces needed by the referral service.
// Grouping into a struct keeps the constructor stable as new ports are added.
type ServiceDeps struct {
	Referrals        repository.ReferralRepository
	Users            repository.UserRepository
	Messages         service.MessageSender
	Notifications    service.NotificationSender
	Stripe           service.StripeService
	Reversals        service.StripeTransferReversalService
	SnapshotProfiles SnapshotProfileLoader
	StripeAccounts   StripeAccountResolver
	OrgMembers       OrgMemberResolver
	ProposalSummaries ProposalSummaryResolver
}

// Service is the referral feature's application service. It implements the
// public ReferralAttributor / Distributor / Clawback / KYCListener ports and
// exposes intro lifecycle use cases (create, respond, cancel, terminate).
type Service struct {
	referrals        repository.ReferralRepository
	users            repository.UserRepository
	messages         service.MessageSender
	notifications    service.NotificationSender
	stripe           service.StripeService
	reversals        service.StripeTransferReversalService
	snapshotProfiles SnapshotProfileLoader
	stripeAccounts   StripeAccountResolver
	orgMembers       OrgMemberResolver
	proposalSummaries ProposalSummaryResolver
}

// Compile-time assertions that the Service satisfies the four exposed ports.
var (
	_ service.ReferralAttributor             = (*Service)(nil)
	_ service.ReferralCommissionDistributor  = (*Service)(nil)
	_ service.ReferralClawback               = (*Service)(nil)
	_ service.ReferralKYCListener            = (*Service)(nil)
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
	}
}
