package skill

import (
	"context"

	"github.com/google/uuid"

	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/port/repository"
)

// Compile-time assertions that the test mocks satisfy the port
// interfaces they stand in for. These lines produce a clean build
// error at the mock definition site if an interface drifts, rather
// than at every test call site.
var (
	_ repository.SkillCatalogRepository = (*mockSkillCatalog)(nil)
	_ repository.ProfileSkillRepository = (*mockProfileSkill)(nil)
	_ OrgTypeResolver                   = (*mockOrgTypeResolver)(nil)
)

// mockSkillCatalog is a hand-written mock for
// repository.SkillCatalogRepository. Each method delegates to its
// *Fn field if set; otherwise it panics with an explicit message so
// an unexpected call in a test produces an obvious failure instead
// of silently returning a zero value.
type mockSkillCatalog struct {
	UpsertFn                  func(ctx context.Context, entry *domainskill.CatalogEntry) error
	FindByTextFn              func(ctx context.Context, skillText string) (*domainskill.CatalogEntry, error)
	ListCuratedByExpertiseFn  func(ctx context.Context, key string, limit int) ([]*domainskill.CatalogEntry, error)
	CountCuratedByExpertiseFn func(ctx context.Context, key string) (int, error)
	SearchAutocompleteFn      func(ctx context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error)
	IncrementUsageCountFn     func(ctx context.Context, skillText string) error
	DecrementUsageCountFn     func(ctx context.Context, skillText string) error
}

func (m *mockSkillCatalog) Upsert(ctx context.Context, entry *domainskill.CatalogEntry) error {
	if m.UpsertFn == nil {
		panic("mockSkillCatalog: unexpected call to Upsert")
	}
	return m.UpsertFn(ctx, entry)
}

func (m *mockSkillCatalog) FindByText(ctx context.Context, skillText string) (*domainskill.CatalogEntry, error) {
	if m.FindByTextFn == nil {
		panic("mockSkillCatalog: unexpected call to FindByText")
	}
	return m.FindByTextFn(ctx, skillText)
}

func (m *mockSkillCatalog) ListCuratedByExpertise(ctx context.Context, key string, limit int) ([]*domainskill.CatalogEntry, error) {
	if m.ListCuratedByExpertiseFn == nil {
		panic("mockSkillCatalog: unexpected call to ListCuratedByExpertise")
	}
	return m.ListCuratedByExpertiseFn(ctx, key, limit)
}

func (m *mockSkillCatalog) CountCuratedByExpertise(ctx context.Context, key string) (int, error) {
	if m.CountCuratedByExpertiseFn == nil {
		panic("mockSkillCatalog: unexpected call to CountCuratedByExpertise")
	}
	return m.CountCuratedByExpertiseFn(ctx, key)
}

func (m *mockSkillCatalog) SearchAutocomplete(ctx context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error) {
	if m.SearchAutocompleteFn == nil {
		panic("mockSkillCatalog: unexpected call to SearchAutocomplete")
	}
	return m.SearchAutocompleteFn(ctx, q, limit)
}

func (m *mockSkillCatalog) IncrementUsageCount(ctx context.Context, skillText string) error {
	if m.IncrementUsageCountFn == nil {
		panic("mockSkillCatalog: unexpected call to IncrementUsageCount")
	}
	return m.IncrementUsageCountFn(ctx, skillText)
}

func (m *mockSkillCatalog) DecrementUsageCount(ctx context.Context, skillText string) error {
	if m.DecrementUsageCountFn == nil {
		panic("mockSkillCatalog: unexpected call to DecrementUsageCount")
	}
	return m.DecrementUsageCountFn(ctx, skillText)
}

// mockProfileSkill is a hand-written mock for
// repository.ProfileSkillRepository following the same convention.
type mockProfileSkill struct {
	ListByOrgIDFn    func(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error)
	ReplaceForOrgFn  func(ctx context.Context, orgID uuid.UUID, skills []*domainskill.ProfileSkill) error
	CountByOrgFn     func(ctx context.Context, orgID uuid.UUID) (int, error)
	DeleteAllByOrgFn func(ctx context.Context, orgID uuid.UUID) error
}

func (m *mockProfileSkill) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error) {
	if m.ListByOrgIDFn == nil {
		panic("mockProfileSkill: unexpected call to ListByOrgID")
	}
	return m.ListByOrgIDFn(ctx, orgID)
}

func (m *mockProfileSkill) ReplaceForOrg(ctx context.Context, orgID uuid.UUID, skills []*domainskill.ProfileSkill) error {
	if m.ReplaceForOrgFn == nil {
		panic("mockProfileSkill: unexpected call to ReplaceForOrg")
	}
	return m.ReplaceForOrgFn(ctx, orgID, skills)
}

func (m *mockProfileSkill) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	if m.CountByOrgFn == nil {
		panic("mockProfileSkill: unexpected call to CountByOrg")
	}
	return m.CountByOrgFn(ctx, orgID)
}

func (m *mockProfileSkill) DeleteAllByOrg(ctx context.Context, orgID uuid.UUID) error {
	if m.DeleteAllByOrgFn == nil {
		panic("mockProfileSkill: unexpected call to DeleteAllByOrg")
	}
	return m.DeleteAllByOrgFn(ctx, orgID)
}

// mockOrgTypeResolver satisfies the local OrgTypeResolver interface.
type mockOrgTypeResolver struct {
	GetOrgTypeFn func(ctx context.Context, orgID uuid.UUID) (string, error)
}

func (m *mockOrgTypeResolver) GetOrgType(ctx context.Context, orgID uuid.UUID) (string, error) {
	if m.GetOrgTypeFn == nil {
		panic("mockOrgTypeResolver: unexpected call to GetOrgType")
	}
	return m.GetOrgTypeFn(ctx, orgID)
}
