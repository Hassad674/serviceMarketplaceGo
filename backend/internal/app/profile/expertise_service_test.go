package profileapp

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/expertise"
	"marketplace-backend/internal/domain/organization"
)

// --- mocks ---

type mockExpertiseRepo struct {
	listByOrgFn    func(ctx context.Context, orgID uuid.UUID) ([]string, error)
	listByOrgIDsFn func(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]string, error)
	replaceFn      func(ctx context.Context, orgID uuid.UUID, keys []string) error
}

func (m *mockExpertiseRepo) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	if m.listByOrgFn != nil {
		return m.listByOrgFn(ctx, orgID)
	}
	return []string{}, nil
}

func (m *mockExpertiseRepo) ListByOrganizationIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if m.listByOrgIDsFn != nil {
		return m.listByOrgIDsFn(ctx, orgIDs)
	}
	return map[uuid.UUID][]string{}, nil
}

func (m *mockExpertiseRepo) Replace(ctx context.Context, orgID uuid.UUID, keys []string) error {
	if m.replaceFn != nil {
		return m.replaceFn(ctx, orgID, keys)
	}
	return nil
}

// mockExpertiseOrgRepo is the tiny subset of repository.OrganizationRepository
// that ExpertiseService exercises — only FindByID. All other methods
// return zero values so the struct satisfies the full interface.
type mockExpertiseOrgRepo struct {
	findByIDFn func(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
}

func (m *mockExpertiseOrgRepo) Create(context.Context, *organization.Organization) error {
	return nil
}
func (m *mockExpertiseOrgRepo) CreateWithOwnerMembership(context.Context, *organization.Organization, *organization.Member) error {
	return nil
}
func (m *mockExpertiseOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return &organization.Organization{ID: id, Type: organization.OrgTypeAgency}, nil
}
func (m *mockExpertiseOrgRepo) FindByOwnerUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockExpertiseOrgRepo) FindByUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockExpertiseOrgRepo) Update(context.Context, *organization.Organization) error {
	return nil
}
func (m *mockExpertiseOrgRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (m *mockExpertiseOrgRepo) CountAll(context.Context) (int, error)   { return 0, nil }
func (m *mockExpertiseOrgRepo) FindByStripeAccountID(context.Context, string) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockExpertiseOrgRepo) ListKYCPending(context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (m *mockExpertiseOrgRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockExpertiseOrgRepo) GetStripeAccountByUserID(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockExpertiseOrgRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (m *mockExpertiseOrgRepo) ClearStripeAccount(context.Context, uuid.UUID) error {
	return nil
}
func (m *mockExpertiseOrgRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockExpertiseOrgRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error {
	return nil
}
func (m *mockExpertiseOrgRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (m *mockExpertiseOrgRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}
func (m *mockExpertiseOrgRepo) SaveRoleOverrides(context.Context, uuid.UUID, organization.RoleOverrides) error {
	return nil
}

// --- helpers ---

func newTestExpertiseService(expRepo *mockExpertiseRepo, orgRepo *mockExpertiseOrgRepo) *ExpertiseService {
	if expRepo == nil {
		expRepo = &mockExpertiseRepo{}
	}
	if orgRepo == nil {
		orgRepo = &mockExpertiseOrgRepo{}
	}
	// We pass the mocks as the repository.OrganizationRepository
	// interface, not as the concrete struct, so the test exercises
	// the same code path as production wiring.
	return NewExpertiseService(expRepo, orgRepo)
}

func agencyOrgResolver(id uuid.UUID) *mockExpertiseOrgRepo {
	return &mockExpertiseOrgRepo{
		findByIDFn: func(_ context.Context, got uuid.UUID) (*organization.Organization, error) {
			if got != id {
				return nil, fmt.Errorf("unexpected org id")
			}
			return &organization.Organization{ID: id, Type: organization.OrgTypeAgency}, nil
		},
	}
}

// --- ListByOrganization tests ---

func TestExpertiseService_ListByOrganization_Success(t *testing.T) {
	orgID := uuid.New()
	expRepo := &mockExpertiseRepo{
		listByOrgFn: func(_ context.Context, got uuid.UUID) ([]string, error) {
			assert.Equal(t, orgID, got)
			return []string{"development", "design_ui_ux"}, nil
		},
	}

	svc := newTestExpertiseService(expRepo, nil)

	result, err := svc.ListByOrganization(context.Background(), orgID)

	require.NoError(t, err)
	assert.Equal(t, []string{"development", "design_ui_ux"}, result)
}

func TestExpertiseService_ListByOrganization_EmptyIsNonNil(t *testing.T) {
	expRepo := &mockExpertiseRepo{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return nil, nil
		},
	}

	svc := newTestExpertiseService(expRepo, nil)

	result, err := svc.ListByOrganization(context.Background(), uuid.New())

	require.NoError(t, err)
	require.NotNil(t, result, "empty list must be non-nil so JSON serializes as [] not null")
	assert.Len(t, result, 0)
}

func TestExpertiseService_ListByOrganization_RepositoryFailure(t *testing.T) {
	expRepo := &mockExpertiseRepo{
		listByOrgFn: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return nil, errors.New("db exploded")
		},
	}

	svc := newTestExpertiseService(expRepo, nil)

	result, err := svc.ListByOrganization(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "list expertise")
}

// --- SetExpertise happy paths ---

func TestExpertiseService_SetExpertise_AgencyHappyPath(t *testing.T) {
	orgID := uuid.New()
	var persistedKeys []string
	expRepo := &mockExpertiseRepo{
		replaceFn: func(_ context.Context, _ uuid.UUID, keys []string) error {
			persistedKeys = keys
			return nil
		},
	}

	svc := newTestExpertiseService(expRepo, agencyOrgResolver(orgID))

	input := []string{"development", "design_ui_ux", "marketing_growth"}
	result, err := svc.SetExpertise(context.Background(), orgID, input)

	require.NoError(t, err)
	assert.Equal(t, input, result, "service should return the normalized list")
	assert.Equal(t, input, persistedKeys, "repository should receive the same list")
}

func TestExpertiseService_SetExpertise_ProviderPersonalHappyPath(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockExpertiseOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeProviderPersonal}, nil
		},
	}
	expRepo := &mockExpertiseRepo{}

	svc := newTestExpertiseService(expRepo, orgRepo)

	result, err := svc.SetExpertise(context.Background(), orgID, []string{"photo_audiovisual"})

	require.NoError(t, err)
	assert.Equal(t, []string{"photo_audiovisual"}, result)
}

func TestExpertiseService_SetExpertise_EmptyListClears(t *testing.T) {
	orgID := uuid.New()
	var called bool
	expRepo := &mockExpertiseRepo{
		replaceFn: func(_ context.Context, _ uuid.UUID, keys []string) error {
			called = true
			assert.Empty(t, keys)
			return nil
		},
	}

	svc := newTestExpertiseService(expRepo, agencyOrgResolver(orgID))

	result, err := svc.SetExpertise(context.Background(), orgID, []string{})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result)
	assert.True(t, called, "repository Replace should still be called to clear the list")
}

func TestExpertiseService_SetExpertise_ReturnsCopyNotAlias(t *testing.T) {
	orgID := uuid.New()
	svc := newTestExpertiseService(&mockExpertiseRepo{}, agencyOrgResolver(orgID))

	input := []string{"development"}
	result, err := svc.SetExpertise(context.Background(), orgID, input)

	require.NoError(t, err)
	require.Equal(t, input, result)

	// Mutating the caller's input must not affect the service's
	// returned slice. Guards against accidental aliasing bugs.
	input[0] = "marketing_growth"
	assert.Equal(t, "development", result[0])
}

// --- SetExpertise error paths ---

func TestExpertiseService_SetExpertise_EnterpriseIsForbidden(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockExpertiseOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise}, nil
		},
	}

	svc := newTestExpertiseService(&mockExpertiseRepo{}, orgRepo)

	result, err := svc.SetExpertise(context.Background(), orgID, []string{"development"})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, expertise.ErrForbiddenOrgType)
}

func TestExpertiseService_SetExpertise_UnknownKey(t *testing.T) {
	orgID := uuid.New()
	svc := newTestExpertiseService(&mockExpertiseRepo{}, agencyOrgResolver(orgID))

	result, err := svc.SetExpertise(context.Background(), orgID, []string{"development", "blockchain"})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, expertise.ErrUnknownKey)
}

func TestExpertiseService_SetExpertise_DuplicateKey(t *testing.T) {
	orgID := uuid.New()
	svc := newTestExpertiseService(&mockExpertiseRepo{}, agencyOrgResolver(orgID))

	result, err := svc.SetExpertise(context.Background(), orgID, []string{"development", "development"})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, expertise.ErrDuplicate)
}

func TestExpertiseService_SetExpertise_OverMax_Agency(t *testing.T) {
	orgID := uuid.New()
	svc := newTestExpertiseService(&mockExpertiseRepo{}, agencyOrgResolver(orgID))

	// 9 distinct valid keys — exceeds the agency max (8).
	over := []string{
		"development", "data_ai_ml", "design_ui_ux", "design_3d_animation",
		"video_motion", "photo_audiovisual", "marketing_growth", "writing_translation",
		"business_dev_sales",
	}
	result, err := svc.SetExpertise(context.Background(), orgID, over)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, expertise.ErrOverMax)
}

func TestExpertiseService_SetExpertise_OverMax_ProviderPersonal(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockExpertiseOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeProviderPersonal}, nil
		},
	}
	svc := newTestExpertiseService(&mockExpertiseRepo{}, orgRepo)

	// 6 distinct valid keys — exceeds the provider_personal max (5).
	over := []string{
		"development", "data_ai_ml", "design_ui_ux", "video_motion",
		"photo_audiovisual", "marketing_growth",
	}
	result, err := svc.SetExpertise(context.Background(), orgID, over)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, expertise.ErrOverMax)
}

func TestExpertiseService_SetExpertise_OrgLookupFailure(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &mockExpertiseOrgRepo{
		findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return nil, organization.ErrOrgNotFound
		},
	}
	svc := newTestExpertiseService(&mockExpertiseRepo{}, orgRepo)

	result, err := svc.SetExpertise(context.Background(), orgID, []string{"development"})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "set expertise: resolve org")
}

func TestExpertiseService_SetExpertise_PersistenceFailure(t *testing.T) {
	orgID := uuid.New()
	expRepo := &mockExpertiseRepo{
		replaceFn: func(_ context.Context, _ uuid.UUID, _ []string) error {
			return errors.New("connection lost")
		},
	}
	svc := newTestExpertiseService(expRepo, agencyOrgResolver(orgID))

	result, err := svc.SetExpertise(context.Background(), orgID, []string{"development"})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "set expertise: persist")
}
