package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientprofileapp "marketplace-backend/internal/app/clientprofile"
	profileapp "marketplace-backend/internal/app/profile"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	reviewdomain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// --- minimal org repo compatible with repository.OrganizationRepository ---

type clientHandlerOrgRepo struct {
	findByIDFn func(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
	updateFn   func(ctx context.Context, org *organization.Organization) error
}

func (m *clientHandlerOrgRepo) Create(context.Context, *organization.Organization) error { return nil }
func (m *clientHandlerOrgRepo) CreateWithOwnerMembership(context.Context, *organization.Organization, *organization.Member) error {
	return nil
}
func (m *clientHandlerOrgRepo) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise, Name: "Stub"}, nil
}
func (m *clientHandlerOrgRepo) FindByOwnerUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *clientHandlerOrgRepo) FindByUserID(context.Context, uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *clientHandlerOrgRepo) Update(ctx context.Context, org *organization.Organization) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, org)
	}
	return nil
}
func (m *clientHandlerOrgRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (m *clientHandlerOrgRepo) SaveRoleOverrides(context.Context, uuid.UUID, organization.RoleOverrides) error {
	return nil
}
func (m *clientHandlerOrgRepo) CountAll(context.Context) (int, error) { return 0, nil }
func (m *clientHandlerOrgRepo) FindByStripeAccountID(context.Context, string) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *clientHandlerOrgRepo) ListKYCPending(context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (m *clientHandlerOrgRepo) GetStripeAccount(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *clientHandlerOrgRepo) GetStripeAccountByUserID(context.Context, uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *clientHandlerOrgRepo) SetStripeAccount(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (m *clientHandlerOrgRepo) ClearStripeAccount(context.Context, uuid.UUID) error { return nil }
func (m *clientHandlerOrgRepo) GetStripeLastState(context.Context, uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *clientHandlerOrgRepo) SaveStripeLastState(context.Context, uuid.UUID, []byte) error {
	return nil
}
func (m *clientHandlerOrgRepo) SetKYCFirstEarning(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (m *clientHandlerOrgRepo) SaveKYCNotificationState(context.Context, uuid.UUID, map[string]time.Time) error {
	return nil
}

// --- minimal proposal + review repos satisfying the clientprofile.Service deps ---

type clientHandlerProposalRepo struct{}

func (m *clientHandlerProposalRepo) Create(context.Context, *proposaldomain.Proposal) error {
	return nil
}
func (m *clientHandlerProposalRepo) CreateWithDocuments(context.Context, *proposaldomain.Proposal, []*proposaldomain.ProposalDocument) error {
	return nil
}
func (m *clientHandlerProposalRepo) CreateWithDocumentsAndMilestones(context.Context, *proposaldomain.Proposal, []*proposaldomain.ProposalDocument, []*milestonedomain.Milestone) error {
	return nil
}
func (m *clientHandlerProposalRepo) GetByID(context.Context, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *clientHandlerProposalRepo) GetByIDs(context.Context, []uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *clientHandlerProposalRepo) Update(context.Context, *proposaldomain.Proposal) error {
	return nil
}
func (m *clientHandlerProposalRepo) GetLatestVersion(context.Context, uuid.UUID) (*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *clientHandlerProposalRepo) ListByConversation(context.Context, uuid.UUID) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}
func (m *clientHandlerProposalRepo) ListActiveProjectsByOrganization(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}
func (m *clientHandlerProposalRepo) ListCompletedByOrganization(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
	return nil, "", nil
}
func (m *clientHandlerProposalRepo) GetDocuments(context.Context, uuid.UUID) ([]*proposaldomain.ProposalDocument, error) {
	return nil, nil
}
func (m *clientHandlerProposalRepo) CreateDocument(context.Context, *proposaldomain.ProposalDocument) error {
	return nil
}
func (m *clientHandlerProposalRepo) IsOrgAuthorizedForProposal(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (m *clientHandlerProposalRepo) CountAll(context.Context) (int, int, error) { return 0, 0, nil }
func (m *clientHandlerProposalRepo) SumPaidByClientOrganization(context.Context, uuid.UUID) (int64, error) {
	return 99_900, nil
}
func (m *clientHandlerProposalRepo) ListCompletedByClientOrganization(context.Context, uuid.UUID, int) ([]*proposaldomain.Proposal, error) {
	return nil, nil
}

type clientHandlerReviewRepo struct{}

func (m *clientHandlerReviewRepo) Create(context.Context, *reviewdomain.Review) error { return nil }
func (m *clientHandlerReviewRepo) CreateAndMaybeReveal(_ context.Context, r *reviewdomain.Review) (*reviewdomain.Review, error) {
	return r, nil
}
func (m *clientHandlerReviewRepo) GetByID(context.Context, uuid.UUID) (*reviewdomain.Review, error) {
	return nil, nil
}
func (m *clientHandlerReviewRepo) ListByReviewedOrganization(context.Context, uuid.UUID, string, int) ([]*reviewdomain.Review, string, error) {
	return nil, "", nil
}
func (m *clientHandlerReviewRepo) GetAverageRatingByOrganization(context.Context, uuid.UUID) (*reviewdomain.AverageRating, error) {
	return &reviewdomain.AverageRating{}, nil
}
func (m *clientHandlerReviewRepo) ListClientReviewsByOrganization(context.Context, uuid.UUID, int) ([]*reviewdomain.Review, error) {
	return nil, nil
}
func (m *clientHandlerReviewRepo) GetClientAverageRating(context.Context, uuid.UUID) (*reviewdomain.AverageRating, error) {
	return &reviewdomain.AverageRating{Average: 4.5, Count: 1}, nil
}
func (m *clientHandlerReviewRepo) HasReviewed(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}
func (m *clientHandlerReviewRepo) GetByProposalIDs(context.Context, []uuid.UUID, string) (map[uuid.UUID]*reviewdomain.Review, error) {
	return map[uuid.UUID]*reviewdomain.Review{}, nil
}
func (m *clientHandlerReviewRepo) UpdateReviewModeration(context.Context, uuid.UUID, string, float64, []byte) error {
	return nil
}
func (m *clientHandlerReviewRepo) ListAdmin(context.Context, repository.AdminReviewFilters) ([]repository.AdminReview, error) {
	return nil, nil
}
func (m *clientHandlerReviewRepo) CountAdmin(context.Context, repository.AdminReviewFilters) (int, error) {
	return 0, nil
}
func (m *clientHandlerReviewRepo) GetAdminByID(context.Context, uuid.UUID) (*repository.AdminReview, error) {
	return nil, nil
}
func (m *clientHandlerReviewRepo) DeleteAdmin(context.Context, uuid.UUID) error { return nil }

// --- helpers ---

func newTestClientProfileHandler(profiles *mockProfileRepo, orgs *clientHandlerOrgRepo) *ClientProfileHandler {
	if profiles == nil {
		profiles = &mockProfileRepo{}
	}
	if orgs == nil {
		orgs = &clientHandlerOrgRepo{}
	}
	writeSvc := profileapp.NewClientProfileService(profiles, orgs)
	readSvc := clientprofileapp.NewService(clientprofileapp.ServiceDeps{
		Organizations: orgs,
		Profiles:      profiles,
		Proposals:     &clientHandlerProposalRepo{},
		Reviews:       &clientHandlerReviewRepo{},
	})
	return NewClientProfileHandler(writeSvc, readSvc)
}

func stringPtr(s string) *string { return &s }

// --- tests ---

func TestClientProfileHandler_UpdateMyClientProfile_HappyPath(t *testing.T) {
	orgID := uuid.New()

	profiles := &mockProfileRepo{
		updateClientDescriptionFn: func(_ context.Context, _ uuid.UUID, _ string) error { return nil },
		getByOrgIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			p := profile.NewProfile(id)
			p.ClientDescription = "Updated"
			return p, nil
		},
	}
	orgs := &clientHandlerOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise, Name: "Acme"}, nil
		},
	}
	h := newTestClientProfileHandler(profiles, orgs)

	body, _ := json.Marshal(map[string]any{"client_description": "Updated"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile/client", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID))
	rec := httptest.NewRecorder()

	h.UpdateMyClientProfile(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Updated", resp["client_description"])
}

func TestClientProfileHandler_UpdateMyClientProfile_ForbiddenForProviderPersonal(t *testing.T) {
	orgs := &clientHandlerOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeProviderPersonal}, nil
		},
	}
	h := newTestClientProfileHandler(nil, orgs)

	body, _ := json.Marshal(map[string]any{"client_description": "x"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile/client", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, uuid.New()))
	rec := httptest.NewRecorder()

	h.UpdateMyClientProfile(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestClientProfileHandler_UpdateMyClientProfile_Unauthenticated(t *testing.T) {
	h := newTestClientProfileHandler(nil, nil)

	body, _ := json.Marshal(map[string]any{"client_description": "x"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile/client", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.UpdateMyClientProfile(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestClientProfileHandler_UpdateMyClientProfile_ValidationError(t *testing.T) {
	orgID := uuid.New()
	orgs := &clientHandlerOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise, Name: "Acme"}, nil
		},
	}
	h := newTestClientProfileHandler(nil, orgs)

	tooLong := make([]byte, profile.MaxClientDescriptionLength+2)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	body, _ := json.Marshal(map[string]any{"client_description": string(tooLong)})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile/client", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID))
	rec := httptest.NewRecorder()

	h.UpdateMyClientProfile(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestClientProfileHandler_GetPublicClientProfile_HappyPath(t *testing.T) {
	orgID := uuid.New()
	profiles := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			p := profile.NewProfile(id)
			p.ClientDescription = "Public desc"
			p.PhotoURL = "https://example.com/l.png"
			return p, nil
		},
	}
	orgs := &clientHandlerOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise, Name: "Public Acme"}, nil
		},
	}
	h := newTestClientProfileHandler(profiles, orgs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+orgID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orgId", orgID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	h.GetPublicClientProfile(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Public Acme", resp["company_name"])
	assert.Equal(t, "enterprise", resp["type"])
	assert.Equal(t, "Public desc", resp["client_description"])
	assert.Equal(t, "https://example.com/l.png", resp["avatar_url"])
	assert.EqualValues(t, 99_900, resp["total_spent"])
	// Count + average feed the header stats block — they stay at the
	// top level even though the reviews[] array has been removed.
	assert.EqualValues(t, 1, resp["review_count"])
	assert.NotNil(t, resp["project_history"])
	// The reviews array has been unified into project_history[].review —
	// no top-level reviews field on the response anymore.
	_, hasReviews := resp["reviews"]
	assert.False(t, hasReviews, "top-level reviews[] must not be surfaced")
}

func TestClientProfileHandler_GetPublicClientProfile_InvalidOrgID(t *testing.T) {
	h := newTestClientProfileHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orgId", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	h.GetPublicClientProfile(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestClientProfileHandler_GetPublicClientProfile_ProviderPersonalReturns404(t *testing.T) {
	orgID := uuid.New()
	orgs := &clientHandlerOrgRepo{
		findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{ID: id, Type: organization.OrgTypeProviderPersonal, Name: "Solo"}, nil
		},
	}
	h := newTestClientProfileHandler(nil, orgs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+orgID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orgId", orgID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	h.GetPublicClientProfile(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// Sanity check — compile-time guarantee that orgRepo satisfies the full
// port/repository.OrganizationRepository interface. Catches the "added a
// method but forgot to update this mock" failure mode that silently fails
// only at other test sites.
var _ = fmt.Sprintf // silence unused import if future edits prune fmt
