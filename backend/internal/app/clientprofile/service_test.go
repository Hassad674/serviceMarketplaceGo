package clientprofile_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/clientprofile"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	reviewdomain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/port/repository"
)

// --- mocks ---

type mockOrgRepo struct {
	findByIDFn func(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
}

func (m *mockOrgRepo) Create(context.Context, *organization.Organization) error { return nil }
func (m *mockOrgRepo) CreateWithOwnerMembership(context.Context, *organization.Organization, *organization.Member) error {
	return nil
}
func (m *mockOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise, Name: "Acme"}, nil
}
func (m *mockOrgRepo) FindByOwnerUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockOrgRepo) FindByUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockOrgRepo) Update(context.Context, *organization.Organization) error { return nil }
func (m *mockOrgRepo) Delete(context.Context, uuid.UUID) error                  { return nil }
func (m *mockOrgRepo) SaveRoleOverrides(context.Context, uuid.UUID, organization.RoleOverrides) error {
	return nil
}
func (m *mockOrgRepo) CountAll(context.Context) (int, error) { return 0, nil }
func (m *mockOrgRepo) FindByStripeAccountID(context.Context, string) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *mockOrgRepo) ListKYCPending(context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (m *mockOrgRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockOrgRepo) GetStripeAccountByUserID(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockOrgRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (m *mockOrgRepo) ClearStripeAccount(context.Context, uuid.UUID) error { return nil }
func (m *mockOrgRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockOrgRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error { return nil }
func (m *mockOrgRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (m *mockOrgRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}
func (m *mockOrgRepo) ListWithStripeAccount(context.Context) ([]uuid.UUID, error) {
	return nil, nil
}

type mockProfileRepo struct {
	getByOrgIDFn           func(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error)
	orgProfilesByUserIDsFn func(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error)
}

func (m *mockProfileRepo) Create(context.Context, *profile.Profile) error { return nil }
func (m *mockProfileRepo) GetByOrganizationID(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	if m.getByOrgIDFn != nil {
		return m.getByOrgIDFn(ctx, orgID)
	}
	p := profile.NewProfile(orgID)
	return p, nil
}
func (m *mockProfileRepo) Update(context.Context, *profile.Profile) error { return nil }
func (m *mockProfileRepo) SearchPublic(context.Context, string, bool, string, int) ([]*profile.PublicProfile, string, error) {
	return nil, "", nil
}
func (m *mockProfileRepo) GetPublicProfilesByOrgIDs(context.Context, []uuid.UUID) ([]*profile.PublicProfile, error) {
	return []*profile.PublicProfile{}, nil
}
func (m *mockProfileRepo) OrgProfilesByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error) {
	if m.orgProfilesByUserIDsFn != nil {
		return m.orgProfilesByUserIDsFn(ctx, userIDs)
	}
	return map[uuid.UUID]*profile.PublicProfile{}, nil
}
func (m *mockProfileRepo) UpdateLocation(context.Context, uuid.UUID, repository.LocationInput) error {
	return nil
}
func (m *mockProfileRepo) UpdateLanguages(context.Context, uuid.UUID, []string, []string) error {
	return nil
}
func (m *mockProfileRepo) UpdateAvailability(context.Context, uuid.UUID, *profile.AvailabilityStatus, *profile.AvailabilityStatus) error {
	return nil
}
func (m *mockProfileRepo) UpdateClientDescription(context.Context, uuid.UUID, string) error {
	return nil
}

// Outbox-aware Tx variants (BUG-05) — clientprofile tests don't drive
// the outbox path, so these are no-op stubs satisfying the interface.
func (m *mockProfileRepo) UpdateTx(context.Context, *sql.Tx, *profile.Profile) error { return nil }
func (m *mockProfileRepo) UpdateLocationTx(context.Context, *sql.Tx, uuid.UUID, repository.LocationInput) error {
	return nil
}
func (m *mockProfileRepo) UpdateLanguagesTx(context.Context, *sql.Tx, uuid.UUID, []string, []string) error {
	return nil
}
func (m *mockProfileRepo) UpdateAvailabilityTx(context.Context, *sql.Tx, uuid.UUID, *profile.AvailabilityStatus, *profile.AvailabilityStatus) error {
	return nil
}

type mockProposalRepo struct {
	sumPaidFn   func(ctx context.Context, orgID uuid.UUID) (int64, error)
	listCompFn  func(ctx context.Context, orgID uuid.UUID, limit int) ([]*proposaldomain.Proposal, error)
}

func (m *mockProposalRepo) Create(context.Context, *proposaldomain.Proposal) error { return nil }
func (m *mockProposalRepo) CreateWithDocuments(context.Context, *proposaldomain.Proposal, []*proposaldomain.ProposalDocument) error {
	return nil
}
func (m *mockProposalRepo) CreateWithDocumentsAndMilestones(context.Context, *proposaldomain.Proposal, []*proposaldomain.ProposalDocument, []*milestonedomain.Milestone) error {
	return nil
}
func (m *mockProposalRepo) GetByID(context.Context, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) GetByIDForOrg(context.Context, uuid.UUID, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) GetByIDs(context.Context, []uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) Update(context.Context, *proposaldomain.Proposal) error { return nil }
func (m *mockProposalRepo) GetLatestVersion(context.Context, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) ListByConversation(context.Context, uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *mockProposalRepo) ListActiveProjectsByOrganization(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}
func (m *mockProposalRepo) ListCompletedByOrganization(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}
func (m *mockProposalRepo) GetDocuments(context.Context, uuid.UUID) ([]*proposaldomain.ProposalDocument, error) {
	return nil, nil
}
func (m *mockProposalRepo) CreateDocument(context.Context, *proposaldomain.ProposalDocument) error {
	return nil
}
func (m *mockProposalRepo) IsOrgAuthorizedForProposal(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockProposalRepo) CountAll(context.Context) (int, int, error) { return 0, 0, nil }
func (m *mockProposalRepo) SumPaidByClientOrganization(ctx context.Context, orgID uuid.UUID) (int64, error) {
	if m.sumPaidFn != nil {
		return m.sumPaidFn(ctx, orgID)
	}
	return 0, nil
}
func (m *mockProposalRepo) ListCompletedByClientOrganization(ctx context.Context, orgID uuid.UUID, limit int) ([]*proposaldomain.Proposal, error) {
	if m.listCompFn != nil {
		return m.listCompFn(ctx, orgID, limit)
	}
	return nil, nil
}

type mockReviewRepo struct {
	listClientFn func(ctx context.Context, orgID uuid.UUID, limit int) ([]*reviewdomain.Review, error)
	clientAvgFn  func(ctx context.Context, orgID uuid.UUID) (*reviewdomain.AverageRating, error)
}

func (m *mockReviewRepo) Create(context.Context, *reviewdomain.Review) error { return nil }
func (m *mockReviewRepo) CreateAndMaybeReveal(_ context.Context, r *reviewdomain.Review) (*reviewdomain.Review, error) {
	return r, nil
}
func (m *mockReviewRepo) GetByID(context.Context, uuid.UUID) (*reviewdomain.Review, error) {
	return nil, nil
}
func (m *mockReviewRepo) GetByIDForOrg(context.Context, uuid.UUID, uuid.UUID) (*reviewdomain.Review, error) {
	return nil, nil
}
func (m *mockReviewRepo) ListByReviewedOrganization(context.Context, uuid.UUID, string, int) ([]*reviewdomain.Review, string, error) {
	return nil, "", nil
}
func (m *mockReviewRepo) GetAverageRatingByOrganization(context.Context, uuid.UUID) (*reviewdomain.AverageRating, error) {
	return &reviewdomain.AverageRating{}, nil
}
func (m *mockReviewRepo) ListClientReviewsByOrganization(ctx context.Context, orgID uuid.UUID, limit int) ([]*reviewdomain.Review, error) {
	if m.listClientFn != nil {
		return m.listClientFn(ctx, orgID, limit)
	}
	return nil, nil
}
func (m *mockReviewRepo) GetClientAverageRating(ctx context.Context, orgID uuid.UUID) (*reviewdomain.AverageRating, error) {
	if m.clientAvgFn != nil {
		return m.clientAvgFn(ctx, orgID)
	}
	return &reviewdomain.AverageRating{}, nil
}
func (m *mockReviewRepo) HasReviewed(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockReviewRepo) GetByProposalIDs(context.Context, []uuid.UUID, string) (map[uuid.UUID]*reviewdomain.Review, error) {
	return map[uuid.UUID]*reviewdomain.Review{}, nil
}
func (m *mockReviewRepo) UpdateReviewModeration(context.Context, uuid.UUID, string, float64, []byte) error {
	return nil
}
func (m *mockReviewRepo) ListAdmin(context.Context, repository.AdminReviewFilters) ([]repository.AdminReview, error) {
	return nil, nil
}
func (m *mockReviewRepo) CountAdmin(context.Context, repository.AdminReviewFilters) (int, error) {
	return 0, nil
}
func (m *mockReviewRepo) GetAdminByID(context.Context, uuid.UUID) (*repository.AdminReview, error) {
	return nil, nil
}
func (m *mockReviewRepo) DeleteAdmin(context.Context, uuid.UUID) error { return nil }

// --- helpers ---

func newTestService(orgs *mockOrgRepo, profiles *mockProfileRepo, proposals *mockProposalRepo, reviews *mockReviewRepo) *clientprofile.Service {
	if orgs == nil {
		orgs = &mockOrgRepo{}
	}
	if profiles == nil {
		profiles = &mockProfileRepo{}
	}
	if proposals == nil {
		proposals = &mockProposalRepo{}
	}
	if reviews == nil {
		reviews = &mockReviewRepo{}
	}
	return clientprofile.NewService(clientprofile.ServiceDeps{
		Organizations: orgs,
		Profiles:      profiles,
		Proposals:     proposals,
		Reviews:       reviews,
	})
}

// --- tests ---

func TestService_GetPublicClientProfile_Enterprise(t *testing.T) {
	orgID := uuid.New()
	providerUserA := uuid.New()

	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise, Name: "Acme Ltd"}, nil
		},
	}
	profiles := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			p := profile.NewProfile(id)
			p.ClientDescription = "We hire great freelancers."
			p.PhotoURL = "https://example.com/logo.png"
			return p, nil
		},
		orgProfilesByUserIDsFn: func(_ context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error) {
			out := map[uuid.UUID]*profile.PublicProfile{}
			for _, uid := range userIDs {
				if uid == providerUserA {
					out[uid] = &profile.PublicProfile{
						OrganizationID: uuid.New(),
						Name:           "Provider Co",
						PhotoURL:       "https://example.com/p.png",
					}
				}
			}
			return out, nil
		},
	}
	proposals := &mockProposalRepo{
		sumPaidFn: func(_ context.Context, _ uuid.UUID) (int64, error) { return 1_234_567, nil },
		listCompFn: func(_ context.Context, _ uuid.UUID, _ int) ([]*proposaldomain.Proposal, error) {
			completedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
			return []*proposaldomain.Proposal{
				{ID: uuid.New(), ProviderID: providerUserA, Title: "Website redesign", Amount: 12345, CompletedAt: &completedAt},
			}, nil
		},
	}
	reviews := &mockReviewRepo{
		clientAvgFn: func(_ context.Context, _ uuid.UUID) (*reviewdomain.AverageRating, error) {
			return &reviewdomain.AverageRating{Average: 4.7, Count: 3}, nil
		},
	}
	svc := newTestService(orgs, profiles, proposals, reviews)

	got, err := svc.GetPublicClientProfile(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "Acme Ltd", got.CompanyName)
	assert.Equal(t, "enterprise", got.Type)
	assert.Equal(t, "We hire great freelancers.", got.ClientDescription)
	assert.Equal(t, "https://example.com/logo.png", got.AvatarURL)
	assert.Equal(t, int64(1_234_567), got.TotalSpent)
	// Count + average come from GetClientAverageRating and feed the
	// header stats block — they must still be populated now that the
	// top-level reviews list is gone.
	assert.Equal(t, 3, got.ReviewCount)
	assert.InDelta(t, 4.7, got.AverageRating, 0.001)
	assert.Equal(t, 1, got.ProjectsCompletedAsClient)
	require.Len(t, got.ProjectHistory, 1)
	assert.Equal(t, "Website redesign", got.ProjectHistory[0].Title)
	require.NotNil(t, got.ProjectHistory[0].Provider, "provider profile must be attached")
	assert.Equal(t, "Provider Co", got.ProjectHistory[0].Provider.Name)
	assert.Equal(t, clientprofile.Currency, got.ProjectHistory[0].Currency)
}

func TestService_GetPublicClientProfile_ProviderPersonalReturnsNotFound(t *testing.T) {
	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeProviderPersonal, Name: "Solo"}, nil
		},
	}
	svc := newTestService(orgs, nil, nil, nil)

	_, err := svc.GetPublicClientProfile(context.Background(), uuid.New())
	assert.ErrorIs(t, err, profile.ErrProfileNotFound)
}

func TestService_GetPublicClientProfile_EmptyProjectHistoryIsSafe(t *testing.T) {
	orgID := uuid.New()
	orgs := &mockOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeAgency, Name: "Newcomer"}, nil
		},
	}
	svc := newTestService(orgs, nil, nil, nil)

	got, err := svc.GetPublicClientProfile(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotNil(t, got.ProjectHistory)
	assert.Empty(t, got.ProjectHistory)
	assert.Equal(t, 0, got.ProjectsCompletedAsClient)
}

func TestService_GetPublicClientProfile_OrgLookupErrorBubblesUp(t *testing.T) {
	wantErr := errors.New("db unavailable")
	orgs := &mockOrgRepo{
		findByIDFn: func(context.Context, uuid.UUID) (*organization.Organization, error) {
			return nil, wantErr
		},
	}
	svc := newTestService(orgs, nil, nil, nil)

	_, err := svc.GetPublicClientProfile(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestService_GetStats(t *testing.T) {
	orgID := uuid.New()
	proposals := &mockProposalRepo{
		sumPaidFn: func(context.Context, uuid.UUID) (int64, error) { return 42_000, nil },
		listCompFn: func(context.Context, uuid.UUID, int) ([]*proposaldomain.Proposal, error) {
			completedAt := time.Now()
			return []*proposaldomain.Proposal{
				{ID: uuid.New(), CompletedAt: &completedAt},
				{ID: uuid.New(), CompletedAt: &completedAt},
			}, nil
		},
	}
	reviews := &mockReviewRepo{
		clientAvgFn: func(context.Context, uuid.UUID) (*reviewdomain.AverageRating, error) {
			return &reviewdomain.AverageRating{Average: 4.2, Count: 2}, nil
		},
	}
	svc := newTestService(nil, nil, proposals, reviews)

	got, err := svc.GetStats(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(42_000), got.TotalSpent)
	assert.Equal(t, 2, got.ReviewCount)
	assert.Equal(t, 2, got.ProjectsCompletedAsClient)
	assert.InDelta(t, 4.2, got.AverageRating, 0.001)
}
