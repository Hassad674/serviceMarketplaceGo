package invoicing_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------- Mocks ----------

// Compile-time interface satisfaction. If a port grows a method the
// compiler fails here — that is the cheapest possible way to keep
// every mock honest.
var (
	_ repository.InvoiceRepository        = (*mockInvoiceRepo)(nil)
	_ repository.BillingProfileRepository = (*mockProfileRepo)(nil)
	_ service.PDFRenderer                 = (*mockPDF)(nil)
	_ service.StorageService              = (*mockStorage)(nil)
	_ service.InvoiceDeliverer            = (*mockDeliverer)(nil)
	_ invoicingapp.IdempotencyClaimer     = (*mockIdempotency)(nil)
)

type mockInvoiceRepo struct {
	createInvoiceFn      func(ctx context.Context, inv *invoicing.Invoice) error
	createCreditNoteFn   func(ctx context.Context, cn *invoicing.CreditNote) error
	reserveNumberFn      func(ctx context.Context, scope invoicing.CounterScope) (int64, error)
	findByIDFn           func(ctx context.Context, id uuid.UUID) (*invoicing.Invoice, error)
	findByEventIDFn      func(ctx context.Context, eventID string) (*invoicing.Invoice, error)
	findByPIIDFn         func(ctx context.Context, paymentIntentID string) (*invoicing.Invoice, error)
	findCnByEventIDFn    func(ctx context.Context, eventID string) (*invoicing.CreditNote, error)
	markCreditedFn       func(ctx context.Context, invoiceID uuid.UUID) error
	listByOrgFn          func(ctx context.Context, organizationID uuid.UUID, cursor string, limit int) ([]*invoicing.Invoice, string, error)
	hasItemForPaymentFn  func(ctx context.Context, paymentRecordID uuid.UUID) (bool, error)
	listReleasedForOrgFn func(ctx context.Context, organizationID uuid.UUID, periodStart, periodEnd time.Time) ([]repository.ReleasedPaymentRecord, error)
	listAdminFn          func(ctx context.Context, filters repository.AdminInvoiceFilters, cursor string, limit int) ([]*repository.AdminInvoiceRow, string, error)
	findCnByIDFn         func(ctx context.Context, id uuid.UUID) (*invoicing.CreditNote, error)

	persistedInvoices    []*invoicing.Invoice
	persistedCreditNotes []*invoicing.CreditNote
	markedCreditedIDs    []uuid.UUID
}

func (m *mockInvoiceRepo) CreateInvoice(ctx context.Context, inv *invoicing.Invoice) error {
	if m.createInvoiceFn != nil {
		return m.createInvoiceFn(ctx, inv)
	}
	m.persistedInvoices = append(m.persistedInvoices, inv)
	return nil
}
func (m *mockInvoiceRepo) CreateCreditNote(ctx context.Context, cn *invoicing.CreditNote) error {
	if m.createCreditNoteFn != nil {
		return m.createCreditNoteFn(ctx, cn)
	}
	m.persistedCreditNotes = append(m.persistedCreditNotes, cn)
	return nil
}
func (m *mockInvoiceRepo) FindInvoiceByStripePaymentIntentID(ctx context.Context, paymentIntentID string) (*invoicing.Invoice, error) {
	if m.findByPIIDFn != nil {
		return m.findByPIIDFn(ctx, paymentIntentID)
	}
	return nil, invoicing.ErrNotFound
}
func (m *mockInvoiceRepo) MarkInvoiceCredited(ctx context.Context, invoiceID uuid.UUID) error {
	if m.markCreditedFn != nil {
		return m.markCreditedFn(ctx, invoiceID)
	}
	m.markedCreditedIDs = append(m.markedCreditedIDs, invoiceID)
	return nil
}
func (m *mockInvoiceRepo) ReserveNumber(ctx context.Context, scope invoicing.CounterScope) (int64, error) {
	if m.reserveNumberFn != nil {
		return m.reserveNumberFn(ctx, scope)
	}
	return 1, nil
}
func (m *mockInvoiceRepo) FindInvoiceByID(ctx context.Context, id uuid.UUID) (*invoicing.Invoice, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, invoicing.ErrNotFound
}
func (m *mockInvoiceRepo) FindInvoiceByStripeEventID(ctx context.Context, eventID string) (*invoicing.Invoice, error) {
	if m.findByEventIDFn != nil {
		return m.findByEventIDFn(ctx, eventID)
	}
	return nil, invoicing.ErrNotFound
}
func (m *mockInvoiceRepo) FindCreditNoteByStripeEventID(ctx context.Context, eventID string) (*invoicing.CreditNote, error) {
	if m.findCnByEventIDFn != nil {
		return m.findCnByEventIDFn(ctx, eventID)
	}
	return nil, invoicing.ErrNotFound
}
func (m *mockInvoiceRepo) ListInvoicesByOrganization(ctx context.Context, organizationID uuid.UUID, cursor string, limit int) ([]*invoicing.Invoice, string, error) {
	if m.listByOrgFn != nil {
		return m.listByOrgFn(ctx, organizationID, cursor, limit)
	}
	return nil, "", nil
}
func (m *mockInvoiceRepo) HasInvoiceItemForPaymentRecord(ctx context.Context, paymentRecordID uuid.UUID) (bool, error) {
	if m.hasItemForPaymentFn != nil {
		return m.hasItemForPaymentFn(ctx, paymentRecordID)
	}
	return false, nil
}
func (m *mockInvoiceRepo) ListReleasedPaymentRecordsForOrg(ctx context.Context, organizationID uuid.UUID, periodStart, periodEnd time.Time) ([]repository.ReleasedPaymentRecord, error) {
	if m.listReleasedForOrgFn != nil {
		return m.listReleasedForOrgFn(ctx, organizationID, periodStart, periodEnd)
	}
	return nil, nil
}
func (m *mockInvoiceRepo) ListInvoicesAdmin(ctx context.Context, filters repository.AdminInvoiceFilters, cursor string, limit int) ([]*repository.AdminInvoiceRow, string, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, filters, cursor, limit)
	}
	return nil, "", nil
}
func (m *mockInvoiceRepo) FindCreditNoteByID(ctx context.Context, id uuid.UUID) (*invoicing.CreditNote, error) {
	if m.findCnByIDFn != nil {
		return m.findCnByIDFn(ctx, id)
	}
	return nil, invoicing.ErrNotFound
}

type mockProfileRepo struct {
	findByOrgFn func(ctx context.Context, organizationID uuid.UUID) (*invoicing.BillingProfile, error)
	upsertFn    func(ctx context.Context, p *invoicing.BillingProfile) error
}

func (m *mockProfileRepo) FindByOrganization(ctx context.Context, organizationID uuid.UUID) (*invoicing.BillingProfile, error) {
	if m.findByOrgFn != nil {
		return m.findByOrgFn(ctx, organizationID)
	}
	return nil, invoicing.ErrNotFound
}
func (m *mockProfileRepo) Upsert(ctx context.Context, p *invoicing.BillingProfile) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, p)
	}
	return nil
}

type mockPDF struct {
	renderInvoiceFn    func(ctx context.Context, inv *invoicing.Invoice, language string) ([]byte, error)
	renderCreditNoteFn func(ctx context.Context, cn *invoicing.CreditNote, language string) ([]byte, error)
	calls              int
	lastLang           string
	lastInvoice        *invoicing.Invoice
}

func (m *mockPDF) RenderInvoice(ctx context.Context, inv *invoicing.Invoice, language string) ([]byte, error) {
	m.calls++
	m.lastLang = language
	m.lastInvoice = inv
	if m.renderInvoiceFn != nil {
		return m.renderInvoiceFn(ctx, inv, language)
	}
	return []byte("%PDF-1.7 fake"), nil
}
func (m *mockPDF) RenderCreditNote(ctx context.Context, cn *invoicing.CreditNote, language string) ([]byte, error) {
	if m.renderCreditNoteFn != nil {
		return m.renderCreditNoteFn(ctx, cn, language)
	}
	return []byte("%PDF-1.7 fake"), nil
}

type mockStorage struct {
	uploadFn       func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	deleteFn       func(ctx context.Context, key string) error
	publicURLFn    func(key string) string
	presignedFn    func(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error)
	downloadFn     func(ctx context.Context, key string) ([]byte, error)
	uploadCalls    int
	lastUploadKey  string
	lastUploadSize int64
	// Captured by GetPresignedDownloadURLAsAttachment so tests can
	// assert the handler passes the right human-readable filename.
	lastAttachmentKey      string
	lastAttachmentFilename string
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	m.uploadCalls++
	m.lastUploadKey = key
	m.lastUploadSize = size
	if m.uploadFn != nil {
		return m.uploadFn(ctx, key, reader, contentType, size)
	}
	return "https://r2.test/" + key, nil
}
func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}
func (m *mockStorage) GetPublicURL(key string) string {
	if m.publicURLFn != nil {
		return m.publicURLFn(key)
	}
	return "https://r2.test/" + key
}
func (m *mockStorage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	if m.presignedFn != nil {
		return m.presignedFn(ctx, key, contentType, expiry)
	}
	return "https://r2.test/upload/" + key, nil
}
func (m *mockStorage) GetPresignedDownloadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://r2.test/download/" + key, nil
}
func (m *mockStorage) GetPresignedDownloadURLAsAttachment(_ context.Context, key string, filename string, _ time.Duration) (string, error) {
	m.lastAttachmentKey = key
	m.lastAttachmentFilename = filename
	return "https://r2.test/download/" + key + "?response-content-disposition=attachment%3B+filename%3D%22" + filename + "%22", nil
}
func (m *mockStorage) Download(ctx context.Context, key string) ([]byte, error) {
	if m.downloadFn != nil {
		return m.downloadFn(ctx, key)
	}
	return nil, nil
}

type mockDeliverer struct {
	deliverInvoiceFn    func(ctx context.Context, inv *invoicing.Invoice, pdfBytes []byte, downloadURL string) error
	deliverCreditNoteFn func(ctx context.Context, cn *invoicing.CreditNote, pdfBytes []byte, downloadURL string) error
	calls               int
	lastURL             string
}

func (m *mockDeliverer) DeliverInvoice(ctx context.Context, inv *invoicing.Invoice, pdfBytes []byte, downloadURL string) error {
	m.calls++
	m.lastURL = downloadURL
	if m.deliverInvoiceFn != nil {
		return m.deliverInvoiceFn(ctx, inv, pdfBytes, downloadURL)
	}
	return nil
}
func (m *mockDeliverer) DeliverCreditNote(ctx context.Context, cn *invoicing.CreditNote, pdfBytes []byte, downloadURL string) error {
	if m.deliverCreditNoteFn != nil {
		return m.deliverCreditNoteFn(ctx, cn, pdfBytes, downloadURL)
	}
	return nil
}

type mockIdempotency struct {
	tryClaimFn func(ctx context.Context, eventID string) (bool, error)
	calls      int
}

func (m *mockIdempotency) TryClaim(ctx context.Context, eventID string) (bool, error) {
	m.calls++
	if m.tryClaimFn != nil {
		return m.tryClaimFn(ctx, eventID)
	}
	return true, nil
}

// ---------- Test helpers ----------

func defaultIssuer() invoicing.IssuerInfo {
	return invoicing.IssuerInfo{
		LegalName:    "Marketplace Test SAS",
		LegalForm:    "SAS",
		SIRET:        "12345678900012",
		APECode:      "6201Z",
		AddressLine1: "1 rue de la République",
		PostalCode:   "75001",
		City:         "Paris",
		Country:      "FR",
		Email:        "billing@test.example",
	}
}

// frProfile returns a complete French billing profile suitable for the
// fr_franchise_base regime.
func frProfile(orgID uuid.UUID) *invoicing.BillingProfile {
	now := time.Now()
	return &invoicing.BillingProfile{
		OrganizationID: orgID,
		ProfileType:    invoicing.ProfileBusiness,
		LegalName:      "Acme Studio SARL",
		LegalForm:      "SARL",
		TaxID:          "98765432100018",
		AddressLine1:   "10 boulevard Test",
		PostalCode:     "75002",
		City:           "Paris",
		Country:        "FR",
		InvoicingEmail: "billing@acme.example",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// deProfile returns a complete German billing profile with a validated
// VAT number — the canonical EU reverse-charge case.
func deProfile(orgID uuid.UUID) *invoicing.BillingProfile {
	now := time.Now()
	validated := now.Add(-1 * time.Hour)
	return &invoicing.BillingProfile{
		OrganizationID:  orgID,
		ProfileType:     invoicing.ProfileBusiness,
		LegalName:       "Berliner Solutions GmbH",
		LegalForm:       "GmbH",
		VATNumber:       "DE123456789",
		VATValidatedAt:  &validated,
		AddressLine1:    "Friedrichstraße 1",
		PostalCode:      "10117",
		City:            "Berlin",
		Country:         "DE",
		InvoicingEmail:  "rechnungen@berliner.example",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// usProfile returns a complete US billing profile — out_of_scope_eu.
func usProfile(orgID uuid.UUID) *invoicing.BillingProfile {
	now := time.Now()
	return &invoicing.BillingProfile{
		OrganizationID: orgID,
		ProfileType:    invoicing.ProfileBusiness,
		LegalName:      "Acme Corp",
		AddressLine1:   "1 Market St",
		PostalCode:     "94105",
		City:           "San Francisco",
		Country:        "US",
		InvoicingEmail: "ap@acme.example",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func newSvc(t *testing.T) (*invoicingapp.Service, *mockInvoiceRepo, *mockProfileRepo, *mockPDF, *mockStorage, *mockDeliverer, *mockIdempotency) {
	t.Helper()
	invRepo := &mockInvoiceRepo{}
	profileRepo := &mockProfileRepo{}
	pdf := &mockPDF{}
	storage := &mockStorage{}
	deliverer := &mockDeliverer{}
	idem := &mockIdempotency{}
	svc := invoicingapp.NewService(invoicingapp.ServiceDeps{
		Invoices:    invRepo,
		Profiles:    profileRepo,
		PDF:         pdf,
		Storage:     storage,
		Deliverer:   deliverer,
		Issuer:      defaultIssuer(),
		Idempotency: idem,
	})
	return svc, invRepo, profileRepo, pdf, storage, deliverer, idem
}

func defaultInput(orgID uuid.UUID) invoicingapp.IssueFromSubscriptionInput {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	return invoicingapp.IssueFromSubscriptionInput{
		OrganizationID:        orgID,
		StripeEventID:         "evt_test_001",
		StripeInvoiceID:       "in_test_001",
		StripePaymentIntentID: "pi_test_001",
		AmountCents:           4900,
		Currency:              "EUR",
		PeriodStart:           now,
		PeriodEnd:             now.AddDate(0, 1, 0),
		PlanLabel:             "Premium Agence — avril 2026",
	}
}

// ---------- Tests ----------

func TestIssueFromSubscription_HappyPath_FRDomestic(t *testing.T) {
	svc, invRepo, profileRepo, pdf, storage, deliverer, idem := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "FAC-000001", out.Number)
	assert.Equal(t, invoicing.RegimeFRFranchiseBase, out.TaxRegime)
	assert.Equal(t, int64(4900), out.AmountInclTaxCents)
	assert.Equal(t, "EUR", out.Currency)
	assert.Equal(t, invoicing.SourceSubscription, out.SourceType)
	assert.True(t, out.IsFinalized())
	assert.Equal(t, 1, idem.calls, "idempotency claim must be called exactly once")
	assert.Equal(t, 1, pdf.calls, "pdf renderer called exactly once")
	assert.Equal(t, "fr", pdf.lastLang, "FR recipient picks fr template")
	assert.Equal(t, 1, storage.uploadCalls)
	assert.Contains(t, storage.lastUploadKey, "invoices/"+orgID.String()+"/FAC-000001.pdf")
	assert.Equal(t, 1, deliverer.calls)
	require.Len(t, invRepo.persistedInvoices, 1)
	// Mentions resolve fr_franchise_base — check at least one mention is non-empty.
	require.NotEmpty(t, out.MentionsRendered)
	var hasFranchise bool
	for _, m := range out.MentionsRendered {
		if contains(m, "TVA non applicable") {
			hasFranchise = true
			break
		}
	}
	assert.True(t, hasFranchise, "fr_franchise_base must surface 'TVA non applicable' mention; got %v", out.MentionsRendered)
}

func TestIssueFromSubscription_HappyPath_EUReverseCharge(t *testing.T) {
	svc, _, profileRepo, pdf, _, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return deProfile(orgID), nil
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, invoicing.RegimeEUReverseCharge, out.TaxRegime)
	assert.Equal(t, "en", pdf.lastLang, "DE recipient picks en template")
	assert.Equal(t, 1, deliverer.calls)
	var hasReverse bool
	for _, m := range out.MentionsRendered {
		if contains(m, "Autoliquidation") || contains(m, "Reverse charge") {
			hasReverse = true
			break
		}
	}
	assert.True(t, hasReverse, "eu_reverse_charge must surface autoliquidation mention; got %v", out.MentionsRendered)
}

func TestIssueFromSubscription_HappyPath_OutsideEU(t *testing.T) {
	svc, _, profileRepo, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return usProfile(orgID), nil
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, invoicing.RegimeOutOfScopeEU, out.TaxRegime)
}

func TestIssueFromSubscription_IdempotencyReplay_ShortCircuits(t *testing.T) {
	svc, invRepo, profileRepo, pdf, storage, deliverer, idem := newSvc(t)
	orgID := uuid.New()
	idem.tryClaimFn = func(_ context.Context, _ string) (bool, error) { return false, nil }
	profileLookups := 0
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		profileLookups++
		return frProfile(orgID), nil
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	assert.NoError(t, err)
	assert.Nil(t, out, "duplicate event returns (nil, nil) — not an error")
	assert.Equal(t, 0, profileLookups, "profile lookup must not happen on duplicate event")
	assert.Equal(t, 0, pdf.calls, "no pdf render on duplicate")
	assert.Equal(t, 0, storage.uploadCalls, "no upload on duplicate")
	assert.Equal(t, 0, deliverer.calls, "no email on duplicate")
	assert.Empty(t, invRepo.persistedInvoices, "no persistence on duplicate")
}

func TestIssueFromSubscription_KeyIsNamespaced_NotCollidingWithGatewayClaim(t *testing.T) {
	// Regression: the webhook dispatcher (stripe_handler.go) claims the
	// bare event_id at gateway level for ALL events, then dispatches.
	// Before this fix, IssueFromSubscription re-claimed the SAME bare
	// event_id, so the inner claim always failed on webhook-driven calls
	// and the invoice was silently skipped as a "duplicate".
	//
	// The inner claim must use a feature-namespaced key so the two
	// idempotency layers protect against different replay axes without
	// stomping on each other.
	svc, _, profileRepo, _, _, _, idem := newSvc(t)
	orgID := uuid.New()
	rawEventID := "evt_test_no_collision"
	gatewayClaimedKey := rawEventID

	// Simulate the gateway already having claimed the bare event id.
	// Inner claim with the bare key would fail; inner claim with a
	// namespaced key must succeed.
	var receivedKey string
	idem.tryClaimFn = func(_ context.Context, key string) (bool, error) {
		receivedKey = key
		if key == gatewayClaimedKey {
			return false, nil // gateway-style raw key — would fail here
		}
		return true, nil
	}
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}

	in := defaultInput(orgID)
	in.StripeEventID = rawEventID
	out, err := svc.IssueFromSubscription(context.Background(), in)

	assert.NoError(t, err)
	assert.NotNil(t, out, "must produce an invoice — not skipped as duplicate")
	assert.NotEqual(t, rawEventID, receivedKey,
		"inner TryClaim MUST use a namespaced key, not the bare event id")
	assert.Contains(t, receivedKey, rawEventID,
		"namespaced key still includes the event id for traceability")
}

func TestIssueFromSubscription_DBLevelDedup_ReturnsExistingWithoutReissue(t *testing.T) {
	svc, invRepo, _, pdf, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	now := time.Now()
	finalized := now.Add(-1 * time.Hour)
	existing := &invoicing.Invoice{
		ID:                      uuid.New(),
		Number:                  "FAC-000042",
		RecipientOrganizationID: orgID,
		Status:                  invoicing.StatusIssued,
		FinalizedAt:             &finalized,
		StripeEventID:           "evt_test_001",
	}
	invRepo.findByEventIDFn = func(_ context.Context, _ string) (*invoicing.Invoice, error) {
		return existing, nil
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "FAC-000042", out.Number, "must return the pre-existing row, not issue a new one")
	assert.Equal(t, 0, pdf.calls)
	assert.Equal(t, 0, storage.uploadCalls)
	assert.Equal(t, 0, deliverer.calls)
}

func TestIssueFromSubscription_MissingBillingProfile_Errors(t *testing.T) {
	svc, _, profileRepo, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return nil, invoicing.ErrNotFound
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, invoicing.ErrNotFound), "error chain must wrap invoicing.ErrNotFound; got %v", err)
}

func TestIssueFromSubscription_WrongCurrency_Errors(t *testing.T) {
	svc, _, profileRepo, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	in := defaultInput(orgID)
	in.Currency = "USD"

	out, err := svc.IssueFromSubscription(context.Background(), in)

	require.Error(t, err)
	assert.Nil(t, out)
	assert.True(t, errors.Is(err, invoicing.ErrInvalidCurrency))
}

func TestIssueFromSubscription_StorageUploadFailure_Errors_NoDBWrite(t *testing.T) {
	svc, invRepo, profileRepo, _, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	storage.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
		return "", fmt.Errorf("r2: connection refused")
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "upload pdf")
	assert.Empty(t, invRepo.persistedInvoices, "no DB write on upload failure")
	assert.Equal(t, 0, deliverer.calls)
}

func TestIssueFromSubscription_EmailFailure_DoesNotFailCall(t *testing.T) {
	svc, invRepo, profileRepo, _, _, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	deliverer.deliverInvoiceFn = func(_ context.Context, _ *invoicing.Invoice, _ []byte, _ string) error {
		return fmt.Errorf("resend: rate limited")
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	require.NoError(t, err, "email failure must NOT bubble — invoice is already persisted")
	require.NotNil(t, out)
	assert.True(t, out.IsFinalized())
	require.Len(t, invRepo.persistedInvoices, 1)
}

func TestIssueFromSubscription_PersistFailureAfterUpload_Errors(t *testing.T) {
	svc, invRepo, profileRepo, _, storage, deliverer, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	invRepo.createInvoiceFn = func(_ context.Context, _ *invoicing.Invoice) error {
		return fmt.Errorf("postgres: deadlock detected")
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "persist invoice")
	assert.Equal(t, 1, storage.uploadCalls, "upload still happened — orphan in r2 is acceptable")
	assert.Equal(t, 0, deliverer.calls, "no email when persistence fails")
}

func TestIssueFromSubscription_NumberFormat_FAC_NNNNNN(t *testing.T) {
	svc, _, profileRepo, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Regexp(t, `^FAC-\d{6,}$`, out.Number)
}

func TestIssueFromSubscription_RecipientSnapshotFrozenFromProfile(t *testing.T) {
	svc, _, profileRepo, _, _, _, _ := newSvc(t)
	orgID := uuid.New()
	p := frProfile(orgID)
	profileRepo.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*invoicing.BillingProfile, error) {
		return p, nil
	}

	out, err := svc.IssueFromSubscription(context.Background(), defaultInput(orgID))
	require.NoError(t, err)

	// The recipient snapshot must mirror the profile fields verbatim
	// — this is the "frozen" guarantee the legal docs rely on.
	assert.Equal(t, p.LegalName, out.RecipientSnapshot.LegalName)
	assert.Equal(t, p.TaxID, out.RecipientSnapshot.TaxID)
	assert.Equal(t, p.City, out.RecipientSnapshot.City)
	assert.Equal(t, p.Country, out.RecipientSnapshot.Country)
	assert.Equal(t, p.InvoicingEmail, out.RecipientSnapshot.Email)
	assert.Equal(t, orgID.String(), out.RecipientSnapshot.OrganizationID)
}

// ---------- helpers ----------

// contains is a tiny wrapper around strings.Contains kept here to make
// the assertion sites read more naturally ("contains the autoliquidation
// fragment in the rendered mention").
func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
