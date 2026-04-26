package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------- in-memory test doubles ----------

type bpFakeProfileRepo struct {
	mu       sync.Mutex
	profiles map[uuid.UUID]*domain.BillingProfile
}

func newBPRepo() *bpFakeProfileRepo {
	return &bpFakeProfileRepo{profiles: map[uuid.UUID]*domain.BillingProfile{}}
}

func (r *bpFakeProfileRepo) FindByOrganization(_ context.Context, organizationID uuid.UUID) (*domain.BillingProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[organizationID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *bpFakeProfileRepo) Upsert(_ context.Context, p *domain.BillingProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.profiles[p.OrganizationID] = &cp
	return nil
}

// no-op invoice repo — billing profile tests never touch invoices.
type bpFakeInvoiceRepo struct{}

func (bpFakeInvoiceRepo) CreateInvoice(_ context.Context, _ *domain.Invoice) error { return nil }
func (bpFakeInvoiceRepo) CreateCreditNote(_ context.Context, _ *domain.CreditNote) error {
	return nil
}
func (bpFakeInvoiceRepo) ReserveNumber(_ context.Context, _ domain.CounterScope) (int64, error) {
	return 0, nil
}
func (bpFakeInvoiceRepo) FindInvoiceByID(_ context.Context, _ uuid.UUID) (*domain.Invoice, error) {
	return nil, domain.ErrNotFound
}
func (bpFakeInvoiceRepo) FindInvoiceByStripeEventID(_ context.Context, _ string) (*domain.Invoice, error) {
	return nil, domain.ErrNotFound
}
func (bpFakeInvoiceRepo) FindCreditNoteByStripeEventID(_ context.Context, _ string) (*domain.CreditNote, error) {
	return nil, domain.ErrNotFound
}
func (bpFakeInvoiceRepo) ListInvoicesByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Invoice, string, error) {
	return nil, "", nil
}
func (bpFakeInvoiceRepo) HasInvoiceItemForPaymentRecord(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (bpFakeInvoiceRepo) ListReleasedPaymentRecordsForOrg(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
	return nil, nil
}

type bpFakeStorage struct{}

func (bpFakeStorage) Upload(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
	return "https://r2.test/" + key, nil
}
func (bpFakeStorage) Delete(_ context.Context, _ string) error           { return nil }
func (bpFakeStorage) GetPublicURL(key string) string                     { return "https://r2.test/" + key }
func (bpFakeStorage) GetPresignedUploadURL(_ context.Context, key string, _ string, _ time.Duration) (string, error) {
	return "https://r2.test/upload/" + key, nil
}
func (bpFakeStorage) GetPresignedDownloadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://r2.test/download/" + key, nil
}
func (bpFakeStorage) Download(_ context.Context, _ string) ([]byte, error) { return nil, nil }

type bpFakePDF struct{}

func (bpFakePDF) RenderInvoice(_ context.Context, _ *domain.Invoice, _ string) ([]byte, error) {
	return nil, nil
}
func (bpFakePDF) RenderCreditNote(_ context.Context, _ *domain.CreditNote, _ string) ([]byte, error) {
	return nil, nil
}

type bpFakeDeliverer struct{}

func (bpFakeDeliverer) DeliverInvoice(_ context.Context, _ *domain.Invoice, _ []byte, _ string) error {
	return nil
}
func (bpFakeDeliverer) DeliverCreditNote(_ context.Context, _ *domain.CreditNote, _ []byte, _ string) error {
	return nil
}

type bpFakeIdempotency struct{}

func (bpFakeIdempotency) TryClaim(_ context.Context, _ string) (bool, error) { return true, nil }

// Stripe KYC reader stub — programmable.
type bpFakeKYC struct {
	snap *service.StripeAccountKYCSnapshot
	err  error
}

func (m *bpFakeKYC) GetAccountKYCSnapshot(_ context.Context, _ string) (*service.StripeAccountKYCSnapshot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.snap, nil
}

// VIES validator stub.
type bpFakeVIES struct {
	res service.VIESResult
	err error
}

func (m *bpFakeVIES) Validate(_ context.Context, _, _ string) (service.VIESResult, error) {
	if m.err != nil {
		return service.VIESResult{}, m.err
	}
	return m.res, nil
}

// Org repo stub — only the StripeAccount lookup is exercised.
type bpFakeOrgRepo struct {
	stripeAccountByOrg map[uuid.UUID]string
}

func (m *bpFakeOrgRepo) Create(_ context.Context, _ *organization.Organization) error { return nil }
func (m *bpFakeOrgRepo) CreateWithOwnerMembership(_ context.Context, _ *organization.Organization, _ *organization.Member) error {
	return nil
}
func (m *bpFakeOrgRepo) FindByID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *bpFakeOrgRepo) FindByOwnerUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *bpFakeOrgRepo) FindByUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *bpFakeOrgRepo) Update(_ context.Context, _ *organization.Organization) error { return nil }
func (m *bpFakeOrgRepo) Delete(_ context.Context, _ uuid.UUID) error                  { return nil }
func (m *bpFakeOrgRepo) SaveRoleOverrides(_ context.Context, _ uuid.UUID, _ organization.RoleOverrides) error {
	return nil
}
func (m *bpFakeOrgRepo) CountAll(_ context.Context) (int, error) { return 0, nil }
func (m *bpFakeOrgRepo) FindByStripeAccountID(_ context.Context, _ string) (*organization.Organization, error) {
	return nil, organization.ErrOrgNotFound
}
func (m *bpFakeOrgRepo) ListKYCPending(_ context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (m *bpFakeOrgRepo) ListWithStripeAccount(_ context.Context) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *bpFakeOrgRepo) GetStripeAccount(_ context.Context, orgID uuid.UUID) (string, string, error) {
	if v, ok := m.stripeAccountByOrg[orgID]; ok {
		return v, "FR", nil
	}
	return "", "", nil
}
func (m *bpFakeOrgRepo) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *bpFakeOrgRepo) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *bpFakeOrgRepo) ClearStripeAccount(_ context.Context, _ uuid.UUID) error { return nil }
func (m *bpFakeOrgRepo) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *bpFakeOrgRepo) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *bpFakeOrgRepo) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *bpFakeOrgRepo) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error {
	return nil
}

// ---------- harness ----------

type bpHarness struct {
	handler  *handler.BillingProfileHandler
	profiles *bpFakeProfileRepo
	kyc      *bpFakeKYC
	vies     *bpFakeVIES
	orgRepo  *bpFakeOrgRepo
	userID   uuid.UUID
	orgID    uuid.UUID
}

func newBPHarness(t *testing.T) *bpHarness {
	t.Helper()
	profiles := newBPRepo()
	kyc := &bpFakeKYC{}
	vies := &bpFakeVIES{}
	orgRepo := &bpFakeOrgRepo{stripeAccountByOrg: map[uuid.UUID]string{}}

	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    bpFakeInvoiceRepo{},
		Profiles:    profiles,
		PDF:         bpFakePDF{},
		Storage:     bpFakeStorage{},
		Deliverer:   bpFakeDeliverer{},
		Issuer:      domain.IssuerInfo{Country: "FR", LegalName: "Issuer SAS"},
		Idempotency: bpFakeIdempotency{},
	})
	svc.SetBillingProfileDeps(invoicingapp.BillingProfileDeps{
		Organizations: orgRepo,
		StripeKYC:     kyc,
		VIESValidator: vies,
	})

	return &bpHarness{
		handler:  handler.NewBillingProfileHandler(svc),
		profiles: profiles,
		kyc:      kyc,
		vies:     vies,
		orgRepo:  orgRepo,
		userID:   uuid.New(),
		orgID:    uuid.New(),
	}
}

// reqWithAuth builds an authenticated request — inserts user_id +
// organization_id into the context the same way the real auth middleware
// does, so the handler's requireOrg helper succeeds.
func (h *bpHarness) reqWithAuth(method, target string, body any) *http.Request {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, target, rdr)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, h.userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, h.orgID)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// ---------- tests ----------

func TestBillingProfile_Get_FirstTime_ReturnsEmptyStub(t *testing.T) {
	h := newBPHarness(t)
	req := h.reqWithAuth(http.MethodGet, "/api/v1/me/billing-profile", nil)
	rec := httptest.NewRecorder()

	h.handler.GetMine(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, false, body["is_complete"])
	missing, _ := body["missing_fields"].([]any)
	assert.NotEmpty(t, missing, "first-time stub should report missing fields")
	profile, _ := body["profile"].(map[string]any)
	assert.Equal(t, "", profile["legal_name"])
}

func TestBillingProfile_Get_WithRow_ReturnsProfile(t *testing.T) {
	h := newBPHarness(t)
	now := time.Now().UTC()
	validated := now
	require.NoError(t, h.profiles.Upsert(context.Background(), &domain.BillingProfile{
		OrganizationID: h.orgID,
		ProfileType:    domain.ProfileBusiness,
		LegalName:      "Acme SAS",
		AddressLine1:   "1 rue de la Paix",
		PostalCode:     "75002",
		City:           "Paris",
		Country:        "FR",
		InvoicingEmail: "billing@acme.test",
		TaxID:          "12345678901234",
		VATNumber:      "FR12345678901",
		VATValidatedAt: &validated,
		CreatedAt:      now,
		UpdatedAt:      now,
	}))

	req := h.reqWithAuth(http.MethodGet, "/api/v1/me/billing-profile", nil)
	rec := httptest.NewRecorder()
	h.handler.GetMine(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, true, body["is_complete"])
	profile, _ := body["profile"].(map[string]any)
	assert.Equal(t, "Acme SAS", profile["legal_name"])
}

func TestBillingProfile_Put_PartialUpdate_Persists(t *testing.T) {
	h := newBPHarness(t)
	body := map[string]any{
		"profile_type":    "business",
		"legal_name":      "Acme SAS",
		"address_line1":   "1 rue de la Paix",
		"postal_code":     "75002",
		"city":            "Paris",
		"country":         "FR",
		"invoicing_email": "billing@acme.test",
	}
	req := h.reqWithAuth(http.MethodPut, "/api/v1/me/billing-profile", body)
	rec := httptest.NewRecorder()
	h.handler.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	saved, err := h.profiles.FindByOrganization(context.Background(), h.orgID)
	require.NoError(t, err)
	assert.Equal(t, "Acme SAS", saved.LegalName)
	assert.Equal(t, "FR", saved.Country)
	// Without TaxID for FR it remains incomplete — surfaced in body too.
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["is_complete"])
}

func TestBillingProfile_Put_ChangingVATClearsValidation(t *testing.T) {
	h := newBPHarness(t)
	now := time.Now().UTC()
	validated := now
	require.NoError(t, h.profiles.Upsert(context.Background(), &domain.BillingProfile{
		OrganizationID: h.orgID,
		ProfileType:    domain.ProfileBusiness,
		LegalName:      "Acme SAS",
		Country:        "FR",
		VATNumber:      "FR12345678901",
		VATValidatedAt: &validated,
		CreatedAt:      now,
		UpdatedAt:      now,
	}))
	body := map[string]any{
		"profile_type": "business",
		"legal_name":   "Acme SAS",
		"country":      "FR",
		"vat_number":   "FR99999999999",
	}
	req := h.reqWithAuth(http.MethodPut, "/api/v1/me/billing-profile", body)
	rec := httptest.NewRecorder()
	h.handler.Update(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	saved, err := h.profiles.FindByOrganization(context.Background(), h.orgID)
	require.NoError(t, err)
	assert.Equal(t, "FR99999999999", saved.VATNumber)
	assert.Nil(t, saved.VATValidatedAt, "changing VAT number must clear validation timestamp")
}

func TestBillingProfile_SyncFromStripe_FillsEmptyOnly(t *testing.T) {
	h := newBPHarness(t)
	// Pre-existing profile with some user-edited values.
	require.NoError(t, h.profiles.Upsert(context.Background(), &domain.BillingProfile{
		OrganizationID: h.orgID,
		ProfileType:    domain.ProfileBusiness,
		LegalName:      "User Edited Name",
		Country:        "", // empty → should be filled by Stripe
	}))
	h.orgRepo.stripeAccountByOrg[h.orgID] = "acct_123"
	h.kyc.snap = &service.StripeAccountKYCSnapshot{
		BusinessType: "company",
		LegalName:    "Stripe Reported Name",
		Country:      "FR",
		AddressLine1: "5 av. Stripe",
		City:         "Paris",
		PostalCode:   "75010",
		SupportEmail: "billing@user.test",
	}

	req := h.reqWithAuth(http.MethodPost, "/api/v1/me/billing-profile/sync-from-stripe", nil)
	rec := httptest.NewRecorder()
	h.handler.SyncFromStripe(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	saved, err := h.profiles.FindByOrganization(context.Background(), h.orgID)
	require.NoError(t, err)
	assert.Equal(t, "User Edited Name", saved.LegalName, "user-edited value preserved")
	assert.Equal(t, "FR", saved.Country, "empty country filled from stripe")
	assert.Equal(t, "5 av. Stripe", saved.AddressLine1)
	assert.Equal(t, "billing@user.test", saved.InvoicingEmail)
	assert.NotNil(t, saved.SyncedFromKYCAt)
}

func TestBillingProfile_ValidateVAT_HappyPath(t *testing.T) {
	h := newBPHarness(t)
	require.NoError(t, h.profiles.Upsert(context.Background(), &domain.BillingProfile{
		OrganizationID: h.orgID,
		ProfileType:    domain.ProfileBusiness,
		Country:        "DE",
		VATNumber:      "DE123456789",
	}))
	h.vies.res = service.VIESResult{
		Valid:          true,
		CountryCode:    "DE",
		VATNumber:      "DE123456789",
		RegisteredName: "Beispiel GmbH",
		RawPayload:     []byte(`{"valid":true}`),
	}

	req := h.reqWithAuth(http.MethodPost, "/api/v1/me/billing-profile/validate-vat", nil)
	rec := httptest.NewRecorder()
	h.handler.ValidateVAT(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, true, body["valid"])
	assert.Equal(t, "Beispiel GmbH", body["registered_name"])

	saved, _ := h.profiles.FindByOrganization(context.Background(), h.orgID)
	assert.NotNil(t, saved.VATValidatedAt)
}

func TestBillingProfile_ValidateVAT_NoVATNumber_400(t *testing.T) {
	h := newBPHarness(t)
	require.NoError(t, h.profiles.Upsert(context.Background(), &domain.BillingProfile{
		OrganizationID: h.orgID,
		ProfileType:    domain.ProfileBusiness,
		Country:        "FR",
	}))
	req := h.reqWithAuth(http.MethodPost, "/api/v1/me/billing-profile/validate-vat", nil)
	rec := httptest.NewRecorder()
	h.handler.ValidateVAT(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&body)
	assert.Equal(t, "vat_number_required", body["error"])
}

func TestBillingProfile_CrossOrgLeakAttempt_403(t *testing.T) {
	// A request with NO org id in context must yield 403 — the handler
	// refuses to fall back to user_id-based lookups under any
	// circumstance. This is the cross-org leak guard at the boundary.
	h := newBPHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/billing-profile", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, h.userID)
	// deliberately omit ContextKeyOrganizationID
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.handler.GetMine(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Sanity: VIES network error → 502 vies_unavailable, not a generic 500.
func TestBillingProfile_ValidateVAT_VIESDown_502(t *testing.T) {
	h := newBPHarness(t)
	require.NoError(t, h.profiles.Upsert(context.Background(), &domain.BillingProfile{
		OrganizationID: h.orgID,
		ProfileType:    domain.ProfileBusiness,
		Country:        "DE",
		VATNumber:      "DE123456789",
	}))
	h.vies.err = errors.New("vies: timeout")
	req := h.reqWithAuth(http.MethodPost, "/api/v1/me/billing-profile/validate-vat", nil)
	rec := httptest.NewRecorder()
	h.handler.ValidateVAT(rec, req)
	assert.Equal(t, http.StatusBadGateway, rec.Code)
}
