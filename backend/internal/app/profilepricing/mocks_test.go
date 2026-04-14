package profilepricing

import (
	"context"

	"github.com/google/uuid"

	domainpricing "marketplace-backend/internal/domain/profilepricing"
	"marketplace-backend/internal/port/repository"
)

// Compile-time assertions that the test mocks satisfy the
// interfaces they stand in for. These lines produce a clean build
// error at the mock definition site if the interface drifts,
// rather than at every test call site.
var (
	_ repository.ProfilePricingRepository = (*mockPricingRepo)(nil)
	_ OrgInfoResolver                     = (*mockOrgInfoResolver)(nil)
)

// mockPricingRepo is a hand-written mock for
// repository.ProfilePricingRepository. Each method delegates to
// its *Fn field when set; otherwise it panics with a descriptive
// message so an unexpected call in a test produces an obvious
// failure instead of a silent zero-value return.
type mockPricingRepo struct {
	UpsertFn       func(ctx context.Context, p *domainpricing.Pricing) error
	FindByOrgIDFn  func(ctx context.Context, orgID uuid.UUID) ([]*domainpricing.Pricing, error)
	ListByOrgIDsFn func(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*domainpricing.Pricing, error)
	DeleteByKindFn func(ctx context.Context, orgID uuid.UUID, kind domainpricing.PricingKind) error
}

func (m *mockPricingRepo) Upsert(ctx context.Context, p *domainpricing.Pricing) error {
	if m.UpsertFn == nil {
		panic("mockPricingRepo: unexpected call to Upsert")
	}
	return m.UpsertFn(ctx, p)
}

func (m *mockPricingRepo) FindByOrgID(ctx context.Context, orgID uuid.UUID) ([]*domainpricing.Pricing, error) {
	if m.FindByOrgIDFn == nil {
		panic("mockPricingRepo: unexpected call to FindByOrgID")
	}
	return m.FindByOrgIDFn(ctx, orgID)
}

func (m *mockPricingRepo) ListByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*domainpricing.Pricing, error) {
	if m.ListByOrgIDsFn == nil {
		panic("mockPricingRepo: unexpected call to ListByOrgIDs")
	}
	return m.ListByOrgIDsFn(ctx, orgIDs)
}

func (m *mockPricingRepo) DeleteByKind(ctx context.Context, orgID uuid.UUID, kind domainpricing.PricingKind) error {
	if m.DeleteByKindFn == nil {
		panic("mockPricingRepo: unexpected call to DeleteByKind")
	}
	return m.DeleteByKindFn(ctx, orgID, kind)
}

// mockOrgInfoResolver satisfies the local OrgInfoResolver contract.
// Defaults to returning a provider_personal + referrer_enabled=true
// when the Fn field is not set, so the happy-path tests do not
// have to wire the resolver explicitly.
type mockOrgInfoResolver struct {
	GetOrgInfoFn func(ctx context.Context, orgID uuid.UUID) (string, bool, error)
}

func (m *mockOrgInfoResolver) GetOrgInfo(ctx context.Context, orgID uuid.UUID) (string, bool, error) {
	if m.GetOrgInfoFn == nil {
		return domainpricing.OrgTypeProviderPersonal, true, nil
	}
	return m.GetOrgInfoFn(ctx, orgID)
}
